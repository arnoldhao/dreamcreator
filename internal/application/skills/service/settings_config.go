package service

import settingsdto "dreamcreator/internal/application/settings/dto"

func resolveSettingsToolsSkills(current settingsdto.Settings) (map[string]any, map[string]any) {
	toolsConfig := cloneSettingsConfigMap(current.Tools)
	skillsConfig := cloneSettingsConfigMap(current.Skills)
	if len(toolsConfig) == 0 {
		toolsConfig = nil
	}
	if len(skillsConfig) == 0 {
		skillsConfig = nil
	}
	return toolsConfig, skillsConfig
}

func cloneSettingsConfigMap(source map[string]any) map[string]any {
	if len(source) == 0 {
		return nil
	}
	result := make(map[string]any, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}
