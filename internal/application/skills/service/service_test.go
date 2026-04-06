package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"dreamcreator/internal/application/skills/dto"
	domainSkills "dreamcreator/internal/domain/skills"
)

func TestRegisterAndResolveSkills(t *testing.T) {
	t.Parallel()
	repo := newMemorySkillsRepo()
	svc := NewSkillsService(repo, nil)

	registered, err := svc.RegisterSkill(context.Background(), dto.RegisterSkillRequest{
		Spec: dto.ProviderSkillSpec{
			ID:          "find-skills",
			ProviderID:  "openai",
			Name:        "Find Skills",
			Description: "Natural language skills search",
			Enabled:     true,
		},
	})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}
	if registered.ID != "find-skills" {
		t.Fatalf("unexpected id %q", registered.ID)
	}

	resolved, err := svc.ResolveSkillsForProvider(context.Background(), dto.ResolveSkillsRequest{
		ProviderID: "openai",
	})
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	item, ok := findSkillByID(resolved, "find-skills")
	if !ok {
		t.Fatalf("expected find-skills in resolved list, got %d items", len(resolved))
	}
	if item.Description != "Natural language skills search" {
		t.Fatalf("unexpected description %q", item.Description)
	}
}

func TestEnableSkillUpdatesState(t *testing.T) {
	t.Parallel()
	repo := newMemorySkillsRepo()
	svc := NewSkillsService(repo, nil)

	_, err := svc.RegisterSkill(context.Background(), dto.RegisterSkillRequest{
		Spec: dto.ProviderSkillSpec{
			ID:         "find-skills",
			ProviderID: "openai",
			Name:       "Find Skills",
			Enabled:    true,
		},
	})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}
	if err := svc.EnableSkill(context.Background(), dto.EnableSkillRequest{ID: "find-skills", Enabled: false}); err != nil {
		t.Fatalf("enable failed: %v", err)
	}

	resolved, err := svc.ResolveSkillsForProvider(context.Background(), dto.ResolveSkillsRequest{
		ProviderID: "openai",
	})
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	item, ok := findSkillByID(resolved, "find-skills")
	if !ok {
		t.Fatalf("expected find-skills in resolved list, got %d items", len(resolved))
	}
	if item.Enabled {
		t.Fatalf("expected disabled skill")
	}
}

func TestResolveSkillsSkipsOrphanPackageManagedSettingsEntries(t *testing.T) {
	t.Parallel()

	repo := newMemorySkillsRepo()
	svc := NewSkillsService(repo, nil)

	_, err := svc.RegisterSkill(context.Background(), dto.RegisterSkillRequest{
		Spec: dto.ProviderSkillSpec{
			ID:         "yt-dlp-downloader",
			ProviderID: "workspace",
			Name:       "yt-dlp-downloader",
			Version:    "1.0.0",
			Enabled:    true,
		},
	})
	if err != nil {
		t.Fatalf("register workspace skill failed: %v", err)
	}
	_, err = svc.RegisterSkill(context.Background(), dto.RegisterSkillRequest{
		Spec: dto.ProviderSkillSpec{
			ID:         "find-skills",
			ProviderID: "openai",
			Name:       "Find Skills",
			Enabled:    true,
		},
	})
	if err != nil {
		t.Fatalf("register openai skill failed: %v", err)
	}

	appRoot := t.TempDir()
	workspaceRoot := filepath.Join(appRoot, "workspaces", "assistant-1")
	if mkErr := os.MkdirAll(workspaceRoot, 0o755); mkErr != nil {
		t.Fatalf("create workspace root failed: %v", mkErr)
	}

	resolved, err := svc.ResolveSkillsForProviderInWorkspace(context.Background(), dto.ResolveSkillsRequest{}, workspaceRoot)
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	if _, ok := findSkillByID(resolved, "yt-dlp-downloader"); ok {
		t.Fatalf("expected orphan workspace skill to be filtered, got %#v", resolved)
	}
	if _, ok := findSkillByID(resolved, "find-skills"); !ok {
		t.Fatalf("expected non-package managed setting skill to remain, got %#v", resolved)
	}
}

func findSkillByID(items []dto.ProviderSkillSpec, id string) (dto.ProviderSkillSpec, bool) {
	for _, item := range items {
		if item.ID == id {
			return item, true
		}
	}
	return dto.ProviderSkillSpec{}, false
}

type memorySkillsRepo struct {
	items map[string]domainSkills.ProviderSkillSpec
}

func newMemorySkillsRepo() *memorySkillsRepo {
	return &memorySkillsRepo{items: map[string]domainSkills.ProviderSkillSpec{}}
}

func (repo *memorySkillsRepo) ListByProvider(_ context.Context, providerID string) ([]domainSkills.ProviderSkillSpec, error) {
	result := make([]domainSkills.ProviderSkillSpec, 0, len(repo.items))
	for _, item := range repo.items {
		if providerID != "" && item.ProviderID != providerID {
			continue
		}
		result = append(result, item)
	}
	return result, nil
}

func (repo *memorySkillsRepo) Get(_ context.Context, id string) (domainSkills.ProviderSkillSpec, error) {
	item, ok := repo.items[id]
	if !ok {
		return domainSkills.ProviderSkillSpec{}, domainSkills.ErrSkillNotFound
	}
	return item, nil
}

func (repo *memorySkillsRepo) Save(_ context.Context, spec domainSkills.ProviderSkillSpec) error {
	if spec.CreatedAt.IsZero() {
		spec.CreatedAt = time.Now()
	}
	if spec.UpdatedAt.IsZero() {
		spec.UpdatedAt = spec.CreatedAt
	}
	repo.items[spec.ID] = spec
	return nil
}

func (repo *memorySkillsRepo) Delete(_ context.Context, id string) error {
	delete(repo.items, id)
	return nil
}
