package externaltools

import "context"

type Repository interface {
	List(ctx context.Context) ([]ExternalTool, error)
	Get(ctx context.Context, name string) (ExternalTool, error)
	Save(ctx context.Context, tool ExternalTool) error
	Delete(ctx context.Context, name string) error
}
