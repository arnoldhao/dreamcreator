package service

import (
	"context"
	"strings"
	"time"
)

type SkillsRealtimeEvent struct {
	Action        string    `json:"action"`
	Stage         string    `json:"stage"`
	Skill         string    `json:"skill,omitempty"`
	Version       string    `json:"version,omitempty"`
	ProviderID    string    `json:"providerId,omitempty"`
	AssistantID   string    `json:"assistantId,omitempty"`
	WorkspaceRoot string    `json:"workspaceRoot,omitempty"`
	Force         bool      `json:"force,omitempty"`
	CatalogCount  int       `json:"catalogCount,omitempty"`
	Error         string    `json:"error,omitempty"`
	Timestamp     time.Time `json:"timestamp"`
}

type SkillsRealtimeNotifier func(ctx context.Context, event SkillsRealtimeEvent)

func (service *SkillsService) SetRealtimeNotifier(notifier SkillsRealtimeNotifier) {
	if service == nil {
		return
	}
	service.realtimeNotifier = notifier
}

func (service *SkillsService) emitRealtimeEvent(ctx context.Context, event SkillsRealtimeEvent) {
	if service == nil || service.realtimeNotifier == nil {
		return
	}
	event.Action = strings.ToLower(strings.TrimSpace(event.Action))
	event.Stage = strings.ToLower(strings.TrimSpace(event.Stage))
	event.Skill = strings.TrimSpace(event.Skill)
	event.Version = strings.TrimSpace(event.Version)
	event.ProviderID = strings.TrimSpace(event.ProviderID)
	event.AssistantID = strings.TrimSpace(event.AssistantID)
	event.WorkspaceRoot = strings.TrimSpace(event.WorkspaceRoot)
	event.Error = strings.TrimSpace(event.Error)
	if event.Action == "" || event.Stage == "" {
		return
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = service.now()
	}
	service.realtimeNotifier(ctx, event)
}
