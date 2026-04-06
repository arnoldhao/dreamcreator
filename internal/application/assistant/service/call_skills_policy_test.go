package service

import (
	"context"
	"testing"
	"time"

	"dreamcreator/internal/application/assistant/dto"
	domainassistant "dreamcreator/internal/domain/assistant"
)

type assistantRepoForCallPolicy struct {
	items map[string]domainassistant.Assistant
}

func newAssistantRepoForCallPolicy(item domainassistant.Assistant) *assistantRepoForCallPolicy {
	return &assistantRepoForCallPolicy{items: map[string]domainassistant.Assistant{item.ID: item}}
}

func (repo *assistantRepoForCallPolicy) List(_ context.Context, _ bool) ([]domainassistant.Assistant, error) {
	result := make([]domainassistant.Assistant, 0, len(repo.items))
	for _, item := range repo.items {
		result = append(result, item)
	}
	return result, nil
}

func (repo *assistantRepoForCallPolicy) Get(_ context.Context, id string) (domainassistant.Assistant, error) {
	item, ok := repo.items[id]
	if !ok {
		return domainassistant.Assistant{}, domainassistant.ErrAssistantNotFound
	}
	return item, nil
}

func (repo *assistantRepoForCallPolicy) Save(_ context.Context, item domainassistant.Assistant) error {
	repo.items[item.ID] = item
	return nil
}

func (repo *assistantRepoForCallPolicy) Delete(_ context.Context, id string) error {
	delete(repo.items, id)
	return nil
}

func (repo *assistantRepoForCallPolicy) SetDefault(_ context.Context, id string) error {
	for key, item := range repo.items {
		item.IsDefault = key == id
		repo.items[key] = item
	}
	return nil
}

func TestUpdateAssistantPreservesCallSkillsAllowList(t *testing.T) {
	t.Parallel()

	now := time.Now()
	initial, err := domainassistant.NewAssistant(domainassistant.AssistantParams{
		ID:        "assistant-1",
		CreatedAt: &now,
		UpdatedAt: &now,
	})
	if err != nil {
		t.Fatalf("new assistant failed: %v", err)
	}

	repo := newAssistantRepoForCallPolicy(initial)
	service := NewAssistantService(repo, nil, nil)

	updated, err := service.UpdateAssistant(context.Background(), dto.UpdateAssistantRequest{
		ID: "assistant-1",
		Call: &domainassistant.AssistantCall{
			Skills: domainassistant.CallSkillsConfig{
				Mode:      domainassistant.CallModeCustom,
				AllowList: []string{"skill-a", " skill-b "},
			},
		},
	})
	if err != nil {
		t.Fatalf("update assistant failed: %v", err)
	}
	if updated.Call.Skills.Mode != domainassistant.CallModeCustom {
		t.Fatalf("expected skills mode custom, got %q", updated.Call.Skills.Mode)
	}
	if len(updated.Call.Skills.AllowList) != 2 {
		t.Fatalf("unexpected allowList: %#v", updated.Call.Skills.AllowList)
	}

	reloaded, err := service.GetAssistant(context.Background(), "assistant-1")
	if err != nil {
		t.Fatalf("get assistant failed: %v", err)
	}
	if len(reloaded.Call.Skills.AllowList) != 2 {
		t.Fatalf("allowList not persisted: %#v", reloaded.Call.Skills.AllowList)
	}
}
