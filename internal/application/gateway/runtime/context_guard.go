package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/cloudwego/eino/schema"

	"dreamcreator/internal/application/agentruntime"
	settingsdto "dreamcreator/internal/application/settings/dto"
)

const (
	defaultToolResultMaxChars      = 400_000
	defaultDetailedBrowserStates   = 1
	minToolResultKeepChars         = 2_000
	defaultContextWindowTokens     = 200_000
	contextInputHeadroomRatio      = 0.75
	maxToolResultContextShare      = 0.3
	compactionFallbackSummary      = "Summary unavailable due to context limits. Older messages were truncated."
	compactionSummaryPrefix        = "Conversation summary:\n"
	compactionNoReplyToken         = "NO_REPLY"
	toolResultCompactedPlaceholder = "[compacted: tool output removed to free context]"
	toolResultTruncationNotice     = "[truncated: output exceeded context limit]"
)

type memoryFlushConfig struct {
	enabled             bool
	softThresholdTokens int
	prompt              string
	systemPrompt        string
}

type contextGuardConfig struct {
	contextWindowTokens int
	warnTokens          int
	hardTokens          int
	compactionMode      string
	reserveTokens       int
	keepRecentTokens    int
	reserveTokensFloor  int
	maxHistoryShare     float64
	memoryFlush         memoryFlushConfig
	toolResultMaxChars  int
	extraTokens         int
}

type contextGuardState struct {
	lastSummary                string
	compactionCount            int
	memoryFlushCompactionCount int
}

type contextGuardReport struct {
	promptTokens       int
	totalTokens        int
	contextLimitTokens int
	truncatedResults   int
	compactedResults   int
	supersededResults  int
	droppedMessages    int
	memoryFlushed      bool
	memoryFlushError   string
	compactionSummary  string
	compactionTimedOut bool
}

type compactionSummaryParams struct {
	Messages            []*schema.Message
	PreviousSummary     string
	ContextWindowTokens int
	ReserveTokens       int
	CustomInstructions  string
}

type memoryFlushParams struct {
	Messages     []*schema.Message
	SystemPrompt string
	UserPrompt   string
	MaxSteps     int
}

type contextGuardHooks struct {
	Summarize              func(ctx context.Context, params compactionSummaryParams) (string, error)
	MemoryFlush            func(ctx context.Context, params memoryFlushParams) (string, error)
	PersistCompactionState func(ctx context.Context, compactionCount int, memoryFlushCompactionCount int) error
}

func resolveContextGuardConfig(settings settingsdto.GatewayRuntimeSettings, contextWindowTokens int) contextGuardConfig {
	if contextWindowTokens <= 0 {
		contextWindowTokens = defaultContextWindowTokens
	}
	mode := strings.TrimSpace(settings.Compaction.Mode)
	if mode == "" {
		mode = "safeguard"
	}
	reserveTokens := settings.Compaction.ReserveTokens
	if reserveTokens < 0 {
		reserveTokens = 0
	}
	reserveFloor := settings.Compaction.ReserveTokensFloor
	if reserveFloor < 0 {
		reserveFloor = 0
	}
	if reserveTokens < reserveFloor {
		reserveTokens = reserveFloor
	}
	keepRecent := settings.Compaction.KeepRecentTokens
	if keepRecent < 0 {
		keepRecent = 0
	}
	maxHistoryShare := settings.Compaction.MaxHistoryShare
	if maxHistoryShare <= 0 || maxHistoryShare > 0.9 {
		maxHistoryShare = 0.5
	}
	memPrompt := ensureNoReplyHint(strings.TrimSpace(settings.Compaction.MemoryFlush.Prompt))
	memSystem := ensureNoReplyHint(strings.TrimSpace(settings.Compaction.MemoryFlush.SystemPrompt))
	memCfg := memoryFlushConfig{
		enabled:             settings.Compaction.MemoryFlush.Enabled,
		softThresholdTokens: settings.Compaction.MemoryFlush.SoftThresholdTokens,
		prompt:              memPrompt,
		systemPrompt:        memSystem,
	}
	maxChars := calculateToolResultMaxChars(contextWindowTokens)
	return contextGuardConfig{
		contextWindowTokens: contextWindowTokens,
		warnTokens:          settings.ContextWindow.WarnTokens,
		hardTokens:          settings.ContextWindow.HardTokens,
		compactionMode:      mode,
		reserveTokens:       reserveTokens,
		keepRecentTokens:    keepRecent,
		reserveTokensFloor:  reserveFloor,
		maxHistoryShare:     maxHistoryShare,
		memoryFlush:         memCfg,
		toolResultMaxChars:  maxChars,
	}
}

func applyContextGuard(ctx context.Context, state agentruntime.AgentState, config contextGuardConfig, guard *contextGuardState, hooks contextGuardHooks) (agentruntime.AgentState, contextGuardReport, error) {
	report := contextGuardReport{contextLimitTokens: config.contextWindowTokens}
	messages := state.Messages
	var superseded int
	messages, superseded = applySameTurnBrowserSupersession(messages, defaultDetailedBrowserStates)
	if superseded > 0 {
		report.supersededResults = superseded
		state.Messages = messages
	}
	if config.toolResultMaxChars > 0 && config.contextWindowTokens > 0 {
		var truncated int
		var compacted int
		messages, truncated, compacted = applyToolResultContextGuard(messages, config)
		if truncated > 0 {
			report.truncatedResults = truncated
		}
		if compacted > 0 {
			report.compactedResults = compacted
		}
		if truncated > 0 || compacted > 0 {
			state.Messages = messages
		}
	}

	tokens := agentruntime.EstimateMessagesTokensSafe(messages)
	if config.extraTokens > 0 {
		tokens += config.extraTokens
	}
	if tokens < 0 {
		tokens = 0
	}
	report.promptTokens = tokens
	report.totalTokens = tokens
	if config.contextWindowTokens > 0 {
		if config.hardTokens > 0 && config.contextWindowTokens < config.hardTokens {
			return state, report, errors.New("context_window_too_small")
		}
	}

	if guard == nil {
		guard = &contextGuardState{}
	}

	if isCompactionEnabled(config.compactionMode) && config.memoryFlush.enabled {
		nextCompactionCount := guard.compactionCount + 1
		needsFlush := guard.memoryFlushCompactionCount < nextCompactionCount
		if needsFlush && shouldRunMemoryFlush(tokens, config.contextWindowTokens, config.reserveTokensFloor, config.memoryFlush.softThresholdTokens) {
			if hooks.MemoryFlush != nil {
				prompt := resolveMemoryFlushPrompt(config.memoryFlush.prompt)
				systemPrompt := strings.TrimSpace(config.memoryFlush.systemPrompt)
				_, err := hooks.MemoryFlush(ctx, memoryFlushParams{
					Messages:     cloneSchemaMessages(messages),
					SystemPrompt: systemPrompt,
					UserPrompt:   prompt,
					MaxSteps:     4,
				})
				guard.memoryFlushCompactionCount = nextCompactionCount
				report.memoryFlushed = true
				if hooks.PersistCompactionState != nil {
					_ = hooks.PersistCompactionState(ctx, guard.compactionCount, guard.memoryFlushCompactionCount)
				}
				if err != nil {
					report.memoryFlushError = err.Error()
				}
			}
		}
	}

	compactionLimit := config.contextWindowTokens
	triggerTokens := compactionLimit
	if compactionLimit > 0 && config.reserveTokens > 0 {
		triggerTokens = compactionLimit - config.reserveTokens
		if triggerTokens <= 0 {
			triggerTokens = compactionLimit
		}
	}
	if isCompactionEnabled(config.compactionMode) && triggerTokens > 0 && tokens > triggerTokens {
		preCompactionSnapshot := cloneSchemaMessages(messages)
		summary, dropped, updated, timedOut := compactMessages(ctx, messages, config, guard, hooks)
		if updated != nil {
			state.Messages = updated
			guard.compactionCount++
			report.compactionSummary = summary
			report.droppedMessages = dropped
			if timedOut {
				report.compactionTimedOut = true
			}
			if hooks.PersistCompactionState != nil {
				_ = hooks.PersistCompactionState(ctx, guard.compactionCount, guard.memoryFlushCompactionCount)
			}
			tokens = agentruntime.EstimateMessagesTokensSafe(updated)
			if config.extraTokens > 0 {
				tokens += config.extraTokens
			}
			if tokens < 0 {
				tokens = 0
			}
			report.promptTokens = tokens
			report.totalTokens = tokens
		} else if timedOut {
			// Timeout during compaction falls back to pre-compaction snapshot.
			state.Messages = preCompactionSnapshot
			report.compactionTimedOut = true
		}
	}

	limit := config.contextWindowTokens
	if limit > 0 && tokens > limit {
		return state, report, errors.New("context_window_exceeded")
	}
	return state, report, nil
}

func applyToolResultContextGuard(messages []*schema.Message, config contextGuardConfig) ([]*schema.Message, int, int) {
	if len(messages) == 0 {
		return messages, 0, 0
	}
	updated := messages
	truncated := 0
	compacted := 0
	changed := false

	for index, message := range updated {
		next, changedMessage := truncateToolResultMessage(message, config.toolResultMaxChars)
		if !changedMessage {
			continue
		}
		if !changed {
			updated = cloneSchemaMessages(messages)
			changed = true
		}
		updated[index] = next
		truncated++
	}

	budgetTokens := int(math.Floor(float64(config.contextWindowTokens) * contextInputHeadroomRatio))
	if budgetTokens <= 0 {
		if changed {
			return updated, truncated, compacted
		}
		return messages, truncated, compacted
	}
	if config.extraTokens > 0 {
		budgetTokens -= config.extraTokens
	}
	if budgetTokens < 1 {
		budgetTokens = 1
	}

	totalTokens := agentruntime.EstimateMessagesTokensSafe(updated)
	if config.extraTokens > 0 {
		totalTokens += config.extraTokens
	}
	if totalTokens <= budgetTokens {
		if changed {
			return updated, truncated, compacted
		}
		return messages, truncated, compacted
	}

	for index, message := range updated {
		if message == nil || message.Role != schema.Tool {
			continue
		}
		if strings.TrimSpace(message.Content) == toolResultCompactedPlaceholder {
			continue
		}
		cp := *message
		cp.Content = toolResultCompactedPlaceholder
		if !changed {
			updated = cloneSchemaMessages(messages)
			changed = true
		}
		updated[index] = &cp
		compacted++

		totalTokens = agentruntime.EstimateMessagesTokensSafe(updated)
		if config.extraTokens > 0 {
			totalTokens += config.extraTokens
		}
		if totalTokens <= budgetTokens {
			break
		}
	}

	if !changed {
		return messages, truncated, compacted
	}
	return updated, truncated, compacted
}

func applySameTurnBrowserSupersession(messages []*schema.Message, keepDetailed int) ([]*schema.Message, int) {
	if len(messages) == 0 || keepDetailed < 0 {
		return messages, 0
	}
	lastUserIndex := -1
	for index := len(messages) - 1; index >= 0; index-- {
		message := messages[index]
		if message != nil && message.Role == schema.User {
			lastUserIndex = index
			break
		}
	}
	if lastUserIndex < 0 || lastUserIndex >= len(messages)-1 {
		return messages, 0
	}

	browserIndexes := make([]int, 0, 4)
	for index := lastUserIndex + 1; index < len(messages); index++ {
		message := messages[index]
		if message == nil || message.Role != schema.Tool {
			continue
		}
		if strings.ToLower(strings.TrimSpace(message.ToolName)) != "browser" {
			continue
		}
		if isBrowserSupersededContent(message.Content) {
			continue
		}
		browserIndexes = append(browserIndexes, index)
	}
	if len(browserIndexes) <= keepDetailed {
		return messages, 0
	}

	updated := cloneSchemaMessages(messages)
	superseded := 0
	for _, index := range browserIndexes[:len(browserIndexes)-keepDetailed] {
		message := updated[index]
		if message == nil {
			continue
		}
		nextContent := buildBrowserSupersededContent()
		if strings.TrimSpace(message.Content) == nextContent {
			continue
		}
		message.Content = nextContent
		superseded++
	}
	if superseded == 0 {
		return messages, 0
	}
	return updated, superseded
}

const browserSupersededContent = "[superseded: older browser tool result removed to keep the latest browser result in this turn]"

func buildBrowserSupersededContent() string {
	return browserSupersededContent
}

func isBrowserSupersededContent(raw string) bool {
	return strings.TrimSpace(raw) == browserSupersededContent
}

func truncateToolResultMessage(message *schema.Message, maxChars int) (*schema.Message, bool) {
	if message == nil || message.Role != schema.Tool || maxChars <= 0 {
		return message, false
	}
	effectiveMaxChars := resolveToolResultMaxChars(message, maxChars)
	if effectiveMaxChars <= 0 {
		return message, false
	}
	content := strings.TrimSpace(message.Content)
	if content == "" {
		return message, false
	}
	if next, ok := truncateToolResultBlocks(content, effectiveMaxChars); ok {
		cp := *message
		cp.Content = next
		return &cp, true
	}
	if len(message.Content) <= effectiveMaxChars {
		return message, false
	}
	cp := *message
	cp.Content = truncateTextWithNotice(message.Content, effectiveMaxChars, toolResultTruncationNotice)
	return &cp, true
}

func resolveToolResultMaxChars(_ *schema.Message, maxChars int) int {
	return maxChars
}

func truncateToolResultBlocks(content string, maxChars int) (string, bool) {
	var blocks []map[string]any
	if err := json.Unmarshal([]byte(content), &blocks); err != nil || len(blocks) == 0 {
		return "", false
	}
	totalTextChars := 0
	textIndexes := make([]int, 0, len(blocks))
	for index, block := range blocks {
		if !strings.EqualFold(strings.TrimSpace(toString(block["type"])), "text") {
			continue
		}
		text := toString(block["text"])
		if text == "" {
			continue
		}
		totalTextChars += len(text)
		textIndexes = append(textIndexes, index)
	}
	if totalTextChars == 0 || totalTextChars <= maxChars {
		return "", false
	}
	mutated := false
	for _, index := range textIndexes {
		block := blocks[index]
		text := toString(block["text"])
		if text == "" {
			continue
		}
		share := float64(len(text)) / float64(totalTextChars)
		blockBudget := int(math.Floor(float64(maxChars) * share))
		if blockBudget < minToolResultKeepChars {
			blockBudget = minToolResultKeepChars
		}
		next := truncateTextWithNotice(text, blockBudget, toolResultTruncationNotice)
		if next != text {
			block["text"] = next
			mutated = true
		}
	}
	if !mutated {
		return "", false
	}
	encoded, err := json.Marshal(blocks)
	if err != nil {
		return "", false
	}
	return string(encoded), true
}

func truncateTextWithNotice(value string, maxChars int, notice string) string {
	if len(value) <= maxChars {
		return value
	}
	keep := maxChars
	if keep < minToolResultKeepChars {
		keep = minToolResultKeepChars
	}
	if keep > len(value) {
		keep = len(value)
	}
	suffix := "\n\n" + strings.TrimSpace(notice)
	if keep > len(suffix)+1 {
		keep -= len(suffix)
	}
	if keep <= 0 {
		return value[:maxChars]
	}
	return value[:keep] + suffix
}

func calculateToolResultMaxChars(contextWindowTokens int) int {
	if contextWindowTokens <= 0 {
		return 0
	}
	maxTokens := int(float64(contextWindowTokens) * maxToolResultContextShare)
	if maxTokens <= 0 {
		return 0
	}
	maxChars := maxTokens * 4
	if maxChars > defaultToolResultMaxChars {
		maxChars = defaultToolResultMaxChars
	}
	if maxChars < minToolResultKeepChars {
		return minToolResultKeepChars
	}
	return maxChars
}

func isCompactionEnabled(mode string) bool {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "", "default", "safeguard":
		return true
	case "off", "none", "disabled":
		return false
	default:
		return true
	}
}

func isSafeguardMode(mode string) bool {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "safeguard":
		return true
	default:
		return false
	}
}

func compactMessages(ctx context.Context, messages []*schema.Message, config contextGuardConfig, guard *contextGuardState, hooks contextGuardHooks) (string, int, []*schema.Message, bool) {
	if len(messages) == 0 {
		return "", 0, nil, false
	}
	systemMessages, existingSummary, nonSystem := splitSystemAndSummary(messages)
	previousSummary := strings.TrimSpace(existingSummary)
	if guard != nil && strings.TrimSpace(guard.lastSummary) != "" {
		previousSummary = guard.lastSummary
	}
	keepRecentTokens := config.keepRecentTokens
	if keepRecentTokens < 0 {
		keepRecentTokens = 0
	}
	recent, older := selectRecentMessages(nonSystem, keepRecentTokens)
	if len(older) == 0 {
		return previousSummary, 0, nil, false
	}
	var droppedForShare []*schema.Message
	messagesToSummarize := older
	if isSafeguardMode(config.compactionMode) && config.maxHistoryShare > 0 {
		kept, dropped := pruneHistoryForShare(messagesToSummarize, config.contextWindowTokens, config.maxHistoryShare)
		if len(dropped) > 0 {
			droppedForShare = dropped
			messagesToSummarize = kept
		}
	}
	messagesToSummarize = stripToolResultDetailsFromMessages(messagesToSummarize)
	droppedForShare = stripToolResultDetailsFromMessages(droppedForShare)

	summary := ""
	if hooks.Summarize != nil {
		if len(droppedForShare) > 0 {
			droppedSummary, err := hooks.Summarize(ctx, compactionSummaryParams{
				Messages:            droppedForShare,
				PreviousSummary:     previousSummary,
				ContextWindowTokens: config.contextWindowTokens,
				ReserveTokens:       config.reserveTokens,
			})
			if err != nil && isCompactionTimeoutError(err) {
				return previousSummary, 0, nil, true
			}
			if err == nil && strings.TrimSpace(droppedSummary) != "" {
				previousSummary = strings.TrimSpace(droppedSummary)
			}
		}
		summaryText, err := hooks.Summarize(ctx, compactionSummaryParams{
			Messages:            messagesToSummarize,
			PreviousSummary:     previousSummary,
			ContextWindowTokens: config.contextWindowTokens,
			ReserveTokens:       config.reserveTokens,
		})
		if err != nil && isCompactionTimeoutError(err) {
			return previousSummary, 0, nil, true
		}
		if err == nil {
			summary = strings.TrimSpace(summaryText)
		}
	}
	if summary == "" {
		summary = compactionFallbackSummary
	}
	if guard != nil {
		guard.lastSummary = summary
	}

	combined := make([]*schema.Message, 0, len(systemMessages)+1+len(recent))
	combined = append(combined, systemMessages...)
	summaryMessage := &schema.Message{Role: schema.System, Content: compactionSummaryPrefix + summary}
	combined = append(combined, summaryMessage)
	combined = append(combined, recent...)
	combined, droppedOrphans := repairToolUseResultPairing(combined)
	dropped := len(older)
	if len(droppedForShare) > 0 {
		dropped += len(droppedForShare)
	}
	if droppedOrphans > 0 {
		dropped += droppedOrphans
	}
	return summary, dropped, combined, false
}

func splitSystemAndSummary(messages []*schema.Message) ([]*schema.Message, string, []*schema.Message) {
	if len(messages) == 0 {
		return nil, "", nil
	}
	systemMessages := make([]*schema.Message, 0)
	nonSystem := make([]*schema.Message, 0, len(messages))
	summary := ""
	for _, message := range messages {
		if message == nil {
			continue
		}
		if message.Role == schema.System {
			content := strings.TrimSpace(message.Content)
			if strings.HasPrefix(content, compactionSummaryPrefix) {
				summary = strings.TrimSpace(strings.TrimPrefix(content, compactionSummaryPrefix))
				continue
			}
			systemMessages = append(systemMessages, message)
			continue
		}
		nonSystem = append(nonSystem, message)
	}
	return systemMessages, summary, nonSystem
}

func selectRecentMessages(messages []*schema.Message, keepTokens int) ([]*schema.Message, []*schema.Message) {
	if len(messages) == 0 {
		return nil, nil
	}
	if keepTokens <= 0 {
		last := messages[len(messages)-1]
		return []*schema.Message{last}, messages[:len(messages)-1]
	}
	kept := make([]*schema.Message, 0)
	budget := keepTokens
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		if msg == nil {
			continue
		}
		tokens := agentruntime.EstimateMessageTokens(msg)
		if len(kept) > 0 && budget-tokens < 0 {
			break
		}
		kept = append(kept, msg)
		budget -= tokens
	}
	if len(kept) == 0 {
		last := messages[len(messages)-1]
		return []*schema.Message{last}, messages[:len(messages)-1]
	}
	reverseMessages(kept)
	index := len(messages) - len(kept)
	if index < 0 {
		index = 0
	}
	older := messages[:index]
	return kept, older
}

func reverseMessages(messages []*schema.Message) {
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}
}

func pruneHistoryForShare(messages []*schema.Message, maxContextTokens int, maxHistoryShare float64) ([]*schema.Message, []*schema.Message) {
	if len(messages) == 0 {
		return nil, nil
	}
	if maxContextTokens <= 0 {
		return messages, nil
	}
	budget := int(math.Max(1, math.Floor(float64(maxContextTokens)*maxHistoryShare)))
	kept := messages
	dropped := make([]*schema.Message, 0)
	for len(kept) > 1 && agentruntime.EstimateMessagesTokens(kept) > budget {
		chunks := splitMessagesByTokenShare(kept, 2)
		if len(chunks) <= 1 {
			break
		}
		dropped = append(dropped, chunks[0]...)
		kept = flattenMessages(chunks[1:])
	}
	return kept, dropped
}

func splitMessagesByTokenShare(messages []*schema.Message, parts int) [][]*schema.Message {
	if len(messages) == 0 {
		return nil
	}
	if parts <= 1 {
		return [][]*schema.Message{messages}
	}
	total := agentruntime.EstimateMessagesTokens(messages)
	if total <= 0 {
		return [][]*schema.Message{messages}
	}
	target := total / parts
	chunks := make([][]*schema.Message, 0, parts)
	current := make([]*schema.Message, 0)
	currentTokens := 0
	for _, msg := range messages {
		msgTokens := agentruntime.EstimateMessageTokens(msg)
		if len(current) > 0 && len(chunks) < parts-1 && currentTokens+msgTokens > target {
			chunks = append(chunks, current)
			current = make([]*schema.Message, 0)
			currentTokens = 0
		}
		current = append(current, msg)
		currentTokens += msgTokens
	}
	if len(current) > 0 {
		chunks = append(chunks, current)
	}
	return chunks
}

func flattenMessages(chunks [][]*schema.Message) []*schema.Message {
	if len(chunks) == 0 {
		return nil
	}
	var total int
	for _, chunk := range chunks {
		total += len(chunk)
	}
	result := make([]*schema.Message, 0, total)
	for _, chunk := range chunks {
		result = append(result, chunk...)
	}
	return result
}

func repairToolUseResultPairing(messages []*schema.Message) ([]*schema.Message, int) {
	toolCalls := make(map[string]struct{})
	for _, msg := range messages {
		if msg == nil || msg.Role != schema.Assistant {
			continue
		}
		for _, call := range msg.ToolCalls {
			id := strings.TrimSpace(call.ID)
			if id != "" {
				toolCalls[id] = struct{}{}
			}
		}
	}
	if len(toolCalls) == 0 {
		return messages, 0
	}
	filtered := make([]*schema.Message, 0, len(messages))
	dropped := 0
	for _, msg := range messages {
		if msg == nil {
			continue
		}
		if msg.Role == schema.Tool {
			id := strings.TrimSpace(msg.ToolCallID)
			if id != "" {
				if _, ok := toolCalls[id]; !ok {
					dropped++
					continue
				}
			}
		}
		filtered = append(filtered, msg)
	}
	return filtered, dropped
}

func stripToolResultDetailsFromMessages(messages []*schema.Message) []*schema.Message {
	if len(messages) == 0 {
		return messages
	}
	out := messages
	changed := false
	for index, message := range messages {
		if message == nil || message.Role != schema.Tool {
			continue
		}
		nextContent, stripped := stripToolResultDetailsJSON(message.Content)
		if !stripped {
			continue
		}
		if !changed {
			out = cloneSchemaMessages(messages)
			changed = true
		}
		cp := *out[index]
		cp.Content = nextContent
		out[index] = &cp
	}
	if !changed {
		return messages
	}
	return out
}

func stripToolResultDetailsJSON(raw string) (string, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return raw, false
	}
	var payload any
	if err := json.Unmarshal([]byte(trimmed), &payload); err != nil {
		return raw, false
	}
	sanitized, changed := stripDetailsRecursive(payload)
	if !changed {
		return raw, false
	}
	encoded, err := json.Marshal(sanitized)
	if err != nil {
		return raw, false
	}
	return string(encoded), true
}

func stripDetailsRecursive(value any) (any, bool) {
	switch typed := value.(type) {
	case map[string]any:
		changed := false
		next := make(map[string]any, len(typed))
		for key, raw := range typed {
			if strings.EqualFold(strings.TrimSpace(key), "details") {
				changed = true
				continue
			}
			nested, nestedChanged := stripDetailsRecursive(raw)
			if nestedChanged {
				changed = true
			}
			next[key] = nested
		}
		if !changed {
			return value, false
		}
		return next, true
	case []any:
		changed := false
		next := make([]any, 0, len(typed))
		for _, raw := range typed {
			nested, nestedChanged := stripDetailsRecursive(raw)
			if nestedChanged {
				changed = true
			}
			next = append(next, nested)
		}
		if !changed {
			return value, false
		}
		return next, true
	default:
		return value, false
	}
}

func isCompactionTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return true
	}
	message := strings.ToLower(strings.TrimSpace(err.Error()))
	return strings.Contains(message, "deadline exceeded") || strings.Contains(message, "timeout")
}

func ensureNoReplyHint(text string) string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return trimmed
	}
	if strings.Contains(trimmed, compactionNoReplyToken) {
		return trimmed
	}
	return trimmed + "\n\nIf nothing to store, reply with " + compactionNoReplyToken + "."
}

func shouldRunMemoryFlush(totalTokens int, contextWindowTokens int, reserveTokensFloor int, softThresholdTokens int) bool {
	if totalTokens <= 0 || contextWindowTokens <= 0 {
		return false
	}
	reserve := reserveTokensFloor
	if reserve < 0 {
		reserve = 0
	}
	soft := softThresholdTokens
	if soft < 0 {
		soft = 0
	}
	threshold := contextWindowTokens - reserve - soft
	if threshold <= 0 {
		return false
	}
	return totalTokens >= threshold
}

func resolveMemoryFlushPrompt(prompt string) string {
	trimmed := strings.TrimSpace(prompt)
	if trimmed == "" {
		return trimmed
	}
	resolved := replaceDateToken(trimmed)
	if strings.Contains(resolved, "Current time:") {
		return resolved
	}
	now := time.Now()
	zone := now.Location().String()
	if zone == "" {
		zone = "local"
	}
	timeLine := fmt.Sprintf("Current time: %s (%s)", now.Format(time.RFC3339), zone)
	return strings.TrimSpace(resolved + "\n" + timeLine)
}

func replaceDateToken(text string) string {
	if !strings.Contains(text, "YYYY-MM-DD") {
		return text
	}
	date := time.Now().Format("2006-01-02")
	return strings.ReplaceAll(text, "YYYY-MM-DD", date)
}

func isNoReplyResponse(text string) bool {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return true
	}
	upper := strings.ToUpper(trimmed)
	return strings.HasPrefix(upper, compactionNoReplyToken)
}

func cloneSchemaMessages(messages []*schema.Message) []*schema.Message {
	if len(messages) == 0 {
		return nil
	}
	out := make([]*schema.Message, 0, len(messages))
	for _, message := range messages {
		if message == nil {
			continue
		}
		cp := *message
		out = append(out, &cp)
	}
	return out
}
