package tools

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveBrowserActionDefaultsStatus(t *testing.T) {
	t.Parallel()

	action, err := resolveBrowserAction(toolArgs{})
	if err != nil {
		t.Fatalf("resolve action: %v", err)
	}
	if action != "status" {
		t.Fatalf("expected default action status, got %q", action)
	}
}

func TestResolveBrowserActionDefaultsOpenWhenURLPresent(t *testing.T) {
	t.Parallel()

	action, err := resolveBrowserAction(toolArgs{"url": "https://example.com"})
	if err != nil {
		t.Fatalf("resolve action: %v", err)
	}
	if action != "open" {
		t.Fatalf("expected default action open with url, got %q", action)
	}
}

func TestResolveBrowserActionRejectsUnsupportedFetchAlias(t *testing.T) {
	t.Parallel()

	_, err := resolveBrowserAction(toolArgs{"action": "fetch"})
	if err == nil {
		t.Fatalf("expected unsupported action error")
	}
}

func TestResolveBrowserRuntimeConfigDefaults(t *testing.T) {
	t.Parallel()

	resolved := resolveBrowserRuntimeConfig(map[string]any{})
	if !resolved.Enabled {
		t.Fatalf("expected browser enabled by default")
	}
	if !resolved.EvaluateEnabled {
		t.Fatalf("expected evaluateEnabled default true")
	}
	if resolved.DefaultProfile != defaultBrowserProfileDreamCreator {
		t.Fatalf("expected default profile %q, got %q", defaultBrowserProfileDreamCreator, resolved.DefaultProfile)
	}
	if _, ok := resolved.Profiles[defaultBrowserProfileDreamCreator]; !ok {
		t.Fatalf("expected dreamcreator profile default")
	}
	if !resolved.SSRFRules.DangerouslyAllowPrivateNetwork {
		t.Fatalf("expected ssrf dangerous allow default true")
	}
}

func TestResolveBrowserRuntimeConfigReadsBrowserSettings(t *testing.T) {
	t.Parallel()

	resolved := resolveBrowserRuntimeConfig(map[string]any{
		"browser": map[string]any{
			"enabled":         false,
			"evaluateEnabled": false,
			"headless":        true,
			"noSandbox":       true,
			"extraArgs":       []any{"--window-size=1280,900"},
			"ssrfPolicy": map[string]any{
				"dangerouslyAllowPrivateNetwork": false,
				"allowedHostnames":               []any{"localhost"},
				"hostnameAllowlist":              []any{"*.example.com"},
			},
		},
	})
	if resolved.Enabled {
		t.Fatalf("expected enabled false")
	}
	if resolved.EvaluateEnabled {
		t.Fatalf("expected evaluateEnabled false")
	}
	if !resolved.Headless {
		t.Fatalf("expected headless true")
	}
	if !resolved.NoSandbox {
		t.Fatalf("expected noSandbox true")
	}
	if len(resolved.ExtraArgs) != 1 || resolved.ExtraArgs[0] != "--window-size=1280,900" {
		t.Fatalf("expected extraArgs from config")
	}
	if resolved.SSRFRules.DangerouslyAllowPrivateNetwork {
		t.Fatalf("expected ssrf dangerous allow false")
	}
	if _, ok := resolved.SSRFRules.AllowedHostnames["localhost"]; !ok {
		t.Fatalf("expected localhost in allowedHostnames")
	}
	if len(resolved.SSRFRules.HostnameAllowlist) != 1 || resolved.SSRFRules.HostnameAllowlist[0] != "*.example.com" {
		t.Fatalf("expected hostnameAllowlist from config")
	}
}

func TestAssertBrowserURLAllowedStrictPolicyBlocksPrivate(t *testing.T) {
	t.Parallel()

	policy := browserSSRFPolicy{
		DangerouslyAllowPrivateNetwork: false,
		AllowedHostnames:               map[string]struct{}{},
	}
	if err := assertBrowserURLAllowed("http://127.0.0.1:3000", policy); err == nil {
		t.Fatalf("expected private loopback to be blocked")
	}
}

func TestAssertBrowserURLAllowedStrictPolicyAllowsAllowlist(t *testing.T) {
	t.Parallel()

	policy := browserSSRFPolicy{
		DangerouslyAllowPrivateNetwork: false,
		AllowedHostnames: map[string]struct{}{
			"localhost": {},
		},
	}
	if err := assertBrowserURLAllowed("http://localhost:3000", policy); err != nil {
		t.Fatalf("expected localhost allowlisted, got %v", err)
	}
}

func TestResolveBrowserActTimeoutMsPrefersRequest(t *testing.T) {
	t.Parallel()

	got := resolveBrowserActTimeoutMs(toolArgs{"timeoutMs": 2345}, toolArgs{"timeoutMs": 9999}, 1111)
	if got != 2345 {
		t.Fatalf("expected request timeoutMs to win, got %d", got)
	}
}

func TestResolveBrowserActTimeoutMsFallsBackToPayload(t *testing.T) {
	t.Parallel()

	got := resolveBrowserActTimeoutMs(toolArgs{}, toolArgs{"timeoutMs": 6789}, 1111)
	if got != 6789 {
		t.Fatalf("expected payload timeoutMs fallback, got %d", got)
	}
}

func TestParseBrowserAriaSnapshotRoleRefs(t *testing.T) {
	t.Parallel()

	snapshot := `
- button "Save":
- button "Save":
`
	items := parseBrowserAriaSnapshot(snapshot, true, 50, false, 0)
	if len(items) != 2 {
		t.Fatalf("expected 2 parsed lines, got %d", len(items))
	}
	if items[0].Ref != "e1" || items[1].Ref != "e2" {
		t.Fatalf("expected generated refs e1/e2, got %q/%q", items[0].Ref, items[1].Ref)
	}
	if items[0].Nth != 0 || items[1].Nth != 1 {
		t.Fatalf("expected nth markers 0/1, got %d/%d", items[0].Nth, items[1].Nth)
	}
}

func TestParseBrowserAriaSnapshotAriaRefs(t *testing.T) {
	t.Parallel()

	snapshot := `
- button "Submit" [ref=s1e12]:
- link "Docs" [ref=s1e15]:
`
	items := parseBrowserAriaSnapshot(snapshot, false, 50, true, 0)
	if len(items) != 2 {
		t.Fatalf("expected 2 parsed lines, got %d", len(items))
	}
	if items[0].Ref != "s1e12" || items[0].AriaRef != "s1e12" {
		t.Fatalf("expected aria ref to be preserved, got ref=%q ariaRef=%q", items[0].Ref, items[0].AriaRef)
	}
	if items[1].Ref != "s1e15" || items[1].AriaRef != "s1e15" {
		t.Fatalf("expected aria ref to be preserved, got ref=%q ariaRef=%q", items[1].Ref, items[1].AriaRef)
	}
}

func TestParseBrowserAriaSnapshotInteractiveOnly(t *testing.T) {
	t.Parallel()

	snapshot := `
- heading "Title" [level=1]
- button "Save"
- listitem: row
`
	items := parseBrowserAriaSnapshot(snapshot, true, 50, false, 0)
	if len(items) != 1 {
		t.Fatalf("expected only interactive roles to remain, got %d", len(items))
	}
	if items[0].Role != "button" {
		t.Fatalf("expected remaining role button, got %q", items[0].Role)
	}
}

func TestParseBrowserAriaSnapshotRespectsMaxDepth(t *testing.T) {
	t.Parallel()

	snapshot := `
- region "Root"
  - button "Save"
    - text "Nested value"
`
	items := parseBrowserAriaSnapshot(snapshot, false, 50, false, 1)
	if len(items) != 2 {
		t.Fatalf("expected maxDepth filter to keep 2 nodes, got %d", len(items))
	}
	if items[0].Role != "region" || items[1].Role != "button" {
		t.Fatalf("unexpected filtered roles: %q/%q", items[0].Role, items[1].Role)
	}
}

func TestBrowserActionActRejectsSelectorForNonWaitBeforeRuntime(t *testing.T) {
	t.Parallel()

	_, err := browserActionAct(toolArgs{
		"request": map[string]any{
			"kind":     "click",
			"selector": "button.save",
		},
	}, &browserProfileState{})
	if err == nil {
		t.Fatalf("expected selector unsupported error")
	}
	if err.Error() != browserSelectorUnsupportedMessage {
		t.Fatalf("unexpected selector error: %q", err.Error())
	}
}

func TestNormalizeBrowserTimeoutMsClampRange(t *testing.T) {
	t.Parallel()

	if got := normalizeBrowserTimeoutMs(10, 1000); got != 500 {
		t.Fatalf("expected min clamp 500, got %d", got)
	}
	if got := normalizeBrowserTimeoutMs(999999, 1000); got != 120000 {
		t.Fatalf("expected max clamp 120000, got %d", got)
	}
	if got := normalizeBrowserTimeoutMs(2500, 1000); got != 2500 {
		t.Fatalf("expected in-range timeout unchanged, got %d", got)
	}
}

func TestRunBrowserActionOnNodeRequiresNodesService(t *testing.T) {
	t.Parallel()

	_, err := runBrowserActionOnNode(context.Background(), toolArgs{"target": "node"}, "status", nil)
	if err == nil {
		t.Fatalf("expected nodes service unavailable error")
	}
	if !strings.Contains(err.Error(), "nodes service unavailable") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolvePathWithinRootRejectsTraversal(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	_, err := resolvePathWithinRoot(root, "../outside.txt")
	if err == nil {
		t.Fatalf("expected traversal to be rejected")
	}
	if !strings.Contains(err.Error(), "Invalid path") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolvePathWithinRootAllowsAbsoluteInside(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	inside := filepath.Join(root, "nested", "a.txt")
	if err := os.MkdirAll(filepath.Dir(inside), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	resolved, err := resolvePathWithinRoot(root, inside)
	if err != nil {
		t.Fatalf("resolve absolute inside: %v", err)
	}
	if resolved != inside {
		t.Fatalf("expected %s, got %s", inside, resolved)
	}
}

func TestResolveBrowserNodeOutputUnwrapsProxyEnvelope(t *testing.T) {
	t.Parallel()

	payload, err := json.Marshal(map[string]any{
		"result": map[string]any{
			"ok":   true,
			"path": "remote://snapshot.png",
		},
		"files": []map[string]any{
			{
				"path":     "remote://snapshot.png",
				"base64":   base64.StdEncoding.EncodeToString([]byte("image-bytes")),
				"mimeType": "image/png",
			},
		},
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	resolved := resolveBrowserNodeOutput(string(payload))
	if resolved == nil {
		t.Fatalf("expected resolved payload")
	}
	obj, ok := resolved.(map[string]any)
	if !ok {
		t.Fatalf("expected map payload, got %T", resolved)
	}
	pathValue, _ := obj["path"].(string)
	if strings.TrimSpace(pathValue) == "" {
		t.Fatalf("expected mapped local path")
	}
	if strings.Contains(pathValue, "remote://") {
		t.Fatalf("expected remote path to be rewritten, got %q", pathValue)
	}
	if _, statErr := os.Stat(pathValue); statErr != nil {
		t.Fatalf("expected mapped path to exist: %v", statErr)
	}
}

func TestResolveBrowserNodeOutputReturnsNilOnInvalidJSON(t *testing.T) {
	t.Parallel()

	if resolved := resolveBrowserNodeOutput("{not-json"); resolved != nil {
		t.Fatalf("expected nil for invalid node output, got %T", resolved)
	}
}

func TestIsBrowserSnapshotForAIUnavailable(t *testing.T) {
	t.Parallel()

	if !isBrowserSnapshotForAIUnavailable(errors.New("Playwright _snapshotForAI is not available")) {
		t.Fatalf("expected _snapshotForAI error to be treated as unavailable")
	}
	if isBrowserSnapshotForAIUnavailable(errors.New("network timeout")) {
		t.Fatalf("expected unrelated error to remain actionable")
	}
}

func TestToBrowserFriendlyInteractionErrorStrictMode(t *testing.T) {
	t.Parallel()

	err := toBrowserFriendlyInteractionError(
		errors.New("strict mode violation: locator resolved to 3 elements"),
		"e12",
	)
	if err == nil {
		t.Fatalf("expected mapped error")
	}
	if !strings.Contains(err.Error(), `matched 3 elements`) {
		t.Fatalf("unexpected mapped message: %v", err)
	}
}
