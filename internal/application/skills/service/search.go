package service

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"dreamcreator/internal/application/skills/dto"
	"dreamcreator/internal/domain/externaltools"
)

const (
	defaultSkillSearchLimit = 20
	maxSkillSearchLimit     = 50
	minSkillSearchLength    = 2
	skillsSearchTimeout     = 20 * time.Second
	skillsManageTimeout     = 2 * time.Minute
)

const (
	ClawHubErrorCodeRateLimited  = "rate_limit_exceeded"
	ClawHubErrorCodeRequireForce = "requires_force"
)

var (
	ErrClawHubRateLimited  = errors.New("clawhub_rate_limited")
	ErrClawHubRequireForce = errors.New("clawhub_requires_force")
)

type ClawHubCommandError struct {
	Command string
	Message string
	Code    string
	Hint    string
	cause   error
}

func (err *ClawHubCommandError) Error() string {
	command := strings.TrimSpace(err.Command)
	message := strings.TrimSpace(err.Message)
	if command == "" {
		command = "command"
	}
	if message == "" {
		message = "unknown error"
	}
	return fmt.Sprintf("clawhub %s failed: %s", command, message)
}

func (err *ClawHubCommandError) Unwrap() error {
	if err == nil {
		return nil
	}
	return err.cause
}

type ClawHubErrorDetail struct {
	Code string
	Hint string
}

func ExtractClawHubErrorDetail(err error) (ClawHubErrorDetail, bool) {
	var commandErr *ClawHubCommandError
	if errors.As(err, &commandErr) && commandErr != nil {
		return ClawHubErrorDetail{
			Code: strings.TrimSpace(commandErr.Code),
			Hint: strings.TrimSpace(commandErr.Hint),
		}, true
	}
	return ClawHubErrorDetail{}, false
}

var (
	ansiPattern                 = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)
	skillIDPattern              = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+(?:@[A-Za-z0-9_.-]+)?$`)
	clawHubSearchLineSplit      = regexp.MustCompile(`\s{2,}`)
	clawHubSearchSlugWithVerExp = regexp.MustCompile(`^([A-Za-z0-9][A-Za-z0-9_.-]*)(?:\s+v([A-Za-z0-9_.-]+))?$`)
	skillMarkdownCandidates     = []string{"SKILL.md", "skill.md"}
)

func (service *SkillsService) SearchSkills(ctx context.Context, request dto.SearchSkillsRequest) ([]dto.SkillSearchResult, error) {
	query := strings.TrimSpace(request.Query)
	if query == "" {
		return nil, nil
	}
	if utf8.RuneCountInString(query) < minSkillSearchLength {
		return nil, nil
	}
	limit := request.Limit
	if limit <= 0 {
		limit = defaultSkillSearchLimit
	}
	if limit > maxSkillSearchLimit {
		limit = maxSkillSearchLimit
	}
	assistantID := strings.TrimSpace(request.AssistantID)
	providerID := "clawhub"
	workspaceRoot, err := service.resolveWorkspaceRoot(ctx, request.AssistantID, request.WorkspaceRoot)
	if err != nil {
		service.appendSkillsAuditRecord(ctx, "skill_manage.search", query, assistantID, providerID, err)
		return nil, err
	}
	output, err := service.runClawHubCommand(ctx, workspaceRoot, skillsSearchTimeout, "search", "--limit", fmt.Sprintf("%d", limit), query)
	if err != nil {
		service.appendSkillsAuditRecord(ctx, "skill_manage.search", query, assistantID, providerID, err)
		return nil, err
	}
	results := parseClawHubSearchOutput(output)
	if len(results) > limit {
		service.appendSkillsAuditRecord(ctx, "skill_manage.search", query, assistantID, providerID, nil)
		return results[:limit], nil
	}
	service.appendSkillsAuditRecord(ctx, "skill_manage.search", query, assistantID, providerID, nil)
	return results, nil
}

func (service *SkillsService) InspectSkill(ctx context.Context, request dto.InspectSkillRequest) (dto.SkillDetail, error) {
	skill := strings.TrimSpace(request.Skill)
	if skill == "" {
		return dto.SkillDetail{}, errors.New("skill is required")
	}
	assistantID := strings.TrimSpace(request.AssistantID)
	providerID := "clawhub"
	workspaceRoot, err := service.resolveWorkspaceRoot(ctx, request.AssistantID, request.WorkspaceRoot)
	if err != nil {
		service.appendSkillsAuditRecord(ctx, "skill_manage.inspect", skill, assistantID, providerID, err)
		return dto.SkillDetail{}, err
	}
	output, err := service.runClawHubCommand(ctx, workspaceRoot, skillsSearchTimeout, "inspect", skill, "--json", "--files")
	if err != nil {
		service.appendSkillsAuditRecord(ctx, "skill_manage.inspect", skill, assistantID, providerID, err)
		return dto.SkillDetail{}, err
	}
	detail, err := parseClawHubInspectOutput(output)
	if err != nil {
		service.appendSkillsAuditRecord(ctx, "skill_manage.inspect", skill, assistantID, providerID, err)
		return dto.SkillDetail{}, err
	}
	if version := service.resolveInstalledSkillVersion(ctx, workspaceRoot, skill); version != "" {
		detail.CurrentVersion = version
	}
	limits := service.resolveSkillsLimits(ctx)
	if markdown := service.fetchSkillMarkdown(ctx, workspaceRoot, skill, limits.MaxSkillFileBytes); markdown != "" {
		detail.SkillMarkdown = markdown
		if runtime := parseSkillRuntimeRequirementsFromMarkdown(markdown); runtime != nil {
			detail.Runtime = runtime
		}
	}
	if detail.ID == "" {
		detail.ID = skill
	}
	if detail.Name == "" {
		detail.Name = detail.ID
	}
	service.appendSkillsAuditRecord(ctx, "skill_manage.inspect", skill, assistantID, providerID, nil)
	return detail, nil
}

func (service *SkillsService) InstallSkill(ctx context.Context, request dto.InstallSkillRequest) error {
	skill := strings.TrimSpace(request.Skill)
	if skill == "" {
		return errors.New("skill is required")
	}
	assistantID := strings.TrimSpace(request.AssistantID)
	providerID := "clawhub"
	workspaceRoot, err := service.resolveWorkspaceRoot(ctx, request.AssistantID, request.WorkspaceRoot)
	if err != nil {
		service.appendSkillsAuditRecord(ctx, "skill_manage.install", skill, assistantID, providerID, err)
		return err
	}
	service.emitRealtimeEvent(ctx, SkillsRealtimeEvent{
		Action:        "install",
		Stage:         "started",
		Skill:         skill,
		Version:       strings.TrimSpace(request.Version),
		AssistantID:   request.AssistantID,
		WorkspaceRoot: workspaceRoot,
		Force:         request.Force,
	})
	args := []string{"install", skill}
	if version := strings.TrimSpace(request.Version); version != "" {
		args = append(args, "--version", version)
	}
	if request.Force {
		args = append(args, "--force")
	}
	_, err = service.runClawHubCommand(ctx, workspaceRoot, skillsManageTimeout, args...)
	if err != nil {
		service.recordInstallAttempt(false)
		service.emitRealtimeEvent(ctx, SkillsRealtimeEvent{
			Action:        "install",
			Stage:         "failed",
			Skill:         skill,
			Version:       strings.TrimSpace(request.Version),
			AssistantID:   request.AssistantID,
			WorkspaceRoot: workspaceRoot,
			Force:         request.Force,
			Error:         err.Error(),
		})
		service.appendSkillsAuditRecord(ctx, "skill_manage.install", skill, assistantID, providerID, err)
		return err
	}
	service.recordInstallAttempt(true)
	service.emitRealtimeEvent(ctx, SkillsRealtimeEvent{
		Action:        "install",
		Stage:         "completed",
		Skill:         skill,
		Version:       strings.TrimSpace(request.Version),
		AssistantID:   request.AssistantID,
		WorkspaceRoot: workspaceRoot,
		Force:         request.Force,
	})
	service.appendSkillsAuditRecord(ctx, "skill_manage.install", skill, assistantID, providerID, nil)
	return err
}

func (service *SkillsService) UpdateSkill(ctx context.Context, request dto.UpdateSkillRequest) error {
	skill := strings.TrimSpace(request.Skill)
	if skill == "" {
		return errors.New("skill is required")
	}
	assistantID := strings.TrimSpace(request.AssistantID)
	providerID := "clawhub"
	workspaceRoot, err := service.resolveWorkspaceRoot(ctx, request.AssistantID, request.WorkspaceRoot)
	if err != nil {
		service.appendSkillsAuditRecord(ctx, "skill_manage.update", skill, assistantID, providerID, err)
		return err
	}
	service.emitRealtimeEvent(ctx, SkillsRealtimeEvent{
		Action:        "update",
		Stage:         "started",
		Skill:         skill,
		Version:       strings.TrimSpace(request.Version),
		AssistantID:   request.AssistantID,
		WorkspaceRoot: workspaceRoot,
		Force:         request.Force,
	})
	args := []string{"update", skill}
	if version := strings.TrimSpace(request.Version); version != "" {
		args = append(args, "--version", version)
	}
	if request.Force {
		args = append(args, "--force")
	}
	_, err = service.runClawHubCommand(ctx, workspaceRoot, skillsManageTimeout, args...)
	if err != nil {
		service.recordInstallAttempt(false)
		service.emitRealtimeEvent(ctx, SkillsRealtimeEvent{
			Action:        "update",
			Stage:         "failed",
			Skill:         skill,
			Version:       strings.TrimSpace(request.Version),
			AssistantID:   request.AssistantID,
			WorkspaceRoot: workspaceRoot,
			Force:         request.Force,
			Error:         err.Error(),
		})
		service.appendSkillsAuditRecord(ctx, "skill_manage.update", skill, assistantID, providerID, err)
		return err
	}
	service.recordInstallAttempt(true)
	service.emitRealtimeEvent(ctx, SkillsRealtimeEvent{
		Action:        "update",
		Stage:         "completed",
		Skill:         skill,
		Version:       strings.TrimSpace(request.Version),
		AssistantID:   request.AssistantID,
		WorkspaceRoot: workspaceRoot,
		Force:         request.Force,
	})
	service.appendSkillsAuditRecord(ctx, "skill_manage.update", skill, assistantID, providerID, nil)
	return err
}

func (service *SkillsService) RemoveSkill(ctx context.Context, request dto.RemoveSkillRequest) error {
	skill := strings.TrimSpace(request.Skill)
	if skill == "" {
		return errors.New("skill is required")
	}
	assistantID := strings.TrimSpace(request.AssistantID)
	providerID := "clawhub"
	workspaceRoot, err := service.resolveWorkspaceRoot(ctx, request.AssistantID, request.WorkspaceRoot)
	if err != nil {
		service.appendSkillsAuditRecord(ctx, "skill_manage.remove", skill, assistantID, providerID, err)
		return err
	}
	service.emitRealtimeEvent(ctx, SkillsRealtimeEvent{
		Action:        "remove",
		Stage:         "started",
		Skill:         skill,
		AssistantID:   request.AssistantID,
		WorkspaceRoot: workspaceRoot,
	})
	_, err = service.runClawHubCommand(ctx, workspaceRoot, skillsManageTimeout, "uninstall", skill, "--yes")
	if err != nil {
		if isClawHubNotInstalledError(err) {
			if cleanupErr := service.removeWorkspaceSkillDirectories(ctx, workspaceRoot, skill); cleanupErr != nil {
				service.emitRealtimeEvent(ctx, SkillsRealtimeEvent{
					Action:        "remove",
					Stage:         "failed",
					Skill:         skill,
					AssistantID:   request.AssistantID,
					WorkspaceRoot: workspaceRoot,
					Error:         cleanupErr.Error(),
				})
				service.appendSkillsAuditRecord(ctx, "skill_manage.remove", skill, assistantID, providerID, cleanupErr)
				return cleanupErr
			}
			service.emitRealtimeEvent(ctx, SkillsRealtimeEvent{
				Action:        "remove",
				Stage:         "completed",
				Skill:         skill,
				AssistantID:   request.AssistantID,
				WorkspaceRoot: workspaceRoot,
			})
			service.appendSkillsAuditRecord(ctx, "skill_manage.remove", skill, assistantID, providerID, nil)
			return nil
		}
		service.emitRealtimeEvent(ctx, SkillsRealtimeEvent{
			Action:        "remove",
			Stage:         "failed",
			Skill:         skill,
			AssistantID:   request.AssistantID,
			WorkspaceRoot: workspaceRoot,
			Error:         err.Error(),
		})
		service.appendSkillsAuditRecord(ctx, "skill_manage.remove", skill, assistantID, providerID, err)
		return err
	}
	if cleanupErr := service.removeWorkspaceSkillDirectories(ctx, workspaceRoot, skill); cleanupErr != nil {
		service.emitRealtimeEvent(ctx, SkillsRealtimeEvent{
			Action:        "remove",
			Stage:         "failed",
			Skill:         skill,
			AssistantID:   request.AssistantID,
			WorkspaceRoot: workspaceRoot,
			Error:         cleanupErr.Error(),
		})
		service.appendSkillsAuditRecord(ctx, "skill_manage.remove", skill, assistantID, providerID, cleanupErr)
		return cleanupErr
	}
	service.emitRealtimeEvent(ctx, SkillsRealtimeEvent{
		Action:        "remove",
		Stage:         "completed",
		Skill:         skill,
		AssistantID:   request.AssistantID,
		WorkspaceRoot: workspaceRoot,
	})
	service.appendSkillsAuditRecord(ctx, "skill_manage.remove", skill, assistantID, providerID, nil)
	return nil
}

func isClawHubNotInstalledError(err error) bool {
	if err == nil {
		return false
	}
	lowered := strings.ToLower(strings.TrimSpace(err.Error()))
	if lowered == "" {
		return false
	}
	return strings.Contains(lowered, "not installed")
}

func (service *SkillsService) removeWorkspaceSkillDirectories(ctx context.Context, workspaceRoot string, skill string) error {
	keys := buildSkillLookupKeys(skill)
	if len(keys) == 0 {
		return nil
	}
	skillsRoot := strings.TrimSpace(resolveDreamCreatorSkillsRoot(workspaceRoot))
	if skillsRoot == "" {
		return nil
	}
	absSkillsRoot, err := filepath.Abs(skillsRoot)
	if err != nil {
		absSkillsRoot = skillsRoot
	}
	entries := service.loadWorkspaceSkillEntries(ctx, workspaceRoot, nil)
	dirs := make(map[string]struct{})
	for _, entry := range entries {
		entryKey := strings.ToLower(strings.TrimSpace(entry.ID))
		if _, ok := keys[entryKey]; !ok {
			continue
		}
		entryPath := strings.TrimSpace(entry.Path)
		if entryPath == "" {
			continue
		}
		dir := filepath.Dir(entryPath)
		if dir == "" || dir == "." {
			continue
		}
		absDir, absErr := filepath.Abs(dir)
		if absErr != nil {
			continue
		}
		if !isSubpathOrEqual(absSkillsRoot, absDir) {
			continue
		}
		dirs[absDir] = struct{}{}
	}
	for dir := range dirs {
		if removeErr := os.RemoveAll(dir); removeErr != nil {
			return removeErr
		}
	}
	if len(dirs) > 0 {
		bumpSkillsSnapshotVersion(strings.TrimSpace(workspaceRoot))
	}
	return nil
}

func isSubpathOrEqual(base string, target string) bool {
	base = strings.TrimSpace(base)
	target = strings.TrimSpace(target)
	if base == "" || target == "" {
		return false
	}
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return false
	}
	normalized := strings.TrimSpace(rel)
	if normalized == "." || normalized == "" {
		return true
	}
	if normalized == ".." || strings.HasPrefix(normalized, ".."+string(filepath.Separator)) {
		return false
	}
	return true
}

func (service *SkillsService) SyncSkills(ctx context.Context, request dto.SyncSkillsRequest) ([]dto.ProviderSkillSpec, error) {
	if service.repo == nil {
		return nil, errSkillsRepositoryUnavailable
	}
	assistantID := strings.TrimSpace(request.AssistantID)
	providerID := strings.TrimSpace(request.ProviderID)
	if providerID == "" {
		providerID = "clawhub"
	}
	workspaceRoot, err := service.resolveWorkspaceRoot(ctx, request.AssistantID, request.WorkspaceRoot)
	if err != nil {
		service.appendSkillsAuditRecord(ctx, "skill_manage.sync", "", assistantID, providerID, err)
		return nil, err
	}
	service.emitRealtimeEvent(ctx, SkillsRealtimeEvent{
		Action:        "sync",
		Stage:         "started",
		ProviderID:    strings.TrimSpace(request.ProviderID),
		AssistantID:   request.AssistantID,
		WorkspaceRoot: workspaceRoot,
	})
	if _, err := service.runClawHubCommand(ctx, workspaceRoot, skillsManageTimeout, "sync"); err != nil {
		service.emitRealtimeEvent(ctx, SkillsRealtimeEvent{
			Action:        "sync",
			Stage:         "failed",
			ProviderID:    strings.TrimSpace(request.ProviderID),
			AssistantID:   request.AssistantID,
			WorkspaceRoot: workspaceRoot,
			Error:         err.Error(),
		})
		service.appendSkillsAuditRecord(ctx, "skill_manage.sync", "", assistantID, providerID, err)
		return nil, err
	}
	result, err := service.ResolveSkillsForProviderInWorkspace(ctx, dto.ResolveSkillsRequest{
		ProviderID: strings.TrimSpace(request.ProviderID),
	}, workspaceRoot)
	if err != nil {
		service.emitRealtimeEvent(ctx, SkillsRealtimeEvent{
			Action:        "sync",
			Stage:         "failed",
			ProviderID:    strings.TrimSpace(request.ProviderID),
			AssistantID:   request.AssistantID,
			WorkspaceRoot: workspaceRoot,
			Error:         err.Error(),
		})
		service.appendSkillsAuditRecord(ctx, "skill_manage.sync", "", assistantID, providerID, err)
		return nil, err
	}
	service.emitRealtimeEvent(ctx, SkillsRealtimeEvent{
		Action:        "sync",
		Stage:         "completed",
		ProviderID:    strings.TrimSpace(request.ProviderID),
		AssistantID:   request.AssistantID,
		WorkspaceRoot: workspaceRoot,
		CatalogCount:  len(result),
	})
	service.appendSkillsAuditRecord(ctx, "skill_manage.sync", "", assistantID, providerID, nil)
	return result, nil
}

func (service *SkillsService) GetSkillsStatus(ctx context.Context, request dto.SkillsStatusRequest) (dto.SkillsStatus, error) {
	assistantID := strings.TrimSpace(request.AssistantID)
	providerID := strings.TrimSpace(request.ProviderID)
	if providerID == "" {
		providerID = "clawhub"
	}
	workspaceRoot, err := service.resolveWorkspaceRoot(ctx, request.AssistantID, request.WorkspaceRoot)
	if err != nil {
		service.appendSkillsAuditRecord(ctx, "skills.status", "", assistantID, providerID, err)
		return dto.SkillsStatus{}, err
	}
	ready, reason, err := service.ClawHubStatus(ctx)
	if err != nil {
		service.appendSkillsAuditRecord(ctx, "skills.status", "", assistantID, providerID, err)
		return dto.SkillsStatus{}, err
	}
	catalog, err := service.ResolveSkillsForProviderInWorkspace(ctx, dto.ResolveSkillsRequest{
		ProviderID: strings.TrimSpace(request.ProviderID),
	}, workspaceRoot)
	if err != nil {
		service.appendSkillsAuditRecord(ctx, "skills.status", "", assistantID, providerID, err)
		return dto.SkillsStatus{}, err
	}
	result := dto.SkillsStatus{
		ClawhubReady:  ready,
		Reason:        reason,
		WorkspaceRoot: workspaceRoot,
		CatalogCount:  len(catalog),
	}
	service.appendSkillsAuditRecord(ctx, "skills.status", "", assistantID, providerID, nil)
	return result, nil
}

func (service *SkillsService) ClawHubStatus(ctx context.Context) (bool, string, error) {
	if service == nil || service.externalTools == nil {
		return false, ErrClawHubUnavailable.Error(), nil
	}
	ready, _, err := service.externalTools.ToolReadiness(ctx, externaltools.ToolClawHub)
	if err != nil {
		return false, "", err
	}
	if !ready {
		return false, ErrClawHubUnavailable.Error(), nil
	}
	return true, "", nil
}

func (service *SkillsService) ensureClawHubReady(ctx context.Context) error {
	ready, _, err := service.ClawHubStatus(ctx)
	if err != nil {
		return err
	}
	if !ready {
		return ErrClawHubUnavailable
	}
	return nil
}

func (service *SkillsService) resolveClawHubExecPath(ctx context.Context) (string, error) {
	if err := service.ensureClawHubReady(ctx); err != nil {
		return "", err
	}
	execPath, err := service.externalTools.ResolveExecPath(ctx, externaltools.ToolClawHub)
	if err != nil || strings.TrimSpace(execPath) == "" {
		return "", ErrClawHubUnavailable
	}
	return execPath, nil
}

func (service *SkillsService) resolveWorkspaceRoot(ctx context.Context, assistantID string, workspaceRoot string) (string, error) {
	if root := strings.TrimSpace(workspaceRoot); root != "" {
		return root, nil
	}
	assistantID = strings.TrimSpace(assistantID)
	if assistantID == "" {
		return "", nil
	}
	if service == nil || service.workspaces == nil {
		return "", errors.New("assistant workspace resolver unavailable")
	}
	directory, err := service.workspaces.GetAssistantWorkspaceDirectory(ctx, assistantID)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(directory.RootPath), nil
}

func (service *SkillsService) runClawHubCommand(ctx context.Context, workspaceRoot string, timeout time.Duration, args ...string) ([]byte, error) {
	if service == nil {
		return nil, errors.New("skills service unavailable")
	}
	adapter := service.packageAdapter
	if adapter == nil {
		adapter = newClawHubPackageAdapter(service)
		service.packageAdapter = adapter
	}
	return adapter.Run(ctx, workspaceRoot, timeout, args...)
}

func classifyClawHubCommandError(args []string, message string) error {
	command := strings.TrimSpace(strings.Join(args, " "))
	normalized := strings.ToLower(strings.TrimSpace(message))
	if normalized == "" {
		return nil
	}
	if strings.Contains(normalized, "rate limit exceeded") ||
		strings.Contains(normalized, "too many requests") {
		return &ClawHubCommandError{
			Command: command,
			Message: message,
			Code:    ClawHubErrorCodeRateLimited,
			Hint:    "retry_later",
			cause:   ErrClawHubRateLimited,
		}
	}
	if strings.Contains(normalized, "use --force to install suspicious skills in non-interactive mode") ||
		(strings.Contains(normalized, "flagged as suspicious") && strings.Contains(normalized, "--force")) {
		return &ClawHubCommandError{
			Command: command,
			Message: message,
			Code:    ClawHubErrorCodeRequireForce,
			Hint:    "retry_with_force",
			cause:   ErrClawHubRequireForce,
		}
	}
	return nil
}

func (service *SkillsService) resolveInstalledSkillVersion(ctx context.Context, workspaceRoot string, skill string) string {
	output, err := service.runClawHubCommand(ctx, workspaceRoot, skillsSearchTimeout, "list")
	if err != nil {
		return ""
	}
	return parseClawHubInstalledVersion(output, skill)
}

func (service *SkillsService) fetchSkillMarkdown(ctx context.Context, workspaceRoot string, skill string, maxBytes int) string {
	for _, path := range skillMarkdownCandidates {
		output, err := service.runClawHubCommand(ctx, workspaceRoot, skillsSearchTimeout, "inspect", skill, "--json", "--file", path)
		if err != nil {
			continue
		}
		content, parseErr := parseClawHubInspectFileContent(output)
		if parseErr != nil {
			continue
		}
		if strings.TrimSpace(content) == "" {
			continue
		}
		if maxBytes > 0 && len(content) > maxBytes {
			continue
		}
		return content
	}
	return ""
}

func parseClawHubSearchOutput(output []byte) []dto.SkillSearchResult {
	if parsed := parseClawHubSearchJSON(output); len(parsed) > 0 {
		return parsed
	}
	return parseSkillsTextOutput(output)
}

func parseClawHubSearchJSON(output []byte) []dto.SkillSearchResult {
	trimmed := strings.TrimSpace(string(output))
	if trimmed == "" {
		return nil
	}

	var array []map[string]any
	if err := json.Unmarshal([]byte(trimmed), &array); err == nil {
		return normalizeSearchResultArray(array)
	}

	var object map[string]any
	if err := json.Unmarshal([]byte(trimmed), &object); err != nil {
		return nil
	}
	for _, key := range []string{"results", "items", "data"} {
		raw, ok := object[key]
		if !ok {
			continue
		}
		entries, ok := raw.([]any)
		if !ok {
			continue
		}
		items := make([]map[string]any, 0, len(entries))
		for _, entry := range entries {
			mapped, ok := entry.(map[string]any)
			if !ok {
				continue
			}
			items = append(items, mapped)
		}
		return normalizeSearchResultArray(items)
	}
	return nil
}

func normalizeSearchResultArray(items []map[string]any) []dto.SkillSearchResult {
	if len(items) == 0 {
		return nil
	}
	result := make([]dto.SkillSearchResult, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		id := firstNonEmpty(
			extractSearchField(item, "id"),
			extractSearchField(item, "slug"),
			extractSearchField(item, "name"),
		)
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		key := strings.ToLower(id)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		entry := dto.SkillSearchResult{
			ID:          id,
			Name:        firstNonEmpty(extractSearchField(item, "name"), id),
			Description: extractSearchField(item, "description"),
			URL: firstNonEmpty(
				extractSearchField(item, "url"),
				extractSearchField(item, "homepage"),
			),
			Source: firstNonEmpty(
				extractSearchField(item, "source"),
				extractSearchField(item, "provider"),
				"clawhub",
			),
		}
		result = append(result, entry)
	}
	return result
}

func extractSearchField(item map[string]any, key string) string {
	if item == nil || key == "" {
		return ""
	}
	raw, ok := item[key]
	if !ok {
		return ""
	}
	switch typed := raw.(type) {
	case string:
		return strings.TrimSpace(typed)
	case fmt.Stringer:
		return strings.TrimSpace(typed.String())
	default:
		return ""
	}
}

func parseSkillsTextOutput(output []byte) []dto.SkillSearchResult {
	text := stripANSI(string(output))
	scanner := bufio.NewScanner(strings.NewReader(text))
	results := make([]dto.SkillSearchResult, 0)
	var current *dto.SkillSearchResult
	seen := make(map[string]struct{})

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if parsed, ok := parseSkillSearchLine(line); ok {
			key := strings.ToLower(parsed.ID)
			if _, exists := seen[key]; exists {
				current = nil
				continue
			}
			seen[key] = struct{}{}
			results = append(results, parsed)
			current = &results[len(results)-1]
			continue
		}

		if current == nil {
			continue
		}

		trimmed := strings.TrimSpace(strings.TrimPrefix(line, "\u2514"))
		if current.URL == "" && strings.HasPrefix(trimmed, "http") {
			current.URL = trimmed
			continue
		}
		if current.Description == "" && !strings.HasPrefix(strings.ToLower(trimmed), "install with") {
			current.Description = trimmed
		}
	}

	return results
}

func parseSkillSearchLine(line string) (dto.SkillSearchResult, bool) {
	if skillIDPattern.MatchString(line) {
		return dto.SkillSearchResult{
			ID:     line,
			Name:   line,
			Source: "clawhub",
		}, true
	}

	parts := clawHubSearchLineSplit.Split(strings.TrimSpace(line), -1)
	if len(parts) == 0 {
		return dto.SkillSearchResult{}, false
	}
	head := strings.TrimSpace(parts[0])
	matches := clawHubSearchSlugWithVerExp.FindStringSubmatch(head)
	if len(matches) < 2 {
		return dto.SkillSearchResult{}, false
	}

	slug := strings.TrimSpace(matches[1])
	if slug == "" {
		return dto.SkillSearchResult{}, false
	}
	name := slug
	if len(parts) > 1 {
		if candidate := strings.TrimSpace(parts[1]); candidate != "" {
			name = candidate
		}
	}
	return dto.SkillSearchResult{
		ID:     slug,
		Name:   name,
		Source: "clawhub",
	}, true
}

func parseClawHubInspectOutput(output []byte) (dto.SkillDetail, error) {
	trimmed := strings.TrimSpace(stripANSI(string(output)))
	if trimmed == "" {
		return dto.SkillDetail{}, errors.New("empty clawhub inspect output")
	}
	payload, err := parseInspectJSONPayload(trimmed)
	if err != nil {
		return dto.SkillDetail{}, fmt.Errorf("invalid clawhub inspect output: %w", err)
	}

	skill, _ := payload["skill"].(map[string]any)
	if len(skill) == 0 && looksLikeInspectSkillMap(payload) {
		skill = payload
	}
	if len(skill) == 0 {
		return dto.SkillDetail{}, errors.New("invalid clawhub inspect output: skill missing")
	}
	owner, _ := payload["owner"].(map[string]any)
	if len(owner) == 0 {
		owner, _ = skill["owner"].(map[string]any)
	}
	latestVersion, _ := payload["latestVersion"].(map[string]any)
	if len(latestVersion) == 0 {
		latestVersion, _ = skill["latestVersion"].(map[string]any)
	}
	selectedVersion, _ := payload["version"].(map[string]any)
	if len(selectedVersion) == 0 {
		selectedVersion, _ = skill["version"].(map[string]any)
	}
	id := firstNonEmpty(
		extractSearchField(skill, "slug"),
		extractSearchField(skill, "id"),
	)
	ownerHandle := firstNonEmpty(extractSearchField(owner, "handle"), extractSearchField(owner, "name"))
	detail := dto.SkillDetail{
		ID:      id,
		Name:    firstNonEmpty(extractSearchField(skill, "displayName"), extractSearchField(skill, "name"), id),
		Summary: firstNonEmpty(extractSearchField(skill, "summary"), extractSearchField(skill, "description")),
		URL: firstNonEmpty(
			extractSearchField(skill, "url"),
			extractSearchField(skill, "homepage"),
			buildClawHubSkillURL(ownerHandle, extractSearchField(skill, "slug")),
		),
		Owner:           firstNonEmpty(extractSearchField(owner, "handle"), extractSearchField(owner, "displayName"), extractSearchField(owner, "name")),
		LatestVersion:   extractSearchField(latestVersion, "version"),
		SelectedVersion: extractSearchField(selectedVersion, "version"),
		CreatedAt:       extractJSONTimestamp(skill["createdAt"]),
		UpdatedAt:       extractJSONTimestamp(skill["updatedAt"]),
		Tags:            parseSkillTags(skill["tags"]),
		Changelog:       firstNonEmpty(extractSearchField(selectedVersion, "changelog"), extractSearchField(latestVersion, "changelog")),
		Files:           parseClawHubInspectFiles(selectedVersion["files"]),
	}
	return detail, nil
}

func parseClawHubInspectFileContent(output []byte) (string, error) {
	trimmed := strings.TrimSpace(stripANSI(string(output)))
	if trimmed == "" {
		return "", errors.New("empty clawhub inspect output")
	}
	payload, err := parseInspectJSONPayload(trimmed)
	if err != nil {
		return "", fmt.Errorf("invalid clawhub inspect output: %w", err)
	}
	file, _ := payload["file"].(map[string]any)
	if len(file) == 0 {
		return "", nil
	}
	switch typed := file["content"].(type) {
	case string:
		return typed, nil
	case nil:
		return "", nil
	default:
		return fmt.Sprint(typed), nil
	}
}

func parseClawHubInstalledVersion(output []byte, skill string) string {
	keys := buildSkillLookupKeys(skill)
	if len(keys) == 0 {
		return ""
	}
	scanner := bufio.NewScanner(strings.NewReader(stripANSI(string(output))))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "- ") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		slug := strings.ToLower(strings.TrimSpace(fields[0]))
		if _, ok := keys[slug]; !ok {
			continue
		}
		if len(fields) < 2 {
			return ""
		}
		return strings.TrimSpace(fields[1])
	}
	return ""
}

func parseClawHubInspectFiles(raw any) []dto.SkillDetailFile {
	entries, ok := raw.([]any)
	if !ok || len(entries) == 0 {
		return nil
	}
	result := make([]dto.SkillDetailFile, 0, len(entries))
	for _, entry := range entries {
		item, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		path := strings.TrimSpace(extractSearchField(item, "path"))
		if path == "" {
			continue
		}
		result = append(result, dto.SkillDetailFile{
			Path:        path,
			Size:        extractJSONTimestamp(item["size"]),
			SHA256:      extractSearchField(item, "sha256"),
			ContentType: extractSearchField(item, "contentType"),
		})
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func buildSkillLookupKeys(skill string) map[string]struct{} {
	trimmed := strings.TrimSpace(skill)
	if trimmed == "" {
		return nil
	}
	keys := map[string]struct{}{
		strings.ToLower(trimmed): {},
	}
	base := trimmed
	if at := strings.Index(base, "@"); at > 0 {
		base = strings.TrimSpace(base[:at])
		if base != "" {
			keys[strings.ToLower(base)] = struct{}{}
		}
	}
	if slash := strings.LastIndex(base, "/"); slash >= 0 && slash+1 < len(base) {
		tail := strings.TrimSpace(base[slash+1:])
		if tail != "" {
			keys[strings.ToLower(tail)] = struct{}{}
		}
	}
	return keys
}

func parseInspectJSONPayload(raw string) (map[string]any, error) {
	var payload map[string]any
	if err := json.Unmarshal([]byte(raw), &payload); err == nil {
		return payload, nil
	}
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start >= 0 && end > start {
		candidate := strings.TrimSpace(raw[start : end+1])
		if candidate != "" {
			if err := json.Unmarshal([]byte(candidate), &payload); err == nil {
				return payload, nil
			}
		}
	}
	return nil, errors.New("json object not found in inspect output")
}

func looksLikeInspectSkillMap(payload map[string]any) bool {
	if len(payload) == 0 {
		return false
	}
	for _, key := range []string{"slug", "id", "displayName", "name", "summary", "description"} {
		if value := strings.TrimSpace(extractSearchField(payload, key)); value != "" {
			return true
		}
	}
	return false
}

func parseSkillTags(raw any) []string {
	if raw == nil {
		return nil
	}
	result := make([]string, 0)
	appendTag := func(tag string) {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			return
		}
		result = append(result, tag)
	}
	switch typed := raw.(type) {
	case []any:
		for _, item := range typed {
			appendTag(fmt.Sprint(item))
		}
	case []string:
		for _, item := range typed {
			appendTag(item)
		}
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			switch value := typed[key].(type) {
			case string:
				value = strings.TrimSpace(value)
				if value == "" {
					appendTag(key)
					continue
				}
				appendTag(fmt.Sprintf("%s=%s", key, value))
			default:
				appendTag(key)
			}
		}
	default:
		appendTag(fmt.Sprint(raw))
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func parseSkillRuntimeRequirementsFromMarkdown(markdown string) *dto.SkillRuntimeRequirements {
	frontmatter, _ := parseFrontmatter(markdown)
	if len(frontmatter) == 0 {
		return nil
	}
	metadata := extractSkillMetadataMap(frontmatter)
	if len(metadata) == 0 {
		return nil
	}
	requires := toStringAnyMap(metadata["requires"])
	runtime := dto.SkillRuntimeRequirements{
		PrimaryEnv: readStringAny(metadata, "primaryEnv", "primary_env"),
		Homepage:   readStringAny(metadata, "homepage"),
		OS:         readStringSliceAny(metadata, "os"),
		Bins:       readStringSliceAny(requires, "bins"),
		AnyBins:    readStringSliceAny(requires, "anyBins", "any_bins"),
		Env:        readStringSliceAny(requires, "env"),
		Config:     readStringSliceAny(requires, "config"),
		Install:    parseSkillRuntimeInstallSpecs(metadata["install"]),
		Nix:        readStringAny(metadata, "nix"),
	}
	if isSkillRuntimeRequirementsEmpty(runtime) {
		return nil
	}
	return &runtime
}

func extractSkillMetadataMap(frontmatter map[string]any) map[string]any {
	if len(frontmatter) == 0 {
		return nil
	}
	metadata := toStringAnyMap(frontmatter["metadata"])
	if len(metadata) > 0 {
		if clawdbot := toStringAnyMap(metadata["clawdbot"]); len(clawdbot) > 0 {
			return clawdbot
		}
		if clawdis := toStringAnyMap(metadata["clawdis"]); len(clawdis) > 0 {
			return clawdis
		}
		return metadata
	}
	if clawdbot := toStringAnyMap(frontmatter["clawdbot"]); len(clawdbot) > 0 {
		return clawdbot
	}
	if clawdis := toStringAnyMap(frontmatter["clawdis"]); len(clawdis) > 0 {
		return clawdis
	}
	return nil
}

func toStringAnyMap(raw any) map[string]any {
	switch typed := raw.(type) {
	case map[string]any:
		return typed
	case map[any]any:
		result := make(map[string]any, len(typed))
		for key, value := range typed {
			keyString := strings.TrimSpace(fmt.Sprint(key))
			if keyString == "" {
				continue
			}
			result[keyString] = value
		}
		return result
	default:
		return nil
	}
}

func readStringAny(source map[string]any, keys ...string) string {
	if len(source) == 0 {
		return ""
	}
	for _, key := range keys {
		if key == "" {
			continue
		}
		raw, ok := source[key]
		if !ok {
			continue
		}
		if value, ok := raw.(string); ok {
			if trimmed := strings.TrimSpace(value); trimmed != "" {
				return trimmed
			}
		}
	}
	return ""
}

func readStringSliceAny(source map[string]any, keys ...string) []string {
	if len(source) == 0 || len(keys) == 0 {
		return nil
	}
	for _, key := range keys {
		if key == "" {
			continue
		}
		raw, ok := source[key]
		if !ok {
			continue
		}
		return stringSliceFromAny(raw)
	}
	return nil
}

func stringSliceFromAny(raw any) []string {
	switch typed := raw.(type) {
	case []string:
		return normalizeStringList(typed)
	case []any:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			switch itemValue := item.(type) {
			case string:
				result = append(result, itemValue)
			default:
				result = append(result, fmt.Sprint(itemValue))
			}
		}
		return normalizeStringList(result)
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed == "" {
			return nil
		}
		return normalizeStringList(strings.Split(trimmed, ","))
	default:
		return nil
	}
}

func parseSkillRuntimeInstallSpecs(raw any) []dto.SkillRuntimeInstallSpec {
	items, ok := raw.([]any)
	if !ok || len(items) == 0 {
		return nil
	}
	result := make([]dto.SkillRuntimeInstallSpec, 0, len(items))
	for _, entry := range items {
		item := toStringAnyMap(entry)
		if len(item) == 0 {
			continue
		}
		parsed := dto.SkillRuntimeInstallSpec{
			Kind:    readStringAny(item, "kind"),
			ID:      readStringAny(item, "id"),
			Label:   readStringAny(item, "label"),
			Bins:    readStringSliceAny(item, "bins"),
			Formula: readStringAny(item, "formula"),
			Tap:     readStringAny(item, "tap"),
			Package: readStringAny(item, "package"),
			Module:  readStringAny(item, "module"),
		}
		if parsed.Kind == "" &&
			parsed.ID == "" &&
			parsed.Label == "" &&
			len(parsed.Bins) == 0 &&
			parsed.Formula == "" &&
			parsed.Tap == "" &&
			parsed.Package == "" &&
			parsed.Module == "" {
			continue
		}
		result = append(result, parsed)
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func isSkillRuntimeRequirementsEmpty(runtime dto.SkillRuntimeRequirements) bool {
	return runtime.PrimaryEnv == "" &&
		runtime.Homepage == "" &&
		len(runtime.OS) == 0 &&
		len(runtime.Bins) == 0 &&
		len(runtime.AnyBins) == 0 &&
		len(runtime.Env) == 0 &&
		len(runtime.Config) == 0 &&
		len(runtime.Install) == 0 &&
		runtime.Nix == ""
}

func extractJSONTimestamp(raw any) int64 {
	switch typed := raw.(type) {
	case float64:
		return int64(typed)
	case float32:
		return int64(typed)
	case int:
		return int64(typed)
	case int64:
		return typed
	case json.Number:
		value, err := typed.Int64()
		if err == nil {
			return value
		}
	}
	return 0
}

func buildClawHubSkillURL(ownerHandle string, slug string) string {
	owner := strings.TrimSpace(ownerHandle)
	skillSlug := strings.TrimSpace(slug)
	if owner == "" || skillSlug == "" {
		return ""
	}
	return fmt.Sprintf("https://clawhub.ai/%s/%s", owner, skillSlug)
}

func stripANSI(input string) string {
	return ansiPattern.ReplaceAllString(input, "")
}
