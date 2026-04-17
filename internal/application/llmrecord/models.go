package llmrecord

import "time"

type Record struct {
	ID                  string    `json:"id"`
	SessionID           string    `json:"sessionId,omitempty"`
	ThreadID            string    `json:"threadId,omitempty"`
	RunID               string    `json:"runId,omitempty"`
	ProviderID          string    `json:"providerId,omitempty"`
	ModelName           string    `json:"modelName,omitempty"`
	RequestSource       string    `json:"requestSource,omitempty"`
	Operation           string    `json:"operation,omitempty"`
	Status              string    `json:"status"`
	FinishReason        string    `json:"finishReason,omitempty"`
	ErrorText           string    `json:"errorText,omitempty"`
	InputTokens         int       `json:"inputTokens,omitempty"`
	OutputTokens        int       `json:"outputTokens,omitempty"`
	TotalTokens         int       `json:"totalTokens,omitempty"`
	ContextPromptTokens int       `json:"contextPromptTokens,omitempty"`
	ContextTotalTokens  int       `json:"contextTotalTokens,omitempty"`
	ContextWindowTokens int       `json:"contextWindowTokens,omitempty"`
	RequestPayloadJSON  string    `json:"requestPayloadJson,omitempty"`
	ResponsePayloadJSON string    `json:"responsePayloadJson,omitempty"`
	PayloadTruncated    bool      `json:"payloadTruncated,omitempty"`
	StartedAt           time.Time `json:"startedAt"`
	FinishedAt          time.Time `json:"finishedAt,omitempty"`
	DurationMS          int64     `json:"durationMs,omitempty"`
}

type ListRequest struct {
	ThreadID      string `json:"threadId,omitempty"`
	RunID         string `json:"runId,omitempty"`
	ProviderID    string `json:"providerId,omitempty"`
	ModelName     string `json:"modelName,omitempty"`
	RequestSource string `json:"requestSource,omitempty"`
	Status        string `json:"status,omitempty"`
	StartAt       string `json:"startAt,omitempty"`
	EndAt         string `json:"endAt,omitempty"`
	Limit         int    `json:"limit,omitempty"`
}
