package agent

import (
	"strings"
	"time"
)

type Agent struct {
	ID          string
	Name        string
	Description string
	Enabled     bool
	ThreadID    string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
}

type AgentParams struct {
	ID          string
	Name        string
	Description string
	Enabled     *bool
	ThreadID    string
	CreatedAt   *time.Time
	UpdatedAt   *time.Time
	DeletedAt   *time.Time
}

func NewAgent(params AgentParams) (Agent, error) {
	id := strings.TrimSpace(params.ID)
	name := strings.TrimSpace(params.Name)
	threadID := strings.TrimSpace(params.ThreadID)
	if id == "" || name == "" || threadID == "" {
		return Agent{}, ErrInvalidAgent
	}
	enabled := true
	if params.Enabled != nil {
		enabled = *params.Enabled
	}
	createdAt := time.Now()
	updatedAt := createdAt
	if params.CreatedAt != nil {
		createdAt = *params.CreatedAt
	}
	if params.UpdatedAt != nil {
		updatedAt = *params.UpdatedAt
	}

	return Agent{
		ID:          id,
		Name:        name,
		Description: strings.TrimSpace(params.Description),
		Enabled:     enabled,
		ThreadID:    threadID,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
		DeletedAt:   params.DeletedAt,
	}, nil
}
