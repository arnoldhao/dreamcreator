package app

import (
	"context"
	"testing"

	gatewaycron "dreamcreator/internal/application/gateway/cron"
	domainsession "dreamcreator/internal/domain/session"
)

type stubCronAnnouncementSessionStore struct {
	entries []domainsession.Entry
}

func (store stubCronAnnouncementSessionStore) List(_ context.Context) ([]domainsession.Entry, error) {
	return append([]domainsession.Entry(nil), store.entries...), nil
}

func TestResolveCronSessionChannel(t *testing.T) {
	t.Parallel()

	telegramKey, err := domainsession.BuildSessionKey(domainsession.KeyParts{
		AgentID:   "agent-1",
		Channel:   "telegram",
		Scope:     "main",
		PrimaryID: "telegram:default:private:123",
		AccountID: "default",
		ThreadRef: "telegram:default:private:123",
	})
	if err != nil {
		t.Fatalf("build telegram key: %v", err)
	}
	appKey, err := domainsession.BuildSessionKey(domainsession.KeyParts{
		AgentID:   "agent-1",
		Channel:   "app",
		Scope:     "main",
		PrimaryID: "thread-1",
		AccountID: "",
		ThreadRef: "thread-1",
	})
	if err != nil {
		t.Fatalf("build app key: %v", err)
	}

	cases := []struct {
		name          string
		sessionKey    string
		originChannel string
		want          string
	}{
		{
			name:          "channel from session key",
			sessionKey:    telegramKey,
			originChannel: "aui",
			want:          "telegram",
		},
		{
			name:          "origin channel alias app to aui",
			sessionKey:    "cron/main",
			originChannel: "app",
			want:          "app",
		},
		{
			name:          "session channel alias app to aui",
			sessionKey:    appKey,
			originChannel: "telegram",
			want:          "app",
		},
		{
			name:          "empty channel",
			sessionKey:    "",
			originChannel: "",
			want:          "",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := resolveCronSessionChannel(tc.sessionKey, tc.originChannel)
			if got != tc.want {
				t.Fatalf("resolve channel mismatch: got=%q want=%q", got, tc.want)
			}
		})
	}
}

func TestResolveCronAnnouncementDeliveryChannel(t *testing.T) {
	t.Parallel()

	telegramKey, err := domainsession.BuildSessionKey(domainsession.KeyParts{
		AgentID:   "agent-1",
		Channel:   "telegram",
		Scope:     "main",
		PrimaryID: "telegram:default:private:123",
		AccountID: "default",
		ThreadRef: "telegram:default:private:123",
	})
	if err != nil {
		t.Fatalf("build telegram key: %v", err)
	}
	entry := domainsession.Entry{
		SessionKey: telegramKey,
		Origin: domainsession.Origin{
			Channel: "telegram",
		},
	}

	cases := []struct {
		name           string
		requestChannel string
		want           string
	}{
		{name: "empty uses session channel", requestChannel: "", want: "telegram"},
		{name: "default uses session channel", requestChannel: "default", want: "telegram"},
		{name: "explicit app", requestChannel: "app", want: "app"},
		{name: "explicit telegram", requestChannel: "telegram", want: "telegram"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := resolveCronAnnouncementDeliveryChannel(tc.requestChannel, entry)
			if got != tc.want {
				t.Fatalf("resolveCronAnnouncementDeliveryChannel mismatch: got=%q want=%q", got, tc.want)
			}
		})
	}
}

func TestResolveCronAnnouncementSessionDoesNotFallbackWhenSpecificSessionMissing(t *testing.T) {
	t.Parallel()

	appKey, err := domainsession.BuildSessionKey(domainsession.KeyParts{
		AgentID:   "agent-1",
		Channel:   "app",
		Scope:     "main",
		PrimaryID: "thread-1",
		ThreadRef: "thread-1",
	})
	if err != nil {
		t.Fatalf("build app key: %v", err)
	}
	store := stubCronAnnouncementSessionStore{
		entries: []domainsession.Entry{
			{
				SessionKey:  appKey,
				AssistantID: "assistant-1",
				Origin:      domainsession.Origin{Channel: "app"},
			},
		},
	}
	request := gatewaycron.AnnouncementRequest{
		AssistantID: "assistant-1",
		SessionKey:  "v2::-::telegram::-::telegram:default:private:999::-::telegram:default:private:999",
	}
	_, ok := resolveCronAnnouncementSession(context.Background(), store, request)
	if ok {
		t.Fatalf("expected no fallback for missing concrete session key")
	}
}

func TestResolveCronAnnouncementSessionAllowsFallbackForSyntheticKey(t *testing.T) {
	t.Parallel()

	telegramKey, err := domainsession.BuildSessionKey(domainsession.KeyParts{
		AgentID:   "agent-1",
		Channel:   "telegram",
		Scope:     "main",
		PrimaryID: "telegram:default:private:123",
		AccountID: "default",
		ThreadRef: "telegram:default:private:123",
	})
	if err != nil {
		t.Fatalf("build telegram key: %v", err)
	}
	store := stubCronAnnouncementSessionStore{
		entries: []domainsession.Entry{
			{
				SessionKey:  telegramKey,
				AssistantID: "assistant-1",
				Origin:      domainsession.Origin{Channel: "telegram"},
			},
		},
	}
	request := gatewaycron.AnnouncementRequest{
		AssistantID: "assistant-1",
		SessionKey:  "cron/main",
	}
	entry, ok := resolveCronAnnouncementSession(context.Background(), store, request)
	if !ok {
		t.Fatalf("expected fallback for synthetic session key")
	}
	if entry.SessionKey != telegramKey {
		t.Fatalf("expected fallback entry session key %q, got %q", telegramKey, entry.SessionKey)
	}
}
