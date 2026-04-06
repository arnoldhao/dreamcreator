package workspace

import (
	"strings"
	"time"
)

type GlobalWorkspace struct {
	ID                       int
	DefaultExecutorModelJSON string
	DefaultMemoryJSON        string
	DefaultPersona           string
	CreatedAt                time.Time
	UpdatedAt                time.Time
}

type GlobalWorkspaceParams struct {
	ID                       int
	DefaultExecutorModelJSON string
	DefaultMemoryJSON        string
	DefaultPersona           string
	CreatedAt                *time.Time
	UpdatedAt                *time.Time
}

func NewGlobalWorkspace(params GlobalWorkspaceParams) (GlobalWorkspace, error) {
	id := params.ID
	if id <= 0 {
		id = 1
	}
	createdAt := time.Now()
	updatedAt := createdAt
	if params.CreatedAt != nil {
		createdAt = *params.CreatedAt
	}
	if params.UpdatedAt != nil {
		updatedAt = *params.UpdatedAt
	}

	return GlobalWorkspace{
		ID:                       id,
		DefaultExecutorModelJSON: strings.TrimSpace(params.DefaultExecutorModelJSON),
		DefaultMemoryJSON:        strings.TrimSpace(params.DefaultMemoryJSON),
		DefaultPersona:           strings.TrimSpace(params.DefaultPersona),
		CreatedAt:                createdAt,
		UpdatedAt:                updatedAt,
	}, nil
}
