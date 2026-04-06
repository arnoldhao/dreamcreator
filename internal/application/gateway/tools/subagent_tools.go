package tools

import (
	"context"

	gatewaysubagent "dreamcreator/internal/application/gateway/subagent"
	toolservice "dreamcreator/internal/application/tools/service"
)

type SubagentToolDeps struct {
	Gateway *gatewaysubagent.GatewayService
}

func RegisterSubagentTools(ctx context.Context, toolSvc *toolservice.ToolService, executor *RegistryExecutor, deps SubagentToolDeps) {
	if toolSvc == nil || executor == nil {
		return
	}
	registerTool(ctx, toolSvc, executor, specSessionsSpawn(), runSessionsSpawnTool(deps.Gateway))
	registerTool(ctx, toolSvc, executor, specSubagents(), runSubagentsTool(deps.Gateway))
}
