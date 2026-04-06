package assistant

import "context"

type Repository interface {
	List(ctx context.Context, includeDisabled bool) ([]Assistant, error)
	Get(ctx context.Context, id string) (Assistant, error)
	Save(ctx context.Context, assistant Assistant) error
	Delete(ctx context.Context, id string) error
	SetDefault(ctx context.Context, id string) error
}
