package runtime

import (
	"context"
	"strings"
	"time"

	"github.com/cloudwego/eino/schema"

	appsession "dreamcreator/internal/application/session"
	domainsession "dreamcreator/internal/domain/session"
	"dreamcreator/internal/domain/thread"
)

const (
	persistedContextStrategyVersion = 1
	promptThreadMessageIDKey        = "threadMessageID"
)

func (service *Service) resolveStoredPromptMessages(
	ctx context.Context,
	sessionID string,
	storedMessages []thread.ThreadMessage,
) ([]*schema.Message, promptContextBuildReport, error) {
	base := storedMessagesToSchema(storedMessages)
	report := promptContextBuildReport{
		Source:             "stored",
		StoredMessageCount: len(storedMessages),
		InputMessageCount:  len(base),
	}
	if service == nil || service.sessions == nil {
		report.InitialEstimatedTokens = estimatePromptMessagesTokens(base)
		report.FinalEstimatedTokens = report.InitialEstimatedTokens
		report.BuiltMessageCount = len(base)
		return base, report, nil
	}
	sessionEntry, err := service.sessions.Get(ctx, sessionID)
	if err != nil {
		report.InitialEstimatedTokens = estimatePromptMessagesTokens(base)
		report.FinalEstimatedTokens = report.InitialEstimatedTokens
		report.BuiltMessageCount = len(base)
		return base, report, nil
	}
	next, applied, stale := applyPersistedContextState(base, sessionEntry)
	report.InitialEstimatedTokens = estimatePromptMessagesTokens(base)
	report.FinalEstimatedTokens = estimatePromptMessagesTokens(next)
	report.BuiltMessageCount = len(next)
	if applied {
		report.UsedPersistedSummary = true
		report.PersistedSummaryChars = len([]rune(strings.TrimSpace(sessionEntry.ContextSummary)))
		report.PersistedFirstKeptMessageID = strings.TrimSpace(sessionEntry.ContextFirstKeptMessageID)
	}
	if stale {
		report.ClearedStalePersistedSummary = true
	}
	if stale {
		service.clearCompactedContextState(ctx, sessionID)
	}
	if applied {
		return next, report, nil
	}
	return base, report, nil
}

func applyPersistedContextState(messages []*schema.Message, entry domainsession.Entry) ([]*schema.Message, bool, bool) {
	if len(messages) == 0 {
		return nil, false, false
	}
	if entry.ContextStrategyVersion != persistedContextStrategyVersion {
		return messages, false, false
	}
	summary := strings.TrimSpace(entry.ContextSummary)
	firstKeptMessageID := strings.TrimSpace(entry.ContextFirstKeptMessageID)
	if summary == "" || firstKeptMessageID == "" {
		return messages, false, false
	}
	index := findPromptMessageIndexByThreadID(messages, firstKeptMessageID)
	if index < 0 {
		return messages, false, true
	}
	suffix := messages[index:]
	if len(suffix) == 0 {
		return messages, false, true
	}
	result := make([]*schema.Message, 0, len(suffix)+1)
	result = append(result, &schema.Message{
		Role:    schema.System,
		Content: compactionSummaryPrefix + summary,
	})
	result = append(result, suffix...)
	return result, true, false
}

func (service *Service) persistCompactedContextState(
	ctx context.Context,
	sessionID string,
	config contextGuardConfig,
	guard *contextGuardState,
) {
	if service == nil || service.sessions == nil || service.messages == nil || guard == nil {
		return
	}
	summary := strings.TrimSpace(guard.lastSummary)
	if summary == "" {
		return
	}
	storedMessages, err := service.messages.ListByThread(ctx, sessionID, 0)
	if err != nil || len(storedMessages) == 0 {
		return
	}
	promptMessages := storedMessagesToSchema(storedMessages)
	recent, older := selectRecentMessages(promptMessages, config.keepRecentTokens)
	if len(older) == 0 {
		_ = service.sessions.UpdateContextCompactionState(ctx, sessionID, appsession.ContextCompactionStateUpdate{})
		return
	}
	firstKeptMessageID := firstPromptThreadMessageID(recent)
	if firstKeptMessageID == "" {
		return
	}
	_ = service.sessions.UpdateContextCompactionState(ctx, sessionID, appsession.ContextCompactionStateUpdate{
		Summary:            summary,
		FirstKeptMessageID: firstKeptMessageID,
		StrategyVersion:    persistedContextStrategyVersion,
		CompactedAt:        service.now(),
	})
}

func findPromptMessageIndexByThreadID(messages []*schema.Message, threadMessageID string) int {
	trimmed := strings.TrimSpace(threadMessageID)
	if trimmed == "" {
		return -1
	}
	for index, message := range messages {
		if promptThreadMessageID(message) == trimmed {
			return index
		}
	}
	return -1
}

func firstPromptThreadMessageID(messages []*schema.Message) string {
	for _, message := range messages {
		if id := promptThreadMessageID(message); id != "" {
			return id
		}
	}
	return ""
}

func promptThreadMessageID(message *schema.Message) string {
	if message == nil || len(message.Extra) == 0 {
		return ""
	}
	value, ok := message.Extra[promptThreadMessageIDKey]
	if !ok {
		return ""
	}
	text, _ := value.(string)
	return strings.TrimSpace(text)
}

func (service *Service) clearCompactedContextState(ctx context.Context, sessionID string) {
	if service == nil || service.sessions == nil {
		return
	}
	_ = service.sessions.UpdateContextCompactionState(ctx, sessionID, appsession.ContextCompactionStateUpdate{
		CompactedAt: time.Time{},
	})
}
