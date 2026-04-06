package auth

import (
	"context"
	"testing"
)

func TestInMemoryAuthenticate(t *testing.T) {
	service := NewInMemoryService()
	_, err := service.Authenticate(context.Background(), Credentials{}, "operator", nil)
	if err == nil {
		t.Fatalf("expected unauthorized for empty token")
	}

	ctx := AuthContext{Subject: "user-1", Scopes: []string{"scope:a"}}
	service.AddToken("token-1", ctx)
	result, err := service.Authenticate(context.Background(), Credentials{Token: "token-1"}, "operator", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Subject != "user-1" {
		t.Fatalf("unexpected subject: %s", result.Subject)
	}
}

func TestScopeGuard(t *testing.T) {
	guard := NewDefaultScopeGuard()
	result := guard.Check("m", AuthContext{Scopes: []string{"a", "b"}}, []string{"a"})
	if !result.Allowed {
		t.Fatalf("expected allowed")
	}
	denied := guard.Check("m", AuthContext{Scopes: []string{"a"}}, []string{"b"})
	if denied.Allowed {
		t.Fatalf("expected denied")
	}
}
