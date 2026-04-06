package memory

import "context"

type STMRepository interface {
	GetByThread(ctx context.Context, threadID string) (STMState, error)
	Save(ctx context.Context, state STMState) error
}

type LTMRepository interface {
	ListByThread(ctx context.Context, threadID string, limit int) ([]LTMEntry, error)
	Save(ctx context.Context, entry LTMEntry) error
}

type DocumentRepository interface {
	Save(ctx context.Context, doc Document) error
	ListByWorkspace(ctx context.Context, workspaceID string, limit int) ([]Document, error)
}
