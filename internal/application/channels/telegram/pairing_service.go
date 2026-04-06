package telegram

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	channelpairing "dreamcreator/internal/application/channels/pairing"
	settingsdto "dreamcreator/internal/application/settings/dto"
	settingsservice "dreamcreator/internal/application/settings/service"
	telegramapi "dreamcreator/internal/infrastructure/telegram"
	"go.uber.org/zap"
)

type PairingListResult struct {
	ChannelID string                   `json:"channelId"`
	Requests  []channelpairing.Request `json:"requests"`
}

type PairingApproveResult struct {
	ChannelID string `json:"channelId"`
	Approved  bool   `json:"approved"`
	RequestID string `json:"requestId,omitempty"`
	Error     string `json:"error,omitempty"`
}

type PairingRejectResult struct {
	ChannelID string `json:"channelId"`
	Rejected  bool   `json:"rejected"`
	RequestID string `json:"requestId,omitempty"`
	Error     string `json:"error,omitempty"`
}

type PairingService struct {
	store      *channelpairing.Store
	settings   *settingsservice.SettingsService
	runtime    *BotService
	httpClient *http.Client
}

func NewPairingService(settings *settingsservice.SettingsService, runtime *BotService, store *channelpairing.Store, client *http.Client) *PairingService {
	return &PairingService{
		store:      store,
		settings:   settings,
		runtime:    runtime,
		httpClient: client,
	}
}

func (service *PairingService) List(ctx context.Context, accountID string) (PairingListResult, error) {
	if service == nil || service.store == nil {
		return PairingListResult{}, fmt.Errorf("pairing service unavailable")
	}
	requests, err := service.store.List(accountID)
	if err != nil {
		return PairingListResult{}, err
	}
	return PairingListResult{ChannelID: "telegram", Requests: requests}, nil
}

func (service *PairingService) Approve(ctx context.Context, code string, accountID string, notify bool) (PairingApproveResult, error) {
	if service == nil || service.store == nil {
		return PairingApproveResult{}, fmt.Errorf("pairing service unavailable")
	}
	entry, err := service.store.Approve(code, accountID)
	if err != nil {
		if err == channelpairing.ErrPairRequestNotFound {
			return PairingApproveResult{ChannelID: "telegram", Approved: false, Error: err.Error()}, nil
		}
		return PairingApproveResult{}, err
	}
	if entry == nil {
		return PairingApproveResult{ChannelID: "telegram", Approved: false, Error: "pairing request not found"}, nil
	}
	if err := service.addAllowFrom(ctx, entry.ID, resolveAccountID(accountID, entry)); err != nil {
		return PairingApproveResult{}, err
	}
	if notify {
		if err := service.notifyApproved(ctx, entry, resolveAccountID(accountID, entry)); err != nil {
			zap.L().Warn("telegram pairing notify failed", zap.Error(err))
		}
	}
	return PairingApproveResult{ChannelID: "telegram", Approved: true, RequestID: entry.ID}, nil
}

func (service *PairingService) Reject(ctx context.Context, code string, accountID string) (PairingRejectResult, error) {
	if service == nil || service.store == nil {
		return PairingRejectResult{}, fmt.Errorf("pairing service unavailable")
	}
	entry, err := service.store.Reject(code, accountID)
	if err != nil {
		if err == channelpairing.ErrPairRequestNotFound {
			return PairingRejectResult{ChannelID: "telegram", Rejected: false, Error: err.Error()}, nil
		}
		return PairingRejectResult{}, err
	}
	if entry == nil {
		return PairingRejectResult{ChannelID: "telegram", Rejected: false, Error: "pairing request not found"}, nil
	}
	return PairingRejectResult{ChannelID: "telegram", Rejected: true, RequestID: entry.ID}, nil
}

func resolveAccountID(accountID string, entry *channelpairing.Request) string {
	trimmed := strings.TrimSpace(accountID)
	if trimmed != "" {
		return trimmed
	}
	if entry == nil || entry.Meta == nil {
		return DefaultTelegramAccountID
	}
	metaAccount := strings.TrimSpace(entry.Meta["accountId"])
	if metaAccount == "" {
		return DefaultTelegramAccountID
	}
	return metaAccount
}

func (service *PairingService) addAllowFrom(ctx context.Context, entry string, accountID string) error {
	if service == nil || service.settings == nil {
		return fmt.Errorf("settings service unavailable")
	}
	entry = strings.TrimSpace(entry)
	if entry == "" {
		return fmt.Errorf("pairing entry is empty")
	}
	current, err := service.settings.GetSettings(ctx)
	if err != nil {
		return err
	}
	channels := copyStringAnyMap(current.Channels)
	channel := resolveMap(channels["telegram"])
	if accountID != "" && accountID != DefaultTelegramAccountID {
		accounts := resolveMap(channel["accounts"])
		account := resolveMap(accounts[accountID])
		allowFrom := resolveStringList(account["allowFrom"])
		if containsString(allowFrom, entry) {
			return nil
		}
		allowFrom = append(allowFrom, entry)
		account["allowFrom"] = allowFrom
		accounts[accountID] = account
		channel["accounts"] = accounts
	} else {
		allowFrom := resolveStringList(channel["allowFrom"])
		if containsString(allowFrom, entry) {
			return nil
		}
		allowFrom = append(allowFrom, entry)
		channel["allowFrom"] = allowFrom
	}
	channels["telegram"] = channel
	updated, err := service.settings.UpdateSettings(ctx, settingsdto.UpdateSettingsRequest{Channels: channels})
	if err != nil {
		return err
	}
	if service.runtime != nil {
		_ = service.runtime.RefreshFromSettings(context.Background(), updated)
	}
	return nil
}

func (service *PairingService) notifyApproved(ctx context.Context, entry *channelpairing.Request, accountID string) error {
	if entry == nil {
		return fmt.Errorf("pairing request missing")
	}
	userID := strings.TrimSpace(entry.ID)
	if userID == "" {
		return fmt.Errorf("pairing request id missing")
	}
	chatID, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		return err
	}
	if service == nil || service.settings == nil {
		return fmt.Errorf("settings service unavailable")
	}
	settings, err := service.settings.GetSettings(ctx)
	if err != nil {
		return err
	}
	botToken := resolveBotTokenFromSettings(settings, accountID)
	if botToken == "" {
		return fmt.Errorf("telegram bot token not configured")
	}
	client := telegramapi.NewClient(botToken, service.httpClient)
	_, err = client.SendMessage(ctx, telegramapi.SendMessageParams{
		ChatID: chatID,
		Text:   channelpairing.ApprovedMessage,
	})
	return err
}

func resolveBotTokenFromSettings(settings settingsdto.Settings, accountID string) string {
	config := ResolveTelegramRuntimeConfig(settings)
	if len(config.Accounts) == 0 {
		return ""
	}
	trimmed := strings.TrimSpace(accountID)
	if trimmed == "" {
		trimmed = DefaultTelegramAccountID
	}
	for _, account := range config.Accounts {
		if strings.TrimSpace(account.AccountID) == trimmed {
			return strings.TrimSpace(account.BotToken)
		}
	}
	return ""
}

func resolveStringList(value any) []string {
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

func containsString(list []string, target string) bool {
	for _, entry := range list {
		if entry == target {
			return true
		}
	}
	return false
}

func copyStringAnyMap(input map[string]any) map[string]any {
	if len(input) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}
