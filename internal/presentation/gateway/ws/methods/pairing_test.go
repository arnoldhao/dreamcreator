package methods

import (
	"context"
	"encoding/json"
	"testing"

	"dreamcreator/internal/application/gateway/auth"
	"dreamcreator/internal/application/gateway/controlplane"
	"dreamcreator/internal/application/gateway/pairing"
)

func TestPairingMethods(t *testing.T) {
	router := controlplane.NewRouter(auth.NewDefaultScopeGuard())
	service := pairing.NewService()
	RegisterPairing(router, service)

	session := &controlplane.SessionContext{Auth: auth.AuthContext{Scopes: []string{ScopeNodePair, ScopeNodeToken}}}

	reqParams, _ := json.Marshal(PairRequestParams{NodeID: "node-1"})
	resp := router.Handle(context.Background(), session, controlplane.RequestFrame{
		Type:   "req",
		ID:     "1",
		Method: "node.pair.request",
		Params: reqParams,
	})
	if !resp.OK {
		t.Fatalf("pair request failed: %#v", resp.Error)
	}

	req, ok := resp.Payload.(pairing.PairRequest)
	if !ok {
		t.Fatalf("unexpected payload type: %#v", resp.Payload)
	}
	approveParams, _ := json.Marshal(PairDecisionParams{RequestID: req.ID})
	approveResp := router.Handle(context.Background(), session, controlplane.RequestFrame{
		Type:   "req",
		ID:     "2",
		Method: "node.pair.approve",
		Params: approveParams,
	})
	if !approveResp.OK {
		t.Fatalf("approve failed: %#v", approveResp.Error)
	}
}
