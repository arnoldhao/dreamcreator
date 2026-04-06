package dto

import "dreamcreator/internal/domain/assistant"

type Assistant struct {
	ID        string                      `json:"id"`
	Builtin   bool                        `json:"builtin"`
	Deletable bool                        `json:"deletable"`
	Identity  assistant.AssistantIdentity `json:"identity"`
	Avatar    assistant.AssistantAvatar   `json:"avatar"`
	User      assistant.AssistantUser     `json:"user"`
	Model     assistant.AssistantModel    `json:"model"`
	Tools     assistant.AssistantTools    `json:"tools"`
	Skills    assistant.AssistantSkills   `json:"skills"`
	Call      assistant.AssistantCall     `json:"call"`
	Memory    assistant.AssistantMemory   `json:"memory"`
	Readiness AssistantReadiness          `json:"readiness"`
	Enabled   bool                        `json:"enabled"`
	IsDefault bool                        `json:"isDefault"`
	CreatedAt string                      `json:"createdAt"`
	UpdatedAt string                      `json:"updatedAt"`
}

type AssistantReadiness struct {
	Ready   bool     `json:"ready"`
	Missing []string `json:"missing,omitempty"`
}

type CreateAssistantRequest struct {
	Identity  assistant.AssistantIdentity `json:"identity"`
	Avatar    assistant.AssistantAvatar   `json:"avatar"`
	User      assistant.AssistantUser     `json:"user"`
	Model     assistant.AssistantModel    `json:"model"`
	Tools     assistant.AssistantTools    `json:"tools"`
	Skills    assistant.AssistantSkills   `json:"skills"`
	Call      assistant.AssistantCall     `json:"call"`
	Memory    assistant.AssistantMemory   `json:"memory"`
	Enabled   *bool                       `json:"enabled,omitempty"`
	IsDefault bool                        `json:"isDefault"`
}

type UpdateAssistantRequest struct {
	ID        string                       `json:"id"`
	Identity  *assistant.AssistantIdentity `json:"identity,omitempty"`
	Avatar    *assistant.AssistantAvatar   `json:"avatar,omitempty"`
	User      *assistant.AssistantUser     `json:"user,omitempty"`
	Model     *assistant.AssistantModel    `json:"model,omitempty"`
	Tools     *assistant.AssistantTools    `json:"tools,omitempty"`
	Skills    *assistant.AssistantSkills   `json:"skills,omitempty"`
	Call      *assistant.AssistantCall     `json:"call,omitempty"`
	Memory    *assistant.AssistantMemory   `json:"memory,omitempty"`
	Enabled   *bool                        `json:"enabled,omitempty"`
	IsDefault *bool                        `json:"isDefault,omitempty"`
}

type DeleteAssistantRequest struct {
	ID string `json:"id"`
}

type SetDefaultAssistantRequest struct {
	ID string `json:"id"`
}

type AssistantAvatarAsset struct {
	Kind        string `json:"kind"`
	Path        string `json:"path"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName,omitempty"`
	UpdatedAt   string `json:"updatedAt,omitempty"`
	Source      string `json:"source,omitempty"`
	AssetID     string `json:"assetId,omitempty"`
}

type ImportAssistantAvatarRequest struct {
	Kind          string `json:"kind"`
	FileName      string `json:"fileName"`
	ContentBase64 string `json:"contentBase64"`
}

type ImportAssistantAvatarFromPathRequest struct {
	Kind string `json:"kind"`
	Path string `json:"path"`
}

type ReadAssistantAvatarSourceRequest struct {
	Kind string `json:"kind"`
	Path string `json:"path"`
}

type ReadAssistantAvatarSourceResponse struct {
	ContentBase64 string `json:"contentBase64"`
	Mime          string `json:"mime"`
	FileName      string `json:"fileName"`
	SizeBytes     int64  `json:"sizeBytes"`
}

type DeleteAssistantAvatarAssetRequest struct {
	Kind string `json:"kind"`
	Path string `json:"path"`
}

type UpdateAssistantAvatarAssetRequest struct {
	Kind        string `json:"kind"`
	Path        string `json:"path"`
	DisplayName string `json:"displayName,omitempty"`
}

type AssistantMemorySummary struct {
	Summary string `json:"summary"`
}

type AssistantProfileOptions struct {
	Roles       []string `json:"roles"`
	DefaultRole string   `json:"defaultRole,omitempty"`
	Vibes       []string `json:"vibes"`
	DefaultVibe string   `json:"defaultVibe,omitempty"`
}
