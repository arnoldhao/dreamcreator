package runtime

import (
	"strings"
	"testing"
	"time"

	assistantdto "dreamcreator/internal/application/assistant/dto"
	gatewayprompt "dreamcreator/internal/application/gateway/prompt"
	skillsdto "dreamcreator/internal/application/skills/dto"
	tooldto "dreamcreator/internal/application/tools/dto"
	workspacedto "dreamcreator/internal/application/workspace/dto"
	domainassistant "dreamcreator/internal/domain/assistant"
)

func TestResolveUserTimeContext_ConvertsByTimezone(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.February, 26, 12, 34, 56, 0, time.UTC)
	timezone, line := resolveUserTimeContext(domainassistant.AssistantUser{
		Timezone: domainassistant.UserLocale{
			Mode:  "manual",
			Value: "Asia/Shanghai",
		},
	}, now)

	if timezone != "Asia/Shanghai" {
		t.Fatalf("expected timezone Asia/Shanghai, got %q", timezone)
	}
	const want = "Current time: 2026-02-26T20:34:56+08:00 (Asia/Shanghai)"
	if line != want {
		t.Fatalf("expected %q, got %q", want, line)
	}
}

func TestResolveUserTimeContext_FallbackWhenTimezoneInvalid(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.February, 26, 12, 34, 56, 0, time.UTC)
	timezone, line := resolveUserTimeContext(domainassistant.AssistantUser{
		Timezone: domainassistant.UserLocale{
			Mode:  "manual",
			Value: "Invalid/Timezone",
		},
	}, now)

	if timezone != "UTC" {
		t.Fatalf("expected fallback timezone UTC, got %q", timezone)
	}
	const want = "Current time: 2026-02-26T12:34:56Z (UTC)"
	if line != want {
		t.Fatalf("expected %q, got %q", want, line)
	}
}

func TestResolveUserTimeContext_UsesTimezoneValueInsteadOfLocalLabel(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.February, 26, 12, 34, 56, 0, time.FixedZone("Local", 8*60*60))
	timezone, line := resolveUserTimeContext(domainassistant.AssistantUser{}, now)

	if timezone != "UTC+08:00" {
		t.Fatalf("expected timezone UTC+08:00, got %q", timezone)
	}
	const want = "Current time: 2026-02-26T12:34:56+08:00 (UTC+08:00)"
	if line != want {
		t.Fatalf("expected %q, got %q", want, line)
	}
}

func TestFormatUserSection_IncludesCurrentTimeLine(t *testing.T) {
	t.Parallel()

	content := formatUserSection(assistantdto.AssistantSnapshot{
		User: domainassistant.AssistantUser{
			Timezone: domainassistant.UserLocale{
				Mode:  "manual",
				Value: "UTC",
			},
		},
	})

	if !strings.Contains(content, "Timezone: UTC") {
		t.Fatalf("expected timezone line in user section, got %q", content)
	}
	if !strings.Contains(content, "Current time: ") {
		t.Fatalf("expected current time line in user section, got %q", content)
	}
	if !strings.Contains(content, "(UTC)") {
		t.Fatalf("expected UTC zone in current time line, got %q", content)
	}
}

func TestFormatWorkspaceSection_SkipsMissingScaffoldFiles(t *testing.T) {
	t.Parallel()

	content := formatWorkspaceSection(workspacedto.RuntimeSnapshot{
		WorkspaceContext: workspacedto.WorkspaceContext{
			Files: []workspacedto.WorkspaceFile{
				{Name: "IDENTITY", Missing: true},
				{Name: "SOUL", Missing: true},
				{Name: "USER", Missing: true},
				{Name: "TOOLS", Missing: true},
				{Name: "MEMORY", Missing: true},
				{Name: "PERSONA", Missing: true},
				{Name: "AGENTS.md", Missing: true},
			},
		},
	})

	if strings.Contains(content, "### IDENTITY") || strings.Contains(content, "### USER") || strings.Contains(content, "### TOOLS") {
		t.Fatalf("expected scaffold files to be skipped, got %q", content)
	}
	if !strings.Contains(content, "### AGENTS.md") {
		t.Fatalf("expected non-scaffold workspace file to remain, got %q", content)
	}
}

func TestFormatWorkspaceSection_ReturnsEmptyWhenOnlyScaffoldFiles(t *testing.T) {
	t.Parallel()

	content := formatWorkspaceSection(workspacedto.RuntimeSnapshot{
		WorkspaceContext: workspacedto.WorkspaceContext{
			Files: []workspacedto.WorkspaceFile{
				{Name: "IDENTITY", Missing: true},
				{Name: "SOUL", Missing: true},
				{Name: "USER", Missing: true},
				{Name: "TOOLS", Missing: true},
				{Name: "MEMORY", Missing: true},
				{Name: "PERSONA", Missing: true},
			},
		},
	})

	if strings.TrimSpace(content) != "" {
		t.Fatalf("expected empty workspace section when only scaffold files exist, got %q", content)
	}
}

func TestFormatRuntimeSection_OnlyIncludesAllowedFields(t *testing.T) {
	t.Parallel()

	content := formatRuntimeSection(runtimePromptInfo{
		RunID:         "run_123",
		SessionID:     "session_123",
		SessionKey:    "session_key_123",
		Channel:       "chat",
		WorkspaceID:   "workspace_123",
		WorkspaceRoot: "/tmp/workspace",
	})

	if !strings.Contains(content, "Run ID: run_123") {
		t.Fatalf("expected run id in runtime section, got %q", content)
	}
	if !strings.Contains(content, "Channel: chat") {
		t.Fatalf("expected channel in runtime section, got %q", content)
	}
	if !strings.Contains(content, "Workspace Root: /tmp/workspace") {
		t.Fatalf("expected workspace root in runtime section, got %q", content)
	}
	if strings.Contains(content, "Session ID:") || strings.Contains(content, "Session Key:") || strings.Contains(content, "Workspace ID:") {
		t.Fatalf("expected runtime section to omit session/workspace ids, got %q", content)
	}
}

func TestBuildPromptDocument_OmitsWorkspaceSection(t *testing.T) {
	t.Parallel()

	_, _, sections := buildPromptDocument(promptBuildInput{
		Mode: domainassistant.PromptModeFull,
		Workspace: workspacedto.RuntimeSnapshot{
			WorkspaceContext: workspacedto.WorkspaceContext{
				Files: []workspacedto.WorkspaceFile{
					{Name: "AGENTS.md", Content: "workspace context"},
				},
			},
		},
		Runtime: runtimePromptInfo{
			RunID:         "run_123",
			Channel:       "chat",
			WorkspaceRoot: "/tmp/workspace",
		},
	})

	for _, section := range sections {
		if section.ID == "workspace" {
			t.Fatalf("expected workspace section to be omitted, got sections %+v", sections)
		}
	}
}

func TestBuildPromptDocument_FullIncludesToolingToolCallStyleAndSkillsMandatory(t *testing.T) {
	t.Parallel()

	_, _, sections := buildPromptDocument(promptBuildInput{
		Mode: domainassistant.PromptModeFull,
		Tools: []tooldto.ToolSpec{
			{
				Name:        "exec",
				Description: "Execute command",
				Enabled:     true,
			},
		},
		Skills: []skillsdto.SkillPromptItem{
			{
				Name:        "skill-a",
				Description: "Do A",
				Path:        "skills/skill-a/SKILL.md",
			},
		},
	})

	tooling := findSectionContent(sections, "tools")
	if !strings.Contains(tooling, "## Tooling") {
		t.Fatalf("expected tooling heading, got %q", tooling)
	}
	if !strings.Contains(tooling, "Tool names are case-sensitive. Call tools exactly as listed.") {
		t.Fatalf("expected case-sensitive tooling rule, got %q", tooling)
	}
	if !strings.Contains(tooling, "Do not loop on `subagents` list or `sessions_list`") {
		t.Fatalf("expected anti-poll tooling rule, got %q", tooling)
	}

	callStyle := findSectionContent(sections, "tool_call_style")
	if !strings.Contains(callStyle, "## Tool Call Style") {
		t.Fatalf("expected tool call style section, got %q", callStyle)
	}
	if !strings.Contains(callStyle, "Default to concise execution") {
		t.Fatalf("expected concise call style rule, got %q", callStyle)
	}

	skills := findSectionContent(sections, "skills")
	if !strings.Contains(skills, "## Skills (mandatory)") {
		t.Fatalf("expected mandatory skills heading, got %q", skills)
	}
	if !strings.Contains(skills, "Recommended flow: `skills.status` -> `skills_manage.search`/`skills_manage.install` -> `skills.status`.") {
		t.Fatalf("expected skills tool protocol rule, got %q", skills)
	}
}

func TestBuildPromptDocument_FullIncludesHeartbeatsSection(t *testing.T) {
	t.Parallel()

	_, _, sections := buildPromptDocument(promptBuildInput{
		Mode:            domainassistant.PromptModeFull,
		HeartbeatPrompt: "Review pending heartbeat checklist/system events.",
	})

	heartbeats := findSectionContent(sections, "heartbeats")
	if !strings.Contains(heartbeats, "## Heartbeats") {
		t.Fatalf("expected heartbeats heading, got %q", heartbeats)
	}
	if !strings.Contains(heartbeats, "Heartbeat prompt: Review pending heartbeat checklist/system events.") {
		t.Fatalf("expected heartbeat prompt line, got %q", heartbeats)
	}
	if !strings.Contains(heartbeats, "HEARTBEAT_OK") {
		t.Fatalf("expected heartbeat ack token rule, got %q", heartbeats)
	}
}

func TestBuildPromptDocument_MinimalIncludesToolingToolCallStyleAndSkillsWhenAvailable(t *testing.T) {
	t.Parallel()

	_, _, sections := buildPromptDocument(promptBuildInput{
		Mode: domainassistant.PromptModeMinimal,
		Tools: []tooldto.ToolSpec{
			{
				Name:        "read",
				Description: "Read file",
				Enabled:     true,
			},
		},
		Skills: []skillsdto.SkillPromptItem{
			{
				Name:        "skill-b",
				Description: "Do B",
			},
		},
	})

	if findSectionContent(sections, "tools") == "" {
		t.Fatalf("expected tooling section in minimal mode")
	}
	if findSectionContent(sections, "tool_call_style") == "" {
		t.Fatalf("expected tool_call_style section in minimal mode")
	}
	if findSectionContent(sections, "skills") == "" {
		t.Fatalf("expected skills section in minimal mode when skills exist")
	}
	if findSectionContent(sections, "heartbeats") != "" {
		t.Fatalf("expected heartbeats section omitted in minimal mode")
	}
}

func TestBuildPromptDocument_MinimalHeartbeatRunIncludesHeartbeatSection(t *testing.T) {
	t.Parallel()

	_, _, sections := buildPromptDocument(promptBuildInput{
		Mode:            domainassistant.PromptModeMinimal,
		RunKind:         "heartbeat",
		HeartbeatPrompt: "Review pending heartbeat checklist/system events.",
	})

	heartbeats := findSectionContent(sections, "heartbeats")
	if !strings.Contains(heartbeats, "## Heartbeats") {
		t.Fatalf("expected heartbeats heading for heartbeat run, got %q", heartbeats)
	}
	if !strings.Contains(heartbeats, "HEARTBEAT_OK") {
		t.Fatalf("expected heartbeat ack token rule for heartbeat run, got %q", heartbeats)
	}
}

func TestBuildPromptDocument_MinimalIncludesToolingWithoutTools(t *testing.T) {
	t.Parallel()

	_, _, sections := buildPromptDocument(promptBuildInput{
		Mode: domainassistant.PromptModeMinimal,
	})

	tooling := findSectionContent(sections, "tools")
	if !strings.Contains(tooling, "## Tooling") {
		t.Fatalf("expected tooling section in minimal mode even when tools empty, got %q", tooling)
	}
	if !strings.Contains(tooling, "- (none)") {
		t.Fatalf("expected explicit empty tooling marker, got %q", tooling)
	}
}

func TestBuildPromptDocument_NoneOmitsToolingToolCallStyleAndSkills(t *testing.T) {
	t.Parallel()

	_, _, sections := buildPromptDocument(promptBuildInput{
		Mode: domainassistant.PromptModeNone,
		Tools: []tooldto.ToolSpec{
			{Name: "exec", Enabled: true},
		},
		Skills: []skillsdto.SkillPromptItem{
			{Name: "skill-c"},
		},
	})

	if findSectionContent(sections, "tools") != "" {
		t.Fatalf("expected tools section omitted in none mode")
	}
	if findSectionContent(sections, "tool_call_style") != "" {
		t.Fatalf("expected tool_call_style section omitted in none mode")
	}
	if findSectionContent(sections, "skills") != "" {
		t.Fatalf("expected skills section omitted in none mode")
	}
}

func TestBuildPromptDocument_SubagentOmitsHeartbeatsSection(t *testing.T) {
	t.Parallel()

	_, _, sections := buildPromptDocument(promptBuildInput{
		Mode:            domainassistant.PromptModeFull,
		HeartbeatPrompt: "heartbeat prompt",
		IsSubagent:      true,
	})

	if findSectionContent(sections, "heartbeats") != "" {
		t.Fatalf("expected heartbeats section omitted for subagent")
	}
}

func findSectionContent(sections []gatewayprompt.Section, id string) string {
	for _, section := range sections {
		if section.ID == id {
			return section.Content
		}
	}
	return ""
}
