package tools

import (
	"context"
	"errors"
	"strconv"
	"strings"

	memorydto "dreamcreator/internal/application/memory/dto"
	memoryservice "dreamcreator/internal/application/memory/service"
	domainsession "dreamcreator/internal/domain/session"
)

func runMemoryRecallTool(memory *memoryservice.MemoryService) func(ctx context.Context, args string) (string, error) {
	return func(ctx context.Context, args string) (string, error) {
		if memory == nil {
			return "", errors.New("memory service unavailable")
		}
		payload, err := parseToolArgs(args)
		if err != nil {
			return "", err
		}
		query := getStringArg(payload, "query", "text", "q")
		if query == "" {
			return "", errors.New("query is required")
		}
		topK, _ := getIntArg(payload, "topK", "limit", "k")
		identity := resolveMemoryIdentityContext(ctx, payload)
		retrieval, err := memory.Recall(ctx, memorydto.MemoryRecallRequest{
			AssistantID: getStringArg(payload, "assistantId", "assistantID"),
			ThreadID:    resolveMemoryThreadID(ctx, payload),
			Channel:     identity.Channel,
			AccountID:   identity.AccountID,
			UserID:      identity.UserID,
			GroupID:     identity.GroupID,
			Query:       query,
			TopK:        topK,
			Category:    getStringArg(payload, "category"),
			Scope:       getStringArg(payload, "scope"),
		})
		if err != nil {
			return "", err
		}
		return marshalResult(map[string]any{
			"count":   len(retrieval.Entries),
			"entries": retrieval.Entries,
		}), nil
	}
}

func runMemoryStoreTool(memory *memoryservice.MemoryService) func(ctx context.Context, args string) (string, error) {
	return func(ctx context.Context, args string) (string, error) {
		if memory == nil {
			return "", errors.New("memory service unavailable")
		}
		payload, err := parseToolArgs(args)
		if err != nil {
			return "", err
		}
		content := getStringArg(payload, "text", "content", "memory", "value")
		if content == "" {
			return "", errors.New("text is required")
		}
		identity := resolveMemoryIdentityContext(ctx, payload)
		confidence := float32(0)
		if value, ok := getFloatArg(payload, "confidence", "importance", "score"); ok {
			confidence = float32(value)
		}
		entry, err := memory.Store(ctx, memorydto.MemoryStoreRequest{
			AssistantID: getStringArg(payload, "assistantId", "assistantID"),
			ThreadID:    resolveMemoryThreadID(ctx, payload),
			Channel:     identity.Channel,
			AccountID:   identity.AccountID,
			UserID:      identity.UserID,
			GroupID:     identity.GroupID,
			Content:     content,
			Category:    getStringArg(payload, "category"),
			Scope:       getStringArg(payload, "scope"),
			Confidence:  confidence,
			Metadata:    getMapArg(payload, "metadata"),
		})
		if err != nil {
			return "", err
		}
		return marshalResult(map[string]any{
			"status": "created",
			"entry":  entry,
		}), nil
	}
}

func runMemoryForgetTool(memory *memoryservice.MemoryService) func(ctx context.Context, args string) (string, error) {
	return func(ctx context.Context, args string) (string, error) {
		if memory == nil {
			return "", errors.New("memory service unavailable")
		}
		payload, err := parseToolArgs(args)
		if err != nil {
			return "", err
		}
		assistantID := getStringArg(payload, "assistantId", "assistantID")
		threadID := resolveMemoryThreadID(ctx, payload)
		scope := getStringArg(payload, "scope")
		identity := resolveMemoryIdentityContext(ctx, payload)
		memoryID := getStringArg(payload, "memoryId", "id")
		if memoryID != "" {
			deleted, err := memory.Forget(ctx, memorydto.MemoryForgetRequest{
				AssistantID: assistantID,
				ThreadID:    threadID,
				Channel:     identity.Channel,
				AccountID:   identity.AccountID,
				UserID:      identity.UserID,
				GroupID:     identity.GroupID,
				MemoryID:    memoryID,
				Scope:       scope,
			})
			if err != nil {
				return "", err
			}
			return marshalResult(map[string]any{
				"status":  ternaryString(deleted, "deleted", "not_found"),
				"deleted": deleted,
				"id":      memoryID,
			}), nil
		}
		query := getStringArg(payload, "query", "text", "q")
		if query == "" {
			return "", errors.New("memoryId or query is required")
		}
		limit, _ := getIntArg(payload, "limit", "topK")
		if limit <= 0 {
			limit = 5
		}
		retrieval, err := memory.Recall(ctx, memorydto.MemoryRecallRequest{
			AssistantID: assistantID,
			ThreadID:    threadID,
			Channel:     identity.Channel,
			AccountID:   identity.AccountID,
			UserID:      identity.UserID,
			GroupID:     identity.GroupID,
			Query:       query,
			TopK:        limit,
			Category:    getStringArg(payload, "category"),
			Scope:       scope,
		})
		if err != nil {
			return "", err
		}
		if len(retrieval.Entries) == 0 {
			return marshalResult(map[string]any{
				"status": "not_found",
				"query":  query,
			}), nil
		}
		target := retrieval.Entries[0]
		if len(retrieval.Entries) > 1 && target.Score < 0.85 {
			return marshalResult(map[string]any{
				"status":     "candidates",
				"query":      query,
				"candidates": retrieval.Entries,
			}), nil
		}
		deleted, err := memory.Forget(ctx, memorydto.MemoryForgetRequest{
			AssistantID: assistantID,
			ThreadID:    threadID,
			Channel:     identity.Channel,
			AccountID:   identity.AccountID,
			UserID:      identity.UserID,
			GroupID:     identity.GroupID,
			MemoryID:    target.ID,
			Scope:       scope,
		})
		if err != nil {
			return "", err
		}
		return marshalResult(map[string]any{
			"status":  ternaryString(deleted, "deleted", "not_found"),
			"deleted": deleted,
			"id":      target.ID,
			"query":   query,
		}), nil
	}
}

func runMemoryUpdateTool(memory *memoryservice.MemoryService) func(ctx context.Context, args string) (string, error) {
	return func(ctx context.Context, args string) (string, error) {
		if memory == nil {
			return "", errors.New("memory service unavailable")
		}
		payload, err := parseToolArgs(args)
		if err != nil {
			return "", err
		}
		memoryID := getStringArg(payload, "memoryId", "id")
		if memoryID == "" {
			return "", errors.New("memoryId is required")
		}
		identity := resolveMemoryIdentityContext(ctx, payload)
		var contentPtr *string
		if content := getStringArg(payload, "text", "content"); content != "" {
			contentPtr = &content
		}
		var categoryPtr *string
		if category := getStringArg(payload, "category"); category != "" {
			categoryPtr = &category
		}
		var scopePtr *string
		if scope := getStringArg(payload, "scope"); scope != "" {
			scopePtr = &scope
		}
		var confidencePtr *float32
		if confidenceValue, ok := getFloatArg(payload, "confidence", "importance", "score"); ok {
			parsed := float32(confidenceValue)
			confidencePtr = &parsed
		}
		entry, err := memory.Update(ctx, memorydto.MemoryUpdateRequest{
			AssistantID: getStringArg(payload, "assistantId", "assistantID"),
			ThreadID:    resolveMemoryThreadID(ctx, payload),
			Channel:     identity.Channel,
			AccountID:   identity.AccountID,
			UserID:      identity.UserID,
			GroupID:     identity.GroupID,
			MemoryID:    memoryID,
			Content:     contentPtr,
			Category:    categoryPtr,
			Scope:       scopePtr,
			Confidence:  confidencePtr,
			Metadata:    getMapArg(payload, "metadata"),
		})
		if err != nil {
			return "", err
		}
		return marshalResult(map[string]any{
			"status": "updated",
			"entry":  entry,
		}), nil
	}
}

func runMemoryStatsTool(memory *memoryservice.MemoryService) func(ctx context.Context, args string) (string, error) {
	return func(ctx context.Context, args string) (string, error) {
		if memory == nil {
			return "", errors.New("memory service unavailable")
		}
		payload, err := parseToolArgs(args)
		if err != nil {
			return "", err
		}
		identity := resolveMemoryIdentityContext(ctx, payload)
		stats, err := memory.Stats(ctx, memorydto.MemoryStatsRequest{
			AssistantID: getStringArg(payload, "assistantId", "assistantID"),
			ThreadID:    resolveMemoryThreadID(ctx, payload),
			Channel:     identity.Channel,
			AccountID:   identity.AccountID,
			UserID:      identity.UserID,
			GroupID:     identity.GroupID,
			Scope:       getStringArg(payload, "scope"),
		})
		if err != nil {
			return "", err
		}
		return marshalResult(stats), nil
	}
}

func runMemoryListTool(memory *memoryservice.MemoryService) func(ctx context.Context, args string) (string, error) {
	return func(ctx context.Context, args string) (string, error) {
		if memory == nil {
			return "", errors.New("memory service unavailable")
		}
		payload, err := parseToolArgs(args)
		if err != nil {
			return "", err
		}
		identity := resolveMemoryIdentityContext(ctx, payload)
		limit, _ := getIntArg(payload, "limit")
		offset, _ := getIntArg(payload, "offset")
		entries, err := memory.List(ctx, memorydto.MemoryListRequest{
			AssistantID: getStringArg(payload, "assistantId", "assistantID"),
			ThreadID:    resolveMemoryThreadID(ctx, payload),
			Channel:     identity.Channel,
			AccountID:   identity.AccountID,
			UserID:      identity.UserID,
			GroupID:     identity.GroupID,
			Category:    getStringArg(payload, "category"),
			Scope:       getStringArg(payload, "scope"),
			Limit:       limit,
			Offset:      offset,
		})
		if err != nil {
			return "", err
		}
		return marshalResult(map[string]any{
			"count":   len(entries),
			"entries": entries,
		}), nil
	}
}

func resolveMemoryThreadID(ctx context.Context, payload toolArgs) string {
	if threadID := getStringArg(payload, "threadId", "threadID", "sessionId", "sessionID"); threadID != "" {
		return threadID
	}
	sessionKey, _ := RuntimeContextFromContext(ctx)
	sessionKey = strings.TrimSpace(sessionKey)
	if sessionKey == "" {
		return ""
	}
	if parts, _, err := domainsession.NormalizeSessionKey(sessionKey); err == nil {
		if threadRef := strings.TrimSpace(parts.ThreadRef); threadRef != "" {
			return threadRef
		}
		if primary := strings.TrimSpace(parts.PrimaryID); primary != "" {
			return primary
		}
	}
	return sessionKey
}

type memoryIdentityContext struct {
	Channel   string
	AccountID string
	UserID    string
	GroupID   string
}

func resolveMemoryIdentityContext(ctx context.Context, payload toolArgs) memoryIdentityContext {
	result := memoryIdentityContext{
		Channel:   strings.TrimSpace(getStringArg(payload, "channel")),
		AccountID: strings.TrimSpace(getStringArg(payload, "accountId")),
		UserID:    strings.TrimSpace(getStringArg(payload, "userId")),
		GroupID:   strings.TrimSpace(getStringArg(payload, "groupId")),
	}
	peerKind := strings.ToLower(strings.TrimSpace(getStringArg(payload, "peerKind")))
	peerID := strings.TrimSpace(getStringArg(payload, "peerId"))
	switch peerKind {
	case "group", "room", "supergroup", "channel":
		if result.GroupID == "" {
			result.GroupID = peerID
		}
	case "direct", "private", "user", "dm":
		if result.UserID == "" {
			result.UserID = peerID
		}
	}
	sessionKey, _ := RuntimeContextFromContext(ctx)
	if parts, _, err := domainsession.NormalizeSessionKey(strings.TrimSpace(sessionKey)); err == nil {
		if result.Channel == "" {
			result.Channel = strings.TrimSpace(parts.Channel)
		}
		if result.AccountID == "" {
			result.AccountID = strings.TrimSpace(parts.AccountID)
		}
	}
	return result
}

func getFloatArg(args toolArgs, keys ...string) (float64, bool) {
	for _, key := range keys {
		if key == "" {
			continue
		}
		raw, ok := args[key]
		if !ok {
			continue
		}
		switch typed := raw.(type) {
		case float64:
			return typed, true
		case float32:
			return float64(typed), true
		case int:
			return float64(typed), true
		case int64:
			return float64(typed), true
		case string:
			trimmed := strings.TrimSpace(typed)
			if trimmed == "" {
				continue
			}
			parsed, err := strconv.ParseFloat(trimmed, 64)
			if err == nil {
				return parsed, true
			}
		}
	}
	return 0, false
}

func ternaryString(condition bool, whenTrue string, whenFalse string) string {
	if condition {
		return whenTrue
	}
	return whenFalse
}
