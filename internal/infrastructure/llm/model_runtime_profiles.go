package llm

import (
	"strings"

	domainproviders "dreamcreator/internal/domain/providers"
)

type ModelRuntimeProfile struct {
	Match      ModelRuntimeProfileMatch
	Reasoning  *ModelReasoningProfile
	Notes      string
	Sources    []string
	VerifiedAt string
}

type ModelRuntimeProfileMatch struct {
	ProviderID    string
	ProviderType  domainproviders.ProviderType
	Compatibility domainproviders.ProviderCompatibility
	ModelExact    string
	ModelPrefix   string
}

type ModelReasoningProfile struct {
	Supported            bool
	DisableSupported     bool
	EffortLevels         []string
	SupportsBudgetTokens bool
	ControlProtocol      ReasoningControlProtocol
	ReasoningRequired    bool
}

type ReasoningControlProtocol string

const (
	ReasoningControlProtocolUnknown               ReasoningControlProtocol = ""
	ReasoningControlProtocolOpenAIReasoningEffort ReasoningControlProtocol = "openai_reasoning_effort"
	ReasoningControlProtocolOpenRouterReasoning   ReasoningControlProtocol = "openrouter_reasoning"
	ReasoningControlProtocolThinkingToggle        ReasoningControlProtocol = "thinking_toggle"
	ReasoningControlProtocolAnthropicThinking     ReasoningControlProtocol = "anthropic_thinking"
	ReasoningControlProtocolQwenThinkingToggle    ReasoningControlProtocol = "qwen_thinking_toggle"
)

func resolveModelRuntimeProfile(match providerRequestCompatibility) (ModelRuntimeProfile, bool) {
	bestScore := -1
	bestIndex := -1
	for index, profile := range modelRuntimeProfiles {
		score, ok := scoreModelRuntimeProfile(profile, match)
		if !ok {
			continue
		}
		if score > bestScore {
			bestScore = score
			bestIndex = index
		}
	}
	if bestIndex < 0 {
		return ModelRuntimeProfile{}, false
	}
	return modelRuntimeProfiles[bestIndex], true
}

func resolveModelReasoningProfile(match providerRequestCompatibility) (ModelReasoningProfile, bool) {
	profile, ok := resolveModelRuntimeProfile(match)
	if !ok || profile.Reasoning == nil {
		return ModelReasoningProfile{}, false
	}
	return *profile.Reasoning, true
}

func scoreModelRuntimeProfile(profile ModelRuntimeProfile, match providerRequestCompatibility) (int, bool) {
	score := 0
	matchedField := false

	if expected := strings.ToLower(strings.TrimSpace(profile.Match.ProviderID)); expected != "" {
		if strings.ToLower(strings.TrimSpace(match.ProviderID)) != expected {
			return 0, false
		}
		score += 32
		matchedField = true
	}
	if expected := strings.TrimSpace(string(profile.Match.ProviderType)); expected != "" {
		if strings.TrimSpace(string(match.ProviderType)) != expected {
			return 0, false
		}
		score += 16
		matchedField = true
	}
	if expected := strings.TrimSpace(string(profile.Match.Compatibility)); expected != "" {
		if strings.TrimSpace(string(match.Compatibility)) != expected {
			return 0, false
		}
		score += 24
		matchedField = true
	}
	if expected := strings.TrimSpace(profile.Match.ModelExact); expected != "" {
		if !matchesRuntimeModelValue(match.ModelName, expected, true) {
			return 0, false
		}
		score += 128 + len(expected)
		matchedField = true
	}
	if expected := strings.TrimSpace(profile.Match.ModelPrefix); expected != "" {
		if !matchesRuntimeModelValue(match.ModelName, expected, false) {
			return 0, false
		}
		score += 64 + len(expected)
		matchedField = true
	}
	if !matchedField {
		return 0, false
	}
	return score, true
}

func matchesRuntimeModelValue(modelName string, expected string, exact bool) bool {
	expected = strings.ToLower(strings.TrimSpace(expected))
	if expected == "" {
		return false
	}
	for _, candidate := range runtimeModelCandidates(modelName) {
		if exact {
			if candidate == expected {
				return true
			}
			continue
		}
		if strings.HasPrefix(candidate, expected) {
			return true
		}
	}
	return false
}

func runtimeModelCandidates(modelName string) []string {
	normalized := strings.ToLower(strings.TrimSpace(modelName))
	if normalized == "" {
		return nil
	}
	candidates := []string{normalized}
	if slash := strings.LastIndex(normalized, "/"); slash >= 0 && slash+1 < len(normalized) {
		shortName := normalized[slash+1:]
		if shortName != normalized {
			candidates = append(candidates, shortName)
		}
	}
	return candidates
}

func reasoningProfileSupportsLevel(profile ModelReasoningProfile, level string) bool {
	if !profile.Supported {
		return false
	}
	level = normalizeProviderThinkingLevel(level)
	if level == "" {
		return false
	}
	if level == "off" {
		return profile.DisableSupported && !profile.ReasoningRequired
	}
	if len(profile.EffortLevels) == 0 {
		return true
	}
	for _, candidate := range profile.EffortLevels {
		if normalizeProviderThinkingLevel(candidate) == level {
			return true
		}
	}
	return false
}

var modelRuntimeProfiles = []ModelRuntimeProfile{
	{
		Match: ModelRuntimeProfileMatch{
			Compatibility: domainproviders.ProviderCompatibilityOpenAI,
			ModelExact:    "gpt-5-pro",
		},
		Reasoning: &ModelReasoningProfile{
			Supported:         true,
			DisableSupported:  false,
			EffortLevels:      []string{"high"},
			ControlProtocol:   ReasoningControlProtocolOpenAIReasoningEffort,
			ReasoningRequired: true,
		},
		Notes: "OpenAI Chat Completions: gpt-5-pro only supports high reasoning effort.",
		Sources: []string{
			"https://platform.openai.com/docs/api-reference/chat/create-chat-completion",
		},
		VerifiedAt: "2026-04-17",
	},
	{
		Match: ModelRuntimeProfileMatch{
			Compatibility: domainproviders.ProviderCompatibilityOpenAI,
			ModelPrefix:   "gpt-5.2-pro",
		},
		Reasoning: &ModelReasoningProfile{
			Supported:         true,
			DisableSupported:  false,
			EffortLevels:      []string{"medium", "high", "xhigh"},
			ControlProtocol:   ReasoningControlProtocolOpenAIReasoningEffort,
			ReasoningRequired: true,
		},
		Notes: "GPT-5.2 pro supports medium, high, xhigh and does not expose none.",
		Sources: []string{
			"https://platform.openai.com/docs/models/gpt-5.2-pro",
		},
		VerifiedAt: "2026-04-17",
	},
	{
		Match: ModelRuntimeProfileMatch{
			Compatibility: domainproviders.ProviderCompatibilityOpenAI,
			ModelPrefix:   "gpt-5.2-codex",
		},
		Reasoning: &ModelReasoningProfile{
			Supported:        true,
			DisableSupported: false,
			EffortLevels:     []string{"low", "medium", "high", "xhigh"},
			ControlProtocol:  ReasoningControlProtocolOpenAIReasoningEffort,
		},
		Notes: "GPT-5.2-Codex supports low, medium, high, xhigh and does not expose none.",
		Sources: []string{
			"https://platform.openai.com/docs/models/gpt-5.2-codex",
		},
		VerifiedAt: "2026-04-17",
	},
	{
		Match: ModelRuntimeProfileMatch{
			Compatibility: domainproviders.ProviderCompatibilityOpenAI,
			ModelPrefix:   "gpt-5.2",
		},
		Reasoning: &ModelReasoningProfile{
			Supported:        true,
			DisableSupported: true,
			EffortLevels:     []string{"low", "medium", "high", "xhigh"},
			ControlProtocol:  ReasoningControlProtocolOpenAIReasoningEffort,
		},
		Notes: "GPT-5.2 supports none, low, medium, high, xhigh.",
		Sources: []string{
			"https://platform.openai.com/docs/guides/latest-model",
			"https://platform.openai.com/docs/models/gpt-5.2",
		},
		VerifiedAt: "2026-04-17",
	},
	{
		Match: ModelRuntimeProfileMatch{
			Compatibility: domainproviders.ProviderCompatibilityOpenAI,
			ModelPrefix:   "gpt-5.1",
		},
		Reasoning: &ModelReasoningProfile{
			Supported:        true,
			DisableSupported: true,
			EffortLevels:     []string{"low", "medium", "high"},
			ControlProtocol:  ReasoningControlProtocolOpenAIReasoningEffort,
		},
		Notes: "OpenAI Chat Completions: gpt-5.1 supports none, low, medium, high.",
		Sources: []string{
			"https://platform.openai.com/docs/api-reference/chat/create-chat-completion",
			"https://platform.openai.com/docs/models/gpt-5.1",
		},
		VerifiedAt: "2026-04-17",
	},
	{
		Match: ModelRuntimeProfileMatch{
			Compatibility: domainproviders.ProviderCompatibilityOpenAI,
			ModelPrefix:   "gpt-5",
		},
		Reasoning: &ModelReasoningProfile{
			Supported:        true,
			DisableSupported: false,
			EffortLevels:     []string{"minimal", "low", "medium", "high"},
			ControlProtocol:  ReasoningControlProtocolOpenAIReasoningEffort,
		},
		Notes: "OpenAI GPT-5 models before gpt-5.1 do not support none.",
		Sources: []string{
			"https://platform.openai.com/docs/api-reference/chat/create-chat-completion",
			"https://platform.openai.com/docs/models/gpt-5",
		},
		VerifiedAt: "2026-04-17",
	},
	{
		Match: ModelRuntimeProfileMatch{
			Compatibility: domainproviders.ProviderCompatibilityDeepSeek,
			ModelPrefix:   "deepseek-chat",
		},
		Reasoning: &ModelReasoningProfile{
			Supported:        true,
			DisableSupported: true,
			ControlProtocol:  ReasoningControlProtocolThinkingToggle,
		},
		Notes: "DeepSeek chat completion supports thinking enabled/disabled via thinking.type.",
		Sources: []string{
			"https://api-docs.deepseek.com/guides/thinking_mode",
			"https://api-docs.deepseek.com/api/create-chat-completion",
		},
		VerifiedAt: "2026-04-17",
	},
	{
		Match: ModelRuntimeProfileMatch{
			Compatibility: domainproviders.ProviderCompatibilityOpenAI,
			ModelPrefix:   "deepseek-chat",
		},
		Reasoning: &ModelReasoningProfile{
			Supported:        true,
			DisableSupported: true,
			ControlProtocol:  ReasoningControlProtocolThinkingToggle,
		},
		Notes: "DeepSeek models exposed through generic OpenAI-compatible gateways still use thinking.type enabled/disabled when the upstream preserves DeepSeek semantics.",
		Sources: []string{
			"https://api-docs.deepseek.com/guides/thinking_mode",
			"https://api-docs.deepseek.com/api/create-chat-completion",
		},
		VerifiedAt: "2026-04-17",
	},
	{
		Match: ModelRuntimeProfileMatch{
			Compatibility: domainproviders.ProviderCompatibilityGoogle,
			ModelPrefix:   "gemini-2.5-pro",
		},
		Reasoning: &ModelReasoningProfile{
			Supported:        true,
			DisableSupported: false,
			EffortLevels:     []string{"minimal", "low", "medium", "high"},
			ControlProtocol:  ReasoningControlProtocolOpenAIReasoningEffort,
		},
		Notes: "Gemini OpenAI compatibility: Gemini 2.5 Pro cannot disable thinking.",
		Sources: []string{
			"https://ai.google.dev/gemini-api/docs/openai",
			"https://ai.google.dev/gemini-api/docs/thinking",
		},
		VerifiedAt: "2026-04-17",
	},
	{
		Match: ModelRuntimeProfileMatch{
			Compatibility: domainproviders.ProviderCompatibilityGoogle,
			ModelPrefix:   "gemini-2.5-flash-lite",
		},
		Reasoning: &ModelReasoningProfile{
			Supported:        true,
			DisableSupported: true,
			EffortLevels:     []string{"minimal", "low", "medium", "high"},
			ControlProtocol:  ReasoningControlProtocolOpenAIReasoningEffort,
		},
		Notes: "Gemini OpenAI compatibility: Gemini 2.5 Flash Lite can disable thinking with reasoning_effort none.",
		Sources: []string{
			"https://ai.google.dev/gemini-api/docs/openai",
			"https://ai.google.dev/gemini-api/docs/thinking",
		},
		VerifiedAt: "2026-04-17",
	},
	{
		Match: ModelRuntimeProfileMatch{
			Compatibility: domainproviders.ProviderCompatibilityGoogle,
			ModelPrefix:   "gemini-2.5-flash",
		},
		Reasoning: &ModelReasoningProfile{
			Supported:        true,
			DisableSupported: true,
			EffortLevels:     []string{"minimal", "low", "medium", "high"},
			ControlProtocol:  ReasoningControlProtocolOpenAIReasoningEffort,
		},
		Notes: "Gemini OpenAI compatibility: Gemini 2.5 Flash can disable thinking with reasoning_effort none.",
		Sources: []string{
			"https://ai.google.dev/gemini-api/docs/openai",
			"https://ai.google.dev/gemini-api/docs/thinking",
		},
		VerifiedAt: "2026-04-17",
	},
	{
		Match: ModelRuntimeProfileMatch{
			Compatibility: domainproviders.ProviderCompatibilityGoogle,
			ModelPrefix:   "gemini-3",
		},
		Reasoning: &ModelReasoningProfile{
			Supported:        true,
			DisableSupported: false,
			EffortLevels:     []string{"minimal", "low", "medium", "high"},
			ControlProtocol:  ReasoningControlProtocolOpenAIReasoningEffort,
		},
		Notes: "Gemini 3 OpenAI compatibility uses reasoning_effort mapping but cannot disable thinking.",
		Sources: []string{
			"https://ai.google.dev/gemini-api/docs/openai",
			"https://ai.google.dev/gemini-api/docs/thinking",
		},
		VerifiedAt: "2026-04-17",
	},
	{
		Match: ModelRuntimeProfileMatch{
			Compatibility: domainproviders.ProviderCompatibilityAnthropic,
			ModelPrefix:   "claude-opus-4",
		},
		Reasoning: &ModelReasoningProfile{
			Supported:            true,
			DisableSupported:     false,
			SupportsBudgetTokens: true,
			ControlProtocol:      ReasoningControlProtocolAnthropicThinking,
		},
		Notes: "Anthropic OpenAI compatibility enables extended thinking with thinking.enabled plus budget_tokens; reasoning_effort is ignored.",
		Sources: []string{
			"https://docs.anthropic.com/en/api/openai-sdk",
			"https://docs.anthropic.com/en/docs/build-with-claude/extended-thinking",
		},
		VerifiedAt: "2026-04-17",
	},
	{
		Match: ModelRuntimeProfileMatch{
			Compatibility: domainproviders.ProviderCompatibilityAnthropic,
			ModelPrefix:   "claude-sonnet-4",
		},
		Reasoning: &ModelReasoningProfile{
			Supported:            true,
			DisableSupported:     false,
			SupportsBudgetTokens: true,
			ControlProtocol:      ReasoningControlProtocolAnthropicThinking,
		},
		Notes: "Anthropic OpenAI compatibility enables extended thinking with thinking.enabled plus budget_tokens; reasoning_effort is ignored.",
		Sources: []string{
			"https://docs.anthropic.com/en/api/openai-sdk",
			"https://docs.anthropic.com/en/docs/build-with-claude/extended-thinking",
		},
		VerifiedAt: "2026-04-17",
	},
	{
		Match: ModelRuntimeProfileMatch{
			Compatibility: domainproviders.ProviderCompatibilityAnthropic,
			ModelPrefix:   "claude-3-7-sonnet",
		},
		Reasoning: &ModelReasoningProfile{
			Supported:            true,
			DisableSupported:     false,
			SupportsBudgetTokens: true,
			ControlProtocol:      ReasoningControlProtocolAnthropicThinking,
		},
		Notes: "Anthropic OpenAI compatibility enables extended thinking with thinking.enabled plus budget_tokens; reasoning_effort is ignored.",
		Sources: []string{
			"https://docs.anthropic.com/en/api/openai-sdk",
			"https://docs.anthropic.com/en/docs/build-with-claude/extended-thinking",
		},
		VerifiedAt: "2026-04-17",
	},
	{
		Match: ModelRuntimeProfileMatch{
			Compatibility: domainproviders.ProviderCompatibilityOpenAI,
			ModelPrefix:   "glm-",
		},
		Reasoning: &ModelReasoningProfile{
			Supported:        true,
			DisableSupported: true,
			ControlProtocol:  ReasoningControlProtocolThinkingToggle,
		},
		Notes: "Zhipu GLM supports thinking.type enabled/disabled on chat completions.",
		Sources: []string{
			"https://docs.bigmodel.cn/cn/guide/capabilities/thinking",
			"https://docs.bigmodel.cn/cn/guide/capabilities/thinking-mode",
		},
		VerifiedAt: "2026-04-17",
	},
	{
		Match: ModelRuntimeProfileMatch{
			Compatibility: domainproviders.ProviderCompatibilityOpenAI,
			ModelPrefix:   "kimi-k2.5",
		},
		Reasoning: &ModelReasoningProfile{
			Supported:        true,
			DisableSupported: true,
			ControlProtocol:  ReasoningControlProtocolThinkingToggle,
		},
		Notes: "Kimi K2.5 supports thinking.type enabled/disabled.",
		Sources: []string{
			"https://platform.kimi.ai/docs/guide/kimi-k2-5-quickstart",
			"https://platform.kimi.ai/docs/guide/use-kimi-k2-thinking-model.en-US",
		},
		VerifiedAt: "2026-04-17",
	},
	{
		Match: ModelRuntimeProfileMatch{
			Compatibility: domainproviders.ProviderCompatibilityOpenAI,
			ModelPrefix:   "kimi-k2-thinking",
		},
		Reasoning: &ModelReasoningProfile{
			Supported:         true,
			DisableSupported:  false,
			ControlProtocol:   ReasoningControlProtocolThinkingToggle,
			ReasoningRequired: true,
		},
		Notes: "Dedicated Kimi thinking models always think and do not expose a disable switch.",
		Sources: []string{
			"https://platform.kimi.ai/docs/guide/use-kimi-k2-thinking-model.en-US",
		},
		VerifiedAt: "2026-04-17",
	},
	{
		Match: ModelRuntimeProfileMatch{
			Compatibility: domainproviders.ProviderCompatibilityOpenAI,
			ModelPrefix:   "qwen3-next-",
		},
		Reasoning: &ModelReasoningProfile{
			Supported:         true,
			DisableSupported:  false,
			ControlProtocol:   ReasoningControlProtocolQwenThinkingToggle,
			ReasoningRequired: true,
		},
		Notes: "Qwen thinking-only models do not support enable_thinking=false.",
		Sources: []string{
			"https://help.aliyun.com/zh/model-studio/deep-thinking",
			"https://help.aliyun.com/document_detail/2975515.html",
		},
		VerifiedAt: "2026-04-17",
	},
	{
		Match: ModelRuntimeProfileMatch{
			Compatibility: domainproviders.ProviderCompatibilityOpenAI,
			ModelPrefix:   "qwen3-",
		},
		Reasoning: &ModelReasoningProfile{
			Supported:        true,
			DisableSupported: true,
			ControlProtocol:  ReasoningControlProtocolQwenThinkingToggle,
		},
		Notes: "Qwen mixed-thinking models use enable_thinking, with some families defaulting on and some off.",
		Sources: []string{
			"https://help.aliyun.com/zh/model-studio/deep-thinking",
			"https://help.aliyun.com/zh/model-studio/vision",
			"https://help.aliyun.com/document_detail/2975515.html",
		},
		VerifiedAt: "2026-04-17",
	},
	{
		Match: ModelRuntimeProfileMatch{
			Compatibility: domainproviders.ProviderCompatibilityOpenAI,
			ModelPrefix:   "qwen-plus",
		},
		Reasoning: &ModelReasoningProfile{
			Supported:        true,
			DisableSupported: true,
			ControlProtocol:  ReasoningControlProtocolQwenThinkingToggle,
		},
		Notes: "Commercial Qwen Plus models expose enable_thinking in DashScope OpenAI compatibility.",
		Sources: []string{
			"https://help.aliyun.com/document_detail/2975515.html",
			"https://help.aliyun.com/zh/model-studio/deep-thinking",
		},
		VerifiedAt: "2026-04-17",
	},
	{
		Match: ModelRuntimeProfileMatch{
			Compatibility: domainproviders.ProviderCompatibilityOpenAI,
			ModelPrefix:   "qwen-flash",
		},
		Reasoning: &ModelReasoningProfile{
			Supported:        true,
			DisableSupported: true,
			ControlProtocol:  ReasoningControlProtocolQwenThinkingToggle,
		},
		Notes: "Commercial Qwen Flash models expose enable_thinking in DashScope OpenAI compatibility.",
		Sources: []string{
			"https://help.aliyun.com/document_detail/2975515.html",
			"https://help.aliyun.com/zh/model-studio/deep-thinking",
		},
		VerifiedAt: "2026-04-17",
	},
	{
		Match: ModelRuntimeProfileMatch{
			Compatibility: domainproviders.ProviderCompatibilityOpenAI,
			ModelPrefix:   "qwen-turbo",
		},
		Reasoning: &ModelReasoningProfile{
			Supported:        true,
			DisableSupported: true,
			ControlProtocol:  ReasoningControlProtocolQwenThinkingToggle,
		},
		Notes: "Commercial Qwen Turbo models expose enable_thinking in DashScope OpenAI compatibility.",
		Sources: []string{
			"https://help.aliyun.com/document_detail/2975515.html",
			"https://help.aliyun.com/zh/model-studio/deep-thinking",
		},
		VerifiedAt: "2026-04-17",
	},
	{
		Match: ModelRuntimeProfileMatch{
			Compatibility: domainproviders.ProviderCompatibilityOpenAI,
			ModelPrefix:   "qwq",
		},
		Reasoning: &ModelReasoningProfile{
			Supported:         true,
			DisableSupported:  false,
			ControlProtocol:   ReasoningControlProtocolQwenThinkingToggle,
			ReasoningRequired: true,
		},
		Notes: "QwQ models are thinking-only in DashScope OpenAI compatibility.",
		Sources: []string{
			"https://help.aliyun.com/document_detail/2975515.html",
		},
		VerifiedAt: "2026-04-17",
	},
	{
		Match: ModelRuntimeProfileMatch{
			Compatibility: domainproviders.ProviderCompatibilityOpenAI,
			ModelExact:    "grok-4.20-reasoning",
		},
		Reasoning: &ModelReasoningProfile{
			Supported:         true,
			DisableSupported:  false,
			ControlProtocol:   ReasoningControlProtocolUnknown,
			ReasoningRequired: true,
		},
		Notes: "xAI reasoning models reason automatically and reject reasoning_effort.",
		Sources: []string{
			"https://docs.x.ai/developers/model-capabilities/text/reasoning",
			"https://docs.x.ai/docs/models",
		},
		VerifiedAt: "2026-04-17",
	},
	{
		Match: ModelRuntimeProfileMatch{
			Compatibility: domainproviders.ProviderCompatibilityOpenAI,
			ModelExact:    "grok-4-1-fast-reasoning",
		},
		Reasoning: &ModelReasoningProfile{
			Supported:         true,
			DisableSupported:  false,
			ControlProtocol:   ReasoningControlProtocolUnknown,
			ReasoningRequired: true,
		},
		Notes: "xAI reasoning models reason automatically and reject reasoning_effort.",
		Sources: []string{
			"https://docs.x.ai/developers/model-capabilities/text/reasoning",
			"https://docs.x.ai/docs/models",
		},
		VerifiedAt: "2026-04-17",
	},
	{
		Match: ModelRuntimeProfileMatch{
			Compatibility: domainproviders.ProviderCompatibilityOpenRouter,
			ModelPrefix:   "gpt-5",
		},
		Reasoning: &ModelReasoningProfile{
			Supported:        true,
			DisableSupported: true,
			EffortLevels:     []string{"minimal", "low", "medium", "high", "xhigh"},
			ControlProtocol:  ReasoningControlProtocolOpenRouterReasoning,
		},
		Notes: "OpenRouter normalizes reasoning control for OpenAI reasoning models with reasoning.effort.",
		Sources: []string{
			"https://openrouter.ai/docs/guides/best-practices/reasoning-tokens",
		},
		VerifiedAt: "2026-04-17",
	},
	{
		Match: ModelRuntimeProfileMatch{
			Compatibility: domainproviders.ProviderCompatibilityOpenRouter,
			ModelPrefix:   "claude-opus-4",
		},
		Reasoning: &ModelReasoningProfile{
			Supported:        true,
			DisableSupported: true,
			EffortLevels:     []string{"minimal", "low", "medium", "high", "xhigh"},
			ControlProtocol:  ReasoningControlProtocolOpenRouterReasoning,
		},
		Notes: "OpenRouter normalizes reasoning control for Anthropic reasoning models.",
		Sources: []string{
			"https://openrouter.ai/docs/guides/best-practices/reasoning-tokens",
		},
		VerifiedAt: "2026-04-17",
	},
	{
		Match: ModelRuntimeProfileMatch{
			Compatibility: domainproviders.ProviderCompatibilityOpenRouter,
			ModelPrefix:   "claude-sonnet-4",
		},
		Reasoning: &ModelReasoningProfile{
			Supported:        true,
			DisableSupported: true,
			EffortLevels:     []string{"minimal", "low", "medium", "high", "xhigh"},
			ControlProtocol:  ReasoningControlProtocolOpenRouterReasoning,
		},
		Notes: "OpenRouter normalizes reasoning control for Anthropic reasoning models.",
		Sources: []string{
			"https://openrouter.ai/docs/guides/best-practices/reasoning-tokens",
		},
		VerifiedAt: "2026-04-17",
	},
	{
		Match: ModelRuntimeProfileMatch{
			Compatibility: domainproviders.ProviderCompatibilityOpenRouter,
			ModelPrefix:   "claude-3-7-sonnet",
		},
		Reasoning: &ModelReasoningProfile{
			Supported:        true,
			DisableSupported: true,
			EffortLevels:     []string{"minimal", "low", "medium", "high", "xhigh"},
			ControlProtocol:  ReasoningControlProtocolOpenRouterReasoning,
		},
		Notes: "OpenRouter normalizes reasoning control for Anthropic reasoning models.",
		Sources: []string{
			"https://openrouter.ai/docs/guides/best-practices/reasoning-tokens",
		},
		VerifiedAt: "2026-04-17",
	},
	{
		Match: ModelRuntimeProfileMatch{
			Compatibility: domainproviders.ProviderCompatibilityOpenRouter,
			ModelPrefix:   "gemini-2.5",
		},
		Reasoning: &ModelReasoningProfile{
			Supported:        true,
			DisableSupported: true,
			EffortLevels:     []string{"minimal", "low", "medium", "high", "xhigh"},
			ControlProtocol:  ReasoningControlProtocolOpenRouterReasoning,
		},
		Notes: "OpenRouter maps reasoning control for Gemini thinking models.",
		Sources: []string{
			"https://openrouter.ai/docs/guides/best-practices/reasoning-tokens",
		},
		VerifiedAt: "2026-04-17",
	},
	{
		Match: ModelRuntimeProfileMatch{
			Compatibility: domainproviders.ProviderCompatibilityOpenRouter,
			ModelPrefix:   "gemini-3",
		},
		Reasoning: &ModelReasoningProfile{
			Supported:        true,
			DisableSupported: true,
			EffortLevels:     []string{"minimal", "low", "medium", "high", "xhigh"},
			ControlProtocol:  ReasoningControlProtocolOpenRouterReasoning,
		},
		Notes: "OpenRouter maps reasoning control for Gemini thinking models.",
		Sources: []string{
			"https://openrouter.ai/docs/guides/best-practices/reasoning-tokens",
		},
		VerifiedAt: "2026-04-17",
	},
	{
		Match: ModelRuntimeProfileMatch{
			Compatibility: domainproviders.ProviderCompatibilityOpenRouter,
			ModelPrefix:   "grok-",
		},
		Reasoning: &ModelReasoningProfile{
			Supported:        true,
			DisableSupported: true,
			EffortLevels:     []string{"minimal", "low", "medium", "high", "xhigh"},
			ControlProtocol:  ReasoningControlProtocolOpenRouterReasoning,
		},
		Notes: "OpenRouter documents reasoning.effort support for Grok models.",
		Sources: []string{
			"https://openrouter.ai/docs/guides/best-practices/reasoning-tokens",
		},
		VerifiedAt: "2026-04-17",
	},
}
