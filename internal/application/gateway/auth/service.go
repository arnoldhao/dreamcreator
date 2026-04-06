package auth

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"
)

var (
	ErrUnauthorized = errors.New("unauthorized")
)

type Credentials struct {
	Token    string
	Password string
}

type AuthContext struct {
	Subject   string    `json:"subject"`
	Role      string    `json:"role"`
	Scopes    []string  `json:"scopes"`
	AuthType  string    `json:"authType"`
	IssuedAt  time.Time `json:"issuedAt"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type ScopeCheckResult struct {
	Method         string   `json:"method"`
	Allowed        bool     `json:"allowed"`
	RequiredScopes []string `json:"requiredScopes"`
	Reason         string   `json:"reason,omitempty"`
}

type Service interface {
	Authenticate(ctx context.Context, creds Credentials, role string, requestedScopes []string) (AuthContext, error)
}

type ScopeGuard interface {
	Check(method string, ctx AuthContext, requiredScopes []string) ScopeCheckResult
}

type InMemoryService struct {
	mu             sync.RWMutex
	tokens         map[string]AuthContext
	allowAnonymous bool
}

func NewInMemoryService() *InMemoryService {
	return &InMemoryService{
		tokens: make(map[string]AuthContext),
	}
}

func (service *InMemoryService) AllowAnonymous(enabled bool) {
	service.mu.Lock()
	defer service.mu.Unlock()
	service.allowAnonymous = enabled
}

func (service *InMemoryService) AddToken(token string, ctx AuthContext) {
	service.mu.Lock()
	defer service.mu.Unlock()
	token = strings.TrimSpace(token)
	if token == "" {
		return
	}
	if ctx.IssuedAt.IsZero() {
		ctx.IssuedAt = time.Now()
	}
	service.tokens[token] = ctx
}

func (service *InMemoryService) Authenticate(_ context.Context, creds Credentials, role string, requestedScopes []string) (AuthContext, error) {
	token := strings.TrimSpace(creds.Token)
	service.mu.RLock()
	allowAnonymous := service.allowAnonymous
	service.mu.RUnlock()

	if token == "" {
		if !allowAnonymous {
			return AuthContext{}, ErrUnauthorized
		}
		return AuthContext{
			Subject:  "anonymous",
			Role:     strings.TrimSpace(role),
			Scopes:   normalizeScopes(requestedScopes),
			AuthType: "anonymous",
			IssuedAt: time.Now(),
		}, nil
	}

	service.mu.RLock()
	ctx, ok := service.tokens[token]
	service.mu.RUnlock()
	if !ok {
		return AuthContext{}, ErrUnauthorized
	}
	if ctx.Role == "" {
		ctx.Role = strings.TrimSpace(role)
	}
	if len(ctx.Scopes) == 0 {
		ctx.Scopes = normalizeScopes(requestedScopes)
	}
	if ctx.AuthType == "" {
		ctx.AuthType = "token"
	}
	if ctx.IssuedAt.IsZero() {
		ctx.IssuedAt = time.Now()
	}
	return ctx, nil
}

type DefaultScopeGuard struct{}

func NewDefaultScopeGuard() *DefaultScopeGuard {
	return &DefaultScopeGuard{}
}

func (guard *DefaultScopeGuard) Check(method string, ctx AuthContext, requiredScopes []string) ScopeCheckResult {
	requiredScopes = normalizeScopes(requiredScopes)
	if len(requiredScopes) == 0 {
		return ScopeCheckResult{
			Method:         method,
			Allowed:        false,
			RequiredScopes: requiredScopes,
			Reason:         "no scopes registered",
		}
	}
	if len(ctx.Scopes) == 0 {
		return ScopeCheckResult{
			Method:         method,
			Allowed:        false,
			RequiredScopes: requiredScopes,
			Reason:         "missing scopes",
		}
	}
	scopeSet := make(map[string]struct{}, len(ctx.Scopes))
	for _, scope := range ctx.Scopes {
		scopeSet[scope] = struct{}{}
	}
	for _, scope := range requiredScopes {
		if _, ok := scopeSet[scope]; !ok {
			return ScopeCheckResult{
				Method:         method,
				Allowed:        false,
				RequiredScopes: requiredScopes,
				Reason:         "scope not granted",
			}
		}
	}
	return ScopeCheckResult{
		Method:         method,
		Allowed:        true,
		RequiredScopes: requiredScopes,
	}
}

func normalizeScopes(scopes []string) []string {
	if len(scopes) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(scopes))
	result := make([]string, 0, len(scopes))
	for _, scope := range scopes {
		trimmed := strings.TrimSpace(scope)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}
