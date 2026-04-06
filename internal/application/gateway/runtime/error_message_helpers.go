package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"dreamcreator/internal/application/chatevent"
)

const (
	runtimeErrorSummaryMaxRunes = 280
	runtimeErrorDetailMaxRunes  = 1200
)

func (service *Service) persistAssistantFailureMessage(
	ctx context.Context,
	threadID string,
	messageID string,
	runErr error,
) bool {
	content, parts, ok := buildRuntimeErrorFinalMessage(runErr)
	if !ok {
		return false
	}
	if err := service.persistAssistantMessage(ctx, threadID, messageID, content, parts); err != nil {
		return false
	}
	return true
}

func buildRuntimeErrorFinalMessage(err error) (string, []chatevent.MessagePart, bool) {
	if err == nil {
		return "", nil, false
	}
	if errors.Is(err, context.Canceled) {
		return "", nil, false
	}

	raw := compactRuntimeErrorText(err.Error())
	if raw == "" {
		raw = "llm request failed"
	}

	summary := raw
	hasHTMLBody := false
	if htmlStart := findRuntimeErrorHTMLStart(raw); htmlStart >= 0 {
		hasHTMLBody = true
		summary = strings.TrimSpace(raw[:htmlStart])
	}
	summary = strings.TrimSpace(strings.TrimSuffix(summary, ":"))
	if summary == "" {
		summary = "llm request failed"
	}
	if hasHTMLBody {
		summary = summary + " (html body omitted)"
	}

	summary = truncateRuntimeErrorText(summary, runtimeErrorSummaryMaxRunes)
	detail := truncateRuntimeErrorText(raw, runtimeErrorDetailMaxRunes)

	parts := []chatevent.MessagePart{{
		Type: "text",
		Text: summary,
	}}

	payload := map[string]any{
		"name": "runtime_error",
		"data": map[string]any{
			"message": summary,
		},
	}
	if detail != "" && detail != summary {
		payload["data"].(map[string]any)["detail"] = detail
	}
	if data, marshalErr := json.Marshal(payload); marshalErr == nil && len(data) > 0 {
		parts = append(parts, chatevent.MessagePart{
			Type: "data",
			Data: data,
		})
	}

	return summary, parts, true
}

func compactRuntimeErrorText(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	return strings.Join(strings.Fields(trimmed), " ")
}

func findRuntimeErrorHTMLStart(value string) int {
	lower := strings.ToLower(value)
	start := -1
	for _, marker := range []string{"<!doctype html", "<html", "<head", "<body"} {
		idx := strings.Index(lower, marker)
		if idx < 0 {
			continue
		}
		if start < 0 || idx < start {
			start = idx
		}
	}
	return start
}

func truncateRuntimeErrorText(value string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	runes := []rune(strings.TrimSpace(value))
	if len(runes) <= maxRunes {
		return string(runes)
	}
	if maxRunes <= 3 {
		return string(runes[:maxRunes])
	}
	return strings.TrimSpace(string(runes[:maxRunes-3])) + "..."
}
