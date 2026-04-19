package tools

import (
	"context"
	"encoding/json"
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
					"preferredBrowser": "chrome",
				},
			},
		},
	}
}

func TestRunWebFetchTool_RejectsNonGETMethods(t *testing.T) {
	t.Parallel()

	handler := runWebFetchTool(builtinWebFetchSettingsStub(), nil)
	output, err := handler(context.Background(), `{"url":"https://example.com","method":"POST"}`)
	if err != nil {
		t.Fatalf("run web_fetch: %v", err)
	}

	var payload webFetchResult
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if payload.Status != webStatusError {
		t.Fatalf("unexpected status: %q", payload.Status)
	}
	if payload.Message != "web_fetch only supports GET" {
		t.Fatalf("unexpected message: %q", payload.Message)
	}
	if payload.NextAction != nextActionUseOtherToolsOrSkills {
		t.Fatalf("unexpected next action: %q", payload.NextAction)
	}
}

func TestResolveWebFetchPreferredBrowser_ReadsTopLevelConfig(t *testing.T) {
	t.Parallel()

	preferred := resolveWebFetchPreferredBrowser(map[string]any{
		"web_fetch": map[string]any{
			"preferredBrowser": "brave",
		},
		"browser": map[string]any{
			"preferredBrowser": "chrome",
		},
	})
	if preferred != "brave" {
		t.Fatalf("expected preferred browser brave, got %q", preferred)
	}

	fallback := resolveWebFetchPreferredBrowser(map[string]any{
		"browser": map[string]any{
			"preferredBrowser": "edge",
		},
	})
	if fallback != "edge" {
		t.Fatalf("expected fallback preferred browser edge, got %q", fallback)
	}
}

func TestResolveWebFetchOptions_ReadsTopLevelWebFetchConfig(t *testing.T) {
	t.Parallel()

	options := resolveWebFetchOptions(toolArgs{}, map[string]any{
		"web_fetch": map[string]any{
			"timeoutSeconds": 21,
			"maxChars":       1234,
			"maxBodyBytes":   4321,
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
}

func TestBuildWebFetchToolResult_AnnotatesCDPSource(t *testing.T) {
	t.Parallel()

	payload := buildWebFetchToolResult("https://example.com", webFetchResponse{
		FinalURL:       "https://example.com/article",
		Status:         200,
		ContentType:    "text/markdown",
		Content:        "# Hello\n\nworld",
		MarkdownTokens: 4,
		ContentSignal:  "article_readability",
	}, nil)
	if payload.Status != webStatusOK {
		t.Fatalf("unexpected status: %q", payload.Status)
	}
	if payload.Data["browserSource"] != webFetchTypeCDP {
		t.Fatalf("expected browser source %q, got %#v", webFetchTypeCDP, payload.Data["browserSource"])
	}
	if payload.ContentSignal != "article_readability" {
		t.Fatalf("unexpected content signal: %q", payload.ContentSignal)
	}
	if payload.MarkdownTokens != 4 {
		t.Fatalf("unexpected markdown tokens: %d", payload.MarkdownTokens)
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

func TestResolveWebFetchTypeDefaultsToCDP(t *testing.T) {
	t.Parallel()

	fetchType, err := resolveWebFetchType(toolArgs{}, map[string]any{})
	if err != nil {
		t.Fatalf("resolve web_fetch type: %v", err)
	}
	if fetchType != webFetchTypeCDP {
		t.Fatalf("expected default fetch type cdp, got %q", fetchType)
	}
}

func TestResolveWebFetchTypeNormalizesLegacyLabelsToCDP(t *testing.T) {
	t.Parallel()

	fetchType := normalizeWebFetchType("builtin")
	if fetchType != webFetchTypeCDP {
		t.Fatalf("expected builtin label to normalize to cdp, got %q", fetchType)
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
	if connectorType := connectorTypeForURL("https://www.youtube.com/watch?v=test"); connectorType != "google" {
		t.Fatalf("expected youtube URLs to use google connector cookies, got %q", connectorType)
	}
	if connectorType := connectorTypeForURL("https://github.com/owner/repo"); connectorType != "github" {
		t.Fatalf("expected github connector, got %q", connectorType)
	}
	if connectorType := connectorTypeForURL("https://www.reddit.com/r/golang/comments/test"); connectorType != "reddit" {
		t.Fatalf("expected reddit connector, got %q", connectorType)
	}
	if connectorType := connectorTypeForURL("https://www.zhihu.com/question/123456"); connectorType != "zhihu" {
		t.Fatalf("expected zhihu connector, got %q", connectorType)
	}
	if connectorType := connectorTypeForURL("https://x.com/example/status/1"); connectorType != "x" {
		t.Fatalf("expected x connector, got %q", connectorType)
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
