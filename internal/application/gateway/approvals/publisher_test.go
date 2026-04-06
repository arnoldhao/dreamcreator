package approvals

import (
	"context"
	"testing"
	"time"

	gatewayevents "dreamcreator/internal/application/gateway/events"
)

func TestGatewayEventPublisherIncludesSessionInEnvelope(t *testing.T) {
	events := gatewayevents.NewBroker(nil)
	publisher := NewGatewayEventPublisher(events)

	recordCh := make(chan gatewayevents.Record, 1)
	unsubscribe := events.Subscribe(gatewayevents.Filter{Type: "exec.approval.requested"}, func(record gatewayevents.Record) {
		recordCh <- record
	})
	defer unsubscribe()

	request := Request{
		ID:         "approval-1",
		SessionKey: "thread-123",
		ToolName:   "exec",
		Action:     "config.schema",
	}
	if err := publisher.Publish(context.Background(), "exec.approval.requested", request); err != nil {
		t.Fatalf("publish approval event: %v", err)
	}

	select {
	case record := <-recordCh:
		if record.Envelope.SessionKey != "thread-123" {
			t.Fatalf("expected session key thread-123, got %q", record.Envelope.SessionKey)
		}
		if record.Envelope.SessionID != "thread-123" {
			t.Fatalf("expected session id thread-123, got %q", record.Envelope.SessionID)
		}
	case <-time.After(time.Second):
		t.Fatal("expected approval event record")
	}
}
