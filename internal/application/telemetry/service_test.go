package telemetry

import (
	"context"
	"sync"
	"testing"
	"time"

	settingsdto "dreamcreator/internal/application/settings/dto"
)

type telemetryStateRepoStub struct {
	mu    sync.Mutex
	state State
}

func (stub *telemetryStateRepoStub) Ensure(_ context.Context) (State, error) {
	stub.mu.Lock()
	defer stub.mu.Unlock()
	return stub.state, nil
}

func (stub *telemetryStateRepoStub) IncrementLaunchCount(_ context.Context, _ time.Time) (State, error) {
	stub.mu.Lock()
	defer stub.mu.Unlock()
	stub.state.LaunchCount++
	return stub.state, nil
}

func (stub *telemetryStateRepoStub) MarkFirstProviderConfigured(_ context.Context, at time.Time) (State, bool, error) {
	stub.mu.Lock()
	defer stub.mu.Unlock()
	if stub.state.FirstProviderConfiguredAt == nil {
		timestamp := at.UTC()
		stub.state.FirstProviderConfiguredAt = &timestamp
		return stub.state, true, nil
	}
	return stub.state, false, nil
}

func (stub *telemetryStateRepoStub) MarkFirstChatCompleted(_ context.Context, at time.Time) (State, bool, error) {
	stub.mu.Lock()
	defer stub.mu.Unlock()
	if stub.state.FirstChatCompletedAt == nil {
		timestamp := at.UTC()
		stub.state.FirstChatCompletedAt = &timestamp
		return stub.state, true, nil
	}
	return stub.state, false, nil
}

func (stub *telemetryStateRepoStub) MarkFirstLibraryCompleted(_ context.Context, at time.Time) (State, bool, error) {
	stub.mu.Lock()
	defer stub.mu.Unlock()
	if stub.state.FirstLibraryCompletedAt == nil {
		timestamp := at.UTC()
		stub.state.FirstLibraryCompletedAt = &timestamp
		return stub.state, true, nil
	}
	return stub.state, false, nil
}

type telemetryEmitterStub struct {
	mu      sync.Mutex
	signals []sentSignal
}

type sentSignal struct {
	Type       string
	FloatValue *float64
	Payload    map[string]any
}

func (stub *telemetryEmitterStub) Emit(signal Signal) {
	stub.mu.Lock()
	defer stub.mu.Unlock()
	copied := make(map[string]any, len(signal.Payload))
	for key, value := range signal.Payload {
		copied[key] = value
	}
	var floatValue *float64
	if signal.FloatValue != nil {
		value := *signal.FloatValue
		floatValue = &value
	}
	stub.signals = append(stub.signals, sentSignal{Type: signal.Type, FloatValue: floatValue, Payload: copied})
}

func (stub *telemetryEmitterStub) waitForCount(t *testing.T, count int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		stub.mu.Lock()
		current := len(stub.signals)
		stub.mu.Unlock()
		if current >= count {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %d signals, got %d", count, current)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func (stub *telemetryEmitterStub) count(signalType string) int {
	stub.mu.Lock()
	defer stub.mu.Unlock()
	total := 0
	for _, item := range stub.signals {
		if item.Type == signalType {
			total++
		}
	}
	return total
}

func (stub *telemetryEmitterStub) payloadFor(signalType string) map[string]any {
	stub.mu.Lock()
	defer stub.mu.Unlock()
	for _, item := range stub.signals {
		if item.Type == signalType {
			return item.Payload
		}
	}
	return nil
}

func (stub *telemetryEmitterStub) signalFor(signalType string) *sentSignal {
	stub.mu.Lock()
	defer stub.mu.Unlock()
	for _, item := range stub.signals {
		if item.Type == signalType {
			copied := item
			return &copied
		}
	}
	return nil
}

type telemetrySettingsStub struct {
	language string
}

func (stub telemetrySettingsStub) GetSettings(context.Context) (settingsdto.Settings, error) {
	return settingsdto.Settings{Language: stub.language}, nil
}

func TestServiceTracksSparseSignalsAndSessionSummary(t *testing.T) {
	base := time.Date(2026, 4, 1, 9, 0, 0, 0, time.FixedZone("UTC+8", 8*3600))
	repo := &telemetryStateRepoStub{
		state: State{
			InstallID:        "install-1",
			InstallCreatedAt: base.Add(-48 * time.Hour),
		},
	}
	emitter := &telemetryEmitterStub{}
	service := NewService(repo, emitter, telemetrySettingsStub{language: "zh-CN"}, "app-123", "1.2.3")
	service.now = func() time.Time { return base.Add(10 * time.Minute) }
	service.startedAt = base

	launchCount, err := service.TrackAppLaunch(context.Background(), AppLaunchContext{LaunchedByAutoStart: true})
	if err != nil {
		t.Fatalf("track app launch failed: %v", err)
	}
	service.TrackProviderConfigured(context.Background(), "openai")
	service.TrackProviderConfigured(context.Background(), "openai")
	service.TrackUserChatCompleted(context.Background(), "run-1")
	service.TrackUserChatCompleted(context.Background(), "run-1")
	service.TrackLibraryOperationCompleted(context.Background(), "op-1", "download")

	if launchCount != 2 {
		t.Fatalf("expected 2 launch signals, got %d", launchCount)
	}
	emitter.waitForCount(t, 7)

	if got := emitter.count("TelemetryDeck.Session.started"); got != 1 {
		t.Fatalf("expected 1 session started signal, got %d", got)
	}
	if got := emitter.count("TelemetryDeck.Acquisition.newInstallDetected"); got != 1 {
		t.Fatalf("expected 1 acquisition signal, got %d", got)
	}

	if got := emitter.count("DreamCreator.Setup.providerConfigured"); got != 2 {
		t.Fatalf("expected 2 provider setup signals, got %d", got)
	}
	if got := emitter.count("DreamCreator.Activation.firstProviderConfigured"); got != 1 {
		t.Fatalf("expected 1 first provider activation signal, got %d", got)
	}
	if got := emitter.count("DreamCreator.Activation.firstChatCompleted"); got != 1 {
		t.Fatalf("expected 1 first chat activation signal, got %d", got)
	}
	if got := emitter.count("DreamCreator.Activation.firstLibraryCompleted"); got != 1 {
		t.Fatalf("expected 1 first library activation signal, got %d", got)
	}

	if err := service.FlushSessionSummary(context.Background()); err != nil {
		t.Fatalf("flush session summary failed: %v", err)
	}

	emitter.waitForCount(t, 8)

	summarySignal := emitter.signalFor("DreamCreator.Session.summaryRecorded")
	if summarySignal == nil {
		t.Fatal("expected session summary signal")
	}
	summaryPayload := summarySignal.Payload
	if got := summaryPayload["DreamCreator.Session.durationBucket"]; got != "5m-15m" {
		t.Fatalf("unexpected duration bucket: %#v", got)
	}
	if summarySignal.FloatValue == nil || *summarySignal.FloatValue != 600 {
		t.Fatalf("unexpected session duration float value: %#v", summarySignal.FloatValue)
	}
	if _, exists := summaryPayload["floatValue"]; exists {
		t.Fatalf("expected floatValue to be excluded from payload, got %#v", summaryPayload["floatValue"])
	}
	if got := summaryPayload["DreamCreator.Session.chatCompletedBucket"]; got != "1" {
		t.Fatalf("unexpected chat bucket: %#v", got)
	}
	if got := summaryPayload["DreamCreator.Session.libraryCompletedBucket"]; got != "1" {
		t.Fatalf("unexpected library bucket: %#v", got)
	}
	if got := summaryPayload["DreamCreator.Session.providerConfiguredBucket"]; got != "2-3" {
		t.Fatalf("unexpected provider bucket: %#v", got)
	}
	if got := summaryPayload["DreamCreator.Locale.language"]; got != "zh-CN" {
		t.Fatalf("unexpected language: %#v", got)
	}

	launchPayload := emitter.payloadFor("TelemetryDeck.Session.started")
	if launchPayload == nil {
		t.Fatal("expected app launch payload")
	}
	if got := launchPayload["DreamCreator.App.version"]; got != "1.2.3" {
		t.Fatalf("unexpected app version: %#v", got)
	}
	if got := launchPayload["DreamCreator.Locale.language"]; got != "zh-CN" {
		t.Fatalf("unexpected locale language: %#v", got)
	}
	if got := launchPayload["DreamCreator.App.launchOrdinalBucket"]; got != "1" {
		t.Fatalf("unexpected launch ordinal bucket: %#v", got)
	}
	if got := launchPayload["DreamCreator.App.launchedByAutoStart"]; got != true {
		t.Fatalf("unexpected launch mode flag: %#v", got)
	}
	if got := launchPayload["DreamCreator.App.startMode"]; got != "autostart" {
		t.Fatalf("unexpected start mode: %#v", got)
	}
	if got := launchPayload["DreamCreator.Install.firstLaunch"]; got != true {
		t.Fatalf("unexpected first launch flag: %#v", got)
	}
	if got := launchPayload["DreamCreator.Install.ageBucket"]; got != "day1-6" {
		t.Fatalf("unexpected install age bucket: %#v", got)
	}
	if _, exists := launchPayload["DreamCreator.Session.startedAt"]; exists {
		t.Fatalf("expected launch payload to omit startedAt, got %#v", launchPayload["DreamCreator.Session.startedAt"])
	}
}

func TestBootstrapMarksDevBuildsAsTestMode(t *testing.T) {
	repo := &telemetryStateRepoStub{
		state: State{
			InstallID:        "install-1",
			InstallCreatedAt: time.Date(2026, 4, 1, 9, 0, 0, 0, time.UTC),
		},
	}
	service := NewService(repo, &telemetryEmitterStub{}, telemetrySettingsStub{}, "app-123", "dev")

	bootstrap, err := service.Bootstrap(context.Background())
	if err != nil {
		t.Fatalf("bootstrap failed: %v", err)
	}
	if !bootstrap.Enabled {
		t.Fatal("expected telemetry bootstrap to be enabled")
	}
	if !bootstrap.TestMode {
		t.Fatal("expected dev builds to use telemetry test mode")
	}
}
