package llmrecord

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	settingsdto "dreamcreator/internal/application/settings/dto"
	domainsettings "dreamcreator/internal/domain/settings"
	"dreamcreator/internal/infrastructure/llm"
)

const maxPersistedPayloadChars = 128 * 1024

type Repository interface {
	Insert(ctx context.Context, record Record) error
	Update(ctx context.Context, record Record) error
	Get(ctx context.Context, id string) (Record, error)
	List(ctx context.Context, filter QueryFilter) ([]Record, error)
	Delete(ctx context.Context, id string) error
	DeleteStartedBefore(ctx context.Context, cutoff time.Time) (int, error)
	DeleteAll(ctx context.Context) (int, error)
}

type SettingsReader interface {
	GetSettings(ctx context.Context) (settingsdto.Settings, error)
}

type QueryFilter struct {
	ThreadID      string
	RunID         string
	ProviderID    string
	ModelName     string
	RequestSource string
	Status        string
	StartAt       time.Time
	EndAt         time.Time
	Limit         int
}

type Service struct {
	repo     Repository
	settings SettingsReader
	now      func() time.Time
	newID    func() string
	pruneMu  sync.Mutex
}

type Config struct {
	SaveStrategy  domainsettings.GatewayCallRecordSaveStrategy
	RetentionDays int
	AutoCleanup   domainsettings.GatewayCallRecordAutoCleanup
}

func NewService(repo Repository, settings SettingsReader) *Service {
	return &Service{
		repo:     repo,
		settings: settings,
		now:      time.Now,
		newID:    uuid.NewString,
	}
}

func (service *Service) StartLLMCall(ctx context.Context, record llm.CallRecordStart) (string, error) {
	if service == nil || service.repo == nil {
		return "", errors.New("llm call record repository unavailable")
	}
	config := service.resolveConfig(ctx)
	service.maybeAutoCleanupOnWrite(ctx, config)
	if config.SaveStrategy == domainsettings.GatewayCallRecordSaveStrategyOff {
		return "", nil
	}
	startedAt := record.StartedAt.UTC()
	if startedAt.IsZero() {
		startedAt = service.now().UTC()
	}
	requestPayload, requestTruncated := trimPayload(record.RequestPayload)
	responsePayload, responseTruncated := trimPayload(record.ResponsePayload)
	item := Record{
		ID:                  service.newID(),
		SessionID:           strings.TrimSpace(record.SessionID),
		ThreadID:            strings.TrimSpace(record.ThreadID),
		RunID:               strings.TrimSpace(record.RunID),
		ProviderID:          normalizeDimension(record.ProviderID),
		ModelName:           normalizeDimension(record.ModelName),
		RequestSource:       normalizeRequestSource(record.RequestSource),
		Operation:           normalizeOperation(record.Operation),
		Status:              llm.CallRecordStatusStarted,
		RequestPayloadJSON:  requestPayload,
		ResponsePayloadJSON: responsePayload,
		PayloadTruncated:    requestTruncated || responseTruncated,
		StartedAt:           startedAt,
	}
	if err := service.repo.Insert(ctx, item); err != nil {
		return "", err
	}
	return item.ID, nil
}

func (service *Service) FinishLLMCall(ctx context.Context, record llm.CallRecordFinish) error {
	if service == nil || service.repo == nil {
		return errors.New("llm call record repository unavailable")
	}
	id := strings.TrimSpace(record.ID)
	if id == "" {
		return errors.New("llm call record id is required")
	}
	config := service.resolveConfig(ctx)
	status := normalizeStatus(record.Status)
	if config.SaveStrategy == domainsettings.GatewayCallRecordSaveStrategyOff ||
		(config.SaveStrategy == domainsettings.GatewayCallRecordSaveStrategyErrors &&
			status == llm.CallRecordStatusCompleted) {
		return service.repo.Delete(ctx, id)
	}
	item, err := service.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	finishedAt := record.FinishedAt.UTC()
	if finishedAt.IsZero() {
		finishedAt = service.now().UTC()
	}
	responsePayload, responseTruncated := trimPayload(record.ResponsePayload)
	item.Status = status
	item.FinishReason = strings.TrimSpace(record.FinishReason)
	item.ErrorText = strings.TrimSpace(record.ErrorText)
	item.InputTokens = maxInt(record.InputTokens)
	item.OutputTokens = maxInt(record.OutputTokens)
	item.TotalTokens = maxInt(record.TotalTokens)
	item.ContextPromptTokens = maxInt(record.ContextPromptTokens)
	item.ContextTotalTokens = maxInt(record.ContextTotalTokens)
	item.ContextWindowTokens = maxInt(record.ContextWindowTokens)
	if responsePayload != "" {
		item.ResponsePayloadJSON = responsePayload
	}
	item.PayloadTruncated = item.PayloadTruncated || responseTruncated
	item.FinishedAt = finishedAt
	if !item.StartedAt.IsZero() && finishedAt.After(item.StartedAt) {
		item.DurationMS = finishedAt.Sub(item.StartedAt).Milliseconds()
	}
	return service.repo.Update(ctx, item)
}

func (service *Service) Get(ctx context.Context, id string) (Record, error) {
	if service == nil || service.repo == nil {
		return Record{}, errors.New("llm call record repository unavailable")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return Record{}, errors.New("llm call record id is required")
	}
	return service.repo.Get(ctx, id)
}

func (service *Service) List(ctx context.Context, request ListRequest) ([]Record, error) {
	if service == nil || service.repo == nil {
		return nil, errors.New("llm call record repository unavailable")
	}
	startAt, err := parseTimeFilter(request.StartAt)
	if err != nil {
		return nil, errors.New("startAt must be RFC3339")
	}
	endAt, err := parseTimeFilter(request.EndAt)
	if err != nil {
		return nil, errors.New("endAt must be RFC3339")
	}
	limit := request.Limit
	if limit <= 0 {
		limit = 200
	}
	if limit > 500 {
		limit = 500
	}
	items, err := service.repo.List(ctx, QueryFilter{
		ThreadID:      strings.TrimSpace(request.ThreadID),
		RunID:         strings.TrimSpace(request.RunID),
		ProviderID:    normalizeDimension(request.ProviderID),
		ModelName:     normalizeDimension(request.ModelName),
		RequestSource: normalizeRequestSource(request.RequestSource),
		Status:        normalizeStatus(request.Status),
		StartAt:       startAt,
		EndAt:         endAt,
		Limit:         limit,
	})
	if err != nil {
		return nil, err
	}
	for i := range items {
		items[i].RequestPayloadJSON = ""
		items[i].ResponsePayloadJSON = ""
	}
	return items, nil
}

func (service *Service) PruneExpired(ctx context.Context) (int, error) {
	if service == nil || service.repo == nil {
		return 0, errors.New("llm call record repository unavailable")
	}
	return service.pruneExpiredWithConfig(ctx, service.resolveConfig(ctx))
}

func (service *Service) RunScheduledCleanup(ctx context.Context) (int, error) {
	if service == nil || service.repo == nil {
		return 0, errors.New("llm call record repository unavailable")
	}
	config := service.resolveConfig(ctx)
	if config.AutoCleanup != domainsettings.GatewayCallRecordAutoCleanupHourly {
		return 0, nil
	}
	return service.pruneExpiredWithConfig(ctx, config)
}

func (service *Service) Clear(ctx context.Context) (int, error) {
	if service == nil || service.repo == nil {
		return 0, errors.New("llm call record repository unavailable")
	}
	return service.repo.DeleteAll(ctx)
}

func (service *Service) maybeAutoCleanupOnWrite(ctx context.Context, config Config) {
	if config.AutoCleanup != domainsettings.GatewayCallRecordAutoCleanupOnWrite {
		return
	}
	if _, err := service.pruneExpiredWithConfig(ctx, config); err != nil {
		zap.L().Warn("llm call record on-write cleanup failed", zap.Error(err))
	}
}

func (service *Service) resolveConfig(ctx context.Context) Config {
	defaults := domainsettings.DefaultGatewaySettings().Runtime.CallRecords
	config := Config{
		SaveStrategy:  defaults.SaveStrategy,
		RetentionDays: defaults.RetentionDays,
		AutoCleanup:   defaults.AutoCleanup,
	}
	if service == nil || service.settings == nil {
		return config
	}
	current, err := service.settings.GetSettings(ctx)
	if err != nil {
		return config
	}
	config.SaveStrategy = domainsettings.ResolveGatewayCallRecordSaveStrategy(
		current.Gateway.Runtime.CallRecords.SaveStrategy,
	)
	config.RetentionDays = domainsettings.NormalizeGatewayCallRecordRetentionDays(
		current.Gateway.Runtime.CallRecords.RetentionDays,
	)
	config.AutoCleanup = domainsettings.ResolveGatewayCallRecordAutoCleanup(
		current.Gateway.Runtime.CallRecords.AutoCleanup,
	)
	return config
}

func (service *Service) pruneExpiredWithConfig(ctx context.Context, config Config) (int, error) {
	if service == nil || service.repo == nil {
		return 0, errors.New("llm call record repository unavailable")
	}
	retentionDays := domainsettings.NormalizeGatewayCallRecordRetentionDays(config.RetentionDays)
	cutoff := service.now().UTC().AddDate(0, 0, -retentionDays)
	service.pruneMu.Lock()
	defer service.pruneMu.Unlock()
	return service.repo.DeleteStartedBefore(ctx, cutoff)
}

func trimPayload(value string) (string, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", false
	}
	runes := []rune(trimmed)
	if len(runes) <= maxPersistedPayloadChars {
		return trimmed, false
	}
	preview := strings.TrimSpace(string(runes[:maxPersistedPayloadChars]))
	payload, err := json.Marshal(map[string]any{
		"truncated":      true,
		"originalLength": len(runes),
		"preview":        preview,
	})
	if err != nil {
		return `{"truncated":true}`, true
	}
	return string(payload), true
}

func parseTimeFilter(value string) (time.Time, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return time.Time{}, nil
	}
	parsed, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		return time.Time{}, err
	}
	return parsed.UTC(), nil
}

func normalizeDimension(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	return trimmed
}

func normalizeOperation(value string) string {
	return strings.TrimSpace(value)
}

func normalizeRequestSource(value string) string {
	return strings.TrimSpace(value)
}

func normalizeStatus(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case llm.CallRecordStatusStarted:
		return llm.CallRecordStatusStarted
	case llm.CallRecordStatusCompleted:
		return llm.CallRecordStatusCompleted
	case llm.CallRecordStatusCancelled:
		return llm.CallRecordStatusCancelled
	case llm.CallRecordStatusError:
		return llm.CallRecordStatusError
	default:
		return ""
	}
}

func maxInt(value int) int {
	if value < 0 {
		return 0
	}
	return value
}
