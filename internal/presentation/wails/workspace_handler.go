package wails

import (
	"context"
	"errors"
	"strings"

	"dreamcreator/internal/application/workspace/dto"
	"dreamcreator/internal/application/workspace/service"
	"dreamcreator/internal/infrastructure/opener"
)

type WorkspaceHandler struct {
	service *service.WorkspaceService
}

func NewWorkspaceHandler(service *service.WorkspaceService) *WorkspaceHandler {
	return &WorkspaceHandler{service: service}
}

func (handler *WorkspaceHandler) ServiceName() string {
	return "WorkspaceHandler"
}

func (handler *WorkspaceHandler) GetGlobalWorkspace(ctx context.Context) (dto.GlobalWorkspace, error) {
	return handler.service.GetGlobal(ctx)
}

func (handler *WorkspaceHandler) UpdateGlobalWorkspace(ctx context.Context, request dto.UpdateGlobalWorkspaceRequest) (dto.GlobalWorkspace, error) {
	return handler.service.UpdateGlobal(ctx, request)
}

func (handler *WorkspaceHandler) GetAssistantWorkspaceDirectory(ctx context.Context, assistantID string) (dto.AssistantWorkspaceDirectory, error) {
	return handler.service.GetAssistantWorkspaceDirectory(ctx, assistantID)
}

func (handler *WorkspaceHandler) OpenAssistantWorkspaceDirectory(ctx context.Context, assistantID string) error {
	directory, err := handler.service.GetAssistantWorkspaceDirectory(ctx, assistantID)
	if err != nil {
		return err
	}
	path := strings.TrimSpace(directory.RootPath)
	if path == "" {
		return errors.New("assistant workspace directory not available")
	}
	return opener.OpenDirectory(path)
}

func (handler *WorkspaceHandler) GetAssistantWorkspace(ctx context.Context, assistantID string) (dto.AssistantWorkspace, error) {
	return handler.service.GetAssistantWorkspace(ctx, assistantID)
}

func (handler *WorkspaceHandler) UpdateAssistantWorkspace(ctx context.Context, request dto.UpdateWorkspaceRequest) (dto.UpdateWorkspaceResponse, error) {
	return handler.service.UpdateWorkspace(ctx, request)
}

func (handler *WorkspaceHandler) ResolveWorkspaceSnapshot(ctx context.Context, request dto.ResolveWorkspaceSnapshotRequest) (dto.ResolveWorkspaceSnapshotResponse, error) {
	return handler.service.ResolveWorkspaceSnapshot(ctx, request)
}

func (handler *WorkspaceHandler) ResolveRuntimeSnapshot(ctx context.Context, request dto.ResolveRuntimeSnapshotRequest) (dto.RuntimeSnapshot, error) {
	return handler.service.ResolveRuntimeSnapshot(ctx, request)
}
