package tools

import (
	"context"

	agentservice "dreamcreator/internal/application/agent/service"
	assistantservice "dreamcreator/internal/application/assistant/service"
	connectorsservice "dreamcreator/internal/application/connectors/service"
	externaltoolsservice "dreamcreator/internal/application/externaltools/service"
	gatewaynodes "dreamcreator/internal/application/gateway/nodes"
	gatewayvoice "dreamcreator/internal/application/gateway/voice"
	libraryservice "dreamcreator/internal/application/library/service"
	memoryservice "dreamcreator/internal/application/memory/service"
	appsession "dreamcreator/internal/application/session"
	skillsservice "dreamcreator/internal/application/skills/service"
	threadservice "dreamcreator/internal/application/thread/service"
	tooldto "dreamcreator/internal/application/tools/dto"
	toolservice "dreamcreator/internal/application/tools/service"
	"dreamcreator/internal/domain/providers"
)

type BuiltinToolDeps struct {
	Settings      SettingsReader
	Sessions      appsession.Store
	Threads       *threadservice.ThreadService
	Agents        *agentservice.AgentService
	Assistant     *assistantservice.AssistantService
	GatewayConfig GatewayConfigToolService
	Nodes         *gatewaynodes.Service
	Voice         *gatewayvoice.Service
	Library       *libraryservice.LibraryService
	Skills        *skillsservice.SkillsService
	Connectors    *connectorsservice.ConnectorsService
	ExternalTools *externaltoolsservice.ExternalToolsService
	Providers     providers.ProviderRepository
	Models        providers.ModelRepository
	Secrets       providers.SecretRepository
	Memory        *memoryservice.MemoryService
}

func RegisterBuiltinTools(ctx context.Context, toolSvc *toolservice.ToolService, executor *RegistryExecutor, deps BuiltinToolDeps) {
	if toolSvc == nil || executor == nil {
		return
	}
	registerTool(ctx, toolSvc, executor, specRead(), runReadTool)
	registerTool(ctx, toolSvc, executor, specWrite(), runWriteTool)
	registerTool(ctx, toolSvc, executor, specEdit(), runEditTool)
	registerTool(ctx, toolSvc, executor, specApplyPatch(), runApplyPatchTool)
	registerTool(ctx, toolSvc, executor, specExec(), runExecTool)
	registerTool(ctx, toolSvc, executor, specProcess(), runProcessTool)
	registerTool(ctx, toolSvc, executor, specWebFetch(), runWebFetchTool(deps.Settings, deps.Connectors))
	registerTool(ctx, toolSvc, executor, specWebSearch(), runWebSearchTool(deps.Settings, deps.Connectors))
	registerTool(ctx, toolSvc, executor, specBrowser(), runBrowserTool(deps.Settings, deps.Connectors, deps.Nodes))
	registerTool(ctx, toolSvc, executor, specCanvas(), runCanvasTool(deps.Nodes))
	registerTool(ctx, toolSvc, executor, specImage(), runImageTool(deps.Settings, deps.Assistant, deps.Providers, deps.Models, deps.Secrets))
	registerTool(ctx, toolSvc, executor, specMessage(ctx, deps.Settings), runMessageTool(deps.Settings))
	registerTool(ctx, toolSvc, executor, specGateway(), runGatewayTool(deps.Settings, deps.GatewayConfig))
	registerTool(ctx, toolSvc, executor, specCron(), runStubTool("cron"))
	registerTool(ctx, toolSvc, executor, specAgentsList(), runAgentsListTool(deps.Agents))
	registerTool(ctx, toolSvc, executor, specSessionsList(), runSessionsListTool(deps.Sessions, deps.Threads))
	registerTool(ctx, toolSvc, executor, specSessionsHistory(), runSessionsHistoryTool(deps.Threads))
	registerTool(ctx, toolSvc, executor, specSessionsSend(), runSessionsSendTool(deps.Threads))
	registerTool(ctx, toolSvc, executor, specSessionsSpawn(), runSessionsSpawnTool(nil))
	registerTool(ctx, toolSvc, executor, specSessionStatus(), runSessionStatusTool(deps.Sessions))
	registerTool(ctx, toolSvc, executor, specSubagents(), runSubagentsTool(nil))
	registerTool(ctx, toolSvc, executor, specNodes(), runNodesTool(deps.Nodes))
	registerTool(ctx, toolSvc, executor, specTTS(), runTTSTool(deps.Voice))
	registerTool(ctx, toolSvc, executor, specMemoryQuery(), runMemoryQueryTool(deps.Memory))
	registerTool(ctx, toolSvc, executor, specMemoryManage(), runMemoryManageTool(deps.Memory))
	if deps.ExternalTools != nil {
		registerTool(ctx, toolSvc, executor, specExternalToolsQuery(), runExternalToolsQueryTool(deps.ExternalTools))
		registerTool(ctx, toolSvc, executor, specExternalToolsManage(), runExternalToolsManageTool(deps.ExternalTools))
	}
	if deps.Skills != nil {
		registerTool(ctx, toolSvc, executor, specSkills(), runSkillsTool(deps.Skills, deps.Assistant, deps.Settings, deps.ExternalTools))
		registerTool(ctx, toolSvc, executor, specSkillsManage(), runSkillsManageTool(deps.Skills, deps.Assistant, deps.Settings, deps.ExternalTools))
	}
	if deps.Library != nil {
		registerTool(ctx, toolSvc, executor, specLibrary(), runLibraryGroupTool(deps.Library, "library"))
		registerTool(ctx, toolSvc, executor, specLibraryManage(), runLibraryManageTool(deps.Library))
	}
}

func registerTool(ctx context.Context, toolSvc *toolservice.ToolService, executor *RegistryExecutor, spec toolSpec, handler func(ctx context.Context, args string) (string, error)) {
	_, _ = toolSvc.RegisterTool(ctx, tooldto.RegisterToolRequest{Spec: spec.toDTO()})
	if handler != nil {
		executor.Register(spec.Name, handler)
	}
}
