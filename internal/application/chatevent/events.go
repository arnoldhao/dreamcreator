package chatevent

import "encoding/json"

type Event struct {
	Type           string          `json:"type"`
	ID             string          `json:"id,omitempty"`
	MessageID      string          `json:"messageId,omitempty"`
	Delta          string          `json:"delta,omitempty"`
	ToolCallID     string          `json:"toolCallId,omitempty"`
	ToolName       string          `json:"toolName,omitempty"`
	ToolDisplayName string         `json:"toolDisplayName,omitempty"`
	Input          json.RawMessage `json:"input,omitempty"`
	InputTextDelta string          `json:"inputTextDelta,omitempty"`
	Output         json.RawMessage `json:"output,omitempty"`
	ErrorText      string          `json:"errorText,omitempty"`
	FinishReason   string          `json:"finishReason,omitempty"`
	SourceID       string          `json:"sourceId,omitempty"`
	URL            string          `json:"url,omitempty"`
	Title          string          `json:"title,omitempty"`
	MediaType      string          `json:"mediaType,omitempty"`
	Filename       string          `json:"filename,omitempty"`
	Data           json.RawMessage `json:"data,omitempty"`
	Transient      *bool           `json:"transient,omitempty"`
	Reason         string          `json:"reason,omitempty"`

	MessageMetadata  map[string]any  `json:"messageMetadata,omitempty"`
	ProviderMetadata json.RawMessage `json:"providerMetadata,omitempty"`
}
