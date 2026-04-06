package automation

import (
	"context"
	"time"
)

type JobRecord struct {
	ID        string
	Kind      string
	Status    string
	Config    any
	CreatedAt time.Time
	UpdatedAt time.Time
}

type RunRecord struct {
	ID        string
	JobID     string
	Status    string
	Error     string
	StartedAt time.Time
	EndedAt   time.Time
}

type TriggerLog struct {
	JobID     string
	EventID   string
	Payload   any
	CreatedAt time.Time
}

type Store interface {
	SaveJob(ctx context.Context, job JobRecord) error
	SaveRun(ctx context.Context, run RunRecord) error
	SaveTriggerLog(ctx context.Context, log TriggerLog) error
}
