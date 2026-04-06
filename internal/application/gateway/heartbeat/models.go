package heartbeat

import (
	"context"
	"errors"
	"time"
)

var (
	ErrEventNotFound = errors.New("heartbeat event not found")
)

type EventStatus string

const (
	StatusSent    EventStatus = "sent"
	StatusOKToken EventStatus = "ok-token"
	StatusOKEmpty EventStatus = "ok-empty"
	StatusSkipped EventStatus = "skipped"
	StatusFailed  EventStatus = "failed"
)

type EventIndicatorType string

const (
	IndicatorOK    EventIndicatorType = "ok"
	IndicatorAlert EventIndicatorType = "alert"
	IndicatorError EventIndicatorType = "error"
)

type Event struct {
	ID          string
	SessionKey  string
	ThreadID    string
	Status      EventStatus
	Message     string
	Error       string
	ContentHash string
	Indicator   EventIndicatorType
	Silent      bool
	Reason      string
	Source      string
	RunID       string
	CreatedAt   time.Time
}

type EventStore interface {
	Save(ctx context.Context, event Event) error
	Last(ctx context.Context, sessionKey string) (Event, error)
	HasDuplicate(ctx context.Context, sessionKey string, contentHash string, since time.Time) (bool, error)
}

type ChecklistItem struct {
	ID       string `json:"id"`
	Text     string `json:"text"`
	Done     bool   `json:"done"`
	Priority string `json:"priority,omitempty"`
}

type Spec struct {
	Title     string          `json:"title,omitempty"`
	Items     []ChecklistItem `json:"items,omitempty"`
	Notes     string          `json:"notes,omitempty"`
	Version   int             `json:"version"`
	UpdatedAt time.Time       `json:"updatedAt"`
}

type SystemEventInput struct {
	SessionKey string `json:"sessionKey"`
	Text       string `json:"text"`
	ContextKey string `json:"contextKey,omitempty"`
	RunID      string `json:"runId,omitempty"`
	Source     string `json:"source,omitempty"`
}

type TriggerInput struct {
	Reason     string `json:"reason,omitempty"`
	SessionKey string `json:"sessionKey,omitempty"`
	Force      bool   `json:"force,omitempty"`
	Source     string `json:"source,omitempty"`
	RunID      string `json:"runId,omitempty"`
}

type TriggerExecutionStatus string

const (
	TriggerExecutionQueued  TriggerExecutionStatus = "queued"
	TriggerExecutionRan     TriggerExecutionStatus = "ran"
	TriggerExecutionSkipped TriggerExecutionStatus = "skipped"
	TriggerExecutionFailed  TriggerExecutionStatus = "failed"
)

type TriggerResult struct {
	Accepted       bool                   `json:"accepted"`
	ExecutedStatus TriggerExecutionStatus `json:"executedStatus"`
	Reason         string                 `json:"reason,omitempty"`
}
