package providers

import "errors"

var (
	ErrInvalidProvider        = errors.New("invalid provider")
	ErrProviderNotFound       = errors.New("provider not found")
	ErrInvalidSecret          = errors.New("invalid provider secret")
	ErrProviderSecretNotFound = errors.New("provider secret not found")
	ErrInvalidModel           = errors.New("invalid model")
	ErrModelNotFound          = errors.New("model not found")
)
