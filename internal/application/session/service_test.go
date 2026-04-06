package session

import (
	"context"
	"testing"

	domainsession "dreamcreator/internal/domain/session"
)

func TestSessionServiceCreate(t *testing.T) {
	service := NewService(NewInMemoryStore())
	entry, err := service.CreateSession(context.Background(), CreateSessionRequest{
		KeyParts: domainsession.KeyParts{
			Channel:   "web",
			AccountID: "acct",
			PrimaryID: "thread-1",
			ThreadRef: "thread-1",
		},
		Title: "Test",
	})
	if err != nil {
		t.Fatalf("create error: %v", err)
	}
	if entry.SessionID == "" {
		t.Fatalf("expected session id")
	}
	if entry.SessionKey == "" {
		t.Fatalf("expected session key")
	}
}

func TestSessionServiceUpdateTitle(t *testing.T) {
	service := NewService(NewInMemoryStore())
	entry, err := service.CreateSession(context.Background(), CreateSessionRequest{Title: "Old"})
	if err != nil {
		t.Fatalf("create error: %v", err)
	}
	if err := service.UpdateTitle(context.Background(), entry.SessionID, "New"); err != nil {
		t.Fatalf("update error: %v", err)
	}
}
