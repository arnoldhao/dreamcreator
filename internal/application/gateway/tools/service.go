package tools

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	gatewayapprovals "dreamcreator/internal/application/gateway/approvals"
	gatewayevents "dreamcreator/internal/application/gateway/events"
	gatewaysandbox "dreamcreator/internal/application/gateway/sandbox"
	tooldto "dreamcreator/internal/application/tools/dto"
	toolservice "dreamcreator/internal/application/tools/service"
)

type Service struct {
	tools                *toolservice.ToolService
	approvals            *gatewayapprovals.Service
	sandbox              *gatewaysandbox.Service
	settings             SettingsReader
	audit                PolicyAuditStore
	events               *gatewayevents.Broker
	requirementsResolver ToolRequirementResolver
	now                  func() time.Time
	newID                func() string
}

type PolicyAuditStore interface {
	Save(ctx context.Context, toolID string, decision string, reason string, context any) error
}

func NewService(tools *toolservice.ToolService, approvals *gatewayapprovals.Service, sandbox *gatewaysandbox.Service, settings SettingsReader, audit PolicyAuditStore, events *gatewayevents.Broker) *Service {
	return &Service{
		tools:     tools,
		approvals: approvals,
		sandbox:   sandbox,
		settings:  settings,
		audit:     audit,
		events:    events,
		now:       time.Now,
		newID:     uuid.NewString,
	}
}

func (service *Service) SetRequirementsResolver(resolver ToolRequirementResolver) {
	if service == nil {
		return
	}
	service.requirementsResolver = resolver
}

func (service *Service) ListTools(ctx context.Context) []tooldto.ToolSpec {
	if service == nil || service.tools == nil {
		return nil
	}
	specs := service.tools.ListTools()
	if len(specs) == 0 {
		return nil
	}
	snapshot := loadToolRequirementSnapshot(ctx, service.settings)
	result := make([]tooldto.ToolSpec, 0, len(specs))
	for _, spec := range specs {
		resolved := resolveEffectiveToolSpecWithResolver(ctx, spec, snapshot, service.requirementsResolver)
		resolved = resolveDynamicToolSpec(ctx, resolved, service.settings)
		result = append(result, resolved)
	}
	return result
}

func (service *Service) Invoke(ctx context.Context, request tooldto.ToolsInvokeRequest) (tooldto.ToolsInvokeResponse, error) {
	policyCtx := tooldto.ToolPolicyContext{
		SessionKey: strings.TrimSpace(request.SessionKey),
		Source:     "http",
	}
	return service.InvokeWithPolicy(ctx, request, policyCtx)
}

func (service *Service) InvokeWithPolicy(ctx context.Context, request tooldto.ToolsInvokeRequest, policyCtx tooldto.ToolPolicyContext) (tooldto.ToolsInvokeResponse, error) {
	if service == nil || service.tools == nil {
		return tooldto.ToolsInvokeResponse{}, errors.New("tool service unavailable")
	}
	toolName := strings.TrimSpace(request.Tool)
	if toolName == "" {
		return tooldto.ToolsInvokeResponse{}, errors.New("tool name is required")
	}
	runtimeSessionKey, runtimeRunID := RuntimeContextFromContext(ctx)
	runtimeSessionKey = strings.TrimSpace(runtimeSessionKey)
	requestSessionKey := strings.TrimSpace(request.SessionKey)
	if runtimeSessionKey == "" && requestSessionKey != "" {
		ctx = WithRuntimeContext(ctx, requestSessionKey, strings.TrimSpace(runtimeRunID))
		runtimeSessionKey = requestSessionKey
	}
	if requestSessionKey == "" && runtimeSessionKey != "" {
		request.SessionKey = runtimeSessionKey
		requestSessionKey = runtimeSessionKey
	}
	if policyCtx.SessionKey == "" {
		policyCtx.SessionKey = requestSessionKey
	}
	if policyCtx.Source == "" {
		policyCtx.Source = "runtime"
	}
	spec, decision, err := service.tools.Decide(ctx, "", toolName, policyCtx)
	if err != nil {
		return tooldto.ToolsInvokeResponse{}, err
	}
	spec = resolveEffectiveToolSpecWithResolver(ctx, spec, loadToolRequirementSnapshot(ctx, service.settings), service.requirementsResolver)
	spec = resolveDynamicToolSpec(ctx, spec, service.settings)
	service.auditDecision(ctx, spec, decision, policyCtx)
	response := tooldto.ToolsInvokeResponse{
		Policy: decision,
		PolicyDetail: tooldto.ToolPolicySnapshot{
			Spec:     spec,
			Decision: decision,
			Context:  policyCtx,
		},
	}
	if decision.Decision == "deny" {
		return response, errors.New(decision.Reason)
	}
	if request.DryRun {
		return response, nil
	}
	if decision.ApprovalRequired {
		if service.approvals == nil {
			return response, errors.New("approval required")
		}
		approval, err := service.approvals.Create(ctx, gatewayapprovals.Request{
			SessionKey: strings.TrimSpace(request.SessionKey),
			ToolCallID: strings.TrimSpace(request.ToolCallID),
			ToolName:   toolName,
			Action:     strings.TrimSpace(request.Action),
			Args:       strings.TrimSpace(request.Args),
		})
		if err != nil {
			return response, err
		}
		resolved, err := service.approvals.Wait(ctx, gatewayapprovals.WaitRequest{ID: approval.ID})
		if err != nil {
			return response, err
		}
		if resolved.Status != gatewayapprovals.StatusApproved {
			return response, errors.New("approval denied")
		}
		// Approval has been granted; execution should continue with an allow decision.
		decision.Decision = "allow"
	}
	if decision.SandboxRequired {
		if service.sandbox == nil {
			return response, errors.New("sandbox_unavailable")
		}
		resolution, err := service.sandbox.Resolve(ctx, gatewaysandbox.ResolveRequest{RequiresSandbox: true})
		if err != nil {
			return response, err
		}
		if !resolution.Allowed {
			reason := strings.TrimSpace(resolution.Reason)
			if reason == "" {
				reason = "sandbox_unavailable"
			}
			return response, errors.New(reason)
		}
	}
	invocation := tooldto.ToolInvocation{
		ID:        service.newID(),
		ToolID:    spec.ID,
		ToolName:  spec.Name,
		InputJSON: request.Args,
	}
	response.Policy = decision
	response.PolicyDetail.Decision = decision
	result, execErr := service.tools.ExecuteToolWithDecision(ctx, spec, decision, invocation)
	response.Result = result
	service.publishAuditEvent(ctx, spec, decision, policyCtx, request, result, execErr)
	if execErr != nil {
		return response, execErr
	}
	return response, nil
}

func (service *Service) CleanupRuntimeSession(_ context.Context, sessionKey string) {
	if strings.TrimSpace(sessionKey) == "" {
		return
	}
	cleanupBrowserToolSessions(sessionKey)
}

func (service *Service) auditDecision(ctx context.Context, spec tooldto.ToolSpec, decision tooldto.ToolPolicyDecision, policyCtx tooldto.ToolPolicyContext) {
	if service == nil || service.audit == nil {
		return
	}
	_ = service.audit.Save(ctx, spec.ID, decision.Decision, decision.Reason, map[string]any{
		"spec":     spec,
		"decision": decision,
		"context":  policyCtx,
	})
}

func (service *Service) publishAuditEvent(ctx context.Context, spec tooldto.ToolSpec, decision tooldto.ToolPolicyDecision, policyCtx tooldto.ToolPolicyContext, request tooldto.ToolsInvokeRequest, result tooldto.ToolResult, execErr error) {
	if service == nil || service.events == nil {
		return
	}
	payload := map[string]any{
		"tool":     spec.Name,
		"toolId":   spec.ID,
		"decision": decision,
		"context":  policyCtx,
		"request": map[string]any{
			"action":     strings.TrimSpace(request.Action),
			"sessionKey": strings.TrimSpace(request.SessionKey),
		},
		"result": result,
	}
	if execErr != nil {
		payload["error"] = execErr.Error()
	}
	envelope := gatewayevents.Envelope{
		Type:       "tool.invoke.audit",
		Topic:      "tool",
		SessionKey: strings.TrimSpace(request.SessionKey),
		Timestamp:  service.now(),
	}
	_, _ = service.events.Publish(ctx, envelope, payload)
}

func resolveDynamicToolSpec(ctx context.Context, spec tooldto.ToolSpec, settings SettingsReader) tooldto.ToolSpec {
	key := resolveToolRequirementKey(spec)
	switch key {
	case "message":
		profile := resolveMessageToolSchemaProfile(ctx, settings)
		spec.SchemaJSON = resolveMessageToolSchemaJSON(specMessageBase().SchemaJSON, profile)
	}
	return spec
}
