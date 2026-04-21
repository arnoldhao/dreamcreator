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
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"github.com/hashicorp/go-retryablehttp"

	"dreamcreator/internal/application/browsercdp"
	connectorsdto "dreamcreator/internal/application/connectors/dto"
	appcookies "dreamcreator/internal/application/cookies"
	"dreamcreator/internal/application/sitepolicy"
)

const webFetchTypeCDP = "cdp"

const webFetchTypeBuiltin = webFetchTypeCDP

const (
	defaultWebFetchType              = webFetchTypeCDP
	defaultWebFetchTimeoutSeconds    = 20
	defaultWebFetchMaxChars          = 50000
	defaultWebFetchMaxBodyBytes      = 2 << 20
	defaultWebSearchTimeoutSeconds   = 30
	defaultWebSearchCacheTtlMinutes  = 15
	maxWebSearchCacheEntries         = 256
	defaultWebSearchCount            = 5
	maxWebSearchCount                = 10
	defaultWebSearchType             = "api"
	defaultWebFetchContentSignalMain = "main_heuristic"
)

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

var defaultExtractorSelectors = []string{
	"article",
	"main",
	"[role=main]",
	".article",
	".post-content",
	".entry-content",
	".content",
}

var defaultRemoveSelectors = []string{
	"script",
	"style",
	"noscript",
	"svg",
	"canvas",
	"iframe",
	"form",
	"nav",
	"aside",
	"footer",
	"header",
	"[role=navigation]",
	".sidebar",
	".comments",
	".recommend",
	".recommendations",
	".related",
	".advertisement",
	".ads",
}

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
		Title   string `json:"title"`
		URL     string `json:"url"`
		Content string `json:"content"`
	} `json:"results"`
}

type webSearchCacheEntry struct {
	value     webSearchResponse
	expiresAt time.Time
	storedAt  time.Time
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
	TimeoutSeconds int
	MaxChars       int
	MaxBodyBytes   int
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

type extractorResult struct {
	Content       string
	ContentSignal string
}

func runWebFetchTool(settings SettingsReader, connectors ConnectorsReader) func(ctx context.Context, args string) (string, error) {
	return func(ctx context.Context, args string) (string, error) {
		payload, err := parseToolArgs(args)
		if err != nil {
			return "", err
		}
		targetURL := getStringArg(payload, "url", "href")
		if targetURL == "" {
			return "", errors.New("url is required")
		}

		config := resolveToolsConfig(ctx, settings)
		if enabled, ok := resolveWebFetchConfigBool(config, "enabled"); ok && !enabled {
			return "", errors.New("web_fetch disabled")
		}

		method := strings.ToUpper(strings.TrimSpace(getStringArg(payload, "method")))
		if method == "" {
			method = http.MethodGet
		}
		if method != http.MethodGet {
			result := buildWebFetchToolResult(targetURL, webFetchResponse{}, errors.New("web_fetch only supports GET"))
			return marshalResult(result), nil
		}

		options := resolveWebFetchOptions(payload, config, defaultWebFetchTimeoutSeconds)
		cookies, err := browsercdp.ResolveConnectorCookiesForURL(ctx, connectors, targetURL)
		if err != nil {
			return "", err
		}
		preferredBrowser := resolveWebFetchPreferredBrowser(config)
		headless := resolveWebFetchHeadless(config)

		response, err := fetchWithCDP(ctx, targetURL, cookies, options, preferredBrowser, headless)
		result := buildWebFetchToolResult(targetURL, response, err)
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

func fetchWithCDP(
	ctx context.Context,
	targetURL string,
	cookies []appcookies.Record,
	options webFetchOptions,
	preferredBrowser string,
	headless bool,
) (webFetchResponse, error) {
	runtimeCtx, cancel := context.WithTimeout(ctx, time.Duration(options.TimeoutSeconds)*time.Second)
	defer cancel()

	runtime, err := browsercdp.Start(runtimeCtx, browsercdp.LaunchOptions{
		PreferredBrowser: preferredBrowser,
		Headless:         headless,
	})
	if err != nil {
		return webFetchResponse{}, err
	}
	defer runtime.Stop()

	tabCtx, tabCancel, _, err := browsercdp.AttachOrCreatePageTarget(runtime, 5*time.Second)
	if err != nil {
		return webFetchResponse{}, err
	}
	defer tabCancel()
	var navResponse *network.Response
	var finalURL string
	var contentType string
	var htmlContent string

	tasks := chromedp.Tasks{
		network.Enable(),
	}
	if len(cookies) > 0 {
		tasks = append(tasks, chromedp.ActionFunc(func(ctx context.Context) error {
			return browsercdp.SetCookies(ctx, targetURL, cookies)
		}))
	}
	if err := chromedp.Run(tabCtx, tasks); err != nil {
		return webFetchResponse{}, err
	}

	navResponse, err = chromedp.RunResponse(tabCtx, chromedp.Navigate(strings.TrimSpace(targetURL)))
	if err != nil {
		return webFetchResponse{}, err
	}
	if navResponse != nil {
		contentType = strings.TrimSpace(navResponse.MimeType)
	}
	if selector := sitepolicy.ReadySelectorForURL(targetURL); selector != "" {
		waitCtx, waitCancel := context.WithTimeout(tabCtx, time.Duration(options.TimeoutSeconds)*time.Second)
		_ = chromedp.Run(waitCtx, chromedp.WaitVisible(selector, chromedp.ByQuery))
		waitCancel()
	}
	if err := chromedp.Run(tabCtx,
		chromedp.Location(&finalURL),
		chromedp.OuterHTML("html", &htmlContent, chromedp.ByQuery),
	); err != nil {
		return webFetchResponse{}, err
	}
	if finalURL == "" {
		finalURL = targetURL
	}
	extracted := extractMainContent(htmlContent, finalURL)
	content := strings.TrimSpace(extracted.Content)
	truncated := false
	if options.MaxBodyBytes > 0 && len(content) > options.MaxBodyBytes {
		content = content[:options.MaxBodyBytes]
		truncated = true
	}
	if options.MaxChars > 0 && len(content) > options.MaxChars {
		content = content[:options.MaxChars]
		truncated = true
	}

	headersMap := map[string]string{}
	if navResponse != nil {
		for key, value := range navResponse.Headers {
			headersMap[key] = fmt.Sprint(value)
		}
	}

	return webFetchResponse{
		URL:            targetURL,
		FinalURL:       finalURL,
		Status:         statusCodeFromResponse(navResponse),
		Headers:        headersMap,
		ContentType:    contentType,
		Content:        content,
		MarkdownTokens: estimateMarkdownTokens(content),
		ContentSignal:  extracted.ContentSignal,
		Truncated:      truncated,
	}, nil
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
	case "external_tools":
		return webSearchResponse{
			Status:             webStatusError,
			Retryable:          false,
			NextAction:         nextActionUseOtherToolsOrSkills,
			Message:            "web_search_external_tools_not_configured",
			Provider:           "web_search",
			Quality:            webQualityEmpty,
			WebSearchAvailable: false,
			Query:              query,
		}
	default:
		result, err := runWebSearchByAPI(ctx, payload, config, query, count)
		if err != nil {
			return webSearchResponse{
				Status:             webStatusError,
				Retryable:          false,
				NextAction:         nextActionUseOtherToolsOrSkills,
				Message:            err.Error(),
				Provider:           "web_search",
				Quality:            webQualityEmpty,
				WebSearchAvailable: true,
				Query:              query,
			}
		}
		return result
	}
}

func runWebSearchByAPI(ctx context.Context, payload toolArgs, config map[string]any, query string, count int) (webSearchResponse, error) {
	provider := strings.ToLower(strings.TrimSpace(getNestedString(config, "web", "search", "provider")))
	if provider == "" {
		provider = "brave"
	}
	cacheKey := fmt.Sprintf("%s:%s:%d", provider, strings.TrimSpace(query), count)
	if cached, ok := loadWebSearchCache(cacheKey); ok {
		cached.Cached = true
		return cached, nil
	}

	var (
		results []webSearchResult
		err     error
	)
	switch provider {
	case "tavily":
		results, err = runTavilySearch(ctx, payload, config, query, count)
	default:
		results, err = runBraveSearch(ctx, payload, config, query, count)
	}
	if err != nil {
		return webSearchResponse{}, err
	}
	response := webSearchResponse{
		Status:             webStatusOK,
		Retryable:          false,
		NextAction:         nextActionContinue,
		Message:            "ok",
		Provider:           provider,
		Quality:            webQualitySufficient,
		WebSearchAvailable: true,
		Query:              query,
		Results:            results,
	}
	storeWebSearchCache(cacheKey, response, resolveWebSearchCacheTTL(config))
	return response, nil
}

func runBraveSearch(ctx context.Context, payload toolArgs, config map[string]any, query string, count int) ([]webSearchResult, error) {
	apiKey := strings.TrimSpace(resolveWebSearchProviderAPIKey(config, "brave"))
	if apiKey == "" {
		return nil, fmt.Errorf("Brave API key is missing")
	}
	reqURL, err := url.Parse(braveSearchEndpoint)
	if err != nil {
		return nil, err
	}
	values := reqURL.Query()
	values.Set("q", query)
	values.Set("count", strconv.Itoa(count))
	if value := getStringArg(payload, "country"); value != "" {
		values.Set("country", value)
	}
	if value := getStringArg(payload, "search_lang", "searchLang"); value != "" {
		values.Set("search_lang", value)
	}
	if value := getStringArg(payload, "ui_lang", "uiLang"); value != "" {
		values.Set("ui_lang", value)
	}
	if value := getStringArg(payload, "freshness"); value != "" {
		values.Set("freshness", value)
	}
	reqURL.RawQuery = values.Encode()

	request, err := retryablehttp.NewRequestWithContext(ctx, http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("X-Subscription-Token", apiKey)

	client := retryablehttp.NewClient()
	client.RetryMax = 1
	client.HTTPClient.Timeout = time.Duration(resolveWebSearchTimeoutSeconds(payload, config)) * time.Second
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 2048))
		return nil, fmt.Errorf("Brave search failed: %s", strings.TrimSpace(string(body)))
	}
	var payloadJSON braveSearchResponse
	if err := json.NewDecoder(response.Body).Decode(&payloadJSON); err != nil {
		return nil, err
	}
	results := make([]webSearchResult, 0, len(payloadJSON.Web.Results))
	for _, item := range payloadJSON.Web.Results {
		results = append(results, webSearchResult{
			Title:       strings.TrimSpace(item.Title),
			URL:         strings.TrimSpace(item.URL),
			Description: strings.TrimSpace(item.Description),
			Age:         strings.TrimSpace(item.Age),
		})
	}
	return results, nil
}

func runTavilySearch(ctx context.Context, payload toolArgs, config map[string]any, query string, count int) ([]webSearchResult, error) {
	apiKey := strings.TrimSpace(resolveWebSearchProviderAPIKey(config, "tavily"))
	if apiKey == "" {
		return nil, fmt.Errorf("Tavily API key is missing")
	}
	requestBody := map[string]any{
		"api_key":     apiKey,
		"query":       query,
		"max_results": count,
	}
	if value := getStringArg(payload, "country"); value != "" {
		requestBody["country"] = value
	}
	if value := getStringArg(payload, "search_depth", "searchDepth"); value != "" {
		requestBody["search_depth"] = value
	}
	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}
	request, err := retryablehttp.NewRequestWithContext(ctx, http.MethodPost, tavilySearchEndpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/json")

	client := retryablehttp.NewClient()
	client.RetryMax = 1
	client.HTTPClient.Timeout = time.Duration(resolveWebSearchTimeoutSeconds(payload, config)) * time.Second
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 2048))
		return nil, fmt.Errorf("Tavily search failed: %s", strings.TrimSpace(string(body)))
	}
	var payloadJSON tavilySearchResponse
	if err := json.NewDecoder(response.Body).Decode(&payloadJSON); err != nil {
		return nil, err
	}
	results := make([]webSearchResult, 0, len(payloadJSON.Results))
	for _, item := range payloadJSON.Results {
		description := strings.TrimSpace(item.Content)
		results = append(results, webSearchResult{
			Title:       strings.TrimSpace(item.Title),
			URL:         strings.TrimSpace(item.URL),
			Description: description,
		})
	}
	return results, nil
}

func loadWebSearchCache(key string) (webSearchResponse, bool) {
	now := time.Now()
	webSearchCache.mu.RLock()
	entry, ok := webSearchCache.entries[key]
	webSearchCache.mu.RUnlock()
	if !ok {
		return webSearchResponse{}, false
	}
	if now.After(entry.expiresAt) {
		webSearchCache.mu.Lock()
		if current, exists := webSearchCache.entries[key]; exists && now.After(current.expiresAt) {
			delete(webSearchCache.entries, key)
		}
		webSearchCache.mu.Unlock()
		return webSearchResponse{}, false
	}
	return entry.value, true
}

func storeWebSearchCache(key string, value webSearchResponse, ttl time.Duration) {
	if ttl <= 0 {
		return
	}
	now := time.Now()
	webSearchCache.mu.Lock()
	pruneExpiredWebSearchCacheLocked(now)
	webSearchCache.entries[key] = webSearchCacheEntry{
		value:     value,
		expiresAt: now.Add(ttl),
		storedAt:  now,
	}
	if len(webSearchCache.entries) > maxWebSearchCacheEntries {
		evictOldestWebSearchCacheEntriesLocked(len(webSearchCache.entries) - maxWebSearchCacheEntries)
	}
	webSearchCache.mu.Unlock()
}

func pruneExpiredWebSearchCacheLocked(now time.Time) {
	for key, entry := range webSearchCache.entries {
		if now.After(entry.expiresAt) {
			delete(webSearchCache.entries, key)
		}
	}
}

func evictOldestWebSearchCacheEntriesLocked(excess int) {
	for excess > 0 {
		oldestKey := ""
		oldestAt := time.Time{}
		for key, entry := range webSearchCache.entries {
			if oldestKey == "" || entry.storedAt.Before(oldestAt) {
				oldestKey = key
				oldestAt = entry.storedAt
			}
		}
		if oldestKey == "" {
			return
		}
		delete(webSearchCache.entries, oldestKey)
		excess--
	}
}

func buildWebFetchToolResult(rawURL string, response webFetchResponse, err error) webFetchResult {
	if err != nil {
		return webFetchResult{
			Status:             webStatusError,
			Retryable:          false,
			NextAction:         nextActionUseOtherToolsOrSkills,
			Message:            err.Error(),
			Provider:           "web_fetch",
			Quality:            webQualityEmpty,
			WebSearchAvailable: true,
			URL:                rawURL,
			FinalURL:           response.FinalURL,
			HTTPStatus:         response.Status,
			ContentType:        response.ContentType,
			Content:            response.Content,
			MarkdownTokens:     response.MarkdownTokens,
			ContentSignal:      response.ContentSignal,
			Truncated:          response.Truncated,
			Data: map[string]any{
				"timeoutStage": response.TimeoutStage,
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
		URL:                rawURL,
		FinalURL:           response.FinalURL,
		HTTPStatus:         response.Status,
		ContentType:        response.ContentType,
		Content:            response.Content,
		MarkdownTokens:     response.MarkdownTokens,
		ContentSignal:      response.ContentSignal,
		Truncated:          response.Truncated,
		Data: map[string]any{
			"extractor":     response.ContentSignal,
			"browserSource": webFetchTypeCDP,
			"timeoutStage":  response.TimeoutStage,
		},
	}
}

func resolveWebFetchType(payload toolArgs, config map[string]any) (string, error) {
	if raw := strings.TrimSpace(getStringArg(payload, "type", "mode")); raw != "" {
		normalized := normalizeWebFetchType(raw)
		if normalized == "" {
			return "", fmt.Errorf("unsupported web_fetch type: %s", raw)
		}
		return normalized, nil
	}
	if fetchConfig := getNestedMap(config, "web_fetch"); fetchConfig != nil {
		if raw := strings.TrimSpace(getStringArg(toolArgs(fetchConfig), "type", "mode")); raw != "" {
			normalized := normalizeWebFetchType(raw)
			if normalized == "" {
				return "", fmt.Errorf("unsupported web_fetch type: %s", raw)
			}
			return normalized, nil
		}
	}
	return defaultWebFetchType, nil
}

func normalizeWebFetchType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "cdp", "chrome", "chromium", "browser", "builtin":
		return webFetchTypeCDP
	default:
		return ""
	}
}

func resolveWebFetchOptions(payload toolArgs, config map[string]any, fallbackTimeoutSeconds int) webFetchOptions {
	fetchConfig := getNestedMap(config, "web_fetch")
	timeoutSeconds := fallbackTimeoutSeconds
	if value, ok := getIntArg(toolArgs(fetchConfig), "timeoutSeconds"); ok && value > 0 {
		timeoutSeconds = value
	}
	if value, ok := getIntArg(payload, "timeoutSeconds"); ok && value > 0 {
		timeoutSeconds = value
	}
	maxChars := defaultWebFetchMaxChars
	if value, ok := getIntArg(toolArgs(fetchConfig), "maxChars"); ok && value > 0 {
		maxChars = value
	}
	if value, ok := getIntArg(payload, "maxChars"); ok && value > 0 {
		maxChars = value
	}
	maxBodyBytes := defaultWebFetchMaxBodyBytes
	if value, ok := getIntArg(toolArgs(fetchConfig), "maxBodyBytes"); ok && value > 0 {
		maxBodyBytes = value
	}
	if value, ok := getIntArg(payload, "maxBodyBytes"); ok && value > 0 {
		maxBodyBytes = value
	}

	return webFetchOptions{
		TimeoutSeconds: timeoutSeconds,
		MaxChars:       maxChars,
		MaxBodyBytes:   maxBodyBytes,
	}
}

func connectorTypeForURL(rawURL string) string {
	return browsercdp.ConnectorTypeForURL(rawURL)
}

func resolveWebFetchConfigBool(config map[string]any, key string) (bool, bool) {
	fetchConfig := getNestedMap(config, "web_fetch")
	return getBoolArg(toolArgs(fetchConfig), key)
}

func resolveWebFetchPreferredBrowser(config map[string]any) string {
	fetchConfig := getNestedMap(config, "web_fetch")
	if value := strings.TrimSpace(getStringArg(toolArgs(fetchConfig), "preferredBrowser")); value != "" {
		return value
	}
	browserConfig := getNestedMap(config, "browser")
	return strings.TrimSpace(getStringArg(toolArgs(browserConfig), "preferredBrowser"))
}

func resolveWebFetchHeadless(config map[string]any) bool {
	fetchConfig := getNestedMap(config, "web_fetch")
	if value, ok := getBoolArg(toolArgs(fetchConfig), "headless"); ok {
		return value
	}
	return true
}

func extractMainContent(rawHTML string, pageURL string) extractorResult {
	policy, _ := sitepolicy.ForURL(pageURL)
	reader := strings.NewReader(rawHTML)
	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return extractorResult{
			Content:       compactText(rawHTML),
			ContentSignal: "body_fallback",
		}
	}
	for _, selector := range append(defaultRemoveSelectors, policy.RemoveSelectors...) {
		doc.Find(selector).Each(func(_ int, selection *goquery.Selection) {
			selection.Remove()
		})
	}

	root := pickBestContentRoot(doc, policy)
	if root == nil || root.Length() == 0 {
		body := strings.TrimSpace(doc.Find("body").First().Text())
		return extractorResult{
			Content:       compactMarkdown(body),
			ContentSignal: "body_fallback",
		}
	}

	htmlFragment, err := root.Html()
	if err != nil {
		text := compactMarkdown(root.Text())
		if text == "" {
			return extractorResult{ContentSignal: "body_fallback"}
		}
		return extractorResult{
			Content:       text,
			ContentSignal: defaultWebFetchContentSignalMain,
		}
	}

	converter := md.NewConverter("", true, nil)
	markdown, err := converter.ConvertString(htmlFragment)
	if err != nil {
		markdown = compactMarkdown(root.Text())
	}
	markdown = compactMarkdown(markdown)
	if markdown == "" {
		markdown = compactMarkdown(root.Text())
	}
	if markdown == "" {
		return extractorResult{
			Content:       compactMarkdown(doc.Text()),
			ContentSignal: "body_fallback",
		}
	}
	return extractorResult{
		Content:       markdown,
		ContentSignal: resolveContentSignal(policy, root),
	}
}

func pickBestContentRoot(doc *goquery.Document, policy sitepolicy.Policy) *goquery.Selection {
	for _, selector := range policy.ExtractorSelectors {
		selection := doc.Find(selector).First()
		if selection.Length() > 0 && len(compactText(selection.Text())) >= 160 {
			return selection
		}
	}
	for _, selector := range defaultExtractorSelectors {
		selection := doc.Find(selector).First()
		if selection.Length() > 0 && len(compactText(selection.Text())) >= 160 {
			return selection
		}
	}

	var (
		best      *goquery.Selection
		bestScore float64
	)
	doc.Find("article,main,section,div").Each(func(_ int, selection *goquery.Selection) {
		text := compactText(selection.Text())
		textLen := len(text)
		if textLen < 160 {
			return
		}
		linkTextLen := len(compactText(selection.Find("a").Text()))
		linkDensity := 0.0
		if textLen > 0 {
			linkDensity = float64(linkTextLen) / float64(textLen)
		}
		paragraphs := selection.Find("p").Length()
		score := float64(textLen) + float64(paragraphs*80) - (linkDensity * 800)
		if score > bestScore {
			bestScore = score
			best = selection
		}
	})
	if best != nil {
		return best
	}
	body := doc.Find("body").First()
	if body.Length() > 0 {
		return body
	}
	return nil
}

func resolveContentSignal(policy sitepolicy.Policy, root *goquery.Selection) string {
	if root == nil || root.Length() == 0 {
		return "body_fallback"
	}
	nodeName := goquery.NodeName(root)
	switch nodeName {
	case "article":
		return "article_readability"
	case "main":
		return "main_heuristic"
	default:
		if policy.Key != "" {
			return defaultWebFetchContentSignalMain
		}
		return defaultWebFetchContentSignalMain
	}
}

func compactMarkdown(input string) string {
	text := strings.ReplaceAll(input, "\r\n", "\n")
	text = markdownCodeFencePattern.ReplaceAllStringFunc(text, func(code string) string {
		return strings.TrimSpace(code)
	})
	lines := strings.Split(text, "\n")
	result := make([]string, 0, len(lines))
	previousBlank := false
	seen := map[string]struct{}{}
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if previousBlank {
				continue
			}
			previousBlank = true
			result = append(result, "")
			continue
		}
		previousBlank = false
		normalized := htmlSpacePattern.ReplaceAllString(trimmed, " ")
		if strings.HasPrefix(normalized, "Recommended") || strings.HasPrefix(normalized, "Related") {
			continue
		}
		if len(normalized) > 4000 {
			normalized = normalized[:4000]
		}
		key := strings.ToLower(normalized)
		if _, exists := seen[key]; exists && len(normalized) > 48 {
			continue
		}
		if len(normalized) > 48 {
			seen[key] = struct{}{}
		}
		result = append(result, normalized)
	}
	return strings.TrimSpace(strings.Join(result, "\n"))
}

func compactText(input string) string {
	text := html.UnescapeString(input)
	text = htmlSpacePattern.ReplaceAllString(strings.TrimSpace(text), " ")
	return text
}

func estimateMarkdownTokens(content string) int {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return 0
	}
	return (len(trimmed) + 3) / 4
}

func statusCodeFromResponse(response *network.Response) int {
	if response == nil {
		return 0
	}
	return int(response.Status)
}

func extractWebPageTitle(content string, contentType string) string {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return ""
	}
	if strings.Contains(strings.ToLower(contentType), "markdown") {
		if matches := markdownFrontMatterTitlePattern.FindStringSubmatch(trimmed); len(matches) > 1 {
			return strings.TrimSpace(matches[1])
		}
		if matches := markdownHeadingPattern.FindStringSubmatch(trimmed); len(matches) > 1 {
			return strings.TrimSpace(matches[1])
		}
		return ""
	}
	if matches := htmlTitlePattern.FindStringSubmatch(trimmed); len(matches) > 1 {
		return compactText(matches[1])
	}
	return ""
}

func extractWebPageSnippet(content string, contentType string, maxChars int) string {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return ""
	}
	var snippet string
	if strings.Contains(strings.ToLower(contentType), "markdown") {
		snippet = markdownCodeFencePattern.ReplaceAllString(trimmed, "")
		snippet = markdownImagePattern.ReplaceAllString(snippet, "$1")
		snippet = markdownLinkPattern.ReplaceAllString(snippet, "$1")
		snippet = markdownHeadingMarkerPattern.ReplaceAllString(snippet, "")
		snippet = markdownListMarkerPattern.ReplaceAllString(snippet, "")
		snippet = markdownBlockQuotePattern.ReplaceAllString(snippet, "")
		snippet = markdownTableSepPattern.ReplaceAllString(snippet, "")
	} else {
		snippet = htmlScriptStylePattern.ReplaceAllString(trimmed, "")
		snippet = htmlTagPattern.ReplaceAllString(snippet, " ")
	}
	snippet = compactText(snippet)
	if maxChars > 0 && len(snippet) > maxChars {
		return snippet[:maxChars]
	}
	return snippet
}

func resolveWebSearchType(config map[string]any) string {
	value := strings.ToLower(strings.TrimSpace(getNestedString(config, "web", "search", "type")))
	switch value {
	case "external_tools":
		return "external_tools"
	case "api":
		return "api"
	default:
		return defaultWebSearchType
	}
}

func resolveWebSearchCount(payload toolArgs, config map[string]any) int {
	if value, ok := getIntArg(payload, "count", "maxResults"); ok && value > 0 {
		if value > maxWebSearchCount {
			return maxWebSearchCount
		}
		return value
	}
	if value, ok := getIntArg(toolArgs(getNestedMap(config, "web", "search")), "maxResults"); ok && value > 0 {
		if value > maxWebSearchCount {
			return maxWebSearchCount
		}
		return value
	}
	return defaultWebSearchCount
}

func resolveWebSearchTimeoutSeconds(payload toolArgs, config map[string]any) int {
	if value, ok := getIntArg(payload, "timeoutSeconds"); ok && value > 0 {
		return value
	}
	if value, ok := getIntArg(toolArgs(getNestedMap(config, "web", "search")), "timeoutSeconds"); ok && value > 0 {
		return value
	}
	return defaultWebSearchTimeoutSeconds
}

func resolveWebSearchCacheTTL(config map[string]any) time.Duration {
	if value, ok := getIntArg(toolArgs(getNestedMap(config, "web", "search")), "cacheTtlMinutes"); ok && value > 0 {
		return time.Duration(value) * time.Minute
	}
	return time.Duration(defaultWebSearchCacheTtlMinutes) * time.Minute
}
