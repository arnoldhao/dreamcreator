package tools

import "context"

type ToolRepository interface {
	List(ctx context.Context) ([]ToolSpec, error)
	Get(ctx context.Context, id string) (ToolSpec, error)
	Save(ctx context.Context, tool ToolSpec) error
	Delete(ctx context.Context, id string) error
}

type InvocationRepository interface {
	Append(ctx context.Context, invocation ToolInvocation, result ToolResult) error
	ListByTool(ctx context.Context, toolID string, limit int) ([]ToolInvocation, error)
}

type ToolRunRepository interface {
	FindByKey(ctx context.Context, runID string, toolName string, inputHash string) (ToolRun, error)
	Create(ctx context.Context, run ToolRun) error
	Update(ctx context.Context, run ToolRun) error
}
