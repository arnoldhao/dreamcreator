package workspace

import "context"

type Repository interface {
	GetGlobal(ctx context.Context) (GlobalWorkspace, error)
	SaveGlobal(ctx context.Context, workspace GlobalWorkspace) error

	GetAssistantWorkspace(ctx context.Context, assistantID string) (AssistantWorkspace, error)
	SaveAssistantWorkspace(ctx context.Context, workspace AssistantWorkspace) error
	UpdateAssistantWorkspace(ctx context.Context, workspace AssistantWorkspace, expectedVersion int64) error
	GetAssistantWorkspaceSnapshot(ctx context.Context, assistantID string, version int64) (AssistantWorkspaceSnapshot, error)
	SaveAssistantWorkspaceSnapshot(ctx context.Context, snapshot AssistantWorkspaceSnapshot) error
}
