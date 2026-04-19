package tools

import (
	"context"
	"strings"
	"testing"

	gatewaynodes "dreamcreator/internal/application/gateway/nodes"
)

func TestRunNodesToolRequiresExplicitCapability(t *testing.T) {
	t.Parallel()

	handler := runNodesTool(newNodesServiceForTest(t, gatewaynodes.InvokerFunc(func(_ context.Context, request gatewaynodes.NodeInvokeRequest) (gatewaynodes.NodeInvokeResult, error) {
		t.Fatalf("unexpected invoke: %#v", request)
		return gatewaynodes.NodeInvokeResult{}, nil
	})))

	_, err := handler(context.Background(), `{"nodeId":"node-1","action":"screen.capture"}`)
	if err == nil {
		t.Fatalf("expected capability required error")
	}
	if !strings.Contains(err.Error(), "capability is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunNodesToolPassesActionPayloadAndTimeout(t *testing.T) {
	t.Parallel()

	var captured gatewaynodes.NodeInvokeRequest
	handler := runNodesTool(newNodesServiceForTest(t, gatewaynodes.InvokerFunc(func(_ context.Context, request gatewaynodes.NodeInvokeRequest) (gatewaynodes.NodeInvokeResult, error) {
		captured = request
		return gatewaynodes.NodeInvokeResult{
			InvokeID: request.InvokeID,
			Ok:       true,
			Output:   `{"ok":true}`,
		}, nil
	})))

	_, err := handler(context.Background(), `{"nodeId":"node-1","capability":"screen","action":"capture","payload":{"format":"png"},"timeoutMs":4321}`)
	if err != nil {
		t.Fatalf("run nodes tool: %v", err)
	}
	if captured.NodeID != "node-1" {
		t.Fatalf("expected node-1, got %q", captured.NodeID)
	}
	if captured.Capability != "screen" {
		t.Fatalf("expected capability screen, got %q", captured.Capability)
	}
	if captured.Action != "capture" {
		t.Fatalf("expected action capture, got %q", captured.Action)
	}
	if captured.Args != `{"format":"png"}` {
		t.Fatalf("unexpected args payload: %q", captured.Args)
	}
	if captured.TimeoutMs != 4321 {
		t.Fatalf("expected timeout 4321, got %d", captured.TimeoutMs)
	}
}

func newNodesServiceForTest(t *testing.T, invoker gatewaynodes.Invoker) *gatewaynodes.Service {
	t.Helper()
	registry := gatewaynodes.NewRegistry(nil, nil)
	_, err := registry.Register(context.Background(), "", gatewaynodes.NodeDescriptor{
		NodeID: "node-1",
		Status: "online",
	})
	if err != nil {
		t.Fatalf("register node: %v", err)
	}
	return gatewaynodes.NewService(registry, nil, invoker, nil, nil)
}
