package providersync

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"dreamcreator/internal/domain/providers"
)

type modelsDevCatalogRepoStub struct {
	entries []ModelsDevCatalogEntry
}

func (repo *modelsDevCatalogRepoStub) Count(_ context.Context) (int, error) {
	return len(repo.entries), nil
}

func (repo *modelsDevCatalogRepoStub) ReplaceAll(_ context.Context, entries []ModelsDevCatalogEntry) error {
	repo.entries = append([]ModelsDevCatalogEntry(nil), entries...)
	return nil
}

func (repo *modelsDevCatalogRepoStub) ListByProviderKeys(_ context.Context, providerKeys []string) ([]ModelsDevCatalogEntry, error) {
	targets := normalizeCatalogStrings(providerKeys)
	allowed := make(map[string]struct{}, len(targets))
	for _, key := range targets {
		allowed[key] = struct{}{}
	}
	result := make([]ModelsDevCatalogEntry, 0)
	for _, entry := range repo.entries {
		if _, ok := allowed[entry.ProviderKey]; ok {
			result = append(result, entry)
		}
	}
	return result, nil
}

func (repo *modelsDevCatalogRepoStub) ListByModelNames(_ context.Context, modelNames []string) ([]ModelsDevCatalogEntry, error) {
	targets := normalizeCatalogStrings(modelNames)
	allowed := make(map[string]struct{}, len(targets))
	for _, name := range targets {
		allowed[name] = struct{}{}
	}
	result := make([]ModelsDevCatalogEntry, 0)
	for _, entry := range repo.entries {
		if _, ok := allowed[strings.ToLower(entry.ModelName)]; ok {
			result = append(result, entry)
		}
	}
	return result, nil
}

func TestModelsDevCatalogServiceRefreshAndResolve(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{
  "openai": {
    "models": {
      "gpt-5.4": {"id":"gpt-5.4","name":"GPT-5.4","limit":{"context":400000,"output":128000},"supports":{"reasoning":true}},
      "gpt-4.1": {"id":"gpt-4.1","name":"GPT-4.1","limit":{"context":128000,"output":8192}}
    }
  },
  "openrouter": {
    "models": {
      "gpt-5.4": {"id":"gpt-5.4","name":"OpenRouter GPT-5.4","pricing":{"input":1.25,"output":10.00}}
    }
  }
}`))
	}))
	defer server.Close()

	now := time.Date(2026, 4, 17, 10, 0, 0, 0, time.UTC)
	repo := &modelsDevCatalogRepoStub{}
	service := NewModelsDevCatalogService(repo, &ModelsDevSyncer{
		apiURL:     server.URL,
		httpClient: server.Client(),
		now:        func() time.Time { return now },
	})
	service.now = func() time.Time { return now }

	count, err := service.Refresh(context.Background())
	if err != nil {
		t.Fatalf("refresh catalog: %v", err)
	}
	if count != 3 {
		t.Fatalf("expected 3 catalog entries, got %d", count)
	}

	hasEntries, err := service.HasEntries(context.Background())
	if err != nil {
		t.Fatalf("has entries: %v", err)
	}
	if !hasEntries {
		t.Fatal("expected catalog to have entries after refresh")
	}

	displayNames, err := service.ResolveModelDisplayNames(context.Background(), []string{"gpt-5.4"})
	if err != nil {
		t.Fatalf("resolve display names: %v", err)
	}
	if got := displayNames["gpt-5.4"]; got != "OpenRouter GPT-5.4" {
		t.Fatalf("expected preferred display name OpenRouter GPT-5.4, got %q", got)
	}

	providerKey, err := service.ResolveProviderKeyByModelIDs(context.Background(), []string{"gpt-5.4"})
	if err != nil {
		t.Fatalf("resolve provider key: %v", err)
	}
	if providerKey != "openrouter" {
		t.Fatalf("expected openrouter to win on pricing score, got %q", providerKey)
	}

	models, err := service.Sync(context.Background(), providers.Provider{ID: "openai", Name: "OpenAI"}, "")
	if err != nil {
		t.Fatalf("sync by provider candidates: %v", err)
	}
	if len(models) != 2 {
		t.Fatalf("expected 2 models for openai catalog sync, got %d", len(models))
	}
	if models[0].ProviderID != "openai" {
		t.Fatalf("expected synced provider id openai, got %q", models[0].ProviderID)
	}
}
