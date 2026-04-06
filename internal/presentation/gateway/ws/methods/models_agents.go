package methods

import (
	"context"
	"encoding/json"

	agentdto "dreamcreator/internal/application/agent/dto"
	agentservice "dreamcreator/internal/application/agent/service"
	"dreamcreator/internal/application/gateway/controlplane"
	gatewaymodels "dreamcreator/internal/application/gateway/models"
)

const (
	ScopeModelsList      = "models.list"
	ScopeAgentsList      = "agents.list"
	ScopeAgentsCreate    = "agents.create"
	ScopeAgentsUpdate    = "agents.update"
	ScopeAgentsDelete    = "agents.delete"
	ScopeAgentsFilesList = "agents.files.list"
	ScopeAgentsFilesGet  = "agents.files.get"
	ScopeAgentsFilesSet  = "agents.files.set"
)

type modelsListParams struct {
	IncludeDisabled bool `json:"includeDisabled,omitempty"`
}

type agentsListParams struct {
	IncludeDisabled bool `json:"includeDisabled,omitempty"`
}

func RegisterModels(router *controlplane.Router, service *gatewaymodels.Service) {
	if router == nil || service == nil {
		return
	}
	router.Register("models.list", []string{ScopeModelsList}, func(ctx context.Context, _ *controlplane.SessionContext, params []byte) (any, *controlplane.GatewayError) {
		payload := modelsListParams{}
		if len(params) > 0 {
			if err := json.Unmarshal(params, &payload); err != nil {
				return nil, controlplane.NewGatewayError("invalid_params", "invalid models.list params")
			}
		}
		response, err := service.List(ctx, payload.IncludeDisabled)
		if err != nil {
			return nil, controlplane.NewGatewayError("internal_error", err.Error())
		}
		return response, nil
	})
}

func RegisterAgents(router *controlplane.Router, service *agentservice.AgentService) {
	if router == nil || service == nil {
		return
	}
	router.Register("agents.list", []string{ScopeAgentsList}, func(ctx context.Context, _ *controlplane.SessionContext, params []byte) (any, *controlplane.GatewayError) {
		payload := agentsListParams{}
		if len(params) > 0 {
			if err := json.Unmarshal(params, &payload); err != nil {
				return nil, controlplane.NewGatewayError("invalid_params", "invalid agents.list params")
			}
		}
		items, err := service.ListAgents(ctx, payload.IncludeDisabled)
		if err != nil {
			return nil, controlplane.NewGatewayError("internal_error", err.Error())
		}
		return items, nil
	})
	router.Register("agents.create", []string{ScopeAgentsCreate}, func(ctx context.Context, _ *controlplane.SessionContext, params []byte) (any, *controlplane.GatewayError) {
		var payload agentdto.CreateAgentRequest
		if err := json.Unmarshal(params, &payload); err != nil {
			return nil, controlplane.NewGatewayError("invalid_params", "invalid agents.create params")
		}
		item, err := service.CreateAgent(ctx, payload)
		if err != nil {
			return nil, controlplane.NewGatewayError("invalid_request", err.Error())
		}
		return item, nil
	})
	router.Register("agents.update", []string{ScopeAgentsUpdate}, func(ctx context.Context, _ *controlplane.SessionContext, params []byte) (any, *controlplane.GatewayError) {
		var payload agentdto.UpdateAgentRequest
		if err := json.Unmarshal(params, &payload); err != nil {
			return nil, controlplane.NewGatewayError("invalid_params", "invalid agents.update params")
		}
		item, err := service.UpdateAgent(ctx, payload)
		if err != nil {
			return nil, controlplane.NewGatewayError("invalid_request", err.Error())
		}
		return item, nil
	})
	router.Register("agents.delete", []string{ScopeAgentsDelete}, func(ctx context.Context, _ *controlplane.SessionContext, params []byte) (any, *controlplane.GatewayError) {
		var payload agentdto.DeleteAgentRequest
		if err := json.Unmarshal(params, &payload); err != nil {
			return nil, controlplane.NewGatewayError("invalid_params", "invalid agents.delete params")
		}
		if err := service.DeleteAgent(ctx, payload); err != nil {
			return nil, controlplane.NewGatewayError("invalid_request", err.Error())
		}
		return map[string]any{"ok": true}, nil
	})
	router.Register("agents.files.list", []string{ScopeAgentsFilesList}, func(ctx context.Context, _ *controlplane.SessionContext, params []byte) (any, *controlplane.GatewayError) {
		var payload agentdto.AgentFilesListRequest
		if err := json.Unmarshal(params, &payload); err != nil {
			return nil, controlplane.NewGatewayError("invalid_params", "invalid agents.files.list params")
		}
		resp, err := service.ListAgentFiles(ctx, payload)
		if err != nil {
			return nil, controlplane.NewGatewayError("invalid_request", err.Error())
		}
		return resp, nil
	})
	router.Register("agents.files.get", []string{ScopeAgentsFilesGet}, func(ctx context.Context, _ *controlplane.SessionContext, params []byte) (any, *controlplane.GatewayError) {
		var payload agentdto.AgentFilesGetRequest
		if err := json.Unmarshal(params, &payload); err != nil {
			return nil, controlplane.NewGatewayError("invalid_params", "invalid agents.files.get params")
		}
		entry, err := service.GetAgentFile(ctx, payload)
		if err != nil {
			return nil, controlplane.NewGatewayError("invalid_request", err.Error())
		}
		return entry, nil
	})
	router.Register("agents.files.set", []string{ScopeAgentsFilesSet}, func(ctx context.Context, _ *controlplane.SessionContext, params []byte) (any, *controlplane.GatewayError) {
		var payload agentdto.AgentFilesSetRequest
		if err := json.Unmarshal(params, &payload); err != nil {
			return nil, controlplane.NewGatewayError("invalid_params", "invalid agents.files.set params")
		}
		entry, err := service.SetAgentFile(ctx, payload)
		if err != nil {
			return nil, controlplane.NewGatewayError("invalid_request", err.Error())
		}
		return entry, nil
	})
}
