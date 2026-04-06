package heartbeat

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

var (
	heartbeatHeaderPattern    = regexp.MustCompile(`^#+(\s|$)`)
	heartbeatEmptyListPattern = regexp.MustCompile(`^[-*+]\s*(\[[\sXx]?\]\s*)?$`)
	heartbeatTagPattern       = regexp.MustCompile(`<[^>]*>`)
)

func isHeartbeatContentEffectivelyEmpty(content string) bool {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if heartbeatHeaderPattern.MatchString(trimmed) {
			continue
		}
		if heartbeatEmptyListPattern.MatchString(trimmed) {
			continue
		}
		return false
	}
	return true
}

func resolveHeartbeatPrompt(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return defaultHeartbeatPrompt
	}
	return trimmed
}

func appendSystemEvents(prompt string, events []SystemEvent) string {
	if len(events) == 0 {
		return strings.TrimSpace(prompt)
	}
	lines := make([]string, 0, len(events))
	for _, event := range events {
		text := strings.TrimSpace(event.Text)
		if text == "" {
			continue
		}
		prefixParts := make([]string, 0, 3)
		if contextKey := strings.TrimSpace(event.ContextKey); contextKey != "" {
			prefixParts = append(prefixParts, contextKey)
		}
		if source := strings.TrimSpace(event.Source); source != "" {
			prefixParts = append(prefixParts, source)
		}
		if runID := strings.TrimSpace(event.RunID); runID != "" {
			prefixParts = append(prefixParts, "run:"+runID)
		}
		if len(prefixParts) > 0 {
			text = fmt.Sprintf("[%s] %s", strings.Join(prefixParts, " | "), text)
		}
		lines = append(lines, text)
	}
	if len(lines) == 0 {
		return strings.TrimSpace(prompt)
	}
	builder := strings.Builder{}
	builder.WriteString("System events:\n")
	for _, line := range lines {
		builder.WriteString("- ")
		builder.WriteString(line)
		builder.WriteString("\n")
	}
	base := strings.TrimSpace(prompt)
	if base != "" {
		builder.WriteString("\n")
		builder.WriteString(base)
	}
	return strings.TrimSpace(builder.String())
}

func appendCurrentTime(prompt string, now time.Time) string {
	base := strings.TrimSpace(prompt)
	if base == "" {
		return base
	}
	if strings.Contains(base, "Current time:") {
		return base
	}
	zone := now.Location().String()
	if zone == "" {
		zone = "local"
	}
	timeLine := fmt.Sprintf("Current time: %s (%s)", now.Format(time.RFC3339), zone)
	return strings.TrimSpace(base + "\n" + timeLine)
}

func stripMarkup(text string) string {
	cleaned := heartbeatTagPattern.ReplaceAllString(text, " ")
	cleaned = strings.ReplaceAll(cleaned, "&nbsp;", " ")
	cleaned = strings.TrimSpace(cleaned)
	cleaned = strings.TrimLeft(cleaned, "*`~_")
	cleaned = strings.TrimRight(cleaned, "*`~_")
	return strings.TrimSpace(cleaned)
}
