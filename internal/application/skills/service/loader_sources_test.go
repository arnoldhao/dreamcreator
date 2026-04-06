package service

import (
	"path/filepath"
	"testing"
)

func TestResolveSkillsLocalSourcesStructuredSortedAndEnabled(t *testing.T) {
	t.Parallel()

	raw := map[string]any{
		"sources": map[string]any{
			"local": []any{
				map[string]any{"id": "source-b", "name": "Source B", "type": "extra", "path": "/skills/source-b", "priority": 30, "enabled": true},
				map[string]any{"id": "source-a", "name": "Source A", "type": "extra", "path": "/skills/source-a", "priority": 20, "enabled": true},
				map[string]any{"id": "ignored", "type": "managed", "path": "/skills/ignored", "priority": 10, "enabled": true},
			},
		},
	}

	items := resolveSkillsLocalSources(raw)
	if len(items) != 3 {
		t.Fatalf("expected workspace builtin plus 2 extra sources, got %d (%v)", len(items), items)
	}
	if items[0].ID != "workspace" || items[0].Type != "workspace" || !items[0].Enabled {
		t.Fatalf("expected workspace builtin source first, got %#v", items[0])
	}
	if items[1].Path != "/skills/source-a" || items[2].Path != "/skills/source-b" {
		t.Fatalf("expected extra sources sorted by priority, got %#v", items)
	}
}

func TestResolveSkillsLocalSourcesInvalidSchemaFallsBackToDefault(t *testing.T) {
	t.Parallel()

	raw := map[string]any{
		"sources": []any{
			map[string]any{"path": "/skills/source-a", "priority": 10},
		},
	}

	items := resolveSkillsLocalSources(raw)
	if len(items) != 1 {
		t.Fatalf("expected only builtin workspace source, got %d (%v)", len(items), items)
	}
	if items[0].ID != "workspace" || items[0].Type != "workspace" {
		t.Fatalf("expected workspace builtin source, got %#v", items[0])
	}
}

func TestResolveSkillsLocalSourcesDefaultWhenEmpty(t *testing.T) {
	t.Parallel()

	items := resolveSkillsLocalSources(nil)
	if len(items) != 1 {
		t.Fatalf("expected only builtin workspace source, got %d (%v)", len(items), items)
	}
	if items[0].ID != "workspace" || items[0].Type != "workspace" || !items[0].Enabled {
		t.Fatalf("expected workspace builtin source, got %#v", items[0])
	}
}

func TestResolveExtraSkillRootPathRelativeContainment(t *testing.T) {
	t.Parallel()

	workspaceRoot := t.TempDir()
	if resolved := resolveExtraSkillRootPath(workspaceRoot, "../outside"); resolved != "" {
		t.Fatalf("expected relative path escape to be blocked, got %q", resolved)
	}

	resolved := resolveExtraSkillRootPath(workspaceRoot, "skills-extra")
	expected := filepath.Join(workspaceRoot, "skills-extra")
	if resolved != expected {
		t.Fatalf("expected %q, got %q", expected, resolved)
	}
}

func TestResolveDreamCreatorSkillsRootUsesSiblingOfWorkspacesDirectory(t *testing.T) {
	t.Parallel()

	appRoot := t.TempDir()
	workspaceRoot := filepath.Join(appRoot, "workspaces", "assistant-1")
	expected := filepath.Join(appRoot, "skills")

	resolved := resolveDreamCreatorSkillsRoot(workspaceRoot)
	if resolved != expected {
		t.Fatalf("expected %q, got %q", expected, resolved)
	}
}
