package tools

import (
	"context"

	tooldto "dreamcreator/internal/application/tools/dto"
)

type ToolRequirementResolver interface {
	ResolveToolRequirements(ctx context.Context, spec tooldto.ToolSpec) []tooldto.ToolRequirement
}

type ToolRequirementResolverFunc func(ctx context.Context, spec tooldto.ToolSpec) []tooldto.ToolRequirement

func (fn ToolRequirementResolverFunc) ResolveToolRequirements(ctx context.Context, spec tooldto.ToolSpec) []tooldto.ToolRequirement {
	if fn == nil {
		return nil
	}
	return fn(ctx, spec)
}

func resolveEffectiveToolSpecWithResolver(
	ctx context.Context,
	spec tooldto.ToolSpec,
	snapshot toolRequirementSnapshot,
	resolver ToolRequirementResolver,
) tooldto.ToolSpec {
	effective := resolveEffectiveToolSpec(spec, snapshot)
	if resolver == nil {
		return effective
	}
	additional := resolver.ResolveToolRequirements(ctx, effective)
	if len(additional) == 0 {
		return effective
	}
	effective.Requirements = append(effective.Requirements, additional...)
	if effective.Enabled && !toolRequirementsSatisfied(effective.Requirements) {
		effective.Enabled = false
	}
	return effective
}
