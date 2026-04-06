package memory

import (
	"strings"
	"time"
)

type STMState struct {
	ID             string
	ThreadID       string
	WindowJSON     string
	Summary        string
	UpdatedAt      time.Time
}

type LTMEntry struct {
	ID             string
	ThreadID       string
	Content        string
	Category       string
	Confidence     float32
	SourceJSON     string
	CreatedAt      time.Time
}

type MemoryWritePolicy struct {
	MinConfidence float32
	MaxEntries    int
}

type RetrievalPolicy struct {
	TopK int
}

type Document struct {
	ID          string
	WorkspaceID string
	Name        string
	Content     string
	CreatedAt   time.Time
}

type Chunk struct {
	ID         string
	DocumentID string
	Content    string
	IndexRef   string
}

type IndexRef struct {
	ID        string
	Provider  string
	Namespace string
}

type STMStateParams struct {
	ID             string
	ThreadID       string
	WindowJSON     string
	Summary        string
	UpdatedAt      *time.Time
}

func NewSTMState(params STMStateParams) (STMState, error) {
	id := strings.TrimSpace(params.ID)
	threadID := strings.TrimSpace(params.ThreadID)
	if id == "" || threadID == "" {
		return STMState{}, ErrInvalidMemory
	}
	updatedAt := time.Now()
	if params.UpdatedAt != nil {
		updatedAt = *params.UpdatedAt
	}
	return STMState{
		ID:             id,
		ThreadID:       threadID,
		WindowJSON:     strings.TrimSpace(params.WindowJSON),
		Summary:        strings.TrimSpace(params.Summary),
		UpdatedAt:      updatedAt,
	}, nil
}
