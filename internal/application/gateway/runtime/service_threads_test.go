package runtime

import (
	"context"
	"testing"
	"time"

	runtimedto "dreamcreator/internal/application/gateway/runtime/dto"
	threaddto "dreamcreator/internal/application/thread/dto"
	"dreamcreator/internal/domain/thread"
)

type runtimeThreadRepositoryStub struct {
	item thread.Thread
}

type runtimeRunRepositoryStub struct {
	item      thread.ThreadRun
	saveCount int
}

func (repo *runtimeThreadRepositoryStub) List(context.Context, bool) ([]thread.Thread, error) {
	return []thread.Thread{repo.item}, nil
}

func (repo *runtimeThreadRepositoryStub) ListPurgeCandidates(context.Context, time.Time, int) ([]thread.Thread, error) {
	return nil, nil
}

func (repo *runtimeThreadRepositoryStub) Get(_ context.Context, id string) (thread.Thread, error) {
	if id != repo.item.ID {
		return thread.Thread{}, thread.ErrThreadNotFound
	}
	return repo.item, nil
}

func (repo *runtimeThreadRepositoryStub) Save(_ context.Context, item thread.Thread) error {
	repo.item = item
	return nil
}

func (repo *runtimeThreadRepositoryStub) SoftDelete(context.Context, string, *time.Time, *time.Time) error {
	return nil
}

func (repo *runtimeThreadRepositoryStub) Restore(context.Context, string) error {
	return nil
}

func (repo *runtimeThreadRepositoryStub) Purge(context.Context, string) error {
	return nil
}

func (repo *runtimeThreadRepositoryStub) SetStatus(context.Context, string, thread.Status, time.Time) error {
	return nil
}

func (repo *runtimeRunRepositoryStub) Get(_ context.Context, id string) (thread.ThreadRun, error) {
	if repo.item.ID != id {
		return thread.ThreadRun{}, thread.ErrRunNotFound
	}
	return repo.item, nil
}

func (repo *runtimeRunRepositoryStub) ListActiveByThread(context.Context, string) ([]thread.ThreadRun, error) {
	return nil, nil
}

func (repo *runtimeRunRepositoryStub) ListByAgentID(context.Context, string, int) ([]thread.ThreadRun, error) {
	return nil, nil
}

func (repo *runtimeRunRepositoryStub) CountActive(context.Context) (int, error) {
	return 0, nil
}

func (repo *runtimeRunRepositoryStub) Save(_ context.Context, run thread.ThreadRun) error {
	repo.item = run
	repo.saveCount++
	return nil
}

type capturingThreadTitleGenerator struct {
	requests chan threaddto.GenerateThreadTitleRequest
}

func (stub *capturingThreadTitleGenerator) GenerateThreadTitle(_ context.Context, request threaddto.GenerateThreadTitleRequest) (threaddto.GenerateThreadTitleResponse, error) {
	stub.requests <- request
	return threaddto.GenerateThreadTitleResponse{
		ThreadID: request.ThreadID,
		Title:    request.ThreadID,
	}, nil
}

func newRuntimeThreadForTitleTest(t *testing.T, id string, title string, titleIsDefault bool, changedBy thread.TitleChangedBy) thread.Thread {
	t.Helper()
	now := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
	item, err := thread.NewThread(thread.ThreadParams{
		ID:             id,
		AssistantID:    "assistant-1",
		Title:          title,
		TitleIsDefault: titleIsDefault,
		TitleChangedBy: changedBy,
		Status:         thread.ThreadStatusRegular,
		CreatedAt:      &now,
		UpdatedAt:      &now,
	})
	if err != nil {
		t.Fatalf("new thread: %v", err)
	}
	return item
}

func TestShouldScheduleThreadTitleGenerationAtRequestStart(t *testing.T) {
	t.Parallel()

	item := newRuntimeThreadForTitleTest(t, "session-1", "帮我做个旅行计划", true, "")
	messages := []runtimedto.Message{
		{Role: "user", Content: "帮我做个旅行计划"},
	}

	if !shouldScheduleThreadTitleGenerationAtRequestStart(item, messages) {
		t.Fatalf("expected default-title thread to schedule title generation")
	}
}

func TestScheduleThreadTitleGenerationAfterRun_UsesPersistedMessages(t *testing.T) {
	t.Parallel()

	repo := &runtimeThreadRepositoryStub{
		item: newRuntimeThreadForTitleTest(t, "session-2", "用户首条消息", true, ""),
	}
	generator := &capturingThreadTitleGenerator{
		requests: make(chan threaddto.GenerateThreadTitleRequest, 1),
	}
	service := &Service{
		threads:      repo,
		threadTitles: generator,
	}

	service.scheduleThreadTitleGenerationAfterRun("session-2")

	select {
	case request := <-generator.requests:
		if request.ThreadID != "session-2" {
			t.Fatalf("unexpected thread id: %q", request.ThreadID)
		}
		if len(request.Messages) != 0 {
			t.Fatalf("after-run should rely on persisted messages, got %d request messages", len(request.Messages))
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for title generation request")
	}
}

func TestScheduleThreadTitleGenerationAfterRun_WaitsForInFlightGeneration(t *testing.T) {
	t.Parallel()

	repo := &runtimeThreadRepositoryStub{
		item: newRuntimeThreadForTitleTest(t, "session-3", defaultRuntimeThreadTitle, true, ""),
	}
	generator := &capturingThreadTitleGenerator{
		requests: make(chan threaddto.GenerateThreadTitleRequest, 1),
	}
	service := &Service{
		threads:                 repo,
		threadTitles:            generator,
		titleGenerationInFlight: map[string]struct{}{"session-3": {}},
	}

	service.scheduleThreadTitleGenerationAfterRun("session-3")

	time.AfterFunc(150*time.Millisecond, func() {
		service.unmarkThreadTitleGenerationInFlight("session-3")
	})

	select {
	case request := <-generator.requests:
		if request.ThreadID != "session-3" {
			t.Fatalf("unexpected thread id: %q", request.ThreadID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for deferred title generation request")
	}
}

func TestAttachRunUserMessageIDUpdatesStoredRun(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.April, 20, 8, 0, 0, 0, time.UTC)
	run, err := thread.NewThreadRun(thread.ThreadRunParams{
		ID:                 "run-1",
		ThreadID:           "thread-1",
		AssistantMessageID: "assistant-msg-1",
		CreatedAt:          &now,
		UpdatedAt:          &now,
	})
	if err != nil {
		t.Fatalf("new run: %v", err)
	}
	repo := &runtimeRunRepositoryStub{item: run}
	service := &Service{
		runs: repo,
		now: func() time.Time {
			return now.Add(30 * time.Second)
		},
	}

	service.attachRunUserMessageID(context.Background(), &run, "user-msg-1")

	if repo.saveCount != 1 {
		t.Fatalf("expected one save, got %d", repo.saveCount)
	}
	if repo.item.UserMessageID != "user-msg-1" {
		t.Fatalf("expected user message id to be persisted, got %q", repo.item.UserMessageID)
	}
	if !repo.item.UpdatedAt.Equal(now.Add(30 * time.Second)) {
		t.Fatalf("expected updatedAt to advance, got %s", repo.item.UpdatedAt)
	}
}
