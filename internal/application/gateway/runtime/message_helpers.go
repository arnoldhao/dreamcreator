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
	parts     []chatevent.MessagePart
}

func dtoMessagesToSchema(messages []dto.Message) []*schema.Message {
	return normalizedMessagesToSchema(normalizeIncomingMessages(messages), renderIncomingUserMessageParts)
}

func storedMessagesToSchema(messages []thread.ThreadMessage) []*schema.Message {
	return normalizedMessagesToSchema(normalizeStoredMessages(messages), renderStoredUserMessageParts)
}

func normalizedMessagesToSchema(
	messages []normalizedMessage,
	renderUserParts func([]chatevent.MessagePart) string,
) []*schema.Message {
	result := make([]*schema.Message, 0, len(messages))
	for _, message := range messages {
		role := strings.TrimSpace(message.role)
		if role == "" {
			continue
		}
		content := strings.TrimSpace(message.content)
		if len(message.parts) > 0 {
			if role == "user" {
				if text := strings.TrimSpace(renderUserParts(message.parts)); text != "" {
					content = text
				}
			} else {
				if text := strings.TrimSpace(joinTextParts(message.parts)); text != "" {
					content = text
				}
			}
			if role == "assistant" {
				if content != "" {
					result = append(result, schemaMessageWithThreadID(schema.Assistant, content, message.id))
				}
				continue
			}
		}
		if content == "" {
			continue
		}
		result = append(result, schemaMessageWithThreadID(schema.RoleType(role), content, message.id))
	}
	return result
}

func (service *Service) buildPromptInputMessages(
	ctx context.Context,
	threadID string,
	messages []dto.Message,
	preferStored bool,
	config promptContextBuildConfig,
) ([]*schema.Message, promptContextBuildReport, error) {
	if !preferStored || service == nil || service.messages == nil {
		base := dtoMessagesToSchema(messages)
		final := buildPromptMessagesToBudget(base, config)
		report := promptContextBuildReport{
			Source:                 "incoming",
			InputMessageCount:      len(base),
			BuiltMessageCount:      len(final),
			ContextWindowTokens:    config.contextWindowTokens,
			ReserveTokens:          config.reserveTokens,
			ExtraTokens:            config.extraTokens,
			AvailablePromptTokens:  resolvePromptMessageBudget(config),
			InitialEstimatedTokens: estimatePromptMessagesTokens(base),
			FinalEstimatedTokens:   estimatePromptMessagesTokens(final),
		}
		report.BudgetApplied = report.FinalEstimatedTokens != report.InitialEstimatedTokens || report.BuiltMessageCount != report.InputMessageCount
		return final, report, nil
	}
	storedMessages, err := service.messages.ListByThread(ctx, threadID, 0)
	if err != nil {
		return nil, promptContextBuildReport{}, err
	}
	if len(storedMessages) == 0 {
		base := dtoMessagesToSchema(messages)
		final := buildPromptMessagesToBudget(base, config)
		report := promptContextBuildReport{
			Source:                 "incoming",
			InputMessageCount:      len(base),
			BuiltMessageCount:      len(final),
			ContextWindowTokens:    config.contextWindowTokens,
			ReserveTokens:          config.reserveTokens,
			ExtraTokens:            config.extraTokens,
			AvailablePromptTokens:  resolvePromptMessageBudget(config),
			InitialEstimatedTokens: estimatePromptMessagesTokens(base),
			FinalEstimatedTokens:   estimatePromptMessagesTokens(final),
		}
		report.BudgetApplied = report.FinalEstimatedTokens != report.InitialEstimatedTokens || report.BuiltMessageCount != report.InputMessageCount
		return final, report, nil
	}
	resolved, report, err := service.resolveStoredPromptMessages(ctx, threadID, storedMessages)
	if err != nil {
		return nil, promptContextBuildReport{}, err
	}
	report.ContextWindowTokens = config.contextWindowTokens
	report.ReserveTokens = config.reserveTokens
	report.ExtraTokens = config.extraTokens
	report.AvailablePromptTokens = resolvePromptMessageBudget(config)
	if report.UsedPersistedSummary {
		final := buildPromptMessagesToBudget(resolved, config)
		report.BudgetApplied = len(final) != len(resolved) || estimatePromptMessagesTokens(final) != estimatePromptMessagesTokens(resolved)
		report.FinalEstimatedTokens = estimatePromptMessagesTokens(final)
		report.BuiltMessageCount = len(final)
		return final, report, nil
	}
	return resolved, report, nil
}

func (service *Service) persistIncomingMessages(
	ctx context.Context,
	threadID string,
	messages []dto.Message,
	replaceHistory bool,
) (bool, string, error) {
	if service == nil || service.messages == nil || service.threads == nil {
		return false, "", nil
	}
	incomingMessages := normalizeIncomingMessages(messages)
	if len(incomingMessages) == 0 {
		return false, "", nil
	}
	hasIncomingUserMessage := false
	for _, message := range incomingMessages {
		if message.role == "user" {
			hasIncomingUserMessage = true
			break
		}
	}
	if replaceHistory {
		lastUserMessageID, err := service.replaceThreadMessages(ctx, threadID, incomingMessages)
		if err != nil {
			return false, "", err
		}
		service.clearCompactedContextState(ctx, threadID)
		item, err := service.threads.Get(ctx, threadID)
		if err != nil {
			return false, "", err
		}
		item.UpdatedAt = service.now()
		return hasIncomingUserMessage, lastUserMessageID, service.threads.Save(ctx, item)
	}
	persistedUserMessage, lastUserMessageID, err := service.persistIncomingUserMessages(ctx, threadID, incomingMessages)
	if err != nil {
		return false, "", err
	}
	if !persistedUserMessage {
		return false, "", nil
	}
	item, err := service.threads.Get(ctx, threadID)
	if err != nil {
		return false, "", err
	}
	item.UpdatedAt = service.now()
	return true, lastUserMessageID, service.threads.Save(ctx, item)
}

func (service *Service) persistIncomingUserMessages(
	ctx context.Context,
	threadID string,
	incomingMessages []normalizedMessage,
) (bool, string, error) {
	existingMessages, err := service.messages.ListByThread(ctx, threadID, 0)
	if err != nil {
		return false, "", err
	}
	existingNormalized := normalizeStoredMessages(existingMessages)
	existingUsers := filterMessagesByRole(existingNormalized, "user")
	incomingUsers := filterMessagesByRole(incomingMessages, "user")
	newMessages := diffIncomingMessages(existingUsers, incomingUsers)
	persistedUserMessage := false
	lastUserMessageID := ""
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
			return false, "", err
		}
		if err := service.messages.Append(ctx, msg); err != nil {
			return false, "", err
		}
		persistedUserMessage = true
		lastUserMessageID = messageID
	}
	return persistedUserMessage, lastUserMessageID, nil
}

func (service *Service) replaceThreadMessages(ctx context.Context, threadID string, incomingMessages []normalizedMessage) (string, error) {
	if err := service.messages.DeleteByThread(ctx, threadID); err != nil {
		return "", err
	}
	if len(incomingMessages) == 0 {
		return "", nil
	}
	baseTime := service.now()
	lastUserMessageID := ""
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
			return "", err
		}
		if err := service.messages.Append(ctx, msg); err != nil {
			return "", err
		}
		if message.role == "user" {
			lastUserMessageID = messageID
		}
	}
	return lastUserMessageID, nil
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
			parts:     cloneMessageParts(message.Parts),
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
			parts:     parseNormalizedMessageParts(partsJSON),
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

func renderIncomingUserMessageParts(parts []chatevent.MessagePart) string {
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

func renderStoredUserMessageParts(parts []chatevent.MessagePart) string {
	text := strings.TrimSpace(joinTextParts(parts))
	if !hasAttachmentParts(parts) {
		return text
	}
	if text == "" {
		return "Attachments were provided with this message."
	}
	return text + "\n\nAttachments were provided with this message."
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

func hasAttachmentParts(parts []chatevent.MessagePart) bool {
	for _, part := range parts {
		partType := strings.TrimSpace(part.Type)
		if partType == "file" || partType == "image" {
			return true
		}
	}
	return false
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

func cloneMessageParts(parts []chatevent.MessagePart) []chatevent.MessagePart {
	if len(parts) == 0 {
		return nil
	}
	cloned := make([]chatevent.MessagePart, len(parts))
	copy(cloned, parts)
	return cloned
}

func parseNormalizedMessageParts(partsJSON string) []chatevent.MessagePart {
	trimmed := strings.TrimSpace(partsJSON)
	if trimmed == "" || trimmed == "[]" {
		return nil
	}
	var parts []chatevent.MessagePart
	if err := json.Unmarshal([]byte(trimmed), &parts); err != nil {
		return nil
	}
	return parts
}

func schemaMessageWithThreadID(role schema.RoleType, content string, threadMessageID string) *schema.Message {
	message := &schema.Message{
		Role:    role,
		Content: content,
	}
	if trimmed := strings.TrimSpace(threadMessageID); trimmed != "" {
		message.Extra = map[string]any{
			promptThreadMessageIDKey: trimmed,
		}
	}
	return message
}

func ptrTime(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}
	return &value
}
