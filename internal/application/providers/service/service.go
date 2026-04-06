package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"dreamcreator/internal/application/providers/dto"
	"dreamcreator/internal/domain/providers"
)

type ProvidersService struct {
	providers providers.ProviderRepository
	models    providers.ModelRepository
	secrets   providers.SecretRepository
	syncer    ModelSyncer
	logo      ProviderLogoResolver
	now       func() time.Time
}

type ModelSyncer interface {
	Sync(ctx context.Context, provider providers.Provider, apiKey string) ([]providers.Model, error)
}

type ModelDisplayNameResolver interface {
	ResolveModelDisplayNames(ctx context.Context, modelIDs []string) (map[string]string, error)
}

type ModelCapabilitiesResolver interface {
	ResolveModelCapabilitiesJSON(ctx context.Context, modelIDs []string) (map[string]string, error)
}

type ProviderLogoResolver interface {
	ResolveProviderLogo(ctx context.Context, providerID string) (string, error)
}

func NewProvidersService(providerRepo providers.ProviderRepository, modelRepo providers.ModelRepository, secretRepo providers.SecretRepository, syncer ModelSyncer, logoResolver ProviderLogoResolver) *ProvidersService {
	return &ProvidersService{
		providers: providerRepo,
		models:    modelRepo,
		secrets:   secretRepo,
		syncer:    syncer,
		logo:      logoResolver,
		now:       time.Now,
	}
}

func (service *ProvidersService) ListProviders(ctx context.Context) ([]dto.Provider, error) {
	items, err := service.providers.List(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]dto.Provider, 0, len(items))
	for _, item := range items {
		result = append(result, dto.Provider{
			ID:       item.ID,
			Name:     item.Name,
			Type:     string(item.Type),
			Endpoint: item.Endpoint,
			Enabled:  item.Enabled,
			Builtin:  item.Builtin,
			Icon:     service.resolveProviderIcon(ctx, item),
		})
	}
	return result, nil
}

func (service *ProvidersService) GetProvider(ctx context.Context, providerID string) (dto.Provider, error) {
	trimmed := strings.TrimSpace(providerID)
	if trimmed == "" {
		return dto.Provider{}, providers.ErrProviderNotFound
	}
	item, err := service.providers.Get(ctx, trimmed)
	if err != nil {
		return dto.Provider{}, err
	}
	return dto.Provider{
		ID:       item.ID,
		Name:     item.Name,
		Type:     string(item.Type),
		Endpoint: item.Endpoint,
		Enabled:  item.Enabled,
		Builtin:  item.Builtin,
	}, nil
}

func (service *ProvidersService) ListEnabledProvidersWithModels(ctx context.Context) ([]dto.ProviderWithModels, error) {
	items, err := service.providers.List(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]dto.ProviderWithModels, 0, len(items))
	for _, item := range items {
		if !item.Enabled {
			continue
		}
		models, err := service.models.ListByProvider(ctx, item.ID)
		if err != nil {
			return nil, err
		}
		enabledModels := make([]dto.ProviderModel, 0, len(models))
		for _, model := range models {
			if !model.Enabled {
				continue
			}
			enabledModels = append(enabledModels, dto.ProviderModel{
				ID:                  model.ID,
				ProviderID:          model.ProviderID,
				Name:                model.Name,
				DisplayName:         model.DisplayName,
				CapabilitiesJSON:    model.CapabilitiesJSON,
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
		result = append(result, dto.ProviderWithModels{
			Provider: dto.Provider{
				ID:       item.ID,
				Name:     item.Name,
				Type:     string(item.Type),
				Endpoint: item.Endpoint,
				Enabled:  item.Enabled,
				Builtin:  item.Builtin,
				Icon:     service.resolveProviderIcon(ctx, item),
			},
			Models: enabledModels,
		})
	}
	return result, nil
}

func (service *ProvidersService) UpsertProvider(ctx context.Context, request dto.UpsertProviderRequest) (dto.Provider, error) {
	providerType, err := sanitizeProviderType(request.Type)
	if err != nil {
		return dto.Provider{}, err
	}

	id := strings.TrimSpace(request.ID)
	createdAt := (*time.Time)(nil)
	builtin := false
	if id == "" {
		id = uuid.NewString()
	} else {
		existing, err := service.providers.Get(ctx, id)
		if err == nil {
			createdAt = &existing.CreatedAt
			builtin = existing.Builtin
		} else if err != providers.ErrProviderNotFound {
			return dto.Provider{}, err
		} else if isDefaultProviderID(id) {
			builtin = true
		}
	}

	now := service.now()
	provider, err := providers.NewProvider(providers.ProviderParams{
		ID:        id,
		Name:      request.Name,
		Type:      string(providerType),
		Endpoint:  request.Endpoint,
		Enabled:   request.Enabled,
		Builtin:   builtin,
		CreatedAt: createdAt,
		UpdatedAt: &now,
	})
	if err != nil {
		return dto.Provider{}, err
	}

	if err := service.providers.Save(ctx, provider); err != nil {
		return dto.Provider{}, err
	}

	return dto.Provider{
		ID:       provider.ID,
		Name:     provider.Name,
		Type:     string(provider.Type),
		Endpoint: provider.Endpoint,
		Enabled:  provider.Enabled,
		Builtin:  provider.Builtin,
		Icon:     service.resolveProviderIcon(ctx, provider),
	}, nil
}

func (service *ProvidersService) DeleteProvider(ctx context.Context, id string) error {
	trimmed := strings.TrimSpace(id)
	if trimmed == "" {
		return providers.ErrInvalidProvider
	}
	existing, err := service.providers.Get(ctx, trimmed)
	if err != nil {
		return err
	}
	if existing.Builtin {
		return fmt.Errorf("builtin provider cannot be deleted")
	}
	return service.providers.Delete(ctx, trimmed)
}

func (service *ProvidersService) ListProviderModels(ctx context.Context, providerID string) ([]dto.ProviderModel, error) {
	trimmed := strings.TrimSpace(providerID)
	if trimmed == "" {
		return nil, providers.ErrProviderNotFound
	}
	items, err := service.models.ListByProvider(ctx, trimmed)
	if err != nil {
		return nil, err
	}
	result := make([]dto.ProviderModel, 0, len(items))
	for _, model := range items {
		result = append(result, dto.ProviderModel{
			ID:                  model.ID,
			ProviderID:          model.ProviderID,
			Name:                model.Name,
			DisplayName:         model.DisplayName,
			CapabilitiesJSON:    model.CapabilitiesJSON,
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
	return result, nil
}

func (service *ProvidersService) UpdateProviderModel(ctx context.Context, request dto.UpdateProviderModelRequest) (dto.ProviderModel, error) {
	modelID := strings.TrimSpace(request.ID)
	if modelID == "" {
		return dto.ProviderModel{}, providers.ErrModelNotFound
	}

	model, err := service.models.Get(ctx, modelID)
	if err != nil {
		return dto.ProviderModel{}, err
	}
	if request.ProviderID != "" && model.ProviderID != request.ProviderID {
		return dto.ProviderModel{}, providers.ErrModelNotFound
	}

	now := service.now()
	updated, err := providers.NewModel(providers.ModelParams{
		ID:                model.ID,
		ProviderID:        model.ProviderID,
		Name:              model.Name,
		DisplayName:       model.DisplayName,
		CapabilitiesJSON:  model.CapabilitiesJSON,
		ContextWindow:     model.ContextWindow,
		MaxOutputTokens:   model.MaxOutputTokens,
		SupportsTools:     model.SupportsTools,
		SupportsReasoning: model.SupportsReasoning,
		SupportsVision:    model.SupportsVision,
		SupportsAudio:     model.SupportsAudio,
		SupportsVideo:     model.SupportsVideo,
		Enabled:           request.Enabled,
		ShowInUI:          request.ShowInUI,
		CreatedAt:         &model.CreatedAt,
		UpdatedAt:         &now,
	})
	if err != nil {
		return dto.ProviderModel{}, err
	}

	if err := service.models.Save(ctx, updated); err != nil {
		return dto.ProviderModel{}, err
	}

	return dto.ProviderModel{
		ID:                  updated.ID,
		ProviderID:          updated.ProviderID,
		Name:                updated.Name,
		DisplayName:         updated.DisplayName,
		CapabilitiesJSON:    updated.CapabilitiesJSON,
		ContextWindowTokens: updated.ContextWindow,
		MaxOutputTokens:     updated.MaxOutputTokens,
		SupportsTools:       updated.SupportsTools,
		SupportsReasoning:   updated.SupportsReasoning,
		SupportsVision:      updated.SupportsVision,
		SupportsAudio:       updated.SupportsAudio,
		SupportsVideo:       updated.SupportsVideo,
		Enabled:             updated.Enabled,
		ShowInUI:            updated.ShowInUI,
	}, nil
}

func (service *ProvidersService) SyncProviderModels(ctx context.Context, providerID string, apiKey string) ([]dto.ProviderModel, error) {
	if service.syncer == nil {
		return nil, fmt.Errorf("model syncer is not configured")
	}
	trimmed := strings.TrimSpace(providerID)
	if trimmed == "" {
		return nil, providers.ErrProviderNotFound
	}

	provider, err := service.providers.Get(ctx, trimmed)
	if err != nil {
		return nil, err
	}

	models, err := service.syncer.Sync(ctx, provider, apiKey)
	if err != nil {
		return nil, err
	}

	existing, err := service.models.ListByProvider(ctx, provider.ID)
	if err != nil {
		return nil, err
	}

	metadataByName := make(map[string]providers.Model, len(models))
	for _, model := range models {
		if model.Name == "" {
			continue
		}
		metadataByName[model.Name] = model
	}

	now := service.now()
	existingByName := make(map[string]providers.Model, len(existing))
	for _, model := range existing {
		if model.Name == "" {
			continue
		}
		existingByName[model.Name] = model
	}

	nextModels := make([]providers.Model, 0, len(metadataByName)+len(existing))
	for _, meta := range metadataByName {
		if meta.Name == "" {
			continue
		}
		if current, ok := existingByName[meta.Name]; ok {
			displayName := current.DisplayName
			if strings.TrimSpace(meta.DisplayName) != "" {
				displayName = meta.DisplayName
			}
			capabilities := current.CapabilitiesJSON
			if strings.TrimSpace(meta.CapabilitiesJSON) != "" {
				capabilities = meta.CapabilitiesJSON
			}
			contextWindow := current.ContextWindow
			if meta.ContextWindow != nil {
				contextWindow = meta.ContextWindow
			}
			maxOutputTokens := current.MaxOutputTokens
			if meta.MaxOutputTokens != nil {
				maxOutputTokens = meta.MaxOutputTokens
			}
			supportsTools := current.SupportsTools
			if meta.SupportsTools != nil {
				supportsTools = meta.SupportsTools
			}
			supportsReasoning := current.SupportsReasoning
			if meta.SupportsReasoning != nil {
				supportsReasoning = meta.SupportsReasoning
			}
			supportsVision := current.SupportsVision
			if meta.SupportsVision != nil {
				supportsVision = meta.SupportsVision
			}
			supportsAudio := current.SupportsAudio
			if meta.SupportsAudio != nil {
				supportsAudio = meta.SupportsAudio
			}
			supportsVideo := current.SupportsVideo
			if meta.SupportsVideo != nil {
				supportsVideo = meta.SupportsVideo
			}
			updated, err := providers.NewModel(providers.ModelParams{
				ID:                current.ID,
				ProviderID:        current.ProviderID,
				Name:              current.Name,
				DisplayName:       displayName,
				CapabilitiesJSON:  capabilities,
				ContextWindow:     contextWindow,
				MaxOutputTokens:   maxOutputTokens,
				SupportsTools:     supportsTools,
				SupportsReasoning: supportsReasoning,
				SupportsVision:    supportsVision,
				SupportsAudio:     supportsAudio,
				SupportsVideo:     supportsVideo,
				Enabled:           current.Enabled,
				ShowInUI:          current.ShowInUI,
				CreatedAt:         &current.CreatedAt,
				UpdatedAt:         &now,
			})
			if err != nil {
				return nil, err
			}
			nextModels = append(nextModels, updated)
			continue
		}

		created, err := providers.NewModel(providers.ModelParams{
			ID:                meta.ID,
			ProviderID:        provider.ID,
			Name:              meta.Name,
			DisplayName:       meta.DisplayName,
			CapabilitiesJSON:  meta.CapabilitiesJSON,
			ContextWindow:     meta.ContextWindow,
			MaxOutputTokens:   meta.MaxOutputTokens,
			SupportsTools:     meta.SupportsTools,
			SupportsReasoning: meta.SupportsReasoning,
			SupportsVision:    meta.SupportsVision,
			SupportsAudio:     meta.SupportsAudio,
			SupportsVideo:     meta.SupportsVideo,
			Enabled:           false,
			ShowInUI:          false,
			UpdatedAt:         &now,
		})
		if err != nil {
			return nil, err
		}
		nextModels = append(nextModels, created)
	}

	// Keep manual models (IDs not generated by sync) to avoid losing
	// user-maintained entries when a provider does not expose `/models`.
	syncedPrefix := provider.ID + ":"
	for _, current := range existing {
		if _, ok := metadataByName[current.Name]; ok {
			continue
		}
		if strings.HasPrefix(current.ID, syncedPrefix) {
			continue
		}
		nextModels = append(nextModels, current)
	}

	if err := service.models.ReplaceByProvider(ctx, provider.ID, nextModels); err != nil {
		return nil, err
	}

	updatedModels, err := service.models.ListByProvider(ctx, provider.ID)
	if err != nil {
		return nil, err
	}
	result := make([]dto.ProviderModel, 0, len(updatedModels))
	for _, model := range updatedModels {
		result = append(result, dto.ProviderModel{
			ID:                  model.ID,
			ProviderID:          model.ProviderID,
			Name:                model.Name,
			DisplayName:         model.DisplayName,
			CapabilitiesJSON:    model.CapabilitiesJSON,
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

	return result, nil
}

func (service *ProvidersService) GetProviderSecret(ctx context.Context, providerID string) (dto.ProviderSecret, error) {
	if service.secrets == nil {
		return dto.ProviderSecret{}, fmt.Errorf("secret repository is not configured")
	}
	trimmed := strings.TrimSpace(providerID)
	if trimmed == "" {
		return dto.ProviderSecret{}, providers.ErrProviderNotFound
	}

	secret, err := service.secrets.GetByProviderID(ctx, trimmed)
	if err != nil {
		if err == providers.ErrProviderSecretNotFound {
			return dto.ProviderSecret{ProviderID: trimmed}, nil
		}
		return dto.ProviderSecret{}, err
	}

	return dto.ProviderSecret{
		ProviderID: secret.ProviderID,
		APIKey:     secret.APIKey,
		OrgRef:     secret.OrgRef,
	}, nil
}

func (service *ProvidersService) UpsertProviderSecret(ctx context.Context, request dto.UpsertProviderSecretRequest) error {
	if service.secrets == nil {
		return fmt.Errorf("secret repository is not configured")
	}
	trimmed := strings.TrimSpace(request.ProviderID)
	if trimmed == "" {
		return providers.ErrProviderNotFound
	}
	if _, err := service.providers.Get(ctx, trimmed); err != nil {
		return err
	}

	apiKey := strings.TrimSpace(request.APIKey)
	orgRef := strings.TrimSpace(request.OrgRef)
	if apiKey == "" && orgRef == "" {
		return service.secrets.DeleteByProviderID(ctx, trimmed)
	}

	now := service.now()
	secret, err := providers.NewProviderSecret(providers.ProviderSecretParams{
		ID:         trimmed,
		ProviderID: trimmed,
		APIKey:     apiKey,
		OrgRef:     orgRef,
		CreatedAt:  &now,
	})
	if err != nil {
		return err
	}

	return service.secrets.Save(ctx, secret)
}

func (service *ProvidersService) EnsureDefaults(ctx context.Context) error {
	defaults := defaultProviderRequests()

	for _, item := range defaults {
		if _, err := service.providers.Get(ctx, item.ID); err == nil {
			continue
		} else if err != providers.ErrProviderNotFound {
			return err
		}

		if _, err := service.UpsertProvider(ctx, item); err != nil {
			return err
		}
	}
	return nil
}

func defaultProviderRequests() []dto.UpsertProviderRequest {
	return []dto.UpsertProviderRequest{
		{ID: "deepseek", Name: "DeepSeek", Type: string(providers.ProviderTypeOpenAI), Endpoint: "https://api.deepseek.com", Enabled: false},
		{ID: "openrouter", Name: "OpenRouter", Type: string(providers.ProviderTypeOpenAI), Endpoint: "https://openrouter.ai/api/v1", Enabled: false},
		{ID: "openai", Name: "OpenAI", Type: string(providers.ProviderTypeOpenAI), Endpoint: "https://api.openai.com/v1", Enabled: false},
		{ID: "anthropic", Name: "Anthropic", Type: string(providers.ProviderTypeAnthropic), Endpoint: "https://api.anthropic.com/v1", Enabled: false},
		{ID: "google", Name: "Google Gemini", Type: string(providers.ProviderTypeOpenAI), Endpoint: "https://generativelanguage.googleapis.com/v1beta/openai", Enabled: false},
		{ID: "aihubmix", Name: "AIHubMix", Type: string(providers.ProviderTypeOpenAI), Endpoint: "https://aihubmix.com/v1", Enabled: false},
		{ID: "moonshotai", Name: "Moonshot AI", Type: string(providers.ProviderTypeOpenAI), Endpoint: "https://api.moonshot.ai/v1", Enabled: false},
		{ID: "zai", Name: "Z.AI", Type: string(providers.ProviderTypeOpenAI), Endpoint: "https://api.z.ai/api/paas/v4", Enabled: false},
		{ID: "github-copilot", Name: "GitHub Copilot", Type: string(providers.ProviderTypeOpenAI), Endpoint: "https://api.githubcopilot.com", Enabled: false},
	}
}

var defaultProviderIDs = func() map[string]struct{} {
	ids := make(map[string]struct{}, 11)
	for _, item := range defaultProviderRequests() {
		ids[item.ID] = struct{}{}
	}
	return ids
}()

func isDefaultProviderID(id string) bool {
	_, ok := defaultProviderIDs[id]
	return ok
}

func (service *ProvidersService) ReplaceProviderModels(ctx context.Context, request dto.ReplaceProviderModelsRequest) error {
	providerID := strings.TrimSpace(request.ProviderID)
	if providerID == "" {
		return providers.ErrProviderNotFound
	}

	displayNameByModelID := map[string]string{}
	capabilitiesByModelID := map[string]string{}
	if resolver, ok := service.syncer.(ModelDisplayNameResolver); ok {
		lookupModelIDs := make([]string, 0, len(request.Models))
		for _, item := range request.Models {
			if strings.TrimSpace(item.DisplayName) != "" && strings.TrimSpace(item.CapabilitiesJSON) != "" {
				continue
			}
			name := strings.TrimSpace(item.Name)
			if name == "" {
				continue
			}
			lookupModelIDs = append(lookupModelIDs, name)
		}
		if len(lookupModelIDs) > 0 {
			if resolved, err := resolver.ResolveModelDisplayNames(ctx, lookupModelIDs); err == nil {
				displayNameByModelID = resolved
			}
			if capabilitiesResolver, ok := service.syncer.(ModelCapabilitiesResolver); ok {
				if resolved, err := capabilitiesResolver.ResolveModelCapabilitiesJSON(ctx, lookupModelIDs); err == nil {
					capabilitiesByModelID = resolved
				}
			}
		}
	}

	now := service.now()
	models := make([]providers.Model, 0, len(request.Models))
	for _, model := range request.Models {
		id := strings.TrimSpace(model.ID)
		if id == "" {
			id = uuid.NewString()
		}
		displayName := strings.TrimSpace(model.DisplayName)
		if displayName == "" {
			displayName = strings.TrimSpace(displayNameByModelID[strings.ToLower(strings.TrimSpace(model.Name))])
		}
		capabilitiesJSON := strings.TrimSpace(model.CapabilitiesJSON)
		if capabilitiesJSON == "" {
			capabilitiesJSON = strings.TrimSpace(capabilitiesByModelID[strings.ToLower(strings.TrimSpace(model.Name))])
		}
		domainModel, err := providers.NewModel(providers.ModelParams{
			ID:                id,
			ProviderID:        providerID,
			Name:              model.Name,
			DisplayName:       displayName,
			CapabilitiesJSON:  capabilitiesJSON,
			ContextWindow:     model.ContextWindowTokens,
			MaxOutputTokens:   model.MaxOutputTokens,
			SupportsTools:     model.SupportsTools,
			SupportsReasoning: model.SupportsReasoning,
			SupportsVision:    model.SupportsVision,
			SupportsAudio:     model.SupportsAudio,
			SupportsVideo:     model.SupportsVideo,
			Enabled:           model.Enabled,
			ShowInUI:          model.ShowInUI,
			UpdatedAt:         &now,
		})
		if err != nil {
			return err
		}
		models = append(models, domainModel)
	}

	if err := service.models.ReplaceByProvider(ctx, providerID, models); err != nil {
		return err
	}
	return nil
}

func sanitizeProviderType(value string) (providers.ProviderType, error) {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	switch providers.ProviderType(trimmed) {
	case providers.ProviderTypeOpenAI,
		providers.ProviderTypeAnthropic:
		return providers.ProviderType(trimmed), nil
	default:
		return "", fmt.Errorf("unsupported provider type: %s", value)
	}
}

func (service *ProvidersService) resolveProviderIcon(ctx context.Context, provider providers.Provider) string {
	if service.logo == nil {
		return ""
	}
	key := strings.TrimSpace(provider.ID)
	if key == "" {
		key = strings.TrimSpace(provider.Name)
	}
	icon, err := service.logo.ResolveProviderLogo(ctx, key)
	if err != nil {
		return ""
	}
	return icon
}
