package runtime

import (
	"context"
	"sync"
)

type AbortRegistry struct {
	mu      sync.Mutex
	entries map[string]*abortEntry
}

type abortEntry struct {
	cancel  context.CancelFunc
	reason  string
	aborted bool
}

func NewAbortRegistry() *AbortRegistry {
	return &AbortRegistry{
		entries: make(map[string]*abortEntry),
	}
}

func (registry *AbortRegistry) Register(runID string, cancel context.CancelFunc) {
	if registry == nil || runID == "" || cancel == nil {
		return
	}
	registry.mu.Lock()
	registry.entries[runID] = &abortEntry{cancel: cancel}
	registry.mu.Unlock()
}

func (registry *AbortRegistry) Unregister(runID string) {
	if registry == nil || runID == "" {
		return
	}
	registry.mu.Lock()
	delete(registry.entries, runID)
	registry.mu.Unlock()
}

func (registry *AbortRegistry) Abort(runID string, reason string) bool {
	if registry == nil || runID == "" {
		return false
	}
	registry.mu.Lock()
	entry := registry.entries[runID]
	if entry != nil {
		entry.reason = reason
		entry.aborted = true
	}
	registry.mu.Unlock()
	if entry == nil || entry.cancel == nil {
		return false
	}
	entry.cancel()
	return true
}

