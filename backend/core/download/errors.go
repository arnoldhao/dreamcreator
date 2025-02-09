package download

import "errors"

var (
	ErrTaskNotFound    = errors.New("task not found")
	ErrQueueFull       = errors.New("download queue is full")
	ErrTaskCancelled   = errors.New("task was cancelled")
	ErrInvalidStatus   = errors.New("invalid task status")
	ErrTaskExists      = errors.New("task already exists")
	ErrEmptyURL        = errors.New("url is empty")
	ErrNoContent       = errors.New("no content found")
	ErrContentNotReady = errors.New("content not ready")
	ErrContentNotFound = errors.New("content not found")
)
