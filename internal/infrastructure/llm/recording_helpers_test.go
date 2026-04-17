package llm

import (
	"context"
	"errors"
	"testing"
)

func TestActiveLLMCallRecordFinishUsesDetachedContext(t *testing.T) {
	recorder := &contextCheckingRecorder{}
	ctx, cancel := context.WithCancel(context.Background())
	record := startActiveLLMCallRecord(ctx, recorder, RuntimeParams{
		ProviderID:    "provider-1",
		ModelName:     "model-1",
		RequestSource: "memory",
		Operation:     "memory.extract",
	}, `{"model":"model-1"}`)
	if record == nil {
		t.Fatal("expected active call record")
	}

	cancel()

	record.finishWithResponse(ctx, "stop", `{"usage":{"total_tokens":370}}`, &openAIUsage{
		PromptTokens:     200,
		CompletionTokens: 170,
		TotalTokens:      370,
	})

	if recorder.finishCount != 1 {
		t.Fatalf("expected finish to be recorded once, got %d", recorder.finishCount)
	}
	if recorder.finished.ID != "call-1" {
		t.Fatalf("expected finished record id %q, got %q", "call-1", recorder.finished.ID)
	}
	if recorder.finished.Status != CallRecordStatusCompleted {
		t.Fatalf("expected completed status, got %q", recorder.finished.Status)
	}
	if recorder.finished.TotalTokens != 370 {
		t.Fatalf("expected total tokens 370, got %d", recorder.finished.TotalTokens)
	}
	if recorder.finished.InputTokens != 200 || recorder.finished.OutputTokens != 170 {
		t.Fatalf("expected usage 200/170, got %d/%d", recorder.finished.InputTokens, recorder.finished.OutputTokens)
	}
}

type contextCheckingRecorder struct {
	finished    CallRecordFinish
	finishCount int
}

func (recorder *contextCheckingRecorder) StartLLMCall(_ context.Context, _ CallRecordStart) (string, error) {
	return "call-1", nil
}

func (recorder *contextCheckingRecorder) FinishLLMCall(ctx context.Context, record CallRecordFinish) error {
	if ctx == nil {
		return errors.New("finish ctx is nil")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	recorder.finishCount++
	recorder.finished = record
	return nil
}
