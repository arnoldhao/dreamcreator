package workspace

import "time"

type WorkspaceLogicalFile struct {
	Name      string
	Path      string
	Content   string
	Source    string
	Required  bool
	MaxChars  int
	Size      int
	Missing   bool
	UpdatedAt time.Time
}

type AssistantWorkspaceSnapshot struct {
	ID                string
	AssistantID       string
	WorkspaceVersion  int64
	LogicalFiles      []WorkspaceLogicalFile
	PromptModeDefault string
	ToolHints         []string
	SkillHints        []string
	GeneratedAt       time.Time
	CreatedAt         time.Time
}
