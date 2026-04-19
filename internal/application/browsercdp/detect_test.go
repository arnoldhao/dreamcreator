package browsercdp

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestWaitForCDPHonorsCancelledContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := WaitForCDP(ctx, "127.0.0.1", 1, time.Second)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled, got %v", err)
	}
}
