package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/cloudwego/eino/schema"

	"dreamcreator/internal/application/agentruntime"
	"dreamcreator/internal/application/chatevent"
	"dreamcreator/internal/application/gateway/runtime/dto"
)

func consumeAgentLoopStream(
	stream *schema.StreamReader[*schema.Message],
	callback dto.RuntimeStreamCallback,
) (string, []chatevent.MessagePart, string, dto.RuntimeUsage, error) {
	if stream == nil {
		return "", nil, "", dto.RuntimeUsage{}, errors.New("assistant response is empty")
	}
	defer stream.Close()
	var textBuilder strings.Builder
	parts := make([]chatevent.MessagePart, 0, 8)
	reasoningPartIndex := -1
	toolPartIndices := make(map[string]int)
	toolArgsDelta := make(map[string]string)
	pendingSourceItems := make([]sourceItem, 0, 4)
	pendingSourceKeys := make(map[string]struct{})
	sourcesFlushed := false
	blockIndex := 0
	nextParentID := func() string {
		blockIndex++
		return fmt.Sprintf("block-%d", blockIndex)
	}
	marshalRaw := func(value any) json.RawMessage {
		data, err := json.Marshal(value)
		if err != nil {
			return nil
		}
		if len(data) == 0 {
			return nil
		}
		return data
	}
	resolveToolKey := func(callID string, toolName string) string {
		if trimmed := strings.TrimSpace(callID); trimmed != "" {
			return trimmed
		}
		return strings.TrimSpace(toolName)
	}
	appendTextPart := func(delta string) {
		if delta == "" {
			return
		}
		if len(parts) > 0 && parts[len(parts)-1].Type == "text" {
			parts[len(parts)-1].Text += delta
			return
		}
		parts = append(parts, chatevent.MessagePart{
			Type:     "text",
			ParentID: nextParentID(),
			Text:     delta,
		})
	}
	appendReasoningPart := func(delta string) {
		if delta == "" {
			return
		}
		if reasoningPartIndex >= 0 && reasoningPartIndex < len(parts) {
			if parts[reasoningPartIndex].Type == "reasoning" {
				parts[reasoningPartIndex].Text += delta
				return
			}
			reasoningPartIndex = -1
		}
		parts = append(parts, chatevent.MessagePart{
			Type:     "reasoning",
			ParentID: nextParentID(),
			Text:     delta,
		})
		reasoningPartIndex = len(parts) - 1
	}
	upsertToolPart := func(callID string, toolName string, apply func(part *chatevent.MessagePart)) {
		key := resolveToolKey(callID, toolName)
		if key == "" {
			return
		}
		index, exists := toolPartIndices[key]
		if !exists {
			part := chatevent.MessagePart{
				Type:       "tool-call",
				ParentID:   nextParentID(),
				ToolCallID: key,
				ToolName:   strings.TrimSpace(toolName),
				State:      "input-available",
				Input:      marshalRaw(map[string]any{}),
			}
			parts = append(parts, part)
			index = len(parts) - 1
			toolPartIndices[key] = index
		}
		part := &parts[index]
		if strings.TrimSpace(part.ToolCallID) == "" {
			part.ToolCallID = key
		}
		if trimmedName := strings.TrimSpace(toolName); trimmedName != "" {
			part.ToolName = trimmedName
		}
		if apply != nil {
			apply(part)
		}
	}
	finishReason := ""
	usage := dto.RuntimeUsage{}
	hasStepUsage := false
	emit := func(event dto.RuntimeStreamEvent) {
		if callback != nil {
			callback(event)
		}
	}
	for {
		message, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return "", parts, "", dto.RuntimeUsage{}, err
		}
		if message == nil {
			continue
		}
		event, ok := agentruntime.ParseEventMessage(message)
		if !ok {
			if strings.TrimSpace(message.Content) != "" {
				textBuilder.WriteString(message.Content)
				appendTextPart(message.Content)
			}
			continue
		}
		switch event.Type {
		case agentruntime.EventTextDelta:
			textBuilder.WriteString(event.Delta)
			appendTextPart(event.Delta)
			emit(dto.RuntimeStreamEvent{
				Type:  dto.RuntimeStreamEventDelta,
				Delta: event.Delta,
			})
		case agentruntime.EventReasoningDelta:
			appendReasoningPart(event.Delta)
		case agentruntime.EventToolCallStart:
			upsertToolPart(event.ToolCallID, event.ToolName, nil)
			emit(dto.RuntimeStreamEvent{
				Type:       dto.RuntimeStreamEventToolStart,
				ToolName:   strings.TrimSpace(event.ToolName),
				ToolCallID: strings.TrimSpace(event.ToolCallID),
			})
		case agentruntime.EventToolCallDelta:
			upsertToolPart(event.ToolCallID, event.ToolName, func(part *chatevent.MessagePart) {
				key := resolveToolKey(event.ToolCallID, event.ToolName)
				if key == "" {
					return
				}
				toolArgsDelta[key] = toolArgsDelta[key] + event.ToolArgsDelta
				nextRaw := strings.TrimSpace(toolArgsDelta[key])
				if nextRaw == "" {
					return
				}
				if json.Valid([]byte(nextRaw)) {
					part.Input = json.RawMessage(nextRaw)
				}
			})
		case agentruntime.EventToolCallReady:
			upsertToolPart(event.ToolCallID, event.ToolName, func(part *chatevent.MessagePart) {
				part.State = "input-available"
				if len(event.ToolArgs) > 0 {
					part.Input = append([]byte(nil), event.ToolArgs...)
				}
				key := resolveToolKey(event.ToolCallID, event.ToolName)
				if key != "" {
					delete(toolArgsDelta, key)
				}
			})
		case agentruntime.EventToolResult:
			upsertToolPart(event.ToolCallID, event.ToolName, func(part *chatevent.MessagePart) {
				part.State = "output-available"
				if len(event.ToolOutput) > 0 {
					part.Output = append([]byte(nil), event.ToolOutput...)
				} else {
					part.Output = marshalRaw(nil)
				}
				part.ErrorText = ""
			})
			emit(dto.RuntimeStreamEvent{
				Type:       dto.RuntimeStreamEventToolResult,
				ToolName:   strings.TrimSpace(event.ToolName),
				ToolCallID: strings.TrimSpace(event.ToolCallID),
			})
			sources := collectSourceItemsFromToolOutput(event.ToolOutput, event.ToolName, event.ToolCallID)
			for _, source := range sources {
				key := normalizeSourceKey(source.URL)
				if key == "" {
					continue
				}
				if _, exists := pendingSourceKeys[key]; exists {
					continue
				}
				pendingSourceKeys[key] = struct{}{}
				pendingSourceItems = append(pendingSourceItems, source)
			}
		case agentruntime.EventToolError:
			upsertToolPart(event.ToolCallID, event.ToolName, func(part *chatevent.MessagePart) {
				part.State = "output-error"
				errorText := strings.TrimSpace(event.ErrorText)
				if errorText == "" {
					errorText = "tool failed"
				}
				part.ErrorText = errorText
				if len(event.ToolOutput) > 0 {
					part.Output = append([]byte(nil), event.ToolOutput...)
				} else {
					part.Output = marshalRaw(map[string]string{"error": errorText})
				}
			})
			emit(dto.RuntimeStreamEvent{
				Type:       dto.RuntimeStreamEventToolResult,
				ToolName:   strings.TrimSpace(event.ToolName),
				ToolCallID: strings.TrimSpace(event.ToolCallID),
			})
		case agentruntime.EventContextSnapshot:
			if event.ContextTokens != nil {
				usage.ContextPromptTokens = event.ContextTokens.PromptTokens
				usage.ContextTotalTokens = event.ContextTokens.TotalTokens
				if event.ContextTokens.ContextLimitTokens > 0 {
					usage.ContextWindowTokens = event.ContextTokens.ContextLimitTokens
				} else if event.ContextTokens.HardTokens > 0 {
					usage.ContextWindowTokens = event.ContextTokens.HardTokens
				}
			}
		case agentruntime.EventRunError:
			errorText := strings.TrimSpace(event.ErrorText)
			if errorText == "" {
				errorText = "agent loop error"
			}
			emit(dto.RuntimeStreamEvent{
				Type:  dto.RuntimeStreamEventError,
				Error: errorText,
			})
			return "", parts, finishReason, usage, errors.New(errorText)
		case agentruntime.EventRunAbort:
			emit(dto.RuntimeStreamEvent{
				Type:  dto.RuntimeStreamEventError,
				Error: context.Canceled.Error(),
			})
			return "", parts, finishReason, usage, context.Canceled
		case agentruntime.EventRunEnd:
			if !sourcesFlushed && len(pendingSourceItems) > 0 {
				parts = appendSourceMessageParts(parts, pendingSourceItems, nextParentID, marshalRaw)
				sourcesFlushed = true
			}
			finishReason = strings.TrimSpace(event.FinishReason)
			if !hasStepUsage {
				usage = mergeRuntimeUsage(usage, event.Usage)
			}
			emit(dto.RuntimeStreamEvent{
				Type:         dto.RuntimeStreamEventEnd,
				FinishReason: finishReason,
				Usage:        usage,
			})
		case agentruntime.EventStepEnd:
			nextUsage := mergeRuntimeUsage(usage, event.Usage)
			if nextUsage.PromptTokens != usage.PromptTokens ||
				nextUsage.CompletionTokens != usage.CompletionTokens ||
				nextUsage.TotalTokens != usage.TotalTokens {
				hasStepUsage = true
			}
			usage = nextUsage
		}
	}
	if !sourcesFlushed && len(pendingSourceItems) > 0 {
		parts = appendSourceMessageParts(parts, pendingSourceItems, nextParentID, marshalRaw)
		sourcesFlushed = true
	}
	content := strings.TrimSpace(textBuilder.String())
	if content == "" {
		content = strings.TrimSpace(joinTextParts(parts))
	}
	if content == "" && len(parts) == 0 {
		return "", nil, finishReason, usage, errors.New("assistant response is empty")
	}
	return content, parts, finishReason, usage, nil
}

func mergeRuntimeUsage(current dto.RuntimeUsage, next *schema.TokenUsage) dto.RuntimeUsage {
	if next == nil {
		return current
	}
	promptTokens := maxInt(next.PromptTokens, 0)
	completionTokens := maxInt(next.CompletionTokens, 0)
	totalTokens := maxInt(next.TotalTokens, 0)
	if totalTokens <= 0 {
		totalTokens = promptTokens + completionTokens
	}
	if promptTokens <= 0 && completionTokens <= 0 && totalTokens <= 0 {
		return current
	}
	current.PromptTokens += promptTokens
	current.CompletionTokens += completionTokens
	current.TotalTokens += totalTokens
	return current
}

func maxInt(value int, lowerBound int) int {
	if value < lowerBound {
		return lowerBound
	}
	return value
}
