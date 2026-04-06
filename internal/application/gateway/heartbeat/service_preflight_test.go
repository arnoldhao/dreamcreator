package heartbeat

import (
	"context"
	"testing"
	"time"

	settingsdto "dreamcreator/internal/application/settings/dto"
	"dreamcreator/internal/domain/session"
)

func mustSessionKey(t *testing.T, channelID string, accountID string) string {
	t.Helper()
	key, err := session.BuildSessionKey(session.KeyParts{
		Channel:   channelID,
		PrimaryID: "thread-1",
		AccountID: accountID,
		ThreadRef: "thread-1",
	})
	if err != nil {
		t.Fatalf("build session key: %v", err)
	}
	return key
}

func TestWithinActiveHours_IncludesStartExcludesEnd(t *testing.T) {
	t.Parallel()

	service := NewService(nil, nil, nil, nil, StoreOptions{}, nil, nil)
	cfg := settingsdto.GatewayHeartbeatActiveHours{
		Start:    "09:00",
		End:      "17:00",
		Timezone: "UTC",
	}

	service.now = func() time.Time {
		return time.Date(2026, time.March, 14, 9, 0, 0, 0, time.UTC)
	}
	if !service.withinActiveHours(cfg) {
		t.Fatalf("expected start boundary to be included")
	}

	service.now = func() time.Time {
		return time.Date(2026, time.March, 14, 17, 0, 0, 0, time.UTC)
	}
	if service.withinActiveHours(cfg) {
		t.Fatalf("expected end boundary to be excluded")
	}
}

func TestWithinActiveHours_OvernightWindow(t *testing.T) {
	t.Parallel()

	service := NewService(nil, nil, nil, nil, StoreOptions{}, nil, nil)
	cfg := settingsdto.GatewayHeartbeatActiveHours{
		Start:    "22:00",
		End:      "06:00",
		Timezone: "UTC",
	}

	service.now = func() time.Time {
		return time.Date(2026, time.March, 14, 23, 30, 0, 0, time.UTC)
	}
	if !service.withinActiveHours(cfg) {
		t.Fatalf("expected late-night time inside overnight window")
	}

	service.now = func() time.Time {
		return time.Date(2026, time.March, 15, 6, 0, 0, 0, time.UTC)
	}
	if service.withinActiveHours(cfg) {
		t.Fatalf("expected end boundary to be excluded for overnight window")
	}
}

func TestWithinActiveHours_SupportsEndOfDay(t *testing.T) {
	t.Parallel()

	service := NewService(nil, nil, nil, nil, StoreOptions{}, nil, nil)
	cfg := settingsdto.GatewayHeartbeatActiveHours{
		Start:    "09:00",
		End:      "24:00",
		Timezone: "UTC",
	}

	service.now = func() time.Time {
		return time.Date(2026, time.March, 14, 23, 59, 0, 0, time.UTC)
	}
	if !service.withinActiveHours(cfg) {
		t.Fatalf("expected 23:59 to be inside 09:00-24:00 window")
	}
}

func TestIsRouteReady_UsesExplicitAccountOverride(t *testing.T) {
	t.Parallel()

	service := NewService(nil, nil, nil, nil, StoreOptions{}, nil, nil)
	called := false
	receivedAccountID := ""
	service.SetChannelReadyCheck(func(_ context.Context, channelID string, accountID string) (bool, string) {
		called = true
		if channelID != "telegram" {
			t.Fatalf("unexpected channel id: %q", channelID)
		}
		receivedAccountID = accountID
		return true, ""
	})

	cfg := settingsdto.GatewayHeartbeatSettings{
		To:        "chat-id",
		AccountID: "acct-override",
	}
	ready, reason := service.isRouteReady(context.Background(), mustSessionKey(t, "telegram", ""), cfg)
	if !ready {
		t.Fatalf("expected route to be ready, reason=%q", reason)
	}
	if !called {
		t.Fatalf("expected readiness check to be called")
	}
	if receivedAccountID != "acct-override" {
		t.Fatalf("expected account override to be used, got %q", receivedAccountID)
	}
}

func TestIsRouteReady_ChannelCheckFallbackReason(t *testing.T) {
	t.Parallel()

	service := NewService(nil, nil, nil, nil, StoreOptions{}, nil, nil)
	service.SetChannelReadyCheck(func(_ context.Context, _ string, _ string) (bool, string) {
		return false, ""
	})

	cfg := settingsdto.GatewayHeartbeatSettings{
		To: "chat-id",
	}
	ready, reason := service.isRouteReady(context.Background(), mustSessionKey(t, "telegram", "acct-1"), cfg)
	if ready {
		t.Fatalf("expected route to be not ready")
	}
	if reason != "channel_not_ready" {
		t.Fatalf("unexpected reason: %q", reason)
	}
}

func TestIsRouteReady_ImplicitSessionRouteChecksReadiness(t *testing.T) {
	t.Parallel()

	service := NewService(nil, nil, nil, nil, StoreOptions{}, nil, nil)
	called := false
	service.SetChannelReadyCheck(func(_ context.Context, channelID string, accountID string) (bool, string) {
		called = true
		if channelID != "telegram" {
			t.Fatalf("unexpected channel id: %q", channelID)
		}
		if accountID != "acct-1" {
			t.Fatalf("unexpected account id: %q", accountID)
		}
		return false, "offline"
	})

	ready, reason := service.isRouteReady(context.Background(), mustSessionKey(t, "telegram", "acct-1"), settingsdto.GatewayHeartbeatSettings{})
	if ready {
		t.Fatalf("expected route to be not ready")
	}
	if !called {
		t.Fatalf("expected readiness check to be called for implicit session route")
	}
	if reason != "offline" {
		t.Fatalf("unexpected reason: %q", reason)
	}
}

func TestIsRouteReady_NoRouteSkipsReadinessCheck(t *testing.T) {
	t.Parallel()

	service := NewService(nil, nil, nil, nil, StoreOptions{}, nil, nil)
	called := false
	service.SetChannelReadyCheck(func(_ context.Context, _ string, _ string) (bool, string) {
		called = true
		return true, ""
	})

	ready, reason := service.isRouteReady(context.Background(), "thread-1", settingsdto.GatewayHeartbeatSettings{})
	if !ready {
		t.Fatalf("expected no-route session to pass, reason=%q", reason)
	}
	if called {
		t.Fatalf("did not expect readiness check when route is unavailable and not explicit")
	}
}
