package automation

import (
	"context"
	"time"

	gatewayheartbeat "dreamcreator/internal/application/gateway/heartbeat"
	sessionmanager "dreamcreator/internal/application/session"
	domainsession "dreamcreator/internal/domain/session"
	"dreamcreator/internal/domain/thread"
)

const defaultHeartbeatInterval = 5 * time.Second

type HeartbeatRunner struct {
	sessions   *sessionmanager.Manager
	runs       thread.RunRepository
	automation *Engine
	heartbeat  *gatewayheartbeat.Service
	interval   time.Duration
	now        func() time.Time
}

type HeartbeatPayload struct {
	Timestamp  string                       `json:"timestamp"`
	Status     string                       `json:"status"`
	ActiveRuns int                          `json:"activeRuns"`
	Queue      sessionmanager.QueueSnapshot `json:"queue"`
}

func NewHeartbeatRunner(sessions *sessionmanager.Manager, runs thread.RunRepository) *HeartbeatRunner {
	return &HeartbeatRunner{
		sessions: sessions,
		runs:     runs,
		interval: defaultHeartbeatInterval,
		now:      time.Now,
	}
}

func (runner *HeartbeatRunner) ConfigureAutomation(engine *Engine) {
	if runner == nil {
		return
	}
	runner.automation = engine
}

func (runner *HeartbeatRunner) SetHeartbeatService(service *gatewayheartbeat.Service) {
	if runner == nil {
		return
	}
	runner.heartbeat = service
}

func (runner *HeartbeatRunner) Start(ctx context.Context) func() {
	if runner == nil {
		return func() {}
	}
	stop := make(chan struct{})
	ticker := time.NewTicker(runner.interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-stop:
				return
			case <-ticker.C:
				runner.publish(ctx)
			}
		}
	}()
	return func() { close(stop) }
}

func (runner *HeartbeatRunner) publish(ctx context.Context) {
	if runner.heartbeat != nil {
		runner.heartbeat.Tick(ctx)
	}
	payload := HeartbeatPayload{
		Timestamp: runner.now().Format(time.RFC3339),
	}
	if runner.sessions != nil {
		payload.Queue = runner.sessions.Snapshot()
	}
	if runner.runs != nil {
		if count, err := runner.runs.CountActive(ctx); err == nil {
			payload.ActiveRuns = count
		}
	}
	payload.Status = resolveHeartbeatStatus(payload)
	if runner.automation != nil {
		sessionKey := "heartbeat"
		if key, err := domainsession.BuildSessionKey(domainsession.KeyParts{
			Channel:   "system",
			PrimaryID: sessionKey,
			ThreadRef: sessionKey,
		}); err == nil {
			sessionKey = key
		}
		_, _ = runner.automation.Trigger(ctx, AutomationAction{
			Type:       "heartbeat.tick",
			SessionKey: sessionKey,
			Payload:    payload,
		})
	}
}

func resolveHeartbeatStatus(payload HeartbeatPayload) string {
	if payload.ActiveRuns > 0 {
		return "busy"
	}
	if payload.Queue.ActiveSessions > 0 || payload.Queue.TotalSessions > 0 {
		return "busy"
	}
	return "idle"
}
