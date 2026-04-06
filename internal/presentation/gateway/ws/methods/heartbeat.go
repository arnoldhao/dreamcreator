package methods

import (
	"context"
	"encoding/json"
	"strings"

	"dreamcreator/internal/application/gateway/controlplane"
	gatewayheartbeat "dreamcreator/internal/application/gateway/heartbeat"
)

const (
	ScopeHeartbeatRead        = "heartbeat.read"
	ScopeHeartbeatTrigger     = "heartbeat.trigger"
	ScopeHeartbeatToggle      = "heartbeat.toggle"
	ScopeHeartbeatSystemEvent = "heartbeat.system_event"
)

type heartbeatLastParams struct {
	SessionKey string `json:"sessionKey"`
}

type heartbeatTriggerParams struct {
	Reason     string `json:"reason,omitempty"`
	SessionKey string `json:"sessionKey,omitempty"`
	Force      bool   `json:"force,omitempty"`
}

type heartbeatToggleParams struct {
	Enabled bool `json:"enabled"`
}

type heartbeatSystemEventParams struct {
	SessionKey string `json:"sessionKey"`
	Text       string `json:"text"`
	ContextKey string `json:"contextKey,omitempty"`
	RunID      string `json:"runId,omitempty"`
	Source     string `json:"source,omitempty"`
}

func RegisterHeartbeat(router *controlplane.Router, service *gatewayheartbeat.Service) {
	if router == nil || service == nil {
		return
	}
	router.Register("heartbeat.last", []string{ScopeHeartbeatRead}, func(ctx context.Context, _ *controlplane.SessionContext, params []byte) (any, *controlplane.GatewayError) {
		var payload heartbeatLastParams
		if err := json.Unmarshal(params, &payload); err != nil {
			return nil, controlplane.NewGatewayError("invalid_params", "invalid heartbeat.last params")
		}
		sessionKey := strings.TrimSpace(payload.SessionKey)
		if sessionKey == "" {
			return nil, controlplane.NewGatewayError("invalid_params", "sessionKey is required")
		}
		event, err := service.Last(ctx, sessionKey)
		if err != nil {
			return nil, controlplane.NewGatewayError("not_found", err.Error())
		}
		return map[string]any{"event": event}, nil
	})
	router.Register("heartbeat.trigger", []string{ScopeHeartbeatTrigger}, func(ctx context.Context, _ *controlplane.SessionContext, params []byte) (any, *controlplane.GatewayError) {
		var payload heartbeatTriggerParams
		if len(params) > 0 {
			if err := json.Unmarshal(params, &payload); err != nil {
				return nil, controlplane.NewGatewayError("invalid_params", "invalid heartbeat.trigger params")
			}
		}
		result := service.TriggerWithResult(ctx, gatewayheartbeat.TriggerInput{
			Reason:     strings.TrimSpace(payload.Reason),
			SessionKey: strings.TrimSpace(payload.SessionKey),
			Force:      payload.Force,
		})
		return map[string]any{
			"ok":             true,
			"accepted":       result.Accepted,
			"executedStatus": result.ExecutedStatus,
			"reason":         result.Reason,
		}, nil
	})
	router.Register("heartbeat.setEnabled", []string{ScopeHeartbeatToggle}, func(_ context.Context, _ *controlplane.SessionContext, params []byte) (any, *controlplane.GatewayError) {
		var payload heartbeatToggleParams
		if err := json.Unmarshal(params, &payload); err != nil {
			return nil, controlplane.NewGatewayError("invalid_params", "invalid heartbeat.setEnabled params")
		}
		service.SetEnabled(payload.Enabled)
		return map[string]any{"ok": true}, nil
	})
	router.Register("heartbeat.systemEvent", []string{ScopeHeartbeatSystemEvent}, func(ctx context.Context, _ *controlplane.SessionContext, params []byte) (any, *controlplane.GatewayError) {
		var payload heartbeatSystemEventParams
		if err := json.Unmarshal(params, &payload); err != nil {
			return nil, controlplane.NewGatewayError("invalid_params", "invalid heartbeat.systemEvent params")
		}
		if strings.TrimSpace(payload.SessionKey) == "" || strings.TrimSpace(payload.Text) == "" {
			return nil, controlplane.NewGatewayError("invalid_params", "sessionKey and text are required")
		}
		queued := service.EnqueueSystemEvent(ctx, gatewayheartbeat.SystemEventInput{
			SessionKey: strings.TrimSpace(payload.SessionKey),
			Text:       strings.TrimSpace(payload.Text),
			ContextKey: strings.TrimSpace(payload.ContextKey),
			RunID:      strings.TrimSpace(payload.RunID),
			Source:     strings.TrimSpace(payload.Source),
		})
		return map[string]any{"ok": true, "queued": queued}, nil
	})
}
