package settings

import "testing"

func TestDefaultCallsToolsConfigIncludesWebFetchDefaults(t *testing.T) {
	t.Parallel()

	defaults := DefaultCallsToolsConfig()
	fetchRaw, ok := defaults["web_fetch"].(map[string]any)
	if !ok || fetchRaw == nil {
		t.Fatalf("expected web_fetch defaults")
	}
	if fetchRaw["acceptMarkdown"] != true {
		t.Fatalf("expected acceptMarkdown default true, got %#v", fetchRaw["acceptMarkdown"])
	}
	if fetchRaw["acceptLanguage"] != "en-US,en;q=0.9" {
		t.Fatalf("expected acceptLanguage default en-US,en;q=0.9, got %#v", fetchRaw["acceptLanguage"])
	}
	if fetchRaw["type"] != "builtin" {
		t.Fatalf("expected type default builtin, got %#v", fetchRaw["type"])
	}
	playwrightRaw, ok := fetchRaw["playwright"].(map[string]any)
	if !ok || playwrightRaw == nil {
		t.Fatalf("expected playwright defaults")
	}
	if playwrightRaw["markdown"] != true {
		t.Fatalf("expected playwright markdown default true, got %#v", playwrightRaw["markdown"])
	}
	webRaw, ok := defaults["web"].(map[string]any)
	if !ok || webRaw == nil {
		t.Fatalf("expected web defaults")
	}
	if _, exists := webRaw["fetch"]; exists {
		t.Fatalf("expected legacy web.fetch defaults to be removed")
	}
}

func TestDefaultCallsToolsConfigIncludesBrowserDefaults(t *testing.T) {
	t.Parallel()

	defaults := DefaultCallsToolsConfig()
	browserRaw, ok := defaults["browser"].(map[string]any)
	if !ok || browserRaw == nil {
		t.Fatalf("expected browser defaults")
	}
	if browserRaw["enabled"] != true {
		t.Fatalf("expected browser enabled default true, got %#v", browserRaw["enabled"])
	}
	if browserRaw["evaluateEnabled"] != true {
		t.Fatalf("expected browser evaluateEnabled default true, got %#v", browserRaw["evaluateEnabled"])
	}
	if browserRaw["headless"] != false {
		t.Fatalf("expected browser headless default false, got %#v", browserRaw["headless"])
	}
	ssrfRaw, ok := browserRaw["ssrfPolicy"].(map[string]any)
	if !ok || ssrfRaw == nil {
		t.Fatalf("expected browser ssrfPolicy defaults")
	}
	if ssrfRaw["dangerouslyAllowPrivateNetwork"] != true {
		t.Fatalf(
			"expected browser ssrfPolicy.dangerouslyAllowPrivateNetwork true, got %#v",
			ssrfRaw["dangerouslyAllowPrivateNetwork"],
		)
	}
}

func TestNormalizeToolsConfigAddsWebFetchDefaultsWhenMissing(t *testing.T) {
	t.Parallel()

	tools := normalizeToolsConfig(map[string]any{})
	fetchRaw, ok := tools["web_fetch"].(map[string]any)
	if !ok || fetchRaw == nil {
		t.Fatalf("expected web_fetch config")
	}
	if fetchRaw["acceptMarkdown"] != true {
		t.Fatalf("expected acceptMarkdown default true, got %#v", fetchRaw["acceptMarkdown"])
	}
	if fetchRaw["userAgent"] != "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Safari/605.1.15" {
		t.Fatalf("expected userAgent default, got %#v", fetchRaw["userAgent"])
	}
}

func TestNormalizeToolsConfigDefaultsWebSearchTypeToAPI(t *testing.T) {
	t.Parallel()

	tools := normalizeToolsConfig(nil)
	webRaw, ok := tools["web"].(map[string]any)
	if !ok || webRaw == nil {
		t.Fatalf("expected web config")
	}
	searchRaw, ok := webRaw["search"].(map[string]any)
	if !ok || searchRaw == nil {
		t.Fatalf("expected web.search config")
	}
	if searchRaw["type"] != "api" {
		t.Fatalf("expected default type api, got %#v", searchRaw["type"])
	}
	if _, exists := searchRaw["external_tools"]; !exists {
		t.Fatalf("expected web.search.external_tools placeholder to exist")
	}
}

func TestNormalizeToolsConfigAddsBrowserDefaultsToExistingConfig(t *testing.T) {
	t.Parallel()

	tools := normalizeToolsConfig(map[string]any{
		"browser": map[string]any{
			"enabled": false,
			"ssrfPolicy": map[string]any{
				"dangerouslyAllowPrivateNetwork": false,
			},
		},
	})
	browserRaw, ok := tools["browser"].(map[string]any)
	if !ok || browserRaw == nil {
		t.Fatalf("expected browser config")
	}
	if browserRaw["enabled"] != false {
		t.Fatalf("expected existing browser enabled=false to be preserved, got %#v", browserRaw["enabled"])
	}
	ssrfRaw, ok := browserRaw["ssrfPolicy"].(map[string]any)
	if !ok || ssrfRaw == nil {
		t.Fatalf("expected browser ssrfPolicy")
	}
	if ssrfRaw["dangerouslyAllowPrivateNetwork"] != false {
		t.Fatalf(
			"expected existing browser ssrfPolicy.dangerouslyAllowPrivateNetwork=false to be preserved, got %#v",
			ssrfRaw["dangerouslyAllowPrivateNetwork"],
		)
	}
}
