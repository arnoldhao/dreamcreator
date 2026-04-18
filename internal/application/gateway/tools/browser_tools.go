package tools

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	gatewaynodes "dreamcreator/internal/application/gateway/nodes"
	"github.com/chromedp/cdproto/emulation"
	pagepkg "github.com/chromedp/cdproto/page"
	runtimepkg "github.com/chromedp/cdproto/runtime"
	targetpkg "github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"

	"dreamcreator/internal/application/browsercdp"
)

var globalBrowserSessions = struct {
	mu       sync.Mutex
	sessions map[string]*browserSessionState
}{
	sessions: map[string]*browserSessionState{},
}

type browserSessionState struct {
	sessionKey string
	profiles   map[string]*browserProfileState
}

type browserProfileState struct {
	mu sync.Mutex

	sessionKey  string
	profileName string
	resolved    browserResolvedConfig
	profile     browserProfileConfig
	runtime     *browsercdp.Runtime

	tabs         map[string]*browserTabState
	activeTarget string

	consoleMessages []browserConsoleMessage
	pendingUploads  map[string]browserPendingUpload
	pendingDialogs  map[string]browserPendingDialog
}

type browserTabState struct {
	TargetID string
	ctx      context.Context
	cancel   context.CancelFunc

	mu             sync.RWMutex
	refs           map[string]browserSnapshotRef
	evaluateResult any
	lastURL        string
}

type browserSnapshotRef struct {
	Selector string
	Role     string
	Name     string
	Nth      int
	Mode     string
	AriaRef  string
	Frame    string
}

type browserConsoleMessage struct {
	TargetID  string `json:"targetId"`
	Type      string `json:"type"`
	Text      string `json:"text"`
	Timestamp string `json:"timestamp"`
}

type browserPendingUpload struct {
	Paths     []string
	ExpiresAt time.Time
}

type browserPendingDialog struct {
	Accept     bool      `json:"accept"`
	PromptText string    `json:"promptText,omitempty"`
	Message    string    `json:"message,omitempty"`
	Type       string    `json:"type,omitempty"`
	ExpiresAt  time.Time `json:"expiresAt"`
}

type browserSnapshotJSItem struct {
	Selector string `json:"selector"`
	Role     string `json:"role"`
	Name     string `json:"name"`
	Text     string `json:"text"`
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
		state := getOrCreateBrowserProfileState(sessionKey, profileName, resolved)
		result, err := runBrowserAction(ctx, payload, action, state, connectors)
		if err != nil {
			return "", err
		}
		return marshalResult(result), nil
	}
}

func getOrCreateBrowserProfileState(sessionKey string, profileName string, resolved browserResolvedConfig) *browserProfileState {
	globalBrowserSessions.mu.Lock()
	defer globalBrowserSessions.mu.Unlock()

	session, ok := globalBrowserSessions.sessions[sessionKey]
	if !ok {
		session = &browserSessionState{
			sessionKey: sessionKey,
			profiles:   map[string]*browserProfileState{},
		}
		globalBrowserSessions.sessions[sessionKey] = session
	}
	state, ok := session.profiles[profileName]
	if !ok {
		state = &browserProfileState{
			sessionKey:      sessionKey,
			profileName:     profileName,
			resolved:        resolved,
			profile:         resolved.Profiles[profileName],
			tabs:            map[string]*browserTabState{},
			pendingUploads:  map[string]browserPendingUpload{},
			pendingDialogs:  map[string]browserPendingDialog{},
			consoleMessages: nil,
		}
		session.profiles[profileName] = state
		return state
	}
	state.mu.Lock()
	state.resolved = resolved
	if profile, exists := resolved.Profiles[profileName]; exists {
		state.profile = profile
	}
	state.mu.Unlock()
	return state
}

func runBrowserAction(ctx context.Context, payload toolArgs, action string, state *browserProfileState, connectors ConnectorsReader) (map[string]any, error) {
	switch action {
	case "status":
		return browserActionStatus(state), nil
	case "start":
		if err := ensureBrowserProfileStarted(state); err != nil {
			return nil, err
		}
		return browserActionStatus(state), nil
	case "stop":
		stopBrowserProfile(state)
		return browserActionStatus(state), nil
	case "profiles":
		return map[string]any{
			"profiles": []map[string]any{
				{
					"name":    state.profileName,
					"color":   state.profile.Color,
					"driver":  state.profile.Driver,
					"default": state.profileName == state.resolved.DefaultProfile,
				},
			},
		}, nil
	case "tabs":
		return browserActionTabs(state), nil
	case "open":
		return browserActionOpen(ctx, payload, state, connectors)
	case "navigate":
		return browserActionNavigate(ctx, payload, state, connectors)
	case "focus":
		return browserActionFocus(ctx, payload, state)
	case "close":
		return browserActionClose(ctx, payload, state)
	case "snapshot":
		return browserActionSnapshot(ctx, payload, state)
	case "screenshot":
		return browserActionScreenshot(ctx, payload, state)
	case "console":
		return browserActionConsole(payload, state), nil
	case "pdf":
		return browserActionPDF(ctx, payload, state)
	case "upload":
		return browserActionUpload(ctx, payload, state)
	case "dialog":
		return browserActionDialog(ctx, payload, state)
	case "act":
		return browserActionAct(payload, state)
	default:
		return nil, errors.New("browser action not supported: " + action)
	}
}

func browserActionStatus(state *browserProfileState) map[string]any {
	state.mu.Lock()
	defer state.mu.Unlock()

	status := resolveBrowserRuntimeAvailability(state.resolved.PreferredBrowser, state.resolved.Headless)
	if state.runtime != nil {
		status = state.runtime.Status()
	}
	return map[string]any{
		"ok":                     status.Ready,
		"cdpReady":               status.Ready,
		"ready":                  status.Ready,
		"candidates":             status.Candidates,
		"selectedBrowser":        status.SelectedBrowser,
		"chosenBrowser":          status.ChosenBrowser,
		"detectedExecutablePath": status.DetectedExecutablePath,
		"detectError":            status.DetectError,
		"cdpUrl":                 status.CDPURL,
		"cdpPort":                status.CDPPort,
		"headless":               status.Headless,
		"profile":                state.profileName,
		"tabCount":               len(state.tabs),
	}
}

func browserActionTabs(state *browserProfileState) map[string]any {
	state.mu.Lock()
	defer state.mu.Unlock()
	items := make([]map[string]any, 0, len(state.tabs))
	for _, tab := range state.tabs {
		tab.mu.RLock()
		items = append(items, map[string]any{
			"targetId": tab.TargetID,
			"url":      tab.lastURL,
			"active":   tab.TargetID == state.activeTarget,
		})
		tab.mu.RUnlock()
	}
	sort.Slice(items, func(i, j int) bool {
		return fmt.Sprint(items[i]["targetId"]) < fmt.Sprint(items[j]["targetId"])
	})
	return map[string]any{"tabs": items, "activeTarget": state.activeTarget}
}

func browserActionOpen(ctx context.Context, payload toolArgs, state *browserProfileState, connectors ConnectorsReader) (map[string]any, error) {
	targetURL := strings.TrimSpace(getStringArg(payload, "targetUrl", "url"))
	timeoutMs := resolveBrowserActionTimeoutMs(payload, 30000)
	if err := assertBrowserURLAllowed(targetURL, state.resolved.SSRFRules); err != nil {
		return nil, err
	}
	tab, err := createBrowserTab(state)
	if err != nil {
		return nil, err
	}
	if err := addConnectorCookiesToTab(ctx, connectors, tab, targetURL); err != nil {
		return nil, err
	}
	if err := navigateBrowserTab(tab, targetURL, timeoutMs); err != nil {
		return nil, err
	}
	state.mu.Lock()
	state.activeTarget = tab.TargetID
	state.mu.Unlock()
	return map[string]any{
		"ok":       true,
		"targetId": tab.TargetID,
		"url":      browserTabURL(tab),
	}, nil
}

func browserActionNavigate(ctx context.Context, payload toolArgs, state *browserProfileState, connectors ConnectorsReader) (map[string]any, error) {
	targetURL := strings.TrimSpace(getStringArg(payload, "targetUrl", "url"))
	timeoutMs := resolveBrowserActionTimeoutMs(payload, 30000)
	if err := assertBrowserURLAllowed(targetURL, state.resolved.SSRFRules); err != nil {
		return nil, err
	}
	if err := ensureBrowserProfileStarted(state); err != nil {
		return nil, err
	}
	tab, err := resolveBrowserTab(state, strings.TrimSpace(getStringArg(payload, "targetId")), true)
	if err != nil {
		return nil, err
	}
	if err := addConnectorCookiesToTab(ctx, connectors, tab, targetURL); err != nil {
		return nil, err
	}
	if err := navigateBrowserTab(tab, targetURL, timeoutMs); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true, "targetId": tab.TargetID, "url": browserTabURL(tab)}, nil
}

func browserActionFocus(ctx context.Context, payload toolArgs, state *browserProfileState) (map[string]any, error) {
	if err := ensureBrowserProfileStarted(state); err != nil {
		return nil, err
	}
	targetID := strings.TrimSpace(getStringArg(payload, "targetId"))
	tab, err := resolveBrowserTab(state, targetID, false)
	if err != nil {
		return nil, err
	}
	timeout := time.Duration(resolveBrowserActionTimeoutMs(payload, 15000)) * time.Millisecond
	runCtx, cancel := context.WithTimeout(tab.ctx, timeout)
	defer cancel()
	if err := chromedp.Run(runCtx, chromedp.ActionFunc(func(ctx context.Context) error {
		return targetpkg.ActivateTarget(targetpkg.ID(tab.TargetID)).Do(ctx)
	})); err != nil {
		return nil, err
	}
	state.mu.Lock()
	state.activeTarget = tab.TargetID
	state.mu.Unlock()
	return map[string]any{"ok": true, "targetId": tab.TargetID}, nil
}

func browserActionClose(_ context.Context, payload toolArgs, state *browserProfileState) (map[string]any, error) {
	if err := ensureBrowserProfileStarted(state); err != nil {
		return nil, err
	}
	targetID := strings.TrimSpace(getStringArg(payload, "targetId"))
	tab, err := resolveBrowserTab(state, targetID, false)
	if err != nil {
		return nil, err
	}
	tab.cancel()
	timeout := time.Duration(resolveBrowserActionTimeoutMs(payload, 15000)) * time.Millisecond
	runCtx, cancel := context.WithTimeout(state.runtime.BrowserContext(), timeout)
	defer cancel()
	_ = chromedp.Run(runCtx, chromedp.ActionFunc(func(ctx context.Context) error {
		return targetpkg.CloseTarget(targetpkg.ID(tab.TargetID)).Do(ctx)
	}))
	state.mu.Lock()
	delete(state.tabs, tab.TargetID)
	if state.activeTarget == tab.TargetID {
		state.activeTarget = ""
	}
	state.mu.Unlock()
	return map[string]any{"ok": true, "targetId": tab.TargetID}, nil
}

func browserActionSnapshot(ctx context.Context, payload toolArgs, state *browserProfileState) (map[string]any, error) {
	if err := ensureBrowserProfileStarted(state); err != nil {
		return nil, err
	}
	tab, err := resolveBrowserTab(state, strings.TrimSpace(getStringArg(payload, "targetId")), true)
	if err != nil {
		return nil, err
	}
	limit := defaultBrowserSnapshotLimit
	if value, ok := getIntArg(payload, "limit"); ok && value > 0 {
		limit = value
	}
	items, refs, err := collectBrowserSnapshot(ctx, tab, limit)
	if err != nil {
		return nil, err
	}
	tab.mu.Lock()
	tab.refs = refs
	tab.mu.Unlock()
	return map[string]any{
		"ok":       true,
		"targetId": tab.TargetID,
		"url":      browserTabURL(tab),
		"items":    items,
	}, nil
}

func browserActionScreenshot(ctx context.Context, payload toolArgs, state *browserProfileState) (map[string]any, error) {
	if err := ensureBrowserProfileStarted(state); err != nil {
		return nil, err
	}
	tab, err := resolveBrowserTab(state, strings.TrimSpace(getStringArg(payload, "targetId")), true)
	if err != nil {
		return nil, err
	}
	var buf []byte
	timeout := time.Duration(resolveBrowserActionTimeoutMs(payload, 15000)) * time.Millisecond
	runCtx, cancel := context.WithTimeout(tab.ctx, timeout)
	defer cancel()
	fullPage, _ := getBoolArg(payload, "fullPage")
	if err := chromedp.Run(runCtx, chromedp.ActionFunc(func(ctx context.Context) error {
		if fullPage {
			return chromedp.FullScreenshot(&buf, 90).Do(ctx)
		}
		return chromedp.CaptureScreenshot(&buf).Do(ctx)
	})); err != nil {
		return nil, err
	}
	path, err := saveBrowserArtifact("png", buf)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"ok":       true,
		"targetId": tab.TargetID,
		"path":     path,
		"base64":   base64.StdEncoding.EncodeToString(buf),
	}, nil
}

func browserActionConsole(payload toolArgs, state *browserProfileState) map[string]any {
	targetID := strings.TrimSpace(getStringArg(payload, "targetId"))
	state.mu.Lock()
	defer state.mu.Unlock()
	messages := make([]browserConsoleMessage, 0, len(state.consoleMessages))
	for _, item := range state.consoleMessages {
		if targetID != "" && item.TargetID != targetID {
			continue
		}
		messages = append(messages, item)
	}
	return map[string]any{"messages": messages}
}

func browserActionPDF(ctx context.Context, payload toolArgs, state *browserProfileState) (map[string]any, error) {
	if err := ensureBrowserProfileStarted(state); err != nil {
		return nil, err
	}
	tab, err := resolveBrowserTab(state, strings.TrimSpace(getStringArg(payload, "targetId")), true)
	if err != nil {
		return nil, err
	}
	var buf []byte
	timeout := time.Duration(resolveBrowserActionTimeoutMs(payload, 15000)) * time.Millisecond
	runCtx, cancel := context.WithTimeout(tab.ctx, timeout)
	defer cancel()
	if err := chromedp.Run(runCtx, chromedp.ActionFunc(func(ctx context.Context) error {
		var printErr error
		buf, _, printErr = pagepkg.PrintToPDF().WithPrintBackground(true).Do(ctx)
		return printErr
	})); err != nil {
		return nil, err
	}
	path, err := saveBrowserArtifact("pdf", buf)
	if err != nil {
		return nil, err
	}
	return map[string]any{"ok": true, "targetId": tab.TargetID, "path": path}, nil
}

func browserActionUpload(ctx context.Context, payload toolArgs, state *browserProfileState) (map[string]any, error) {
	if err := ensureBrowserProfileStarted(state); err != nil {
		return nil, err
	}
	tab, err := resolveBrowserTab(state, strings.TrimSpace(getStringArg(payload, "targetId")), true)
	if err != nil {
		return nil, err
	}
	request := getMapArg(payload, "request")
	if request == nil {
		request = map[string]any{}
	}
	ref := strings.TrimSpace(getStringArg(toolArgs(request), "ref"))
	if ref == "" {
		ref = strings.TrimSpace(getStringArg(payload, "ref"))
	}
	selector, err := resolveBrowserRefSelector(tab, ref)
	if err != nil {
		return nil, err
	}
	rawPaths := getStringSliceArg(toolArgs(request), "paths")
	if len(rawPaths) == 0 {
		rawPaths = getStringSliceArg(payload, "paths")
	}
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
	if len(paths) == 0 {
		return nil, errors.New("paths are required")
	}
	timeout := time.Duration(resolveBrowserActionTimeoutMs(payload, 15000)) * time.Millisecond
	runCtx, cancel := context.WithTimeout(tab.ctx, timeout)
	defer cancel()
	if err := chromedp.Run(runCtx, chromedp.SetUploadFiles(selector, paths, chromedp.ByQuery)); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true, "targetId": tab.TargetID, "paths": paths}, nil
}

func browserActionDialog(ctx context.Context, payload toolArgs, state *browserProfileState) (map[string]any, error) {
	if err := ensureBrowserProfileStarted(state); err != nil {
		return nil, err
	}
	tab, err := resolveBrowserTab(state, strings.TrimSpace(getStringArg(payload, "targetId")), true)
	if err != nil {
		return nil, err
	}
	targetID := tab.TargetID
	state.mu.Lock()
	pending, exists := state.pendingDialogs[targetID]
	state.mu.Unlock()
	if !exists {
		return map[string]any{"ok": true, "targetId": targetID, "pending": nil}, nil
	}
	if accept, ok := getBoolArg(payload, "accept"); ok {
		promptText := strings.TrimSpace(getStringArg(payload, "promptText"))
		timeout := time.Duration(resolveBrowserActionTimeoutMs(payload, 15000)) * time.Millisecond
		runCtx, cancel := context.WithTimeout(tab.ctx, timeout)
		defer cancel()
		if err := chromedp.Run(runCtx, chromedp.ActionFunc(func(ctx context.Context) error {
			return pagepkg.HandleJavaScriptDialog(accept).WithPromptText(promptText).Do(ctx)
		})); err != nil {
			return nil, err
		}
		state.mu.Lock()
		delete(state.pendingDialogs, targetID)
		state.mu.Unlock()
		return map[string]any{"ok": true, "targetId": targetID}, nil
	}
	return map[string]any{"ok": true, "targetId": targetID, "pending": pending}, nil
}

func browserActionAct(payload toolArgs, state *browserProfileState) (map[string]any, error) {
	request := getMapArg(payload, "request")
	if request == nil {
		return nil, errors.New("request is required")
	}
	kind := strings.ToLower(strings.TrimSpace(getStringArg(toolArgs(request), "kind")))
	if kind == "" {
		return nil, errors.New("request.kind is required")
	}
	if !containsString(browserActKinds, kind) {
		return nil, errors.New("act kind not supported: " + kind)
	}
	if _, hasSelector := request["selector"]; hasSelector && kind != "wait" {
		return nil, errors.New(browserSelectorUnsupportedMessage)
	}
	if err := ensureBrowserProfileStarted(state); err != nil {
		return nil, err
	}
	targetID := strings.TrimSpace(getStringArg(toolArgs(request), "targetId"))
	if targetID == "" {
		targetID = strings.TrimSpace(getStringArg(payload, "targetId"))
	}
	tab, err := resolveBrowserTab(state, targetID, true)
	if err != nil {
		return nil, err
	}
	switch kind {
	case "click":
		err = browserActClick(tab, toolArgs(request), payload)
	case "type":
		err = browserActType(tab, toolArgs(request), payload)
	case "press":
		err = browserActPress(tab, toolArgs(request), payload)
	case "hover":
		err = browserActHover(tab, toolArgs(request), payload)
	case "drag":
		err = errors.New("drag is not implemented yet")
	case "select":
		err = browserActSelect(tab, toolArgs(request), payload)
	case "fill":
		err = browserActFill(tab, toolArgs(request), payload)
	case "resize":
		err = browserActResize(tab, toolArgs(request), payload)
	case "wait":
		err = browserActWait(tab, toolArgs(request), payload)
	case "evaluate":
		err = browserActEvaluate(tab, toolArgs(request), payload)
	case "close":
		_, err = browserActionClose(context.Background(), toolArgs{"targetId": tab.TargetID}, state)
	}
	if err != nil {
		return nil, err
	}
	result := map[string]any{"ok": true, "targetId": tab.TargetID}
	if kind == "evaluate" {
		result["result"] = tabResultFromEvaluate(tab)
	}
	return result, nil
}

func browserActClick(tab *browserTabState, request toolArgs, payload toolArgs) error {
	selector, err := resolveBrowserRefSelector(tab, strings.TrimSpace(getStringArg(request, "ref")))
	if err != nil {
		return err
	}
	timeout := time.Duration(resolveBrowserActTimeoutMs(request, payload, 15000)) * time.Millisecond
	runCtx, cancel := context.WithTimeout(tab.ctx, timeout)
	defer cancel()
	return chromedp.Run(runCtx,
		chromedp.ScrollIntoView(selector, chromedp.ByQuery),
		chromedp.Click(selector, chromedp.ByQuery),
	)
}

func browserActType(tab *browserTabState, request toolArgs, payload toolArgs) error {
	selector, err := resolveBrowserRefSelector(tab, strings.TrimSpace(getStringArg(request, "ref")))
	if err != nil {
		return err
	}
	value := getStringArg(request, "text", "value")
	timeout := time.Duration(resolveBrowserActTimeoutMs(request, payload, 15000)) * time.Millisecond
	runCtx, cancel := context.WithTimeout(tab.ctx, timeout)
	defer cancel()
	return chromedp.Run(runCtx,
		chromedp.Focus(selector, chromedp.ByQuery),
		chromedp.SendKeys(selector, value, chromedp.ByQuery),
	)
}

func browserActPress(tab *browserTabState, request toolArgs, payload toolArgs) error {
	keys := strings.TrimSpace(getStringArg(request, "key", "keys", "text"))
	if keys == "" {
		return errors.New("key is required")
	}
	timeout := time.Duration(resolveBrowserActTimeoutMs(request, payload, 15000)) * time.Millisecond
	runCtx, cancel := context.WithTimeout(tab.ctx, timeout)
	defer cancel()
	return chromedp.Run(runCtx, chromedp.KeyEvent(keys))
}

func browserActHover(tab *browserTabState, request toolArgs, payload toolArgs) error {
	selector, err := resolveBrowserRefSelector(tab, strings.TrimSpace(getStringArg(request, "ref")))
	if err != nil {
		return err
	}
	timeout := time.Duration(resolveBrowserActTimeoutMs(request, payload, 15000)) * time.Millisecond
	runCtx, cancel := context.WithTimeout(tab.ctx, timeout)
	defer cancel()
	return chromedp.Run(runCtx, chromedp.EvaluateAsDevTools(fmt.Sprintf(`(() => { const el = document.querySelector(%q); if (!el) throw new Error("element not found"); el.dispatchEvent(new MouseEvent("mouseover", {bubbles:true})); el.dispatchEvent(new MouseEvent("mouseenter", {bubbles:true})); })()`, selector), nil))
}

func browserActSelect(tab *browserTabState, request toolArgs, payload toolArgs) error {
	selector, err := resolveBrowserRefSelector(tab, strings.TrimSpace(getStringArg(request, "ref")))
	if err != nil {
		return err
	}
	value := getStringArg(request, "value", "text")
	timeout := time.Duration(resolveBrowserActTimeoutMs(request, payload, 15000)) * time.Millisecond
	runCtx, cancel := context.WithTimeout(tab.ctx, timeout)
	defer cancel()
	return chromedp.Run(runCtx, chromedp.EvaluateAsDevTools(fmt.Sprintf(`(() => { const el = document.querySelector(%q); if (!el) throw new Error("element not found"); el.value = %q; el.dispatchEvent(new Event("input", {bubbles:true})); el.dispatchEvent(new Event("change", {bubbles:true})); })()`, selector, value), nil))
}

func browserActFill(tab *browserTabState, request toolArgs, payload toolArgs) error {
	selector, err := resolveBrowserRefSelector(tab, strings.TrimSpace(getStringArg(request, "ref")))
	if err != nil {
		return err
	}
	value := getStringArg(request, "value", "text")
	timeout := time.Duration(resolveBrowserActTimeoutMs(request, payload, 15000)) * time.Millisecond
	runCtx, cancel := context.WithTimeout(tab.ctx, timeout)
	defer cancel()
	return chromedp.Run(runCtx, chromedp.SetValue(selector, value, chromedp.ByQuery))
}

func browserActResize(tab *browserTabState, request toolArgs, payload toolArgs) error {
	width, ok := getIntArg(request, "width")
	if !ok || width <= 0 {
		return errors.New("width is required")
	}
	height, ok := getIntArg(request, "height")
	if !ok || height <= 0 {
		return errors.New("height is required")
	}
	timeout := time.Duration(resolveBrowserActTimeoutMs(request, payload, 15000)) * time.Millisecond
	runCtx, cancel := context.WithTimeout(tab.ctx, timeout)
	defer cancel()
	return chromedp.Run(runCtx, chromedp.ActionFunc(func(ctx context.Context) error {
		return emulation.SetDeviceMetricsOverride(int64(width), int64(height), 1, false).Do(ctx)
	}))
}

func browserActWait(tab *browserTabState, request toolArgs, payload toolArgs) error {
	timeoutMs := resolveBrowserActTimeoutMs(request, payload, 15000)
	if timeMs, ok := getIntArg(request, "timeMs"); ok && timeMs > 0 {
		time.Sleep(time.Duration(timeMs) * time.Millisecond)
		return nil
	}
	if selector := strings.TrimSpace(getStringArg(request, "selector")); selector != "" {
		runCtx, cancel := context.WithTimeout(tab.ctx, time.Duration(timeoutMs)*time.Millisecond)
		defer cancel()
		return chromedp.Run(runCtx, chromedp.WaitVisible(selector, chromedp.ByQuery))
	}
	if text := strings.TrimSpace(getStringArg(request, "text")); text != "" {
		return waitForText(tab.ctx, text, timeoutMs, false)
	}
	if textGone := strings.TrimSpace(getStringArg(request, "textGone")); textGone != "" {
		return waitForText(tab.ctx, textGone, timeoutMs, true)
	}
	if urlWait := strings.TrimSpace(getStringArg(request, "url")); urlWait != "" {
		return waitForURL(tab.ctx, urlWait, timeoutMs)
	}
	if fn := strings.TrimSpace(getStringArg(request, "fn")); fn != "" {
		return waitBrowserEvaluateCondition(tab.ctx, fn, timeoutMs)
	}
	return errors.New(browserWaitRequiresConditionMessage)
}

func browserActEvaluate(tab *browserTabState, request toolArgs, payload toolArgs) error {
	expression := strings.TrimSpace(getStringArg(request, "expression", "fn", "script"))
	if expression == "" {
		return errors.New("expression is required")
	}
	timeout := time.Duration(resolveBrowserActTimeoutMs(request, payload, 15000)) * time.Millisecond
	runCtx, cancel := context.WithTimeout(tab.ctx, timeout)
	defer cancel()
	var result any
	if err := chromedp.Run(runCtx, chromedp.Evaluate(expression, &result)); err != nil {
		return err
	}
	tab.mu.Lock()
	tab.evaluateResult = result
	tab.mu.Unlock()
	return nil
}

func ensureBrowserProfileStarted(state *browserProfileState) error {
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.runtime != nil && state.runtime.Status().Ready {
		return nil
	}
	userDataDir := filepath.Join(os.TempDir(), "dreamcreator", "browser", state.sessionKey, state.profileName)
	startCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	runtime, err := browsercdp.Start(startCtx, browsercdp.LaunchOptions{
		PreferredBrowser: state.resolved.PreferredBrowser,
		Headless:         state.resolved.Headless,
		UserDataDir:      userDataDir,
	})
	if err != nil {
		return err
	}
	state.runtime = runtime
	return nil
}

func stopBrowserProfile(state *browserProfileState) {
	state.mu.Lock()
	defer state.mu.Unlock()
	for _, tab := range state.tabs {
		tab.cancel()
	}
	state.tabs = map[string]*browserTabState{}
	state.activeTarget = ""
	if state.runtime != nil {
		state.runtime.Stop()
		state.runtime = nil
	}
}

func createBrowserTab(state *browserProfileState) (*browserTabState, error) {
	if err := ensureBrowserProfileStarted(state); err != nil {
		return nil, err
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	tabCtx, cancel := chromedp.NewContext(state.runtime.BrowserContext())
	tab := &browserTabState{
		ctx:    tabCtx,
		cancel: cancel,
		refs:   map[string]browserSnapshotRef{},
	}
	runCtx, runCancel := context.WithTimeout(tabCtx, 10*time.Second)
	defer runCancel()
	if err := chromedp.Run(runCtx, chromedp.ActionFunc(func(ctx context.Context) error {
		targetID := string(chromedp.FromContext(ctx).Target.TargetID)
		tab.TargetID = targetID
		return nil
	})); err != nil {
		cancel()
		return nil, err
	}
	attachBrowserTab(state, tab)
	state.tabs[tab.TargetID] = tab
	state.activeTarget = tab.TargetID
	return tab, nil
}

func attachBrowserTab(state *browserProfileState, tab *browserTabState) {
	chromedp.ListenTarget(tab.ctx, func(ev any) {
		switch event := ev.(type) {
		case *runtimepkg.EventConsoleAPICalled:
			state.mu.Lock()
			state.consoleMessages = append(state.consoleMessages, browserConsoleMessage{
				TargetID:  tab.TargetID,
				Type:      string(event.Type),
				Text:      runtimeConsoleMessageText(event),
				Timestamp: time.Now().Format(time.RFC3339),
			})
			state.mu.Unlock()
		case *pagepkg.EventJavascriptDialogOpening:
			state.mu.Lock()
			state.pendingDialogs[tab.TargetID] = browserPendingDialog{
				Message:   strings.TrimSpace(event.Message),
				Type:      string(event.Type),
				ExpiresAt: time.Now().Add(5 * time.Minute),
			}
			state.mu.Unlock()
		}
	})
}

func resolveBrowserTab(state *browserProfileState, targetID string, allowActive bool) (*browserTabState, error) {
	state.mu.Lock()
	defer state.mu.Unlock()
	targetID = strings.TrimSpace(targetID)
	if targetID != "" {
		tab, ok := state.tabs[targetID]
		if !ok {
			return nil, errors.New("tab not found")
		}
		return tab, nil
	}
	if allowActive && state.activeTarget != "" {
		if tab, ok := state.tabs[state.activeTarget]; ok {
			return tab, nil
		}
	}
	for _, tab := range state.tabs {
		return tab, nil
	}
	return nil, errors.New("no browser tab is open")
}

func navigateBrowserTab(tab *browserTabState, targetURL string, timeoutMs int) error {
	timeout := time.Duration(normalizeBrowserTimeoutMs(timeoutMs, 30000)) * time.Millisecond
	runCtx, cancel := context.WithTimeout(tab.ctx, timeout)
	defer cancel()
	var finalURL string
	if err := chromedp.Run(runCtx,
		chromedp.Navigate(targetURL),
		chromedp.WaitReady("body", chromedp.ByQuery),
		chromedp.Location(&finalURL),
	); err != nil {
		return err
	}
	tab.mu.Lock()
	tab.lastURL = finalURL
	tab.refs = map[string]browserSnapshotRef{}
	tab.mu.Unlock()
	return nil
}

func addConnectorCookiesToTab(ctx context.Context, connectors ConnectorsReader, tab *browserTabState, targetURL string) error {
	if connectors == nil || tab == nil {
		return nil
	}
	cookies, err := resolveConnectorCookiesForURL(ctx, connectors, targetURL)
	if err != nil || len(cookies) == 0 {
		return err
	}
	runCtx, cancel := context.WithTimeout(tab.ctx, 10*time.Second)
	defer cancel()
	return chromedp.Run(runCtx, chromedp.ActionFunc(func(ctx context.Context) error {
		return browsercdp.SetCookies(ctx, targetURL, connectorCookiesToRecords(cookies))
	}))
}

func collectBrowserSnapshot(ctx context.Context, tab *browserTabState, limit int) ([]browserSnapshotItem, map[string]browserSnapshotRef, error) {
	if limit <= 0 {
		limit = defaultBrowserSnapshotLimit
	}
	script := fmt.Sprintf(`(() => {
  const inferRole = (el) => {
    const explicit = (el.getAttribute("role") || "").trim();
    if (explicit) return explicit.toLowerCase();
    const tag = el.tagName.toLowerCase();
    if (tag === "a") return "link";
    if (tag === "button") return "button";
    if (tag === "textarea" || tag === "input") return "textbox";
    if (tag === "select") return "combobox";
    return "element";
  };
  const cssPath = (el) => {
    if (el.id) return "#" + CSS.escape(el.id);
    const parts = [];
    let node = el;
    while (node && node.nodeType === 1 && parts.length < 6) {
      let part = node.tagName.toLowerCase();
      if (node.classList && node.classList.length > 0) {
        part += "." + Array.from(node.classList).slice(0, 2).map((item) => CSS.escape(item)).join(".");
      }
      const parent = node.parentElement;
      if (parent) {
        const siblings = Array.from(parent.children).filter((candidate) => candidate.tagName === node.tagName);
        if (siblings.length > 1) {
          part += ":nth-of-type(" + (siblings.indexOf(node) + 1) + ")";
        }
      }
      parts.unshift(part);
      node = parent;
    }
    return parts.join(" > ");
  };
  const visible = (el) => {
    const style = window.getComputedStyle(el);
    const rect = el.getBoundingClientRect();
    return style && style.visibility !== "hidden" && style.display !== "none" && rect.width > 0 && rect.height > 0;
  };
  const candidates = Array.from(document.querySelectorAll('a,button,input,textarea,select,summary,[role="button"],[role="link"],[role="menuitem"],[tabindex]'))
    .filter((el) => visible(el))
    .slice(0, %d)
    .map((el) => {
      const text = (el.innerText || el.textContent || el.value || "").replace(/\s+/g, " ").trim();
      const name = (el.getAttribute("aria-label") || el.getAttribute("title") || el.placeholder || text).replace(/\s+/g, " ").trim();
      return {
        selector: cssPath(el),
        role: inferRole(el),
        name,
        text,
      };
    });
  return candidates;
})()`, limit)
	var raw []browserSnapshotJSItem
	runCtx, cancel := context.WithTimeout(tab.ctx, 10*time.Second)
	defer cancel()
	if err := chromedp.Run(runCtx, chromedp.EvaluateAsDevTools(script, &raw)); err != nil {
		return nil, nil, err
	}
	items := make([]browserSnapshotItem, 0, len(raw))
	refs := map[string]browserSnapshotRef{}
	countByRoleName := map[string]int{}
	for index, item := range raw {
		ref := fmt.Sprintf("e%d", index+1)
		key := item.Role + "\n" + item.Name
		nth := countByRoleName[key]
		countByRoleName[key] = nth + 1
		items = append(items, browserSnapshotItem{
			Ref:   ref,
			Role:  item.Role,
			Name:  item.Name,
			Text:  item.Text,
			Nth:   nth,
			Depth: 0,
		})
		refs[ref] = browserSnapshotRef{
			Selector: item.Selector,
			Role:     item.Role,
			Name:     item.Name,
			Nth:      nth,
			Mode:     "css",
		}
	}
	return items, refs, nil
}

func resolveBrowserRefSelector(tab *browserTabState, ref string) (string, error) {
	if tab == nil {
		return "", errors.New("tab unavailable")
	}
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", errors.New("ref is required")
	}
	tab.mu.RLock()
	defer tab.mu.RUnlock()
	item, ok := tab.refs[ref]
	if !ok {
		return "", errors.New("ref not found; run snapshot first")
	}
	if strings.TrimSpace(item.Selector) == "" {
		return "", errors.New("ref selector unavailable")
	}
	return item.Selector, nil
}

func browserTabURL(tab *browserTabState) string {
	if tab == nil {
		return ""
	}
	tab.mu.RLock()
	defer tab.mu.RUnlock()
	return tab.lastURL
}

func waitForText(ctx context.Context, expected string, timeoutMs int, gone bool) error {
	deadline := time.Now().Add(time.Duration(timeoutMs) * time.Millisecond)
	expected = strings.TrimSpace(expected)
	for {
		var bodyText string
		_ = chromedp.Run(ctx, chromedp.Text("body", &bodyText, chromedp.ByQuery))
		contains := strings.Contains(bodyText, expected)
		if (!gone && contains) || (gone && !contains) {
			return nil
		}
		if time.Now().After(deadline) {
			return errors.New("wait text timeout")
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func waitForURL(ctx context.Context, expected string, timeoutMs int) error {
	deadline := time.Now().Add(time.Duration(timeoutMs) * time.Millisecond)
	expected = strings.TrimSpace(expected)
	for {
		var current string
		_ = chromedp.Run(ctx, chromedp.Location(&current))
		if current == expected {
			return nil
		}
		if time.Now().After(deadline) {
			return errors.New("wait url timeout")
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func waitBrowserEvaluateCondition(ctx context.Context, fn string, timeoutMs int) error {
	if timeoutMs <= 0 {
		timeoutMs = 15000
	}
	deadline := time.Now().Add(time.Duration(timeoutMs) * time.Millisecond)
	for {
		var result any
		err := chromedp.Run(ctx, chromedp.Evaluate(fn, &result))
		if err == nil {
			switch typed := result.(type) {
			case bool:
				if typed {
					return nil
				}
			case string:
				if strings.TrimSpace(strings.ToLower(typed)) == "true" {
					return nil
				}
			case float64:
				if typed != 0 {
					return nil
				}
			}
		}
		if time.Now().After(deadline) {
			if err != nil {
				return err
			}
			return errors.New("wait fn timeout")
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func runtimeConsoleMessageText(event *runtimepkg.EventConsoleAPICalled) string {
	if event == nil || len(event.Args) == 0 {
		return ""
	}
	parts := make([]string, 0, len(event.Args))
	for _, arg := range event.Args {
		if arg == nil {
			continue
		}
		if arg.Value != nil {
			parts = append(parts, fmt.Sprint(arg.Value))
			continue
		}
		if strings.TrimSpace(arg.Description) != "" {
			parts = append(parts, strings.TrimSpace(arg.Description))
			continue
		}
		parts = append(parts, strings.TrimSpace(arg.Type.String()))
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}
