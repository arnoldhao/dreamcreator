package pairing

import (
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

var (
	ErrPairRequestNotFound = errors.New("pair request not found")
	ErrTokenNotFound       = errors.New("token not found")
)

type PairStatus string

const (
	PairStatusPending  PairStatus = "pending"
	PairStatusApproved PairStatus = "approved"
	PairStatusRejected PairStatus = "rejected"
)

type PairRequest struct {
	ID        string
	NodeID    string
	Status    PairStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}

type DeviceToken struct {
	ID        string
	NodeID    string
	IssuedAt  time.Time
	ExpiresAt time.Time
	RevokedAt time.Time
}

type Service struct {
	mu       sync.Mutex
	requests map[string]PairRequest
	tokens   map[string]DeviceToken
	now      func() time.Time
}

func NewService() *Service {
	return &Service{
		requests: make(map[string]PairRequest),
		tokens:   make(map[string]DeviceToken),
		now:      time.Now,
	}
}

func (service *Service) Request(nodeID string) (PairRequest, error) {
	nodeID = strings.TrimSpace(nodeID)
	if nodeID == "" {
		return PairRequest{}, errors.New("node id is required")
	}
	now := service.now()
	req := PairRequest{
		ID:        uuid.NewString(),
		NodeID:    nodeID,
		Status:    PairStatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}
	service.mu.Lock()
	service.requests[req.ID] = req
	service.mu.Unlock()
	return req, nil
}

func (service *Service) Approve(requestID string) (PairRequest, error) {
	return service.updateStatus(requestID, PairStatusApproved)
}

func (service *Service) Reject(requestID string) (PairRequest, error) {
	return service.updateStatus(requestID, PairStatusRejected)
}

func (service *Service) updateStatus(requestID string, status PairStatus) (PairRequest, error) {
	requestID = strings.TrimSpace(requestID)
	if requestID == "" {
		return PairRequest{}, ErrPairRequestNotFound
	}
	service.mu.Lock()
	defer service.mu.Unlock()
	req, ok := service.requests[requestID]
	if !ok {
		return PairRequest{}, ErrPairRequestNotFound
	}
	req.Status = status
	req.UpdatedAt = service.now()
	service.requests[requestID] = req
	return req, nil
}

func (service *Service) IssueToken(nodeID string, ttl time.Duration) (DeviceToken, error) {
	nodeID = strings.TrimSpace(nodeID)
	if nodeID == "" {
		return DeviceToken{}, errors.New("node id is required")
	}
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	now := service.now()
	token := DeviceToken{
		ID:        uuid.NewString(),
		NodeID:    nodeID,
		IssuedAt:  now,
		ExpiresAt: now.Add(ttl),
	}
	service.mu.Lock()
	service.tokens[token.ID] = token
	service.mu.Unlock()
	return token, nil
}

func (service *Service) RotateToken(tokenID string, ttl time.Duration) (DeviceToken, error) {
	tokenID = strings.TrimSpace(tokenID)
	if tokenID == "" {
		return DeviceToken{}, ErrTokenNotFound
	}
	service.mu.Lock()
	token, ok := service.tokens[tokenID]
	if ok {
		token.RevokedAt = service.now()
		service.tokens[tokenID] = token
	}
	service.mu.Unlock()
	if !ok {
		return DeviceToken{}, ErrTokenNotFound
	}
	return service.IssueToken(token.NodeID, ttl)
}

func (service *Service) RevokeToken(tokenID string) (DeviceToken, error) {
	tokenID = strings.TrimSpace(tokenID)
	if tokenID == "" {
		return DeviceToken{}, ErrTokenNotFound
	}
	service.mu.Lock()
	defer service.mu.Unlock()
	token, ok := service.tokens[tokenID]
	if !ok {
		return DeviceToken{}, ErrTokenNotFound
	}
	token.RevokedAt = service.now()
	service.tokens[tokenID] = token
	return token, nil
}

func (service *Service) ValidateToken(tokenID string) bool {
	tokenID = strings.TrimSpace(tokenID)
	if tokenID == "" {
		return false
	}
	service.mu.Lock()
	defer service.mu.Unlock()
	token, ok := service.tokens[tokenID]
	if !ok {
		return false
	}
	if !token.RevokedAt.IsZero() {
		return false
	}
	if !token.ExpiresAt.IsZero() && service.now().After(token.ExpiresAt) {
		return false
	}
	return true
}

func (service *Service) GetToken(tokenID string) (DeviceToken, error) {
	service.mu.Lock()
	defer service.mu.Unlock()
	token, ok := service.tokens[tokenID]
	if !ok {
		return DeviceToken{}, ErrTokenNotFound
	}
	return token, nil
}
