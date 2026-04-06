package runtime

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	assistantdto "dreamcreator/internal/application/assistant/dto"
	gatewayprompt "dreamcreator/internal/application/gateway/prompt"
	skillsdto "dreamcreator/internal/application/skills/dto"
	tooldto "dreamcreator/internal/application/tools/dto"
	workspacedto "dreamcreator/internal/application/workspace/dto"
	domainassistant "dreamcreator/internal/domain/assistant"
)

const promptPreviewLimit = 400

type runtimePromptInfo struct {
	SessionID     string
	SessionKey    string
	Channel       string
	RunID         string
	WorkspaceID   string
	WorkspaceRoot string
}

type promptBuildInput struct {
	Mode              string
	RunKind           string
	Assistant         assistantdto.AssistantSnapshot
	Workspace         workspacedto.RuntimeSnapshot
	Tools             []tooldto.ToolSpec
	Skills            []skillsdto.SkillPromptItem
	HeartbeatPrompt   string
	IsSubagent        bool
	ExtraSystemPrompt string
	Runtime           runtimePromptInfo
}

func buildPromptDocument(input promptBuildInput) (gatewayprompt.Document, gatewayprompt.BuildReport, []gatewayprompt.Section) {
	mode := strings.ToLower(strings.TrimSpace(input.Mode))
	if mode == "" {
		mode = domainassistant.PromptModeFull
	}
	isMinimal := mode == domainassistant.PromptModeMinimal
	isNone := mode == domainassistant.PromptModeNone
	isHeartbeatRun := strings.EqualFold(strings.TrimSpace(input.RunKind), "heartbeat")

	sections := make([]gatewayprompt.Section, 0, 9)

	if isNone {
		if line := formatIdentityLine(input.Assistant); line != "" {
			sections = append(sections, makeSection("identity", "Identity", line))
		}
		if extra := strings.TrimSpace(input.ExtraSystemPrompt); extra != "" {
			sections = append(sections, makeSection("extra", "Extra", extra))
		}
		doc, report := gatewayprompt.NewSectionComposer(0).Compose(sections)
		return doc, report, sections
	}

	if content := formatIdentitySection(input.Assistant); content != "" {
		sections = append(sections, makeSection("identity", "Identity", content))
	}
	if !isMinimal {
		if content := formatUserSection(input.Assistant); content != "" {
			sections = append(sections, makeSection("user", "User", content))
		}
		if content := formatPersonaSection(input.Workspace); content != "" {
			sections = append(sections, makeSection("persona", "Persona", content))
		}
		if content := formatMemorySection(input.Assistant, input.Workspace); content != "" {
			sections = append(sections, makeSection("memory", "Memory", content))
		}
	}

	if content := formatToolsSection(input.Tools); content != "" {
		sections = append(sections, makeSection("tools", "Tooling", content))
	}
	if content := formatToolCallStyleSection(); content != "" {
		sections = append(sections, makeSection("tool_call_style", "Tool Call Style", content))
	}

	if !input.IsSubagent && !strings.EqualFold(strings.TrimSpace(input.Assistant.Skills.Mode), domainassistant.SkillsModeOff) {
		if content := formatSkillsSection(input.Skills); content != "" {
			sections = append(sections, makeSection("skills", "Skills (mandatory)", content))
		}
	}

	if !input.IsSubagent {
		if content := formatSubagentsSection(); content != "" {
			sections = append(sections, makeSection("subagents", "Subagents", content))
		}
	}
	if (!isMinimal || isHeartbeatRun) && !input.IsSubagent {
		if content := formatHeartbeatsSection(input.HeartbeatPrompt); content != "" {
			sections = append(sections, makeSection("heartbeats", "Heartbeats", content))
		}
	}

	if content := formatRuntimeSection(input.Runtime); content != "" {
		sections = append(sections, makeSection("runtime", "Runtime", content))
	}
	if extra := strings.TrimSpace(input.ExtraSystemPrompt); extra != "" {
		sections = append(sections, makeSection("extra", "Extra", extra))
	}

	doc, report := gatewayprompt.NewSectionComposer(0).Compose(sections)
	return doc, report, sections
}

func makeSection(id string, label string, content string) gatewayprompt.Section {
	return gatewayprompt.Section{
		ID:      id,
		Label:   label,
		Content: content,
	}
}

func formatIdentityLine(snapshot assistantdto.AssistantSnapshot) string {
	name := strings.TrimSpace(snapshot.Identity.Name)
	creature := strings.TrimSpace(snapshot.Identity.Creature)
	if name == "" && creature == "" {
		return ""
	}
	if creature == "" {
		return fmt.Sprintf("You are %s.", name)
	}
	if name == "" {
		return fmt.Sprintf("You are %s.", creature)
	}
	return fmt.Sprintf("You are %s, %s.", name, creature)
}

func formatIdentitySection(snapshot assistantdto.AssistantSnapshot) string {
	lines := []string{"## Identity"}
	if line := formatIdentityLine(snapshot); line != "" {
		lines = append(lines, line)
	}
	if value := strings.TrimSpace(snapshot.Identity.Emoji); value != "" {
		lines = append(lines, "Emoji: "+value)
	}
	if value := strings.TrimSpace(snapshot.Identity.Role); value != "" {
		lines = append(lines, "Role: "+value)
	}
	soul := snapshot.Identity.Soul
	appendBulletList(&lines, "Core Truths", soul.CoreTruths)
	appendBulletList(&lines, "Boundaries", soul.Boundaries)
	appendBulletList(&lines, "Rules", soul.Rules)
	if value := strings.TrimSpace(soul.Vibe); value != "" {
		lines = append(lines, "Vibe: "+value)
	}
	if value := strings.TrimSpace(soul.Continuity); value != "" {
		lines = append(lines, "Continuity: "+value)
	}
	return joinLines(lines)
}

func formatUserSection(snapshot assistantdto.AssistantSnapshot) string {
	user := snapshot.User
	lines := []string{"## User"}
	if value := strings.TrimSpace(user.Name); value != "" {
		lines = append(lines, "Name: "+value)
	}
	if value := strings.TrimSpace(user.PreferredAddress); value != "" {
		lines = append(lines, "Preferred address: "+value)
	}
	if value := strings.TrimSpace(user.Pronouns); value != "" {
		lines = append(lines, "Pronouns: "+value)
	}
	if value := strings.TrimSpace(user.Notes); value != "" {
		lines = append(lines, "Notes: "+value)
	}
	if value := strings.TrimSpace(domainassistant.ResolveUserLocaleValue(user.Language)); value != "" {
		lines = append(lines, "Language: "+value)
	}
	timezone, currentTimeLine := resolveUserTimeContext(user, time.Now())
	if timezone != "" {
		lines = append(lines, "Timezone: "+timezone)
	}
	if value := strings.TrimSpace(currentTimeLine); value != "" {
		lines = append(lines, value)
	}
	if value := strings.TrimSpace(domainassistant.ResolveUserLocaleValue(user.Location)); value != "" {
		lines = append(lines, "Location: "+value)
	}
	if len(user.Extra) > 0 {
		lines = append(lines, "Preferences:")
		for _, extra := range user.Extra {
			key := strings.TrimSpace(extra.Key)
			value := strings.TrimSpace(extra.Value)
			if key == "" {
				continue
			}
			if value == "" {
				lines = append(lines, "- "+key)
				continue
			}
			lines = append(lines, fmt.Sprintf("- %s: %s", key, value))
		}
	}
	return joinLines(lines)
}

func resolveUserTimeContext(user domainassistant.AssistantUser, now time.Time) (string, string) {
	location := now.Location()
	if value := strings.TrimSpace(domainassistant.ResolveUserLocaleValue(user.Timezone)); value != "" {
		if loaded, err := time.LoadLocation(value); err == nil {
			location = loaded
		}
	}
	current := now.In(location)
	timezone := resolveTimezoneLabel(current)
	return timezone, fmt.Sprintf("Current time: %s (%s)", current.Format(time.RFC3339), timezone)
}

func resolveTimezoneLabel(current time.Time) string {
	if value := normalizeRuntimeTimezoneValue(current.Location().String()); value != "" {
		return value
	}
	_, offsetSeconds := current.Zone()
	return formatUTCOffset(offsetSeconds)
}

func formatUTCOffset(offsetSeconds int) string {
	sign := "+"
	if offsetSeconds < 0 {
		sign = "-"
		offsetSeconds = -offsetSeconds
	}
	hours := offsetSeconds / 3600
	minutes := (offsetSeconds % 3600) / 60
	return fmt.Sprintf("UTC%s%02d:%02d", sign, hours, minutes)
}

func formatPersonaSection(snapshot workspacedto.RuntimeSnapshot) string {
	persona := strings.TrimSpace(snapshot.Persona)
	if persona == "" {
		return ""
	}
	return joinLines([]string{
		"## Persona",
		persona,
	})
}

func formatMemorySection(snapshot assistantdto.AssistantSnapshot, workspace workspacedto.RuntimeSnapshot) string {
	enabled := snapshot.Memory.Enabled
	memory := strings.TrimSpace(workspace.MemoryJSON)
	if !enabled && memory == "" {
		return ""
	}
	lines := []string{"## Memory"}
	if enabled {
		lines = append(lines, "Memory: enabled")
	} else {
		lines = append(lines, "Memory: disabled")
	}
	if memory != "" {
		lines = append(lines, "Memory config:")
		lines = append(lines, memory)
	}
	return joinLines(lines)
}

func formatToolsSection(tools []tooldto.ToolSpec) string {
	sorted := append([]tooldto.ToolSpec(nil), tools...)
	sort.Slice(sorted, func(i, j int) bool {
		return strings.ToLower(sorted[i].Name) < strings.ToLower(sorted[j].Name)
	})
	lines := []string{
		"## Tooling",
		"Tool availability (filtered by policy):",
	}
	appended := false
	for _, tool := range sorted {
		name := strings.TrimSpace(tool.Name)
		if name == "" {
			continue
		}
		desc := strings.TrimSpace(tool.Description)
		line := "- " + name
		if desc != "" {
			line += ": " + desc
		}
		lines = append(lines, line)
		appended = true
	}
	if !appended {
		lines = append(lines, "- (none)")
	}
	lines = append(lines,
		"Tool names are case-sensitive. Call tools exactly as listed.",
		"Avoid poll loops for long waits. Prefer `exec`/`process` with bounded waits and event-driven follow-ups.",
		"For complex work, use `sessions_spawn`; completion is push-based.",
		"Do not loop on `subagents` list or `sessions_list` to wait for completion.",
	)
	return joinLines(lines)
}

func formatSkillsSection(skills []skillsdto.SkillPromptItem) string {
	lines := append([]string{}, skillsSectionPreambleLines()...)
	if len(skills) == 0 {
		lines = append(lines,
			"- (none currently eligible)",
			"- Use `skill_manage.search` to discover skills, `skill_manage.install` to install, then `skills.status` to refresh.",
		)
		return joinLines(lines)
	}
	sorted := append([]skillsdto.SkillPromptItem(nil), skills...)
	sort.Slice(sorted, func(i, j int) bool {
		return strings.ToLower(sorted[i].Name) < strings.ToLower(sorted[j].Name)
	})
	for _, item := range sorted {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			continue
		}
		desc := strings.TrimSpace(item.Description)
		line := "- " + name
		if desc != "" {
			line += ": " + desc
		}
		if path := strings.TrimSpace(item.Path); path != "" {
			line += " (path: " + path + ")"
		}
		lines = append(lines, line)
	}
	return joinLines(lines)
}

func formatToolCallStyleSection() string {
	lines := []string{
		"## Tool Call Style",
		"Default to concise execution; do not narrate routine low-risk tool calls.",
		"Narrate briefly when actions are multi-step, risky, user-visible, or explicitly requested.",
		"Keep narration short, high-signal, and outcome-focused.",
		"In non-technical conversations, prefer natural language over tool jargon.",
	}
	return joinLines(lines)
}

func skillsSectionPreambleLines() []string {
	return []string{
		"## Skills (mandatory)",
		"Skills protocol:",
		"- First scan the available skills list.",
		"- If one skill clearly matches, read its `SKILL.md` before acting.",
		"- If no skill currently matches the task, call `skill_manage` instead of shell commands.",
		"- Recommended flow: `skills.status` -> `skill_manage.search`/`skill_manage.install` -> `skills.status`.",
		"- If status reports missing runtime dependencies, call `skills.install`, then re-check with `skills.status`.",
		"- Use `skills.update` for per-skill settings (enabled/apiKey/env/config) when required.",
		"Available skills:",
	}
}

func formatSubagentsSection() string {
	lines := []string{
		"## Subagents",
		"Use `sessions_spawn` for multi-step or long-running tasks.",
		"Subagent completion is push-based; do not wait with polling loops.",
		"Do not loop on `subagents` list or `sessions_list` to wait for completion.",
	}
	return joinLines(lines)
}

func formatHeartbeatsSection(configuredPrompt string) string {
	prompt := strings.TrimSpace(configuredPrompt)
	if prompt == "" {
		prompt = "(configured)"
	}
	lines := []string{
		"## Heartbeats",
		"Heartbeat prompt: " + prompt,
		"If you receive a heartbeat poll (a user message matching the heartbeat prompt above), and there is nothing that needs attention, reply exactly:",
		"HEARTBEAT_OK",
		"If something needs attention, do not include HEARTBEAT_OK and reply with compact JSON containing code, severity, params, and optional action.",
	}
	return joinLines(lines)
}

func formatWorkspaceSection(snapshot workspacedto.RuntimeSnapshot) string {
	files := snapshot.WorkspaceContext.Files
	if len(files) == 0 {
		return ""
	}
	lines := []string{"## Workspace"}
	for _, file := range files {
		if shouldSkipWorkspaceMissingScaffold(file) {
			continue
		}
		name := strings.TrimSpace(file.Name)
		if name == "" {
			continue
		}
		lines = append(lines, "### "+name)
		if path := strings.TrimSpace(file.Path); path != "" {
			lines = append(lines, "Path: "+path)
		}
		if file.Missing {
			lines = append(lines, "(missing)")
			continue
		}
		content := strings.TrimSpace(file.Content)
		if content == "" {
			continue
		}
		maxChars := file.MaxChars
		lines = append(lines, truncatePreview(content, maxChars))
	}
	if len(lines) == 1 {
		return ""
	}
	return joinLines(lines)
}

func shouldSkipWorkspaceMissingScaffold(file workspacedto.WorkspaceFile) bool {
	if !file.Missing {
		return false
	}
	if strings.TrimSpace(file.Content) != "" {
		return false
	}
	switch strings.ToUpper(strings.TrimSpace(file.Name)) {
	case "IDENTITY", "SOUL", "USER", "TOOLS", "MEMORY", "PERSONA":
		return true
	default:
		return false
	}
}

func formatRuntimeSection(info runtimePromptInfo) string {
	lines := []string{"## Runtime"}
	if value := strings.TrimSpace(info.RunID); value != "" {
		lines = append(lines, "Run ID: "+value)
	}
	if value := strings.TrimSpace(info.Channel); value != "" {
		lines = append(lines, "Channel: "+value)
	}
	if value := strings.TrimSpace(info.WorkspaceRoot); value != "" {
		lines = append(lines, "Workspace Root: "+value)
	}
	if len(lines) == 1 {
		return ""
	}
	return joinLines(lines)
}

func summarizeToolSchema(schemaJSON string) string {
	trimmed := strings.TrimSpace(schemaJSON)
	if trimmed == "" {
		return ""
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(trimmed), &payload); err != nil {
		return "schema available"
	}
	props, _ := payload["properties"].(map[string]any)
	requiredRaw, _ := payload["required"].([]any)
	keys := make([]string, 0, len(props))
	for key := range props {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	required := make([]string, 0, len(requiredRaw))
	for _, entry := range requiredRaw {
		if value, ok := entry.(string); ok && strings.TrimSpace(value) != "" {
			required = append(required, strings.TrimSpace(value))
		}
	}
	summaryParts := make([]string, 0, 2)
	if len(keys) > 0 {
		summaryParts = append(summaryParts, "params: "+strings.Join(keys, ", "))
	}
	if len(required) > 0 {
		summaryParts = append(summaryParts, "required: "+strings.Join(required, ", "))
	}
	if len(summaryParts) == 0 {
		return "schema available"
	}
	return strings.Join(summaryParts, "; ")
}

func appendBulletList(lines *[]string, label string, items []string) {
	if lines == nil || len(items) == 0 {
		return
	}
	clean := make([]string, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		clean = append(clean, trimmed)
	}
	if len(clean) == 0 {
		return
	}
	*lines = append(*lines, label+":")
	for _, item := range clean {
		*lines = append(*lines, "- "+item)
	}
}

func joinLines(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	trimmed := make([]string, 0, len(lines))
	for _, line := range lines {
		value := strings.TrimSpace(line)
		if value == "" {
			continue
		}
		trimmed = append(trimmed, value)
	}
	return strings.TrimSpace(strings.Join(trimmed, "\n"))
}

func truncatePreview(value string, maxChars int) string {
	limit := maxChars
	if limit <= 0 {
		limit = promptPreviewLimit
	}
	trimmed := strings.TrimSpace(value)
	if len(trimmed) <= limit {
		return trimmed
	}
	return strings.TrimSpace(trimmed[:limit]) + "..."
}
