package voice

import (
	"context"
	"time"
)

type TTSProviderCatalogItem struct {
	ProviderID   string `json:"providerId"`
	DisplayName  string `json:"displayName"`
	Available    bool   `json:"available"`
	RequiresAuth bool   `json:"requiresAuth,omitempty"`
}

type TTSConfig struct {
	ProviderID string `json:"providerId,omitempty"`
	VoiceID    string `json:"voiceId,omitempty"`
	ModelID    string `json:"modelId,omitempty"`
	Format     string `json:"format,omitempty"`
	APIKey     string `json:"apiKey,omitempty"`
}

type TTSStatusResponse struct {
	Enabled   bool                     `json:"enabled"`
	Providers []TTSProviderCatalogItem `json:"providers"`
	Config    TTSConfig                `json:"config"`
}

type TTSConvertRequest struct {
	RequestID  string `json:"requestId,omitempty"`
	Text       string `json:"text"`
	ProviderID string `json:"providerId,omitempty"`
	VoiceID    string `json:"voiceId,omitempty"`
	ModelID    string `json:"modelId,omitempty"`
	Format     string `json:"format,omitempty"`
	Channel    string `json:"channel,omitempty"`
}

type TTSMediaArtifact struct {
	ArtifactID  string `json:"artifactId"`
	ProviderID  string `json:"providerId"`
	VoiceID     string `json:"voiceId,omitempty"`
	Format      string `json:"format"`
	ContentType string `json:"contentType"`
	Path        string `json:"path,omitempty"`
	SizeBytes   int    `json:"sizeBytes"`
	DurationMs  int    `json:"durationMs,omitempty"`
}

type TTSConvertResponse struct {
	JobID      string           `json:"jobId"`
	Artifact   TTSMediaArtifact `json:"artifact"`
	CostMicros int64            `json:"costMicros"`
}

type TalkConfig struct {
	VoiceID           string            `json:"voiceId,omitempty"`
	VoiceAliases      map[string]string `json:"voiceAliases,omitempty"`
	ModelID           string            `json:"modelId,omitempty"`
	OutputFormat      string            `json:"outputFormat,omitempty"`
	APIKey            string            `json:"apiKey,omitempty"`
	InterruptOnSpeech *bool             `json:"interruptOnSpeech,omitempty"`
}

type TalkConfigRequest struct {
	IncludeSecrets bool `json:"includeSecrets,omitempty"`
}

type TalkSessionConfig struct {
	MainKey string `json:"mainKey,omitempty"`
}

type TalkUIConfig struct {
	SeamColor string `json:"seamColor,omitempty"`
}

type TalkConfigEnvelope struct {
	Talk    *TalkConfig        `json:"talk,omitempty"`
	Session *TalkSessionConfig `json:"session,omitempty"`
	UI      *TalkUIConfig      `json:"ui,omitempty"`
}

type TalkConfigResponse struct {
	Config TalkConfigEnvelope `json:"config"`
}

type TalkConfigSetRequest struct {
	Config TalkConfigEnvelope `json:"config"`
}

type TalkModeRequest struct {
	Enabled bool   `json:"enabled"`
	Phase   string `json:"phase,omitempty"`
}

type TalkModeResponse struct {
	Enabled       bool      `json:"enabled"`
	Phase         string    `json:"phase,omitempty"`
	VoiceLocked   bool      `json:"voiceLocked"`
	LockedVoiceID string    `json:"lockedVoiceId,omitempty"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type VoiceWakeGetResponse struct {
	Version  int      `json:"version"`
	Triggers []string `json:"triggers"`
}

type VoiceWakeSetRequest struct {
	Triggers []string `json:"triggers"`
}

type VoiceWakeSetResponse struct {
	Version  int      `json:"version"`
	Triggers []string `json:"triggers"`
}

type VoiceWakeChangedEvent struct {
	Version   int       `json:"version"`
	Triggers  []string  `json:"triggers"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type VoiceConfig struct {
	Version   int
	Triggers  []string
	TTS       TTSConfig
	Talk      TalkConfig
	UpdatedAt time.Time
}

type TTSJob struct {
	ID         string
	ProviderID string
	VoiceID    string
	ModelID    string
	Format     string
	Status     string
	InputText  string
	OutputJSON string
	CostMicros int64
	CreatedAt  time.Time
}

type ConfigRepository interface {
	Get(ctx context.Context) (VoiceConfig, error)
	Save(ctx context.Context, config VoiceConfig) error
}

type JobRepository interface {
	Save(ctx context.Context, job TTSJob) error
}
