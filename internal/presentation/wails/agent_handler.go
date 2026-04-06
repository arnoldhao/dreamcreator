package wails

import (
	"context"

	"dreamcreator/internal/application/agent/dto"
	"dreamcreator/internal/application/agent/service"
)

type AgentHandler struct {
	service *service.AgentService
}

func NewAgentHandler(service *service.AgentService) *AgentHandler {
	return &AgentHandler{service: service}
}

func (handler *AgentHandler) ServiceName() string {
	return "AgentHandler"
}

func (handler *AgentHandler) ListAgents(ctx context.Context, includeDisabled bool) ([]dto.Agent, error) {
	return handler.service.ListAgents(ctx, includeDisabled)
}

func (handler *AgentHandler) GetAgent(ctx context.Context, id string) (dto.Agent, error) {
	return handler.service.GetAgent(ctx, id)
}

func (handler *AgentHandler) CreateAgent(ctx context.Context, request dto.CreateAgentRequest) (dto.Agent, error) {
	return handler.service.CreateAgent(ctx, request)
}

func (handler *AgentHandler) UpdateAgent(ctx context.Context, request dto.UpdateAgentRequest) (dto.Agent, error) {
	return handler.service.UpdateAgent(ctx, request)
}

func (handler *AgentHandler) DeleteAgent(ctx context.Context, request dto.DeleteAgentRequest) error {
	return handler.service.DeleteAgent(ctx, request)
}

func (handler *AgentHandler) ListAgentRuns(ctx context.Context, request dto.ListAgentRunsRequest) ([]dto.AgentRun, error) {
	return handler.service.ListAgentRuns(ctx, request)
}

func (handler *AgentHandler) ListAgentRunEvents(ctx context.Context, request dto.ListAgentRunEventsRequest) ([]dto.AgentRunEvent, error) {
	return handler.service.ListAgentRunEvents(ctx, request)
}
