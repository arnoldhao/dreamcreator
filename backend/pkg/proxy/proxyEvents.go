package proxy

import (
	"CanMe/backend/pkg/events"
	"time"

	"github.com/google/uuid"
)

type ProxyConfigChangedEvent struct {
	events.BaseEvent
	OldConfig *Config `json:"old_config"`
	NewConfig *Config `json:"new_config"`
}

func NewProxyConfigChangedEvent(source string, oldConfig, newConfig *Config) *ProxyConfigChangedEvent {
	return &ProxyConfigChangedEvent{
		BaseEvent: events.BaseEvent{
			ID:        uuid.New().String(),
			Type:      "proxy.config.changed",
			Source:    source,
			Timestamp: time.Now(),
			Data:      map[string]interface{}{"old": oldConfig, "new": newConfig},
			Metadata:  make(map[string]interface{}),
		},
		OldConfig: oldConfig,
		NewConfig: newConfig,
	}
}
