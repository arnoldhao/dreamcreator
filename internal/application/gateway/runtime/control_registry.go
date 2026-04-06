package runtime

import (
	"strings"
	"sync"

	"dreamcreator/internal/application/agentruntime"
)

type ControlRegistry struct {
	mu      sync.Mutex
	entries map[string]*agentruntime.AgentController
}

func NewControlRegistry() *ControlRegistry {
	return &ControlRegistry{
		entries: make(map[string]*agentruntime.AgentController),
	}
}

func (registry *ControlRegistry) Register(runID string, controller *agentruntime.AgentController) {
	if registry == nil || controller == nil {
		return
	}
	trimmed := strings.TrimSpace(runID)
	if trimmed == "" {
		return
	}
	registry.mu.Lock()
	registry.entries[trimmed] = controller
	registry.mu.Unlock()
}

func (registry *ControlRegistry) Unregister(runID string) {
	if registry == nil {
		return
	}
	trimmed := strings.TrimSpace(runID)
	if trimmed == "" {
		return
	}
	registry.mu.Lock()
	delete(registry.entries, trimmed)
	registry.mu.Unlock()
}

func (registry *ControlRegistry) Steer(runID string, message string) bool {
	controller := registry.lookup(runID)
	if controller == nil {
		return false
	}
	controller.Steer(message)
	return true
}

func (registry *ControlRegistry) FollowUp(runID string, message string) bool {
	controller := registry.lookup(runID)
	if controller == nil {
		return false
	}
	controller.FollowUp(message)
	return true
}

func (registry *ControlRegistry) lookup(runID string) *agentruntime.AgentController {
	if registry == nil {
		return nil
	}
	trimmed := strings.TrimSpace(runID)
	if trimmed == "" {
		return nil
	}
	registry.mu.Lock()
	controller := registry.entries[trimmed]
	registry.mu.Unlock()
	return controller
}
