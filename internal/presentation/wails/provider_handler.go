package wails

import (
	"context"
	"strings"

	gatewayusage "dreamcreator/internal/application/gateway/usage"
	"dreamcreator/internal/application/providers/dto"
	"dreamcreator/internal/application/providers/service"
	"dreamcreator/internal/domain/providers"
)

type ProviderHandler struct {
	service            *service.ProvidersService
	onProvidersUpdated ProvidersUpdatedNotifier
	usageService       *gatewayusage.Service
	pricingSyncer      ProviderPricingSyncer
	telemetry          providerTelemetry
}

type ProvidersUpdatedNotifier interface {
	ProvidersUpdated()
}

type ProviderPricingSyncer interface {
	Sync(ctx context.Context, provider providers.Provider, apiKey string) ([]providers.Model, error)
}

type providerTelemetry interface {
	TrackProviderConfigured(ctx context.Context, providerID string)
}

func NewProviderHandler(service *service.ProvidersService, notifier ProvidersUpdatedNotifier, usageService *gatewayusage.Service, syncer ProviderPricingSyncer, telemetry providerTelemetry) *ProviderHandler {
	return &ProviderHandler{
		service:            service,
		onProvidersUpdated: notifier,
		usageService:       usageService,
		pricingSyncer:      syncer,
		telemetry:          telemetry,
	}
}

func (handler *ProviderHandler) ServiceName() string {
	return "ProviderHandler"
}

func (handler *ProviderHandler) ListProviders(ctx context.Context) ([]dto.Provider, error) {
	return handler.service.ListProviders(ctx)
}

func (handler *ProviderHandler) ListEnabledProvidersWithModels(ctx context.Context) ([]dto.ProviderWithModels, error) {
	return handler.service.ListEnabledProvidersWithModels(ctx)
}

func (handler *ProviderHandler) UpsertProvider(ctx context.Context, request dto.UpsertProviderRequest) (dto.Provider, error) {
	provider, err := handler.service.UpsertProvider(ctx, request)
	if err != nil {
		return dto.Provider{}, err
	}
	handler.notifyProvidersUpdated()
	return provider, nil
}

func (handler *ProviderHandler) DeleteProvider(ctx context.Context, id string) error {
	if err := handler.service.DeleteProvider(ctx, id); err != nil {
		return err
	}
	handler.notifyProvidersUpdated()
	return nil
}

func (handler *ProviderHandler) ListProviderModels(ctx context.Context, providerID string) ([]dto.ProviderModel, error) {
	return handler.service.ListProviderModels(ctx, providerID)
}

func (handler *ProviderHandler) UpdateProviderModel(ctx context.Context, request dto.UpdateProviderModelRequest) (dto.ProviderModel, error) {
	model, err := handler.service.UpdateProviderModel(ctx, request)
	if err != nil {
		return dto.ProviderModel{}, err
	}
	handler.notifyProvidersUpdated()
	return model, nil
}

func (handler *ProviderHandler) ReplaceProviderModels(ctx context.Context, request dto.ReplaceProviderModelsRequest) error {
	if err := handler.service.ReplaceProviderModels(ctx, request); err != nil {
		return err
	}
	handler.syncPricingFromModelsDev(ctx, request.ProviderID)
	handler.notifyProvidersUpdated()
	return nil
}

func (handler *ProviderHandler) SyncProviderModels(ctx context.Context, request dto.SyncProviderModelsRequest) ([]dto.ProviderModel, error) {
	models, err := handler.service.SyncProviderModels(ctx, request.ProviderID, request.APIKey)
	if err != nil {
		return nil, err
	}
	handler.syncPricingFromModelsDev(ctx, request.ProviderID)
	handler.notifyProvidersUpdated()
	return models, nil
}

func (handler *ProviderHandler) GetProviderSecret(ctx context.Context, providerID string) (dto.ProviderSecret, error) {
	return handler.service.GetProviderSecret(ctx, providerID)
}

func (handler *ProviderHandler) UpsertProviderSecret(ctx context.Context, request dto.UpsertProviderSecretRequest) error {
	if err := handler.service.UpsertProviderSecret(ctx, request); err != nil {
		return err
	}
	if handler.telemetry != nil && (strings.TrimSpace(request.APIKey) != "" || strings.TrimSpace(request.OrgRef) != "") {
		handler.telemetry.TrackProviderConfigured(ctx, request.ProviderID)
	}
	return nil
}

func (handler *ProviderHandler) notifyProvidersUpdated() {
	if handler == nil || handler.onProvidersUpdated == nil {
		return
	}
	handler.onProvidersUpdated.ProvidersUpdated()
}
