package service

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"dreamcreator/internal/application/tools/dto"
)

var errToolNotFound = errors.New("tool not found")

type ToolExecutor interface {
	Execute(ctx context.Context, spec dto.ToolSpec, args string) (string, error)
}

type ToolPolicyPipeline interface {
	Decide(ctx context.Context, spec dto.ToolSpec, policyCtx dto.ToolPolicyContext) (dto.ToolPolicyDecision, error)
}

type DefaultPolicyPipeline struct{}

func (pipeline *DefaultPolicyPipeline) Decide(_ context.Context, spec dto.ToolSpec, _ dto.ToolPolicyContext) (dto.ToolPolicyDecision, error) {
	if !spec.Enabled {
		return dto.ToolPolicyDecision{Decision: "deny", Reason: "tool disabled"}, nil
	}
	decision := dto.ToolPolicyDecision{Decision: "allow"}
	if spec.RequiresApproval {
		decision.Decision = "ask"
		decision.ApprovalRequired = true
	}
	if spec.RequiresSandbox || strings.EqualFold(spec.RiskLevel, "high") {
		decision.SandboxRequired = true
	}
	return decision, nil
}

type ToolService struct {
	mu       sync.RWMutex
	tools    map[string]dto.ToolSpec
	executor ToolExecutor
	policy   ToolPolicyPipeline
	now      func() time.Time
}

func NewToolService() *ToolService {
	return &ToolService{
		tools:  make(map[string]dto.ToolSpec),
		policy: &DefaultPolicyPipeline{},
		now:    time.Now,
	}
}

func (service *ToolService) SetExecutor(executor ToolExecutor) {
	if service == nil {
		return
	}
	service.executor = executor
}

func (service *ToolService) SetPolicy(policy ToolPolicyPipeline) {
	if service == nil || policy == nil {
		return
	}
	service.policy = policy
}

func (service *ToolService) RegisterTool(_ context.Context, request dto.RegisterToolRequest) (dto.ToolSpec, error) {
	spec := normalizeToolSpec(request.Spec)
	if spec.Name == "" {
		return dto.ToolSpec{}, errors.New("tool name is required")
	}
	if spec.ID == "" {
		spec.ID = spec.Name
	}
	service.mu.Lock()
	service.tools[spec.ID] = spec
	service.tools[spec.Name] = spec
	service.mu.Unlock()
	return spec, nil
}

func (service *ToolService) EnableTool(_ context.Context, request dto.EnableToolRequest) error {
	id := strings.TrimSpace(request.ID)
	if id == "" {
		return errToolNotFound
	}
	service.mu.Lock()
	defer service.mu.Unlock()
	spec, ok := service.tools[id]
	if !ok {
		return errToolNotFound
	}
	spec.Enabled = request.Enabled
	service.tools[id] = spec
	service.tools[spec.Name] = spec
	return nil
}

func (service *ToolService) ExecuteTool(ctx context.Context, request dto.ExecuteToolRequest) (dto.ToolResult, error) {
	spec, err := service.resolveSpec(request.Invocation.ToolID, request.Invocation.ToolName)
	if err != nil {
		return dto.ToolResult{}, err
	}
	policy := service.policyDecision(ctx, spec, dto.ToolPolicyContext{})
	return service.executeWithDecision(ctx, spec, policy, request.Invocation)
}

func (service *ToolService) ExecuteToolWithDecision(ctx context.Context, spec dto.ToolSpec, decision dto.ToolPolicyDecision, invocation dto.ToolInvocation) (dto.ToolResult, error) {
	if service == nil {
		return dto.ToolResult{}, errors.New("tool service unavailable")
	}
	return service.executeWithDecision(ctx, spec, decision, invocation)
}

func (service *ToolService) Decide(ctx context.Context, toolID string, toolName string, policyCtx dto.ToolPolicyContext) (dto.ToolSpec, dto.ToolPolicyDecision, error) {
	spec, err := service.resolveSpec(toolID, toolName)
	if err != nil {
		return dto.ToolSpec{}, dto.ToolPolicyDecision{}, err
	}
	decision := service.policyDecision(ctx, spec, policyCtx)
	return spec, decision, nil
}

func (service *ToolService) QueryToolLogs(_ context.Context, _ dto.QueryToolLogsRequest) ([]dto.ToolInvocation, error) {
	return []dto.ToolInvocation{}, nil
}

func (service *ToolService) ListTools() []dto.ToolSpec {
	if service == nil {
		return nil
	}
	service.mu.RLock()
	defer service.mu.RUnlock()
	seen := make(map[string]struct{}, len(service.tools))
	result := make([]dto.ToolSpec, 0, len(service.tools))
	for _, spec := range service.tools {
		key := strings.TrimSpace(spec.ID)
		if key == "" {
			key = strings.TrimSpace(spec.Name)
		}
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, spec)
	}
	return result
}

func (service *ToolService) resolveSpec(id string, name string) (dto.ToolSpec, error) {
	key := strings.TrimSpace(id)
	if key == "" {
		key = strings.TrimSpace(name)
	}
	if key == "" {
		return dto.ToolSpec{}, errToolNotFound
	}
	service.mu.RLock()
	spec, ok := service.tools[key]
	service.mu.RUnlock()
	if !ok {
		return dto.ToolSpec{}, errToolNotFound
	}
	return spec, nil
}

func (service *ToolService) policyDecision(ctx context.Context, spec dto.ToolSpec, policyCtx dto.ToolPolicyContext) dto.ToolPolicyDecision {
	if service.policy == nil {
		return dto.ToolPolicyDecision{Decision: "allow"}
	}
	decision, err := service.policy.Decide(ctx, spec, policyCtx)
	if err != nil {
		return dto.ToolPolicyDecision{Decision: "deny", Reason: err.Error()}
	}
	return decision
}

func (service *ToolService) executeWithDecision(ctx context.Context, spec dto.ToolSpec, policy dto.ToolPolicyDecision, invocation dto.ToolInvocation) (dto.ToolResult, error) {
	if policy.Decision == "deny" {
		return dto.ToolResult{ID: invocation.ID, ToolID: spec.ID, ErrorMessage: policy.Reason, Policy: policy}, errors.New(policy.Reason)
	}
	if policy.Decision == "ask" {
		return dto.ToolResult{ID: invocation.ID, ToolID: spec.ID, ErrorMessage: "approval required", Policy: policy}, errors.New("approval required")
	}
	output := "null"
	if service.executor != nil {
		if result, execErr := service.executor.Execute(ctx, spec, invocation.InputJSON); execErr != nil {
			return dto.ToolResult{ID: invocation.ID, ToolID: spec.ID, ErrorMessage: execErr.Error(), Policy: policy}, execErr
		} else if strings.TrimSpace(result) != "" {
			output = result
		}
	}
	return dto.ToolResult{
		ID:         invocation.ID,
		ToolID:     spec.ID,
		OutputJSON: output,
		Policy:     policy,
	}, nil
}

func normalizeToolSpec(spec dto.ToolSpec) dto.ToolSpec {
	spec.ID = strings.TrimSpace(spec.ID)
	spec.Name = strings.TrimSpace(spec.Name)
	spec.Description = strings.TrimSpace(spec.Description)
	spec.PromptSnippet = strings.TrimSpace(spec.PromptSnippet)
	spec.Kind = strings.TrimSpace(spec.Kind)
	spec.SchemaJSON = strings.TrimSpace(spec.SchemaJSON)
	spec.SideEffectLevel = strings.TrimSpace(spec.SideEffectLevel)
	spec.Category = strings.TrimSpace(spec.Category)
	spec.RiskLevel = strings.TrimSpace(spec.RiskLevel)
	return spec
}
