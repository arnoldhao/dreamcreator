package queue

import (
	"context"
	"testing"
	"time"

	gatewayevents "dreamcreator/internal/application/gateway/events"
)

func TestQueueEnqueue(t *testing.T) {
	manager := NewManager(nil, NewPolicyResolver(Policy{DefaultMode: "followup"}), nil, nil)
	ticket, event, err := manager.Enqueue(nil, EnqueueRequest{SessionKey: "session-1", Mode: "followup", Payload: "hello"})
	if err != nil {
		t.Fatalf("enqueue error: %v", err)
	}
	if ticket.TicketID == "" {
		t.Fatalf("expected ticket id")
	}
	if event.Type == "" {
		t.Fatalf("expected event type")
	}
}

func TestResetAllLanesClearsActiveLaneState(t *testing.T) {
	t.Parallel()

	broker := gatewayevents.NewBroker(nil)
	manager := NewManager(nil, NewPolicyResolver(Policy{DefaultMode: "followup"}), nil, broker)
	manager.UpdateLaneCaps(LaneCaps{Main: 1})

	if err := manager.AcquireLane(context.Background(), LaneMain); err != nil {
		t.Fatalf("first acquire failed: %v", err)
	}

	manager.ResetAllLanes(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	if err := manager.AcquireLane(ctx, LaneMain); err != nil {
		t.Fatalf("acquire after reset failed: %v", err)
	}
	manager.ReleaseLane(LaneMain)
}
