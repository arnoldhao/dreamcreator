package innerinterfaces

type WebSocketServiceInterface interface {
	SendToClient(clientID string, message string)
	RemoveTranslation(id string)
}
