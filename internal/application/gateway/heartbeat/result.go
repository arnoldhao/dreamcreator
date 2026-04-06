package heartbeat

import (
	"encoding/json"
	"strings"

	domainnotice "dreamcreator/internal/domain/notice"
)

type heartbeatResponseResult struct {
	Ack     bool
	Cleaned string
	Alert   heartbeatAlert
}

type heartbeatAlert struct {
	Code     string
	Severity domainnotice.Severity
	Params   map[string]string
	Action   domainnotice.Action
}

type heartbeatAlertPayload struct {
	Code     string                    `json:"code"`
	Severity string                    `json:"severity"`
	Params   map[string]any            `json:"params"`
	Action   *heartbeatAlertActionBody `json:"action,omitempty"`
}

type heartbeatAlertActionBody struct {
	Type     string         `json:"type"`
	LabelKey string         `json:"labelKey"`
	Target   string         `json:"target"`
	Params   map[string]any `json:"params,omitempty"`
}

func parseHeartbeatResponse(content string) heartbeatResponseResult {
	trimmed := strings.TrimSpace(content)
	trimmed = strings.TrimSpace(trimmed)
	if trimmed == "" {
		return heartbeatResponseResult{Ack: true}
	}
	if isExactHeartbeatAck(trimmed) || isExactHeartbeatAck(stripMarkup(trimmed)) {
		return heartbeatResponseResult{Ack: true}
	}
	if alert, ok := parseHeartbeatAlertPayload(trimmed); ok {
		cleaned := resolveHeartbeatAlertMessage(alert, trimmed)
		return heartbeatResponseResult{
			Ack:     false,
			Cleaned: cleaned,
			Alert:   alert,
		}
	}
	cleaned := strings.TrimSpace(trimmed)
	if cleaned == "" {
		cleaned = "heartbeat attention required"
	}
	return heartbeatResponseResult{
		Ack:     false,
		Cleaned: cleaned,
		Alert: heartbeatAlert{
			Code:     "heartbeat.generic_attention",
			Severity: domainnotice.SeverityWarning,
			Params: map[string]string{
				"detail": cleaned,
			},
		},
	}
}

func parseHeartbeatAlertPayload(content string) (heartbeatAlert, bool) {
	var payload heartbeatAlertPayload
	for _, candidate := range heartbeatJSONCandidates(content) {
		if candidate == "" {
			continue
		}
		if err := json.Unmarshal([]byte(candidate), &payload); err != nil {
			continue
		}
		code := strings.TrimSpace(payload.Code)
		if code == "" {
			continue
		}
		alert := heartbeatAlert{
			Code:     code,
			Severity: normalizeNoticeSeverity(payload.Severity),
			Params:   normalizeHeartbeatStringMap(payload.Params),
		}
		if alert.Severity == "" {
			alert.Severity = domainnotice.SeverityWarning
		}
		if payload.Action != nil {
			alert.Action = domainnotice.Action{
				Type:     strings.TrimSpace(payload.Action.Type),
				LabelKey: strings.TrimSpace(payload.Action.LabelKey),
				Target:   strings.TrimSpace(payload.Action.Target),
				Params:   normalizeHeartbeatStringMap(payload.Action.Params),
			}
		}
		return alert, true
	}
	return heartbeatAlert{}, false
}

func heartbeatJSONCandidates(content string) []string {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return nil
	}
	candidates := []string{trimmed}
	if strings.HasPrefix(trimmed, "```") {
		lines := strings.Split(trimmed, "\n")
		if len(lines) >= 3 {
			body := strings.Join(lines[1:len(lines)-1], "\n")
			body = strings.TrimSpace(body)
			if body != "" {
				candidates = append(candidates, body)
			}
		}
	}
	return candidates
}

func isExactHeartbeatAck(content string) bool {
	return strings.TrimSpace(content) == heartbeatToken
}

func normalizeHeartbeatStringMap(input map[string]any) map[string]string {
	if len(input) == 0 {
		return map[string]string{}
	}
	result := make(map[string]string, len(input))
	for key, value := range input {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" || value == nil {
			continue
		}
		result[trimmedKey] = strings.TrimSpace(strings.TrimSpace(strings.Trim(strings.ReplaceAll(strings.TrimSpace(toJSONStringValue(value)), "\n", " "), `"`)))
	}
	return result
}

func toJSONStringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	default:
		encoded, err := json.Marshal(typed)
		if err != nil {
			return ""
		}
		return string(encoded)
	}
}

func resolveHeartbeatAlertMessage(alert heartbeatAlert, raw string) string {
	if detail := strings.TrimSpace(alert.Params["detail"]); detail != "" {
		return detail
	}
	if summary := strings.TrimSpace(alert.Params["summary"]); summary != "" {
		return summary
	}
	if body := strings.TrimSpace(alert.Params["body"]); body != "" {
		return body
	}
	if trimmedRaw := strings.TrimSpace(raw); trimmedRaw != "" && !strings.HasPrefix(trimmedRaw, "{") {
		return trimmedRaw
	}
	code := strings.TrimSpace(alert.Code)
	if code == "" {
		return "heartbeat attention required"
	}
	return strings.ReplaceAll(code, ".", " ")
}

func normalizeNoticeSeverity(raw string) domainnotice.Severity {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(domainnotice.SeveritySuccess):
		return domainnotice.SeveritySuccess
	case string(domainnotice.SeverityWarning):
		return domainnotice.SeverityWarning
	case string(domainnotice.SeverityError):
		return domainnotice.SeverityError
	case string(domainnotice.SeverityCritical):
		return domainnotice.SeverityCritical
	default:
		return domainnotice.SeverityInfo
	}
}

func noticeSeverityRank(value domainnotice.Severity) int {
	switch value {
	case domainnotice.SeveritySuccess:
		return 1
	case domainnotice.SeverityWarning:
		return 2
	case domainnotice.SeverityError:
		return 3
	case domainnotice.SeverityCritical:
		return 4
	default:
		return 0
	}
}

func noticeSeverityMeetsMin(value domainnotice.Severity, min string) bool {
	trimmedMin := strings.TrimSpace(min)
	if trimmedMin == "" {
		return false
	}
	return noticeSeverityRank(value) >= noticeSeverityRank(normalizeNoticeSeverity(trimmedMin))
}
