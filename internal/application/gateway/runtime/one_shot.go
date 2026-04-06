package runtime

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"

	"dreamcreator/internal/application/agentruntime"
	assistantdto "dreamcreator/internal/application/assistant/dto"
	runtimedto "dreamcreator/internal/application/gateway/runtime/dto"
	settingsdto "dreamcreator/internal/application/settings/dto"
	tooldto "dreamcreator/internal/application/tools/dto"
	workspacedto "dreamcreator/internal/application/workspace/dto"
	"dreamcreator/internal/infrastructure/llm"
)

func (service *Service) RunOneShot(ctx context.Context, request runtimedto.RuntimeRunRequest) (runtimedto.RuntimeRunResult, error) {
	if service == nil {
		return runtimedto.RuntimeRunResult{}, ErrInvalidRequest
	}
	if len(request.Input.Messages) == 0 {
		return runtimedto.RuntimeRunResult{}, errors.New("runtime input messages required")
	}

	flags := resolveRunFlags(request.Metadata)
	gatewaySettings := settingsdto.GatewaySettings{}
	appLanguage := ""
	if service.settings != nil {
		if current, err := service.settings.GetSettings(ctx); err == nil {
			gatewaySettings = current.Gateway
			appLanguage = strings.TrimSpace(current.Language)
		}
	}

	assistantSnapshot, err := service.resolveOneShotAssistantSnapshot(ctx, strings.TrimSpace(request.AssistantID), appLanguage)
	if err != nil {
		return runtimedto.RuntimeRunResult{}, err
	}
	missing, err := service.checkAssistantReady(ctx, assistantSnapshot)
	if err != nil {
		return runtimedto.RuntimeRunResult{}, err
	}
	if len(missing) > 0 {
		return runtimedto.RuntimeRunResult{}, fmt.Errorf("assistant not ready: missing %s", strings.Join(missing, ","))
	}

	resolvedModel, chatModel, err := service.resolveRunModel(ctx, request.Model, assistantSnapshot.Model)
	if err != nil {
		return runtimedto.RuntimeRunResult{}, err
	}

	runCtx := llm.WithRuntimeParams(ctx, llm.RuntimeParams{
		ProviderID:       resolvedModel.ProviderID,
		ModelName:        resolvedModel.ModelName,
		ThinkingLevel:    resolveThinkingLevel(request.Thinking, request.Metadata),
		StructuredOutput: resolveStructuredOutputConfig(request.Metadata),
	})

	if service.queue != nil && flags.UseQueue {
		runLane := resolveRunLane(request.Metadata, flags.IsSubagent)
		if err := service.queue.AcquireLane(runCtx, runLane); err != nil {
			return runtimedto.RuntimeRunResult{}, err
		}
		defer service.queue.ReleaseLane(runLane)
	}

	promptMode := resolvePromptMode(request.PromptMode, "", flags.IsSubagent)
	availableToolSpecs := make([]tooldto.ToolSpec, 0)
	if service.tools != nil {
		availableToolSpecs = service.tools.ListTools(runCtx)
	}
	toolConfig := mergeToolExecutionConfig(request.Tools, assistantSnapshot.Call, flags.IsSubagent)
	toolSpecs := service.filterToolSpecs(availableToolSpecs, toolConfig, assistantSnapshot.Tools)
	skillItems := service.resolveSkillPromptItems(
		runCtx,
		resolvedModel.ProviderID,
		assistantSnapshot.Call,
		assistantSnapshot.Skills,
		promptMode,
		flags.IsSubagent,
		"",
	)
	promptDoc, _, _ := buildPromptDocument(promptBuildInput{
		Mode:              promptMode,
		RunKind:           resolveRunKind(request, flags),
		Assistant:         assistantSnapshot,
		Workspace:         workspacedto.RuntimeSnapshot{},
		Tools:             toolSpecs,
		Skills:            skillItems,
		HeartbeatPrompt:   strings.TrimSpace(gatewaySettings.Heartbeat.Prompt),
		IsSubagent:        flags.IsSubagent,
		ExtraSystemPrompt: resolveExtraSystemPrompt(request.Metadata),
		Runtime: runtimePromptInfo{
			Channel: resolveMetadataString(request.Metadata, "channel"),
			RunID:   strings.TrimSpace(request.RunID),
		},
	})

	streamFn := agentruntime.StreamFunction(chatModel.Stream)
	policyCtx := tooldto.ToolPolicyContext{
		ProviderID:      strings.TrimSpace(resolvedModel.ProviderID),
		Source:          "runtime_one_shot",
		IsSubagent:      flags.IsSubagent,
		RequireSandbox:  toolConfig.RequireSandbox,
		RequireApproval: toolConfig.RequireApproval,
	}
	toolInfos, toolAdapters := service.resolveToolAdapters(runCtx, "", strings.TrimSpace(request.RunID), toolConfig, assistantSnapshot.Tools, policyCtx)
	if len(toolInfos) > 0 {
		if toolModel, ok := resolveToolCallingModel(chatModel); ok {
			if bound, bindErr := toolModel.WithTools(toolInfos); bindErr == nil {
				streamFn = agentruntime.StreamFunction(bound.Stream)
			}
		}
	}

	controller := agentruntime.NewAgentController()
	if timeout := resolveLoopTimeout(request.Metadata, flags.IsSubagent); timeout > 0 {
		controller.SetTimeout(timeout)
	}
	maxSteps := resolveLoopMaxSteps(request.Metadata, flags.IsSubagent)
	if maxSteps <= 0 {
		if len(toolAdapters) > 0 {
			maxSteps = 4
		} else {
			maxSteps = 1
		}
	}

	var toolExecutor *agentruntime.ToolExecutor
	var toolLoopDetector *agentruntime.ToolLoopDetector
	if len(toolAdapters) > 0 {
		toolExecutor = &agentruntime.ToolExecutor{
			Validator: agentruntime.JSONToolValidator{},
			Tools:     toolAdapters,
		}
		toolLoopDetector = agentruntime.NewToolLoopDetector(resolveToolLoopDetectionConfig(request.Metadata, gatewaySettings.Runtime.ToolLoopDetection))
	}

	loop := &agentruntime.AgentLoop{
		StreamFunction: streamFn,
		ConvertToLlm: func(_ context.Context, state agentruntime.AgentState) ([]*schema.Message, error) {
			return state.Messages, nil
		},
		ToolExecutor: toolExecutor,
		Controller:   controller,
		BuildOptions: func() []model.Option {
			return service.buildChatOptions(runCtx, resolvedModel.Config, request.Metadata)
		},
		MaxSteps:         maxSteps,
		ToolLoopDetector: toolLoopDetector,
	}

	stream, err := loop.RunStream(runCtx, agentruntime.AgentState{
		Messages:     dtoMessagesToSchema(request.Input.Messages),
		SystemPrompt: strings.TrimSpace(promptDoc.Content),
		IsStreaming:  true,
	})
	if err != nil {
		return runtimedto.RuntimeRunResult{}, err
	}

	content, parts, finishReason, usage, err := consumeAgentLoopStream(stream, nil)
	if err != nil {
		return runtimedto.RuntimeRunResult{}, err
	}
	if flags.PersistUsage {
		runID := strings.TrimSpace(request.RunID)
		if runID == "" && service.newID != nil {
			runID = service.newID()
		}
		service.ingestUsage(runCtx, usage, resolvedModel, resolveMetadataString(request.Metadata, "channel"), resolveUsageSource(request.Metadata, resolveMetadataString(request.Metadata, "channel")), runID)
	}

	return runtimedto.RuntimeRunResult{
		Status: "completed",
		AssistantMessage: runtimedto.Message{
			Role:    "assistant",
			Content: content,
			Parts:   parts,
		},
		FinishReason: finishReason,
		Model: &runtimedto.ModelSelection{
			ProviderID: resolvedModel.ProviderID,
			Name:       resolvedModel.ModelName,
		},
		FinishedAt: service.now(),
		Usage:      usage,
	}, nil
}

func (service *Service) resolveOneShotAssistantSnapshot(ctx context.Context, assistantID string, appLanguage string) (assistantdto.AssistantSnapshot, error) {
	if service == nil || service.assistantSnapshots == nil {
		if strings.TrimSpace(assistantID) == "" {
			return assistantdto.AssistantSnapshot{}, errors.New("assistant id is required")
		}
		return assistantdto.AssistantSnapshot{AssistantID: strings.TrimSpace(assistantID)}, nil
	}

	var (
		snapshot assistantdto.AssistantSnapshot
		err      error
	)
	if strings.TrimSpace(assistantID) == "" {
		snapshot, err = service.assistantSnapshots.ResolveDefaultAssistantSnapshot(ctx)
	} else {
		snapshot, err = service.assistantSnapshots.ResolveAssistantSnapshot(ctx, assistantdto.ResolveAssistantSnapshotRequest{
			AssistantID: strings.TrimSpace(assistantID),
		})
	}
	if err != nil {
		return assistantdto.AssistantSnapshot{}, err
	}
	snapshot.User = service.resolveRuntimeUserLocale(snapshot.AssistantID, snapshot.User, appLanguage)
	return snapshot, nil
}
