package runtime

import (
	"context"
	"strings"
	"testing"

	runtimedto "dreamcreator/internal/application/gateway/runtime/dto"
	sessionapp "dreamcreator/internal/application/session"
)

func TestResolveSession_RebuildsCanonicalKeyForCustomChannelSessionKey(t *testing.T) {
	t.Parallel()

	sessionService := sessionapp.NewService(sessionapp.NewInMemoryStore())
	runtimeService := &Service{
		sessions: sessionService,
	}

	request := runtimedto.RuntimeRunRequest{
		SessionID:  "telegram:default:private:5234834060:conv:26291424",
		SessionKey: "telegram:default:private:5234834060:conv:26291424",
		Metadata: map[string]any{
			"channel":       "telegram",
			"accountId":     "default",
			"peerKind":      "direct",
			"peerId":        "5234834060",
			"peerName":      "Arnold",
			"peerUsername":  "arnold",
			"peerAvatarUrl": "https://t.me/i/userpic/320/arnold.jpg",
		},
	}

	sessionID, sessionKey, err := runtimeService.resolveSession(request)
	if err != nil {
		t.Fatalf("resolve session: %v", err)
	}
	if sessionID != request.SessionID {
		t.Fatalf("unexpected session id: got %q want %q", sessionID, request.SessionID)
	}
	if !strings.HasPrefix(sessionKey, "v2::") {
		t.Fatalf("expected canonical session key, got %q", sessionKey)
	}

	stored, err := sessionService.Get(context.Background(), sessionID)
	if err != nil {
		t.Fatalf("load stored session: %v", err)
	}
	if stored.SessionKey != sessionKey {
		t.Fatalf("stored session key mismatch: got %q want %q", stored.SessionKey, sessionKey)
	}
	if stored.Origin.Channel != "telegram" {
		t.Fatalf("origin channel mismatch: %q", stored.Origin.Channel)
	}
	if stored.Origin.AccountID != "default" {
		t.Fatalf("origin accountId mismatch: %q", stored.Origin.AccountID)
	}
	if stored.Origin.PeerID != "5234834060" {
		t.Fatalf("origin peerId mismatch: %q", stored.Origin.PeerID)
	}
	if stored.Origin.PeerName != "Arnold" {
		t.Fatalf("origin peerName mismatch: %q", stored.Origin.PeerName)
	}
	if stored.Origin.PeerUsername != "arnold" {
		t.Fatalf("origin peerUsername mismatch: %q", stored.Origin.PeerUsername)
	}
	if stored.Origin.PeerAvatarURL == "" {
		t.Fatalf("origin peerAvatarUrl should not be empty")
	}
}

func TestPersistSession_UpdatesAssistantIDForExistingSession(t *testing.T) {
	t.Parallel()

	sessionService := sessionapp.NewService(sessionapp.NewInMemoryStore())
	runtimeService := &Service{
		sessions: sessionService,
	}

	request := runtimedto.RuntimeRunRequest{
		SessionID:  "telegram:default:private:5234834060",
		SessionKey: "telegram:default:private:5234834060",
		Metadata: map[string]any{
			"channel":      "telegram",
			"accountId":    "default",
			"peerKind":     "direct",
			"peerId":       "5234834060",
			"peerName":     "Arnold HAO",
			"peerUsername": "arnold",
		},
	}
	sessionID, sessionKey, err := runtimeService.resolveSession(request)
	if err != nil {
		t.Fatalf("resolve session: %v", err)
	}

	before, err := sessionService.Get(context.Background(), sessionID)
	if err != nil {
		t.Fatalf("load session before assistant update: %v", err)
	}
	if strings.TrimSpace(before.AssistantID) != "" {
		t.Fatalf("expected empty assistant id before update, got %q", before.AssistantID)
	}

	runtimeService.persistSession(context.Background(), sessionID, sessionKey, "", "assistant-123", request.Metadata)
	after, err := sessionService.Get(context.Background(), sessionID)
	if err != nil {
		t.Fatalf("load session after assistant update: %v", err)
	}
	if after.AssistantID != "assistant-123" {
		t.Fatalf("assistant id was not updated: got %q", after.AssistantID)
	}
	if after.Origin.PeerName != "Arnold HAO" {
		t.Fatalf("origin peer name mismatch: got %q", after.Origin.PeerName)
	}
}

func TestPersistSession_PreservesExistingOriginWhenIncomingMetadataIsIncomplete(t *testing.T) {
	t.Parallel()

	sessionService := sessionapp.NewService(sessionapp.NewInMemoryStore())
	runtimeService := &Service{
		sessions: sessionService,
	}

	request := runtimedto.RuntimeRunRequest{
		SessionID:  "telegram:default:private:5234834060",
		SessionKey: "telegram:default:private:5234834060",
		Metadata: map[string]any{
			"channel":       "telegram",
			"accountId":     "default",
			"peerKind":      "direct",
			"peerId":        "5234834060",
			"peerName":      "Arnold HAO",
			"peerUsername":  "arnold",
			"peerAvatarUrl": "https://t.me/i/userpic/320/arnold.jpg",
		},
	}
	sessionID, sessionKey, err := runtimeService.resolveSession(request)
	if err != nil {
		t.Fatalf("resolve session: %v", err)
	}
	runtimeService.persistSession(context.Background(), sessionID, sessionKey, "", "assistant-123", map[string]any{
		"channel": "aui",
	})

	stored, err := sessionService.Get(context.Background(), sessionID)
	if err != nil {
		t.Fatalf("load session: %v", err)
	}
	if stored.AssistantID != "assistant-123" {
		t.Fatalf("assistant id mismatch: got %q", stored.AssistantID)
	}
	if stored.Origin.Channel != "telegram" {
		t.Fatalf("origin channel should be preserved: got %q", stored.Origin.Channel)
	}
	if stored.Origin.AccountID != "default" {
		t.Fatalf("origin accountId should be preserved: got %q", stored.Origin.AccountID)
	}
	if stored.Origin.PeerID != "5234834060" {
		t.Fatalf("origin peerId should be preserved: got %q", stored.Origin.PeerID)
	}
	if stored.Origin.PeerName != "Arnold HAO" {
		t.Fatalf("origin peerName should be preserved: got %q", stored.Origin.PeerName)
	}
	if stored.Origin.PeerAvatarURL == "" {
		t.Fatalf("origin peerAvatarUrl should be preserved")
	}
}
