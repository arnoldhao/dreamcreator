package automation

type HookDefinition struct {
	HookID  string           `json:"hookId"`
	Trigger HookTrigger      `json:"trigger"`
	Action  AutomationAction `json:"action"`
	Enabled bool             `json:"enabled"`
}

type HookTrigger struct {
	EventType string            `json:"eventType"`
	Filters   map[string]string `json:"filters,omitempty"`
}

type AutomationAction struct {
	Type       string `json:"type"`
	SessionKey string `json:"sessionKey"`
	Payload    any    `json:"payload,omitempty"`
}

type HeartbeatConfig struct {
	Enabled          bool   `json:"enabled"`
	IntervalMs       int    `json:"intervalMs"`
	TargetSessionKey string `json:"targetSessionKey,omitempty"`
	Template         string `json:"template,omitempty"`
}

type HeartbeatTickEvent struct {
	Timestamp string `json:"timestamp"`
}
