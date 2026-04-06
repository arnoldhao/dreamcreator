package providers

import "context"

type ProviderRepository interface {
	List(ctx context.Context) ([]Provider, error)
	Get(ctx context.Context, id string) (Provider, error)
	Save(ctx context.Context, provider Provider) error
	Delete(ctx context.Context, id string) error
}

type ModelRepository interface {
	ListByProvider(ctx context.Context, providerID string) ([]Model, error)
	Get(ctx context.Context, id string) (Model, error)
	Save(ctx context.Context, model Model) error
	ReplaceByProvider(ctx context.Context, providerID string, models []Model) error
	Delete(ctx context.Context, id string) error
}

type SecretRepository interface {
	GetByProviderID(ctx context.Context, providerID string) (ProviderSecret, error)
	Save(ctx context.Context, secret ProviderSecret) error
	DeleteByProviderID(ctx context.Context, providerID string) error
}
