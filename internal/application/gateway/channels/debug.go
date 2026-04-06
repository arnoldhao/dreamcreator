package channels

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
)

type ChannelDebugAccount struct {
	AccountID             string     `json:"accountId"`
	Enabled               bool       `json:"enabled"`
	Configured            bool       `json:"configured"`
	Running               bool       `json:"running"`
	Mode                  string     `json:"mode,omitempty"`
	BotUsername           string     `json:"botUsername,omitempty"`
	BotID                 int64      `json:"botId,omitempty"`
	WebhookURL            string     `json:"webhookUrl,omitempty"`
	WebhookSecretSet      bool       `json:"webhookSecretSet"`
	DmPolicy              string     `json:"dmPolicy,omitempty"`
	GroupPolicy           string     `json:"groupPolicy,omitempty"`
	AllowFromCount        int        `json:"allowFromCount"`
	GroupAllowFromCount   int        `json:"groupAllowFromCount"`
	GroupsCount           int        `json:"groupsCount"`
	LastInboundAt         *time.Time `json:"lastInboundAt,omitempty"`
	LastInboundType       string     `json:"lastInboundType,omitempty"`
	LastInboundUpdateID   int64      `json:"lastInboundUpdateId,omitempty"`
	LastInboundMessageID  int64      `json:"lastInboundMessageId,omitempty"`
	LastInboundChatID     string     `json:"lastInboundChatId,omitempty"`
	LastInboundUserID     string     `json:"lastInboundUserId,omitempty"`
	LastInboundCommand    string     `json:"lastInboundCommand,omitempty"`
	LastDeniedReason      string     `json:"lastDeniedReason,omitempty"`
	LastDeniedAt          *time.Time `json:"lastDeniedAt,omitempty"`
	LastRunAt             *time.Time `json:"lastRunAt,omitempty"`
	LastRunID             string     `json:"lastRunId,omitempty"`
	LastRunError          string     `json:"lastRunError,omitempty"`
	LastOutboundAt        *time.Time `json:"lastOutboundAt,omitempty"`
	LastOutboundMessageID int64      `json:"lastOutboundMessageId,omitempty"`
	LastOutboundError     string     `json:"lastOutboundError,omitempty"`
	InboundCount          int        `json:"inboundCount"`
	OutboundCount         int        `json:"outboundCount"`
	DeniedCount           int        `json:"deniedCount"`
	ErrorCount            int        `json:"errorCount"`
	Notes                 []string   `json:"notes,omitempty"`
}

type ChannelDebugSnapshot struct {
	ChannelID   string                `json:"channelId"`
	DisplayName string                `json:"displayName"`
	Kind        string                `json:"kind"`
	Enabled     bool                  `json:"enabled"`
	State       string                `json:"state"`
	UpdatedAt   time.Time             `json:"updatedAt"`
	LastError   string                `json:"lastError,omitempty"`
	Accounts    []ChannelDebugAccount `json:"accounts,omitempty"`
	Notes       []string              `json:"notes,omitempty"`
}

type ChannelDebugProvider interface {
	ChannelID() string
	Debug(ctx context.Context) ([]ChannelDebugAccount, []string, error)
}

type DebugService struct {
	registry  *Registry
	providers map[string]ChannelDebugProvider
	now       func() time.Time
}

func NewDebugService(registry *Registry) *DebugService {
	return &DebugService{
		registry:  registry,
		providers: make(map[string]ChannelDebugProvider),
		now:       time.Now,
	}
}

func (service *DebugService) RegisterProvider(provider ChannelDebugProvider) {
	if service == nil || provider == nil {
		return
	}
	channelID := strings.TrimSpace(provider.ChannelID())
	if channelID == "" {
		return
	}
	service.providers[channelID] = provider
}

func (service *DebugService) List(ctx context.Context) ([]ChannelDebugSnapshot, error) {
	if service == nil || service.registry == nil {
		return nil, fmt.Errorf("channels debug service unavailable")
	}
	entries, err := service.registry.List(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]ChannelDebugSnapshot, 0, len(entries))
	for _, entry := range entries {
		snapshot := ChannelDebugSnapshot{
			ChannelID:   entry.ChannelID,
			DisplayName: entry.DisplayName,
			Kind:        entry.Kind,
			Enabled:     entry.Enabled,
			State:       entry.State,
			UpdatedAt:   entry.UpdatedAt,
			LastError:   entry.LastError,
		}
		if provider, ok := service.providers[entry.ChannelID]; ok {
			accounts, notes, err := provider.Debug(ctx)
			if err != nil {
				snapshot.Notes = append(snapshot.Notes, fmt.Sprintf("debug error: %s", err.Error()))
			}
			if len(notes) > 0 {
				snapshot.Notes = append(snapshot.Notes, notes...)
			}
			snapshot.Accounts = accounts
		} else {
			snapshot.Notes = append(snapshot.Notes, "debug_not_supported")
		}
		result = append(result, snapshot)
	}
	sort.Slice(result, func(i, j int) bool {
		left := strings.ToLower(strings.TrimSpace(result[i].DisplayName))
		right := strings.ToLower(strings.TrimSpace(result[j].DisplayName))
		if left == "" {
			left = strings.ToLower(result[i].ChannelID)
		}
		if right == "" {
			right = strings.ToLower(result[j].ChannelID)
		}
		if left == right {
			return result[i].ChannelID < result[j].ChannelID
		}
		return left < right
	})
	return result, nil
}
