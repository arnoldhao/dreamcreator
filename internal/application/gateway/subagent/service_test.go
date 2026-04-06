package subagent

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	gatewayevents "dreamcreator/internal/application/gateway/events"
	gatewayheartbeat "dreamcreator/internal/application/gateway/heartbeat"
	runtimedto "dreamcreator/internal/application/gateway/runtime/dto"
	subagentservice "dreamcreator/internal/application/subagent/service"
)

type runtimeRunnerStub struct {
	result   runtimedto.RuntimeRunResult
	err      error
	requests chan runtimedto.RuntimeRunRequest
}

func (stub *runtimeRunnerStub) Run(_ context.Context, request runtimedto.RuntimeRunRequest) (runtimedto.RuntimeRunResult, error) {
	if stub.requests != nil {
		stub.requests <- request
	}
	return stub.result, stub.err
}

type heartbeatSinkStub struct {
	events   []gatewayheartbeat.SystemEventInput
	triggers []gatewayheartbeat.TriggerInput
}

func (stub *heartbeatSinkStub) EnqueueSystemEvent(_ context.Context, input gatewayheartbeat.SystemEventInput) bool {
	stub.events = append(stub.events, input)
	return true
}

func (stub *heartbeatSinkStub) TriggerWithInput(_ context.Context, input gatewayheartbeat.TriggerInput) bool {
	stub.triggers = append(stub.triggers, input)
	return true
}

func TestGatewaySpawnUsesIsolatedChildSessionAndAnnouncePayload(t *testing.T) {
	spawner := subagentservice.NewSpawner()
	events := gatewayevents.NewBroker(nil)
	heartbeat := &heartbeatSinkStub{}
	runtimeStub := &runtimeRunnerStub{
		result: runtimedto.RuntimeRunResult{
			Status: "completed",
			AssistantMessage: runtimedto.Message{
				Role:    "assistant",
				Content: "child done",
			},
			Usage: runtimedto.RuntimeUsage{
				PromptTokens:     10,
				CompletionTokens: 4,
				TotalTokens:      14,
			},
		},
		requests: make(chan runtimedto.RuntimeRunRequest, 1),
	}
	gateway := NewGatewayService(spawner, nil, nil, events, runtimeStub, nil, nil, nil, nil, heartbeat)

	received := make(chan gatewayevents.Record, 1)
	unsubscribe := events.Subscribe(gatewayevents.Filter{Type: "subagent.announced"}, func(record gatewayevents.Record) {
		received <- record
	})
	defer unsubscribe()

	record, err := gateway.Spawn(context.Background(), subagentservice.SpawnRequest{
		ParentSessionKey: "v2::agent::web::-::thread-parent::-::thread-parent",
		AgentID:          "worker",
		Task:             "run test task",
	})
	if err != nil {
		t.Fatalf("spawn error: %v", err)
	}
	if record.RunID == "" {
		t.Fatalf("expected run id")
	}
	if record.ChildSessionID == "" {
		t.Fatalf("expected child session id")
	}
	if !strings.HasPrefix(record.ChildSessionKey, "agent:worker:subagent:") {
		t.Fatalf("unexpected child session key: %s", record.ChildSessionKey)
	}

	select {
	case request := <-runtimeStub.requests:
		if request.SessionKey != record.ChildSessionKey {
			t.Fatalf("runtime session key mismatch: got %s want %s", request.SessionKey, record.ChildSessionKey)
		}
		if request.SessionKey == record.ParentSessionKey {
			t.Fatalf("runtime should not reuse parent session key")
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("expected runtime request")
	}

	select {
	case eventRecord := <-received:
		if eventRecord.Envelope.Type != "subagent.announced" {
			t.Fatalf("unexpected event type: %s", eventRecord.Envelope.Type)
		}
		var payload AnnounceEvent
		if err := json.Unmarshal(eventRecord.Payload, &payload); err != nil {
			t.Fatalf("unmarshal announce payload failed: %v", err)
		}
		if payload.Status != "success" {
			t.Fatalf("unexpected announce status: %s", payload.Status)
		}
		if payload.RunID != record.RunID {
			t.Fatalf("unexpected run id: %s", payload.RunID)
		}
		if payload.ChildSessionKey != record.ChildSessionKey {
			t.Fatalf("announce child session key mismatch: %s", payload.ChildSessionKey)
		}
		if payload.Result != "child done" {
			t.Fatalf("unexpected announce result: %s", payload.Result)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("expected announce event")
	}
	if len(heartbeat.events) != 1 {
		t.Fatalf("expected one heartbeat event, got %d", len(heartbeat.events))
	}
	if len(heartbeat.triggers) != 0 {
		t.Fatalf("expected no heartbeat trigger on success")
	}
}

func TestGatewaySpawnFailureTriggersHeartbeat(t *testing.T) {
	spawner := subagentservice.NewSpawner()
	heartbeat := &heartbeatSinkStub{}
	runtimeStub := &runtimeRunnerStub{
		err: errors.New("timeout exceeded"),
	}
	gateway := NewGatewayService(spawner, nil, nil, nil, runtimeStub, nil, nil, nil, nil, heartbeat)

	_, err := gateway.Spawn(context.Background(), subagentservice.SpawnRequest{
		ParentSessionKey: "session-parent",
		Task:             "run fail task",
	})
	if err != nil {
		t.Fatalf("spawn should still be accepted: %v", err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for len(heartbeat.events) == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if len(heartbeat.events) == 0 {
		t.Fatalf("expected heartbeat event for failed run")
	}
	if len(heartbeat.triggers) == 0 {
		t.Fatalf("expected heartbeat trigger for failed run")
	}
	if heartbeat.triggers[0].Reason != "subagent-event" {
		t.Fatalf("unexpected heartbeat trigger reason: %s", heartbeat.triggers[0].Reason)
	}
}

func TestResolveSubagentToolsPolicyChain(t *testing.T) {
	gateway := &GatewayService{}
	config := gateway.resolveSubagentTools(context.Background(), map[string]any{
		"globalToolsPolicy": map[string]any{
			"allow": []any{"read"},
			"deny":  []any{"exec"},
		},
		"channelToolsPolicy": map[string]any{
			"alsoAllow": []any{"write"},
		},
		"providerToolsPolicy": map[string]any{
			"deny": []any{"write"},
		},
		"subagentToolsPolicy": map[string]any{
			"alsoAllow": []any{"process"},
			"deny":      []any{"edit"},
		},
	}, 1, 1)

	joinedAllow := strings.Join(config.AllowList, ",")
	if !strings.Contains(joinedAllow, "read") || !strings.Contains(joinedAllow, "write") || !strings.Contains(joinedAllow, "process") {
		t.Fatalf("unexpected allow list: %v", config.AllowList)
	}
	joinedDeny := strings.Join(config.DenyList, ",")
	for _, expected := range []string{"exec", "write", "edit", "sessions_spawn"} {
		if !strings.Contains(joinedDeny, expected) {
			t.Fatalf("deny list should contain %s, got %v", expected, config.DenyList)
		}
	}
}
