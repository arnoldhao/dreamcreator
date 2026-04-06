package externaltools

import "errors"

var (
	ErrToolNotFound = errors.New("external tool not found")
	ErrInvalidTool  = errors.New("invalid external tool")
)
