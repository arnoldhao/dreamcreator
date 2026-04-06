package voice

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"dreamcreator/internal/application/gateway/controlplane"
	gatewayusage "dreamcreator/internal/application/gateway/usage"
	settingsservice "dreamcreator/internal/application/settings/service"
)

const (
	openAITTSEndpoint  = "https://api.openai.com/v1/audio/speech"
	elevenLabsEndpoint = "https://api.elevenlabs.io/v1/text-to-speech"
	defaultTTSModelID  = "gpt-4o-mini-tts"
	defaultTTSVoiceID  = "alloy"
)

var defaultTTSHTTPClient = &http.Client{Timeout: 30 * time.Second}

type Service struct {
	configRepo   ConfigRepository
	jobRepo      JobRepository
	usageService *gatewayusage.Service
	settings     *settingsservice.SettingsService
	publisher    controlplane.EventPublisher

	mu        sync.Mutex
	talkState TalkModeResponse
	now       func() time.Time
	newID     func() string
}

func NewService(configRepo ConfigRepository, jobRepo JobRepository, usageService *gatewayusage.Service, settings *settingsservice.SettingsService, publisher controlplane.EventPublisher) *Service {
	return &Service{
		configRepo:   configRepo,
		jobRepo:      jobRepo,
		usageService: usageService,
		settings:     settings,
		publisher:    publisher,
		talkState: TalkModeResponse{
			Enabled:   false,
			Phase:     "idle",
			UpdatedAt: time.Now(),
		},
		now:   time.Now,
		newID: uuid.NewString,
	}
}

func (service *Service) Status(ctx context.Context) (TTSStatusResponse, error) {
	config, err := service.loadConfig(ctx)
	if err != nil {
		return TTSStatusResponse{}, err
	}
	enabled := service.voiceEnabled(ctx)
	sanitized := config.TTS
	if sanitized.APIKey != "" {
		sanitized.APIKey = ""
	}
	return TTSStatusResponse{
		Enabled: enabled,
		Providers: []TTSProviderCatalogItem{
			{ProviderID: "openai", DisplayName: "OpenAI", Available: config.TTS.APIKey != "", RequiresAuth: true},
			{ProviderID: "elevenlabs", DisplayName: "ElevenLabs", Available: config.TTS.APIKey != "", RequiresAuth: true},
			{ProviderID: "edge", DisplayName: "Edge-TTS", Available: true},
		},
		Config: sanitized,
	}, nil
}

func (service *Service) SetTTSConfig(ctx context.Context, config TTSConfig) (TTSStatusResponse, error) {
	current, err := service.loadConfig(ctx)
	if err != nil {
		return TTSStatusResponse{}, err
	}
	current.TTS = config
	if err := service.saveConfig(ctx, current); err != nil {
		return TTSStatusResponse{}, err
	}
	return service.Status(ctx)
}

func (service *Service) Convert(ctx context.Context, request TTSConvertRequest) (TTSConvertResponse, error) {
	if strings.TrimSpace(request.Text) == "" {
		return TTSConvertResponse{}, errors.New("tts.convert text is required")
	}
	if !service.voiceEnabled(ctx) {
		return TTSConvertResponse{}, errors.New("voice is disabled")
	}
	config, err := service.loadConfig(ctx)
	if err != nil {
		return TTSConvertResponse{}, err
	}
	providerID := strings.TrimSpace(request.ProviderID)
	if providerID == "" {
		providerID = strings.TrimSpace(config.TTS.ProviderID)
	}
	if providerID == "" {
		providerID = "edge"
	}
	voiceID := strings.TrimSpace(request.VoiceID)
	if voiceID == "" {
		voiceID = strings.TrimSpace(config.TTS.VoiceID)
	}
	format := resolveFormat(request.Format, config.TTS.Format, request.Channel)
	format = normalizeProviderFormat(providerID, format, request.Channel)
	adapter, err := resolveProviderAdapter(providerID, config.TTS)
	if err != nil {
		return TTSConvertResponse{}, err
	}
	artifact, cost, err := adapter.Convert(ctx, request, providerID, voiceID, format, config.TTS)
	if err != nil {
		return TTSConvertResponse{}, err
	}
	jobID := service.newID()
	artifact.ArtifactID = jobID
	artifact.ProviderID = providerID
	artifact.VoiceID = voiceID
	artifact.Format = format
	artifact.ContentType = resolveContentType(format)
	if service.jobRepo != nil {
		outputJSON, _ := json.Marshal(artifact)
		_ = service.jobRepo.Save(ctx, TTSJob{
			ID:         jobID,
			ProviderID: providerID,
			VoiceID:    voiceID,
			ModelID:    strings.TrimSpace(request.ModelID),
			Format:     format,
			Status:     "completed",
			InputText:  request.Text,
			OutputJSON: string(outputJSON),
			CostMicros: cost,
			CreatedAt:  service.now(),
		})
	}
	if service.usageService != nil {
		_ = service.usageService.Ingest(ctx, gatewayusage.LedgerEntry{
			ID:            jobID,
			Category:      gatewayusage.CategoryTTS,
			ProviderID:    providerID,
			ModelName:     strings.TrimSpace(request.ModelID),
			Channel:       strings.TrimSpace(request.Channel),
			RequestID:     strings.TrimSpace(request.RequestID),
			RequestSource: gatewayusage.RequestSourceRelay,
			Units:         len(request.Text),
			CostMicros:    cost,
			CostBasis:     gatewayusage.CostBasisEstimated,
			CreatedAt:     service.now(),
		})
	}
	service.lockVoiceIfNeeded(voiceID)
	service.publishEvent("tts.completed", map[string]any{
		"jobId":      jobID,
		"providerId": providerID,
		"format":     format,
		"costMicros": cost,
	})
	return TTSConvertResponse{JobID: jobID, Artifact: artifact, CostMicros: cost}, nil
}

func (service *Service) TalkConfig(ctx context.Context, request TalkConfigRequest) (TalkConfigResponse, error) {
	config, err := service.loadConfig(ctx)
	if err != nil {
		return TalkConfigResponse{}, err
	}
	return TalkConfigResponse{Config: buildTalkConfigEnvelope(config.Talk, request.IncludeSecrets)}, nil
}

func (service *Service) SetTalkConfig(ctx context.Context, request TalkConfigSetRequest) (TalkConfigResponse, error) {
	if request.Config.Talk == nil {
		return TalkConfigResponse{}, errors.New("talk config is required")
	}
	current, err := service.loadConfig(ctx)
	if err != nil {
		return TalkConfigResponse{}, err
	}
	current.Talk = normalizeTalkConfig(*request.Config.Talk)
	if err := service.saveConfig(ctx, current); err != nil {
		return TalkConfigResponse{}, err
	}
	return TalkConfigResponse{Config: buildTalkConfigEnvelope(current.Talk, true)}, nil
}

func (service *Service) TalkMode(request TalkModeRequest) TalkModeResponse {
	service.mu.Lock()
	defer service.mu.Unlock()
	service.talkState.Enabled = request.Enabled
	if strings.TrimSpace(request.Phase) != "" {
		service.talkState.Phase = request.Phase
	}
	if !request.Enabled {
		service.talkState.VoiceLocked = false
		service.talkState.LockedVoiceID = ""
		service.talkState.Phase = "idle"
	}
	service.talkState.UpdatedAt = service.now()
	service.publishEvent("talk.phase.changed", service.talkState)
	return service.talkState
}

func (service *Service) VoiceWakeGet(ctx context.Context) (VoiceWakeGetResponse, error) {
	config, err := service.loadConfig(ctx)
	if err != nil {
		return VoiceWakeGetResponse{}, err
	}
	return VoiceWakeGetResponse{Version: config.Version, Triggers: config.Triggers}, nil
}

func (service *Service) VoiceWakeSet(ctx context.Context, request VoiceWakeSetRequest) (VoiceWakeSetResponse, error) {
	if !service.voiceWakeEnabled(ctx) {
		return VoiceWakeSetResponse{}, errors.New("voicewake disabled")
	}
	config, err := service.loadConfig(ctx)
	if err != nil {
		return VoiceWakeSetResponse{}, err
	}
	config.Triggers = normalizeTriggers(request.Triggers)
	config.Version++
	config.UpdatedAt = service.now()
	if err := service.saveConfig(ctx, config); err != nil {
		return VoiceWakeSetResponse{}, err
	}
	event := VoiceWakeChangedEvent{Version: config.Version, Triggers: config.Triggers, UpdatedAt: config.UpdatedAt}
	service.publishEvent("voicewake.changed", event)
	return VoiceWakeSetResponse{Version: config.Version, Triggers: config.Triggers}, nil
}

func (service *Service) loadConfig(ctx context.Context) (VoiceConfig, error) {
	if service.configRepo == nil {
		return DefaultConfig(), nil
	}
	config, err := service.configRepo.Get(ctx)
	if err != nil {
		return VoiceConfig{}, err
	}
	if config.Version == 0 {
		config.Version = 1
	}
	return config, nil
}

func (service *Service) saveConfig(ctx context.Context, config VoiceConfig) error {
	if service.configRepo == nil {
		return nil
	}
	if config.Version == 0 {
		config.Version = 1
	}
	if config.UpdatedAt.IsZero() {
		config.UpdatedAt = service.now()
	}
	return service.configRepo.Save(ctx, config)
}

func normalizeTalkConfig(config TalkConfig) TalkConfig {
	normalized := TalkConfig{
		VoiceID:           strings.TrimSpace(config.VoiceID),
		ModelID:           strings.TrimSpace(config.ModelID),
		OutputFormat:      strings.TrimSpace(config.OutputFormat),
		APIKey:            strings.TrimSpace(config.APIKey),
		InterruptOnSpeech: config.InterruptOnSpeech,
	}
	normalized.VoiceAliases = normalizeVoiceAliases(config.VoiceAliases)
	return normalized
}

func normalizeVoiceAliases(aliases map[string]string) map[string]string {
	if len(aliases) == 0 {
		return nil
	}
	result := make(map[string]string)
	for key, value := range aliases {
		trimmedKey := strings.TrimSpace(key)
		trimmedValue := strings.TrimSpace(value)
		if trimmedKey == "" || trimmedValue == "" {
			continue
		}
		result[trimmedKey] = trimmedValue
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func buildTalkConfigEnvelope(config TalkConfig, includeSecrets bool) TalkConfigEnvelope {
	normalized := normalizeTalkConfig(config)
	if !includeSecrets {
		normalized.APIKey = ""
	}
	if isTalkConfigEmpty(normalized) {
		return TalkConfigEnvelope{}
	}
	return TalkConfigEnvelope{Talk: &normalized}
}

func isTalkConfigEmpty(config TalkConfig) bool {
	if config.VoiceID != "" {
		return false
	}
	if len(config.VoiceAliases) > 0 {
		return false
	}
	if config.ModelID != "" {
		return false
	}
	if config.OutputFormat != "" {
		return false
	}
	if config.APIKey != "" {
		return false
	}
	if config.InterruptOnSpeech != nil {
		return false
	}
	return true
}

func (service *Service) voiceEnabled(ctx context.Context) bool {
	if service.settings == nil {
		return true
	}
	current, err := service.settings.GetSettings(ctx)
	if err != nil {
		return true
	}
	return current.Gateway.VoiceEnabled
}

func (service *Service) voiceWakeEnabled(ctx context.Context) bool {
	if service.settings == nil {
		return true
	}
	current, err := service.settings.GetSettings(ctx)
	if err != nil {
		return true
	}
	return current.Gateway.VoiceWakeEnabled
}

func (service *Service) lockVoiceIfNeeded(voiceID string) {
	service.mu.Lock()
	defer service.mu.Unlock()
	if !service.talkState.Enabled || service.talkState.VoiceLocked {
		return
	}
	service.talkState.VoiceLocked = true
	service.talkState.LockedVoiceID = strings.TrimSpace(voiceID)
	service.talkState.Phase = "speaking"
	service.talkState.UpdatedAt = service.now()
	service.publishEvent("talk.phase.changed", service.talkState)
}

func (service *Service) publishEvent(name string, payload any) {
	if service == nil || service.publisher == nil {
		return
	}
	_ = service.publisher.Publish(controlplane.EventFrame{
		Type:      "event",
		Event:     name,
		Payload:   payload,
		Timestamp: service.now(),
	})
}

func resolveFormat(requested string, fallback string, channel string) string {
	value := strings.TrimSpace(requested)
	if value != "" {
		return value
	}
	value = strings.TrimSpace(fallback)
	if value != "" {
		return value
	}
	switch strings.ToLower(strings.TrimSpace(channel)) {
	case "telegram":
		return "ogg"
	default:
		return "wav"
	}
}

func resolveContentType(format string) string {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "ogg", "opus":
		return "audio/ogg"
	case "mp3":
		return "audio/mpeg"
	default:
		return "audio/wav"
	}
}

func resolveAudioExt(format string) string {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "ogg", "opus":
		return ".ogg"
	case "mp3":
		return ".mp3"
	default:
		return ".wav"
	}
}

func writePlaceholderAudio(format string, text string) (string, int, error) {
	ext := resolveAudioExt(format)
	file, err := os.CreateTemp("", "tts-*"+ext)
	if err != nil {
		return "", 0, err
	}
	defer file.Close()
	if _, err := file.WriteString(text); err != nil {
		return "", 0, err
	}
	info, err := file.Stat()
	if err != nil {
		return file.Name(), len(text), nil
	}
	path, _ := filepath.Abs(file.Name())
	return path, int(info.Size()), nil
}

func writeAudioBytes(format string, data []byte) (string, int, error) {
	ext := resolveAudioExt(format)
	file, err := os.CreateTemp("", "tts-*"+ext)
	if err != nil {
		return "", 0, err
	}
	defer file.Close()
	if _, err := file.Write(data); err != nil {
		return "", 0, err
	}
	info, err := file.Stat()
	if err != nil {
		return file.Name(), len(data), nil
	}
	path, _ := filepath.Abs(file.Name())
	return path, int(info.Size()), nil
}

type providerAdapter interface {
	Convert(ctx context.Context, request TTSConvertRequest, providerID string, voiceID string, format string, config TTSConfig) (TTSMediaArtifact, int64, error)
}

type placeholderAdapter struct {
	requiresKey bool
}

func (adapter placeholderAdapter) Convert(_ context.Context, request TTSConvertRequest, providerID string, voiceID string, format string, config TTSConfig) (TTSMediaArtifact, int64, error) {
	if adapter.requiresKey && strings.TrimSpace(config.APIKey) == "" {
		return TTSMediaArtifact{}, 0, errors.New("tts provider api key missing")
	}
	path, size, err := writePlaceholderAudio(format, request.Text)
	if err != nil {
		return TTSMediaArtifact{}, 0, err
	}
	cost := int64(len(request.Text)) * 10
	return TTSMediaArtifact{
		ArtifactID:  "",
		ProviderID:  providerID,
		VoiceID:     voiceID,
		Format:      format,
		ContentType: resolveContentType(format),
		Path:        path,
		SizeBytes:   size,
	}, cost, nil
}

type openAIAdapter struct {
	client *http.Client
}

func (adapter openAIAdapter) Convert(ctx context.Context, request TTSConvertRequest, providerID string, voiceID string, format string, config TTSConfig) (TTSMediaArtifact, int64, error) {
	apiKey := strings.TrimSpace(config.APIKey)
	if apiKey == "" {
		return TTSMediaArtifact{}, 0, errors.New("tts provider api key missing")
	}
	modelID := strings.TrimSpace(request.ModelID)
	if modelID == "" {
		modelID = strings.TrimSpace(config.ModelID)
	}
	if modelID == "" {
		modelID = defaultTTSModelID
	}
	voice := strings.TrimSpace(voiceID)
	if voice == "" {
		voice = defaultTTSVoiceID
	}
	payload := map[string]any{
		"model": modelID,
		"input": request.Text,
		"voice": voice,
	}
	if respFormat := normalizeOpenAIFormat(format); respFormat != "" {
		payload["response_format"] = respFormat
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return TTSMediaArtifact{}, 0, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, openAITTSEndpoint, bytes.NewReader(body))
	if err != nil {
		return TTSMediaArtifact{}, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	client := adapter.client
	if client == nil {
		client = defaultTTSHTTPClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return TTSMediaArtifact{}, 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return TTSMediaArtifact{}, 0, readHTTPError(resp)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return TTSMediaArtifact{}, 0, err
	}
	path, size, err := writeAudioBytes(format, data)
	if err != nil {
		return TTSMediaArtifact{}, 0, err
	}
	cost := estimateTextCostMicros(providerID, request.Text)
	return TTSMediaArtifact{
		ArtifactID:  "",
		ProviderID:  providerID,
		VoiceID:     voice,
		Format:      format,
		ContentType: resolveContentType(format),
		Path:        path,
		SizeBytes:   size,
	}, cost, nil
}

type elevenLabsAdapter struct {
	client *http.Client
}

func (adapter elevenLabsAdapter) Convert(ctx context.Context, request TTSConvertRequest, providerID string, voiceID string, format string, config TTSConfig) (TTSMediaArtifact, int64, error) {
	apiKey := strings.TrimSpace(config.APIKey)
	if apiKey == "" {
		return TTSMediaArtifact{}, 0, errors.New("tts provider api key missing")
	}
	voice := strings.TrimSpace(voiceID)
	if voice == "" {
		return TTSMediaArtifact{}, 0, errors.New("tts voice id is required")
	}
	payload := map[string]any{
		"text": request.Text,
	}
	modelID := strings.TrimSpace(request.ModelID)
	if modelID == "" {
		modelID = strings.TrimSpace(config.ModelID)
	}
	if modelID != "" {
		payload["model_id"] = modelID
	}
	if outputFormat := normalizeElevenLabsFormat(format); outputFormat != "" {
		payload["output_format"] = outputFormat
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return TTSMediaArtifact{}, 0, err
	}
	endpoint := strings.TrimRight(elevenLabsEndpoint, "/") + "/" + voice
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return TTSMediaArtifact{}, 0, err
	}
	req.Header.Set("xi-api-key", apiKey)
	req.Header.Set("Content-Type", "application/json")
	client := adapter.client
	if client == nil {
		client = defaultTTSHTTPClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return TTSMediaArtifact{}, 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return TTSMediaArtifact{}, 0, readHTTPError(resp)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return TTSMediaArtifact{}, 0, err
	}
	path, size, err := writeAudioBytes(format, data)
	if err != nil {
		return TTSMediaArtifact{}, 0, err
	}
	cost := estimateTextCostMicros(providerID, request.Text)
	return TTSMediaArtifact{
		ArtifactID:  "",
		ProviderID:  providerID,
		VoiceID:     voice,
		Format:      format,
		ContentType: resolveContentType(format),
		Path:        path,
		SizeBytes:   size,
	}, cost, nil
}

func resolveProviderAdapter(providerID string, config TTSConfig) (providerAdapter, error) {
	switch strings.ToLower(strings.TrimSpace(providerID)) {
	case "openai":
		return openAIAdapter{}, nil
	case "elevenlabs":
		return elevenLabsAdapter{}, nil
	case "edge":
		return placeholderAdapter{requiresKey: false}, nil
	default:
		return placeholderAdapter{requiresKey: false}, nil
	}
}

func normalizeOpenAIFormat(format string) string {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "mp3", "wav", "opus", "aac", "flac", "pcm":
		return strings.ToLower(strings.TrimSpace(format))
	case "ogg":
		return "opus"
	default:
		return ""
	}
}

func normalizeElevenLabsFormat(format string) string {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "mp3":
		return "mp3_44100_128"
	case "wav":
		return "wav"
	default:
		return ""
	}
}

func estimateTextCostMicros(providerID string, text string) int64 {
	if strings.TrimSpace(text) == "" {
		return 0
	}
	return int64(len(text)) * 10
}

func readHTTPError(resp *http.Response) error {
	if resp == nil {
		return errors.New("tts request failed")
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8*1024))
	message := strings.TrimSpace(string(body))
	if message == "" {
		message = resp.Status
	}
	return errors.New(message)
}

func normalizeProviderFormat(providerID string, format string, channel string) string {
	format = strings.ToLower(strings.TrimSpace(format))
	if format == "" {
		format = "mp3"
	}
	switch strings.ToLower(strings.TrimSpace(providerID)) {
	case "openai":
		if format != "mp3" && format != "wav" {
			return "mp3"
		}
	case "elevenlabs":
		if format != "mp3" && format != "wav" && format != "ogg" {
			return "mp3"
		}
	case "edge":
		if strings.EqualFold(strings.TrimSpace(channel), "telegram") && format == "mp3" {
			return "ogg"
		}
		if format != "mp3" && format != "wav" && format != "ogg" {
			return "mp3"
		}
	}
	return format
}

func normalizeTriggers(triggers []string) []string {
	result := make([]string, 0, len(triggers))
	seen := make(map[string]struct{})
	for _, raw := range triggers {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			continue
		}
		lower := strings.ToLower(trimmed)
		if _, ok := seen[lower]; ok {
			continue
		}
		seen[lower] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func DefaultConfig() VoiceConfig {
	return VoiceConfig{
		Version:  1,
		Triggers: []string{"hey dreamcreator"},
		TTS: TTSConfig{
			ProviderID: "edge",
			Format:     "wav",
		},
		Talk:      TalkConfig{},
		UpdatedAt: time.Now(),
	}
}
