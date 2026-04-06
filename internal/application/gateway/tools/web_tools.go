package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/playwright-community/playwright-go"

	connectorsdto "dreamcreator/internal/application/connectors/dto"
	domainweb "dreamcreator/internal/domain/web"
)

const webFetchTypeBuiltin = "builtin"
const webFetchTypePlaywright = "playwright"

const defaultWebFetchType = webFetchTypeBuiltin
const defaultWebFetchTimeoutSeconds = 20
const defaultWebFetchMaxChars = 50000
const defaultWebFetchMaxRedirects = 3
const defaultWebFetchRetryMax = 2
const defaultWebFetchAcceptMarkdown = true
const defaultWebFetchPlaywrightMarkdown = true
const defaultWebFetchEnableUserAgent = true
const defaultWebFetchUserAgent = domainweb.DefaultBrowserRequestUserAgent
const defaultWebFetchAcceptLanguage = domainweb.DefaultBrowserRequestAcceptLanguage
const defaultWebSearchTimeoutSeconds = 30
const defaultWebSearchCacheTtlMinutes = 15
const defaultWebSearchCount = 5
const maxWebSearchCount = 10
const defaultWebSearchType = "api"

const (
	webStatusOK    = "ok"
	webStatusError = "error"

	webQualitySufficient = "sufficient"
	webQualityEmpty      = "empty"

	nextActionContinue               = "continue"
	nextActionInspectErrorThenSwitch = "inspect_error_then_switch"
	nextActionUseOtherToolsOrSkills  = "use_other_tools_or_skills"
)

const braveSearchEndpoint = "https://api.search.brave.com/res/v1/web/search"
const tavilySearchEndpoint = "https://api.tavily.com/search"

var htmlScriptStylePattern = regexp.MustCompile(`(?is)<(script|style)[^>]*>.*?</(script|style)>`)
var htmlTagPattern = regexp.MustCompile(`(?s)<[^>]+>`)
var htmlSpacePattern = regexp.MustCompile(`\s+`)
var htmlTitlePattern = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)
var markdownFrontMatterTitlePattern = regexp.MustCompile(`(?im)^title:\s*(.+)$`)
var markdownHeadingPattern = regexp.MustCompile(`(?m)^\s*#\s+(.+)$`)
var markdownCodeFencePattern = regexp.MustCompile("(?s)```.*?```")
var markdownImagePattern = regexp.MustCompile(`!\[(.*?)\]\([^)]+\)`)
var markdownLinkPattern = regexp.MustCompile(`\[(.*?)\]\([^)]+\)`)
var markdownHeadingMarkerPattern = regexp.MustCompile(`(?m)^\s{0,3}#{1,6}\s*`)
var markdownListMarkerPattern = regexp.MustCompile(`(?m)^\s*[-*+]\s+`)
var markdownBlockQuotePattern = regexp.MustCompile(`(?m)^\s*>\s*`)
var markdownTableSepPattern = regexp.MustCompile(`(?m)^\s*\|?[-:\s|]+\|?\s*$`)

type webSearchResult struct {
	Title       string `json:"title,omitempty"`
	URL         string `json:"url,omitempty"`
	Description string `json:"description,omitempty"`
	Age         string `json:"age,omitempty"`
}

type webSearchResponse struct {
	Status             string            `json:"status"`
	Retryable          bool              `json:"retryable"`
	NextAction         string            `json:"next_action"`
	Message            string            `json:"message"`
	Provider           string            `json:"provider"`
	Quality            string            `json:"quality"`
	SufficiencyReason  string            `json:"sufficiency_reason,omitempty"`
	WebSearchAvailable bool              `json:"web_search_available"`
	Query              string            `json:"query"`
	Results            []webSearchResult `json:"results,omitempty"`
	Cached             bool              `json:"cached,omitempty"`
	Data               map[string]any    `json:"data,omitempty"`
}

type braveSearchResponse struct {
	Web struct {
		Results []struct {
			Title       string `json:"title"`
			URL         string `json:"url"`
			Description string `json:"description"`
			Age         string `json:"age"`
		} `json:"results"`
	} `json:"web"`
}

type tavilySearchResponse struct {
	Query   string `json:"query"`
	Answer  string `json:"answer"`
	Results []struct {
		Title      string      `json:"title"`
		URL        string      `json:"url"`
		Content    string      `json:"content"`
		RawContent interface{} `json:"raw_content"`
	} `json:"results"`
}

type webSearchCacheEntry struct {
	value     webSearchResponse
	expiresAt time.Time
}

var webSearchCache = struct {
	mu      sync.RWMutex
	entries map[string]webSearchCacheEntry
}{
	entries: make(map[string]webSearchCacheEntry),
}

type webFetchResult struct {
	Status             string         `json:"status"`
	Retryable          bool           `json:"retryable"`
	NextAction         string         `json:"next_action"`
	Message            string         `json:"message"`
	Provider           string         `json:"provider"`
	Quality            string         `json:"quality"`
	SufficiencyReason  string         `json:"sufficiency_reason,omitempty"`
	WebSearchAvailable bool           `json:"web_search_available"`
	URL                string         `json:"url"`
	FinalURL           string         `json:"finalUrl,omitempty"`
	HTTPStatus         int            `json:"httpStatus,omitempty"`
	ContentType        string         `json:"contentType,omitempty"`
	Content            string         `json:"content"`
	MarkdownTokens     int            `json:"markdownTokens,omitempty"`
	ContentSignal      string         `json:"contentSignal,omitempty"`
	Truncated          bool           `json:"truncated,omitempty"`
	Data               map[string]any `json:"data,omitempty"`
}

type webFetchOptions struct {
	TimeoutSeconds  int
	MaxChars        int
	MaxRedirects    int
	RetryMax        int
	AcceptMarkdown  bool
	EnableUserAgent bool
	UserAgent       string
	AcceptLanguage  string
	Headers         map[string]any
}

type webFetchPlaywrightOptions struct {
	Markdown bool
}

type webFetchResponse struct {
	URL            string
	FinalURL       string
	Status         int
	Headers        map[string]string
	ContentType    string
	Content        string
	MarkdownTokens int
	ContentSignal  string
	TimeoutStage   string
	Truncated      bool
}

type ConnectorsReader interface {
	ListConnectors(ctx context.Context) ([]connectorsdto.Connector, error)
}

func runWebFetchTool(settings SettingsReader, connectors ConnectorsReader) func(ctx context.Context, args string) (string, error) {
	return func(ctx context.Context, args string) (string, error) {
		payload, err := parseToolArgs(args)
		if err != nil {
			return "", err
		}
		url := getStringArg(payload, "url", "href")
		if url == "" {
			return "", errors.New("url is required")
		}
		config := resolveToolsConfig(ctx, settings)
		enabled := true
		if value, ok := resolveWebFetchConfigBool(config, "enabled"); ok {
			enabled = value
		}
		if !enabled {
			return "", errors.New("web_fetch disabled")
		}
		method := strings.ToUpper(getStringArg(payload, "method"))
		if method == "" {
			method = http.MethodGet
		}
		fetchType, err := resolveWebFetchType(payload, config)
		if err != nil {
			return "", err
		}
		options := resolveWebFetchOptions(payload, config, defaultWebFetchTimeoutSeconds)
		playwrightOptions := resolveWebFetchPlaywrightOptions(payload, config)
		cookies, err := resolveConnectorCookiesForURL(ctx, connectors, url)
		if err != nil {
			return "", err
		}
		response, err := fetchByWebFetchType(ctx, fetchType, method, url, cookies, options, playwrightOptions)
		result := buildWebFetchToolResult(url, response, err)
		return marshalResult(result), nil
	}
}

func runWebSearchTool(settings SettingsReader, connectors ConnectorsReader) func(ctx context.Context, args string) (string, error) {
	return func(ctx context.Context, args string) (string, error) {
		payload, err := parseToolArgs(args)
		if err != nil {
			return "", err
		}
		query := getStringArg(payload, "query", "q")
		if query == "" {
			return "", errors.New("query is required")
		}
		config := resolveToolsConfig(ctx, settings)
		count := resolveWebSearchCount(payload, config)
		response := runWebSearchWithFallback(ctx, payload, config, connectors, query, count)
		return marshalResult(response), nil
	}
}

func runWebSearchWithFallback(
	ctx context.Context,
	payload toolArgs,
	config map[string]any,
	connectors ConnectorsReader,
	query string,
	count int,
) webSearchResponse {
	_ = connectors
	searchType := resolveWebSearchType(config)
	switch searchType {
	case "api":
		provider := strings.ToLower(strings.TrimSpace(getNestedString(config, "web", "search", "provider")))
		if provider == "" {
			provider = "brave"
		}
		result, err := runWebSearchByAPI(ctx, payload, config, query, count)
		if err != nil {
			return webSearchResponse{
				Status:             webStatusError,
				Retryable:          isTimeoutError(err),
				NextAction:         nextActionInspectErrorThenSwitch,
				Message:            trimToMaxChars(strings.TrimSpace(err.Error()), 260),
				Provider:           provider,
				Quality:            webQualityEmpty,
				WebSearchAvailable: true,
				Query:              query,
			}
		}
		quality := webQualitySufficient
		if len(result.Results) == 0 {
			quality = webQualityEmpty
		}
		return webSearchResponse{
			Status:             webStatusOK,
			Retryable:          false,
			NextAction:         nextActionContinue,
			Message:            "search_completed",
			Provider:           provider,
			Quality:            quality,
			WebSearchAvailable: true,
			Query:              query,
			Results:            result.Results,
			Cached:             result.Cached,
			Data: map[string]any{
				"results_count": len(result.Results),
			},
		}
	case "external_tools":
		return webSearchResponse{
			Status:             webStatusError,
			Retryable:          false,
			NextAction:         nextActionUseOtherToolsOrSkills,
			Message:            "web_search_external_tools_not_configured",
			Provider:           "external_tools",
			Quality:            webQualityEmpty,
			WebSearchAvailable: false,
			Query:              query,
		}
	default:
		return webSearchResponse{
			Status:             webStatusError,
			Retryable:          false,
			NextAction:         nextActionUseOtherToolsOrSkills,
			Message:            "web_search_type_not_supported",
			Provider:           "web_search",
			Quality:            webQualityEmpty,
			WebSearchAvailable: false,
			Query:              query,
			Data: map[string]any{
				"type": searchType,
			},
		}
	}
}

func buildWebFetchToolResult(requestURL string, response webFetchResponse, err error) webFetchResult {
	finalURL := strings.TrimSpace(response.FinalURL)
	if finalURL == "" {
		finalURL = strings.TrimSpace(requestURL)
	}
	if err != nil {
		return webFetchResult{
			Status:             webStatusError,
			Retryable:          isTimeoutError(err),
			NextAction:         nextActionInspectErrorThenSwitch,
			Message:            trimToMaxChars(strings.TrimSpace(err.Error()), 260),
			Provider:           "web_fetch",
			Quality:            webQualityEmpty,
			WebSearchAvailable: true,
			URL:                strings.TrimSpace(requestURL),
			FinalURL:           finalURL,
			HTTPStatus:         response.Status,
			ContentType:        response.ContentType,
			Content:            response.Content,
			MarkdownTokens:     response.MarkdownTokens,
			ContentSignal:      response.ContentSignal,
			Truncated:          response.Truncated,
			Data: map[string]any{
				"error":        trimToMaxChars(strings.TrimSpace(err.Error()), 260),
				"httpStatus":   response.Status,
				"finalURL":     finalURL,
				"timeoutStage": response.TimeoutStage,
				"truncated":    response.Truncated,
			},
		}
	}
	quality := webQualitySufficient
	if strings.TrimSpace(response.Content) == "" {
		quality = webQualityEmpty
	}
	return webFetchResult{
		Status:             webStatusOK,
		Retryable:          false,
		NextAction:         nextActionContinue,
		Message:            "ok",
		Provider:           "web_fetch",
		Quality:            quality,
		WebSearchAvailable: true,
		URL:                strings.TrimSpace(requestURL),
		FinalURL:           finalURL,
		HTTPStatus:         response.Status,
		ContentType:        response.ContentType,
		Content:            response.Content,
		MarkdownTokens:     response.MarkdownTokens,
		ContentSignal:      response.ContentSignal,
		Truncated:          response.Truncated,
		Data: map[string]any{
			"httpStatus":   response.Status,
			"finalURL":     finalURL,
			"timeoutStage": response.TimeoutStage,
			"truncated":    response.Truncated,
		},
	}
}

func runWebSearchByAPI(ctx context.Context, payload toolArgs, config map[string]any, query string, count int) (webSearchResponse, error) {
	provider := strings.ToLower(getNestedString(config, "web", "search", "provider"))
	if provider == "" {
		provider = "brave"
	}
	country := resolveWebSearchString(payload, config, "country", "US")
	searchLang := resolveWebSearchString(payload, config, "search_lang", "")
	uiLang := resolveWebSearchString(payload, config, "ui_lang", "")
	freshness := resolveWebSearchString(payload, config, "freshness", "")
	cacheTtlMinutes := resolveWebSearchInt(config, "cacheTtlMinutes", defaultWebSearchCacheTtlMinutes)
	cacheKey := normalizeWebSearchCacheKey(provider, query, count, country, searchLang, uiLang, freshness)
	if cacheTtlMinutes > 0 {
		if cached, ok := readWebSearchCache(cacheKey); ok {
			cached.Cached = true
			return cached, nil
		}
	}
	timeoutSeconds := resolveWebSearchInt(config, "timeoutSeconds", defaultWebSearchTimeoutSeconds)
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
	defer cancel()

	var (
		response webSearchResponse
		err      error
	)
	switch provider {
	case "brave":
		response, err = runBraveSearch(timeoutCtx, config, query, count, country, searchLang, uiLang, freshness, timeoutSeconds)
	case "tavily":
		response, err = runTavilySearch(timeoutCtx, config, payload, query, count, timeoutSeconds)
	default:
		return webSearchResponse{}, errors.New("web_search provider not implemented: " + provider)
	}
	if err != nil {
		return webSearchResponse{}, err
	}
	response.Query = query
	response.Provider = provider
	if cacheTtlMinutes > 0 {
		writeWebSearchCache(cacheKey, response, time.Duration(cacheTtlMinutes)*time.Minute)
	}
	return response, nil
}

func resolveWebSearchCount(payload toolArgs, config map[string]any) int {
	count, ok := getIntArg(payload, "count", "maxResults", "max_results")
	if !ok || count <= 0 {
		count = resolveWebSearchInt(config, "maxResults", defaultWebSearchCount)
	}
	if count <= 0 {
		count = defaultWebSearchCount
	}
	if count > maxWebSearchCount {
		count = maxWebSearchCount
	}
	if count < 1 {
		count = 1
	}
	return count
}

func resolveWebSearchString(payload toolArgs, config map[string]any, key string, fallback string) string {
	value := getStringArg(payload, key)
	if value == "" {
		value = getNestedString(config, "web", "search", key)
	}
	if value == "" {
		return fallback
	}
	return value
}

func normalizeWebSearchType(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch normalized {
	case "api":
		return "api"
	case "external_tools", "external-tools", "external tools":
		return "external_tools"
	default:
		return ""
	}
}

func resolveWebSearchType(config map[string]any) string {
	if value := normalizeWebSearchType(getNestedString(config, "web", "search", "type")); value != "" {
		return value
	}
	if enabled, ok := getNestedBool(config, "web", "search", "enabled"); ok && enabled {
		return "api"
	}
	return defaultWebSearchType
}

func normalizeConnectorType(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch normalized {
	case "google":
		return "google"
	case "xiaohongshu", "xhs":
		return "xiaohongshu"
	case "bilibili", "b23":
		return "bilibili"
	default:
		return ""
	}
}

func normalizeWebFetchType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case webFetchTypePlaywright:
		return webFetchTypePlaywright
	case webFetchTypeBuiltin:
		return webFetchTypeBuiltin
	default:
		return ""
	}
}

func resolveWebFetchType(payload toolArgs, config map[string]any) (string, error) {
	if raw := strings.TrimSpace(getStringArg(payload, "type", "mode")); raw != "" {
		if value := normalizeWebFetchType(raw); value != "" {
			return value, nil
		}
		return "", fmt.Errorf("unsupported web_fetch type: %s", raw)
	}
	fetchConfig := toolArgs(resolveWebFetchConfig(config))
	if raw := strings.TrimSpace(getStringArg(fetchConfig, "type", "mode")); raw != "" {
		if value := normalizeWebFetchType(raw); value != "" {
			return value, nil
		}
		return "", fmt.Errorf("unsupported web_fetch type: %s", raw)
	}
	return defaultWebFetchType, nil
}

func resolveWebFetchPlaywrightOptions(payload toolArgs, config map[string]any) webFetchPlaywrightOptions {
	markdown := defaultWebFetchPlaywrightMarkdown
	fetchConfig := toolArgs(resolveWebFetchConfig(config))
	if playwrightConfig := getMapArg(fetchConfig, "playwright"); playwrightConfig != nil {
		if value, ok := getBoolArg(toolArgs(playwrightConfig), "markdown", "toMarkdown"); ok {
			markdown = value
		}
	}
	if payloadPlaywright := getMapArg(payload, "playwright"); payloadPlaywright != nil {
		if value, ok := getBoolArg(toolArgs(payloadPlaywright), "markdown", "toMarkdown"); ok {
			markdown = value
		}
	}
	if value, ok := getBoolArg(payload, "markdown", "toMarkdown"); ok {
		markdown = value
	}
	return webFetchPlaywrightOptions{
		Markdown: markdown,
	}
}

func fetchByWebFetchType(
	ctx context.Context,
	fetchType string,
	method string,
	targetURL string,
	cookies []connectorsdto.ConnectorCookie,
	options webFetchOptions,
	playwrightOptions webFetchPlaywrightOptions,
) (webFetchResponse, error) {
	switch normalizeWebFetchType(fetchType) {
	case webFetchTypePlaywright:
		if strings.ToUpper(strings.TrimSpace(method)) != http.MethodGet {
			return webFetchResponse{}, errors.New("web_fetch playwright mode only supports GET")
		}
		return fetchWithPlaywrightOptions(ctx, targetURL, cookies, options, playwrightOptions)
	case webFetchTypeBuiltin:
		return fetchWithBuiltinOptions(ctx, method, targetURL, cookies, options)
	default:
		return fetchWithBuiltinOptions(ctx, method, targetURL, cookies, options)
	}
}

func resolveConnectorCookiesForURL(
	ctx context.Context,
	connectors ConnectorsReader,
	targetURL string,
) ([]connectorsdto.ConnectorCookie, error) {
	if connectors == nil {
		return nil, nil
	}
	connectorType := connectorTypeForURL(targetURL)
	if connectorType == "" {
		return nil, nil
	}
	items, err := connectors.ListConnectors(ctx)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		if normalizeConnectorType(item.Type) != connectorType {
			continue
		}
		if len(item.Cookies) == 0 {
			continue
		}
		return item.Cookies, nil
	}
	return nil, nil
}

func connectorTypeForURL(targetURL string) string {
	parsed, err := url.Parse(strings.TrimSpace(targetURL))
	if err != nil {
		return ""
	}
	host := strings.ToLower(strings.TrimSpace(parsed.Hostname()))
	if host == "" {
		return ""
	}
	switch {
	case hostMatchesDomain(host, "google.com"), hostMatchesDomain(host, "youtube.com"), hostMatchesDomain(host, "youtu.be"):
		return "google"
	case hostMatchesDomain(host, "xiaohongshu.com"), hostMatchesDomain(host, "xhslink.com"), hostMatchesDomain(host, "redbook.com"):
		return "xiaohongshu"
	case hostMatchesDomain(host, "bilibili.com"), hostMatchesDomain(host, "b23.tv"):
		return "bilibili"
	default:
		return ""
	}
}

func hostMatchesDomain(host string, domain string) bool {
	normalizedHost := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(host)), ".")
	normalizedDomain := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(domain)), ".")
	if normalizedHost == "" || normalizedDomain == "" {
		return false
	}
	return normalizedHost == normalizedDomain || strings.HasSuffix(normalizedHost, "."+normalizedDomain)
}

func fetchWithPlaywrightOptions(
	ctx context.Context,
	targetURL string,
	cookies []connectorsdto.ConnectorCookie,
	options webFetchOptions,
	playwrightOptions webFetchPlaywrightOptions,
) (webFetchResponse, error) {
	if strings.TrimSpace(targetURL) == "" {
		return webFetchResponse{}, errors.New("target url is required")
	}
	pw, err := playwright.Run()
	if err != nil {
		return webFetchResponse{}, err
	}
	defer pw.Stop()

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
		Args: []string{
			"--headless=new",
		},
	})
	if err != nil {
		return webFetchResponse{}, err
	}
	defer browser.Close()

	contextOptions := playwright.BrowserNewContextOptions{
		Viewport: &playwright.Size{
			Width:  1366,
			Height: 900,
		},
		TimezoneId: playwright.String("UTC"),
	}
	if options.EnableUserAgent {
		userAgent := strings.TrimSpace(options.UserAgent)
		if userAgent == "" {
			userAgent = defaultWebFetchUserAgent
		}
		contextOptions.UserAgent = playwright.String(userAgent)
	}
	if locale := localeFromAcceptLanguage(options.AcceptLanguage); locale != "" {
		contextOptions.Locale = playwright.String(locale)
	}
	if headers := webFetchExtraHTTPHeaders(options); len(headers) > 0 {
		contextOptions.ExtraHttpHeaders = headers
	}
	browserCtx, err := browser.NewContext(contextOptions)
	if err != nil {
		return webFetchResponse{}, err
	}
	defer browserCtx.Close()

	if len(cookies) > 0 {
		if err := browserCtx.AddCookies(toPlaywrightCookies(cookies, targetURL)); err != nil {
			return webFetchResponse{}, err
		}
	}

	page, err := browserCtx.NewPage()
	if err != nil {
		return webFetchResponse{}, err
	}

	timeoutMs := float64(resolveWebFetchTimeoutSeconds(options.TimeoutSeconds) * 1000)
	navTimeoutMs := timeoutMs * 0.6
	if navTimeoutMs < 1000 {
		navTimeoutMs = timeoutMs
	}
	readyTimeoutMs := timeoutMs * 0.25
	if readyTimeoutMs < 600 {
		readyTimeoutMs = 600
	}
	extractTimeoutMs := timeoutMs * 0.15
	if extractTimeoutMs < 500 {
		extractTimeoutMs = 500
	}
	response, err := page.Goto(strings.TrimSpace(targetURL), playwright.PageGotoOptions{
		Timeout:   playwright.Float(navTimeoutMs),
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	})
	if err != nil {
		return webFetchResponse{}, err
	}

	timeoutStage := ""
	finalURL := strings.TrimSpace(page.URL())
	if finalURL == "" {
		finalURL = strings.TrimSpace(targetURL)
	}
	if selector := resolvePlaywrightReadySelector(finalURL, targetURL); selector != "" {
		if readyErr := waitForPlaywrightReady(page, selector, readyTimeoutMs); readyErr != nil {
			timeoutStage = "ready"
		}
	}

	status := http.StatusOK
	contentType := ""
	headers := map[string]string{}
	if response != nil {
		status = response.Status()
		headers = normalizeHTTPHeaderMap(response.Headers())
		contentType = strings.TrimSpace(headers["content-type"])
		if contentType == "" {
			allHeaders, err := response.AllHeaders()
			if err == nil {
				for key, value := range allHeaders {
					if _, exists := headers[strings.ToLower(strings.TrimSpace(key))]; !exists {
						headers[strings.ToLower(strings.TrimSpace(key))] = strings.TrimSpace(value)
					}
				}
				contentType = strings.TrimSpace(headers["content-type"])
			}
		}
	}

	content, err := contentWithTimeout(ctx, page, time.Duration(extractTimeoutMs)*time.Millisecond)
	if err != nil {
		return webFetchResponse{}, err
	}
	if playwrightOptions.Markdown {
		if markdown, err := convertHTMLToMarkdown(content, finalURL); err == nil && strings.TrimSpace(markdown) != "" {
			content = markdown
			contentType = "text/markdown; charset=utf-8"
		}
	}
	content, truncated := truncateWebFetchContent([]byte(content), options.MaxChars)

	return webFetchResponse{
		URL:          strings.TrimSpace(targetURL),
		FinalURL:     finalURL,
		Status:       status,
		Headers:      headers,
		ContentType:  contentType,
		Content:      content,
		TimeoutStage: timeoutStage,
		Truncated:    truncated,
	}, nil
}

func fetchWithBuiltinOptions(
	ctx context.Context,
	method string,
	targetURL string,
	cookies []connectorsdto.ConnectorCookie,
	options webFetchOptions,
) (webFetchResponse, error) {
	if strings.TrimSpace(targetURL) == "" {
		return webFetchResponse{}, errors.New("target url is required")
	}
	httpMethod := strings.ToUpper(strings.TrimSpace(method))
	if httpMethod == "" {
		httpMethod = http.MethodGet
	}
	request, err := retryablehttp.NewRequestWithContext(ctx, httpMethod, targetURL, nil)
	if err != nil {
		return webFetchResponse{}, err
	}
	applyWebFetchHeaders(request.Request, options)
	if cookieHeader := buildCookieHeader(targetURL, cookies); cookieHeader != "" {
		if existing := strings.TrimSpace(request.Header.Get("Cookie")); existing != "" {
			request.Header.Set("Cookie", existing+"; "+cookieHeader)
		} else {
			request.Header.Set("Cookie", cookieHeader)
		}
	}
	timeoutSeconds := resolveWebFetchTimeoutSeconds(options.TimeoutSeconds)
	maxRedirects := resolveWebFetchMaxRedirects(options.MaxRedirects)
	retryMax := resolveWebFetchRetryMax(options.RetryMax)
	client := retryablehttp.NewClient()
	client.RetryMax = retryMax
	client.RetryWaitMin = 200 * time.Millisecond
	client.RetryWaitMax = 2 * time.Second
	client.Logger = nil
	client.CheckRetry = func(ctx context.Context, resp *http.Response, err error) (bool, error) {
		if !isWebFetchRetryableMethod(httpMethod) {
			return false, err
		}
		return retryablehttp.DefaultRetryPolicy(ctx, resp, err)
	}
	client.HTTPClient.Timeout = time.Duration(timeoutSeconds) * time.Second
	client.HTTPClient.CheckRedirect = func(_ *http.Request, via []*http.Request) error {
		if len(via) > maxRedirects {
			return errors.New("too many redirects")
		}
		return nil
	}
	resp, err := client.Do(request)
	if err != nil {
		return webFetchResponse{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return webFetchResponse{}, err
	}
	content, truncated := truncateWebFetchContent(body, options.MaxChars)
	markdownTokens, _ := strconv.Atoi(strings.TrimSpace(resp.Header.Get("x-markdown-tokens")))
	finalURL := strings.TrimSpace(targetURL)
	if resp.Request != nil && resp.Request.URL != nil {
		if resolved := strings.TrimSpace(resp.Request.URL.String()); resolved != "" {
			finalURL = resolved
		}
	}
	return webFetchResponse{
		URL:            strings.TrimSpace(targetURL),
		FinalURL:       finalURL,
		Status:         resp.StatusCode,
		Headers:        normalizeHTTPHeaderValues(resp.Header),
		ContentType:    strings.TrimSpace(resp.Header.Get("Content-Type")),
		Content:        content,
		MarkdownTokens: markdownTokens,
		ContentSignal:  strings.TrimSpace(resp.Header.Get("content-signal")),
		Truncated:      truncated,
	}, nil
}

func isWebFetchRetryableMethod(method string) bool {
	switch strings.ToUpper(strings.TrimSpace(method)) {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	default:
		return false
	}
}

func resolveWebFetchTimeoutSeconds(value int) int {
	timeoutSeconds := value
	if timeoutSeconds <= 0 {
		timeoutSeconds = defaultWebFetchTimeoutSeconds
	}
	return timeoutSeconds
}

func resolveWebFetchMaxRedirects(value int) int {
	maxRedirects := value
	if maxRedirects < 0 {
		maxRedirects = 0
	}
	return maxRedirects
}

func resolveWebFetchRetryMax(value int) int {
	retryMax := value
	if retryMax < 0 {
		retryMax = 0
	}
	return retryMax
}

func webFetchExtraHTTPHeaders(options webFetchOptions) map[string]string {
	result := make(map[string]string)
	for key, value := range options.Headers {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		result[trimmedKey] = toString(value)
	}
	if _, exists := result["Accept"]; !exists {
		if options.AcceptMarkdown {
			result["Accept"] = "text/markdown, text/html;q=0.9, application/xhtml+xml;q=0.8"
		} else {
			result["Accept"] = "text/html,application/xhtml+xml"
		}
	}
	if _, exists := result["Accept-Language"]; !exists && strings.TrimSpace(options.AcceptLanguage) != "" {
		result["Accept-Language"] = strings.TrimSpace(options.AcceptLanguage)
	}
	return result
}

func localeFromAcceptLanguage(value string) string {
	segments := strings.Split(strings.TrimSpace(value), ",")
	for _, segment := range segments {
		base := strings.TrimSpace(segment)
		if base == "" {
			continue
		}
		if index := strings.Index(base, ";"); index >= 0 {
			base = strings.TrimSpace(base[:index])
		}
		if base != "" {
			return base
		}
	}
	return ""
}

func convertHTMLToMarkdown(content string, targetURL string) (string, error) {
	converter := md.NewConverter(md.DomainFromURL(strings.TrimSpace(targetURL)), true, nil)
	markdown, err := converter.ConvertString(content)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(markdown), nil
}

func resolveWebFetchOptions(payload toolArgs, config map[string]any, timeoutFallback int) webFetchOptions {
	fetchConfig := toolArgs(resolveWebFetchConfig(config))
	timeoutSeconds, ok := getIntArg(fetchConfig, "timeoutSeconds")
	if !ok || timeoutSeconds <= 0 {
		timeoutSeconds, _ = getIntArg(payload, "timeoutSeconds")
	}
	if timeoutSeconds <= 0 {
		timeoutSeconds = timeoutFallback
	}
	if timeoutSeconds <= 0 {
		timeoutSeconds = defaultWebFetchTimeoutSeconds
	}

	maxChars, ok := getIntArg(fetchConfig, "maxChars")
	if !ok || maxChars <= 0 {
		maxChars, _ = getIntArg(payload, "maxChars")
	}
	if maxChars <= 0 {
		maxChars = defaultWebFetchMaxChars
	}

	maxRedirects := defaultWebFetchMaxRedirects
	if value, present := getIntArg(fetchConfig, "maxRedirects"); present {
		maxRedirects = value
	}
	if value, present := getIntArg(payload, "maxRedirects"); present {
		maxRedirects = value
	}
	if maxRedirects < 0 {
		maxRedirects = 0
	}

	retryMax := defaultWebFetchRetryMax
	if value, present := getIntArg(fetchConfig, "retryMax"); present {
		retryMax = value
	}
	if value, present := getIntArg(payload, "retryMax"); present {
		retryMax = value
	}
	if retryMax < 0 {
		retryMax = 0
	}

	acceptMarkdown, ok := getBoolArg(fetchConfig, "acceptMarkdown")
	if !ok {
		acceptMarkdown = defaultWebFetchAcceptMarkdown
	}
	if value, present := getBoolArg(payload, "acceptMarkdown", "preferMarkdown", "markdown"); present {
		acceptMarkdown = value
	}

	enableUserAgent, ok := getBoolArg(fetchConfig, "enableUserAgent")
	if !ok {
		enableUserAgent = defaultWebFetchEnableUserAgent
	}
	if value, present := getBoolArg(payload, "enableUserAgent", "useUserAgent"); present {
		enableUserAgent = value
	}

	userAgent := getStringArg(payload, "userAgent")
	if userAgent == "" {
		userAgent = getStringArg(fetchConfig, "userAgent")
	}
	if userAgent == "" {
		userAgent = defaultWebFetchUserAgent
	}

	acceptLanguage := getStringArg(payload, "acceptLanguage")
	if acceptLanguage == "" {
		acceptLanguage = getStringArg(fetchConfig, "acceptLanguage")
	}
	if acceptLanguage == "" {
		acceptLanguage = defaultWebFetchAcceptLanguage
	}
	headers := mergeAnyMap(getMapArg(fetchConfig, "headers"), getMapArg(payload, "headers"))

	return webFetchOptions{
		TimeoutSeconds:  timeoutSeconds,
		MaxChars:        maxChars,
		MaxRedirects:    maxRedirects,
		RetryMax:        retryMax,
		AcceptMarkdown:  acceptMarkdown,
		EnableUserAgent: enableUserAgent,
		UserAgent:       strings.TrimSpace(userAgent),
		AcceptLanguage:  strings.TrimSpace(acceptLanguage),
		Headers:         headers,
	}
}

func resolveWebFetchConfig(config map[string]any) map[string]any {
	current := getNestedMap(config, "web_fetch")
	if current == nil {
		return nil
	}
	return current
}

func resolveWebFetchConfigBool(config map[string]any, key string) (bool, bool) {
	return getBoolArg(toolArgs(resolveWebFetchConfig(config)), key)
}

func applyWebFetchHeaders(request *http.Request, options webFetchOptions) {
	if request == nil {
		return
	}
	for key, value := range options.Headers {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		request.Header.Set(trimmedKey, toString(value))
	}
	if strings.TrimSpace(request.Header.Get("Accept")) == "" {
		if options.AcceptMarkdown {
			request.Header.Set("Accept", "text/markdown, text/html;q=0.9, application/xhtml+xml;q=0.8")
		} else {
			request.Header.Set("Accept", "text/html,application/xhtml+xml")
		}
	}
	if strings.TrimSpace(request.Header.Get("User-Agent")) == "" && options.EnableUserAgent {
		userAgent := strings.TrimSpace(options.UserAgent)
		if userAgent == "" {
			userAgent = defaultWebFetchUserAgent
		}
		request.Header.Set("User-Agent", userAgent)
	} else if strings.TrimSpace(request.Header.Get("User-Agent")) == "" && !options.EnableUserAgent {
		// Prevent net/http from injecting the default Go user-agent when disabled.
		request.Header.Set("User-Agent", "")
	}
	if strings.TrimSpace(request.Header.Get("Accept-Language")) == "" && strings.TrimSpace(options.AcceptLanguage) != "" {
		request.Header.Set("Accept-Language", strings.TrimSpace(options.AcceptLanguage))
	}
}

func truncateWebFetchContent(body []byte, maxChars int) (string, bool) {
	content := string(body)
	if maxChars <= 0 {
		return content, false
	}
	runes := []rune(content)
	if len(runes) <= maxChars {
		return content, false
	}
	return string(runes[:maxChars]), true
}

func buildCookieHeader(targetURL string, cookies []connectorsdto.ConnectorCookie) string {
	if len(cookies) == 0 {
		return ""
	}
	parsed, err := url.Parse(targetURL)
	if err != nil {
		return ""
	}
	host := strings.ToLower(strings.TrimSpace(parsed.Hostname()))
	path := strings.TrimSpace(parsed.Path)
	if path == "" {
		path = "/"
	}
	now := time.Now().Unix()
	pairs := make([]string, 0, len(cookies))
	for _, cookie := range cookies {
		name := strings.TrimSpace(cookie.Name)
		if name == "" {
			continue
		}
		if cookie.Expires > 0 && cookie.Expires < now {
			continue
		}
		if !cookieDomainMatches(host, cookie.Domain) {
			continue
		}
		if !cookiePathMatches(path, cookie.Path) {
			continue
		}
		pairs = append(pairs, name+"="+cookie.Value)
	}
	return strings.Join(pairs, "; ")
}

func toPlaywrightCookies(cookies []connectorsdto.ConnectorCookie, targetURL string) []playwright.OptionalCookie {
	if len(cookies) == 0 {
		return nil
	}
	result := make([]playwright.OptionalCookie, 0, len(cookies))
	for _, item := range cookies {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			continue
		}
		cookie := playwright.OptionalCookie{
			Name:  name,
			Value: item.Value,
		}
		domain := strings.TrimSpace(item.Domain)
		path := strings.TrimSpace(item.Path)
		if domain != "" {
			cookie.Domain = playwright.String(domain)
		} else if strings.TrimSpace(targetURL) != "" {
			cookie.URL = playwright.String(strings.TrimSpace(targetURL))
		}
		if path != "" {
			cookie.Path = playwright.String(path)
		}
		if item.Expires > 0 {
			cookie.Expires = playwright.Float(float64(item.Expires))
		}
		cookie.HttpOnly = playwright.Bool(item.HttpOnly)
		cookie.Secure = playwright.Bool(item.Secure)
		if sameSite := toPlaywrightSameSite(item.SameSite); sameSite != nil {
			cookie.SameSite = sameSite
		}
		result = append(result, cookie)
	}
	return result
}

func toPlaywrightSameSite(value string) *playwright.SameSiteAttribute {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "lax":
		return playwright.SameSiteAttributeLax
	case "strict":
		return playwright.SameSiteAttributeStrict
	case "none":
		return playwright.SameSiteAttributeNone
	default:
		return nil
	}
}

func cookieDomainMatches(host string, cookieDomain string) bool {
	domain := strings.ToLower(strings.TrimSpace(cookieDomain))
	if domain == "" {
		return true
	}
	domain = strings.TrimPrefix(domain, ".")
	if domain == "" {
		return true
	}
	return host == domain || strings.HasSuffix(host, "."+domain)
}

func cookiePathMatches(requestPath string, cookiePath string) bool {
	path := strings.TrimSpace(cookiePath)
	if path == "" || path == "/" {
		return true
	}
	return strings.HasPrefix(requestPath, path)
}

func extractHTMLTitle(content string) string {
	if strings.TrimSpace(content) == "" {
		return ""
	}
	matches := htmlTitlePattern.FindStringSubmatch(content)
	if len(matches) < 2 {
		return ""
	}
	return compactPlainText(matches[1], 120)
}

func extractHTMLSnippet(content string, limit int) string {
	if strings.TrimSpace(content) == "" {
		return ""
	}
	sanitized := htmlScriptStylePattern.ReplaceAllString(content, " ")
	sanitized = htmlTagPattern.ReplaceAllString(sanitized, " ")
	return compactPlainText(sanitized, limit)
}

func extractWebPageTitle(content string, contentType string) string {
	if strings.Contains(strings.ToLower(contentType), "markdown") {
		if title := extractMarkdownTitle(content); title != "" {
			return title
		}
	}
	if title := extractHTMLTitle(content); title != "" {
		return title
	}
	return extractMarkdownTitle(content)
}

func extractWebPageSnippet(content string, contentType string, limit int) string {
	if strings.Contains(strings.ToLower(contentType), "markdown") {
		if snippet := extractMarkdownSnippet(content, limit); snippet != "" {
			return snippet
		}
	}
	if snippet := extractHTMLSnippet(content, limit); snippet != "" {
		return snippet
	}
	return extractMarkdownSnippet(content, limit)
}

func extractMarkdownTitle(content string) string {
	if strings.TrimSpace(content) == "" {
		return ""
	}
	if strings.HasPrefix(strings.TrimSpace(content), "---") {
		matches := markdownFrontMatterTitlePattern.FindStringSubmatch(content)
		if len(matches) >= 2 {
			return compactPlainText(matches[1], 120)
		}
	}
	matches := markdownHeadingPattern.FindStringSubmatch(content)
	if len(matches) >= 2 {
		return compactPlainText(matches[1], 120)
	}
	return ""
}

func extractMarkdownSnippet(content string, limit int) string {
	if strings.TrimSpace(content) == "" {
		return ""
	}
	normalized := markdownCodeFencePattern.ReplaceAllString(content, " ")
	normalized = markdownImagePattern.ReplaceAllString(normalized, "$1")
	normalized = markdownLinkPattern.ReplaceAllString(normalized, "$1")
	normalized = markdownHeadingMarkerPattern.ReplaceAllString(normalized, "")
	normalized = markdownListMarkerPattern.ReplaceAllString(normalized, "")
	normalized = markdownBlockQuotePattern.ReplaceAllString(normalized, "")
	normalized = markdownTableSepPattern.ReplaceAllString(normalized, " ")
	normalized = strings.NewReplacer("*", " ", "_", " ", "`", " ", "~", " ").Replace(normalized)
	return compactPlainText(normalized, limit)
}

func compactPlainText(value string, limit int) string {
	if limit <= 0 {
		limit = 320
	}
	text := html.UnescapeString(value)
	text = htmlSpacePattern.ReplaceAllString(strings.TrimSpace(text), " ")
	if len(text) > limit {
		return strings.TrimSpace(text[:limit]) + "..."
	}
	return text
}

func resolveWebSearchProviderMap(config map[string]any, provider string) map[string]any {
	if provider == "" {
		return nil
	}
	return getNestedMap(config, "web", "search", "providers", provider)
}

func resolveWebSearchProviderString(config map[string]any, provider string, key string) string {
	if key == "" {
		return ""
	}
	providerConfig := resolveWebSearchProviderMap(config, provider)
	if providerConfig == nil {
		return ""
	}
	value, ok := providerConfig[key]
	if !ok {
		return ""
	}
	if str, ok := value.(string); ok {
		return strings.TrimSpace(str)
	}
	return ""
}

func resolveWebSearchInt(config map[string]any, key string, fallback int) int {
	value, ok := getNestedInt(config, "web", "search", key)
	if !ok || value <= 0 {
		return fallback
	}
	return value
}

func normalizeWebSearchCacheKey(provider string, query string, count int, country string, searchLang string, uiLang string, freshness string) string {
	parts := []string{
		strings.ToLower(strings.TrimSpace(provider)),
		strings.ToLower(strings.TrimSpace(query)),
		strings.ToLower(strings.TrimSpace(country)),
		strings.ToLower(strings.TrimSpace(searchLang)),
		strings.ToLower(strings.TrimSpace(uiLang)),
		strings.ToLower(strings.TrimSpace(freshness)),
		strconv.Itoa(count),
	}
	return strings.Join(parts, "|")
}

func readWebSearchCache(key string) (webSearchResponse, bool) {
	webSearchCache.mu.RLock()
	entry, ok := webSearchCache.entries[key]
	webSearchCache.mu.RUnlock()
	if !ok {
		return webSearchResponse{}, false
	}
	if time.Now().After(entry.expiresAt) {
		webSearchCache.mu.Lock()
		delete(webSearchCache.entries, key)
		webSearchCache.mu.Unlock()
		return webSearchResponse{}, false
	}
	return entry.value, true
}

func writeWebSearchCache(key string, value webSearchResponse, ttl time.Duration) {
	if ttl <= 0 {
		return
	}
	webSearchCache.mu.Lock()
	webSearchCache.entries[key] = webSearchCacheEntry{
		value:     value,
		expiresAt: time.Now().Add(ttl),
	}
	webSearchCache.mu.Unlock()
}

func runBraveSearch(ctx context.Context, config map[string]any, query string, count int, country string, searchLang string, uiLang string, freshness string, timeoutSeconds int) (webSearchResponse, error) {
	apiKey := resolveWebSearchProviderString(config, "brave", "apiKey")
	if apiKey == "" {
		apiKey = getNestedString(config, "web", "search", "brave", "apiKey")
	}
	if apiKey == "" {
		apiKey = getNestedString(config, "web", "search", "apiKey")
	}
	if apiKey == "" {
		apiKey = strings.TrimSpace(os.Getenv("BRAVE_API_KEY"))
	}
	if apiKey == "" {
		return webSearchResponse{}, errors.New("web_search needs a Brave API key")
	}
	endpoint := resolveWebSearchProviderString(config, "brave", "apiBaseUrl")
	if endpoint == "" {
		endpoint = getNestedString(config, "web", "search", "brave", "baseUrl")
	}
	if endpoint == "" {
		endpoint = getNestedString(config, "web", "search", "baseUrl")
	}
	if endpoint == "" {
		endpoint = braveSearchEndpoint
	}
	values := url.Values{}
	values.Set("q", query)
	if count > 0 {
		values.Set("count", strconv.Itoa(count))
	}
	if country != "" {
		values.Set("country", country)
	}
	if searchLang != "" {
		values.Set("search_lang", searchLang)
	}
	if uiLang != "" {
		values.Set("ui_lang", uiLang)
	}
	if freshness != "" {
		values.Set("freshness", freshness)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"?"+values.Encode(), nil)
	if err != nil {
		return webSearchResponse{}, err
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("X-Subscription-Token", apiKey)
	timeout := time.Duration(timeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = time.Duration(defaultWebSearchTimeoutSeconds) * time.Second
	}
	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(request)
	if err != nil {
		return webSearchResponse{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return webSearchResponse{}, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusBadRequest {
		message := strings.TrimSpace(string(body))
		if message == "" {
			message = resp.Status
		}
		return webSearchResponse{}, fmt.Errorf("HTTP %d: %s", resp.StatusCode, message)
	}
	var parsed braveSearchResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return webSearchResponse{}, err
	}
	results := make([]webSearchResult, 0, len(parsed.Web.Results))
	for _, item := range parsed.Web.Results {
		results = append(results, webSearchResult{
			Title:       strings.TrimSpace(item.Title),
			URL:         strings.TrimSpace(item.URL),
			Description: strings.TrimSpace(item.Description),
			Age:         strings.TrimSpace(item.Age),
		})
	}
	if count > 0 && len(results) > count {
		results = results[:count]
	}
	return webSearchResponse{
		Query:    query,
		Provider: "brave",
		Results:  results,
	}, nil
}

func runTavilySearch(ctx context.Context, config map[string]any, payload toolArgs, query string, count int, timeoutSeconds int) (webSearchResponse, error) {
	apiKey := resolveWebSearchProviderString(config, "tavily", "apiKey")
	if apiKey == "" {
		apiKey = getNestedString(config, "web", "search", "tavily", "apiKey")
	}
	if apiKey == "" {
		apiKey = getNestedString(config, "web", "search", "apiKey")
	}
	if apiKey == "" {
		apiKey = strings.TrimSpace(os.Getenv("TAVILY_API_KEY"))
	}
	if apiKey == "" {
		return webSearchResponse{}, errors.New("web_search needs a Tavily API key")
	}
	endpoint := resolveWebSearchProviderString(config, "tavily", "apiBaseUrl")
	if endpoint == "" {
		endpoint = getNestedString(config, "web", "search", "tavily", "baseUrl")
	}
	if endpoint == "" {
		endpoint = getNestedString(config, "web", "search", "baseUrl")
	}
	if endpoint == "" {
		endpoint = tavilySearchEndpoint
	}

	requestBody := map[string]any{
		"query": query,
	}
	if count > 0 {
		requestBody["max_results"] = count
	}
	if value := getStringArg(payload, "search_depth", "searchDepth"); value != "" {
		requestBody["search_depth"] = value
	}
	if value := getStringArg(payload, "topic"); value != "" {
		requestBody["topic"] = value
	}
	if value := getStringArg(payload, "time_range", "timeRange"); value != "" {
		requestBody["time_range"] = value
	}
	if value := getStringArg(payload, "start_date", "startDate"); value != "" {
		requestBody["start_date"] = value
	}
	if value := getStringArg(payload, "end_date", "endDate"); value != "" {
		requestBody["end_date"] = value
	}
	if value := getStringArg(payload, "country"); value != "" {
		requestBody["country"] = value
	}
	if value := getStringArg(payload, "include_answer", "includeAnswer"); value != "" {
		requestBody["include_answer"] = value
	} else if includeAnswer, ok := getBoolArg(payload, "include_answer", "includeAnswer"); ok {
		requestBody["include_answer"] = includeAnswer
	}
	if value := getStringArg(payload, "include_raw_content", "includeRawContent"); value != "" {
		requestBody["include_raw_content"] = value
	} else if includeRaw, ok := getBoolArg(payload, "include_raw_content", "includeRawContent"); ok {
		requestBody["include_raw_content"] = includeRaw
	}
	if includeImages, ok := getBoolArg(payload, "include_images", "includeImages"); ok {
		requestBody["include_images"] = includeImages
	}
	if includeDescriptions, ok := getBoolArg(payload, "include_image_descriptions", "includeImageDescriptions"); ok {
		requestBody["include_image_descriptions"] = includeDescriptions
	}
	if includeFavicon, ok := getBoolArg(payload, "include_favicon", "includeFavicon"); ok {
		requestBody["include_favicon"] = includeFavicon
	}
	if autoParams, ok := getBoolArg(payload, "auto_parameters", "autoParameters"); ok {
		requestBody["auto_parameters"] = autoParams
	}
	if chunks, ok := getIntArg(payload, "chunks_per_source", "chunksPerSource"); ok && chunks > 0 {
		requestBody["chunks_per_source"] = chunks
	}
	if includeDomains := getStringSliceArg(payload, "include_domains", "includeDomains"); len(includeDomains) > 0 {
		requestBody["include_domains"] = includeDomains
	}
	if excludeDomains := getStringSliceArg(payload, "exclude_domains", "excludeDomains"); len(excludeDomains) > 0 {
		requestBody["exclude_domains"] = excludeDomains
	}

	encoded, err := json.Marshal(requestBody)
	if err != nil {
		return webSearchResponse{}, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(encoded))
	if err != nil {
		return webSearchResponse{}, err
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+apiKey)

	timeout := time.Duration(timeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = time.Duration(defaultWebSearchTimeoutSeconds) * time.Second
	}
	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(request)
	if err != nil {
		return webSearchResponse{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return webSearchResponse{}, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusBadRequest {
		message := strings.TrimSpace(string(body))
		if message == "" {
			message = resp.Status
		}
		return webSearchResponse{}, fmt.Errorf("HTTP %d: %s", resp.StatusCode, message)
	}
	var parsed tavilySearchResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return webSearchResponse{}, err
	}
	results := make([]webSearchResult, 0, len(parsed.Results))
	for _, item := range parsed.Results {
		description := strings.TrimSpace(item.Content)
		if description == "" {
			if raw, ok := item.RawContent.(string); ok {
				description = strings.TrimSpace(raw)
			}
		}
		results = append(results, webSearchResult{
			Title:       strings.TrimSpace(item.Title),
			URL:         strings.TrimSpace(item.URL),
			Description: description,
		})
	}
	if count > 0 && len(results) > count {
		results = results[:count]
	}
	return webSearchResponse{
		Query:    query,
		Provider: "tavily",
		Results:  results,
	}, nil
}

func isTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	lower := strings.ToLower(strings.TrimSpace(err.Error()))
	return strings.Contains(lower, "timeout") || strings.Contains(lower, "timed out")
}

func trimToMaxChars(value string, max int) string {
	trimmed := strings.TrimSpace(value)
	if max <= 0 {
		return trimmed
	}
	runes := []rune(trimmed)
	if len(runes) <= max {
		return trimmed
	}
	return string(runes[:max])
}

func normalizeHTTPHeaderValues(header http.Header) map[string]string {
	result := make(map[string]string, len(header))
	for key, values := range header {
		trimmedKey := strings.ToLower(strings.TrimSpace(key))
		if trimmedKey == "" {
			continue
		}
		result[trimmedKey] = strings.TrimSpace(strings.Join(values, ", "))
	}
	return result
}

func normalizeHTTPHeaderMap(headers map[string]string) map[string]string {
	result := make(map[string]string, len(headers))
	for key, value := range headers {
		trimmedKey := strings.ToLower(strings.TrimSpace(key))
		if trimmedKey == "" {
			continue
		}
		result[trimmedKey] = strings.TrimSpace(value)
	}
	return result
}

func resolvePlaywrightReadySelector(finalURL string, targetURL string) string {
	host := extractHostname(finalURL)
	if host == "" {
		host = extractHostname(targetURL)
	}
	switch {
	case hostMatchesDomain(host, "google.com"):
		return "#search div.g, #search a h3"
	case hostMatchesDomain(host, "xiaohongshu.com"):
		return ".note-item, .search-result, section"
	default:
		return "main, article, body"
	}
}

func waitForPlaywrightReady(page playwright.Page, selector string, timeoutMs float64) error {
	if strings.TrimSpace(selector) == "" {
		return nil
	}
	_, err := page.WaitForSelector(selector, playwright.PageWaitForSelectorOptions{
		Timeout: playwright.Float(timeoutMs),
	})
	return err
}

func contentWithTimeout(ctx context.Context, page playwright.Page, timeout time.Duration) (string, error) {
	type result struct {
		content string
		err     error
	}
	done := make(chan result, 1)
	go func() {
		content, err := page.Content()
		done <- result{content: content, err: err}
	}()

	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case <-timer.C:
		return "", errors.New("playwright extract timeout")
	case output := <-done:
		return output.content, output.err
	}
}

func extractHostname(rawURL string) string {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(parsed.Hostname()))
}
