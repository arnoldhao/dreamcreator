package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/cloudwego/eino/schema"

	"dreamcreator/internal/application/chatevent"
	"dreamcreator/internal/application/gateway/runtime/dto"
	"dreamcreator/internal/domain/thread"
)

type normalizedMessage struct {
	id        string
	role      string
	content   string
	partsJSON string
}

func dtoMessagesToSchema(messages []dto.Message) []*schema.Message {
	result := make([]*schema.Message, 0, len(messages))
	for _, message := range messages {
		role := strings.TrimSpace(message.Role)
		if role == "" {
			continue
		}
		content := strings.TrimSpace(message.Content)
		if len(message.Parts) > 0 {
			if role == "user" {
				if text := strings.TrimSpace(joinUserMessageParts(message.Parts)); text != "" {
					content = text
				}
			} else {
				if text := strings.TrimSpace(joinTextParts(message.Parts)); text != "" {
					content = text
				}
			}
			if role == "assistant" {
				toolParts := extractToolParts(message.Parts)
				var toolCalls []schema.ToolCall
				if len(toolParts) > 0 {
					toolCalls = make([]schema.ToolCall, 0, len(toolParts))
					for _, toolPart := range toolParts {
						if toolPart.callID == "" || toolPart.name == "" || !isReplayableToolState(toolPart.state) {
							continue
						}
						arguments := strings.TrimSpace(string(toolPart.inputRaw))
						if arguments == "" {
							arguments = "{}"
						}
						toolCalls = append(toolCalls, schema.ToolCall{
							ID:   toolPart.callID,
							Type: "function",
							Function: schema.FunctionCall{
								Name:      toolPart.name,
								Arguments: arguments,
							},
						})
					}
				}
				if content != "" || len(toolCalls) > 0 {
					result = append(result, &schema.Message{
						Role:      schema.Assistant,
						Content:   content,
						ToolCalls: toolCalls,
					})
				}
				for _, toolPart := range toolParts {
					toolName := toolPart.name
					if toolPart.callID == "" || toolName == "" {
						continue
					}
					switch toolPart.state {
					case "output-available":
						output := strings.TrimSpace(string(toolPart.outputRaw))
						if output == "" {
							output = "null"
						}
						result = append(result, schema.ToolMessage(output, toolPart.callID, schema.WithToolName(toolName)))
					case "output-error":
						output := strings.TrimSpace(toolPart.errorText)
						if output == "" {
							output = strings.TrimSpace(string(toolPart.outputRaw))
						}
						if output == "" {
							output = "tool output error"
						}
						result = append(result, schema.ToolMessage(output, toolPart.callID, schema.WithToolName(toolName)))
					case "input-error":
						output := strings.TrimSpace(toolPart.errorText)
						if output == "" {
							output = "tool input error"
						}
						result = append(result, schema.ToolMessage(output, toolPart.callID, schema.WithToolName(toolName)))
					case "output-denied":
						output := "tool execution denied"
						if toolPart.errorText != "" {
							output = toolPart.errorText
						}
						result = append(result, schema.ToolMessage(output, toolPart.callID, schema.WithToolName(toolName)))
					}
				}
				continue
			}
		}
		if content == "" {
			continue
		}
		result = append(result, &schema.Message{
			Role:    schema.RoleType(role),
			Content: content,
		})
	}
	return result
}

func (service *Service) persistIncomingMessages(
	ctx context.Context,
	threadID string,
	messages []dto.Message,
	replaceHistory bool,
) (bool, error) {
	if service == nil || service.messages == nil || service.threads == nil {
		return false, nil
	}
	incomingMessages := normalizeIncomingMessages(messages)
	if len(incomingMessages) == 0 {
		return false, nil
	}
	hasIncomingUserMessage := false
	for _, message := range incomingMessages {
		if message.role == "user" {
			hasIncomingUserMessage = true
			break
		}
	}
	if replaceHistory {
		if err := service.replaceThreadMessages(ctx, threadID, incomingMessages); err != nil {
			return false, err
		}
		item, err := service.threads.Get(ctx, threadID)
		if err != nil {
			return false, err
		}
		item.UpdatedAt = service.now()
		return hasIncomingUserMessage, service.threads.Save(ctx, item)
	}
	persistedUserMessage, err := service.persistIncomingUserMessages(ctx, threadID, incomingMessages)
	if err != nil {
		return false, err
	}
	if !persistedUserMessage {
		return false, nil
	}
	item, err := service.threads.Get(ctx, threadID)
	if err != nil {
		return false, err
	}
	item.UpdatedAt = service.now()
	return true, service.threads.Save(ctx, item)
}

func (service *Service) persistIncomingUserMessages(
	ctx context.Context,
	threadID string,
	incomingMessages []normalizedMessage,
) (bool, error) {
	existingMessages, err := service.messages.ListByThread(ctx, threadID, 0)
	if err != nil {
		return false, err
	}
	existingNormalized := normalizeStoredMessages(existingMessages)
	existingUsers := filterMessagesByRole(existingNormalized, "user")
	incomingUsers := filterMessagesByRole(incomingMessages, "user")
	newMessages := diffIncomingMessages(existingUsers, incomingUsers)
	persistedUserMessage := false
	for _, message := range newMessages {
		messageID := strings.TrimSpace(message.id)
		if messageID == "" {
			messageID = service.newID()
		}
		msg, err := thread.NewThreadMessage(thread.ThreadMessageParams{
			ID:        messageID,
			ThreadID:  threadID,
			Role:      message.role,
			Content:   message.content,
			PartsJSON: message.partsJSON,
			CreatedAt: ptrTime(service.now()),
		})
		if err != nil {
			return false, err
		}
		if err := service.messages.Append(ctx, msg); err != nil {
			return false, err
		}
		persistedUserMessage = true
	}
	return persistedUserMessage, nil
}

func (service *Service) replaceThreadMessages(ctx context.Context, threadID string, incomingMessages []normalizedMessage) error {
	if err := service.messages.DeleteByThread(ctx, threadID); err != nil {
		return err
	}
	if len(incomingMessages) == 0 {
		return nil
	}
	baseTime := service.now()
	for index, message := range incomingMessages {
		createdAt := baseTime.Add(time.Duration(index) * time.Millisecond)
		messageID := strings.TrimSpace(message.id)
		if messageID == "" {
			messageID = service.newID()
		}
		msg, err := thread.NewThreadMessage(thread.ThreadMessageParams{
			ID:        messageID,
			ThreadID:  threadID,
			Role:      message.role,
			Content:   message.content,
			PartsJSON: message.partsJSON,
			CreatedAt: ptrTime(createdAt),
		})
		if err != nil {
			return err
		}
		if err := service.messages.Append(ctx, msg); err != nil {
			return err
		}
	}
	return nil
}

func (service *Service) persistAssistantMessage(
	ctx context.Context,
	threadID string,
	messageID string,
	content string,
	parts []chatevent.MessagePart,
) error {
	if service == nil || service.messages == nil || service.threads == nil {
		return nil
	}
	trimmedContent := strings.TrimSpace(content)
	partsJSON := normalizeMessagePartsJSON(parts, trimmedContent)
	if trimmedContent == "" && partsJSON == "[]" {
		return errors.New("assistant response is empty")
	}
	if messageID == "" {
		messageID = service.newID()
	}
	assistantDomain, err := thread.NewThreadMessage(thread.ThreadMessageParams{
		ID:        messageID,
		ThreadID:  threadID,
		Role:      "assistant",
		Content:   trimmedContent,
		PartsJSON: partsJSON,
		CreatedAt: ptrTime(service.now()),
	})
	if err != nil {
		return err
	}
	if err := service.messages.Append(ctx, assistantDomain); err != nil {
		return err
	}
	item, err := service.threads.Get(ctx, threadID)
	if err != nil {
		return err
	}
	item.UpdatedAt = service.now()
	return service.threads.Save(ctx, item)
}

func normalizeIncomingMessages(messages []dto.Message) []normalizedMessage {
	result := make([]normalizedMessage, 0, len(messages))
	for _, message := range messages {
		role := normalizeRole(message.Role)
		if role != "user" && role != "assistant" {
			continue
		}
		id := strings.TrimSpace(message.ID)
		content := strings.TrimSpace(message.Content)
		partsJSON := normalizeMessagePartsJSON(message.Parts, content)
		if content == "" && partsJSON == "[]" {
			continue
		}
		result = append(result, normalizedMessage{
			id:        id,
			role:      role,
			content:   content,
			partsJSON: partsJSON,
		})
	}
	return result
}

func normalizeStoredMessages(messages []thread.ThreadMessage) []normalizedMessage {
	result := make([]normalizedMessage, 0, len(messages))
	for _, message := range messages {
		role := normalizeRole(message.Role)
		if role != "user" && role != "assistant" {
			continue
		}
		id := strings.TrimSpace(message.ID)
		content := strings.TrimSpace(message.Content)
		partsJSON := strings.TrimSpace(message.PartsJSON)
		if content == "" && partsJSON == "[]" {
			continue
		}
		result = append(result, normalizedMessage{
			id:        id,
			role:      role,
			content:   content,
			partsJSON: partsJSON,
		})
	}
	return result
}

func filterMessagesByRole(messages []normalizedMessage, role string) []normalizedMessage {
	if len(messages) == 0 {
		return nil
	}
	result := make([]normalizedMessage, 0, len(messages))
	for _, message := range messages {
		if message.role == role {
			result = append(result, message)
		}
	}
	return result
}

func diffIncomingMessages(existing []normalizedMessage, incoming []normalizedMessage) []normalizedMessage {
	if len(incoming) == 0 {
		return nil
	}
	if len(existing) == 0 {
		return incoming
	}
	max := len(existing)
	if len(incoming) < max {
		max = len(incoming)
	}
	matchLen := 0
	for k := max; k >= 1; k-- {
		if messagesMatch(existing[len(existing)-k:], incoming[:k]) {
			matchLen = k
			break
		}
	}
	return incoming[matchLen:]
}

func messagesMatch(existing []normalizedMessage, incoming []normalizedMessage) bool {
	if len(existing) != len(incoming) {
		return false
	}
	for i := range existing {
		if existing[i].role != incoming[i].role ||
			existing[i].content != incoming[i].content ||
			existing[i].partsJSON != incoming[i].partsJSON {
			return false
		}
	}
	return true
}

func normalizeMessagePartsJSON(parts []chatevent.MessagePart, fallbackContent string) string {
	if len(parts) == 0 {
		content := strings.TrimSpace(fallbackContent)
		if content == "" {
			return "[]"
		}
		data, err := json.Marshal([]chatevent.MessagePart{{
			Type: "text",
			Text: content,
		}})
		if err != nil {
			return "[]"
		}
		return string(data)
	}
	data, err := json.Marshal(parts)
	if err != nil {
		return "[]"
	}
	if len(data) == 0 {
		return "[]"
	}
	return string(data)
}

func normalizeRole(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

type toolPart struct {
	callID    string
	name      string
	state     string
	inputRaw  json.RawMessage
	outputRaw json.RawMessage
	errorText string
}

func extractToolParts(parts []chatevent.MessagePart) []toolPart {
	if len(parts) == 0 {
		return nil
	}
	result := make([]toolPart, 0)
	for _, part := range parts {
		if strings.TrimSpace(part.Type) != "tool-call" {
			continue
		}
		payload := parseToolPart(part)
		if payload.callID == "" || payload.name == "" {
			continue
		}
		result = append(result, payload)
	}
	return result
}

func parseToolPart(part chatevent.MessagePart) toolPart {
	return toolPart{
		callID:    strings.TrimSpace(part.ToolCallID),
		name:      strings.TrimSpace(part.ToolName),
		state:     strings.TrimSpace(part.State),
		inputRaw:  part.Input,
		outputRaw: part.Output,
		errorText: strings.TrimSpace(part.ErrorText),
	}
}

func isReplayableToolState(state string) bool {
	switch strings.TrimSpace(state) {
	case "output-available", "output-error", "output-denied", "input-error":
		return true
	default:
		return false
	}
}

func joinUserMessageParts(parts []chatevent.MessagePart) string {
	text := strings.TrimSpace(joinTextParts(parts))
	attachmentPaths := extractAttachmentPaths(parts)
	if len(attachmentPaths) == 0 {
		return text
	}
	var builder strings.Builder
	if text != "" {
		builder.WriteString(text)
		builder.WriteString("\n\n")
	}
	builder.WriteString("Attached files in workspace:\n")
	for _, path := range attachmentPaths {
		builder.WriteString("- ")
		builder.WriteString(path)
		builder.WriteString("\n")
	}
	builder.WriteString("Use the read tool with these paths when needed.")
	return strings.TrimSpace(builder.String())
}

func joinTextParts(parts []chatevent.MessagePart) string {
	var builder strings.Builder
	for _, part := range parts {
		if strings.TrimSpace(part.Type) == "text" {
			builder.WriteString(part.Text)
		}
	}
	return builder.String()
}

func extractAttachmentPaths(parts []chatevent.MessagePart) []string {
	if len(parts) == 0 {
		return nil
	}
	paths := make([]string, 0)
	seen := make(map[string]struct{})
	for _, part := range parts {
		partType := strings.TrimSpace(part.Type)
		if partType != "file" && partType != "image" {
			continue
		}
		path := parseAttachmentPath(part.Data)
		if path == "" {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		paths = append(paths, path)
	}
	return paths
}

func parseAttachmentPath(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var payload struct {
		Path     string `json:"path"`
		Filename string `json:"filename"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return ""
	}
	if trimmed := strings.TrimSpace(payload.Path); trimmed != "" {
		return trimmed
	}
	return strings.TrimSpace(payload.Filename)
}

func ptrTime(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}
	return &value
}
