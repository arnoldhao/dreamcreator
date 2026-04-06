package skillsrepo

import (
	"context"
	"strings"
	"sync"

	domainSettings "dreamcreator/internal/domain/settings"
	domainSkills "dreamcreator/internal/domain/skills"
)

type SettingsRepository struct {
	settings domainSettings.Repository
	mu       sync.Mutex
}

func NewSettingsRepository(settingsRepo domainSettings.Repository) *SettingsRepository {
	return &SettingsRepository{settings: settingsRepo}
}

func (repo *SettingsRepository) ListByProvider(ctx context.Context, providerID string) ([]domainSkills.ProviderSkillSpec, error) {
	current, err := repo.settings.Get(ctx)
	if err != nil {
		return nil, err
	}
	filterProvider := strings.TrimSpace(providerID)
	items := current.Skills()
	result := make([]domainSkills.ProviderSkillSpec, 0, len(items))
	for _, item := range items {
		if filterProvider != "" && strings.TrimSpace(item.ProviderID) != filterProvider {
			continue
		}
		mapped, mapErr := mapSettingSkillToDomain(item)
		if mapErr != nil {
			continue
		}
		result = append(result, mapped)
	}
	return result, nil
}

func (repo *SettingsRepository) Get(ctx context.Context, id string) (domainSkills.ProviderSkillSpec, error) {
	targetID := strings.TrimSpace(id)
	if targetID == "" {
		return domainSkills.ProviderSkillSpec{}, domainSkills.ErrInvalidSkill
	}
	current, err := repo.settings.Get(ctx)
	if err != nil {
		return domainSkills.ProviderSkillSpec{}, err
	}
	for _, item := range current.Skills() {
		if strings.TrimSpace(item.ID) != targetID {
			continue
		}
		mapped, mapErr := mapSettingSkillToDomain(item)
		if mapErr != nil {
			return domainSkills.ProviderSkillSpec{}, domainSkills.ErrSkillNotFound
		}
		return mapped, nil
	}
	return domainSkills.ProviderSkillSpec{}, domainSkills.ErrSkillNotFound
}

func (repo *SettingsRepository) Save(ctx context.Context, spec domainSkills.ProviderSkillSpec) error {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	current, err := repo.settings.Get(ctx)
	if err != nil {
		return err
	}

	next := current.Skills()
	updated := false
	for idx := range next {
		if strings.TrimSpace(next[idx].ID) != spec.ID {
			continue
		}
		next[idx] = mapDomainSkillToSetting(spec)
		updated = true
		break
	}
	if !updated {
		next = append(next, mapDomainSkillToSetting(spec))
	}

	return repo.settings.Save(ctx, current.WithSkills(next))
}

func (repo *SettingsRepository) Delete(ctx context.Context, id string) error {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	targetID := strings.TrimSpace(id)
	if targetID == "" {
		return nil
	}

	current, err := repo.settings.Get(ctx)
	if err != nil {
		return err
	}

	origin := current.Skills()
	if len(origin) == 0 {
		return nil
	}

	next := make([]domainSettings.SkillSpec, 0, len(origin))
	for _, item := range origin {
		if strings.TrimSpace(item.ID) == targetID {
			continue
		}
		next = append(next, item)
	}
	if len(next) == len(origin) {
		return nil
	}
	return repo.settings.Save(ctx, current.WithSkills(next))
}

func mapSettingSkillToDomain(spec domainSettings.SkillSpec) (domainSkills.ProviderSkillSpec, error) {
	return domainSkills.NewProviderSkillSpec(domainSkills.ProviderSkillSpecParams{
		ID:          spec.ID,
		ProviderID:  spec.ProviderID,
		Name:        spec.Name,
		Description: spec.Description,
		Version:     spec.Version,
		Enabled:     spec.Enabled,
	})
}

func mapDomainSkillToSetting(spec domainSkills.ProviderSkillSpec) domainSettings.SkillSpec {
	return domainSettings.SkillSpec{
		ID:          spec.ID,
		ProviderID:  spec.ProviderID,
		Name:        spec.Name,
		Description: spec.Description,
		Version:     spec.Version,
		Enabled:     spec.Enabled,
	}
}
