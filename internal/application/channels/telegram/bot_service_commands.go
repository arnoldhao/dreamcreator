package telegram

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	appcommands "dreamcreator/internal/application/commands"
	gatewayapprovals "dreamcreator/internal/application/gateway/approvals"
	settingsdto "dreamcreator/internal/application/settings/dto"
	skillsdto "dreamcreator/internal/application/skills/dto"
	telegramapi "dreamcreator/internal/infrastructure/telegram"
	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
	"go.uber.org/zap"
)

func (service *BotService) answerCallbackQuery(ctx context.Context, state *telegramAccountState, callback *telegramapi.CallbackQuery) {
	if service == nil || state == nil || state.bot == nil || callback == nil {
		return
	}
	callbackID := strings.TrimSpace(callback.ID)
	if callbackID == "" {
		return
	}
	ackCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	params := tu.CallbackQuery(callbackID)
	if _, _, handled, _ := parseExecApprovalCommand(callback.Data); handled {
		params = params.WithText("Processing approval...")
	}
	if err := state.bot.AnswerCallbackQuery(ackCtx, params); err != nil {
		zap.L().Debug(
			"telegram callback ack failed",
			zap.String("accountId", state.config.AccountID),
			zap.String("error", redactTelegramToken(err.Error(), state.config.BotToken)),
		)
	}
}

func parseExecApprovalCommand(text string) (approvalID string, decision string, handled bool, usageError string) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return "", "", false, ""
	}
	parts := strings.Fields(trimmed)
	if len(parts) == 0 {
		return "", "", false, ""
	}
	command := strings.ToLower(strings.TrimSpace(parts[0]))
	if !strings.HasPrefix(command, "/") {
		return "", "", false, ""
	}
	command = strings.TrimPrefix(command, "/")
	if at := strings.Index(command, "@"); at >= 0 {
		command = command[:at]
	}
	if command != "approve" {
		return "", "", false, ""
	}
	if len(parts) < 2 || strings.TrimSpace(parts[1]) == "" {
		return "", "", true, "Usage: /approve <id> approve|deny"
	}
	approvalID = strings.TrimSpace(parts[1])
	decision = "approve"
	if len(parts) >= 3 {
		switch strings.ToLower(strings.TrimSpace(parts[2])) {
		case "approve", "approved", "allow", "yes":
			decision = "approve"
		case "deny", "denied", "reject", "block", "no":
			decision = "deny"
		default:
			return "", "", true, "Usage: /approve <id> approve|deny"
		}
	}
	return approvalID, decision, true, ""
}

func isCallbackApprovalCommand(update telegramapi.Update, text string) bool {
	if update.CallbackQuery == nil {
		return false
	}
	_, _, handled, _ := parseExecApprovalCommand(text)
	return handled
}

func isExecApprovalUpdate(update telegramapi.Update) bool {
	text := ""
	if update.CallbackQuery != nil {
		text = strings.TrimSpace(update.CallbackQuery.Data)
	}
	if text == "" {
		message := firstMessage(update)
		if message != nil {
			text = strings.TrimSpace(message.Text)
			if text == "" {
				text = strings.TrimSpace(message.Caption)
			}
		}
	}
	_, _, handled, _ := parseExecApprovalCommand(text)
	return handled
}

func (service *BotService) sendSystemMessage(
	ctx context.Context,
	state *telegramAccountState,
	chatID int64,
	threadID int,
	replyTo int,
	text string,
	replyMarkup telego.ReplyMarkup,
) error {
	if service == nil || state == nil || state.bot == nil {
		return errors.New("telegram bot unavailable")
	}
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return nil
	}
	if target, ok := telegramEditTargetFromContext(ctx); ok && target.ChatID == chatID {
		params := &telego.EditMessageTextParams{
			ChatID:    tu.ID(target.ChatID),
			MessageID: target.MessageID,
			Text:      trimmed,
		}
		if replyMarkup != nil {
			if inlineMarkup, ok := replyMarkup.(*telego.InlineKeyboardMarkup); ok {
				params.ReplyMarkup = inlineMarkup
			}
		} else {
			params.ReplyMarkup = &telego.InlineKeyboardMarkup{InlineKeyboard: [][]telego.InlineKeyboardButton{}}
		}
		edited, err := state.bot.EditMessageText(ctx, params)
		if err == nil {
			editedMessageID := target.MessageID
			if edited != nil && edited.MessageID > 0 {
				editedMessageID = edited.MessageID
			}
			service.recordOutboundSuccess(state, int64(editedMessageID))
			return nil
		}
		if isTelegramMessageNotModifiedError(err) {
			return nil
		}
		replyTo = 0
	}
	msg := tu.Message(tu.ID(chatID), trimmed)
	if threadID > 0 {
		msg.MessageThreadID = threadID
	}
	if replyTo > 0 {
		msg.ReplyParameters = &telego.ReplyParameters{MessageID: replyTo}
	}
	if replyMarkup != nil {
		msg.ReplyMarkup = replyMarkup
	}
	sent, err := state.bot.SendMessage(ctx, msg)
	if err != nil {
		service.recordOutboundError(state, err)
		return err
	}
	if sent != nil {
		service.recordOutboundSuccess(state, int64(sent.MessageID))
	}
	return nil
}

func (service *BotService) handleExecApprovalCommand(
	ctx context.Context,
	state *telegramAccountState,
	message *telegramapi.Message,
	text string,
	senderID string,
) bool {
	approvalID, decision, handled, usageError := parseExecApprovalCommand(text)
	if !handled {
		return false
	}
	if state == nil {
		return true
	}
	chatID := int64(0)
	threadID := 0
	replyToMessageID := 0
	if message != nil {
		chatID = message.Chat.ID
		threadID = message.MessageThreadID
		replyToMessageID = message.MessageID
	}
	if usageError != "" {
		if chatID != 0 {
			_ = service.sendSystemMessage(ctx, state, chatID, threadID, replyToMessageID, usageError, nil)
		}
		return true
	}
	service.mu.Lock()
	resolver := service.approvals
	service.mu.Unlock()
	if resolver == nil {
		if chatID != 0 {
			_ = service.sendSystemMessage(
				ctx,
				state,
				chatID,
				threadID,
				replyToMessageID,
				"Exec approval is unavailable right now.",
				nil,
			)
		}
		return true
	}
	reason := "telegram"
	if trimmed := strings.TrimSpace(senderID); trimmed != "" {
		reason = fmt.Sprintf("telegram:%s", trimmed)
	}
	resolveCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	resolvedRequest, err := resolver.Resolve(resolveCtx, approvalID, decision, reason)
	if err != nil {
		if chatID != 0 {
			_ = service.sendSystemMessage(
				ctx,
				state,
				chatID,
				threadID,
				replyToMessageID,
				fmt.Sprintf("Failed to resolve approval %s: %s", approvalID, err.Error()),
				nil,
			)
		}
		return true
	}
	resultText := "approved"
	if resolvedRequest.Status == gatewayapprovals.StatusDenied ||
		(resolvedRequest.Status == "" && decision == "deny") {
		resultText = "denied"
	}
	if chatID != 0 {
		_ = service.sendSystemMessage(
			ctx,
			state,
			chatID,
			threadID,
			replyToMessageID,
			fmt.Sprintf("Exec approval %s: %s", resultText, approvalID),
			nil,
		)
	}
	if target, ok := telegramEditTargetFromContext(ctx); ok && chatID != 0 && target.ChatID == chatID {
		previous := service.swapTelegramStatusCard(state.config.AccountID, target.ChatID, threadID, target.MessageID)
		if previous > 0 && previous != target.MessageID {
			service.deleteTelegramMessageBestEffort(state, target.ChatID, previous)
		}
		service.scheduleTelegramMessageDelete(
			state,
			state.config.AccountID,
			target.ChatID,
			threadID,
			target.MessageID,
			telegramApprovalStatusTTL,
		)
	}
	return true
}

func (service *BotService) handleExecApprovalCommandAsync(
	ctx context.Context,
	state *telegramAccountState,
	message *telegramapi.Message,
	text string,
	senderID string,
) {
	if service == nil {
		return
	}
	approvalID, decision, handled, _ := parseExecApprovalCommand(text)
	if !handled {
		return
	}
	accountID := ""
	if state != nil {
		accountID = strings.TrimSpace(state.config.AccountID)
	}
	messageCopy := cloneTelegramMessageEnvelope(message)
	target, hasTarget := telegramEditTargetFromContext(ctx)
	go func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				zap.L().Warn(
					"telegram callback approval handler panicked",
					zap.String("accountId", accountID),
					zap.String("approvalId", approvalID),
					zap.Any("panic", recovered),
				)
			}
		}()
		asyncCtx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
		defer cancel()
		if hasTarget {
			asyncCtx = withTelegramEditTarget(asyncCtx, target.ChatID, target.MessageID)
		} else if messageCopy != nil && messageCopy.MessageID > 0 {
			asyncCtx = withTelegramEditTarget(asyncCtx, messageCopy.Chat.ID, messageCopy.MessageID)
		}
		zap.L().Debug(
			"telegram callback approval handling",
			zap.String("accountId", accountID),
			zap.String("approvalId", approvalID),
			zap.String("decision", decision),
		)
		service.handleExecApprovalCommand(asyncCtx, state, messageCopy, text, senderID)
	}()
}

func cloneTelegramMessageEnvelope(message *telegramapi.Message) *telegramapi.Message {
	if message == nil {
		return nil
	}
	copied := *message
	return &copied
}

func (service *BotService) resolveTelegramNativeCommandSpecs(ctx context.Context) ([]appcommands.NativeCommandSpec, map[string]appcommands.NativeCommandSpec) {
	if service == nil || service.settings == nil {
		return nil, nil
	}
	current, err := service.settings.GetSettings(ctx)
	if err != nil {
		return nil, nil
	}
	menu := ResolveMenuConfig(current)
	if !menu.NativeCommandsEnabled {
		return nil, nil
	}
	specs := FilterTelegramMenuNativeCommandSpecs(appcommands.ListNativeCommandSpecsForSettings(current, "telegram"))
	if len(specs) == 0 {
		return nil, nil
	}
	lookup := make(map[string]appcommands.NativeCommandSpec, len(specs))
	for _, spec := range specs {
		name := NormalizeTelegramCommandName(spec.Name)
		if name == "" {
			continue
		}
		lookup[name] = spec
	}
	return specs, lookup
}

func (service *BotService) isTelegramRegisteredCustomCommand(
	ctx context.Context,
	commandName string,
	nativeLookup map[string]appcommands.NativeCommandSpec,
) bool {
	normalized := NormalizeTelegramCommandName(commandName)
	if normalized == "" {
		return false
	}
	if _, ok := nativeLookup[normalized]; ok {
		return true
	}
	if service == nil || service.settings == nil {
		return false
	}
	current, err := service.settings.GetSettings(ctx)
	if err != nil {
		return false
	}
	return service.isTelegramRegisteredCommandInSettings(ctx, current, normalized)
}

func (service *BotService) isTelegramRegisteredCommandInSettings(
	ctx context.Context,
	settings settingsdto.Settings,
	normalizedCommand string,
) bool {
	if isTelegramRegisteredCustomCommandInSettings(settings, normalizedCommand) {
		return true
	}
	menu := ResolveMenuConfig(settings)
	if !menu.NativeCommandsEnabled || !menu.NativeSkillsEnabled {
		return false
	}
	service.mu.Lock()
	skills := service.skills
	service.mu.Unlock()
	if skills == nil {
		return false
	}
	providerID := strings.TrimSpace(settings.AgentModelProviderID)
	if providerID == "" {
		return false
	}
	response, err := skills.ResolveSkillPromptItems(ctx, skillsdto.ResolveSkillPromptRequest{ProviderID: providerID})
	if err != nil {
		return false
	}
	target := NormalizeTelegramCommandName(normalizedCommand)
	if target == "" {
		return false
	}
	for _, item := range response.Items {
		if NormalizeTelegramCommandName(item.Name) == target {
			return true
		}
	}
	return false
}

func isTelegramRegisteredCustomCommandInSettings(settings settingsdto.Settings, normalizedCommand string) bool {
	normalized := NormalizeTelegramCommandName(normalizedCommand)
	if normalized == "" {
		return false
	}
	menu := ResolveMenuConfig(settings)
	if len(menu.CustomCommands) == 0 {
		return false
	}
	allNativeCommands := appcommands.ListNativeCommandSpecsForSettings(settings, "telegram")
	reserved := make(map[string]struct{}, len(allNativeCommands))
	for _, command := range allNativeCommands {
		name := NormalizeTelegramCommandName(command.Name)
		if name == "" {
			continue
		}
		reserved[name] = struct{}{}
	}
	resolution := ResolveTelegramCustomCommands(ResolveCustomCommandsParams{
		Commands:         menu.CustomCommands,
		ReservedCommands: reserved,
	})
	for _, command := range resolution.Commands {
		if strings.EqualFold(strings.TrimSpace(command.Command), normalized) {
			return true
		}
	}
	return false
}

func (service *BotService) handleTelegramNativeCommand(
	ctx context.Context,
	state *telegramAccountState,
	message *telegramapi.Message,
	senderID string,
	baseSessionKey string,
	command telegramInboundCommand,
	specLookup map[string]appcommands.NativeCommandSpec,
) (bool, error) {
	if service == nil || state == nil || message == nil {
		return false, nil
	}
	name := strings.ToLower(strings.TrimSpace(command.Name))
	if name == "" {
		return false, nil
	}
	if name == "stop" || name == "abort" {
		err := service.handleTelegramStopCommand(ctx, state, message, command.Args)
		return true, err
	}
	spec, ok := specLookup[name]
	if !ok {
		return false, nil
	}
	switch spec.Key {
	case "help", "commands":
		return true, service.sendTelegramNativeCommandsHelp(ctx, state, message)
	case "whoami":
		resolvedSender := strings.TrimSpace(senderID)
		if resolvedSender == "" {
			resolvedSender = resolveMessageUserID(message)
		}
		if resolvedSender == "" {
			resolvedSender = "unknown"
		}
		return true, service.sendSystemMessage(
			ctx,
			state,
			message.Chat.ID,
			message.MessageThreadID,
			message.MessageID,
			fmt.Sprintf("Your Telegram sender id: %s", resolvedSender),
			nil,
		)
	case "status":
		mode := strings.TrimSpace(state.mode)
		if mode == "" {
			mode = "inactive"
		}
		commandState := service.getSessionCommandState(state.config.AccountID, baseSessionKey)
		activeModel := service.describeEffectiveModel(ctx, commandState)
		activeThink := normalizeTelegramThinkingLevel(commandState.ThinkingLevel)
		if activeThink == "" {
			activeThink = "default"
		}
		activeQueue := normalizeTelegramQueueMode(commandState.QueueMode)
		if activeQueue == "" {
			activeQueue = "default"
		}
		activeUsage := normalizeTelegramUsageMode(commandState.UsageMode)
		if activeUsage == "" {
			activeUsage = "off"
		}
		statusLines := []string{
			fmt.Sprintf("Account: %s", state.config.AccountID),
			fmt.Sprintf("Mode: %s", mode),
			fmt.Sprintf("Stream mode: %s", resolveTelegramStreamMode(state.config.StreamMode)),
			fmt.Sprintf("Session: %s", service.resolveTelegramRuntimeSessionKey(baseSessionKey, commandState)),
			fmt.Sprintf("Model: %s", activeModel),
			fmt.Sprintf("Think: %s", activeThink),
			fmt.Sprintf("Queue: %s", activeQueue),
			fmt.Sprintf("Usage: %s", activeUsage),
		}
		if state.lastRunID != "" {
			statusLines = append(statusLines, fmt.Sprintf("Last run: %s", state.lastRunID))
		}
		if state.lastRunError != "" {
			statusLines = append(statusLines, fmt.Sprintf("Last error: %s", state.lastRunError))
		}
		return true, service.sendSystemMessage(
			ctx,
			state,
			message.Chat.ID,
			message.MessageThreadID,
			message.MessageID,
			strings.Join(statusLines, "\n"),
			nil,
		)
	case "model":
		return true, service.handleTelegramModelCommand(ctx, state, message, baseSessionKey, command.Args)
	case "models":
		return true, service.handleTelegramModelsCommand(ctx, state, message, baseSessionKey, command.Args)
	case "new":
		return true, service.handleTelegramNewCommand(ctx, state, message, baseSessionKey, command.Args)
	case "reset":
		return true, service.handleTelegramResetCommand(ctx, state, message, baseSessionKey, command.Args)
	case "sessions":
		return true, service.handleTelegramSessionsCommand(ctx, state, message, baseSessionKey, command.Args)
	case "compact":
		return true, service.handleTelegramCompactCommand(ctx, state, message, baseSessionKey, command.Args)
	case "think":
		return true, service.handleTelegramThinkCommand(ctx, state, message, baseSessionKey, command.Args)
	case "usage":
		return true, service.handleTelegramUsageCommand(ctx, state, message, baseSessionKey, command.Args)
	case "reasoning":
		return true, service.handleTelegramReasoningCommand(ctx, state, message, baseSessionKey, command.Args)
	case "verbose":
		return true, service.handleTelegramVerboseCommand(ctx, state, message, baseSessionKey, command.Args)
	case "queue":
		return true, service.handleTelegramQueueCommand(ctx, state, message, baseSessionKey, command.Args)
	case "skill":
		return true, service.handleTelegramSkillCommand(ctx, state, message, baseSessionKey, command.Args)
	case "tts":
		return true, service.handleTelegramTTSCommand(ctx, state, message, baseSessionKey, command.Args)
	case "activation":
		return true, service.handleTelegramActivationCommand(ctx, state, message, baseSessionKey, command.Args)
	case "send":
		return true, service.handleTelegramSendCommand(ctx, state, message, baseSessionKey, command.Args)
	case "subagents":
		return true, service.handleTelegramSubagentsCommand(ctx, state, message, baseSessionKey, command.Args)
	case "kill":
		return true, service.handleTelegramKillCommand(ctx, state, message, baseSessionKey, command.Args)
	case "steer":
		return true, service.handleTelegramSteerCommand(ctx, state, message, baseSessionKey, command.Args)
	default:
		return true, service.sendSystemMessage(
			ctx,
			state,
			message.Chat.ID,
			message.MessageThreadID,
			message.MessageID,
			fmt.Sprintf("Command /%s is recognized but not implemented in Telegram yet.", NormalizeTelegramCommandName(spec.Name)),
			nil,
		)
	}
}

func (service *BotService) sendTelegramNativeCommandsHelp(ctx context.Context, state *telegramAccountState, message *telegramapi.Message) error {
	specs, _ := service.resolveTelegramNativeCommandSpecs(ctx)
	specs = FilterTelegramMenuNativeCommandSpecs(specs)
	if len(specs) == 0 {
		return service.sendSystemMessage(
			ctx,
			state,
			message.Chat.ID,
			message.MessageThreadID,
			message.MessageID,
			"No native Telegram commands are enabled.",
			nil,
		)
	}
	sort.SliceStable(specs, func(i, j int) bool {
		left := NormalizeTelegramCommandName(specs[i].Name)
		right := NormalizeTelegramCommandName(specs[j].Name)
		return left < right
	})
	var builder strings.Builder
	builder.WriteString("Available Telegram commands:\n")
	for _, spec := range specs {
		name := NormalizeTelegramCommandName(spec.Name)
		if name == "" {
			continue
		}
		desc := strings.TrimSpace(spec.Description)
		if desc == "" {
			desc = "No description."
		}
		builder.WriteString("/")
		builder.WriteString(name)
		builder.WriteString(" - ")
		builder.WriteString(desc)
		builder.WriteString("\n")
	}
	builder.WriteString("\nTip: /stop can cancel the current running reply.")
	text := strings.TrimSpace(builder.String())
	if utf8.RuneCountInString(text) > 3800 {
		text = truncateToRunes(text, 3800)
	}
	return service.sendSystemMessage(
		ctx,
		state,
		message.Chat.ID,
		message.MessageThreadID,
		message.MessageID,
		text,
		nil,
	)
}

func truncateToRunes(value string, limit int) string {
	if limit <= 0 || value == "" {
		return ""
	}
	if utf8.RuneCountInString(value) <= limit {
		return value
	}
	index := 0
	count := 0
	for i, r := range value {
		if count == limit {
			break
		}
		index = i + utf8.RuneLen(r)
		count++
	}
	trimmed := strings.TrimSpace(value[:index])
	if trimmed == "" {
		return ""
	}
	return trimmed + "…"
}

func buildTelegramToolStatusText(toolName string) string {
	name := strings.TrimSpace(strings.ReplaceAll(toolName, "\n", " "))
	emoji := "🛠️"
	normalized := strings.ToLower(name)
	for _, token := range telegramToolStatusWebTokens {
		if token != "" && strings.Contains(normalized, token) {
			emoji = "🌐"
			break
		}
	}
	if emoji == "🛠️" {
		for _, token := range telegramToolStatusCodingTokens {
			if token != "" && strings.Contains(normalized, token) {
				emoji = "💻"
				break
			}
		}
	}
	if name == "" {
		return fmt.Sprintf("Running tool... %s", emoji)
	}
	return fmt.Sprintf("Running tool: %s %s", truncateToRunes(name, telegramToolStatusNameLimit), emoji)
}

func firstField(value string) string {
	parts := strings.Fields(strings.TrimSpace(value))
	if len(parts) == 0 {
		return ""
	}
	return strings.TrimSpace(parts[0])
}

func (service *BotService) handleTelegramStopCommand(ctx context.Context, state *telegramAccountState, message *telegramapi.Message, args string) error {
	if service == nil || state == nil || message == nil {
		return nil
	}
	sessionKey := buildSessionKey(state.config.AccountID, message.Chat, int64(message.MessageThreadID))
	runID := firstField(args)
	aborted, resolvedRunID, err := service.abortActiveRun(ctx, state.config.AccountID, sessionKey, runID, "telegram stop command")
	if err != nil {
		return service.sendSystemMessage(
			ctx,
			state,
			message.Chat.ID,
			message.MessageThreadID,
			message.MessageID,
			fmt.Sprintf("Failed to stop run: %s", err.Error()),
			nil,
		)
	}
	if !aborted {
		return service.sendSystemMessage(
			ctx,
			state,
			message.Chat.ID,
			message.MessageThreadID,
			message.MessageID,
			"No active run to stop.",
			nil,
		)
	}
	if strings.TrimSpace(resolvedRunID) == "" {
		return service.sendSystemMessage(
			ctx,
			state,
			message.Chat.ID,
			message.MessageThreadID,
			message.MessageID,
			"Stop signal sent.",
			nil,
		)
	}
	return service.sendSystemMessage(
		ctx,
		state,
		message.Chat.ID,
		message.MessageThreadID,
		message.MessageID,
		fmt.Sprintf("Stop signal sent for run %s.", strings.TrimSpace(resolvedRunID)),
		nil,
	)
}
