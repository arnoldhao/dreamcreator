package tools

import (
	"context"
	"errors"
	"strings"
	"sync"

	tooldto "dreamcreator/internal/application/tools/dto"
)

type RegistryExecutor struct {
	mu    sync.RWMutex
	tools map[string]func(ctx context.Context, args string) (string, error)
}

func NewRegistryExecutor() *RegistryExecutor {
	return &RegistryExecutor{tools: make(map[string]func(ctx context.Context, args string) (string, error))}
}

func (executor *RegistryExecutor) Register(name string, fn func(ctx context.Context, args string) (string, error)) {
	if executor == nil {
		return
	}
	trimmed := strings.TrimSpace(name)
	if trimmed == "" || fn == nil {
		return
	}
	executor.mu.Lock()
	executor.tools[trimmed] = fn
	executor.mu.Unlock()
}

func (executor *RegistryExecutor) Execute(ctx context.Context, spec tooldto.ToolSpec, args string) (string, error) {
	if executor == nil {
		return "", errors.New("executor unavailable")
	}
	name := strings.TrimSpace(spec.Name)
	if name == "" {
		name = strings.TrimSpace(spec.ID)
	}
	if name == "" {
		return "", errors.New("tool name is required")
	}
	executor.mu.RLock()
	fn := executor.tools[name]
	executor.mu.RUnlock()
	if fn == nil {
		return "", errors.New("tool executor not registered")
	}
	return fn(ctx, args)
}
