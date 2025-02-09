package events

import (
	"sync"
)

type Event struct {
	Topic string
	Data  interface{}
}

type EventBus interface {
	Publish(topic string, data interface{})
	Subscribe(topic string, handler func(Event))
	Unsubscribe(topic string, handler func(Event))
}

type eventBus struct {
	subscribers map[string][]func(Event)
	mu          sync.RWMutex
}

func NewEventBus() EventBus {
	return &eventBus{
		subscribers: make(map[string][]func(Event)),
	}
}

func (eb *eventBus) Publish(topic string, data interface{}) {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	event := Event{
		Topic: topic,
		Data:  data,
	}

	if handlers, exists := eb.subscribers[topic]; exists {
		for _, handler := range handlers {
			go handler(event) // Run handlers concurrently
		}
	}
}

func (eb *eventBus) Subscribe(topic string, handler func(Event)) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	eb.subscribers[topic] = append(eb.subscribers[topic], handler)
}

func (eb *eventBus) Unsubscribe(topic string, handler func(Event)) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	if handlers, exists := eb.subscribers[topic]; exists {
		for i, h := range handlers {
			if &h == &handler {
				eb.subscribers[topic] = append(handlers[:i], handlers[i+1:]...)
				break
			}
		}
		// Remove the topic if no handlers left
		if len(eb.subscribers[topic]) == 0 {
			delete(eb.subscribers, topic)
		}
	}
}
