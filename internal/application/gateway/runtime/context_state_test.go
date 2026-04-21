package runtime

import (
	"context"
	"testing"
	"time"

	sessionapp "dreamcreator/internal/application/session"
	"dreamcreator/internal/domain/thread"
)

func TestPersistCompactedContextStateStoresSummaryAndBoundary(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.April, 20, 12, 30, 0, 0, time.UTC)
	sessionService := sessionapp.NewService(sessionapp.NewInMemoryStore())
	if _, err := sessionService.CreateSession(context.Background(), sessionapp.CreateSessionRequest{
		SessionID: "thread-state-1",
	}); err != nil {
		t.Fatalf("create session: %v", err)
	}
	service := &Service{
		messages: &runtimeMessageRepositoryStub{
			items: []thread.ThreadMessage{
				mustThreadMessage(t, "msg-1", "thread-state-1", "user", "第一轮用户消息", now),
				mustThreadMessage(t, "msg-2", "thread-state-1", "assistant", "第一轮结论", now.Add(time.Second)),
				mustThreadMessage(t, "msg-3", "thread-state-1", "user", "第二轮用户消息", now.Add(2*time.Second)),
			},
		},
		sessions: sessionService,
		now: func() time.Time {
			return now.Add(3 * time.Second)
		},
	}

	service.persistCompactedContextState(context.Background(), "thread-state-1", contextGuardConfig{
		keepRecentTokens: 1,
	}, &contextGuardState{
		lastSummary: "历史已压缩。",
	})

	stored, err := sessionService.Get(context.Background(), "thread-state-1")
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	if stored.ContextSummary != "历史已压缩。" {
		t.Fatalf("unexpected context summary: %q", stored.ContextSummary)
	}
	if stored.ContextFirstKeptMessageID != "msg-3" {
		t.Fatalf("unexpected first kept message id: %q", stored.ContextFirstKeptMessageID)
	}
	if stored.ContextStrategyVersion != persistedContextStrategyVersion {
		t.Fatalf("unexpected strategy version: %d", stored.ContextStrategyVersion)
	}
}
