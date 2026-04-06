package tools

import (
	"context"
	"strings"

	tooldto "dreamcreator/internal/application/tools/dto"
)

type PolicyPipeline struct {
	settings SettingsReader
}

func NewPolicyPipeline(settings SettingsReader) *PolicyPipeline {
	return &PolicyPipeline{settings: settings}
}

func (pipeline *PolicyPipeline) Decide(ctx context.Context, spec tooldto.ToolSpec, policyCtx tooldto.ToolPolicyContext) (tooldto.ToolPolicyDecision, error) {
	if !spec.Enabled {
		return tooldto.ToolPolicyDecision{
			Decision:     "deny",
			Reason:       "tool disabled",
			MatchedRules: []string{"disabled"},
		}, nil
	}
	snapshot := loadToolRequirementSnapshot(ctx, pipeline.settings)
	effectiveSpec := resolveEffectiveToolSpec(spec, snapshot)
	if !effectiveSpec.Enabled {
		reason := "tool requirements unavailable"
		if requirement, ok := firstUnavailableToolRequirement(effectiveSpec.Requirements); ok {
			if trimmed := strings.TrimSpace(requirement.Reason); trimmed != "" {
				reason = trimmed
			}
		}
		return tooldto.ToolPolicyDecision{
			Decision:     "deny",
			Reason:       reason,
			MatchedRules: []string{"requirements_unavailable"},
		}, nil
	}
	decision := tooldto.ToolPolicyDecision{Decision: "allow"}
	if strings.EqualFold(spec.RiskLevel, "high") {
		decision.MatchedRules = append(decision.MatchedRules, "risk_high")
		decision.SandboxRequired = true
		decision.ApprovalRequired = true
	}
	if spec.RequiresSandbox {
		decision.MatchedRules = append(decision.MatchedRules, "requires_sandbox")
		decision.SandboxRequired = true
	}
	if spec.RequiresApproval {
		decision.MatchedRules = append(decision.MatchedRules, "requires_approval")
		decision.ApprovalRequired = true
	}
	if policyCtx.RequireSandbox {
		decision.MatchedRules = append(decision.MatchedRules, "require_sandbox")
		decision.SandboxRequired = true
	}
	if policyCtx.RequireApproval {
		decision.MatchedRules = append(decision.MatchedRules, "require_approval")
		decision.ApprovalRequired = true
	}
	if snapshot.execPermissionMode == ExecPermissionModeAllAccess {
		decision.MatchedRules = append(decision.MatchedRules, "exec_permission_all_access")
		decision.SandboxRequired = false
		decision.ApprovalRequired = false
	}
	if decision.ApprovalRequired {
		decision.Decision = "ask"
	}
	return decision, nil
}

func (pipeline *PolicyPipeline) policyEnabled(ctx context.Context) bool {
	return true
}
