package agentruntime

import (
	"context"
	"errors"
	"io"
	"strconv"
	"strings"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

var (
	ErrStreamFunctionRequired = errors.New("stream function is required")
	ErrAssistantEmpty         = errors.New("assistant response is empty")
	ErrToolExecutorRequired   = errors.New("tool executor is required")
)

type StreamFunction func(ctx context.Context, messages []*schema.Message, options ...model.Option) (*schema.StreamReader[*schema.Message], error)
type TransformContextHook func(ctx context.Context, state AgentState) (AgentState, error)
type ConvertToLlmHook func(ctx context.Context, state AgentState) ([]*schema.Message, error)

type AgentLoop struct {
	StreamFunction   StreamFunction
	TransformContext TransformContextHook
	ConvertToLlm     ConvertToLlmHook
	ToolExecutor     *ToolExecutor
	Controller       *AgentController
	BuildOptions     func() []model.Option
	Emit             func(Event)
	MaxSteps         int
	ToolLoopDetector *ToolLoopDetector
}

func (loop *AgentLoop) RunStream(ctx context.Context, state AgentState) (*schema.StreamReader[*schema.Message], error) {
	if loop == nil || loop.StreamFunction == nil {
		return nil, ErrStreamFunctionRequired
	}
	reader, writer := schema.Pipe[*schema.Message](64)
	go loop.run(ctx, state, writer)
	return reader, nil
}

func (loop *AgentLoop) run(ctx context.Context, state AgentState, writer *schema.StreamWriter[*schema.Message]) {
	defer writer.Close()
	controller := loop.Controller
	if controller == nil {
		controller = NewAgentController()
	}
	controller.BeginRun()
	defer controller.EndRun()

	runCtx := ctx
	if timeout := controller.Timeout(); timeout > 0 {
		var cancel context.CancelFunc
		runCtx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	history := cloneMessages(state.Messages)
	systemPrompt := strings.TrimSpace(state.SystemPrompt)
	if systemPrompt != "" {
		history = append([]*schema.Message{{Role: schema.System, Content: systemPrompt}}, history...)
	}

	_ = loop.sendEvent(writer, Event{Type: EventRunStart})
	step := 0
	for {
		nextStep := step + 1
		if loop.MaxSteps > 0 && nextStep > loop.MaxSteps {
			_ = loop.sendEvent(writer, Event{
				Type:      EventRunAbort,
				Step:      nextStep,
				ErrorText: "max_steps_exceeded",
			})
			return
		}
		if reason, aborted := controller.Aborted(); aborted {
			_ = loop.sendEvent(writer, Event{
				Type:      EventRunAbort,
				Step:      nextStep,
				ErrorText: strings.TrimSpace(reason),
			})
			return
		}
		if runCtx.Err() != nil {
			_ = loop.sendEvent(writer, Event{
				Type:      EventRunAbort,
				Step:      nextStep,
				ErrorText: runCtx.Err().Error(),
			})
			return
		}
		history, _ = loop.drainQueuedUserMessages(writer, history, controller.NextSteer, "steer")
		step = nextStep

		_ = loop.sendEvent(writer, Event{
			Type: EventStepStart,
			Step: step,
		})
		state.CurrentLoopStep = step
		state.Messages = cloneMessages(history)

		var err error
		if loop.TransformContext != nil {
			state, err = loop.TransformContext(runCtx, state)
			if err != nil {
				_ = loop.sendEvent(writer, Event{
					Type:      EventRunError,
					Step:      step,
					ErrorText: err.Error(),
				})
				_ = writer.Send(nil, err)
				return
			}
		}

		modelInput := cloneMessages(state.Messages)
		if loop.ConvertToLlm != nil {
			modelInput, err = loop.ConvertToLlm(runCtx, state)
			if err != nil {
				_ = loop.sendEvent(writer, Event{
					Type:      EventRunError,
					Step:      step,
					ErrorText: err.Error(),
				})
				_ = writer.Send(nil, err)
				return
			}
		}
		if len(modelInput) == 0 {
			modelInput = cloneMessages(history)
		}

		stream, err := loop.StreamFunction(runCtx, modelInput, loop.resolveOptions()...)
		if err != nil {
			_ = loop.sendEvent(writer, Event{
				Type:      EventRunError,
				Step:      step,
				ErrorText: err.Error(),
			})
			_ = writer.Send(nil, err)
			return
		}

		acc := newLoopAccumulator()
		for {
			msg, recvErr := stream.Recv()
			if recvErr != nil {
				if errors.Is(recvErr, io.EOF) {
					break
				}
				stream.Close()
				_ = loop.sendEvent(writer, Event{
					Type:      EventRunError,
					Step:      step,
					ErrorText: recvErr.Error(),
				})
				_ = writer.Send(nil, recvErr)
				return
			}
			if msg == nil {
				continue
			}
			acc.consume(msg)
			if msg.Content != "" {
				_ = loop.sendEvent(writer, Event{
					Type:  EventTextDelta,
					Step:  step,
					Delta: msg.Content,
				})
			}
			if msg.ReasoningContent != "" {
				_ = loop.sendEvent(writer, Event{
					Type:  EventReasoningDelta,
					Step:  step,
					Delta: msg.ReasoningContent,
				})
			}
		}
		stream.Close()

		assistantMessage, toolCalls := acc.buildAssistantMessage()
		var usage *schema.TokenUsage
		if assistantMessage.ResponseMeta != nil {
			usage = assistantMessage.ResponseMeta.Usage
		}
		history = append(history, assistantMessage)
		if len(toolCalls) == 0 {
			if strings.TrimSpace(assistantMessage.Content) == "" && strings.TrimSpace(assistantMessage.ReasoningContent) == "" {
				_ = loop.sendEvent(writer, Event{
					Type:      EventRunError,
					Step:      step,
					ErrorText: ErrAssistantEmpty.Error(),
				})
				_ = writer.Send(nil, ErrAssistantEmpty)
				return
			}
			finishReason := ""
			if assistantMessage.ResponseMeta != nil {
				finishReason = strings.TrimSpace(assistantMessage.ResponseMeta.FinishReason)
			}
			_ = loop.sendEvent(writer, Event{
				Type:         EventStepEnd,
				Step:         step,
				FinishReason: finishReason,
				Usage:        usage,
			})
			history, hasSteer := loop.drainQueuedUserMessages(writer, history, controller.NextSteer, "steer")
			if hasSteer {
				continue
			}
			history, hasFollowUp := loop.drainQueuedUserMessages(writer, history, controller.NextFollowUp, "follow_up")
			if hasFollowUp {
				continue
			}
			_ = loop.sendEvent(writer, Event{
				Type:         EventRunEnd,
				Step:         step,
				FinishReason: finishReason,
				Usage:        usage,
			})
			return
		}

		if loop.ToolLoopDetector != nil {
			result := loop.ToolLoopDetector.ObserveCalls(toolCalls)
			if result.Stuck {
				metadata := map[string]any{
					"level":    result.Level,
					"detector": result.Detector,
					"count":    result.Count,
					"tool":     result.ToolName,
					"message":  result.Message,
				}
				if result.PairedToolName != "" {
					metadata["pairedTool"] = result.PairedToolName
				}
				if result.WarningKey != "" {
					metadata["warningKey"] = result.WarningKey
				}
				if result.Level == "warning" {
					_ = loop.sendEvent(writer, Event{
						Type:     EventToolLoopWarning,
						Step:     step,
						Metadata: metadata,
					})
				}
				if result.Level == "critical" {
					_ = loop.sendEvent(writer, Event{
						Type:      EventRunAbort,
						Step:      step,
						ErrorText: "tool_loop_detected",
						Metadata:  metadata,
					})
					return
				}
			}
		}

		if loop.ToolExecutor == nil {
			_ = loop.sendEvent(writer, Event{
				Type:      EventRunError,
				Step:      step,
				ErrorText: ErrToolExecutorRequired.Error(),
			})
			_ = writer.Send(nil, ErrToolExecutorRequired)
			return
		}
		executor := *loop.ToolExecutor
		executor.Emit = func(event Event) {
			if event.Step <= 0 {
				event.Step = step
			}
			_ = loop.sendEvent(writer, event)
		}
		if loop.ToolLoopDetector != nil {
			executor.LoopRecorder = loop.ToolLoopDetector
		}
		toolMessages, execErr := executor.Execute(runCtx, step, normalizeToolCalls(toolCalls))
		if execErr != nil {
			_ = loop.sendEvent(writer, Event{
				Type:      EventRunError,
				Step:      step,
				ErrorText: execErr.Error(),
			})
			_ = writer.Send(nil, execErr)
			return
		}
		history = append(history, toolMessages...)
		_ = loop.sendEvent(writer, Event{
			Type:  EventStepEnd,
			Step:  step,
			Usage: usage,
		})
	}
}

func (loop *AgentLoop) resolveOptions() []model.Option {
	if loop == nil || loop.BuildOptions == nil {
		return nil
	}
	return loop.BuildOptions()
}

func (loop *AgentLoop) sendEvent(writer *schema.StreamWriter[*schema.Message], event Event) bool {
	if loop != nil && loop.Emit != nil {
		loop.Emit(event)
	}
	if writer == nil {
		return false
	}
	message := BuildEventMessage(event)
	if message == nil {
		return false
	}
	return writer.Send(message, nil)
}

func (loop *AgentLoop) drainQueuedUserMessages(
	writer *schema.StreamWriter[*schema.Message],
	history []*schema.Message,
	dequeue func() (string, bool),
	kind string,
) ([]*schema.Message, bool) {
	drained := false
	for {
		message, ok := dequeue()
		if !ok {
			break
		}
		history = append(history, &schema.Message{
			Role:    schema.User,
			Content: message,
		})
		_ = loop.sendEvent(writer, Event{
			Type: EventStatus,
			Metadata: map[string]any{
				"kind": kind,
			},
		})
		drained = true
	}
	return history, drained
}

func cloneMessages(messages []*schema.Message) []*schema.Message {
	if len(messages) == 0 {
		return nil
	}
	cloned := make([]*schema.Message, 0, len(messages))
	for _, message := range messages {
		if message == nil {
			continue
		}
		cp := *message
		cloned = append(cloned, &cp)
	}
	return cloned
}

type loopAccumulator struct {
	content      strings.Builder
	reasoning    strings.Builder
	responseMeta *schema.ResponseMeta
	callOrder    []string
	calls        map[string]*loopToolCall
}

type loopToolCall struct {
	id       string
	name     string
	callType string
	index    *int
	args     strings.Builder
}

func newLoopAccumulator() *loopAccumulator {
	return &loopAccumulator{
		calls: make(map[string]*loopToolCall),
	}
}

func (acc *loopAccumulator) consume(message *schema.Message) {
	if message == nil {
		return
	}
	if message.Content != "" {
		acc.content.WriteString(message.Content)
	}
	if message.ReasoningContent != "" {
		acc.reasoning.WriteString(message.ReasoningContent)
	}
	if message.ResponseMeta != nil {
		if acc.responseMeta == nil {
			acc.responseMeta = &schema.ResponseMeta{}
		}
		if message.ResponseMeta.FinishReason != "" {
			acc.responseMeta.FinishReason = message.ResponseMeta.FinishReason
		}
		if message.ResponseMeta.Usage != nil {
			acc.responseMeta.Usage = message.ResponseMeta.Usage
		}
		if message.ResponseMeta.LogProbs != nil {
			acc.responseMeta.LogProbs = message.ResponseMeta.LogProbs
		}
	}
	if len(message.ToolCalls) > 0 {
		acc.consumeToolCalls(message.ToolCalls)
	}
}

func (acc *loopAccumulator) consumeToolCalls(calls []schema.ToolCall) {
	for _, call := range calls {
		key := loopToolCallKey(call)
		state := acc.calls[key]
		if state == nil {
			state = &loopToolCall{}
			acc.calls[key] = state
			acc.callOrder = append(acc.callOrder, key)
		}
		if call.ID != "" {
			state.id = call.ID
		}
		if call.Type != "" {
			state.callType = call.Type
		}
		if call.Index != nil {
			cp := *call.Index
			state.index = &cp
		}
		if call.Function.Name != "" {
			state.name = call.Function.Name
		}
		if call.Function.Arguments != "" {
			state.args.WriteString(call.Function.Arguments)
		}
	}
}

func (acc *loopAccumulator) buildAssistantMessage() (*schema.Message, []schema.ToolCall) {
	toolCalls := acc.buildToolCalls()
	return &schema.Message{
		Role:             schema.Assistant,
		Content:          acc.content.String(),
		ReasoningContent: acc.reasoning.String(),
		ToolCalls:        toolCalls,
		ResponseMeta:     acc.responseMeta,
	}, toolCalls
}

func (acc *loopAccumulator) buildToolCalls() []schema.ToolCall {
	if len(acc.callOrder) == 0 {
		return nil
	}
	result := make([]schema.ToolCall, 0, len(acc.callOrder))
	for idx, key := range acc.callOrder {
		state := acc.calls[key]
		if state == nil {
			continue
		}
		name := strings.TrimSpace(state.name)
		if name == "" {
			continue
		}
		id := strings.TrimSpace(state.id)
		if id == "" {
			id = fallbackToolCallID(idx)
		}
		callType := strings.TrimSpace(state.callType)
		if callType == "" {
			callType = "function"
		}
		args := strings.TrimSpace(state.args.String())
		if args == "" {
			args = "{}"
		}
		call := schema.ToolCall{
			ID:   id,
			Type: callType,
			Function: schema.FunctionCall{
				Name:      name,
				Arguments: args,
			},
		}
		if state.index != nil {
			cp := *state.index
			call.Index = &cp
		}
		result = append(result, call)
	}
	return result
}

func normalizeToolCalls(calls []schema.ToolCall) []schema.ToolCall {
	if len(calls) == 0 {
		return nil
	}
	result := make([]schema.ToolCall, 0, len(calls))
	for index, call := range calls {
		name := strings.TrimSpace(call.Function.Name)
		if name == "" {
			continue
		}
		call.Function.Name = name
		call.Type = strings.TrimSpace(call.Type)
		if call.Type == "" {
			call.Type = "function"
		}
		call.ID = strings.TrimSpace(call.ID)
		if call.ID == "" {
			call.ID = fallbackToolCallID(index)
		}
		call.Function.Arguments = strings.TrimSpace(call.Function.Arguments)
		if call.Function.Arguments == "" {
			call.Function.Arguments = "{}"
		}
		result = append(result, call)
	}
	return result
}

func loopToolCallKey(call schema.ToolCall) string {
	if call.Index != nil {
		return "index:" + strconv.Itoa(*call.Index)
	}
	if strings.TrimSpace(call.ID) != "" {
		return "id:" + strings.TrimSpace(call.ID)
	}
	if strings.TrimSpace(call.Function.Name) != "" {
		return "name:" + strings.TrimSpace(call.Function.Name)
	}
	return "fallback"
}
