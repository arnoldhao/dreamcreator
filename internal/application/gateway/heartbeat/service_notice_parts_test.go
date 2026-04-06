package heartbeat

import (
	"encoding/json"
	"testing"
)

func TestBuildHeartbeatAssistantMessageParts_WithCronNotice(t *testing.T) {
	parts := buildHeartbeatAssistantMessageParts(
		"Reminder: check release notes",
		"cron.wake.now",
		"",
		"run-1",
		true,
		false,
	)
	if len(parts) != 2 {
		t.Fatalf("expected 2 parts, got %d", len(parts))
	}
	if parts[0].Type != "data" {
		t.Fatalf("expected first part type data, got %q", parts[0].Type)
	}
	if parts[1].Type != "text" {
		t.Fatalf("expected second part type text, got %q", parts[1].Type)
	}
	if parts[1].Text != "Reminder: check release notes" {
		t.Fatalf("unexpected text part content: %q", parts[1].Text)
	}
	if parts[0].ParentID == "" || parts[0].ParentID != parts[1].ParentID {
		t.Fatalf("expected matching parent ids, got data=%q text=%q", parts[0].ParentID, parts[1].ParentID)
	}

	payload := map[string]any{}
	if err := json.Unmarshal(parts[0].Data, &payload); err != nil {
		t.Fatalf("unmarshal data part failed: %v", err)
	}
	if payload["name"] != "system_notice" {
		t.Fatalf("expected name system_notice, got %#v", payload["name"])
	}
	data, ok := payload["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected data payload object, got %#v", payload["data"])
	}
	if data["origin"] != "heartbeat" {
		t.Fatalf("expected origin heartbeat, got %#v", data["origin"])
	}
	if data["source"] != "cron" {
		t.Fatalf("expected source cron, got %#v", data["source"])
	}
	if data["kind"] != "cron_reminder" {
		t.Fatalf("expected kind cron_reminder, got %#v", data["kind"])
	}
	if data["runId"] != "run-1" {
		t.Fatalf("expected runId run-1, got %#v", data["runId"])
	}
	if data["reason"] != "cron.wake.now" {
		t.Fatalf("expected reason cron.wake.now, got %#v", data["reason"])
	}
}

func TestBuildHeartbeatAssistantMessageParts_WithoutContent(t *testing.T) {
	parts := buildHeartbeatAssistantMessageParts("", "interval", "", "", false, false)
	if len(parts) != 0 {
		t.Fatalf("expected empty parts for empty content, got %d", len(parts))
	}
}

func TestResolveHeartbeatNoticeSource_InfersExecFromReason(t *testing.T) {
	source := resolveHeartbeatNoticeSource("", "exec.completed", false, false)
	if source != "exec" {
		t.Fatalf("expected source exec, got %q", source)
	}
}
