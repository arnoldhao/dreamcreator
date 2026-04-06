package methods

import (
	"context"
	"encoding/json"

	"dreamcreator/internal/application/gateway/controlplane"
	gatewayruntime "dreamcreator/internal/application/gateway/runtime"
	runtimedto "dreamcreator/internal/application/gateway/runtime/dto"
)

func RegisterRuntime(router *controlplane.Router, runtime *gatewayruntime.Service) {
	if router == nil || runtime == nil {
		return
	}
	router.Register("runtime.run", []string{"runtime.run"}, func(ctx context.Context, _ *controlplane.SessionContext, params []byte) (any, *controlplane.GatewayError) {
		var request runtimedto.RuntimeRunRequest
		if len(params) > 0 {
			if err := json.Unmarshal(params, &request); err != nil {
				return nil, controlplane.NewGatewayError("invalid_request", err.Error())
			}
		}
		response, err := runtime.Start(ctx, request)
		if err != nil {
			return nil, controlplane.NewGatewayError("runtime_error", err.Error())
		}
		return response, nil
	})
	router.Register("runtime.abort", []string{"runtime.abort"}, func(ctx context.Context, _ *controlplane.SessionContext, params []byte) (any, *controlplane.GatewayError) {
		var request gatewayruntime.AbortRequest
		if len(params) > 0 {
			if err := json.Unmarshal(params, &request); err != nil {
				return nil, controlplane.NewGatewayError("invalid_request", err.Error())
			}
		}
		response, err := runtime.Abort(ctx, request)
		if err != nil {
			return nil, controlplane.NewGatewayError("runtime_error", err.Error())
		}
		return response, nil
	})
}
