package settings

import "testing"

func TestDefaultCallsToolsConfigIncludesWebFetchDefaults(t *testing.T) {
	t.Parallel()

	defaults := DefaultCallsToolsConfig()
	fetchRaw, ok := defaults["web_fetch"].(map[string]any)
	if !ok || fetchRaw == nil {
		t.Fatalf("expected web_fetch defaults")
	}
	if fetchRaw["preferredBrowser"] != "chrome" {
		t.Fatalf("expected preferredBrowser default chrome, got %#v", fetchRaw["preferredBrowser"])
	}
	if fetchRaw["headless"] != true {
		t.Fatalf("expected web_fetch headless default true, got %#v", fetchRaw["headless"])
	}
	if _, exists := fetchRaw["type"]; exists {
		t.Fatalf("expected legacy web_fetch type to be removed")
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
	if browserRaw["headless"] != true {
		t.Fatalf("expected browser headless default true, got %#v", browserRaw["headless"])
	}
	if browserRaw["preferredBrowser"] != "chrome" {
		t.Fatalf("expected browser preferredBrowser default chrome, got %#v", browserRaw["preferredBrowser"])
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
	if fetchRaw["preferredBrowser"] != "chrome" {
		t.Fatalf("expected preferredBrowser default, got %#v", fetchRaw["preferredBrowser"])
	}
	if fetchRaw["headless"] != true {
		t.Fatalf("expected headless default true, got %#v", fetchRaw["headless"])
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
