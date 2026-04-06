package events

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"dreamcreator/internal/domain/thread"
)

type RunEventSink struct {
	events *Broker
}

func NewRunEventSink(events *Broker) *RunEventSink {
	return &RunEventSink{events: events}
}

func (sink *RunEventSink) Publish(ctx context.Context, event thread.ThreadRunEvent, sessionKey string) {
	if sink == nil || sink.events == nil {
		return
	}
	payload := json.RawMessage(event.PayloadJSON)
	envelope := Envelope{
		Type:       strings.TrimSpace(event.EventType),
		Topic:      "run",
		SessionID:  strings.TrimSpace(event.ThreadID),
		SessionKey: strings.TrimSpace(sessionKey),
		RunID:      strings.TrimSpace(event.RunID),
		Timestamp:  event.CreatedAt,
	}
	if envelope.Timestamp.IsZero() {
		envelope.Timestamp = time.Now()
	}
	_, _ = sink.events.Publish(ctx, envelope, payload)
}
