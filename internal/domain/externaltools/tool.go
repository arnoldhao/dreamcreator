package externaltools

import (
	"strings"
	"time"
)

type ToolName string

const (
	ToolYTDLP   ToolName = "yt-dlp"
	ToolFFmpeg  ToolName = "ffmpeg"
	ToolBun     ToolName = "bun"
	ToolClawHub ToolName = "clawhub"
)

type ToolKind string

const (
	KindBin     ToolKind = "bin"
	KindRuntime ToolKind = "runtime"
)

type ToolStatus string

const (
	StatusMissing   ToolStatus = "missing"
	StatusInstalled ToolStatus = "installed"
	StatusInvalid   ToolStatus = "invalid"
)

type ExternalTool struct {
	Name        ToolName
	ExecPath    string
	Version     string
	Status      ToolStatus
	InstalledAt *time.Time
	UpdatedAt   time.Time
}

type ExternalToolParams struct {
	Name        string
	ExecPath    string
	Version     string
	Status      string
	InstalledAt *time.Time
	UpdatedAt   *time.Time
}

func NewExternalTool(params ExternalToolParams) (ExternalTool, error) {
	name := ToolName(strings.TrimSpace(params.Name))
	if name == "" {
		return ExternalTool{}, ErrInvalidTool
	}
	status := ToolStatus(strings.TrimSpace(params.Status))
	if status == "" {
		status = StatusMissing
	}
	updatedAt := time.Now()
	if params.UpdatedAt != nil {
		updatedAt = *params.UpdatedAt
	}
	return ExternalTool{
		Name:        name,
		ExecPath:    strings.TrimSpace(params.ExecPath),
		Version:     strings.TrimSpace(params.Version),
		Status:      status,
		InstalledAt: params.InstalledAt,
		UpdatedAt:   updatedAt,
	}, nil
}
