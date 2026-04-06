package thread

import "errors"

var (
    ErrInvalidThread         = errors.New("invalid thread")
    ErrThreadNotFound        = errors.New("thread not found")
    ErrInvalidThreadMessage  = errors.New("invalid thread message")
    ErrInvalidThreadRun      = errors.New("invalid thread run")
    ErrInvalidThreadRunEvent = errors.New("invalid thread run event")
)
