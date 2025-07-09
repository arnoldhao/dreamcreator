package types

import (
	"CanMe/backend/consts"
)

type WSRequest struct {
	Namespace consts.WSNamespace        `json:"namespace"`
	Event     consts.WSRequestEventType `json:"event" `
	Data      any                       `json:"data"`
}

type WSResponse struct {
	Namespace consts.WSNamespace         `json:"namespace"`
	Event     consts.WSResponseEventType `json:"event" `
	Data      any                        `json:"data"`
	ClientID  string                     `json:"client_id,omitempty"`
}

type HeartbeatData struct {
	Timestamp int64  `json:"timestamp"`
	Message   string `json:"message"`
}
