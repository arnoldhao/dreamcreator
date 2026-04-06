package tools

import (
	"strings"
	"time"
)

type ToolKind string

const (
	ToolKindLocal ToolKind = "local"
	ToolKindWeb   ToolKind = "web"
)

type SideEffectLevel string

const (
	SideEffectNone        SideEffectLevel = "none"
	SideEffectRead        SideEffectLevel = "read"
	SideEffectWrite       SideEffectLevel = "write"
	SideEffectDestructive SideEffectLevel = "destructive"
)

type ToolSpec struct {
	ID              string
	Name            string
	Description     string
	Kind            ToolKind
	SchemaJSON      string
	SideEffectLevel SideEffectLevel
	Enabled         bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type ToolSpecParams struct {
	ID              string
	Name            string
	Description     string
	Kind            string
	SchemaJSON      string
	SideEffectLevel string
	Enabled         bool
	CreatedAt       *time.Time
	UpdatedAt       *time.Time
}

func NewToolSpec(params ToolSpecParams) (ToolSpec, error) {
	id := strings.TrimSpace(params.ID)
	name := strings.TrimSpace(params.Name)
	if id == "" || name == "" {
		return ToolSpec{}, ErrInvalidTool
	}

	createdAt := time.Now()
	updatedAt := createdAt
	if params.CreatedAt != nil {
		createdAt = *params.CreatedAt
	}
	if params.UpdatedAt != nil {
		updatedAt = *params.UpdatedAt
	}

	return ToolSpec{
		ID:              id,
		Name:            name,
		Description:     strings.TrimSpace(params.Description),
		Kind:            ToolKind(strings.TrimSpace(params.Kind)),
		SchemaJSON:      strings.TrimSpace(params.SchemaJSON),
		SideEffectLevel: SideEffectLevel(strings.TrimSpace(params.SideEffectLevel)),
		Enabled:         params.Enabled,
		CreatedAt:       createdAt,
		UpdatedAt:       updatedAt,
	}, nil
}

type ToolInvocation struct {
	ID          string
	ToolID      string
	InputJSON   string
	RequestedAt time.Time
}

type ToolInvocationParams struct {
	ID          string
	ToolID      string
	InputJSON   string
	RequestedAt *time.Time
}

func NewToolInvocation(params ToolInvocationParams) (ToolInvocation, error) {
	id := strings.TrimSpace(params.ID)
	toolID := strings.TrimSpace(params.ToolID)
	if id == "" || toolID == "" {
		return ToolInvocation{}, ErrInvalidInvocation
	}
	requestedAt := time.Now()
	if params.RequestedAt != nil {
		requestedAt = *params.RequestedAt
	}
	return ToolInvocation{
		ID:          id,
		ToolID:      toolID,
		InputJSON:   strings.TrimSpace(params.InputJSON),
		RequestedAt: requestedAt,
	}, nil
}

type ToolResult struct {
	ID           string
	ToolID       string
	OutputJSON   string
	ErrorMessage string
	StartedAt    *time.Time
	FinishedAt   *time.Time
}

type GuardrailsPolicy struct {
	AllowedKinds []ToolKind
	DeniedKinds  []ToolKind
}
