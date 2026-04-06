package wails

import (
	"context"
	"time"

	"dreamcreator/internal/application/events"
	"dreamcreator/internal/infrastructure/ws"
	"github.com/wailsapp/wails/v3/pkg/services/notifications"
)

type RealtimeHandler struct {
	server   *ws.Server
	bus      events.Bus
	notifier *notifications.NotificationService
}

func NewRealtimeHandler(server *ws.Server, bus events.Bus, notifier *notifications.NotificationService) *RealtimeHandler {
	return &RealtimeHandler{server: server, bus: bus, notifier: notifier}
}

func (handler *RealtimeHandler) ServiceName() string {
	return "RealtimeHandler"
}

func (handler *RealtimeHandler) WebSocketURL(_ context.Context) (string, error) {
	return handler.server.URL(), nil
}

func (handler *RealtimeHandler) HTTPBaseURL(_ context.Context) (string, error) {
	return handler.server.HTTPURL(), nil
}

func (handler *RealtimeHandler) PublishDebugEvent(ctx context.Context, topic string, payload any) error {
	if handler.bus == nil {
		return nil
	}
	return handler.bus.Publish(ctx, events.Event{
		Topic:   topic,
		Type:    "debug",
		Payload: payload,
	})
}

type SystemNotificationRequest struct {
	ID       string                 `json:"id"`
	Title    string                 `json:"title"`
	Subtitle string                 `json:"subtitle"`
	Body     string                 `json:"body"`
	Data     map[string]interface{} `json:"data"`
}

func (handler *RealtimeHandler) RequestSystemNotificationAuthorization(_ context.Context) (bool, error) {
	if handler.notifier == nil {
		return false, nil
	}
	return handler.notifier.RequestNotificationAuthorization()
}

func (handler *RealtimeHandler) SendSystemNotification(ctx context.Context, request SystemNotificationRequest) error {
	if handler.notifier == nil {
		return nil
	}

	id := request.ID
	if id == "" {
		id = time.Now().Format(time.RFC3339Nano)
	}

	return handler.notifier.SendNotification(notifications.NotificationOptions{
		ID:       id,
		Title:    request.Title,
		Subtitle: request.Subtitle,
		Body:     request.Body,
		Data:     request.Data,
	})
}
