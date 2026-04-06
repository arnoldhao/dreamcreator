package providers

import (
	"strings"
	"time"
)

type Model struct {
	ID                string
	ProviderID        string
	Name              string
	DisplayName       string
	CapabilitiesJSON  string
	ContextWindow     *int
	MaxOutputTokens   *int
	SupportsTools     *bool
	SupportsReasoning *bool
	SupportsVision    *bool
	SupportsAudio     *bool
	SupportsVideo     *bool
	Enabled           bool
	ShowInUI          bool
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type ModelParams struct {
	ID                string
	ProviderID        string
	Name              string
	DisplayName       string
	CapabilitiesJSON  string
	ContextWindow     *int
	MaxOutputTokens   *int
	SupportsTools     *bool
	SupportsReasoning *bool
	SupportsVision    *bool
	SupportsAudio     *bool
	SupportsVideo     *bool
	Enabled           bool
	ShowInUI          bool
	CreatedAt         *time.Time
	UpdatedAt         *time.Time
}

func NewModel(params ModelParams) (Model, error) {
	id := strings.TrimSpace(params.ID)
	providerID := strings.TrimSpace(params.ProviderID)
	name := strings.TrimSpace(params.Name)

	if id == "" || providerID == "" || name == "" {
		return Model{}, ErrInvalidModel
	}

	createdAt := time.Now()
	updatedAt := createdAt
	if params.CreatedAt != nil {
		createdAt = *params.CreatedAt
	}
	if params.UpdatedAt != nil {
		updatedAt = *params.UpdatedAt
	}

	return Model{
		ID:                id,
		ProviderID:        providerID,
		Name:              name,
		DisplayName:       strings.TrimSpace(params.DisplayName),
		CapabilitiesJSON:  strings.TrimSpace(params.CapabilitiesJSON),
		ContextWindow:     normalizeOptionalInt(params.ContextWindow),
		MaxOutputTokens:   normalizeOptionalInt(params.MaxOutputTokens),
		SupportsTools:     cloneBoolPtr(params.SupportsTools),
		SupportsReasoning: cloneBoolPtr(params.SupportsReasoning),
		SupportsVision:    cloneBoolPtr(params.SupportsVision),
		SupportsAudio:     cloneBoolPtr(params.SupportsAudio),
		SupportsVideo:     cloneBoolPtr(params.SupportsVideo),
		Enabled:           params.Enabled,
		ShowInUI:          params.ShowInUI,
		CreatedAt:         createdAt,
		UpdatedAt:         updatedAt,
	}, nil
}

func normalizeOptionalInt(value *int) *int {
	if value == nil {
		return nil
	}
	if *value <= 0 {
		return nil
	}
	normalized := *value
	return &normalized
}

func cloneBoolPtr(value *bool) *bool {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
