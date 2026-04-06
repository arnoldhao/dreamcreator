package tools

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	telegramchannel "dreamcreator/internal/application/channels/telegram"
	settingsdto "dreamcreator/internal/application/settings/dto"
	telegramapi "dreamcreator/internal/infrastructure/telegram"
)

var (
	messageToolReasoningTagRE = regexp.MustCompile(`(?is)<think>.*?</think>`)
	messageToolNumericRE      = regexp.MustCompile(`^-?\d+$`)
	messageToolUsernameRE     = regexp.MustCompile(`^[A-Za-z0-9_]{5,}$`)
	messageToolTMeLinkRE      = regexp.MustCompile(`^(?i)(?:https?://)?t\.me/([A-Za-z0-9_]+)$`)
	messageToolDataURLRE      = regexp.MustCompile(`^data:([^;,]+)?(?:;charset=[^;,]+)?(;base64)?,(.*)$`)
	messageToolImageExts      = map[string]struct{}{
		".jpg":  {},
		".jpeg": {},
		".png":  {},
		".webp": {},
		".gif":  {},
		".bmp":  {},
		".tiff": {},
		".svg":  {},
	}
)

type messageToolRuntimeRoute struct {
	channel   string
	accountID string
	chat      string
	chatID    int64
	threadID  int
	hasChat   bool
}

type messageToolTarget struct {
	chat     string
	chatID   int64
	threadID int
}

type messageToolGatewayOptions struct {
	url      string
	token    string
	timeout  time.Duration
	provided bool
}

type messageToolMedia struct {
	source      string
	payload     []byte
	filename    string
	contentType string
	document    bool
}

type messageToolSchemaProfile struct {
	loaded            bool
	actions           []string
	includeButtons    bool
	includeCards      bool
	includeComponents bool
}

type messageToolSchemaActionGate struct {
	action string
	gate   string
}

type messageToolChannelHandler func(
	ctx context.Context,
	payload toolArgs,
	route messageToolRuntimeRoute,
	settings SettingsReader,
	action string,
) (map[string]any, error)

var messageToolChannelHandlers = map[string]messageToolChannelHandler{
	"telegram": runMessageToolForTelegramChannel,
}

var messageToolTelegramSchemaActionGates = []messageToolSchemaActionGate{
	{action: "send", gate: "sendMessage"},
	{action: "broadcast", gate: "sendMessage"},
	{action: "reply", gate: "sendMessage"},
	{action: "sendWithEffect", gate: "sendMessage"},
	{action: "sendAttachment", gate: "sendMessage"},
	{action: "thread-reply", gate: "sendMessage"},
	{action: "poll", gate: "polls"},
	{action: "react", gate: "reactions"},
	{action: "delete", gate: "deleteMessage"},
	{action: "unsend", gate: "deleteMessage"},
	{action: "edit", gate: "editMessage"},
}

func resolveMessageToolSchemaProfile(ctx context.Context, settings SettingsReader) messageToolSchemaProfile {
	if settings == nil {
		return messageToolSchemaProfile{}
	}
	current, err := settings.GetSettings(ctx)
	if err != nil {
		return messageToolSchemaProfile{}
	}
	actionSet := map[string]struct{}{}
	profile := messageToolSchemaProfile{
		loaded:            true,
		includeButtons:    false,
		includeCards:      false,
		includeComponents: false,
	}
	hasResolvedAction := false
	for _, channel := range resolveMessageToolConfiguredChannelsFromMap(current.Channels) {
		switch channel {
		case "telegram":
			actions := resolveMessageToolTelegramSchemaActions(current)
			if len(actions) > 0 {
				hasResolvedAction = true
			}
			for _, action := range actions {
				actionSet[action] = struct{}{}
			}
			if resolveMessageToolTelegramHasRunnableAccount(current) {
				profile.includeButtons = true
			}
		}
	}
	if !hasResolvedAction {
		actionSet["send"] = struct{}{}
		actionSet["broadcast"] = struct{}{}
	}
	profile.actions = resolveMessageToolSortedActions(actionSet)
	if len(profile.actions) == 0 {
		profile.actions = []string{"send"}
	}
	return profile
}

func resolveMessageToolTelegramSchemaActions(current settingsdto.Settings) []string {
	if !resolveMessageToolTelegramHasRunnableAccount(current) {
		return nil
	}
	channelRaw, _ := current.Channels["telegram"].(map[string]any)
	channel := cloneAnyMap(channelRaw)
	if channel == nil {
		channel = map[string]any{}
	}
	actionSet := map[string]struct{}{}
	runtime := telegramchannel.ResolveTelegramRuntimeConfig(current)
	for _, account := range runtime.Accounts {
		if !account.Enabled || strings.TrimSpace(account.BotToken) == "" {
			continue
		}
		accountID := strings.TrimSpace(account.AccountID)
		if accountID == "" {
			accountID = telegramchannel.DefaultTelegramAccountID
		}
		for _, item := range messageToolTelegramSchemaActionGates {
			if resolveMessageToolActionGate(channel, accountID, item.gate, true) {
				actionSet[item.action] = struct{}{}
			}
		}
	}
	return resolveMessageToolSortedActions(actionSet)
}

func resolveMessageToolTelegramHasRunnableAccount(current settingsdto.Settings) bool {
	runtime := telegramchannel.ResolveTelegramRuntimeConfig(current)
	for _, account := range runtime.Accounts {
		if !account.Enabled {
			continue
		}
		if strings.TrimSpace(account.BotToken) != "" {
			return true
		}
	}
	return false
}

func resolveMessageToolSortedActions(actions map[string]struct{}) []string {
	if len(actions) == 0 {
		return nil
	}
	result := make([]string, 0, len(actions))
	for action := range actions {
		trimmed := strings.TrimSpace(action)
		if trimmed == "" {
			continue
		}
		result = append(result, trimmed)
	}
	if len(result) == 0 {
		return nil
	}
	sort.Slice(result, func(i, j int) bool {
		left := strings.ToLower(result[i])
		right := strings.ToLower(result[j])
		if left == right {
			return result[i] < result[j]
		}
		return left < right
	})
	return result
}

func runMessageTool(settings SettingsReader) func(ctx context.Context, args string) (string, error) {
	return func(ctx context.Context, args string) (string, error) {
		payload, err := parseToolArgs(args)
		if err != nil {
			return "", err
		}
		rawAction := strings.TrimSpace(getStringArg(payload, "action"))
		if rawAction == "" {
			return "", errors.New("message action is required")
		}
		action := strings.ToLower(rawAction)
		route := resolveMessageToolRuntimeRoute(ctx)
		channel := resolveMessageToolChannel(payload, route, settings, ctx)
		if action == "broadcast" && strings.EqualFold(strings.TrimSpace(getStringArg(payload, "channel")), "all") {
			result, broadcastErr := runMessageToolBroadcastAllChannels(ctx, payload, route, settings)
			if broadcastErr != nil {
				return "", broadcastErr
			}
			result["action"] = rawAction
			return marshalResult(result), nil
		}
		result, invokeErr := runMessageToolForChannel(ctx, payload, route, settings, channel, action)
		if invokeErr != nil {
			return "", invokeErr
		}
		result["action"] = rawAction
		return marshalResult(result), nil
	}
}

func runMessageToolForChannel(
	ctx context.Context,
	payload toolArgs,
	route messageToolRuntimeRoute,
	settings SettingsReader,
	channel string,
	action string,
) (map[string]any, error) {
	handler, ok := messageToolChannelHandlers[channel]
	if !ok {
		return nil, fmt.Errorf("message channel not implemented yet: %s", channel)
	}
	return handler(ctx, payload, route, settings, action)
}

func runMessageToolForTelegramChannel(
	ctx context.Context,
	payload toolArgs,
	route messageToolRuntimeRoute,
	settings SettingsReader,
	action string,
) (map[string]any, error) {
	telegramCfg, err := resolveMessageToolTelegramAccount(ctx, settings, payload, route)
	if err != nil {
		return nil, err
	}
	dryRun, _ := getBoolArg(payload, "dryRun")
	accountID := strings.TrimSpace(telegramCfg.account.AccountID)
	switch action {
	case "send", "reply", "sendwitheffect", "sendattachment", "thread-reply":
		if !resolveMessageToolActionGate(telegramCfg.channel, accountID, "sendMessage", true) {
			return nil, errors.New("telegram sendMessage is disabled")
		}
		return runMessageToolSend(ctx, payload, route, telegramCfg, dryRun, action)
	case "broadcast":
		if !resolveMessageToolActionGate(telegramCfg.channel, accountID, "sendMessage", true) {
			return nil, errors.New("telegram sendMessage is disabled")
		}
		return runMessageToolBroadcast(ctx, payload, route, telegramCfg, dryRun)
	case "poll":
		if !resolveMessageToolActionGate(telegramCfg.channel, accountID, "polls", true) {
			return nil, errors.New("telegram polls is disabled")
		}
		return runMessageToolPoll(ctx, payload, route, telegramCfg, dryRun)
	case "react":
		if !resolveMessageToolActionGate(telegramCfg.channel, accountID, "reactions", true) {
			return nil, errors.New("telegram reactions is disabled")
		}
		return runMessageToolReact(ctx, payload, route, telegramCfg, dryRun)
	case "delete", "unsend":
		if !resolveMessageToolActionGate(telegramCfg.channel, accountID, "deleteMessage", true) {
			return nil, errors.New("telegram deleteMessage is disabled")
		}
		return runMessageToolDelete(ctx, payload, route, telegramCfg, dryRun)
	case "edit":
		if !resolveMessageToolActionGate(telegramCfg.channel, accountID, "editMessage", true) {
			return nil, errors.New("telegram editMessage is disabled")
		}
		return runMessageToolEdit(ctx, payload, route, telegramCfg, dryRun)
	default:
		return nil, fmt.Errorf("unsupported message action: %s", strings.TrimSpace(getStringArg(payload, "action")))
	}
}

func runMessageToolBroadcastAllChannels(
	ctx context.Context,
	payload toolArgs,
	route messageToolRuntimeRoute,
	settings SettingsReader,
) (map[string]any, error) {
	targets := getStringSliceArg(payload, "targets")
	if len(targets) == 0 {
		return nil, errors.New("broadcast requires at least one target")
	}
	channels := resolveMessageToolConfiguredChannels(settings, ctx)
	if len(channels) == 0 {
		channels = []string{"telegram"}
	}
	results := make([]map[string]any, 0, len(channels)*len(targets))
	for _, channel := range channels {
		for _, target := range targets {
			callPayload := cloneMessageToolArgs(payload)
			callPayload["action"] = "send"
			callPayload["channel"] = channel
			callPayload["target"] = target
			item, err := runMessageToolForChannel(ctx, callPayload, route, settings, channel, "send")
			entry := map[string]any{
				"channel": channel,
				"to":      target,
				"ok":      err == nil,
			}
			if err != nil {
				entry["error"] = err.Error()
			} else {
				entry["result"] = item
			}
			results = append(results, entry)
		}
	}
	dryRun, _ := getBoolArg(payload, "dryRun")
	return map[string]any{
		"ok":      true,
		"kind":    "broadcast",
		"dryRun":  dryRun,
		"results": results,
	}, nil
}

func runMessageToolBroadcast(
	ctx context.Context,
	payload toolArgs,
	route messageToolRuntimeRoute,
	cfg messageToolTelegramAccount,
	dryRun bool,
) (map[string]any, error) {
	targets := getStringSliceArg(payload, "targets")
	if len(targets) == 0 {
		return nil, errors.New("broadcast requires at least one target")
	}
	results := make([]map[string]any, 0, len(targets))
	for _, target := range targets {
		callPayload := cloneMessageToolArgs(payload)
		callPayload["target"] = target
		callPayload["action"] = "send"
		result, err := runMessageToolSend(ctx, callPayload, route, cfg, dryRun, "send")
		entry := map[string]any{
			"channel": "telegram",
			"to":      target,
			"ok":      err == nil,
		}
		if err != nil {
			entry["error"] = err.Error()
		} else {
			entry["result"] = result
		}
		results = append(results, entry)
	}
	return map[string]any{
		"ok":      true,
		"channel": "telegram",
		"kind":    "broadcast",
		"dryRun":  dryRun,
		"results": results,
	}, nil
}

func runMessageToolSend(
	ctx context.Context,
	payload toolArgs,
	route messageToolRuntimeRoute,
	cfg messageToolTelegramAccount,
	dryRun bool,
	action string,
) (map[string]any, error) {
	targetText, err := resolveMessageToolTarget(payload, route)
	if err != nil {
		return nil, err
	}
	target, err := parseMessageToolTelegramTarget(targetText)
	if err != nil {
		return nil, err
	}
	if threadOverride := resolveMessageToolOptionalPositiveInt(payload, "threadId"); threadOverride > 0 {
		target.threadID = threadOverride
	}
	replyTo := resolveMessageToolOptionalPositiveInt(payload, "replyTo")
	message := resolveMessageToolMessageText(payload)
	caption := resolveMessageToolCaption(payload, message)
	media, err := resolveMessageToolMedia(payload)
	if err != nil {
		return nil, err
	}
	if strings.EqualFold(action, "sendattachment") && media == nil {
		return nil, errors.New("sendAttachment requires media, path, filePath, or buffer")
	}
	if message == "" && media == nil {
		return nil, errors.New("send requires text or media")
	}
	buttons, err := resolveMessageToolButtons(payload)
	if err != nil {
		return nil, err
	}
	silent, _ := getBoolArg(payload, "silent")
	to := formatMessageToolTelegramTarget(target)
	if dryRun {
		return map[string]any{
			"ok":        true,
			"kind":      "send",
			"channel":   "telegram",
			"accountId": cfg.account.AccountID,
			"to":        to,
			"dryRun":    true,
		}, nil
	}
	if cfg.client == nil {
		return nil, errors.New("telegram client unavailable")
	}
	var sent telegramapi.Message
	if media != nil {
		if media.document {
			sent, err = cfg.client.SendDocument(ctx, telegramapi.SendDocumentParams{
				ChatID:              target.chatID,
				Chat:                target.chat,
				Document:            media.source,
				DocumentData:        media.payload,
				Filename:            media.filename,
				Caption:             caption,
				ReplyToMessageID:    int64(replyTo),
				MessageThreadID:     int64(target.threadID),
				DisableNotification: silent,
				Buttons:             buttons,
			})
		} else {
			sent, err = cfg.client.SendPhoto(ctx, telegramapi.SendPhotoParams{
				ChatID:              target.chatID,
				Chat:                target.chat,
				Photo:               media.source,
				PhotoData:           media.payload,
				Filename:            media.filename,
				Caption:             caption,
				ReplyToMessageID:    int64(replyTo),
				MessageThreadID:     int64(target.threadID),
				DisableNotification: silent,
				Buttons:             buttons,
			})
		}
	} else {
		formattedMessage, parseMode := formatMessageToolTelegramText(message)
		sent, err = cfg.client.SendMessage(ctx, telegramapi.SendMessageParams{
			ChatID:              target.chatID,
			Chat:                target.chat,
			Text:                formattedMessage,
			ReplyToMessageID:    int64(replyTo),
			MessageThreadID:     int64(target.threadID),
			DisableNotification: silent,
			ParseMode:           parseMode,
			Buttons:             buttons,
		})
		if err != nil && parseMode != "" {
			// Fallback to plain text if Telegram HTML parsing rejects this payload.
			sent, err = cfg.client.SendMessage(ctx, telegramapi.SendMessageParams{
				ChatID:              target.chatID,
				Chat:                target.chat,
				Text:                message,
				ReplyToMessageID:    int64(replyTo),
				MessageThreadID:     int64(target.threadID),
				DisableNotification: silent,
				Buttons:             buttons,
			})
		}
	}
	if err != nil {
		return nil, err
	}
	chatID := strings.TrimSpace(target.chat)
	if sent.Chat.ID != 0 {
		chatID = fmt.Sprintf("%d", sent.Chat.ID)
	}
	return map[string]any{
		"ok":        true,
		"kind":      "send",
		"channel":   "telegram",
		"accountId": cfg.account.AccountID,
		"to":        to,
		"chatId":    chatID,
		"messageId": sent.MessageID,
		"threadId":  target.threadID,
		"dryRun":    false,
	}, nil
}

func runMessageToolPoll(
	ctx context.Context,
	payload toolArgs,
	route messageToolRuntimeRoute,
	cfg messageToolTelegramAccount,
	dryRun bool,
) (map[string]any, error) {
	targetText, err := resolveMessageToolTarget(payload, route)
	if err != nil {
		return nil, err
	}
	target, err := parseMessageToolTelegramTarget(targetText)
	if err != nil {
		return nil, err
	}
	if threadOverride := resolveMessageToolOptionalPositiveInt(payload, "threadId"); threadOverride > 0 {
		target.threadID = threadOverride
	}
	question := strings.TrimSpace(getStringArg(payload, "pollQuestion"))
	if question == "" {
		return nil, errors.New("pollQuestion is required")
	}
	options := getStringSliceArg(payload, "pollOption")
	if len(options) < 2 {
		return nil, errors.New("pollOption requires at least two values")
	}
	allowMultiple, _ := getBoolArg(payload, "pollMulti")
	openPeriod := 0
	if durationHours, ok := getIntArg(payload, "pollDurationHours"); ok {
		seconds := durationHours * 3600
		if seconds > 0 {
			if seconds > 600 {
				seconds = 600
			}
			if seconds < 5 {
				seconds = 5
			}
			openPeriod = seconds
		}
	}
	silent, _ := getBoolArg(payload, "silent")
	to := formatMessageToolTelegramTarget(target)
	if dryRun {
		return map[string]any{
			"ok":        true,
			"kind":      "poll",
			"channel":   "telegram",
			"accountId": cfg.account.AccountID,
			"to":        to,
			"dryRun":    true,
		}, nil
	}
	if cfg.client == nil {
		return nil, errors.New("telegram client unavailable")
	}
	sent, err := cfg.client.SendPoll(ctx, telegramapi.SendPollParams{
		ChatID:              target.chatID,
		Chat:                target.chat,
		Question:            question,
		Options:             options,
		MessageThreadID:     int64(target.threadID),
		DisableNotification: silent,
		OpenPeriodSeconds:   openPeriod,
		AllowMultiple:       allowMultiple,
	})
	if err != nil {
		return nil, err
	}
	chatID := strings.TrimSpace(target.chat)
	if sent.Chat.ID != 0 {
		chatID = fmt.Sprintf("%d", sent.Chat.ID)
	}
	return map[string]any{
		"ok":        true,
		"kind":      "poll",
		"channel":   "telegram",
		"accountId": cfg.account.AccountID,
		"to":        to,
		"chatId":    chatID,
		"messageId": sent.MessageID,
		"threadId":  target.threadID,
		"dryRun":    false,
	}, nil
}

func runMessageToolReact(
	ctx context.Context,
	payload toolArgs,
	route messageToolRuntimeRoute,
	cfg messageToolTelegramAccount,
	dryRun bool,
) (map[string]any, error) {
	targetText, err := resolveMessageToolTarget(payload, route)
	if err != nil {
		return nil, err
	}
	target, err := parseMessageToolTelegramTarget(targetText)
	if err != nil {
		return nil, err
	}
	messageID := resolveMessageToolOptionalPositiveInt(payload, "messageId", "message_id")
	if messageID <= 0 {
		return nil, errors.New("messageId is required")
	}
	remove, _ := getBoolArg(payload, "remove")
	emoji := strings.TrimSpace(getStringArg(payload, "emoji"))
	if !remove && emoji == "" {
		return nil, errors.New("emoji is required")
	}
	to := formatMessageToolTelegramTarget(target)
	if dryRun {
		result := map[string]any{
			"ok":        true,
			"kind":      "react",
			"channel":   "telegram",
			"accountId": cfg.account.AccountID,
			"to":        to,
			"messageId": messageID,
			"dryRun":    true,
		}
		if remove {
			result["removed"] = true
		} else {
			result["added"] = emoji
		}
		return result, nil
	}
	if cfg.client == nil {
		return nil, errors.New("telegram client unavailable")
	}
	if err := cfg.client.SetMessageReaction(ctx, telegramapi.SetMessageReactionParams{
		ChatID:    target.chatID,
		Chat:      target.chat,
		MessageID: int64(messageID),
		Emoji:     emoji,
		Remove:    remove,
	}); err != nil {
		return nil, err
	}
	result := map[string]any{
		"ok":        true,
		"kind":      "react",
		"channel":   "telegram",
		"accountId": cfg.account.AccountID,
		"to":        to,
		"messageId": messageID,
		"dryRun":    false,
	}
	if remove {
		result["removed"] = true
	} else {
		result["added"] = emoji
	}
	return result, nil
}

func runMessageToolDelete(
	ctx context.Context,
	payload toolArgs,
	route messageToolRuntimeRoute,
	cfg messageToolTelegramAccount,
	dryRun bool,
) (map[string]any, error) {
	targetText, err := resolveMessageToolTarget(payload, route)
	if err != nil {
		return nil, err
	}
	target, err := parseMessageToolTelegramTarget(targetText)
	if err != nil {
		return nil, err
	}
	messageID := resolveMessageToolOptionalPositiveInt(payload, "messageId", "message_id")
	if messageID <= 0 {
		return nil, errors.New("messageId is required")
	}
	to := formatMessageToolTelegramTarget(target)
	if dryRun {
		return map[string]any{
			"ok":        true,
			"kind":      "action",
			"channel":   "telegram",
			"accountId": cfg.account.AccountID,
			"to":        to,
			"messageId": messageID,
			"deleted":   true,
			"dryRun":    true,
		}, nil
	}
	if cfg.client == nil {
		return nil, errors.New("telegram client unavailable")
	}
	if err := cfg.client.DeleteMessage(ctx, telegramapi.DeleteMessageParams{
		ChatID:    target.chatID,
		Chat:      target.chat,
		MessageID: int64(messageID),
	}); err != nil {
		return nil, err
	}
	return map[string]any{
		"ok":        true,
		"kind":      "action",
		"channel":   "telegram",
		"accountId": cfg.account.AccountID,
		"to":        to,
		"messageId": messageID,
		"deleted":   true,
		"dryRun":    false,
	}, nil
}

func runMessageToolEdit(
	ctx context.Context,
	payload toolArgs,
	route messageToolRuntimeRoute,
	cfg messageToolTelegramAccount,
	dryRun bool,
) (map[string]any, error) {
	targetText, err := resolveMessageToolTarget(payload, route)
	if err != nil {
		return nil, err
	}
	target, err := parseMessageToolTelegramTarget(targetText)
	if err != nil {
		return nil, err
	}
	messageID := resolveMessageToolOptionalPositiveInt(payload, "messageId", "message_id")
	if messageID <= 0 {
		return nil, errors.New("messageId is required")
	}
	message := resolveMessageToolMessageText(payload)
	if message == "" {
		return nil, errors.New("message is required")
	}
	buttons, err := resolveMessageToolButtons(payload)
	if err != nil {
		return nil, err
	}
	to := formatMessageToolTelegramTarget(target)
	if dryRun {
		return map[string]any{
			"ok":        true,
			"kind":      "action",
			"channel":   "telegram",
			"accountId": cfg.account.AccountID,
			"to":        to,
			"messageId": messageID,
			"edited":    true,
			"dryRun":    true,
		}, nil
	}
	if cfg.client == nil {
		return nil, errors.New("telegram client unavailable")
	}
	formattedMessage, parseMode := formatMessageToolTelegramText(message)
	edited, err := cfg.client.EditMessage(ctx, telegramapi.EditMessageParams{
		ChatID:    target.chatID,
		Chat:      target.chat,
		MessageID: int64(messageID),
		Text:      formattedMessage,
		ParseMode: parseMode,
		Buttons:   buttons,
	})
	if err != nil && parseMode != "" {
		edited, err = cfg.client.EditMessage(ctx, telegramapi.EditMessageParams{
			ChatID:    target.chatID,
			Chat:      target.chat,
			MessageID: int64(messageID),
			Text:      message,
			Buttons:   buttons,
		})
	}
	if err != nil {
		return nil, err
	}
	chatID := strings.TrimSpace(target.chat)
	if edited.Chat.ID != 0 {
		chatID = fmt.Sprintf("%d", edited.Chat.ID)
	}
	editedID := messageID
	if edited.MessageID > 0 {
		editedID = edited.MessageID
	}
	return map[string]any{
		"ok":        true,
		"kind":      "action",
		"channel":   "telegram",
		"accountId": cfg.account.AccountID,
		"to":        to,
		"chatId":    chatID,
		"messageId": editedID,
		"edited":    true,
		"dryRun":    false,
	}, nil
}

type messageToolTelegramAccount struct {
	account telegramchannel.TelegramAccountConfig
	channel map[string]any
	client  *telegramapi.Client
}

func resolveMessageToolTelegramAccount(
	ctx context.Context,
	settings SettingsReader,
	payload toolArgs,
	route messageToolRuntimeRoute,
) (messageToolTelegramAccount, error) {
	if settings == nil {
		return messageToolTelegramAccount{}, errors.New("settings service unavailable")
	}
	current, err := settings.GetSettings(ctx)
	if err != nil {
		return messageToolTelegramAccount{}, err
	}
	runtime := telegramchannel.ResolveTelegramRuntimeConfig(current)
	if len(runtime.Accounts) == 0 {
		return messageToolTelegramAccount{}, errors.New("telegram is not configured")
	}
	accountID := strings.TrimSpace(getStringArg(payload, "accountId"))
	if accountID == "" {
		accountID = strings.TrimSpace(route.accountID)
	}
	if accountID == "" {
		accountID = telegramchannel.DefaultTelegramAccountID
	}
	var selected telegramchannel.TelegramAccountConfig
	found := false
	for _, account := range runtime.Accounts {
		if strings.EqualFold(strings.TrimSpace(account.AccountID), accountID) {
			selected = account
			found = true
			break
		}
	}
	if !found {
		if strings.EqualFold(accountID, telegramchannel.DefaultTelegramAccountID) {
			selected = runtime.Accounts[0]
			found = true
			accountID = selected.AccountID
		} else {
			return messageToolTelegramAccount{}, fmt.Errorf("telegram account not found: %s", accountID)
		}
	}
	if !selected.Enabled {
		return messageToolTelegramAccount{}, fmt.Errorf("telegram account is disabled: %s", selected.AccountID)
	}
	gateway := resolveMessageToolGatewayOptions(payload)
	if gateway.token != "" {
		selected.BotToken = gateway.token
	}
	if strings.TrimSpace(selected.BotToken) == "" {
		return messageToolTelegramAccount{}, fmt.Errorf("telegram bot token missing for account: %s", selected.AccountID)
	}
	client, err := newMessageToolTelegramClient(selected, gateway)
	if err != nil {
		return messageToolTelegramAccount{}, err
	}
	channelRaw, _ := current.Channels["telegram"].(map[string]any)
	channelCfg := cloneAnyMap(channelRaw)
	if channelCfg == nil {
		channelCfg = map[string]any{}
	}
	return messageToolTelegramAccount{account: selected, channel: channelCfg, client: client}, nil
}

func newMessageToolTelegramClient(account telegramchannel.TelegramAccountConfig, gateway messageToolGatewayOptions) (*telegramapi.Client, error) {
	timeout := 10 * time.Second
	if account.Network.TimeoutSeconds > 0 {
		timeout = time.Duration(account.Network.TimeoutSeconds) * time.Second
	}
	if gateway.timeout > 0 {
		timeout = gateway.timeout
	}
	client := &http.Client{Timeout: timeout}
	proxyAddress := strings.TrimSpace(account.Network.Proxy)
	if proxyAddress != "" {
		proxyURL, err := url.Parse(proxyAddress)
		if err != nil {
			return nil, fmt.Errorf("invalid telegram proxy: %w", err)
		}
		transport := http.DefaultTransport.(*http.Transport).Clone()
		transport.Proxy = http.ProxyURL(proxyURL)
		client.Transport = transport
	}
	if gateway.url != "" {
		return telegramapi.NewClientWithOptions(account.BotToken, client, telegramapi.ClientOptions{
			APIServer: gateway.url,
		}), nil
	}
	return telegramapi.NewClient(account.BotToken, client), nil
}

func resolveMessageToolGatewayOptions(payload toolArgs) messageToolGatewayOptions {
	var options messageToolGatewayOptions
	if timeoutMs, ok := getIntArg(payload, "timeoutMs"); ok && timeoutMs > 0 {
		options.timeout = time.Duration(timeoutMs) * time.Millisecond
		options.provided = true
	}
	token := strings.TrimSpace(getStringArg(payload, "gatewayToken"))
	if token != "" {
		options.token = token
		options.provided = true
	}
	rawURL := strings.TrimSpace(getStringArg(payload, "gatewayUrl"))
	if rawURL != "" {
		parsed, err := url.Parse(rawURL)
		if err == nil && parsed.Scheme != "" && parsed.Host != "" {
			options.url = strings.TrimRight(rawURL, "/")
			options.provided = true
		}
	}
	return options
}

func resolveMessageToolActionGate(channel map[string]any, accountID string, key string, fallback bool) bool {
	if channel == nil {
		return fallback
	}
	accountsRaw, _ := channel["accounts"].(map[string]any)
	if accountCfg, ok := accountsRaw[accountID].(map[string]any); ok {
		if accountActions, ok := accountCfg["actions"].(map[string]any); ok {
			if value, ok := resolveMessageToolActionGateValue(accountActions[key]); ok {
				return value
			}
		}
	}
	if topActions, ok := channel["actions"].(map[string]any); ok {
		if value, ok := resolveMessageToolActionGateValue(topActions[key]); ok {
			return value
		}
	}
	return fallback
}

func resolveMessageToolActionGateValue(raw any) (bool, bool) {
	switch typed := raw.(type) {
	case bool:
		return typed, true
	case string:
		value := strings.ToLower(strings.TrimSpace(typed))
		if value == "true" || value == "1" || value == "yes" || value == "on" {
			return true, true
		}
		if value == "false" || value == "0" || value == "no" || value == "off" {
			return false, true
		}
	}
	return false, false
}

func resolveMessageToolRuntimeRoute(ctx context.Context) messageToolRuntimeRoute {
	sessionKey, _ := RuntimeContextFromContext(ctx)
	trimmed := strings.TrimSpace(sessionKey)
	if trimmed == "" {
		return messageToolRuntimeRoute{}
	}
	parts := strings.Split(trimmed, ":")
	if len(parts) < 4 || !strings.EqualFold(strings.TrimSpace(parts[0]), "telegram") {
		return messageToolRuntimeRoute{}
	}
	route := messageToolRuntimeRoute{channel: "telegram"}
	route.accountID = strings.TrimSpace(parts[1])
	if route.accountID == "" {
		route.accountID = telegramchannel.DefaultTelegramAccountID
	}
	route.chat = strings.TrimSpace(parts[3])
	if route.chat == "" {
		return route
	}
	route.hasChat = true
	if chatID, err := strconv.ParseInt(route.chat, 10, 64); err == nil {
		route.chatID = chatID
	}
	if len(parts) >= 6 && strings.EqualFold(strings.TrimSpace(parts[4]), "thread") {
		if parsed, parseErr := strconv.ParseInt(strings.TrimSpace(parts[5]), 10, 64); parseErr == nil && parsed > 0 {
			route.threadID = int(parsed)
		}
	}
	return route
}

func resolveMessageToolChannel(payload toolArgs, route messageToolRuntimeRoute, settings SettingsReader, ctx context.Context) string {
	channel := normalizeMessageToolChannel(getStringArg(payload, "channel"))
	if channel == "" {
		channel = normalizeMessageToolChannel(route.channel)
	}
	if channel == "" {
		configured := resolveMessageToolConfiguredChannels(settings, ctx)
		if len(configured) > 0 {
			return configured[0]
		}
		return "telegram"
	}
	return channel
}

func normalizeMessageToolChannel(raw string) string {
	channel := strings.ToLower(strings.TrimSpace(raw))
	if channel == "tg" {
		return "telegram"
	}
	return channel
}

func resolveMessageToolConfiguredChannels(settings SettingsReader, ctx context.Context) []string {
	if settings == nil {
		return nil
	}
	current, err := settings.GetSettings(ctx)
	if err != nil {
		return nil
	}
	return resolveMessageToolConfiguredChannelsFromMap(current.Channels)
}

func resolveMessageToolConfiguredChannelsFromMap(channels map[string]any) []string {
	if len(channels) == 0 {
		return nil
	}
	result := make([]string, 0, len(channels))
	for key, value := range channels {
		channel := normalizeMessageToolChannel(key)
		if channel == "" {
			continue
		}
		config, _ := value.(map[string]any)
		if config != nil {
			if enabled, ok := config["enabled"].(bool); ok && !enabled {
				continue
			}
		}
		result = append(result, channel)
	}
	if len(result) == 0 {
		return nil
	}
	return resolveMessageToolSortedStrings(result)
}

func resolveMessageToolSortedStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		normalized := strings.ToLower(strings.TrimSpace(value))
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		result = append(result, normalized)
	}
	if len(result) == 0 {
		return nil
	}
	for i := 0; i < len(result)-1; i++ {
		for j := i + 1; j < len(result); j++ {
			if result[j] < result[i] {
				result[i], result[j] = result[j], result[i]
			}
		}
	}
	return result
}

func resolveMessageToolTarget(payload toolArgs, route messageToolRuntimeRoute) (string, error) {
	target := strings.TrimSpace(getStringArg(payload, "target"))
	if target == "" {
		target = strings.TrimSpace(getStringArg(payload, "to", "channelId"))
	}
	if target == "" && route.hasChat {
		target = route.chat
		if route.threadID > 0 {
			target = fmt.Sprintf("%s:topic:%d", target, route.threadID)
		}
	}
	if target == "" {
		return "", errors.New("target is required")
	}
	return target, nil
}

func parseMessageToolTelegramTarget(raw string) (messageToolTarget, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return messageToolTarget{}, errors.New("target is required")
	}
	value = stripMessageToolTelegramPrefixes(value)
	chatPart := value
	threadID := 0
	if topicIndex := strings.LastIndex(strings.ToLower(value), ":topic:"); topicIndex > 0 {
		chatPart = strings.TrimSpace(value[:topicIndex])
		threadValue := strings.TrimSpace(value[topicIndex+len(":topic:"):])
		parsedThread, err := strconv.Atoi(threadValue)
		if err != nil || parsedThread <= 0 {
			return messageToolTarget{}, errors.New("invalid telegram topic id")
		}
		threadID = parsedThread
	} else {
		lastColon := strings.LastIndex(value, ":")
		if lastColon > 0 {
			chatCandidate := strings.TrimSpace(value[:lastColon])
			threadCandidate := strings.TrimSpace(value[lastColon+1:])
			if messageToolNumericRE.MatchString(chatCandidate) && messageToolNumericRE.MatchString(threadCandidate) {
				parsedThread, err := strconv.Atoi(threadCandidate)
				if err == nil && parsedThread > 0 {
					chatPart = chatCandidate
					threadID = parsedThread
				}
			}
		}
	}
	chat, numericID, ok := normalizeMessageToolTelegramLookupTarget(chatPart)
	if !ok {
		return messageToolTarget{}, errors.New("telegram target must be numeric chat id or username")
	}
	return messageToolTarget{chat: chat, chatID: numericID, threadID: threadID}, nil
}

func stripMessageToolTelegramPrefixes(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	strippedTelegram := false
	for {
		lower := strings.ToLower(trimmed)
		switch {
		case strings.HasPrefix(lower, "telegram:"):
			trimmed = strings.TrimSpace(trimmed[len("telegram:"):])
			strippedTelegram = true
			continue
		case strings.HasPrefix(lower, "tg:"):
			trimmed = strings.TrimSpace(trimmed[len("tg:"):])
			strippedTelegram = true
			continue
		case strippedTelegram && strings.HasPrefix(lower, "group:"):
			trimmed = strings.TrimSpace(trimmed[len("group:"):])
			continue
		default:
			return trimmed
		}
	}
}

func normalizeMessageToolTelegramLookupTarget(raw string) (string, int64, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", 0, false
	}
	if messageToolNumericRE.MatchString(trimmed) {
		parsed, err := strconv.ParseInt(trimmed, 10, 64)
		if err != nil {
			return "", 0, false
		}
		return trimmed, parsed, true
	}
	if match := messageToolTMeLinkRE.FindStringSubmatch(trimmed); len(match) == 2 {
		return "@" + match[1], 0, true
	}
	if strings.HasPrefix(trimmed, "@") {
		handle := strings.TrimPrefix(trimmed, "@")
		if messageToolUsernameRE.MatchString(handle) {
			return "@" + handle, 0, true
		}
		return "", 0, false
	}
	if messageToolUsernameRE.MatchString(trimmed) {
		return "@" + trimmed, 0, true
	}
	return "", 0, false
}

func resolveMessageToolMessageText(payload toolArgs) string {
	for _, key := range []string{"message", "text", "content"} {
		raw, ok := payload[key]
		if !ok {
			continue
		}
		cleaned := resolveMessageToolText(raw)
		if cleaned == "" {
			continue
		}
		return cleaned
	}
	return ""
}

func resolveMessageToolCaption(payload toolArgs, fallback string) string {
	caption := resolveMessageToolText(payload["caption"])
	if caption != "" {
		return caption
	}
	return fallback
}

func resolveMessageToolText(raw any) string {
	text, ok := raw.(string)
	if !ok {
		return ""
	}
	cleaned := strings.TrimSpace(messageToolReasoningTagRE.ReplaceAllString(text, ""))
	if cleaned == "" {
		return ""
	}
	return strings.ReplaceAll(cleaned, "\\n", "\n")
}

func formatMessageToolTelegramText(message string) (string, string) {
	trimmed := strings.TrimSpace(message)
	if trimmed == "" {
		return "", ""
	}
	rendered := strings.TrimSpace(telegramapi.RenderTelegramHTML(trimmed))
	if rendered == "" {
		return trimmed, ""
	}
	return rendered, "HTML"
}

func resolveMessageToolMedia(payload toolArgs) (*messageToolMedia, error) {
	buffer := strings.TrimSpace(resolveMessageToolRawString(payload["buffer"]))
	source := strings.TrimSpace(getStringArg(payload, "media", "path", "filePath"))
	contentType := strings.TrimSpace(getStringArg(payload, "contentType", "mimeType"))
	filename := strings.TrimSpace(getStringArg(payload, "filename"))
	if buffer == "" && strings.HasPrefix(strings.ToLower(source), "data:") {
		buffer = source
		source = ""
	}
	if buffer != "" {
		decoded, inferredContentType, err := resolveMessageToolPayloadBuffer(buffer)
		if err != nil {
			return nil, err
		}
		if contentType == "" {
			contentType = inferredContentType
		}
		if filename == "" {
			filename = resolveMessageToolFilename("", contentType)
		}
		return &messageToolMedia{
			payload:     decoded,
			filename:    filename,
			contentType: contentType,
			document:    resolveMessageToolDocumentMode("", filename, contentType, true),
		}, nil
	}
	if source == "" {
		return nil, nil
	}
	if filename == "" {
		filename = resolveMessageToolFilename(source, contentType)
	}
	return &messageToolMedia{
		source:      source,
		filename:    filename,
		contentType: contentType,
		document:    resolveMessageToolDocumentMode(source, filename, contentType, false),
	}, nil
}

func resolveMessageToolPayloadBuffer(raw string) ([]byte, string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, "", errors.New("buffer must not be empty")
	}
	if matches := messageToolDataURLRE.FindStringSubmatch(trimmed); len(matches) == 4 {
		contentType := strings.TrimSpace(matches[1])
		payload := strings.TrimSpace(matches[3])
		if strings.EqualFold(strings.TrimSpace(matches[2]), ";base64") {
			decoded, err := decodeMessageToolBase64(payload)
			if err != nil {
				return nil, "", errors.New("buffer data URL is not valid base64")
			}
			return decoded, contentType, nil
		}
		decoded, err := url.QueryUnescape(payload)
		if err != nil {
			return nil, "", errors.New("buffer data URL is invalid")
		}
		return []byte(decoded), contentType, nil
	}
	decoded, err := decodeMessageToolBase64(trimmed)
	if err != nil {
		return nil, "", errors.New("buffer must be base64 payload or data URL")
	}
	return decoded, "", nil
}

func decodeMessageToolBase64(raw string) ([]byte, error) {
	cleaned := strings.Map(func(r rune) rune {
		switch r {
		case ' ', '\n', '\r', '\t':
			return -1
		default:
			return r
		}
	}, raw)
	if cleaned == "" {
		return nil, errors.New("empty base64 payload")
	}
	if decoded, err := base64.StdEncoding.DecodeString(cleaned); err == nil {
		return decoded, nil
	}
	if decoded, err := base64.RawStdEncoding.DecodeString(cleaned); err == nil {
		return decoded, nil
	}
	return nil, errors.New("invalid base64 payload")
}

func resolveMessageToolFilename(source string, contentType string) string {
	candidate := resolveMessageToolFilenameCandidate(source)
	if candidate != "" {
		return candidate
	}
	contentType = strings.TrimSpace(strings.ToLower(contentType))
	if contentType != "" {
		extensions, err := mime.ExtensionsByType(contentType)
		if err == nil {
			for _, ext := range extensions {
				trimmed := strings.TrimSpace(ext)
				if trimmed != "" {
					return "attachment" + trimmed
				}
			}
		}
	}
	return "attachment"
}

func resolveMessageToolFilenameCandidate(source string) string {
	trimmed := strings.TrimSpace(source)
	if trimmed == "" {
		return ""
	}
	if parsed, err := url.Parse(trimmed); err == nil {
		switch strings.ToLower(parsed.Scheme) {
		case "http", "https", "file":
			base := strings.TrimSpace(filepath.Base(parsed.Path))
			if base != "" && base != "." && base != "/" {
				return base
			}
		}
	}
	base := strings.TrimSpace(filepath.Base(trimmed))
	if base == "" || base == "." || base == "/" {
		return ""
	}
	if strings.Contains(base, ":") && !strings.Contains(trimmed, "/") {
		return ""
	}
	return base
}

func resolveMessageToolDocumentMode(source string, filename string, contentType string, preferDocument bool) bool {
	contentType = strings.TrimSpace(strings.ToLower(contentType))
	if contentType != "" {
		return !strings.HasPrefix(contentType, "image/")
	}
	ext := strings.ToLower(strings.TrimSpace(filepath.Ext(filename)))
	if ext == "" {
		ext = resolveMessageToolSourceExt(source)
	}
	if ext != "" {
		_, isImage := messageToolImageExts[ext]
		return !isImage
	}
	return preferDocument
}

func resolveMessageToolSourceExt(source string) string {
	trimmed := strings.TrimSpace(source)
	if trimmed == "" {
		return ""
	}
	if parsed, err := url.Parse(trimmed); err == nil {
		switch strings.ToLower(parsed.Scheme) {
		case "http", "https", "file":
			return strings.ToLower(strings.TrimSpace(filepath.Ext(parsed.Path)))
		}
	}
	if strings.HasPrefix(trimmed, "/") || strings.Contains(trimmed, `\`) || strings.Contains(trimmed, "/") {
		if stat, err := os.Stat(trimmed); err == nil && !stat.IsDir() {
			return strings.ToLower(strings.TrimSpace(filepath.Ext(trimmed)))
		}
		return strings.ToLower(strings.TrimSpace(filepath.Ext(trimmed)))
	}
	return ""
}

func resolveMessageToolButtons(payload toolArgs) ([][]telegramapi.InlineButton, error) {
	raw, exists := payload["buttons"]
	if !exists || raw == nil {
		return nil, nil
	}
	rowsRaw, ok := raw.([]any)
	if !ok {
		return nil, errors.New("buttons must be an array")
	}
	rows := make([][]telegramapi.InlineButton, 0, len(rowsRaw))
	for i, rowRaw := range rowsRaw {
		cellsRaw, ok := rowRaw.([]any)
		if !ok {
			return nil, fmt.Errorf("buttons[%d] must be an array", i)
		}
		buttons := make([]telegramapi.InlineButton, 0, len(cellsRaw))
		for j, itemRaw := range cellsRaw {
			item, ok := itemRaw.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("buttons[%d][%d] must be an object", i, j)
			}
			text := strings.TrimSpace(resolveMessageToolRawString(item["text"]))
			callback := strings.TrimSpace(resolveMessageToolRawString(item["callback_data"]))
			if text == "" || callback == "" {
				return nil, fmt.Errorf("buttons[%d][%d] requires text and callback_data", i, j)
			}
			style := strings.ToLower(strings.TrimSpace(resolveMessageToolRawString(item["style"])))
			if style != "" && style != "danger" && style != "success" && style != "primary" {
				return nil, fmt.Errorf("buttons[%d][%d] style must be one of danger/success/primary", i, j)
			}
			buttons = append(buttons, telegramapi.InlineButton{
				Text:         text,
				CallbackData: callback,
				Style:        style,
			})
		}
		if len(buttons) > 0 {
			rows = append(rows, buttons)
		}
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows, nil
}

func resolveMessageToolRawString(raw any) string {
	switch typed := raw.(type) {
	case string:
		return typed
	case []byte:
		return string(typed)
	default:
		return ""
	}
}

func resolveMessageToolOptionalPositiveInt(payload toolArgs, keys ...string) int {
	value, ok := getIntArg(payload, keys...)
	if !ok || value <= 0 {
		return 0
	}
	return value
}

func cloneMessageToolArgs(payload toolArgs) toolArgs {
	if len(payload) == 0 {
		return toolArgs{}
	}
	cloned := make(toolArgs, len(payload))
	for key, value := range payload {
		cloned[key] = value
	}
	return cloned
}

func formatMessageToolTelegramTarget(target messageToolTarget) string {
	base := "telegram:" + strings.TrimSpace(target.chat)
	if target.threadID > 0 {
		return fmt.Sprintf("%s:topic:%d", base, target.threadID)
	}
	return base
}
