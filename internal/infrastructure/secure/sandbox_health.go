package secure

import (
	"context"
	"time"
)

type SandboxHealthStatus struct {
	Available bool
	Message   string
	CheckedAt time.Time
}

type HealthChecker interface {
	Check(ctx context.Context) SandboxHealthStatus
}

type NoopHealthChecker struct{}

func (NoopHealthChecker) Check(_ context.Context) SandboxHealthStatus {
	return SandboxHealthStatus{
		Available: true,
		Message:   "ok",
		CheckedAt: time.Now(),
	}
}
