package threadrepo

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"dreamcreator/internal/domain/thread"
	"dreamcreator/internal/infrastructure/persistence"
)

func TestSQLiteThreadMessageRepository_ListByThreadStableAfterUpsert(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "thread_messages.db")
	database, err := persistence.OpenSQLite(ctx, persistence.SQLiteConfig{Path: dbPath})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer database.Close()

	threadRepo := NewSQLiteThreadRepository(database.Bun)
	messageRepo := NewSQLiteThreadMessageRepository(database.Bun)

	threadID := "thread-1"
	baseTime := time.Date(2026, 4, 3, 10, 0, 0, 0, time.UTC)
	threadItem, err := thread.NewThread(thread.ThreadParams{
		ID:                threadID,
		AssistantID:       "assistant-1",
		Title:             "Test",
		Status:            thread.ThreadStatusRegular,
		CreatedAt:         &baseTime,
		UpdatedAt:         &baseTime,
		LastInteractiveAt: &baseTime,
	})
	if err != nil {
		t.Fatalf("new thread: %v", err)
	}
	if err := threadRepo.Save(ctx, threadItem); err != nil {
		t.Fatalf("save thread: %v", err)
	}

	appendMessage := func(messageID string, role string, content string, createdAt time.Time) {
		t.Helper()
		msg, msgErr := thread.NewThreadMessage(thread.ThreadMessageParams{
			ID:        messageID,
			ThreadID:  threadID,
			Role:      role,
			Content:   content,
			CreatedAt: &createdAt,
		})
		if msgErr != nil {
			t.Fatalf("new thread message: %v", msgErr)
		}
		if appendErr := messageRepo.Append(ctx, msg); appendErr != nil {
			t.Fatalf("append message: %v", appendErr)
		}
	}

	t1 := baseTime.Add(1 * time.Second)
	t2 := baseTime.Add(2 * time.Second)
	t3 := baseTime.Add(3 * time.Second)
	t4 := baseTime.Add(10 * time.Second)

	appendMessage("req-a", "user", "request a", t1)
	appendMessage("resp-a", "assistant", "response a", t2)
	appendMessage("req-b", "user", "request b", t3)

	// Duplicate write for the same message id should not shift historical ordering.
	appendMessage("req-a", "user", "request a", t4)

	items, err := messageRepo.ListByThread(ctx, threadID, 0)
	if err != nil {
		t.Fatalf("list messages: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(items))
	}

	if items[0].ID != "req-a" || items[1].ID != "resp-a" || items[2].ID != "req-b" {
		t.Fatalf("unexpected order: got [%s, %s, %s]", items[0].ID, items[1].ID, items[2].ID)
	}
	if !items[0].CreatedAt.Equal(t1) {
		t.Fatalf("created_at should preserve first write: got %s want %s", items[0].CreatedAt, t1)
	}
}

func TestSQLiteThreadMessageRepository_ListByThreadUsesInsertionOrder(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "thread_messages_order.db")
	database, err := persistence.OpenSQLite(ctx, persistence.SQLiteConfig{Path: dbPath})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer database.Close()

	threadRepo := NewSQLiteThreadRepository(database.Bun)
	messageRepo := NewSQLiteThreadMessageRepository(database.Bun)

	threadID := "thread-2"
	baseTime := time.Date(2026, 4, 3, 11, 0, 0, 0, time.UTC)
	threadItem, err := thread.NewThread(thread.ThreadParams{
		ID:                threadID,
		AssistantID:       "assistant-1",
		Title:             "Test",
		Status:            thread.ThreadStatusRegular,
		CreatedAt:         &baseTime,
		UpdatedAt:         &baseTime,
		LastInteractiveAt: &baseTime,
	})
	if err != nil {
		t.Fatalf("new thread: %v", err)
	}
	if err := threadRepo.Save(ctx, threadItem); err != nil {
		t.Fatalf("save thread: %v", err)
	}

	appendMessage := func(messageID string, role string, content string, createdAt time.Time) {
		t.Helper()
		msg, msgErr := thread.NewThreadMessage(thread.ThreadMessageParams{
			ID:        messageID,
			ThreadID:  threadID,
			Role:      role,
			Content:   content,
			CreatedAt: &createdAt,
		})
		if msgErr != nil {
			t.Fatalf("new thread message: %v", msgErr)
		}
		if appendErr := messageRepo.Append(ctx, msg); appendErr != nil {
			t.Fatalf("append message: %v", appendErr)
		}
	}

	// Persist in conversation order, but with out-of-order timestamps.
	appendMessage("req-a", "user", "request a", baseTime.Add(3*time.Second))
	appendMessage("resp-a", "assistant", "response a", baseTime.Add(1*time.Second))
	appendMessage("req-b", "user", "request b", baseTime.Add(2*time.Second))

	items, err := messageRepo.ListByThread(ctx, threadID, 0)
	if err != nil {
		t.Fatalf("list messages: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(items))
	}
	if items[0].ID != "req-a" || items[1].ID != "resp-a" || items[2].ID != "req-b" {
		t.Fatalf("unexpected order: got [%s, %s, %s]", items[0].ID, items[1].ID, items[2].ID)
	}
}
