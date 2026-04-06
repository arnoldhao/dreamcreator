package agentruntime

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/cloudwego/eino/schema"
)

func TestToolExecutorEmitsEventsAndToolMessage(t *testing.T) {
	events := make([]Event, 0)
	executor := &ToolExecutor{
		Validator: JSONToolValidator{},
		Tools: map[string]ToolDefinition{
			"echo": {
				Name: "echo",
				Invoke: func(_ context.Context, args string) (string, error) {
					return args, nil
				},
			},
		},
		Emit: func(event Event) {
			events = append(events, event)
		},
	}

	messages, err := executor.Execute(context.Background(), 1, []schema.ToolCall{
		{
			ID:   "call-1",
			Type: "function",
			Function: schema.FunctionCall{
				Name:      "echo",
				Arguments: `{"hello":"world"}`,
			},
		},
	})
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("expected one tool message, got %d", len(messages))
	}
	if len(events) < 4 {
		t.Fatalf("expected emitted tool events, got %d", len(events))
	}
	if events[len(events)-1].Type != EventToolResult {
		t.Fatalf("expected final tool result event, got %s", events[len(events)-1].Type)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(messages[0].Content), &payload); err != nil {
		t.Fatalf("tool output should be json object: %v", err)
	}
}
