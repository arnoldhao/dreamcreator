package tools

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	settingsdto "dreamcreator/internal/application/settings/dto"
)

type settingsReaderStub struct {
	settings settingsdto.Settings
	err      error
}

func (stub *settingsReaderStub) GetSettings(context.Context) (settingsdto.Settings, error) {
	if stub.err != nil {
		return settingsdto.Settings{}, stub.err
	}
	return stub.settings, nil
}

func builtinWebFetchSettingsStub() *settingsReaderStub {
	return &settingsReaderStub{
		settings: settingsdto.Settings{
			Tools: map[string]any{
				"web_fetch": map[string]any{
					"type": "builtin",
				},
			},
		},
	}
}

func TestRunWebFetchTool_DefaultLLMHeaders(t *testing.T) {
	t.Parallel()

	var acceptHeader string
	var userAgentHeader string
	var acceptLanguageHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		acceptHeader = r.Header.Get("Accept")
		userAgentHeader = r.Header.Get("User-Agent")
		acceptLanguageHeader = r.Header.Get("Accept-Language")
		w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
		w.Header().Set("x-markdown-tokens", "256")
		w.Header().Set("content-signal", "ai-input=yes")
		_, _ = w.Write([]byte("# Hello\n\nworld"))
	}))
	defer server.Close()

	handler := runWebFetchTool(builtinWebFetchSettingsStub(), nil)
	output, err := handler(context.Background(), `{"url":"`+server.URL+`"}`)
	if err != nil {
		t.Fatalf("run web_fetch: %v", err)
	}

	if !strings.Contains(acceptHeader, "text/markdown") {
		t.Fatalf("expected markdown accept header, got %q", acceptHeader)
	}
	if !strings.Contains(userAgentHeader, "Version/17.0") || !strings.Contains(userAgentHeader, "Safari/605.1.15") {
		t.Fatalf("expected default user-agent, got %q", userAgentHeader)
	}
	if acceptLanguageHeader != "en-US,en;q=0.9" {
		t.Fatalf("expected default accept-language, got %q", acceptLanguageHeader)
	}

	var payload webFetchResult
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if payload.MarkdownTokens != 256 {
		t.Fatalf("unexpected markdownTokens: %d", payload.MarkdownTokens)
	}
	if payload.ContentSignal != "ai-input=yes" {
		t.Fatalf("unexpected content signal: %q", payload.ContentSignal)
	}
	if !strings.Contains(strings.ToLower(payload.ContentType), "markdown") {
		t.Fatalf("expected markdown content type, got %q", payload.ContentType)
	}
}

func TestRunWebFetchTool_HeaderSwitches(t *testing.T) {
	t.Parallel()

	var acceptHeader string
	var userAgentHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		acceptHeader = r.Header.Get("Accept")
		userAgentHeader = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("<html><title>ok</title></html>"))
	}))
	defer server.Close()

	handler := runWebFetchTool(builtinWebFetchSettingsStub(), nil)
	output, err := handler(context.Background(), `{"url":"`+server.URL+`","acceptMarkdown":false,"enableUserAgent":false}`)
	if err != nil {
		t.Fatalf("run web_fetch: %v", err)
	}
	if strings.Contains(strings.ToLower(acceptHeader), "text/markdown") {
		t.Fatalf("expected html accept header, got %q", acceptHeader)
	}
	if userAgentHeader != "" {
		t.Fatalf("expected empty user-agent when disabled, got %q", userAgentHeader)
	}

	var payload webFetchResult
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if payload.Status != webStatusOK {
		t.Fatalf("unexpected status: %q", payload.Status)
	}
	if payload.HTTPStatus != http.StatusOK {
		t.Fatalf("unexpected http status: %d", payload.HTTPStatus)
	}
}

func TestRunWebFetchTool_ReadsTopLevelWebFetchSettings(t *testing.T) {
	t.Parallel()

	var acceptHeader string
	var userAgentHeader string
	var acceptLanguageHeader string
	var customHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		acceptHeader = r.Header.Get("Accept")
		userAgentHeader = r.Header.Get("User-Agent")
		acceptLanguageHeader = r.Header.Get("Accept-Language")
		customHeader = r.Header.Get("X-Test")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	handler := runWebFetchTool(&settingsReaderStub{
		settings: settingsdto.Settings{
			Tools: map[string]any{
				"web_fetch": map[string]any{
					"type":            "builtin",
					"acceptMarkdown":  false,
					"enableUserAgent": false,
					"acceptLanguage":  "en-US,en;q=0.9",
					"headers": map[string]any{
						"X-Test": "enabled",
					},
				},
			},
		},
	}, nil)
	output, err := handler(context.Background(), `{"url":"`+server.URL+`"}`)
	if err != nil {
		t.Fatalf("run web_fetch: %v", err)
	}
	if strings.Contains(strings.ToLower(acceptHeader), "text/markdown") {
		t.Fatalf("expected html accept header from settings, got %q", acceptHeader)
	}
	if userAgentHeader != "" {
		t.Fatalf("expected empty user-agent from settings, got %q", userAgentHeader)
	}
	if acceptLanguageHeader != "en-US,en;q=0.9" {
		t.Fatalf("expected accept-language from settings, got %q", acceptLanguageHeader)
	}
	if customHeader != "enabled" {
		t.Fatalf("expected custom header from settings, got %q", customHeader)
	}

	var payload webFetchResult
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if payload.Status != webStatusOK {
		t.Fatalf("unexpected status: %q", payload.Status)
	}
	if payload.HTTPStatus != http.StatusOK {
		t.Fatalf("unexpected http status: %d", payload.HTTPStatus)
	}
}

func TestResolveWebFetchOptions_ReadsTopLevelWebFetchConfig(t *testing.T) {
	t.Parallel()

	options := resolveWebFetchOptions(toolArgs{}, map[string]any{
		"web_fetch": map[string]any{
			"timeoutSeconds":  21,
			"maxChars":        1234,
			"maxBodyBytes":    4321,
			"acceptMarkdown":  false,
			"enableUserAgent": false,
			"userAgent":       "top-level-agent",
			"headers": map[string]any{
				"X-One": "1",
			},
		},
	}, defaultWebFetchTimeoutSeconds)

	if options.TimeoutSeconds != 21 {
		t.Fatalf("expected timeout from web_fetch, got %d", options.TimeoutSeconds)
	}
	if options.MaxChars != 1234 {
		t.Fatalf("expected maxChars from web_fetch, got %d", options.MaxChars)
	}
	if options.MaxBodyBytes != 4321 {
		t.Fatalf("expected maxBodyBytes from web_fetch, got %d", options.MaxBodyBytes)
	}
	if options.MaxRedirects != defaultWebFetchMaxRedirects {
		t.Fatalf("expected default maxRedirects, got %d", options.MaxRedirects)
	}
	if options.RetryMax != defaultWebFetchRetryMax {
		t.Fatalf("expected default retryMax, got %d", options.RetryMax)
	}
	if options.AcceptMarkdown {
		t.Fatalf("expected acceptMarkdown=false from web_fetch")
	}
	if options.EnableUserAgent {
		t.Fatalf("expected enableUserAgent=false from web_fetch")
	}
	if options.UserAgent != "top-level-agent" {
		t.Fatalf("expected userAgent from web_fetch, got %q", options.UserAgent)
	}
	if options.Headers["X-One"] != "1" {
		t.Fatalf("expected headers from web_fetch, got %#v", options.Headers)
	}
}

func TestRunWebFetchTool_TruncatesLargeBodiesByMaxBodyBytes(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte(strings.Repeat("a", 512)))
	}))
	defer server.Close()

	handler := runWebFetchTool(builtinWebFetchSettingsStub(), nil)
	output, err := handler(context.Background(), `{"url":"`+server.URL+`","maxBodyBytes":64,"maxChars":200}`)
	if err != nil {
		t.Fatalf("run web_fetch: %v", err)
	}

	var payload webFetchResult
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if payload.Status != webStatusOK {
		t.Fatalf("unexpected status: %q", payload.Status)
	}
	if !payload.Truncated {
		t.Fatalf("expected payload to be truncated")
	}
	if len(payload.Content) != 64 {
		t.Fatalf("expected content to be capped at 64 bytes, got %d", len(payload.Content))
	}
	if payload.Content != strings.Repeat("a", 64) {
		t.Fatalf("unexpected truncated content length=%d", len(payload.Content))
	}
}

func TestExtractWebPageTitleAndSnippetFromMarkdown(t *testing.T) {
	t.Parallel()

	markdown := `---
title: Cloudflare Markdown for Agents
---

# Cloudflare Markdown for Agents

Markdown has quickly become the lingua franca for agents.`

	title := extractWebPageTitle(markdown, "text/markdown; charset=utf-8")
	if title != "Cloudflare Markdown for Agents" {
		t.Fatalf("unexpected markdown title: %q", title)
	}
	snippet := extractWebPageSnippet(markdown, "text/markdown", 80)
	if strings.TrimSpace(snippet) == "" || strings.Contains(snippet, "<") {
		t.Fatalf("unexpected markdown snippet: %q", snippet)
	}
}

func TestResolveWebFetchTypeDefaultsToBuiltin(t *testing.T) {
	t.Parallel()

	fetchType, err := resolveWebFetchType(toolArgs{}, map[string]any{})
	if err != nil {
		t.Fatalf("resolve web_fetch type: %v", err)
	}
	if fetchType != webFetchTypeBuiltin {
		t.Fatalf("expected default fetch type builtin, got %q", fetchType)
	}
}

func TestResolveWebFetchPlaywrightOptionsDefaultsMarkdownEnabled(t *testing.T) {
	t.Parallel()

	options := resolveWebFetchPlaywrightOptions(toolArgs{}, map[string]any{})
	if !options.Markdown {
		t.Fatalf("expected markdown conversion enabled by default")
	}
}

func TestResolveWebFetchTypeOnlyAcceptsBuiltinAndPlaywright(t *testing.T) {
	t.Parallel()

	fetchType := normalizeWebFetchType("builtin")
	if fetchType != webFetchTypeBuiltin {
		t.Fatalf("expected builtin fetch type, got %q", fetchType)
	}

	if _, err := resolveWebFetchType(toolArgs{"type": "terminal"}, map[string]any{}); err == nil {
		t.Fatalf("expected terminal type to be rejected")
	}
	if _, err := resolveWebFetchType(toolArgs{"type": "native"}, map[string]any{}); err == nil {
		t.Fatalf("expected native type to be rejected")
	}
	if _, err := resolveWebFetchType(toolArgs{"type": "http"}, map[string]any{}); err == nil {
		t.Fatalf("expected http type to be rejected")
	}
}

func TestConnectorTypeForURL(t *testing.T) {
	t.Parallel()

	if connectorType := connectorTypeForURL("https://www.google.com/search?q=test"); connectorType != "google" {
		t.Fatalf("expected google connector, got %q", connectorType)
	}
	if connectorType := connectorTypeForURL("https://www.xiaohongshu.com/explore"); connectorType != "xiaohongshu" {
		t.Fatalf("expected xiaohongshu connector, got %q", connectorType)
	}
	if connectorType := connectorTypeForURL("https://search.yahoo.com/search?p=test"); connectorType != "" {
		t.Fatalf("expected unsupported search domain to return empty connector, got %q", connectorType)
	}
	if connectorType := connectorTypeForURL("https://example.com/"); connectorType != "" {
		t.Fatalf("expected no connector match, got %q", connectorType)
	}
}

func TestRunWebSearchWithFallbackIgnoredForAPIType(t *testing.T) {
	t.Parallel()

	config := map[string]any{
		"web": map[string]any{
			"search": map[string]any{
				"type":     "api",
				"provider": "unknown-provider",
			},
		},
	}

	response := runWebSearchWithFallback(context.Background(), toolArgs{}, config, nil, "test query", 5)
	if !response.WebSearchAvailable {
		t.Fatalf("expected web_search_available=true in api mode")
	}
	if response.Message == "web_search_external_tools_not_configured" {
		t.Fatalf("did not expect external_tools placeholder message in api mode")
	}
}

func TestRunWebSearchWithFallbackExternalToolsPlaceholder(t *testing.T) {
	t.Parallel()

	config := map[string]any{
		"web": map[string]any{
			"search": map[string]any{
				"type": "external_tools",
			},
		},
	}

	response := runWebSearchWithFallback(context.Background(), toolArgs{}, config, nil, "test query", 5)
	if response.Message != "web_search_external_tools_not_configured" {
		t.Fatalf("expected external tools placeholder message, got %q", response.Message)
	}
	if response.WebSearchAvailable {
		t.Fatalf("expected web_search_available=false for external tools placeholder")
	}
}
