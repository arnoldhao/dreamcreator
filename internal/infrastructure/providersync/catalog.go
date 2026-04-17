package providersync

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"dreamcreator/internal/domain/providers"
)

type ModelsDevCatalogEntry struct {
	ID                string
	ProviderKey       string
	ModelName         string
	DisplayName       string
	CapabilitiesJSON  string
	ContextWindow     *int
	MaxOutputTokens   *int
	SupportsTools     *bool
	SupportsReasoning *bool
	SupportsVision    *bool
	SupportsAudio     *bool
	SupportsVideo     *bool
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type ModelsDevCatalogRepository interface {
	Count(ctx context.Context) (int, error)
	ReplaceAll(ctx context.Context, entries []ModelsDevCatalogEntry) error
	ListByProviderKeys(ctx context.Context, providerKeys []string) ([]ModelsDevCatalogEntry, error)
	ListByModelNames(ctx context.Context, modelNames []string) ([]ModelsDevCatalogEntry, error)
}

type ModelsDevCatalogService struct {
	repo   ModelsDevCatalogRepository
	syncer *ModelsDevSyncer
	now    func() time.Time
}

func NewModelsDevCatalogService(repo ModelsDevCatalogRepository, syncer *ModelsDevSyncer) *ModelsDevCatalogService {
	return &ModelsDevCatalogService{
		repo:   repo,
		syncer: syncer,
		now:    time.Now,
	}
}

func (service *ModelsDevCatalogService) HasEntries(ctx context.Context) (bool, error) {
	if service == nil || service.repo == nil {
		return false, nil
	}
	count, err := service.repo.Count(ctx)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (service *ModelsDevCatalogService) Refresh(ctx context.Context) (int, error) {
	if service == nil || service.repo == nil || service.syncer == nil {
		return 0, fmt.Errorf("models.dev catalog service unavailable")
	}
	entries, err := service.fetchEntries(ctx)
	if err != nil {
		return 0, err
	}
	if err := service.repo.ReplaceAll(ctx, entries); err != nil {
		return 0, err
	}
	return len(entries), nil
}

func (service *ModelsDevCatalogService) Sync(ctx context.Context, provider providers.Provider, _ string) ([]providers.Model, error) {
	if service == nil || service.repo == nil {
		return nil, fmt.Errorf("models.dev catalog unavailable")
	}
	candidates := buildProviderKeyCandidates(provider)
	if len(candidates) == 0 {
		return nil, fmt.Errorf("provider name is required")
	}
	entries, err := service.repo.ListByProviderKeys(ctx, candidates)
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("provider not found in local models.dev catalog: %s", strings.Join(candidates, ","))
	}
	byProvider := groupCatalogEntriesByProvider(entries)
	for _, candidate := range candidates {
		normalized := strings.ToLower(strings.TrimSpace(candidate))
		if normalized == "" {
			continue
		}
		matched := byProvider[normalized]
		if len(matched) == 0 {
			continue
		}
		modelProvider := provider
		if strings.TrimSpace(modelProvider.ID) == "" {
			modelProvider.ID = normalized
		}
		return catalogEntriesToModels(modelProvider, matched)
	}
	return nil, fmt.Errorf("provider not found in local models.dev catalog: %s", strings.Join(candidates, ","))
}

func (service *ModelsDevCatalogService) SyncByProviderKey(ctx context.Context, providerKey string) ([]providers.Model, error) {
	if service == nil || service.repo == nil {
		return nil, fmt.Errorf("models.dev catalog unavailable")
	}
	normalized := strings.ToLower(strings.TrimSpace(providerKey))
	if normalized == "" {
		return nil, fmt.Errorf("provider key is required")
	}
	entries, err := service.repo.ListByProviderKeys(ctx, []string{normalized})
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("provider not found in local models.dev catalog: %s", normalized)
	}
	return catalogEntriesToModels(providers.Provider{
		ID:   normalized,
		Name: normalized,
	}, entries)
}

func (service *ModelsDevCatalogService) ResolveProviderKeyByModelIDs(ctx context.Context, modelIDs []string) (string, error) {
	if service == nil || service.repo == nil {
		return "", fmt.Errorf("models.dev catalog unavailable")
	}
	targets := normalizeCatalogLookupValues(modelIDs)
	if len(targets) == 0 {
		return "", fmt.Errorf("model ids are required")
	}
	entries, err := service.repo.ListByModelNames(ctx, mapKeys(targets))
	if err != nil {
		return "", err
	}
	if len(entries) == 0 {
		return "", fmt.Errorf("provider not found in local models.dev catalog by model ids")
	}

	scores := make(map[string]int)
	for _, entry := range entries {
		providerKey := strings.ToLower(strings.TrimSpace(entry.ProviderKey))
		modelName := strings.ToLower(strings.TrimSpace(entry.ModelName))
		if providerKey == "" || modelName == "" {
			continue
		}
		if _, ok := targets[modelName]; !ok {
			continue
		}
		scores[providerKey] += 10
		if modelHasPositivePricing(json.RawMessage(entry.CapabilitiesJSON)) {
			scores[providerKey] += 5
		}
	}

	bestKey := ""
	bestScore := 0
	keys := make([]string, 0, len(scores))
	for key := range scores {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		score := scores[key]
		if score > bestScore {
			bestScore = score
			bestKey = key
			continue
		}
		if score > 0 && score == bestScore {
			if len(key) < len(bestKey) {
				bestKey = key
				continue
			}
			if len(key) == len(bestKey) && key < bestKey {
				bestKey = key
			}
		}
	}
	if bestScore <= 0 || bestKey == "" {
		return "", fmt.Errorf("provider not found in local models.dev catalog by model ids")
	}
	return bestKey, nil
}

func (service *ModelsDevCatalogService) ResolveModelDisplayNames(ctx context.Context, modelIDs []string) (map[string]string, error) {
	metadata, err := service.resolveModelMetadata(ctx, modelIDs)
	if err != nil {
		return nil, err
	}
	result := make(map[string]string, len(metadata))
	for key, item := range metadata {
		displayName := strings.TrimSpace(item.DisplayName)
		if displayName == "" {
			continue
		}
		result[key] = displayName
	}
	return result, nil
}

func (service *ModelsDevCatalogService) ResolveModelCapabilitiesJSON(ctx context.Context, modelIDs []string) (map[string]string, error) {
	metadata, err := service.resolveModelMetadata(ctx, modelIDs)
	if err != nil {
		return nil, err
	}
	result := make(map[string]string, len(metadata))
	for key, item := range metadata {
		capabilitiesJSON := strings.TrimSpace(item.CapabilitiesJSON)
		if capabilitiesJSON == "" {
			continue
		}
		result[key] = capabilitiesJSON
	}
	return result, nil
}

func (service *ModelsDevCatalogService) resolveModelMetadata(ctx context.Context, modelIDs []string) (map[string]modelsDevModelMetadata, error) {
	if service == nil || service.repo == nil {
		return map[string]modelsDevModelMetadata{}, nil
	}
	targets := normalizeCatalogLookupValues(modelIDs)
	if len(targets) == 0 {
		return map[string]modelsDevModelMetadata{}, nil
	}
	entries, err := service.repo.ListByModelNames(ctx, mapKeys(targets))
	if err != nil {
		return nil, err
	}
	result := make(map[string]modelsDevModelMetadata, len(targets))
	for _, entry := range entries {
		normalizedName := strings.ToLower(strings.TrimSpace(entry.ModelName))
		if normalizedName == "" {
			continue
		}
		if _, ok := targets[normalizedName]; !ok {
			continue
		}
		candidate := modelsDevModelMetadata{
			DisplayName:      strings.TrimSpace(entry.DisplayName),
			CapabilitiesJSON: strings.TrimSpace(entry.CapabilitiesJSON),
			Capabilities: modelCapabilities{
				ContextWindow:     cloneIntPointer(entry.ContextWindow),
				MaxOutputTokens:   cloneIntPointer(entry.MaxOutputTokens),
				SupportsTools:     cloneBoolPointer(entry.SupportsTools),
				SupportsReasoning: cloneBoolPointer(entry.SupportsReasoning),
				SupportsVision:    cloneBoolPointer(entry.SupportsVision),
				SupportsAudio:     cloneBoolPointer(entry.SupportsAudio),
				SupportsVideo:     cloneBoolPointer(entry.SupportsVideo),
			},
		}
		existing, found := result[normalizedName]
		if !found {
			result[normalizedName] = candidate
			continue
		}
		name := strings.TrimSpace(entry.ModelName)
		if strings.TrimSpace(existing.DisplayName) == "" || strings.EqualFold(existing.DisplayName, name) {
			if strings.TrimSpace(candidate.DisplayName) != "" && !strings.EqualFold(candidate.DisplayName, name) {
				result[normalizedName] = candidate
				continue
			}
		}
		if strings.TrimSpace(existing.CapabilitiesJSON) == "" && strings.TrimSpace(candidate.CapabilitiesJSON) != "" {
			result[normalizedName] = candidate
		}
	}
	return result, nil
}

func (service *ModelsDevCatalogService) fetchEntries(ctx context.Context) ([]ModelsDevCatalogEntry, error) {
	catalog, err := service.syncer.fetchCatalog(ctx)
	if err != nil {
		return nil, err
	}

	providerKeys := make([]string, 0, len(catalog))
	for key := range catalog {
		providerKeys = append(providerKeys, key)
	}
	sort.Strings(providerKeys)

	now := service.now()
	result := make([]ModelsDevCatalogEntry, 0)
	for _, providerKey := range providerKeys {
		rawProvider := catalog[providerKey]
		if len(rawProvider) == 0 {
			continue
		}
		var parsed modelsDevProvider
		if err := json.Unmarshal(rawProvider, &parsed); err != nil {
			continue
		}
		modelKeys := make([]string, 0, len(parsed.Models))
		for key := range parsed.Models {
			modelKeys = append(modelKeys, key)
		}
		sort.Strings(modelKeys)

		normalizedProviderKey := strings.ToLower(strings.TrimSpace(providerKey))
		for _, modelKey := range modelKeys {
			rawModel := parsed.Models[modelKey]
			if len(rawModel) == 0 {
				continue
			}
			name, displayName := parseModelsDevModel(rawModel, modelKey)
			capabilities := parseModelCapabilities(string(rawModel))
			entry := ModelsDevCatalogEntry{
				ID:                buildModelsDevCatalogID(normalizedProviderKey, name),
				ProviderKey:       normalizedProviderKey,
				ModelName:         strings.TrimSpace(name),
				DisplayName:       strings.TrimSpace(displayName),
				CapabilitiesJSON:  strings.TrimSpace(string(rawModel)),
				ContextWindow:     capabilities.ContextWindow,
				MaxOutputTokens:   capabilities.MaxOutputTokens,
				SupportsTools:     capabilities.SupportsTools,
				SupportsReasoning: capabilities.SupportsReasoning,
				SupportsVision:    capabilities.SupportsVision,
				SupportsAudio:     capabilities.SupportsAudio,
				SupportsVideo:     capabilities.SupportsVideo,
				CreatedAt:         now,
				UpdatedAt:         now,
			}
			result = append(result, entry)
		}
	}
	return result, nil
}

func catalogEntriesToModels(provider providers.Provider, entries []ModelsDevCatalogEntry) ([]providers.Model, error) {
	sort.Slice(entries, func(i, j int) bool {
		return strings.ToLower(strings.TrimSpace(entries[i].ModelName)) < strings.ToLower(strings.TrimSpace(entries[j].ModelName))
	})
	result := make([]providers.Model, 0, len(entries))
	for _, entry := range entries {
		model, err := providers.NewModel(providers.ModelParams{
			ID:                buildModelID(provider.ID, entry.ModelName),
			ProviderID:        provider.ID,
			Name:              entry.ModelName,
			DisplayName:       entry.DisplayName,
			CapabilitiesJSON:  entry.CapabilitiesJSON,
			ContextWindow:     cloneIntPointer(entry.ContextWindow),
			MaxOutputTokens:   cloneIntPointer(entry.MaxOutputTokens),
			SupportsTools:     cloneBoolPointer(entry.SupportsTools),
			SupportsReasoning: cloneBoolPointer(entry.SupportsReasoning),
			SupportsVision:    cloneBoolPointer(entry.SupportsVision),
			SupportsAudio:     cloneBoolPointer(entry.SupportsAudio),
			SupportsVideo:     cloneBoolPointer(entry.SupportsVideo),
			Enabled:           false,
			ShowInUI:          false,
			UpdatedAt:         &entry.UpdatedAt,
		})
		if err != nil {
			return nil, err
		}
		result = append(result, model)
	}
	return result, nil
}

func groupCatalogEntriesByProvider(entries []ModelsDevCatalogEntry) map[string][]ModelsDevCatalogEntry {
	result := make(map[string][]ModelsDevCatalogEntry)
	for _, entry := range entries {
		key := strings.ToLower(strings.TrimSpace(entry.ProviderKey))
		if key == "" {
			continue
		}
		result[key] = append(result[key], entry)
	}
	return result
}

func buildModelsDevCatalogID(providerKey string, modelName string) string {
	return strings.ToLower(strings.TrimSpace(providerKey)) + ":" + strings.ToLower(strings.TrimSpace(modelName))
}

func normalizeCatalogLookupValues(values []string) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		normalized := strings.ToLower(strings.TrimSpace(value))
		if normalized == "" {
			continue
		}
		result[normalized] = struct{}{}
	}
	return result
}

func mapKeys(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for key := range values {
		result = append(result, key)
	}
	sort.Strings(result)
	return result
}

func cloneIntPointer(value *int) *int {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneBoolPointer(value *bool) *bool {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
