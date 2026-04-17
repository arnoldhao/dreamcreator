package runtime

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/schema"

	assistantdto "dreamcreator/internal/application/assistant/dto"
	"dreamcreator/internal/application/chatevent"
	runtimedto "dreamcreator/internal/application/gateway/runtime/dto"
	"dreamcreator/internal/application/runtimeconfig"
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
	if ctx == nil {
		ctx = context.Background()
	}
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		timeoutCtx, cancel := context.WithTimeout(ctx, runtimeconfig.DefaultAuxiliaryLLMTimeout)
		defer cancel()
		ctx = timeoutCtx
	}

	flags := resolveRunFlags(request.Metadata)
	runKind := resolveRunKind(request, flags)
	if !isOneShotRunKind(runKind) {
		runKind = runKindOneShot
	}
	appLanguage := ""
	if service.settings != nil {
		if current, err := service.settings.GetSettings(ctx); err == nil {
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
		SessionID:        strings.TrimSpace(request.SessionID),
		ThreadID:         strings.TrimSpace(request.SessionID),
		RunID:            strings.TrimSpace(request.RunID),
		RequestSource:    resolveUsageSource(request.Metadata, resolveMetadataString(request.Metadata, "channel"), runKind),
		Operation:        resolveLLMOperation(runKind, request.Metadata, flags.IsSubagent),
		ProviderID:       resolvedModel.ProviderID,
		ModelName:        resolvedModel.ModelName,
		ThinkingLevel:    resolveOneShotThinkingLevel(request.Thinking, request.Metadata),
		StructuredOutput: resolveStructuredOutputConfig(request.Metadata),
	})

	if service.queue != nil && flags.UseQueue {
		runLane := resolveRunLane(request.Metadata, flags.IsSubagent)
		if err := service.queue.AcquireLane(runCtx, runLane); err != nil {
			return runtimedto.RuntimeRunResult{}, err
		}
		defer service.queue.ReleaseLane(runLane)
	}

	promptMode := resolveOneShotPromptMode(request.PromptMode)
	promptDoc, _, _ := buildPromptDocument(promptBuildInput{
		Mode:              promptMode,
		RunKind:           runKind,
		Assistant:         assistantSnapshot,
		Workspace:         workspacedto.RuntimeSnapshot{},
		Tools:             nil,
		Skills:            nil,
		HeartbeatPrompt:   "",
		IsSubagent:        false,
		ExtraSystemPrompt: resolveExtraSystemPrompt(request.Metadata),
		Runtime: runtimePromptInfo{
			Channel: resolveMetadataString(request.Metadata, "channel"),
			RunID:   strings.TrimSpace(request.RunID),
		},
	})
	messages := dtoMessagesToSchema(request.Input.Messages)
	if systemPrompt := strings.TrimSpace(promptDoc.Content); systemPrompt != "" {
		messages = append([]*schema.Message{{
			Role:    schema.System,
			Content: systemPrompt,
		}}, messages...)
	}
	response, err := chatModel.Generate(runCtx, messages, service.buildChatOptions(runCtx, resolvedModel.Config, request.Metadata)...)
	if err != nil {
		return runtimedto.RuntimeRunResult{}, err
	}
	content, parts, finishReason, usage := buildOneShotResult(response)
	if flags.PersistUsage {
		runID := strings.TrimSpace(request.RunID)
		if runID == "" && service.newID != nil {
			runID = service.newID()
		}
		service.ingestUsage(runCtx, usage, resolvedModel, resolveMetadataString(request.Metadata, "channel"), resolveUsageSource(request.Metadata, resolveMetadataString(request.Metadata, "channel"), runKind), runID)
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

func buildOneShotResult(message *schema.Message) (string, []chatevent.MessagePart, string, runtimedto.RuntimeUsage) {
	if message == nil {
		return "", nil, "", runtimedto.RuntimeUsage{}
	}
	content := strings.TrimSpace(message.Content)
	parts := make([]chatevent.MessagePart, 0, 2)
	if content != "" {
		parts = append(parts, chatevent.MessagePart{Type: "text", Text: content})
	}
	if reasoning := strings.TrimSpace(message.ReasoningContent); reasoning != "" {
		parts = append(parts, chatevent.MessagePart{Type: "reasoning", Text: reasoning})
	}
	finishReason := ""
	usage := runtimedto.RuntimeUsage{}
	if message.ResponseMeta != nil {
		finishReason = strings.TrimSpace(message.ResponseMeta.FinishReason)
		usage = mergeRuntimeUsage(usage, message.ResponseMeta.Usage)
	}
	return content, parts, finishReason, usage
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
