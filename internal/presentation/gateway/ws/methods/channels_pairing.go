package methods

import (
	"context"
	"encoding/json"
	"strings"

	telegramservice "dreamcreator/internal/application/channels/telegram"
	"dreamcreator/internal/application/gateway/controlplane"
)

const (
	ScopeChannelsPairingList    = "channels.pairing.list"
	ScopeChannelsPairingApprove = "channels.pairing.approve"
	ScopeChannelsPairingReject  = "channels.pairing.reject"
)

type channelPairingListParams struct {
	ChannelID string `json:"channelId"`
	AccountID string `json:"accountId,omitempty"`
}

type channelPairingApproveParams struct {
	ChannelID string `json:"channelId"`
	Code      string `json:"code"`
	AccountID string `json:"accountId,omitempty"`
	Notify    bool   `json:"notify,omitempty"`
}

type channelPairingRejectParams struct {
	ChannelID string `json:"channelId"`
	Code      string `json:"code"`
	AccountID string `json:"accountId,omitempty"`
}

func RegisterChannelPairing(router *controlplane.Router, service *telegramservice.PairingService) {
	if router == nil || service == nil {
		return
	}
	router.Register("channels.pairing.list", []string{ScopeChannelsPairingList}, func(ctx context.Context, _ *controlplane.SessionContext, params []byte) (any, *controlplane.GatewayError) {
		var payload channelPairingListParams
		if err := json.Unmarshal(params, &payload); err != nil {
			return nil, controlplane.NewGatewayError("invalid_params", "invalid channels.pairing.list params")
		}
		if strings.TrimSpace(payload.ChannelID) != "telegram" {
			return nil, controlplane.NewGatewayError("invalid_request", "channel pairing not supported")
		}
		result, err := service.List(ctx, payload.AccountID)
		if err != nil {
			return nil, controlplane.NewGatewayError("internal_error", err.Error())
		}
		return result, nil
	})
	router.Register("channels.pairing.approve", []string{ScopeChannelsPairingApprove}, func(ctx context.Context, _ *controlplane.SessionContext, params []byte) (any, *controlplane.GatewayError) {
		var payload channelPairingApproveParams
		if err := json.Unmarshal(params, &payload); err != nil {
			return nil, controlplane.NewGatewayError("invalid_params", "invalid channels.pairing.approve params")
		}
		if strings.TrimSpace(payload.ChannelID) != "telegram" {
			return nil, controlplane.NewGatewayError("invalid_request", "channel pairing not supported")
		}
		if strings.TrimSpace(payload.Code) == "" {
			return nil, controlplane.NewGatewayError("invalid_params", "pairing code required")
		}
		result, err := service.Approve(ctx, payload.Code, payload.AccountID, payload.Notify)
		if err != nil {
			return nil, controlplane.NewGatewayError("internal_error", err.Error())
		}
		return result, nil
	})
	router.Register("channels.pairing.reject", []string{ScopeChannelsPairingReject}, func(ctx context.Context, _ *controlplane.SessionContext, params []byte) (any, *controlplane.GatewayError) {
		var payload channelPairingRejectParams
		if err := json.Unmarshal(params, &payload); err != nil {
			return nil, controlplane.NewGatewayError("invalid_params", "invalid channels.pairing.reject params")
		}
		if strings.TrimSpace(payload.ChannelID) != "telegram" {
			return nil, controlplane.NewGatewayError("invalid_request", "channel pairing not supported")
		}
		if strings.TrimSpace(payload.Code) == "" {
			return nil, controlplane.NewGatewayError("invalid_params", "pairing code required")
		}
		result, err := service.Reject(ctx, payload.Code, payload.AccountID)
		if err != nil {
			return nil, controlplane.NewGatewayError("internal_error", err.Error())
		}
		return result, nil
	})
}
