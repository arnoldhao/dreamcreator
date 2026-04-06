package methods

import (
	"context"
	"encoding/json"

	"dreamcreator/internal/application/gateway/controlplane"
	gatewaysubagent "dreamcreator/internal/application/gateway/subagent"
	subagentservice "dreamcreator/internal/application/subagent/service"
)

const (
	ScopeSubagentSpawn = "subagent.spawn"
	ScopeSubagentGet   = "subagent.get"
	ScopeSubagentList  = "subagent.list"
)

type subagentSpawnParams struct {
	ParentSessionKey  string         `json:"parentSessionKey"`
	ParentRunID       string         `json:"parentRunId,omitempty"`
	AgentID           string         `json:"agentId,omitempty"`
	Task              string         `json:"task,omitempty"`
	Label             string         `json:"label,omitempty"`
	Model             string         `json:"model,omitempty"`
	Thinking          string         `json:"thinking,omitempty"`
	RunTimeoutSeconds int            `json:"runTimeoutSeconds,omitempty"`
	Cleanup           string         `json:"cleanup,omitempty"`
	Payload           map[string]any `json:"payload,omitempty"`
}

type subagentGetParams struct {
	RunID string `json:"runId"`
}

type subagentListParams struct {
	ParentSessionKey string `json:"parentSessionKey"`
}

func RegisterSubagent(router *controlplane.Router, service *gatewaysubagent.GatewayService) {
	if router == nil || service == nil {
		return
	}
	router.Register("subagent.spawn", []string{ScopeSubagentSpawn}, func(ctx context.Context, _ *controlplane.SessionContext, params []byte) (any, *controlplane.GatewayError) {
		var payload subagentSpawnParams
		if err := json.Unmarshal(params, &payload); err != nil {
			return nil, controlplane.NewGatewayError("invalid_params", "invalid subagent.spawn params")
		}
		record, err := service.Spawn(ctx, subagentservice.SpawnRequest{
			ParentSessionKey:  payload.ParentSessionKey,
			ParentRunID:       payload.ParentRunID,
			AgentID:           payload.AgentID,
			Task:              payload.Task,
			Label:             payload.Label,
			Model:             payload.Model,
			Thinking:          payload.Thinking,
			RunTimeoutSeconds: payload.RunTimeoutSeconds,
			CleanupPolicy:     subagentservice.ParseCleanupPolicy(payload.Cleanup),
			Payload:           payload.Payload,
		})
		if err != nil {
			return nil, controlplane.NewGatewayError("spawn_failed", err.Error())
		}
		return record, nil
	})
	router.Register("subagent.get", []string{ScopeSubagentGet}, func(ctx context.Context, _ *controlplane.SessionContext, params []byte) (any, *controlplane.GatewayError) {
		var payload subagentGetParams
		if err := json.Unmarshal(params, &payload); err != nil {
			return nil, controlplane.NewGatewayError("invalid_params", "invalid subagent.get params")
		}
		record, err := service.Get(ctx, payload.RunID)
		if err != nil {
			return nil, controlplane.NewGatewayError("not_found", err.Error())
		}
		return record, nil
	})
	router.Register("subagent.list", []string{ScopeSubagentList}, func(ctx context.Context, _ *controlplane.SessionContext, params []byte) (any, *controlplane.GatewayError) {
		var payload subagentListParams
		if err := json.Unmarshal(params, &payload); err != nil {
			return nil, controlplane.NewGatewayError("invalid_params", "invalid subagent.list params")
		}
		records, err := service.ListByParent(ctx, payload.ParentSessionKey)
		if err != nil {
			return nil, controlplane.NewGatewayError("not_found", err.Error())
		}
		return records, nil
	})
}
