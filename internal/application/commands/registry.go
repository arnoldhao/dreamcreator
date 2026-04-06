package commands

import (
	"strings"

	settingsdto "dreamcreator/internal/application/settings/dto"
	domaincommands "dreamcreator/internal/domain/commands"
)

type CommandDescriptor struct {
	Key         string                      `json:"key"`
	NativeName  string                      `json:"nativeName,omitempty"`
	Description string                      `json:"description"`
	Scope       domaincommands.Scope        `json:"scope"`
	AcceptsArgs bool                        `json:"acceptsArgs"`
	Args        []domaincommands.CommandArg `json:"args,omitempty"`
	Enabled     bool                        `json:"enabled"`
}

type NativeCommandSpec struct {
	Key         string
	Name        string
	Description string
	AcceptsArgs bool
	Args        []domaincommands.CommandArg
}

type CommandRunRequest struct {
	Key       string         `json:"key"`
	Args      map[string]any `json:"args,omitempty"`
	SessionID string         `json:"sessionId,omitempty"`
}

type CommandRunResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
}

func ListCommandDescriptors(settings settingsdto.Settings, provider string) []CommandDescriptor {
	caps := ResolveCapabilities(settings)
	flags := settings.Commands
	result := make([]CommandDescriptor, 0, len(commandRegistry))
	for _, command := range commandRegistry {
		nativeName := resolveNativeName(command, provider)
		enabled := isCommandEnabled(flags, command.Key) && capsAllow(caps, command.Requires)
		result = append(result, CommandDescriptor{
			Key:         command.Key,
			NativeName:  nativeName,
			Description: command.Description,
			Scope:       command.Scope,
			AcceptsArgs: command.AcceptsArgs,
			Args:        command.Args,
			Enabled:     enabled,
		})
	}
	return result
}

func ListEnabledCommands(settings settingsdto.Settings, provider string) []CommandDescriptor {
	descriptors := ListCommandDescriptors(settings, provider)
	filtered := make([]CommandDescriptor, 0, len(descriptors))
	for _, descriptor := range descriptors {
		if descriptor.Enabled {
			filtered = append(filtered, descriptor)
		}
	}
	return filtered
}

func ListNativeCommandSpecsForSettings(settings settingsdto.Settings, provider string) []NativeCommandSpec {
	descriptors := ListEnabledCommands(settings, provider)
	result := make([]NativeCommandSpec, 0, len(descriptors))
	for _, descriptor := range descriptors {
		if descriptor.Scope == domaincommands.ScopeText {
			continue
		}
		name := strings.TrimSpace(descriptor.NativeName)
		if name == "" {
			continue
		}
		result = append(result, NativeCommandSpec{
			Key:         descriptor.Key,
			Name:        name,
			Description: descriptor.Description,
			AcceptsArgs: descriptor.AcceptsArgs,
			Args:        descriptor.Args,
		})
	}
	return result
}

func resolveNativeName(command domaincommands.CommandDefinition, provider string) string {
	name := strings.TrimSpace(command.NativeName)
	if provider == "" || command.ProviderOverride == nil {
		return name
	}
	normalized := strings.ToLower(strings.TrimSpace(provider))
	if normalized == "" {
		return name
	}
	if override, ok := command.ProviderOverride[normalized]; ok {
		if strings.TrimSpace(override) != "" {
			return strings.TrimSpace(override)
		}
	}
	if override, ok := command.ProviderOverride[provider]; ok {
		if strings.TrimSpace(override) != "" {
			return strings.TrimSpace(override)
		}
	}
	return name
}

func isCommandEnabled(flags map[string]bool, key string) bool {
	if key == "" {
		return true
	}
	if flags == nil {
		return true
	}
	enabled, ok := flags[key]
	if !ok {
		return true
	}
	return enabled
}

func capsAllow(caps Capabilities, requires []domaincommands.CapabilityKey) bool {
	if len(requires) == 0 {
		return true
	}
	for _, capability := range requires {
		if !caps.Has(capability) {
			return false
		}
	}
	return true
}
