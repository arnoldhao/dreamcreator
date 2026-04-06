package channels

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	"dreamcreator/internal/application/gateway/controlplane"
)

type ChannelDescriptor struct {
	ChannelID    string   `json:"channelId"`
	DisplayName  string   `json:"displayName"`
	Kind         string   `json:"kind"`
	Capabilities []string `json:"capabilities,omitempty"`
	Enabled      bool     `json:"enabled"`
}

type ChannelStatus struct {
	ChannelID string    `json:"channelId"`
	State     string    `json:"state"`
	AccountID string    `json:"accountId,omitempty"`
	LatencyMs int       `json:"latencyMs,omitempty"`
	LastError string    `json:"lastError,omitempty"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type ChannelOverview struct {
	ChannelID    string    `json:"channelId"`
	DisplayName  string    `json:"displayName"`
	Kind         string    `json:"kind"`
	Capabilities []string  `json:"capabilities,omitempty"`
	Enabled      bool      `json:"enabled"`
	State        string    `json:"state"`
	AccountID    string    `json:"accountId,omitempty"`
	LatencyMs    int       `json:"latencyMs,omitempty"`
	LastError    string    `json:"lastError,omitempty"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type ChannelPresenceEvent struct {
	ChannelID string    `json:"channelId"`
	State     string    `json:"state"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type ChannelProbeResult struct {
	ChannelID string    `json:"channelId"`
	State     string    `json:"state,omitempty"`
	Success   bool      `json:"success"`
	Error     string    `json:"error,omitempty"`
	CheckedAt time.Time `json:"checkedAt"`
}

type ChannelLogoutResult struct {
	ChannelID string `json:"channelId"`
	Success   bool   `json:"success"`
	Error     string `json:"error,omitempty"`
}

type WebhookIngestRequest struct {
	ChannelID  string `json:"channelId"`
	SessionKey string `json:"sessionKey,omitempty"`
	EventID    string `json:"eventId,omitempty"`
	Payload    string `json:"payload,omitempty"`
}

type WebhookIngestResult struct {
	EventID   string    `json:"eventId"`
	SessionKey string   `json:"sessionKey,omitempty"`
	Queued    bool      `json:"queued"`
	QueuedAt  time.Time `json:"queuedAt"`
}

type ChannelHandlers struct {
	Status  func(ctx context.Context) (ChannelStatus, error)
	Logout  func(ctx context.Context) error
	Probe   func(ctx context.Context) (ChannelProbeResult, error)
	Webhook func(ctx context.Context, request WebhookIngestRequest) (WebhookIngestResult, error)
}

type Registry struct {
	mu        sync.RWMutex
	entries   map[string]ChannelDescriptor
	handlers  map[string]ChannelHandlers
	statuses  map[string]ChannelStatus
	publisher controlplane.EventPublisher
	now       func() time.Time
}

func NewRegistry(publisher controlplane.EventPublisher) *Registry {
	return &Registry{
		entries:   make(map[string]ChannelDescriptor),
		handlers:  make(map[string]ChannelHandlers),
		statuses:  make(map[string]ChannelStatus),
		publisher: publisher,
		now:       time.Now,
	}
}

func (registry *Registry) Register(descriptor ChannelDescriptor, handlers ChannelHandlers) {
	if registry == nil {
		return
	}
	id := strings.TrimSpace(descriptor.ChannelID)
	if id == "" {
		return
	}
	registry.mu.Lock()
	registry.entries[id] = descriptor
	registry.handlers[id] = handlers
	registry.mu.Unlock()
}

func (registry *Registry) StatusAll(ctx context.Context) ([]ChannelStatus, error) {
	if registry == nil {
		return nil, errors.New("registry unavailable")
	}
	registry.mu.RLock()
	entries := make([]ChannelDescriptor, 0, len(registry.entries))
	for _, item := range registry.entries {
		entries = append(entries, item)
	}
	registry.mu.RUnlock()

	result := make([]ChannelStatus, 0, len(entries))
	for _, descriptor := range entries {
		status, err := registry.statusFor(ctx, descriptor.ChannelID)
		if err != nil {
			status = ChannelStatus{
				ChannelID: descriptor.ChannelID,
				State:     "unknown",
				UpdatedAt: registry.now(),
				LastError: err.Error(),
			}
		}
		result = append(result, status)
	}
	return result, nil
}

func (registry *Registry) List(ctx context.Context) ([]ChannelOverview, error) {
	if registry == nil {
		return nil, errors.New("registry unavailable")
	}
	registry.mu.RLock()
	entries := make([]ChannelDescriptor, 0, len(registry.entries))
	for _, item := range registry.entries {
		entries = append(entries, item)
	}
	registry.mu.RUnlock()

	sort.Slice(entries, func(i, j int) bool {
		left := strings.ToLower(strings.TrimSpace(entries[i].DisplayName))
		right := strings.ToLower(strings.TrimSpace(entries[j].DisplayName))
		if left == "" {
			left = strings.ToLower(entries[i].ChannelID)
		}
		if right == "" {
			right = strings.ToLower(entries[j].ChannelID)
		}
		if left == right {
			return entries[i].ChannelID < entries[j].ChannelID
		}
		return left < right
	})

	result := make([]ChannelOverview, 0, len(entries))
	for _, descriptor := range entries {
		status, err := registry.statusFor(ctx, descriptor.ChannelID)
		if err != nil && status.LastError == "" {
			status.LastError = err.Error()
		}
		if status.ChannelID == "" {
			status.ChannelID = descriptor.ChannelID
		}
		if status.UpdatedAt.IsZero() {
			status.UpdatedAt = registry.now()
		}
		if strings.TrimSpace(status.State) == "" {
			status.State = "unknown"
		}
		result = append(result, ChannelOverview{
			ChannelID:    descriptor.ChannelID,
			DisplayName:  descriptor.DisplayName,
			Kind:         descriptor.Kind,
			Capabilities: descriptor.Capabilities,
			Enabled:      descriptor.Enabled,
			State:        status.State,
			AccountID:    status.AccountID,
			LatencyMs:    status.LatencyMs,
			LastError:    status.LastError,
			UpdatedAt:    status.UpdatedAt,
		})
	}
	return result, nil
}

func (registry *Registry) Logout(ctx context.Context, channelID string) (ChannelLogoutResult, error) {
	handler, ok := registry.handler(channelID)
	if !ok || handler.Logout == nil {
		return ChannelLogoutResult{ChannelID: channelID, Success: false, Error: "logout not supported"}, nil
	}
	if err := handler.Logout(ctx); err != nil {
		return ChannelLogoutResult{ChannelID: channelID, Success: false, Error: err.Error()}, err
	}
	return ChannelLogoutResult{ChannelID: channelID, Success: true}, nil
}

func (registry *Registry) Probe(ctx context.Context, channelID string) (ChannelProbeResult, error) {
	handler, ok := registry.handler(channelID)
	if !ok || handler.Probe == nil {
		return ChannelProbeResult{ChannelID: channelID, Success: false, Error: "probe not supported", CheckedAt: registry.now()}, nil
	}
	result, err := handler.Probe(ctx)
	if err != nil {
		result.Error = err.Error()
		result.Success = false
	}
	if result.CheckedAt.IsZero() {
		result.CheckedAt = registry.now()
	}
	return result, err
}

func (registry *Registry) IngestWebhook(ctx context.Context, channelID string, request WebhookIngestRequest) (WebhookIngestResult, error) {
	handler, ok := registry.handler(channelID)
	if !ok || handler.Webhook == nil {
		return WebhookIngestResult{EventID: request.EventID, Queued: false}, errors.New("webhook not supported")
	}
	return handler.Webhook(ctx, request)
}

func (registry *Registry) handler(channelID string) (ChannelHandlers, bool) {
	if registry == nil {
		return ChannelHandlers{}, false
	}
	id := strings.TrimSpace(channelID)
	registry.mu.RLock()
	handler, ok := registry.handlers[id]
	registry.mu.RUnlock()
	return handler, ok
}

func (registry *Registry) statusFor(ctx context.Context, channelID string) (ChannelStatus, error) {
	handler, ok := registry.handler(channelID)
	if !ok || handler.Status == nil {
		return ChannelStatus{ChannelID: channelID, State: "unknown", UpdatedAt: registry.now()}, nil
	}
	status, err := handler.Status(ctx)
	if status.ChannelID == "" {
		status.ChannelID = channelID
	}
	if status.UpdatedAt.IsZero() {
		status.UpdatedAt = registry.now()
	}
	registry.trackStatus(status)
	return status, err
}

func (registry *Registry) trackStatus(status ChannelStatus) {
	if registry == nil {
		return
	}
	registry.mu.Lock()
	prev, ok := registry.statuses[status.ChannelID]
	registry.statuses[status.ChannelID] = status
	registry.mu.Unlock()
	if !ok || prev.State != status.State {
		registry.publishPresence(status)
	}
}

func (registry *Registry) publishPresence(status ChannelStatus) {
	if registry == nil || registry.publisher == nil {
		return
	}
	_ = registry.publisher.Publish(controlplane.EventFrame{
		Type:      "event",
		Event:     "channel.presence.changed",
		Payload:   ChannelPresenceEvent{ChannelID: status.ChannelID, State: status.State, UpdatedAt: status.UpdatedAt},
		Timestamp: registry.now(),
	})
}
