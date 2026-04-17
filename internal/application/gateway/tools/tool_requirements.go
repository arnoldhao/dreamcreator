package tools

import (
	"context"
	"fmt"
	"os"
	"strings"

	"dreamcreator/internal/application/browsercdp"
	tooldto "dreamcreator/internal/application/tools/dto"
)

type toolRequirementSnapshot struct {
	loaded                     bool
	gatewayControlPlaneEnabled bool
	toolsConfig                map[string]any
	execPermissionMode         string
}

func loadToolRequirementSnapshot(ctx context.Context, settings SettingsReader) toolRequirementSnapshot {
	if settings == nil {
		return toolRequirementSnapshot{}
	}
	current, err := settings.GetSettings(ctx)
	if err != nil {
		return toolRequirementSnapshot{}
	}
	toolsConfig := cloneAnyMap(current.Tools)
	if toolsConfig == nil {
		toolsConfig = map[string]any{}
	}
	if len(current.Skills) > 0 {
		toolsConfig["skills"] = cloneAnyMap(current.Skills)
	}
	return toolRequirementSnapshot{
		loaded:                     true,
		gatewayControlPlaneEnabled: current.Gateway.ControlPlaneEnabled,
		toolsConfig:                toolsConfig,
		execPermissionMode:         resolveExecPermissionMode(toolsConfig),
	}
}

func resolveEffectiveToolSpec(spec tooldto.ToolSpec, snapshot toolRequirementSnapshot) tooldto.ToolSpec {
	key := resolveToolRequirementKey(spec)
	if snapshot.loaded {
		switch key {
		case "browser":
			if enabled, ok := resolveBrowserConfigBool(snapshot.toolsConfig, "enabled"); ok && !enabled {
				spec.Enabled = false
			}
		case "web_fetch":
			if enabled, ok := resolveWebFetchConfigBool(snapshot.toolsConfig, "enabled"); ok && !enabled {
				spec.Enabled = false
			}
		}
	}
	requirements := resolveToolRequirements(spec, snapshot)
	spec.Requirements = requirements
	if !spec.Enabled {
		return spec
	}
	if !toolRequirementsSatisfied(requirements) {
		spec.Enabled = false
	}
	return spec
}

func toolRequirementsSatisfied(requirements []tooldto.ToolRequirement) bool {
	for _, requirement := range requirements {
		if !requirement.Available {
			return false
		}
	}
	return true
}

func firstUnavailableToolRequirement(requirements []tooldto.ToolRequirement) (tooldto.ToolRequirement, bool) {
	for _, requirement := range requirements {
		if !requirement.Available {
			return requirement, true
		}
	}
	return tooldto.ToolRequirement{}, false
}

func resolveToolRequirements(spec tooldto.ToolSpec, snapshot toolRequirementSnapshot) []tooldto.ToolRequirement {
	if !snapshot.loaded {
		return nil
	}
	key := resolveToolRequirementKey(spec)
	switch key {
	case "gateway":
		requirement := tooldto.ToolRequirement{
			ID:        "gateway.control_plane_enabled",
			Name:      "Gateway control plane",
			Available: snapshot.gatewayControlPlaneEnabled,
		}
		if !requirement.Available {
			requirement.Reason = "Control plane is disabled"
		}
		return []tooldto.ToolRequirement{requirement}
	case "web_search":
		return resolveWebSearchRequirements(snapshot.toolsConfig)
	case "web_fetch":
		return resolveWebFetchRequirements(snapshot.toolsConfig)
	case "browser":
		return resolveBrowserRequirements(snapshot.toolsConfig)
	default:
		return nil
	}
}

func resolveBrowserRequirements(config map[string]any) []tooldto.ToolRequirement {
	resolved := resolveBrowserRuntimeConfig(config)
	status := browsercdp.ResolveStatus(resolved.PreferredBrowser, resolved.Headless)
	requirements := []tooldto.ToolRequirement{
		{
			ID:        "browser.cdp_runtime",
			Name:      "Local CDP browser",
			Available: status.Ready,
			Reason:    strings.TrimSpace(status.DetectError),
			Data: map[string]any{
				"candidates":             status.Candidates,
				"selectedBrowser":        status.SelectedBrowser,
				"chosenBrowser":          status.ChosenBrowser,
				"detectedExecutablePath": status.DetectedExecutablePath,
				"headless":               status.Headless,
			},
		},
	}
	return requirements
}

func resolveToolRequirementKey(spec tooldto.ToolSpec) string {
	key := strings.ToLower(strings.TrimSpace(spec.ID))
	if key == "" {
		key = strings.ToLower(strings.TrimSpace(spec.Name))
	}
	return key
}

func resolveWebFetchRequirements(config map[string]any) []tooldto.ToolRequirement {
	status := browsercdp.ResolveStatus(resolveWebFetchPreferredBrowser(config), resolveWebFetchHeadless(config))
	browserRequirement := tooldto.ToolRequirement{
		ID:        "web_fetch.local_browser",
		Name:      "Local CDP browser",
		Available: status.Ready,
		Reason:    strings.TrimSpace(status.DetectError),
		Data: map[string]any{
			"candidates":             status.Candidates,
			"selectedBrowser":        status.SelectedBrowser,
			"chosenBrowser":          status.ChosenBrowser,
			"detectedExecutablePath": status.DetectedExecutablePath,
			"headless":               status.Headless,
		},
	}
	return []tooldto.ToolRequirement{browserRequirement}
}

func resolveWebSearchRequirements(config map[string]any) []tooldto.ToolRequirement {
	searchType := resolveWebSearchType(config)
	switch searchType {
	case "api":
		provider := strings.ToLower(strings.TrimSpace(getNestedString(config, "web", "search", "provider")))
		if provider == "" {
			provider = "brave"
		}
		requirements := []tooldto.ToolRequirement{
			{
				ID:        "web_search.mode_supported",
				Name:      "Search mode",
				Available: true,
			},
		}
		providerLabel := resolveWebSearchProviderLabel(provider)
		providerSupported := isWebSearchProviderSupported(provider)
		providerRequirement := tooldto.ToolRequirement{
			ID:        "web_search.provider_supported",
			Name:      "Provider",
			Available: providerSupported,
		}
		if !providerSupported {
			providerRequirement.Reason = fmt.Sprintf("%s is not supported in API mode", providerLabel)
			return append(requirements, providerRequirement)
		}
		requirements = append(requirements, providerRequirement)
		apiKey := strings.TrimSpace(resolveWebSearchProviderAPIKey(config, provider))
		apiKeyRequirement := tooldto.ToolRequirement{
			ID:        "web_search.provider_api_key",
			Name:      "Provider API key",
			Available: apiKey != "",
		}
		if !apiKeyRequirement.Available {
			apiKeyRequirement.Reason = fmt.Sprintf("%s API key is missing", providerLabel)
		}
		return append(requirements, apiKeyRequirement)
	case "external_tools":
		return []tooldto.ToolRequirement{
			{
				ID:        "web_search.mode_supported",
				Name:      "Search mode",
				Available: true,
			},
			{
				ID:        "web_search.external_tools_supported",
				Name:      "External tools runtime",
				Available: false,
				Reason:    "External tools mode is not implemented",
			},
		}
	default:
		return []tooldto.ToolRequirement{
			{
				ID:        "web_search.mode_supported",
				Name:      "Search mode",
				Available: false,
				Reason:    "Search mode is not supported",
			},
		}
	}
}

func isWebSearchProviderSupported(provider string) bool {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "brave", "tavily":
		return true
	default:
		return false
	}
}

func resolveWebSearchProviderLabel(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "brave":
		return "Brave"
	case "tavily":
		return "Tavily"
	case "perplexity":
		return "Perplexity"
	case "grok":
		return "Grok"
	default:
		trimmed := strings.TrimSpace(provider)
		if trimmed == "" {
			return "Provider"
		}
		return trimmed
	}
}

func resolveWebSearchProviderAPIKey(config map[string]any, provider string) string {
	provider = strings.ToLower(strings.TrimSpace(provider))
	if provider == "" {
		return ""
	}
	apiKey := resolveWebSearchProviderString(config, provider, "apiKey")
	if apiKey == "" {
		apiKey = getNestedString(config, "web", "search", provider, "apiKey")
	}
	if apiKey == "" {
		apiKey = getNestedString(config, "web", "search", "apiKey")
	}
	if apiKey == "" {
		switch provider {
		case "brave":
			apiKey = strings.TrimSpace(os.Getenv("BRAVE_API_KEY"))
		case "tavily":
			apiKey = strings.TrimSpace(os.Getenv("TAVILY_API_KEY"))
		case "perplexity":
			apiKey = strings.TrimSpace(os.Getenv("PERPLEXITY_API_KEY"))
		case "grok":
			apiKey = strings.TrimSpace(os.Getenv("XAI_API_KEY"))
		}
	}
	return strings.TrimSpace(apiKey)
}

func resolveWebSearchProviderString(config map[string]any, provider string, key string) string {
	provider = strings.ToLower(strings.TrimSpace(provider))
	key = strings.TrimSpace(key)
	if provider == "" || key == "" {
		return ""
	}
	return getNestedString(config, "web", "search", "providers", provider, key)
}
