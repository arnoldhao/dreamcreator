package methods

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"dreamcreator/internal/application/gateway/controlplane"
	"dreamcreator/internal/application/gateway/pairing"
)

const (
	ScopeNodePair  = "node.pair"
	ScopeNodeToken = "node.token"
)

type PairRequestParams struct {
	NodeID string `json:"nodeId"`
}

type PairDecisionParams struct {
	RequestID string `json:"requestId"`
}

type TokenRotateParams struct {
	TokenID    string `json:"tokenId"`
	TTLSeconds int    `json:"ttlSeconds,omitempty"`
}

type TokenRevokeParams struct {
	TokenID string `json:"tokenId"`
}

func RegisterPairing(router *controlplane.Router, service *pairing.Service) {
	if router == nil || service == nil {
		return
	}
	router.Register("node.pair.request", []string{ScopeNodePair}, func(_ context.Context, _ *controlplane.SessionContext, params []byte) (any, *controlplane.GatewayError) {
		var payload PairRequestParams
		if err := json.Unmarshal(params, &payload); err != nil {
			return nil, controlplane.NewGatewayError("invalid_params", "invalid pair request")
		}
		req, err := service.Request(payload.NodeID)
		if err != nil {
			return nil, controlplane.NewGatewayError("invalid_request", err.Error())
		}
		return req, nil
	})
	router.Register("node.pair.approve", []string{ScopeNodePair}, func(_ context.Context, _ *controlplane.SessionContext, params []byte) (any, *controlplane.GatewayError) {
		var payload PairDecisionParams
		if err := json.Unmarshal(params, &payload); err != nil {
			return nil, controlplane.NewGatewayError("invalid_params", "invalid approve request")
		}
		req, err := service.Approve(payload.RequestID)
		if err != nil {
			return nil, controlplane.NewGatewayError("not_found", err.Error())
		}
		token, err := service.IssueToken(req.NodeID, 24*time.Hour)
		if err != nil {
			return nil, controlplane.NewGatewayError("token_failed", err.Error())
		}
		return map[string]any{"request": req, "token": token}, nil
	})
	router.Register("node.pair.reject", []string{ScopeNodePair}, func(_ context.Context, _ *controlplane.SessionContext, params []byte) (any, *controlplane.GatewayError) {
		var payload PairDecisionParams
		if err := json.Unmarshal(params, &payload); err != nil {
			return nil, controlplane.NewGatewayError("invalid_params", "invalid reject request")
		}
		req, err := service.Reject(payload.RequestID)
		if err != nil {
			return nil, controlplane.NewGatewayError("not_found", err.Error())
		}
		return req, nil
	})
	router.Register("node.token.rotate", []string{ScopeNodeToken}, func(_ context.Context, _ *controlplane.SessionContext, params []byte) (any, *controlplane.GatewayError) {
		var payload TokenRotateParams
		if err := json.Unmarshal(params, &payload); err != nil {
			return nil, controlplane.NewGatewayError("invalid_params", "invalid rotate request")
		}
		ttl := time.Duration(payload.TTLSeconds) * time.Second
		token, err := service.RotateToken(payload.TokenID, ttl)
		if err != nil {
			return nil, controlplane.NewGatewayError("not_found", err.Error())
		}
		return token, nil
	})
	router.Register("node.token.revoke", []string{ScopeNodeToken}, func(_ context.Context, _ *controlplane.SessionContext, params []byte) (any, *controlplane.GatewayError) {
		var payload TokenRevokeParams
		if err := json.Unmarshal(params, &payload); err != nil {
			return nil, controlplane.NewGatewayError("invalid_params", "invalid revoke request")
		}
		tokenID := strings.TrimSpace(payload.TokenID)
		if tokenID == "" {
			return nil, controlplane.NewGatewayError("invalid_params", "token id is required")
		}
		token, err := service.RevokeToken(tokenID)
		if err != nil {
			return nil, controlplane.NewGatewayError("not_found", err.Error())
		}
		return token, nil
	})
}
