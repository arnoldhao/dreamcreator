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

	"dreamcreator/internal/application/agentruntime"
	"dreamcreator/internal/application/browsercdp"
	"github.com/cloudwego/eino/schema"
)

func TestResolveBrowserActionRequiresExplicitAction(t *testing.T) {
	t.Parallel()

	_, err := resolveBrowserAction(toolArgs{})
	if err == nil {
		t.Fatalf("expected missing action error")
	}
}

func TestResolveBrowserActionRequiresExplicitActionEvenWhenURLPresent(t *testing.T) {
	t.Parallel()

	_, err := resolveBrowserAction(toolArgs{"targetUrl": "https://example.com"})
	if err == nil {
		t.Fatalf("expected missing action error")
	}
}

func TestResolveBrowserActionRejectsUnsupportedFetchAlias(t *testing.T) {
	t.Parallel()

	_, err := resolveBrowserAction(toolArgs{"action": "fetch"})
	if err == nil {
		t.Fatalf("expected unsupported action error")
	}
}

func TestSpecBrowserExposesWorkflowActionsOnly(t *testing.T) {
	t.Parallel()

	var schema map[string]any
	if err := json.Unmarshal([]byte(specBrowser().SchemaJSON), &schema); err != nil {
		t.Fatalf("decode schema: %v", err)
	}
	properties, _ := schema["properties"].(map[string]any)
	actionDef, _ := properties["action"].(map[string]any)
	rawEnum, _ := actionDef["enum"].([]any)
	got := make([]string, 0, len(rawEnum))
	for _, item := range rawEnum {
		if value, ok := item.(string); ok && value != "" {
			got = append(got, value)
		}
	}
	want := []string{"open", "navigate", "snapshot", "act", "wait", "scroll", "upload", "dialog", "reset"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("unexpected browser actions: got %v want %v", got, want)
	}
}

func TestSpecBrowserDescriptionExplainsSnapshotWorkflow(t *testing.T) {
	t.Parallel()

	description := specBrowser().Description
	if !strings.Contains(description, "browser-use style loop") {
		t.Fatalf("expected browser description to mention browser-use loop, got %q", description)
	}
	if !strings.Contains(description, "pass `url` or `targetUrl`") {
		t.Fatalf("expected browser description to document url alias, got %q", description)
	}
	if !strings.Contains(description, "return `stateAvailable`, `itemCount`, and the current page `state`/`items`") {
		t.Fatalf("expected browser description to explain open/navigate result state, got %q", description)
	}
	if !strings.Contains(description, "`stateAvailable=false`") {
		t.Fatalf("expected browser description to explain stateAvailable fallback, got %q", description)
	}
	if !strings.Contains(description, "call `snapshot` to refresh them") {
		t.Fatalf("expected browser description to explain open/navigate flow, got %q", description)
	}
	if !strings.Contains(description, "use `ref` from the latest state") {
		t.Fatalf("expected browser description to prefer refs, got %q", description)
	}
}

func TestSpecBrowserSchemaAcceptsOpenURLAlias(t *testing.T) {
	t.Parallel()

	validator := agentruntime.JSONToolValidator{
		Tools: map[string]agentruntime.ToolDefinition{
			"browser": {
				Name:       "browser",
				SchemaJSON: specBrowser().SchemaJSON,
			},
		},
	}

	err := validator.Validate(schema.ToolCall{
		Function: schema.FunctionCall{
			Name:      "browser",
			Arguments: `{"action":"open","url":"https://example.com"}`,
		},
	})
	if err != nil {
		t.Fatalf("expected open url alias to validate, got %v", err)
	}
}

func TestSpecBrowserSchemaAcceptsNavigateURLAlias(t *testing.T) {
	t.Parallel()

	validator := agentruntime.JSONToolValidator{
		Tools: map[string]agentruntime.ToolDefinition{
			"browser": {
				Name:       "browser",
				SchemaJSON: specBrowser().SchemaJSON,
			},
		},
	}

	err := validator.Validate(schema.ToolCall{
		Function: schema.FunctionCall{
			Name:      "browser",
			Arguments: `{"action":"navigate","url":"https://example.com"}`,
		},
	})
	if err != nil {
		t.Fatalf("expected navigate url alias to validate, got %v", err)
	}
}

func TestResolveBrowserRuntimeConfigDefaults(t *testing.T) {
	t.Parallel()

	resolved := resolveBrowserRuntimeConfig(map[string]any{})
	if !resolved.Enabled {
		t.Fatalf("expected browser enabled by default")
	}
	if resolved.DefaultProfile != defaultBrowserProfileDreamCreator {
		t.Fatalf("expected default profile %q, got %q", defaultBrowserProfileDreamCreator, resolved.DefaultProfile)
	}
	if _, ok := resolved.Profiles[defaultBrowserProfileDreamCreator]; !ok {
		t.Fatalf("expected dreamcreator profile default")
	}
	if resolved.SSRFRules.DangerouslyAllowPrivateNetwork {
		t.Fatalf("expected ssrf dangerous allow default false")
	}
}

func TestResolveBrowserRuntimeConfigReadsBrowserSettings(t *testing.T) {
	t.Parallel()

	resolved := resolveBrowserRuntimeConfig(map[string]any{
		"browser": map[string]any{
			"enabled":  false,
			"headless": true,
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
	if !resolved.Headless {
		t.Fatalf("expected headless true")
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

func TestBrowserToolRuntimeConfigChangedDetectsRuntimeChanges(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		previous map[string]any
		current  map[string]any
		want     bool
	}{
		{
			name: "same config",
			previous: map[string]any{
				"browser": map[string]any{
					"enabled":          true,
					"headless":         true,
					"preferredBrowser": "chrome",
				},
			},
			current: map[string]any{
				"browser": map[string]any{
					"enabled":          true,
					"headless":         true,
					"preferredBrowser": "chrome",
				},
			},
			want: false,
		},
		{
			name: "headless changed",
			previous: map[string]any{
				"browser": map[string]any{
					"headless": false,
				},
			},
			current: map[string]any{
				"browser": map[string]any{
					"headless": true,
				},
			},
			want: true,
		},
		{
			name: "preferred browser changed",
			previous: map[string]any{
				"browser": map[string]any{
					"preferredBrowser": "chrome",
				},
			},
			current: map[string]any{
				"browser": map[string]any{
					"preferredBrowser": "brave",
				},
			},
			want: true,
		},
		{
			name: "enabled changed",
			previous: map[string]any{
				"browser": map[string]any{
					"enabled": true,
				},
			},
			current: map[string]any{
				"browser": map[string]any{
					"enabled": false,
				},
			},
			want: true,
		},
		{
			name: "ssrf changed only",
			previous: map[string]any{
				"browser": map[string]any{
					"headless": true,
					"ssrfPolicy": map[string]any{
						"dangerouslyAllowPrivateNetwork": false,
					},
				},
			},
			current: map[string]any{
				"browser": map[string]any{
					"headless": true,
					"ssrfPolicy": map[string]any{
						"dangerouslyAllowPrivateNetwork": true,
					},
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := BrowserToolRuntimeConfigChanged(tt.previous, tt.current); got != tt.want {
				t.Fatalf("BrowserToolRuntimeConfigChanged() = %v, want %v", got, tt.want)
			}
		})
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

	_, err := browserActionAct(context.Background(), toolArgs{
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

func TestResolveBrowserActionAcceptsSnapshotAction(t *testing.T) {
	t.Parallel()

	action, err := resolveBrowserAction(toolArgs{
		"action": "snapshot",
	})
	if err != nil {
		t.Fatalf("expected snapshot action to be accepted, got %v", err)
	}
	if action != "snapshot" {
		t.Fatalf("unexpected action: got %q want snapshot", action)
	}
}

func TestResolveBrowserActionRejectsStateAction(t *testing.T) {
	t.Parallel()

	_, err := resolveBrowserAction(toolArgs{
		"action": "state",
	})
	if err == nil {
		t.Fatalf("expected state action to be rejected")
	}
	if !strings.Contains(err.Error(), "not supported") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBrowserActionActRejectsUnsupportedNavigateKind(t *testing.T) {
	t.Parallel()

	_, err := browserActionAct(context.Background(), toolArgs{
		"request": map[string]any{
			"kind": "navigate",
		},
	}, &browserProfileState{})
	if err == nil {
		t.Fatalf("expected unsupported kind error")
	}
	if !strings.Contains(err.Error(), "act kind not supported") {
		t.Fatalf("unexpected error: %v", err)
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

func TestResolveBrowserScrollDeltaDefaultsDown(t *testing.T) {
	t.Parallel()

	x, y := resolveBrowserScrollDelta(toolArgs{})
	if x != 0 || y != 700 {
		t.Fatalf("expected default down scroll, got %d/%d", x, y)
	}
}

func TestResolveBrowserScrollDeltaHonorsDirectionAndAmount(t *testing.T) {
	t.Parallel()

	x, y := resolveBrowserScrollDelta(toolArgs{"direction": "left", "amount": 320})
	if x != -320 || y != 0 {
		t.Fatalf("expected left scroll -320/0, got %d/%d", x, y)
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

	if !isBrowserSnapshotForAIUnavailable(errors.New("browser snapshotForAI is unavailable")) {
		t.Fatalf("expected snapshotForAI unavailable error to be treated as unavailable")
	}
	if isBrowserSnapshotForAIUnavailable(errors.New("network timeout")) {
		t.Fatalf("expected unrelated error to remain actionable")
	}
}

func TestBrowserResultMapPreservesSnapshotPayload(t *testing.T) {
	t.Parallel()

	result := browserResultMap(browsercdp.ActionResult{
		OK:           true,
		TargetID:     "tab-1",
		URL:          "https://example.com",
		StateVersion: 7,
		Items: []browsercdp.SnapshotItem{
			{Ref: "e1", Role: "button", Name: "Save"},
		},
		State: &browsercdp.PageState{
			Version:    7,
			URL:        "https://example.com",
			ItemCount:  1,
			CapturedAt: "2026-04-18T00:00:00Z",
		},
		StateAvailable: true,
		StateError:     "warning",
	})

	if _, ok := result["state"]; !ok {
		t.Fatalf("expected state to be preserved")
	}
	if got := result["stateVersion"]; got != float64(7) {
		t.Fatalf("expected stateVersion to remain, got %#v", got)
	}
	if _, ok := result["items"]; !ok {
		t.Fatalf("expected items to be preserved")
	}
	if got := result["stateAvailable"]; got != true {
		t.Fatalf("expected stateAvailable to remain, got %#v", got)
	}
	if got := result["itemCount"]; got != 1 {
		t.Fatalf("expected itemCount to remain, got %#v", got)
	}
	if got := result["stateError"]; got != "warning" {
		t.Fatalf("expected stateError to remain, got %#v", got)
	}
	if got := result["targetId"]; got != "tab-1" {
		t.Fatalf("expected targetId to remain, got %#v", got)
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
