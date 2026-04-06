package wails

import (
	"context"

	"dreamcreator/internal/application/skills/dto"
	"dreamcreator/internal/application/skills/service"
)

type SkillsHandler struct {
	service *service.SkillsService
}

func NewSkillsHandler(service *service.SkillsService) *SkillsHandler {
	return &SkillsHandler{service: service}
}

func (handler *SkillsHandler) ServiceName() string {
	return "SkillsHandler"
}

func (handler *SkillsHandler) RegisterSkill(ctx context.Context, request dto.RegisterSkillRequest) (dto.ProviderSkillSpec, error) {
	return handler.service.RegisterSkill(ctx, request)
}

func (handler *SkillsHandler) EnableSkill(ctx context.Context, request dto.EnableSkillRequest) error {
	return handler.service.EnableSkill(ctx, request)
}

func (handler *SkillsHandler) DeleteSkill(ctx context.Context, request dto.DeleteSkillRequest) error {
	return handler.service.DeleteSkill(ctx, request)
}

func (handler *SkillsHandler) SearchSkills(ctx context.Context, request dto.SearchSkillsRequest) ([]dto.SkillSearchResult, error) {
	return handler.service.SearchSkills(ctx, request)
}

func (handler *SkillsHandler) InspectSkill(ctx context.Context, request dto.InspectSkillRequest) (dto.SkillDetail, error) {
	return handler.service.InspectSkill(ctx, request)
}

func (handler *SkillsHandler) GetSkillsStatus(ctx context.Context, request dto.SkillsStatusRequest) (dto.SkillsStatus, error) {
	return handler.service.GetSkillsStatus(ctx, request)
}

func (handler *SkillsHandler) InstallSkill(ctx context.Context, request dto.InstallSkillRequest) error {
	return handler.service.InstallSkill(ctx, request)
}

func (handler *SkillsHandler) UpdateSkill(ctx context.Context, request dto.UpdateSkillRequest) error {
	return handler.service.UpdateSkill(ctx, request)
}

func (handler *SkillsHandler) RemoveSkill(ctx context.Context, request dto.RemoveSkillRequest) error {
	if err := handler.service.RemoveSkill(ctx, request); err != nil {
		return err
	}
	_ = handler.service.DeleteSkill(ctx, dto.DeleteSkillRequest{ID: request.Skill})
	return nil
}

func (handler *SkillsHandler) SyncSkills(ctx context.Context, request dto.SyncSkillsRequest) ([]dto.ProviderSkillSpec, error) {
	return handler.service.SyncSkills(ctx, request)
}

func (handler *SkillsHandler) ResolveSkillsForProvider(ctx context.Context, request dto.ResolveSkillsRequest) ([]dto.ProviderSkillSpec, error) {
	return handler.service.ResolveSkillsForProvider(ctx, request)
}
