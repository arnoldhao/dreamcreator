package usage

import (
	"encoding/json"
	"math"
	"strconv"
	"strings"
)

type ParsedPricing struct {
	InputPerMillion       float64
	OutputPerMillion      float64
	CachedInputPerMillion float64
	ReasoningPerMillion   float64
	AudioInputPerMillion  float64
	AudioOutputPerMillion float64
	PerRequest            float64
}

var (
	inputPerTokenPaths = []string{
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
	inputPerMillionPaths = []string{
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
	inputPer1KPaths = []string{
		"pricing.prompt_per_1k",
		"pricing.input_per_1k",
		"pricing.input_price_per_1k",
		"pricing.prompt_price_per_1k",
		"input_cost_per_1k",
		"input_price_per_1k",
		"prompt_cost_per_1k",
		"prompt_price_per_1k",
	}

	outputPerTokenPaths = []string{
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
	outputPerMillionPaths = []string{
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
	outputPer1KPaths = []string{
		"pricing.completion_per_1k",
		"pricing.output_per_1k",
		"pricing.output_price_per_1k",
		"pricing.completion_price_per_1k",
		"output_cost_per_1k",
		"output_price_per_1k",
		"completion_cost_per_1k",
		"completion_price_per_1k",
	}

	cachedPerTokenPaths = []string{
		"pricing.cached_input",
		"pricing.cache_read",
		"cached_input_cost_per_token",
		"cache_read_cost_per_token",
	}
	cachedPerMillionPaths = []string{
		"pricing.cached_input_per_million",
		"pricing.cache_read_per_million",
		"cost.cached_input",
		"cost.cache_input",
		"cost.cache_read",
		"cost.cached_input_per_million",
		"cost.cache_input_per_million",
		"cost.cache_read_per_million",
		"cached_input_cost_per_million",
		"cache_read_cost_per_million",
	}
	cachedPer1KPaths = []string{
		"pricing.cached_input_per_1k",
		"pricing.cache_read_per_1k",
		"cost.cached_input_per_1k",
		"cost.cache_input_per_1k",
		"cost.cache_read_per_1k",
		"cached_input_cost_per_1k",
		"cache_read_cost_per_1k",
	}

	reasoningPerTokenPaths = []string{
		"pricing.reasoning",
		"reasoning_cost_per_token",
		"reasoning_price_per_token",
	}
	reasoningPerMillionPaths = []string{
		"pricing.reasoning_per_million",
		"cost.reasoning",
		"cost.reasoning_per_million",
		"reasoning_cost_per_million",
		"reasoning_price_per_million",
	}
	reasoningPer1KPaths = []string{
		"pricing.reasoning_per_1k",
		"cost.reasoning_per_1k",
		"reasoning_cost_per_1k",
		"reasoning_price_per_1k",
	}

	audioInputPerTokenPaths = []string{
		"pricing.audio_input",
		"pricing.input_audio",
		"audio_input_cost_per_token",
		"audio_input_price_per_token",
	}
	audioInputPerMillionPaths = []string{
		"pricing.audio_input_per_million",
		"pricing.input_audio_per_million",
		"cost.audio_input",
		"cost.input_audio",
		"cost.audio_input_per_million",
		"cost.input_audio_per_million",
		"audio_input_cost_per_million",
		"audio_input_price_per_million",
	}
	audioInputPer1KPaths = []string{
		"pricing.audio_input_per_1k",
		"pricing.input_audio_per_1k",
		"cost.audio_input_per_1k",
		"cost.input_audio_per_1k",
		"audio_input_cost_per_1k",
		"audio_input_price_per_1k",
	}

	audioOutputPerTokenPaths = []string{
		"pricing.audio_output",
		"pricing.output_audio",
		"audio_output_cost_per_token",
		"audio_output_price_per_token",
	}
	audioOutputPerMillionPaths = []string{
		"pricing.audio_output_per_million",
		"pricing.output_audio_per_million",
		"cost.audio_output",
		"cost.output_audio",
		"cost.audio_output_per_million",
		"cost.output_audio_per_million",
		"audio_output_cost_per_million",
		"audio_output_price_per_million",
	}
	audioOutputPer1KPaths = []string{
		"pricing.audio_output_per_1k",
		"pricing.output_audio_per_1k",
		"cost.audio_output_per_1k",
		"cost.output_audio_per_1k",
		"audio_output_cost_per_1k",
		"audio_output_price_per_1k",
	}

	requestCostPaths = []string{
		"pricing.request",
		"pricing.per_request",
		"request_cost",
		"cost_per_request",
		"cost.request",
	}
)

func ParsePricingFromCapabilities(raw string) (ParsedPricing, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ParsedPricing{}, false
	}
	var payload any
	if err := json.Unmarshal([]byte(trimmed), &payload); err != nil {
		return ParsedPricing{}, false
	}

	result := ParsedPricing{}
	hasPricing := false

	if value, ok := extractUSDPerMillion(payload, inputPerTokenPaths, inputPerMillionPaths, inputPer1KPaths); ok {
		result.InputPerMillion = value
		hasPricing = true
	}
	if value, ok := extractUSDPerMillion(payload, outputPerTokenPaths, outputPerMillionPaths, outputPer1KPaths); ok {
		result.OutputPerMillion = value
		hasPricing = true
	}
	if value, ok := extractUSDPerMillion(payload, cachedPerTokenPaths, cachedPerMillionPaths, cachedPer1KPaths); ok {
		result.CachedInputPerMillion = value
		hasPricing = true
	}
	if value, ok := extractUSDPerMillion(payload, reasoningPerTokenPaths, reasoningPerMillionPaths, reasoningPer1KPaths); ok {
		result.ReasoningPerMillion = value
		hasPricing = true
	}
	if value, ok := extractUSDPerMillion(payload, audioInputPerTokenPaths, audioInputPerMillionPaths, audioInputPer1KPaths); ok {
		result.AudioInputPerMillion = value
		hasPricing = true
	}
	if value, ok := extractUSDPerMillion(payload, audioOutputPerTokenPaths, audioOutputPerMillionPaths, audioOutputPer1KPaths); ok {
		result.AudioOutputPerMillion = value
		hasPricing = true
	}
	if value, ok := extractUSDAmount(payload, requestCostPaths); ok {
		result.PerRequest = value
		hasPricing = true
	}

	return result, hasPricing
}

func extractUSDPerMillion(payload any, perTokenPaths []string, perMillionPaths []string, per1KPaths []string) (float64, bool) {
	if value, ok := extractUSDAmount(payload, perMillionPaths); ok {
		return value, true
	}
	if value, ok := extractUSDAmount(payload, per1KPaths); ok {
		return value * 1_000, true
	}
	if value, ok := extractUSDAmount(payload, perTokenPaths); ok {
		if value > 1 {
			// Some catalogs expose per-million value in generic keys.
			return value, true
		}
		return value * 1_000_000, true
	}
	return 0, false
}

func extractUSDAmount(payload any, paths []string) (float64, bool) {
	for _, path := range paths {
		value, ok := lookupValueByPath(payload, path)
		if !ok {
			continue
		}
		parsed, parsedOK := parsePricingDecimal(value)
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

func lookupValueByPath(payload any, path string) (any, bool) {
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

func parsePricingDecimal(value any) (float64, bool) {
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
