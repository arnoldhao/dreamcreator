package service

import (
	"context"
	"testing"
	"time"

	"dreamcreator/internal/application/thread/dto"
	"dreamcreator/internal/domain/thread"
)

type appendMessageRepoStub struct {
	last thread.ThreadMessage
}

func (repo *appendMessageRepoStub) ListByThread(context.Context, string, int) ([]thread.ThreadMessage, error) {
	return nil, nil
}

func (repo *appendMessageRepoStub) Append(_ context.Context, item thread.ThreadMessage) error {
	repo.last = item
	return nil
}

func (repo *appendMessageRepoStub) DeleteByThread(context.Context, string) error {
	return nil
}

func newThreadForAppendTest(t *testing.T, lastInteractiveAt time.Time) thread.Thread {
	t.Helper()
	createdAt := lastInteractiveAt.Add(-5 * time.Minute)
	updatedAt := lastInteractiveAt.Add(-1 * time.Minute)
	item, err := thread.NewThread(thread.ThreadParams{
		ID:                "thread-append",
		AssistantID:       "assistant-1",
		Title:             "Test",
		Status:            thread.ThreadStatusRegular,
		CreatedAt:         &createdAt,
		UpdatedAt:         &updatedAt,
		LastInteractiveAt: &lastInteractiveAt,
	})
	if err != nil {
		t.Fatalf("new thread: %v", err)
	}
	return item
}

func TestAppendMessage_NoticeDoesNotAdvanceLastInteractiveAt(t *testing.T) {
	t.Parallel()

	lastInteractiveAt := time.Date(2026, 4, 1, 9, 0, 0, 0, time.UTC)
	threadRepo := &threadRepositoryStub{item: newThreadForAppendTest(t, lastInteractiveAt)}
	messageRepo := &appendMessageRepoStub{}
	now := lastInteractiveAt.Add(2 * time.Minute)
	service := &ThreadService{
		threads:  threadRepo,
		messages: messageRepo,
		now:      func() time.Time { return now },
		newID:    func() string { return "msg-notice" },
	}

	if err := service.AppendMessage(context.Background(), dto.AppendMessageRequest{
		ThreadID: "thread-append",
		Kind:     "notice",
		Role:     "assistant",
		Content:  "Background notice",
	}); err != nil {
		t.Fatalf("append notice: %v", err)
	}

	if messageRepo.last.Kind != thread.ThreadMessageKindNotice {
		t.Fatalf("expected notice kind, got %q", messageRepo.last.Kind)
	}
	if !threadRepo.item.LastInteractiveAt.Equal(lastInteractiveAt) {
		t.Fatalf("lastInteractiveAt changed: got %s want %s", threadRepo.item.LastInteractiveAt, lastInteractiveAt)
	}
	if !threadRepo.item.UpdatedAt.Equal(now) {
		t.Fatalf("updatedAt not advanced: got %s want %s", threadRepo.item.UpdatedAt, now)
	}
}

func TestAppendMessage_ChatAdvancesLastInteractiveAt(t *testing.T) {
	t.Parallel()

	lastInteractiveAt := time.Date(2026, 4, 1, 9, 0, 0, 0, time.UTC)
	threadRepo := &threadRepositoryStub{item: newThreadForAppendTest(t, lastInteractiveAt)}
	messageRepo := &appendMessageRepoStub{}
	now := lastInteractiveAt.Add(3 * time.Minute)
	service := &ThreadService{
		threads:  threadRepo,
		messages: messageRepo,
		now:      func() time.Time { return now },
		newID:    func() string { return "msg-chat" },
	}

	if err := service.AppendMessage(context.Background(), dto.AppendMessageRequest{
		ThreadID: "thread-append",
		Role:     "assistant",
		Content:  "Normal reply",
	}); err != nil {
		t.Fatalf("append chat: %v", err)
	}

	if messageRepo.last.Kind != thread.ThreadMessageKindChat {
		t.Fatalf("expected chat kind, got %q", messageRepo.last.Kind)
	}
	if !threadRepo.item.LastInteractiveAt.Equal(now) {
		t.Fatalf("lastInteractiveAt not advanced: got %s want %s", threadRepo.item.LastInteractiveAt, now)
	}
}
