package runtime

import (
	"context"
	"encoding/json"
	"math"
	"strconv"
	"strings"

	"dreamcreator/internal/application/gateway/runtime/dto"
	"dreamcreator/internal/domain/providers"
)

type modelTokenPricing struct {
	PromptUSDPerToken     float64
	CompletionUSDPerToken float64
	RequestUSD            float64
}

var (
	promptCostPerTokenPaths = []string{
		"pricing.prompt",
		"pricing.input",
		"pricing.input_text",
		"pricing.input_price",
		"pricing.prompt_price",
		"input_cost_per_token",
		"input_price",
		"prompt_cost_per_token",
		"prompt_price",
	}
	promptCostPerMillionPaths = []string{
		"pricing.prompt_per_million",
		"pricing.input_per_million",
		"pricing.input_price_per_million",
		"pricing.prompt_price_per_million",
		"cost.input",
		"cost.prompt",
		"cost.input_text",
		"cost.prompt_text",
		"cost.input_per_million",
		"cost.prompt_per_million",
		"input_cost_per_million",
		"input_price_per_million",
		"prompt_cost_per_million",
		"prompt_price_per_million",
	}
	promptCostPer1KPaths = []string{
		"pricing.prompt_per_1k",
		"pricing.input_per_1k",
		"pricing.input_price_per_1k",
		"pricing.prompt_price_per_1k",
		"input_cost_per_1k",
		"input_price_per_1k",
		"prompt_cost_per_1k",
		"prompt_price_per_1k",
	}

	completionCostPerTokenPaths = []string{
		"pricing.completion",
		"pricing.output",
		"pricing.output_text",
		"pricing.output_price",
		"pricing.completion_price",
		"output_cost_per_token",
		"output_price",
		"completion_cost_per_token",
		"completion_price",
	}
	completionCostPerMillionPaths = []string{
		"pricing.completion_per_million",
		"pricing.output_per_million",
		"pricing.output_price_per_million",
		"pricing.completion_price_per_million",
		"cost.output",
		"cost.completion",
		"cost.output_text",
		"cost.completion_text",
		"cost.output_per_million",
		"cost.completion_per_million",
		"output_cost_per_million",
		"output_price_per_million",
		"completion_cost_per_million",
		"completion_price_per_million",
	}
	completionCostPer1KPaths = []string{
		"pricing.completion_per_1k",
		"pricing.output_per_1k",
		"pricing.output_price_per_1k",
		"pricing.completion_price_per_1k",
		"output_cost_per_1k",
		"output_price_per_1k",
		"completion_cost_per_1k",
		"completion_price_per_1k",
	}

	requestCostPaths = []string{
		"pricing.request",
		"request_cost",
		"cost_per_request",
		"pricing.per_request",
	}
)

func (service *Service) estimateUsageCostMicros(ctx context.Context, usage dto.RuntimeUsage, model resolvedRunModel) int64 {
	if service == nil {
		return 0
	}
	if usage.TotalTokens <= 0 && usage.PromptTokens <= 0 && usage.CompletionTokens <= 0 {
		return 0
	}
	pricing, ok := service.resolveModelTokenPricing(ctx, model.ProviderID, model.ModelName)
	if !ok {
		return 0
	}
	return calculateUsageCostMicros(usage, pricing)
}

func (service *Service) resolveModelTokenPricing(ctx context.Context, providerID string, modelName string) (modelTokenPricing, bool) {
	if service == nil || service.models == nil {
		return modelTokenPricing{}, false
	}
	providerID = strings.TrimSpace(providerID)
	modelName = strings.TrimSpace(modelName)
	if providerID == "" || modelName == "" {
		return modelTokenPricing{}, false
	}
	models, err := service.models.ListByProvider(ctx, providerID)
	if err != nil {
		return modelTokenPricing{}, false
	}
	model, ok := findModelByName(models, modelName)
	if !ok {
		return modelTokenPricing{}, false
	}
	return parseModelTokenPricing(model.CapabilitiesJSON)
}

func findModelByName(models []providers.Model, target string) (providers.Model, bool) {
	normalizedTarget := strings.ToLower(strings.TrimSpace(target))
	if normalizedTarget == "" {
		return providers.Model{}, false
	}
	for _, model := range models {
		if strings.EqualFold(strings.TrimSpace(model.Name), normalizedTarget) {
			return model, true
		}
	}
	targetWithoutProvider := trimModelProviderPrefix(normalizedTarget)
	if targetWithoutProvider == "" {
		return providers.Model{}, false
	}
	for _, model := range models {
		if trimModelProviderPrefix(strings.ToLower(strings.TrimSpace(model.Name))) == targetWithoutProvider {
			return model, true
		}
	}
	return providers.Model{}, false
}

func trimModelProviderPrefix(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	separator := strings.Index(trimmed, "/")
	if separator <= 0 || separator >= len(trimmed)-1 {
		return trimmed
	}
	return strings.TrimSpace(trimmed[separator+1:])
}

func parseModelTokenPricing(raw string) (modelTokenPricing, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return modelTokenPricing{}, false
	}
	var payload any
	if err := json.Unmarshal([]byte(trimmed), &payload); err != nil {
		return modelTokenPricing{}, false
	}

	pricing := modelTokenPricing{}
	hasPricing := false

	if value, ok := extractUSDPerToken(payload, promptCostPerTokenPaths, promptCostPerMillionPaths, promptCostPer1KPaths); ok {
		pricing.PromptUSDPerToken = value
		hasPricing = true
	}
	if value, ok := extractUSDPerToken(payload, completionCostPerTokenPaths, completionCostPerMillionPaths, completionCostPer1KPaths); ok {
		pricing.CompletionUSDPerToken = value
		hasPricing = true
	}
	if value, ok := extractUSDAmount(payload, requestCostPaths); ok {
		pricing.RequestUSD = value
		hasPricing = true
	}

	return pricing, hasPricing
}

func extractUSDPerToken(payload any, perTokenPaths []string, perMillionPaths []string, per1KPaths []string) (float64, bool) {
	if value, ok := extractUSDAmount(payload, perTokenPaths); ok {
		return value, true
	}
	if value, ok := extractUSDAmount(payload, perMillionPaths); ok {
		return value / 1_000_000, true
	}
	if value, ok := extractUSDAmount(payload, per1KPaths); ok {
		return value / 1_000, true
	}
	return 0, false
}

func extractUSDAmount(payload any, paths []string) (float64, bool) {
	for _, path := range paths {
		value, ok := lookupValueByPath(payload, path)
		if !ok {
			continue
		}
		parsed, parsedOK := parseDecimal(value)
		if !parsedOK {
			continue
		}
		if parsed < 0 {
			parsed = 0
		}
		return parsed, true
	}
	return 0, false
}

func parseDecimal(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		if math.IsNaN(typed) || math.IsInf(typed, 0) {
			return 0, false
		}
		return typed, true
	case float32:
		parsed := float64(typed)
		if math.IsNaN(parsed) || math.IsInf(parsed, 0) {
			return 0, false
		}
		return parsed, true
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
		if math.IsNaN(parsed) || math.IsInf(parsed, 0) {
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
		if math.IsNaN(parsed) || math.IsInf(parsed, 0) {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

func calculateUsageCostMicros(usage dto.RuntimeUsage, pricing modelTokenPricing) int64 {
	totalUSD := 0.0
	promptRate := pricing.PromptUSDPerToken
	completionRate := pricing.CompletionUSDPerToken
	if promptRate <= 0 {
		promptRate = completionRate
	}
	if completionRate <= 0 {
		completionRate = promptRate
	}

	if usage.PromptTokens > 0 || usage.CompletionTokens > 0 {
		if usage.PromptTokens > 0 && promptRate > 0 {
			totalUSD += float64(usage.PromptTokens) * promptRate
		}
		if usage.CompletionTokens > 0 && completionRate > 0 {
			totalUSD += float64(usage.CompletionTokens) * completionRate
		}
		accounted := usage.PromptTokens + usage.CompletionTokens
		if usage.TotalTokens > accounted {
			remaining := usage.TotalTokens - accounted
			if promptRate > 0 {
				totalUSD += float64(remaining) * promptRate
			}
		}
	} else if usage.TotalTokens > 0 {
		if promptRate > 0 {
			totalUSD += float64(usage.TotalTokens) * promptRate
		}
	}

	if (usage.TotalTokens > 0 || usage.PromptTokens > 0 || usage.CompletionTokens > 0) && pricing.RequestUSD > 0 {
		totalUSD += pricing.RequestUSD
	}
	if totalUSD <= 0 {
		return 0
	}
	return int64(math.Round(totalUSD * 1_000_000))
}
