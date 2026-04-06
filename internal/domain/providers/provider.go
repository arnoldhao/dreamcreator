package providers

import (
	"strings"
	"time"
)

type ProviderType string

const (
	ProviderTypeOpenAI     ProviderType = "openai"
	ProviderTypeAnthropic  ProviderType = "anthropic"
)

type Provider struct {
	ID        string
	Name      string
	Type      ProviderType
	Endpoint  string
	Enabled   bool
	Builtin   bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type ProviderParams struct {
	ID        string
	Name      string
	Type      string
	Endpoint  string
	Enabled   bool
	Builtin   bool
	CreatedAt *time.Time
	UpdatedAt *time.Time
}

func NewProvider(params ProviderParams) (Provider, error) {
	id := strings.TrimSpace(params.ID)
	name := strings.TrimSpace(params.Name)
	providerType := ProviderType(strings.TrimSpace(params.Type))
	endpoint := strings.TrimSpace(params.Endpoint)

	if id == "" || name == "" || providerType == "" {
		return Provider{}, ErrInvalidProvider
	}

	createdAt := time.Now()
	updatedAt := createdAt
	if params.CreatedAt != nil {
		createdAt = *params.CreatedAt
	}
	if params.UpdatedAt != nil {
		updatedAt = *params.UpdatedAt
	}

	return Provider{
		ID:        id,
		Name:      name,
		Type:      providerType,
		Endpoint:  endpoint,
		Enabled:   params.Enabled,
		Builtin:   params.Builtin,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}, nil
}
