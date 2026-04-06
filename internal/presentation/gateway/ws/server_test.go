package ws

import (
	"testing"
)

func TestNegotiateProtocol(t *testing.T) {
	if version, ok := negotiateProtocol(0, 0); !ok || version == 0 {
		t.Fatalf("expected default protocol")
	}
	if _, ok := negotiateProtocol(2, 3); ok {
		t.Fatalf("expected mismatch for unsupported protocol")
	}
}

func TestParseRequestFrame(t *testing.T) {
	raw := []byte(`{"type":"req","id":"1","method":"ping","params":{}}`)
	frame, ok := parseRequestFrame(raw)
	if !ok {
		t.Fatalf("expected parse ok")
	}
	if frame.Method != "ping" || frame.ID != "1" {
		t.Fatalf("unexpected frame: %#v", frame)
	}
}
