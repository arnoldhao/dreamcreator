package approvals

import (
	"context"
	"strings"
	"time"

	gatewayevents "dreamcreator/internal/application/gateway/events"
)

type GatewayEventPublisher struct {
	events *gatewayevents.Broker
}

func NewGatewayEventPublisher(events *gatewayevents.Broker) *GatewayEventPublisher {
	return &GatewayEventPublisher{events: events}
}

func (publisher *GatewayEventPublisher) Publish(ctx context.Context, eventType string, payload any) error {
	if publisher == nil || publisher.events == nil {
		return nil
	}
	sessionKey := resolveSessionKey(payload)
	envelope := gatewayevents.Envelope{
		Type:       eventType,
		Topic:      "exec.approval",
		SessionID:  sessionKey,
		SessionKey: sessionKey,
		Timestamp:  time.Now(),
	}
	_, err := publisher.events.Publish(ctx, envelope, payload)
	return err
}

func resolveSessionKey(payload any) string {
	switch value := payload.(type) {
	case Request:
		return strings.TrimSpace(value.SessionKey)
	case *Request:
		if value == nil {
			return ""
		}
		return strings.TrimSpace(value.SessionKey)
	default:
		return ""
	}
}
