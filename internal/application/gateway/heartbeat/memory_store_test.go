package heartbeat

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestMemoryEventStoreRetainsBoundedRecentEventsAndLast(t *testing.T) {
	t.Parallel()

	store := NewMemoryEventStore()
	sessionKey := "session-1"
	base := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)

	for index := 0; index < maxMemoryEventsPerSession+24; index++ {
		err := store.Save(context.Background(), Event{
			ID:          fmt.Sprintf("evt-%03d", index),
			SessionKey:  sessionKey,
			ContentHash: fmt.Sprintf("hash-%03d", index),
			CreatedAt:   base.Add(time.Duration(index) * time.Second),
		})
		if err != nil {
			t.Fatalf("save event %d: %v", index, err)
		}
	}

	store.mu.RLock()
	session := store.entries[sessionKey]
	store.mu.RUnlock()
	if session == nil {
		t.Fatalf("expected session cache")
	}
	if len(session.recent) != maxMemoryEventsPerSession {
		t.Fatalf("expected bounded recent cache size %d, got %d", maxMemoryEventsPerSession, len(session.recent))
	}
	if got := session.recent[0].ID; got != "evt-024" {
		t.Fatalf("expected oldest retained event evt-024, got %q", got)
	}

	last, err := store.Last(context.Background(), sessionKey)
	if err != nil {
		t.Fatalf("last event: %v", err)
	}
	if last.ID != fmt.Sprintf("evt-%03d", maxMemoryEventsPerSession+23) {
		t.Fatalf("unexpected last event %q", last.ID)
	}
}

func TestMemoryEventStoreHasDuplicateChecksRetainedWindow(t *testing.T) {
	t.Parallel()

	store := NewMemoryEventStore()
	sessionKey := "session-2"
	base := time.Date(2026, time.February, 3, 4, 5, 6, 0, time.UTC)

	for index := 0; index < maxMemoryEventsPerSession+8; index++ {
		hash := fmt.Sprintf("hash-%03d", index)
		if index == maxMemoryEventsPerSession+7 {
			hash = "target-hash"
		}
		err := store.Save(context.Background(), Event{
			ID:          fmt.Sprintf("evt-%03d", index),
			SessionKey:  sessionKey,
			ContentHash: hash,
			CreatedAt:   base.Add(time.Duration(index) * time.Second),
		})
		if err != nil {
			t.Fatalf("save event %d: %v", index, err)
		}
	}

	duplicate, err := store.HasDuplicate(context.Background(), sessionKey, "target-hash", base)
	if err != nil {
		t.Fatalf("has duplicate: %v", err)
	}
	if !duplicate {
		t.Fatalf("expected retained duplicate to be found")
	}

	trimmedHash := "hash-000"
	duplicate, err = store.HasDuplicate(context.Background(), sessionKey, trimmedHash, base)
	if err != nil {
		t.Fatalf("has duplicate for trimmed hash: %v", err)
	}
	if duplicate {
		t.Fatalf("expected trimmed hash %q to be evicted from recent cache", trimmedHash)
	}
}

func TestMemoryEventStorePrunesExpiredSessions(t *testing.T) {
	store := NewMemoryEventStore()
	base := time.Date(2026, time.April, 5, 6, 7, 8, 0, time.UTC)
	store.now = func() time.Time { return base }

	if err := store.Save(context.Background(), Event{
		ID:         "old",
		SessionKey: "session-old",
		CreatedAt:  base,
	}); err != nil {
		t.Fatalf("save old session: %v", err)
	}

	store.now = func() time.Time { return base.Add(memoryEventSessionTTL + time.Minute) }
	if err := store.Save(context.Background(), Event{
		ID:         "new",
		SessionKey: "session-new",
		CreatedAt:  base.Add(memoryEventSessionTTL + time.Minute),
	}); err != nil {
		t.Fatalf("save new session: %v", err)
	}

	store.mu.RLock()
	defer store.mu.RUnlock()
	if _, ok := store.entries["session-old"]; ok {
		t.Fatalf("expected stale memory-event session to be pruned")
	}
	if _, ok := store.entries["session-new"]; !ok {
		t.Fatalf("expected fresh memory-event session to remain")
	}
}
