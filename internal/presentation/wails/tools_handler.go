package wails

import (
	"context"

	"dreamcreator/internal/application/tools/dto"
	"dreamcreator/internal/application/tools/service"
)

type ToolsHandler struct {
	service     *service.ToolService
	listService ToolListService
}

type ToolListService interface {
	ListTools(ctx context.Context) []dto.ToolSpec
}

type toolListAdapter struct {
	service *service.ToolService
}

func (adapter toolListAdapter) ListTools(_ context.Context) []dto.ToolSpec {
	if adapter.service == nil {
		return nil
	}
	return adapter.service.ListTools()
}

func NewToolsHandler(service *service.ToolService, listService ToolListService) *ToolsHandler {
	if listService == nil {
		listService = toolListAdapter{service: service}
	}
	return &ToolsHandler{
		service:     service,
		listService: listService,
	}
}

func (handler *ToolsHandler) ServiceName() string {
	return "ToolsHandler"
}

func (handler *ToolsHandler) ListTools(ctx context.Context) ([]dto.ToolSpec, error) {
	if handler == nil || handler.listService == nil {
		return nil, nil
	}
	return handler.listService.ListTools(ctx), nil
}

func (handler *ToolsHandler) RegisterTool(ctx context.Context, request dto.RegisterToolRequest) (dto.ToolSpec, error) {
	return handler.service.RegisterTool(ctx, request)
}

func (handler *ToolsHandler) EnableTool(ctx context.Context, request dto.EnableToolRequest) error {
	return handler.service.EnableTool(ctx, request)
}

func (handler *ToolsHandler) ExecuteTool(ctx context.Context, request dto.ExecuteToolRequest) (dto.ToolResult, error) {
	return handler.service.ExecuteTool(ctx, request)
}

func (handler *ToolsHandler) QueryToolLogs(ctx context.Context, request dto.QueryToolLogsRequest) ([]dto.ToolInvocation, error) {
	return handler.service.QueryToolLogs(ctx, request)
}
