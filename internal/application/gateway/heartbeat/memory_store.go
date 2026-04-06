package heartbeat

import (
	"context"
	"strings"
	"sync"
	"time"
)

type MemoryEventStore struct {
	mu      sync.RWMutex
	entries map[string][]Event
}

func NewMemoryEventStore() *MemoryEventStore {
	return &MemoryEventStore{
		entries: make(map[string][]Event),
	}
}

func (store *MemoryEventStore) Save(_ context.Context, event Event) error {
	if store == nil {
		return nil
	}
	key := strings.TrimSpace(event.SessionKey)
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}
	store.mu.Lock()
	store.entries[key] = append(store.entries[key], event)
	store.mu.Unlock()
	return nil
}

func (store *MemoryEventStore) Last(_ context.Context, sessionKey string) (Event, error) {
	if store == nil {
		return Event{}, ErrEventNotFound
	}
	key := strings.TrimSpace(sessionKey)
	store.mu.RLock()
	defer store.mu.RUnlock()
	items := store.entries[key]
	if len(items) == 0 {
		return Event{}, ErrEventNotFound
	}
	latest := items[0]
	for _, item := range items[1:] {
		if item.CreatedAt.After(latest.CreatedAt) {
			latest = item
		}
	}
	return latest, nil
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
	store.mu.RLock()
	defer store.mu.RUnlock()
	for _, item := range store.entries[key] {
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
