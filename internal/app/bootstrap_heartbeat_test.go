package app

import (
	"context"
	"testing"

	settingsdto "dreamcreator/internal/application/settings/dto"
	settingsservice "dreamcreator/internal/application/settings/service"
	"dreamcreator/internal/domain/settings"
)

type heartbeatSettingsRepoStub struct {
	current settings.Settings
	exists  bool
}

func (stub *heartbeatSettingsRepoStub) Get(_ context.Context) (settings.Settings, error) {
	if !stub.exists {
		return settings.Settings{}, settings.ErrSettingsNotFound
	}
	return stub.current, nil
}

func (stub *heartbeatSettingsRepoStub) Save(_ context.Context, current settings.Settings) error {
	stub.current = current
	stub.exists = true
	return nil
}

type heartbeatThemeProviderStub struct{}

func (heartbeatThemeProviderStub) IsDarkMode(context.Context) (bool, error) {
	return false, nil
}

func (heartbeatThemeProviderStub) AccentColor(context.Context) (string, error) {
	return "", nil
}

func TestResolveHeartbeatAccountReadiness(t *testing.T) {
	t.Parallel()

	repo := &heartbeatSettingsRepoStub{}
	settingsService := settingsservice.NewSettingsService(repo, heartbeatThemeProviderStub{}, settings.DefaultSettings())

	_, err := settingsService.UpdateSettings(context.Background(), settingsdto.UpdateSettingsRequest{
		Channels: map[string]any{
			"telegram": map[string]any{
				"accounts": map[string]any{
					"acct-ready": map[string]any{
						"enabled":  true,
						"botToken": "123:ready",
					},
					"acct-disabled": map[string]any{
						"enabled":  false,
						"botToken": "123:disabled",
					},
					"acct-token-missing": map[string]any{
						"enabled": true,
					},
				},
			},
			"slack": map[string]any{
				"accounts": map[string]any{
					"acct-slack-ready": map[string]any{
						"enabled": true,
					},
					"acct-slack-disabled": map[string]any{
						"enabled": false,
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("update settings failed: %v", err)
	}

	cases := []struct {
		name       string
		channelID  string
		accountID  string
		wantReady  bool
		wantReason string
	}{
		{
			name:       "ready account",
			channelID:  "telegram",
			accountID:  "acct-ready",
			wantReady:  true,
			wantReason: "",
		},
		{
			name:       "unknown account",
			channelID:  "telegram",
			accountID:  "acct-unknown",
			wantReady:  false,
			wantReason: "unknown_account",
		},
		{
			name:       "disabled account",
			channelID:  "telegram",
			accountID:  "acct-disabled",
			wantReady:  false,
			wantReason: "account_disabled",
		},
		{
			name:       "missing token",
			channelID:  "telegram",
			accountID:  "acct-token-missing",
			wantReady:  false,
			wantReason: "account_token_missing",
		},
		{
			name:       "slack account ready",
			channelID:  "slack",
			accountID:  "acct-slack-ready",
			wantReady:  true,
			wantReason: "",
		},
		{
			name:       "slack unknown account",
			channelID:  "slack",
			accountID:  "acct-slack-unknown",
			wantReady:  false,
			wantReason: "unknown_account",
		},
		{
			name:       "slack disabled account",
			channelID:  "slack",
			accountID:  "acct-slack-disabled",
			wantReady:  false,
			wantReason: "account_disabled",
		},
		{
			name:       "channel without accounts bypass",
			channelID:  "web",
			accountID:  "acct-ready",
			wantReady:  true,
			wantReason: "",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ready, reason := resolveHeartbeatAccountReadiness(context.Background(), settingsService, tc.channelID, tc.accountID)
			if ready != tc.wantReady {
				t.Fatalf("ready mismatch: got=%v want=%v (reason=%q)", ready, tc.wantReady, reason)
			}
			if reason != tc.wantReason {
				t.Fatalf("reason mismatch: got=%q want=%q", reason, tc.wantReason)
			}
		})
	}
}
