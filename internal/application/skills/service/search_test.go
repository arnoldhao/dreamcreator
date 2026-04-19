package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"dreamcreator/internal/application/skills/dto"
	workspaceDTO "dreamcreator/internal/application/workspace/dto"
	"dreamcreator/internal/domain/externaltools"
)

type skillsExternalToolsStub struct {
	ready    bool
	reason   string
	execPath string
	err      error
}

func (stub *skillsExternalToolsStub) IsToolReady(_ context.Context, _ externaltools.ToolName) (bool, error) {
	return stub.ready, stub.err
}

func (stub *skillsExternalToolsStub) ResolveExecPath(_ context.Context, _ externaltools.ToolName) (string, error) {
	if stub.err != nil {
		return "", stub.err
	}
	if !stub.ready {
		return "", errors.New("not ready")
	}
	return stub.execPath, nil
}

func (stub *skillsExternalToolsStub) ToolReadiness(_ context.Context, _ externaltools.ToolName) (bool, string, error) {
	if stub.err != nil {
		return false, "", stub.err
	}
	return stub.ready, stub.reason, nil
}

type skillsWorkspaceResolverStub struct {
	rootPath string
}

func (stub *skillsWorkspaceResolverStub) GetAssistantWorkspaceDirectory(_ context.Context, assistantID string) (workspaceDTO.AssistantWorkspaceDirectory, error) {
	return workspaceDTO.AssistantWorkspaceDirectory{
		AssistantID: assistantID,
		WorkspaceID: assistantID,
		RootPath:    stub.rootPath,
	}, nil
}

func TestSearchSkillsReturnsClawHubUnavailableWhenNotReady(t *testing.T) {
	t.Parallel()

	svc := NewSkillsService(newMemorySkillsRepo(), nil)
	svc.SetExternalTools(&skillsExternalToolsStub{ready: false, reason: "not_installed"})

	_, err := svc.SearchSkills(context.Background(), dto.SearchSkillsRequest{Query: "test"})
	if !errors.Is(err, ErrClawHubUnavailable) {
		t.Fatalf("expected clawhub unavailable error, got %v", err)
	}
}

func TestSearchSkillsParsesClawHubOutput(t *testing.T) {
	t.Parallel()

	scriptPath := filepath.Join(t.TempDir(), "clawhub")
	script := "#!/bin/sh\necho \"owner/skill@1.2.3\"\necho \"A demo skill\"\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write script failed: %v", err)
	}

	svc := NewSkillsService(newMemorySkillsRepo(), nil)
	svc.SetExternalTools(&skillsExternalToolsStub{ready: true, execPath: scriptPath})

	result, err := svc.SearchSkills(context.Background(), dto.SearchSkillsRequest{Query: "demo"})
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected one result, got %d", len(result))
	}
	if result[0].ID != "owner/skill@1.2.3" {
		t.Fatalf("unexpected id %q", result[0].ID)
	}
	if result[0].Source != "clawhub" {
		t.Fatalf("unexpected source %q", result[0].Source)
	}
	if result[0].URL != "" {
		t.Fatalf("expected empty url when clawhub does not return one, got %q", result[0].URL)
	}
}

func TestSearchSkillsParsesClawHubSearchLineFormat(t *testing.T) {
	t.Parallel()

	scriptPath := filepath.Join(t.TempDir(), "clawhub")
	script := "#!/bin/sh\necho \"- Searching\"\necho \"my-skill-pack v1.2.3  My Skill Pack  (0.987)\"\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write script failed: %v", err)
	}

	svc := NewSkillsService(newMemorySkillsRepo(), nil)
	svc.SetExternalTools(&skillsExternalToolsStub{ready: true, execPath: scriptPath})

	result, err := svc.SearchSkills(context.Background(), dto.SearchSkillsRequest{Query: "my skill"})
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected one result, got %d", len(result))
	}
	if result[0].ID != "my-skill-pack" {
		t.Fatalf("unexpected id %q", result[0].ID)
	}
	if result[0].Name != "My Skill Pack" {
		t.Fatalf("unexpected name %q", result[0].Name)
	}
	if result[0].URL != "" {
		t.Fatalf("expected empty url for text search output, got %q", result[0].URL)
	}
}

func TestInspectSkillParsesClawHubInspectOutput(t *testing.T) {
	t.Parallel()

	scriptPath := filepath.Join(t.TempDir(), "clawhub")
	script := "#!/bin/sh\ncat <<'EOF'\n{\"skill\":{\"slug\":\"demo-skill\",\"displayName\":\"Demo Skill\",\"summary\":\"Demo summary\",\"tags\":{\"stable\":\"1.2.3\",\"latest\":\"1.2.3\"},\"createdAt\":1700000000000,\"updatedAt\":1700100000000},\"latestVersion\":{\"version\":\"1.2.3\",\"changelog\":\"latest changelog\"},\"owner\":{\"handle\":\"demo\"},\"version\":{\"version\":\"1.2.3\",\"changelog\":\"selected changelog\",\"files\":[{\"path\":\"SKILL.md\",\"size\":1234,\"sha256\":\"abc123\",\"contentType\":\"text/markdown\"}]}}\nEOF\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write script failed: %v", err)
	}

	svc := NewSkillsService(newMemorySkillsRepo(), nil)
	svc.SetExternalTools(&skillsExternalToolsStub{ready: true, execPath: scriptPath})

	detail, err := svc.InspectSkill(context.Background(), dto.InspectSkillRequest{Skill: "demo-skill"})
	if err != nil {
		t.Fatalf("inspect failed: %v", err)
	}
	if detail.ID != "demo-skill" {
		t.Fatalf("unexpected id %q", detail.ID)
	}
	if detail.Name != "Demo Skill" {
		t.Fatalf("unexpected name %q", detail.Name)
	}
	if detail.Summary != "Demo summary" {
		t.Fatalf("unexpected summary %q", detail.Summary)
	}
	if detail.Owner != "demo" {
		t.Fatalf("unexpected owner %q", detail.Owner)
	}
	if detail.LatestVersion != "1.2.3" {
		t.Fatalf("unexpected latest version %q", detail.LatestVersion)
	}
	if detail.SelectedVersion != "1.2.3" {
		t.Fatalf("unexpected selected version %q", detail.SelectedVersion)
	}
	if detail.Changelog != "selected changelog" {
		t.Fatalf("unexpected changelog %q", detail.Changelog)
	}
	if len(detail.Tags) != 2 {
		t.Fatalf("unexpected tags %#v", detail.Tags)
	}
	if detail.URL != "https://clawhub.ai/demo/demo-skill" {
		t.Fatalf("unexpected detail url %q", detail.URL)
	}
	if len(detail.Files) != 1 {
		t.Fatalf("unexpected detail files %#v", detail.Files)
	}
	if detail.Files[0].Path != "SKILL.md" {
		t.Fatalf("unexpected detail file path %q", detail.Files[0].Path)
	}
}

func TestInspectSkillParsesInspectOutputWithSpinnerPrefix(t *testing.T) {
	t.Parallel()

	scriptPath := filepath.Join(t.TempDir(), "clawhub")
	script := "#!/bin/sh\ncat <<'EOF'\n- Fetching skill\n{\"skill\":{\"slug\":\"demo-skill\",\"displayName\":\"Demo Skill\",\"summary\":\"Demo summary\",\"createdAt\":1700000000000,\"updatedAt\":1700100000000},\"latestVersion\":{\"version\":\"1.2.3\"},\"owner\":{\"handle\":\"demo\"},\"version\":{\"version\":\"1.2.3\"}}\nEOF\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write script failed: %v", err)
	}

	svc := NewSkillsService(newMemorySkillsRepo(), nil)
	svc.SetExternalTools(&skillsExternalToolsStub{ready: true, execPath: scriptPath})

	detail, err := svc.InspectSkill(context.Background(), dto.InspectSkillRequest{Skill: "demo-skill"})
	if err != nil {
		t.Fatalf("inspect failed: %v", err)
	}
	if detail.ID != "demo-skill" {
		t.Fatalf("unexpected id %q", detail.ID)
	}
	if detail.Owner != "demo" {
		t.Fatalf("unexpected owner %q", detail.Owner)
	}
	if detail.LatestVersion != "1.2.3" {
		t.Fatalf("unexpected latest version %q", detail.LatestVersion)
	}
	if detail.URL != "https://clawhub.ai/demo/demo-skill" {
		t.Fatalf("unexpected detail url %q", detail.URL)
	}
}

func TestInspectSkillReturnsUnavailableWhenNotReady(t *testing.T) {
	t.Parallel()

	svc := NewSkillsService(newMemorySkillsRepo(), nil)
	svc.SetExternalTools(&skillsExternalToolsStub{ready: false, reason: "not_installed"})

	_, err := svc.InspectSkill(context.Background(), dto.InspectSkillRequest{Skill: "demo"})
	if !errors.Is(err, ErrClawHubUnavailable) {
		t.Fatalf("expected clawhub unavailable error, got %v", err)
	}
}

func TestParseClawHubInspectFileContent(t *testing.T) {
	t.Parallel()

	output := []byte(`{"file":{"path":"SKILL.md","content":"---\nname: Demo\n---\n# Demo"}}`)
	content, err := parseClawHubInspectFileContent(output)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if !strings.Contains(content, "name: Demo") {
		t.Fatalf("unexpected content %q", content)
	}
}

func TestFetchSkillMarkdownRespectsMaxSkillFileBytes(t *testing.T) {
	t.Parallel()

	scriptPath := filepath.Join(t.TempDir(), "clawhub")
	script := "#!/bin/sh\ncat <<'EOF'\n{\"file\":{\"path\":\"SKILL.md\",\"content\":\"0123456789ABCDEF\"}}\nEOF\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write script failed: %v", err)
	}

	svc := NewSkillsService(newMemorySkillsRepo(), nil)
	svc.SetExternalTools(&skillsExternalToolsStub{ready: true, execPath: scriptPath})

	workspaceRoot := t.TempDir()
	oversized := svc.fetchSkillMarkdown(context.Background(), workspaceRoot, "demo-skill", 10)
	if oversized != "" {
		t.Fatalf("expected markdown to be skipped when max bytes exceeded, got %q", oversized)
	}
	allowed := svc.fetchSkillMarkdown(context.Background(), workspaceRoot, "demo-skill", 64)
	if strings.TrimSpace(allowed) != "0123456789ABCDEF" {
		t.Fatalf("unexpected markdown content %q", allowed)
	}
}

func TestParseSkillRuntimeRequirementsFromMarkdown(t *testing.T) {
	t.Parallel()

	markdown := strings.Join([]string{
		"---",
		"metadata:",
		"  clawdbot:",
		"    primaryEnv: OPENAI_API_KEY",
		"    homepage: https://example.com",
		"    os: [darwin, linux]",
		"    requires:",
		"      bins: [curl, git]",
		"      anyBins: [python3, uv]",
		"      env: [OPENAI_API_KEY]",
		"      config: [~/.config/demo.json]",
		"    install:",
		"      - kind: node",
		"        package: demo-skill",
		"        bins: [demo]",
		"---",
		"# Demo",
	}, "\n")

	runtime := parseSkillRuntimeRequirementsFromMarkdown(markdown)
	if runtime == nil {
		t.Fatalf("expected runtime requirements")
	}
	if runtime.PrimaryEnv != "OPENAI_API_KEY" {
		t.Fatalf("unexpected primary env %q", runtime.PrimaryEnv)
	}
	if len(runtime.Bins) != 2 || runtime.Bins[0] != "curl" {
		t.Fatalf("unexpected bins %#v", runtime.Bins)
	}
	if len(runtime.Install) != 1 || runtime.Install[0].Package != "demo-skill" {
		t.Fatalf("unexpected install specs %#v", runtime.Install)
	}
}

func TestRemoveSkillUsesClawHubUninstallCommand(t *testing.T) {
	t.Parallel()

	workdir := t.TempDir()
	argsPath := filepath.Join(workdir, "args.txt")
	scriptPath := filepath.Join(workdir, "clawhub")
	script := "#!/bin/sh\nprintf \"%s\\n\" \"$@\" > \"" + argsPath + "\"\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write script failed: %v", err)
	}

	svc := NewSkillsService(newMemorySkillsRepo(), nil)
	svc.SetExternalTools(&skillsExternalToolsStub{ready: true, execPath: scriptPath})

	if err := svc.RemoveSkill(context.Background(), dto.RemoveSkillRequest{
		Skill:         "demo-skill",
		WorkspaceRoot: workdir,
	}); err != nil {
		t.Fatalf("remove failed: %v", err)
	}
	data, err := os.ReadFile(argsPath)
	if err != nil {
		t.Fatalf("read args failed: %v", err)
	}
	argv := string(data)
	if !strings.Contains(argv, "uninstall\n") {
		t.Fatalf("expected uninstall command, got %q", argv)
	}
	if !strings.Contains(argv, "demo-skill\n") {
		t.Fatalf("expected skill slug, got %q", argv)
	}
	if !strings.Contains(argv, "--yes\n") {
		t.Fatalf("expected --yes option, got %q", argv)
	}
	if !strings.Contains(argv, "--no-input\n") {
		t.Fatalf("expected --no-input option, got %q", argv)
	}
}

func TestRemoveSkillTreatsNotInstalledAsSuccess(t *testing.T) {
	t.Parallel()

	appRoot := t.TempDir()
	workdir := filepath.Join(appRoot, "workspaces", "assistant-1")
	if err := os.MkdirAll(workdir, 0o755); err != nil {
		t.Fatalf("create workspace root failed: %v", err)
	}
	skillDir := filepath.Join(appRoot, "skills", "demo-skill")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("create skill dir failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nid: demo-skill\nname: Demo Skill\n---\n# Demo Skill\n"), 0o644); err != nil {
		t.Fatalf("write skill file failed: %v", err)
	}
	scriptPath := filepath.Join(appRoot, "clawhub")
	script := "#!/bin/sh\necho \"Error: Not installed: demo-skill\" 1>&2\nexit 1\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write script failed: %v", err)
	}

	svc := NewSkillsService(newMemorySkillsRepo(), nil)
	svc.SetExternalTools(&skillsExternalToolsStub{ready: true, execPath: scriptPath})

	if err := svc.RemoveSkill(context.Background(), dto.RemoveSkillRequest{
		Skill:         "demo-skill",
		WorkspaceRoot: workdir,
	}); err != nil {
		t.Fatalf("remove should be idempotent for not installed skill, got %v", err)
	}
	if _, err := os.Stat(skillDir); !os.IsNotExist(err) {
		t.Fatalf("expected local skill directory to be cleaned, stat err=%v", err)
	}
}

func TestSyncSkillsUsesAssistantWorkspaceRoot(t *testing.T) {
	t.Parallel()

	appRoot := t.TempDir()
	root := filepath.Join(appRoot, "workspaces", "assistant-1")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("create workspace root failed: %v", err)
	}
	skillDir := filepath.Join(appRoot, "skills", "local-skill")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("create skill dir failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nid: local-skill\nname: Local Skill\n---\n# Local Skill\n"), 0o644); err != nil {
		t.Fatalf("write skill file failed: %v", err)
	}

	scriptPath := filepath.Join(t.TempDir(), "clawhub")
	if err := os.WriteFile(scriptPath, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write script failed: %v", err)
	}

	svc := NewSkillsService(newMemorySkillsRepo(), nil)
	svc.SetExternalTools(&skillsExternalToolsStub{ready: true, execPath: scriptPath})
	svc.SetWorkspaceResolver(&skillsWorkspaceResolverStub{rootPath: root})

	result, err := svc.SyncSkills(context.Background(), dto.SyncSkillsRequest{
		AssistantID: "assistant-1",
	})
	if err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	if len(result) != 1 || result[0].ID != "local-skill" {
		t.Fatalf("unexpected sync result: %#v", result)
	}
}

func TestInstallSkillUsesForceFlag(t *testing.T) {
	t.Parallel()

	appRoot := t.TempDir()
	workdir := filepath.Join(appRoot, "workspaces", "assistant-1")
	if err := os.MkdirAll(workdir, 0o755); err != nil {
		t.Fatalf("create workspace root failed: %v", err)
	}
	argsPath := filepath.Join(appRoot, "args.txt")
	scriptPath := filepath.Join(appRoot, "clawhub")
	script := "#!/bin/sh\nprintf \"%s\\n\" \"$@\" > \"" + argsPath + "\"\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write script failed: %v", err)
	}

	svc := NewSkillsService(newMemorySkillsRepo(), nil)
	svc.SetExternalTools(&skillsExternalToolsStub{ready: true, execPath: scriptPath})

	err := svc.InstallSkill(context.Background(), dto.InstallSkillRequest{
		Skill:         "web-search-pro",
		Version:       "1.2.3",
		Force:         true,
		WorkspaceRoot: workdir,
	})
	if err != nil {
		t.Fatalf("install failed: %v", err)
	}
	data, readErr := os.ReadFile(argsPath)
	if readErr != nil {
		t.Fatalf("read args failed: %v", readErr)
	}
	argv := string(data)
	if !strings.Contains(argv, "install\n") {
		t.Fatalf("expected install command, got %q", argv)
	}
	if !strings.Contains(argv, "web-search-pro\n") {
		t.Fatalf("expected skill id, got %q", argv)
	}
	if !strings.Contains(argv, "--version\n") || !strings.Contains(argv, "1.2.3\n") {
		t.Fatalf("expected version args, got %q", argv)
	}
	if !strings.Contains(argv, "--force\n") {
		t.Fatalf("expected --force option, got %q", argv)
	}
	if !strings.Contains(argv, "--workdir\n") || !strings.Contains(argv, appRoot+"\n") {
		t.Fatalf("expected global app root workdir, got %q", argv)
	}
}

func TestClassifyClawHubCommandError(t *testing.T) {
	t.Parallel()

	rateLimited := classifyClawHubCommandError(
		[]string{"install", "file-search"},
		"- Resolving file-search\n✖ Rate limit exceeded\nError: Rate limit exceeded",
	)
	if rateLimited == nil {
		t.Fatalf("expected classified rate limited error")
	}
	if !errors.Is(rateLimited, ErrClawHubRateLimited) {
		t.Fatalf("expected ErrClawHubRateLimited, got %v", rateLimited)
	}
	rateDetail, ok := ExtractClawHubErrorDetail(rateLimited)
	if !ok {
		t.Fatalf("expected error detail")
	}
	if rateDetail.Code != ClawHubErrorCodeRateLimited {
		t.Fatalf("unexpected rate detail code: %#v", rateDetail)
	}

	requiresForce := classifyClawHubCommandError(
		[]string{"install", "web-search-pro"},
		"Error: Use --force to install suspicious skills in non-interactive mode",
	)
	if requiresForce == nil {
		t.Fatalf("expected classified require-force error")
	}
	if !errors.Is(requiresForce, ErrClawHubRequireForce) {
		t.Fatalf("expected ErrClawHubRequireForce, got %v", requiresForce)
	}
	forceDetail, ok := ExtractClawHubErrorDetail(requiresForce)
	if !ok {
		t.Fatalf("expected force error detail")
	}
	if forceDetail.Code != ClawHubErrorCodeRequireForce {
		t.Fatalf("unexpected force detail code: %#v", forceDetail)
	}
}

func TestInstallSkillEmitsRealtimeStartedAndCompletedEvents(t *testing.T) {
	t.Parallel()

	svc := NewSkillsService(newMemorySkillsRepo(), nil)
	svc.SetPackageAdapter(&skillsPackageAdapterStub{})
	fixedNow := time.Date(2026, time.March, 17, 12, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return fixedNow }

	workspaceRoot := filepath.Join(t.TempDir(), "workspaces", "assistant-1")
	events := make([]SkillsRealtimeEvent, 0, 2)
	svc.SetRealtimeNotifier(func(_ context.Context, event SkillsRealtimeEvent) {
		events = append(events, event)
	})

	err := svc.InstallSkill(context.Background(), dto.InstallSkillRequest{
		Skill:         "web-search-pro",
		Version:       "1.2.3",
		Force:         true,
		AssistantID:   "assistant-1",
		WorkspaceRoot: workspaceRoot,
	})
	if err != nil {
		t.Fatalf("install failed: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 realtime events, got %d (%#v)", len(events), events)
	}
	if events[0].Action != "install" || events[0].Stage != "started" {
		t.Fatalf("unexpected started event %#v", events[0])
	}
	if events[1].Action != "install" || events[1].Stage != "completed" {
		t.Fatalf("unexpected completed event %#v", events[1])
	}
	for _, event := range events {
		if event.Skill != "web-search-pro" {
			t.Fatalf("unexpected skill in event %#v", event)
		}
		if event.Version != "1.2.3" {
			t.Fatalf("unexpected version in event %#v", event)
		}
		if !event.Force {
			t.Fatalf("expected force=true in event %#v", event)
		}
		if event.AssistantID != "assistant-1" {
			t.Fatalf("unexpected assistant id in event %#v", event)
		}
		if event.WorkspaceRoot != workspaceRoot {
			t.Fatalf("unexpected workspace root in event %#v", event)
		}
		if !event.Timestamp.Equal(fixedNow) {
			t.Fatalf("unexpected timestamp in event %#v", event)
		}
	}
}

func TestInstallSkillEmitsRealtimeFailedEvent(t *testing.T) {
	t.Parallel()

	svc := NewSkillsService(newMemorySkillsRepo(), nil)
	svc.SetPackageAdapter(&skillsPackageAdapterStub{
		run: func(_ context.Context, _ string, _ time.Duration, _ ...string) ([]byte, error) {
			return nil, errors.New("install failed")
		},
	})
	fixedNow := time.Date(2026, time.March, 17, 12, 0, 1, 0, time.UTC)
	svc.now = func() time.Time { return fixedNow }

	events := make([]SkillsRealtimeEvent, 0, 2)
	svc.SetRealtimeNotifier(func(_ context.Context, event SkillsRealtimeEvent) {
		events = append(events, event)
	})

	err := svc.InstallSkill(context.Background(), dto.InstallSkillRequest{Skill: "web-search-pro"})
	if err == nil {
		t.Fatalf("expected install error")
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 realtime events, got %d (%#v)", len(events), events)
	}
	if events[0].Action != "install" || events[0].Stage != "started" {
		t.Fatalf("unexpected started event %#v", events[0])
	}
	if events[1].Action != "install" || events[1].Stage != "failed" {
		t.Fatalf("unexpected failed event %#v", events[1])
	}
	if strings.TrimSpace(events[1].Error) == "" {
		t.Fatalf("expected failed event error, got %#v", events[1])
	}
	if !events[1].Timestamp.Equal(fixedNow) {
		t.Fatalf("unexpected failed event timestamp %#v", events[1])
	}
}
