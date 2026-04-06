package tools

import "errors"

var (
	ErrInvalidTool       = errors.New("invalid tool")
	ErrToolNotFound      = errors.New("tool not found")
	ErrInvalidInvocation = errors.New("invalid tool invocation")
	ErrInvalidToolRun    = errors.New("invalid tool run")
	ErrToolRunNotFound   = errors.New("tool run not found")
	ErrNotImplemented    = errors.New("not implemented")
)
