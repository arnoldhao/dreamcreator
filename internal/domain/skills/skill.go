package skills

import (
	"strings"
	"time"
)

type ProviderSkillSpec struct {
	ID          string
	ProviderID  string
	Name        string
	Description string
	Version     string
	Enabled     bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type ProviderSkillSpecParams struct {
	ID          string
	ProviderID  string
	Name        string
	Description string
	Version     string
	Enabled     bool
	CreatedAt   *time.Time
	UpdatedAt   *time.Time
}

func NewProviderSkillSpec(params ProviderSkillSpecParams) (ProviderSkillSpec, error) {
	id := strings.TrimSpace(params.ID)
	providerID := strings.TrimSpace(params.ProviderID)
	name := strings.TrimSpace(params.Name)
	if id == "" || providerID == "" || name == "" {
		return ProviderSkillSpec{}, ErrInvalidSkill
	}

	createdAt := time.Now()
	updatedAt := createdAt
	if params.CreatedAt != nil {
		createdAt = *params.CreatedAt
	}
	if params.UpdatedAt != nil {
		updatedAt = *params.UpdatedAt
	}

	return ProviderSkillSpec{
		ID:          id,
		ProviderID:  providerID,
		Name:        name,
		Description: strings.TrimSpace(params.Description),
		Version:     strings.TrimSpace(params.Version),
		Enabled:     params.Enabled,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}, nil
}

type SkillBinding struct {
	ID         string
	ProviderID string
	SkillID    string
	Enabled    bool
}
