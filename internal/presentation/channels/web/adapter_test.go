package web

import (
	"encoding/json"
	"testing"

	"dreamcreator/internal/application/agentruntime"
)

func TestEncodeAgentEvent(t *testing.T) {
	adapter := NewAdapter()
	encoded, err := adapter.EncodeAgentEvent(agentruntime.Event{
		Type:  agentruntime.EventTextDelta,
		Delta: "hello",
	})
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}
	if encoded.Type != "data-agent-event" {
		t.Fatalf("unexpected event type: %s", encoded.Type)
	}
	if encoded.Transient == nil || !*encoded.Transient {
		t.Fatalf("expected transient event")
	}

	var decoded agentruntime.Event
	if err := json.Unmarshal(encoded.Data, &decoded); err != nil {
		t.Fatalf("decode payload failed: %v", err)
	}
	if decoded.Type != agentruntime.EventTextDelta {
		t.Fatalf("unexpected decoded event type: %s", decoded.Type)
	}
	if decoded.Delta != "hello" {
		t.Fatalf("unexpected decoded delta: %q", decoded.Delta)
	}
}
