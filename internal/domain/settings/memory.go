package settings

import "strings"

type MemorySettings struct {
	Enabled           bool    `json:"enabled"`
	EmbeddingProvider string  `json:"embeddingProviderId"`
	EmbeddingModel    string  `json:"embeddingModel"`
	LLMProvider       string  `json:"llmProviderId"`
	LLMModel          string  `json:"llmModel"`
	RecallTopK        int     `json:"recallTopK"`
	VectorWeight      float64 `json:"vectorWeight"`
	TextWeight        float64 `json:"textWeight"`
	RecencyWeight     float64 `json:"recencyWeight"`
	RecencyHalfLife   float64 `json:"recencyHalfLifeDays"`
	MinScore          float64 `json:"minScore"`
	AutoRecall        bool    `json:"autoRecall"`
	AutoCapture       bool    `json:"autoCapture"`
	SessionLifecycle  bool    `json:"sessionLifecycle"`
	CaptureMaxEntries int     `json:"captureMaxEntries"`
}

type MemorySettingsParams struct {
	Enabled           bool    `json:"enabled"`
	EmbeddingProvider string  `json:"embeddingProviderId"`
	EmbeddingModel    string  `json:"embeddingModel"`
	LLMProvider       string  `json:"llmProviderId"`
	LLMModel          string  `json:"llmModel"`
	RecallTopK        int     `json:"recallTopK"`
	VectorWeight      float64 `json:"vectorWeight"`
	TextWeight        float64 `json:"textWeight"`
	RecencyWeight     float64 `json:"recencyWeight"`
	RecencyHalfLife   float64 `json:"recencyHalfLifeDays"`
	MinScore          float64 `json:"minScore"`
	AutoRecall        bool    `json:"autoRecall"`
	AutoCapture       bool    `json:"autoCapture"`
	SessionLifecycle  bool    `json:"sessionLifecycle"`
	CaptureMaxEntries int     `json:"captureMaxEntries"`
}

func DefaultMemorySettings() MemorySettings {
	return MemorySettings{
		Enabled:           true,
		EmbeddingProvider: "",
		EmbeddingModel:    "",
		LLMProvider:       "",
		LLMModel:          "",
		RecallTopK:        5,
		VectorWeight:      0.7,
		TextWeight:        0.3,
		RecencyWeight:     0.15,
		RecencyHalfLife:   14,
		MinScore:          0.35,
		AutoRecall:        true,
		AutoCapture:       true,
		SessionLifecycle:  true,
		CaptureMaxEntries: 3,
	}
}

func ResolveMemorySettings(params MemorySettingsParams) MemorySettings {
	defaults := DefaultMemorySettings()
	settings := MemorySettings{
		Enabled:           params.Enabled,
		EmbeddingProvider: strings.TrimSpace(params.EmbeddingProvider),
		EmbeddingModel:    strings.TrimSpace(params.EmbeddingModel),
		LLMProvider:       strings.TrimSpace(params.LLMProvider),
		LLMModel:          strings.TrimSpace(params.LLMModel),
		RecallTopK:        params.RecallTopK,
		VectorWeight:      params.VectorWeight,
		TextWeight:        params.TextWeight,
		RecencyWeight:     params.RecencyWeight,
		RecencyHalfLife:   params.RecencyHalfLife,
		MinScore:          params.MinScore,
		AutoRecall:        params.AutoRecall,
		AutoCapture:       params.AutoCapture,
		SessionLifecycle:  params.SessionLifecycle,
		CaptureMaxEntries: params.CaptureMaxEntries,
	}

	if settings.RecallTopK <= 0 || settings.RecallTopK > 50 {
		settings.RecallTopK = defaults.RecallTopK
	}
	if settings.CaptureMaxEntries <= 0 || settings.CaptureMaxEntries > 20 {
		settings.CaptureMaxEntries = defaults.CaptureMaxEntries
	}
	if settings.VectorWeight < 0 {
		settings.VectorWeight = 0
	}
	if settings.TextWeight < 0 {
		settings.TextWeight = 0
	}
	if settings.RecencyWeight < 0 {
		settings.RecencyWeight = 0
	}
	if settings.RecencyWeight > 1 {
		settings.RecencyWeight = 1
	}
	if settings.RecencyHalfLife <= 0 || settings.RecencyHalfLife > 365 {
		settings.RecencyHalfLife = defaults.RecencyHalfLife
	}
	if settings.VectorWeight == 0 && settings.TextWeight == 0 {
		settings.VectorWeight = defaults.VectorWeight
		settings.TextWeight = defaults.TextWeight
	}
	if settings.MinScore < 0 || settings.MinScore > 1 {
		settings.MinScore = defaults.MinScore
	}
	return settings
}
