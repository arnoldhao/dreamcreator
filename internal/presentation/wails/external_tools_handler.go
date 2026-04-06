package wails

import (
	"context"
	"strings"

	"dreamcreator/internal/application/externaltools/dto"
	"dreamcreator/internal/application/externaltools/service"
	"dreamcreator/internal/domain/externaltools"
	"dreamcreator/internal/infrastructure/opener"
)

type ExternalToolsHandler struct {
	service   *service.ExternalToolsService
	events    externalToolsEvents
	telemetry externalToolsTelemetry
}

type externalToolsEvents interface {
	EmitExternalToolsUpdated()
}

type externalToolsTelemetry interface {
	TrackExternalToolInstalled(ctx context.Context, toolName string)
}

func NewExternalToolsHandler(service *service.ExternalToolsService, events externalToolsEvents, telemetry externalToolsTelemetry) *ExternalToolsHandler {
	return &ExternalToolsHandler{
		service:   service,
		events:    events,
		telemetry: telemetry,
	}
}

func (handler *ExternalToolsHandler) ServiceName() string {
	return "ExternalToolsHandler"
}

func (handler *ExternalToolsHandler) ListTools(ctx context.Context) ([]dto.ExternalTool, error) {
	return handler.service.ListTools(ctx)
}

func (handler *ExternalToolsHandler) ListToolUpdates(ctx context.Context) ([]dto.ExternalToolUpdateInfo, error) {
	return handler.service.ListToolUpdates(ctx)
}

func (handler *ExternalToolsHandler) GetToolInstallState(ctx context.Context, request dto.GetExternalToolInstallStateRequest) (dto.ExternalToolInstallState, error) {
	return handler.service.GetInstallState(ctx, request)
}

func (handler *ExternalToolsHandler) InstallTool(ctx context.Context, request dto.InstallExternalToolRequest) (dto.ExternalTool, error) {
	name := strings.TrimSpace(request.Name)
	if name == "" {
		return dto.ExternalTool{}, externaltools.ErrInvalidTool
	}
	if state, err := handler.service.GetInstallState(ctx, dto.GetExternalToolInstallStateRequest{Name: name}); err == nil {
		switch strings.TrimSpace(state.Stage) {
		case "downloading", "extracting", "verifying":
			return dto.ExternalTool{Name: name}, nil
		}
	}
	go func(request dto.InstallExternalToolRequest) {
		result, err := handler.service.InstallTool(context.Background(), request)
		if handler.events != nil {
			handler.events.EmitExternalToolsUpdated()
		}
		if err == nil && handler.telemetry != nil {
			handler.telemetry.TrackExternalToolInstalled(context.Background(), result.Name)
		}
	}(request)
	return dto.ExternalTool{Name: name}, nil
}

func (handler *ExternalToolsHandler) VerifyTool(ctx context.Context, request dto.VerifyExternalToolRequest) (dto.ExternalTool, error) {
	result, err := handler.service.VerifyTool(ctx, request)
	if err == nil && handler.events != nil {
		handler.events.EmitExternalToolsUpdated()
	}
	return result, err
}

func (handler *ExternalToolsHandler) SetToolPath(ctx context.Context, request dto.SetExternalToolPathRequest) (dto.ExternalTool, error) {
	result, err := handler.service.SetToolPath(ctx, request)
	if err == nil && handler.events != nil {
		handler.events.EmitExternalToolsUpdated()
	}
	return result, err
}

func (handler *ExternalToolsHandler) RemoveTool(ctx context.Context, request dto.RemoveExternalToolRequest) error {
	if err := handler.service.RemoveTool(ctx, request); err != nil {
		return err
	}
	if handler.events != nil {
		handler.events.EmitExternalToolsUpdated()
	}
	return nil
}

func (handler *ExternalToolsHandler) OpenToolDirectory(ctx context.Context, request dto.OpenExternalToolDirectoryRequest) error {
	name := externaltools.ToolName(request.Name)
	if name == "" {
		return externaltools.ErrInvalidTool
	}
	dir, err := handler.service.ResolveToolDirectory(ctx, name)
	if err != nil {
		return err
	}
	return opener.OpenDirectory(dir)
}
