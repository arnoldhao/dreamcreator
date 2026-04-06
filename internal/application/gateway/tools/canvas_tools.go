package tools

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math"
	"os"
	"strings"

	gatewaynodes "dreamcreator/internal/application/gateway/nodes"
)

const (
	defaultCanvasInvokeTimeoutMs = 30000
	minCanvasInvokeTimeoutMs     = 500
	maxCanvasInvokeTimeoutMs     = 120000
)

func runCanvasTool(nodes *gatewaynodes.Service) func(ctx context.Context, args string) (string, error) {
	return func(ctx context.Context, args string) (string, error) {
		if nodes == nil {
			return "", errors.New("nodes service unavailable")
		}
		payload, err := parseToolArgs(args)
		if err != nil {
			return "", err
		}
		bridge, err := resolveCanvasNodeInvoker(ctx, payload, nodes)
		if err != nil {
			return "", err
		}
		defer bridge.Close()
		action, err := resolveCanvasAction(payload)
		if err != nil {
			return "", err
		}
		nodeID, err := resolveCanvasNodeID(ctx, payload, bridge.Invoker)
		if err != nil {
			return "", err
		}
		timeoutMs := resolveCanvasTimeoutMs(payload, defaultCanvasInvokeTimeoutMs)

		switch action {
		case "present":
			invokeParams := map[string]any{}
			if target := getStringArg(payload, "target", "url"); target != "" {
				invokeParams["url"] = target
			}
			placement := map[string]any{}
			if value, ok := getNumberArg(payload, "x"); ok {
				placement["x"] = value
			}
			if value, ok := getNumberArg(payload, "y"); ok {
				placement["y"] = value
			}
			if value, ok := getNumberArg(payload, "width"); ok {
				placement["width"] = value
			}
			if value, ok := getNumberArg(payload, "height"); ok {
				placement["height"] = value
			}
			if len(placement) > 0 {
				invokeParams["placement"] = placement
			}
			if _, err := invokeCanvasNodeCommand(ctx, bridge.Invoker, nodeID, "present", invokeParams, timeoutMs); err != nil {
				return "", err
			}
			return marshalResult(map[string]any{"ok": true, "nodeId": nodeID}), nil
		case "hide":
			if _, err := invokeCanvasNodeCommand(ctx, bridge.Invoker, nodeID, "hide", nil, timeoutMs); err != nil {
				return "", err
			}
			return marshalResult(map[string]any{"ok": true, "nodeId": nodeID}), nil
		case "navigate":
			urlValue := getStringArg(payload, "url", "target")
			if urlValue == "" {
				return "", errors.New("url is required")
			}
			if _, err := invokeCanvasNodeCommand(ctx, bridge.Invoker, nodeID, "navigate", map[string]any{"url": urlValue}, timeoutMs); err != nil {
				return "", err
			}
			return marshalResult(map[string]any{"ok": true, "nodeId": nodeID}), nil
		case "eval":
			javaScript := strings.TrimSpace(getStringArg(payload, "javaScript"))
			if javaScript == "" {
				return "", errors.New("javaScript is required")
			}
			raw, err := invokeCanvasNodeCommand(ctx, bridge.Invoker, nodeID, "eval", map[string]any{"javaScript": javaScript}, timeoutMs)
			if err != nil {
				return "", err
			}
			result := map[string]any{"ok": true, "nodeId": nodeID}
			if evalText := extractCanvasEvalResult(raw); evalText != "" {
				result["result"] = evalText
				result["content"] = []map[string]any{
					{
						"type": "text",
						"text": evalText,
					},
				}
				result["details"] = map[string]any{
					"result": evalText,
				}
			} else if raw != nil {
				result["output"] = raw
			}
			return marshalResult(result), nil
		case "snapshot":
			format := normalizeCanvasSnapshotFormat(getStringArg(payload, "outputFormat"))
			invokeParams := map[string]any{"format": format}
			if maxWidth, ok := getNumberArg(payload, "maxWidth"); ok {
				invokeParams["maxWidth"] = maxWidth
			}
			if quality, ok := getNumberArg(payload, "quality"); ok {
				invokeParams["quality"] = quality
			}
			if delayMs, ok := getNumberArg(payload, "delayMs"); ok {
				invokeParams["delayMs"] = delayMs
			}
			raw, err := invokeCanvasNodeCommand(ctx, bridge.Invoker, nodeID, "snapshot", invokeParams, timeoutMs)
			if err != nil {
				return "", err
			}
			if existingPath := extractCanvasSnapshotPath(raw); existingPath != "" {
				encoded := ""
				if bytes, readErr := os.ReadFile(existingPath); readErr == nil && len(bytes) > 0 {
					encoded = base64.StdEncoding.EncodeToString(bytes)
				}
				return marshalResult(buildCanvasSnapshotResult(nodeID, existingPath, format, encoded)), nil
			}
			payloadFormat, encoded, err := extractCanvasSnapshotPayload(raw)
			if err != nil {
				return "", err
			}
			bytes, err := base64.StdEncoding.DecodeString(encoded)
			if err != nil {
				return "", errors.New("invalid canvas.snapshot payload")
			}
			ext := "png"
			if payloadFormat == "jpeg" {
				ext = "jpg"
			}
			path, err := saveBrowserArtifact(ext, bytes)
			if err != nil {
				return "", err
			}
			return marshalResult(buildCanvasSnapshotResult(nodeID, path, payloadFormat, encoded)), nil
		case "a2ui_push":
			jsonl := strings.TrimSpace(getStringArg(payload, "jsonl"))
			if jsonl == "" {
				jsonlPath := strings.TrimSpace(getStringArg(payload, "jsonlPath"))
				if jsonlPath == "" {
					return "", errors.New("jsonl or jsonlPath required")
				}
				resolvedPath, err := resolveInboundPath(jsonlPath, nil)
				if err != nil {
					return "", errors.New("jsonlPath outside allowed roots")
				}
				bytes, err := os.ReadFile(resolvedPath)
				if err != nil {
					return "", err
				}
				jsonl = strings.TrimSpace(string(bytes))
			}
			if jsonl == "" {
				return "", errors.New("jsonl or jsonlPath required")
			}
			if _, err := invokeCanvasNodeCommand(ctx, bridge.Invoker, nodeID, "a2ui.pushJSONL", map[string]any{"jsonl": jsonl}, timeoutMs); err != nil {
				return "", err
			}
			return marshalResult(map[string]any{"ok": true, "nodeId": nodeID}), nil
		case "a2ui_reset":
			if _, err := invokeCanvasNodeCommand(ctx, bridge.Invoker, nodeID, "a2ui.reset", nil, timeoutMs); err != nil {
				return "", err
			}
			return marshalResult(map[string]any{"ok": true, "nodeId": nodeID}), nil
		default:
			return "", errors.New("canvas action not supported: " + action)
		}
	}
}

func resolveCanvasAction(payload toolArgs) (string, error) {
	action := strings.ToLower(strings.TrimSpace(getStringArg(payload, "action")))
	if action == "" {
		return "", errors.New("action is required")
	}
	switch action {
	case "present", "hide", "navigate", "eval", "snapshot", "a2ui_push", "a2ui_reset":
		return action, nil
	default:
		return "", errors.New("canvas action not supported: " + action)
	}
}

func resolveCanvasNodeID(ctx context.Context, payload toolArgs, nodes canvasNodeInvoker) (string, error) {
	requested := strings.TrimSpace(getStringArg(payload, "node", "nodeId", "nodeID"))
	if requested != "" {
		return requested, nil
	}
	if nodes == nil {
		return "", errors.New("nodes service unavailable")
	}
	list, err := nodes.ListNodes(ctx)
	if err != nil {
		return "", err
	}
	picked := pickDefaultCanvasNode(list)
	if picked != "" {
		return picked, nil
	}
	return "", errors.New("node is required")
}

func pickDefaultCanvasNode(list []gatewaynodes.NodeDescriptor) string {
	if len(list) == 0 {
		return ""
	}
	withCanvas := make([]gatewaynodes.NodeDescriptor, 0, len(list))
	for _, node := range list {
		nodeID := strings.TrimSpace(node.NodeID)
		if nodeID == "" {
			continue
		}
		if !nodeHasCanvasCapability(node) {
			continue
		}
		withCanvas = append(withCanvas, node)
	}
	if len(withCanvas) == 0 {
		return ""
	}
	connected := make([]gatewaynodes.NodeDescriptor, 0, len(withCanvas))
	for _, node := range withCanvas {
		if isCanvasNodeConnected(node.Status) {
			connected = append(connected, node)
		}
	}
	candidates := withCanvas
	if len(connected) > 0 {
		candidates = connected
	}
	if len(candidates) == 1 {
		return strings.TrimSpace(candidates[0].NodeID)
	}
	localMac := make([]gatewaynodes.NodeDescriptor, 0, len(candidates))
	for _, node := range candidates {
		nodeID := strings.TrimSpace(node.NodeID)
		if nodeID == "" {
			continue
		}
		platform := strings.ToLower(strings.TrimSpace(node.Platform))
		if strings.HasPrefix(platform, "mac") && strings.HasPrefix(nodeID, "mac-") {
			localMac = append(localMac, node)
		}
	}
	if len(localMac) == 1 {
		return strings.TrimSpace(localMac[0].NodeID)
	}
	return ""
}

func nodeHasCanvasCapability(node gatewaynodes.NodeDescriptor) bool {
	if len(node.Capabilities) == 0 {
		return true
	}
	for _, capability := range node.Capabilities {
		name := strings.ToLower(strings.TrimSpace(capability.Name))
		if name == "canvas" || strings.HasPrefix(name, "canvas.") {
			return true
		}
	}
	return false
}

func isCanvasNodeConnected(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "", "online", "connected":
		return true
	default:
		return false
	}
}

func invokeCanvasNodeCommand(
	ctx context.Context,
	nodes canvasNodeInvoker,
	nodeID string,
	action string,
	params map[string]any,
	timeoutMs int,
) (any, error) {
	if nodes == nil {
		return nil, errors.New("nodes service unavailable")
	}
	args := ""
	if len(params) > 0 {
		encoded, err := json.Marshal(params)
		if err != nil {
			return nil, err
		}
		args = string(encoded)
	}
	result, err := nodes.Invoke(ctx, gatewaynodes.NodeInvokeRequest{
		NodeID:     strings.TrimSpace(nodeID),
		Capability: "canvas",
		Action:     strings.TrimSpace(action),
		Args:       args,
		TimeoutMs:  timeoutMs,
	})
	if err != nil {
		if message := strings.TrimSpace(result.Error); message != "" {
			return nil, errors.New(message)
		}
		return nil, err
	}
	if !result.Ok {
		if message := strings.TrimSpace(result.Error); message != "" {
			return nil, errors.New(message)
		}
		return nil, errors.New("canvas invoke failed")
	}
	return parseCanvasNodeOutput(result.Output), nil
}

func parseCanvasNodeOutput(output string) any {
	trimmed := strings.TrimSpace(output)
	if trimmed == "" {
		return nil
	}
	if parsed := resolveBrowserNodeOutput(trimmed); parsed != nil {
		return parsed
	}
	var decoded any
	if err := json.Unmarshal([]byte(trimmed), &decoded); err == nil {
		return decoded
	}
	return map[string]any{"output": trimmed}
}

func extractCanvasEvalResult(raw any) string {
	candidates := collectCanvasPayloadMaps(raw)
	for _, candidate := range candidates {
		if value := strings.TrimSpace(toCanvasString(candidate["result"])); value != "" {
			return value
		}
	}
	return ""
}

func extractCanvasSnapshotPath(raw any) string {
	candidates := collectCanvasPayloadMaps(raw)
	for _, candidate := range candidates {
		if value := strings.TrimSpace(toCanvasString(candidate["path"])); value != "" {
			return value
		}
		if value := strings.TrimSpace(toCanvasString(candidate["imagePath"])); value != "" {
			return value
		}
		download, ok := candidate["download"].(map[string]any)
		if ok {
			if value := strings.TrimSpace(toCanvasString(download["path"])); value != "" {
				return value
			}
		}
	}
	return ""
}

func extractCanvasSnapshotPayload(raw any) (string, string, error) {
	candidates := collectCanvasPayloadMaps(raw)
	for _, candidate := range candidates {
		format := normalizeCanvasSnapshotFormat(toCanvasString(candidate["format"]))
		encoded := strings.TrimSpace(toCanvasString(candidate["base64"]))
		if format != "" && encoded != "" {
			return format, encoded, nil
		}
	}
	return "", "", errors.New("invalid canvas.snapshot payload")
}

func collectCanvasPayloadMaps(raw any) []map[string]any {
	result := make([]map[string]any, 0, 4)
	queue := []any{raw}
	for len(queue) > 0 {
		item := queue[0]
		queue = queue[1:]
		object, ok := item.(map[string]any)
		if !ok {
			continue
		}
		result = append(result, object)
		if nested, ok := object["payload"].(map[string]any); ok {
			queue = append(queue, nested)
		}
		if nested, ok := object["result"].(map[string]any); ok {
			queue = append(queue, nested)
		}
	}
	return result
}

func toCanvasString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case json.RawMessage:
		var asString string
		if err := json.Unmarshal(typed, &asString); err == nil {
			return asString
		}
		return string(typed)
	default:
		if typed == nil {
			return ""
		}
		encoded, err := json.Marshal(typed)
		if err != nil {
			return ""
		}
		return string(encoded)
	}
}

func normalizeCanvasSnapshotFormat(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "jpeg", "jpg":
		return "jpeg"
	case "png", "":
		return "png"
	default:
		return "png"
	}
}

func canvasSnapshotMimeType(format string) string {
	switch normalizeCanvasSnapshotFormat(format) {
	case "jpeg":
		return "image/jpeg"
	default:
		return "image/png"
	}
}

func buildCanvasSnapshotResult(nodeID string, path string, format string, encoded string) map[string]any {
	details := map[string]any{
		"path":   path,
		"format": normalizeCanvasSnapshotFormat(format),
	}
	content := []map[string]any{
		{
			"type": "text",
			"text": "MEDIA:" + path,
		},
	}
	if strings.TrimSpace(encoded) != "" {
		content = append(content, map[string]any{
			"type":     "image",
			"data":     encoded,
			"mimeType": canvasSnapshotMimeType(format),
		})
	}
	return map[string]any{
		"ok":      true,
		"nodeId":  nodeID,
		"path":    path,
		"format":  normalizeCanvasSnapshotFormat(format),
		"content": content,
		"details": details,
	}
}

func resolveCanvasTimeoutMs(payload toolArgs, fallback int) int {
	value := fallback
	if parsed, ok := getNumberArg(payload, "timeoutMs"); ok {
		value = int(parsed)
	}
	if value <= 0 {
		value = fallback
	}
	if value < minCanvasInvokeTimeoutMs {
		return minCanvasInvokeTimeoutMs
	}
	if value > maxCanvasInvokeTimeoutMs {
		return maxCanvasInvokeTimeoutMs
	}
	return value
}

func getNumberArg(args toolArgs, keys ...string) (float64, bool) {
	for _, key := range keys {
		if key == "" {
			continue
		}
		raw, ok := args[key]
		if !ok {
			continue
		}
		switch typed := raw.(type) {
		case float64:
			if !math.IsNaN(typed) && !math.IsInf(typed, 0) {
				return typed, true
			}
		case float32:
			value := float64(typed)
			if !math.IsNaN(value) && !math.IsInf(value, 0) {
				return value, true
			}
		case int:
			return float64(typed), true
		case int64:
			return float64(typed), true
		case json.Number:
			if value, err := typed.Float64(); err == nil {
				return value, true
			}
		case string:
			value := strings.TrimSpace(typed)
			if value == "" {
				continue
			}
			if parsed, err := json.Number(value).Float64(); err == nil {
				return parsed, true
			}
		}
	}
	return 0, false
}
