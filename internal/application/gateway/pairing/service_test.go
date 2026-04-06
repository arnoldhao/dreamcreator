package pairing

import "testing"

func TestPairingLifecycle(t *testing.T) {
	service := NewService()
	req, err := service.Request("node-1")
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	if req.Status != PairStatusPending {
		t.Fatalf("unexpected status: %s", req.Status)
	}
	approved, err := service.Approve(req.ID)
	if err != nil {
		t.Fatalf("approve error: %v", err)
	}
	if approved.Status != PairStatusApproved {
		t.Fatalf("expected approved")
	}
	token, err := service.IssueToken(req.NodeID, 0)
	if err != nil {
		t.Fatalf("token error: %v", err)
	}
	if !service.ValidateToken(token.ID) {
		t.Fatalf("token should be valid")
	}
	if _, err := service.RevokeToken(token.ID); err != nil {
		t.Fatalf("revoke error: %v", err)
	}
	if service.ValidateToken(token.ID) {
		t.Fatalf("token should be revoked")
	}
}
