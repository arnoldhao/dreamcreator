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

func TestToolExecutorKeepsRawToolOutputForEventsAndPrompt(t *testing.T) {
	events := make([]Event, 0)
	executor := &ToolExecutor{
		Validator: JSONToolValidator{},
		Tools: map[string]ToolDefinition{
			"browser": {
				Name: "browser",
				Invoke: func(_ context.Context, args string) (string, error) {
					return `{"url":"https://example.com","itemCount":42,"items":[{"ref":"e1","role":"link","name":"Home"}]}`, nil
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
				Name:      "browser",
				Arguments: `{"action":"snapshot"}`,
			},
		},
	})
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("expected one tool message, got %d", len(messages))
	}
	if got := messages[0].Content; got != `{"url":"https://example.com","itemCount":42,"items":[{"ref":"e1","role":"link","name":"Home"}]}` {
		t.Fatalf("unexpected prompt output: %s", got)
	}
	if len(events) == 0 {
		t.Fatal("expected emitted events")
	}
	last := events[len(events)-1]
	if last.Type != EventToolResult {
		t.Fatalf("expected final tool result event, got %s", last.Type)
	}
	var payload map[string]any
	if err := json.Unmarshal(last.ToolOutput, &payload); err != nil {
		t.Fatalf("tool event output should be raw json: %v", err)
	}
	if payload["itemCount"] != float64(42) {
		t.Fatalf("expected raw event output, got %+v", payload)
	}
}
