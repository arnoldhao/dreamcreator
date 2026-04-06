package tools

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	agentservice "dreamcreator/internal/application/agent/service"
	gatewaysubagent "dreamcreator/internal/application/gateway/subagent"
	appsession "dreamcreator/internal/application/session"
	subagentservice "dreamcreator/internal/application/subagent/service"
	"dreamcreator/internal/application/thread/dto"
	threadservice "dreamcreator/internal/application/thread/service"
	"dreamcreator/internal/infrastructure/llm"
)

type sessionListItem struct {
	SessionID  string    `json:"sessionId"`
	SessionKey string    `json:"sessionKey"`
	Title      string    `json:"title,omitempty"`
	Status     string    `json:"status,omitempty"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

func runAgentsListTool(agents *agentservice.AgentService) func(ctx context.Context, args string) (string, error) {
	return func(ctx context.Context, _ string) (string, error) {
		if agents == nil {
			return "", errors.New("agent service unavailable")
		}
		items, err := agents.ListAgents(ctx, false)
		if err != nil {
			return "", err
		}
		return marshalResult(items), nil
	}
}

func runSessionsListTool(store appsession.Store, threads *threadservice.ThreadService) func(ctx context.Context, args string) (string, error) {
	return func(ctx context.Context, _ string) (string, error) {
		if store != nil {
			items, err := store.List(ctx)
			if err != nil {
				return "", err
			}
			result := make([]sessionListItem, 0, len(items))
			for _, item := range items {
				result = append(result, sessionListItem{
					SessionID:  item.SessionID,
					SessionKey: item.SessionKey,
					Title:      item.Title,
					Status:     string(item.Status),
					UpdatedAt:  item.UpdatedAt,
				})
			}
			return marshalResult(result), nil
		}
		if threads == nil {
			return "", errors.New("thread service unavailable")
		}
		items, err := threads.ListThreads(ctx, false)
		if err != nil {
			return "", err
		}
		return marshalResult(items), nil
	}
}

func runSessionsHistoryTool(threads *threadservice.ThreadService) func(ctx context.Context, args string) (string, error) {
	return func(ctx context.Context, args string) (string, error) {
		if threads == nil {
			return "", errors.New("thread service unavailable")
		}
		payload, err := parseToolArgs(args)
		if err != nil {
			return "", err
		}
		threadID := getStringArg(payload, "threadId", "sessionId", "threadID", "sessionID")
		if threadID == "" {
			return "", errors.New("threadId is required")
		}
		limit, _ := getIntArg(payload, "limit", "max")
		items, err := threads.ListMessages(ctx, threadID, limit)
		if err != nil {
			return "", err
		}
		return marshalResult(items), nil
	}
}

func runSessionsSendTool(threads *threadservice.ThreadService) func(ctx context.Context, args string) (string, error) {
	return func(ctx context.Context, args string) (string, error) {
		if threads == nil {
			return "", errors.New("thread service unavailable")
		}
		payload, err := parseToolArgs(args)
		if err != nil {
			return "", err
		}
		threadID := getStringArg(payload, "threadId", "sessionId", "threadID", "sessionID")
		if threadID == "" {
			return "", errors.New("threadId is required")
		}
		content := getStringArg(payload, "content", "message", "text")
		if content == "" {
			return "", errors.New("content is required")
		}
		role := getStringArg(payload, "role")
		if role == "" {
			role = "user"
		}
		request := dto.AppendMessageRequest{
			ThreadID: threadID,
			Role:     role,
			Content:  content,
		}
		if err := threads.AppendMessage(ctx, request); err != nil {
			return "", err
		}
		return marshalResult(map[string]any{"ok": true}), nil
	}
}

func runSessionsSpawnTool(gateway *gatewaysubagent.GatewayService) func(ctx context.Context, args string) (string, error) {
	return func(ctx context.Context, args string) (string, error) {
		if gateway == nil {
			return "", errors.New("subagent gateway unavailable")
		}
		payload, err := parseToolArgs(args)
		if err != nil {
			return "", err
		}
		task := getStringArg(payload, "task")
		if task == "" {
			return "", errors.New("task is required")
		}
		sessionKey, runID := RuntimeContextFromContext(ctx)
		if sessionKey == "" {
			sessionKey = getStringArg(payload, "sessionKey", "parentSessionKey")
		}
		if sessionKey == "" {
			return "", errors.New("sessionKey is required")
		}
		parentRunID := getStringArg(payload, "parentRunId")
		if parentRunID == "" {
			parentRunID = runID
		}
		runtimeParams := llm.RuntimeParamsFromContext(ctx)
		callerModel := ""
		if strings.TrimSpace(runtimeParams.ProviderID) != "" && strings.TrimSpace(runtimeParams.ModelName) != "" {
			callerModel = strings.TrimSpace(runtimeParams.ProviderID) + "/" + strings.TrimSpace(runtimeParams.ModelName)
		}
		cleanup := getStringArg(payload, "cleanup")
		if cleanup != "" && !strings.EqualFold(cleanup, "keep") && !strings.EqualFold(cleanup, "delete") {
			return "", errors.New("cleanup must be keep or delete")
		}
		label := getStringArg(payload, "label")
		model := getStringArg(payload, "model")
		thinking := getStringArg(payload, "thinking")
		runTimeoutSeconds, _ := getIntArg(payload, "runTimeoutSeconds", "timeoutSeconds", "timeout")
		payloadMap := map[string]any{
			"task": task,
		}
		if label != "" {
			payloadMap["label"] = label
		}
		if model != "" {
			payloadMap["model"] = model
		}
		if thinking != "" {
			payloadMap["thinking"] = thinking
		}
		if runTimeoutSeconds > 0 {
			payloadMap["runTimeoutSeconds"] = runTimeoutSeconds
		}
		if cleanup != "" {
			payloadMap["cleanup"] = strings.ToLower(strings.TrimSpace(cleanup))
		}
		record, err := gateway.Spawn(ctx, subagentservice.SpawnRequest{
			ParentSessionKey:  sessionKey,
			ParentRunID:       parentRunID,
			AgentID:           getStringArg(payload, "agentId"),
			Task:              task,
			Label:             label,
			Model:             model,
			Thinking:          thinking,
			RunTimeoutSeconds: runTimeoutSeconds,
			CleanupPolicy:     subagentservice.ParseCleanupPolicy(cleanup),
			CallerModel:       callerModel,
			CallerThinking:    strings.TrimSpace(runtimeParams.ThinkingLevel),
			Payload:           payloadMap,
		})
		if err != nil {
			return "", err
		}
		return marshalResult(map[string]any{
			"status":          "accepted",
			"runId":           record.RunID,
			"childSessionKey": strings.TrimSpace(record.ChildSessionKey),
			"childSessionId":  strings.TrimSpace(record.ChildSessionID),
		}), nil
	}
}

func runSessionStatusTool(store appsession.Store) func(ctx context.Context, args string) (string, error) {
	return func(ctx context.Context, args string) (string, error) {
		if store == nil {
			return "", errors.New("session store unavailable")
		}
		payload, err := parseToolArgs(args)
		if err != nil {
			return "", err
		}
		sessionID := getStringArg(payload, "sessionId", "threadId", "sessionID", "threadID")
		if sessionID == "" {
			return "", errors.New("sessionId is required")
		}
		entry, err := store.Get(ctx, sessionID)
		if err != nil {
			if errors.Is(err, appsession.ErrSessionNotFound) {
				return "", errors.New("session not found")
			}
			return "", err
		}
		return marshalResult(sessionListItem{
			SessionID:  entry.SessionID,
			SessionKey: entry.SessionKey,
			Title:      entry.Title,
			Status:     string(entry.Status),
			UpdatedAt:  entry.UpdatedAt,
		}), nil
	}
}

func runSubagentsTool(gateway *gatewaysubagent.GatewayService) func(ctx context.Context, args string) (string, error) {
	return func(ctx context.Context, args string) (string, error) {
		payload, err := parseToolArgs(args)
		if err != nil {
			return "", err
		}
		action := strings.ToLower(getStringArg(payload, "action", "type"))
		if action == "" {
			action = "list"
		}
		sessionKey, _ := RuntimeContextFromContext(ctx)
		if sessionKey == "" {
			sessionKey = getStringArg(payload, "sessionKey", "parentSessionKey")
		}
		target := getStringArg(payload, "target", "runId", "id")
		switch action {
		case "profiles", "list-profiles":
			return "", errors.New("action profiles is not supported")
		case "list":
			if gateway == nil {
				return "", errors.New("subagent gateway unavailable")
			}
			if sessionKey == "" {
				return "", errors.New("sessionKey is required")
			}
			items, err := gateway.ListByParent(ctx, sessionKey)
			if err != nil {
				return "", err
			}
			return marshalResult(items), nil
		case "info", "log":
			if gateway == nil {
				return "", errors.New("subagent gateway unavailable")
			}
			runID, err := resolveSubagentTarget(ctx, gateway, sessionKey, target)
			if err != nil {
				return "", err
			}
			record, err := gateway.Get(ctx, runID)
			if err != nil {
				return "", err
			}
			return marshalResult(record), nil
		case "kill":
			if gateway == nil {
				return "", errors.New("subagent gateway unavailable")
			}
			if strings.EqualFold(target, "all") {
				if sessionKey == "" {
					return "", errors.New("sessionKey is required")
				}
				stopped, err := gateway.KillAll(ctx, sessionKey)
				if err != nil {
					return "", err
				}
				return marshalResult(map[string]any{"stopped": stopped}), nil
			}
			runID, err := resolveSubagentTarget(ctx, gateway, sessionKey, target)
			if err != nil {
				return "", err
			}
			if err := gateway.Kill(ctx, runID); err != nil {
				return "", err
			}
			return marshalResult(map[string]any{"stopped": 1}), nil
		case "steer", "send":
			if gateway == nil {
				return "", errors.New("subagent gateway unavailable")
			}
			message := getStringArg(payload, "message", "value", "input", "prompt")
			if message == "" {
				return "", errors.New("message is required")
			}
			runID, err := resolveSubagentTarget(ctx, gateway, sessionKey, target)
			if err != nil {
				return "", err
			}
			if action == "send" {
				if err := gateway.Send(ctx, runID, message); err != nil {
					return "", err
				}
			} else {
				if err := gateway.Steer(ctx, runID, message); err != nil {
					return "", err
				}
			}
			return marshalResult(map[string]any{"runId": runID, "queued": true}), nil
		default:
			return "", errors.New("unknown subagent action")
		}
	}
}

func resolveSubagentTarget(ctx context.Context, gateway *gatewaysubagent.GatewayService, sessionKey string, target string) (string, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return "", errors.New("target is required")
	}
	if gateway == nil {
		return "", errors.New("subagent gateway unavailable")
	}
	if strings.EqualFold(target, "all") {
		return "", errors.New("target all is only valid for kill")
	}
	if strings.HasPrefix(target, "#") {
		index, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(target, "#")))
		if err != nil || index <= 0 {
			return "", errors.New("subagent not found")
		}
		return resolveSubagentByIndex(ctx, gateway, sessionKey, index)
	}
	if index, err := strconv.Atoi(target); err == nil && index > 0 {
		return resolveSubagentByIndex(ctx, gateway, sessionKey, index)
	}
	record, err := gateway.Get(ctx, target)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(sessionKey) != "" && strings.TrimSpace(record.ParentSessionKey) != strings.TrimSpace(sessionKey) {
		return "", errors.New("subagent not found")
	}
	return record.RunID, nil
}

func resolveSubagentByIndex(ctx context.Context, gateway *gatewaysubagent.GatewayService, sessionKey string, index int) (string, error) {
	if index <= 0 {
		return "", errors.New("subagent not found")
	}
	if strings.TrimSpace(sessionKey) == "" {
		return "", errors.New("sessionKey is required")
	}
	items, err := gateway.ListByParent(ctx, sessionKey)
	if err != nil {
		return "", err
	}
	if index > len(items) {
		return "", errors.New("subagent not found")
	}
	return items[index-1].RunID, nil
}
