package providersync

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
)

type modelCapabilities struct {
	ContextWindow     *int
	MaxOutputTokens   *int
	SupportsTools     *bool
	SupportsReasoning *bool
	SupportsVision    *bool
	SupportsAudio     *bool
	SupportsVideo     *bool
}

func parseModelCapabilities(raw string) modelCapabilities {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return modelCapabilities{}
	}
	var payload any
	if err := json.Unmarshal([]byte(trimmed), &payload); err != nil {
		return modelCapabilities{}
	}
	return parseModelCapabilitiesPayload(payload)
}

func parseModelCapabilitiesPayload(payload any) modelCapabilities {
	contextWindowTokens := findTokensByPaths(payload, []string{
		"limit.context",
		"limits.context",
		"context.window",
		"context.length",
		"context.max",
		"context_window",
		"contextWindow",
		"context_length",
		"contextLength",
		"max_context_length",
		"maxContextLength",
		"max_context_tokens",
		"maxContextTokens",
	})
	if contextWindowTokens <= 0 {
		contextWindowTokens = findTokensByKey(payload, isContextWindowKey)
	}

	maxOutputTokens := findTokensByPaths(payload, []string{
		"limit.output",
		"limits.output",
		"output.max",
		"output.tokens",
		"max_output_tokens",
		"maxOutputTokens",
		"max_completion_tokens",
		"maxCompletionTokens",
	})
	if maxOutputTokens <= 0 {
		maxOutputTokens = findTokensByKey(payload, isMaxOutputTokensKey)
	}

	capabilities := modelCapabilities{
		ContextWindow:   normalizeOptionalInt(contextWindowTokens),
		MaxOutputTokens: normalizeOptionalInt(maxOutputTokens),
	}

	capabilities.SupportsTools = lookupSupportFlag(payload, []string{
		"supports.tools",
		"supports.tool_calling",
		"supports.toolCalling",
		"supports.function_calling",
		"supports.functionCalling",
		"supports.functions",
		"tool_call",
		"tool_calling",
		"toolCalling",
		"function_calling",
		"functionCalling",
		"tools",
		"functions",
	})
	capabilities.SupportsReasoning = lookupSupportFlag(payload, []string{
		"supports.reasoning",
		"supports.chain_of_thought",
		"supports.cot",
		"reasoning",
		"chain_of_thought",
		"cot",
	})

	visionFlag := lookupSupportFlag(payload, []string{
		"supports.vision",
		"vision",
		"supports.image",
		"image",
		"supports.images",
		"images",
	})
	audioFlag := lookupSupportFlag(payload, []string{
		"supports.audio",
		"audio",
	})
	videoFlag := lookupSupportFlag(payload, []string{
		"supports.video",
		"video",
	})

	if visionFlag != nil {
		capabilities.SupportsVision = visionFlag
	}
	if audioFlag != nil {
		capabilities.SupportsAudio = audioFlag
	}
	if videoFlag != nil {
		capabilities.SupportsVideo = videoFlag
	}

	modalitiesFound, modalities := extractModalities(payload)
	if modalitiesFound {
		if capabilities.SupportsVision == nil {
			capabilities.SupportsVision = boolPointer(modalities["image"] || modalities["vision"])
		}
		if capabilities.SupportsAudio == nil {
			capabilities.SupportsAudio = boolPointer(modalities["audio"])
		}
		if capabilities.SupportsVideo == nil {
			capabilities.SupportsVideo = boolPointer(modalities["video"])
		}
	}

	return capabilities
}

func mergeModelCapabilities(primary modelCapabilities, fallback modelCapabilities) modelCapabilities {
	return modelCapabilities{
		ContextWindow:     firstIntPointer(primary.ContextWindow, fallback.ContextWindow),
		MaxOutputTokens:   firstIntPointer(primary.MaxOutputTokens, fallback.MaxOutputTokens),
		SupportsTools:     firstBoolPointer(primary.SupportsTools, fallback.SupportsTools),
		SupportsReasoning: firstBoolPointer(primary.SupportsReasoning, fallback.SupportsReasoning),
		SupportsVision:    firstBoolPointer(primary.SupportsVision, fallback.SupportsVision),
		SupportsAudio:     firstBoolPointer(primary.SupportsAudio, fallback.SupportsAudio),
		SupportsVideo:     firstBoolPointer(primary.SupportsVideo, fallback.SupportsVideo),
	}
}

func firstIntPointer(primary *int, fallback *int) *int {
	if primary != nil {
		return primary
	}
	return fallback
}

func firstBoolPointer(primary *bool, fallback *bool) *bool {
	if primary != nil {
		return primary
	}
	return fallback
}

func boolPointer(value bool) *bool {
	result := value
	return &result
}

func normalizeOptionalInt(value int) *int {
	if value <= 0 {
		return nil
	}
	normalized := value
	return &normalized
}

func extractModalities(payload any) (bool, map[string]bool) {
	modalities := make(map[string]bool)
	found := false
	paths := []string{
		"modalities",
		"modalities.input",
		"modalities.output",
		"input_modalities",
		"output_modalities",
		"inputModalities",
		"outputModalities",
	}
	for _, path := range paths {
		value, ok := lookupValueByPath(payload, path)
		if !ok {
			continue
		}
		found = true
		collectModalities(modalities, value)
	}
	return found, modalities
}

func collectModalities(target map[string]bool, value any) {
	switch typed := value.(type) {
	case []any:
		for _, entry := range typed {
			if modality := mapModality(entry); modality != "" {
				target[modality] = true
			}
		}
	case []string:
		for _, entry := range typed {
			if modality := mapModality(entry); modality != "" {
				target[modality] = true
			}
		}
	case map[string]any:
		for key := range typed {
			if modality := mapModality(key); modality != "" {
				target[modality] = true
			}
		}
	case map[string]string:
		for key := range typed {
			if modality := mapModality(key); modality != "" {
				target[modality] = true
			}
		}
	case string:
		if modality := mapModality(typed); modality != "" {
			target[modality] = true
		}
	}
}

func mapModality(value any) string {
	text, ok := value.(string)
	if !ok {
		return ""
	}
	normalized := strings.ToLower(strings.TrimSpace(text))
	switch normalized {
	case "vision", "image", "images":
		return "image"
	case "audio", "speech":
		return "audio"
	case "video":
		return "video"
	default:
		return ""
	}
}

func lookupSupportFlag(payload any, paths []string) *bool {
	for _, path := range paths {
		value, ok := lookupValueByPath(payload, path)
		if !ok {
			continue
		}
		if parsed, ok := resolveSupportFlag(value); ok {
			return boolPointer(parsed)
		}
	}
	return nil
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
		typed, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		next, ok := typed[segment]
		if !ok {
			return nil, false
		}
		current = next
	}
	return current, true
}

func resolveSupportFlag(value any) (bool, bool) {
	switch typed := value.(type) {
	case bool:
		return typed, true
	case float64:
		return typed > 0, true
	case float32:
		return typed > 0, true
	case int:
		return typed > 0, true
	case int64:
		return typed > 0, true
	case json.Number:
		if parsed, err := typed.Int64(); err == nil {
			return parsed > 0, true
		}
	case string:
		normalized := strings.TrimSpace(strings.ToLower(typed))
		if normalized == "" {
			return false, false
		}
		switch normalized {
		case "true", "yes", "1":
			return true, true
		case "false", "no", "0":
			return false, true
		}
	case []any:
		return len(typed) > 0, true
	case map[string]any:
		return len(typed) > 0, true
	}
	return false, false
}

func findTokensByKey(payload any, match func(string) bool) int {
	max := 0
	switch typed := payload.(type) {
	case map[string]any:
		for key, raw := range typed {
			if match(key) {
				if candidate := coerceInt(raw); candidate > max {
					max = candidate
				}
			}
			if nested := findTokensByKey(raw, match); nested > max {
				max = nested
			}
		}
	case []any:
		for _, raw := range typed {
			if nested := findTokensByKey(raw, match); nested > max {
				max = nested
			}
		}
	}
	return max
}

func findTokensByPaths(payload any, paths []string) int {
	max := 0
	for _, path := range paths {
		value, ok := lookupValueByPath(payload, path)
		if !ok {
			continue
		}
		if candidate := coerceInt(value); candidate > max {
			max = candidate
		}
	}
	return max
}

func isContextWindowKey(key string) bool {
	lower := strings.ToLower(strings.TrimSpace(key))
	if lower == "" {
		return false
	}
	if strings.Contains(lower, "context") && strings.Contains(lower, "window") {
		return true
	}
	if strings.Contains(lower, "context") && strings.Contains(lower, "length") {
		return true
	}
	if strings.Contains(lower, "context") && strings.Contains(lower, "tokens") {
		return true
	}
	if strings.Contains(lower, "max_context") && strings.Contains(lower, "tokens") {
		return true
	}
	return false
}

func isMaxOutputTokensKey(key string) bool {
	lower := strings.ToLower(strings.TrimSpace(key))
	if lower == "" {
		return false
	}
	if strings.Contains(lower, "context") {
		return false
	}
	if strings.Contains(lower, "max_output") && strings.Contains(lower, "token") {
		return true
	}
	if strings.Contains(lower, "max_completion") && strings.Contains(lower, "token") {
		return true
	}
	if strings.Contains(lower, "output") && strings.Contains(lower, "token") && strings.Contains(lower, "max") {
		return true
	}
	if lower == "max_tokens" || lower == "max-token" || lower == "max_token" || lower == "maxTokens" {
		return true
	}
	return false
}

var numericTokenPattern = regexp.MustCompile(`^(\d+(?:\.\d+)?)(k)?$`)

func coerceInt(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case json.Number:
		if parsed, err := typed.Int64(); err == nil {
			return int(parsed)
		}
	case string:
		trimmed := strings.TrimSpace(strings.ToLower(typed))
		if trimmed == "" {
			return 0
		}
		if match := numericTokenPattern.FindStringSubmatch(trimmed); match != nil {
			value := match[1]
			multiplier := 1.0
			if match[2] != "" {
				multiplier = 1000
			}
			if parsed, err := strconv.ParseFloat(value, 64); err == nil {
				return int(parsed * multiplier)
			}
		}
		if parsed, err := strconv.Atoi(trimmed); err == nil {
			return parsed
		}
	}
	return 0
}
