package dto

import "time"

type RuntimeRunRequest struct {
	RunID       string              `json:"runId,omitempty"`
	SessionID   string              `json:"sessionId,omitempty"`
	SessionKey  string              `json:"sessionKey,omitempty"`
	AgentID     string              `json:"agentId,omitempty"`
	AssistantID string              `json:"assistantId,omitempty"`
	PromptMode  string              `json:"promptMode,omitempty"`
	RunKind     string              `json:"runKind,omitempty"`
	Input       RuntimeInput        `json:"input"`
	Model       *ModelSelection     `json:"model,omitempty"`
	Thinking    ThinkingConfig      `json:"thinking,omitempty"`
	Tools       ToolExecutionConfig `json:"tools,omitempty"`
	Metadata    map[string]any      `json:"metadata,omitempty"`
}

type RuntimeInput struct {
	Messages       []Message           `json:"messages"`
	Attachments    []RuntimeAttachment `json:"attachments,omitempty"`
	ReplaceHistory bool                `json:"replaceHistory,omitempty"`
}

type ModelSelection struct {
	ProviderID string `json:"providerId,omitempty"`
	Name       string `json:"name,omitempty"`
}

type RuntimeAttachment struct {
	ID       string `json:"id,omitempty"`
	Kind     string `json:"kind,omitempty"`
	Name     string `json:"name,omitempty"`
	MimeType string `json:"mimeType,omitempty"`
	Path     string `json:"path,omitempty"`
	Size     int64  `json:"size,omitempty"`
}

type ThinkingConfig struct {
	Enabled bool   `json:"enabled,omitempty"`
	Mode    string `json:"mode,omitempty"`
}

type ToolExecutionConfig struct {
	Mode            string   `json:"mode,omitempty"`
	AllowList       []string `json:"allowList,omitempty"`
	DenyList        []string `json:"denyList,omitempty"`
	RequireSandbox  bool     `json:"requireSandbox,omitempty"`
	RequireApproval bool     `json:"requireApproval,omitempty"`
}

type ToolInvocation struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
	Args string `json:"args,omitempty"`
}

type ToolInvocationResult struct {
	ID     string `json:"id,omitempty"`
	Name   string `json:"name,omitempty"`
	Output string `json:"output,omitempty"`
	Error  string `json:"error,omitempty"`
}

type RuntimeRunResult struct {
	Status           string          `json:"status"`
	AssistantMessage Message         `json:"assistantMessage,omitempty"`
	FinishReason     string          `json:"finishReason,omitempty"`
	Usage            RuntimeUsage    `json:"usage,omitempty"`
	Model            *ModelSelection `json:"model,omitempty"`
	Error            string          `json:"error,omitempty"`
	ErrorDetail      *RuntimeError   `json:"errorDetail,omitempty"`
	FinishedAt       time.Time       `json:"finishedAt"`
}

const (
	RuntimeStreamEventDelta      = "delta"
	RuntimeStreamEventToolStart  = "tool_start"
	RuntimeStreamEventToolResult = "tool_result"
	RuntimeStreamEventEnd        = "end"
	RuntimeStreamEventError      = "error"
)

type RuntimeStreamEvent struct {
	Type         string       `json:"type"`
	Delta        string       `json:"delta,omitempty"`
	ToolName     string       `json:"toolName,omitempty"`
	ToolCallID   string       `json:"toolCallId,omitempty"`
	FinishReason string       `json:"finishReason,omitempty"`
	Usage        RuntimeUsage `json:"usage,omitempty"`
	Error        string       `json:"error,omitempty"`
}

type RuntimeStreamCallback func(event RuntimeStreamEvent)

type RuntimeEvent struct {
	EventID    string    `json:"eventId,omitempty"`
	Type       string    `json:"type"`
	RunID      string    `json:"runId,omitempty"`
	SessionID  string    `json:"sessionId,omitempty"`
	SessionKey string    `json:"sessionKey,omitempty"`
	Payload    any       `json:"payload,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
}

type RuntimeUsage struct {
	PromptTokens        int `json:"promptTokens,omitempty"`
	CompletionTokens    int `json:"completionTokens,omitempty"`
	TotalTokens         int `json:"totalTokens,omitempty"`
	ContextPromptTokens int `json:"contextPromptTokens,omitempty"`
	ContextTotalTokens  int `json:"contextTotalTokens,omitempty"`
	ContextWindowTokens int `json:"contextWindowTokens,omitempty"`
}

type RuntimeError struct {
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
	Detail  string `json:"detail,omitempty"`
}

type RuntimeStartResponse struct {
	RunID  string `json:"runId"`
	Status string `json:"status"`
}
