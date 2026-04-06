package tools

import (
	"context"
	"strings"
	"testing"

	gatewayapprovals "dreamcreator/internal/application/gateway/approvals"
	settingsdto "dreamcreator/internal/application/settings/dto"
	tooldto "dreamcreator/internal/application/tools/dto"
	toolservice "dreamcreator/internal/application/tools/service"
)

type testToolExecutor struct {
	output string
	calls  int
}

func (executor *testToolExecutor) Execute(_ context.Context, _ tooldto.ToolSpec, _ string) (string, error) {
	executor.calls++
	return executor.output, nil
}

type runtimeContextCaptureExecutor struct {
	sessionKey string
	calls      int
}

func (executor *runtimeContextCaptureExecutor) Execute(ctx context.Context, _ tooldto.ToolSpec, _ string) (string, error) {
	executor.calls++
	sessionKey, _ := RuntimeContextFromContext(ctx)
	executor.sessionKey = strings.TrimSpace(sessionKey)
	return `{"ok":true}`, nil
}

type autoApprovePublisher struct {
	service *gatewayapprovals.Service
}

func (publisher autoApprovePublisher) Publish(ctx context.Context, eventType string, payload any) error {
	if eventType != "exec.approval.requested" || publisher.service == nil {
		return nil
	}
	request, ok := payload.(gatewayapprovals.Request)
	if !ok || strings.TrimSpace(request.ID) == "" {
		return nil
	}
	_, err := publisher.service.Resolve(ctx, request.ID, "approve", "auto-approved in test")
	return err
}

func TestInvokeWithPolicyApprovalGrantedExecutesTool(t *testing.T) {
	t.Parallel()

	settings := gatewayToolSettingsStub{
		settings: settingsdto.Settings{
			Gateway: settingsdto.GatewaySettings{ControlPlaneEnabled: true},
			Tools:   map[string]any{},
		},
	}
	tools := toolservice.NewToolService()
	tools.SetPolicy(NewPolicyPipeline(settings))
	executor := &testToolExecutor{output: `{"ok":true}`}
	tools.SetExecutor(executor)
	if _, err := tools.RegisterTool(context.Background(), tooldto.RegisterToolRequest{
		Spec: tooldto.ToolSpec{
			ID:               "approval_tool",
			Name:             "approval_tool",
			Enabled:          true,
			RiskLevel:        "low",
			RequiresApproval: true,
		},
	}); err != nil {
		t.Fatalf("register approval_tool: %v", err)
	}

	approvals := gatewayapprovals.NewService(nil)
	approvals.SetEventPublisher(autoApprovePublisher{service: approvals})
	service := NewService(tools, approvals, nil, settings, nil, nil)

	response, err := service.InvokeWithPolicy(
		context.Background(),
		tooldto.ToolsInvokeRequest{
			Tool:       "approval_tool",
			Args:       `{"value":"ok"}`,
			SessionKey: "session-1",
		},
		tooldto.ToolPolicyContext{SessionKey: "session-1"},
	)
	if err != nil {
		t.Fatalf("invoke approval_tool: %v", err)
	}
	if response.Policy.Decision != "allow" {
		t.Fatalf("expected allow decision after approval, got %q", response.Policy.Decision)
	}
	if response.Result.ErrorMessage != "" {
		t.Fatalf("expected empty error message, got %q", response.Result.ErrorMessage)
	}
	if strings.TrimSpace(response.Result.OutputJSON) != `{"ok":true}` {
		t.Fatalf("expected output json to be returned, got %q", response.Result.OutputJSON)
	}
	if executor.calls != 1 {
		t.Fatalf("expected executor called once, got %d", executor.calls)
	}
}

func TestInvokeWithPolicyFullAccessBypassesApprovalAndSandbox(t *testing.T) {
	t.Parallel()

	settings := gatewayToolSettingsStub{
		settings: settingsdto.Settings{
			Gateway: settingsdto.GatewaySettings{ControlPlaneEnabled: true},
			Tools: map[string]any{
				"execPermissionMode": "full access",
			},
		},
	}
	tools := toolservice.NewToolService()
	tools.SetPolicy(NewPolicyPipeline(settings))
	executor := &testToolExecutor{output: `{"ok":true}`}
	tools.SetExecutor(executor)
	if _, err := tools.RegisterTool(context.Background(), tooldto.RegisterToolRequest{
		Spec: tooldto.ToolSpec{
			ID:        "high_risk_tool",
			Name:      "high_risk_tool",
			Enabled:   true,
			RiskLevel: "high",
		},
	}); err != nil {
		t.Fatalf("register high_risk_tool: %v", err)
	}

	service := NewService(tools, nil, nil, settings, nil, nil)

	response, err := service.InvokeWithPolicy(
		context.Background(),
		tooldto.ToolsInvokeRequest{
			Tool:       "high_risk_tool",
			Args:       `{"value":"ok"}`,
			SessionKey: "session-1",
		},
		tooldto.ToolPolicyContext{SessionKey: "session-1"},
	)
	if err != nil {
		t.Fatalf("invoke high_risk_tool: %v", err)
	}
	if response.Policy.Decision != "allow" {
		t.Fatalf("expected allow decision in full access mode, got %q", response.Policy.Decision)
	}
	if response.Result.ErrorMessage != "" {
		t.Fatalf("expected empty error message, got %q", response.Result.ErrorMessage)
	}
	if strings.TrimSpace(response.Result.OutputJSON) != `{"ok":true}` {
		t.Fatalf("expected output json to be returned, got %q", response.Result.OutputJSON)
	}
	if executor.calls != 1 {
		t.Fatalf("expected executor called once, got %d", executor.calls)
	}
}

func TestInvokeWithPolicyInjectsSessionKeyIntoRuntimeContext(t *testing.T) {
	t.Parallel()

	settings := gatewayToolSettingsStub{
		settings: settingsdto.Settings{
			Gateway: settingsdto.GatewaySettings{ControlPlaneEnabled: true},
			Tools:   map[string]any{},
		},
	}
	tools := toolservice.NewToolService()
	tools.SetPolicy(NewPolicyPipeline(settings))
	executor := &runtimeContextCaptureExecutor{}
	tools.SetExecutor(executor)
	if _, err := tools.RegisterTool(context.Background(), tooldto.RegisterToolRequest{
		Spec: tooldto.ToolSpec{
			ID:        "session_context_tool",
			Name:      "session_context_tool",
			Enabled:   true,
			RiskLevel: "low",
		},
	}); err != nil {
		t.Fatalf("register session_context_tool: %v", err)
	}

	service := NewService(tools, nil, nil, settings, nil, nil)
	requestSessionKey := "v2::agent::telegram::-::telegram:default:private:123::default::telegram:default:private:123"
	_, err := service.InvokeWithPolicy(
		context.Background(),
		tooldto.ToolsInvokeRequest{
			Tool:       "session_context_tool",
			Args:       `{"ok":true}`,
			SessionKey: requestSessionKey,
		},
		tooldto.ToolPolicyContext{},
	)
	if err != nil {
		t.Fatalf("invoke session_context_tool: %v", err)
	}
	if executor.calls != 1 {
		t.Fatalf("expected executor called once, got %d", executor.calls)
	}
	if executor.sessionKey != requestSessionKey {
		t.Fatalf("expected runtime context session key %q, got %q", requestSessionKey, executor.sessionKey)
	}
}
