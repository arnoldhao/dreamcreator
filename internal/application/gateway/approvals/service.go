package approvals

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusPending  Status = "pending"
	StatusApproved Status = "approved"
	StatusDenied   Status = "denied"
)

type Request struct {
	ID          string    `json:"id"`
	SessionKey  string    `json:"sessionKey,omitempty"`
	ToolCallID  string    `json:"toolCallId,omitempty"`
	ToolName    string    `json:"toolName,omitempty"`
	Action      string    `json:"action,omitempty"`
	Args        string    `json:"args,omitempty"`
	Status      Status    `json:"status"`
	Decision    string    `json:"decision,omitempty"`
	Reason      string    `json:"reason,omitempty"`
	RequestedAt time.Time `json:"requestedAt"`
	ResolvedAt  time.Time `json:"resolvedAt,omitempty"`
}

type WaitRequest struct {
	ID string `json:"id"`
}

type Service struct {
	mu     sync.RWMutex
	items  map[string]Request
	store  Store
	events EventPublisher
	now    func() time.Time
}

type Store interface {
	Save(ctx context.Context, request Request) error
	Get(ctx context.Context, id string) (Request, error)
	Update(ctx context.Context, request Request) error
}

type EventPublisher interface {
	Publish(ctx context.Context, eventType string, payload any) error
}

func NewService(store Store) *Service {
	return &Service{
		items: make(map[string]Request),
		store: store,
		now:   time.Now,
	}
}

func (service *Service) Create(ctx context.Context, request Request) (Request, error) {
	if service == nil {
		return Request{}, errors.New("approval service unavailable")
	}
	id := strings.TrimSpace(request.ID)
	if id == "" {
		id = uuid.NewString()
	}
	now := service.now()
	request.ID = id
	request.Status = StatusPending
	request.RequestedAt = now
	service.mu.Lock()
	service.items[id] = request
	service.mu.Unlock()
	if service.store != nil {
		if err := service.store.Save(ctx, request); err != nil {
			return Request{}, err
		}
	}
	if service.events != nil {
		_ = service.events.Publish(ctx, "exec.approval.requested", request)
	}
	return request, nil
}

func (service *Service) Resolve(ctx context.Context, id string, decision string, reason string) (Request, error) {
	if service == nil {
		return Request{}, errors.New("approval service unavailable")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return Request{}, errors.New("approval id is required")
	}
	service.mu.Lock()
	defer service.mu.Unlock()
	item, ok := service.items[id]
	if !ok {
		if service.store != nil {
			if stored, err := service.store.Get(ctx, id); err == nil {
				item = stored
			} else {
				return Request{}, errors.New("approval not found")
			}
		} else {
			return Request{}, errors.New("approval not found")
		}
	}
	if item.Status != StatusPending {
		// Keep already-resolved approvals immutable so repeated button presses
		// cannot flip a prior decision.
		return item, nil
	}
	item.Decision = strings.TrimSpace(decision)
	item.Reason = strings.TrimSpace(reason)
	item.ResolvedAt = service.now()
	if strings.EqualFold(item.Decision, "approve") || strings.EqualFold(item.Decision, "approved") {
		item.Status = StatusApproved
	} else {
		item.Status = StatusDenied
	}
	service.items[id] = item
	if service.store != nil {
		if err := service.store.Update(ctx, item); err != nil {
			return Request{}, err
		}
	}
	if service.events != nil {
		_ = service.events.Publish(ctx, "exec.approval.resolved", item)
	}
	return item, nil
}

func (service *Service) Wait(ctx context.Context, request WaitRequest) (Request, error) {
	if service == nil {
		return Request{}, errors.New("approval service unavailable")
	}
	id := strings.TrimSpace(request.ID)
	if id == "" {
		return Request{}, errors.New("approval id is required")
	}
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	for {
		item, ok := service.loadRequest(ctx, id)
		if ok && item.Status != StatusPending {
			return item, nil
		}
		select {
		case <-ctx.Done():
			return Request{}, ctx.Err()
		case <-ticker.C:
		}
	}
}

func (service *Service) SetEventPublisher(publisher EventPublisher) {
	if service == nil {
		return
	}
	service.events = publisher
}

func (service *Service) loadRequest(ctx context.Context, id string) (Request, bool) {
	service.mu.RLock()
	item, ok := service.items[id]
	service.mu.RUnlock()
	if ok {
		return item, true
	}
	if service.store == nil {
		return Request{}, false
	}
	stored, err := service.store.Get(ctx, id)
	if err != nil {
		return Request{}, false
	}
	service.mu.Lock()
	service.items[id] = stored
	service.mu.Unlock()
	return stored, true
}
