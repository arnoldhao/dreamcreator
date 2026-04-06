package service

import (
	"context"
	"strings"
	"time"

	settingsdto "dreamcreator/internal/application/settings/dto"
)

const (
	defaultSkillsAuditMaxEntries    = 200
	defaultSkillsAuditRetentionDays = 14
	maxSkillsAuditRetentionDays     = 365

	SkillsAuditSourceWeb      = "web"
	SkillsAuditSourceToolCall = "tool_call"
)

type skillsAuditSourceContextKey struct{}

func WithSkillsAuditSource(ctx context.Context, source string) context.Context {
	normalized := normalizeSkillsAuditSource(source)
	if normalized == "" || ctx == nil {
		return ctx
	}
	return context.WithValue(ctx, skillsAuditSourceContextKey{}, normalized)
}

func normalizeSkillsAuditSource(source string) string {
	switch strings.ToLower(strings.TrimSpace(source)) {
	case SkillsAuditSourceWeb, "ui", "frontend":
		return SkillsAuditSourceWeb
	case SkillsAuditSourceToolCall, "toolcall", "tool-call":
		return SkillsAuditSourceToolCall
	default:
		return ""
	}
}

func resolveSkillsAuditSourceFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	value, _ := ctx.Value(skillsAuditSourceContextKey{}).(string)
	return normalizeSkillsAuditSource(value)
}

func resolveSkillsAuditGroup(action string) string {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "skills.status", "skills.bins", "skill_manage.search", "skill_manage.list":
		return "read"
	case "skill_manage.install", "skill_manage.update", "skill_manage.remove", "skill_manage.sync":
		return "package_write"
	case "skills.install":
		return "deps_write"
	case "skills.update":
		return "config_write"
	case "source_write":
		return "source_write"
	default:
		return "read"
	}
}

func resolveSkillsAuditTool(action string) string {
	switch {
	case strings.HasPrefix(strings.ToLower(strings.TrimSpace(action)), "skills."):
		return "skills"
	case strings.HasPrefix(strings.ToLower(strings.TrimSpace(action)), "skill_manage."):
		return "skill_manage"
	default:
		return ""
	}
}

func (service *SkillsService) appendSkillsAuditRecord(
	ctx context.Context,
	action string,
	skill string,
	assistantID string,
	providerID string,
	opErr error,
) {
	if service == nil || service.settings == nil {
		return
	}
	source := resolveSkillsAuditSourceFromContext(ctx)
	if source == SkillsAuditSourceToolCall {
		return
	}
	if source == "" {
		source = SkillsAuditSourceWeb
	}

	action = strings.TrimSpace(action)
	if action == "" {
		return
	}

	record := map[string]any{
		"action":      action,
		"group":       resolveSkillsAuditGroup(action),
		"tool":        resolveSkillsAuditTool(action),
		"skill":       strings.TrimSpace(skill),
		"assistantId": strings.TrimSpace(assistantID),
		"providerId":  strings.TrimSpace(providerID),
		"source":      source,
		"ok":          opErr == nil,
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
	}
	if opErr != nil {
		record["error"] = strings.TrimSpace(opErr.Error())
		if detail, ok := ExtractClawHubErrorDetail(opErr); ok && strings.TrimSpace(detail.Code) != "" {
			record["errorCode"] = strings.TrimSpace(detail.Code)
		}
	}

	current, err := service.settings.GetSettings(ctx)
	if err != nil {
		return
	}

	toolsConfig, skillsConfig := resolveSettingsToolsSkills(current)
	if toolsConfig == nil {
		toolsConfig = map[string]any{}
	}
	if skillsConfig == nil {
		skillsConfig = map[string]any{}
	}

	auditEntries := make([]any, 0)
	switch typed := skillsConfig["audit"].(type) {
	case []any:
		auditEntries = append(auditEntries, typed...)
	case []map[string]any:
		for _, item := range typed {
			auditEntries = append(auditEntries, item)
		}
	}
	now := time.Now().UTC()
	auditEntries = pruneSkillsAuditEntriesByRetention(
		auditEntries,
		resolveSkillsAuditRetentionDays(skillsConfig),
		now,
	)
	if !(source == SkillsAuditSourceWeb && resolveSkillsAuditHideUIOperationRecords(skillsConfig)) {
		record["timestamp"] = now.Format(time.RFC3339)
		auditEntries = append(auditEntries, record)
	}
	maxEntries := resolveSkillsAuditMaxEntries(skillsConfig)
	if maxEntries > 0 && len(auditEntries) > maxEntries {
		auditEntries = append([]any(nil), auditEntries[len(auditEntries)-maxEntries:]...)
	}
	skillsConfig["audit"] = auditEntries

	_, _ = service.settings.UpdateSettings(ctx, settingsdto.UpdateSettingsRequest{
		Tools:  toolsConfig,
		Skills: skillsConfig,
	})
}

func resolveSkillsAuditMaxEntries(skillsConfig map[string]any) int {
	if value, ok := getNestedSkillsAuditInt(skillsConfig, "security", "audit", "maxEntries"); ok && value > 0 {
		return value
	}
	if value, ok := getNestedSkillsAuditInt(skillsConfig, "audit", "maxEntries"); ok && value > 0 {
		return value
	}
	return defaultSkillsAuditMaxEntries
}

func resolveSkillsAuditRetentionDays(skillsConfig map[string]any) int {
	if value, ok := getNestedSkillsAuditInt(skillsConfig, "auditConfig", "retentionDays"); ok {
		return normalizeSkillsAuditRetentionDays(value)
	}
	if value, ok := getNestedSkillsAuditInt(skillsConfig, "audit", "retentionDays"); ok {
		return normalizeSkillsAuditRetentionDays(value)
	}
	return defaultSkillsAuditRetentionDays
}

func normalizeSkillsAuditRetentionDays(value int) int {
	if value <= 0 {
		return defaultSkillsAuditRetentionDays
	}
	if value > maxSkillsAuditRetentionDays {
		return maxSkillsAuditRetentionDays
	}
	return value
}

func resolveSkillsAuditHideUIOperationRecords(skillsConfig map[string]any) bool {
	if value, ok := getNestedSkillsAuditBool(skillsConfig, "auditConfig", "hideUiOperationRecords"); ok {
		return value
	}
	return true
}

func pruneSkillsAuditEntriesByRetention(entries []any, retentionDays int, now time.Time) []any {
	if retentionDays <= 0 || len(entries) == 0 {
		return entries
	}
	cutoff := now.AddDate(0, 0, -retentionDays)
	pruned := make([]any, 0, len(entries))
	for _, entry := range entries {
		record, ok := entry.(map[string]any)
		if !ok {
			pruned = append(pruned, entry)
			continue
		}
		rawTimestamp, _ := record["timestamp"].(string)
		timestamp, err := time.Parse(time.RFC3339, strings.TrimSpace(rawTimestamp))
		if err == nil && timestamp.Before(cutoff) {
			continue
		}
		pruned = append(pruned, entry)
	}
	return pruned
}

func getNestedSkillsAuditInt(source map[string]any, path ...string) (int, bool) {
	if len(path) == 0 {
		return 0, false
	}
	current := source
	for index, key := range path {
		if current == nil {
			return 0, false
		}
		value, ok := current[key]
		if !ok {
			return 0, false
		}
		if index == len(path)-1 {
			switch typed := value.(type) {
			case int:
				return typed, true
			case int8:
				return int(typed), true
			case int16:
				return int(typed), true
			case int32:
				return int(typed), true
			case int64:
				return int(typed), true
			case float32:
				return int(typed), true
			case float64:
				return int(typed), true
			default:
				return 0, false
			}
		}
		next, ok := value.(map[string]any)
		if !ok {
			return 0, false
		}
		current = next
	}
	return 0, false
}

func getNestedSkillsAuditBool(source map[string]any, path ...string) (bool, bool) {
	if len(path) == 0 {
		return false, false
	}
	current := source
	for index, key := range path {
		if current == nil {
			return false, false
		}
		value, ok := current[key]
		if !ok {
			return false, false
		}
		if index == len(path)-1 {
			switch typed := value.(type) {
			case bool:
				return typed, true
			case string:
				normalized := strings.ToLower(strings.TrimSpace(typed))
				if normalized == "true" || normalized == "1" || normalized == "yes" {
					return true, true
				}
				if normalized == "false" || normalized == "0" || normalized == "no" {
					return false, true
				}
			}
			return false, false
		}
		next, ok := value.(map[string]any)
		if !ok {
			return false, false
		}
		current = next
	}
	return false, false
}
