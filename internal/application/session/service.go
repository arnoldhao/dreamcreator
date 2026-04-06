package session

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	domainsession "dreamcreator/internal/domain/session"
)

var ErrSessionNotFound = errors.New("session not found")

type Store interface {
	Save(ctx context.Context, entry domainsession.Entry) error
	Get(ctx context.Context, sessionID string) (domainsession.Entry, error)
	List(ctx context.Context) ([]domainsession.Entry, error)
}

type InMemoryStore struct {
	mu      sync.RWMutex
	entries map[string]domainsession.Entry
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{entries: make(map[string]domainsession.Entry)}
}

func (store *InMemoryStore) Save(_ context.Context, entry domainsession.Entry) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.entries[entry.SessionID] = entry
	return nil
}

func (store *InMemoryStore) Get(_ context.Context, sessionID string) (domainsession.Entry, error) {
	store.mu.RLock()
	defer store.mu.RUnlock()
	entry, ok := store.entries[sessionID]
	if !ok {
		return domainsession.Entry{}, ErrSessionNotFound
	}
	return entry, nil
}

func (store *InMemoryStore) List(_ context.Context) ([]domainsession.Entry, error) {
	store.mu.RLock()
	defer store.mu.RUnlock()
	result := make([]domainsession.Entry, 0, len(store.entries))
	for _, entry := range store.entries {
		result = append(result, entry)
	}
	return result, nil
}

type Service struct {
	store Store
	now   func() time.Time
	newID func() string
}

type CreateSessionRequest struct {
	SessionID   string
	SessionKey  string
	KeyParts    domainsession.KeyParts
	AgentID     string
	AssistantID string
	Title       string
	Origin      domainsession.Origin
}

type ContextSnapshotUpdate struct {
	PromptTokens int
	TotalTokens  int
	WindowTokens int
	UpdatedAt    time.Time
	Fresh        bool
}

func NewService(store Store) *Service {
	if store == nil {
		store = NewInMemoryStore()
	}
	return &Service{
		store: store,
		now:   time.Now,
		newID: uuid.NewString,
	}
}

func (service *Service) CreateSession(ctx context.Context, request CreateSessionRequest) (domainsession.Entry, error) {
	sessionID := strings.TrimSpace(request.SessionID)
	if sessionID == "" {
		sessionID = service.newID()
	}
	key := strings.TrimSpace(request.SessionKey)
	if key == "" {
		parts := request.KeyParts
		parts.AgentID = strings.TrimSpace(parts.AgentID)
		if parts.ThreadRef == "" {
			parts.ThreadRef = sessionID
		}
		built, err := domainsession.BuildSessionKey(parts)
		if err != nil {
			return domainsession.Entry{}, err
		}
		key = built
	} else {
		_, normalized, err := domainsession.NormalizeSessionKey(key)
		if err != nil {
			return domainsession.Entry{}, err
		}
		key = normalized
	}
	now := service.now()
	if strings.TrimSpace(request.Origin.ThreadRef) == "" {
		request.Origin.ThreadRef = sessionID
	}
	entry := domainsession.Entry{
		SessionID:   sessionID,
		SessionKey:  key,
		AgentID:     strings.TrimSpace(request.AgentID),
		AssistantID: strings.TrimSpace(request.AssistantID),
		Title:       strings.TrimSpace(request.Title),
		Status:      domainsession.StatusActive,
		Origin:      request.Origin,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := service.store.Save(ctx, entry); err != nil {
		return domainsession.Entry{}, err
	}
	return entry, nil
}

func (service *Service) UpdateTitle(ctx context.Context, sessionID, title string) error {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return ErrSessionNotFound
	}
	entry, err := service.store.Get(ctx, sessionID)
	if err != nil {
		return err
	}
	entry.Title = strings.TrimSpace(title)
	entry.UpdatedAt = service.now()
	return service.store.Save(ctx, entry)
}

func (service *Service) SetStatus(ctx context.Context, sessionID string, status domainsession.Status) error {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return ErrSessionNotFound
	}
	entry, err := service.store.Get(ctx, sessionID)
	if err != nil {
		return err
	}
	entry.Status = status
	entry.UpdatedAt = service.now()
	return service.store.Save(ctx, entry)
}

func (service *Service) Get(ctx context.Context, sessionID string) (domainsession.Entry, error) {
	if service == nil || service.store == nil {
		return domainsession.Entry{}, ErrSessionNotFound
	}
	trimmed := strings.TrimSpace(sessionID)
	if trimmed == "" {
		return domainsession.Entry{}, ErrSessionNotFound
	}
	return service.store.Get(ctx, trimmed)
}

func (service *Service) UpdateCompactionCounters(ctx context.Context, sessionID string, compactionCount int, memoryFlushCompactionCount int) error {
	if service == nil || service.store == nil {
		return ErrSessionNotFound
	}
	trimmed := strings.TrimSpace(sessionID)
	if trimmed == "" {
		return ErrSessionNotFound
	}
	entry, err := service.store.Get(ctx, trimmed)
	if err != nil {
		return err
	}
	if compactionCount < 0 {
		compactionCount = 0
	}
	if memoryFlushCompactionCount < 0 {
		memoryFlushCompactionCount = 0
	}
	entry.CompactionCount = compactionCount
	entry.MemoryFlushCompactionCount = memoryFlushCompactionCount
	entry.UpdatedAt = service.now()
	return service.store.Save(ctx, entry)
}

func (service *Service) UpdateContextSnapshot(ctx context.Context, sessionID string, update ContextSnapshotUpdate) error {
	if service == nil || service.store == nil {
		return ErrSessionNotFound
	}
	trimmed := strings.TrimSpace(sessionID)
	if trimmed == "" {
		return ErrSessionNotFound
	}
	entry, err := service.store.Get(ctx, trimmed)
	if err != nil {
		return err
	}
	if update.PromptTokens < 0 {
		update.PromptTokens = 0
	}
	if update.TotalTokens < 0 {
		update.TotalTokens = 0
	}
	if update.WindowTokens < 0 {
		update.WindowTokens = 0
	}
	if update.UpdatedAt.IsZero() {
		update.UpdatedAt = service.now()
	}
	entry.ContextPromptTokens = update.PromptTokens
	entry.ContextTotalTokens = update.TotalTokens
	entry.ContextWindowTokens = update.WindowTokens
	entry.ContextUpdatedAt = update.UpdatedAt
	entry.ContextFresh = update.Fresh
	entry.UpdatedAt = service.now()
	return service.store.Save(ctx, entry)
}
