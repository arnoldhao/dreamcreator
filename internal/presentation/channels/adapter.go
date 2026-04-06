package channels

import "dreamcreator/internal/application/agentruntime"

type InboundMessage struct {
	Channel    string
	ThreadID   string
	UserID     string
	Message    string
	AgentID    string
	ProviderID string
	ModelName  string
	Metadata   map[string]any
}

type OutboundMessage struct {
	Channel  string             `json:"channel"`
	ThreadID string             `json:"threadId,omitempty"`
	RunID    string             `json:"runId,omitempty"`
	Text     string             `json:"text,omitempty"`
	Event    agentruntime.Event `json:"event"`
}

type Adapter interface {
	Name() string
}
