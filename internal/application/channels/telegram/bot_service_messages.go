package telegram

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"dreamcreator/internal/application/chatevent"
	telegramapi "dreamcreator/internal/infrastructure/telegram"
	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
	"go.uber.org/zap"
)

func buildTelegramReply(content string, parts []chatevent.MessagePart) string {
	trimmed := strings.TrimSpace(content)
	sources := collectTelegramSourceItems(parts)
	if len(sources) == 0 {
		return trimmed
	}
	var builder strings.Builder
	if trimmed != "" {
		builder.WriteString(trimmed)
		builder.WriteString("\n\n")
	}
	builder.WriteString("来源:\n")
	for index, source := range sources {
		builder.WriteString(buildTelegramSourceLine(index+1, source))
		builder.WriteString("\n")
	}
	return strings.TrimSpace(builder.String())
}

func buildTelegramSourceLine(index int, source telegramSourceItem) string {
	urlValue := strings.TrimSpace(source.URL)
	label := strings.TrimSpace(source.Title)
	if label == "" {
		label = urlValue
	}
	label = strings.TrimSpace(strings.ReplaceAll(label, "\n", " "))
	if strings.EqualFold(label, urlValue) {
		return fmt.Sprintf("[%d] <%s>", index, urlValue)
	}
	return fmt.Sprintf("[%d] [%s](<%s>)", index, escapeTelegramMarkdownLinkLabel(label), urlValue)
}

func escapeTelegramMarkdownLinkLabel(value string) string {
	replacer := strings.NewReplacer(
		`\\`, `\\\\`,
		`[`, `\[`,
		`]`, `\]`,
	)
	return replacer.Replace(value)
}

func collectTelegramSourceItems(parts []chatevent.MessagePart) []telegramSourceItem {
	if len(parts) == 0 {
		return nil
	}
	result := make([]telegramSourceItem, 0, 4)
	seen := make(map[string]struct{})
	for _, part := range parts {
		if strings.TrimSpace(part.Type) != "source" {
			continue
		}
		urlValue := strings.TrimSpace(part.Text)
		title := ""
		if len(part.Data) > 0 {
			var payload struct {
				URL   string `json:"url"`
				Title string `json:"title"`
			}
			if err := json.Unmarshal(part.Data, &payload); err == nil {
				if strings.TrimSpace(payload.URL) != "" {
					urlValue = strings.TrimSpace(payload.URL)
				}
				title = strings.TrimSpace(payload.Title)
			}
		}
		if urlValue == "" {
			continue
		}
		key := strings.ToLower(urlValue)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, telegramSourceItem{
			URL:   urlValue,
			Title: title,
		})
	}
	return result
}

func (service *BotService) sendReply(ctx context.Context, state *telegramAccountState, message *telegramapi.Message, sessionKey string, reply string, draftStream *telegramDraftStream) error {
	if service == nil || state == nil || message == nil {
		return nil
	}
	if state.bot == nil {
		return errors.New("telegram bot unavailable")
	}
	chunks := buildTelegramChunks(reply, state.config.Chunk)
	if len(chunks) == 0 {
		return nil
	}
	replyTo := resolveReplyToMessageID(state, sessionKey, message.MessageID)
	threadID := message.MessageThreadID
	previewFinalized := false
	if draftStream != nil {
		previewID := draftStream.MessageID()
		if previewID > 0 && len(chunks) == 1 {
			if editedID, err := service.editPlaceholder(ctx, state, message.Chat.ID, previewID, chunks[0], true); err == nil {
				service.recordOutboundSuccess(state, int64(editedID))
				previewFinalized = true
			}
		}
	}
	if !previewFinalized {
		if err := service.sendChunks(ctx, state, message.Chat.ID, threadID, chunks, replyTo); err != nil {
			return err
		}
	}
	if draftStream != nil && !previewFinalized {
		draftStream.Clear()
	}
	return nil
}

func resolveReplyToMessageID(state *telegramAccountState, sessionKey string, messageID int) int {
	mode := strings.ToLower(strings.TrimSpace(state.config.ReplyToMode))
	switch mode {
	case "off", "none", "disabled":
		return 0
	case "first":
		if state.replyTracker == nil {
			state.replyTracker = newReplyTracker()
		}
		if state.replyTracker.ShouldReplyFirst(sessionKey) {
			return messageID
		}
		return 0
	case "all", "reply", "always":
		return messageID
	default:
		return 0
	}
}

func buildTelegramChunks(text string, chunk TelegramChunkConfig) []telegramapi.FormattedChunk {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return nil
	}
	mode := strings.ToLower(strings.TrimSpace(chunk.Mode))
	limit := chunk.TextChunkLimit
	if limit <= 0 {
		limit = 3800
	}
	html := telegramapi.RenderTelegramHTML(trimmed)
	if html == "" {
		return nil
	}
	if mode == "off" || mode == "none" || mode == "disabled" {
		if utf8.RuneCountInString(html) <= limit {
			return []telegramapi.FormattedChunk{{HTML: html, Text: telegramapi.PlainTextFromHTML(html)}}
		}
		// Even when chunk mode is off, keep payload within Telegram entity limits.
		return telegramapi.SplitTelegramHTML(html, limit)
	}
	return telegramapi.SplitTelegramHTML(html, limit)
}

func resolveTelegramStreamMode(raw string) string {
	mode := strings.ToLower(strings.TrimSpace(raw))
	switch mode {
	case "off", "partial", "block":
		return mode
	default:
		return "partial"
	}
}

var telegramParseErrorPattern = regexp.MustCompile(`(?i)can't parse entities|parse entities|find end of the entity`)
var telegramMessageNotModifiedPattern = regexp.MustCompile(`(?i)message is not modified`)

const telegramApprovalStatusTTL = 3 * time.Second

func isTelegramParseError(err error) bool {
	if err == nil {
		return false
	}
	return telegramParseErrorPattern.MatchString(err.Error())
}

func isTelegramMessageNotModifiedError(err error) bool {
	if err == nil {
		return false
	}
	return telegramMessageNotModifiedPattern.MatchString(err.Error())
}

func (service *BotService) sendChunks(ctx context.Context, state *telegramAccountState, chatID int64, threadID int, chunks []telegramapi.FormattedChunk, replyTo int) error {
	return service.sendChunksWithOptions(ctx, state, chatID, threadID, chunks, replyTo, false)
}

func (service *BotService) sendChunksWithOptions(
	ctx context.Context,
	state *telegramAccountState,
	chatID int64,
	threadID int,
	chunks []telegramapi.FormattedChunk,
	replyTo int,
	silent bool,
) error {
	if len(chunks) == 0 {
		return nil
	}
	for i, chunk := range chunks {
		msg := tu.Message(tu.ID(chatID), chunk.HTML)
		msg.ParseMode = telego.ModeHTML
		msg.DisableNotification = silent
		if threadID > 0 {
			msg.MessageThreadID = threadID
		}
		if i == 0 && replyTo > 0 {
			msg.ReplyParameters = &telego.ReplyParameters{MessageID: replyTo}
		}
		sent, err := state.bot.SendMessage(ctx, msg)
		if err != nil && isTelegramParseError(err) && strings.TrimSpace(chunk.Text) != "" {
			fallback := tu.Message(tu.ID(chatID), chunk.Text)
			fallback.DisableNotification = silent
			if threadID > 0 {
				fallback.MessageThreadID = threadID
			}
			if i == 0 && replyTo > 0 {
				fallback.ReplyParameters = &telego.ReplyParameters{MessageID: replyTo}
			}
			sent, err = state.bot.SendMessage(ctx, fallback)
		}
		if err != nil {
			return err
		}
		service.recordOutboundSuccess(state, int64(sent.MessageID))
	}
	return nil
}

func (service *BotService) sendPlaceholder(ctx context.Context, state *telegramAccountState, chatID int64, threadID int, replyTo int, runID string) int {
	msg := tu.Message(tu.ID(chatID), telegramRunStatusThinkingText)
	if threadID > 0 {
		msg.MessageThreadID = threadID
	}
	if replyTo > 0 {
		msg.ReplyParameters = &telego.ReplyParameters{MessageID: replyTo}
	}
	if strings.TrimSpace(runID) != "" {
		msg.ReplyMarkup = &telego.InlineKeyboardMarkup{
			InlineKeyboard: [][]telego.InlineKeyboardButton{
				{
					tu.InlineKeyboardButton("Stop").WithCallbackData(fmt.Sprintf("/stop %s", strings.TrimSpace(runID))),
				},
			},
		}
	}
	sent, err := state.bot.SendMessage(ctx, msg)
	if err != nil {
		return 0
	}
	return sent.MessageID
}

func (service *BotService) sendRunStatusCard(
	ctx context.Context,
	state *telegramAccountState,
	chatID int64,
	threadID int,
	replyTo int,
	runID string,
) int {
	if service == nil || state == nil {
		return 0
	}
	messageID := service.sendPlaceholder(ctx, state, chatID, threadID, replyTo, runID)
	if messageID <= 0 {
		return 0
	}
	previous := service.swapTelegramRunStatusCard(state.config.AccountID, chatID, threadID, messageID)
	if previous > 0 && previous != messageID {
		service.deleteTelegramMessageBestEffort(state, chatID, previous)
	}
	return messageID
}

func (service *BotService) updateTelegramRunStatusCard(
	state *telegramAccountState,
	chatID int64,
	messageID int,
	runID string,
	text string,
	withStop bool,
) {
	if service == nil || state == nil || state.bot == nil || chatID == 0 || messageID <= 0 {
		return
	}
	trimmedText := strings.TrimSpace(text)
	if trimmedText == "" {
		return
	}
	editCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	params := &telego.EditMessageTextParams{
		ChatID:    tu.ID(chatID),
		MessageID: messageID,
		Text:      trimmedText,
	}
	if withStop && strings.TrimSpace(runID) != "" {
		params.ReplyMarkup = &telego.InlineKeyboardMarkup{
			InlineKeyboard: [][]telego.InlineKeyboardButton{
				{
					tu.InlineKeyboardButton("Stop").WithCallbackData(fmt.Sprintf("/stop %s", strings.TrimSpace(runID))),
				},
			},
		}
	} else {
		params.ReplyMarkup = &telego.InlineKeyboardMarkup{InlineKeyboard: [][]telego.InlineKeyboardButton{}}
	}
	if _, err := state.bot.EditMessageText(editCtx, params); err != nil {
		if isTelegramMessageNotModifiedError(err) {
			return
		}
		zap.L().Debug(
			"telegram run status update failed",
			zap.String("accountId", state.config.AccountID),
			zap.Int64("chatId", chatID),
			zap.Int("messageId", messageID),
			zap.String("error", redactTelegramToken(err.Error(), state.config.BotToken)),
		)
	}
}

func (service *BotService) finishTelegramRunStatusCard(
	state *telegramAccountState,
	accountID string,
	chatID int64,
	threadID int,
	messageID int,
	terminalText string,
) {
	if service == nil || state == nil || state.bot == nil || chatID == 0 || messageID <= 0 {
		return
	}
	text := strings.TrimSpace(terminalText)
	if text == "" {
		text = telegramRunStatusDoneText
	}
	service.updateTelegramRunStatusCard(state, chatID, messageID, "", text, false)
	service.scheduleTelegramRunStatusDelete(state, accountID, chatID, threadID, messageID, telegramApprovalStatusTTL)
}

func (service *BotService) editPlaceholder(ctx context.Context, state *telegramAccountState, chatID int64, messageID int, chunk telegramapi.FormattedChunk, clearReplyMarkup bool) (int, error) {
	params := &telego.EditMessageTextParams{
		ChatID:    tu.ID(chatID),
		MessageID: messageID,
		Text:      chunk.HTML,
		ParseMode: telego.ModeHTML,
	}
	if clearReplyMarkup {
		params.ReplyMarkup = &telego.InlineKeyboardMarkup{InlineKeyboard: [][]telego.InlineKeyboardButton{}}
	}
	edited, err := state.bot.EditMessageText(ctx, params)
	if err != nil && isTelegramParseError(err) && strings.TrimSpace(chunk.Text) != "" {
		fallbackParams := &telego.EditMessageTextParams{
			ChatID:    tu.ID(chatID),
			MessageID: messageID,
			Text:      chunk.Text,
		}
		if clearReplyMarkup {
			fallbackParams.ReplyMarkup = &telego.InlineKeyboardMarkup{InlineKeyboard: [][]telego.InlineKeyboardButton{}}
		}
		edited, err = state.bot.EditMessageText(ctx, fallbackParams)
	}
	if err != nil {
		return 0, err
	}
	if edited != nil {
		return edited.MessageID, nil
	}
	return messageID, nil
}

func (service *BotService) deletePlaceholder(ctx context.Context, state *telegramAccountState, chatID int64, messageID int) {
	if service == nil || state == nil || messageID <= 0 {
		return
	}
	if state.bot == nil {
		return
	}
	_ = state.bot.DeleteMessage(ctx, &telego.DeleteMessageParams{
		ChatID:    tu.ID(chatID),
		MessageID: messageID,
	})
}

func (service *BotService) scheduleTelegramMessageDelete(
	state *telegramAccountState,
	accountID string,
	chatID int64,
	threadID int,
	messageID int,
	delay time.Duration,
) {
	if service == nil || state == nil || chatID == 0 || messageID <= 0 {
		return
	}
	if delay <= 0 {
		delay = telegramApprovalStatusTTL
	}
	go func() {
		timer := time.NewTimer(delay)
		defer timer.Stop()
		<-timer.C
		service.deleteTelegramMessageBestEffort(state, chatID, messageID)
		service.clearTelegramStatusCard(accountID, chatID, threadID, messageID)
	}()
}

func (service *BotService) scheduleTelegramRunStatusDelete(
	state *telegramAccountState,
	accountID string,
	chatID int64,
	threadID int,
	messageID int,
	delay time.Duration,
) {
	if service == nil || state == nil || chatID == 0 || messageID <= 0 {
		return
	}
	if delay <= 0 {
		delay = telegramApprovalStatusTTL
	}
	go func() {
		timer := time.NewTimer(delay)
		defer timer.Stop()
		<-timer.C
		service.deleteTelegramMessageBestEffort(state, chatID, messageID)
		service.clearTelegramRunStatusCard(accountID, chatID, threadID, messageID)
	}()
}

func (service *BotService) deleteTelegramMessageBestEffort(state *telegramAccountState, chatID int64, messageID int) {
	if service == nil || state == nil || chatID == 0 || messageID <= 0 || state.bot == nil {
		return
	}
	deleteCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = state.bot.DeleteMessage(deleteCtx, &telego.DeleteMessageParams{
		ChatID:    tu.ID(chatID),
		MessageID: messageID,
	})
}

func (service *BotService) swapTelegramStatusCard(accountID string, chatID int64, threadID int, messageID int) int {
	if service == nil || chatID == 0 || messageID <= 0 {
		return 0
	}
	key := telegramStatusCardKey(accountID, chatID, threadID)
	service.mu.Lock()
	defer service.mu.Unlock()
	previous := service.statusCards[key]
	service.statusCards[key] = messageID
	return previous
}

func (service *BotService) swapTelegramRunStatusCard(accountID string, chatID int64, threadID int, messageID int) int {
	if service == nil || chatID == 0 || messageID <= 0 {
		return 0
	}
	key := telegramStatusCardKey(accountID, chatID, threadID)
	service.mu.Lock()
	defer service.mu.Unlock()
	previous := service.runStatus[key]
	service.runStatus[key] = messageID
	return previous
}

func (service *BotService) clearTelegramStatusCard(accountID string, chatID int64, threadID int, messageID int) {
	if service == nil || chatID == 0 || messageID <= 0 {
		return
	}
	key := telegramStatusCardKey(accountID, chatID, threadID)
	service.mu.Lock()
	defer service.mu.Unlock()
	current, ok := service.statusCards[key]
	if !ok || current != messageID {
		return
	}
	delete(service.statusCards, key)
}

func (service *BotService) clearTelegramRunStatusCard(accountID string, chatID int64, threadID int, messageID int) {
	if service == nil || chatID == 0 || messageID <= 0 {
		return
	}
	key := telegramStatusCardKey(accountID, chatID, threadID)
	service.mu.Lock()
	defer service.mu.Unlock()
	current, ok := service.runStatus[key]
	if !ok || current != messageID {
		return
	}
	delete(service.runStatus, key)
}

func telegramStatusCardKey(accountID string, chatID int64, threadID int) string {
	return fmt.Sprintf("%s:%d:%d", strings.TrimSpace(accountID), chatID, threadID)
}

func (service *BotService) sendDraftMessage(ctx context.Context, state *telegramAccountState, chatID int64, threadID int, replyTo int, chunk telegramapi.FormattedChunk) (int, error) {
	if service == nil || state == nil || state.bot == nil {
		return 0, errors.New("telegram bot unavailable")
	}
	msg := tu.Message(tu.ID(chatID), chunk.HTML)
	msg.ParseMode = telego.ModeHTML
	if threadID > 0 {
		msg.MessageThreadID = threadID
	}
	if replyTo > 0 {
		msg.ReplyParameters = &telego.ReplyParameters{MessageID: replyTo}
	}
	sent, err := state.bot.SendMessage(ctx, msg)
	if err != nil && isTelegramParseError(err) && strings.TrimSpace(chunk.Text) != "" {
		fallback := tu.Message(tu.ID(chatID), chunk.Text)
		if threadID > 0 {
			fallback.MessageThreadID = threadID
		}
		if replyTo > 0 {
			fallback.ReplyParameters = &telego.ReplyParameters{MessageID: replyTo}
		}
		sent, err = state.bot.SendMessage(ctx, fallback)
	}
	if err != nil {
		return 0, err
	}
	if sent == nil {
		return 0, nil
	}
	return sent.MessageID, nil
}

func (service *BotService) deleteDraftMessage(ctx context.Context, state *telegramAccountState, chatID int64, messageID int) error {
	if service == nil || state == nil || messageID <= 0 {
		return nil
	}
	if state.bot == nil {
		return nil
	}
	return state.bot.DeleteMessage(ctx, &telego.DeleteMessageParams{
		ChatID:    tu.ID(chatID),
		MessageID: messageID,
	})
}
