package innerinterfaces

type WebSocketServiceInterface interface {
	SendToClient(clientID string, message string)
	CommonSendToClient(clientID string, message string)
	RemoveTranslation(id string)
}
