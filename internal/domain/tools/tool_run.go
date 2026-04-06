package tools

import (
	"strings"
	"time"
)

type ToolRunStatus string

const (
	ToolRunStatusRequested ToolRunStatus = "requested"
	ToolRunStatusRunning   ToolRunStatus = "running"
	ToolRunStatusCompleted ToolRunStatus = "completed"
	ToolRunStatusFailed    ToolRunStatus = "failed"
	ToolRunStatusTimeout   ToolRunStatus = "timeout"
	ToolRunStatusCanceled  ToolRunStatus = "canceled"
)

type ToolRun struct {
	ID         string
	RunID      string
	ToolCallID string
	ToolName   string
	InputHash  string
	InputJSON  string
	OutputJSON string
	ErrorText  string
	JobID      string
	Status     ToolRunStatus
	CreatedAt  time.Time
	StartedAt  *time.Time
	FinishedAt *time.Time
}

type ToolRunParams struct {
	ID         string
	RunID      string
	ToolCallID string
	ToolName   string
	InputHash  string
	InputJSON  string
	OutputJSON string
	ErrorText  string
	JobID      string
	Status     string
	CreatedAt  *time.Time
	StartedAt  *time.Time
	FinishedAt *time.Time
}

func NewToolRun(params ToolRunParams) (ToolRun, error) {
	id := strings.TrimSpace(params.ID)
	runID := strings.TrimSpace(params.RunID)
	toolName := strings.TrimSpace(params.ToolName)
	inputHash := strings.TrimSpace(params.InputHash)
	if id == "" || runID == "" || toolName == "" || inputHash == "" {
		return ToolRun{}, ErrInvalidToolRun
	}

	createdAt := time.Now()
	if params.CreatedAt != nil {
		createdAt = *params.CreatedAt
	}
	status := ToolRunStatus(strings.TrimSpace(params.Status))
	if status == "" {
		status = ToolRunStatusRequested
	}

	return ToolRun{
		ID:         id,
		RunID:      runID,
		ToolCallID: strings.TrimSpace(params.ToolCallID),
		ToolName:   toolName,
		InputHash:  inputHash,
		InputJSON:  strings.TrimSpace(params.InputJSON),
		OutputJSON: strings.TrimSpace(params.OutputJSON),
		ErrorText:  strings.TrimSpace(params.ErrorText),
		JobID:      strings.TrimSpace(params.JobID),
		Status:     status,
		CreatedAt:  createdAt,
		StartedAt:  params.StartedAt,
		FinishedAt: params.FinishedAt,
	}, nil
}
