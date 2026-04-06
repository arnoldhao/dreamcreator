package heartbeat

import (
	"context"
	"testing"
)

func TestTriggerWithResultQueuesWake(t *testing.T) {
	t.Parallel()

	service := NewService(nil, nil, nil, nil, StoreOptions{}, nil, nil)
	result := service.TriggerWithResult(context.Background(), TriggerInput{Reason: "manual-settings"})
	if !result.Accepted {
		t.Fatalf("expected accepted trigger result")
	}
	if result.ExecutedStatus != TriggerExecutionQueued {
		t.Fatalf("expected queued status, got %q", result.ExecutedStatus)
	}
	if result.Reason != "manual-settings" {
		t.Fatalf("expected reason passthrough, got %q", result.Reason)
	}
}

func TestResolveWakePriority(t *testing.T) {
	t.Parallel()

	if got := resolveWakePriority("retry"); got != 0 {
		t.Fatalf("expected retry priority 0, got %d", got)
	}
	if got := resolveWakePriority("interval"); got != 1 {
		t.Fatalf("expected interval priority 1, got %d", got)
	}
	if got := resolveWakePriority("manual-settings"); got != 3 {
		t.Fatalf("expected manual force priority 3, got %d", got)
	}
	if got := resolveWakePriority("normal"); got != 2 {
		t.Fatalf("expected default priority 2, got %d", got)
	}
}

func TestResolveIndicatorStatus(t *testing.T) {
	t.Parallel()

	if got := resolveIndicatorStatus(StatusOKToken, true); got != IndicatorOK {
		t.Fatalf("expected ok indicator, got %q", got)
	}
	if got := resolveIndicatorStatus(StatusSent, true); got != IndicatorAlert {
		t.Fatalf("expected alert indicator, got %q", got)
	}
	if got := resolveIndicatorStatus(StatusFailed, true); got != IndicatorError {
		t.Fatalf("expected error indicator, got %q", got)
	}
	if got := resolveIndicatorStatus(StatusSent, false); got != "" {
		t.Fatalf("expected empty indicator when disabled, got %q", got)
	}
}
