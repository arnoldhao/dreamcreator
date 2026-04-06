package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadSkillsFromRootSkipsOversizedSkillFile(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	limitedDir := filepath.Join(root, "limited")
	oversizedDir := filepath.Join(root, "oversized")
	if err := os.MkdirAll(limitedDir, 0o755); err != nil {
		t.Fatalf("mkdir limited failed: %v", err)
	}
	if err := os.MkdirAll(oversizedDir, 0o755); err != nil {
		t.Fatalf("mkdir oversized failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(limitedDir, "SKILL.md"), []byte("---\nid: limited-skill\nname: Limited Skill\n---\n# Limited Skill\n"), 0o644); err != nil {
		t.Fatalf("write limited skill failed: %v", err)
	}
	oversizedContent := "---\nid: oversized-skill\nname: Oversized Skill\n---\n" + strings.Repeat("x", 1024)
	if err := os.WriteFile(filepath.Join(oversizedDir, "SKILL.md"), []byte(oversizedContent), 0o644); err != nil {
		t.Fatalf("write oversized skill failed: %v", err)
	}

	items := loadSkillsFromRoot(skillRoot{
		Path:       root,
		Source:     "workspace",
		SourceID:   "workspace",
		SourceName: "Workspace",
		SourceKind: "local",
		SourceType: "workspace",
	}, 100, 100, 256)
	if len(items) != 1 {
		t.Fatalf("expected 1 skill after max file bytes filter, got %d (%#v)", len(items), items)
	}
	if items[0].ID != "limited-skill" {
		t.Fatalf("expected limited-skill, got %#v", items[0])
	}
}

func TestLoadSkillsFromRootSkipsOversizedRootSkillFile(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	content := "---\nid: root-skill\nname: Root Skill\n---\n" + strings.Repeat("x", 2048)
	if err := os.WriteFile(filepath.Join(root, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write root skill failed: %v", err)
	}

	items := loadSkillsFromRoot(skillRoot{
		Path:       root,
		Source:     "workspace",
		SourceID:   "workspace",
		SourceName: "Workspace",
		SourceKind: "local",
		SourceType: "workspace",
	}, 100, 100, 128)
	if len(items) != 0 {
		t.Fatalf("expected oversized root skill to be skipped, got %d (%#v)", len(items), items)
	}
}
