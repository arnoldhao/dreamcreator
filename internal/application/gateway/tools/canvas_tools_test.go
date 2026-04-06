package tools

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	gatewaynodes "dreamcreator/internal/application/gateway/nodes"
)

func TestRunCanvasToolSnapshotPersistsImage(t *testing.T) {
	t.Parallel()

	nodes := newCanvasNodesServiceForTest(t, gatewaynodes.InvokerFunc(func(_ context.Context, request gatewaynodes.NodeInvokeRequest) (gatewaynodes.NodeInvokeResult, error) {
		if request.Capability != "canvas" {
			t.Fatalf("expected canvas capability, got %q", request.Capability)
		}
		if request.Action != "snapshot" {
			t.Fatalf("expected snapshot action, got %q", request.Action)
		}
		output := `{"payload":{"format":"png","base64":"` + base64.StdEncoding.EncodeToString([]byte("png-bytes")) + `"}}`
		return gatewaynodes.NodeInvokeResult{InvokeID: request.InvokeID, Ok: true, Output: output}, nil
	}))

	handler := runCanvasTool(nodes)
	output, err := handler(context.Background(), `{"action":"snapshot","node":"node-1"}`)
	if err != nil {
		t.Fatalf("run canvas snapshot: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	if ok, _ := result["ok"].(bool); !ok {
		t.Fatalf("expected ok=true, got %#v", result["ok"])
	}
	path, _ := result["path"].(string)
	if strings.TrimSpace(path) == "" {
		t.Fatalf("expected snapshot path, got %#v", result)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected snapshot file written: %v", err)
	}
	if format, _ := result["format"].(string); format != "png" {
		t.Fatalf("expected format png, got %q", format)
	}
}

func TestRunCanvasToolPresentUsesTargetAndPlacement(t *testing.T) {
	t.Parallel()

	var captured gatewaynodes.NodeInvokeRequest
	nodes := newCanvasNodesServiceForTest(t, gatewaynodes.InvokerFunc(func(_ context.Context, request gatewaynodes.NodeInvokeRequest) (gatewaynodes.NodeInvokeResult, error) {
		captured = request
		return gatewaynodes.NodeInvokeResult{InvokeID: request.InvokeID, Ok: true, Output: "{}"}, nil
	}))

	handler := runCanvasTool(nodes)
	_, err := handler(context.Background(), `{"action":"present","node":"node-1","target":"https://example.com","x":10,"width":320}`)
	if err != nil {
		t.Fatalf("run canvas present: %v", err)
	}
	if captured.Action != "present" {
		t.Fatalf("expected present action, got %q", captured.Action)
	}
	var invokePayload map[string]any
	if err := json.Unmarshal([]byte(captured.Args), &invokePayload); err != nil {
		t.Fatalf("decode invoke args: %v", err)
	}
	if invokePayload["url"] != "https://example.com" {
		t.Fatalf("expected url from target alias, got %#v", invokePayload["url"])
	}
	placement, ok := invokePayload["placement"].(map[string]any)
	if !ok {
		t.Fatalf("expected placement payload, got %#v", invokePayload)
	}
	if placement["x"] != float64(10) {
		t.Fatalf("expected x=10, got %#v", placement["x"])
	}
	if placement["width"] != float64(320) {
		t.Fatalf("expected width=320, got %#v", placement["width"])
	}
}

func TestRunCanvasToolA2UIPushBlocksPathOutsideAllowedRoots(t *testing.T) {
	t.Parallel()

	nodes := newCanvasNodesServiceForTest(t, gatewaynodes.InvokerFunc(func(_ context.Context, request gatewaynodes.NodeInvokeRequest) (gatewaynodes.NodeInvokeResult, error) {
		return gatewaynodes.NodeInvokeResult{InvokeID: request.InvokeID, Ok: true, Output: "{}"}, nil
	}))
	handler := runCanvasTool(nodes)
	workspaceRoot, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	inputPath := filepath.Join(workspaceRoot, "go.mod")
	args, err := json.Marshal(map[string]any{
		"action":    "a2ui_push",
		"node":      "node-1",
		"jsonlPath": inputPath,
	})
	if err != nil {
		t.Fatalf("marshal args: %v", err)
	}
	_, err = handler(context.Background(), string(args))
	if err == nil {
		t.Fatalf("expected jsonlPath blocked error")
	}
	if !strings.Contains(err.Error(), "outside allowed roots") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNormalizeCanvasGatewayURL_DefaultPath(t *testing.T) {
	t.Parallel()

	wsURL, origin, err := normalizeCanvasGatewayURL("ws://127.0.0.1:18789")
	if err != nil {
		t.Fatalf("normalize gateway url: %v", err)
	}
	if wsURL != "ws://127.0.0.1:18789/gateway/ws" {
		t.Fatalf("unexpected websocket url: %s", wsURL)
	}
	if origin != "http://127.0.0.1:18789" {
		t.Fatalf("unexpected origin: %s", origin)
	}
}

func TestNormalizeCanvasGatewayURL_InvalidProtocol(t *testing.T) {
	t.Parallel()

	_, _, err := normalizeCanvasGatewayURL("http://127.0.0.1:18789")
	if err == nil {
		t.Fatalf("expected invalid protocol error")
	}
	if !strings.Contains(err.Error(), "protocol") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNormalizeCanvasGatewayURL_RejectsQueryAndPath(t *testing.T) {
	t.Parallel()

	_, _, err := normalizeCanvasGatewayURL("ws://127.0.0.1:18789/custom/path")
	if err == nil {
		t.Fatalf("expected invalid path error")
	}
	if !strings.Contains(err.Error(), "path") {
		t.Fatalf("unexpected error: %v", err)
	}

	_, _, err = normalizeCanvasGatewayURL("ws://127.0.0.1:18789/gateway/ws")
	if err == nil {
		t.Fatalf("expected path override to be rejected")
	}
	if !strings.Contains(err.Error(), "path") {
		t.Fatalf("unexpected error: %v", err)
	}

	_, _, err = normalizeCanvasGatewayURL("ws://127.0.0.1:18789?debug=1")
	if err == nil {
		t.Fatalf("expected invalid query error")
	}
	if !strings.Contains(err.Error(), "query/hash") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNormalizeCanvasGatewayURL_RejectsNonLoopbackHost(t *testing.T) {
	t.Parallel()

	_, _, err := normalizeCanvasGatewayURL("ws://example.com:18789")
	if err == nil {
		t.Fatalf("expected non-loopback host to be rejected")
	}
	if !strings.Contains(err.Error(), "loopback") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func newCanvasNodesServiceForTest(t *testing.T, invoker gatewaynodes.Invoker) *gatewaynodes.Service {
	t.Helper()
	registry := gatewaynodes.NewRegistry(nil, nil)
	_, err := registry.Register(context.Background(), "", gatewaynodes.NodeDescriptor{
		NodeID: "node-1",
		Capabilities: []gatewaynodes.NodeCapability{
			{Name: "canvas"},
		},
		Status: "online",
	})
	if err != nil {
		t.Fatalf("register node: %v", err)
	}
	return gatewaynodes.NewService(registry, nil, invoker, nil, nil)
}
