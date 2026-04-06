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
