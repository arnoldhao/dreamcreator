package cron

import (
	"strings"
	"testing"

	domainsession "dreamcreator/internal/domain/session"
)

func TestResolveCronSourceChannel(t *testing.T) {
	t.Parallel()

	telegramKey, err := domainsession.BuildSessionKey(domainsession.KeyParts{
		AgentID:   "assistant-1",
		Channel:   "telegram",
		Scope:     "main",
		PrimaryID: "telegram:default:private:123",
		AccountID: "default",
		ThreadRef: "telegram:default:private:123",
	})
	if err != nil {
		t.Fatalf("build telegram key: %v", err)
	}
	auiKey, err := domainsession.BuildSessionKey(domainsession.KeyParts{
		AgentID:   "assistant-1",
		Channel:   "aui",
		Scope:     "main",
		PrimaryID: "thread-1",
		ThreadRef: "thread-1",
	})
	if err != nil {
		t.Fatalf("build aui key: %v", err)
	}

	cases := []struct {
		name       string
		sessionKey string
		want       string
	}{
		{
			name:       "telegram",
			sessionKey: telegramKey,
			want:       "telegram",
		},
		{
			name:       "aui aliases to app",
			sessionKey: auiKey,
			want:       "app",
		},
		{
			name:       "invalid key",
			sessionKey: "cron/main",
			want:       "",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := resolveCronSourceChannel(tc.sessionKey)
			if got != tc.want {
				t.Fatalf("resolveCronSourceChannel mismatch: got=%q want=%q", got, tc.want)
			}
		})
	}
}

func TestToJobResponseIncludesSourceChannel(t *testing.T) {
	t.Parallel()

	sessionKey, err := domainsession.BuildSessionKey(domainsession.KeyParts{
		AgentID:   "assistant-1",
		Channel:   "telegram",
		Scope:     "main",
		PrimaryID: "telegram:default:private:100",
		AccountID: "default",
		ThreadRef: "telegram:default:private:100",
	})
	if err != nil {
		t.Fatalf("build session key: %v", err)
	}
	response := ToJobResponse(CronJob{
		JobID:       "job-1",
		AssistantID: "assistant-1",
		Name:        "test",
		SessionKey:  sessionKey,
		Schedule: CronSchedule{
			Kind:    "every",
			EveryMs: 60_000,
		},
		SessionTarget: "main",
		WakeMode:      "now",
		PayloadSpec: CronPayload{
			Kind: "systemEvent",
			Text: "ping",
		},
		Enabled: true,
	})
	if response.SourceChannel != "telegram" {
		t.Fatalf("expected sourceChannel=telegram, got=%q", response.SourceChannel)
	}
}

func TestValidateDeliveryAnnounceChannel(t *testing.T) {
	t.Parallel()

	valid := []string{"", "default", "app", "telegram"}
	for _, channel := range valid {
		channel := channel
		t.Run("valid_"+channel, func(t *testing.T) {
			t.Parallel()
			err := validateDelivery(&DeliveryDTO{
				Mode:    "announce",
				Channel: channel,
			})
			if err != nil {
				t.Fatalf("expected channel %q to be valid, got %v", channel, err)
			}
		})
	}

	if err := validateDelivery(&DeliveryDTO{
		Mode:    "announce",
		Channel: "web",
	}); err == nil {
		t.Fatalf("expected invalid announce channel to fail")
	}
}

func TestDecodeCreateInputDoesNotRequireAssistantID(t *testing.T) {
	t.Parallel()

	params := []byte(`{
		"name":"job-a",
		"enabled":true,
		"schedule":{"kind":"every","everyMs":60000},
		"sessionTarget":"main",
		"wakeMode":"now",
		"payload":{"kind":"systemEvent","text":"ping"}
	}`)

	input, err := DecodeCreateInput(params)
	if err != nil {
		t.Fatalf("decode create input: %v", err)
	}
	if strings.TrimSpace(input.Name) != "job-a" {
		t.Fatalf("unexpected name: %q", input.Name)
	}
	if strings.TrimSpace(input.AssistantID) != "" {
		t.Fatalf("assistantId should be optional and empty by default, got %q", input.AssistantID)
	}
}

func TestDecodeCreateInputSupportsAgentTurnExtendedPayloadAndFailureDestination(t *testing.T) {
	t.Parallel()

	params := []byte(`{
		"name":"job-agentturn",
		"enabled":true,
		"schedule":{"kind":"every","everyMs":300000},
		"sessionTarget":"isolated",
		"wakeMode":"now",
		"payload":{
			"kind":"agentTurn",
			"message":"run with extended fields",
			"model":"openai/gpt-4.1",
			"thinking":"low",
			"timeoutSeconds":120,
			"lightContext":true
		},
		"delivery":{
			"mode":"webhook",
			"to":"https://example.com/webhook",
			"failureDestination":{
				"mode":"announce",
				"channel":"telegram"
			}
		}
	}`)

	input, err := DecodeCreateInput(params)
	if err != nil {
		t.Fatalf("decode create input: %v", err)
	}
	if input.Payload.Model != "openai/gpt-4.1" {
		t.Fatalf("unexpected payload.model: %q", input.Payload.Model)
	}
	if input.Delivery == nil || input.Delivery.FailureDestination == nil {
		t.Fatalf("expected failure destination to be decoded")
	}
	if input.Delivery.FailureDestination.Channel != "telegram" {
		t.Fatalf("unexpected failure destination channel: %q", input.Delivery.FailureDestination.Channel)
	}
}

func TestDecodeCreateInputRejectsLegacyPayloadPassthroughFields(t *testing.T) {
	t.Parallel()

	params := []byte(`{
		"name":"job-legacy-payload-fields",
		"enabled":true,
		"schedule":{"kind":"every","everyMs":60000},
		"sessionTarget":"isolated",
		"wakeMode":"now",
		"payload":{
			"kind":"agentTurn",
			"message":"hello",
			"allowUnsafeExternalContent":true
		}
	}`)

	_, err := DecodeCreateInput(params)
	if err == nil {
		t.Fatalf("expected legacy payload passthrough fields to be rejected")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "allowunsafeexternalcontent") {
		t.Fatalf("expected unknown-field error, got %q", err.Error())
	}
}

func TestDecodeUpdateInputRejectsLegacyPayloadPassthroughFields(t *testing.T) {
	t.Parallel()

	params := []byte(`{
		"id":"job-legacy-payload-fields",
		"patch":{
			"payload":{
				"kind":"agentTurn",
				"message":"hello",
				"allowUnsafeExternalContent":true
			}
		}
	}`)

	_, err := DecodeUpdateInput(params)
	if err == nil {
		t.Fatalf("expected legacy payload passthrough fields in update patch to be rejected")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "allowunsafeexternalcontent") {
		t.Fatalf("expected unknown-field error, got %q", err.Error())
	}
}

func TestDecodeCreateInputRejectsLegacyPayloadFallbacksField(t *testing.T) {
	t.Parallel()

	params := []byte(`{
		"name":"job-legacy-fallbacks",
		"enabled":true,
		"schedule":{"kind":"every","everyMs":60000},
		"sessionTarget":"isolated",
		"wakeMode":"now",
		"payload":{
			"kind":"agentTurn",
			"message":"hello",
			"fallbacks":["openai/gpt-4.1"]
		}
	}`)
	_, err := DecodeCreateInput(params)
	if err == nil {
		t.Fatalf("expected legacy payload fallbacks field to be rejected")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "fallbacks") {
		t.Fatalf("expected unknown-field error, got %q", err.Error())
	}
}

func TestValidateDeliveryRejectsInvalidFailureDestination(t *testing.T) {
	t.Parallel()

	err := validateDelivery(&DeliveryDTO{
		Mode: "announce",
		FailureDestination: &FailureDestinationDTO{
			Mode:    "announce",
			Channel: "web",
		},
	})
	if err == nil {
		t.Fatalf("expected invalid failure destination channel to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "failuredestination") {
		t.Fatalf("expected failure destination error, got %q", err.Error())
	}
}
