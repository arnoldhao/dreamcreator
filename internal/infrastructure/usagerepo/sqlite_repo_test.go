package usagerepo

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	gatewayusage "dreamcreator/internal/application/gateway/usage"
	"dreamcreator/internal/infrastructure/persistence"
)

func TestSQLiteUsageRepository_ListLedgerRespectsLocalWindowForUTCTimestamps(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "usage.db")
	database, err := persistence.OpenSQLite(ctx, persistence.SQLiteConfig{Path: dbPath})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer database.Close()

	repo := NewSQLiteUsageLedgerRepository(database.Bun)
	createdAt := time.Date(2026, 3, 2, 2, 43, 53, 0, time.UTC)
	event, err := repo.UpsertEvent(ctx, gatewayusage.UsageEvent{
		ID:            "event-1",
		RequestID:     "run-1",
		StepID:        "run",
		ProviderID:    "provider-custom-a",
		ModelName:     "glm-5",
		Category:      gatewayusage.CategoryTokens,
		RequestSource: gatewayusage.RequestSourceDialogue,
		UsageStatus:   "final",
		OccurredAt:    createdAt,
		CreatedAt:     createdAt,
		UpdatedAt:     createdAt,
	})
	if err != nil {
		t.Fatalf("upsert event: %v", err)
	}
	if err := repo.UpsertLedger(ctx, gatewayusage.LedgerEntry{
		ID:            "usage-1",
		EventID:       event.ID,
		Category:      gatewayusage.CategoryTokens,
		ProviderID:    "provider-custom-a",
		ModelName:     "glm-5",
		RequestID:     "run-1",
		RequestSource: gatewayusage.RequestSourceDialogue,
		CostBasis:     gatewayusage.CostBasisEstimated,
		Units:         22,
		CreatedAt:     createdAt,
	}); err != nil {
		t.Fatalf("upsert ledger: %v", err)
	}

	cst := time.FixedZone("CST", 8*3600)
	startAt := time.Date(2026, 3, 2, 9, 40, 0, 0, cst)
	endAt := time.Date(2026, 3, 2, 10, 50, 0, 0, cst)
	rows, err := repo.ListLedger(ctx, gatewayusage.QueryFilter{
		StartAt:    startAt,
		EndAt:      endAt,
		Category:   gatewayusage.CategoryTokens,
		ProviderID: "provider-custom-a",
		ModelName:  "glm-5",
	})
	if err != nil {
		t.Fatalf("list ledger: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row in local-time window, got %d", len(rows))
	}
	if rows[0].Units != 22 {
		t.Fatalf("expected units 22, got %d", rows[0].Units)
	}
}

func TestSQLiteUsageRepository_ResolvePricingVersionByTimeWindow(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "pricing.db")
	database, err := persistence.OpenSQLite(ctx, persistence.SQLiteConfig{Path: dbPath})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer database.Close()

	repo := NewSQLiteUsageLedgerRepository(database.Bun)
	version, err := repo.UpsertPricingVersion(ctx, gatewayusage.PricingVersion{
		ID:               "pricing-1",
		ProviderID:       "provider-custom-a",
		ModelName:        "glm-5",
		Currency:         "USD",
		InputPerMillion:  2,
		OutputPerMillion: 4,
		PerRequest:       0,
		Source:           "manual",
		EffectiveFrom:    time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		IsActive:         true,
		CreatedAt:        time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:        time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("upsert pricing version: %v", err)
	}
	if version.ID == "" {
		t.Fatal("expected pricing version id")
	}

	resolved, ok, err := repo.ResolvePricingVersion(ctx, "provider-custom-a", "glm-5", time.Date(2026, 3, 7, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("resolve pricing version: %v", err)
	}
	if !ok {
		t.Fatal("expected pricing version to be resolved")
	}
	if resolved.ID != version.ID {
		t.Fatalf("expected pricing id %q, got %q", version.ID, resolved.ID)
	}
}
