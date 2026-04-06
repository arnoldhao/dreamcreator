package thread

import (
	"strings"
	"time"
)

type MessageKind string

const (
	ThreadMessageKindChat   MessageKind = "chat"
	ThreadMessageKindNotice MessageKind = "notice"
)

type ThreadMessage struct {
	ID        string
	ThreadID  string
	Kind      MessageKind
	Role      string
	Content   string
	PartsJSON string
	CreatedAt time.Time
}

type ThreadMessageParams struct {
	ID        string
	ThreadID  string
	Kind      MessageKind
	Role      string
	Content   string
	PartsJSON string
	CreatedAt *time.Time
}

func NewThreadMessage(params ThreadMessageParams) (ThreadMessage, error) {
	id := strings.TrimSpace(params.ID)
	threadID := strings.TrimSpace(params.ThreadID)
	role := strings.TrimSpace(params.Role)
	content := strings.TrimSpace(params.Content)
	partsJSON := strings.TrimSpace(params.PartsJSON)
	kind := normalizeMessageKind(params.Kind)
	if partsJSON == "" {
		partsJSON = "[]"
	}
	if id == "" || threadID == "" || role == "" {
		return ThreadMessage{}, ErrInvalidThreadMessage
	}
	if content == "" && (partsJSON == "" || partsJSON == "[]") {
		return ThreadMessage{}, ErrInvalidThreadMessage
	}
	createdAt := time.Now()
	if params.CreatedAt != nil {
		createdAt = *params.CreatedAt
	}

	return ThreadMessage{
		ID:        id,
		ThreadID:  threadID,
		Kind:      kind,
		Role:      role,
		Content:   content,
		PartsJSON: partsJSON,
		CreatedAt: createdAt,
	}, nil
}

func normalizeMessageKind(value MessageKind) MessageKind {
	switch MessageKind(strings.ToLower(strings.TrimSpace(string(value)))) {
	case ThreadMessageKindNotice:
		return ThreadMessageKindNotice
	default:
		return ThreadMessageKindChat
	}
}
