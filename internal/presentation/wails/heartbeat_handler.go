package wails

import (
	"context"
	"errors"
	"strings"

	gatewayheartbeat "dreamcreator/internal/application/gateway/heartbeat"
)

type HeartbeatHandler struct {
	service *gatewayheartbeat.Service
}

type HeartbeatLastStatus struct {
	Available  bool   `json:"available"`
	SessionKey string `json:"sessionKey"`
	ThreadID   string `json:"threadId"`
	Status     string `json:"status"`
	Message    string `json:"message"`
	Error      string `json:"error"`
	Reason     string `json:"reason"`
	Indicator  string `json:"indicator"`
	CreatedAt  string `json:"createdAt"`
}

func NewHeartbeatHandler(service *gatewayheartbeat.Service) *HeartbeatHandler {
	return &HeartbeatHandler{service: service}
}

func (handler *HeartbeatHandler) ServiceName() string {
	return "HeartbeatHandler"
}

func (handler *HeartbeatHandler) GetLast(ctx context.Context, sessionKey string) (HeartbeatLastStatus, error) {
	if handler == nil || handler.service == nil {
		return HeartbeatLastStatus{}, nil
	}
	trimmedSessionKey := strings.TrimSpace(sessionKey)
	if trimmedSessionKey == "" {
		return HeartbeatLastStatus{}, nil
	}
	event, err := handler.service.Last(ctx, trimmedSessionKey)
	if err != nil {
		if errors.Is(err, gatewayheartbeat.ErrEventNotFound) {
			return HeartbeatLastStatus{}, nil
		}
		return HeartbeatLastStatus{}, err
	}
	return HeartbeatLastStatus{
		Available:  true,
		SessionKey: event.SessionKey,
		ThreadID:   event.ThreadID,
		Status:     string(event.Status),
		Message:    event.Message,
		Error:      event.Error,
		Reason:     event.Reason,
		Indicator:  string(event.Indicator),
		CreatedAt:  event.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}
