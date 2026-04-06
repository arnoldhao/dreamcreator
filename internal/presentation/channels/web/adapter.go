package web

import (
	"dreamcreator/internal/application/agentruntime"
	"dreamcreator/internal/application/chatevent"
)

type Adapter struct{}

func NewAdapter() *Adapter {
	return &Adapter{}
}

func (adapter *Adapter) Name() string {
	return "web"
}

func (adapter *Adapter) EncodeAgentEvent(event agentruntime.Event) (chatevent.Event, error) {
	return agentruntime.EncodeChatEvent(event)
}
