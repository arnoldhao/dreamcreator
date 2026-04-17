package providersync

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"dreamcreator/internal/domain/providers"
)

const defaultAnthropicVersion = "2023-06-01"

type EndpointModelsSyncer struct {
	httpClient *http.Client
	now        func() time.Time
	modelsDev  modelsDevMetadataSyncer
}

type modelsDevMetadataSyncer interface {
	Sync(ctx context.Context, provider providers.Provider, apiKey string) ([]providers.Model, error)
	ResolveModelDisplayNames(ctx context.Context, modelIDs []string) (map[string]string, error)
	ResolveModelCapabilitiesJSON(ctx context.Context, modelIDs []string) (map[string]string, error)
}

func NewEndpointModelsSyncer(modelsDev modelsDevMetadataSyncer) *EndpointModelsSyncer {
	return &EndpointModelsSyncer{
		httpClient: &http.Client{Timeout: 20 * time.Second},
		now:        time.Now,
		modelsDev:  modelsDev,
	}
}

func (syncer *EndpointModelsSyncer) ResolveModelDisplayNames(ctx context.Context, modelIDs []string) (map[string]string, error) {
	result := make(map[string]string)
	if syncer.modelsDev == nil || len(modelIDs) == 0 {
		return result, nil
	}
	displayNames, err := syncer.modelsDev.ResolveModelDisplayNames(ctx, modelIDs)
	if err != nil {
		return nil, err
	}
	for key, displayName := range displayNames {
		displayName = strings.TrimSpace(displayName)
		if displayName == "" {
			continue
		}
		result[key] = displayName
	}
	return result, nil
}

func (syncer *EndpointModelsSyncer) ResolveModelCapabilitiesJSON(ctx context.Context, modelIDs []string) (map[string]string, error) {
	result := make(map[string]string)
	if syncer.modelsDev == nil || len(modelIDs) == 0 {
		return result, nil
	}
	capabilitiesByModelID, err := syncer.modelsDev.ResolveModelCapabilitiesJSON(ctx, modelIDs)
	if err != nil {
		return nil, err
	}
	for key, capabilities := range capabilitiesByModelID {
		capabilities = strings.TrimSpace(capabilities)
		if capabilities == "" {
			continue
		}
		result[key] = capabilities
	}
	return result, nil
}

func (syncer *EndpointModelsSyncer) Sync(ctx context.Context, provider providers.Provider, apiKey string) ([]providers.Model, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(provider.Endpoint), "/")
	if baseURL == "" {
		return nil, fmt.Errorf("provider endpoint is required")
	}

	items, err := syncer.fetchModels(ctx, provider, apiKey, baseURL+"/models")
	if err != nil {
		return nil, err
	}

	metaByName := make(map[string]providers.Model)
	if syncer.modelsDev != nil {
		if metaModels, err := syncer.modelsDev.Sync(ctx, provider, ""); err == nil {
			for _, model := range metaModels {
				if model.Name == "" {
					continue
				}
				metaByName[model.Name] = model
			}
		}
	}

	modelNames := make([]string, 0, len(items))
	for _, item := range items {
		name := strings.TrimSpace(item.ID)
		if name == "" {
			name = strings.TrimSpace(item.Name)
		}
		if name == "" {
			continue
		}
		modelNames = append(modelNames, name)
	}
	displayNameByModelID := make(map[string]string)
	capabilitiesByModelID := make(map[string]string)
	if syncer.modelsDev != nil {
		if displayNames, err := syncer.modelsDev.ResolveModelDisplayNames(ctx, modelNames); err == nil {
			displayNameByModelID = displayNames
		}
		if capabilities, err := syncer.modelsDev.ResolveModelCapabilitiesJSON(ctx, modelNames); err == nil {
			capabilitiesByModelID = capabilities
		}
	}

	now := syncer.now()
	result := make([]providers.Model, 0, len(items))
	for _, item := range items {
		name := strings.TrimSpace(item.ID)
		if name == "" {
			name = strings.TrimSpace(item.Name)
		}
		if name == "" {
			continue
		}
		displayName := strings.TrimSpace(item.Name)
		capabilities := strings.TrimSpace(item.Raw)
		providerCaps := parseModelCapabilities(item.Raw)
		metaCaps := modelCapabilities{}
		if meta, ok := metaByName[name]; ok {
			if strings.TrimSpace(meta.DisplayName) != "" {
				displayName = meta.DisplayName
			}
			if strings.TrimSpace(meta.CapabilitiesJSON) != "" {
				capabilities = meta.CapabilitiesJSON
			}
			metaCaps = modelCapabilities{
				ContextWindow:     meta.ContextWindow,
				MaxOutputTokens:   meta.MaxOutputTokens,
				SupportsTools:     meta.SupportsTools,
				SupportsReasoning: meta.SupportsReasoning,
				SupportsVision:    meta.SupportsVision,
				SupportsAudio:     meta.SupportsAudio,
				SupportsVideo:     meta.SupportsVideo,
			}
		}
		normalizedName := strings.ToLower(strings.TrimSpace(name))
		if resolvedDisplayName := strings.TrimSpace(displayNameByModelID[normalizedName]); resolvedDisplayName != "" {
			displayName = resolvedDisplayName
		}
		if resolvedCapabilities := strings.TrimSpace(capabilitiesByModelID[normalizedName]); resolvedCapabilities != "" {
			capabilities = resolvedCapabilities
			metaCaps = mergeModelCapabilities(metaCaps, parseModelCapabilities(resolvedCapabilities))
		}
		if displayName == "" {
			displayName = name
		}
		mergedCaps := mergeModelCapabilities(metaCaps, providerCaps)

		model, err := providers.NewModel(providers.ModelParams{
			ID:                buildModelID(provider.ID, name),
			ProviderID:        provider.ID,
			Name:              name,
			DisplayName:       displayName,
			CapabilitiesJSON:  capabilities,
			ContextWindow:     mergedCaps.ContextWindow,
			MaxOutputTokens:   mergedCaps.MaxOutputTokens,
			SupportsTools:     mergedCaps.SupportsTools,
			SupportsReasoning: mergedCaps.SupportsReasoning,
			SupportsVision:    mergedCaps.SupportsVision,
			SupportsAudio:     mergedCaps.SupportsAudio,
			SupportsVideo:     mergedCaps.SupportsVideo,
			Enabled:           false,
			ShowInUI:          false,
			UpdatedAt:         &now,
		})
		if err != nil {
			return nil, err
		}
		result = append(result, model)
	}

	return result, nil
}

type modelListItem struct {
	ID   string
	Name string
	Raw  string
}

type modelListWrapper struct {
	Data   []json.RawMessage `json:"data"`
	Models []json.RawMessage `json:"models"`
}

type modelListEntry struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (syncer *EndpointModelsSyncer) fetchModels(ctx context.Context, provider providers.Provider, apiKey string, modelsURL string) ([]modelListItem, error) {
	client := syncer.httpClient
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, modelsURL, nil)
	if err != nil {
		return nil, err
	}
	applyAuthHeaders(request, provider, apiKey)

	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("models request failed: %s", strings.TrimSpace(string(body)))
	}

	items, err := parseModelList(body)
	if err != nil {
		return nil, err
	}
	return items, nil
}

func applyAuthHeaders(request *http.Request, provider providers.Provider, apiKey string) {
	request.Header.Set("Accept", "application/json")
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return
	}
	if provider.Type == providers.ProviderTypeAnthropic {
		request.Header.Set("x-api-key", apiKey)
		request.Header.Set("anthropic-version", defaultAnthropicVersion)
		return
	}
	request.Header.Set("Authorization", "Bearer "+apiKey)
}

func parseModelList(body []byte) ([]modelListItem, error) {
	var wrapper modelListWrapper
	if err := json.Unmarshal(body, &wrapper); err == nil {
		items := wrapper.Data
		if len(items) == 0 {
			items = wrapper.Models
		}
		if len(items) == 0 {
			return []modelListItem{}, nil
		}
		return buildModelList(items), nil
	}

	var rawArray []json.RawMessage
	if err := json.Unmarshal(body, &rawArray); err == nil {
		if len(rawArray) == 0 {
			return []modelListItem{}, nil
		}
		return buildModelList(rawArray), nil
	}

	return nil, fmt.Errorf("unsupported models response")
}

func buildModelList(items []json.RawMessage) []modelListItem {
	result := make([]modelListItem, 0, len(items))
	for _, raw := range items {
		var entry modelListEntry
		if err := json.Unmarshal(raw, &entry); err != nil {
			continue
		}
		result = append(result, modelListItem{
			ID:   entry.ID,
			Name: entry.Name,
			Raw:  strings.TrimSpace(string(raw)),
		})
	}
	return result
}
