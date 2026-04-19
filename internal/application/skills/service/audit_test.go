package service

import (
	"testing"
	"time"
)

func TestResolveSkillsAuditRetentionDays(t *testing.T) {
	value := resolveSkillsAuditRetentionDays(map[string]any{
		"auditConfig": map[string]any{
			"retentionDays": 28,
		},
	})
	if value != 28 {
		t.Fatalf("expected retentionDays=28, got %d", value)
	}

	fallback := resolveSkillsAuditRetentionDays(map[string]any{})
	if fallback != defaultSkillsAuditRetentionDays {
		t.Fatalf("expected default retentionDays=%d, got %d", defaultSkillsAuditRetentionDays, fallback)
	}
}

func TestResolveSkillsAuditHideUIOperationRecords(t *testing.T) {
	enabled := resolveSkillsAuditHideUIOperationRecords(map[string]any{
		"auditConfig": map[string]any{
			"hideUiOperationRecords": true,
		},
	})
	if !enabled {
		t.Fatal("expected hideUiOperationRecords=true")
	}

	disabled := resolveSkillsAuditHideUIOperationRecords(map[string]any{
		"auditConfig": map[string]any{},
	})
	if !disabled {
		t.Fatal("expected hideUiOperationRecords default true")
	}
}

func TestPruneSkillsAuditEntriesByRetention(t *testing.T) {
	now := time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC)
	entries := []any{
		map[string]any{
			"action":    "skills.status",
			"timestamp": now.AddDate(0, 0, -1).Format(time.RFC3339),
		},
		map[string]any{
			"action":    "skills_manage.search",
			"timestamp": now.AddDate(0, 0, -30).Format(time.RFC3339),
		},
	}

	pruned := pruneSkillsAuditEntriesByRetention(entries, 14, now)
	if len(pruned) != 1 {
		t.Fatalf("expected 1 entry after prune, got %d", len(pruned))
	}
	record, ok := pruned[0].(map[string]any)
	if !ok {
		t.Fatalf("expected map entry, got %#v", pruned[0])
	}
	if record["action"] != "skills.status" {
		t.Fatalf("expected newest entry to remain, got %#v", record)
	}
}
