package providers

import (
	"strings"
	"time"
)

type ProviderType string

const (
	ProviderTypeOpenAI    ProviderType = "openai"
	ProviderTypeAnthropic ProviderType = "anthropic"
)

type ProviderCompatibility string

const (
	ProviderCompatibilityOpenAI     ProviderCompatibility = "openai"
	ProviderCompatibilityAnthropic  ProviderCompatibility = "anthropic"
	ProviderCompatibilityDeepSeek   ProviderCompatibility = "deepseek"
	ProviderCompatibilityOpenRouter ProviderCompatibility = "openrouter"
	ProviderCompatibilityGoogle     ProviderCompatibility = "google"
)

type Provider struct {
	ID            string
	Name          string
	Type          ProviderType
	Compatibility ProviderCompatibility
	Endpoint      string
	Enabled       bool
	Builtin       bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type ProviderParams struct {
	ID            string
	Name          string
	Type          string
	Compatibility string
	Endpoint      string
	Enabled       bool
	Builtin       bool
	CreatedAt     *time.Time
	UpdatedAt     *time.Time
}

func NewProvider(params ProviderParams) (Provider, error) {
	id := strings.TrimSpace(params.ID)
	name := strings.TrimSpace(params.Name)
	providerType := ProviderType(strings.TrimSpace(params.Type))
	compatibility := ProviderCompatibility(strings.TrimSpace(params.Compatibility))
	endpoint := strings.TrimSpace(params.Endpoint)

	if id == "" || name == "" || providerType == "" {
		return Provider{}, ErrInvalidProvider
	}
	if compatibility == "" {
		compatibility = defaultCompatibilityForType(providerType)
	}
	if !providerTypeSupportsCompatibility(providerType, compatibility) {
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
		ID:            id,
		Name:          name,
		Type:          providerType,
		Compatibility: compatibility,
		Endpoint:      endpoint,
		Enabled:       params.Enabled,
		Builtin:       params.Builtin,
		CreatedAt:     createdAt,
		UpdatedAt:     updatedAt,
	}, nil
}

func defaultCompatibilityForType(providerType ProviderType) ProviderCompatibility {
	switch providerType {
	case ProviderTypeAnthropic:
		return ProviderCompatibilityAnthropic
	default:
		return ProviderCompatibilityOpenAI
	}
}

func providerTypeSupportsCompatibility(providerType ProviderType, compatibility ProviderCompatibility) bool {
	switch providerType {
	case ProviderTypeAnthropic:
		return compatibility == ProviderCompatibilityAnthropic
	case ProviderTypeOpenAI:
		switch compatibility {
		case ProviderCompatibilityOpenAI,
			ProviderCompatibilityDeepSeek,
			ProviderCompatibilityOpenRouter,
			ProviderCompatibilityGoogle:
			return true
		default:
			return false
		}
	default:
		return false
	}
}
