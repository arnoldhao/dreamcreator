package methods

import (
	"context"
	"encoding/json"

	"dreamcreator/internal/application/gateway/controlplane"
	"dreamcreator/internal/application/gateway/nodes"
)

const (
	ScopeNodeList     = "node.list"
	ScopeNodeDescribe = "node.describe"
	ScopeNodeInvoke   = "node.invoke"
)

type nodeDescribeParams struct {
	NodeID string `json:"nodeId"`
}

type nodeInvokeParams struct {
	InvokeID   string `json:"invokeId"`
	NodeID     string `json:"nodeId"`
	Capability string `json:"capability"`
	Action     string `json:"action,omitempty"`
	Args       string `json:"args,omitempty"`
	TimeoutMs  int    `json:"timeoutMs,omitempty"`
}

func RegisterNodes(router *controlplane.Router, service *nodes.Service) {
	if router == nil || service == nil {
		return
	}
	router.Register("node.list", []string{ScopeNodeList}, func(ctx context.Context, _ *controlplane.SessionContext, _ []byte) (any, *controlplane.GatewayError) {
		items, err := service.ListNodes(ctx)
		if err != nil {
			return nil, controlplane.NewGatewayError("internal_error", err.Error())
		}
		return items, nil
	})
	router.Register("node.describe", []string{ScopeNodeDescribe}, func(ctx context.Context, _ *controlplane.SessionContext, params []byte) (any, *controlplane.GatewayError) {
		var payload nodeDescribeParams
		if err := json.Unmarshal(params, &payload); err != nil {
			return nil, controlplane.NewGatewayError("invalid_params", "invalid node.describe params")
		}
		item, err := service.DescribeNode(ctx, payload.NodeID)
		if err != nil {
			return nil, controlplane.NewGatewayError("not_found", err.Error())
		}
		return item, nil
	})
	router.Register("node.invoke", []string{ScopeNodeInvoke}, func(ctx context.Context, _ *controlplane.SessionContext, params []byte) (any, *controlplane.GatewayError) {
		var payload nodeInvokeParams
		if err := json.Unmarshal(params, &payload); err != nil {
			return nil, controlplane.NewGatewayError("invalid_params", "invalid node.invoke params")
		}
		result, err := service.Invoke(ctx, nodes.NodeInvokeRequest{
			InvokeID:   payload.InvokeID,
			NodeID:     payload.NodeID,
			Capability: payload.Capability,
			Action:     payload.Action,
			Args:       payload.Args,
			TimeoutMs:  payload.TimeoutMs,
		})
		if err != nil {
			return nil, controlplane.NewGatewayError("invoke_failed", err.Error())
		}
		return result, nil
	})
}
