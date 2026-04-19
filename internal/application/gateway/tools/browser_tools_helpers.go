package tools

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync/atomic"
	"time"

	"dreamcreator/internal/application/browsercdp"
	gatewaynodes "dreamcreator/internal/application/gateway/nodes"
	targetpkg "github.com/chromedp/cdproto/target"
)

const (
	browserTypeCDP     = "cdp"
	defaultBrowserType = browserTypeCDP

	defaultBrowserWaitUntil = "domcontentloaded"

	defaultBrowserProfileDreamCreator = "dreamcreator"
	defaultBrowserColor               = "#FF4500"

	defaultBrowserSnapshotAIMaxChars          = 80000
	defaultBrowserSnapshotAIEfficientMaxChars = 10000
	defaultBrowserSnapshotDepth               = 6
	defaultBrowserSnapshotLimit               = 200
	defaultBrowserViewportWidth               = 1366
	defaultBrowserViewportHeight              = 900
)

var browserToolActions = []string{
	"open",
	"navigate",
	"snapshot",
	"act",
	"wait",
	"scroll",
	"upload",
	"dialog",
	"reset",
}

var browserSelectorUnsupportedMessage = strings.Join([]string{
	"Error: 'selector' is not supported. Use 'ref' from snapshot instead.",
	"",
	"Example workflow:",
	"1. open or navigate to load the page",
	"2. read returned items, or run snapshot to get fresh refs",
	`3. act with ref: "e123" to interact with an element`,
	"4. after the page changes, read returned items or run snapshot again",
	"",
	"This is more reliable for modern SPAs.",
}, "\n")

var browserWaitRequiresConditionMessage = "wait requires at least one of: timeMs, text, textGone, selector, url, fn"
var errBrowserNoOpenTab = errors.New("no browser tab is open")
var errBrowserSnapshotForAIUnavailable = errors.New("browser snapshotForAI is unavailable")
var browserActKinds = []string{"click", "type", "press", "hover", "select", "fill", "resize", "wait", "evaluate", "close"}
var browserGlobalTabCounter uint64

type browserSnapshotItem struct {
	Ref     string `json:"ref,omitempty"`
	AriaRef string `json:"ariaRef,omitempty"`
	Role    string `json:"role,omitempty"`
	Name    string `json:"name,omitempty"`
	Text    string `json:"text,omitempty"`
	Depth   int    `json:"depth,omitempty"`
	Nth     int    `json:"nth,omitempty"`
}

var browserAriaSnapshotLinePattern = regexp.MustCompile(`^(\s*)-\s*([^\s":]+)(?:\s+"([^"]*)")?(.*)$`)
var browserAriaSnapshotRefPattern = regexp.MustCompile(`\[ref=([^\]]+)\]`)
var browserStrictModeCountPattern = regexp.MustCompile(`resolved to (\d+) elements`)

type browserResolvedConfig struct {
	Enabled          bool
	Color            string
	Headless         bool
	DefaultProfile   string
	PreferredBrowser string
	Profiles         map[string]browserProfileConfig
	SSRFRules        browserSSRFPolicy
}

type browserProfileConfig struct {
	Name   string
	Color  string
	Driver string
}

type browserSSRFPolicy struct {
	DangerouslyAllowPrivateNetwork bool
	AllowedHostnames               map[string]struct{}
	HostnameAllowlist              []string
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

func resolveBrowserAction(payload toolArgs) (string, error) {
	rawAction := strings.ToLower(strings.TrimSpace(getStringArg(payload, "action")))
	if rawAction == "" {
		return "", errors.New("browser action is required")
	}
	switch rawAction {
	case "open", "navigate", "snapshot", "act", "wait", "scroll", "upload", "dialog", "reset":
		return rawAction, nil
	default:
		return "", errors.New("browser action not supported: " + rawAction)
	}
}

func pickReusableBrowserTargetID(infos []*targetpkg.Info) string {
	choose := func(requireUnattached bool, preferBlank bool) string {
		for _, info := range infos {
			if info == nil || info.Type != "page" {
				continue
			}
			if requireUnattached && info.Attached {
				continue
			}
			if preferBlank && !isReusableBrowserPageURL(info.URL) {
				continue
			}
			return string(info.TargetID)
		}
		return ""
	}
	for _, candidate := range []string{
		choose(true, true),
		choose(true, false),
		choose(false, true),
		choose(false, false),
	} {
		if strings.TrimSpace(candidate) != "" {
			return candidate
		}
	}
	return ""
}

func isReusableBrowserPageURL(rawURL string) bool {
	trimmed := strings.TrimSpace(strings.ToLower(rawURL))
	switch trimmed {
	case "", "about:blank", "chrome://newtab/", "chrome-search://local-ntp/local-ntp.html":
		return true
	default:
		return false
	}
}

func shouldTreatBrowserNavigationAsComplete(observedURL string, previousURL string, targetURL string) bool {
	observed := strings.TrimSpace(observedURL)
	if observed == "" || observed == "about:blank" {
		return false
	}
	if urlsEqual(observed, targetURL) {
		return true
	}
	previous := strings.TrimSpace(previousURL)
	if previous == "" || previous == "about:blank" {
		return true
	}
	return !urlsEqual(observed, previous)
}

func urlsEqual(left string, right string) bool {
	return strings.TrimSpace(left) == strings.TrimSpace(right)
}

func shouldResetBrowserProfileAfterError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(strings.TrimSpace(err.Error()))
	switch {
	case strings.Contains(message, "browser runtime unavailable"),
		strings.Contains(message, "context canceled"),
		strings.Contains(message, "target closed"),
		strings.Contains(message, "connection closed"),
		strings.Contains(message, "websocket"),
		strings.Contains(message, "session closed"),
		strings.Contains(message, "browser session reset"):
		return true
	default:
		return false
	}
}

func shouldDeferBrowserStateCaptureError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(strings.TrimSpace(err.Error()))
	switch {
	case strings.Contains(message, "context deadline exceeded"),
		strings.Contains(message, "execution context was destroyed"),
		strings.Contains(message, "cannot find context with specified id"),
		strings.Contains(message, "unique context id not found"):
		return true
	default:
		return false
	}
}

func isBrowserNodeTargetRequest(payload toolArgs) bool {
	target := strings.ToLower(strings.TrimSpace(getStringArg(payload, "target")))
	nodeID := strings.TrimSpace(getStringArg(payload, "node", "nodeId"))
	return target == "node" || nodeID != ""
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
		if nodeID := strings.TrimSpace(descriptor.NodeID); nodeID != "" {
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

func resolveBrowserRuntimeConfig(config map[string]any) browserResolvedConfig {
	browserConfig := getNestedMap(config, "browser")
	resolved := browserResolvedConfig{
		Enabled:          true,
		Color:            defaultBrowserColor,
		Headless:         false,
		DefaultProfile:   defaultBrowserProfileDreamCreator,
		PreferredBrowser: string(browsercdp.BrowserChrome),
		Profiles: map[string]browserProfileConfig{
			defaultBrowserProfileDreamCreator: {
				Name:   defaultBrowserProfileDreamCreator,
				Color:  defaultBrowserColor,
				Driver: browserTypeCDP,
			},
		},
		SSRFRules: browserSSRFPolicy{
			DangerouslyAllowPrivateNetwork: false,
			AllowedHostnames:               map[string]struct{}{},
		},
	}
	if browserConfig == nil {
		return resolved
	}
	if value, ok := getBoolArg(toolArgs(browserConfig), "enabled"); ok {
		resolved.Enabled = value
	}
	if value := strings.TrimSpace(getStringArg(toolArgs(browserConfig), "color")); value != "" {
		resolved.Color = value
	}
	if value, ok := getBoolArg(toolArgs(browserConfig), "headless"); ok {
		resolved.Headless = value
	}
	if value := strings.TrimSpace(getStringArg(toolArgs(browserConfig), "preferredBrowser")); value != "" {
		resolved.PreferredBrowser = strings.ToLower(value)
	}
	if ssrfRaw := getMapArg(toolArgs(browserConfig), "ssrfPolicy"); ssrfRaw != nil {
		if value, ok := getBoolArg(toolArgs(ssrfRaw), "dangerouslyAllowPrivateNetwork"); ok {
			resolved.SSRFRules.DangerouslyAllowPrivateNetwork = value
		}
		for _, hostname := range getStringSliceArg(toolArgs(ssrfRaw), "allowedHostnames") {
			resolved.SSRFRules.AllowedHostnames[strings.ToLower(strings.TrimSpace(hostname))] = struct{}{}
		}
		if allowlist := normalizeStringSlice(getStringSliceArg(toolArgs(ssrfRaw), "hostnameAllowlist")); len(allowlist) > 0 {
			resolved.SSRFRules.HostnameAllowlist = allowlist
		}
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
	return profile
}

func resolveBrowserConfig(config map[string]any) map[string]any {
	return getNestedMap(config, "browser")
}

func resolveBrowserConfigBool(config map[string]any, key string) (bool, bool) {
	return getBoolArg(toolArgs(resolveBrowserConfig(config)), key)
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

func resolveBrowserSnapshotLimit(payload toolArgs) int {
	if value, ok := getIntArg(payload, "limit"); ok && value > 0 {
		return value
	}
	return defaultBrowserSnapshotLimit
}

func resolveBrowserScrollDelta(payload toolArgs) (int, int) {
	if x, ok := getIntArg(payload, "x"); ok {
		if y, ok := getIntArg(payload, "y"); ok {
			return x, y
		}
		return x, 0
	}
	if y, ok := getIntArg(payload, "y"); ok {
		return 0, y
	}
	amount, ok := getIntArg(payload, "amount")
	if !ok || amount <= 0 {
		amount = 700
	}
	switch strings.ToLower(strings.TrimSpace(getStringArg(payload, "direction"))) {
	case "up":
		return 0, -amount
	case "left":
		return -amount, 0
	case "right":
		return amount, 0
	default:
		return 0, amount
	}
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

func parseBrowserAriaSnapshot(snapshot string, interactive bool, limit int, useAriaRefs bool, maxDepth int) []browserSnapshotItem {
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
		nextRef++
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
			spaces++
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
	case "button", "link", "textbox", "checkbox", "radio", "combobox", "listbox", "menuitem", "menuitemcheckbox",
		"menuitemradio", "option", "searchbox", "slider", "spinbutton", "switch", "tab", "treeitem":
		return true
	default:
		return false
	}
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

func isBrowserSnapshotForAIUnavailable(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(strings.TrimSpace(err.Error()))
	return strings.Contains(message, "_snapshotforai is not available") ||
		strings.Contains(message, "snapshotforai is unavailable")
}

func toBrowserFriendlyInteractionError(err error, selector string) error {
	if err == nil {
		return nil
	}
	message := strings.TrimSpace(err.Error())
	if strings.Contains(message, "strict mode violation") {
		count := "multiple"
		if match := browserStrictModeCountPattern.FindStringSubmatch(message); len(match) >= 2 {
			count = strings.TrimSpace(match[1])
		}
		return fmt.Errorf(`Selector "%s" matched %s elements. Run snapshot again to get updated refs, or use a different ref.`, selector, count)
	}
	return err
}

func assertBrowserURLAllowed(rawURL string, policy browserSSRFPolicy) error {
	return browsercdp.AssertURLAllowed(rawURL, browsercdp.SSRFPolicy{
		DangerouslyAllowPrivateNetwork: policy.DangerouslyAllowPrivateNetwork,
		AllowedHostnames:               cloneBrowserAllowedHostnames(policy.AllowedHostnames),
		HostnameAllowlist:              append([]string(nil), policy.HostnameAllowlist...),
	})
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
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() || ip.IsMulticast() {
		return true
	}
	if ip4 := ip.To4(); ip4 != nil {
		if ip4[0] == 100 && (ip4[1]&0xC0) == 64 {
			return true
		}
	}
	return false
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
	return filepath.Abs(path)
}

func resolveBrowserRuntimeAvailability(preferred string, headless bool) browsercdp.Status {
	return browsercdp.ResolveStatus(preferred, headless)
}

func minInt(left int, right int) int {
	if left < right {
		return left
	}
	return right
}
