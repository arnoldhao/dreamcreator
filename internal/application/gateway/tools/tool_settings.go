package tools

import (
	"context"
	"strings"

	settingsdto "dreamcreator/internal/application/settings/dto"
)

const (
	ExecPermissionModeStandard  = "default permissions"
	ExecPermissionModeAllAccess = "full access"
)

type SettingsReader interface {
	GetSettings(ctx context.Context) (settingsdto.Settings, error)
}

func resolveToolsConfig(ctx context.Context, settingsService SettingsReader) map[string]any {
	if settingsService == nil {
		return nil
	}
	loaded, err := settingsService.GetSettings(ctx)
	if err != nil {
		return nil
	}
	toolsConfig := cloneAnyMap(loaded.Tools)
	if toolsConfig == nil {
		toolsConfig = map[string]any{}
	}
	if len(loaded.Skills) > 0 {
		toolsConfig["skills"] = cloneAnyMap(loaded.Skills)
	}
	if len(toolsConfig) == 0 {
		return nil
	}
	return toolsConfig
}

func cloneAnyMap(source map[string]any) map[string]any {
	if len(source) == 0 {
		return nil
	}
	result := make(map[string]any, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}

func mergeAnyMap(base map[string]any, overlay map[string]any) map[string]any {
	result := cloneAnyMap(base)
	if result == nil {
		result = map[string]any{}
	}
	for key, value := range overlay {
		overlayMap, ok := value.(map[string]any)
		if !ok {
			result[key] = value
			continue
		}
		if existing, ok := result[key].(map[string]any); ok {
			result[key] = mergeAnyMap(existing, overlayMap)
			continue
		}
		result[key] = mergeAnyMap(nil, overlayMap)
	}
	return result
}

func normalizeExecPermissionMode(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "full access", "full_access", "all_access", "all-access", "all access", "full", "danger-full-access":
		return ExecPermissionModeAllAccess
	case "default permissions", "default_permissions", "default permission", "standard", "default", "safe", "ask":
		return ExecPermissionModeStandard
	default:
		return ""
	}
}

func resolveExecPermissionMode(config map[string]any) string {
	candidates := []string{
		getNestedString(config, "execPermissionMode"),
		getNestedString(config, "permissionMode"),
		getNestedString(config, "permissions", "mode"),
		getNestedString(config, "exec", "permissionMode"),
	}
	for _, candidate := range candidates {
		if normalized := normalizeExecPermissionMode(candidate); normalized != "" {
			return normalized
		}
	}
	return ExecPermissionModeStandard
}
