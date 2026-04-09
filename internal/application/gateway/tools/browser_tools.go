package tools

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	gatewaynodes "dreamcreator/internal/application/gateway/nodes"
	"github.com/playwright-community/playwright-go"
)

const (
	browserTypePlaywright = "playwright"
	defaultBrowserType    = browserTypePlaywright

	defaultBrowserWaitUntil = "domcontentloaded"

	defaultBrowserSnapshotModeEfficient = "efficient"

	defaultBrowserProfileDreamCreator = "dreamcreator"
	defaultBrowserColor               = "#FF4500"

	defaultBrowserSnapshotAIMaxChars          = 80000
	defaultBrowserSnapshotAIEfficientMaxChars = 10000
	defaultBrowserSnapshotDepth               = 6
	defaultBrowserSnapshotLimit               = 200
	defaultBrowserViewportWidth               = 1366
	defaultBrowserViewportHeight              = 900

	defaultBrowserHookTimeoutMs = 20000

	browserRuntimeCheckCacheTTL = 10 * time.Second
)

var browserToolActions = []string{
	"status",
	"start",
	"stop",
	"profiles",
	"tabs",
	"open",
	"focus",
	"close",
	"snapshot",
	"screenshot",
	"navigate",
	"console",
	"pdf",
	"upload",
	"dialog",
	"act",
}

var browserSelectorUnsupportedMessage = strings.Join([]string{
	"Error: 'selector' is not supported. Use 'ref' from snapshot instead.",
	"",
	"Example workflow:",
	"1. snapshot action to get page state with refs",
	`2. act with ref: "e123" to interact with element`,
	"",
	"This is more reliable for modern SPAs.",
}, "\n")

var browserWaitFnDisabledMessage = strings.Join([]string{
	"wait --fn is disabled by config (browser.evaluateEnabled=false).",
	"Docs: /gateway/configuration#browser-playwright-managed-browser",
}, "\n")

var browserEvaluateDisabledMessage = strings.Join([]string{
	"act:evaluate is disabled by config (browser.evaluateEnabled=false).",
	"Docs: /gateway/configuration#browser-playwright-managed-browser",
}, "\n")

var browserWaitRequiresConditionMessage = "wait requires at least one of: timeMs, text, textGone, selector, url, loadState, fn"

var errBrowserSnapshotForAIUnavailable = errors.New("playwright snapshotForAI is unavailable")

var browserActKinds = []string{
	"click",
	"type",
	"press",
	"hover",
	"drag",
	"select",
	"fill",
	"resize",
	"wait",
	"evaluate",
	"close",
}

var browserPlaywrightRuntimeCache = struct {
	mu        sync.Mutex
	checkedAt time.Time
	available bool
	reason    string
	execPath  string
}{}

var browserGlobalTabCounter uint64

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

	profileName string
	resolved    browserResolvedConfig
	profile     browserProfileConfig

	pw      *playwright.Playwright
	browser playwright.Browser
	context playwright.BrowserContext

	tabs         map[string]*browserTabState
	pageToTarget map[playwright.Page]string
	activeTarget string

	consoleMessages []browserConsoleMessage
	pendingUploads  map[string]browserPendingUpload
	pendingDialogs  map[string]browserPendingDialog
}

type browserTabState struct {
	TargetID string
	Page     playwright.Page

	mu             sync.RWMutex
	refs           map[string]browserSnapshotRef
	evaluateResult any
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
	Accept     bool
	PromptText string
	ExpiresAt  time.Time
}

type browserResolvedConfig struct {
	Enabled                     bool
	EvaluateEnabled             bool
	CDPURL                      string
	RemoteCdpTimeoutMs          int
	RemoteCdpHandshakeTimeoutMs int
	Color                       string
	Headless                    bool
	NoSandbox                   bool
	AttachOnly                  bool
	DefaultProfile              string
	Profiles                    map[string]browserProfileConfig
	SnapshotDefaultMode         string
	SSRFRules                   browserSSRFPolicy
	ExtraArgs                   []string
}

type browserProfileConfig struct {
	Name    string
	CDPURL  string
	CDPPort int
	Color   string
	Driver  string
}

type browserSSRFPolicy struct {
	DangerouslyAllowPrivateNetwork bool
	AllowedHostnames               map[string]struct{}
	HostnameAllowlist              []string
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

func resolveBrowserAction(payload toolArgs) (string, error) {
	rawAction := strings.ToLower(strings.TrimSpace(getStringArg(payload, "action", "method")))
	if rawAction == "" {
		if getStringArg(payload, "targetUrl", "url") != "" {
			rawAction = "open"
		} else {
			rawAction = "status"
		}
	}
	switch rawAction {
	case "navigate", "status", "start", "stop", "profiles", "tabs", "open", "focus", "close", "snapshot", "screenshot", "console", "pdf", "upload", "dialog", "act":
		return rawAction, nil
	default:
		return "", errors.New("browser action not supported: " + rawAction)
	}
}

func isBrowserNodeTargetRequest(payload toolArgs) bool {
	target := strings.ToLower(strings.TrimSpace(getStringArg(payload, "target")))
	nodeID := strings.TrimSpace(getStringArg(payload, "node", "nodeId"))
	if target == "node" {
		return true
	}
	return nodeID != ""
}

func resolveBrowserNodeID(ctx context.Context, payload toolArgs, nodes *gatewaynodes.Service) (string, error) {
	requestedNode := strings.TrimSpace(getStringArg(payload, "node", "nodeId"))
	if requestedNode != "" {
		return requestedNode, nil
	}
	if nodes == nil {
		return "", errors.New("nodes service unavailable")
	}
	list, err := nodes.ListNodes(ctx)
	if err != nil {
		return "", err
	}
	for _, descriptor := range list {
		nodeID := strings.TrimSpace(descriptor.NodeID)
		if nodeID == "" {
			continue
		}
		for _, capability := range descriptor.Capabilities {
			if strings.EqualFold(strings.TrimSpace(capability.Name), "browser.control") {
				return nodeID, nil
			}
		}
	}
	for _, descriptor := range list {
		nodeID := strings.TrimSpace(descriptor.NodeID)
		if nodeID != "" {
			return nodeID, nil
		}
	}
	return "", errors.New("nodeId is required")
}

func runBrowserActionOnNode(ctx context.Context, payload toolArgs, action string, nodes *gatewaynodes.Service) (string, error) {
	if nodes == nil {
		return "", errors.New("nodes service unavailable")
	}
	target := strings.ToLower(strings.TrimSpace(getStringArg(payload, "target")))
	if target != "" && target != "node" {
		return "", errors.New(`node is only supported with target="node"`)
	}
	nodeID, err := resolveBrowserNodeID(ctx, payload, nodes)
	if err != nil {
		return "", err
	}
	argsJSON, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	request := gatewaynodes.NodeInvokeRequest{
		NodeID:     nodeID,
		Capability: "browser.control",
		Action:     action,
		Args:       string(argsJSON),
		TimeoutMs:  resolveBrowserActionTimeoutMs(payload, 30000),
	}
	result, invokeErr := nodes.Invoke(ctx, request)
	if invokeErr != nil {
		return marshalResult(result), invokeErr
	}
	if !result.Ok {
		if strings.TrimSpace(result.Error) != "" {
			return marshalResult(result), errors.New(strings.TrimSpace(result.Error))
		}
		return marshalResult(result), errors.New("node browser invoke failed")
	}
	if parsed := resolveBrowserNodeOutput(result.Output); parsed != nil {
		return marshalResult(parsed), nil
	}
	return marshalResult(result), nil
}

type browserNodeProxyEnvelope struct {
	Result any                    `json:"result"`
	Files  []browserNodeProxyFile `json:"files"`
}

type browserNodeProxyFile struct {
	Path     string `json:"path"`
	Base64   string `json:"base64"`
	MimeType string `json:"mimeType"`
}

func resolveBrowserNodeOutput(output string) any {
	trimmedOutput := strings.TrimSpace(output)
	if trimmedOutput == "" {
		return nil
	}
	var parsed any
	if err := json.Unmarshal([]byte(trimmedOutput), &parsed); err != nil {
		return nil
	}

	envelope := browserNodeProxyEnvelope{}
	if err := json.Unmarshal([]byte(trimmedOutput), &envelope); err == nil && envelope.Result != nil {
		mapping := persistBrowserNodeProxyFiles(envelope.Files)
		applyBrowserProxyPathMapping(envelope.Result, mapping)
		return envelope.Result
	}
	return parsed
}

func persistBrowserNodeProxyFiles(files []browserNodeProxyFile) map[string]string {
	if len(files) == 0 {
		return nil
	}
	mapping := map[string]string{}
	for _, file := range files {
		remotePath := strings.TrimSpace(file.Path)
		encoded := strings.TrimSpace(file.Base64)
		if remotePath == "" || encoded == "" {
			continue
		}
		bytes, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			continue
		}
		localPath, err := saveBrowserArtifact(resolveBrowserProxyFileExt(file), bytes)
		if err != nil {
			continue
		}
		mapping[remotePath] = localPath
	}
	if len(mapping) == 0 {
		return nil
	}
	return mapping
}

func resolveBrowserProxyFileExt(file browserNodeProxyFile) string {
	ext := strings.TrimSpace(strings.TrimPrefix(filepath.Ext(strings.TrimSpace(file.Path)), "."))
	if ext != "" {
		return ext
	}
	mimeType := strings.ToLower(strings.TrimSpace(file.MimeType))
	switch {
	case strings.Contains(mimeType, "png"):
		return "png"
	case strings.Contains(mimeType, "jpeg"), strings.Contains(mimeType, "jpg"):
		return "jpg"
	case strings.Contains(mimeType, "pdf"):
		return "pdf"
	case strings.Contains(mimeType, "json"):
		return "json"
	case strings.Contains(mimeType, "text"), strings.Contains(mimeType, "plain"):
		return "txt"
	default:
		return "bin"
	}
}

func applyBrowserProxyPathMapping(result any, mapping map[string]string) {
	if len(mapping) == 0 || result == nil {
		return
	}
	obj, ok := result.(map[string]any)
	if !ok {
		return
	}
	if pathValue, ok := obj["path"].(string); ok {
		if mapped, exists := mapping[pathValue]; exists {
			obj["path"] = mapped
		}
	}
	if imagePathValue, ok := obj["imagePath"].(string); ok {
		if mapped, exists := mapping[imagePathValue]; exists {
			obj["imagePath"] = mapped
		}
	}
	if downloadRaw, exists := obj["download"]; exists {
		if downloadObj, ok := downloadRaw.(map[string]any); ok {
			if pathValue, ok := downloadObj["path"].(string); ok {
				if mapped, exists := mapping[pathValue]; exists {
					downloadObj["path"] = mapped
				}
			}
		}
	}
}

func resolveBrowserSessionKey(ctx context.Context, payload toolArgs) string {
	sessionKey, _ := RuntimeContextFromContext(ctx)
	sessionKey = strings.TrimSpace(sessionKey)
	if sessionKey == "" {
		sessionKey = strings.TrimSpace(getStringArg(payload, "sessionKey", "session_key"))
	}
	if sessionKey == "" {
		sessionKey = "default"
	}
	return sessionKey
}

func runBrowserAction(
	ctx context.Context,
	payload toolArgs,
	action string,
	state *browserProfileState,
	connectors ConnectorsReader,
) (any, error) {
	switch action {
	case "status":
		return browserActionStatus(state)
	case "start":
		if err := ensureBrowserProfileStarted(state); err != nil {
			return nil, err
		}
		return browserActionStatus(state)
	case "stop":
		if err := stopBrowserProfile(state); err != nil {
			return nil, err
		}
		return browserActionStatus(state)
	case "profiles":
		return browserActionProfiles(state), nil
	case "tabs":
		return browserActionTabs(state)
	case "open":
		return browserActionOpen(ctx, payload, state, connectors)
	case "focus":
		return browserActionFocus(payload, state)
	case "close":
		return browserActionClose(payload, state)
	case "snapshot":
		return browserActionSnapshot(payload, state)
	case "screenshot":
		return browserActionScreenshot(payload, state)
	case "navigate":
		return browserActionNavigate(ctx, payload, state, connectors)
	case "console":
		return browserActionConsole(payload, state)
	case "pdf":
		return browserActionPDF(payload, state)
	case "upload":
		return browserActionUpload(payload, state)
	case "dialog":
		return browserActionDialog(payload, state)
	case "act":
		return browserActionAct(payload, state)
	default:
		return nil, errors.New("browser action not supported: " + action)
	}
}

func browserActionStatus(state *browserProfileState) (map[string]any, error) {
	available, reason, execPath := resolveBrowserPlaywrightRuntimeAvailability()

	state.mu.Lock()
	defer state.mu.Unlock()
	pruneClosedTabsLocked(state)

	running := state.browser != nil && state.context != nil
	detectedPath := any(nil)
	if strings.TrimSpace(execPath) != "" {
		detectedPath = strings.TrimSpace(execPath)
	}
	detectError := any(nil)
	if !available && strings.TrimSpace(reason) != "" {
		detectError = strings.TrimSpace(reason)
	}
	chosenBrowser := any(nil)
	if running {
		chosenBrowser = "chromium"
	}

	return map[string]any{
		"enabled":                state.resolved.Enabled,
		"profile":                state.profileName,
		"running":                running,
		"cdpReady":               false,
		"cdpHttp":                false,
		"pid":                    nil,
		"cdpPort":                nil,
		"cdpUrl":                 nil,
		"chosenBrowser":          chosenBrowser,
		"detectedBrowser":        "chromium",
		"detectedExecutablePath": detectedPath,
		"detectError":            detectError,
		"userDataDir":            nil,
		"color":                  state.profile.Color,
		"headless":               state.resolved.Headless,
		"noSandbox":              state.resolved.NoSandbox,
		"executablePath":         detectedPath,
		"attachOnly":             false,
		"tabCount":               len(state.tabs),
		"activeTargetId":         state.activeTarget,
	}, nil
}

func browserActionProfiles(state *browserProfileState) map[string]any {
	state.mu.Lock()
	defer state.mu.Unlock()
	pruneClosedTabsLocked(state)

	names := make([]string, 0, len(state.resolved.Profiles))
	for name := range state.resolved.Profiles {
		names = append(names, name)
	}
	sort.Strings(names)

	profiles := make([]map[string]any, 0, len(names))
	for _, name := range names {
		cfg := state.resolved.Profiles[name]
		isCurrent := name == state.profileName
		running := isCurrent && state.browser != nil && state.context != nil
		tabCount := 0
		if isCurrent {
			tabCount = len(state.tabs)
		}
		profiles = append(profiles, map[string]any{
			"name":      name,
			"cdpPort":   cfg.CDPPort,
			"cdpUrl":    cfg.CDPURL,
			"color":     cfg.Color,
			"running":   running,
			"tabCount":  tabCount,
			"isDefault": name == state.resolved.DefaultProfile,
			"isRemote":  cfg.CDPURL != "",
		})
	}

	return map[string]any{"profiles": profiles}
}

func browserActionTabs(state *browserProfileState) (map[string]any, error) {
	state.mu.Lock()
	running := state.browser != nil && state.context != nil
	state.mu.Unlock()
	if !running {
		return map[string]any{"running": false, "tabs": []any{}}, nil
	}

	tabs, err := listBrowserTabs(state)
	if err != nil {
		return nil, err
	}
	return map[string]any{"running": true, "tabs": tabs}, nil
}

func browserActionOpen(ctx context.Context, payload toolArgs, state *browserProfileState, connectors ConnectorsReader) (map[string]any, error) {
	targetURL := getStringArg(payload, "targetUrl", "url")
	if targetURL == "" {
		return nil, errors.New("targetUrl is required")
	}
	if err := assertBrowserURLAllowed(targetURL, state.resolved.SSRFRules); err != nil {
		return nil, err
	}
	if err := ensureBrowserProfileStarted(state); err != nil {
		return nil, err
	}

	state.mu.Lock()
	browserCtx := state.context
	state.mu.Unlock()
	if browserCtx == nil {
		return nil, errors.New("browser context unavailable")
	}

	if err := addConnectorCookiesToContext(ctx, connectors, browserCtx, targetURL); err != nil {
		return nil, err
	}

	page, err := browserCtx.NewPage()
	if err != nil {
		return nil, err
	}
	if _, err := page.Goto(strings.TrimSpace(targetURL), playwright.PageGotoOptions{
		Timeout:   playwright.Float(float64(resolveBrowserActionTimeoutMs(payload, 30000))),
		WaitUntil: resolveBrowserWaitUntilState(resolveBrowserWaitUntil(payload, defaultBrowserWaitUntil)),
	}); err != nil {
		_ = page.Close()
		return nil, err
	}

	tab := attachBrowserTab(state, page)
	title, _ := page.Title()
	urlValue := strings.TrimSpace(page.URL())
	if urlValue == "" {
		urlValue = strings.TrimSpace(targetURL)
	}
	if err := assertBrowserURLAllowed(urlValue, state.resolved.SSRFRules); err != nil {
		return nil, err
	}

	return map[string]any{
		"targetId": tab.TargetID,
		"title":    strings.TrimSpace(title),
		"url":      urlValue,
		"type":     "page",
	}, nil
}

func browserActionFocus(payload toolArgs, state *browserProfileState) (map[string]any, error) {
	targetID := strings.TrimSpace(getStringArg(payload, "targetId"))
	if targetID == "" {
		return nil, errors.New("targetId is required")
	}
	tab, err := resolveBrowserTab(state, targetID, false)
	if err != nil {
		return nil, err
	}
	if err := tab.Page.BringToFront(); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true, "targetId": tab.TargetID}, nil
}

func browserActionClose(payload toolArgs, state *browserProfileState) (map[string]any, error) {
	targetID := strings.TrimSpace(getStringArg(payload, "targetId"))
	tab, err := resolveBrowserTab(state, targetID, false)
	if err != nil {
		return nil, err
	}
	if err := tab.Page.Close(); err != nil {
		return nil, err
	}

	state.mu.Lock()
	delete(state.tabs, tab.TargetID)
	delete(state.pageToTarget, tab.Page)
	if state.activeTarget == tab.TargetID {
		state.activeTarget = ""
	}
	pruneClosedTabsLocked(state)
	state.mu.Unlock()

	return map[string]any{"ok": true, "targetId": tab.TargetID}, nil
}

func browserActionSnapshot(payload toolArgs, state *browserProfileState) (map[string]any, error) {
	if err := ensureBrowserProfileStarted(state); err != nil {
		return nil, err
	}
	targetID := strings.TrimSpace(getStringArg(payload, "targetId"))
	tab, err := resolveBrowserTab(state, targetID, true)
	if err != nil {
		return nil, err
	}

	format := strings.ToLower(strings.TrimSpace(getStringArg(payload, "snapshotFormat", "format")))
	if format != "aria" {
		format = "ai"
	}
	refsMode := strings.ToLower(strings.TrimSpace(getStringArg(payload, "refs")))
	if refsMode != "aria" {
		refsMode = "role"
	}
	mode := strings.ToLower(strings.TrimSpace(getStringArg(payload, "mode")))
	if mode == "" {
		mode = strings.TrimSpace(state.resolved.SnapshotDefaultMode)
	}
	if mode != defaultBrowserSnapshotModeEfficient {
		mode = ""
	}
	labels, _ := getBoolArg(payload, "labels")
	if format == "aria" && (labels || mode == defaultBrowserSnapshotModeEfficient) {
		return nil, errors.New("labels/mode=efficient require format=ai")
	}

	interactive, hasInteractive := getBoolArg(payload, "interactive")
	if !hasInteractive {
		interactive = mode == defaultBrowserSnapshotModeEfficient
	}
	compact, hasCompact := getBoolArg(payload, "compact")
	if !hasCompact {
		compact = mode == defaultBrowserSnapshotModeEfficient
	}

	depth, hasDepth := getIntArg(payload, "depth")
	if hasDepth && depth <= 0 {
		hasDepth = false
	}
	if !hasDepth && mode == defaultBrowserSnapshotModeEfficient {
		depth = defaultBrowserSnapshotDepth
		hasDepth = true
	}

	limit, ok := getIntArg(payload, "limit")
	if !ok || limit <= 0 {
		limit = defaultBrowserSnapshotLimit
	}

	maxChars := 0
	_, hasMaxChars := payload["maxChars"]
	if value, ok := getIntArg(payload, "maxChars"); ok && value > 0 {
		maxChars = value
	}
	if format == "ai" && !hasMaxChars {
		if mode == defaultBrowserSnapshotModeEfficient {
			maxChars = defaultBrowserSnapshotAIEfficientMaxChars
		} else {
			maxChars = defaultBrowserSnapshotAIMaxChars
		}
	}

	selector := strings.TrimSpace(getStringArg(payload, "selector"))
	frameSelector := strings.TrimSpace(getStringArg(payload, "frame"))
	if refsMode == "aria" && (selector != "" || frameSelector != "") {
		return nil, errors.New("refs=aria does not support selector/frame snapshots yet")
	}

	maxDepth := 0
	if hasDepth {
		maxDepth = depth
	}
	wantsRoleSnapshot := labels ||
		mode == defaultBrowserSnapshotModeEfficient ||
		interactive ||
		compact ||
		hasDepth ||
		selector != "" ||
		frameSelector != ""
	if format == "ai" && !wantsRoleSnapshot {
		aiResult, aiErr := collectBrowserAISnapshot(tab.Page, maxChars)
		if aiErr == nil {
			state.mu.Lock()
			tab.refs = aiResult.Refs
			state.mu.Unlock()

			result := map[string]any{
				"ok":        true,
				"format":    "ai",
				"targetId":  tab.TargetID,
				"url":       strings.TrimSpace(tab.Page.URL()),
				"snapshot":  aiResult.Snapshot,
				"truncated": aiResult.Truncated,
				"refs":      aiResult.RefsJSON,
				"stats":     aiResult.Stats,
			}
			return result, nil
		}
		if !isBrowserSnapshotForAIUnavailable(aiErr) {
			return nil, aiErr
		}
	}

	items, err := collectBrowserSnapshotItems(tab.Page, selector, frameSelector, interactive, limit, refsMode, maxDepth)
	if err != nil {
		return nil, err
	}

	refs := make(map[string]browserSnapshotRef, len(items))
	refsJSON := make(map[string]map[string]any, len(items))
	lines := make([]string, 0, len(items))
	nodes := make([]map[string]any, 0, len(items))
	interactiveCount := 0
	for index, item := range items {
		if index >= limit {
			break
		}
		ref := strings.TrimSpace(item.Ref)
		if ref == "" {
			ref = fmt.Sprintf("e%d", index+1)
		}
		entryMode := refsMode
		if entryMode == "aria" && strings.TrimSpace(item.AriaRef) == "" {
			entryMode = "role"
		}
		entry := browserSnapshotRef{
			Role:    item.Role,
			Name:    item.Name,
			Nth:     item.Nth,
			Mode:    entryMode,
			AriaRef: item.AriaRef,
			Frame:   frameSelector,
		}
		refs[ref] = entry
		refsJSON[ref] = map[string]any{
			"role": entry.Role,
			"name": entry.Name,
		}
		if entry.Nth > 0 {
			refsJSON[ref]["nth"] = entry.Nth
		}
		if isBrowserInteractiveRole(entry.Role) {
			interactiveCount += 1
		}
		line := fmt.Sprintf("[%s] role=%s name=%s", ref, entry.Role, entry.Name)
		if !compact && strings.TrimSpace(item.Text) != "" {
			line += " text=" + trimToMaxChars(item.Text, 120)
		}
		lines = append(lines, line)
		nodeDepth := item.Depth
		if maxDepth > 0 {
			nodeDepth = minInt(nodeDepth, maxDepth)
		}
		nodes = append(nodes, map[string]any{
			"ref":   ref,
			"role":  entry.Role,
			"name":  entry.Name,
			"depth": nodeDepth,
		})
	}

	state.mu.Lock()
	tab.refs = refs
	state.mu.Unlock()

	if format == "aria" {
		result := map[string]any{
			"ok":       true,
			"format":   "aria",
			"targetId": tab.TargetID,
			"url":      strings.TrimSpace(tab.Page.URL()),
			"nodes":    nodes,
		}
		return result, nil
	}

	snapshot := strings.Join(lines, "\n")
	truncated := false
	if maxChars > 0 && len(snapshot) > maxChars {
		snapshot = trimToMaxChars(snapshot, maxChars)
		truncated = true
	}

	result := map[string]any{
		"ok":        true,
		"format":    "ai",
		"targetId":  tab.TargetID,
		"url":       strings.TrimSpace(tab.Page.URL()),
		"snapshot":  snapshot,
		"truncated": truncated,
		"refs":      refsJSON,
		"stats": map[string]any{
			"lines":       len(lines),
			"chars":       len(snapshot),
			"refs":        len(refsJSON),
			"interactive": interactiveCount,
		},
	}

	if labels {
		img, err := tab.Page.Screenshot(playwright.PageScreenshotOptions{Type: playwright.ScreenshotTypePng})
		if err == nil {
			path, writeErr := saveBrowserArtifact("png", img)
			if writeErr == nil {
				result["labels"] = true
				result["imagePath"] = path
				result["imageType"] = "png"
			}
		}
	}

	return result, nil
}

func browserActionScreenshot(payload toolArgs, state *browserProfileState) (map[string]any, error) {
	if err := ensureBrowserProfileStarted(state); err != nil {
		return nil, err
	}
	targetID := strings.TrimSpace(getStringArg(payload, "targetId"))
	tab, err := resolveBrowserTab(state, targetID, true)
	if err != nil {
		return nil, err
	}

	imageType := strings.ToLower(strings.TrimSpace(getStringArg(payload, "type")))
	if imageType != "jpeg" {
		imageType = "png"
	}

	fullPage, _ := getBoolArg(payload, "fullPage")
	ref := strings.TrimSpace(getStringArg(payload, "ref"))
	element := strings.TrimSpace(getStringArg(payload, "element"))
	if fullPage && (ref != "" || element != "") {
		return nil, errors.New("fullPage is not supported for element screenshots")
	}

	var bytes []byte
	if ref != "" {
		locator, err := resolveBrowserRefLocator(tab, ref)
		if err != nil {
			return nil, err
		}
		bytes, err = locator.Screenshot(playwright.LocatorScreenshotOptions{
			Type:    toPlaywrightScreenshotType(imageType),
			Timeout: playwright.Float(float64(resolveBrowserActionTimeoutMs(payload, 15000))),
		})
		if err != nil {
			return nil, err
		}
	} else if element != "" {
		locator := tab.Page.Locator(strings.TrimSpace(element))
		bytes, err = locator.Screenshot(playwright.LocatorScreenshotOptions{
			Type:    toPlaywrightScreenshotType(imageType),
			Timeout: playwright.Float(float64(resolveBrowserActionTimeoutMs(payload, 15000))),
		})
		if err != nil {
			return nil, err
		}
	} else {
		bytes, err = tab.Page.Screenshot(playwright.PageScreenshotOptions{
			Type:     toPlaywrightScreenshotType(imageType),
			FullPage: playwright.Bool(fullPage),
			Timeout:  playwright.Float(float64(resolveBrowserActionTimeoutMs(payload, 15000))),
		})
		if err != nil {
			return nil, err
		}
	}

	path, err := saveBrowserArtifact(imageType, bytes)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"ok":       true,
		"path":     path,
		"targetId": tab.TargetID,
		"url":      strings.TrimSpace(tab.Page.URL()),
	}, nil
}

func browserActionNavigate(ctx context.Context, payload toolArgs, state *browserProfileState, connectors ConnectorsReader) (map[string]any, error) {
	targetURL := getStringArg(payload, "targetUrl", "url")
	if targetURL == "" {
		return nil, errors.New("targetUrl is required")
	}
	if err := assertBrowserURLAllowed(targetURL, state.resolved.SSRFRules); err != nil {
		return nil, err
	}
	if err := ensureBrowserProfileStarted(state); err != nil {
		return nil, err
	}

	targetID := strings.TrimSpace(getStringArg(payload, "targetId"))
	tab, err := resolveBrowserTab(state, targetID, true)
	if err != nil {
		return nil, err
	}

	state.mu.Lock()
	browserCtx := state.context
	state.mu.Unlock()
	if browserCtx != nil {
		if err := addConnectorCookiesToContext(ctx, connectors, browserCtx, targetURL); err != nil {
			return nil, err
		}
	}

	resp, err := tab.Page.Goto(strings.TrimSpace(targetURL), playwright.PageGotoOptions{
		Timeout:   playwright.Float(float64(resolveBrowserActionTimeoutMs(payload, 30000))),
		WaitUntil: resolveBrowserWaitUntilState(resolveBrowserWaitUntil(payload, defaultBrowserWaitUntil)),
	})
	if err != nil {
		return nil, err
	}

	finalURL := strings.TrimSpace(tab.Page.URL())
	if finalURL == "" {
		finalURL = strings.TrimSpace(targetURL)
	}
	if err := assertBrowserURLAllowed(finalURL, state.resolved.SSRFRules); err != nil {
		return nil, err
	}

	status := http.StatusOK
	if resp != nil {
		status = resp.Status()
	}

	return map[string]any{
		"ok":       true,
		"targetId": tab.TargetID,
		"url":      finalURL,
		"status":   status,
	}, nil
}

func browserActionConsole(payload toolArgs, state *browserProfileState) (map[string]any, error) {
	if err := ensureBrowserProfileStarted(state); err != nil {
		return nil, err
	}
	targetID := strings.TrimSpace(getStringArg(payload, "targetId"))
	tab, err := resolveBrowserTab(state, targetID, false)
	if err != nil {
		return nil, err
	}

	level := strings.ToLower(strings.TrimSpace(getStringArg(payload, "level")))
	state.mu.Lock()
	defer state.mu.Unlock()

	messages := make([]browserConsoleMessage, 0, len(state.consoleMessages))
	for _, item := range state.consoleMessages {
		if item.TargetID != tab.TargetID {
			continue
		}
		if level != "" && level != "all" && item.Type != level {
			continue
		}
		messages = append(messages, item)
	}

	return map[string]any{
		"ok":       true,
		"targetId": tab.TargetID,
		"messages": messages,
	}, nil
}

func browserActionPDF(payload toolArgs, state *browserProfileState) (map[string]any, error) {
	if err := ensureBrowserProfileStarted(state); err != nil {
		return nil, err
	}
	targetID := strings.TrimSpace(getStringArg(payload, "targetId"))
	tab, err := resolveBrowserTab(state, targetID, true)
	if err != nil {
		return nil, err
	}

	bytes, err := tab.Page.PDF(playwright.PagePdfOptions{
		PrintBackground: playwright.Bool(true),
	})
	if err != nil {
		return nil, err
	}
	path, err := saveBrowserArtifact("pdf", bytes)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"ok":       true,
		"path":     path,
		"targetId": tab.TargetID,
		"url":      strings.TrimSpace(tab.Page.URL()),
	}, nil
}

func browserActionUpload(payload toolArgs, state *browserProfileState) (map[string]any, error) {
	if err := ensureBrowserProfileStarted(state); err != nil {
		return nil, err
	}

	paths, err := parseBrowserUploadPaths(payload)
	if err != nil {
		return nil, err
	}

	targetID := strings.TrimSpace(getStringArg(payload, "targetId"))
	tab, err := resolveBrowserTab(state, targetID, true)
	if err != nil {
		return nil, err
	}

	inputRef := strings.TrimSpace(getStringArg(payload, "inputRef"))
	ref := strings.TrimSpace(getStringArg(payload, "ref"))
	element := strings.TrimSpace(getStringArg(payload, "element"))
	if inputRef != "" || ref != "" || element != "" {
		locator, err := resolveBrowserUploadLocator(tab, inputRef, ref, element)
		if err != nil {
			return nil, err
		}
		if err := locator.SetInputFiles(paths); err != nil {
			return nil, err
		}
		return map[string]any{
			"ok":       true,
			"targetId": tab.TargetID,
			"paths":    paths,
			"armed":    false,
		}, nil
	}

	timeoutMs := resolveBrowserActionTimeoutMs(payload, defaultBrowserHookTimeoutMs)
	state.mu.Lock()
	state.pendingUploads[tab.TargetID] = browserPendingUpload{
		Paths:     append([]string(nil), paths...),
		ExpiresAt: time.Now().Add(time.Duration(timeoutMs) * time.Millisecond),
	}
	state.mu.Unlock()

	return map[string]any{
		"ok":       true,
		"targetId": tab.TargetID,
		"paths":    paths,
		"armed":    true,
	}, nil
}

func browserActionDialog(payload toolArgs, state *browserProfileState) (map[string]any, error) {
	if err := ensureBrowserProfileStarted(state); err != nil {
		return nil, err
	}
	targetID := strings.TrimSpace(getStringArg(payload, "targetId"))
	tab, err := resolveBrowserTab(state, targetID, true)
	if err != nil {
		return nil, err
	}
	accept, _ := getBoolArg(payload, "accept")
	promptText := strings.TrimSpace(getStringArg(payload, "promptText"))
	timeoutMs := resolveBrowserActionTimeoutMs(payload, defaultBrowserHookTimeoutMs)

	state.mu.Lock()
	state.pendingDialogs[tab.TargetID] = browserPendingDialog{
		Accept:     accept,
		PromptText: promptText,
		ExpiresAt:  time.Now().Add(time.Duration(timeoutMs) * time.Millisecond),
	}
	state.mu.Unlock()

	return map[string]any{
		"ok":       true,
		"targetId": tab.TargetID,
		"armed":    true,
		"accept":   accept,
	}, nil
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
		err = browserActDrag(tab, toolArgs(request), payload)
	case "select":
		err = browserActSelect(tab, toolArgs(request), payload)
	case "fill":
		err = browserActFill(tab, toolArgs(request), payload)
	case "resize":
		err = browserActResize(tab, toolArgs(request))
	case "wait":
		err = browserActWait(tab, toolArgs(request), state.resolved.EvaluateEnabled)
	case "evaluate":
		err = browserActEvaluate(tab, toolArgs(request), payload, state.resolved.EvaluateEnabled)
	case "close":
		err = browserActClose(tab, state)
	default:
		err = errors.New("act kind not supported: " + kind)
	}
	if err != nil {
		return nil, err
	}

	result := map[string]any{
		"ok":       true,
		"targetId": tab.TargetID,
		"url":      strings.TrimSpace(tab.Page.URL()),
	}
	if kind == "evaluate" {
		result["result"] = tabResultFromEvaluate(tab)
	}
	return result, nil
}

func tabResultFromEvaluate(tab *browserTabState) any {
	if tab == nil {
		return nil
	}
	tab.mu.RLock()
	defer tab.mu.RUnlock()
	return tab.evaluateResult
}

func browserActClick(tab *browserTabState, request toolArgs, payload toolArgs) error {
	ref := strings.TrimSpace(getStringArg(request, "ref"))
	if ref == "" {
		return errors.New("ref is required")
	}
	locator, err := resolveBrowserRefLocator(tab, ref)
	if err != nil {
		return err
	}
	clickOptions := playwright.LocatorClickOptions{}
	if timeout := resolveBrowserActTimeoutMs(request, payload, 10000); timeout > 0 {
		clickOptions.Timeout = playwright.Float(float64(timeout))
	}
	buttonRaw := strings.TrimSpace(getStringArg(request, "button"))
	if buttonRaw != "" {
		button := toPlaywrightMouseButton(buttonRaw)
		if button == nil {
			return errors.New("button must be left|right|middle")
		}
		clickOptions.Button = button
	}
	modifiers, err := toPlaywrightKeyboardModifiers(getStringSliceArg(request, "modifiers"))
	if err != nil {
		return err
	}
	if len(modifiers) > 0 {
		clickOptions.Modifiers = modifiers
	}
	if doubleClick, _ := getBoolArg(request, "doubleClick"); doubleClick {
		dblOptions := playwright.LocatorDblclickOptions{}
		if clickOptions.Timeout != nil {
			dblOptions.Timeout = clickOptions.Timeout
		}
		if clickOptions.Button != nil {
			dblOptions.Button = clickOptions.Button
		}
		if len(clickOptions.Modifiers) > 0 {
			dblOptions.Modifiers = clickOptions.Modifiers
		}
		if err := locator.Dblclick(dblOptions); err != nil {
			return toBrowserFriendlyInteractionError(err, ref)
		}
		return nil
	}
	if err := locator.Click(clickOptions); err != nil {
		return toBrowserFriendlyInteractionError(err, ref)
	}
	return nil
}

func browserActType(tab *browserTabState, request toolArgs, payload toolArgs) error {
	ref := strings.TrimSpace(getStringArg(request, "ref"))
	if ref == "" {
		return errors.New("ref is required")
	}
	textRaw, ok := request["text"]
	if !ok {
		return errors.New("text is required")
	}
	text, ok := textRaw.(string)
	if !ok {
		return errors.New("text is required")
	}
	locator, err := resolveBrowserRefLocator(tab, ref)
	if err != nil {
		return err
	}
	timeout := resolveBrowserActTimeoutMs(request, payload, 10000)
	if slowly, _ := getBoolArg(request, "slowly"); slowly {
		typeOptions := playwright.LocatorTypeOptions{}
		if timeout > 0 {
			typeOptions.Timeout = playwright.Float(float64(timeout))
		}
		typeOptions.Delay = playwright.Float(75)
		if err := locator.Click(playwright.LocatorClickOptions{Timeout: playwright.Float(float64(timeout))}); err != nil {
			return toBrowserFriendlyInteractionError(err, ref)
		}
		if err := locator.Type(text, typeOptions); err != nil {
			return toBrowserFriendlyInteractionError(err, ref)
		}
	} else {
		fillOptions := playwright.LocatorFillOptions{}
		if timeout > 0 {
			fillOptions.Timeout = playwright.Float(float64(timeout))
		}
		if err := locator.Fill(text, fillOptions); err != nil {
			return toBrowserFriendlyInteractionError(err, ref)
		}
	}
	if submit, _ := getBoolArg(request, "submit"); submit {
		pressOptions := playwright.LocatorPressOptions{}
		if timeout > 0 {
			pressOptions.Timeout = playwright.Float(float64(timeout))
		}
		if err := locator.Press("Enter", pressOptions); err != nil {
			return toBrowserFriendlyInteractionError(err, ref)
		}
	}
	return nil
}

func browserActPress(tab *browserTabState, request toolArgs, payload toolArgs) error {
	key := strings.TrimSpace(getStringArg(request, "key"))
	if key == "" {
		return errors.New("key is required")
	}
	options := playwright.KeyboardPressOptions{}
	if delayMs, ok := getIntArg(request, "delayMs"); ok && delayMs >= 0 {
		options.Delay = playwright.Float(float64(delayMs))
	}
	if timeout := resolveBrowserActTimeoutMs(request, payload, 10000); timeout > 0 {
		tab.Page.SetDefaultTimeout(float64(timeout))
		defer tab.Page.SetDefaultTimeout(30000)
	}
	return tab.Page.Keyboard().Press(key, options)
}

func browserActHover(tab *browserTabState, request toolArgs, payload toolArgs) error {
	ref := strings.TrimSpace(getStringArg(request, "ref"))
	if ref == "" {
		return errors.New("ref is required")
	}
	locator, err := resolveBrowserRefLocator(tab, ref)
	if err != nil {
		return err
	}
	hoverOptions := playwright.LocatorHoverOptions{}
	if timeout := resolveBrowserActTimeoutMs(request, payload, 10000); timeout > 0 {
		hoverOptions.Timeout = playwright.Float(float64(timeout))
	}
	if err := locator.Hover(hoverOptions); err != nil {
		return toBrowserFriendlyInteractionError(err, ref)
	}
	return nil
}

func browserActDrag(tab *browserTabState, request toolArgs, payload toolArgs) error {
	startRef := strings.TrimSpace(getStringArg(request, "startRef"))
	endRef := strings.TrimSpace(getStringArg(request, "endRef"))
	if startRef == "" || endRef == "" {
		return errors.New("startRef and endRef are required")
	}
	startLocator, err := resolveBrowserRefLocator(tab, startRef)
	if err != nil {
		return err
	}
	endLocator, err := resolveBrowserRefLocator(tab, endRef)
	if err != nil {
		return err
	}
	dragOptions := playwright.LocatorDragToOptions{}
	if timeout := resolveBrowserActTimeoutMs(request, payload, 12000); timeout > 0 {
		dragOptions.Timeout = playwright.Float(float64(timeout))
	}
	if err := startLocator.DragTo(endLocator, dragOptions); err != nil {
		return toBrowserFriendlyInteractionError(err, startRef+" -> "+endRef)
	}
	return nil
}

func browserActSelect(tab *browserTabState, request toolArgs, payload toolArgs) error {
	ref := strings.TrimSpace(getStringArg(request, "ref"))
	values := getStringSliceArg(request, "values")
	if ref == "" || len(values) == 0 {
		return errors.New("ref and values are required")
	}
	locator, err := resolveBrowserRefLocator(tab, ref)
	if err != nil {
		return err
	}
	timeout := resolveBrowserActTimeoutMs(request, payload, 10000)
	_, err = locator.SelectOption(playwright.SelectOptionValues{Values: &values}, playwright.LocatorSelectOptionOptions{
		Timeout: playwright.Float(float64(timeout)),
	})
	if err != nil {
		return toBrowserFriendlyInteractionError(err, ref)
	}
	return nil
}

func browserActFill(tab *browserTabState, request toolArgs, payload toolArgs) error {
	rawFields, ok := request["fields"].([]any)
	if !ok || len(rawFields) == 0 {
		return errors.New("fields are required")
	}
	timeout := resolveBrowserActTimeoutMs(request, payload, 10000)
	for _, raw := range rawFields {
		field, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		ref := strings.TrimSpace(getStringArg(toolArgs(field), "ref"))
		if ref == "" {
			continue
		}
		typeValue := strings.ToLower(strings.TrimSpace(getStringArg(toolArgs(field), "type")))
		locator, err := resolveBrowserRefLocator(tab, ref)
		if err != nil {
			return err
		}
		switch typeValue {
		case "checkbox", "radio", "bool", "boolean":
			checked, _ := getBoolArg(toolArgs(field), "value")
			if checked {
				err = locator.Check(playwright.LocatorCheckOptions{Timeout: playwright.Float(float64(timeout))})
			} else {
				err = locator.Uncheck(playwright.LocatorUncheckOptions{Timeout: playwright.Float(float64(timeout))})
			}
		case "select", "option":
			value := strings.TrimSpace(getStringArg(toolArgs(field), "value"))
			if value == "" {
				continue
			}
			values := []string{value}
			_, err = locator.SelectOption(playwright.SelectOptionValues{Values: &values}, playwright.LocatorSelectOptionOptions{
				Timeout: playwright.Float(float64(timeout)),
			})
		default:
			value := strings.TrimSpace(getStringArg(toolArgs(field), "value"))
			err = locator.Fill(value, playwright.LocatorFillOptions{Timeout: playwright.Float(float64(timeout))})
		}
		if err != nil {
			return toBrowserFriendlyInteractionError(err, ref)
		}
	}
	return nil
}

func browserActResize(tab *browserTabState, request toolArgs) error {
	width, hasWidth := getIntArg(request, "width")
	height, hasHeight := getIntArg(request, "height")
	if !hasWidth || !hasHeight || width <= 0 || height <= 0 {
		return errors.New("width and height are required")
	}
	return tab.Page.SetViewportSize(width, height)
}

func browserActWait(tab *browserTabState, request toolArgs, evaluateEnabled bool) error {
	timeMs, hasTime := getIntArg(request, "timeMs")
	text := strings.TrimSpace(getStringArg(request, "text"))
	textGone := strings.TrimSpace(getStringArg(request, "textGone"))
	selector := strings.TrimSpace(getStringArg(request, "selector"))
	urlWait := strings.TrimSpace(getStringArg(request, "url"))
	loadState := strings.ToLower(strings.TrimSpace(getStringArg(request, "loadState")))
	fn := strings.TrimSpace(getStringArg(request, "fn"))
	timeoutMs := resolveBrowserActionTimeoutMs(request, 15000)

	if fn != "" && !evaluateEnabled {
		return errors.New(browserWaitFnDisabledMessage)
	}

	hasCondition := false
	if hasTime && timeMs > 0 {
		hasCondition = true
		tab.Page.WaitForTimeout(float64(timeMs))
	}
	if text != "" {
		hasCondition = true
		if err := tab.Page.Locator("text=" + text).First().WaitFor(playwright.LocatorWaitForOptions{
			State:   playwright.WaitForSelectorStateVisible,
			Timeout: playwright.Float(float64(timeoutMs)),
		}); err != nil {
			return err
		}
	}
	if textGone != "" {
		hasCondition = true
		if err := tab.Page.Locator("text=" + textGone).First().WaitFor(playwright.LocatorWaitForOptions{
			State:   playwright.WaitForSelectorStateHidden,
			Timeout: playwright.Float(float64(timeoutMs)),
		}); err != nil {
			return err
		}
	}
	if selector != "" {
		hasCondition = true
		if _, err := tab.Page.WaitForSelector(selector, playwright.PageWaitForSelectorOptions{
			State:   playwright.WaitForSelectorStateVisible,
			Timeout: playwright.Float(float64(timeoutMs)),
		}); err != nil {
			return err
		}
	}
	if urlWait != "" {
		hasCondition = true
		if err := tab.Page.WaitForURL(urlWait, playwright.PageWaitForURLOptions{
			Timeout: playwright.Float(float64(timeoutMs)),
		}); err != nil {
			return err
		}
	}
	if loadState != "" {
		hasCondition = true
		state := resolveBrowserLoadState(loadState)
		if state == nil {
			return errors.New("loadState must be load|domcontentloaded|networkidle")
		}
		if err := tab.Page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
			State:   state,
			Timeout: playwright.Float(float64(timeoutMs)),
		}); err != nil {
			return err
		}
	}
	if fn != "" {
		hasCondition = true
		if err := waitBrowserEvaluateCondition(tab.Page, fn, timeoutMs); err != nil {
			return err
		}
	}
	if !hasCondition {
		return errors.New(browserWaitRequiresConditionMessage)
	}
	return nil
}

func browserActEvaluate(tab *browserTabState, request toolArgs, payload toolArgs, evaluateEnabled bool) error {
	if !evaluateEnabled {
		return errors.New(browserEvaluateDisabledMessage)
	}
	fn := strings.TrimSpace(getStringArg(request, "fn"))
	if fn == "" {
		return errors.New("fn is required")
	}
	timeoutMs := resolveBrowserActTimeoutMs(request, payload, 20000)
	ref := strings.TrimSpace(getStringArg(request, "ref"))
	var (
		result any
		err    error
	)
	if ref != "" {
		locator, resolveErr := resolveBrowserRefLocator(tab, ref)
		if resolveErr != nil {
			return resolveErr
		}
		result, err = locator.Evaluate(browserEvaluateElementExpression, []any{fn, timeoutMs})
	} else {
		result, err = tab.Page.Evaluate(browserEvaluatePageExpression, []any{fn, timeoutMs})
	}
	if err != nil {
		return err
	}
	tab.mu.Lock()
	tab.evaluateResult = result
	tab.mu.Unlock()
	return nil
}

func browserActClose(tab *browserTabState, state *browserProfileState) error {
	if tab == nil || tab.Page == nil {
		return errors.New("tab not found")
	}
	if err := tab.Page.Close(); err != nil {
		return err
	}
	state.mu.Lock()
	delete(state.tabs, tab.TargetID)
	delete(state.pageToTarget, tab.Page)
	if state.activeTarget == tab.TargetID {
		state.activeTarget = ""
	}
	pruneClosedTabsLocked(state)
	state.mu.Unlock()
	return nil
}

type browserSnapshotItem struct {
	Ref     string
	AriaRef string
	Role    string
	Name    string
	Text    string
	Depth   int
	Nth     int
}

var browserAriaSnapshotLinePattern = regexp.MustCompile(`^(\s*)-\s*([^\s":]+)(?:\s+"([^"]*)")?(.*)$`)
var browserAriaSnapshotRefPattern = regexp.MustCompile(`\[ref=([^\]]+)\]`)
var browserStrictModeCountPattern = regexp.MustCompile(`resolved to (\d+) elements`)
var browserEvaluatePageExpression = `([fnBody, timeoutMs]) => {
	try {
		const candidate = eval("(" + fnBody + ")");
		const result = typeof candidate === "function" ? candidate() : candidate;
		if (result && typeof result.then === "function") {
			return Promise.race([
				result,
				new Promise((_, reject) =>
					setTimeout(() => reject(new Error("evaluate timed out after " + timeoutMs + "ms")), timeoutMs),
				),
			]);
		}
		return result;
	} catch (err) {
		throw new Error("Invalid evaluate function: " + (err && err.message ? err.message : String(err)));
	}
}`
var browserEvaluateElementExpression = `(el, [fnBody, timeoutMs]) => {
	try {
		const candidate = eval("(" + fnBody + ")");
		const result = typeof candidate === "function" ? candidate(el) : candidate;
		if (result && typeof result.then === "function") {
			return Promise.race([
				result,
				new Promise((_, reject) =>
					setTimeout(() => reject(new Error("evaluate timed out after " + timeoutMs + "ms")), timeoutMs),
				),
			]);
		}
		return result;
	} catch (err) {
		throw new Error("Invalid evaluate function: " + (err && err.message ? err.message : String(err)));
	}
}`

type browserAISnapshotResult struct {
	Snapshot  string
	Truncated bool
	Refs      map[string]browserSnapshotRef
	RefsJSON  map[string]map[string]any
	Stats     map[string]any
}

func collectBrowserAISnapshot(page playwright.Page, maxChars int) (*browserAISnapshotResult, error) {
	snapshot, err := captureBrowserPrivateAISnapshot(page, 5000)
	if err != nil {
		return nil, err
	}
	items := parseBrowserAriaSnapshot(snapshot, false, 2000, true, 0)
	refs := make(map[string]browserSnapshotRef, len(items))
	refsJSON := make(map[string]map[string]any, len(items))
	interactiveCount := 0
	for index, item := range items {
		ref := strings.TrimSpace(item.Ref)
		if ref == "" {
			ref = fmt.Sprintf("e%d", index+1)
		}
		entryMode := "aria"
		if strings.TrimSpace(item.AriaRef) == "" {
			entryMode = "role"
		}
		entry := browserSnapshotRef{
			Role:    item.Role,
			Name:    item.Name,
			Nth:     item.Nth,
			Mode:    entryMode,
			AriaRef: item.AriaRef,
		}
		refs[ref] = entry
		refsJSON[ref] = map[string]any{
			"role": entry.Role,
			"name": entry.Name,
		}
		if entry.Nth > 0 {
			refsJSON[ref]["nth"] = entry.Nth
		}
		if isBrowserInteractiveRole(entry.Role) {
			interactiveCount += 1
		}
	}

	truncated := false
	if maxChars > 0 && len(snapshot) > maxChars {
		snapshot = trimToMaxChars(snapshot, maxChars)
		truncated = true
	}

	lines := 0
	for _, line := range strings.Split(snapshot, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		lines += 1
	}
	return &browserAISnapshotResult{
		Snapshot:  snapshot,
		Truncated: truncated,
		Refs:      refs,
		RefsJSON:  refsJSON,
		Stats: map[string]any{
			"lines":       lines,
			"chars":       len(snapshot),
			"refs":        len(refsJSON),
			"interactive": interactiveCount,
		},
	}, nil
}

func captureBrowserPrivateAISnapshot(page playwright.Page, timeoutMs int) (snapshot string, err error) {
	defer func() {
		if recover() != nil {
			snapshot = ""
			err = errBrowserSnapshotForAIUnavailable
		}
	}()
	if page == nil {
		return "", errBrowserSnapshotForAIUnavailable
	}

	value := reflect.ValueOf(page)
	if !value.IsValid() {
		return "", errBrowserSnapshotForAIUnavailable
	}
	if value.Kind() == reflect.Interface {
		value = value.Elem()
	}
	if value.Kind() == reflect.Pointer {
		value = value.Elem()
	}
	if !value.IsValid() || value.Kind() != reflect.Struct {
		return "", errBrowserSnapshotForAIUnavailable
	}

	channelOwnerField := value.FieldByName("channelOwner")
	if !channelOwnerField.IsValid() {
		return "", errBrowserSnapshotForAIUnavailable
	}
	channelField := channelOwnerField.FieldByName("channel")
	if !channelField.IsValid() || (channelField.Kind() == reflect.Pointer && channelField.IsNil()) {
		return "", errBrowserSnapshotForAIUnavailable
	}
	sendMethod := channelField.MethodByName("Send")
	if !sendMethod.IsValid() {
		return "", errBrowserSnapshotForAIUnavailable
	}

	calls := sendMethod.Call([]reflect.Value{
		reflect.ValueOf("snapshotForAI"),
		reflect.ValueOf(map[string]any{
			"timeout": normalizeBrowserTimeoutMs(timeoutMs, 5000),
			"track":   "response",
		}),
	})
	if len(calls) != 2 {
		return "", errBrowserSnapshotForAIUnavailable
	}
	if errValue := calls[1]; errValue.IsValid() && !errValue.IsNil() {
		if invokeErr, ok := errValue.Interface().(error); ok {
			return "", invokeErr
		}
		return "", fmt.Errorf("snapshotForAI call failed")
	}

	result := calls[0].Interface()
	snapshot = extractBrowserAISnapshotText(result)
	if strings.TrimSpace(snapshot) == "" {
		return "", errBrowserSnapshotForAIUnavailable
	}
	return snapshot, nil
}

func extractBrowserAISnapshotText(raw any) string {
	switch value := raw.(type) {
	case map[string]any:
		if full, ok := value["full"]; ok {
			return fmt.Sprint(full)
		}
		if snapshot, ok := value["snapshot"]; ok {
			return fmt.Sprint(snapshot)
		}
	case string:
		return value
	}
	return ""
}

func isBrowserSnapshotForAIUnavailable(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, errBrowserSnapshotForAIUnavailable) {
		return true
	}
	message := strings.ToLower(strings.TrimSpace(err.Error()))
	return strings.Contains(message, "_snapshotforai") ||
		strings.Contains(message, "snapshotforai") ||
		strings.Contains(message, "playwright snapshotforai is unavailable")
}

func collectBrowserSnapshotItems(
	page playwright.Page,
	selector string,
	frameSelector string,
	interactive bool,
	limit int,
	refsMode string,
	maxDepth int,
) ([]browserSnapshotItem, error) {
	if limit <= 0 {
		limit = defaultBrowserSnapshotLimit
	}
	locator := resolveBrowserSnapshotLocator(page, selector, frameSelector)
	snapshot, err := locator.AriaSnapshot(playwright.LocatorAriaSnapshotOptions{
		Ref: playwright.Bool(strings.EqualFold(refsMode, "aria")),
	})
	if err != nil {
		return nil, err
	}
	return parseBrowserAriaSnapshot(snapshot, interactive, limit, strings.EqualFold(refsMode, "aria"), maxDepth), nil
}

func resolveBrowserSnapshotLocator(page playwright.Page, selector string, frameSelector string) playwright.Locator {
	selector = strings.TrimSpace(selector)
	frameSelector = strings.TrimSpace(frameSelector)
	if frameSelector != "" {
		frame := page.FrameLocator(frameSelector)
		if selector != "" {
			return frame.Locator(selector)
		}
		return frame.Locator(":root")
	}
	if selector != "" {
		return page.Locator(selector)
	}
	return page.Locator(":root")
}

func parseBrowserAriaSnapshot(
	snapshot string,
	interactive bool,
	limit int,
	useAriaRefs bool,
	maxDepth int,
) []browserSnapshotItem {
	if limit <= 0 {
		limit = defaultBrowserSnapshotLimit
	}
	lines := strings.Split(snapshot, "\n")
	items := make([]browserSnapshotItem, 0, minInt(len(lines), limit))
	countByRoleName := map[string]int{}
	nextRef := 0

	for _, rawLine := range lines {
		if len(items) >= limit {
			break
		}
		line := strings.TrimRight(rawLine, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}

		match := browserAriaSnapshotLinePattern.FindStringSubmatch(line)
		if len(match) < 5 {
			continue
		}
		role := strings.ToLower(strings.TrimSpace(match[2]))
		if role == "" || strings.HasPrefix(role, "/") {
			continue
		}
		if interactive && !isBrowserInteractiveRole(role) {
			continue
		}

		name := strings.TrimSpace(match[3])
		suffix := strings.TrimSpace(match[4])
		text := browserAriaSnapshotTextFromSuffix(suffix)
		depth := browserAriaSnapshotDepth(line)
		if maxDepth > 0 && depth > maxDepth {
			continue
		}

		ariaRef := ""
		if refMatch := browserAriaSnapshotRefPattern.FindStringSubmatch(suffix); len(refMatch) >= 2 {
			ariaRef = strings.TrimSpace(refMatch[1])
		}

		key := role + "\n" + name
		nth := countByRoleName[key]
		countByRoleName[key] = nth + 1

		nextRef += 1
		ref := fmt.Sprintf("e%d", nextRef)
		if useAriaRefs && ariaRef != "" {
			ref = ariaRef
		}

		item := browserSnapshotItem{
			Ref:     ref,
			AriaRef: ariaRef,
			Role:    role,
			Name:    name,
			Text:    text,
			Depth:   depth,
			Nth:     nth,
		}
		if item.Role == "" {
			item.Role = "element"
		}
		if item.Name == "" {
			item.Name = item.Text
		}
		if item.Name == "" {
			item.Name = "(empty)"
		}
		items = append(items, item)
	}
	return items
}

func browserAriaSnapshotDepth(line string) int {
	spaces := 0
	for _, r := range line {
		if r == ' ' {
			spaces += 1
			continue
		}
		if r == '\t' {
			spaces += 2
			continue
		}
		break
	}
	return spaces / 2
}

func browserAriaSnapshotTextFromSuffix(suffix string) string {
	suffix = strings.TrimSpace(suffix)
	if suffix == "" {
		return ""
	}
	if idx := strings.LastIndex(suffix, ":"); idx >= 0 && idx+1 < len(suffix) {
		return strings.TrimSpace(suffix[idx+1:])
	}
	return ""
}

func isBrowserInteractiveRole(role string) bool {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "button",
		"link",
		"textbox",
		"checkbox",
		"radio",
		"combobox",
		"listbox",
		"menuitem",
		"menuitemcheckbox",
		"menuitemradio",
		"option",
		"searchbox",
		"slider",
		"spinbutton",
		"switch",
		"tab",
		"treeitem":
		return true
	default:
		return false
	}
}

func resolveBrowserUploadLocator(tab *browserTabState, inputRef string, ref string, element string) (playwright.Locator, error) {
	switch {
	case strings.TrimSpace(inputRef) != "":
		return tab.Page.Locator(strings.TrimSpace(inputRef)), nil
	case strings.TrimSpace(ref) != "":
		return resolveBrowserRefLocator(tab, strings.TrimSpace(ref))
	case strings.TrimSpace(element) != "":
		return tab.Page.Locator(strings.TrimSpace(element)), nil
	default:
		return nil, errors.New("upload requires ref/inputRef/element")
	}
}

func parseBrowserUploadPaths(payload toolArgs) ([]string, error) {
	raw := getStringSliceArg(payload, "paths")
	if len(raw) == 0 {
		return nil, errors.New("paths are required")
	}
	rootDir, err := resolveBrowserUploadRootDir()
	if err != nil {
		return nil, err
	}
	paths := make([]string, 0, len(raw))
	for _, item := range raw {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		absPath, err := resolvePathWithinRoot(rootDir, trimmed)
		if err != nil {
			return nil, err
		}
		info, err := os.Stat(absPath)
		if err != nil {
			return nil, fmt.Errorf("upload path not found: %s", absPath)
		}
		if info.IsDir() {
			return nil, fmt.Errorf("upload path must be a file: %s", absPath)
		}
		paths = append(paths, absPath)
	}
	if len(paths) == 0 {
		return nil, errors.New("paths are required")
	}
	return paths, nil
}

func resolveBrowserUploadRootDir() (string, error) {
	dir := filepath.Join(os.TempDir(), "dreamcreator", "browser", "uploads")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return filepath.Abs(dir)
}

func resolvePathWithinRoot(rootDir string, requestedPath string) (string, error) {
	root := strings.TrimSpace(rootDir)
	if root == "" {
		return "", errors.New("upload root is required")
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	raw := strings.TrimSpace(requestedPath)
	if raw == "" {
		return "", errors.New("paths are required")
	}

	var candidate string
	if filepath.IsAbs(raw) {
		candidate = filepath.Clean(raw)
	} else {
		candidate = filepath.Join(rootAbs, raw)
	}
	candidateAbs, err := filepath.Abs(candidate)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(rootAbs, candidateAbs)
	if err != nil {
		return "", err
	}
	rel = filepath.ToSlash(strings.TrimSpace(rel))
	if rel == "." || strings.HasPrefix(rel, "../") || rel == ".." {
		return "", fmt.Errorf("Invalid path: must stay within uploads directory (%s)", rootAbs)
	}
	return candidateAbs, nil
}

func resolveBrowserRefLocator(tab *browserTabState, ref string) (playwright.Locator, error) {
	if tab == nil {
		return nil, errors.New("tab unavailable")
	}
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return nil, errors.New("ref is required")
	}
	if tab.refs == nil {
		return nil, errors.New("no snapshot refs available; run action=snapshot first")
	}
	entry, ok := tab.refs[ref]
	if !ok {
		return nil, fmt.Errorf("ref not found: %s (run action=snapshot again)", ref)
	}

	mode := strings.ToLower(strings.TrimSpace(entry.Mode))
	if mode == "aria" {
		ariaRef := strings.TrimSpace(entry.AriaRef)
		if ariaRef == "" {
			ariaRef = ref
		}
		if ariaRef == "" {
			return nil, fmt.Errorf("ref selector missing for %s", ref)
		}
		locator := resolveBrowserScopedLocator(tab.Page, entry.Frame, "aria-ref="+ariaRef)
		return locator, nil
	}
	if strings.TrimSpace(entry.Role) != "" {
		locator, err := resolveBrowserRoleLocator(tab.Page, entry.Role, entry.Name, entry.Nth, entry.Frame)
		if err == nil {
			return locator, nil
		}
		// Fall back to selector-based resolution below.
	}

	selector := strings.TrimSpace(entry.Selector)
	if selector == "" {
		return nil, fmt.Errorf("ref selector missing for %s", ref)
	}
	return resolveBrowserScopedLocator(tab.Page, entry.Frame, selector), nil
}

func resolveBrowserScopedLocator(page playwright.Page, frameSelector string, selector string) playwright.Locator {
	frameSelector = strings.TrimSpace(frameSelector)
	selector = strings.TrimSpace(selector)
	if frameSelector == "" {
		if strings.HasPrefix(selector, "//") || strings.HasPrefix(selector, "/") {
			return page.Locator("xpath=" + selector)
		}
		return page.Locator(selector)
	}
	frame := page.FrameLocator(frameSelector)
	if strings.HasPrefix(selector, "//") || strings.HasPrefix(selector, "/") {
		return frame.Locator("xpath=" + selector)
	}
	return frame.Locator(selector)
}

func resolveBrowserRoleLocator(page playwright.Page, role string, name string, nth int, frameSelector string) (playwright.Locator, error) {
	role = strings.TrimSpace(strings.ToLower(role))
	if role == "" {
		return nil, errors.New("role is required")
	}

	frameSelector = strings.TrimSpace(frameSelector)
	name = strings.TrimSpace(name)

	var locator playwright.Locator
	if frameSelector != "" {
		frame := page.FrameLocator(frameSelector)
		if name != "" {
			locator = frame.GetByRole(playwright.AriaRole(role), playwright.FrameLocatorGetByRoleOptions{
				Name:  name,
				Exact: playwright.Bool(true),
			})
		} else {
			locator = frame.GetByRole(playwright.AriaRole(role))
		}
	} else {
		if name != "" {
			locator = page.GetByRole(playwright.AriaRole(role), playwright.PageGetByRoleOptions{
				Name:  name,
				Exact: playwright.Bool(true),
			})
		} else {
			locator = page.GetByRole(playwright.AriaRole(role))
		}
	}
	if nth > 0 {
		locator = locator.Nth(nth)
	}
	return locator, nil
}

func listBrowserTabs(state *browserProfileState) ([]map[string]any, error) {
	state.mu.Lock()
	defer state.mu.Unlock()
	pruneClosedTabsLocked(state)
	items := make([]map[string]any, 0, len(state.tabs))
	ids := make([]string, 0, len(state.tabs))
	for id := range state.tabs {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		tab := state.tabs[id]
		title, _ := tab.Page.Title()
		items = append(items, map[string]any{
			"targetId": tab.TargetID,
			"title":    strings.TrimSpace(title),
			"url":      strings.TrimSpace(tab.Page.URL()),
			"type":     "page",
		})
	}
	return items, nil
}

func resolveBrowserTab(state *browserProfileState, targetID string, autoCreate bool) (*browserTabState, error) {
	state.mu.Lock()
	pruneClosedTabsLocked(state)

	targetID = strings.TrimSpace(targetID)
	if targetID != "" {
		if tab, ok := state.tabs[targetID]; ok {
			state.activeTarget = tab.TargetID
			state.mu.Unlock()
			return tab, nil
		}
		matches := make([]*browserTabState, 0, len(state.tabs))
		targetLower := strings.ToLower(targetID)
		for id, tab := range state.tabs {
			if strings.HasPrefix(strings.ToLower(strings.TrimSpace(id)), targetLower) {
				matches = append(matches, tab)
			}
		}
		if len(matches) == 1 {
			state.activeTarget = matches[0].TargetID
			state.mu.Unlock()
			return matches[0], nil
		}
		if len(matches) > 1 {
			state.mu.Unlock()
			return nil, errors.New("ambiguous target id prefix")
		}
		if len(state.tabs) == 1 {
			for _, only := range state.tabs {
				state.activeTarget = only.TargetID
				state.mu.Unlock()
				return only, nil
			}
		}
		state.mu.Unlock()
		return nil, errors.New("tab not found")
	}

	if state.activeTarget != "" {
		if tab, ok := state.tabs[state.activeTarget]; ok {
			state.mu.Unlock()
			return tab, nil
		}
	}
	if len(state.tabs) == 1 {
		for _, only := range state.tabs {
			state.activeTarget = only.TargetID
			state.mu.Unlock()
			return only, nil
		}
	}
	if len(state.tabs) > 1 {
		ids := make([]string, 0, len(state.tabs))
		for id := range state.tabs {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		first := state.tabs[ids[0]]
		state.activeTarget = first.TargetID
		state.mu.Unlock()
		return first, nil
	}

	browserCtx := state.context
	state.mu.Unlock()
	if !autoCreate || browserCtx == nil {
		return nil, errors.New("no browser tab available")
	}
	page, err := browserCtx.NewPage()
	if err != nil {
		return nil, err
	}
	tab := attachBrowserTab(state, page)
	return tab, nil
}

func attachBrowserTab(state *browserProfileState, page playwright.Page) *browserTabState {
	state.mu.Lock()
	defer state.mu.Unlock()
	if targetID, ok := state.pageToTarget[page]; ok {
		if existing, exists := state.tabs[targetID]; exists {
			state.activeTarget = existing.TargetID
			return existing
		}
	}
	targetID := fmt.Sprintf("T%d", atomic.AddUint64(&browserGlobalTabCounter, 1))
	tab := &browserTabState{
		TargetID: targetID,
		Page:     page,
		refs:     map[string]browserSnapshotRef{},
	}
	state.tabs[targetID] = tab
	state.pageToTarget[page] = targetID
	state.activeTarget = targetID
	attachBrowserPageObservers(state, tab)
	return tab
}

func attachBrowserPageObservers(state *browserProfileState, tab *browserTabState) {
	if tab == nil || tab.Page == nil {
		return
	}
	targetID := tab.TargetID

	tab.Page.OnConsole(func(message playwright.ConsoleMessage) {
		entry := browserConsoleMessage{
			TargetID:  targetID,
			Type:      strings.ToLower(strings.TrimSpace(message.Type())),
			Text:      trimToMaxChars(strings.TrimSpace(message.Text()), 4000),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}
		state.mu.Lock()
		state.consoleMessages = append(state.consoleMessages, entry)
		if len(state.consoleMessages) > 400 {
			state.consoleMessages = append([]browserConsoleMessage(nil), state.consoleMessages[len(state.consoleMessages)-400:]...)
		}
		state.mu.Unlock()
	})

	tab.Page.OnDialog(func(dialog playwright.Dialog) {
		state.mu.Lock()
		pending, ok := state.pendingDialogs[targetID]
		if ok {
			delete(state.pendingDialogs, targetID)
		}
		state.mu.Unlock()
		if ok && time.Now().Before(pending.ExpiresAt) {
			if pending.Accept {
				if pending.PromptText != "" {
					_ = dialog.Accept(pending.PromptText)
				} else {
					_ = dialog.Accept()
				}
			} else {
				_ = dialog.Dismiss()
			}
			return
		}
		_ = dialog.Dismiss()
	})

	tab.Page.OnFileChooser(func(chooser playwright.FileChooser) {
		state.mu.Lock()
		pending, ok := state.pendingUploads[targetID]
		if ok {
			delete(state.pendingUploads, targetID)
		}
		state.mu.Unlock()
		if ok && time.Now().Before(pending.ExpiresAt) {
			_ = chooser.SetFiles(pending.Paths)
		}
	})
}

func ensureBrowserProfileStarted(state *browserProfileState) error {
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.browser != nil && state.context != nil {
		pruneClosedTabsLocked(state)
		return nil
	}

	var err error
	if state.pw == nil {
		state.pw, err = playwright.Run(&playwright.RunOptions{Verbose: false, Stdout: io.Discard, Stderr: io.Discard})
		if err != nil {
			return err
		}
	}

	launchOptions := playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(state.resolved.Headless),
	}
	args := append([]string(nil), state.resolved.ExtraArgs...)
	if state.resolved.NoSandbox {
		args = append(args, "--no-sandbox")
	}
	if state.resolved.Headless && !hasBrowserHeadlessArg(args) {
		args = append(args, "--headless=new")
	}
	if len(args) > 0 {
		launchOptions.Args = args
	}
	state.browser, err = state.pw.Chromium.Launch(launchOptions)
	if err != nil {
		return err
	}
	state.context, err = state.browser.NewContext(playwright.BrowserNewContextOptions{
		Viewport: &playwright.Size{Width: defaultBrowserViewportWidth, Height: defaultBrowserViewportHeight},
	})
	if err != nil {
		_ = state.browser.Close()
		state.browser = nil
		return err
	}

	if state.tabs == nil {
		state.tabs = map[string]*browserTabState{}
	}
	if state.pageToTarget == nil {
		state.pageToTarget = map[playwright.Page]string{}
	}
	if state.pendingUploads == nil {
		state.pendingUploads = map[string]browserPendingUpload{}
	}
	if state.pendingDialogs == nil {
		state.pendingDialogs = map[string]browserPendingDialog{}
	}
	for _, page := range state.context.Pages() {
		if page == nil || page.IsClosed() {
			continue
		}
		if targetID, ok := state.pageToTarget[page]; ok {
			if _, exists := state.tabs[targetID]; exists {
				continue
			}
		}
		targetID := fmt.Sprintf("T%d", atomic.AddUint64(&browserGlobalTabCounter, 1))
		tab := &browserTabState{TargetID: targetID, Page: page, refs: map[string]browserSnapshotRef{}}
		state.tabs[targetID] = tab
		state.pageToTarget[page] = targetID
		if state.activeTarget == "" {
			state.activeTarget = targetID
		}
		attachBrowserPageObservers(state, tab)
	}
	pruneClosedTabsLocked(state)
	return nil
}

func stopBrowserProfile(state *browserProfileState) error {
	state.mu.Lock()
	defer state.mu.Unlock()

	if state.context != nil {
		_ = state.context.Close()
		state.context = nil
	}
	if state.browser != nil {
		_ = state.browser.Close()
		state.browser = nil
	}
	if state.pw != nil {
		_ = state.pw.Stop()
		state.pw = nil
	}

	state.tabs = map[string]*browserTabState{}
	state.pageToTarget = map[playwright.Page]string{}
	state.activeTarget = ""
	state.pendingUploads = map[string]browserPendingUpload{}
	state.pendingDialogs = map[string]browserPendingDialog{}
	state.consoleMessages = nil
	return nil
}

func pruneClosedTabsLocked(state *browserProfileState) {
	if state == nil {
		return
	}
	for targetID, tab := range state.tabs {
		if tab == nil || tab.Page == nil || tab.Page.IsClosed() {
			delete(state.tabs, targetID)
			if tab != nil && tab.Page != nil {
				delete(state.pageToTarget, tab.Page)
			}
			if state.activeTarget == targetID {
				state.activeTarget = ""
			}
		}
	}
	if state.activeTarget != "" {
		if _, ok := state.tabs[state.activeTarget]; ok {
			return
		}
	}
	if len(state.tabs) == 0 {
		state.activeTarget = ""
		return
	}
	ids := make([]string, 0, len(state.tabs))
	for id := range state.tabs {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	state.activeTarget = ids[0]
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
		profile := resolved.Profiles[profileName]
		state = &browserProfileState{
			profileName:     profileName,
			resolved:        resolved,
			profile:         profile,
			tabs:            map[string]*browserTabState{},
			pageToTarget:    map[playwright.Page]string{},
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
	state.profileName = profileName
	state.mu.Unlock()
	return state
}

func resolveBrowserRuntimeConfig(config map[string]any) browserResolvedConfig {
	browserConfig := resolveBrowserConfig(config)
	resolved := browserResolvedConfig{
		Enabled:         true,
		EvaluateEnabled: true,
		Color:           defaultBrowserColor,
		Headless:        false,
		NoSandbox:       false,
		DefaultProfile:  defaultBrowserProfileDreamCreator,
		Profiles: map[string]browserProfileConfig{
			defaultBrowserProfileDreamCreator: {
				Name:   defaultBrowserProfileDreamCreator,
				Color:  defaultBrowserColor,
				Driver: browserTypePlaywright,
			},
		},
		SSRFRules: browserSSRFPolicy{
			DangerouslyAllowPrivateNetwork: true,
			AllowedHostnames:               map[string]struct{}{},
			HostnameAllowlist:              nil,
		},
		ExtraArgs: nil,
	}
	if browserConfig == nil {
		return resolved
	}

	if value, ok := getBoolArg(toolArgs(browserConfig), "enabled"); ok {
		resolved.Enabled = value
	}
	if value, ok := getBoolArg(toolArgs(browserConfig), "evaluateEnabled"); ok {
		resolved.EvaluateEnabled = value
	}
	if value := strings.TrimSpace(getStringArg(toolArgs(browserConfig), "color")); value != "" {
		resolved.Color = value
	}
	if value, ok := getBoolArg(toolArgs(browserConfig), "headless"); ok {
		resolved.Headless = value
	}
	if value, ok := getBoolArg(toolArgs(browserConfig), "noSandbox"); ok {
		resolved.NoSandbox = value
	}
	if values := normalizeStringSlice(getStringSliceArg(toolArgs(browserConfig), "extraArgs")); len(values) > 0 {
		resolved.ExtraArgs = values
	}

	if snapshotDefaults := getMapArg(toolArgs(browserConfig), "snapshotDefaults"); snapshotDefaults != nil {
		if value := strings.TrimSpace(getStringArg(toolArgs(snapshotDefaults), "mode")); value == defaultBrowserSnapshotModeEfficient {
			resolved.SnapshotDefaultMode = value
		}
	}

	if ssrfRaw := getMapArg(toolArgs(browserConfig), "ssrfPolicy"); ssrfRaw != nil {
		if value, ok := getBoolArg(toolArgs(ssrfRaw), "dangerouslyAllowPrivateNetwork"); ok {
			resolved.SSRFRules.DangerouslyAllowPrivateNetwork = value
		} else if value, ok := getBoolArg(toolArgs(ssrfRaw), "allowPrivateNetwork"); ok {
			resolved.SSRFRules.DangerouslyAllowPrivateNetwork = value
		}
		for _, hostname := range getStringSliceArg(toolArgs(ssrfRaw), "allowedHostnames") {
			resolved.SSRFRules.AllowedHostnames[strings.ToLower(strings.TrimSpace(hostname))] = struct{}{}
		}
		if allowlist := normalizeStringSlice(getStringSliceArg(toolArgs(ssrfRaw), "hostnameAllowlist")); len(allowlist) > 0 {
			resolved.SSRFRules.HostnameAllowlist = allowlist
		}
	}

	if _, ok := resolved.Profiles[resolved.DefaultProfile]; !ok {
		resolved.DefaultProfile = defaultBrowserProfileDreamCreator
	}
	for name, profile := range resolved.Profiles {
		if profile.Name == "" {
			profile.Name = name
		}
		if profile.Color == "" {
			profile.Color = resolved.Color
		}
		if profile.Driver == "" {
			profile.Driver = browserTypePlaywright
		}
		resolved.Profiles[name] = profile
	}

	return resolved
}

func resolveBrowserProfileName(payload toolArgs, resolved browserResolvedConfig) string {
	profile := strings.TrimSpace(getStringArg(payload, "profile"))
	if profile == "" {
		profile = strings.TrimSpace(resolved.DefaultProfile)
	}
	if profile == "" {
		profile = defaultBrowserProfileDreamCreator
	}
	if _, ok := resolved.Profiles[profile]; !ok {
		profileCfg := browserProfileConfig{
			Name:   profile,
			Color:  resolved.Color,
			Driver: browserTypePlaywright,
		}
		resolved.Profiles[profile] = profileCfg
	}
	return profile
}

func resolveBrowserConfig(config map[string]any) map[string]any {
	return getNestedMap(config, "browser")
}

func resolveBrowserConfigBool(config map[string]any, key string) (bool, bool) {
	return getBoolArg(toolArgs(resolveBrowserConfig(config)), key)
}

func resolveBrowserType(payload toolArgs, config map[string]any) string {
	if raw := getStringArg(payload, "type", "mode", "engine"); raw != "" {
		return normalizeBrowserType(raw)
	}
	if payloadBrowser := getMapArg(payload, "browser"); payloadBrowser != nil {
		if raw := getStringArg(toolArgs(payloadBrowser), "type", "mode", "engine"); raw != "" {
			return normalizeBrowserType(raw)
		}
	}
	if raw := getStringArg(toolArgs(resolveBrowserConfig(config)), "type", "mode", "engine"); raw != "" {
		return normalizeBrowserType(raw)
	}
	return defaultBrowserType
}

func normalizeBrowserType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case browserTypePlaywright, "browser", "headless", "chromium":
		return browserTypePlaywright
	default:
		return ""
	}
}

func resolveBrowserWaitUntil(payload toolArgs, fallback string) string {
	if value := normalizeBrowserWaitUntil(getStringArg(payload, "waitUntil")); value != "" {
		return value
	}
	return normalizeBrowserWaitUntil(fallback)
}

func resolveBrowserWaitUntilState(value string) *playwright.WaitUntilState {
	switch normalizeBrowserWaitUntil(value) {
	case "commit":
		return playwright.WaitUntilStateCommit
	case "load":
		return playwright.WaitUntilStateLoad
	case "networkidle":
		return playwright.WaitUntilStateNetworkidle
	default:
		return playwright.WaitUntilStateDomcontentloaded
	}
}

func normalizeBrowserWaitUntil(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "commit":
		return "commit"
	case "load":
		return "load"
	case "networkidle", "network_idle":
		return "networkidle"
	case "domcontentloaded", "dom_content_loaded", "dom-content-loaded":
		return "domcontentloaded"
	default:
		return ""
	}
}

func resolveBrowserLoadState(value string) *playwright.LoadState {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "load":
		return playwright.LoadStateLoad
	case "domcontentloaded", "dom_content_loaded", "dom-content-loaded":
		return playwright.LoadStateDomcontentloaded
	case "networkidle", "network_idle":
		return playwright.LoadStateNetworkidle
	default:
		return nil
	}
}

func resolveBrowserActionTimeoutMs(payload toolArgs, fallback int) int {
	if timeoutMs, ok := getIntArg(payload, "timeoutMs"); ok && timeoutMs > 0 {
		return normalizeBrowserTimeoutMs(timeoutMs, fallback)
	}
	if timeoutSeconds, ok := getIntArg(payload, "timeoutSeconds"); ok && timeoutSeconds > 0 {
		return normalizeBrowserTimeoutMs(timeoutSeconds*1000, fallback)
	}
	return normalizeBrowserTimeoutMs(fallback, fallback)
}

func resolveBrowserActTimeoutMs(request toolArgs, payload toolArgs, fallback int) int {
	if timeoutMs, ok := getIntArg(request, "timeoutMs"); ok && timeoutMs > 0 {
		return normalizeBrowserTimeoutMs(timeoutMs, fallback)
	}
	if timeoutSeconds, ok := getIntArg(request, "timeoutSeconds"); ok && timeoutSeconds > 0 {
		return normalizeBrowserTimeoutMs(timeoutSeconds*1000, fallback)
	}
	return resolveBrowserActionTimeoutMs(payload, fallback)
}

func normalizeBrowserTimeoutMs(value int, fallback int) int {
	if value <= 0 {
		value = fallback
	}
	if value < 500 {
		return 500
	}
	if value > 120000 {
		return 120000
	}
	return value
}

func toBrowserFriendlyInteractionError(err error, selector string) error {
	if err == nil {
		return nil
	}
	message := strings.TrimSpace(err.Error())
	if message == "" {
		return err
	}
	if strings.Contains(message, "strict mode violation") {
		count := "multiple"
		if match := browserStrictModeCountPattern.FindStringSubmatch(message); len(match) >= 2 {
			count = strings.TrimSpace(match[1])
		}
		return fmt.Errorf(`Selector "%s" matched %s elements. Run a new snapshot to get updated refs, or use a different ref.`, selector, count)
	}
	if (strings.Contains(message, "Timeout") || strings.Contains(message, "waiting for")) &&
		(strings.Contains(message, "to be visible") || strings.Contains(message, "not visible")) {
		return fmt.Errorf(`Element "%s" not found or not visible. Run a new snapshot to see current page elements.`, selector)
	}
	if strings.Contains(message, "intercepts pointer events") ||
		strings.Contains(message, "not receive pointer events") ||
		strings.Contains(message, "not visible") {
		return fmt.Errorf(`Element "%s" is not interactable (hidden or covered). Try scrolling it into view, closing overlays, or re-snapshotting.`, selector)
	}
	return err
}

func addConnectorCookiesToContext(ctx context.Context, connectors ConnectorsReader, browserCtx playwright.BrowserContext, targetURL string) error {
	if connectors == nil || browserCtx == nil {
		return nil
	}
	cookies, err := resolveConnectorCookiesForURL(ctx, connectors, targetURL)
	if err != nil {
		return err
	}
	if len(cookies) == 0 {
		return nil
	}
	return browserCtx.AddCookies(toPlaywrightCookies(cookies, targetURL))
}

func assertBrowserURLAllowed(rawURL string, policy browserSSRFPolicy) error {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return errors.New("targetUrl is required")
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return fmt.Errorf("invalid url: %w", err)
	}
	if !strings.EqualFold(parsed.Scheme, "http") && !strings.EqualFold(parsed.Scheme, "https") {
		return errors.New("only http(s) urls are supported")
	}
	hostname := strings.ToLower(strings.TrimSpace(parsed.Hostname()))
	if hostname == "" {
		return errors.New("url hostname is required")
	}
	if isHostnameExplicitlyAllowed(hostname, policy) {
		return nil
	}
	if policy.DangerouslyAllowPrivateNetwork {
		return nil
	}
	if hostname == "localhost" || strings.HasSuffix(hostname, ".local") || strings.HasSuffix(hostname, ".internal") {
		return fmt.Errorf("blocked private hostname: %s", hostname)
	}
	if ip := net.ParseIP(hostname); ip != nil {
		if isPrivateOrLocalIP(ip) {
			return fmt.Errorf("blocked private IP: %s", hostname)
		}
	}
	return nil
}

func isHostnameExplicitlyAllowed(hostname string, policy browserSSRFPolicy) bool {
	hostname = strings.ToLower(strings.TrimSpace(hostname))
	if hostname == "" {
		return false
	}
	if _, ok := policy.AllowedHostnames[hostname]; ok {
		return true
	}
	for _, pattern := range policy.HostnameAllowlist {
		pattern = strings.ToLower(strings.TrimSpace(pattern))
		if pattern == "" {
			continue
		}
		if pattern == hostname {
			return true
		}
		if strings.HasPrefix(pattern, "*.") {
			suffix := strings.TrimPrefix(pattern, "*.")
			if strings.HasSuffix(hostname, "."+suffix) || hostname == suffix {
				return true
			}
			continue
		}
		if matched, _ := filepath.Match(pattern, hostname); matched {
			return true
		}
	}
	return false
}

func isPrivateOrLocalIP(ip net.IP) bool {
	if ip == nil {
		return false
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}
	if ip.IsUnspecified() || ip.IsMulticast() {
		return true
	}
	if ip4 := ip.To4(); ip4 != nil {
		// Carrier-grade NAT: 100.64.0.0/10
		if ip4[0] == 100 && (ip4[1]&0xC0) == 64 {
			return true
		}
	}
	return false
}

func waitBrowserEvaluateCondition(page playwright.Page, fn string, timeoutMs int) error {
	if timeoutMs <= 0 {
		timeoutMs = 15000
	}
	deadline := time.Now().Add(time.Duration(timeoutMs) * time.Millisecond)
	for {
		result, err := page.Evaluate(fn)
		if err == nil {
			if value, ok := result.(bool); ok && value {
				return nil
			}
			if result != nil {
				switch typed := result.(type) {
				case string:
					if strings.TrimSpace(strings.ToLower(typed)) == "true" {
						return nil
					}
				case float64:
					if typed != 0 {
						return nil
					}
				case int:
					if typed != 0 {
						return nil
					}
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

func saveBrowserArtifact(ext string, content []byte) (string, error) {
	ext = strings.TrimSpace(strings.TrimPrefix(ext, "."))
	if ext == "" {
		ext = "bin"
	}
	dir := filepath.Join(os.TempDir(), "dreamcreator", "browser")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	filename := fmt.Sprintf("%d-%d.%s", time.Now().UnixNano(), atomic.AddUint64(&browserGlobalTabCounter, 1), ext)
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		return "", err
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return path, nil
	}
	return abs, nil
}

func resolveBrowserPlaywrightRuntimeAvailability() (bool, string, string) {
	now := time.Now()
	browserPlaywrightRuntimeCache.mu.Lock()
	if !browserPlaywrightRuntimeCache.checkedAt.IsZero() && now.Sub(browserPlaywrightRuntimeCache.checkedAt) < browserRuntimeCheckCacheTTL {
		available := browserPlaywrightRuntimeCache.available
		reason := browserPlaywrightRuntimeCache.reason
		execPath := browserPlaywrightRuntimeCache.execPath
		browserPlaywrightRuntimeCache.mu.Unlock()
		return available, reason, execPath
	}
	browserPlaywrightRuntimeCache.mu.Unlock()

	available := true
	reason := ""
	execPath := ""

	pw, err := playwright.Run(&playwright.RunOptions{
		Verbose: false,
		Stdout:  io.Discard,
		Stderr:  io.Discard,
	})
	if err != nil {
		available = false
		reason = trimToMaxChars(strings.TrimSpace(err.Error()), 220)
	} else {
		defer pw.Stop()
		execPath = strings.TrimSpace(pw.Chromium.ExecutablePath())

		browser, launchErr := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
			Headless: playwright.Bool(true),
			Args:     []string{"--headless=new"},
		})
		if launchErr != nil {
			available = false
			reason = trimToMaxChars(strings.TrimSpace(launchErr.Error()), 220)
		} else {
			_ = browser.Close()
		}
	}

	browserPlaywrightRuntimeCache.mu.Lock()
	browserPlaywrightRuntimeCache.checkedAt = now
	browserPlaywrightRuntimeCache.available = available
	browserPlaywrightRuntimeCache.reason = reason
	browserPlaywrightRuntimeCache.execPath = execPath
	browserPlaywrightRuntimeCache.mu.Unlock()

	return available, reason, execPath
}

func hasBrowserHeadlessArg(args []string) bool {
	for _, arg := range args {
		if strings.Contains(strings.ToLower(strings.TrimSpace(arg)), "headless") {
			return true
		}
	}
	return false
}

func toPlaywrightScreenshotType(value string) *playwright.ScreenshotType {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "jpeg", "jpg":
		return playwright.ScreenshotTypeJpeg
	default:
		return playwright.ScreenshotTypePng
	}
}

func toPlaywrightMouseButton(value string) *playwright.MouseButton {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "left":
		return playwright.MouseButtonLeft
	case "right":
		return playwright.MouseButtonRight
	case "middle":
		return playwright.MouseButtonMiddle
	default:
		return nil
	}
}

func toPlaywrightKeyboardModifiers(values []string) ([]playwright.KeyboardModifier, error) {
	if len(values) == 0 {
		return nil, nil
	}
	result := make([]playwright.KeyboardModifier, 0, len(values))
	for _, value := range values {
		switch strings.ToLower(strings.TrimSpace(value)) {
		case "alt":
			if playwright.KeyboardModifierAlt != nil {
				result = append(result, *playwright.KeyboardModifierAlt)
			}
		case "control", "ctrl":
			if playwright.KeyboardModifierControl != nil {
				result = append(result, *playwright.KeyboardModifierControl)
			}
		case "controlormeta", "control_or_meta":
			if playwright.KeyboardModifierControlOrMeta != nil {
				result = append(result, *playwright.KeyboardModifierControlOrMeta)
			}
		case "meta", "command", "cmd":
			if playwright.KeyboardModifierMeta != nil {
				result = append(result, *playwright.KeyboardModifierMeta)
			}
		case "shift":
			if playwright.KeyboardModifierShift != nil {
				result = append(result, *playwright.KeyboardModifierShift)
			}
		case "":
			continue
		default:
			return nil, errors.New("modifiers must be Alt|Control|ControlOrMeta|Meta|Shift")
		}
	}
	if len(result) == 0 {
		return nil, nil
	}
	return result, nil
}

func containsString(values []string, candidate string) bool {
	for _, value := range values {
		if value == candidate {
			return true
		}
	}
	return false
}

func anyToString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case int:
		return strconv.Itoa(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	default:
		return ""
	}
}

func anyToInt(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case string:
		parsed, _ := strconv.Atoi(strings.TrimSpace(typed))
		return parsed
	default:
		return 0
	}
}

func minInt(a int, b int) int {
	if a <= b {
		return a
	}
	return b
}

func maxInt(a int, b int) int {
	if a >= b {
		return a
	}
	return b
}
