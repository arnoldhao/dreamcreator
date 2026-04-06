package service

import "testing"

func TestSpawnerSpawn(t *testing.T) {
	spawner := NewSpawner()
	record, err := spawner.Spawn(nil, SpawnRequest{ParentSessionKey: "session-1", AgentID: "agent"})
	if err != nil {
		t.Fatalf("spawn error: %v", err)
	}
	if record.RunID == "" {
		t.Fatalf("expected run id")
	}
	if record.ParentSessionKey != "session-1" {
		t.Fatalf("unexpected parent session key: %s", record.ParentSessionKey)
	}
	if record.CleanupPolicy != CleanupKeep {
		t.Fatalf("expected default cleanup keep, got %s", record.CleanupPolicy)
	}
	if record.Status != RunStatusRunning {
		t.Fatalf("expected status running, got %s", record.Status)
	}
}
