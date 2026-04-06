package memory

import "errors"

var (
	ErrInvalidMemory  = errors.New("invalid memory")
	ErrMemoryNotFound = errors.New("memory not found")
	ErrNotImplemented = errors.New("not implemented")
)
