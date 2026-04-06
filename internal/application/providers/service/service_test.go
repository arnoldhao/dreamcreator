package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"dreamcreator/internal/application/providers/dto"
	"dreamcreator/internal/domain/providers"
)

var errForeignKeyConstraint = errors.New("sqlite3: constraint failed: FOREIGN KEY constraint failed")

type providerRepoStub struct {
	items map[string]providers.Provider
}

func newProviderRepoStub(items ...providers.Provider) *providerRepoStub {
	repo := &providerRepoStub{items: make(map[string]providers.Provider, len(items))}
	for _, item := range items {
		repo.items[item.ID] = item
	}
	return repo
}

func (repo *providerRepoStub) List(_ context.Context) ([]providers.Provider, error) {
	result := make([]providers.Provider, 0, len(repo.items))
	for _, item := range repo.items {
		result = append(result, item)
	}
	return result, nil
}

func (repo *providerRepoStub) Get(_ context.Context, id string) (providers.Provider, error) {
	item, ok := repo.items[id]
	if !ok {
		return providers.Provider{}, providers.ErrProviderNotFound
	}
	return item, nil
}

func (repo *providerRepoStub) Save(_ context.Context, provider providers.Provider) error {
	repo.items[provider.ID] = provider
	return nil
}

func (repo *providerRepoStub) Delete(_ context.Context, id string) error {
	delete(repo.items, id)
	return nil
}

type modelRepoStub struct {
	providers *providerRepoStub
	items     map[string]providers.Model
}

func newModelRepoStub(providerRepo *providerRepoStub, items ...providers.Model) *modelRepoStub {
	repo := &modelRepoStub{
		providers: providerRepo,
		items:     make(map[string]providers.Model, len(items)),
	}
	for _, item := range items {
		repo.items[item.ID] = item
	}
	return repo
}

func (repo *modelRepoStub) ListByProvider(_ context.Context, providerID string) ([]providers.Model, error) {
	result := make([]providers.Model, 0)
	for _, item := range repo.items {
		if item.ProviderID == providerID {
			result = append(result, item)
		}
	}
	return result, nil
}

func (repo *modelRepoStub) Get(_ context.Context, id string) (providers.Model, error) {
	item, ok := repo.items[id]
	if !ok {
		return providers.Model{}, providers.ErrModelNotFound
	}
	return item, nil
}

func (repo *modelRepoStub) Save(_ context.Context, model providers.Model) error {
	if _, ok := repo.providers.items[model.ProviderID]; !ok {
		return errForeignKeyConstraint
	}
	repo.items[model.ID] = model
	return nil
}

func (repo *modelRepoStub) ReplaceByProvider(_ context.Context, providerID string, models []providers.Model) error {
	if _, ok := repo.providers.items[providerID]; !ok {
		return errForeignKeyConstraint
	}
	for id, item := range repo.items {
		if item.ProviderID == providerID {
			delete(repo.items, id)
		}
	}
	for _, item := range models {
		repo.items[item.ID] = item
	}
	return nil
}

func (repo *modelRepoStub) Delete(_ context.Context, id string) error {
	delete(repo.items, id)
	return nil
}

type syncerStub struct {
	models []providers.Model
}

func (stub syncerStub) Sync(_ context.Context, _ providers.Provider, _ string) ([]providers.Model, error) {
	return stub.models, nil
}

func TestUpdateProviderModelRecreatesMissingDefaultProvider(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 6, 18, 20, 0, 0, time.UTC)
	model, err := providers.NewModel(providers.ModelParams{
		ID:         "openai:gpt-4.1",
		ProviderID: "openai",
		Name:       "gpt-4.1",
		CreatedAt:  &now,
		UpdatedAt:  &now,
	})
	if err != nil {
		t.Fatalf("new model: %v", err)
	}

	providerRepo := newProviderRepoStub()
	modelRepo := newModelRepoStub(providerRepo, model)
	service := NewProvidersService(providerRepo, modelRepo, nil, nil, nil)
	service.now = func() time.Time { return now }

	updated, err := service.UpdateProviderModel(context.Background(), dto.UpdateProviderModelRequest{
		ID:         model.ID,
		ProviderID: "openai",
		Enabled:    true,
		ShowInUI:   true,
	})
	if err != nil {
		t.Fatalf("update provider model: %v", err)
	}
	if !updated.Enabled || !updated.ShowInUI {
		t.Fatalf("expected updated model enabled/showInUI to be true, got %+v", updated)
	}
	provider, err := providerRepo.Get(context.Background(), "openai")
	if err != nil {
		t.Fatalf("provider should be recreated: %v", err)
	}
	if !provider.Builtin {
		t.Fatalf("expected recreated provider to be builtin")
	}
}

func TestReplaceProviderModelsRecreatesMissingDefaultProvider(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 6, 18, 21, 0, 0, time.UTC)
	providerRepo := newProviderRepoStub()
	modelRepo := newModelRepoStub(providerRepo)
	service := NewProvidersService(providerRepo, modelRepo, nil, syncerStub{}, nil)
	service.now = func() time.Time { return now }

	err := service.ReplaceProviderModels(context.Background(), dto.ReplaceProviderModelsRequest{
		ProviderID: "openai",
		Models: []dto.ProviderModel{{
			ID:       "openai:gpt-4.1",
			Name:     "gpt-4.1",
			Enabled:  true,
			ShowInUI: true,
		}},
	})
	if err != nil {
		t.Fatalf("replace provider models: %v", err)
	}
	if _, err := providerRepo.Get(context.Background(), "openai"); err != nil {
		t.Fatalf("provider should be recreated: %v", err)
	}
	if _, err := modelRepo.Get(context.Background(), "openai:gpt-4.1"); err != nil {
		t.Fatalf("model should be saved after provider recreation: %v", err)
	}
}
