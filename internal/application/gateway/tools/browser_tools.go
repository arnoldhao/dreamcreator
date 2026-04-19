package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"dreamcreator/internal/application/browsercdp"
	appcookies "dreamcreator/internal/application/cookies"
	gatewaynodes "dreamcreator/internal/application/gateway/nodes"
)

var browserToolSessions = browsercdp.NewSessionRegistry()

func cleanupBrowserToolSessions(sessionKey string) {
	browserToolSessions.CloseSessionKey(strings.TrimSpace(sessionKey))
}

func CleanupAllBrowserToolSessions() {
	browserToolSessions.CloseAll()
}

func BrowserToolRuntimeConfigChanged(previousTools map[string]any, currentTools map[string]any) bool {
	previous := resolveBrowserRuntimeConfig(previousTools)
	current := resolveBrowserRuntimeConfig(currentTools)
	return previous.Enabled != current.Enabled ||
		previous.Headless != current.Headless ||
		previous.PreferredBrowser != current.PreferredBrowser
}

type browserProfileState struct {
	sessionKey  string
	profileName string
	resolved    browserResolvedConfig
	session     *browsercdp.Session
}

func runBrowserTool(settings SettingsReader, connectors ConnectorsReader, nodes *gatewaynodes.Service) func(ctx context.Context, args string) (string, error) {
	return func(ctx context.Context, args string) (string, error) {
		payload, err := parseToolArgs(args)
		if err != nil {
			return "", err
		}
		action, err := resolveBrowserAction(payload)
		if err != nil {
			return "", err
		}
		if isBrowserNodeTargetRequest(payload) {
			return runBrowserActionOnNode(ctx, payload, action, nodes)
		}
		toolsConfig := resolveToolsConfig(ctx, settings)
		resolved := resolveBrowserRuntimeConfig(toolsConfig)
		if !resolved.Enabled {
			return "", errors.New("browser disabled")
		}
		profileName := resolveBrowserProfileName(payload, resolved)
		sessionKey := resolveBrowserSessionKey(ctx, payload)
		state := getBrowserProfileState(sessionKey, profileName, resolved, connectors)
		result, err := runBrowserAction(ctx, payload, action, state)
		if err != nil {
			if browsercdp.IsFatalError(err) {
				return "", fmt.Errorf("browser session reset after runtime failure: %w", err)
			}
			return "", err
		}
		return marshalResult(result), nil
	}
}

func getBrowserProfileState(sessionKey string, profileName string, resolved browserResolvedConfig, connectors ConnectorsReader) *browserProfileState {
	options := browsercdp.SessionOptions{
		SessionKey:       sessionKey,
		ProfileName:      profileName,
		PreferredBrowser: resolved.PreferredBrowser,
		Headless:         resolved.Headless,
		UserDataDir:      filepath.Join(os.TempDir(), "dreamcreator", "browser", sessionKey, profileName),
		SSRFRules: browsercdp.SSRFPolicy{
			DangerouslyAllowPrivateNetwork: resolved.SSRFRules.DangerouslyAllowPrivateNetwork,
			AllowedHostnames:               cloneBrowserAllowedHostnames(resolved.SSRFRules.AllowedHostnames),
			HostnameAllowlist:              append([]string(nil), resolved.SSRFRules.HostnameAllowlist...),
		},
		Cookies: browsercdp.ConnectorCookieProviderFunc(func(ctx context.Context, rawURL string) ([]appcookies.Record, error) {
			return browsercdp.ResolveConnectorCookiesForURL(ctx, connectors, rawURL)
		}),
	}
	return &browserProfileState{
		sessionKey:  sessionKey,
		profileName: profileName,
		resolved:    resolved,
		session:     browserToolSessions.GetOrCreate(sessionKey, profileName, options),
	}
}

func cloneBrowserAllowedHostnames(values map[string]struct{}) map[string]struct{} {
	if len(values) == 0 {
		return map[string]struct{}{}
	}
	cloned := make(map[string]struct{}, len(values))
	for key := range values {
		cloned[key] = struct{}{}
	}
	return cloned
}

func runBrowserAction(ctx context.Context, payload toolArgs, action string, state *browserProfileState) (map[string]any, error) {
	switch action {
	case "open":
		return browserActionOpen(ctx, payload, state)
	case "navigate":
		return browserActionNavigate(ctx, payload, state)
	case "snapshot":
		return browserActionSnapshot(payload, state)
	case "wait":
		return browserActionWait(ctx, payload, state)
	case "scroll":
		return browserActionScroll(payload, state)
	case "upload":
		return browserActionUpload(payload, state)
	case "dialog":
		return browserActionDialog(payload, state)
	case "act":
		return browserActionAct(ctx, payload, state)
	case "reset":
		return browserActionReset(payload, state)
	default:
		return nil, errors.New("browser action not supported: " + action)
	}
}

func browserActionOpen(ctx context.Context, payload toolArgs, state *browserProfileState) (map[string]any, error) {
	result, err := state.session.Open(ctx, strings.TrimSpace(getStringArg(payload, "targetUrl", "url")), browserCommandOptions(payload, 30000))
	if err != nil {
		return nil, err
	}
	return browserResultMap(result), nil
}

func browserActionNavigate(ctx context.Context, payload toolArgs, state *browserProfileState) (map[string]any, error) {
	newTab, _ := getBoolArg(payload, "newTab")
	result, err := state.session.Navigate(
		ctx,
		strings.TrimSpace(getStringArg(payload, "targetId")),
		strings.TrimSpace(getStringArg(payload, "targetUrl", "url")),
		newTab,
		browserCommandOptions(payload, 30000),
	)
	if err != nil {
		return nil, err
	}
	return browserResultMap(result), nil
}

func browserActionSnapshot(payload toolArgs, state *browserProfileState) (map[string]any, error) {
	result, err := state.session.State(strings.TrimSpace(getStringArg(payload, "targetId")), resolveBrowserSnapshotLimit(payload))
	if err != nil {
		return nil, err
	}
	return browserResultMap(result), nil
}

func browserActionWait(ctx context.Context, payload toolArgs, state *browserProfileState) (map[string]any, error) {
	result, err := state.session.Wait(
		ctx,
		strings.TrimSpace(getStringArg(payload, "targetId")),
		browserWaitRequestFromArgs(payload),
		browserCommandOptions(payload, 15000),
	)
	if err != nil {
		return nil, err
	}
	return browserResultMap(result), nil
}

func browserActionScroll(payload toolArgs, state *browserProfileState) (map[string]any, error) {
	deltaX, deltaY := resolveBrowserScrollDelta(payload)
	result, err := state.session.Scroll(browsercdp.ScrollRequest{
		TargetID: strings.TrimSpace(getStringArg(payload, "targetId")),
		Ref:      strings.TrimSpace(getStringArg(payload, "ref")),
		DeltaX:   deltaX,
		DeltaY:   deltaY,
		Limit:    resolveBrowserSnapshotLimit(payload),
		Timeout:  browserTimeoutDuration(payload, 15000),
	})
	if err != nil {
		return nil, err
	}
	return browserResultMap(result), nil
}

func browserActionUpload(payload toolArgs, state *browserProfileState) (map[string]any, error) {
	rawPaths := getStringSliceArg(payload, "paths")
	rootDir, err := resolveBrowserUploadRootDir()
	if err != nil {
		return nil, err
	}
	paths := make([]string, 0, len(rawPaths))
	for _, item := range rawPaths {
		resolvedPath, err := resolvePathWithinRoot(rootDir, item)
		if err != nil {
			return nil, err
		}
		paths = append(paths, resolvedPath)
	}
	result, err := state.session.Upload(browsercdp.UploadRequest{
		TargetID: strings.TrimSpace(getStringArg(payload, "targetId")),
		Ref:      strings.TrimSpace(getStringArg(payload, "ref")),
		Paths:    paths,
		Limit:    resolveBrowserSnapshotLimit(payload),
		Timeout:  browserTimeoutDuration(payload, 15000),
	})
	if err != nil {
		return nil, err
	}
	return browserResultMap(result), nil
}

func browserActionDialog(payload toolArgs, state *browserProfileState) (map[string]any, error) {
	var accept *bool
	if value, ok := getBoolArg(payload, "accept"); ok {
		accept = &value
	}
	result, err := state.session.Dialog(browsercdp.DialogRequest{
		TargetID:   strings.TrimSpace(getStringArg(payload, "targetId")),
		Accept:     accept,
		PromptText: strings.TrimSpace(getStringArg(payload, "promptText")),
		Limit:      resolveBrowserSnapshotLimit(payload),
		Timeout:    browserTimeoutDuration(payload, 15000),
	})
	if err != nil {
		return nil, err
	}
	return browserResultMap(result), nil
}

func browserActionAct(ctx context.Context, payload toolArgs, state *browserProfileState) (map[string]any, error) {
	request := getMapArg(payload, "request")
	if request == nil {
		return nil, errors.New("request is required")
	}
	requestArgs := toolArgs(request)
	kind := strings.ToLower(strings.TrimSpace(getStringArg(requestArgs, "kind")))
	if kind == "" {
		return nil, errors.New("request.kind is required")
	}
	if !containsString(browserActKinds, kind) {
		return nil, errors.New("act kind not supported: " + kind)
	}
	if _, hasSelector := request["selector"]; hasSelector && kind != "wait" {
		return nil, errors.New(browserSelectorUnsupportedMessage)
	}
	if state == nil || state.session == nil {
		return nil, errors.New("browser session unavailable")
	}
	actRequest := browsercdp.ActRequest{
		Kind:       kind,
		TargetID:   firstNonEmptyString(strings.TrimSpace(getStringArg(requestArgs, "targetId")), strings.TrimSpace(getStringArg(payload, "targetId"))),
		Ref:        strings.TrimSpace(getStringArg(requestArgs, "ref")),
		Text:       getStringArg(requestArgs, "text"),
		Key:        strings.TrimSpace(getStringArg(requestArgs, "key")),
		Value:      getStringArg(requestArgs, "value"),
		Expression: getStringArg(requestArgs, "expression"),
		Limit:      resolveBrowserSnapshotLimit(payload),
		Timeout:    browserActTimeoutDuration(requestArgs, payload, 15000),
		Wait:       browserWaitRequestFromArgs(requestArgs),
		WaitFor:    browserOptionalWaitRequest(getMapArg(requestArgs, "waitFor")),
	}
	if width, ok := getIntArg(requestArgs, "width"); ok {
		actRequest.Width = width
	}
	if height, ok := getIntArg(requestArgs, "height"); ok {
		actRequest.Height = height
	}
	result, err := state.session.Act(ctx, actRequest)
	if err != nil {
		return nil, err
	}
	return browserResultMap(result), nil
}

func browserActionReset(payload toolArgs, state *browserProfileState) (map[string]any, error) {
	restart, _ := getBoolArg(payload, "restart")
	result, err := state.session.Reset(restart)
	if err != nil {
		return nil, err
	}
	output := browserResultMap(result)
	output["profile"] = state.profileName
	return output, nil
}

func browserCommandOptions(payload toolArgs, fallbackTimeoutMs int) browsercdp.CommandOptions {
	return browsercdp.CommandOptions{
		Limit:   resolveBrowserSnapshotLimit(payload),
		Timeout: browserTimeoutDuration(payload, fallbackTimeoutMs),
		WaitFor: browserOptionalWaitRequest(getMapArg(payload, "waitFor")),
	}
}

func browserOptionalWaitRequest(raw map[string]any) *browsercdp.WaitRequest {
	if raw == nil {
		return nil
	}
	request := browserWaitRequestFromArgs(toolArgs(raw))
	if browserWaitRequestEmpty(request) {
		return nil
	}
	return &request
}

func browserWaitRequestFromArgs(args toolArgs) browsercdp.WaitRequest {
	request := browsercdp.WaitRequest{
		Selector: strings.TrimSpace(getStringArg(args, "selector")),
		Text:     strings.TrimSpace(getStringArg(args, "text")),
		TextGone: strings.TrimSpace(getStringArg(args, "textGone")),
		URL:      strings.TrimSpace(getStringArg(args, "url")),
		Fn:       strings.TrimSpace(getStringArg(args, "fn")),
	}
	if timeMs, ok := getIntArg(args, "timeMs"); ok && timeMs > 0 {
		request.Time = time.Duration(timeMs) * time.Millisecond
	}
	if timeoutMs, ok := getIntArg(args, "timeoutMs"); ok && timeoutMs > 0 {
		request.Timeout = time.Duration(timeoutMs) * time.Millisecond
	}
	return request
}

func browserWaitRequestEmpty(request browsercdp.WaitRequest) bool {
	return request.Time <= 0 &&
		request.Selector == "" &&
		request.Text == "" &&
		request.TextGone == "" &&
		request.URL == "" &&
		request.Fn == ""
}

func browserTimeoutDuration(payload toolArgs, fallbackTimeoutMs int) time.Duration {
	return time.Duration(resolveBrowserActionTimeoutMs(payload, fallbackTimeoutMs)) * time.Millisecond
}

func browserActTimeoutDuration(request toolArgs, payload toolArgs, fallbackTimeoutMs int) time.Duration {
	return time.Duration(resolveBrowserActTimeoutMs(request, payload, fallbackTimeoutMs)) * time.Millisecond
}

func browserResultMap(result browsercdp.ActionResult) map[string]any {
	data, _ := json.Marshal(result)
	decoded := map[string]any{}
	_ = json.Unmarshal(data, &decoded)
	decoded["stateAvailable"] = result.State != nil || result.StateAvailable
	decoded["itemCount"] = browserResultItemCount(result)
	return decoded
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func browserResultItemCount(result browsercdp.ActionResult) int {
	if result.State != nil && result.State.ItemCount > 0 {
		return result.State.ItemCount
	}
	return len(result.Items)
}
