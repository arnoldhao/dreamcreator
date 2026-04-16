package llmrecordrepo

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"dreamcreator/internal/application/llmrecord"
	"dreamcreator/internal/infrastructure/persistence"
)

func TestSQLiteRepository_ListFiltersByThreadAndStatus(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "llm_call_records.db")
	database, err := persistence.OpenSQLite(ctx, persistence.SQLiteConfig{Path: dbPath})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer database.Close()

	repo := NewSQLiteRepository(database.Bun)
	base := time.Date(2026, 4, 16, 12, 0, 0, 0, time.UTC)
	records := []llmrecord.Record{
		{
			ID:                 "call-1",
			ThreadID:           "thread-1",
			RunID:              "run-1",
			ProviderID:         "openai",
			ModelName:          "gpt-5",
			RequestSource:      "dialogue",
			Operation:          "runtime.run",
			Status:             "completed",
			RequestPayloadJSON: `{"model":"gpt-5"}`,
			StartedAt:          base,
			FinishedAt:         base.Add(250 * time.Millisecond),
			DurationMS:         250,
		},
		{
			ID:                 "call-2",
			ThreadID:           "thread-2",
			RunID:              "run-2",
			ProviderID:         "openai",
			ModelName:          "gpt-5-mini",
			RequestSource:      "memory",
			Operation:          "memory.summary",
			Status:             "error",
			RequestPayloadJSON: `{"model":"gpt-5-mini"}`,
			StartedAt:          base.Add(2 * time.Second),
		},
	}
	for _, record := range records {
		if err := repo.Insert(ctx, record); err != nil {
			t.Fatalf("insert record %s: %v", record.ID, err)
		}
	}

	items, err := repo.List(ctx, llmrecord.QueryFilter{
		ThreadID: "thread-1",
		Status:   "completed",
		Limit:    20,
	})
	if err != nil {
		t.Fatalf("list records: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 record, got %d", len(items))
	}
	if items[0].ID != "call-1" {
		t.Fatalf("expected call-1, got %s", items[0].ID)
	}

	stored, err := repo.Get(ctx, "call-1")
	if err != nil {
		t.Fatalf("get record: %v", err)
	}
	if stored.RequestPayloadJSON == "" {
		t.Fatal("expected request payload to be stored")
	}

	stored.Status = "completed"
	stored.FinishReason = "stop"
	stored.InputTokens = 123
	stored.OutputTokens = 45
	stored.TotalTokens = 168
	stored.ResponsePayloadJSON = `{"id":"resp-1"}`
	if err := repo.Update(ctx, stored); err != nil {
		t.Fatalf("update record: %v", err)
	}

	updated, err := repo.Get(ctx, "call-1")
	if err != nil {
		t.Fatalf("get updated record: %v", err)
	}
	if updated.TotalTokens != 168 {
		t.Fatalf("expected total tokens 168, got %d", updated.TotalTokens)
	}
	if updated.ResponsePayloadJSON == "" {
		t.Fatal("expected response payload to be stored")
	}
}
