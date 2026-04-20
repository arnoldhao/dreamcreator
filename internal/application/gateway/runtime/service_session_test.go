package runtime

import (
	"context"
	"strings"
	"testing"

	runtimedto "dreamcreator/internal/application/gateway/runtime/dto"
	sessionapp "dreamcreator/internal/application/session"
)

const (
	testTelegramSessionPeerID      = "test-user-001"
	testTelegramSessionThreadID    = "test-thread-001"
	testTelegramSessionPeerName    = "Test User"
	testTelegramSessionPeerNameAlt = "Test User Updated"
	testTelegramSessionUsername    = "testuser"
	testTelegramSessionAvatarURL   = "https://example.com/avatars/test-user-001.jpg"
)

func TestResolveSession_RebuildsCanonicalKeyForCustomChannelSessionKey(t *testing.T) {
	t.Parallel()

	sessionService := sessionapp.NewService(sessionapp.NewInMemoryStore())
	runtimeService := &Service{
		sessions: sessionService,
	}

	request := runtimedto.RuntimeRunRequest{
		SessionID:  "telegram:default:private:" + testTelegramSessionPeerID + ":conv:" + testTelegramSessionThreadID,
		SessionKey: "telegram:default:private:" + testTelegramSessionPeerID + ":conv:" + testTelegramSessionThreadID,
		Metadata: map[string]any{
			"channel":       "telegram",
			"accountId":     "default",
			"peerKind":      "direct",
			"peerId":        testTelegramSessionPeerID,
			"peerName":      testTelegramSessionPeerName,
			"peerUsername":  testTelegramSessionUsername,
			"peerAvatarUrl": testTelegramSessionAvatarURL,
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
	if stored.Origin.PeerID != testTelegramSessionPeerID {
		t.Fatalf("origin peerId mismatch: %q", stored.Origin.PeerID)
	}
	if stored.Origin.PeerName != testTelegramSessionPeerName {
		t.Fatalf("origin peerName mismatch: %q", stored.Origin.PeerName)
	}
	if stored.Origin.PeerUsername != testTelegramSessionUsername {
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
		SessionID:  "telegram:default:private:" + testTelegramSessionPeerID,
		SessionKey: "telegram:default:private:" + testTelegramSessionPeerID,
		Metadata: map[string]any{
			"channel":      "telegram",
			"accountId":    "default",
			"peerKind":     "direct",
			"peerId":       testTelegramSessionPeerID,
			"peerName":     testTelegramSessionPeerNameAlt,
			"peerUsername": testTelegramSessionUsername,
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
	if after.Origin.PeerName != testTelegramSessionPeerNameAlt {
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
		SessionID:  "telegram:default:private:" + testTelegramSessionPeerID,
		SessionKey: "telegram:default:private:" + testTelegramSessionPeerID,
		Metadata: map[string]any{
			"channel":       "telegram",
			"accountId":     "default",
			"peerKind":      "direct",
			"peerId":        testTelegramSessionPeerID,
			"peerName":      testTelegramSessionPeerNameAlt,
			"peerUsername":  testTelegramSessionUsername,
			"peerAvatarUrl": testTelegramSessionAvatarURL,
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
	if stored.Origin.PeerID != testTelegramSessionPeerID {
		t.Fatalf("origin peerId should be preserved: got %q", stored.Origin.PeerID)
	}
	if stored.Origin.PeerName != testTelegramSessionPeerNameAlt {
		t.Fatalf("origin peerName should be preserved: got %q", stored.Origin.PeerName)
	}
	if stored.Origin.PeerAvatarURL == "" {
		t.Fatalf("origin peerAvatarUrl should be preserved")
	}
}

func TestPersistSession_PreservesExistingContextSnapshot(t *testing.T) {
	t.Parallel()

	sessionService := sessionapp.NewService(sessionapp.NewInMemoryStore())
	runtimeService := &Service{
		sessions: sessionService,
	}

	request := runtimedto.RuntimeRunRequest{
		SessionID:  "telegram:default:private:" + testTelegramSessionPeerID,
		SessionKey: "telegram:default:private:" + testTelegramSessionPeerID,
		Metadata: map[string]any{
			"channel":   "telegram",
			"accountId": "default",
			"peerKind":  "direct",
			"peerId":    testTelegramSessionPeerID,
		},
	}
	sessionID, sessionKey, err := runtimeService.resolveSession(request)
	if err != nil {
		t.Fatalf("resolve session: %v", err)
	}
	if err := sessionService.UpdateContextSnapshot(context.Background(), sessionID, sessionapp.ContextSnapshotUpdate{
		PromptTokens: 4800,
		TotalTokens:  6200,
		WindowTokens: 131072,
		Fresh:        true,
	}); err != nil {
		t.Fatalf("update context snapshot: %v", err)
	}

	runtimeService.persistSession(context.Background(), sessionID, sessionKey, "", "assistant-ctx", map[string]any{
		"channel": "telegram",
	})

	stored, err := sessionService.Get(context.Background(), sessionID)
	if err != nil {
		t.Fatalf("load session: %v", err)
	}
	if stored.ContextPromptTokens != 4800 || stored.ContextTotalTokens != 6200 || stored.ContextWindowTokens != 131072 {
		t.Fatalf("expected context snapshot to be preserved, got %d/%d/%d", stored.ContextPromptTokens, stored.ContextTotalTokens, stored.ContextWindowTokens)
	}
	if !stored.ContextFresh {
		t.Fatal("expected context freshness to be preserved")
	}
	if stored.AssistantID != "assistant-ctx" {
		t.Fatalf("assistant id mismatch: got %q", stored.AssistantID)
	}
}
