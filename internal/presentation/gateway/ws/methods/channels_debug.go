package methods

import (
	"context"

	"dreamcreator/internal/application/gateway/channels"
	"dreamcreator/internal/application/gateway/controlplane"
)

const ScopeChannelsDebug = "channels.debug"

func RegisterChannelsDebug(router *controlplane.Router, service *channels.DebugService) {
	if router == nil || service == nil {
		return
	}
	router.Register("channels.debug", []string{ScopeChannelsDebug}, func(ctx context.Context, _ *controlplane.SessionContext, _ []byte) (any, *controlplane.GatewayError) {
		result, err := service.List(ctx)
		if err != nil {
			return nil, controlplane.NewGatewayError("internal_error", err.Error())
		}
		return result, nil
	})
}
