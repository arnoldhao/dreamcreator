package agentruntime

import (
	"encoding/json"
	"strings"

	"github.com/cloudwego/eino/schema"
)

const eventExtraKey = "agent_event"

type EventType string

const (
	EventRunStart       EventType = "run_start"
	EventRunEnd         EventType = "run_end"
	EventRunError       EventType = "run_error"
	EventRunAbort       EventType = "run_abort"
	EventPromptReport   EventType = "prompt_report"
	EventStatus         EventType = "status"
	EventStepStart      EventType = "step_start"
	EventStepEnd        EventType = "step_end"
	EventTextDelta      EventType = "text_delta"
	EventReasoningDelta EventType = "reasoning_delta"
	EventToolCallStart  EventType = "tool_call_start"
	EventToolCallDelta  EventType = "tool_call_delta"
	EventToolCallReady  EventType = "tool_call_ready"
	EventToolResult     EventType = "tool_result"
	EventToolError      EventType = "tool_error"
	EventContextSnapshot EventType = "context_snapshot"
	EventToolLoopWarning EventType = "tool_loop_warning"
)

type Event struct {
	Type          EventType          `json:"event"`
	RunID         string             `json:"runId,omitempty"`
	ThreadID      string             `json:"threadId,omitempty"`
	MessageID     string             `json:"messageId,omitempty"`
	Step          int                `json:"step,omitempty"`
	Delta         string             `json:"delta,omitempty"`
	ToolCallID    string             `json:"toolCallId,omitempty"`
	ToolName      string             `json:"toolName,omitempty"`
	ToolType      string             `json:"toolType,omitempty"`
	ToolArgs      json.RawMessage    `json:"toolArgs,omitempty"`
	ToolArgsDelta string             `json:"toolArgsDelta,omitempty"`
	ToolOutput    json.RawMessage    `json:"toolOutput,omitempty"`
	ErrorText     string             `json:"errorText,omitempty"`
	FinishReason  string             `json:"finishReason,omitempty"`
	Attempt       int                `json:"attempt,omitempty"`
	MaxAttempts   int                `json:"maxAttempts,omitempty"`
	Usage         *schema.TokenUsage `json:"usage,omitempty"`
	ContextTokens *ContextTokenSnapshot `json:"contextTokens,omitempty"`
	Metadata      map[string]any     `json:"metadata,omitempty"`
}

type ContextTokenSnapshot struct {
	PromptTokens       int `json:"promptTokens,omitempty"`
	TotalTokens        int `json:"totalTokens,omitempty"`
	ContextLimitTokens int `json:"contextLimitTokens,omitempty"`
	WarnTokens         int `json:"warnTokens,omitempty"`
	HardTokens         int `json:"hardTokens,omitempty"`
}

func BuildEventMessage(event Event) *schema.Message {
	data, err := json.Marshal(event)
	if err != nil {
		return nil
	}
	return &schema.Message{
		Role: schema.Assistant,
		Extra: map[string]any{
			eventExtraKey: string(data),
		},
	}
}

func ParseEventMessage(msg *schema.Message) (Event, bool) {
	if msg == nil || len(msg.Extra) == 0 {
		return Event{}, false
	}
	raw, ok := msg.Extra[eventExtraKey]
	if !ok {
		return Event{}, false
	}
	data := ""
	switch typed := raw.(type) {
	case string:
		data = strings.TrimSpace(typed)
	case []byte:
		data = strings.TrimSpace(string(typed))
	default:
		encoded, err := json.Marshal(typed)
		if err != nil {
			return Event{}, false
		}
		data = strings.TrimSpace(string(encoded))
	}
	if data == "" {
		return Event{}, false
	}
	var event Event
	if err := json.Unmarshal([]byte(data), &event); err != nil {
		return Event{}, false
	}
	event.Type = EventType(strings.TrimSpace(string(event.Type)))
	if event.Type == "" {
		return Event{}, false
	}
	return event, true
}
