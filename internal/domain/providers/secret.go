package providers

import (
	"strings"
	"time"
)

type ProviderSecret struct {
	ID         string
	ProviderID string
	APIKey     string
	OrgRef     string
	CreatedAt  time.Time
}

type ProviderSecretParams struct {
	ID         string
	ProviderID string
	APIKey     string
	OrgRef     string
	CreatedAt  *time.Time
}

func NewProviderSecret(params ProviderSecretParams) (ProviderSecret, error) {
	providerID := strings.TrimSpace(params.ProviderID)
	if providerID == "" {
		return ProviderSecret{}, ErrInvalidSecret
	}
	id := strings.TrimSpace(params.ID)
	if id == "" {
		id = providerID
	}

	createdAt := time.Now()
	if params.CreatedAt != nil {
		createdAt = *params.CreatedAt
	}

	return ProviderSecret{
		ID:         id,
		ProviderID: providerID,
		APIKey:     strings.TrimSpace(params.APIKey),
		OrgRef:     strings.TrimSpace(params.OrgRef),
		CreatedAt:  createdAt,
	}, nil
}
