package thread

import (
	"strings"
	"time"
)

type Status string

type TitleChangedBy string

const (
	ThreadStatusRegular  Status = "regular"
	ThreadStatusArchived Status = "archived"

	ThreadTitleChangedByUser    TitleChangedBy = "user"
	ThreadTitleChangedBySummary TitleChangedBy = "summary"
)

type Thread struct {
	ID                string
	AgentID           string
	AssistantID       string
	Title             string
	TitleIsDefault    bool
	TitleChangedBy    TitleChangedBy
	Status            Status
	CreatedAt         time.Time
	UpdatedAt         time.Time
	LastInteractiveAt time.Time
	DeletedAt         *time.Time
	PurgeAfter        *time.Time
}

type ThreadParams struct {
	ID                string
	AgentID           string
	AssistantID       string
	Title             string
	TitleIsDefault    bool
	TitleChangedBy    TitleChangedBy
	Status            Status
	CreatedAt         *time.Time
	UpdatedAt         *time.Time
	LastInteractiveAt *time.Time
	DeletedAt         *time.Time
	PurgeAfter        *time.Time
}

func NewThread(params ThreadParams) (Thread, error) {
	id := strings.TrimSpace(params.ID)
	if id == "" {
		return Thread{}, ErrInvalidThread
	}
	createdAt := time.Now()
	updatedAt := createdAt
	lastInteractiveAt := updatedAt
	if params.CreatedAt != nil {
		createdAt = *params.CreatedAt
	}
	if params.UpdatedAt != nil {
		updatedAt = *params.UpdatedAt
	}
	if params.LastInteractiveAt != nil {
		lastInteractiveAt = *params.LastInteractiveAt
	} else if params.UpdatedAt != nil {
		lastInteractiveAt = *params.UpdatedAt
	}
	status := params.Status
	if status == "" {
		status = ThreadStatusRegular
	}

	return Thread{
		ID:                id,
		AgentID:           strings.TrimSpace(params.AgentID),
		AssistantID:       strings.TrimSpace(params.AssistantID),
		Title:             strings.TrimSpace(params.Title),
		TitleIsDefault:    params.TitleIsDefault,
		TitleChangedBy:    normalizeTitleChangedBy(params.TitleChangedBy),
		Status:            status,
		CreatedAt:         createdAt,
		UpdatedAt:         updatedAt,
		LastInteractiveAt: lastInteractiveAt,
		DeletedAt:         params.DeletedAt,
		PurgeAfter:        params.PurgeAfter,
	}, nil
}

func normalizeTitleChangedBy(value TitleChangedBy) TitleChangedBy {
	normalized := strings.ToLower(strings.TrimSpace(string(value)))
	switch TitleChangedBy(normalized) {
	case ThreadTitleChangedByUser, ThreadTitleChangedBySummary:
		return TitleChangedBy(normalized)
	default:
		return ""
	}
}
