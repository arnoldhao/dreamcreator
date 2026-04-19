package tools

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	externaltoolsdto "dreamcreator/internal/application/externaltools/dto"
	settingsdto "dreamcreator/internal/application/settings/dto"
	skillsservice "dreamcreator/internal/application/skills/service"
	workspaceDTO "dreamcreator/internal/application/workspace/dto"
	"dreamcreator/internal/domain/externaltools"
	domainskills "dreamcreator/internal/domain/skills"
)

type skillsRepoStub struct {
	items map[string]domainskills.ProviderSkillSpec
}

func newSkillsRepoStub() *skillsRepoStub {
	return &skillsRepoStub{items: make(map[string]domainskills.ProviderSkillSpec)}
}

func (repo *skillsRepoStub) ListByProvider(_ context.Context, providerID string) ([]domainskills.ProviderSkillSpec, error) {
	result := make([]domainskills.ProviderSkillSpec, 0, len(repo.items))
	for _, item := range repo.items {
		if providerID != "" && item.ProviderID != providerID {
			continue
		}
		result = append(result, item)
	}
	return result, nil
}

func (repo *skillsRepoStub) Get(_ context.Context, id string) (domainskills.ProviderSkillSpec, error) {
	item, ok := repo.items[id]
	if !ok {
		return domainskills.ProviderSkillSpec{}, domainskills.ErrSkillNotFound
	}
	return item, nil
}

func (repo *skillsRepoStub) Save(_ context.Context, spec domainskills.ProviderSkillSpec) error {
	if spec.CreatedAt.IsZero() {
		spec.CreatedAt = time.Now()
	}
	if spec.UpdatedAt.IsZero() {
		spec.UpdatedAt = spec.CreatedAt
	}
	repo.items[spec.ID] = spec
	return nil
}

func (repo *skillsRepoStub) Delete(_ context.Context, id string) error {
	delete(repo.items, id)
	return nil
}

type skillsExternalToolsStubForGateway struct {
	ready    bool
	execPath string
	err      error
}

func (stub *skillsExternalToolsStubForGateway) IsToolReady(_ context.Context, _ externaltools.ToolName) (bool, error) {
	return stub.ready, stub.err
}

func (stub *skillsExternalToolsStubForGateway) ResolveExecPath(_ context.Context, _ externaltools.ToolName) (string, error) {
	if stub.err != nil {
		return "", stub.err
	}
	if !stub.ready {
		return "", errors.New("not ready")
	}
	return stub.execPath, nil
}

func (stub *skillsExternalToolsStubForGateway) ToolReadiness(_ context.Context, _ externaltools.ToolName) (bool, string, error) {
	if stub.err != nil {
		return false, "", stub.err
	}
	if !stub.ready {
		return false, "not_installed", nil
	}
	return true, "", nil
}

type skillsPackageAdapterStubForGateway struct {
	run func(ctx context.Context, workspaceRoot string, timeout time.Duration, args ...string) ([]byte, error)
}

func (stub *skillsPackageAdapterStubForGateway) Run(
	ctx context.Context,
	workspaceRoot string,
	timeout time.Duration,
	args ...string,
) ([]byte, error) {
	if stub == nil || stub.run == nil {
		return []byte("ok"), nil
	}
	return stub.run(ctx, workspaceRoot, timeout, args...)
}

type skillsDepsInstallerStub struct {
	mu         sync.Mutex
	ready      map[externaltools.ToolName]bool
	installErr error
}

func newSkillsDepsInstallerStub() *skillsDepsInstallerStub {
	return &skillsDepsInstallerStub{ready: make(map[externaltools.ToolName]bool)}
}

func (stub *skillsDepsInstallerStub) InstallTool(_ context.Context, request externaltoolsdto.InstallExternalToolRequest) (externaltoolsdto.ExternalTool, error) {
	if stub.installErr != nil {
		return externaltoolsdto.ExternalTool{}, stub.installErr
	}
	name := externaltools.ToolName(strings.TrimSpace(request.Name))
	stub.mu.Lock()
	stub.ready[name] = true
	stub.mu.Unlock()
	return externaltoolsdto.ExternalTool{Name: request.Name, Status: string(externaltools.StatusInstalled)}, nil
}

func (stub *skillsDepsInstallerStub) ToolReadiness(_ context.Context, name externaltools.ToolName) (bool, string, error) {
	stub.mu.Lock()
	defer stub.mu.Unlock()
	if stub.ready[name] {
		return true, "", nil
	}
	return false, "not_installed", nil
}

type skillsWorkspaceResolverStubForGateway struct{}

func (skillsWorkspaceResolverStubForGateway) GetAssistantWorkspaceDirectory(_ context.Context, assistantID string) (workspaceDTO.AssistantWorkspaceDirectory, error) {
	return workspaceDTO.AssistantWorkspaceDirectory{
		AssistantID: assistantID,
		WorkspaceID: assistantID,
		RootPath:    "",
	}, nil
}

type skillsSettingsStub struct {
	mu      sync.Mutex
	current settingsdto.Settings
}

func newSkillsSettingsStub(tools map[string]any) *skillsSettingsStub {
	parsedTools := cloneAnyMapDeep(tools)
	if parsedTools == nil {
		parsedTools = map[string]any{}
	}
	skillsConfig := map[string]any{}
	if existing, ok := parsedTools["skills"].(map[string]any); ok {
		skillsConfig = cloneAnyMapDeep(existing)
	}
	delete(parsedTools, "skills")
	if len(parsedTools) == 0 {
		parsedTools = nil
	}
	if len(skillsConfig) == 0 {
		skillsConfig = nil
	}
	return &skillsSettingsStub{
		current: settingsdto.Settings{
			Tools:  cloneAnyMapDeep(parsedTools),
			Skills: cloneAnyMapDeep(skillsConfig),
		},
	}
}

func (stub *skillsSettingsStub) GetSettings(_ context.Context) (settingsdto.Settings, error) {
	stub.mu.Lock()
	defer stub.mu.Unlock()
	copied := stub.current
	copied.Tools = cloneAnyMapDeep(stub.current.Tools)
	copied.Skills = cloneAnyMapDeep(stub.current.Skills)
	return copied, nil
}

func (stub *skillsSettingsStub) UpdateSettings(_ context.Context, request settingsdto.UpdateSettingsRequest) (settingsdto.Settings, error) {
	stub.mu.Lock()
	defer stub.mu.Unlock()
	if request.Tools != nil {
		stub.current.Tools = cloneAnyMapDeep(request.Tools)
	}
	if request.Skills != nil {
		stub.current.Skills = cloneAnyMapDeep(request.Skills)
	}
	copied := stub.current
	copied.Tools = cloneAnyMapDeep(stub.current.Tools)
	copied.Skills = cloneAnyMapDeep(stub.current.Skills)
	return copied, nil
}

func TestSkillsManageToolSearchReturnsClawHubUnavailable(t *testing.T) {
	t.Parallel()

	repo := newSkillsRepoStub()
	svc := skillsservice.NewSkillsService(repo, nil)
	svc.SetExternalTools(&skillsExternalToolsStubForGateway{ready: false})
	svc.SetWorkspaceResolver(skillsWorkspaceResolverStubForGateway{})

	handler := runSkillsManageTool(svc, nil, nil, nil)
	output, err := handler(context.Background(), `{"action":"search","query":"demo"}`)
	if err != nil {
		t.Fatalf("skills_manage tool failed: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("unmarshal output failed: %v", err)
	}
	if payload["ok"] != false {
		t.Fatalf("expected ok=false, got %#v", payload["ok"])
	}
	if payload["error"] != clawhubUnavailableReason {
		t.Fatalf("expected clawhub_unavailable, got %#v", payload["error"])
	}
}

func TestSkillsManageToolInstallRequireForceAndRetrySuccess(t *testing.T) {
	t.Parallel()

	repo := newSkillsRepoStub()
	svc := skillsservice.NewSkillsService(repo, nil)
	svc.SetExternalTools(&skillsExternalToolsStubForGateway{ready: true, execPath: "/usr/bin/true"})
	svc.SetWorkspaceResolver(skillsWorkspaceResolverStubForGateway{})
	svc.SetPackageAdapter(&skillsPackageAdapterStubForGateway{
		run: func(_ context.Context, _ string, _ time.Duration, args ...string) ([]byte, error) {
			for _, arg := range args {
				if arg == "--force" {
					return []byte("ok"), nil
				}
			}
			return nil, &skillsservice.ClawHubCommandError{
				Command: "install web-search-pro",
				Message: "Use --force to install suspicious skills in non-interactive mode",
				Code:    skillsservice.ClawHubErrorCodeRequireForce,
				Hint:    "retry_with_force",
			}
		},
	})
	handler := runSkillsManageTool(svc, nil, nil, nil)

	output, err := handler(context.Background(), `{"action":"install","skill":"web-search-pro"}`)
	if err != nil {
		t.Fatalf("skills_manage tool install failed: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("unmarshal output failed: %v", err)
	}
	if payload["ok"] != false {
		t.Fatalf("expected ok=false, got %#v", payload["ok"])
	}
	if payload["errorCode"] != skillsservice.ClawHubErrorCodeRequireForce {
		t.Fatalf("expected requires_force error code, got %#v", payload["errorCode"])
	}
	if payload["requiresForce"] != true {
		t.Fatalf("expected requiresForce=true, got %#v", payload["requiresForce"])
	}
	if payload["action"] != "skills_manage.install" {
		t.Fatalf("expected action skills_manage.install, got %#v", payload["action"])
	}

	output, err = handler(context.Background(), `{"action":"install","skill":"web-search-pro","force":true}`)
	if err != nil {
		t.Fatalf("skills_manage tool install(force) failed: %v", err)
	}
	payload = map[string]any{}
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("unmarshal output failed: %v", err)
	}
	if payload["ok"] != true {
		t.Fatalf("expected ok=true after force, got %#v", payload["ok"])
	}
}

func TestSkillsManageToolInstallRequireForceBlockedByScannerPolicy(t *testing.T) {
	t.Parallel()

	repo := newSkillsRepoStub()
	svc := skillsservice.NewSkillsService(repo, nil)
	svc.SetExternalTools(&skillsExternalToolsStubForGateway{ready: true, execPath: "/usr/bin/true"})
	svc.SetWorkspaceResolver(skillsWorkspaceResolverStubForGateway{})
	svc.SetPackageAdapter(&skillsPackageAdapterStubForGateway{
		run: func(_ context.Context, _ string, _ time.Duration, _ ...string) ([]byte, error) {
			return nil, &skillsservice.ClawHubCommandError{
				Command: "install web-search-pro",
				Message: "Use --force to install suspicious skills in non-interactive mode",
				Code:    skillsservice.ClawHubErrorCodeRequireForce,
				Hint:    "retry_with_force",
			}
		},
	})
	settings := newSkillsSettingsStub(map[string]any{
		"skills": map[string]any{
			"security": map[string]any{
				"install": map[string]any{
					"scannerMode":       "block",
					"allowForceInstall": true,
					"requireApproval":   false,
				},
			},
		},
	})
	handler := runSkillsManageTool(svc, nil, settings, nil)

	output, err := handler(context.Background(), `{"action":"install","skill":"web-search-pro"}`)
	if err != nil {
		t.Fatalf("skills_manage tool install failed: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("unmarshal output failed: %v", err)
	}
	if payload["ok"] != false {
		t.Fatalf("expected ok=false, got %#v", payload["ok"])
	}
	if payload["errorCode"] != skillsToolErrorPolicyDenied {
		t.Fatalf("expected policy_denied, got %#v", payload["errorCode"])
	}
}

func TestSkillsManageToolForceInstallRequiresApprovalWhenEnabled(t *testing.T) {
	t.Parallel()

	repo := newSkillsRepoStub()
	svc := skillsservice.NewSkillsService(repo, nil)
	svc.SetExternalTools(&skillsExternalToolsStubForGateway{ready: true, execPath: "/usr/bin/true"})
	svc.SetWorkspaceResolver(skillsWorkspaceResolverStubForGateway{})
	settings := newSkillsSettingsStub(map[string]any{
		"skills": map[string]any{
			"security": map[string]any{
				"install": map[string]any{
					"scannerMode":       "warn",
					"allowForceInstall": true,
					"requireApproval":   true,
				},
			},
		},
	})
	handler := runSkillsManageTool(svc, nil, settings, nil)

	output, err := handler(context.Background(), `{"action":"install","skill":"web-search-pro","force":true}`)
	if err != nil {
		t.Fatalf("skills_manage tool install failed: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("unmarshal output failed: %v", err)
	}
	if payload["ok"] != false {
		t.Fatalf("expected ok=false, got %#v", payload["ok"])
	}
	if payload["errorCode"] != skillsToolErrorApprovalRequired {
		t.Fatalf("expected approval_required, got %#v", payload["errorCode"])
	}
}

func TestSkillsToolInstallCanonicalized(t *testing.T) {
	t.Parallel()

	repo := newSkillsRepoStub()
	svc := skillsservice.NewSkillsService(repo, nil)
	svc.SetExternalTools(&skillsExternalToolsStubForGateway{ready: false})
	svc.SetWorkspaceResolver(skillsWorkspaceResolverStubForGateway{})

	handler := runSkillsTool(svc, nil, nil, nil)
	output, err := handler(context.Background(), `{"action":"install","skill":"skill-a"}`)
	if err != nil {
		t.Fatalf("skills tool failed: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("unmarshal output failed: %v", err)
	}
	if payload["action"] != "skills.install" {
		t.Fatalf("expected canonical action skills.install, got %#v", payload["action"])
	}
	if payload["error"] != clawhubUnavailableReason {
		t.Fatalf("expected clawhub_unavailable, got %#v", payload["error"])
	}
}

func TestSkillsToolRejectsUnknownField(t *testing.T) {
	t.Parallel()

	repo := newSkillsRepoStub()
	svc := skillsservice.NewSkillsService(repo, nil)
	handler := runSkillsTool(svc, nil, nil, nil)

	output, err := handler(context.Background(), `{"action":"status","unexpected":1}`)
	if err != nil {
		t.Fatalf("skills tool failed: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("unmarshal output failed: %v", err)
	}
	if payload["ok"] != false {
		t.Fatalf("expected ok=false, got %#v", payload["ok"])
	}
	if payload["errorCode"] != skillsToolErrorInvalidArgs {
		t.Fatalf("expected invalid_arguments, got %#v", payload["errorCode"])
	}
}

func TestSkillsToolRejectsUnsupportedAction(t *testing.T) {
	t.Parallel()

	repo := newSkillsRepoStub()
	svc := skillsservice.NewSkillsService(repo, nil)
	handler := runSkillsTool(svc, nil, nil, nil)

	output, err := handler(context.Background(), `{"action":"search"}`)
	if err != nil {
		t.Fatalf("skills tool failed: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("unmarshal output failed: %v", err)
	}
	if payload["errorCode"] != skillsToolErrorUnsupported {
		t.Fatalf("expected unsupported_action, got %#v", payload["errorCode"])
	}
}

func TestSkillsToolRejectsLegacyAliasAction(t *testing.T) {
	t.Parallel()

	repo := newSkillsRepoStub()
	svc := skillsservice.NewSkillsService(repo, nil)
	handler := runSkillsTool(svc, nil, nil, nil)

	output, err := handler(context.Background(), `{"action":"install_deps","skill":"demo-skill"}`)
	if err != nil {
		t.Fatalf("skills tool failed: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("unmarshal output failed: %v", err)
	}
	if payload["errorCode"] != skillsToolErrorUnsupported {
		t.Fatalf("expected unsupported_action, got %#v", payload["errorCode"])
	}
}

func TestSkillsToolActionModeDeny(t *testing.T) {
	t.Parallel()

	repo := newSkillsRepoStub()
	svc := skillsservice.NewSkillsService(repo, nil)
	settings := newSkillsSettingsStub(map[string]any{
		"skills": map[string]any{
			"security": map[string]any{
				"actionModes": map[string]any{
					"config_write": "deny",
				},
			},
		},
	})
	handler := runSkillsTool(svc, nil, settings, nil)

	output, err := handler(context.Background(), `{"action":"update","skill":"skill-a","enabled":false}`)
	if err != nil {
		t.Fatalf("skills tool failed: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("unmarshal output failed: %v", err)
	}
	if payload["errorCode"] != skillsToolErrorPolicyDenied {
		t.Fatalf("expected policy_denied, got %#v", payload["errorCode"])
	}
}

func TestSkillsManageToolActionModeAsk(t *testing.T) {
	t.Parallel()

	repo := newSkillsRepoStub()
	svc := skillsservice.NewSkillsService(repo, nil)
	settings := newSkillsSettingsStub(map[string]any{
		"skills": map[string]any{
			"security": map[string]any{
				"actionModes": map[string]any{
					"package_write": "ask",
				},
			},
		},
	})
	handler := runSkillsManageTool(svc, nil, settings, nil)

	output, err := handler(context.Background(), `{"action":"install","skill":"skill-a"}`)
	if err != nil {
		t.Fatalf("skills_manage tool failed: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("unmarshal output failed: %v", err)
	}
	if payload["errorCode"] != skillsToolErrorApprovalRequired {
		t.Fatalf("expected approval_required, got %#v", payload["errorCode"])
	}
}

func TestSkillsToolUpdatePersistsEntry(t *testing.T) {
	t.Parallel()

	repo := newSkillsRepoStub()
	svc := skillsservice.NewSkillsService(repo, nil)
	settings := newSkillsSettingsStub(nil)
	handler := runSkillsTool(svc, nil, settings, nil)

	output, err := handler(context.Background(), `{"action":"update","skill":"skill-a","enabled":false,"apiKey":"demo","env":{"TOKEN":"abc"},"config":{"timeout":30}}`)
	if err != nil {
		t.Fatalf("skills tool failed: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("unmarshal output failed: %v", err)
	}
	if payload["ok"] != true {
		t.Fatalf("expected ok=true, got %#v payload=%v output=%s", payload["ok"], payload, output)
	}

	latest, err := settings.GetSettings(context.Background())
	if err != nil {
		t.Fatalf("get settings failed: %v", err)
	}
	skillsConfig := cloneAnyMapDeep(latest.Skills)
	entries, _ := skillsConfig["entries"].(map[string]any)
	entry, _ := entries["skill-a"].(map[string]any)
	if entry["apiKey"] != "demo" {
		t.Fatalf("expected apiKey=demo, got %#v", entry["apiKey"])
	}
	env, _ := entry["env"].(map[string]any)
	if env["TOKEN"] != "abc" {
		t.Fatalf("expected env TOKEN=abc, got %#v", env["TOKEN"])
	}
	config, _ := entry["config"].(map[string]any)
	if config["timeout"] != float64(30) {
		t.Fatalf("expected config.timeout=30, got %#v", config["timeout"])
	}
}

func TestSkillsToolInstallViaExternalTools(t *testing.T) {
	t.Parallel()

	workdir := t.TempDir()
	scriptPath := filepath.Join(workdir, "clawhub")
	script := strings.Join([]string{
		"#!/bin/sh",
		"cmd=\"\"",
		"for arg in \"$@\"; do",
		"  if [ \"$arg\" = \"inspect\" ] || [ \"$arg\" = \"list\" ]; then",
		"    cmd=\"$arg\"",
		"    break",
		"  fi",
		"done",
		"if [ \"$cmd\" = \"inspect\" ]; then",
		"  has_file=false",
		"  for arg in \"$@\"; do",
		"    if [ \"$arg\" = \"--file\" ]; then",
		"      has_file=true",
		"      break",
		"    fi",
		"  done",
		"  if [ \"$has_file\" = \"true\" ]; then",
		"    cat <<'EOF'",
		"{\"file\":{\"path\":\"SKILL.md\",\"content\":\"---\\nmetadata:\\n  clawdbot:\\n    requires:\\n      bins: [\\\"ffmpeg\\\"]\\n    install:\\n      - kind: brew\\n        formula: ffmpeg\\n        bins: [\\\"ffmpeg\\\"]\\n---\\n# Demo\\n\"}}",
		"EOF",
		"    exit 0",
		"  fi",
		"  cat <<'EOF'",
		"{\"skill\":{\"slug\":\"demo-skill\",\"displayName\":\"Demo Skill\"},\"version\":{\"version\":\"1.0.0\",\"files\":[{\"path\":\"SKILL.md\"}]}}",
		"EOF",
		"  exit 0",
		"fi",
		"if [ \"$cmd\" = \"list\" ]; then",
		"  echo \"demo-skill 1.0.0\"",
		"  exit 0",
		"fi",
		"echo \"[]\"",
		"exit 0",
	}, "\n")
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write script failed: %v", err)
	}

	repo := newSkillsRepoStub()
	svc := skillsservice.NewSkillsService(repo, nil)
	svc.SetExternalTools(&skillsExternalToolsStubForGateway{ready: true, execPath: scriptPath})
	svc.SetWorkspaceResolver(skillsWorkspaceResolverStubForGateway{})

	depsInstaller := newSkillsDepsInstallerStub()
	handler := runSkillsTool(svc, nil, nil, depsInstaller)

	output, err := handler(context.Background(), `{"action":"install","skill":"demo-skill","timeoutMs":2000}`)
	if err != nil {
		t.Fatalf("skills tool failed: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("unmarshal output failed: %v", err)
	}
	if payload["ok"] != true {
		t.Fatalf("expected ok=true, got %#v payload=%v output=%s", payload["ok"], payload, output)
	}
	data, _ := payload["data"].(map[string]any)
	successBins, _ := data["successBins"].([]any)
	if len(successBins) == 0 {
		t.Fatalf("expected success bins, got %#v", data["successBins"])
	}
	if payload["action"] != "skills.install" {
		t.Fatalf("expected action skills.install, got %#v", payload["action"])
	}
}

func cloneAnyMapDeep(source map[string]any) map[string]any {
	if len(source) == 0 {
		return nil
	}
	data, err := json.Marshal(source)
	if err != nil {
		return cloneAnyMap(source)
	}
	var target map[string]any
	if err := json.Unmarshal(data, &target); err != nil {
		return cloneAnyMap(source)
	}
	return target
}
