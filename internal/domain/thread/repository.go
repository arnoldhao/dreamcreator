package thread

import (
    "context"
    "time"
)

type Repository interface {
    List(ctx context.Context, includeDeleted bool) ([]Thread, error)
    ListPurgeCandidates(ctx context.Context, before time.Time, limit int) ([]Thread, error)
    Get(ctx context.Context, id string) (Thread, error)
    Save(ctx context.Context, thread Thread) error
    SoftDelete(ctx context.Context, id string, deletedAt, purgeAfter *time.Time) error
    Restore(ctx context.Context, id string) error
    Purge(ctx context.Context, id string) error
    SetStatus(ctx context.Context, id string, status Status, updatedAt time.Time) error
}

type MessageRepository interface {
    ListByThread(ctx context.Context, threadID string, limit int) ([]ThreadMessage, error)
    Append(ctx context.Context, message ThreadMessage) error
    DeleteByThread(ctx context.Context, threadID string) error
}

type RunRepository interface {
    Get(ctx context.Context, id string) (ThreadRun, error)
    ListActiveByThread(ctx context.Context, threadID string) ([]ThreadRun, error)
    ListByAgentID(ctx context.Context, agentID string, limit int) ([]ThreadRun, error)
    CountActive(ctx context.Context) (int, error)
    Save(ctx context.Context, run ThreadRun) error
}

type RunEventRepository interface {
    Append(ctx context.Context, event ThreadRunEvent) (ThreadRunEvent, error)
    ListAfter(ctx context.Context, runID string, afterID int64, limit int) ([]ThreadRunEvent, error)
    ListByThread(ctx context.Context, threadID string, afterID int64, limit int, eventTypePrefix string) ([]ThreadRunEvent, error)
    GetLastEventID(ctx context.Context, runID string) (int64, error)
}
