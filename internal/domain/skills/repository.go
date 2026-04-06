package skills

import "context"

type Repository interface {
	ListByProvider(ctx context.Context, providerID string) ([]ProviderSkillSpec, error)
	Get(ctx context.Context, id string) (ProviderSkillSpec, error)
	Save(ctx context.Context, spec ProviderSkillSpec) error
	Delete(ctx context.Context, id string) error
}
