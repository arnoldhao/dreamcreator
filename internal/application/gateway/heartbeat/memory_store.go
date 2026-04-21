package heartbeat

import (
	"context"
	"strings"
	"sync"
	"time"
)

type MemoryEventStore struct {
	mu      sync.RWMutex
	entries map[string]*memoryEventSession
	now     func() time.Time
}

type memoryEventSession struct {
	last      Event
	recent    []Event
	touchedAt time.Time
}

const maxMemoryEventsPerSession = 256
const maxMemoryEventSessions = 512
const memoryEventSessionTTL = 24 * time.Hour

func NewMemoryEventStore() *MemoryEventStore {
	return &MemoryEventStore{
		entries: make(map[string]*memoryEventSession),
		now:     time.Now,
	}
}

func (store *MemoryEventStore) Save(_ context.Context, event Event) error {
	if store == nil {
		return nil
	}
	key := strings.TrimSpace(event.SessionKey)
	if event.CreatedAt.IsZero() {
		event.CreatedAt = store.now()
	}
	store.mu.Lock()
	store.pruneSessionsLocked(store.now())
	session := store.entries[key]
	if session == nil {
		session = &memoryEventSession{}
		store.entries[key] = session
	}
	if session.last.CreatedAt.IsZero() || event.CreatedAt.After(session.last.CreatedAt) {
		session.last = event
	}
	if len(session.recent) < maxMemoryEventsPerSession {
		session.recent = append(session.recent, event)
	} else {
		copy(session.recent, session.recent[1:])
		session.recent[len(session.recent)-1] = event
	}
	session.touchedAt = store.now()
	store.evictOverflowLocked()
	store.mu.Unlock()
	return nil
}

func (store *MemoryEventStore) Last(_ context.Context, sessionKey string) (Event, error) {
	if store == nil {
		return Event{}, ErrEventNotFound
	}
	key := strings.TrimSpace(sessionKey)
	store.mu.Lock()
	defer store.mu.Unlock()
	store.pruneSessionsLocked(store.now())
	session := store.entries[key]
	if session == nil || session.last.CreatedAt.IsZero() {
		return Event{}, ErrEventNotFound
	}
	session.touchedAt = store.now()
	return session.last, nil
}

func (store *MemoryEventStore) HasDuplicate(_ context.Context, sessionKey string, contentHash string, since time.Time) (bool, error) {
	if store == nil {
		return false, nil
	}
	key := strings.TrimSpace(sessionKey)
	hash := strings.TrimSpace(contentHash)
	if key == "" || hash == "" {
		return false, nil
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	store.pruneSessionsLocked(store.now())
	session := store.entries[key]
	if session == nil {
		return false, nil
	}
	session.touchedAt = store.now()
	for index := len(session.recent) - 1; index >= 0; index-- {
		item := session.recent[index]
		if strings.TrimSpace(item.ContentHash) != hash {
			continue
		}
		if item.CreatedAt.Before(since) {
			continue
		}
		return true, nil
	}
	return false, nil
}

func (store *MemoryEventStore) pruneSessionsLocked(now time.Time) {
	if store == nil {
		return
	}
	if memoryEventSessionTTL > 0 {
		for key, session := range store.entries {
			if session == nil {
				delete(store.entries, key)
				continue
			}
			if !session.touchedAt.IsZero() && now.Sub(session.touchedAt) > memoryEventSessionTTL {
				delete(store.entries, key)
			}
		}
	}
}

func (store *MemoryEventStore) evictOverflowLocked() {
	for len(store.entries) > maxMemoryEventSessions {
		oldestKey := ""
		oldestAt := time.Time{}
		for key, session := range store.entries {
			if session == nil {
				oldestKey = key
				break
			}
			if oldestKey == "" || session.touchedAt.Before(oldestAt) {
				oldestKey = key
				oldestAt = session.touchedAt
			}
		}
		if oldestKey == "" {
			return
		}
		delete(store.entries, oldestKey)
	}
}

func sanitizeSpec(spec Spec) Spec {
	items := make([]ChecklistItem, 0, len(spec.Items))
	for _, item := range spec.Items {
		id := strings.TrimSpace(item.ID)
		text := strings.TrimSpace(item.Text)
		if id == "" && text == "" {
			continue
		}
		items = append(items, ChecklistItem{
			ID:       id,
			Text:     text,
			Done:     item.Done,
			Priority: strings.TrimSpace(item.Priority),
		})
	}
	result := Spec{
		Title:     strings.TrimSpace(spec.Title),
		Items:     items,
		Notes:     strings.TrimSpace(spec.Notes),
		Version:   spec.Version,
		UpdatedAt: spec.UpdatedAt,
	}
	return result
}

func cloneSpec(spec Spec) Spec {
	items := make([]ChecklistItem, len(spec.Items))
	copy(items, spec.Items)
	return Spec{
		Title:     spec.Title,
		Items:     items,
		Notes:     spec.Notes,
		Version:   spec.Version,
		UpdatedAt: spec.UpdatedAt,
	}
}
