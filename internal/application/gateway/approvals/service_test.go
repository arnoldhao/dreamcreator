package approvals

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestServiceWaitReturnsAfterResolve(t *testing.T) {
	service := NewService(nil)
	created, err := service.Create(context.Background(), Request{
		SessionKey: "session-main",
		ToolName:   "exec",
		Action:     "config.schema",
	})
	if err != nil {
		t.Fatalf("create approval: %v", err)
	}

	resultCh := make(chan Request, 1)
	errCh := make(chan error, 1)
	go func() {
		resolved, waitErr := service.Wait(context.Background(), WaitRequest{ID: created.ID})
		if waitErr != nil {
			errCh <- waitErr
			return
		}
		resultCh <- resolved
	}()

	time.Sleep(30 * time.Millisecond)
	if _, err := service.Resolve(context.Background(), created.ID, "approve", "approved in test"); err != nil {
		t.Fatalf("resolve approval: %v", err)
	}

	select {
	case waitErr := <-errCh:
		t.Fatalf("wait returned error: %v", waitErr)
	case resolved := <-resultCh:
		if resolved.Status != StatusApproved {
			t.Fatalf("expected approved status, got %q", resolved.Status)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("wait did not return after resolve")
	}
}

func TestServiceWaitReturnsPromptlyAfterResolve(t *testing.T) {
	service := NewService(nil)
	created, err := service.Create(context.Background(), Request{ToolName: "exec"})
	if err != nil {
		t.Fatalf("create approval: %v", err)
	}

	resultCh := make(chan Request, 1)
	errCh := make(chan error, 1)
	go func() {
		resolved, waitErr := service.Wait(context.Background(), WaitRequest{ID: created.ID})
		if waitErr != nil {
			errCh <- waitErr
			return
		}
		resultCh <- resolved
	}()

	time.Sleep(10 * time.Millisecond)
	if _, err := service.Resolve(context.Background(), created.ID, "approve", "approved quickly"); err != nil {
		t.Fatalf("resolve approval: %v", err)
	}

	select {
	case waitErr := <-errCh:
		t.Fatalf("wait returned error: %v", waitErr)
	case resolved := <-resultCh:
		if resolved.Status != StatusApproved {
			t.Fatalf("expected approved status, got %q", resolved.Status)
		}
	case <-time.After(150 * time.Millisecond):
		t.Fatal("wait did not return promptly after resolve")
	}
}

func TestServiceWaitWithoutTimeoutUsesContextCancellation(t *testing.T) {
	service := NewService(nil)
	created, err := service.Create(context.Background(), Request{ToolName: "exec"})
	if err != nil {
		t.Fatalf("create approval: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
	defer cancel()
	_, err = service.Wait(ctx, WaitRequest{ID: created.ID})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context deadline exceeded, got %v", err)
	}

	item, ok := service.loadRequest(context.Background(), created.ID)
	if !ok {
		t.Fatal("expected approval request to remain available")
	}
	if item.Status != StatusPending {
		t.Fatalf("expected pending status after context timeout, got %q", item.Status)
	}
}

func TestServiceResolveIsIdempotentAfterFinalDecision(t *testing.T) {
	service := NewService(nil)
	created, err := service.Create(context.Background(), Request{ToolName: "exec"})
	if err != nil {
		t.Fatalf("create approval: %v", err)
	}

	first, err := service.Resolve(context.Background(), created.ID, "approve", "first")
	if err != nil {
		t.Fatalf("first resolve approval: %v", err)
	}
	if first.Status != StatusApproved {
		t.Fatalf("expected approved status on first resolve, got %q", first.Status)
	}

	second, err := service.Resolve(context.Background(), created.ID, "deny", "second")
	if err != nil {
		t.Fatalf("second resolve approval: %v", err)
	}
	if second.Status != StatusApproved {
		t.Fatalf("expected approval status to remain immutable, got %q", second.Status)
	}
	if second.Decision != first.Decision {
		t.Fatalf("expected decision to remain unchanged, got %q want %q", second.Decision, first.Decision)
	}
}

func TestServiceWaitRemovesCancelledWaiter(t *testing.T) {
	service := NewService(nil)
	created, err := service.Create(context.Background(), Request{ToolName: "exec"})
	if err != nil {
		t.Fatalf("create approval: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	_, err = service.Wait(ctx, WaitRequest{ID: created.ID})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context deadline exceeded, got %v", err)
	}

	service.mu.RLock()
	waiters := service.waiters[created.ID]
	service.mu.RUnlock()
	if len(waiters) != 0 {
		t.Fatalf("expected cancelled waiter to be removed, got %d", len(waiters))
	}
}

func TestServiceCleanupResolvedCacheRemovesExpiredItems(t *testing.T) {
	service := NewService(nil)
	base := time.Date(2026, time.March, 4, 5, 6, 7, 0, time.UTC)
	now := base
	service.now = func() time.Time { return now }
	service.resolvedCacheTTL = time.Minute

	first, err := service.Create(context.Background(), Request{ToolName: "exec"})
	if err != nil {
		t.Fatalf("create first approval: %v", err)
	}
	if _, err := service.Resolve(context.Background(), first.ID, "approve", "first"); err != nil {
		t.Fatalf("resolve first approval: %v", err)
	}

	now = base.Add(2 * time.Minute)
	second, err := service.Create(context.Background(), Request{ToolName: "exec"})
	if err != nil {
		t.Fatalf("create second approval: %v", err)
	}

	service.mu.RLock()
	_, firstExists := service.items[first.ID]
	secondItem, secondExists := service.items[second.ID]
	service.mu.RUnlock()
	if firstExists {
		t.Fatalf("expected expired resolved approval to be evicted from memory")
	}
	if !secondExists || secondItem.Status != StatusPending {
		t.Fatalf("expected newly created pending approval to remain cached")
	}
}
