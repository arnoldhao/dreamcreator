package telemetry

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"time"

	settingsdto "dreamcreator/internal/application/settings/dto"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type State struct {
	InstallID                 string
	InstallCreatedAt          time.Time
	LaunchCount               int
	FirstProviderConfiguredAt *time.Time
	FirstChatCompletedAt      *time.Time
	FirstLibraryCompletedAt   *time.Time
}

type StateRepository interface {
	Ensure(ctx context.Context) (State, error)
	IncrementLaunchCount(ctx context.Context, at time.Time) (State, error)
	MarkFirstProviderConfigured(ctx context.Context, at time.Time) (State, bool, error)
	MarkFirstChatCompleted(ctx context.Context, at time.Time) (State, bool, error)
	MarkFirstLibraryCompleted(ctx context.Context, at time.Time) (State, bool, error)
}

type Signal struct {
	Type       string         `json:"type"`
	FloatValue *float64       `json:"floatValue,omitempty"`
	Payload    map[string]any `json:"payload,omitempty"`
}

type Bootstrap struct {
	Enabled    bool   `json:"enabled"`
	AppID      string `json:"appId,omitempty"`
	AppVersion string `json:"appVersion,omitempty"`
	InstallID  string `json:"installId,omitempty"`
	SessionID  string `json:"sessionId,omitempty"`
	TestMode   bool   `json:"testMode"`
}

type Emitter interface {
	Emit(signal Signal)
}

type SettingsReader interface {
	GetSettings(ctx context.Context) (settingsdto.Settings, error)
}

type AppLaunchContext struct {
	LaunchedByAutoStart bool
}

type Service struct {
	repo       StateRepository
	emitter    Emitter
	settings   SettingsReader
	appID      string
	appVersion string
	sessionID  string
	startedAt  time.Time
	now        func() time.Time

	mu       sync.Mutex
	launched bool
	flushed  bool
	session  sessionMetrics
	state    *State
	language string
}

type sessionMetrics struct {
	chatCompleted         int
	libraryCompleted      int
	providerConfigured    int
	connectorConnected    int
	externalToolInstalled int
	updateReadyToRestart  int
	runIDs                map[string]struct{}
	operationIDs          map[string]struct{}
}

func NewService(repo StateRepository, emitter Emitter, settings SettingsReader, appID string, appVersion string) *Service {
	return &Service{
		repo:       repo,
		emitter:    emitter,
		settings:   settings,
		appID:      strings.TrimSpace(appID),
		appVersion: strings.TrimSpace(appVersion),
		sessionID:  uuid.NewString(),
		startedAt:  time.Now(),
		now:        time.Now,
		session: sessionMetrics{
			runIDs:       make(map[string]struct{}),
			operationIDs: make(map[string]struct{}),
		},
	}
}

func (service *Service) Enabled() bool {
	return service != nil && service.repo != nil && strings.TrimSpace(service.appID) != ""
}

func (service *Service) Bootstrap(ctx context.Context) (Bootstrap, error) {
	if !service.Enabled() {
		return Bootstrap{
			Enabled:    false,
			AppVersion: normalizeVersion(service.appVersion),
			TestMode:   releaseChannel(service.appVersion) == "dev",
		}, nil
	}
	state, err := service.resolveState(ctx)
	if err != nil {
		return Bootstrap{}, err
	}
	return Bootstrap{
		Enabled:    true,
		AppID:      service.appID,
		AppVersion: normalizeVersion(service.appVersion),
		InstallID:  state.InstallID,
		SessionID:  service.sessionID,
		TestMode:   releaseChannel(service.appVersion) == "dev",
	}, nil
}

func (service *Service) TrackAppLaunch(ctx context.Context, launch AppLaunchContext) (int, error) {
	if !service.Enabled() {
		return 0, nil
	}
	service.mu.Lock()
	if service.launched {
		service.mu.Unlock()
		return 0, nil
	}
	service.launched = true
	service.mu.Unlock()

	state, err := service.repo.IncrementLaunchCount(ctx, service.now())
	if err != nil {
		return 0, err
	}
	service.cacheState(state)

	payload := service.buildPayload(ctx, state)
	payload["DreamCreator.App.launchCount"] = state.LaunchCount
	payload["DreamCreator.App.launchOrdinalBucket"] = bucketLaunchOrdinal(state.LaunchCount)
	payload["DreamCreator.App.launchedByAutoStart"] = launch.LaunchedByAutoStart
	payload["DreamCreator.App.startMode"] = startMode(launch)
	payload["DreamCreator.Install.firstLaunch"] = state.LaunchCount == 1

	signals := []Signal{{
		Type:    "TelemetryDeck.Session.started",
		Payload: payload,
	}}
	if state.LaunchCount == 1 {
		signals = append(signals, Signal{
			Type:    "TelemetryDeck.Acquisition.newInstallDetected",
			Payload: payload,
		})
	}
	for _, signal := range signals {
		service.emit(signal)
	}
	return len(signals), nil
}

func (service *Service) TrackProviderConfigured(ctx context.Context, providerID string) {
	if !service.Enabled() {
		return
	}
	normalizedProviderID := strings.TrimSpace(providerID)
	if normalizedProviderID == "" {
		return
	}

	service.incrementCounter(func(metrics *sessionMetrics) {
		metrics.providerConfigured++
	})

	state, first, err := service.repo.MarkFirstProviderConfigured(ctx, service.now())
	if err != nil {
		zap.L().Debug("telemetry: provider configuration state update failed", zap.Error(err))
		state, err = service.repo.Ensure(ctx)
		if err != nil {
			zap.L().Debug("telemetry: provider configuration state ensure failed", zap.Error(err))
			return
		}
	}
	service.cacheState(state)

	payload := service.buildPayload(ctx, state)
	payload["DreamCreator.Setup.providerId"] = normalizedProviderID
	service.emitAsync(Signal{Type: "DreamCreator.Setup.providerConfigured", Payload: payload})
	if first {
		service.emitAsync(Signal{Type: "DreamCreator.Activation.firstProviderConfigured", Payload: payload})
	}
}

func (service *Service) TrackConnectorConnected(ctx context.Context, connectorType string) {
	if !service.Enabled() {
		return
	}
	normalizedConnectorType := strings.TrimSpace(connectorType)
	if normalizedConnectorType == "" {
		return
	}
	service.incrementCounter(func(metrics *sessionMetrics) {
		metrics.connectorConnected++
	})
	state, err := service.repo.Ensure(ctx)
	if err != nil {
		zap.L().Debug("telemetry: connector state ensure failed", zap.Error(err))
		return
	}
	service.cacheState(state)
	payload := service.buildPayload(ctx, state)
	payload["DreamCreator.Setup.connectorType"] = normalizedConnectorType
	service.emitAsync(Signal{Type: "DreamCreator.Setup.connectorConnected", Payload: payload})
}

func (service *Service) TrackExternalToolInstalled(ctx context.Context, toolName string) {
	if !service.Enabled() {
		return
	}
	normalizedToolName := strings.TrimSpace(toolName)
	if normalizedToolName == "" {
		return
	}
	service.incrementCounter(func(metrics *sessionMetrics) {
		metrics.externalToolInstalled++
	})
	state, err := service.repo.Ensure(ctx)
	if err != nil {
		zap.L().Debug("telemetry: external tool state ensure failed", zap.Error(err))
		return
	}
	service.cacheState(state)
	payload := service.buildPayload(ctx, state)
	payload["DreamCreator.Setup.externalTool"] = normalizedToolName
	service.emitAsync(Signal{Type: "DreamCreator.Setup.externalToolInstalled", Payload: payload})
}

func (service *Service) TrackUserChatCompleted(ctx context.Context, runID string) {
	if !service.Enabled() {
		return
	}
	normalizedRunID := strings.TrimSpace(runID)
	if normalizedRunID == "" {
		return
	}
	if !service.markRunCompleted(normalizedRunID) {
		return
	}

	state, first, err := service.repo.MarkFirstChatCompleted(ctx, service.now())
	if err != nil {
		zap.L().Debug("telemetry: chat completion state update failed", zap.Error(err))
		return
	}
	service.cacheState(state)
	if !first {
		return
	}
	payload := service.buildPayload(ctx, state)
	service.emitAsync(Signal{Type: "DreamCreator.Activation.firstChatCompleted", Payload: payload})
}

func (service *Service) TrackLibraryOperationCompleted(ctx context.Context, operationID string, kind string) {
	if !service.Enabled() {
		return
	}
	normalizedOperationID := strings.TrimSpace(operationID)
	if normalizedOperationID == "" {
		return
	}
	if !service.markLibraryOperationCompleted(normalizedOperationID) {
		return
	}

	state, first, err := service.repo.MarkFirstLibraryCompleted(ctx, service.now())
	if err != nil {
		zap.L().Debug("telemetry: library completion state update failed", zap.Error(err))
		return
	}
	service.cacheState(state)
	if !first {
		return
	}
	payload := service.buildPayload(ctx, state)
	if normalizedKind := strings.TrimSpace(kind); normalizedKind != "" {
		payload["DreamCreator.Library.operationKind"] = normalizedKind
	}
	service.emitAsync(Signal{Type: "DreamCreator.Activation.firstLibraryCompleted", Payload: payload})
}

func (service *Service) TrackUpdateReadyToRestart(ctx context.Context, latestVersion string) {
	if !service.Enabled() {
		return
	}
	normalizedVersion := strings.TrimSpace(latestVersion)
	if normalizedVersion == "" {
		return
	}
	service.incrementCounter(func(metrics *sessionMetrics) {
		metrics.updateReadyToRestart++
	})
	state, err := service.repo.Ensure(ctx)
	if err != nil {
		zap.L().Debug("telemetry: update state ensure failed", zap.Error(err))
		return
	}
	service.cacheState(state)
	payload := service.buildPayload(ctx, state)
	payload["DreamCreator.App.targetVersion"] = normalizedVersion
	service.emitAsync(Signal{Type: "DreamCreator.App.updateReadyToRestart", Payload: payload})
}

func (service *Service) FlushSessionSummary(ctx context.Context) error {
	if !service.Enabled() {
		return nil
	}

	service.mu.Lock()
	if service.flushed || !service.launched {
		service.mu.Unlock()
		return nil
	}
	service.flushed = true
	snapshot := service.session
	startedAt := service.startedAt
	service.mu.Unlock()

	state, err := service.resolveState(ctx)
	if err != nil {
		return err
	}

	durationSeconds := service.now().Sub(startedAt).Seconds()
	if durationSeconds < 0 {
		durationSeconds = 0
	}

	payload := service.buildPayload(ctx, state)
	payload["DreamCreator.Session.durationBucket"] = bucketSessionDuration(time.Duration(durationSeconds * float64(time.Second)))
	payload["DreamCreator.Session.chatCompletedBucket"] = bucketCount(snapshot.chatCompleted)
	payload["DreamCreator.Session.libraryCompletedBucket"] = bucketCount(snapshot.libraryCompleted)
	payload["DreamCreator.Session.providerConfiguredBucket"] = bucketCount(snapshot.providerConfigured)
	payload["DreamCreator.Session.connectorConnectedBucket"] = bucketCount(snapshot.connectorConnected)
	payload["DreamCreator.Session.externalToolInstalledBucket"] = bucketCount(snapshot.externalToolInstalled)
	payload["DreamCreator.Session.updateReadyToRestartBucket"] = bucketCount(snapshot.updateReadyToRestart)
	service.emit(Signal{
		Type:       "DreamCreator.Session.summaryRecorded",
		FloatValue: float64Ptr(durationSeconds),
		Payload:    payload,
	})
	return nil
}

func (service *Service) markRunCompleted(runID string) bool {
	service.mu.Lock()
	defer service.mu.Unlock()
	if _, exists := service.session.runIDs[runID]; exists {
		return false
	}
	service.session.runIDs[runID] = struct{}{}
	service.session.chatCompleted++
	return true
}

func (service *Service) markLibraryOperationCompleted(operationID string) bool {
	service.mu.Lock()
	defer service.mu.Unlock()
	if _, exists := service.session.operationIDs[operationID]; exists {
		return false
	}
	service.session.operationIDs[operationID] = struct{}{}
	service.session.libraryCompleted++
	return true
}

func (service *Service) incrementCounter(update func(metrics *sessionMetrics)) {
	if service == nil || update == nil {
		return
	}
	service.mu.Lock()
	defer service.mu.Unlock()
	update(&service.session)
}

func (service *Service) buildPayload(ctx context.Context, state State) map[string]any {
	appVersion := normalizeVersion(service.appVersion)
	buildNumber := buildNumberFromVersion(service.appVersion)
	platform := normalizedPlatform(runtime.GOOS)
	timeZone := service.timeZoneName()
	isDebugBuild := releaseChannel(service.appVersion) == "dev"

	payload := map[string]any{
		"DreamCreator.App.version":       appVersion,
		"DreamCreator.App.channel":       releaseChannel(service.appVersion),
		"DreamCreator.App.isDebugBuild":  isDebugBuild,
		"DreamCreator.Platform.os":       platform,
		"DreamCreator.Platform.arch":     runtime.GOARCH,
		"DreamCreator.Locale.timeZone":   timeZone,
		"DreamCreator.Install.ageBucket": bucketInstallAge(service.now().Sub(state.InstallCreatedAt)),
	}
	if buildNumber != "" {
		payload["DreamCreator.App.buildNumber"] = buildNumber
		payload["DreamCreator.App.versionAndBuildNumber"] = appVersion + " " + buildNumber
	}
	if locale := normalizeLocale(service.currentLanguage(ctx)); locale != "" {
		payload["DreamCreator.Locale.language"] = locale
		if language := primaryLanguage(locale); language != "" {
			payload["DreamCreator.Locale.primaryLanguage"] = language
		}
		if region := regionFromLocale(locale); region != "" {
			payload["DreamCreator.Locale.region"] = region
		}
	}
	return payload
}

func (service *Service) currentLanguage(ctx context.Context) string {
	if service == nil {
		return ""
	}
	if service.settings != nil {
		settings, err := service.settings.GetSettings(ctx)
		if err == nil {
			language := strings.TrimSpace(settings.Language)
			if language != "" {
				service.mu.Lock()
				service.language = language
				service.mu.Unlock()
				return language
			}
		}
	}
	service.mu.Lock()
	defer service.mu.Unlock()
	return service.language
}

func (service *Service) timeZoneName() string {
	if service == nil {
		return ""
	}
	location := service.now().Location()
	if location == nil {
		return ""
	}
	return location.String()
}

func startMode(launch AppLaunchContext) string {
	if launch.LaunchedByAutoStart {
		return "autostart"
	}
	return "manual"
}

func (service *Service) emit(signal Signal) {
	if !service.Enabled() || service.emitter == nil {
		return
	}
	service.emitter.Emit(signal)
}

func (service *Service) emitAsync(signal Signal) {
	go func() {
		service.emit(signal)
	}()
}

func (service *Service) resolveState(ctx context.Context) (State, error) {
	if service == nil {
		return State{}, fmt.Errorf("telemetry service is nil")
	}
	service.mu.Lock()
	if service.state != nil {
		cached := *service.state
		service.mu.Unlock()
		return cached, nil
	}
	service.mu.Unlock()
	state, err := service.repo.Ensure(ctx)
	if err != nil {
		return State{}, err
	}
	service.cacheState(state)
	return state, nil
}

func (service *Service) cacheState(state State) {
	if service == nil {
		return
	}
	service.mu.Lock()
	defer service.mu.Unlock()
	cached := state
	service.state = &cached
}

func buildNumberFromVersion(version string) string {
	trimmed := strings.TrimSpace(strings.TrimPrefix(version, "v"))
	_, buildNumber, found := strings.Cut(trimmed, "+")
	if !found {
		return ""
	}
	return strings.TrimSpace(buildNumber)
}

func normalizeVersion(version string) string {
	trimmed := strings.TrimSpace(version)
	if trimmed == "" {
		return "dev"
	}
	if baseVersion, _, found := strings.Cut(strings.TrimPrefix(trimmed, "v"), "+"); found {
		return strings.TrimSpace(baseVersion)
	}
	return strings.TrimPrefix(trimmed, "v")
}

func normalizeLocale(locale string) string {
	return strings.ReplaceAll(strings.TrimSpace(locale), "_", "-")
}

func primaryLanguage(locale string) string {
	normalized := normalizeLocale(locale)
	if normalized == "" {
		return ""
	}
	language, _, found := strings.Cut(normalized, "-")
	if !found {
		return normalized
	}
	return strings.TrimSpace(language)
}

func regionFromLocale(locale string) string {
	normalized := normalizeLocale(locale)
	if normalized == "" {
		return ""
	}
	parts := strings.Split(normalized, "-")
	if len(parts) < 2 {
		return ""
	}
	return strings.ToUpper(strings.TrimSpace(parts[len(parts)-1]))
}

func normalizedPlatform(goos string) string {
	switch strings.ToLower(strings.TrimSpace(goos)) {
	case "darwin":
		return "macOS"
	case "windows":
		return "Windows"
	case "linux":
		return "Linux"
	default:
		return goos
	}
}

func releaseChannel(version string) string {
	normalized := strings.ToLower(normalizeVersion(version))
	switch {
	case normalized == "dev":
		return "dev"
	case strings.Contains(normalized, "alpha"):
		return "alpha"
	case strings.Contains(normalized, "beta"):
		return "beta"
	case strings.Contains(normalized, "rc"):
		return "rc"
	default:
		return "stable"
	}
}

func bucketInstallAge(duration time.Duration) string {
	if duration < 0 {
		duration = 0
	}
	days := int(duration.Hours() / 24)
	switch {
	case days <= 0:
		return "day0"
	case days < 7:
		return "day1-6"
	case days < 30:
		return "day7-29"
	case days < 90:
		return "day30-89"
	default:
		return "day90+"
	}
}

func bucketLaunchOrdinal(launchCount int) string {
	switch {
	case launchCount <= 1:
		return "1"
	case launchCount <= 3:
		return "2-3"
	case launchCount <= 9:
		return "4-9"
	case launchCount <= 29:
		return "10-29"
	default:
		return "30+"
	}
}

func bucketSessionDuration(duration time.Duration) string {
	switch {
	case duration < time.Minute:
		return "lt1m"
	case duration < 5*time.Minute:
		return "1m-5m"
	case duration < 15*time.Minute:
		return "5m-15m"
	case duration < time.Hour:
		return "15m-60m"
	default:
		return "60m+"
	}
}

func bucketCount(value int) string {
	switch {
	case value <= 0:
		return "0"
	case value == 1:
		return "1"
	case value <= 3:
		return "2-3"
	case value <= 9:
		return "4-9"
	default:
		return "10+"
	}
}

func float64Ptr(value float64) *float64 {
	return &value
}
