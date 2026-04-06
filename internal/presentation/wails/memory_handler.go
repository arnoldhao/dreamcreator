package wails

import (
	"context"

	"dreamcreator/internal/application/memory/dto"
	"dreamcreator/internal/application/memory/service"
)

type MemoryHandler struct {
	service *service.MemoryService
}

func NewMemoryHandler(service *service.MemoryService) *MemoryHandler {
	return &MemoryHandler{service: service}
}

func (handler *MemoryHandler) ServiceName() string {
	return "MemoryHandler"
}

func (handler *MemoryHandler) UpdateSTM(ctx context.Context, request dto.UpdateSTMRequest) (dto.STMState, error) {
	return handler.service.UpdateSTM(ctx, request)
}

func (handler *MemoryHandler) RetrieveForContext(ctx context.Context, request dto.RetrieveForContextRequest) (dto.MemoryRetrieval, error) {
	return handler.service.RetrieveForContext(ctx, request)
}

func (handler *MemoryHandler) ProposeWrites(ctx context.Context, request dto.ProposeWritesRequest) ([]dto.LTMEntry, error) {
	return handler.service.ProposeWrites(ctx, request)
}

func (handler *MemoryHandler) CommitWrites(ctx context.Context, request dto.CommitWritesRequest) error {
	return handler.service.CommitWrites(ctx, request)
}

func (handler *MemoryHandler) ImportDocs(ctx context.Context, request dto.ImportDocsRequest) error {
	return handler.service.ImportDocs(ctx, request)
}

func (handler *MemoryHandler) BuildIndex(ctx context.Context, request dto.BuildIndexRequest) error {
	return handler.service.BuildIndex(ctx, request)
}

func (handler *MemoryHandler) RetrieveRAG(ctx context.Context, request dto.RetrieveRAGRequest) ([]dto.LTMEntry, error) {
	return handler.service.RetrieveRAG(ctx, request)
}

func (handler *MemoryHandler) Embed(ctx context.Context, request dto.EmbedRequest) ([]float32, error) {
	return handler.service.Embed(ctx, request)
}

func (handler *MemoryHandler) Recall(ctx context.Context, request dto.MemoryRecallRequest) (dto.MemoryRetrieval, error) {
	return handler.service.Recall(ctx, request)
}

func (handler *MemoryHandler) Store(ctx context.Context, request dto.MemoryStoreRequest) (dto.LTMEntry, error) {
	return handler.service.Store(ctx, request)
}

func (handler *MemoryHandler) Forget(ctx context.Context, request dto.MemoryForgetRequest) (bool, error) {
	return handler.service.Forget(ctx, request)
}

func (handler *MemoryHandler) Update(ctx context.Context, request dto.MemoryUpdateRequest) (dto.LTMEntry, error) {
	return handler.service.Update(ctx, request)
}

func (handler *MemoryHandler) List(ctx context.Context, request dto.MemoryListRequest) ([]dto.LTMEntry, error) {
	return handler.service.List(ctx, request)
}

func (handler *MemoryHandler) Stats(ctx context.Context, request dto.MemoryStatsRequest) (dto.MemoryStats, error) {
	return handler.service.Stats(ctx, request)
}

func (handler *MemoryHandler) GetSummary(ctx context.Context, request dto.MemorySummaryRequest) (dto.MemorySummary, error) {
	return handler.service.GetSummary(ctx, request)
}

func (handler *MemoryHandler) BrowseOptions(ctx context.Context, request dto.MemoryBrowseOptionsRequest) (dto.MemoryBrowseOptions, error) {
	return handler.service.BrowseOptions(ctx, request)
}

func (handler *MemoryHandler) ListPrincipals(ctx context.Context, request dto.MemoryPrincipalListRequest) ([]dto.MemoryPrincipalItem, error) {
	return handler.service.ListPrincipals(ctx, request)
}

func (handler *MemoryHandler) RefreshPrincipal(
	ctx context.Context,
	request dto.MemoryPrincipalRefreshRequest,
) (dto.MemoryPrincipalRefreshResult, error) {
	return handler.service.RefreshPrincipal(ctx, request)
}
