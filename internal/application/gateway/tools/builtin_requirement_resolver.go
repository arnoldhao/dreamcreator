package tools

import (
	"context"
	"strings"

	assistantservice "dreamcreator/internal/application/assistant/service"
	gatewayvoice "dreamcreator/internal/application/gateway/voice"
	tooldto "dreamcreator/internal/application/tools/dto"
	"dreamcreator/internal/domain/providers"
)

const (
	imageRequirementID               = "image.model_runtime"
	ttsServiceRequirementID          = "tts.voice_service"
	ttsVoiceEnabledRequirementID     = "tts.voice_enabled"
	ttsProviderRequirementID         = "tts.provider_supported"
	ttsProviderAPIKeyRequirementID   = "tts.provider_api_key"
	ttsVoiceIDRequirementID          = "tts.voice_id"
	imageModelNotConfiguredReason    = "Image model is not configured"
	providerRepositoriesReason       = "Provider repositories are unavailable"
	voiceDisabledReason              = "Voice is disabled"
	voiceServiceUnavailableReason    = "Voice service unavailable"
	ttsProviderAPIKeyMissingReason   = "TTS provider API key is missing"
	ttsVoiceIDMissingReason          = "TTS voice ID is not configured"
	ttsEdgeProviderUnavailableReason = "Edge-TTS provider is not implemented yet"
	ttsUnsupportedProviderReason     = "TTS provider is not supported"
)

type voiceToolStatusProvider interface {
	Status(ctx context.Context) (gatewayvoice.TTSStatusResponse, error)
}

type BuiltinRequirementDeps struct {
	Settings   SettingsReader
	Assistants *assistantservice.AssistantService
	Providers  providers.ProviderRepository
	Models     providers.ModelRepository
	Secrets    providers.SecretRepository
	Voice      voiceToolStatusProvider
}

type builtinRequirementResolver struct {
	deps BuiltinRequirementDeps
}

func NewBuiltinRequirementResolver(deps BuiltinRequirementDeps) ToolRequirementResolver {
	return builtinRequirementResolver{deps: deps}
}

func (resolver builtinRequirementResolver) ResolveToolRequirements(ctx context.Context, spec tooldto.ToolSpec) []tooldto.ToolRequirement {
	switch resolveToolRequirementKey(spec) {
	case "image":
		return resolver.resolveImageRequirements(ctx)
	case "tts":
		return resolver.resolveTTSRequirements(ctx)
	default:
		return nil
	}
}

func (resolver builtinRequirementResolver) resolveImageRequirements(ctx context.Context) []tooldto.ToolRequirement {
	requirement := tooldto.ToolRequirement{
		ID:        imageRequirementID,
		Name:      "Image model",
		Available: true,
	}
	if resolver.deps.Providers == nil || resolver.deps.Secrets == nil || resolver.deps.Models == nil {
		requirement.Available = false
		requirement.Reason = providerRepositoriesReason
		return []tooldto.ToolRequirement{requirement}
	}
	configuredPrimaryRef := resolveImageToolConfiguredPrimaryRef(ctx, resolver.deps.Settings)
	_, _, err := resolveImageToolCandidates(
		ctx,
		resolver.deps.Assistants,
		"",
		configuredPrimaryRef,
		resolver.deps.Providers,
		resolver.deps.Models,
		resolver.deps.Secrets,
	)
	if err == nil {
		return []tooldto.ToolRequirement{requirement}
	}
	requirement.Available = false
	requirement.Reason = normalizeBuiltinRequirementReason(err, imageModelNotConfiguredReason)
	return []tooldto.ToolRequirement{requirement}
}

func (resolver builtinRequirementResolver) resolveTTSRequirements(ctx context.Context) []tooldto.ToolRequirement {
	if resolver.deps.Voice == nil {
		return []tooldto.ToolRequirement{
			{
				ID:        ttsServiceRequirementID,
				Name:      "Voice service",
				Available: false,
				Reason:    voiceServiceUnavailableReason,
			},
		}
	}
	status, err := resolver.deps.Voice.Status(ctx)
	if err != nil {
		return []tooldto.ToolRequirement{
			{
				ID:        ttsServiceRequirementID,
				Name:      "Voice service",
				Available: false,
				Reason:    normalizeBuiltinRequirementReason(err, voiceServiceUnavailableReason),
			},
		}
	}
	providerID := strings.ToLower(strings.TrimSpace(status.Config.ProviderID))
	if providerID == "" {
		providerID = "edge"
	}
	requirements := []tooldto.ToolRequirement{
		{
			ID:        ttsVoiceEnabledRequirementID,
			Name:      "Voice feature",
			Available: status.Enabled,
		},
	}
	if !status.Enabled {
		requirements[0].Reason = voiceDisabledReason
	}
	switch providerID {
	case "edge":
		requirements = append(requirements, tooldto.ToolRequirement{
			ID:        ttsProviderRequirementID,
			Name:      "Provider",
			Available: false,
			Reason:    ttsEdgeProviderUnavailableReason,
			Data: map[string]any{
				"providerId": providerID,
			},
		})
		return requirements
	case "openai", "elevenlabs":
		providerReady := false
		for _, provider := range status.Providers {
			if strings.EqualFold(strings.TrimSpace(provider.ProviderID), providerID) {
				providerReady = provider.Available
				break
			}
		}
		requirements = append(requirements,
			tooldto.ToolRequirement{
				ID:        ttsProviderRequirementID,
				Name:      "Provider",
				Available: true,
				Data: map[string]any{
					"providerId": providerID,
				},
			},
			tooldto.ToolRequirement{
				ID:        ttsProviderAPIKeyRequirementID,
				Name:      "Provider API key",
				Available: providerReady,
			},
		)
		if !providerReady {
			requirements[len(requirements)-1].Reason = ttsProviderAPIKeyMissingReason
		}
		if providerID == "elevenlabs" {
			voiceIDConfigured := strings.TrimSpace(status.Config.VoiceID) != ""
			requirements = append(requirements, tooldto.ToolRequirement{
				ID:        ttsVoiceIDRequirementID,
				Name:      "Voice ID",
				Available: voiceIDConfigured,
				Data: map[string]any{
					"value": strings.TrimSpace(status.Config.VoiceID),
				},
			})
			if !voiceIDConfigured {
				requirements[len(requirements)-1].Reason = ttsVoiceIDMissingReason
			}
		}
		return requirements
	default:
		requirements = append(requirements, tooldto.ToolRequirement{
			ID:        ttsProviderRequirementID,
			Name:      "Provider",
			Available: false,
			Reason:    ttsUnsupportedProviderReason,
			Data: map[string]any{
				"providerId": providerID,
			},
		})
		return requirements
	}
}

func normalizeBuiltinRequirementReason(err error, fallback string) string {
	if err == nil {
		return fallback
	}
	message := strings.TrimSpace(err.Error())
	if message == "" {
		return fallback
	}
	return message
}
