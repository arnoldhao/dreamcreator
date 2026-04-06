package telegram

import (
	"fmt"
	"strings"

	"go.uber.org/zap"
)

type filteredLogger struct {
	token string
	debug bool
}

func newFilteredLogger(token string, debug bool) *filteredLogger {
	return &filteredLogger{token: strings.TrimSpace(token), debug: debug}
}

func (logger *filteredLogger) Debugf(format string, args ...any) {
	if logger == nil || !logger.debug {
		return
	}
	message := logger.redact(fmt.Sprintf(format, args...))
	zap.L().Debug("telegram api debug", zap.String("message", message))
}

func (logger *filteredLogger) Errorf(format string, args ...any) {
	if logger == nil {
		return
	}
	message := logger.redact(fmt.Sprintf(format, args...))
	if shouldSuppressTelegramLog(message) {
		return
	}
	zap.L().Warn("telegram api error", zap.String("error", message))
}

func (logger *filteredLogger) redact(message string) string {
	if logger == nil || message == "" {
		return message
	}
	token := strings.TrimSpace(logger.token)
	if token == "" {
		return message
	}
	masked := maskToken(token)
	redacted := strings.ReplaceAll(message, "bot"+token, "bot"+masked)
	if token != masked {
		redacted = strings.ReplaceAll(redacted, token, masked)
	}
	return redacted
}

func shouldSuppressTelegramLog(message string) bool {
	lower := strings.ToLower(message)
	if !strings.Contains(lower, "getupdates") {
		return false
	}
	if strings.Contains(lower, "context canceled") || strings.Contains(lower, "context deadline exceeded") {
		return true
	}
	return false
}

func maskToken(token string) string {
	const keepStart = 6
	const keepEnd = 4
	if len(token) < keepStart+keepEnd+1 {
		return "***"
	}
	return token[:keepStart] + "..." + token[len(token)-keepEnd:]
}
