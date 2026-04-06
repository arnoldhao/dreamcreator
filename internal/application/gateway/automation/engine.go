package automation

import (
	"context"
	"errors"
	"strings"
	"time"

	gatewayevents "dreamcreator/internal/application/gateway/events"
	"dreamcreator/internal/application/gateway/queue"
	domainsession "dreamcreator/internal/domain/session"

	"github.com/google/uuid"
)

type Engine struct {
	queue    *queue.Manager
	store    Store
	events   *gatewayevents.Broker
	registry *TriggerRegistry
	now      func() time.Time
	newID    func() string
}

func NewEngine(queueManager *queue.Manager, store Store, events *gatewayevents.Broker) *Engine {
	return &Engine{
		queue:    queueManager,
		store:    store,
		events:   events,
		registry: NewTriggerRegistry(),
		now:      time.Now,
		newID:    uuid.NewString,
	}
}

func (engine *Engine) Trigger(ctx context.Context, action AutomationAction) (queue.Ticket, error) {
	if engine == nil || engine.queue == nil {
		return queue.Ticket{}, errors.New("queue manager unavailable")
	}
	sessionKey := strings.TrimSpace(action.SessionKey)
	if sessionKey == "" {
		return queue.Ticket{}, errors.New("session key is required")
	}
	if _, _, err := domainsession.NormalizeSessionKey(sessionKey); err != nil {
		if fallback, buildErr := domainsession.BuildSessionKey(domainsession.KeyParts{
			Channel:   "system",
			PrimaryID: sessionKey,
			ThreadRef: sessionKey,
		}); buildErr == nil {
			sessionKey = fallback
		}
	}
	jobID := resolveJobID(action)
	now := engine.now()
	run := RunRecord{
		ID:        engine.newID(),
		JobID:     jobID,
		Status:    "queued",
		StartedAt: now,
	}
	if engine.store != nil {
		_ = engine.store.SaveJob(ctx, JobRecord{
			ID:        jobID,
			Kind:      resolveJobKind(action),
			Status:    "enabled",
			Config:    action,
			CreatedAt: now,
			UpdatedAt: now,
		})
		_ = engine.store.SaveRun(ctx, run)
	}
	engine.publishEvent(ctx, "automation.triggered", sessionKey, run)
	ticket, _, err := engine.queue.Enqueue(ctx, queue.EnqueueRequest{
		SessionKey: sessionKey,
		Mode:       "",
		Payload:    action,
	})
	if err != nil {
		run.Status = "failed"
		run.Error = err.Error()
		run.EndedAt = engine.now()
		if engine.store != nil {
			_ = engine.store.SaveRun(ctx, run)
		}
		engine.publishEvent(ctx, "automation.failed", sessionKey, run)
		return queue.Ticket{}, err
	}
	engine.publishEvent(ctx, "automation.queued", sessionKey, run)
	return ticket, nil
}

func (engine *Engine) RegisterTrigger(id string, fn TriggerFunc) {
	if engine == nil || engine.registry == nil {
		return
	}
	engine.registry.Register(id, fn)
}

func (engine *Engine) TriggerRegistered(ctx context.Context, id string, action AutomationAction) error {
	if engine == nil || engine.registry == nil {
		return errors.New("trigger registry unavailable")
	}
	return engine.registry.Trigger(ctx, id, action)
}

func (engine *Engine) publishEvent(ctx context.Context, eventType string, sessionKey string, payload any) {
	if engine == nil || engine.events == nil {
		return
	}
	envelope := gatewayevents.Envelope{
		Type:       eventType,
		Topic:      "automation",
		SessionKey: strings.TrimSpace(sessionKey),
		Timestamp:  engine.now(),
	}
	if envelope.SessionKey != "" {
		if parts, _, err := domainsession.NormalizeSessionKey(envelope.SessionKey); err == nil {
			envelope.SessionID = strings.TrimSpace(parts.ThreadRef)
		}
	}
	_, _ = engine.events.Publish(ctx, envelope, payload)
}

func resolveJobID(action AutomationAction) string {
	if action.Payload != nil {
		if payload, ok := action.Payload.(map[string]any); ok {
			if id := getString(payload, "jobId", "jobID", "id"); id != "" {
				return id
			}
		}
	}
	if strings.TrimSpace(action.Type) != "" {
		return "auto:" + strings.TrimSpace(action.Type)
	}
	return "auto:trigger"
}

func resolveJobKind(action AutomationAction) string {
	actionType := strings.ToLower(strings.TrimSpace(action.Type))
	switch {
	case strings.HasPrefix(actionType, "cron"):
		return "cron"
	case strings.HasPrefix(actionType, "heartbeat"):
		return "heartbeat"
	case strings.HasPrefix(actionType, "run."):
		return "hook"
	default:
		return "automation"
	}
}

func getString(values map[string]any, keys ...string) string {
	if values == nil {
		return ""
	}
	for _, key := range keys {
		if raw, ok := values[key]; ok {
			if str, ok := raw.(string); ok {
				str = strings.TrimSpace(str)
				if str != "" {
					return str
				}
			}
		}
	}
	return ""
}
