package methods

import (
	"context"
	"encoding/json"

	"dreamcreator/internal/application/gateway/channels"
	"dreamcreator/internal/application/gateway/controlplane"
)

const (
	ScopeChannelsList   = "channels.list"
	ScopeChannelsStatus = "channels.status"
	ScopeChannelsLogout = "channels.logout"
	ScopeChannelsProbe  = "channels.probe"
)

type channelTargetParams struct {
	ChannelID string `json:"channelId"`
}

func RegisterChannels(router *controlplane.Router, registry *channels.Registry) {
	if router == nil || registry == nil {
		return
	}
	router.Register("channels.list", []string{ScopeChannelsList}, func(ctx context.Context, _ *controlplane.SessionContext, _ []byte) (any, *controlplane.GatewayError) {
		entries, err := registry.List(ctx)
		if err != nil {
			return nil, controlplane.NewGatewayError("internal_error", err.Error())
		}
		return entries, nil
	})
	router.Register("channels.status", []string{ScopeChannelsStatus}, func(ctx context.Context, _ *controlplane.SessionContext, _ []byte) (any, *controlplane.GatewayError) {
		statuses, err := registry.StatusAll(ctx)
		if err != nil {
			return nil, controlplane.NewGatewayError("internal_error", err.Error())
		}
		return statuses, nil
	})
	router.Register("channels.logout", []string{ScopeChannelsLogout}, func(ctx context.Context, _ *controlplane.SessionContext, params []byte) (any, *controlplane.GatewayError) {
		var payload channelTargetParams
		if err := json.Unmarshal(params, &payload); err != nil {
			return nil, controlplane.NewGatewayError("invalid_params", "invalid channels.logout params")
		}
		result, err := registry.Logout(ctx, payload.ChannelID)
		if err != nil {
			return nil, controlplane.NewGatewayError("invalid_request", err.Error())
		}
		return result, nil
	})
	router.Register("channels.probe", []string{ScopeChannelsProbe}, func(ctx context.Context, _ *controlplane.SessionContext, params []byte) (any, *controlplane.GatewayError) {
		var payload channelTargetParams
		if err := json.Unmarshal(params, &payload); err != nil {
			return nil, controlplane.NewGatewayError("invalid_params", "invalid channels.probe params")
		}
		result, err := registry.Probe(ctx, payload.ChannelID)
		if err != nil {
			return nil, controlplane.NewGatewayError("invalid_request", err.Error())
		}
		return result, nil
	})
}
