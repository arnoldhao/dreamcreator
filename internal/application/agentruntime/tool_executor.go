package agentruntime

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/cloudwego/eino/schema"
)

var ErrToolNotFound = errors.New("tool not found")

type ToolDefinition struct {
	Name       string
	Type       string
	SchemaJSON string
	Invoke     func(ctx context.Context, args string) (string, error)
}

type ToolExecutor struct {
	Validator    ToolValidator
	Tools        map[string]ToolDefinition
	Emit         func(Event)
	LoopRecorder ToolLoopRecorder
}

func (executor *ToolExecutor) Execute(
	ctx context.Context,
	step int,
	calls []schema.ToolCall,
) ([]*schema.Message, error) {
	if len(calls) == 0 {
		return nil, nil
	}
	results := make([]*schema.Message, 0, len(calls))
	validator := executor.Validator
	if validator == nil {
		validator = JSONToolValidator{}
	}
	for index, call := range calls {
		if ctx.Err() != nil {
			return results, ctx.Err()
		}
		id := strings.TrimSpace(call.ID)
		if id == "" {
			id = fallbackToolCallID(index)
		}
		name := strings.TrimSpace(call.Function.Name)
		args := strings.TrimSpace(call.Function.Arguments)
		if args == "" {
			args = "{}"
		}

		executor.emit(Event{
			Type:       EventToolCallStart,
			Step:       step,
			ToolCallID: id,
			ToolName:   name,
			ToolType:   strings.TrimSpace(call.Type),
		})
		executor.emit(Event{
			Type:          EventToolCallDelta,
			Step:          step,
			ToolCallID:    id,
			ToolName:      name,
			ToolArgsDelta: args,
		})
		executor.emit(Event{
			Type:       EventToolCallReady,
			Step:       step,
			ToolCallID: id,
			ToolName:   name,
			ToolArgs:   normalizeJSON(args),
		})

		if err := validator.Validate(schema.ToolCall{
			ID:   id,
			Type: call.Type,
			Function: schema.FunctionCall{
				Name:      name,
				Arguments: args,
			},
		}); err != nil {
			executor.recordOutcome(name, args, id, "", err)
			executor.emit(Event{
				Type:       EventToolError,
				Step:       step,
				ToolCallID: id,
				ToolName:   name,
				ErrorText:  err.Error(),
			})
			results = append(results, &schema.Message{
				Role:       schema.Tool,
				ToolCallID: id,
				ToolName:   name,
				Content:    marshalError(err),
			})
			continue
		}

		definition, ok := executor.Tools[name]
		if !ok || definition.Invoke == nil {
			err := errors.New(ErrToolNotFound.Error() + ": " + name)
			executor.recordOutcome(name, args, id, "", err)
			executor.emit(Event{
				Type:       EventToolError,
				Step:       step,
				ToolCallID: id,
				ToolName:   name,
				ErrorText:  err.Error(),
			})
			results = append(results, &schema.Message{
				Role:       schema.Tool,
				ToolCallID: id,
				ToolName:   name,
				Content:    marshalError(err),
			})
			continue
		}

		invokeCtx := WithToolCallContext(ctx, id, name)
		started := time.Now()
		output, err := definition.Invoke(invokeCtx, args)
		elapsed := time.Since(started).Milliseconds()
		if err != nil {
			executor.recordOutcome(name, args, id, "", err)
			executor.emit(Event{
				Type:       EventToolError,
				Step:       step,
				ToolCallID: id,
				ToolName:   name,
				ErrorText:  err.Error(),
				Metadata: map[string]any{
					"elapsedMs": elapsed,
				},
			})
			results = append(results, &schema.Message{
				Role:       schema.Tool,
				ToolCallID: id,
				ToolName:   name,
				Content:    marshalError(err),
			})
			continue
		}
		if strings.TrimSpace(output) == "" {
			output = "null"
		}
		executor.recordOutcome(name, args, id, output, nil)
		executor.emit(Event{
			Type:       EventToolResult,
			Step:       step,
			ToolCallID: id,
			ToolName:   name,
			ToolOutput: normalizeJSON(output),
			Metadata: map[string]any{
				"elapsedMs": elapsed,
			},
		})
		results = append(results, &schema.Message{
			Role:       schema.Tool,
			ToolCallID: id,
			ToolName:   name,
			Content:    output,
		})
	}
	return results, nil
}

type ToolLoopRecorder interface {
	RecordOutcome(toolName string, args string, toolCallID string, output string, err error)
}

func (executor *ToolExecutor) emit(event Event) {
	if executor != nil && executor.Emit != nil {
		executor.Emit(event)
	}
}

func (executor *ToolExecutor) recordOutcome(toolName string, args string, toolCallID string, output string, err error) {
	if executor == nil || executor.LoopRecorder == nil {
		return
	}
	executor.LoopRecorder.RecordOutcome(toolName, args, toolCallID, output, err)
}

func normalizeJSON(value string) json.RawMessage {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return json.RawMessage("null")
	}
	var decoded any
	if err := json.Unmarshal([]byte(trimmed), &decoded); err != nil {
		encoded, _ := json.Marshal(trimmed)
		return encoded
	}
	encoded, err := json.Marshal(decoded)
	if err != nil {
		return json.RawMessage("null")
	}
	return encoded
}

func marshalError(err error) string {
	if err == nil {
		return `{"error":"unknown"}`
	}
	encoded, _ := json.Marshal(map[string]any{
		"error": err.Error(),
	})
	return string(encoded)
}

func fallbackToolCallID(index int) string {
	return "tool-call-" + strconv.Itoa(index+1)
}
