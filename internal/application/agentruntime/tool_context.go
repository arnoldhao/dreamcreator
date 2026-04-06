package agentruntime

import (
	"context"
	"strings"
)

type toolCallContextKey struct{}

type toolCallContext struct {
	id   string
	name string
}

func WithToolCallContext(ctx context.Context, toolCallID string, toolName string) context.Context {
	if ctx == nil {
		return ctx
	}
	value := toolCallContext{
		id:   strings.TrimSpace(toolCallID),
		name: strings.TrimSpace(toolName),
	}
	return context.WithValue(ctx, toolCallContextKey{}, value)
}

func ToolCallContextFromContext(ctx context.Context) (string, string) {
	if ctx == nil {
		return "", ""
	}
	value, ok := ctx.Value(toolCallContextKey{}).(toolCallContext)
	if !ok {
		return "", ""
	}
	return value.id, value.name
}
