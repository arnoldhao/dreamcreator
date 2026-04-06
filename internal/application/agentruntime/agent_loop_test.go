package agentruntime

import (
	"context"
	"errors"
	"io"
	"sync"
	"testing"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

func TestAgentLoopCompletesSingleTurn(t *testing.T) {
	loop := &AgentLoop{
		StreamFunction: func(_ context.Context, _ []*schema.Message, _ ...model.Option) (*schema.StreamReader[*schema.Message], error) {
			return streamMessages(&schema.Message{
				Role:    schema.Assistant,
				Content: "hello",
			}), nil
		},
	}

	stream, err := loop.RunStream(context.Background(), AgentState{})
	if err != nil {
		t.Fatalf("run stream failed: %v", err)
	}
	events := collectEvents(t, stream)
	if !containsEvent(events, EventRunStart) {
		t.Fatalf("missing run_start event")
	}
	if !containsEvent(events, EventTextDelta) {
		t.Fatalf("missing text_delta event")
	}
	if !containsEvent(events, EventRunEnd) {
		t.Fatalf("missing run_end event")
	}
}

func TestAgentLoopSupportsToolRoundtrip(t *testing.T) {
	var (
		mu    sync.Mutex
		calls int
	)
	loop := &AgentLoop{
		StreamFunction: func(_ context.Context, _ []*schema.Message, _ ...model.Option) (*schema.StreamReader[*schema.Message], error) {
			mu.Lock()
			calls++
			current := calls
			mu.Unlock()
			if current == 1 {
				return streamMessages(&schema.Message{
					Role: schema.Assistant,
					ToolCalls: []schema.ToolCall{
						{
							ID:   "call-1",
							Type: "function",
							Function: schema.FunctionCall{
								Name:      "echo",
								Arguments: `{"value":"ok"}`,
							},
						},
					},
				}), nil
			}
			return streamMessages(&schema.Message{
				Role:    schema.Assistant,
				Content: "done",
			}), nil
		},
		ToolExecutor: &ToolExecutor{
			Validator: JSONToolValidator{},
			Tools: map[string]ToolDefinition{
				"echo": {
					Name: "echo",
					Invoke: func(_ context.Context, args string) (string, error) {
						return args, nil
					},
				},
			},
		},
	}

	stream, err := loop.RunStream(context.Background(), AgentState{})
	if err != nil {
		t.Fatalf("run stream failed: %v", err)
	}
	events := collectEvents(t, stream)
	if !containsEvent(events, EventToolCallStart) {
		t.Fatalf("missing tool_call_start")
	}
	if !containsEvent(events, EventToolResult) {
		t.Fatalf("missing tool_result")
	}
	if !containsEvent(events, EventRunEnd) {
		t.Fatalf("missing run_end")
	}
}

func TestAgentLoopContinuesWhenSteeringQueuedAfterToolExecution(t *testing.T) {
	controller := NewAgentController()
	var (
		mu          sync.Mutex
		calls       int
		secondInput []*schema.Message
	)
	loop := &AgentLoop{
		Controller: controller,
		StreamFunction: func(_ context.Context, input []*schema.Message, _ ...model.Option) (*schema.StreamReader[*schema.Message], error) {
			mu.Lock()
			calls++
			current := calls
			if current == 2 {
				secondInput = cloneMessages(input)
			}
			mu.Unlock()
			if current == 1 {
				return streamMessages(&schema.Message{
					Role: schema.Assistant,
					ToolCalls: []schema.ToolCall{
						{
							ID:   "call-1",
							Type: "function",
							Function: schema.FunctionCall{
								Name:      "echo",
								Arguments: `{"value":"ok"}`,
							},
						},
					},
				}), nil
			}
			return streamMessages(&schema.Message{
				Role:    schema.Assistant,
				Content: "done",
			}), nil
		},
		ToolExecutor: &ToolExecutor{
			Validator: JSONToolValidator{},
			Tools: map[string]ToolDefinition{
				"echo": {
					Name: "echo",
					Invoke: func(_ context.Context, args string) (string, error) {
						controller.Steer("继续完成任务")
						return args, nil
					},
				},
			},
		},
	}

	stream, err := loop.RunStream(context.Background(), AgentState{})
	if err != nil {
		t.Fatalf("run stream failed: %v", err)
	}
	events := collectEvents(t, stream)
	if !containsStatusKind(events, "steer") {
		t.Fatalf("missing steer status event")
	}
	if !containsEvent(events, EventRunEnd) {
		t.Fatalf("missing run_end")
	}
	if !containsUserContent(secondInput, "继续完成任务") {
		t.Fatalf("expected steering message in second llm input")
	}
}

func TestAgentLoopContinuesWhenFollowUpQueuedAfterTurn(t *testing.T) {
	controller := NewAgentController()
	var (
		mu    sync.Mutex
		calls int
	)
	loop := &AgentLoop{
		Controller: controller,
		StreamFunction: func(_ context.Context, _ []*schema.Message, _ ...model.Option) (*schema.StreamReader[*schema.Message], error) {
			mu.Lock()
			calls++
			current := calls
			mu.Unlock()
			if current == 1 {
				controller.FollowUp("补充一个问题")
				return streamMessages(&schema.Message{
					Role:    schema.Assistant,
					Content: "first",
				}), nil
			}
			return streamMessages(&schema.Message{
				Role:    schema.Assistant,
				Content: "second",
			}), nil
		},
	}

	stream, err := loop.RunStream(context.Background(), AgentState{})
	if err != nil {
		t.Fatalf("run stream failed: %v", err)
	}
	events := collectEvents(t, stream)
	if !containsStatusKind(events, "follow_up") {
		t.Fatalf("missing follow_up status event")
	}
	if got := countEvent(events, EventStepStart); got < 2 {
		t.Fatalf("expected at least 2 steps, got %d", got)
	}
	if !containsEvent(events, EventRunEnd) {
		t.Fatalf("missing run_end")
	}
}

func TestAgentLoopAbortTakesPriorityOverQueueDrain(t *testing.T) {
	controller := NewAgentController()
	loop := &AgentLoop{
		Controller: controller,
		StreamFunction: func(_ context.Context, _ []*schema.Message, _ ...model.Option) (*schema.StreamReader[*schema.Message], error) {
			return streamMessages(&schema.Message{
				Role: schema.Assistant,
				ToolCalls: []schema.ToolCall{
					{
						ID:   "call-1",
						Type: "function",
						Function: schema.FunctionCall{
							Name:      "echo",
							Arguments: `{"value":"ok"}`,
						},
					},
				},
			}), nil
		},
		ToolExecutor: &ToolExecutor{
			Validator: JSONToolValidator{},
			Tools: map[string]ToolDefinition{
				"echo": {
					Name: "echo",
					Invoke: func(_ context.Context, args string) (string, error) {
						controller.Abort("manual stop")
						return args, nil
					},
				},
			},
		},
	}

	stream, err := loop.RunStream(context.Background(), AgentState{})
	if err != nil {
		t.Fatalf("run stream failed: %v", err)
	}
	events := collectEvents(t, stream)
	if !containsEvent(events, EventRunAbort) {
		t.Fatalf("missing run_abort")
	}
	if containsEvent(events, EventRunEnd) {
		t.Fatalf("run_end should not exist after abort")
	}
}

func TestAgentLoopErrorTakesPriorityOverQueueDrain(t *testing.T) {
	controller := NewAgentController()
	controller.FollowUp("this should be ignored")
	loop := &AgentLoop{
		Controller: controller,
		StreamFunction: func(_ context.Context, _ []*schema.Message, _ ...model.Option) (*schema.StreamReader[*schema.Message], error) {
			return streamMessages(&schema.Message{
				Role: schema.Assistant,
				ToolCalls: []schema.ToolCall{
					{
						ID:   "call-1",
						Type: "function",
						Function: schema.FunctionCall{
							Name:      "echo",
							Arguments: `{"value":"ok"}`,
						},
					},
				},
			}), nil
		},
	}

	stream, err := loop.RunStream(context.Background(), AgentState{})
	if err != nil {
		t.Fatalf("run stream failed: %v", err)
	}
	events, streamErr := collectEventsWithErr(stream)
	if streamErr == nil {
		t.Fatalf("expected stream error")
	}
	if !errors.Is(streamErr, ErrToolExecutorRequired) {
		t.Fatalf("expected ErrToolExecutorRequired, got %v", streamErr)
	}
	if !containsEvent(events, EventRunError) {
		t.Fatalf("missing run_error")
	}
	if containsEvent(events, EventRunEnd) {
		t.Fatalf("run_end should not exist after run_error")
	}
}

func TestAgentLoopSupportsLongToolChainsWithoutStepCap(t *testing.T) {
	const toolTurns = 80
	var (
		mu    sync.Mutex
		calls int
	)
	loop := &AgentLoop{
		StreamFunction: func(_ context.Context, _ []*schema.Message, _ ...model.Option) (*schema.StreamReader[*schema.Message], error) {
			mu.Lock()
			calls++
			current := calls
			mu.Unlock()
			if current <= toolTurns {
				return streamMessages(&schema.Message{
					Role: schema.Assistant,
					ToolCalls: []schema.ToolCall{
						{
							ID:   "call-1",
							Type: "function",
							Function: schema.FunctionCall{
								Name:      "echo",
								Arguments: `{"value":"ok"}`,
							},
						},
					},
				}), nil
			}
			return streamMessages(&schema.Message{
				Role:    schema.Assistant,
				Content: "done",
			}), nil
		},
		ToolExecutor: &ToolExecutor{
			Validator: JSONToolValidator{},
			Tools: map[string]ToolDefinition{
				"echo": {
					Name: "echo",
					Invoke: func(_ context.Context, args string) (string, error) {
						return args, nil
					},
				},
			},
		},
	}

	stream, err := loop.RunStream(context.Background(), AgentState{})
	if err != nil {
		t.Fatalf("run stream failed: %v", err)
	}
	events := collectEvents(t, stream)
	if !containsEvent(events, EventRunEnd) {
		t.Fatalf("missing run_end")
	}
	if containsEvent(events, EventRunError) {
		t.Fatalf("unexpected run_error")
	}
	if got := countEvent(events, EventStepStart); got < toolTurns+1 {
		t.Fatalf("expected at least %d steps, got %d", toolTurns+1, got)
	}
}

func TestLoopAccumulatorMergesResponseMetaAcrossChunks(t *testing.T) {
	t.Parallel()

	acc := newLoopAccumulator()
	acc.consume(&schema.Message{
		Role: schema.Assistant,
		ResponseMeta: &schema.ResponseMeta{
			FinishReason: "stop",
		},
	})
	acc.consume(&schema.Message{
		Role: schema.Assistant,
		ResponseMeta: &schema.ResponseMeta{
			Usage: &schema.TokenUsage{
				PromptTokens:     12,
				CompletionTokens: 8,
				TotalTokens:      20,
			},
		},
	})

	message, _ := acc.buildAssistantMessage()
	if message.ResponseMeta == nil {
		t.Fatalf("expected response meta")
	}
	if message.ResponseMeta.FinishReason != "stop" {
		t.Fatalf("expected finish reason stop, got %q", message.ResponseMeta.FinishReason)
	}
	if message.ResponseMeta.Usage == nil {
		t.Fatalf("expected usage to be preserved")
	}
	if message.ResponseMeta.Usage.TotalTokens != 20 {
		t.Fatalf("expected total tokens 20, got %d", message.ResponseMeta.Usage.TotalTokens)
	}
}

func streamMessages(messages ...*schema.Message) *schema.StreamReader[*schema.Message] {
	reader, writer := schema.Pipe[*schema.Message](8)
	go func() {
		defer writer.Close()
		for _, message := range messages {
			_ = writer.Send(message, nil)
		}
	}()
	return reader
}

func collectEvents(t *testing.T, stream *schema.StreamReader[*schema.Message]) []Event {
	t.Helper()
	events, err := collectEventsWithErr(stream)
	if err != nil {
		t.Fatalf("unexpected stream error: %v", err)
	}
	return events
}

func collectEventsWithErr(stream *schema.StreamReader[*schema.Message]) ([]Event, error) {
	events := make([]Event, 0)
	for {
		message, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				return events, nil
			}
			return events, err
		}
		event, ok := ParseEventMessage(message)
		if !ok {
			continue
		}
		events = append(events, event)
	}
}

func containsEvent(events []Event, target EventType) bool {
	for _, event := range events {
		if event.Type == target {
			return true
		}
	}
	return false
}

func countEvent(events []Event, target EventType) int {
	count := 0
	for _, event := range events {
		if event.Type == target {
			count++
		}
	}
	return count
}

func containsStatusKind(events []Event, kind string) bool {
	for _, event := range events {
		if event.Type != EventStatus {
			continue
		}
		if event.Metadata == nil {
			continue
		}
		value, _ := event.Metadata["kind"].(string)
		if value == kind {
			return true
		}
	}
	return false
}

func containsUserContent(messages []*schema.Message, expected string) bool {
	for _, message := range messages {
		if message == nil {
			continue
		}
		if message.Role != schema.User {
			continue
		}
		if message.Content == expected {
			return true
		}
	}
	return false
}
