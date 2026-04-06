package heartbeat

import (
	"context"
	"testing"
	"time"

	runtimedto "dreamcreator/internal/application/gateway/runtime/dto"
	settingsservice "dreamcreator/internal/application/settings/service"
	"dreamcreator/internal/domain/session"
	domainsettings "dreamcreator/internal/domain/settings"
	"dreamcreator/internal/domain/thread"
)

func boolPtr(value bool) *bool {
	return &value
}

func intPtr(value int) *int {
	return &value
}

func stringPtr(value string) *string {
	return &value
}

type runtimeTestThreadRepo struct {
	item thread.Thread
}

func (repo *runtimeTestThreadRepo) List(context.Context, bool) ([]thread.Thread, error) {
	return []thread.Thread{repo.item}, nil
}

func (repo *runtimeTestThreadRepo) ListPurgeCandidates(context.Context, time.Time, int) ([]thread.Thread, error) {
	return nil, nil
}

func (repo *runtimeTestThreadRepo) Get(_ context.Context, id string) (thread.Thread, error) {
	if id != repo.item.ID {
		return thread.Thread{}, thread.ErrThreadNotFound
	}
	return repo.item, nil
}

func (repo *runtimeTestThreadRepo) Save(_ context.Context, item thread.Thread) error {
	repo.item = item
	return nil
}

func (repo *runtimeTestThreadRepo) SoftDelete(context.Context, string, *time.Time, *time.Time) error {
	return nil
}

func (repo *runtimeTestThreadRepo) Restore(context.Context, string) error {
	return nil
}

func (repo *runtimeTestThreadRepo) Purge(context.Context, string) error {
	return nil
}

func (repo *runtimeTestThreadRepo) SetStatus(context.Context, string, thread.Status, time.Time) error {
	return nil
}

type runtimeTestSettingsRepo struct {
	current domainsettings.Settings
}

func (repo *runtimeTestSettingsRepo) Get(context.Context) (domainsettings.Settings, error) {
	return repo.current, nil
}

func (repo *runtimeTestSettingsRepo) Save(_ context.Context, current domainsettings.Settings) error {
	repo.current = current
	return nil
}

type runtimeCallTracker struct {
	calls int
}

func (tracker *runtimeCallTracker) Run(context.Context, runtimedto.RuntimeRunRequest) (runtimedto.RuntimeRunResult, error) {
	tracker.calls++
	return runtimedto.RuntimeRunResult{}, nil
}

func TestRun_PeriodicWithoutPendingEventsShortCircuitsToOK(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.April, 3, 10, 0, 0, 0, time.UTC)
	threadItem, err := thread.NewThread(thread.ThreadParams{
		ID:          "thread-1",
		AssistantID: "assistant-1",
		Title:       "Thread",
		CreatedAt:   &now,
		UpdatedAt:   &now,
	})
	if err != nil {
		t.Fatalf("new thread: %v", err)
	}
	sessionKey, err := session.BuildSessionKey(session.KeyParts{
		Channel:   "web",
		PrimaryID: threadItem.ID,
		ThreadRef: threadItem.ID,
	})
	if err != nil {
		t.Fatalf("build session key: %v", err)
	}

	defaults, err := domainsettings.NewSettings(domainsettings.SettingsParams{
		Appearance: string(domainsettings.AppearanceLight),
		Language:   domainsettings.DefaultLanguage.String(),
		Gateway: domainsettings.GatewaySettingsParams{
			Heartbeat: &domainsettings.GatewayHeartbeatSettingsParams{
				Enabled:      boolPtr(true),
				RunSession:   stringPtr(sessionKey),
				Every:        stringPtr("30m"),
				EveryMinutes: intPtr(30),
				Prompt:       stringPtr(""),
				PromptAppend: stringPtr(""),
				Periodic: &domainsettings.GatewayHeartbeatPeriodicSettingsParams{
					Enabled: boolPtr(true),
					Every:   stringPtr("30m"),
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("new settings: %v", err)
	}
	settingsRepo := &runtimeTestSettingsRepo{current: defaults}
	threadRepo := &runtimeTestThreadRepo{item: threadItem}
	runtimeTracker := &runtimeCallTracker{}
	service := NewService(
		settingsservice.NewSettingsService(settingsRepo, nil, defaults),
		threadRepo,
		nil,
		runtimeTracker,
		StoreOptions{},
		nil,
		nil,
	)
	service.now = func() time.Time { return now }
	service.newID = func() string { return "hb-1" }

	result := service.run(context.Background(), TriggerInput{Reason: "interval", SessionKey: sessionKey, Force: true})
	if result.ExecutedStatus != TriggerExecutionRan {
		t.Fatalf("expected ran result, got %+v", result)
	}
	if result.Reason != string(StatusOKEmpty) {
		t.Fatalf("expected ok-empty reason, got %q", result.Reason)
	}
	if runtimeTracker.calls != 0 {
		t.Fatalf("expected runtime not to be called, got %d calls", runtimeTracker.calls)
	}

	event, err := service.Last(context.Background(), sessionKey)
	if err != nil {
		t.Fatalf("last event: %v", err)
	}
	if event.Status != StatusOKEmpty {
		t.Fatalf("expected ok-empty event, got %q", event.Status)
	}
	if event.Indicator != IndicatorOK {
		t.Fatalf("expected ok indicator, got %q", event.Indicator)
	}
	if !event.Silent {
		t.Fatalf("expected short-circuit heartbeat to be silent")
	}
}
