package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"dreamcreator/internal/application/skills/dto"
)

type skillsPackageAdapterStub struct {
	run func(ctx context.Context, workspaceRoot string, timeout time.Duration, args ...string) ([]byte, error)
}

func (stub *skillsPackageAdapterStub) Run(
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

func TestSkillsMetricsTrackInstallAttempts(t *testing.T) {
	t.Parallel()

	svc := NewSkillsService(newMemorySkillsRepo(), nil)
	sequence := 0
	svc.SetPackageAdapter(&skillsPackageAdapterStub{
		run: func(_ context.Context, _ string, _ time.Duration, args ...string) ([]byte, error) {
			if len(args) > 0 && args[0] == "install" {
				sequence++
				if sequence == 2 {
					return nil, errors.New("install failed")
				}
			}
			return []byte("ok"), nil
		},
	})

	if err := svc.InstallSkill(context.Background(), dto.InstallSkillRequest{Skill: "demo-a"}); err != nil {
		t.Fatalf("first install should succeed: %v", err)
	}
	if err := svc.InstallSkill(context.Background(), dto.InstallSkillRequest{Skill: "demo-b"}); err == nil {
		t.Fatalf("second install should fail")
	}
	if err := svc.UpdateSkill(context.Background(), dto.UpdateSkillRequest{Skill: "demo-a"}); err != nil {
		t.Fatalf("update should succeed: %v", err)
	}

	snapshot := svc.GetMetricsSnapshot()
	if snapshot.InstallAttempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", snapshot.InstallAttempts)
	}
	if snapshot.InstallSuccess != 2 {
		t.Fatalf("expected 2 success, got %d", snapshot.InstallSuccess)
	}
	if snapshot.InstallFailed != 1 {
		t.Fatalf("expected 1 failed, got %d", snapshot.InstallFailed)
	}
}

func TestSkillsMetricsTrackPromptDiscoveryAndTruncation(t *testing.T) {
	t.Parallel()

	appRoot := t.TempDir()
	workspaceRoot := filepath.Join(appRoot, "workspaces", "assistant-1")
	skillsRoot := filepath.Join(appRoot, "skills")
	if err := os.MkdirAll(workspaceRoot, 0o755); err != nil {
		t.Fatalf("create workspace root: %v", err)
	}
	if err := os.MkdirAll(skillsRoot, 0o755); err != nil {
		t.Fatalf("create skills root: %v", err)
	}
	for index := 0; index < 151; index++ {
		id := fmt.Sprintf("skill-%03d", index)
		dir := filepath.Join(skillsRoot, id)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("create skill dir: %v", err)
		}
		content := fmt.Sprintf("---\nid: %s\nname: %s\ncommands:\n  - %s\n---\n# %s\n", id, id, id, id)
		if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
			t.Fatalf("write skill file: %v", err)
		}
	}

	svc := NewSkillsService(newMemorySkillsRepo(), nil)
	response, err := svc.ResolveSkillPromptItemsForWorkspace(
		context.Background(),
		dto.ResolveSkillPromptRequest{ProviderID: "workspace"},
		workspaceRoot,
	)
	if err != nil {
		t.Fatalf("resolve skill prompt items failed: %v", err)
	}
	if len(response.Items) != 150 {
		t.Fatalf("expected 150 prompt items after truncation, got %d", len(response.Items))
	}

	snapshot := svc.GetMetricsSnapshot()
	if snapshot.DiscoverTotal < 151 {
		t.Fatalf("expected discover total >= 151, got %d", snapshot.DiscoverTotal)
	}
	if snapshot.EligibleTotal < 151 {
		t.Fatalf("expected eligible total >= 151, got %d", snapshot.EligibleTotal)
	}
	if snapshot.PromptTruncatedTotal < 1 {
		t.Fatalf("expected prompt truncation counter >= 1, got %d", snapshot.PromptTruncatedTotal)
	}
}
