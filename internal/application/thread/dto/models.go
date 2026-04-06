package dto

import (
	"dreamcreator/internal/application/chatevent"
	"dreamcreator/internal/domain/thread"
)

type Thread struct {
	ID                string        `json:"id"`
	AssistantID       string        `json:"assistantId"`
	Title             string        `json:"title"`
	TitleIsDefault    bool          `json:"titleIsDefault"`
	TitleChangedBy    string        `json:"titleChangedBy,omitempty"`
	Status            thread.Status `json:"status"`
	CreatedAt         string        `json:"createdAt"`
	UpdatedAt         string        `json:"updatedAt"`
	LastInteractiveAt string        `json:"lastInteractiveAt"`
	DeletedAt         string        `json:"deletedAt"`
	PurgeAfter        string        `json:"purgeAfter"`
}

type Message struct {
	ID           string `json:"id"`
	Kind         string `json:"kind,omitempty"`
	Role         string `json:"role"`
	Content      string `json:"content"`
	PartsJSON    string `json:"partsJson,omitempty"`
	PartsVersion int    `json:"partsVersion,omitempty"`
	CreatedAt    string `json:"createdAt"`
}

type ThreadRunEvent struct {
	ID          int64  `json:"id"`
	RunID       string `json:"runId"`
	ThreadID    string `json:"threadId"`
	EventType   string `json:"eventType"`
	PayloadJSON string `json:"payloadJson"`
	CreatedAt   string `json:"createdAt"`
}

type ListThreadRunEventsRequest struct {
	ThreadID        string `json:"threadId"`
	AfterID         int64  `json:"afterId"`
	Limit           int    `json:"limit"`
	EventTypePrefix string `json:"eventTypePrefix,omitempty"`
}

type AppendMessageRequest struct {
	ID       string                  `json:"id,omitempty"`
	ThreadID string                  `json:"threadId"`
	Kind     string                  `json:"kind,omitempty"`
	Role     string                  `json:"role"`
	Content  string                  `json:"content"`
	Parts    []chatevent.MessagePart `json:"parts,omitempty"`
}

type NewThreadRequest struct {
	Title          string `json:"title"`
	IsDefaultTitle bool   `json:"isDefaultTitle"`
	AssistantID    string `json:"assistantId"`
}

type NewThreadResponse struct {
	ThreadID    string `json:"threadId"`
	AssistantID string `json:"assistantId"`
}

type ContextTokensSnapshot struct {
	PromptTokens        int    `json:"promptTokens,omitempty"`
	TotalTokens         int    `json:"totalTokens,omitempty"`
	ContextWindowTokens int    `json:"contextWindowTokens,omitempty"`
	UpdatedAt           string `json:"updatedAt,omitempty"`
	Fresh               bool   `json:"fresh"`
}

type RenameThreadRequest struct {
	ThreadID string `json:"threadId"`
	Title    string `json:"title"`
}

type SetThreadStatusRequest struct {
	ThreadID string        `json:"threadId"`
	Status   thread.Status `json:"status"`
}

type GenerateThreadTitleRequest struct {
	ThreadID      string                       `json:"threadId"`
	FallbackTitle string                       `json:"fallbackTitle,omitempty"`
	Messages      []GenerateThreadTitleMessage `json:"messages,omitempty"`
}

type GenerateThreadTitleMessage struct {
	Role    string                  `json:"role"`
	Content string                  `json:"content"`
	Parts   []chatevent.MessagePart `json:"parts,omitempty"`
}

type GenerateThreadTitleResponse struct {
	ThreadID       string `json:"threadId"`
	Title          string `json:"title"`
	TitleIsDefault bool   `json:"titleIsDefault"`
	TitleChangedBy string `json:"titleChangedBy,omitempty"`
	Updated        bool   `json:"updated,omitempty"`
}
