package heartbeat

import "testing"

func TestParseHeartbeatResponse_ExactAck(t *testing.T) {
	t.Parallel()

	parsed := parseHeartbeatResponse("HEARTBEAT_OK")
	if !parsed.Ack {
		t.Fatalf("expected ack=true")
	}
	if parsed.Cleaned != "" {
		t.Fatalf("expected cleaned empty for exact ack, got %q", parsed.Cleaned)
	}
}

func TestParseHeartbeatResponse_JSONAlert(t *testing.T) {
	t.Parallel()

	parsed := parseHeartbeatResponse(`{"code":"heartbeat.exec_attention","severity":"error","params":{"detail":"exec failed"}}`)
	if parsed.Ack {
		t.Fatalf("expected ack=false for alert payload")
	}
	if parsed.Alert.Code != "heartbeat.exec_attention" {
		t.Fatalf("unexpected code %q", parsed.Alert.Code)
	}
	if parsed.Cleaned != "exec failed" {
		t.Fatalf("unexpected cleaned detail %q", parsed.Cleaned)
	}
}
