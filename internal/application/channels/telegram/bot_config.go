package telegram

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	settingsdto "dreamcreator/internal/application/settings/dto"
)

const (
	DefaultTelegramAccountID = "default"
)

type DMPolicy string
type GroupPolicy string

const (
	DMPolicyPairing   DMPolicy = "pairing"
	DMPolicyAllowlist DMPolicy = "allowlist"
	DMPolicyOpen      DMPolicy = "open"
	DMPolicyDisabled  DMPolicy = "disabled"

	GroupPolicyAllowlist GroupPolicy = "allowlist"
	GroupPolicyOpen      GroupPolicy = "open"
	GroupPolicyDisabled  GroupPolicy = "disabled"
)

type TelegramGroupConfig struct {
	Enabled        *bool
	RequireMention *bool
	AllowFrom      map[string]struct{}
	Tools          []string
	SystemPrompt   string
	Topics         map[string]TelegramTopicConfig
}

type TelegramTopicConfig struct {
	Enabled        *bool
	RequireMention *bool
	AllowFrom      map[string]struct{}
	Tools          []string
	SystemPrompt   string
}

type TelegramNetworkConfig struct {
	Proxy                  string
	TimeoutSeconds         int
	RetryMax               int
	RetryBackoffSeconds    int
	RetryMaxBackoffSeconds int
}

type TelegramPollingConfig struct {
	Limit             int
	TimeoutSeconds    int
	Concurrency       int
	QueueSize         int
	BackoffSeconds    int
	MaxBackoffSeconds int
}

type TelegramChunkConfig struct {
	TextChunkLimit int
	Mode           string
}

type TelegramDraftChunkConfig struct {
	MinChars        int
	MaxChars        int
	BreakPreference string
}

type TelegramAccountConfig struct {
	AccountID           string
	Enabled             bool
	BotToken            string
	AckReaction         string
	AckReactionScope    string
	RemoveAckAfterReply bool
	DMPolicy            DMPolicy
	GroupPolicy         GroupPolicy
	AllowFrom           map[string]struct{}
	GroupAllowFrom      map[string]struct{}
	Groups              map[string]TelegramGroupConfig
	StreamMode          string
	ReplyToMode         string
	WebhookURL          string
	WebhookSecret       string
	WebhookPath         string
	WebhookHost         string
	Network             TelegramNetworkConfig
	Polling             TelegramPollingConfig
	Chunk               TelegramChunkConfig
	DraftChunk          TelegramDraftChunkConfig
}

type TelegramRuntimeConfig struct {
	Accounts []TelegramAccountConfig
}

func ResolveTelegramRuntimeConfig(settings settingsdto.Settings) TelegramRuntimeConfig {
	channel := resolveMap(settings.Channels["telegram"])
	if len(channel) == 0 {
		return TelegramRuntimeConfig{}
	}
	base := parseTelegramAccount(DefaultTelegramAccountID, channel, channel)
	accounts := make([]TelegramAccountConfig, 0)

	if accountMap := resolveMap(channel["accounts"]); len(accountMap) > 0 {
		for accountID, raw := range accountMap {
			entry := resolveMap(raw)
			if len(entry) == 0 {
				continue
			}
			accounts = append(accounts, parseTelegramAccount(accountID, channel, entry))
		}
	}
	if base.BotToken != "" || len(accounts) == 0 {
		accounts = append(accounts, base)
	}
	return TelegramRuntimeConfig{Accounts: accounts}
}

func parseTelegramAccount(accountID string, base map[string]any, record map[string]any) TelegramAccountConfig {
	cfg := TelegramAccountConfig{
		AccountID:           accountID,
		Enabled:             resolveBoolWithFallback(record, base, "enabled", true),
		BotToken:            resolveBotToken(accountID, record, base),
		AckReaction:         normalizeAckReaction(resolveStringWithFallback(record, base, "ackReaction")),
		AckReactionScope:    normalizeAckReactionScope(resolveStringWithFallback(record, base, "ackReactionScope")),
		RemoveAckAfterReply: resolveBoolWithFallback(record, base, "removeAckAfterReply", false),
		StreamMode:          resolveStringWithFallback(record, base, "streamMode"),
		ReplyToMode:         resolveStringWithFallback(record, base, "replyToMode"),
		WebhookURL:          resolveStringWithFallback(record, base, "webhookUrl"),
		WebhookSecret:       resolveStringWithFallback(record, base, "webhookSecret"),
		WebhookPath:         resolveStringWithFallback(record, base, "webhookPath"),
		WebhookHost:         resolveStringWithFallback(record, base, "webhookHost"),
	}
	cfg.DMPolicy = normalizeDMPolicy(resolveStringWithFallback(record, base, "dmPolicy"))
	cfg.GroupPolicy = normalizeGroupPolicy(resolveStringWithFallback(record, base, "groupPolicy"))
	cfg.AllowFrom = mergeStringSet(resolveStringSet(base["allowFrom"]), resolveStringSet(record["allowFrom"]))
	cfg.GroupAllowFrom = mergeStringSet(resolveStringSet(base["groupAllowFrom"]), resolveStringSet(record["groupAllowFrom"]))
	cfg.Groups = mergeGroupConfigs(resolveTelegramGroups(base["groups"]), resolveTelegramGroups(record["groups"]))
	cfg.Chunk = resolveTelegramChunkConfig(record, base)
	cfg.DraftChunk = resolveTelegramDraftChunkConfig(record, base, cfg.Chunk.TextChunkLimit)
	cfg.Network = resolveTelegramNetworkConfig(record, base)
	cfg.Polling = resolveTelegramPollingConfig(record, base)
	return cfg
}

func normalizeDMPolicy(value string) DMPolicy {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "open":
		return DMPolicyOpen
	case "allowlist":
		return DMPolicyAllowlist
	case "disabled":
		return DMPolicyDisabled
	case "pairing":
		return DMPolicyPairing
	default:
		return DMPolicyPairing
	}
}

func normalizeGroupPolicy(value string) GroupPolicy {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "open":
		return GroupPolicyOpen
	case "disabled":
		return GroupPolicyDisabled
	case "allowlist":
		return GroupPolicyAllowlist
	default:
		return GroupPolicyAllowlist
	}
}

func normalizeAckReaction(value string) string {
	return strings.TrimSpace(value)
}

func normalizeAckReactionScope(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "group-all":
		return "group-all"
	case "direct":
		return "direct"
	case "all":
		return "all"
	default:
		return "group-mentions"
	}
}

func resolveBotToken(accountID string, record map[string]any, base map[string]any) string {
	if token := normalizeBotToken(readTokenFile(resolveString(record["tokenFile"]))); token != "" {
		return token
	}
	if token := normalizeBotToken(strings.TrimSpace(resolveString(record["botToken"]))); token != "" {
		return token
	}
	if token := normalizeBotToken(readTokenFile(resolveString(base["tokenFile"]))); token != "" {
		return token
	}
	if token := normalizeBotToken(strings.TrimSpace(resolveString(base["botToken"]))); token != "" {
		return token
	}
	if strings.TrimSpace(accountID) == "" || accountID == DefaultTelegramAccountID {
		return normalizeBotToken(strings.TrimSpace(os.Getenv("TELEGRAM_BOT_TOKEN")))
	}
	return ""
}

func normalizeBotToken(token string) string {
	trimmed := strings.TrimSpace(token)
	if trimmed == "" {
		return ""
	}
	lower := strings.ToLower(trimmed)
	if strings.Contains(lower, "telegram.org/bot") {
		if idx := strings.LastIndex(lower, "/bot"); idx >= 0 && idx+4 < len(trimmed) {
			trimmed = strings.TrimSpace(trimmed[idx+4:])
			lower = strings.ToLower(trimmed)
		}
	}
	if strings.HasPrefix(lower, "bot") && strings.Contains(trimmed, ":") {
		trimmed = strings.TrimSpace(trimmed[3:])
		lower = strings.ToLower(trimmed)
	}
	if !strings.Contains(trimmed, ":") {
		return ""
	}
	return trimmed
}

func readTokenFile(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return ""
	}
	data, err := os.ReadFile(trimmed)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func resolveStringWithFallback(record map[string]any, base map[string]any, key string) string {
	if record != nil {
		if value, ok := record[key]; ok {
			return strings.TrimSpace(resolveString(value))
		}
	}
	if base != nil {
		if value, ok := base[key]; ok {
			return strings.TrimSpace(resolveString(value))
		}
	}
	return ""
}

func resolveBoolWithFallback(record map[string]any, base map[string]any, key string, fallback bool) bool {
	if record != nil {
		if value, ok := record[key]; ok {
			return resolveBool(value, fallback)
		}
	}
	if base != nil {
		if value, ok := base[key]; ok {
			return resolveBool(value, fallback)
		}
	}
	return fallback
}

func resolveTelegramChunkConfig(record map[string]any, base map[string]any) TelegramChunkConfig {
	limit := resolveIntWithFallback(record, base, "textChunkLimit", 0)
	if limit <= 0 {
		limit = 3800
	}
	mode := resolveStringWithFallback(record, base, "chunkMode")
	return TelegramChunkConfig{
		TextChunkLimit: limit,
		Mode:           mode,
	}
}

func resolveTelegramDraftChunkConfig(record map[string]any, base map[string]any, textLimit int) TelegramDraftChunkConfig {
	draftRecord := resolveMap(record["draftChunk"])
	draftBase := resolveMap(base["draftChunk"])
	minChars, ok := resolveFirstInt([]map[string]any{draftRecord, draftBase}, []string{"minChars"})
	if !ok || minChars <= 0 {
		minChars = 200
	}
	maxChars, ok := resolveFirstInt([]map[string]any{draftRecord, draftBase}, []string{"maxChars"})
	if !ok || maxChars <= 0 {
		maxChars = 800
	}
	if textLimit <= 0 {
		textLimit = 4096
	}
	if maxChars > textLimit {
		maxChars = textLimit
	}
	if minChars > maxChars {
		minChars = maxChars
	}
	breakPref, ok := resolveFirstString([]map[string]any{draftRecord, draftBase}, []string{"breakPreference"})
	if !ok {
		breakPref = ""
	}
	switch strings.ToLower(strings.TrimSpace(breakPref)) {
	case "newline", "sentence":
		// keep
	default:
		breakPref = "paragraph"
	}
	return TelegramDraftChunkConfig{
		MinChars:        minChars,
		MaxChars:        maxChars,
		BreakPreference: breakPref,
	}
}

func resolveTelegramNetworkConfig(record map[string]any, base map[string]any) TelegramNetworkConfig {
	networkRecord := resolveMap(record["network"])
	networkBase := resolveMap(base["network"])
	retryRecord := resolveMap(record["retry"])
	retryBase := resolveMap(base["retry"])

	proxy, _ := resolveFirstString([]map[string]any{record, networkRecord, base, networkBase}, []string{"proxy"})
	timeoutSeconds, ok := resolveFirstInt([]map[string]any{record, networkRecord, base, networkBase}, []string{"timeoutSeconds", "timeout"})
	if !ok {
		if timeoutMs, ok := resolveFirstInt([]map[string]any{record, networkRecord, base, networkBase}, []string{"timeoutMs"}); ok {
			timeoutSeconds = (timeoutMs + 999) / 1000
		}
	}
	retryMax, _ := resolveFirstInt([]map[string]any{record, retryRecord, base, retryBase}, []string{"retryMax", "max", "maxAttempts"})
	retryBackoff, _ := resolveFirstInt([]map[string]any{record, retryRecord, base, retryBase}, []string{"retryBackoffSeconds", "backoffSeconds", "backoff"})
	retryMaxBackoff, _ := resolveFirstInt([]map[string]any{record, retryRecord, base, retryBase}, []string{"retryMaxBackoffSeconds", "maxBackoffSeconds", "maxBackoff"})
	return TelegramNetworkConfig{
		Proxy:                  proxy,
		TimeoutSeconds:         timeoutSeconds,
		RetryMax:               retryMax,
		RetryBackoffSeconds:    retryBackoff,
		RetryMaxBackoffSeconds: retryMaxBackoff,
	}
}

func resolveTelegramPollingConfig(record map[string]any, base map[string]any) TelegramPollingConfig {
	pollingRecord := resolveMap(record["polling"])
	pollingBase := resolveMap(base["polling"])
	limit, _ := resolveFirstInt([]map[string]any{record, pollingRecord, base, pollingBase}, []string{"pollingLimit", "limit"})
	timeoutSeconds, _ := resolveFirstInt([]map[string]any{record, pollingRecord, base, pollingBase}, []string{"pollingTimeoutSeconds", "timeoutSeconds", "timeout"})
	concurrency, _ := resolveFirstInt([]map[string]any{record, pollingRecord, base, pollingBase}, []string{"pollingConcurrency", "concurrency", "workers"})
	queueSize, _ := resolveFirstInt([]map[string]any{record, pollingRecord, base, pollingBase}, []string{"pollingQueueSize", "queueSize"})
	backoffSeconds, _ := resolveFirstInt([]map[string]any{record, pollingRecord, base, pollingBase}, []string{"pollingBackoffSeconds", "backoffSeconds", "backoff"})
	maxBackoffSeconds, _ := resolveFirstInt([]map[string]any{record, pollingRecord, base, pollingBase}, []string{"pollingMaxBackoffSeconds", "maxBackoffSeconds", "maxBackoff"})
	return TelegramPollingConfig{
		Limit:             limit,
		TimeoutSeconds:    timeoutSeconds,
		Concurrency:       concurrency,
		QueueSize:         queueSize,
		BackoffSeconds:    backoffSeconds,
		MaxBackoffSeconds: maxBackoffSeconds,
	}
}

func resolveTelegramGroups(raw any) map[string]TelegramGroupConfig {
	groups := resolveMap(raw)
	if len(groups) == 0 {
		return nil
	}
	result := make(map[string]TelegramGroupConfig, len(groups))
	for key, value := range groups {
		entry := resolveMap(value)
		if len(entry) == 0 {
			result[key] = TelegramGroupConfig{}
			continue
		}
		cfg := TelegramGroupConfig{
			AllowFrom:    resolveStringSet(entry["allowFrom"]),
			SystemPrompt: resolveString(entry["systemPrompt"]),
			Tools:        resolveStringSlice(entry["tools"]),
			Topics:       resolveTelegramTopics(entry["topics"]),
		}
		if enabled, ok := resolveOptionalBool(entry, "enabled"); ok {
			cfg.Enabled = &enabled
		} else if enabled, ok := resolveOptionalBool(entry, "enable"); ok {
			cfg.Enabled = &enabled
		}
		if requireMention, ok := resolveOptionalBool(entry, "requireMention"); ok {
			cfg.RequireMention = &requireMention
		}
		result[key] = cfg
	}
	return result
}

func resolveTelegramTopics(raw any) map[string]TelegramTopicConfig {
	topics := resolveMap(raw)
	if len(topics) == 0 {
		return nil
	}
	result := make(map[string]TelegramTopicConfig, len(topics))
	for key, value := range topics {
		entry := resolveMap(value)
		if len(entry) == 0 {
			result[key] = TelegramTopicConfig{}
			continue
		}
		cfg := TelegramTopicConfig{
			AllowFrom:    resolveStringSet(entry["allowFrom"]),
			SystemPrompt: resolveString(entry["systemPrompt"]),
			Tools:        resolveStringSlice(entry["tools"]),
		}
		if enabled, ok := resolveOptionalBool(entry, "enabled"); ok {
			cfg.Enabled = &enabled
		} else if enabled, ok := resolveOptionalBool(entry, "enable"); ok {
			cfg.Enabled = &enabled
		}
		if requireMention, ok := resolveOptionalBool(entry, "requireMention"); ok {
			cfg.RequireMention = &requireMention
		}
		result[key] = cfg
	}
	return result
}

func resolveOptionalBool(record map[string]any, key string) (bool, bool) {
	if record == nil {
		return false, false
	}
	value, ok := record[key]
	if !ok {
		return false, false
	}
	return resolveBool(value, false), true
}

func resolveStringSlice(value any) []string {
	switch typed := value.(type) {
	case []string:
		return append([]string(nil), typed...)
	case []any:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			switch entry := item.(type) {
			case string:
				trimmed := strings.TrimSpace(entry)
				if trimmed != "" {
					result = append(result, trimmed)
				}
			case float64:
				result = append(result, fmt.Sprintf("%.0f", entry))
			case int:
				result = append(result, fmt.Sprintf("%d", entry))
			case int64:
				result = append(result, fmt.Sprintf("%d", entry))
			}
		}
		return result
	default:
		return nil
	}
}

func mergeStringSet(base map[string]struct{}, overrides map[string]struct{}) map[string]struct{} {
	if len(base) == 0 && len(overrides) == 0 {
		return nil
	}
	result := make(map[string]struct{}, len(base)+len(overrides))
	for key := range base {
		result[key] = struct{}{}
	}
	for key := range overrides {
		result[key] = struct{}{}
	}
	return result
}

func mergeGroupConfigs(base map[string]TelegramGroupConfig, overrides map[string]TelegramGroupConfig) map[string]TelegramGroupConfig {
	if len(base) == 0 && len(overrides) == 0 {
		return nil
	}
	result := make(map[string]TelegramGroupConfig, len(base)+len(overrides))
	for key, value := range base {
		result[key] = value
	}
	for key, value := range overrides {
		if existing, ok := result[key]; ok {
			result[key] = mergeGroupConfig(existing, value)
			continue
		}
		result[key] = value
	}
	return result
}

func mergeGroupConfig(base TelegramGroupConfig, override TelegramGroupConfig) TelegramGroupConfig {
	result := base
	if override.Enabled != nil {
		result.Enabled = override.Enabled
	}
	if override.RequireMention != nil {
		result.RequireMention = override.RequireMention
	}
	if len(override.AllowFrom) > 0 {
		result.AllowFrom = mergeStringSet(base.AllowFrom, override.AllowFrom)
	}
	if strings.TrimSpace(override.SystemPrompt) != "" {
		result.SystemPrompt = override.SystemPrompt
	}
	if len(override.Tools) > 0 {
		result.Tools = override.Tools
	}
	if len(override.Topics) > 0 {
		result.Topics = mergeTopicConfigs(base.Topics, override.Topics)
	}
	return result
}

func mergeTopicConfigs(base map[string]TelegramTopicConfig, overrides map[string]TelegramTopicConfig) map[string]TelegramTopicConfig {
	if len(base) == 0 && len(overrides) == 0 {
		return nil
	}
	result := make(map[string]TelegramTopicConfig, len(base)+len(overrides))
	for key, value := range base {
		result[key] = value
	}
	for key, value := range overrides {
		if existing, ok := result[key]; ok {
			result[key] = mergeTopicConfig(existing, value)
			continue
		}
		result[key] = value
	}
	return result
}

func mergeTopicConfig(base TelegramTopicConfig, override TelegramTopicConfig) TelegramTopicConfig {
	result := base
	if override.Enabled != nil {
		result.Enabled = override.Enabled
	}
	if override.RequireMention != nil {
		result.RequireMention = override.RequireMention
	}
	if len(override.AllowFrom) > 0 {
		result.AllowFrom = mergeStringSet(base.AllowFrom, override.AllowFrom)
	}
	if strings.TrimSpace(override.SystemPrompt) != "" {
		result.SystemPrompt = override.SystemPrompt
	}
	if len(override.Tools) > 0 {
		result.Tools = override.Tools
	}
	return result
}

func resolveFirstString(records []map[string]any, keys []string) (string, bool) {
	for _, record := range records {
		if record == nil {
			continue
		}
		for _, key := range keys {
			if value, ok := record[key]; ok {
				return strings.TrimSpace(resolveString(value)), true
			}
		}
	}
	return "", false
}

func resolveFirstInt(records []map[string]any, keys []string) (int, bool) {
	for _, record := range records {
		if record == nil {
			continue
		}
		for _, key := range keys {
			if value, ok := record[key]; ok {
				if parsed, ok := resolveInt(value); ok {
					return parsed, true
				}
			}
		}
	}
	return 0, false
}

func resolveIntWithFallback(record map[string]any, base map[string]any, key string, fallback int) int {
	if record != nil {
		if value, ok := record[key]; ok {
			if parsed, ok := resolveInt(value); ok {
				return parsed
			}
		}
	}
	if base != nil {
		if value, ok := base[key]; ok {
			if parsed, ok := resolveInt(value); ok {
				return parsed
			}
		}
	}
	return fallback
}

func resolveInt(value any) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, true
	case int64:
		return int(typed), true
	case float64:
		return int(typed), true
	case float32:
		return int(typed), true
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed == "" {
			return 0, false
		}
		parsed, err := strconv.Atoi(trimmed)
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

func resolveStringSet(value any) map[string]struct{} {
	if value == nil {
		return nil
	}
	list, ok := value.([]any)
	if !ok {
		if stringsSlice, ok := value.([]string); ok {
			set := make(map[string]struct{}, len(stringsSlice))
			for _, entry := range stringsSlice {
				trimmed := strings.TrimSpace(entry)
				if trimmed == "" {
					continue
				}
				set[trimmed] = struct{}{}
			}
			return set
		}
		return nil
	}
	result := make(map[string]struct{}, len(list))
	for _, entry := range list {
		switch typed := entry.(type) {
		case string:
			trimmed := strings.TrimSpace(typed)
			if trimmed == "" {
				continue
			}
			result[trimmed] = struct{}{}
		case float64:
			result[fmt.Sprintf("%.0f", typed)] = struct{}{}
		case int:
			result[fmt.Sprintf("%d", typed)] = struct{}{}
		case int64:
			result[fmt.Sprintf("%d", typed)] = struct{}{}
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}
