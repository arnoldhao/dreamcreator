package providersync

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"dreamcreator/internal/domain/providers"
)

const modelsDevAPIURL = "https://models.dev/api.json"

type ModelsDevSyncer struct {
	apiURL     string
	httpClient *http.Client
	now        func() time.Time
}

func NewModelsDevSyncer() *ModelsDevSyncer {
	return &ModelsDevSyncer{
		apiURL:     modelsDevAPIURL,
		httpClient: &http.Client{Timeout: 20 * time.Second},
		now:        time.Now,
	}
}

func (syncer *ModelsDevSyncer) Sync(ctx context.Context, provider providers.Provider, _ string) ([]providers.Model, error) {
	candidates := buildProviderKeyCandidates(provider)
	if len(candidates) == 0 {
		return nil, fmt.Errorf("provider name is required")
	}

	providerKey, rawProvider, err := syncer.fetchProviderByCandidates(ctx, candidates)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(provider.ID) == "" {
		provider.ID = providerKey
	}
	return syncer.buildModels(provider, rawProvider)
}

func (syncer *ModelsDevSyncer) SyncByProviderKey(ctx context.Context, providerKey string) ([]providers.Model, error) {
	normalized := strings.ToLower(strings.TrimSpace(providerKey))
	if normalized == "" {
		return nil, fmt.Errorf("provider key is required")
	}
	rawProvider, err := syncer.fetchProvider(ctx, normalized)
	if err != nil {
		return nil, err
	}
	provider := providers.Provider{
		ID:   normalized,
		Name: normalized,
	}
	return syncer.buildModels(provider, rawProvider)
}

func (syncer *ModelsDevSyncer) ResolveProviderKeyByModelIDs(ctx context.Context, modelIDs []string) (string, error) {
	targets := make(map[string]struct{}, len(modelIDs))
	for _, item := range modelIDs {
		normalized := strings.ToLower(strings.TrimSpace(item))
		if normalized == "" {
			continue
		}
		targets[normalized] = struct{}{}
	}
	if len(targets) == 0 {
		return "", fmt.Errorf("model ids are required")
	}

	catalog, err := syncer.fetchCatalog(ctx)
	if err != nil {
		return "", err
	}

	keys := make([]string, 0, len(catalog))
	for key := range catalog {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	bestKey := ""
	bestScore := 0
	for _, key := range keys {
		rawProvider := catalog[key]
		if len(rawProvider) == 0 {
			continue
		}
		var parsed modelsDevProvider
		if err := json.Unmarshal(rawProvider, &parsed); err != nil {
			continue
		}
		score := 0
		for modelKey, rawModel := range parsed.Models {
			name, _ := parseModelsDevModel(rawModel, modelKey)
			normalized := strings.ToLower(strings.TrimSpace(name))
			if normalized == "" {
				continue
			}
			if _, ok := targets[normalized]; ok {
				score += 10
				if modelHasPositivePricing(rawModel) {
					score += 5
				}
			}
		}
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
		return "", fmt.Errorf("provider not found in models.dev by model ids")
	}
	return bestKey, nil
}

func (syncer *ModelsDevSyncer) fetchProviderByCandidates(ctx context.Context, candidates []string) (string, json.RawMessage, error) {
	for _, candidate := range candidates {
		normalized := strings.ToLower(strings.TrimSpace(candidate))
		if normalized == "" {
			continue
		}
		rawProvider, err := syncer.fetchProvider(ctx, normalized)
		if err != nil {
			continue
		}
		return normalized, rawProvider, nil
	}
	return "", nil, fmt.Errorf("provider not found in models.dev: %s", strings.Join(candidates, ","))
}

func (syncer *ModelsDevSyncer) buildModels(provider providers.Provider, rawProvider json.RawMessage) ([]providers.Model, error) {

	var parsed modelsDevProvider
	if err := json.Unmarshal(rawProvider, &parsed); err != nil {
		return nil, fmt.Errorf("decode models.dev provider failed: %w", err)
	}
	if len(parsed.Models) == 0 {
		return nil, nil
	}

	keys := make([]string, 0, len(parsed.Models))
	for key := range parsed.Models {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	now := syncer.now()
	result := make([]providers.Model, 0, len(keys))
	for _, key := range keys {
		rawModel := parsed.Models[key]
		if len(rawModel) == 0 {
			continue
		}
		name, displayName := parseModelsDevModel(rawModel, key)
		capabilities := parseModelCapabilities(string(rawModel))
		model, err := providers.NewModel(providers.ModelParams{
			ID:                buildModelID(provider.ID, name),
			ProviderID:        provider.ID,
			Name:              name,
			DisplayName:       displayName,
			CapabilitiesJSON:  strings.TrimSpace(string(rawModel)),
			ContextWindow:     capabilities.ContextWindow,
			MaxOutputTokens:   capabilities.MaxOutputTokens,
			SupportsTools:     capabilities.SupportsTools,
			SupportsReasoning: capabilities.SupportsReasoning,
			SupportsVision:    capabilities.SupportsVision,
			SupportsAudio:     capabilities.SupportsAudio,
			SupportsVideo:     capabilities.SupportsVideo,
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

func buildProviderKeyCandidates(provider providers.Provider) []string {
	result := make([]string, 0, 8)
	seen := make(map[string]struct{}, 8)
	add := func(value string) {
		normalized := strings.ToLower(strings.TrimSpace(value))
		if normalized == "" {
			return
		}
		if _, ok := seen[normalized]; ok {
			return
		}
		seen[normalized] = struct{}{}
		result = append(result, normalized)
	}

	add(provider.ID)
	add(provider.Name)

	endpoint := strings.TrimSpace(provider.Endpoint)
	if endpoint != "" {
		if parsed, err := url.Parse(endpoint); err == nil {
			host := strings.ToLower(strings.TrimSpace(parsed.Hostname()))
			add(host)
			parts := strings.Split(host, ".")
			if len(parts) >= 2 {
				add(parts[len(parts)-2])
				add(strings.Join(parts[len(parts)-2:], "."))
			}
		}
	}

	return result
}

var positivePricingPaths = []string{
	"cost.input",
	"cost.output",
	"cost.cache_read",
	"cost.cache_write",
	"cost.reasoning",
	"cost.input_audio",
	"cost.output_audio",
	"pricing.input",
	"pricing.output",
	"pricing.prompt",
	"pricing.completion",
	"pricing.request",
	"pricing.per_request",
	"input_cost_per_million",
	"output_cost_per_million",
}

func modelHasPositivePricing(raw json.RawMessage) bool {
	if len(raw) == 0 {
		return false
	}
	var payload any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return false
	}
	for _, path := range positivePricingPaths {
		value, ok := lookupPathValue(payload, path)
		if !ok {
			continue
		}
		parsed, parseOK := parsePricingNumber(value)
		if !parseOK {
			continue
		}
		if parsed > 0 {
			return true
		}
	}
	return false
}

func lookupPathValue(payload any, path string) (any, bool) {
	if path == "" {
		return nil, false
	}
	current := payload
	for _, segment := range strings.Split(path, ".") {
		if segment == "" {
			return nil, false
		}
		object, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		next, ok := object[segment]
		if !ok {
			return nil, false
		}
		current = next
	}
	return current, true
}

func parsePricingNumber(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int8:
		return float64(typed), true
	case int16:
		return float64(typed), true
	case int32:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case uint:
		return float64(typed), true
	case uint8:
		return float64(typed), true
	case uint16:
		return float64(typed), true
	case uint32:
		return float64(typed), true
	case uint64:
		return float64(typed), true
	case json.Number:
		parsed, err := typed.Float64()
		if err != nil {
			return 0, false
		}
		return parsed, true
	case string:
		trimmed := strings.TrimSpace(strings.TrimPrefix(typed, "$"))
		trimmed = strings.ReplaceAll(trimmed, ",", "")
		if trimmed == "" {
			return 0, false
		}
		parsed, err := strconv.ParseFloat(trimmed, 64)
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

func (syncer *ModelsDevSyncer) fetchProvider(ctx context.Context, providerKey string) (json.RawMessage, error) {
	providersMap, err := syncer.fetchCatalog(ctx)
	if err != nil {
		return nil, err
	}
	raw, ok := providersMap[providerKey]
	if !ok {
		return nil, fmt.Errorf("provider not found in models.dev: %s", providerKey)
	}
	return raw, nil
}

func (syncer *ModelsDevSyncer) fetchCatalog(ctx context.Context) (map[string]json.RawMessage, error) {
	client := syncer.httpClient
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}
	apiURL := strings.TrimSpace(syncer.apiURL)
	if apiURL == "" {
		apiURL = modelsDevAPIURL
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusBadRequest {
		body, _ := io.ReadAll(response.Body)
		return nil, fmt.Errorf("models.dev request failed: %s", strings.TrimSpace(string(body)))
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	providersMap := make(map[string]json.RawMessage)
	if err := json.Unmarshal(body, &providersMap); err != nil {
		return nil, fmt.Errorf("decode models.dev response failed: %w", err)
	}
	return providersMap, nil
}

type modelsDevProvider struct {
	Models map[string]json.RawMessage `json:"models"`
}

type modelsDevModel struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type modelsDevModelMetadata struct {
	DisplayName      string
	CapabilitiesJSON string
	Capabilities     modelCapabilities
}

func (syncer *ModelsDevSyncer) ResolveModelMetadata(ctx context.Context, modelIDs []string) (map[string]modelsDevModelMetadata, error) {
	targetIDs := make(map[string]struct{}, len(modelIDs))
	for _, item := range modelIDs {
		normalized := strings.ToLower(strings.TrimSpace(item))
		if normalized == "" {
			continue
		}
		targetIDs[normalized] = struct{}{}
	}
	if len(targetIDs) == 0 {
		return map[string]modelsDevModelMetadata{}, nil
	}

	catalog, err := syncer.fetchCatalog(ctx)
	if err != nil {
		return nil, err
	}

	result := make(map[string]modelsDevModelMetadata, len(targetIDs))
	for _, rawProvider := range catalog {
		if len(rawProvider) == 0 {
			continue
		}
		var parsedProvider modelsDevProvider
		if err := json.Unmarshal(rawProvider, &parsedProvider); err != nil {
			continue
		}
		for key, rawModel := range parsedProvider.Models {
			if len(rawModel) == 0 {
				continue
			}
			name, displayName := parseModelsDevModel(rawModel, key)
			normalizedName := strings.ToLower(strings.TrimSpace(name))
			if normalizedName == "" {
				continue
			}
			if _, wanted := targetIDs[normalizedName]; !wanted {
				continue
			}
			candidate := modelsDevModelMetadata{
				DisplayName:      strings.TrimSpace(displayName),
				CapabilitiesJSON: strings.TrimSpace(string(rawModel)),
				Capabilities:     parseModelCapabilities(string(rawModel)),
			}
			existing, found := result[normalizedName]
			if !found {
				result[normalizedName] = candidate
				continue
			}
			// Prefer entries that provide a more descriptive display name.
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
	}

	return result, nil
}

func parseModelsDevModel(raw json.RawMessage, fallback string) (string, string) {
	var parsed modelsDevModel
	if err := json.Unmarshal(raw, &parsed); err != nil {
		trimmed := strings.TrimSpace(fallback)
		return trimmed, trimmed
	}
	name := strings.TrimSpace(parsed.ID)
	if name == "" {
		name = strings.TrimSpace(fallback)
	}
	displayName := strings.TrimSpace(parsed.Name)
	if displayName == "" {
		displayName = name
	}
	return name, displayName
}
