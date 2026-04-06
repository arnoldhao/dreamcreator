package service

import (
	"context"
	"errors"
	"strings"
	"time"

	settingsservice "dreamcreator/internal/application/settings/service"
	"dreamcreator/internal/application/skills/dto"
	workspaceDTO "dreamcreator/internal/application/workspace/dto"
	"dreamcreator/internal/domain/externaltools"
	domainSkills "dreamcreator/internal/domain/skills"
)

var errSkillsRepositoryUnavailable = errors.New("skills repository unavailable")
var ErrClawHubUnavailable = errors.New("clawhub_unavailable")

type ExternalToolsReadiness interface {
	IsToolReady(ctx context.Context, name externaltools.ToolName) (bool, error)
	ResolveExecPath(ctx context.Context, name externaltools.ToolName) (string, error)
	ToolReadiness(ctx context.Context, name externaltools.ToolName) (bool, string, error)
}

type WorkspaceDirectoryResolver interface {
	GetAssistantWorkspaceDirectory(ctx context.Context, assistantID string) (workspaceDTO.AssistantWorkspaceDirectory, error)
}

type SkillsService struct {
	repo             domainSkills.Repository
	settings         *settingsservice.SettingsService
	externalTools    ExternalToolsReadiness
	workspaces       WorkspaceDirectoryResolver
	packageAdapter   SkillsPackageAdapter
	metrics          *skillsMetrics
	realtimeNotifier SkillsRealtimeNotifier
	now              func() time.Time
}

func NewSkillsService(repo domainSkills.Repository, settings *settingsservice.SettingsService) *SkillsService {
	service := &SkillsService{
		repo:     repo,
		settings: settings,
		now:      time.Now,
		metrics:  newSkillsMetrics(),
	}
	service.packageAdapter = newClawHubPackageAdapter(service)
	return service
}

func (service *SkillsService) SetExternalTools(external ExternalToolsReadiness) {
	if service == nil {
		return
	}
	service.externalTools = external
}

func (service *SkillsService) SetWorkspaceResolver(workspaces WorkspaceDirectoryResolver) {
	if service == nil {
		return
	}
	service.workspaces = workspaces
}

func (service *SkillsService) SetPackageAdapter(adapter SkillsPackageAdapter) {
	if service == nil {
		return
	}
	if adapter == nil {
		service.packageAdapter = newClawHubPackageAdapter(service)
		return
	}
	service.packageAdapter = adapter
}

func (service *SkillsService) RegisterSkill(ctx context.Context, request dto.RegisterSkillRequest) (dto.ProviderSkillSpec, error) {
	if service.repo == nil {
		return dto.ProviderSkillSpec{}, errSkillsRepositoryUnavailable
	}
	normalized := normalizeSpec(request.Spec)
	item, err := domainSkills.NewProviderSkillSpec(domainSkills.ProviderSkillSpecParams{
		ID:          normalized.ID,
		ProviderID:  normalized.ProviderID,
		Name:        normalized.Name,
		Description: normalized.Description,
		Version:     normalized.Version,
		Enabled:     normalized.Enabled,
	})
	if err != nil {
		service.emitRealtimeEvent(ctx, SkillsRealtimeEvent{
			Action: "register",
			Stage:  "failed",
			Skill:  normalized.ID,
			Error:  err.Error(),
		})
		return dto.ProviderSkillSpec{}, err
	}
	if err := service.repo.Save(ctx, item); err != nil {
		service.emitRealtimeEvent(ctx, SkillsRealtimeEvent{
			Action: "register",
			Stage:  "failed",
			Skill:  normalized.ID,
			Error:  err.Error(),
		})
		return dto.ProviderSkillSpec{}, err
	}
	service.emitRealtimeEvent(ctx, SkillsRealtimeEvent{
		Action: "register",
		Stage:  "completed",
		Skill:  normalized.ID,
	})
	return toDTO(item), nil
}

func (service *SkillsService) EnableSkill(ctx context.Context, request dto.EnableSkillRequest) error {
	if service.repo == nil {
		return errSkillsRepositoryUnavailable
	}
	id := strings.TrimSpace(request.ID)
	if id == "" {
		return domainSkills.ErrInvalidSkill
	}
	item, err := service.repo.Get(ctx, id)
	if err != nil {
		service.emitRealtimeEvent(ctx, SkillsRealtimeEvent{
			Action: "enable",
			Stage:  "failed",
			Skill:  id,
			Error:  err.Error(),
		})
		return err
	}
	item.Enabled = request.Enabled
	item.UpdatedAt = service.now()
	if err := service.repo.Save(ctx, item); err != nil {
		service.emitRealtimeEvent(ctx, SkillsRealtimeEvent{
			Action: "enable",
			Stage:  "failed",
			Skill:  id,
			Error:  err.Error(),
		})
		return err
	}
	service.emitRealtimeEvent(ctx, SkillsRealtimeEvent{
		Action: "enable",
		Stage:  "completed",
		Skill:  id,
	})
	return nil
}

func (service *SkillsService) DeleteSkill(ctx context.Context, request dto.DeleteSkillRequest) error {
	if service.repo == nil {
		return errSkillsRepositoryUnavailable
	}
	id := strings.TrimSpace(request.ID)
	if id == "" {
		return domainSkills.ErrInvalidSkill
	}
	if err := service.repo.Delete(ctx, id); err != nil {
		service.emitRealtimeEvent(ctx, SkillsRealtimeEvent{
			Action: "delete",
			Stage:  "failed",
			Skill:  id,
			Error:  err.Error(),
		})
		return err
	}
	service.emitRealtimeEvent(ctx, SkillsRealtimeEvent{
		Action: "delete",
		Stage:  "completed",
		Skill:  id,
	})
	return nil
}

func (service *SkillsService) ResolveSkillsForProvider(ctx context.Context, request dto.ResolveSkillsRequest) ([]dto.ProviderSkillSpec, error) {
	return service.ResolveSkillsForProviderInWorkspace(ctx, request, "")
}

func (service *SkillsService) ResolveSkillsForProviderInWorkspace(ctx context.Context, request dto.ResolveSkillsRequest, workspaceRoot string) ([]dto.ProviderSkillSpec, error) {
	settingsItems, err := service.listProviderSkills(ctx, strings.TrimSpace(request.ProviderID))
	if err != nil {
		return nil, err
	}
	workspaceEntries := service.loadWorkspaceSkillEntries(ctx, strings.TrimSpace(workspaceRoot), nil)
	merged := mergeProviderSkills(settingsItems, workspaceEntries)
	return filterOrphanPackageManagedSkills(merged, workspaceEntries), nil
}

func (service *SkillsService) ResolveSkillPromptItems(ctx context.Context, request dto.ResolveSkillPromptRequest) (dto.ResolveSkillPromptResponse, error) {
	return service.resolveSkillPromptItems(ctx, request, "")
}

func (service *SkillsService) ResolveSkillPromptItemsForWorkspace(ctx context.Context, request dto.ResolveSkillPromptRequest, workspaceRoot string) (dto.ResolveSkillPromptResponse, error) {
	return service.resolveSkillPromptItems(ctx, request, workspaceRoot)
}

func (service *SkillsService) resolveSkillPromptItems(ctx context.Context, request dto.ResolveSkillPromptRequest, workspaceRoot string) (dto.ResolveSkillPromptResponse, error) {
	settingsItems, err := service.listProviderSkills(ctx, strings.TrimSpace(request.ProviderID))
	if err != nil {
		return dto.ResolveSkillPromptResponse{}, err
	}
	limits := service.resolveSkillsLimits(ctx)
	workspaceEntries := service.loadWorkspaceSkillEntries(ctx, workspaceRoot, &limits)
	entryConfigLookup := service.resolvePromptSkillEntryConfigs(ctx)
	enabledLookup := make(map[string]bool, len(settingsItems))
	for _, item := range settingsItems {
		enabledLookup[strings.ToLower(strings.TrimSpace(item.ID))] = item.Enabled
	}
	filteredEntries := make([]skillEntry, 0, len(workspaceEntries))
	for _, entry := range workspaceEntries {
		if entry.ID == "" {
			continue
		}
		entryConfig := resolvePromptSkillEntryConfig(entry, entryConfigLookup)
		if enabled, ok := readMapBool(entryConfig, "enabled"); ok && !enabled {
			continue
		}
		if enabled, ok := enabledLookup[strings.ToLower(entry.ID)]; ok && !enabled {
			continue
		}
		if !isSkillEntryRuntimeEligible(entry, entryConfig) {
			continue
		}
		filteredEntries = append(filteredEntries, entry)
	}
	remainingSettings := make([]domainSkills.ProviderSkillSpec, 0, len(settingsItems))
	for _, item := range settingsItems {
		entryConfig := entryConfigLookup[strings.ToLower(strings.TrimSpace(item.ID))]
		if enabled, ok := readMapBool(entryConfig, "enabled"); ok && !enabled {
			continue
		}
		if !item.Enabled {
			continue
		}
		if entryExists(filteredEntries, item.ID) {
			continue
		}
		remainingSettings = append(remainingSettings, item)
	}
	items := buildSkillPromptItems(filteredEntries, remainingSettings, workspaceRoot, limits.MaxSkillsInPrompt)
	discoveredCount := len(workspaceEntries) + len(settingsItems)
	eligibleCount := len(filteredEntries) + len(remainingSettings)
	truncated := limits.MaxSkillsInPrompt > 0 && eligibleCount > len(items)
	service.recordPromptDiscovery(discoveredCount, eligibleCount, truncated)
	return dto.ResolveSkillPromptResponse{Items: items}, nil
}

func (service *SkillsService) listProviderSkills(ctx context.Context, providerID string) ([]domainSkills.ProviderSkillSpec, error) {
	if service.repo == nil {
		return nil, nil
	}
	items, err := service.repo.ListByProvider(ctx, strings.TrimSpace(providerID))
	if err != nil {
		return nil, err
	}
	return items, nil
}

func filterOrphanPackageManagedSkills(items []dto.ProviderSkillSpec, workspaceEntries []skillEntry) []dto.ProviderSkillSpec {
	if len(items) == 0 {
		return nil
	}
	installed := make(map[string]struct{}, len(workspaceEntries))
	for _, entry := range workspaceEntries {
		key := strings.ToLower(strings.TrimSpace(entry.ID))
		if key == "" {
			continue
		}
		installed[key] = struct{}{}
	}
	filtered := make([]dto.ProviderSkillSpec, 0, len(items))
	for _, item := range items {
		key := strings.ToLower(strings.TrimSpace(item.ID))
		if key == "" {
			continue
		}
		if _, ok := installed[key]; ok {
			filtered = append(filtered, item)
			continue
		}
		if strings.TrimSpace(item.SourceKind) != "" ||
			strings.TrimSpace(item.SourceType) != "" ||
			strings.TrimSpace(item.SourcePath) != "" ||
			strings.TrimSpace(item.SourceID) != "" {
			filtered = append(filtered, item)
			continue
		}
		if isPackageManagedSkillsProvider(item.ProviderID) {
			continue
		}
		filtered = append(filtered, item)
	}
	if len(filtered) == 0 {
		return nil
	}
	return filtered
}

func isPackageManagedSkillsProvider(providerID string) bool {
	switch strings.ToLower(strings.TrimSpace(providerID)) {
	case "", "workspace", "local", "clawhub":
		return true
	default:
		return false
	}
}

func toDTOList(items []domainSkills.ProviderSkillSpec) []dto.ProviderSkillSpec {
	if len(items) == 0 {
		return nil
	}
	result := make([]dto.ProviderSkillSpec, 0, len(items))
	for _, item := range items {
		result = append(result, toDTO(item))
	}
	return result
}

func toDTO(item domainSkills.ProviderSkillSpec) dto.ProviderSkillSpec {
	return dto.ProviderSkillSpec{
		ID:          item.ID,
		ProviderID:  item.ProviderID,
		Name:        item.Name,
		Description: item.Description,
		Version:     item.Version,
		Enabled:     item.Enabled,
	}
}

func normalizeSpec(spec dto.ProviderSkillSpec) dto.ProviderSkillSpec {
	spec.ID = strings.TrimSpace(spec.ID)
	spec.ProviderID = strings.TrimSpace(spec.ProviderID)
	spec.Name = strings.TrimSpace(spec.Name)
	spec.Description = strings.TrimSpace(spec.Description)
	spec.Version = strings.TrimSpace(spec.Version)
	return spec
}
