package assistant

import (
	"strings"
	"time"
)

type Assistant struct {
	ID        string
	Builtin   bool
	Deletable bool
	Identity  AssistantIdentity
	Avatar    AssistantAvatar
	User      AssistantUser
	Model     AssistantModel
	Tools     AssistantTools
	Skills    AssistantSkills
	Call      AssistantCall
	Memory    AssistantMemory
	Enabled   bool
	IsDefault bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type AssistantParams struct {
	ID        string
	Builtin   *bool
	Deletable *bool
	Identity  AssistantIdentity
	Avatar    AssistantAvatar
	User      AssistantUser
	Model     AssistantModel
	Tools     AssistantTools
	Skills    AssistantSkills
	Call      AssistantCall
	Memory    AssistantMemory
	Enabled   *bool
	IsDefault *bool
	CreatedAt *time.Time
	UpdatedAt *time.Time
}

func NewAssistant(params AssistantParams) (Assistant, error) {
	id := strings.TrimSpace(params.ID)
	if id == "" {
		return Assistant{}, ErrInvalidAssistant
	}
	builtin := false
	if params.Builtin != nil {
		builtin = *params.Builtin
	}
	deletable := true
	if params.Deletable != nil {
		deletable = *params.Deletable
	}
	enabled := true
	if params.Enabled != nil {
		enabled = *params.Enabled
	}
	isDefault := false
	if params.IsDefault != nil {
		isDefault = *params.IsDefault
	}
	createdAt := time.Now()
	updatedAt := createdAt
	if params.CreatedAt != nil {
		createdAt = *params.CreatedAt
	}
	if params.UpdatedAt != nil {
		updatedAt = *params.UpdatedAt
	}
	identity := normalizeAssistantIdentity(params.Identity)
	avatar := normalizeAssistantAvatar(params.Avatar)
	user := normalizeAssistantUser(params.User)
	model := normalizeAssistantModel(params.Model)
	tools := normalizeAssistantTools(params.Tools)
	skills := normalizeAssistantSkills(params.Skills)
	call := normalizeAssistantCall(params.Call)
	memory := normalizeAssistantMemory(params.Memory)

	return Assistant{
		ID:        id,
		Builtin:   builtin,
		Deletable: deletable,
		Identity:  identity,
		Avatar:    avatar,
		User:      user,
		Model:     model,
		Tools:     tools,
		Skills:    skills,
		Call:      call,
		Memory:    memory,
		Enabled:   enabled,
		IsDefault: isDefault,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}, nil
}
