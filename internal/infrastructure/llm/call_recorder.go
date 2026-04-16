package llm

import (
	"context"
	"strings"
	"time"
)

const (
	CallRecordStatusStarted   = "started"
	CallRecordStatusCompleted = "completed"
	CallRecordStatusError     = "error"
	CallRecordStatusCancelled = "cancelled"
)

type CallRecorder interface {
	StartLLMCall(ctx context.Context, record CallRecordStart) (string, error)
	FinishLLMCall(ctx context.Context, record CallRecordFinish) error
}

type CallRecordStart struct {
	ProviderID      string
	ModelName       string
	SessionID       string
	ThreadID        string
	RunID           string
	RequestSource   string
	Operation       string
	RequestPayload  string
	ResponsePayload string
	StartedAt       time.Time
}

type CallRecordFinish struct {
	ID                  string
	Status              string
	FinishReason        string
	ErrorText           string
	ResponsePayload     string
	InputTokens         int
	OutputTokens        int
	TotalTokens         int
	ContextPromptTokens int
	ContextTotalTokens  int
	ContextWindowTokens int
	FinishedAt          time.Time
}

func normalizeCallRecordStatus(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case CallRecordStatusCompleted:
		return CallRecordStatusCompleted
	case CallRecordStatusCancelled:
		return CallRecordStatusCancelled
	case CallRecordStatusStarted:
		return CallRecordStatusStarted
	default:
		return CallRecordStatusError
	}
}
