package heartbeat

import "strings"

func buildExecEventPrompt(deliverToUser bool) string {
	if deliverToUser {
		return "An async command you ran earlier has completed. The result is shown in the system messages above. Please relay the command output to the user in a helpful way. If the command succeeded, share the relevant output. If it failed, explain what went wrong."
	}
	return "An async command completed. Review the system event details above and decide whether it needs attention. Do not relay anything to the user or send chat messages. If nothing needs attention, reply exactly HEARTBEAT_OK. If something needs attention, return only compact JSON with code, severity, params, and optional action."
}

func buildCronEventPrompt(pending []string, deliverToUser bool) string {
	eventText := strings.TrimSpace(strings.Join(pending, "\n"))
	if eventText == "" {
		return "A scheduled cron event was triggered, but no event content was found. Reply exactly HEARTBEAT_OK."
	}
	if !deliverToUser {
		return "A scheduled reminder event was triggered. Review the reminder content below and decide whether it needs attention. Do not relay anything to the user or send chat messages.\n\n" + eventText + "\n\nIf nothing needs attention, reply exactly HEARTBEAT_OK. If something needs attention, return only compact JSON with code, severity, params, and optional action."
	}
	return "A scheduled reminder has been triggered. The reminder content is:\n\n" + eventText + "\n\nPlease relay this reminder to the user in a helpful and friendly way."
}

func isExecCompletionEvent(text string) bool {
	return strings.Contains(strings.ToLower(text), "exec finished")
}

func isCronSystemEvent(text string) bool {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return false
	}
	return !isHeartbeatNoiseEvent(trimmed) && !isExecCompletionEvent(trimmed)
}

func isHeartbeatNoiseEvent(text string) bool {
	lower := strings.ToLower(strings.TrimSpace(text))
	if lower == "" {
		return false
	}
	if isHeartbeatAckEvent(lower) {
		return true
	}
	if strings.Contains(lower, "heartbeat poll") || strings.Contains(lower, "heartbeat wake") {
		return true
	}
	return false
}

func isHeartbeatAckEvent(text string) bool {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return false
	}
	lower := strings.ToLower(trimmed)
	token := strings.ToLower(heartbeatToken)
	if !strings.HasPrefix(lower, token) {
		return false
	}
	suffix := strings.TrimSpace(lower[len(token):])
	if suffix == "" {
		return true
	}
	first := suffix[0]
	if (first >= 'a' && first <= 'z') || (first >= '0' && first <= '9') || first == '_' {
		return false
	}
	return true
}

func isForceReason(reason string) bool {
	lower := strings.ToLower(strings.TrimSpace(reason))
	if lower == "" {
		return false
	}
	if strings.Contains(lower, "cron") || strings.Contains(lower, "exec") || strings.Contains(lower, "wake") {
		return true
	}
	if strings.Contains(lower, "hook") || strings.Contains(lower, "manual") {
		return true
	}
	return false
}

func isCronReason(reason string) bool {
	return strings.Contains(strings.ToLower(strings.TrimSpace(reason)), "cron")
}

func isExecReason(reason string) bool {
	return strings.Contains(strings.ToLower(strings.TrimSpace(reason)), "exec")
}
