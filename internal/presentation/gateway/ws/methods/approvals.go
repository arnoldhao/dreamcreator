package methods

import (
	"context"
	"encoding/json"

	"dreamcreator/internal/application/gateway/approvals"
	"dreamcreator/internal/application/gateway/controlplane"
)

const (
	ScopeExecApprovalRequest = "exec.approval.request"
	ScopeExecApprovalResolve = "exec.approval.resolve"
	ScopeExecApprovalWait    = "exec.approval.wait"
)

type approvalRequestParams struct {
	SessionKey string `json:"sessionKey,omitempty"`
	ToolCallID string `json:"toolCallId,omitempty"`
	ToolName   string `json:"toolName,omitempty"`
	Action     string `json:"action,omitempty"`
	Args       string `json:"args,omitempty"`
}

type approvalResolveParams struct {
	ID       string `json:"id"`
	Decision string `json:"decision"`
	Reason   string `json:"reason,omitempty"`
}

type approvalWaitParams struct {
	ID            string `json:"id"`
	TimeoutMillis int    `json:"timeoutMs,omitempty"`
}

func RegisterApprovals(router *controlplane.Router, service *approvals.Service) {
	if router == nil || service == nil {
		return
	}
	router.Register("exec.approval.request", []string{ScopeExecApprovalRequest}, func(ctx context.Context, _ *controlplane.SessionContext, params []byte) (any, *controlplane.GatewayError) {
		var payload approvalRequestParams
		if err := json.Unmarshal(params, &payload); err != nil {
			return nil, controlplane.NewGatewayError("invalid_params", "invalid approval request")
		}
		req, err := service.Create(ctx, approvals.Request{
			SessionKey: payload.SessionKey,
			ToolCallID: payload.ToolCallID,
			ToolName:   payload.ToolName,
			Action:     payload.Action,
			Args:       payload.Args,
		})
		if err != nil {
			return nil, controlplane.NewGatewayError("invalid_request", err.Error())
		}
		return req, nil
	})
	router.Register("exec.approval.resolve", []string{ScopeExecApprovalResolve}, func(ctx context.Context, _ *controlplane.SessionContext, params []byte) (any, *controlplane.GatewayError) {
		var payload approvalResolveParams
		if err := json.Unmarshal(params, &payload); err != nil {
			return nil, controlplane.NewGatewayError("invalid_params", "invalid approval resolve params")
		}
		req, err := service.Resolve(ctx, payload.ID, payload.Decision, payload.Reason)
		if err != nil {
			return nil, controlplane.NewGatewayError("invalid_request", err.Error())
		}
		return req, nil
	})
	router.Register("exec.approval.wait", []string{ScopeExecApprovalWait}, func(ctx context.Context, _ *controlplane.SessionContext, params []byte) (any, *controlplane.GatewayError) {
		var payload approvalWaitParams
		if err := json.Unmarshal(params, &payload); err != nil {
			return nil, controlplane.NewGatewayError("invalid_params", "invalid approval wait params")
		}
		req, err := service.Wait(ctx, approvals.WaitRequest{ID: payload.ID})
		if err != nil {
			return nil, controlplane.NewGatewayError("invalid_request", err.Error())
		}
		return req, nil
	})
}
