package workspace

import (
	"strings"
	"time"

	"dreamcreator/internal/domain/assistant"
)

const defaultWorkspaceVersion int64 = 1

type AssistantWorkspace struct {
	AssistantID       string
	Version           int64
	Identity          assistant.AssistantIdentity
	Persona           string
	UserProfile       assistant.AssistantUser
	Tooling           assistant.AssistantCall
	Memory            assistant.AssistantMemory
	MemoryJSON        string
	ExtraFiles        []WorkspaceLogicalFile
	PromptModeDefault string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type AssistantWorkspaceParams struct {
	AssistantID       string
	Version           int64
	Identity          assistant.AssistantIdentity
	Persona           string
	UserProfile       assistant.AssistantUser
	Tooling           assistant.AssistantCall
	Memory            assistant.AssistantMemory
	MemoryJSON        string
	ExtraFiles        []WorkspaceLogicalFile
	PromptModeDefault string
	CreatedAt         *time.Time
	UpdatedAt         *time.Time
}

func NewAssistantWorkspace(params AssistantWorkspaceParams) (AssistantWorkspace, error) {
	assistantID := strings.TrimSpace(params.AssistantID)
	if assistantID == "" {
		return AssistantWorkspace{}, ErrInvalidWorkspace
	}
	version := params.Version
	if version <= 0 {
		version = defaultWorkspaceVersion
	}
	createdAt := time.Now()
	updatedAt := createdAt
	if params.CreatedAt != nil {
		createdAt = *params.CreatedAt
	}
	if params.UpdatedAt != nil {
		updatedAt = *params.UpdatedAt
	}
	promptMode := normalizePromptMode(params.PromptModeDefault)
	if promptMode == "" {
		promptMode = assistant.PromptModeFull
	}
	return AssistantWorkspace{
		AssistantID:       assistantID,
		Version:           version,
		Identity:          params.Identity,
		Persona:           strings.TrimSpace(params.Persona),
		UserProfile:       params.UserProfile,
		Tooling:           params.Tooling,
		Memory:            params.Memory,
		MemoryJSON:        strings.TrimSpace(params.MemoryJSON),
		ExtraFiles:        normalizeWorkspaceFiles(params.ExtraFiles),
		PromptModeDefault: promptMode,
		CreatedAt:         createdAt,
		UpdatedAt:         updatedAt,
	}, nil
}

func normalizePromptMode(value string) string {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	switch trimmed {
	case assistant.PromptModeFull, assistant.PromptModeMinimal, assistant.PromptModeNone:
		return trimmed
	default:
		return ""
	}
}

func normalizeWorkspaceFiles(files []WorkspaceLogicalFile) []WorkspaceLogicalFile {
	if len(files) == 0 {
		return nil
	}
	result := make([]WorkspaceLogicalFile, 0, len(files))
	for _, file := range files {
		name := strings.TrimSpace(file.Name)
		if name == "" {
			continue
		}
		file.Name = name
		file.Path = strings.TrimSpace(file.Path)
		file.Content = strings.TrimSpace(file.Content)
		file.Source = strings.TrimSpace(file.Source)
		result = append(result, file)
	}
	if len(result) == 0 {
		return nil
	}
	return result
}
