package assistant

import "errors"

var (
	ErrInvalidAssistant           = errors.New("invalid assistant")
	ErrAssistantNotFound          = errors.New("assistant not found")
	ErrInvalidAssistantID         = errors.New("assistant id is required")
	ErrAssistantDeletionNotAllowed = errors.New("assistant is not deletable")
)
