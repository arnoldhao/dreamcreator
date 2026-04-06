package heartbeat

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	settingsdto "dreamcreator/internal/application/settings/dto"
)

var heartbeatEveryPattern = regexp.MustCompile(`^(\d+)\s*([smhd])?$`)

func resolveHeartbeatInterval(cfg settingsdto.GatewayHeartbeatSettings) time.Duration {
	if !cfg.Periodic.Enabled {
		return 0
	}
	if raw := strings.TrimSpace(cfg.Periodic.Every); raw != "" {
		if parsed := parseHeartbeatDuration(raw); parsed > 0 {
			return parsed
		}
	}
	if raw := strings.TrimSpace(cfg.Every); raw != "" {
		if parsed := parseHeartbeatDuration(raw); parsed > 0 {
			return parsed
		}
	}
	if cfg.EveryMinutes > 0 {
		return time.Duration(cfg.EveryMinutes) * time.Minute
	}
	return 0
}

func parseHeartbeatDuration(value string) time.Duration {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	if trimmed == "" {
		return 0
	}
	match := heartbeatEveryPattern.FindStringSubmatch(trimmed)
	if match == nil {
		return 0
	}
	amount, err := strconv.Atoi(match[1])
	if err != nil || amount <= 0 {
		return 0
	}
	unit := match[2]
	switch unit {
	case "s":
		return time.Duration(amount) * time.Second
	case "h":
		return time.Duration(amount) * time.Hour
	case "d":
		return time.Duration(amount) * 24 * time.Hour
	default:
		return time.Duration(amount) * time.Minute
	}
}
