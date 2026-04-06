package runtime

import (
	"strings"
	"testing"

	"github.com/cloudwego/eino/schema"
)

func TestCollectPromptMessages_IncludesSystemAndInputMessages(t *testing.T) {
	t.Parallel()

	messages := []*schema.Message{
		{
			Role:    schema.User,
			Content: "user question",
		},
		{
			Role:             schema.Assistant,
			Content:          "assistant answer",
			ReasoningContent: "assistant reasoning",
		},
		{
			Role: schema.Assistant,
			ToolCalls: []schema.ToolCall{
				{
					ID:   "call_1",
					Type: "function",
					Function: schema.FunctionCall{
						Name:      "web_search",
						Arguments: `{"q":"hello"}`,
					},
				},
			},
		},
		schema.ToolMessage(`{"ok":true}`, "call_1", schema.WithToolName("web_search")),
	}

	result := collectPromptMessages("system prompt content", messages)
	if len(result) != 5 {
		t.Fatalf("expected 5 messages, got %d", len(result))
	}

	if result[0].Role != "system" || result[0].Content != "system prompt content" {
		t.Fatalf("unexpected system snapshot: %+v", result[0])
	}
	if result[1].Role != "user" || result[1].Content != "user question" {
		t.Fatalf("unexpected user snapshot: %+v", result[1])
	}
	if result[2].Role != "assistant" || result[2].Reasoning != "assistant reasoning" {
		t.Fatalf("unexpected assistant snapshot: %+v", result[2])
	}
	if result[3].Role != "assistant" || !strings.Contains(result[3].Content, "web_search") {
		t.Fatalf("expected assistant tool-call snapshot, got %+v", result[3])
	}
	if result[4].Role != "tool" || result[4].ToolCallID != "call_1" {
		t.Fatalf("unexpected tool snapshot: %+v", result[4])
	}
}

func TestCollectPromptMessages_EmptyInputReturnsNil(t *testing.T) {
	t.Parallel()

	result := collectPromptMessages("", []*schema.Message{
		{
			Role:    schema.User,
			Content: "   ",
		},
		nil,
	})
	if result != nil {
		t.Fatalf("expected nil result, got %+v", result)
	}
}
