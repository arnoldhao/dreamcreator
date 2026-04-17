package llm

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

const maxStreamTranscriptChars = 128 * 1024
const callRecordPersistenceTimeout = 5 * time.Second

type activeLLMCallRecord struct {
	recorder  CallRecorder
	id        string
	startedAt time.Time
	finish    sync.Once
}

func startActiveLLMCallRecord(ctx context.Context, recorder CallRecorder, params RuntimeParams, requestPayload string) *activeLLMCallRecord {
	if recorder == nil {
		return nil
	}
	startedAt := time.Now().UTC()
	id, err := recorder.StartLLMCall(ctx, CallRecordStart{
		ProviderID:      strings.TrimSpace(params.ProviderID),
		ModelName:       strings.TrimSpace(params.ModelName),
		SessionID:       strings.TrimSpace(params.SessionID),
		ThreadID:        strings.TrimSpace(params.ThreadID),
		RunID:           strings.TrimSpace(params.RunID),
		RequestSource:   strings.TrimSpace(params.RequestSource),
		Operation:       strings.TrimSpace(params.Operation),
		RequestPayload:  requestPayload,
		ResponsePayload: "",
		StartedAt:       startedAt,
	})
	if err != nil || strings.TrimSpace(id) == "" {
		return nil
	}
	return &activeLLMCallRecord{
		recorder:  recorder,
		id:        strings.TrimSpace(id),
		startedAt: startedAt,
	}
}

func (record *activeLLMCallRecord) finishWithResponse(ctx context.Context, finishReason string, responsePayload string, usage *openAIUsage) {
	record.finishWithStatus(ctx, CallRecordStatusCompleted, finishReason, "", responsePayload, usage)
}

func (record *activeLLMCallRecord) finishWithError(ctx context.Context, err error, responsePayload string) {
	if record == nil {
		return
	}
	status := CallRecordStatusError
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		status = CallRecordStatusCancelled
	}
	record.finishWithStatus(ctx, status, "", errorText(err), responsePayload, nil)
}

func (record *activeLLMCallRecord) finishWithStatus(ctx context.Context, status string, finishReason string, errorText string, responsePayload string, usage *openAIUsage) {
	if record == nil || record.recorder == nil || strings.TrimSpace(record.id) == "" {
		return
	}
	record.finish.Do(func() {
		finishedAt := time.Now().UTC()
		update := CallRecordFinish{
			ID:              record.id,
			Status:          normalizeCallRecordStatus(status),
			FinishReason:    strings.TrimSpace(finishReason),
			ErrorText:       strings.TrimSpace(errorText),
			ResponsePayload: responsePayload,
			FinishedAt:      finishedAt,
		}
		if usage != nil {
			update.InputTokens = usage.PromptTokens
			update.OutputTokens = usage.CompletionTokens
			update.TotalTokens = usage.TotalTokens
		}
		persistCtx := context.Background()
		if ctx != nil {
			persistCtx = context.WithoutCancel(ctx)
		}
		persistCtx, cancel := context.WithTimeout(persistCtx, callRecordPersistenceTimeout)
		defer cancel()
		if err := record.recorder.FinishLLMCall(persistCtx, update); err != nil {
			zap.L().Warn("llm call record finish failed",
				zap.String("callRecordID", record.id),
				zap.String("status", update.Status),
				zap.Error(err),
			)
		}
	})
}

type streamTranscriptBuilder struct {
	builder   strings.Builder
	truncated bool
}

func (builder *streamTranscriptBuilder) Append(payload string) {
	if builder == nil || payload == "" {
		return
	}
	if builder.builder.Len() >= maxStreamTranscriptChars {
		builder.truncated = true
		return
	}
	next := payload
	if builder.builder.Len()+len(next)+1 > maxStreamTranscriptChars {
		remaining := maxStreamTranscriptChars - builder.builder.Len()
		if remaining <= 0 {
			builder.truncated = true
			return
		}
		if len(next) > remaining {
			next = next[:remaining]
			builder.truncated = true
		}
	}
	if builder.builder.Len() > 0 {
		builder.builder.WriteByte('\n')
	}
	builder.builder.WriteString(next)
}

func (builder *streamTranscriptBuilder) JSONPayload() string {
	if builder == nil || builder.builder.Len() == 0 {
		return ""
	}
	payload, err := json.Marshal(map[string]any{
		"stream":    true,
		"sse":       builder.builder.String(),
		"truncated": builder.truncated,
	})
	if err != nil {
		return ""
	}
	return string(payload)
}

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return strings.TrimSpace(err.Error())
}
