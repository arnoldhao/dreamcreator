package runtime

import (
	"context"
	"testing"
	"time"

	"dreamcreator/internal/application/gateway/runtime/dto"
	gatewayusage "dreamcreator/internal/application/gateway/usage"
	"github.com/cloudwego/eino/schema"
)

type usageLedgerRepoSpy struct {
	events   []gatewayusage.UsageEvent
	entries  []gatewayusage.LedgerEntry
	pricings []gatewayusage.PricingVersion
}

func (repo *usageLedgerRepoSpy) UpsertEvent(_ context.Context, event gatewayusage.UsageEvent) (gatewayusage.UsageEvent, error) {
	if event.ID == "" {
		event.ID = "event-" + event.RequestID
	}
	repo.events = append(repo.events, event)
	return event, nil
}

func (repo *usageLedgerRepoSpy) UpsertLedger(_ context.Context, entry gatewayusage.LedgerEntry) error {
	repo.entries = append(repo.entries, entry)
	return nil
}

func (repo *usageLedgerRepoSpy) ListLedger(_ context.Context, _ gatewayusage.QueryFilter) ([]gatewayusage.LedgerEntry, error) {
	return append([]gatewayusage.LedgerEntry(nil), repo.entries...), nil
}

func (repo *usageLedgerRepoSpy) ResolvePricingVersion(_ context.Context, providerID string, modelName string, at time.Time) (gatewayusage.PricingVersion, bool, error) {
	for _, pricing := range repo.pricings {
		if pricing.ProviderID == providerID && pricing.ModelName == modelName && pricing.IsActive && !pricing.EffectiveFrom.After(at) {
			return pricing, true, nil
		}
	}
	return gatewayusage.PricingVersion{}, false, nil
}

func (repo *usageLedgerRepoSpy) ListPricingVersions(_ context.Context, _ gatewayusage.PricingVersionFilter) ([]gatewayusage.PricingVersion, error) {
	return append([]gatewayusage.PricingVersion(nil), repo.pricings...), nil
}

func (repo *usageLedgerRepoSpy) UpsertPricingVersion(_ context.Context, version gatewayusage.PricingVersion) (gatewayusage.PricingVersion, error) {
	repo.pricings = append(repo.pricings, version)
	return version, nil
}

func (repo *usageLedgerRepoSpy) DeletePricingVersion(_ context.Context, _ string) error {
	return nil
}

func (repo *usageLedgerRepoSpy) ActivatePricingVersion(_ context.Context, _ string) error {
	return nil
}

func TestIngestUsage_RecordsRequestForZeroTokenUsage(t *testing.T) {
	t.Parallel()

	repo := &usageLedgerRepoSpy{}
	service := &Service{
		usage: gatewayusage.NewService(repo),
	}

	service.ingestUsage(context.Background(), dto.RuntimeUsage{}, resolvedRunModel{
		ProviderID: "provider-custom-a",
		ModelName:  "custom-model",
	}, "chat", "dialogue", "run-1")

	if len(repo.entries) != 1 {
		t.Fatalf("expected 1 usage ledger entry, got %d", len(repo.entries))
	}
	entry := repo.entries[0]
	if entry.Category != gatewayusage.CategoryTokens {
		t.Fatalf("expected category=tokens, got %q", entry.Category)
	}
	if entry.ProviderID != "provider-custom-a" || entry.ModelName != "custom-model" {
		t.Fatalf("expected provider/model to be recorded, got %q/%q", entry.ProviderID, entry.ModelName)
	}
	if entry.Units != 0 || entry.CostMicros != 0 {
		t.Fatalf("expected zero units/cost for zero token usage, got units=%d cost=%d", entry.Units, entry.CostMicros)
	}
	if entry.RequestSource != "dialogue" {
		t.Fatalf("expected request source dialogue, got %q", entry.RequestSource)
	}
}

func TestIngestUsage_RecordsContextUsageAlongsideTokenRequest(t *testing.T) {
	t.Parallel()

	repo := &usageLedgerRepoSpy{}
	service := &Service{
		usage: gatewayusage.NewService(repo),
	}

	service.ingestUsage(context.Background(), dto.RuntimeUsage{
		ContextTotalTokens: 1200,
	}, resolvedRunModel{
		ProviderID: "provider-custom-a",
		ModelName:  "custom-model",
	}, "chat", "dialogue", "run-2")

	if len(repo.entries) != 2 {
		t.Fatalf("expected 2 usage ledger entries, got %d", len(repo.entries))
	}
	if repo.entries[0].Category != gatewayusage.CategoryTokens {
		t.Fatalf("expected first entry category=tokens, got %q", repo.entries[0].Category)
	}
	if repo.entries[1].Category != gatewayusage.CategoryContextToken {
		t.Fatalf("expected second entry category=context_tokens, got %q", repo.entries[1].Category)
	}
	if repo.entries[1].Units != 1200 {
		t.Fatalf("expected context tokens=1200, got %d", repo.entries[1].Units)
	}
}

func TestIngestUsage_StoresInputAndOutputTokens(t *testing.T) {
	t.Parallel()

	repo := &usageLedgerRepoSpy{}
	service := &Service{
		usage: gatewayusage.NewService(repo),
	}

	service.ingestUsage(context.Background(), dto.RuntimeUsage{
		PromptTokens:     100,
		CompletionTokens: 25,
		TotalTokens:      125,
	}, resolvedRunModel{
		ProviderID: "provider-custom-a",
		ModelName:  "custom-model",
	}, "chat", "relay", "run-3")

	if len(repo.entries) != 1 {
		t.Fatalf("expected 1 usage ledger entry, got %d", len(repo.entries))
	}
	entry := repo.entries[0]
	if entry.InputTokens != 100 || entry.OutputTokens != 25 || entry.Units != 125 {
		t.Fatalf("unexpected token split: input=%d output=%d units=%d", entry.InputTokens, entry.OutputTokens, entry.Units)
	}
	if entry.RequestSource != "relay" {
		t.Fatalf("expected request source relay, got %q", entry.RequestSource)
	}
}

func TestResolveUsageSource(t *testing.T) {
	t.Parallel()

	if got := resolveUsageSource(map[string]any{}, "aui", "one-shot"); got != usageSourceOneShot {
		t.Fatalf("expected one-shot source, got %q", got)
	}
	if got := resolveUsageSource(map[string]any{"usageSource": "relay"}, "aui", "one-shot"); got != usageSourceRelay {
		t.Fatalf("expected relay source override, got %q", got)
	}
	if got := resolveUsageSource(map[string]any{}, "aui", "user"); got != usageSourceDialogue {
		t.Fatalf("expected dialogue source, got %q", got)
	}
	if got := resolveUsageSource(map[string]any{}, "", "user"); got != usageSourceRelay {
		t.Fatalf("expected relay default for empty channel, got %q", got)
	}
}

func TestResolveLLMOperation_ForOneShotKinds(t *testing.T) {
	t.Parallel()

	if got := resolveLLMOperation("one-shot", map[string]any{"oneShotKind": "title_generation"}, false); got != "runtime.title_generation" {
		t.Fatalf("expected title generation operation, got %q", got)
	}
	if got := resolveLLMOperation("one-shot", map[string]any{"oneShotKind": "subtitle_translate"}, false); got != "runtime.subtitle_translate" {
		t.Fatalf("expected subtitle translate operation, got %q", got)
	}
	if got := resolveLLMOperation("one-shot", map[string]any{"oneShotKind": "subtitle_proofread"}, false); got != "runtime.subtitle_proofread" {
		t.Fatalf("expected subtitle proofread operation, got %q", got)
	}
	if got := resolveLLMOperation("one-shot", map[string]any{}, false); got != "runtime.one_shot" {
		t.Fatalf("expected generic one-shot operation, got %q", got)
	}
}

func TestMergeRuntimeUsage_AccumulatesStepUsage(t *testing.T) {
	t.Parallel()

	var usage dto.RuntimeUsage
	usage = mergeRuntimeUsage(usage, &schema.TokenUsage{
		PromptTokens:     10,
		CompletionTokens: 2,
		TotalTokens:      12,
	})
	usage = mergeRuntimeUsage(usage, &schema.TokenUsage{
		PromptTokens:     20,
		CompletionTokens: 3,
		TotalTokens:      23,
	})

	if usage.PromptTokens != 30 || usage.CompletionTokens != 5 || usage.TotalTokens != 35 {
		t.Fatalf("unexpected merged usage: prompt=%d completion=%d total=%d", usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens)
	}
}

func TestMergeRuntimeUsage_FallsBackToSplitTotalWhenTotalMissing(t *testing.T) {
	t.Parallel()

	var usage dto.RuntimeUsage
	usage = mergeRuntimeUsage(usage, &schema.TokenUsage{
		PromptTokens:     8,
		CompletionTokens: 4,
		TotalTokens:      0,
	})
	if usage.TotalTokens != 12 {
		t.Fatalf("expected fallback total 12, got %d", usage.TotalTokens)
	}
}
