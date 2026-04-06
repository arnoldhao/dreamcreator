package sandbox

import (
	"context"
	"strings"
	"time"

	"dreamcreator/internal/infrastructure/secure"
)

type Resolution struct {
	Allowed          bool   `json:"allowed"`
	Mode             string `json:"mode,omitempty"`
	WorkspaceAccess  string `json:"workspaceAccess,omitempty"`
	SandboxID        string `json:"sandboxId,omitempty"`
	Reason           string `json:"reason,omitempty"`
	ApprovalRequired bool   `json:"approvalRequired,omitempty"`
}

type ResolveRequest struct {
	RequiresSandbox bool   `json:"requiresSandbox,omitempty"`
	WorkspacePath   string `json:"workspacePath,omitempty"`
}

type HealthStatus struct {
	Available bool      `json:"available"`
	Message   string    `json:"message,omitempty"`
	CheckedAt time.Time `json:"checkedAt"`
}

type Service struct {
	enabled bool
	health  secure.HealthChecker
	now     func() time.Time
}

func NewService(enabled bool, healthChecker secure.HealthChecker) *Service {
	if healthChecker == nil {
		healthChecker = secure.NoopHealthChecker{}
	}
	return &Service{
		enabled: enabled,
		health:  healthChecker,
		now:     time.Now,
	}
}

func (service *Service) Resolve(ctx context.Context, request ResolveRequest) (Resolution, error) {
	if !request.RequiresSandbox {
		return Resolution{Allowed: true, Mode: "off"}, nil
	}
	if !service.enabled {
		return Resolution{Allowed: false, Mode: "off", Reason: "sandbox_unavailable"}, nil
	}
	status := service.health.Check(ctx)
	if !status.Available {
		reason := strings.TrimSpace(status.Message)
		if reason == "" {
			reason = "sandbox_unavailable"
		}
		return Resolution{Allowed: false, Mode: "off", Reason: reason}, nil
	}
	workspacePath := strings.TrimSpace(request.WorkspacePath)
	if workspacePath == "" {
		workspacePath = "/"
	}
	return Resolution{
		Allowed:         true,
		Mode:            "local",
		WorkspaceAccess: "rw",
		SandboxID:       "local",
	}, nil
}

func (service *Service) Health(ctx context.Context) HealthStatus {
	status := service.health.Check(ctx)
	return HealthStatus{
		Available: status.Available,
		Message:   status.Message,
		CheckedAt: status.CheckedAt,
	}
}
