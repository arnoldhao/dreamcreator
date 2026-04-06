package usage

import (
	"context"
	"testing"
	"time"
)

type memoryRepo struct {
	events   []UsageEvent
	entries  []LedgerEntry
	pricings []PricingVersion
}

func (repo *memoryRepo) UpsertEvent(_ context.Context, event UsageEvent) (UsageEvent, error) {
	for index, item := range repo.events {
		if item.RequestID == event.RequestID && item.StepID == event.StepID && item.ProviderID == event.ProviderID && item.ModelName == event.ModelName {
			event.ID = item.ID
			repo.events[index] = event
			return event, nil
		}
	}
	repo.events = append(repo.events, event)
	return event, nil
}

func (repo *memoryRepo) UpsertLedger(_ context.Context, entry LedgerEntry) error {
	for index, item := range repo.entries {
		if item.EventID == entry.EventID && item.PricingVersionID == entry.PricingVersionID && item.CostBasis == entry.CostBasis && item.Category == entry.Category {
			repo.entries[index] = entry
			return nil
		}
	}
	repo.entries = append(repo.entries, entry)
	return nil
}

func (repo *memoryRepo) ListLedger(_ context.Context, filter QueryFilter) ([]LedgerEntry, error) {
	result := make([]LedgerEntry, 0, len(repo.entries))
	for _, entry := range repo.entries {
		if !filter.StartAt.IsZero() && entry.CreatedAt.Before(filter.StartAt) {
			continue
		}
		if !filter.EndAt.IsZero() && entry.CreatedAt.After(filter.EndAt) {
			continue
		}
		if filter.ProviderID != "" && entry.ProviderID != filter.ProviderID {
			continue
		}
		if filter.ModelName != "" && entry.ModelName != filter.ModelName {
			continue
		}
		if filter.Channel != "" && entry.Channel != filter.Channel {
			continue
		}
		if filter.Category != "" && entry.Category != filter.Category {
			continue
		}
		if filter.RequestSource != "" && entry.RequestSource != filter.RequestSource {
			continue
		}
		if filter.CostBasis != "" && entry.CostBasis != filter.CostBasis {
			continue
		}
		result = append(result, entry)
	}
	return result, nil
}

func (repo *memoryRepo) ResolvePricingVersion(_ context.Context, providerID string, modelName string, at time.Time) (PricingVersion, bool, error) {
	for _, pricing := range repo.pricings {
		if pricing.ProviderID != providerID || pricing.ModelName != modelName || !pricing.IsActive {
			continue
		}
		if pricing.EffectiveFrom.After(at) {
			continue
		}
		if pricing.EffectiveTo != nil && !pricing.EffectiveTo.After(at) {
			continue
		}
		return pricing, true, nil
	}
	return PricingVersion{}, false, nil
}

func (repo *memoryRepo) ListPricingVersions(_ context.Context, filter PricingVersionFilter) ([]PricingVersion, error) {
	result := make([]PricingVersion, 0, len(repo.pricings))
	for _, pricing := range repo.pricings {
		if filter.ProviderID != "" && pricing.ProviderID != filter.ProviderID {
			continue
		}
		if filter.ModelName != "" && pricing.ModelName != filter.ModelName {
			continue
		}
		if filter.Source != "" && pricing.Source != filter.Source {
			continue
		}
		if filter.ActiveOnly && !pricing.IsActive {
			continue
		}
		result = append(result, pricing)
	}
	return result, nil
}

func (repo *memoryRepo) UpsertPricingVersion(_ context.Context, version PricingVersion) (PricingVersion, error) {
	for index, item := range repo.pricings {
		if item.ID == version.ID {
			repo.pricings[index] = version
			return version, nil
		}
	}
	repo.pricings = append(repo.pricings, version)
	return version, nil
}

func (repo *memoryRepo) DeletePricingVersion(_ context.Context, id string) error {
	next := make([]PricingVersion, 0, len(repo.pricings))
	for _, item := range repo.pricings {
		if item.ID == id {
			continue
		}
		next = append(next, item)
	}
	repo.pricings = next
	return nil
}

func (repo *memoryRepo) ActivatePricingVersion(_ context.Context, id string) error {
	var providerID, modelName string
	for _, item := range repo.pricings {
		if item.ID == id {
			providerID = item.ProviderID
			modelName = item.ModelName
			break
		}
	}
	for index, item := range repo.pricings {
		if item.ProviderID == providerID && item.ModelName == modelName {
			repo.pricings[index].IsActive = item.ID == id
		}
	}
	return nil
}

func TestIngestComputesTokenCostFromPricingVersion(t *testing.T) {
	now := time.Date(2026, 3, 7, 8, 0, 0, 0, time.UTC)
	repo := &memoryRepo{
		pricings: []PricingVersion{{
			ID:               "pricing-1",
			ProviderID:       "provider-a",
			ModelName:        "glm-5",
			InputPerMillion:  2,
			OutputPerMillion: 4,
			PerRequest:       0,
			IsActive:         true,
			EffectiveFrom:    now.Add(-time.Hour),
		}},
	}
	service := NewService(repo)
	service.now = func() time.Time { return now }
	service.newID = func() string { return "fixed-id" }

	if err := service.Ingest(context.Background(), LedgerEntry{
		Category:      CategoryTokens,
		ProviderID:    "provider-a",
		ModelName:     "glm-5",
		RequestID:     "run-1",
		RequestSource: RequestSourceDialogue,
		InputTokens:   100,
		OutputTokens:  50,
		Units:         150,
	}); err != nil {
		t.Fatalf("unexpected ingest error: %v", err)
	}

	if len(repo.entries) != 1 {
		t.Fatalf("expected 1 ledger entry, got %d", len(repo.entries))
	}
	entry := repo.entries[0]
	if entry.CostMicros != 400 {
		t.Fatalf("expected cost micros 400, got %d", entry.CostMicros)
	}
	if entry.PricingVersionID != "pricing-1" {
		t.Fatalf("expected pricing version id pricing-1, got %q", entry.PricingVersionID)
	}
}

func TestUsageStatusGroupsInputOutputTokens(t *testing.T) {
	now := time.Date(2026, 3, 7, 12, 0, 0, 0, time.UTC)
	repo := &memoryRepo{
		entries: []LedgerEntry{
			{ID: "1", Category: CategoryTokens, ProviderID: "openai", ModelName: "gpt-4o", RequestSource: RequestSourceDialogue, Units: 10, InputTokens: 8, OutputTokens: 2, CostMicros: 100, CreatedAt: now.Add(-time.Hour)},
			{ID: "2", Category: CategoryTokens, ProviderID: "openai", ModelName: "gpt-4o", RequestSource: RequestSourceDialogue, Units: 5, InputTokens: 4, OutputTokens: 1, CostMicros: 50, CreatedAt: now.Add(-30 * time.Minute)},
			{ID: "3", Category: CategoryTTS, ProviderID: "edge", ModelName: "tts", RequestSource: RequestSourceRelay, Units: 2, InputTokens: 1, OutputTokens: 1, CostMicros: 20, CreatedAt: now.Add(-10 * time.Minute)},
		},
	}
	service := NewService(repo)
	service.now = func() time.Time { return now }

	resp, err := service.Status(context.Background(), UsageStatusRequest{
		Window:  "24h",
		GroupBy: []string{"provider", "model"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Totals.Requests != 3 {
		t.Fatalf("expected 3 requests, got %d", resp.Totals.Requests)
	}
	if resp.Totals.InputTokens != 13 {
		t.Fatalf("expected 13 input tokens, got %d", resp.Totals.InputTokens)
	}
	if resp.Totals.OutputTokens != 4 {
		t.Fatalf("expected 4 output tokens, got %d", resp.Totals.OutputTokens)
	}
	if len(resp.Buckets) != 2 {
		t.Fatalf("expected 2 buckets, got %d", len(resp.Buckets))
	}
}

func TestUsageStatusFiltersByRequestSourceAlias(t *testing.T) {
	now := time.Date(2026, 3, 7, 12, 0, 0, 0, time.UTC)
	repo := &memoryRepo{
		entries: []LedgerEntry{
			{ID: "1", Category: CategoryTokens, ProviderID: "provider-a", ModelName: "glm-5", RequestSource: RequestSourceOneShot, Units: 8, CreatedAt: now.Add(-time.Hour)},
			{ID: "2", Category: CategoryTokens, ProviderID: "provider-a", ModelName: "glm-5", RequestSource: RequestSourceRelay, Units: 3, CreatedAt: now.Add(-30 * time.Minute)},
		},
	}
	service := NewService(repo)
	service.now = func() time.Time { return now }

	resp, err := service.Status(context.Background(), UsageStatusRequest{
		Window:        "24h",
		RequestSource: "oneshot",
		GroupBy:       []string{"request-source"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Totals.Requests != 1 {
		t.Fatalf("expected 1 request, got %d", resp.Totals.Requests)
	}
	if len(resp.Buckets) != 1 || resp.Buckets[0].RequestSource != RequestSourceOneShot {
		t.Fatalf("unexpected bucket response: %+v", resp.Buckets)
	}
}
