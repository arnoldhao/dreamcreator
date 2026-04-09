package approvals

import (
	"context"
	"errors"
	"sort"
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
	mu               sync.RWMutex
	items            map[string]Request
	waiters          map[string][]chan Request
	store            Store
	events           EventPublisher
	now              func() time.Time
	resolvedCacheTTL time.Duration
	maxResolvedItems int
}

type Store interface {
	Save(ctx context.Context, request Request) error
	Get(ctx context.Context, id string) (Request, error)
	Update(ctx context.Context, request Request) error
}

type EventPublisher interface {
	Publish(ctx context.Context, eventType string, payload any) error
}

const defaultResolvedCacheTTL = 10 * time.Minute
const defaultMaxResolvedItems = 256

func NewService(store Store) *Service {
	return &Service{
		items:            make(map[string]Request),
		waiters:          make(map[string][]chan Request),
		store:            store,
		now:              time.Now,
		resolvedCacheTTL: defaultResolvedCacheTTL,
		maxResolvedItems: defaultMaxResolvedItems,
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
	service.cleanupResolvedLocked(now)
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
	item, ok := service.loadRequest(ctx, id)
	if !ok {
		return Request{}, errors.New("approval not found")
	}
	now := service.now()
	service.mu.Lock()
	service.cleanupResolvedLocked(now)
	if current, exists := service.items[id]; exists {
		item = current
	} else {
		service.items[id] = item
	}
	if item.Status != StatusPending {
		// Keep already-resolved approvals immutable so repeated button presses
		// cannot flip a prior decision.
		service.mu.Unlock()
		return item, nil
	}
	item.Decision = strings.TrimSpace(decision)
	item.Reason = strings.TrimSpace(reason)
	item.ResolvedAt = now
	if strings.EqualFold(item.Decision, "approve") || strings.EqualFold(item.Decision, "approved") {
		item.Status = StatusApproved
	} else {
		item.Status = StatusDenied
	}
	service.items[id] = item
	waiters := service.takeWaitersLocked(id)
	service.cleanupResolvedLocked(now)
	service.mu.Unlock()
	service.notifyWaiters(waiters, item)
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
	item, ok := service.loadRequest(ctx, id)
	if ok && item.Status != StatusPending {
		return item, nil
	}
	waiter := make(chan Request, 1)
	if item, ready := service.addWaiter(id, waiter); ready {
		return item, nil
	}
	defer service.removeWaiter(id, waiter)
	select {
	case <-ctx.Done():
		return Request{}, ctx.Err()
	case resolved := <-waiter:
		return resolved, nil
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
	service.cleanupResolvedLocked(service.now())
	service.items[id] = stored
	service.mu.Unlock()
	return stored, true
}

func (service *Service) addWaiter(id string, waiter chan Request) (Request, bool) {
	if service == nil {
		return Request{}, false
	}
	service.mu.Lock()
	defer service.mu.Unlock()
	service.cleanupResolvedLocked(service.now())
	if item, ok := service.items[id]; ok && item.Status != StatusPending {
		return item, true
	}
	service.waiters[id] = append(service.waiters[id], waiter)
	return Request{}, false
}

func (service *Service) removeWaiter(id string, waiter chan Request) {
	if service == nil {
		return
	}
	service.mu.Lock()
	defer service.mu.Unlock()
	waiters := service.waiters[id]
	for index, current := range waiters {
		if current != waiter {
			continue
		}
		service.waiters[id] = append(waiters[:index], waiters[index+1:]...)
		if len(service.waiters[id]) == 0 {
			delete(service.waiters, id)
		}
		return
	}
}

func (service *Service) takeWaitersLocked(id string) []chan Request {
	waiters := append([]chan Request(nil), service.waiters[id]...)
	delete(service.waiters, id)
	return waiters
}

func (service *Service) notifyWaiters(waiters []chan Request, item Request) {
	for _, waiter := range waiters {
		if waiter == nil {
			continue
		}
		select {
		case waiter <- item:
		default:
		}
	}
}

func (service *Service) cleanupResolvedLocked(now time.Time) {
	if service == nil {
		return
	}
	type resolvedEntry struct {
		id         string
		resolvedAt time.Time
	}
	resolved := make([]resolvedEntry, 0)
	cutoff := time.Time{}
	if service.resolvedCacheTTL > 0 {
		cutoff = now.Add(-service.resolvedCacheTTL)
	}
	for id, item := range service.items {
		if item.Status == StatusPending {
			continue
		}
		if !cutoff.IsZero() && !item.ResolvedAt.IsZero() && item.ResolvedAt.Before(cutoff) {
			delete(service.items, id)
			continue
		}
		resolved = append(resolved, resolvedEntry{id: id, resolvedAt: item.ResolvedAt})
	}
	if service.maxResolvedItems <= 0 || len(resolved) <= service.maxResolvedItems {
		return
	}
	sort.Slice(resolved, func(left int, right int) bool {
		if resolved[left].resolvedAt.Equal(resolved[right].resolvedAt) {
			return resolved[left].id < resolved[right].id
		}
		return resolved[left].resolvedAt.Before(resolved[right].resolvedAt)
	})
	excess := len(resolved) - service.maxResolvedItems
	for index := 0; index < excess; index++ {
		delete(service.items, resolved[index].id)
	}
}
