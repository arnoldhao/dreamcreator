package wails

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	gatewayusage "dreamcreator/internal/application/gateway/usage"
	"dreamcreator/internal/domain/providers"

	"go.uber.org/zap"
)

const (
	modelsDevPricingSource    = "models.dev"
	modelsDevPricingUpdatedBy = "system:models.dev.sync"
)

type modelsDevProviderResolver interface {
	ResolveProviderKeyByModelIDs(ctx context.Context, modelIDs []string) (string, error)
	SyncByProviderKey(ctx context.Context, providerKey string) ([]providers.Model, error)
}

func (handler *ProviderHandler) syncPricingFromModelsDev(ctx context.Context, providerID string) {
	if handler == nil || handler.pricingSyncer == nil || handler.usageService == nil {
		return
	}
	providerID = strings.TrimSpace(providerID)
	if providerID == "" {
		return
	}

	providerDTO, err := handler.service.GetProvider(ctx, providerID)
	if err != nil {
		zap.L().Warn("sync models.dev pricing skipped: provider lookup failed",
			zap.String("provider_id", providerID),
			zap.Error(err),
		)
		return
	}

	provider := providers.Provider{
		ID:       strings.TrimSpace(providerDTO.ID),
		Name:     strings.TrimSpace(providerDTO.Name),
		Type:     providers.ProviderType(strings.ToLower(strings.TrimSpace(providerDTO.Type))),
		Endpoint: strings.TrimSpace(providerDTO.Endpoint),
		Enabled:  providerDTO.Enabled,
		Builtin:  providerDTO.Builtin,
	}
	models, err := handler.pricingSyncer.Sync(ctx, provider, "")
	if err != nil {
		models, err = handler.syncPricingByResolvedProviderKey(ctx, provider.ID)
		if err == nil {
			goto pricingSync
		}
		zap.L().Warn("sync models.dev pricing skipped: catalog fetch failed",
			zap.String("provider_id", providerID),
			zap.Error(err),
		)
		return
	}
	if len(models) == 0 {
		models, err = handler.syncPricingByResolvedProviderKey(ctx, provider.ID)
		if err != nil {
			zap.L().Warn("sync models.dev pricing skipped: resolved provider key fetch failed",
				zap.String("provider_id", providerID),
				zap.Error(err),
			)
			return
		}
	}

pricingSync:
	if len(models) == 0 {
		return
	}

	existing, err := handler.usageService.PricingList(ctx, gatewayusage.PricingListRequest{
		ProviderID: provider.ID,
	})
	if err != nil {
		zap.L().Warn("sync models.dev pricing skipped: pricing list failed",
			zap.String("provider_id", providerID),
			zap.Error(err),
		)
		return
	}

	protectedActiveModels := make(map[string]struct{})
	latestModelsDevByModel := make(map[string]gatewayusage.PricingVersion)
	for _, item := range existing.Items {
		key := normalizePricingModelKey(item.ModelName)
		if key == "" {
			continue
		}
		source := strings.ToLower(strings.TrimSpace(item.Source))
		if item.IsActive && source != modelsDevPricingSource {
			protectedActiveModels[key] = struct{}{}
		}
		if source != modelsDevPricingSource {
			continue
		}
		if _, exists := latestModelsDevByModel[key]; !exists {
			latestModelsDevByModel[key] = item
		}
	}

	nowRFC3339 := time.Now().UTC().Format(time.RFC3339)
	failedCount := 0
	for _, model := range models {
		modelName := strings.TrimSpace(model.Name)
		if modelName == "" {
			continue
		}
		parsed, ok := gatewayusage.ParsePricingFromCapabilities(model.CapabilitiesJSON)
		if !ok {
			continue
		}
		modelKey := normalizePricingModelKey(modelName)
		_, hasProtected := protectedActiveModels[modelKey]
		shouldActivate := !hasProtected

		if current, exists := latestModelsDevByModel[modelKey]; exists {
			if pricingVersionMatchesParsed(current, parsed) {
				if current.IsActive == shouldActivate {
					continue
				}
				if shouldActivate {
					if err := handler.usageService.PricingActivate(ctx, gatewayusage.PricingActivateRequest{ID: current.ID}); err != nil {
						failedCount++
						zap.L().Warn("activate models.dev pricing failed",
							zap.String("provider_id", provider.ID),
							zap.String("model_name", modelName),
							zap.String("pricing_id", current.ID),
							zap.Error(err),
						)
					}
					continue
				}
				if err := handler.deactivatePricingVersion(ctx, current); err != nil {
					failedCount++
					zap.L().Warn("deactivate models.dev pricing failed",
						zap.String("provider_id", provider.ID),
						zap.String("model_name", modelName),
						zap.String("pricing_id", current.ID),
						zap.Error(err),
					)
				}
				continue
			}
		}

		if _, err := handler.usageService.PricingUpsert(ctx, gatewayusage.PricingUpsertRequest{
			ProviderID:            provider.ID,
			ModelName:             modelName,
			Currency:              "USD",
			InputPerMillion:       parsed.InputPerMillion,
			OutputPerMillion:      parsed.OutputPerMillion,
			CachedInputPerMillion: parsed.CachedInputPerMillion,
			ReasoningPerMillion:   parsed.ReasoningPerMillion,
			AudioInputPerMillion:  parsed.AudioInputPerMillion,
			AudioOutputPerMillion: parsed.AudioOutputPerMillion,
			PerRequest:            parsed.PerRequest,
			Source:                modelsDevPricingSource,
			EffectiveFrom:         nowRFC3339,
			IsActive:              shouldActivate,
			UpdatedBy:             modelsDevPricingUpdatedBy,
		}); err != nil {
			failedCount++
			zap.L().Warn("upsert models.dev pricing failed",
				zap.String("provider_id", provider.ID),
				zap.String("model_name", modelName),
				zap.Error(err),
			)
		}
	}

	if failedCount > 0 {
		zap.L().Warn("models.dev pricing sync completed with failures",
			zap.String("provider_id", provider.ID),
			zap.Int("failures", failedCount),
		)
	}
}

func (handler *ProviderHandler) syncPricingByResolvedProviderKey(ctx context.Context, providerID string) ([]providers.Model, error) {
	resolver, ok := handler.pricingSyncer.(modelsDevProviderResolver)
	if !ok {
		return nil, fmt.Errorf("pricing syncer does not support provider key resolution")
	}
	localModels, err := handler.service.ListProviderModels(ctx, providerID)
	if err != nil {
		return nil, err
	}
	modelNames := make([]string, 0, len(localModels))
	for _, item := range localModels {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			continue
		}
		modelNames = append(modelNames, name)
	}
	if len(modelNames) == 0 {
		return nil, fmt.Errorf("provider has no local models to resolve provider key")
	}
	providerKey, err := resolver.ResolveProviderKeyByModelIDs(ctx, modelNames)
	if err != nil {
		return nil, err
	}
	return resolver.SyncByProviderKey(ctx, providerKey)
}

func (handler *ProviderHandler) deactivatePricingVersion(ctx context.Context, version gatewayusage.PricingVersion) error {
	request := gatewayusage.PricingUpsertRequest{
		ID:                    strings.TrimSpace(version.ID),
		ProviderID:            strings.TrimSpace(version.ProviderID),
		ModelName:             strings.TrimSpace(version.ModelName),
		Currency:              normalizePricingCurrency(version.Currency),
		InputPerMillion:       version.InputPerMillion,
		OutputPerMillion:      version.OutputPerMillion,
		CachedInputPerMillion: version.CachedInputPerMillion,
		ReasoningPerMillion:   version.ReasoningPerMillion,
		AudioInputPerMillion:  version.AudioInputPerMillion,
		AudioOutputPerMillion: version.AudioOutputPerMillion,
		PerRequest:            version.PerRequest,
		Source:                strings.TrimSpace(version.Source),
		EffectiveFrom:         version.EffectiveFrom.UTC().Format(time.RFC3339),
		IsActive:              false,
		UpdatedBy:             modelsDevPricingUpdatedBy,
	}
	if version.EffectiveTo != nil {
		request.EffectiveTo = version.EffectiveTo.UTC().Format(time.RFC3339)
	}
	_, err := handler.usageService.PricingUpsert(ctx, request)
	return err
}

func normalizePricingModelKey(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func normalizePricingCurrency(value string) string {
	trimmed := strings.ToUpper(strings.TrimSpace(value))
	if trimmed == "" {
		return "USD"
	}
	return trimmed
}

func pricingVersionMatchesParsed(version gatewayusage.PricingVersion, parsed gatewayusage.ParsedPricing) bool {
	return pricingFloatEqual(version.InputPerMillion, parsed.InputPerMillion) &&
		pricingFloatEqual(version.OutputPerMillion, parsed.OutputPerMillion) &&
		pricingFloatEqual(version.CachedInputPerMillion, parsed.CachedInputPerMillion) &&
		pricingFloatEqual(version.ReasoningPerMillion, parsed.ReasoningPerMillion) &&
		pricingFloatEqual(version.AudioInputPerMillion, parsed.AudioInputPerMillion) &&
		pricingFloatEqual(version.AudioOutputPerMillion, parsed.AudioOutputPerMillion) &&
		pricingFloatEqual(version.PerRequest, parsed.PerRequest)
}

func pricingFloatEqual(left float64, right float64) bool {
	return math.Abs(left-right) <= 1e-9
}
