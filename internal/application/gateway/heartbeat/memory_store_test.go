package heartbeat

import (
	"testing"
	"time"
)

func TestSanitizeSpec_TrimAndDropEmptyItems(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.February, 26, 8, 0, 0, 0, time.UTC)
	cleaned := sanitizeSpec(Spec{
		Title:     " Daily ",
		Notes:     " track async jobs ",
		Version:   2,
		UpdatedAt: now,
		Items: []ChecklistItem{
			{ID: "1", Text: " sync ", Done: false, Priority: " high "},
			{ID: " ", Text: " ", Done: false, Priority: ""},
		},
	})
	if cleaned.Title != "Daily" {
		t.Fatalf("unexpected title: %q", cleaned.Title)
	}
	if cleaned.Notes != "track async jobs" {
		t.Fatalf("unexpected notes: %q", cleaned.Notes)
	}
	if cleaned.Version != 2 {
		t.Fatalf("unexpected version: %d", cleaned.Version)
	}
	if len(cleaned.Items) != 1 {
		t.Fatalf("unexpected item count: %d", len(cleaned.Items))
	}
	if cleaned.Items[0].Text != "sync" {
		t.Fatalf("unexpected item text: %q", cleaned.Items[0].Text)
	}
	if cleaned.Items[0].Priority != "high" {
		t.Fatalf("unexpected item priority: %q", cleaned.Items[0].Priority)
	}
}

func TestSystemEventQueue_DeduplicatesByContextRunAndText(t *testing.T) {
	t.Parallel()

	queue := NewSystemEventQueue()
	input := SystemEventInput{
		SessionKey: "session-1",
		Text:       "done",
		ContextKey: "subagent:success",
		RunID:      "run-1",
		Source:     "subagent",
	}
	if !queue.Enqueue(input) {
		t.Fatalf("first enqueue should be queued")
	}
	if queue.Enqueue(input) {
		t.Fatalf("duplicate enqueue should be skipped")
	}
	if !queue.Enqueue(SystemEventInput{
		SessionKey: "session-1",
		Text:       "done",
		ContextKey: "subagent:success",
		RunID:      "run-2",
		Source:     "subagent",
	}) {
		t.Fatalf("different run id should be queued")
	}
	items := queue.Drain("session-1")
	if len(items) != 2 {
		t.Fatalf("expected 2 events, got %d", len(items))
	}
}
