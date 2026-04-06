package events

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	appevents "dreamcreator/internal/application/events"
	"dreamcreator/internal/application/gateway/controlplane"
)

type Envelope = appevents.GatewayEventEnvelope

type Record struct {
	Envelope Envelope        `json:"envelope"`
	Payload  json.RawMessage `json:"payload,omitempty"`
}

type Filter struct {
	SessionID  string
	SessionKey string
	RunID      string
	Topic      string
	Type       string
	Limit      int
}

type Store interface {
	Append(ctx context.Context, record Record) (Record, error)
	Query(ctx context.Context, filter Filter) ([]Record, error)
}

type subscriber struct {
	id      int64
	filter  Filter
	handler func(Record)
}

type Broker struct {
	store       Store
	publisher   controlplane.EventPublisher
	mu          sync.RWMutex
	subscribers map[int64]subscriber
	seq         atomic.Int64
}

func NewBroker(store Store) *Broker {
	return &Broker{
		store:       store,
		subscribers: make(map[int64]subscriber),
	}
}

func (broker *Broker) SetPublisher(publisher controlplane.EventPublisher) {
	if broker == nil {
		return
	}
	broker.mu.Lock()
	broker.publisher = publisher
	broker.mu.Unlock()
}

func (broker *Broker) Publish(ctx context.Context, envelope Envelope, payload any) (Record, error) {
	if broker == nil {
		return Record{}, errors.New("event broker unavailable")
	}
	record, err := buildRecord(envelope, payload)
	if err != nil {
		return Record{}, err
	}
	if record.Envelope.Sequence == 0 {
		record.Envelope.Sequence = broker.seq.Add(1)
	}
	if broker.store != nil {
		stored, err := broker.store.Append(ctx, record)
		if err != nil {
			return Record{}, err
		}
		record = stored
	}
	broker.dispatch(record)
	return record, nil
}

func (broker *Broker) Query(ctx context.Context, filter Filter) ([]Record, error) {
	if broker == nil || broker.store == nil {
		return nil, errors.New("event store unavailable")
	}
	return broker.store.Query(ctx, filter)
}

func (broker *Broker) Subscribe(filter Filter, handler func(Record)) func() {
	if broker == nil || handler == nil {
		return func() {}
	}
	broker.mu.Lock()
	id := broker.seq.Add(1)
	broker.subscribers[id] = subscriber{id: id, filter: filter, handler: handler}
	broker.mu.Unlock()
	return func() {
		broker.mu.Lock()
		delete(broker.subscribers, id)
		broker.mu.Unlock()
	}
}

func (broker *Broker) dispatch(record Record) {
	broker.mu.RLock()
	publisher := broker.publisher
	subs := make([]subscriber, 0, len(broker.subscribers))
	for _, sub := range broker.subscribers {
		subs = append(subs, sub)
	}
	broker.mu.RUnlock()

	for _, sub := range subs {
		if !matchesFilter(record, sub.filter) {
			continue
		}
		sub.handler(record)
	}
	if publisher != nil {
		_ = publisher.Publish(toEventFrame(record))
	}
}

func buildRecord(envelope Envelope, payload any) (Record, error) {
	record := Record{Envelope: envelope}
	if record.Envelope.EventID == "" {
		record.Envelope.EventID = time.Now().Format(time.RFC3339Nano)
	}
	if record.Envelope.Timestamp.IsZero() {
		record.Envelope.Timestamp = time.Now()
	}
	if payload == nil {
		return record, nil
	}
	if raw, ok := payload.(json.RawMessage); ok {
		record.Payload = raw
		return record, nil
	}
	if bytes, ok := payload.([]byte); ok {
		record.Payload = json.RawMessage(bytes)
		return record, nil
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return Record{}, err
	}
	record.Payload = json.RawMessage(data)
	return record, nil
}

func matchesFilter(record Record, filter Filter) bool {
	if filter.SessionID != "" && record.Envelope.SessionID != filter.SessionID {
		return false
	}
	if filter.SessionKey != "" && record.Envelope.SessionKey != filter.SessionKey {
		return false
	}
	if filter.RunID != "" && record.Envelope.RunID != filter.RunID {
		return false
	}
	if filter.Topic != "" && record.Envelope.Topic != filter.Topic {
		return false
	}
	if filter.Type != "" && record.Envelope.Type != filter.Type {
		return false
	}
	return true
}

func toEventFrame(record Record) controlplane.EventFrame {
	payload := any(record.Payload)
	if len(record.Payload) == 0 {
		payload = nil
	} else if json.Valid(record.Payload) {
		payload = json.RawMessage(record.Payload)
	}
	return controlplane.EventFrame{
		Type:       "event",
		Event:      record.Envelope.Type,
		Payload:    payload,
		Seq:        record.Envelope.Sequence,
		Timestamp:  record.Envelope.Timestamp,
		SessionID:  record.Envelope.SessionID,
		SessionKey: record.Envelope.SessionKey,
		RunID:      record.Envelope.RunID,
	}
}
