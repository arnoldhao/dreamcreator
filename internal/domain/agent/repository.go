package agent

import (
	"context"
	"time"
)

type Repository interface {
	List(ctx context.Context, includeDisabled bool) ([]Agent, error)
	Get(ctx context.Context, id string) (Agent, error)
	Save(ctx context.Context, agent Agent) error
	SoftDelete(ctx context.Context, id string, deletedAt *time.Time) error
}
