package automation

import (
	"context"
	"errors"
	"strings"
	"sync"
)

type TriggerFunc func(ctx context.Context, action AutomationAction) error

type TriggerRegistry struct {
	mu       sync.RWMutex
	triggers map[string]TriggerFunc
}

func NewTriggerRegistry() *TriggerRegistry {
	return &TriggerRegistry{triggers: make(map[string]TriggerFunc)}
}

func (registry *TriggerRegistry) Register(id string, fn TriggerFunc) {
	if registry == nil || fn == nil {
		return
	}
	key := strings.TrimSpace(id)
	if key == "" {
		return
	}
	registry.mu.Lock()
	registry.triggers[key] = fn
	registry.mu.Unlock()
}

func (registry *TriggerRegistry) Trigger(ctx context.Context, id string, action AutomationAction) error {
	if registry == nil {
		return errors.New("trigger registry unavailable")
	}
	key := strings.TrimSpace(id)
	if key == "" {
		return errors.New("trigger id is required")
	}
	registry.mu.RLock()
	fn := registry.triggers[key]
	registry.mu.RUnlock()
	if fn == nil {
		return errors.New("trigger not registered")
	}
	return fn(ctx, action)
}
