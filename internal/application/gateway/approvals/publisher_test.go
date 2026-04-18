package approvals

import (
	"context"
	"testing"
	"time"

	gatewayevents "dreamcreator/internal/application/gateway/events"
	domainsession "dreamcreator/internal/domain/session"
)

func TestGatewayEventPublisherIncludesSessionInEnvelope(t *testing.T) {
	events := gatewayevents.NewBroker(nil)
	publisher := NewGatewayEventPublisher(events)

	recordCh := make(chan gatewayevents.Record, 1)
	unsubscribe := events.Subscribe(gatewayevents.Filter{Type: "exec.approval.requested"}, func(record gatewayevents.Record) {
		recordCh <- record
	})
	defer unsubscribe()

	sessionKey, err := domainsession.BuildSessionKey(domainsession.KeyParts{
		Channel:   "aui",
		PrimaryID: "thread-123",
		ThreadRef: "thread-123",
	})
	if err != nil {
		t.Fatalf("build session key: %v", err)
	}
	request := Request{
		ID:         "approval-1",
		SessionKey: sessionKey,
		ToolName:   "exec",
		Action:     "config.schema",
	}
	if err := publisher.Publish(context.Background(), "exec.approval.requested", request); err != nil {
		t.Fatalf("publish approval event: %v", err)
	}

	select {
	case record := <-recordCh:
		if record.Envelope.SessionKey != sessionKey {
			t.Fatalf("expected session key %q, got %q", sessionKey, record.Envelope.SessionKey)
		}
		if record.Envelope.SessionID != "thread-123" {
			t.Fatalf("expected session id thread-123, got %q", record.Envelope.SessionID)
		}
	case <-time.After(time.Second):
		t.Fatal("expected approval event record")
	}
}

func TestGatewayEventPublisherFallsBackToRawSessionKey(t *testing.T) {
	events := gatewayevents.NewBroker(nil)
	publisher := NewGatewayEventPublisher(events)

	recordCh := make(chan gatewayevents.Record, 1)
	unsubscribe := events.Subscribe(gatewayevents.Filter{Type: "exec.approval.requested"}, func(record gatewayevents.Record) {
		recordCh <- record
	})
	defer unsubscribe()

	request := Request{
		ID:         "approval-2",
		SessionKey: "raw-thread-id",
		ToolName:   "exec",
		Action:     "config.schema",
	}
	if err := publisher.Publish(context.Background(), "exec.approval.requested", request); err != nil {
		t.Fatalf("publish approval event: %v", err)
	}

	select {
	case record := <-recordCh:
		if record.Envelope.SessionKey != "raw-thread-id" {
			t.Fatalf("expected raw session key, got %q", record.Envelope.SessionKey)
		}
		if record.Envelope.SessionID != "raw-thread-id" {
			t.Fatalf("expected raw session id, got %q", record.Envelope.SessionID)
		}
	case <-time.After(time.Second):
		t.Fatal("expected approval event record")
	}
}
