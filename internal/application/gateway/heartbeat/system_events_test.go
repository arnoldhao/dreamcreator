package heartbeat

import (
	"testing"
	"time"
)

func TestSystemEventQueuePrunesExpiredSessions(t *testing.T) {
	queue := NewSystemEventQueue()
	base := time.Date(2026, time.March, 4, 5, 6, 7, 0, time.UTC)
	queue.now = func() time.Time { return base }

	if !queue.Enqueue(SystemEventInput{SessionKey: "session-stale", Text: "first"}) {
		t.Fatalf("expected initial enqueue to succeed")
	}

	queue.now = func() time.Time { return base.Add(systemEventSessionTTL + time.Minute) }
	if !queue.Enqueue(SystemEventInput{SessionKey: "session-fresh", Text: "second"}) {
		t.Fatalf("expected second enqueue to succeed")
	}

	queue.mu.Lock()
	defer queue.mu.Unlock()
	if _, ok := queue.entries["session-stale"]; ok {
		t.Fatalf("expected stale system-event session to be pruned")
	}
	if _, ok := queue.entries["session-fresh"]; !ok {
		t.Fatalf("expected fresh system-event session to remain")
	}
}
