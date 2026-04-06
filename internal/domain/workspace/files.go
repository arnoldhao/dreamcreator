package workspace

import "time"

type WorkspaceFileSpec struct {
	Name           string
	Path           string
	DefaultContent string
	MaxChars       int
	Required       bool
}

type WorkspaceFile struct {
	Name      string
	Path      string
	Content   string
	MaxChars  int
	Size      int
	Missing   bool
	UpdatedAt time.Time
}

type SkillMeta struct {
	ID          string
	Name        string
	Description string
	Path        string
}
