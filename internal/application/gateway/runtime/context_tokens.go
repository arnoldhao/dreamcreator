package runtime

import (
	"context"
	"encoding/json"
	"math"
	"sort"
	"strconv"
	"strings"

	"dreamcreator/internal/application/agentruntime"
	skillsdto "dreamcreator/internal/application/skills/dto"
	tooldto "dreamcreator/internal/application/tools/dto"
	domainassistant "dreamcreator/internal/domain/assistant"
)

var contextWindowPathCandidates = []string{
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
}

func resolveContextWindowOverride(metadata map[string]any) int {
	if metadata == nil {
		return 0
	}
	for _, key := range []string{
		"contextWindowTokens",
		"modelContextWindowTokens",
		"contextTokens",
		"maxContextTokens",
	} {
		if value, ok := resolveMetadataInt(metadata, key); ok && value > 0 {
			return value
		}
	}
	return 0
}

func pickModelContextWindowTokenMin(current int, candidate int) int {
	if candidate <= 0 {
		return current
	}
	if current <= 0 || candidate < current {
		return candidate
	}
	return current
}

func extractModelContextWindowTokens(modelName string, candidateName string, contextWindow *int, capabilities string) int {
	if !strings.EqualFold(strings.TrimSpace(candidateName), strings.TrimSpace(modelName)) {
		return 0
	}
	if contextWindow != nil && *contextWindow > 0 {
		return *contextWindow
	}
	return extractContextWindowTokens(capabilities)
}

func (service *Service) resolveContextWindowTokens(ctx context.Context, providerID string, modelName string, metadata map[string]any) int {
	if override := resolveContextWindowOverride(metadata); override > 0 {
		return override
	}
	if service == nil || service.models == nil {
		return defaultContextWindowTokens
	}
	modelName = strings.TrimSpace(modelName)
	if modelName == "" {
		return defaultContextWindowTokens
	}

	selected := 0
	providerID = strings.TrimSpace(providerID)
	if providerID != "" {
		models, err := service.models.ListByProvider(ctx, providerID)
		if err == nil {
			for _, model := range models {
				candidate := extractModelContextWindowTokens(modelName, model.Name, model.ContextWindow, model.CapabilitiesJSON)
				selected = pickModelContextWindowTokenMin(selected, candidate)
			}
		}
	}
	if selected > 0 {
		return selected
	}

	// Fallback chain: scan local model registry across providers and choose the
	// minimum context window for duplicate model names as a fail-safe default.
	if service.providers != nil {
		providers, err := service.providers.List(ctx)
		if err == nil {
			for _, provider := range providers {
				models, modelErr := service.models.ListByProvider(ctx, provider.ID)
				if modelErr != nil {
					continue
				}
				for _, model := range models {
					candidate := extractModelContextWindowTokens(modelName, model.Name, model.ContextWindow, model.CapabilitiesJSON)
					selected = pickModelContextWindowTokenMin(selected, candidate)
				}
			}
		}
	}
	if selected > 0 {
		return selected
	}
	return defaultContextWindowTokens
}

func extractContextWindowTokens(raw string) int {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0
	}
	var payload any
	if err := json.Unmarshal([]byte(trimmed), &payload); err != nil {
		return 0
	}
	pathMatch := findTokensByPaths(payload, contextWindowPathCandidates)
	keyMatch := findContextWindowTokens(payload)
	if pathMatch > keyMatch {
		return pathMatch
	}
	return keyMatch
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

func findContextWindowTokens(value any) int {
	max := 0
	switch typed := value.(type) {
	case map[string]any:
		for key, raw := range typed {
			if isContextWindowKey(key) {
				if candidate := coerceInt(raw); candidate > max {
					max = candidate
				}
			}
			if nested := findContextWindowTokens(raw); nested > max {
				max = nested
			}
		}
	case []any:
		for _, raw := range typed {
			if nested := findContextWindowTokens(raw); nested > max {
				max = nested
			}
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
	if lower == "context" {
		return true
	}
	return false
}

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
		trimmed := strings.ToLower(strings.TrimSpace(typed))
		if trimmed == "" {
			return 0
		}
		if strings.HasSuffix(trimmed, "k") {
			base := strings.TrimSpace(strings.TrimSuffix(trimmed, "k"))
			if parsed, err := strconv.ParseFloat(base, 64); err == nil {
				return int(math.Round(parsed * 1000))
			}
		}
		if parsed, err := strconv.Atoi(trimmed); err == nil {
			return parsed
		}
		if parsed, err := strconv.ParseFloat(trimmed, 64); err == nil {
			return int(math.Round(parsed))
		}
	}
	return 0
}

func estimateToolSpecTokens(specs []tooldto.ToolSpec) int {
	if len(specs) == 0 {
		return 0
	}
	total := 0
	for _, spec := range specs {
		total += agentruntime.EstimateTextTokens(spec.Name)
		total += agentruntime.EstimateTextTokens(spec.Description)
		total += agentruntime.EstimateTextTokens(spec.SchemaJSON)
	}
	if total < 0 {
		return 0
	}
	return total
}

func limitSkillPromptItems(items []skillsdto.SkillPromptItem, config domainassistant.AssistantSkills) []skillsdto.SkillPromptItem {
	if len(items) == 0 {
		return nil
	}
	sorted := append([]skillsdto.SkillPromptItem(nil), items...)
	sort.Slice(sorted, func(i, j int) bool {
		return strings.ToLower(sorted[i].Name) < strings.ToLower(sorted[j].Name)
	})
	maxCount := config.MaxSkillsInPrompt
	if maxCount > 0 && len(sorted) > maxCount {
		sorted = sorted[:maxCount]
	}
	maxChars := config.MaxPromptChars
	used := len(strings.Join(skillsSectionPreambleLines(), "\n")) + 1
	limited := make([]skillsdto.SkillPromptItem, 0, len(sorted))
	for _, item := range sorted {
		line := buildSkillPromptLine(item)
		if line == "" {
			continue
		}
		next := used + len(line) + 1
		if maxChars > 0 && used > 0 && next > maxChars {
			break
		}
		limited = append(limited, item)
		used = next
	}
	return limited
}

func buildSkillPromptLine(item skillsdto.SkillPromptItem) string {
	name := strings.TrimSpace(item.Name)
	if name == "" {
		return ""
	}
	desc := strings.TrimSpace(item.Description)
	line := "- " + name
	if desc != "" {
		line += ": " + desc
	}
	if path := strings.TrimSpace(item.Path); path != "" {
		line += " (path: " + path + ")"
	}
	return line
}
