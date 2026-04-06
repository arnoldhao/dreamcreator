package chatevent

import "encoding/json"

type PlainMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type MessagePart struct {
	Type       string          `json:"type"`
	ParentID   string          `json:"parentId,omitempty"`
	Text       string          `json:"text,omitempty"`
	State      string          `json:"state,omitempty"`
	ToolCallID string          `json:"toolCallId,omitempty"`
	ToolName   string          `json:"toolName,omitempty"`
	Input      json.RawMessage `json:"input,omitempty"`
	Output     json.RawMessage `json:"output,omitempty"`
	ErrorText  string          `json:"errorText,omitempty"`
	Data       json.RawMessage `json:"data,omitempty"`
}

type Message struct {
	ID    string        `json:"id"`
	Role  string        `json:"role"`
	Parts []MessagePart `json:"parts"`
}
