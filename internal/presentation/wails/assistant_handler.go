package wails

import (
	"context"

	"dreamcreator/internal/application/assistant/dto"
	"dreamcreator/internal/application/assistant/service"
)

type AssistantHandler struct {
	service *service.AssistantService
}

func NewAssistantHandler(service *service.AssistantService) *AssistantHandler {
	return &AssistantHandler{service: service}
}

func (handler *AssistantHandler) ServiceName() string {
	return "AssistantHandler"
}

func (handler *AssistantHandler) ListAssistants(ctx context.Context, includeDisabled bool) ([]dto.Assistant, error) {
	return handler.service.ListAssistants(ctx, includeDisabled)
}

func (handler *AssistantHandler) GetAssistant(ctx context.Context, id string) (dto.Assistant, error) {
	return handler.service.GetAssistant(ctx, id)
}

func (handler *AssistantHandler) CreateAssistant(ctx context.Context, request dto.CreateAssistantRequest) (dto.Assistant, error) {
	return handler.service.CreateAssistant(ctx, request)
}

func (handler *AssistantHandler) UpdateAssistant(ctx context.Context, request dto.UpdateAssistantRequest) (dto.Assistant, error) {
	return handler.service.UpdateAssistant(ctx, request)
}

func (handler *AssistantHandler) DeleteAssistant(ctx context.Context, request dto.DeleteAssistantRequest) error {
	return handler.service.DeleteAssistant(ctx, request)
}

func (handler *AssistantHandler) SetDefaultAssistant(ctx context.Context, request dto.SetDefaultAssistantRequest) error {
	return handler.service.SetDefaultAssistant(ctx, request)
}

func (handler *AssistantHandler) ListAvatarAssets(ctx context.Context, kind string) ([]dto.AssistantAvatarAsset, error) {
	return handler.service.ListAvatarAssets(ctx, kind)
}

func (handler *AssistantHandler) ImportAvatarAsset(ctx context.Context, request dto.ImportAssistantAvatarRequest) (dto.AssistantAvatarAsset, error) {
	return handler.service.ImportAvatarAsset(ctx, request)
}

func (handler *AssistantHandler) ImportAvatarAssetFromPath(ctx context.Context, request dto.ImportAssistantAvatarFromPathRequest) (dto.AssistantAvatarAsset, error) {
	return handler.service.ImportAvatarAssetFromPath(ctx, request)
}

func (handler *AssistantHandler) ReadAvatarSource(ctx context.Context, request dto.ReadAssistantAvatarSourceRequest) (dto.ReadAssistantAvatarSourceResponse, error) {
	return handler.service.ReadAvatarSource(ctx, request)
}

func (handler *AssistantHandler) DeleteAvatarAsset(ctx context.Context, request dto.DeleteAssistantAvatarAssetRequest) error {
	return handler.service.DeleteAvatarAsset(ctx, request)
}

func (handler *AssistantHandler) UpdateAvatarAsset(ctx context.Context, request dto.UpdateAssistantAvatarAssetRequest) (dto.AssistantAvatarAsset, error) {
	return handler.service.UpdateAvatarAsset(ctx, request)
}

func (handler *AssistantHandler) GetAssistantMemorySummary(ctx context.Context, assistantID string) (dto.AssistantMemorySummary, error) {
	return handler.service.GetAssistantMemorySummary(ctx, assistantID)
}

func (handler *AssistantHandler) GetAssistantProfileOptions(ctx context.Context) (dto.AssistantProfileOptions, error) {
	return handler.service.GetAssistantProfileOptions(ctx)
}

func (handler *AssistantHandler) RefreshAssistantUserLocale(ctx context.Context, assistantID string) (dto.Assistant, error) {
	return handler.service.RefreshAssistantUserLocale(ctx, assistantID)
}
