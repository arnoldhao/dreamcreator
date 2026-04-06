package runtime

import (
	"context"
	"errors"
	"strings"
	"time"

	"dreamcreator/internal/application/gateway/runtime/dto"
	memorydto "dreamcreator/internal/application/memory/dto"
	appsession "dreamcreator/internal/application/session"
	domainassistant "dreamcreator/internal/domain/assistant"
	domainsession "dreamcreator/internal/domain/session"
)

func (service *Service) resolveSession(request dto.RuntimeRunRequest) (string, string, error) {
	sessionID := strings.TrimSpace(request.SessionID)
	rawSessionKey := strings.TrimSpace(request.SessionKey)
	sessionKey := rawSessionKey
	if rawSessionKey != "" {
		parts, normalized, err := domainsession.NormalizeSessionKey(rawSessionKey)
		if err == nil {
			sessionKey = normalized
			if sessionID == "" {
				sessionID = strings.TrimSpace(parts.ThreadRef)
			}
		} else {
			// Custom channel keys (for example telegram conversation ids) are valid as thread refs
			// but not valid session keys. Keep the thread id and rebuild a canonical session key.
			sessionKey = ""
			if sessionID == "" {
				sessionID = rawSessionKey
			}
		}
	}
	if sessionID == "" {
		sessionID = sessionKey
	}
	if sessionID == "" {
		return "", "", errors.New("session id is required")
	}
	if sessionKey == "" {
		channel := resolveMetadataString(request.Metadata, "channel")
		key, err := domainsession.BuildSessionKey(domainsession.KeyParts{
			AgentID:   strings.TrimSpace(request.AgentID),
			Channel:   channel,
			PrimaryID: sessionID,
			ThreadRef: sessionID,
		})
		if err == nil {
			sessionKey = key
		}
	}
	if sessionKey == "" {
		sessionKey = sessionID
	}
	service.persistSession(context.Background(), sessionID, sessionKey, strings.TrimSpace(request.AgentID), strings.TrimSpace(request.AssistantID), request.Metadata)
	return sessionID, sessionKey, nil
}

func (service *Service) persistSession(
	ctx context.Context,
	sessionID string,
	sessionKey string,
	agentID string,
	assistantID string,
	metadata map[string]any,
) {
	if service == nil || service.sessions == nil {
		return
	}
	sessionID = strings.TrimSpace(sessionID)
	sessionKey = strings.TrimSpace(sessionKey)
	if sessionID == "" {
		return
	}

	existing, existingErr := service.sessions.Get(ctx, sessionID)
	hasExisting := existingErr == nil
	if existingErr != nil && !errors.Is(existingErr, appsession.ErrSessionNotFound) {
		hasExisting = false
	}
	if sessionKey == "" && hasExisting {
		sessionKey = strings.TrimSpace(existing.SessionKey)
	}
	if sessionKey == "" {
		return
	}

	agentID = strings.TrimSpace(agentID)
	if agentID == "" && hasExisting {
		agentID = strings.TrimSpace(existing.AgentID)
	}
	assistantID = strings.TrimSpace(assistantID)
	if assistantID == "" && hasExisting {
		assistantID = strings.TrimSpace(existing.AssistantID)
	}
	title := ""
	if hasExisting {
		title = strings.TrimSpace(existing.Title)
	}

	originChannelIncoming := resolveMetadataString(metadata, "channel")
	originAccountIDIncoming := resolveMetadataString(metadata, "accountId")
	originPeerKindIncoming := resolveMetadataString(metadata, "peerKind")
	if originPeerKindIncoming == "" {
		originPeerKindIncoming = resolveMetadataString(metadata, "chatType")
	}
	originPeerIDIncoming := resolveMetadataString(metadata, "peerId")
	if originPeerIDIncoming == "" {
		originPeerIDIncoming = resolveMetadataString(metadata, "chatId")
	}
	originPeerNameIncoming := resolveMetadataString(metadata, "peerName")
	originPeerUsernameIncoming := resolveMetadataString(metadata, "peerUsername")
	originPeerAvatarURLIncoming := resolveMetadataString(metadata, "peerAvatarUrl")
	originPeerAvatarKeyIncoming := resolveMetadataString(metadata, "peerAvatarKey")
	originPeerAvatarSourceIncoming := resolveMetadataString(metadata, "peerAvatarSourceUrl")
	incomingAvatarRemote := strings.HasPrefix(strings.ToLower(strings.TrimSpace(originPeerAvatarURLIncoming)), "http://") ||
		strings.HasPrefix(strings.ToLower(strings.TrimSpace(originPeerAvatarURLIncoming)), "https://")
	if originPeerAvatarSourceIncoming == "" && incomingAvatarRemote {
		originPeerAvatarSourceIncoming = originPeerAvatarURLIncoming
	}

	origin := domainsession.Origin{ThreadRef: sessionID}
	if hasExisting {
		origin = existing.Origin
		if strings.TrimSpace(origin.ThreadRef) == "" {
			origin.ThreadRef = sessionID
		}
	}
	if strings.TrimSpace(origin.Channel) == "" {
		origin.Channel = originChannelIncoming
	}
	if strings.TrimSpace(origin.AccountID) == "" {
		origin.AccountID = originAccountIDIncoming
	}
	if strings.TrimSpace(origin.ChatType) == "" {
		origin.ChatType = originPeerKindIncoming
	}
	if strings.TrimSpace(origin.PeerID) == "" {
		origin.PeerID = originPeerIDIncoming
	}
	if originPeerNameIncoming != "" {
		origin.PeerName = originPeerNameIncoming
	}
	if originPeerUsernameIncoming != "" {
		origin.PeerUsername = originPeerUsernameIncoming
	}
	if originPeerAvatarURLIncoming != "" {
		hasCachedAvatarMeta := strings.TrimSpace(origin.PeerAvatarKey) != "" || strings.TrimSpace(origin.PeerAvatarSourceURL) != ""
		if !(incomingAvatarRemote && hasCachedAvatarMeta) {
			origin.PeerAvatarURL = originPeerAvatarURLIncoming
		}
	}
	if originPeerAvatarKeyIncoming != "" {
		origin.PeerAvatarKey = originPeerAvatarKeyIncoming
	}
	if originPeerAvatarSourceIncoming != "" {
		origin.PeerAvatarSourceURL = originPeerAvatarSourceIncoming
	}
	if strings.TrimSpace(origin.Channel) == "" {
		origin.Channel = originChannelIncoming
	}
	if strings.TrimSpace(origin.AccountID) == "" {
		origin.AccountID = originAccountIDIncoming
	}

	_, _ = service.sessions.CreateSession(ctx, appsession.CreateSessionRequest{
		SessionID:   sessionID,
		SessionKey:  sessionKey,
		AgentID:     agentID,
		AssistantID: assistantID,
		Title:       title,
		KeyParts: domainsession.KeyParts{
			AgentID:   agentID,
			Channel:   strings.TrimSpace(origin.Channel),
			AccountID: strings.TrimSpace(origin.AccountID),
			PrimaryID: sessionID,
			ThreadRef: sessionID,
		},
		Origin: origin,
	})
}

func resolveMemoryRecallQuery(messages []dto.Message) string {
	for i := len(messages) - 1; i >= 0; i-- {
		role := strings.ToLower(strings.TrimSpace(messages[i].Role))
		content := strings.TrimSpace(messages[i].Content)
		if content == "" {
			continue
		}
		if role == "user" {
			return content
		}
	}
	for i := len(messages) - 1; i >= 0; i-- {
		content := strings.TrimSpace(messages[i].Content)
		if content != "" {
			return content
		}
	}
	return ""
}

func buildMemoryLifecycleMessages(input []dto.Message, assistantContent string) []memorydto.MemoryMessage {
	result := make([]memorydto.MemoryMessage, 0, len(input)+1)
	for _, message := range input {
		content := strings.TrimSpace(message.Content)
		if content == "" {
			continue
		}
		result = append(result, memorydto.MemoryMessage{
			Role:    strings.TrimSpace(message.Role),
			Content: content,
		})
	}
	if content := strings.TrimSpace(assistantContent); content != "" {
		result = append(result, memorydto.MemoryMessage{
			Role:    "assistant",
			Content: content,
		})
	}
	return result
}

func (service *Service) resolveRuntimeUserLocale(
	assistantID string,
	user domainassistant.AssistantUser,
	appLanguage string,
) domainassistant.AssistantUser {
	updated := user
	needsPersist := false
	languageFallback := strings.TrimSpace(appLanguage)
	timezoneFallback := normalizeRuntimeTimezoneValue(time.Now().Location().String())

	updated.Language, needsPersist = hydrateRuntimeLocaleCurrent(updated.Language, languageFallback, needsPersist)
	updated.Timezone, needsPersist = hydrateRuntimeLocaleCurrent(updated.Timezone, timezoneFallback, needsPersist)
	updated.Location, needsPersist = markRuntimeLocaleNeedsRefresh(updated.Location, needsPersist)

	if needsPersist && service != nil && service.assistants != nil {
		id := strings.TrimSpace(assistantID)
		if id != "" {
			go func() {
				refreshCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				_, _ = service.assistants.RefreshAssistantUserLocale(refreshCtx, id)
			}()
		}
	}

	return updated
}

func hydrateRuntimeLocaleCurrent(
	locale domainassistant.UserLocale,
	fallback string,
	needsPersist bool,
) (domainassistant.UserLocale, bool) {
	mode := strings.ToLower(strings.TrimSpace(locale.Mode))
	if mode == "manual" {
		return locale, needsPersist
	}
	current := strings.TrimSpace(locale.Current)
	if !runtimeLocaleCurrentMissing(current) {
		return locale, needsPersist
	}
	if strings.TrimSpace(fallback) == "" {
		return locale, needsPersist
	}
	locale.Current = strings.TrimSpace(fallback)
	return locale, true
}

func markRuntimeLocaleNeedsRefresh(locale domainassistant.UserLocale, needsPersist bool) (domainassistant.UserLocale, bool) {
	mode := strings.ToLower(strings.TrimSpace(locale.Mode))
	if mode == "manual" {
		return locale, needsPersist
	}
	if runtimeLocaleCurrentMissing(locale.Current) {
		return locale, true
	}
	return locale, needsPersist
}

func runtimeLocaleCurrentMissing(value string) bool {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return true
	}
	return strings.EqualFold(trimmed, "unknown")
}

func normalizeRuntimeTimezoneValue(value string) string {
	trimmed := strings.TrimSpace(value)
	if strings.EqualFold(trimmed, "local") {
		return ""
	}
	return trimmed
}
