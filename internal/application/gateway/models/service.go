package models

import (
	"context"
	"strings"

	"dreamcreator/internal/domain/providers"
)

type ModelInfo struct {
	ID                  string `json:"id"`
	ProviderID          string `json:"providerId"`
	Name                string `json:"name"`
	DisplayName         string `json:"displayName,omitempty"`
	CapabilitiesJSON    string `json:"capabilitiesJson,omitempty"`
	ContextWindowTokens *int   `json:"contextWindowTokens,omitempty"`
	MaxOutputTokens     *int   `json:"maxOutputTokens,omitempty"`
	SupportsTools       *bool  `json:"supportsTools,omitempty"`
	SupportsReasoning   *bool  `json:"supportsReasoning,omitempty"`
	SupportsVision      *bool  `json:"supportsVision,omitempty"`
	SupportsAudio       *bool  `json:"supportsAudio,omitempty"`
	SupportsVideo       *bool  `json:"supportsVideo,omitempty"`
	Enabled             bool   `json:"enabled"`
	ShowInUI            bool   `json:"showInUi"`
}

type ModelsListResponse struct {
	Models []ModelInfo `json:"models"`
}

type Service struct {
	providers providers.ProviderRepository
	models    providers.ModelRepository
}

func NewService(providerRepo providers.ProviderRepository, modelRepo providers.ModelRepository) *Service {
	return &Service{providers: providerRepo, models: modelRepo}
}

func (service *Service) List(ctx context.Context, includeDisabled bool) (ModelsListResponse, error) {
	if service == nil || service.providers == nil || service.models == nil {
		return ModelsListResponse{}, nil
	}
	providersList, err := service.providers.List(ctx)
	if err != nil {
		return ModelsListResponse{}, err
	}
	result := make([]ModelInfo, 0)
	for _, provider := range providersList {
		if !includeDisabled && !provider.Enabled {
			continue
		}
		modelsList, err := service.models.ListByProvider(ctx, provider.ID)
		if err != nil {
			return ModelsListResponse{}, err
		}
		for _, model := range modelsList {
			if !includeDisabled && !model.Enabled {
				continue
			}
			result = append(result, ModelInfo{
				ID:                  strings.TrimSpace(model.ID),
				ProviderID:          strings.TrimSpace(model.ProviderID),
				Name:                strings.TrimSpace(model.Name),
				DisplayName:         strings.TrimSpace(model.DisplayName),
				CapabilitiesJSON:    strings.TrimSpace(model.CapabilitiesJSON),
				ContextWindowTokens: model.ContextWindow,
				MaxOutputTokens:     model.MaxOutputTokens,
				SupportsTools:       model.SupportsTools,
				SupportsReasoning:   model.SupportsReasoning,
				SupportsVision:      model.SupportsVision,
				SupportsAudio:       model.SupportsAudio,
				SupportsVideo:       model.SupportsVideo,
				Enabled:             model.Enabled,
				ShowInUI:            model.ShowInUI,
			})
		}
	}
	return ModelsListResponse{Models: result}, nil
}
