package agentruntime

import "github.com/cloudwego/eino/schema"

// AgentState stores the mutable runtime state for a single agent loop run.
type AgentState struct {
	SystemPrompt     string
	Model            string
	Tools            []string
	Messages         []*schema.Message
	IsStreaming      bool
	StreamMessage    string
	PendingToolCalls []schema.ToolCall
	Error            string
	LastFinishReason string
	CurrentLoopStep  int
	CurrentMessageID string
}
