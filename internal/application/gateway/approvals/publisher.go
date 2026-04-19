package approvals

import (
	"context"
	"strings"
	"time"

	gatewayevents "dreamcreator/internal/application/gateway/events"
	domainsession "dreamcreator/internal/domain/session"
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
	sessionID, sessionKey := resolveSessionIdentity(payload)
	envelope := gatewayevents.Envelope{
		Type:       eventType,
		Topic:      "exec.approval",
		SessionID:  sessionID,
		SessionKey: sessionKey,
		Timestamp:  time.Now(),
	}
	_, err := publisher.events.Publish(ctx, envelope, payload)
	return err
}

func resolveSessionIdentity(payload any) (string, string) {
	sessionKey := ""
	switch value := payload.(type) {
	case Request:
		sessionKey = strings.TrimSpace(value.SessionKey)
	case *Request:
		if value == nil {
			return "", ""
		}
		sessionKey = strings.TrimSpace(value.SessionKey)
	default:
		return "", ""
	}
	if parts, _, err := domainsession.NormalizeSessionKey(sessionKey); err == nil {
		sessionID := strings.TrimSpace(parts.ThreadRef)
		if sessionID == "" {
			sessionID = strings.TrimSpace(parts.PrimaryID)
		}
		if sessionID != "" {
			return sessionID, sessionKey
		}
	}
	return sessionKey, sessionKey
}
