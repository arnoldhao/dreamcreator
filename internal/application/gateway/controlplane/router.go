package controlplane

import (
	"context"
	"strings"
	"sync"
	"time"

	"dreamcreator/internal/application/gateway/auth"
)

type MethodHandler func(ctx context.Context, session *SessionContext, params []byte) (any, *GatewayError)
type LockGuard func(method string, session *SessionContext) bool
type AuditHandler func(ctx context.Context, session *SessionContext, result auth.ScopeCheckResult)

type MethodSpec struct {
	Handler        MethodHandler
	RequiredScopes []string
}

type SessionContext struct {
	ID          string
	Role        string
	Scopes      []string
	Auth        auth.AuthContext
	ConnectedAt time.Time
}

type Router struct {
	mu         sync.RWMutex
	methods    map[string]MethodSpec
	scopeGuard auth.ScopeGuard
	lockGuard  LockGuard
	audit      AuditHandler
}

func NewRouter(scopeGuard auth.ScopeGuard) *Router {
	return &Router{
		methods:    make(map[string]MethodSpec),
		scopeGuard: scopeGuard,
	}
}

func (router *Router) SetLockGuard(guard LockGuard) {
	router.mu.Lock()
	defer router.mu.Unlock()
	router.lockGuard = guard
}

func (router *Router) SetAuditHandler(handler AuditHandler) {
	router.mu.Lock()
	defer router.mu.Unlock()
	router.audit = handler
}

func (router *Router) Register(method string, requiredScopes []string, handler MethodHandler) {
	method = strings.TrimSpace(method)
	if method == "" || handler == nil {
		return
	}
	router.mu.Lock()
	defer router.mu.Unlock()
	router.methods[method] = MethodSpec{
		Handler:        handler,
		RequiredScopes: requiredScopes,
	}
}

func (router *Router) Methods() []string {
	router.mu.RLock()
	defer router.mu.RUnlock()
	methods := make([]string, 0, len(router.methods))
	for method := range router.methods {
		methods = append(methods, method)
	}
	return methods
}

func (router *Router) Handle(ctx context.Context, session *SessionContext, request RequestFrame) ResponseFrame {
	method := strings.TrimSpace(request.Method)
	response := ResponseFrame{
		Type: "res",
		ID:   request.ID,
	}
	if method == "" {
		response.OK = false
		response.Error = NewGatewayError("invalid_request", "missing method")
		return response
	}

	router.mu.RLock()
	spec, ok := router.methods[method]
	router.mu.RUnlock()
	if !ok || spec.Handler == nil {
		response.OK = false
		response.Error = NewGatewayError("method_not_found", "method not registered")
		return response
	}

	if session == nil {
		response.OK = false
		response.Error = NewGatewayError("unauthorized", "session not authorized")
		return response
	}

	if router.lockGuard != nil && !router.lockGuard(method, session) {
		response.OK = false
		response.Error = NewGatewayError("gateway_locked", "gateway is locked")
		return response
	}

	if router.scopeGuard != nil {
		result := router.scopeGuard.Check(method, session.Auth, spec.RequiredScopes)
		if !result.Allowed {
			if router.audit != nil {
				router.audit(ctx, session, result)
			}
			response.OK = false
			response.Error = &GatewayError{
				Code:    "forbidden",
				Message: "scope denied",
				Details: result,
			}
			return response
		}
	}

	payload, err := spec.Handler(ctx, session, request.Params)
	if err != nil {
		response.OK = false
		response.Error = err
		return response
	}
	response.OK = true
	response.Payload = payload
	return response
}

func NewGatewayError(code, message string) *GatewayError {
	return &GatewayError{
		Code:    strings.TrimSpace(code),
		Message: strings.TrimSpace(message),
	}
}
