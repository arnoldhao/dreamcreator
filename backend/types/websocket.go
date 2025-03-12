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
}
