package innerinterfaces

import "CanMe/backend/types"

type WebSocketServiceInterface interface {
	SendToClient(message types.WSResponse)
}
