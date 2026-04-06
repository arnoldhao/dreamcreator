package workspace

import "errors"

var (
	ErrInvalidWorkspace         = errors.New("invalid workspace")
	ErrWorkspaceNotFound        = errors.New("workspace not found")
	ErrWorkspaceVersionConflict = errors.New("workspace version conflict")
)
