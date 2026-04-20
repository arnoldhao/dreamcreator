package session

import (
	"context"
	"testing"
	"time"

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

func TestSessionServiceCreatePreservesExistingContextState(t *testing.T) {
	service := NewService(NewInMemoryStore())
	createdAt := time.Date(2026, time.April, 20, 10, 0, 0, 0, time.UTC)
	updatedAt := createdAt.Add(2 * time.Minute)
	contextUpdatedAt := createdAt.Add(time.Minute)
	service.now = func() time.Time { return createdAt }
	entry, err := service.CreateSession(context.Background(), CreateSessionRequest{
		SessionID: "session-1",
		Title:     "Original",
	})
	if err != nil {
		t.Fatalf("create error: %v", err)
	}
	entry.ContextPromptTokens = 1200
	entry.ContextTotalTokens = 2400
	entry.ContextWindowTokens = 128000
	entry.ContextUpdatedAt = contextUpdatedAt
	entry.ContextFresh = true
	entry.CompactionCount = 3
	entry.MemoryFlushCompactionCount = 1
	entry.UpdatedAt = updatedAt
	if err := service.store.Save(context.Background(), entry); err != nil {
		t.Fatalf("save error: %v", err)
	}

	service.now = func() time.Time { return updatedAt.Add(time.Minute) }
	updated, err := service.CreateSession(context.Background(), CreateSessionRequest{
		SessionID: "session-1",
		Title:     "",
	})
	if err != nil {
		t.Fatalf("upsert error: %v", err)
	}

	if updated.Title != "Original" {
		t.Fatalf("expected title to be preserved, got %q", updated.Title)
	}
	if updated.ContextPromptTokens != 1200 || updated.ContextTotalTokens != 2400 || updated.ContextWindowTokens != 128000 {
		t.Fatalf("expected context tokens to be preserved, got %d/%d/%d", updated.ContextPromptTokens, updated.ContextTotalTokens, updated.ContextWindowTokens)
	}
	if !updated.ContextUpdatedAt.Equal(contextUpdatedAt) {
		t.Fatalf("expected context updated at %s, got %s", contextUpdatedAt, updated.ContextUpdatedAt)
	}
	if !updated.ContextFresh {
		t.Fatal("expected context freshness to be preserved")
	}
	if updated.CompactionCount != 3 || updated.MemoryFlushCompactionCount != 1 {
		t.Fatalf("expected compaction counters to be preserved, got %d/%d", updated.CompactionCount, updated.MemoryFlushCompactionCount)
	}
	if !updated.CreatedAt.Equal(createdAt) {
		t.Fatalf("expected createdAt %s, got %s", createdAt, updated.CreatedAt)
	}
}

func TestSessionServiceUpdateContextCompactionState(t *testing.T) {
	service := NewService(NewInMemoryStore())
	entry, err := service.CreateSession(context.Background(), CreateSessionRequest{
		SessionID: "session-ctx-1",
		Title:     "Compaction",
	})
	if err != nil {
		t.Fatalf("create error: %v", err)
	}
	compactedAt := time.Date(2026, time.April, 20, 12, 0, 0, 0, time.UTC)
	if err := service.UpdateContextCompactionState(context.Background(), entry.SessionID, ContextCompactionStateUpdate{
		Summary:            "Older messages summarized.",
		FirstKeptMessageID: "msg-3",
		StrategyVersion:    1,
		CompactedAt:        compactedAt,
	}); err != nil {
		t.Fatalf("update compaction state error: %v", err)
	}

	stored, err := service.Get(context.Background(), entry.SessionID)
	if err != nil {
		t.Fatalf("get error: %v", err)
	}
	if stored.ContextSummary != "Older messages summarized." {
		t.Fatalf("unexpected summary: %q", stored.ContextSummary)
	}
	if stored.ContextFirstKeptMessageID != "msg-3" {
		t.Fatalf("unexpected first kept message id: %q", stored.ContextFirstKeptMessageID)
	}
	if stored.ContextStrategyVersion != 1 {
		t.Fatalf("unexpected strategy version: %d", stored.ContextStrategyVersion)
	}
	if !stored.ContextCompactedAt.Equal(compactedAt) {
		t.Fatalf("unexpected compactedAt: %s", stored.ContextCompactedAt)
	}

	if err := service.UpdateContextCompactionState(context.Background(), entry.SessionID, ContextCompactionStateUpdate{}); err != nil {
		t.Fatalf("clear compaction state error: %v", err)
	}
	cleared, err := service.Get(context.Background(), entry.SessionID)
	if err != nil {
		t.Fatalf("get cleared session error: %v", err)
	}
	if cleared.ContextSummary != "" || cleared.ContextFirstKeptMessageID != "" || cleared.ContextStrategyVersion != 0 || !cleared.ContextCompactedAt.IsZero() {
		t.Fatalf("expected compaction state to be cleared, got %+v", cleared)
	}
}
