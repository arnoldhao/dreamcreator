package service

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	goruntime "runtime"
	"strings"
)

func (service *SkillsService) resolvePromptSkillEntryConfigs(ctx context.Context) map[string]map[string]any {
	if service == nil || service.settings == nil {
		return nil
	}
	current, err := service.settings.GetSettings(ctx)
	if err != nil {
		return nil
	}
	_, skillsConfig := resolveSettingsToolsSkills(current)
	if len(skillsConfig) == 0 {
		return nil
	}
	entriesRaw := toStringAnyMap(skillsConfig["entries"])
	if len(entriesRaw) == 0 {
		return nil
	}
	result := make(map[string]map[string]any, len(entriesRaw))
	for rawKey, rawValue := range entriesRaw {
		key := strings.ToLower(strings.TrimSpace(rawKey))
		if key == "" {
			continue
		}
		entryMap := toStringAnyMap(rawValue)
		if len(entryMap) == 0 {
			continue
		}
		copied := make(map[string]any, len(entryMap))
		for entryKey, entryValue := range entryMap {
			copied[entryKey] = entryValue
		}
		result[key] = copied
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func resolvePromptSkillEntryConfig(
	entry skillEntry,
	configs map[string]map[string]any,
) map[string]any {
	if len(configs) == 0 {
		return nil
	}
	candidates := []string{entry.ID, entry.Name}
	for _, candidate := range candidates {
		key := strings.ToLower(strings.TrimSpace(candidate))
		if key == "" {
			continue
		}
		if entryConfig, ok := configs[key]; ok {
			return entryConfig
		}
	}
	return nil
}

func isSkillEntryRuntimeEligible(entry skillEntry, entryConfig map[string]any) bool {
	requirements := entry.Runtime
	if requirements == nil {
		return true
	}
	if len(requirements.OS) > 0 && !containsStringFold(requirements.OS, goruntime.GOOS) {
		return false
	}
	if len(requirements.Bins) > 0 {
		for _, bin := range requirements.Bins {
			if !skillBinaryExists(bin) {
				return false
			}
		}
	}
	if len(requirements.AnyBins) > 0 {
		found := false
		for _, bin := range requirements.AnyBins {
			if skillBinaryExists(bin) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	if len(requirements.Env) > 0 {
		for _, envName := range requirements.Env {
			if !isSkillRequiredEnvSatisfied(envName, requirements.PrimaryEnv, entryConfig) {
				return false
			}
		}
	}
	if len(requirements.Config) > 0 {
		for _, configPath := range requirements.Config {
			if !isSkillConfigPathTruthy(entryConfig, configPath) {
				return false
			}
		}
	}
	return true
}

func containsStringFold(values []string, target string) bool {
	target = strings.TrimSpace(target)
	if target == "" || len(values) == 0 {
		return false
	}
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), target) {
			return true
		}
	}
	return false
}

func skillBinaryExists(bin string) bool {
	trimmed := strings.TrimSpace(bin)
	if trimmed == "" {
		return false
	}
	_, err := exec.LookPath(trimmed)
	return err == nil
}

func isSkillRequiredEnvSatisfied(envName string, primaryEnv string, entryConfig map[string]any) bool {
	trimmedName := strings.TrimSpace(envName)
	if trimmedName == "" {
		return false
	}
	if value, ok := os.LookupEnv(trimmedName); ok && strings.TrimSpace(value) != "" {
		return true
	}
	if configuredEnv := readSkillConfiguredEnv(entryConfig); len(configuredEnv) > 0 {
		if value, ok := configuredEnv[trimmedName]; ok && strings.TrimSpace(value) != "" {
			return true
		}
	}
	if strings.EqualFold(strings.TrimSpace(primaryEnv), trimmedName) {
		if apiKey, ok := entryConfig["apiKey"].(string); ok && strings.TrimSpace(apiKey) != "" {
			return true
		}
	}
	return false
}

func readSkillConfiguredEnv(entryConfig map[string]any) map[string]string {
	if len(entryConfig) == 0 {
		return nil
	}
	raw, ok := entryConfig["env"]
	if !ok {
		return nil
	}
	switch typed := raw.(type) {
	case map[string]string:
		if len(typed) == 0 {
			return nil
		}
		return typed
	case map[string]any:
		result := make(map[string]string, len(typed))
		for key, value := range typed {
			normalizedKey := strings.TrimSpace(key)
			if normalizedKey == "" {
				continue
			}
			normalizedValue := strings.TrimSpace(fmt.Sprint(value))
			if normalizedValue == "" {
				continue
			}
			result[normalizedKey] = normalizedValue
		}
		if len(result) == 0 {
			return nil
		}
		return result
	default:
		return nil
	}
}

func isSkillConfigPathTruthy(entryConfig map[string]any, configPath string) bool {
	if len(entryConfig) == 0 {
		return false
	}
	configMap := toStringAnyMap(entryConfig["config"])
	if len(configMap) == 0 {
		return false
	}
	value, ok := resolveNestedSkillConfigValue(configMap, configPath)
	if !ok {
		return false
	}
	return isTruthySkillConfigValue(value)
}

func resolveNestedSkillConfigValue(root map[string]any, configPath string) (any, bool) {
	segments := strings.Split(strings.TrimSpace(configPath), ".")
	if len(segments) == 0 {
		return nil, false
	}
	var current any = root
	for _, segment := range segments {
		key := strings.TrimSpace(segment)
		if key == "" {
			continue
		}
		switch typed := current.(type) {
		case map[string]any:
			next, ok := typed[key]
			if !ok {
				return nil, false
			}
			current = next
		case map[string]string:
			next, ok := typed[key]
			if !ok {
				return nil, false
			}
			current = next
		default:
			return nil, false
		}
	}
	return current, true
}

func isTruthySkillConfigValue(value any) bool {
	switch typed := value.(type) {
	case nil:
		return false
	case bool:
		return typed
	case string:
		return strings.TrimSpace(typed) != ""
	case int:
		return typed != 0
	case int8:
		return typed != 0
	case int16:
		return typed != 0
	case int32:
		return typed != 0
	case int64:
		return typed != 0
	case uint:
		return typed != 0
	case uint8:
		return typed != 0
	case uint16:
		return typed != 0
	case uint32:
		return typed != 0
	case uint64:
		return typed != 0
	case float32:
		return typed != 0
	case float64:
		return typed != 0
	case []string:
		return len(typed) > 0
	case []any:
		return len(typed) > 0
	case map[string]any:
		return len(typed) > 0
	case map[string]string:
		return len(typed) > 0
	default:
		return true
	}
}
