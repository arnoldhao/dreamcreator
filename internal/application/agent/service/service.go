package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"dreamcreator/internal/application/agent/dto"
	"dreamcreator/internal/domain/agent"
	"dreamcreator/internal/domain/assistant"
	"dreamcreator/internal/domain/thread"
)

const defaultAgentName = "Default Subagent"
const defaultAgentPurgeDelay = 7 * 24 * time.Hour

var errInvalidAgentName = errors.New("agent name is required")

type AgentService struct {
	repo       agent.Repository
	threads    thread.Repository
	runs       thread.RunRepository
	runEvents  thread.RunEventRepository
	assistants assistant.Repository
	now        func() time.Time
	newID      func() string
}

func NewAgentService(
	repo agent.Repository,
	threadRepo thread.Repository,
	runRepo thread.RunRepository,
	runEventRepo thread.RunEventRepository,
	assistantRepo assistant.Repository,
) *AgentService {
	return &AgentService{
		repo:       repo,
		threads:    threadRepo,
		runs:       runRepo,
		runEvents:  runEventRepo,
		assistants: assistantRepo,
		now:        time.Now,
		newID:      uuid.NewString,
	}
}

func (service *AgentService) EnsureDefaults(ctx context.Context) error {
	agents, err := service.repo.List(ctx, true)
	if err != nil {
		return err
	}
	if len(agents) > 0 {
		return nil
	}
	_, err = service.CreateAgent(ctx, dto.CreateAgentRequest{Name: defaultAgentName, Description: ""})
	return err
}

func (service *AgentService) ListAgents(ctx context.Context, includeDisabled bool) ([]dto.Agent, error) {
	items, err := service.repo.List(ctx, includeDisabled)
	if err != nil {
		return nil, err
	}
	result := make([]dto.Agent, 0, len(items))
	for _, item := range items {
		result = append(result, toDTO(item))
	}
	return result, nil
}

func (service *AgentService) GetAgent(ctx context.Context, id string) (dto.Agent, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return dto.Agent{}, agent.ErrAgentNotFound
	}
	item, err := service.repo.Get(ctx, id)
	if err != nil {
		return dto.Agent{}, err
	}
	return toDTO(item), nil
}

func (service *AgentService) CreateAgent(ctx context.Context, request dto.CreateAgentRequest) (dto.Agent, error) {
	name := strings.TrimSpace(request.Name)
	if name == "" {
		return dto.Agent{}, errInvalidAgentName
	}
	description := strings.TrimSpace(request.Description)
	agentID := service.newID()
	threadID := service.newID()

	now := service.now()
	assistantID := ""
	if service.assistants != nil {
		assistantItem, err := resolveDefaultAssistant(ctx, service.assistants)
		if err != nil {
			return dto.Agent{}, err
		}
		assistantID = assistantItem.ID
	}
	if assistantID == "" {
		return dto.Agent{}, errors.New("assistant id is required")
	}

	conv, err := thread.NewThread(thread.ThreadParams{
		ID:             threadID,
		AgentID:        agentID,
		AssistantID:    assistantID,
		Title:          name,
		TitleIsDefault: true,
		Status:         thread.ThreadStatusRegular,
		CreatedAt:      &now,
		UpdatedAt:      &now,
	})
	if err != nil {
		return dto.Agent{}, err
	}
	if err := service.threads.Save(ctx, conv); err != nil {
		return dto.Agent{}, err
	}

	enabled := true
	item, err := agent.NewAgent(agent.AgentParams{
		ID:          agentID,
		Name:        name,
		Description: description,
		Enabled:     &enabled,
		ThreadID:    threadID,
		CreatedAt:   &now,
		UpdatedAt:   &now,
	})
	if err != nil {
		return dto.Agent{}, err
	}
	if err := service.repo.Save(ctx, item); err != nil {
		return dto.Agent{}, err
	}
	return toDTO(item), nil
}

func (service *AgentService) UpdateAgent(ctx context.Context, request dto.UpdateAgentRequest) (dto.Agent, error) {
	id := strings.TrimSpace(request.ID)
	if id == "" {
		return dto.Agent{}, agent.ErrAgentNotFound
	}
	current, err := service.repo.Get(ctx, id)
	if err != nil {
		return dto.Agent{}, err
	}
	name := current.Name
	if request.Name != nil {
		name = strings.TrimSpace(*request.Name)
	}
	if name == "" {
		return dto.Agent{}, errInvalidAgentName
	}
	description := current.Description
	if request.Description != nil {
		description = strings.TrimSpace(*request.Description)
	}
	enabled := current.Enabled
	if request.Enabled != nil {
		enabled = *request.Enabled
	}
	updatedAt := service.now()

	item, err := agent.NewAgent(agent.AgentParams{
		ID:          current.ID,
		Name:        name,
		Description: description,
		Enabled:     &enabled,
		ThreadID:    current.ThreadID,
		CreatedAt:   &current.CreatedAt,
		UpdatedAt:   &updatedAt,
		DeletedAt:   current.DeletedAt,
	})
	if err != nil {
		return dto.Agent{}, err
	}
	if err := service.repo.Save(ctx, item); err != nil {
		return dto.Agent{}, err
	}
	return toDTO(item), nil
}

func (service *AgentService) DeleteAgent(ctx context.Context, request dto.DeleteAgentRequest) error {
	id := strings.TrimSpace(request.ID)
	if id == "" {
		return agent.ErrAgentNotFound
	}
	item, err := service.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	now := service.now()
	purgeAfter := now.Add(defaultAgentPurgeDelay)
	if service.threads != nil {
		if err := service.threads.SoftDelete(ctx, item.ThreadID, &now, &purgeAfter); err != nil &&
			!errors.Is(err, thread.ErrThreadNotFound) {
			return err
		}
	}
	item.Enabled = false
	item.DeletedAt = &now
	item.UpdatedAt = now
	if err := service.repo.Save(ctx, item); err != nil {
		return err
	}
	return service.repo.SoftDelete(ctx, id, &now)
}

const (
	agentFileAgents    = "AGENTS.md"
	agentFileSoul      = "SOUL.md"
	agentFileTools     = "TOOLS.md"
	agentFileIdentity  = "IDENTITY.md"
	agentFileUser      = "USER.md"
	agentFileHeartbeat = "HEARTBEAT.md"
	agentFileBootstrap = "BOOTSTRAP.md"
	agentFileMemory    = "MEMORY.md"
)

var agentFileNames = []string{
	agentFileAgents,
	agentFileSoul,
	agentFileTools,
	agentFileIdentity,
	agentFileUser,
	agentFileHeartbeat,
	agentFileBootstrap,
	agentFileMemory,
}

func (service *AgentService) ListAgentFiles(ctx context.Context, request dto.AgentFilesListRequest) (dto.AgentFilesListResponse, error) {
	agentID := strings.TrimSpace(request.AgentID)
	if agentID == "" {
		return dto.AgentFilesListResponse{}, agent.ErrAgentNotFound
	}
	_, assistantItem, err := service.resolveAgentAssistant(ctx, agentID)
	if err != nil {
		return dto.AgentFilesListResponse{}, err
	}
	files := buildAgentFileEntries(assistantItem)
	return dto.AgentFilesListResponse{AgentID: agentID, Files: files}, nil
}

func (service *AgentService) GetAgentFile(ctx context.Context, request dto.AgentFilesGetRequest) (dto.AgentFileEntry, error) {
	agentID := strings.TrimSpace(request.AgentID)
	if agentID == "" {
		return dto.AgentFileEntry{}, agent.ErrAgentNotFound
	}
	name := strings.TrimSpace(request.Name)
	if name == "" {
		return dto.AgentFileEntry{}, errors.New("file name is required")
	}
	_, assistantItem, err := service.resolveAgentAssistant(ctx, agentID)
	if err != nil {
		return dto.AgentFileEntry{}, err
	}
	files := buildAgentFileEntries(assistantItem)
	for _, entry := range files {
		if entry.Name == name {
			return entry, nil
		}
	}
	return dto.AgentFileEntry{}, errors.New("unsupported agent file")
}

func (service *AgentService) SetAgentFile(ctx context.Context, request dto.AgentFilesSetRequest) (dto.AgentFileEntry, error) {
	agentID := strings.TrimSpace(request.AgentID)
	if agentID == "" {
		return dto.AgentFileEntry{}, agent.ErrAgentNotFound
	}
	name := strings.TrimSpace(request.Name)
	if name == "" {
		return dto.AgentFileEntry{}, errors.New("file name is required")
	}
	currentAgent, assistantItem, err := service.resolveAgentAssistant(ctx, agentID)
	if err != nil {
		return dto.AgentFileEntry{}, err
	}
	updated, err := applyAgentFileUpdate(name, request.Content, assistantItem)
	if err != nil {
		return dto.AgentFileEntry{}, err
	}
	if err := service.assistants.Save(ctx, updated); err != nil {
		return dto.AgentFileEntry{}, err
	}
	if strings.TrimSpace(currentAgent.ThreadID) != "" && strings.TrimSpace(updated.ID) != "" {
		if conv, convErr := service.threads.Get(ctx, currentAgent.ThreadID); convErr == nil {
			if strings.TrimSpace(conv.AssistantID) == "" {
				conv.AssistantID = updated.ID
				conv.UpdatedAt = service.now()
				_ = service.threads.Save(ctx, conv)
			}
		}
	}
	files := buildAgentFileEntries(updated)
	for _, entry := range files {
		if entry.Name == name {
			return entry, nil
		}
	}
	return dto.AgentFileEntry{}, errors.New("unsupported agent file")
}

func buildAgentFileEntries(assistantItem assistant.Assistant) []dto.AgentFileEntry {
	entries := make([]dto.AgentFileEntry, 0, len(agentFileNames))
	updatedAt := assistantItem.UpdatedAt.Format(time.RFC3339)
	for _, name := range agentFileNames {
		entry := dto.AgentFileEntry{
			Name:      name,
			Missing:   true,
			UpdatedAt: updatedAt,
		}
		content := ""
		switch name {
		case agentFileIdentity:
			payload := struct {
				Name     string `json:"name,omitempty"`
				Creature string `json:"creature,omitempty"`
			}{
				Name:     assistantItem.Identity.Name,
				Creature: assistantItem.Identity.Creature,
			}
			content = marshalAgentFile(payload)
		case agentFileSoul:
			content = marshalAgentFile(assistantItem.Identity.Soul)
		case agentFileUser:
			content = marshalAgentFile(assistantItem.User)
		case agentFileTools:
			content = marshalAgentFile(assistantItem.Call)
		case agentFileMemory:
			content = marshalAgentFile(assistantItem.Memory)
		default:
			// keep missing true for unsupported files
		}
		if content != "" {
			entry.Content = content
			entry.Size = len(content)
			entry.Missing = false
		}
		entries = append(entries, entry)
	}
	return entries
}

func marshalAgentFile(payload any) string {
	encoded, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return ""
	}
	return string(encoded)
}

func applyAgentFileUpdate(name string, content string, assistantItem assistant.Assistant) (assistant.Assistant, error) {
	now := time.Now()
	updated := assistantItem
	switch name {
	case agentFileIdentity:
		var payload struct {
			Name     string `json:"name,omitempty"`
			Creature string `json:"creature,omitempty"`
			Emoji    string `json:"emoji,omitempty"`
			Role     string `json:"role,omitempty"`
		}
		if err := json.Unmarshal([]byte(content), &payload); err != nil {
			return assistant.Assistant{}, err
		}
		updated.Identity.Name = strings.TrimSpace(payload.Name)
		updated.Identity.Creature = strings.TrimSpace(payload.Creature)
		updated.Identity.Emoji = strings.TrimSpace(payload.Emoji)
		updated.Identity.Role = strings.TrimSpace(payload.Role)
	case agentFileSoul:
		var payload assistant.AssistantSoul
		if err := json.Unmarshal([]byte(content), &payload); err != nil {
			return assistant.Assistant{}, err
		}
		updated.Identity.Soul = payload
	case agentFileUser:
		var payload assistant.AssistantUser
		if err := json.Unmarshal([]byte(content), &payload); err != nil {
			return assistant.Assistant{}, err
		}
		updated.User = payload
	case agentFileTools:
		var payload assistant.AssistantCall
		if err := json.Unmarshal([]byte(content), &payload); err != nil {
			return assistant.Assistant{}, err
		}
		updated.Call = payload
	case agentFileMemory:
		var payload assistant.AssistantMemory
		if err := json.Unmarshal([]byte(content), &payload); err != nil {
			return assistant.Assistant{}, err
		}
		updated.Memory = payload
	default:
		return assistant.Assistant{}, errors.New("unsupported agent file")
	}

	result, err := assistant.NewAssistant(assistant.AssistantParams{
		ID:        updated.ID,
		Builtin:   &updated.Builtin,
		Deletable: &updated.Deletable,
		Identity:  updated.Identity,
		Avatar:    updated.Avatar,
		User:      updated.User,
		Model:     updated.Model,
		Tools:     updated.Tools,
		Skills:    updated.Skills,
		Call:      updated.Call,
		Memory:    updated.Memory,
		Enabled:   &updated.Enabled,
		IsDefault: &updated.IsDefault,
		CreatedAt: &updated.CreatedAt,
		UpdatedAt: &now,
	})
	if err != nil {
		return assistant.Assistant{}, err
	}
	return result, nil
}

func (service *AgentService) resolveAgentAssistant(ctx context.Context, agentID string) (agent.Agent, assistant.Assistant, error) {
	if service.repo == nil || service.threads == nil || service.assistants == nil {
		return agent.Agent{}, assistant.Assistant{}, errors.New("agent dependencies unavailable")
	}
	item, err := service.repo.Get(ctx, agentID)
	if err != nil {
		return agent.Agent{}, assistant.Assistant{}, err
	}
	threadID := strings.TrimSpace(item.ThreadID)
	if threadID == "" {
		return item, assistant.Assistant{}, errors.New("agent thread is missing")
	}
	conv, err := service.threads.Get(ctx, threadID)
	if err != nil {
		return item, assistant.Assistant{}, err
	}
	assistantID := strings.TrimSpace(conv.AssistantID)
	if assistantID == "" {
		assistantItem, err := resolveDefaultAssistant(ctx, service.assistants)
		if err != nil {
			return item, assistant.Assistant{}, err
		}
		assistantID = assistantItem.ID
		conv.AssistantID = assistantID
		conv.UpdatedAt = service.now()
		_ = service.threads.Save(ctx, conv)
	}
	assistantItem, err := service.assistants.Get(ctx, assistantID)
	if err != nil {
		return item, assistant.Assistant{}, err
	}
	return item, assistantItem, nil
}

func resolveDefaultAssistant(ctx context.Context, repo assistant.Repository) (assistant.Assistant, error) {
	items, err := repo.List(ctx, true)
	if err != nil {
		return assistant.Assistant{}, err
	}
	if len(items) == 0 {
		return assistant.Assistant{}, assistant.ErrAssistantNotFound
	}
	for _, item := range items {
		if item.IsDefault {
			return item, nil
		}
	}
	return items[0], nil
}

func (service *AgentService) ListAgentRuns(ctx context.Context, request dto.ListAgentRunsRequest) ([]dto.AgentRun, error) {
	agentID := strings.TrimSpace(request.AgentID)
	if agentID == "" {
		return nil, agent.ErrAgentNotFound
	}
	if service.runs == nil {
		return nil, errors.New("run repository unavailable")
	}
	limit := request.Limit
	if limit <= 0 {
		limit = 50
	}
	items, err := service.runs.ListByAgentID(ctx, agentID, limit)
	if err != nil {
		if errors.Is(err, thread.ErrRunNotFound) {
			return []dto.AgentRun{}, nil
		}
		return nil, err
	}
	result := make([]dto.AgentRun, 0, len(items))
	for _, item := range items {
		result = append(result, dto.AgentRun{
			ID:                 item.ID,
			ThreadID:           item.ThreadID,
			AssistantMessageID: item.AssistantMessageID,
			UserMessageID:      item.UserMessageID,
			AgentID:            item.AgentID,
			Status:             string(item.Status),
			ContentPartial:     item.ContentPartial,
			CreatedAt:          item.CreatedAt.Format(time.RFC3339),
			UpdatedAt:          item.UpdatedAt.Format(time.RFC3339),
		})
	}
	return result, nil
}

func (service *AgentService) ListAgentRunEvents(ctx context.Context, request dto.ListAgentRunEventsRequest) ([]dto.AgentRunEvent, error) {
	runID := strings.TrimSpace(request.RunID)
	if runID == "" {
		return nil, thread.ErrRunNotFound
	}
	if service.runEvents == nil {
		return nil, errors.New("run event repository unavailable")
	}
	limit := request.Limit
	if limit <= 0 {
		limit = 200
	}
	items, err := service.runEvents.ListAfter(ctx, runID, request.AfterID, limit)
	if err != nil {
		return nil, err
	}
	result := make([]dto.AgentRunEvent, 0, len(items))
	for _, item := range items {
		result = append(result, dto.AgentRunEvent{
			ID:          item.ID,
			RunID:       item.RunID,
			EventType:   item.EventType,
			PayloadJSON: item.PayloadJSON,
			CreatedAt:   item.CreatedAt.Format(time.RFC3339),
		})
	}
	return result, nil
}

func toDTO(item agent.Agent) dto.Agent {
	return dto.Agent{
		ID:          item.ID,
		Name:        item.Name,
		Description: item.Description,
		Enabled:     item.Enabled,
		ThreadID:    item.ThreadID,
		CreatedAt:   item.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   item.UpdatedAt.Format(time.RFC3339),
	}
}
