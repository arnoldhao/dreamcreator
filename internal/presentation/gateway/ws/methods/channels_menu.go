package methods

import (
	"context"
	"encoding/json"
	"strings"

	telegrammenu "dreamcreator/internal/application/channels/telegram"
	"dreamcreator/internal/application/gateway/controlplane"
)

const (
	ScopeChannelsMenuSync = "channels.menu.sync"
)

type channelMenuSyncParams struct {
	ChannelID string `json:"channelId"`
}

func RegisterChannelMenus(router *controlplane.Router, menuService *telegrammenu.MenuService) {
	if router == nil || menuService == nil {
		return
	}
	router.Register("channels.menu.sync", []string{ScopeChannelsMenuSync}, func(ctx context.Context, _ *controlplane.SessionContext, params []byte) (any, *controlplane.GatewayError) {
		var payload channelMenuSyncParams
		if err := json.Unmarshal(params, &payload); err != nil {
			return nil, controlplane.NewGatewayError("invalid_params", "invalid channels.menu.sync params")
		}
		if strings.TrimSpace(payload.ChannelID) != "telegram" {
			return nil, controlplane.NewGatewayError("invalid_request", "channel menu sync not supported")
		}
		result, err := menuService.Sync(ctx)
		if err != nil {
			return nil, controlplane.NewGatewayError("invalid_request", err.Error())
		}
		return result, nil
	})
}
