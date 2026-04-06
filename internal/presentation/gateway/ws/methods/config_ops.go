package methods

import (
	"context"
	"encoding/json"
	"errors"

	"dreamcreator/internal/application/gateway/config"
	"dreamcreator/internal/application/gateway/controlplane"
)

const (
	ScopeConfigGet    = "config.get"
	ScopeConfigSet    = "config.set"
	ScopeConfigPatch  = "config.patch"
	ScopeConfigApply  = "config.apply"
	ScopeConfigSchema = "config.schema"
)

func RegisterConfig(router *controlplane.Router, cfgService *config.Service) {
	if router == nil || cfgService == nil {
		return
	}
	router.Register("config.get", []string{ScopeConfigGet}, func(ctx context.Context, _ *controlplane.SessionContext, params []byte) (any, *controlplane.GatewayError) {
		var payload config.ConfigGetRequest
		if len(params) > 0 {
			if err := json.Unmarshal(params, &payload); err != nil {
				return nil, controlplane.NewGatewayError("invalid_params", "invalid config.get params")
			}
		}
		resp, err := cfgService.Get(ctx, payload)
		if err != nil {
			return nil, controlplane.NewGatewayError("invalid_request", err.Error())
		}
		return resp, nil
	})
	router.Register("config.set", []string{ScopeConfigSet}, func(ctx context.Context, _ *controlplane.SessionContext, params []byte) (any, *controlplane.GatewayError) {
		var payload config.ConfigSetRequest
		if err := json.Unmarshal(params, &payload); err != nil {
			return nil, controlplane.NewGatewayError("invalid_params", "invalid config.set params")
		}
		resp, err := cfgService.Set(ctx, payload)
		if err != nil {
			code := "invalid_request"
			if errors.Is(err, config.ErrVersionConflict) {
				code = "version_conflict"
			}
			return nil, controlplane.NewGatewayError(code, err.Error())
		}
		return resp, nil
	})
	router.Register("config.patch", []string{ScopeConfigPatch}, func(ctx context.Context, _ *controlplane.SessionContext, params []byte) (any, *controlplane.GatewayError) {
		var payload config.ConfigPatchRequest
		if err := json.Unmarshal(params, &payload); err != nil {
			return nil, controlplane.NewGatewayError("invalid_params", "invalid config.patch params")
		}
		resp, err := cfgService.Patch(ctx, payload)
		if err != nil {
			code := "invalid_request"
			if errors.Is(err, config.ErrVersionConflict) {
				code = "version_conflict"
			}
			return nil, controlplane.NewGatewayError(code, err.Error())
		}
		return resp, nil
	})
	router.Register("config.apply", []string{ScopeConfigApply}, func(ctx context.Context, _ *controlplane.SessionContext, params []byte) (any, *controlplane.GatewayError) {
		var payload config.ConfigApplyRequest
		if err := json.Unmarshal(params, &payload); err != nil {
			return nil, controlplane.NewGatewayError("invalid_params", "invalid config.apply params")
		}
		resp, err := cfgService.Apply(ctx, payload)
		if err != nil {
			code := "invalid_request"
			if errors.Is(err, config.ErrVersionConflict) {
				code = "version_conflict"
			}
			return nil, controlplane.NewGatewayError(code, err.Error())
		}
		return resp, nil
	})
	router.Register("config.schema", []string{ScopeConfigSchema}, func(ctx context.Context, _ *controlplane.SessionContext, params []byte) (any, *controlplane.GatewayError) {
		var payload config.ConfigSchemaRequest
		if len(params) > 0 {
			if err := json.Unmarshal(params, &payload); err != nil {
				return nil, controlplane.NewGatewayError("invalid_params", "invalid config.schema params")
			}
		}
		resp, err := cfgService.Schema(ctx, payload)
		if err != nil {
			return nil, controlplane.NewGatewayError("invalid_request", err.Error())
		}
		return resp, nil
	})
}
