package connectors

import "errors"

var (
	ErrConnectorNotFound = errors.New("connector not found")
	ErrInvalidConnector  = errors.New("invalid connector")
	ErrNoCookies         = errors.New("no cookies stored")
)
