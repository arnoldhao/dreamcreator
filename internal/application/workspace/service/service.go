package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	workspaceDTO "dreamcreator/internal/application/workspace/dto"
	assistantdomain "dreamcreator/internal/domain/assistant"
	"dreamcreator/internal/domain/workspace"
)

type WorkspaceService struct {
	repo       workspace.Repository
	fsManager  WorkspaceManager
	assistants assistantdomain.Repository
	now        func() time.Time
	newID      func() string
}

type WorkspaceManager interface {
	CreateWorkspace(ctx context.Context, workspaceID string) (string, error)
}

func NewWorkspaceService(repo workspace.Repository, fsManager WorkspaceManager, assistants assistantdomain.Repository) *WorkspaceService {
	return &WorkspaceService{
		repo:       repo,
		fsManager:  fsManager,
		assistants: assistants,
		now:        time.Now,
		newID:      uuid.NewString,
	}
}

func (service *WorkspaceService) GetGlobal(ctx context.Context) (workspaceDTO.GlobalWorkspace, error) {
	global, err := service.getOrCreateGlobal(ctx)
	if err != nil {
		return workspaceDTO.GlobalWorkspace{}, err
	}
	return toGlobalDTO(global), nil
}

func (service *WorkspaceService) UpdateGlobal(ctx context.Context, request workspaceDTO.UpdateGlobalWorkspaceRequest) (workspaceDTO.GlobalWorkspace, error) {
	global, err := service.getOrCreateGlobal(ctx)
	if err != nil {
		return workspaceDTO.GlobalWorkspace{}, err
	}

	if request.DefaultExecutorModelJSON != nil {
		global.DefaultExecutorModelJSON = strings.TrimSpace(*request.DefaultExecutorModelJSON)
	}
	if request.DefaultMemoryJSON != nil {
		global.DefaultMemoryJSON = strings.TrimSpace(*request.DefaultMemoryJSON)
	}
	if request.DefaultPersona != nil {
		global.DefaultPersona = strings.TrimSpace(*request.DefaultPersona)
	}
	global.UpdatedAt = service.now()

	updated, err := workspace.NewGlobalWorkspace(workspace.GlobalWorkspaceParams{
		ID:                       global.ID,
		DefaultExecutorModelJSON: global.DefaultExecutorModelJSON,
		DefaultMemoryJSON:        global.DefaultMemoryJSON,
		DefaultPersona:           global.DefaultPersona,
		CreatedAt:                &global.CreatedAt,
		UpdatedAt:                &global.UpdatedAt,
	})
	if err != nil {
		return workspaceDTO.GlobalWorkspace{}, err
	}

	if err := service.repo.SaveGlobal(ctx, updated); err != nil {
		return workspaceDTO.GlobalWorkspace{}, err
	}
	return toGlobalDTO(updated), nil
}

func (service *WorkspaceService) GetAssistantWorkspaceDirectory(ctx context.Context, assistantID string) (workspaceDTO.AssistantWorkspaceDirectory, error) {
	trimmed := strings.TrimSpace(assistantID)
	if trimmed == "" {
		return workspaceDTO.AssistantWorkspaceDirectory{}, errors.New("assistant id is required")
	}
	rootPath := ""
	if service.fsManager != nil {
		path, err := service.fsManager.CreateWorkspace(ctx, trimmed)
		if err != nil {
			return workspaceDTO.AssistantWorkspaceDirectory{}, err
		}
		rootPath = path
	}
	return workspaceDTO.AssistantWorkspaceDirectory{
		AssistantID: trimmed,
		WorkspaceID: trimmed,
		RootPath:    rootPath,
	}, nil
}

func (service *WorkspaceService) EnsureAssistantWorkspace(ctx context.Context, assistantID string) error {
	trimmed := strings.TrimSpace(assistantID)
	if trimmed == "" {
		return errors.New("assistant id is required")
	}
	if service.fsManager != nil {
		if _, err := service.fsManager.CreateWorkspace(ctx, trimmed); err != nil {
			return err
		}
	}
	if service.repo == nil {
		return errors.New("workspace repository unavailable")
	}
	_, err := service.ensureAssistantWorkspace(ctx, trimmed)
	return err
}

func (service *WorkspaceService) ResolveRuntimeSnapshot(ctx context.Context, request workspaceDTO.ResolveRuntimeSnapshotRequest) (workspaceDTO.RuntimeSnapshot, error) {
	assistantID := strings.TrimSpace(request.AssistantID)
	if assistantID == "" {
		return workspaceDTO.RuntimeSnapshot{}, errors.New("assistant id is required")
	}
	global, err := service.getOrCreateGlobal(ctx)
	if err != nil {
		return workspaceDTO.RuntimeSnapshot{}, err
	}

	executorModel := global.DefaultExecutorModelJSON
	memory := global.DefaultMemoryJSON
	persona := global.DefaultPersona
	workspaceContext := workspaceDTO.WorkspaceContext{PromptMode: workspaceDTO.PromptModeFull}
	if request.IncludeWorkspaceContext {
		workspaceItem, snapshot, err := service.resolveAssistantWorkspaceSnapshot(ctx, assistantID, request.ForRunID, nil)
		if err != nil {
			return workspaceDTO.RuntimeSnapshot{}, err
		}
		if value := strings.TrimSpace(snapshot.PromptModeDefault); value != "" {
			workspaceContext.PromptMode = value
		}
		if workspaceItem.Persona != "" {
			persona = workspaceItem.Persona
		}
		if workspaceItem.MemoryJSON != "" {
			memory = workspaceItem.MemoryJSON
		}
		workspaceContext.Files = toWorkspaceContextFiles(snapshot.LogicalFiles)
	}

	rootPath := ""
	if service.fsManager != nil {
		path, err := service.fsManager.CreateWorkspace(ctx, assistantID)
		if err != nil {
			return workspaceDTO.RuntimeSnapshot{}, err
		}
		rootPath = path
	}

	return workspaceDTO.RuntimeSnapshot{
		GlobalWorkspaceID: global.ID,
		AssistantID:       assistantID,
		ThreadID:          strings.TrimSpace(request.ThreadID),
		RootPath:          rootPath,
		ExecutorModelJSON: executorModel,
		MemoryJSON:        memory,
		Persona:           persona,
		WorkspaceContext:  workspaceContext,
	}, nil
}

func (service *WorkspaceService) GetAssistantWorkspace(ctx context.Context, assistantID string) (workspaceDTO.AssistantWorkspace, error) {
	assistantID = strings.TrimSpace(assistantID)
	if assistantID == "" {
		return workspaceDTO.AssistantWorkspace{}, errors.New("assistant id is required")
	}
	item, err := service.ensureAssistantWorkspace(ctx, assistantID)
	if err != nil {
		return workspaceDTO.AssistantWorkspace{}, err
	}
	return toAssistantWorkspaceDTO(item), nil
}

func (service *WorkspaceService) UpdateWorkspace(ctx context.Context, request workspaceDTO.UpdateWorkspaceRequest) (workspaceDTO.UpdateWorkspaceResponse, error) {
	assistantID := strings.TrimSpace(request.AssistantID)
	if assistantID == "" {
		return workspaceDTO.UpdateWorkspaceResponse{}, errors.New("assistant id is required")
	}
	if request.ExpectedVersion <= 0 {
		return workspaceDTO.UpdateWorkspaceResponse{}, errors.New("expected version is required")
	}
	item, err := service.ensureAssistantWorkspace(ctx, assistantID)
	if err != nil {
		return workspaceDTO.UpdateWorkspaceResponse{}, err
	}
	if item.Version != request.ExpectedVersion {
		return workspaceDTO.UpdateWorkspaceResponse{}, workspace.ErrWorkspaceVersionConflict
	}
	patch := request.Patch
	if patch.Identity != nil {
		item.Identity = *patch.Identity
	}
	if patch.Persona != nil {
		item.Persona = strings.TrimSpace(*patch.Persona)
	}
	if patch.UserProfile != nil {
		item.UserProfile = *patch.UserProfile
	}
	if patch.Tooling != nil {
		item.Tooling = *patch.Tooling
	}
	if patch.Memory != nil {
		item.Memory = *patch.Memory
	}
	if patch.MemoryJSON != nil {
		item.MemoryJSON = strings.TrimSpace(*patch.MemoryJSON)
	}
	if patch.ExtraFiles != nil {
		item.ExtraFiles = toDomainLogicalFiles(*patch.ExtraFiles)
	}
	if patch.PromptModeDefault != nil {
		item.PromptModeDefault = *patch.PromptModeDefault
	}
	item.Version++
	item.UpdatedAt = service.now()
	normalized, err := workspace.NewAssistantWorkspace(workspace.AssistantWorkspaceParams{
		AssistantID:       item.AssistantID,
		Version:           item.Version,
		Identity:          item.Identity,
		Persona:           item.Persona,
		UserProfile:       item.UserProfile,
		Tooling:           item.Tooling,
		Memory:            item.Memory,
		MemoryJSON:        item.MemoryJSON,
		ExtraFiles:        item.ExtraFiles,
		PromptModeDefault: item.PromptModeDefault,
		CreatedAt:         &item.CreatedAt,
		UpdatedAt:         &item.UpdatedAt,
	})
	if err != nil {
		return workspaceDTO.UpdateWorkspaceResponse{}, err
	}
	if service.repo == nil {
		return workspaceDTO.UpdateWorkspaceResponse{}, errors.New("workspace repository unavailable")
	}
	if err := service.repo.UpdateAssistantWorkspace(ctx, normalized, request.ExpectedVersion); err != nil {
		return workspaceDTO.UpdateWorkspaceResponse{}, err
	}
	return workspaceDTO.UpdateWorkspaceResponse{Workspace: toAssistantWorkspaceDTO(normalized)}, nil
}

func (service *WorkspaceService) ResolveWorkspaceSnapshot(ctx context.Context, request workspaceDTO.ResolveWorkspaceSnapshotRequest) (workspaceDTO.ResolveWorkspaceSnapshotResponse, error) {
	assistantID := strings.TrimSpace(request.AssistantID)
	if assistantID == "" {
		return workspaceDTO.ResolveWorkspaceSnapshotResponse{}, errors.New("assistant id is required")
	}
	if service.repo == nil {
		return workspaceDTO.ResolveWorkspaceSnapshotResponse{}, errors.New("workspace repository unavailable")
	}
	workspaceItem, snapshot, err := service.resolveAssistantWorkspaceSnapshot(ctx, assistantID, request.ForRunID, request.WorkspaceVersion)
	if err != nil {
		return workspaceDTO.ResolveWorkspaceSnapshotResponse{}, err
	}
	_ = workspaceItem
	return workspaceDTO.ResolveWorkspaceSnapshotResponse{Snapshot: toAssistantSnapshotDTO(snapshot)}, nil
}

func (service *WorkspaceService) resolveAssistantWorkspaceSnapshot(ctx context.Context, assistantID string, _ string, requestedVersion *int64) (workspace.AssistantWorkspace, workspace.AssistantWorkspaceSnapshot, error) {
	if service.repo == nil {
		return workspace.AssistantWorkspace{}, workspace.AssistantWorkspaceSnapshot{}, errors.New("workspace repository unavailable")
	}
	assistantID = strings.TrimSpace(assistantID)
	if assistantID == "" {
		return workspace.AssistantWorkspace{}, workspace.AssistantWorkspaceSnapshot{}, errors.New("assistant id is required")
	}
	item, err := service.ensureAssistantWorkspace(ctx, assistantID)
	if err != nil {
		return workspace.AssistantWorkspace{}, workspace.AssistantWorkspaceSnapshot{}, err
	}
	if requestedVersion != nil && *requestedVersion > 0 {
		snapshot, err := service.repo.GetAssistantWorkspaceSnapshot(ctx, assistantID, *requestedVersion)
		return item, snapshot, err
	}
	snapshot, err := service.repo.GetAssistantWorkspaceSnapshot(ctx, assistantID, item.Version)
	if err == nil {
		return item, snapshot, nil
	}
	if !errors.Is(err, workspace.ErrWorkspaceNotFound) {
		return item, workspace.AssistantWorkspaceSnapshot{}, err
	}
	snapshot = service.buildAssistantWorkspaceSnapshot(item)
	_ = service.repo.SaveAssistantWorkspaceSnapshot(ctx, snapshot)
	return item, snapshot, nil
}

func (service *WorkspaceService) ensureAssistantWorkspace(ctx context.Context, assistantID string) (workspace.AssistantWorkspace, error) {
	if service.repo == nil {
		return workspace.AssistantWorkspace{}, errors.New("workspace repository unavailable")
	}
	assistantID = strings.TrimSpace(assistantID)
	if assistantID == "" {
		return workspace.AssistantWorkspace{}, errors.New("assistant id is required")
	}
	item, err := service.repo.GetAssistantWorkspace(ctx, assistantID)
	if err == nil {
		return item, nil
	}
	if !errors.Is(err, workspace.ErrWorkspaceNotFound) {
		return workspace.AssistantWorkspace{}, err
	}
	if service.assistants == nil {
		return workspace.AssistantWorkspace{}, errors.New("assistant repository unavailable")
	}
	assistantItem, err := service.assistants.Get(ctx, assistantID)
	if err != nil {
		return workspace.AssistantWorkspace{}, err
	}
	global, globalErr := service.getOrCreateGlobal(ctx)
	persona := ""
	memoryJSON := ""
	if globalErr == nil {
		persona = global.DefaultPersona
		memoryJSON = global.DefaultMemoryJSON
	}
	now := service.now()
	created, err := workspace.NewAssistantWorkspace(workspace.AssistantWorkspaceParams{
		AssistantID:       assistantID,
		Version:           1,
		Identity:          assistantItem.Identity,
		Persona:           persona,
		UserProfile:       assistantItem.User,
		Tooling:           assistantItem.Call,
		Memory:            assistantItem.Memory,
		MemoryJSON:        memoryJSON,
		ExtraFiles:        nil,
		PromptModeDefault: workspaceDTO.PromptModeFull,
		CreatedAt:         &now,
		UpdatedAt:         &now,
	})
	if err != nil {
		return workspace.AssistantWorkspace{}, err
	}
	if err := service.repo.SaveAssistantWorkspace(ctx, created); err != nil {
		return workspace.AssistantWorkspace{}, err
	}
	return created, nil
}

func (service *WorkspaceService) buildAssistantWorkspaceSnapshot(item workspace.AssistantWorkspace) workspace.AssistantWorkspaceSnapshot {
	now := service.now()
	return workspace.AssistantWorkspaceSnapshot{
		ID:                service.newID(),
		AssistantID:       item.AssistantID,
		WorkspaceVersion:  item.Version,
		LogicalFiles:      buildLogicalFiles(item),
		PromptModeDefault: strings.TrimSpace(item.PromptModeDefault),
		ToolHints:         normalizeHints(item.Tooling.Tools.AllowList),
		SkillHints:        normalizeHints(item.Tooling.Skills.AllowList),
		GeneratedAt:       now,
		CreatedAt:         now,
	}
}

func buildLogicalFiles(item workspace.AssistantWorkspace) []workspace.WorkspaceLogicalFile {
	updatedAt := item.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = time.Now()
	}
	files := make([]workspace.WorkspaceLogicalFile, 0, 8+len(item.ExtraFiles))
	if file, ok := newLogicalFile("IDENTITY", formatIdentityContent(item.Identity), true, "assistant-field", updatedAt); ok {
		files = append(files, file)
	}
	if file, ok := newLogicalFile("SOUL", formatSoulContent(item.Identity), true, "assistant-field", updatedAt); ok {
		files = append(files, file)
	}
	if file, ok := newLogicalFile("USER", formatUserContent(item.UserProfile), true, "assistant-field", updatedAt); ok {
		files = append(files, file)
	}
	if file, ok := newLogicalFile("TOOLS", formatToolingContent(item.Tooling), true, "assistant-field", updatedAt); ok {
		files = append(files, file)
	}
	if file, ok := newLogicalFile("MEMORY", formatMemoryContent(item.Memory, item.MemoryJSON), true, "assistant-field", updatedAt); ok {
		files = append(files, file)
	}
	if file, ok := newLogicalFile("PERSONA", formatPersonaContent(item.Persona), false, "assistant-field", updatedAt); ok {
		files = append(files, file)
	}
	for _, extra := range item.ExtraFiles {
		name := strings.TrimSpace(extra.Name)
		if name == "" {
			continue
		}
		extra.Name = name
		if extra.Source == "" {
			extra.Source = "user-input"
		}
		if extra.UpdatedAt.IsZero() {
			extra.UpdatedAt = updatedAt
		}
		if strings.TrimSpace(extra.Content) == "" && extra.Required {
			extra.Missing = true
		}
		if extra.Size <= 0 && extra.Content != "" {
			extra.Size = len(extra.Content)
		}
		files = append(files, extra)
	}
	return files
}

func newLogicalFile(name string, content string, required bool, source string, updatedAt time.Time) (workspace.WorkspaceLogicalFile, bool) {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" && !required {
		return workspace.WorkspaceLogicalFile{}, false
	}
	missing := false
	if trimmed == "" && required {
		missing = true
	}
	return workspace.WorkspaceLogicalFile{
		Name:      name,
		Path:      "",
		Content:   trimmed,
		Source:    strings.TrimSpace(source),
		Required:  required,
		MaxChars:  0,
		Size:      len(trimmed),
		Missing:   missing,
		UpdatedAt: updatedAt,
	}, true
}

func formatIdentityContent(identity assistantdomain.AssistantIdentity) string {
	lines := make([]string, 0, 2)
	if value := strings.TrimSpace(identity.Name); value != "" {
		lines = append(lines, "Name: "+value)
	}
	if value := strings.TrimSpace(identity.Creature); value != "" {
		lines = append(lines, "Creature: "+value)
	}
	if value := strings.TrimSpace(identity.Emoji); value != "" {
		lines = append(lines, "Emoji: "+value)
	}
	if value := strings.TrimSpace(identity.Role); value != "" {
		lines = append(lines, "Role: "+value)
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func formatSoulContent(identity assistantdomain.AssistantIdentity) string {
	lines := make([]string, 0, 6)
	appendBulletSection(&lines, "CoreTruths", identity.Soul.CoreTruths)
	appendBulletSection(&lines, "Boundaries", identity.Soul.Boundaries)
	appendBulletSection(&lines, "Rules", identity.Soul.Rules)
	if value := strings.TrimSpace(identity.Soul.Vibe); value != "" {
		lines = append(lines, "Vibe: "+value)
	}
	if value := strings.TrimSpace(identity.Soul.Continuity); value != "" {
		lines = append(lines, "Continuity: "+value)
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func formatUserContent(user assistantdomain.AssistantUser) string {
	lines := make([]string, 0, 8)
	if value := strings.TrimSpace(user.Name); value != "" {
		lines = append(lines, "Name: "+value)
	}
	if value := strings.TrimSpace(user.PreferredAddress); value != "" {
		lines = append(lines, "Preferred address: "+value)
	}
	if value := strings.TrimSpace(user.Pronouns); value != "" {
		lines = append(lines, "Pronouns: "+value)
	}
	if value := strings.TrimSpace(user.Notes); value != "" {
		lines = append(lines, "Notes: "+value)
	}
	if value := strings.TrimSpace(assistantdomain.ResolveUserLocaleValue(user.Language)); value != "" {
		lines = append(lines, "Language: "+value)
	}
	if value := strings.TrimSpace(assistantdomain.ResolveUserLocaleValue(user.Timezone)); value != "" {
		lines = append(lines, "Timezone: "+value)
	}
	if value := strings.TrimSpace(assistantdomain.ResolveUserLocaleValue(user.Location)); value != "" {
		lines = append(lines, "Location: "+value)
	}
	if len(user.Extra) > 0 {
		lines = append(lines, "Preferences:")
		for _, extra := range user.Extra {
			key := strings.TrimSpace(extra.Key)
			value := strings.TrimSpace(extra.Value)
			if key == "" {
				continue
			}
			if value == "" {
				lines = append(lines, "- "+key)
			} else {
				lines = append(lines, "- "+key+": "+value)
			}
		}
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func formatToolingContent(tooling assistantdomain.AssistantCall) string {
	lines := make([]string, 0, 6)
	if value := strings.TrimSpace(tooling.Tools.Mode); value != "" {
		lines = append(lines, "Tools Mode: "+value)
	}
	if len(tooling.Tools.AllowList) > 0 {
		lines = append(lines, "Tools AllowList: "+strings.Join(normalizeHints(tooling.Tools.AllowList), ", "))
	}
	if len(tooling.Tools.DenyList) > 0 {
		lines = append(lines, "Tools DenyList: "+strings.Join(normalizeHints(tooling.Tools.DenyList), ", "))
	}
	if value := strings.TrimSpace(tooling.Skills.Mode); value != "" {
		lines = append(lines, "Skills Mode: "+value)
	}
	if len(tooling.Skills.AllowList) > 0 {
		lines = append(lines, "Skills AllowList: "+strings.Join(normalizeHints(tooling.Skills.AllowList), ", "))
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func formatMemoryContent(memory assistantdomain.AssistantMemory, memoryJSON string) string {
	lines := make([]string, 0, 3)
	if memory.Enabled {
		lines = append(lines, "Enabled: true")
	} else {
		lines = append(lines, "Enabled: false")
	}
	if value := strings.TrimSpace(memoryJSON); value != "" {
		lines = append(lines, "Config: "+value)
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func formatPersonaContent(persona string) string {
	return strings.TrimSpace(persona)
}

func appendBulletSection(lines *[]string, label string, items []string) {
	if lines == nil || len(items) == 0 {
		return
	}
	clean := normalizeHints(items)
	if len(clean) == 0 {
		return
	}
	*lines = append(*lines, label+":")
	for _, item := range clean {
		*lines = append(*lines, "- "+item)
	}
}

func normalizeHints(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		result = append(result, trimmed)
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func toWorkspaceContextFiles(files []workspace.WorkspaceLogicalFile) []workspaceDTO.WorkspaceFile {
	if len(files) == 0 {
		return nil
	}
	result := make([]workspaceDTO.WorkspaceFile, 0, len(files))
	for _, file := range files {
		name := strings.TrimSpace(file.Name)
		if name == "" {
			continue
		}
		result = append(result, workspaceDTO.WorkspaceFile{
			Name:      name,
			Path:      strings.TrimSpace(file.Path),
			Content:   file.Content,
			MaxChars:  file.MaxChars,
			Size:      file.Size,
			Missing:   file.Missing,
			UpdatedAt: formatTime(&file.UpdatedAt),
		})
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func toDomainLogicalFiles(files []workspaceDTO.WorkspaceLogicalFile) []workspace.WorkspaceLogicalFile {
	if len(files) == 0 {
		return nil
	}
	result := make([]workspace.WorkspaceLogicalFile, 0, len(files))
	for _, file := range files {
		name := strings.TrimSpace(file.Name)
		if name == "" {
			continue
		}
		result = append(result, workspace.WorkspaceLogicalFile{
			Name:      name,
			Path:      strings.TrimSpace(file.Path),
			Content:   strings.TrimSpace(file.Content),
			Source:    strings.TrimSpace(file.Source),
			Required:  file.Required,
			MaxChars:  file.MaxChars,
			Size:      file.Size,
			Missing:   file.Missing,
			UpdatedAt: parseTime(file.UpdatedAt),
		})
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func (service *WorkspaceService) getOrCreateGlobal(ctx context.Context) (workspace.GlobalWorkspace, error) {
	global, err := service.repo.GetGlobal(ctx)
	if err == nil {
		return global, nil
	}
	if err != workspace.ErrWorkspaceNotFound {
		return workspace.GlobalWorkspace{}, err
	}

	now := service.now()
	created, err := workspace.NewGlobalWorkspace(workspace.GlobalWorkspaceParams{
		ID:        1,
		CreatedAt: &now,
		UpdatedAt: &now,
	})
	if err != nil {
		return workspace.GlobalWorkspace{}, err
	}
	if err := service.repo.SaveGlobal(ctx, created); err != nil {
		return workspace.GlobalWorkspace{}, err
	}
	return created, nil
}

func toGlobalDTO(workspace workspace.GlobalWorkspace) workspaceDTO.GlobalWorkspace {
	return workspaceDTO.GlobalWorkspace{
		ID:                       workspace.ID,
		DefaultExecutorModelJSON: workspace.DefaultExecutorModelJSON,
		DefaultMemoryJSON:        workspace.DefaultMemoryJSON,
		DefaultPersona:           workspace.DefaultPersona,
		CreatedAt:                workspace.CreatedAt.Format(time.RFC3339),
		UpdatedAt:                workspace.UpdatedAt.Format(time.RFC3339),
	}
}

func toAssistantWorkspaceDTO(item workspace.AssistantWorkspace) workspaceDTO.AssistantWorkspace {
	return workspaceDTO.AssistantWorkspace{
		AssistantID:       item.AssistantID,
		Version:           item.Version,
		Identity:          item.Identity,
		Persona:           item.Persona,
		UserProfile:       item.UserProfile,
		Tooling:           item.Tooling,
		Memory:            item.Memory,
		MemoryJSON:        item.MemoryJSON,
		ExtraFiles:        toAssistantLogicalFilesDTO(item.ExtraFiles),
		PromptModeDefault: item.PromptModeDefault,
		UpdatedAt:         item.UpdatedAt.Format(time.RFC3339),
	}
}

func toAssistantSnapshotDTO(snapshot workspace.AssistantWorkspaceSnapshot) workspaceDTO.AssistantWorkspaceSnapshot {
	return workspaceDTO.AssistantWorkspaceSnapshot{
		SnapshotID:        snapshot.ID,
		AssistantID:       snapshot.AssistantID,
		WorkspaceVersion:  snapshot.WorkspaceVersion,
		LogicalFiles:      toAssistantLogicalFilesDTO(snapshot.LogicalFiles),
		PromptModeDefault: snapshot.PromptModeDefault,
		ToolHints:         snapshot.ToolHints,
		SkillHints:        snapshot.SkillHints,
		GeneratedAt:       formatTime(&snapshot.GeneratedAt),
		CreatedAt:         snapshot.CreatedAt.Format(time.RFC3339),
	}
}

func toAssistantLogicalFilesDTO(files []workspace.WorkspaceLogicalFile) []workspaceDTO.WorkspaceLogicalFile {
	if len(files) == 0 {
		return nil
	}
	result := make([]workspaceDTO.WorkspaceLogicalFile, 0, len(files))
	for _, file := range files {
		result = append(result, workspaceDTO.WorkspaceLogicalFile{
			Name:      file.Name,
			Path:      file.Path,
			Content:   file.Content,
			Source:    file.Source,
			Required:  file.Required,
			MaxChars:  file.MaxChars,
			Size:      file.Size,
			Missing:   file.Missing,
			UpdatedAt: formatTime(&file.UpdatedAt),
		})
	}
	return result
}

func formatTime(value *time.Time) string {
	if value == nil || value.IsZero() {
		return ""
	}
	return value.Format(time.RFC3339)
}

func parseTime(value string) time.Time {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return time.Time{}
	}
	parsed, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		return time.Time{}
	}
	return parsed
}
