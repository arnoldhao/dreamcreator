package controlplane

import (
	"context"
	"encoding/json"
	"testing"

	"dreamcreator/internal/application/gateway/auth"
)

func TestRouterHandleWithScopes(t *testing.T) {
	guard := auth.NewDefaultScopeGuard()
	router := NewRouter(guard)
	router.Register("echo", []string{"scope:echo"}, func(_ context.Context, _ *SessionContext, params []byte) (any, *GatewayError) {
		var payload map[string]string
		_ = json.Unmarshal(params, &payload)
		return payload, nil
	})

	session := &SessionContext{
		Auth: auth.AuthContext{Scopes: []string{"scope:echo"}},
	}

	req := RequestFrame{
		Type:   "req",
		ID:     "1",
		Method: "echo",
		Params: json.RawMessage(`{"message":"hi"}`),
	}
	resp := router.Handle(context.Background(), session, req)
	if !resp.OK {
		t.Fatalf("expected ok response, got error: %#v", resp.Error)
	}
	payload, ok := resp.Payload.(map[string]string)
	if !ok || payload["message"] != "hi" {
		t.Fatalf("unexpected payload: %#v", resp.Payload)
	}
}

func TestRouterDeniedWhenScopeMissing(t *testing.T) {
	guard := auth.NewDefaultScopeGuard()
	router := NewRouter(guard)
	router.Register("secret", []string{"scope:secret"}, func(_ context.Context, _ *SessionContext, _ []byte) (any, *GatewayError) {
		return map[string]string{"ok": "true"}, nil
	})

	session := &SessionContext{
		Auth: auth.AuthContext{Scopes: []string{"scope:public"}},
	}

	req := RequestFrame{
		Type:   "req",
		ID:     "2",
		Method: "secret",
	}
	resp := router.Handle(context.Background(), session, req)
	if resp.OK || resp.Error == nil || resp.Error.Code != "forbidden" {
		t.Fatalf("expected forbidden response, got %#v", resp)
	}
}
