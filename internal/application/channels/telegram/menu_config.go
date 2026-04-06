package telegram

import (
	"fmt"
	"strings"

	settingsdto "dreamcreator/internal/application/settings/dto"
)

type MenuConfig struct {
	Enabled                bool
	BotToken               string
	NativeCommandsEnabled  bool
	NativeSkillsEnabled    bool
	NativeDisabledExplicit bool
	CustomCommands         []CustomCommandInput
}

func ResolveMenuConfig(settings settingsdto.Settings) MenuConfig {
	channels := settings.Channels
	channel := resolveMap(channels["telegram"])
	enabled := resolveBool(channel["enabled"], true)
	botToken := resolveBotToken(DefaultTelegramAccountID, channel, channel)
	commands := resolveMap(channel["commands"])
	nativeEnabled, nativeExplicit := resolveCommandSetting(commands["native"])
	nativeSkillsEnabled, _ := resolveCommandSetting(commands["nativeSkills"])
	customCommands := resolveCustomCommands(channel["customCommands"])
	return MenuConfig{
		Enabled:                enabled,
		BotToken:               botToken,
		NativeCommandsEnabled:  nativeEnabled,
		NativeSkillsEnabled:    nativeSkillsEnabled,
		NativeDisabledExplicit: nativeExplicit,
		CustomCommands:         customCommands,
	}
}

func resolveMap(value any) map[string]any {
	if typed, ok := value.(map[string]any); ok {
		return typed
	}
	return map[string]any{}
}

func resolveString(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	default:
		return strings.TrimSpace(toString(value))
	}
}

func resolveBool(value any, fallback bool) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		lower := strings.ToLower(strings.TrimSpace(typed))
		if lower == "true" || lower == "1" || lower == "yes" || lower == "on" {
			return true
		}
		if lower == "false" || lower == "0" || lower == "no" || lower == "off" {
			return false
		}
	}
	return fallback
}

func resolveCommandSetting(value any) (enabled bool, explicit bool) {
	switch typed := value.(type) {
	case bool:
		if typed {
			return true, false
		}
		return false, true
	case string:
		lower := strings.ToLower(strings.TrimSpace(typed))
		switch lower {
		case "", "auto", "default":
			return true, false
		case "true", "on", "enabled":
			return true, false
		case "false", "off", "disabled":
			return false, true
		}
	}
	return true, false
}

func resolveCustomCommands(value any) []CustomCommandInput {
	switch typed := value.(type) {
	case []any:
		result := make([]CustomCommandInput, 0, len(typed))
		for _, item := range typed {
			entry, ok := item.(map[string]any)
			if !ok {
				continue
			}
			result = append(result, CustomCommandInput{
				Command:     resolveString(entry["command"]),
				Description: resolveString(entry["description"]),
			})
		}
		return result
	case []map[string]any:
		result := make([]CustomCommandInput, 0, len(typed))
		for _, entry := range typed {
			result = append(result, CustomCommandInput{
				Command:     resolveString(entry["command"]),
				Description: resolveString(entry["description"]),
			})
		}
		return result
	default:
		return nil
	}
}

func toString(value any) string {
	if value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	case []byte:
		return string(typed)
	default:
		return fmt.Sprint(value)
	}
}
