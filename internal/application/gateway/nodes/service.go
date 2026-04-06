package nodes

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"dreamcreator/internal/application/gateway/approvals"
	gatewayevents "dreamcreator/internal/application/gateway/events"
	"dreamcreator/internal/application/gateway/pairing"
)

type NodeCapability struct {
	Name        string            `json:"name"`
	Version     string            `json:"version,omitempty"`
	Constraints map[string]string `json:"constraints,omitempty"`
}

type NodeDescriptor struct {
	NodeID       string          `json:"nodeId"`
	DisplayName  string          `json:"displayName,omitempty"`
	Platform     string          `json:"platform,omitempty"`
	Version      string          `json:"version,omitempty"`
	Capabilities []NodeCapability `json:"capabilities,omitempty"`
	Status       string          `json:"status,omitempty"`
	UpdatedAt    time.Time       `json:"updatedAt"`
}

type NodePairResult struct {
	Status      string `json:"status"`
	DeviceToken string `json:"deviceToken,omitempty"`
}

type NodeInvokeRequest struct {
	InvokeID   string `json:"invokeId"`
	NodeID     string `json:"nodeId"`
	Capability string `json:"capability"`
	Action     string `json:"action,omitempty"`
	Args       string `json:"args,omitempty"`
	TimeoutMs  int    `json:"timeoutMs,omitempty"`
}

type NodeInvokeResult struct {
	InvokeID string `json:"invokeId"`
	Ok       bool   `json:"ok"`
	Output   string `json:"output,omitempty"`
	Error    string `json:"error,omitempty"`
}

type Registry struct {
	mu       sync.RWMutex
	nodes    map[string]NodeDescriptor
	pairing  *pairing.Service
	store    RegistryStore
	now      func() time.Time
}

type RegistryStore interface {
	Save(ctx context.Context, descriptor NodeDescriptor) error
	List(ctx context.Context) ([]NodeDescriptor, error)
	Get(ctx context.Context, nodeID string) (NodeDescriptor, error)
}

type InvokeLogStore interface {
	Save(ctx context.Context, request NodeInvokeRequest, result NodeInvokeResult) error
}

type Invoker interface {
	Invoke(ctx context.Context, request NodeInvokeRequest) (NodeInvokeResult, error)
}

func NewRegistry(pairingService *pairing.Service, store RegistryStore) *Registry {
	return &Registry{
		nodes:   make(map[string]NodeDescriptor),
		pairing: pairingService,
		store:   store,
		now:     time.Now,
	}
}

func (registry *Registry) Register(ctx context.Context, token string, descriptor NodeDescriptor) (NodeDescriptor, error) {
	if registry == nil {
		return NodeDescriptor{}, errors.New("registry unavailable")
	}
	nodeID := strings.TrimSpace(descriptor.NodeID)
	if nodeID == "" {
		return NodeDescriptor{}, errors.New("node id is required")
	}
	if registry.pairing != nil {
		if !registry.pairing.ValidateToken(token) {
			return NodeDescriptor{}, errors.New("invalid pair token")
		}
	}
	descriptor.UpdatedAt = registry.now()
	if descriptor.Status == "" {
		descriptor.Status = "online"
	}
	registry.mu.Lock()
	registry.nodes[nodeID] = descriptor
	registry.mu.Unlock()
	if registry.store != nil {
		_ = registry.store.Save(ctx, descriptor)
	}
	_ = ctx
	return descriptor, nil
}

func (registry *Registry) List(ctx context.Context) []NodeDescriptor {
	if registry == nil {
		return nil
	}
	if registry.store != nil {
		if list, err := registry.store.List(ctx); err == nil {
			return list
		}
	}
	registry.mu.RLock()
	defer registry.mu.RUnlock()
	result := make([]NodeDescriptor, 0, len(registry.nodes))
	for _, node := range registry.nodes {
		result = append(result, node)
	}
	_ = ctx
	return result
}

func (registry *Registry) Get(ctx context.Context, nodeID string) (NodeDescriptor, error) {
	if registry == nil {
		return NodeDescriptor{}, errors.New("registry unavailable")
	}
	nodeID = strings.TrimSpace(nodeID)
	if registry.store != nil {
		if item, err := registry.store.Get(ctx, nodeID); err == nil {
			return item, nil
		}
	}
	registry.mu.RLock()
	node, ok := registry.nodes[nodeID]
	registry.mu.RUnlock()
	if !ok {
		return NodeDescriptor{}, errors.New("node not found")
	}
	_ = ctx
	return node, nil
}

type Service struct {
	registry  *Registry
	approvals *approvals.Service
	invoker   Invoker
	logs      InvokeLogStore
	events    *gatewayevents.Broker
	now       func() time.Time
}

func NewService(registry *Registry, approvalsSvc *approvals.Service, invoker Invoker, logs InvokeLogStore, events *gatewayevents.Broker) *Service {
	return &Service{
		registry:  registry,
		approvals: approvalsSvc,
		invoker:   invoker,
		logs:      logs,
		events:    events,
		now:       time.Now,
	}
}

func (service *Service) ListNodes(ctx context.Context) ([]NodeDescriptor, error) {
	if service == nil || service.registry == nil {
		return nil, errors.New("node registry unavailable")
	}
	return service.registry.List(ctx), nil
}

func (service *Service) DescribeNode(ctx context.Context, nodeID string) (NodeDescriptor, error) {
	if service == nil || service.registry == nil {
		return NodeDescriptor{}, errors.New("node registry unavailable")
	}
	return service.registry.Get(ctx, nodeID)
}

func (service *Service) Invoke(ctx context.Context, request NodeInvokeRequest) (NodeInvokeResult, error) {
	if service == nil || service.registry == nil {
		return NodeInvokeResult{}, errors.New("node registry unavailable")
	}
	if _, err := service.registry.Get(ctx, request.NodeID); err != nil {
		return NodeInvokeResult{InvokeID: request.InvokeID, Ok: false, Error: err.Error()}, err
	}
	if strings.EqualFold(request.Capability, "system.run") && service.approvals != nil {
		approval, err := service.approvals.Create(ctx, approvals.Request{
			SessionKey: request.NodeID,
			ToolName:   request.Capability,
			Action:     request.Action,
			Args:       request.Args,
		})
		if err != nil {
			return NodeInvokeResult{InvokeID: request.InvokeID, Ok: false, Error: err.Error()}, err
		}
		resolved, err := service.approvals.Wait(ctx, approvals.WaitRequest{ID: approval.ID})
		if err != nil {
			return NodeInvokeResult{InvokeID: request.InvokeID, Ok: false, Error: err.Error()}, err
		}
		if resolved.Status != approvals.StatusApproved {
			return NodeInvokeResult{InvokeID: request.InvokeID, Ok: false, Error: "approval denied"}, errors.New("approval denied")
		}
	}
	if service.invoker == nil {
		return NodeInvokeResult{InvokeID: request.InvokeID, Ok: false, Error: "node invoker unavailable"}, errors.New("node invoker unavailable")
	}
	timeout := time.Duration(request.TimeoutMs) * time.Millisecond
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	result, err := service.invoker.Invoke(runCtx, request)
	if err != nil {
		result.Ok = false
		if result.Error == "" {
			result.Error = err.Error()
		}
	}
	if service.logs != nil {
		_ = service.logs.Save(ctx, request, result)
	}
	service.publishInvokeResult(ctx, result)
	if err != nil {
		return result, err
	}
	return result, nil
}

func (service *Service) publishInvokeResult(ctx context.Context, result NodeInvokeResult) {
	if service == nil || service.events == nil {
		return
	}
	envelope := gatewayevents.Envelope{
		Type:      "node.invoke.result",
		Topic:     "node",
		Timestamp: service.now(),
	}
	_, _ = service.events.Publish(ctx, envelope, result)
}
