package dto

import "dreamcreator/internal/domain/assistant"

type AssistantSnapshot struct {
	AssistantID string                      `json:"assistantId"`
	Builtin     bool                        `json:"builtin"`
	Deletable   bool                        `json:"deletable"`
	Identity    assistant.AssistantIdentity `json:"identity"`
	Avatar      assistant.AssistantAvatar   `json:"avatar"`
	User        assistant.AssistantUser     `json:"user"`
	Model       assistant.AssistantModel    `json:"model"`
	Tools       assistant.AssistantTools    `json:"tools"`
	Skills      assistant.AssistantSkills   `json:"skills"`
	Call        assistant.AssistantCall     `json:"call"`
	Memory      assistant.AssistantMemory   `json:"memory"`
	Enabled     bool                        `json:"enabled"`
	IsDefault   bool                        `json:"isDefault"`
	CreatedAt   string                      `json:"createdAt"`
	UpdatedAt   string                      `json:"updatedAt"`
}

type ResolveAssistantSnapshotRequest struct {
	ThreadID    string `json:"threadId"`
	AssistantID string `json:"assistantId,omitempty"`
}
