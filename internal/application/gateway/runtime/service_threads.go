package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"go.uber.org/zap"

	"dreamcreator/internal/application/agentruntime"
	"dreamcreator/internal/application/chatevent"
	appevents "dreamcreator/internal/application/events"
	"dreamcreator/internal/application/gateway/runtime/dto"
	gatewayusage "dreamcreator/internal/application/gateway/usage"
	"dreamcreator/internal/application/runtimeconfig"
	appsession "dreamcreator/internal/application/session"
	threaddto "dreamcreator/internal/application/thread/dto"
	"dreamcreator/internal/domain/thread"
)

const threadTitleGenerationTimeout = runtimeconfig.DefaultAuxiliaryLLMTimeout

func (service *Service) startRun(ctx context.Context, threadID string, agentID string, requestedRunID string, persist bool) (thread.ThreadRun, error) {
	if persist && service.runs == nil {
		return thread.ThreadRun{}, errors.New("run repository unavailable")
	}
	now := service.now()
	runID := strings.TrimSpace(requestedRunID)
	if runID == "" {
		runID = service.newID()
	}
	run, err := thread.NewThreadRun(thread.ThreadRunParams{
		ID:                 runID,
		ThreadID:           threadID,
		AssistantMessageID: service.newID(),
		AgentID:            strings.TrimSpace(agentID),
		Status:             thread.RunStatusActive,
		CreatedAt:          &now,
		UpdatedAt:          &now,
	})
	if err != nil {
		return thread.ThreadRun{}, err
	}
	if persist && service.runs != nil {
		if err := service.runs.Save(ctx, run); err != nil {
			return thread.ThreadRun{}, err
		}
	}
	return run, nil
}

func (service *Service) failRun(ctx context.Context, run thread.ThreadRun, err error) error {
	if service.runs == nil {
		return errors.New("run repository unavailable")
	}
	_ = err
	run.Status = thread.RunStatusError
	run.UpdatedAt = service.now()
	return service.runs.Save(ctx, run)
}

func (service *Service) emitRuntimeEvent(ctx context.Context, run thread.ThreadRun, sessionKey string, event agentruntime.Event) {
	event.RunID = strings.TrimSpace(run.ID)
	event.ThreadID = strings.TrimSpace(run.ThreadID)
	event.MessageID = strings.TrimSpace(run.AssistantMessageID)
	encoded, err := agentruntime.EncodeChatEvent(event)
	if err != nil {
		return
	}
	service.appendRunEvent(ctx, run, sessionKey, encoded)
}

func (service *Service) appendRunEvent(ctx context.Context, run thread.ThreadRun, sessionKey string, eventPayload chatevent.Event) {
	if service.runEvents == nil {
		return
	}
	data, err := json.Marshal(eventPayload)
	if err != nil {
		return
	}
	now := service.now()
	event, err := thread.NewThreadRunEvent(thread.ThreadRunEventParams{
		RunID:       run.ID,
		ThreadID:    run.ThreadID,
		EventType:   eventPayload.Type,
		PayloadJSON: string(data),
		CreatedAt:   &now,
	})
	if err != nil {
		return
	}
	stored, err := service.runEvents.Append(ctx, event)
	if err != nil {
		return
	}
	_ = sessionKey
	if service.events != nil {
		service.events.Publish(ctx, stored, sessionKey)
	}
}

func (service *Service) emitThreadUpdated(ctx context.Context, threadID string, change string, reason string) {
	if service == nil || service.eventBus == nil {
		return
	}
	threadID = strings.TrimSpace(threadID)
	if threadID == "" {
		return
	}
	payload := map[string]any{
		"threadId": threadID,
	}
	if strings.TrimSpace(change) != "" {
		payload["change"] = change
	}
	if strings.TrimSpace(reason) != "" {
		payload["reason"] = reason
	}
	if strings.EqualFold(change, "upsert") && service.threads != nil {
		if item, err := service.threads.Get(ctx, threadID); err == nil {
			snapshot := map[string]any{
				"id":             strings.TrimSpace(item.ID),
				"assistantId":    strings.TrimSpace(item.AssistantID),
				"title":          strings.TrimSpace(item.Title),
				"titleIsDefault": item.TitleIsDefault,
				"status":         strings.TrimSpace(string(item.Status)),
				"createdAt":      formatThreadEventTime(item.CreatedAt),
				"updatedAt":      formatThreadEventTime(item.UpdatedAt),
			}
			if changedBy := strings.TrimSpace(string(item.TitleChangedBy)); changedBy != "" {
				snapshot["titleChangedBy"] = changedBy
			}
			if item.DeletedAt != nil {
				snapshot["deletedAt"] = formatThreadEventTime(*item.DeletedAt)
			}
			if item.PurgeAfter != nil {
				snapshot["purgeAfter"] = formatThreadEventTime(*item.PurgeAfter)
			}
			payload["thread"] = snapshot
			if updatedAt, ok := snapshot["updatedAt"].(string); ok && strings.TrimSpace(updatedAt) != "" {
				payload["threadVersion"] = updatedAt
			}
		}
	}
	_ = service.eventBus.Publish(ctx, appevents.Event{
		Topic:   "chat.thread.updated",
		Type:    "thread-updated",
		Payload: payload,
	})
}

func formatThreadEventTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}

func (service *Service) scheduleThreadTitleGenerationAtRequestStart(item thread.Thread, messages []dto.Message) {
	if service == nil || service.threadTitles == nil {
		return
	}
	requestMessages := toThreadTitleRequestMessages(messages)
	if len(requestMessages) == 0 {
		return
	}
	service.scheduleThreadTitleGeneration(item, requestMessages)
}

func (service *Service) scheduleThreadTitleGeneration(item thread.Thread, requestMessages []threaddto.GenerateThreadTitleMessage) {
	threadID := strings.TrimSpace(item.ID)
	if threadID == "" || !item.TitleIsDefault || strings.EqualFold(strings.TrimSpace(string(item.TitleChangedBy)), string(thread.ThreadTitleChangedByUser)) {
		return
	}
	if !service.markThreadTitleGenerationInFlight(threadID) {
		return
	}
	go func(threadID string, requestMessages []threaddto.GenerateThreadTitleMessage) {
		defer service.unmarkThreadTitleGenerationInFlight(threadID)
		titleCtx, cancel := context.WithTimeout(context.Background(), threadTitleGenerationTimeout)
		defer cancel()
		response, err := service.threadTitles.GenerateThreadTitle(titleCtx, threaddto.GenerateThreadTitleRequest{
			ThreadID: threadID,
			Messages: requestMessages,
		})
		if err != nil {
			zap.L().Warn("thread title generation failed",
				zap.String("threadID", threadID),
				zap.Error(err),
			)
			return
		}
		if response.Updated {
			service.emitThreadUpdated(titleCtx, threadID, "upsert", "auto-generate-title")
		}
	}(threadID, requestMessages)
}

func shouldScheduleThreadTitleGenerationAtRequestStart(item thread.Thread, messages []dto.Message) bool {
	threadID := strings.TrimSpace(item.ID)
	if threadID == "" || !item.TitleIsDefault || strings.EqualFold(strings.TrimSpace(string(item.TitleChangedBy)), string(thread.ThreadTitleChangedByUser)) {
		return false
	}
	return len(toThreadTitleRequestMessages(messages)) > 0
}

func (service *Service) scheduleThreadTitleGenerationAfterRun(threadID string) {
	if service == nil || service.threadTitles == nil || service.threads == nil {
		return
	}
	threadID = strings.TrimSpace(threadID)
	if threadID == "" {
		return
	}
	go func(threadID string) {
		waitCtx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()
		if !service.waitForThreadTitleGenerationIdle(waitCtx, threadID) {
			return
		}
		item, err := service.threads.Get(waitCtx, threadID)
		if err != nil {
			return
		}
		service.scheduleThreadTitleGeneration(item, nil)
	}(threadID)
}

func toThreadTitleRequestMessages(messages []dto.Message) []threaddto.GenerateThreadTitleMessage {
	if len(messages) == 0 {
		return nil
	}
	result := make([]threaddto.GenerateThreadTitleMessage, 0, len(messages))
	for _, message := range messages {
		role := strings.ToLower(strings.TrimSpace(message.Role))
		if role != "user" && role != "assistant" {
			continue
		}
		content := strings.TrimSpace(message.Content)
		if content == "" && len(message.Parts) == 0 {
			continue
		}
		result = append(result, threaddto.GenerateThreadTitleMessage{
			Role:    role,
			Content: content,
			Parts:   message.Parts,
		})
	}
	if len(result) == 0 {
		return nil
	}
	const maxMessages = 12
	if len(result) > maxMessages {
		return append([]threaddto.GenerateThreadTitleMessage(nil), result[len(result)-maxMessages:]...)
	}
	return result
}

func (service *Service) markThreadTitleGenerationInFlight(threadID string) bool {
	service.titleGenerationMu.Lock()
	defer service.titleGenerationMu.Unlock()
	if service.titleGenerationInFlight == nil {
		service.titleGenerationInFlight = make(map[string]struct{})
	}
	if _, exists := service.titleGenerationInFlight[threadID]; exists {
		return false
	}
	service.titleGenerationInFlight[threadID] = struct{}{}
	return true
}

func (service *Service) unmarkThreadTitleGenerationInFlight(threadID string) {
	service.titleGenerationMu.Lock()
	defer service.titleGenerationMu.Unlock()
	delete(service.titleGenerationInFlight, threadID)
}

func (service *Service) isThreadTitleGenerationInFlight(threadID string) bool {
	service.titleGenerationMu.Lock()
	defer service.titleGenerationMu.Unlock()
	_, exists := service.titleGenerationInFlight[threadID]
	return exists
}

func (service *Service) waitForThreadTitleGenerationIdle(ctx context.Context, threadID string) bool {
	if !service.isThreadTitleGenerationInFlight(threadID) {
		return true
	}
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return false
		case <-ticker.C:
			if !service.isThreadTitleGenerationInFlight(threadID) {
				return true
			}
		}
	}
}

func (service *Service) persistSessionContextSnapshot(ctx context.Context, sessionID string, usage dto.RuntimeUsage) {
	if service == nil || service.sessions == nil {
		return
	}
	if usage.ContextPromptTokens <= 0 && usage.ContextTotalTokens <= 0 && usage.ContextWindowTokens <= 0 {
		return
	}
	_ = service.sessions.UpdateContextSnapshot(ctx, sessionID, appsession.ContextSnapshotUpdate{
		PromptTokens: usage.ContextPromptTokens,
		TotalTokens:  usage.ContextTotalTokens,
		WindowTokens: usage.ContextWindowTokens,
		UpdatedAt:    service.now(),
		Fresh:        true,
	})
}

func (service *Service) ingestUsage(ctx context.Context, usage dto.RuntimeUsage, model resolvedRunModel, channel string, source string, runID string) {
	if service == nil || service.usage == nil {
		return
	}
	units := usage.TotalTokens
	if units <= 0 && (usage.PromptTokens > 0 || usage.CompletionTokens > 0) {
		units = usage.PromptTokens + usage.CompletionTokens
	}
	_ = service.usage.Ingest(ctx, gatewayusage.LedgerEntry{
		Category:      gatewayusage.CategoryTokens,
		ProviderID:    strings.TrimSpace(model.ProviderID),
		ModelName:     strings.TrimSpace(model.ModelName),
		Channel:       strings.TrimSpace(channel),
		RequestID:     strings.TrimSpace(runID),
		RequestSource: normalizeUsageSource(source),
		Units:         units,
		InputTokens:   usage.PromptTokens,
		OutputTokens:  usage.CompletionTokens,
		CostBasis:     gatewayusage.CostBasisEstimated,
	})
	if usage.ContextTotalTokens > 0 {
		_ = service.usage.Ingest(ctx, gatewayusage.LedgerEntry{
			Category:      gatewayusage.CategoryContextToken,
			ProviderID:    strings.TrimSpace(model.ProviderID),
			ModelName:     strings.TrimSpace(model.ModelName),
			Channel:       strings.TrimSpace(channel),
			RequestID:     strings.TrimSpace(runID),
			RequestSource: normalizeUsageSource(source),
			Units:         usage.ContextTotalTokens,
			CostBasis:     gatewayusage.CostBasisEstimated,
		})
	}
}

func resolveUsageSource(metadata map[string]any, channel string, runKind string) string {
	for _, key := range []string{"usageSource", "requestSource"} {
		if source := normalizeUsageSource(resolveMetadataString(metadata, key)); source != usageSourceUnknown {
			return source
		}
	}
	if isOneShotRunKind(runKind) || isOneShotRunKind(resolveMetadataString(metadata, "runKind")) {
		return usageSourceOneShot
	}
	if strings.TrimSpace(channel) != "" {
		return usageSourceDialogue
	}
	return usageSourceRelay
}

func normalizeUsageSource(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "dialogue", "dialog", "chat", "channel":
		return usageSourceDialogue
	case "relay", "proxy", "forward":
		return usageSourceRelay
	case "oneshot", "one-shot", "single", "single-shot", "title":
		return usageSourceOneShot
	default:
		return usageSourceUnknown
	}
}

func resolveLLMOperation(runKind string, metadata map[string]any, isSubagent bool) string {
	normalizedKind := normalizeRunKind(runKind)
	if normalizedKind == "" {
		normalizedKind = normalizeRunKind(resolveMetadataString(metadata, "runKind"))
	}
	if isOneShotRunKind(normalizedKind) {
		if kind := resolveOneShotOperationKind(metadata); kind != "" {
			return "runtime." + kind
		}
		return "runtime.one_shot"
	}
	if isSubagent {
		return "runtime.subagent"
	}
	switch normalizedKind {
	case "heartbeat":
		return "runtime.heartbeat"
	case "cron":
		return "runtime.cron"
	case "subagent":
		return "runtime.subagent"
	case "user", "":
		return "runtime.run"
	default:
		return "runtime." + normalizedKind
	}
}
