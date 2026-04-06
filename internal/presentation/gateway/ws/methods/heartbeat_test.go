package methods

import (
	"context"
	"testing"

	"dreamcreator/internal/application/gateway/controlplane"
	gatewayheartbeat "dreamcreator/internal/application/gateway/heartbeat"
)

func TestRegisterHeartbeatSystemEventQueueFlag(t *testing.T) {
	t.Parallel()

	router := controlplane.NewRouter(nil)
	service := gatewayheartbeat.NewService(nil, nil, nil, nil, gatewayheartbeat.StoreOptions{}, nil, nil)
	RegisterHeartbeat(router, service)

	session := &controlplane.SessionContext{ID: "test-session"}
	first := router.Handle(context.Background(), session, controlplane.RequestFrame{
		ID:     "1",
		Method: "heartbeat.systemEvent",
		Params: []byte(`{"sessionKey":"session-1","text":"done","contextKey":"subagent:success","runId":"run-1","source":"subagent"}`),
	})
	if !first.OK {
		t.Fatalf("systemEvent should succeed: %+v", first.Error)
	}
	firstPayload, ok := first.Payload.(map[string]any)
	if !ok {
		t.Fatalf("unexpected payload type: %T", first.Payload)
	}
	if queued, _ := firstPayload["queued"].(bool); !queued {
		t.Fatalf("expected first queue accepted")
	}

	second := router.Handle(context.Background(), session, controlplane.RequestFrame{
		ID:     "2",
		Method: "heartbeat.systemEvent",
		Params: []byte(`{"sessionKey":"session-1","text":"done","contextKey":"subagent:success","runId":"run-1","source":"subagent"}`),
	})
	if !second.OK {
		t.Fatalf("duplicate systemEvent should still succeed: %+v", second.Error)
	}
	secondPayload, ok := second.Payload.(map[string]any)
	if !ok {
		t.Fatalf("unexpected payload type: %T", second.Payload)
	}
	if queued, _ := secondPayload["queued"].(bool); queued {
		t.Fatalf("expected duplicate queue skipped")
	}
}

func TestRegisterHeartbeatTriggerReturnsExecutionStatus(t *testing.T) {
	t.Parallel()

	router := controlplane.NewRouter(nil)
	service := gatewayheartbeat.NewService(nil, nil, nil, nil, gatewayheartbeat.StoreOptions{}, nil, nil)
	RegisterHeartbeat(router, service)

	session := &controlplane.SessionContext{ID: "test-session"}
	response := router.Handle(context.Background(), session, controlplane.RequestFrame{
		ID:     "1",
		Method: "heartbeat.trigger",
		Params: []byte(`{"reason":"manual-settings"}`),
	})
	if !response.OK {
		t.Fatalf("trigger should succeed: %+v", response.Error)
	}
	payload, ok := response.Payload.(map[string]any)
	if !ok {
		t.Fatalf("unexpected payload type: %T", response.Payload)
	}
	if accepted, _ := payload["accepted"].(bool); !accepted {
		t.Fatalf("expected accepted=true, got payload=%+v", payload)
	}
	if status := payload["executedStatus"]; status == nil || status.(gatewayheartbeat.TriggerExecutionStatus) != gatewayheartbeat.TriggerExecutionQueued {
		t.Fatalf("expected executedStatus=queued, got payload=%+v", payload)
	}
}
