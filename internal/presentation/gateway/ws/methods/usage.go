package methods

import (
	"context"
	"encoding/json"

	"dreamcreator/internal/application/gateway/controlplane"
	gatewayusage "dreamcreator/internal/application/gateway/usage"
)

const (
	ScopeUsageStatus          = "usage.status"
	ScopeUsageCost            = "usage.cost"
	ScopeUsagePricingList     = "usage.pricing.list"
	ScopeUsagePricingUpsert   = "usage.pricing.upsert"
	ScopeUsagePricingDelete   = "usage.pricing.delete"
	ScopeUsagePricingActivate = "usage.pricing.activate"
)

func RegisterUsage(router *controlplane.Router, usageService *gatewayusage.Service) {
	if router == nil || usageService == nil {
		return
	}
	router.Register("usage.status", []string{ScopeUsageStatus}, func(ctx context.Context, _ *controlplane.SessionContext, params []byte) (any, *controlplane.GatewayError) {
		var payload gatewayusage.UsageStatusRequest
		if len(params) > 0 {
			if err := json.Unmarshal(params, &payload); err != nil {
				return nil, controlplane.NewGatewayError("invalid_params", "invalid usage.status params")
			}
		}
		resp, err := usageService.Status(ctx, payload)
		if err != nil {
			return nil, controlplane.NewGatewayError("invalid_request", err.Error())
		}
		return resp, nil
	})
	router.Register("usage.cost", []string{ScopeUsageCost}, func(ctx context.Context, _ *controlplane.SessionContext, params []byte) (any, *controlplane.GatewayError) {
		var payload gatewayusage.UsageCostRequest
		if len(params) > 0 {
			if err := json.Unmarshal(params, &payload); err != nil {
				return nil, controlplane.NewGatewayError("invalid_params", "invalid usage.cost params")
			}
		}
		resp, err := usageService.Cost(ctx, payload)
		if err != nil {
			return nil, controlplane.NewGatewayError("invalid_request", err.Error())
		}
		return resp, nil
	})
	router.Register("usage.pricing.list", []string{ScopeUsagePricingList}, func(ctx context.Context, _ *controlplane.SessionContext, params []byte) (any, *controlplane.GatewayError) {
		var payload gatewayusage.PricingListRequest
		if len(params) > 0 {
			if err := json.Unmarshal(params, &payload); err != nil {
				return nil, controlplane.NewGatewayError("invalid_params", "invalid usage.pricing.list params")
			}
		}
		resp, err := usageService.PricingList(ctx, payload)
		if err != nil {
			return nil, controlplane.NewGatewayError("invalid_request", err.Error())
		}
		return resp, nil
	})
	router.Register("usage.pricing.upsert", []string{ScopeUsagePricingUpsert}, func(ctx context.Context, _ *controlplane.SessionContext, params []byte) (any, *controlplane.GatewayError) {
		var payload gatewayusage.PricingUpsertRequest
		if len(params) == 0 {
			return nil, controlplane.NewGatewayError("invalid_params", "usage.pricing.upsert params required")
		}
		if err := json.Unmarshal(params, &payload); err != nil {
			return nil, controlplane.NewGatewayError("invalid_params", "invalid usage.pricing.upsert params")
		}
		resp, err := usageService.PricingUpsert(ctx, payload)
		if err != nil {
			return nil, controlplane.NewGatewayError("invalid_request", err.Error())
		}
		return resp, nil
	})
	router.Register("usage.pricing.delete", []string{ScopeUsagePricingDelete}, func(ctx context.Context, _ *controlplane.SessionContext, params []byte) (any, *controlplane.GatewayError) {
		var payload gatewayusage.PricingDeleteRequest
		if len(params) == 0 {
			return nil, controlplane.NewGatewayError("invalid_params", "usage.pricing.delete params required")
		}
		if err := json.Unmarshal(params, &payload); err != nil {
			return nil, controlplane.NewGatewayError("invalid_params", "invalid usage.pricing.delete params")
		}
		if err := usageService.PricingDelete(ctx, payload); err != nil {
			return nil, controlplane.NewGatewayError("invalid_request", err.Error())
		}
		return map[string]any{"ok": true}, nil
	})
	router.Register("usage.pricing.activate", []string{ScopeUsagePricingActivate}, func(ctx context.Context, _ *controlplane.SessionContext, params []byte) (any, *controlplane.GatewayError) {
		var payload gatewayusage.PricingActivateRequest
		if len(params) == 0 {
			return nil, controlplane.NewGatewayError("invalid_params", "usage.pricing.activate params required")
		}
		if err := json.Unmarshal(params, &payload); err != nil {
			return nil, controlplane.NewGatewayError("invalid_params", "invalid usage.pricing.activate params")
		}
		if err := usageService.PricingActivate(ctx, payload); err != nil {
			return nil, controlplane.NewGatewayError("invalid_request", err.Error())
		}
		return map[string]any{"ok": true}, nil
	})
}
