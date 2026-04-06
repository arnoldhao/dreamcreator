package dto

import "dreamcreator/internal/domain/assistant"

const (
	PromptModeFull    = "full"
	PromptModeMinimal = "minimal"
	PromptModeNone    = "none"
)

type GlobalWorkspace struct {
	ID                       int    `json:"id"`
	DefaultExecutorModelJSON string `json:"defaultExecutorModelJson"`
	DefaultMemoryJSON        string `json:"defaultMemoryJson"`
	DefaultPersona           string `json:"defaultPersona"`
	CreatedAt                string `json:"createdAt"`
	UpdatedAt                string `json:"updatedAt"`
}

type AssistantWorkspaceDirectory struct {
	AssistantID string `json:"assistantId"`
	WorkspaceID string `json:"workspaceId,omitempty"`
	RootPath    string `json:"rootPath,omitempty"`
}

type RuntimeSnapshot struct {
	GlobalWorkspaceID int              `json:"globalWorkspaceId"`
	AssistantID       string           `json:"assistantId"`
	ThreadID          string           `json:"threadId,omitempty"`
	RootPath          string           `json:"rootPath"`
	ExecutorModelJSON string           `json:"executorModelJson"`
	MemoryJSON        string           `json:"memoryJson"`
	Persona           string           `json:"persona"`
	WorkspaceContext  WorkspaceContext `json:"workspaceContext"`
}

type WorkspaceContext struct {
	PromptMode string          `json:"promptMode"`
	Files      []WorkspaceFile `json:"files"`
	Skills     []SkillMeta     `json:"skills"`
}

type WorkspaceFile struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	Content   string `json:"content,omitempty"`
	MaxChars  int    `json:"maxChars"`
	Size      int    `json:"size"`
	Missing   bool   `json:"missing"`
	UpdatedAt string `json:"updatedAt"`
}

type WorkspaceLogicalFile struct {
	Name      string `json:"name"`
	Path      string `json:"path,omitempty"`
	Content   string `json:"content,omitempty"`
	Source    string `json:"source,omitempty"`
	Required  bool   `json:"required,omitempty"`
	MaxChars  int    `json:"maxChars,omitempty"`
	Size      int    `json:"size,omitempty"`
	Missing   bool   `json:"missing,omitempty"`
	UpdatedAt string `json:"updatedAt,omitempty"`
}

type SkillMeta struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Path        string `json:"path"`
}

type UpdateGlobalWorkspaceRequest struct {
	DefaultExecutorModelJSON *string `json:"defaultExecutorModelJson"`
	DefaultMemoryJSON        *string `json:"defaultMemoryJson"`
	DefaultPersona           *string `json:"defaultPersona"`
}

type ResolveRuntimeSnapshotRequest struct {
	AssistantID             string `json:"assistantId"`
	ThreadID                string `json:"threadId,omitempty"`
	ForRunID                string `json:"forRunId,omitempty"`
	IncludeWorkspaceContext bool   `json:"includeWorkspaceContext,omitempty"`
}

type AssistantWorkspace struct {
	AssistantID       string                      `json:"assistantId"`
	Version           int64                       `json:"version"`
	Identity          assistant.AssistantIdentity `json:"identity"`
	Persona           string                      `json:"persona,omitempty"`
	UserProfile       assistant.AssistantUser     `json:"userProfile"`
	Tooling           assistant.AssistantCall     `json:"tooling"`
	Memory            assistant.AssistantMemory   `json:"memory"`
	MemoryJSON        string                      `json:"memoryJson,omitempty"`
	ExtraFiles        []WorkspaceLogicalFile      `json:"extraFiles,omitempty"`
	PromptModeDefault string                      `json:"promptModeDefault,omitempty"`
	UpdatedAt         string                      `json:"updatedAt"`
}

type WorkspacePatch struct {
	Identity          *assistant.AssistantIdentity `json:"identity,omitempty"`
	Persona           *string                      `json:"persona,omitempty"`
	UserProfile       *assistant.AssistantUser     `json:"userProfile,omitempty"`
	Tooling           *assistant.AssistantCall     `json:"tooling,omitempty"`
	Memory            *assistant.AssistantMemory   `json:"memory,omitempty"`
	MemoryJSON        *string                      `json:"memoryJson,omitempty"`
	ExtraFiles        *[]WorkspaceLogicalFile      `json:"extraFiles,omitempty"`
	PromptModeDefault *string                      `json:"promptModeDefault,omitempty"`
}

type UpdateWorkspaceRequest struct {
	AssistantID     string         `json:"assistantId"`
	ExpectedVersion int64          `json:"expectedVersion"`
	Patch           WorkspacePatch `json:"patch"`
}

type UpdateWorkspaceResponse struct {
	Workspace AssistantWorkspace `json:"workspace"`
}

type AssistantWorkspaceSnapshot struct {
	SnapshotID        string                 `json:"snapshotId"`
	AssistantID       string                 `json:"assistantId"`
	WorkspaceVersion  int64                  `json:"workspaceVersion"`
	LogicalFiles      []WorkspaceLogicalFile `json:"logicalFiles"`
	PromptModeDefault string                 `json:"promptModeDefault,omitempty"`
	ToolHints         []string               `json:"toolHints,omitempty"`
	SkillHints        []string               `json:"skillHints,omitempty"`
	GeneratedAt       string                 `json:"generatedAt,omitempty"`
	CreatedAt         string                 `json:"createdAt,omitempty"`
}

type ResolveWorkspaceSnapshotRequest struct {
	AssistantID      string `json:"assistantId"`
	ForRunID         string `json:"forRunId,omitempty"`
	WorkspaceVersion *int64 `json:"workspaceVersion,omitempty"`
}

type ResolveWorkspaceSnapshotResponse struct {
	Snapshot AssistantWorkspaceSnapshot `json:"snapshot"`
}
