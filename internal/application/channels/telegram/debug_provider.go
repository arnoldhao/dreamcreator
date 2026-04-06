package telegram

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	gatewaychannels "dreamcreator/internal/application/gateway/channels"
)

type DebugProvider struct {
	service *BotService
}

func NewDebugProvider(service *BotService) *DebugProvider {
	return &DebugProvider{service: service}
}

func (provider *DebugProvider) ChannelID() string {
	return "telegram"
}

func (provider *DebugProvider) Debug(_ context.Context) ([]gatewaychannels.ChannelDebugAccount, []string, error) {
	if provider == nil || provider.service == nil {
		return nil, nil, fmt.Errorf("telegram debug provider unavailable")
	}
	service := provider.service
	service.mu.Lock()
	defer service.mu.Unlock()

	accounts := make([]gatewaychannels.ChannelDebugAccount, 0, len(service.accounts))
	for _, state := range service.accounts {
		if state == nil {
			continue
		}
		enabled := state.config.Enabled
		configured := strings.TrimSpace(state.config.BotToken) != ""
		mode := strings.TrimSpace(state.mode)
		if !state.running {
			switch {
			case !enabled:
				mode = "disabled"
			case !configured:
				mode = "not_configured"
			case mode == "":
				mode = "stopped"
			}
		}
		account := gatewaychannels.ChannelDebugAccount{
			AccountID:             state.config.AccountID,
			Enabled:               enabled,
			Configured:            configured,
			Running:               state.running,
			Mode:                  mode,
			BotUsername:           state.botUsername,
			BotID:                 state.botID,
			WebhookURL:            strings.TrimSpace(state.config.WebhookURL),
			WebhookSecretSet:      strings.TrimSpace(state.config.WebhookSecret) != "",
			DmPolicy:              string(state.config.DMPolicy),
			GroupPolicy:           string(state.config.GroupPolicy),
			AllowFromCount:        len(state.config.AllowFrom),
			GroupAllowFromCount:   len(state.config.GroupAllowFrom),
			GroupsCount:           len(state.config.Groups),
			LastInboundAt:         timePtr(state.lastInbound),
			LastInboundType:       state.lastInboundType,
			LastInboundUpdateID:   state.lastInboundUpdateID,
			LastInboundMessageID:  state.lastInboundMessageID,
			LastInboundChatID:     formatChatID(state.lastInboundChatID),
			LastInboundUserID:     state.lastInboundUserID,
			LastInboundCommand:    state.lastInboundCommand,
			LastDeniedReason:      state.lastDeniedReason,
			LastDeniedAt:          timePtr(state.lastDeniedAt),
			LastRunAt:             timePtr(state.lastRunAt),
			LastRunID:             state.lastRunID,
			LastRunError:          state.lastRunError,
			LastOutboundAt:        timePtr(state.lastOutbound),
			LastOutboundMessageID: state.lastOutboundMessageID,
			LastOutboundError:     state.lastOutboundError,
			InboundCount:          state.inboundCount,
			OutboundCount:         state.outboundCount,
			DeniedCount:           state.deniedCount,
			ErrorCount:            state.errorCount,
		}
		account.Notes = telegramAccountNotes(state, enabled, configured)
		accounts = append(accounts, account)
	}
	sort.Slice(accounts, func(i, j int) bool {
		left := strings.ToLower(strings.TrimSpace(accounts[i].AccountID))
		right := strings.ToLower(strings.TrimSpace(accounts[j].AccountID))
		if left == right {
			return accounts[i].AccountID < accounts[j].AccountID
		}
		return left < right
	})
	return accounts, nil, nil
}

func telegramAccountNotes(state *telegramAccountState, enabled, configured bool) []string {
	if state == nil {
		return nil
	}
	notes := []string{}
	if enabled && !configured {
		notes = append(notes, "missing_bot_token")
	}
	if enabled && configured && !state.running {
		notes = append(notes, "not_running")
	}
	if state.config.DMPolicy == DMPolicyPairing && len(state.config.AllowFrom) == 0 {
		notes = append(notes, "pairing_required")
	}
	if state.config.GroupPolicy == GroupPolicyAllowlist && len(state.config.Groups) == 0 {
		notes = append(notes, "group_allowlist_empty")
	}
	return notes
}

func timePtr(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}
	return &value
}

func formatChatID(value int64) string {
	if value == 0 {
		return ""
	}
	return fmt.Sprintf("%d", value)
}
