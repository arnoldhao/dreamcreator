package agentruntime

import (
	"context"
	"testing"
	"time"
)

func TestAgentControllerQueuesAndAbort(t *testing.T) {
	controller := NewAgentController()
	controller.Steer("  first ")
	controller.FollowUp(" second ")

	if message, ok := controller.NextSteer(); !ok || message != "first" {
		t.Fatalf("expected steer queue value, got %q %v", message, ok)
	}
	if message, ok := controller.NextFollowUp(); !ok || message != "second" {
		t.Fatalf("expected follow-up queue value, got %q %v", message, ok)
	}

	controller.Abort("cancel")
	if reason, aborted := controller.Aborted(); !aborted || reason != "cancel" {
		t.Fatalf("expected aborted reason, got %q %v", reason, aborted)
	}
	controller.ResetAbort()
	if _, aborted := controller.Aborted(); aborted {
		t.Fatalf("expected abort state reset")
	}
}

func TestAgentControllerWaitForIdle(t *testing.T) {
	controller := NewAgentController()
	controller.BeginRun()

	done := make(chan struct{})
	go func() {
		defer close(done)
		time.Sleep(10 * time.Millisecond)
		controller.EndRun()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := controller.WaitForIdle(ctx); err != nil {
		t.Fatalf("wait for idle failed: %v", err)
	}
	<-done
}
