package heartbeat

import (
	"strings"
	"sync"
	"time"
)

const maxSystemEvents = 20

var allowedSystemEventPrefixes = map[string]struct{}{
	"exec":     {},
	"subagent": {},
	"cron":     {},
	"system":   {},
}

type SystemEvent struct {
	Text       string
	Timestamp  time.Time
	ContextKey string
	RunID      string
	Source     string
}

type SystemEventQueue struct {
	mu      sync.Mutex
	entries map[string]*systemEventSession
	now     func() time.Time
}

type systemEventSession struct {
	queue     []SystemEvent
	seen      map[string]struct{}
	touchedAt time.Time
}

const maxSystemEventSessions = 256
const systemEventSessionTTL = 24 * time.Hour

func NewSystemEventQueue() *SystemEventQueue {
	return &SystemEventQueue{
		entries: make(map[string]*systemEventSession),
		now:     time.Now,
	}
}

func (queue *SystemEventQueue) Enqueue(input SystemEventInput) bool {
	if queue == nil {
		return false
	}
	key := strings.TrimSpace(input.SessionKey)
	if key == "" {
		return false
	}
	cleaned := strings.TrimSpace(input.Text)
	if cleaned == "" {
		return false
	}
	source := normalizeSource(input.Source)
	contextKey := normalizeContextKey(input.ContextKey, source)
	runID := strings.TrimSpace(input.RunID)
	dedupeKey := buildEventDedupeKey(contextKey, runID, cleaned)
	now := queue.now()

	queue.mu.Lock()
	defer queue.mu.Unlock()
	queue.pruneSessionsLocked(now)
	entry := queue.entries[key]
	if entry == nil {
		entry = &systemEventSession{
			seen: make(map[string]struct{}),
		}
		queue.entries[key] = entry
	}
	if _, exists := entry.seen[dedupeKey]; exists {
		return false
	}
	event := SystemEvent{
		Text:       cleaned,
		Timestamp:  now,
		ContextKey: contextKey,
		RunID:      runID,
		Source:     source,
	}
	entry.queue = append(entry.queue, event)
	entry.seen[dedupeKey] = struct{}{}
	entry.touchedAt = now
	for len(entry.queue) > maxSystemEvents {
		removed := entry.queue[0]
		entry.queue = entry.queue[1:]
		delete(entry.seen, buildEventDedupeKey(removed.ContextKey, strings.TrimSpace(removed.RunID), strings.TrimSpace(removed.Text)))
	}
	queue.evictOverflowLocked()
	return true
}

func (queue *SystemEventQueue) Drain(sessionKey string) []SystemEvent {
	if queue == nil {
		return nil
	}
	key := strings.TrimSpace(sessionKey)
	if key == "" {
		return nil
	}

	queue.mu.Lock()
	defer queue.mu.Unlock()
	queue.pruneSessionsLocked(queue.now())
	entry := queue.entries[key]
	if entry == nil || len(entry.queue) == 0 {
		return nil
	}
	out := append([]SystemEvent(nil), entry.queue...)
	delete(queue.entries, key)
	return out
}

func (queue *SystemEventQueue) Has(sessionKey string) bool {
	if queue == nil {
		return false
	}
	key := strings.TrimSpace(sessionKey)
	if key == "" {
		return false
	}
	queue.mu.Lock()
	defer queue.mu.Unlock()
	queue.pruneSessionsLocked(queue.now())
	entry := queue.entries[key]
	if entry == nil || len(entry.queue) == 0 {
		return false
	}
	entry.touchedAt = queue.now()
	return true
}

func (queue *SystemEventQueue) pruneSessionsLocked(now time.Time) {
	if queue == nil {
		return
	}
	if systemEventSessionTTL > 0 {
		for key, entry := range queue.entries {
			if entry == nil {
				delete(queue.entries, key)
				continue
			}
			if !entry.touchedAt.IsZero() && now.Sub(entry.touchedAt) > systemEventSessionTTL {
				delete(queue.entries, key)
			}
		}
	}
}

func (queue *SystemEventQueue) evictOverflowLocked() {
	for len(queue.entries) > maxSystemEventSessions {
		oldestKey := ""
		oldestAt := time.Time{}
		for key, entry := range queue.entries {
			if entry == nil {
				oldestKey = key
				break
			}
			if oldestKey == "" || entry.touchedAt.Before(oldestAt) {
				oldestKey = key
				oldestAt = entry.touchedAt
			}
		}
		if oldestKey == "" {
			return
		}
		delete(queue.entries, oldestKey)
	}
}

func normalizeSource(value string) string {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	if _, ok := allowedSystemEventPrefixes[trimmed]; ok {
		return trimmed
	}
	return "system"
}

func normalizeContextKey(value string, source string) string {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	if trimmed != "" {
		parts := strings.SplitN(trimmed, ":", 2)
		prefix := strings.TrimSpace(parts[0])
		if _, ok := allowedSystemEventPrefixes[prefix]; ok {
			return trimmed
		}
		if len(parts) == 2 && strings.TrimSpace(parts[1]) != "" {
			return "system:" + strings.TrimSpace(parts[1])
		}
	}
	if source == "" {
		source = "system"
	}
	return source + ":event"
}

func buildEventDedupeKey(contextKey string, runID string, text string) string {
	return strings.TrimSpace(contextKey) + "|" + strings.TrimSpace(runID) + "|" + strings.TrimSpace(text)
}
