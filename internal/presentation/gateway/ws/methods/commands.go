package methods

import (
	"context"
	"encoding/json"

	appcommands "dreamcreator/internal/application/commands"
	gatewaycommands "dreamcreator/internal/application/gateway/commands"
	"dreamcreator/internal/application/gateway/controlplane"
)

const (
	ScopeCommandsList = "commands.list"
	ScopeCommandsRun  = "commands.run"
)

type commandsListParams struct {
	Provider        string `json:"provider,omitempty"`
	IncludeDisabled bool   `json:"includeDisabled,omitempty"`
}

func RegisterCommands(router *controlplane.Router, service *gatewaycommands.Service) {
	if router == nil || service == nil {
		return
	}
	router.Register("commands.list", []string{ScopeCommandsList}, func(ctx context.Context, _ *controlplane.SessionContext, params []byte) (any, *controlplane.GatewayError) {
		payload := commandsListParams{}
		if len(params) > 0 {
			if err := json.Unmarshal(params, &payload); err != nil {
				return nil, controlplane.NewGatewayError("invalid_params", "invalid commands.list params")
			}
		}
		response, err := service.List(ctx, gatewaycommands.ListRequest{
			Provider:        payload.Provider,
			IncludeDisabled: payload.IncludeDisabled,
		})
		if err != nil {
			return nil, controlplane.NewGatewayError("internal_error", err.Error())
		}
		return response, nil
	})
	router.Register("commands.run", []string{ScopeCommandsRun}, func(ctx context.Context, _ *controlplane.SessionContext, params []byte) (any, *controlplane.GatewayError) {
		var payload appcommands.CommandRunRequest
		if err := json.Unmarshal(params, &payload); err != nil {
			return nil, controlplane.NewGatewayError("invalid_params", "invalid commands.run params")
		}
		response, err := service.Run(ctx, payload)
		if err != nil {
			return nil, controlplane.NewGatewayError("invalid_request", err.Error())
		}
		return response, nil
	})
}
