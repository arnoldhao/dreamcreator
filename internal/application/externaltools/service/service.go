package service

import (
	"archive/zip"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"dreamcreator/internal/application/externaltools/dto"
	"dreamcreator/internal/application/softwareupdate"
	"dreamcreator/internal/domain/externaltools"
)

const (
	defaultDownloadTimeout = 30 * time.Minute
)

const (
	sourceKindGitHubRelease = "github_release"
	sourceKindNPMRegistry   = "npm_registry"
	sourceKindRuntime       = "runtime"

	toolManagerNPM = "npm"
	toolManagerBun = "bun"
)

type externalToolSource struct {
	ToolKind  string
	Kind      string
	SourceRef string
	Manager   string
}

var (
	toolSources = map[externaltools.ToolName]externalToolSource{
		externaltools.ToolYTDLP: {
			ToolKind:  string(externaltools.KindBin),
			Kind:      sourceKindGitHubRelease,
			SourceRef: "yt-dlp/yt-dlp",
		},
		externaltools.ToolFFmpeg: {
			ToolKind:  string(externaltools.KindBin),
			Kind:      sourceKindGitHubRelease,
			SourceRef: "jellyfin/jellyfin-ffmpeg",
		},
		externaltools.ToolBun: {
			ToolKind:  string(externaltools.KindBin),
			Kind:      sourceKindGitHubRelease,
			SourceRef: "oven-sh/bun",
		},
		externaltools.ToolClawHub: {
			ToolKind:  string(externaltools.KindBin),
			Kind:      sourceKindNPMRegistry,
			SourceRef: "clawhub",
			Manager:   toolManagerBun,
		},
	}

	semverTokenPattern = regexp.MustCompile(`^v?\d+\.\d+\.\d+(?:[-+][0-9A-Za-z.-]+)?$`)
)

type ExternalToolsService struct {
	repo         externaltools.Repository
	updates      *softwareupdate.Service
	appVersion   string
	now          func() time.Time
	installMu    sync.RWMutex
	installState map[externaltools.ToolName]dto.ExternalToolInstallState
}

func NewExternalToolsService(repo externaltools.Repository, updates *softwareupdate.Service, appVersion string) *ExternalToolsService {
	return &ExternalToolsService{
		repo:         repo,
		updates:      updates,
		appVersion:   strings.TrimSpace(appVersion),
		now:          time.Now,
		installState: make(map[externaltools.ToolName]dto.ExternalToolInstallState),
	}
}

const (
	installStageIdle        = "idle"
	installStageDownloading = "downloading"
	installStageExtracting  = "extracting"
	installStageVerifying   = "verifying"
	installStageDone        = "done"
	installStageError       = "error"

	downloadProgressStart = 0
	downloadProgressEnd   = 80
	extractProgressStart  = 80
	extractProgressEnd    = 95
	verifyProgressStart   = 95
	verifyProgressEnd     = 100
)

func (service *ExternalToolsService) setInstallState(name externaltools.ToolName, stage string, progress int, message string) {
	if name == "" {
		return
	}
	if progress < 0 {
		progress = 0
	}
	if progress > 100 {
		progress = 100
	}
	service.installMu.Lock()
	defer service.installMu.Unlock()
	service.installState[name] = dto.ExternalToolInstallState{
		Name:      string(name),
		Stage:     stage,
		Progress:  progress,
		Message:   message,
		UpdatedAt: service.now().Format(time.RFC3339),
	}
}

func (service *ExternalToolsService) GetInstallState(ctx context.Context, request dto.GetExternalToolInstallStateRequest) (dto.ExternalToolInstallState, error) {
	name := strings.TrimSpace(request.Name)
	if name == "" {
		return dto.ExternalToolInstallState{}, externaltools.ErrInvalidTool
	}
	toolName := externaltools.ToolName(name)
	service.installMu.RLock()
	defer service.installMu.RUnlock()
	if state, ok := service.installState[toolName]; ok {
		return state, nil
	}
	return dto.ExternalToolInstallState{
		Name:      name,
		Stage:     installStageIdle,
		Progress:  0,
		UpdatedAt: service.now().Format(time.RFC3339),
	}, nil
}

func (service *ExternalToolsService) EnsureDefaults(ctx context.Context) error {
	defaults := []externaltools.ToolName{
		externaltools.ToolYTDLP,
		externaltools.ToolFFmpeg,
		externaltools.ToolBun,
		externaltools.ToolClawHub,
	}
	existing, err := service.repo.List(ctx)
	if err != nil {
		return err
	}
	seen := make(map[externaltools.ToolName]struct{}, len(existing))
	for _, item := range existing {
		seen[item.Name] = struct{}{}
	}
	for _, tool := range defaults {
		if _, ok := seen[tool]; ok {
			continue
		}
		now := service.now()
		entry, err := externaltools.NewExternalTool(externaltools.ExternalToolParams{
			Name:      string(tool),
			Status:    string(externaltools.StatusMissing),
			UpdatedAt: &now,
		})
		if err != nil {
			return err
		}
		if err := service.repo.Save(ctx, entry); err != nil {
			return err
		}
	}
	return nil
}

func (service *ExternalToolsService) ListTools(ctx context.Context) ([]dto.ExternalTool, error) {
	items, err := service.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]dto.ExternalTool, 0, len(items))
	for _, item := range items {
		if _, err := resolveToolSource(item.Name); err != nil {
			continue
		}
		status := item.Status
		if status == "" {
			status = externaltools.StatusMissing
		}
		if item.ExecPath != "" {
			switch {
			case !pathExists(item.ExecPath):
				status = externaltools.StatusInvalid
			case item.Name == externaltools.ToolFFmpeg && !pathExists(ffprobePathForFFmpegExec(item.ExecPath)):
				status = externaltools.StatusInvalid
			}
		}
		entry := toExternalToolDTO(item)
		entry.Status = string(status)
		result = append(result, entry)
	}
	return result, nil
}

func (service *ExternalToolsService) ListToolUpdates(ctx context.Context) ([]dto.ExternalToolUpdateInfo, error) {
	items, err := service.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]dto.ExternalToolUpdateInfo, 0, len(items))
	for _, item := range items {
		source, err := resolveToolSource(item.Name)
		if err != nil {
			continue
		}
		info := dto.ExternalToolUpdateInfo{
			Name: string(item.Name),
		}
		release, releaseErr := service.resolveToolRelease(ctx, item.Name)
		if releaseErr == nil {
			info.LatestVersion = release.TargetVersion()
			info.RecommendedVersion = release.RecommendedVersion
			info.UpstreamVersion = release.UpstreamVersion
			info.ReleaseNotes = release.Notes
			info.ReleaseNotesURL = release.ReleasePage
			info.AutoUpdate = release.AutoUpdate
			info.Required = release.Required
		}
		// Runtime tools don't expose a stable remote "latest" endpoint.
		// Use the installed runtime version as the target display version.
		if source.Kind == sourceKindRuntime && strings.TrimSpace(info.LatestVersion) == "" {
			info.LatestVersion = strings.TrimSpace(item.Version)
		}
		result = append(result, info)
	}
	return result, nil
}

func (service *ExternalToolsService) InstallTool(ctx context.Context, request dto.InstallExternalToolRequest) (dto.ExternalTool, error) {
	name := strings.TrimSpace(request.Name)
	if name == "" {
		return dto.ExternalTool{}, externaltools.ErrInvalidTool
	}
	toolName := externaltools.ToolName(name)
	source, err := resolveToolSource(toolName)
	if err != nil {
		service.setInstallState(toolName, installStageError, downloadProgressStart, "invalid tool")
		return dto.ExternalTool{}, err
	}
	manager := strings.ToLower(strings.TrimSpace(request.Manager))
	if manager == "" {
		manager = source.Manager
	}
	if manager == "" && source.Kind == sourceKindNPMRegistry {
		manager = toolManagerNPM
	}
	service.setInstallState(toolName, installStageDownloading, 0, "")
	switch source.Kind {
	case sourceKindGitHubRelease:
		if manager != "" {
			service.setInstallState(toolName, installStageError, downloadProgressStart, "manager is unsupported for this tool")
			return dto.ExternalTool{}, fmt.Errorf("manager is unsupported for tool %s", toolName)
		}
		if installed, handled, err := service.installToolFromReleaseCatalog(ctx, toolName, request.Version); handled {
			if err != nil {
				service.setInstallState(toolName, installStageError, downloadProgressStart, err.Error())
			}
			return installed, err
		}
		switch toolName {
		case externaltools.ToolYTDLP:
			return service.installYTDLP(ctx, request.Version)
		case externaltools.ToolFFmpeg:
			return service.installFFmpeg(ctx, request.Version)
		case externaltools.ToolBun:
			return service.installBun(ctx, request.Version)
		default:
			service.setInstallState(toolName, installStageError, downloadProgressStart, "invalid tool")
			return dto.ExternalTool{}, externaltools.ErrInvalidTool
		}
	case sourceKindNPMRegistry:
		return service.installNPMRegistryTool(ctx, toolName, source, request.Version, manager)
	case sourceKindRuntime:
		service.setInstallState(toolName, installStageError, downloadProgressStart, "runtime tools are not supported")
		return dto.ExternalTool{}, externaltools.ErrInvalidTool
	default:
		service.setInstallState(toolName, installStageError, downloadProgressStart, "unsupported source")
		return dto.ExternalTool{}, fmt.Errorf("unsupported source for tool %s", toolName)
	}
}

func (service *ExternalToolsService) VerifyTool(ctx context.Context, request dto.VerifyExternalToolRequest) (dto.ExternalTool, error) {
	name := strings.TrimSpace(request.Name)
	if name == "" {
		return dto.ExternalTool{}, externaltools.ErrInvalidTool
	}
	tool, err := service.repo.Get(ctx, name)
	if err != nil {
		return dto.ExternalTool{}, err
	}
	status := tool.Status
	version := tool.Version
	if tool.ExecPath == "" || !pathExists(tool.ExecPath) {
		status = externaltools.StatusMissing
		version = ""
	} else {
		ver, verErr := resolveInstalledToolVersion(ctx, tool.Name, tool.ExecPath)
		if verErr != nil {
			status = externaltools.StatusInvalid
			version = ""
		} else {
			status = externaltools.StatusInstalled
			version = ver
		}
	}
	now := service.now()
	updated, err := externaltools.NewExternalTool(externaltools.ExternalToolParams{
		Name:        string(tool.Name),
		ExecPath:    tool.ExecPath,
		Version:     version,
		Status:      string(status),
		InstalledAt: tool.InstalledAt,
		UpdatedAt:   &now,
	})
	if err != nil {
		return dto.ExternalTool{}, err
	}
	if err := service.repo.Save(ctx, updated); err != nil {
		return dto.ExternalTool{}, err
	}
	return toExternalToolDTO(updated), nil
}

func (service *ExternalToolsService) SetToolPath(ctx context.Context, request dto.SetExternalToolPathRequest) (dto.ExternalTool, error) {
	name := strings.TrimSpace(request.Name)
	if name == "" {
		return dto.ExternalTool{}, externaltools.ErrInvalidTool
	}
	toolName := externaltools.ToolName(name)
	if source, err := resolveToolSource(toolName); err == nil && source.ToolKind == string(externaltools.KindRuntime) {
		return dto.ExternalTool{}, fmt.Errorf("manual path is unsupported for runtime tool %s", toolName)
	}
	execPath := strings.TrimSpace(request.ExecPath)
	if execPath == "" {
		return dto.ExternalTool{}, externaltools.ErrInvalidTool
	}
	now := service.now()
	version := ""
	status := externaltools.StatusMissing
	var installedAt *time.Time
	if pathExists(execPath) {
		if resolved, err := resolveInstalledToolVersion(ctx, toolName, execPath); err == nil {
			version = resolved
			status = externaltools.StatusInstalled
			installedAt = &now
		} else {
			status = externaltools.StatusInvalid
		}
	}
	updated, err := externaltools.NewExternalTool(externaltools.ExternalToolParams{
		Name:        name,
		ExecPath:    execPath,
		Version:     version,
		Status:      string(status),
		InstalledAt: installedAt,
		UpdatedAt:   &now,
	})
	if err != nil {
		return dto.ExternalTool{}, err
	}
	if err := service.repo.Save(ctx, updated); err != nil {
		return dto.ExternalTool{}, err
	}
	return toExternalToolDTO(updated), nil
}

func (service *ExternalToolsService) RemoveTool(ctx context.Context, request dto.RemoveExternalToolRequest) error {
	name := strings.TrimSpace(request.Name)
	if name == "" {
		return externaltools.ErrInvalidTool
	}
	tool, err := service.repo.Get(ctx, name)
	if err != nil && err != externaltools.ErrToolNotFound {
		return err
	}
	if err == nil && tool.ExecPath != "" {
		_ = os.RemoveAll(filepath.Dir(tool.ExecPath))
	}
	if baseDir, baseErr := externalToolsBaseDir(); baseErr == nil {
		_ = os.RemoveAll(filepath.Join(baseDir, name))
	}
	now := service.now()
	updated, err := externaltools.NewExternalTool(externaltools.ExternalToolParams{
		Name:      name,
		Status:    string(externaltools.StatusMissing),
		UpdatedAt: &now,
	})
	if err != nil {
		return err
	}
	return service.repo.Save(ctx, updated)
}

func (service *ExternalToolsService) ResolveExecPath(ctx context.Context, name externaltools.ToolName) (string, error) {
	tool, err := service.repo.Get(ctx, string(name))
	if err != nil {
		return "", err
	}
	if tool.ExecPath == "" || !pathExists(tool.ExecPath) {
		return "", fmt.Errorf("%s is not installed", name)
	}
	if name == externaltools.ToolFFmpeg && !pathExists(ffprobePathForFFmpegExec(tool.ExecPath)) {
		return "", fmt.Errorf("ffprobe is not installed")
	}
	return tool.ExecPath, nil
}

func (service *ExternalToolsService) IsToolReady(ctx context.Context, name externaltools.ToolName) (bool, error) {
	ready, _, err := service.ToolReadiness(ctx, name)
	return ready, err
}

func (service *ExternalToolsService) ToolReadiness(ctx context.Context, name externaltools.ToolName) (bool, string, error) {
	if strings.TrimSpace(string(name)) == "" {
		return false, "invalid_tool", externaltools.ErrInvalidTool
	}
	tool, err := service.repo.Get(ctx, string(name))
	if err != nil {
		if errors.Is(err, externaltools.ErrToolNotFound) {
			return false, "not_found", nil
		}
		return false, "", err
	}
	if tool.Status != externaltools.StatusInstalled {
		if tool.Status == externaltools.StatusInvalid {
			return false, "invalid", nil
		}
		return false, "not_installed", nil
	}
	if strings.TrimSpace(tool.ExecPath) == "" {
		return false, "missing_exec_path", nil
	}
	if !pathExists(tool.ExecPath) {
		return false, "exec_not_found", nil
	}
	if name == externaltools.ToolFFmpeg && !pathExists(ffprobePathForFFmpegExec(tool.ExecPath)) {
		return false, "ffprobe_not_found", nil
	}
	return true, "", nil
}

func (service *ExternalToolsService) ResolveToolDirectory(ctx context.Context, name externaltools.ToolName) (string, error) {
	execPath, err := service.ResolveExecPath(ctx, name)
	if err != nil {
		return "", err
	}
	dir := filepath.Dir(execPath)
	if !pathExists(dir) {
		return "", fmt.Errorf("%s directory not found", name)
	}
	return dir, nil
}

func (service *ExternalToolsService) resolveToolRelease(ctx context.Context, name externaltools.ToolName) (softwareupdate.ToolRelease, error) {
	if service.updates != nil {
		release, err := service.updates.ResolveToolRelease(ctx, softwareupdate.ToolRequest{
			AppVersion: service.appVersion,
			Name:       name,
		})
		if err == nil {
			return release, nil
		}
	}
	return resolveToolReleaseLegacy(ctx, name)
}

func resolveToolReleaseLegacy(ctx context.Context, name externaltools.ToolName) (softwareupdate.ToolRelease, error) {
	source, err := resolveToolSource(name)
	if err != nil {
		return softwareupdate.ToolRelease{}, err
	}
	latest, notes, notesURL, err := resolveToolUpdate(ctx, name)
	if err != nil {
		return softwareupdate.ToolRelease{}, err
	}
	displayName := string(name)
	if name == externaltools.ToolFFmpeg {
		displayName = "FFmpeg"
	} else if name == externaltools.ToolBun {
		displayName = "Bun"
	}
	return softwareupdate.ToolRelease{
		Name:               name,
		DisplayName:        displayName,
		Kind:               "external-tool",
		Source:             legacySourceRef(source.SourceRef),
		UpstreamVersion:    latest,
		RecommendedVersion: latest,
		Notes:              notes,
		ReleasePage:        notesURL,
	}, nil
}

func legacySourceRef(raw string) softwareupdate.SourceRef {
	parts := strings.Split(strings.TrimSpace(raw), "/")
	ref := softwareupdate.SourceRef{}
	if len(parts) >= 2 {
		ref.Owner = parts[0]
		ref.Repo = parts[1]
	}
	switch {
	case strings.HasPrefix(strings.TrimSpace(raw), "http"):
		ref.Provider = "url"
	case strings.Contains(strings.TrimSpace(raw), "/"):
		ref.Provider = "github-release"
	default:
		ref.Provider = "custom"
	}
	return ref
}

func (service *ExternalToolsService) installToolFromReleaseCatalog(ctx context.Context, name externaltools.ToolName, version string) (dto.ExternalTool, bool, error) {
	if service.updates == nil {
		return dto.ExternalTool{}, false, nil
	}
	if name != externaltools.ToolYTDLP && name != externaltools.ToolFFmpeg && name != externaltools.ToolBun {
		return dto.ExternalTool{}, false, nil
	}
	release, err := service.updates.ResolveToolRelease(ctx, softwareupdate.ToolRequest{
		AppVersion: service.appVersion,
		Name:       name,
	})
	if err != nil {
		return dto.ExternalTool{}, false, nil
	}
	targetVersion := strings.TrimSpace(release.TargetVersion())
	if targetVersion == "" || len(release.Asset.DownloadURLs()) == 0 {
		return dto.ExternalTool{}, false, nil
	}
	requestedVersion := normalizeManagedToolVersion(name, version)
	if requestedVersion != "" && requestedVersion != "latest" && requestedVersion != normalizeManagedToolVersion(name, targetVersion) {
		return dto.ExternalTool{}, false, nil
	}
	installed, err := service.installCatalogRelease(ctx, release)
	return installed, true, err
}

func normalizeManagedToolVersion(name externaltools.ToolName, version string) string {
	trimmed := strings.TrimSpace(version)
	if trimmed == "" {
		return ""
	}
	if strings.EqualFold(trimmed, "latest") {
		return "latest"
	}
	switch name {
	case externaltools.ToolBun:
		return normalizeBunVersion(trimmed)
	case externaltools.ToolFFmpeg:
		return normalizeFFmpegVersion(trimmed)
	default:
		return strings.TrimPrefix(trimmed, "v")
	}
}

func managedToolVersionFromPath(name externaltools.ToolName, execPath string) string {
	trimmedExecPath := strings.TrimSpace(execPath)
	if name == "" || trimmedExecPath == "" {
		return ""
	}
	normalizedPath := strings.Trim(filepath.ToSlash(filepath.Clean(trimmedExecPath)), "/")
	if normalizedPath == "" {
		return ""
	}
	parts := strings.Split(normalizedPath, "/")
	for index := 0; index+2 < len(parts); index++ {
		if parts[index] != "external-tools" {
			continue
		}
		if !strings.EqualFold(parts[index+1], string(name)) {
			continue
		}
		version := normalizeManagedToolVersion(name, parts[index+2])
		if version == "" || version == "latest" {
			return ""
		}
		return version
	}
	return ""
}

func (service *ExternalToolsService) installCatalogRelease(ctx context.Context, release softwareupdate.ToolRelease) (dto.ExternalTool, error) {
	switch strings.ToLower(strings.TrimSpace(release.Asset.InstallStrategy)) {
	case "binary":
		return service.installCatalogBinaryRelease(ctx, release)
	case "archive":
		return service.installCatalogArchiveRelease(ctx, release)
	default:
		return dto.ExternalTool{}, fmt.Errorf("unsupported install strategy %s for %s", release.Asset.InstallStrategy, release.Name)
	}
}

func (service *ExternalToolsService) installCatalogBinaryRelease(ctx context.Context, release softwareupdate.ToolRelease) (dto.ExternalTool, error) {
	baseDir, err := externalToolsBaseDir()
	if err != nil {
		return dto.ExternalTool{}, err
	}
	version := release.TargetVersion()
	toolDir := filepath.Join(baseDir, string(release.Name), version)
	execName := strings.TrimSpace(release.Asset.PrimaryExecutableName())
	if execName == "" {
		execName = executableName(release.Name)
	}
	execPath := filepath.Join(toolDir, execName)
	if err := downloadFromSourcesWithProgress(ctx, release.Asset.DownloadURLs(), execPath, func(progress int) {
		mapped := mapProgress(progress, downloadProgressStart, downloadProgressEnd)
		service.setInstallState(release.Name, installStageDownloading, mapped, "")
	}); err != nil {
		return dto.ExternalTool{}, err
	}
	if err := validateDownloadedExecutable(execPath); err != nil {
		return dto.ExternalTool{}, err
	}
	if err := markExecutable(execPath); err != nil {
		return dto.ExternalTool{}, err
	}
	service.setInstallState(release.Name, installStageVerifying, verifyProgressStart, "")
	resolvedVersion, err := resolveInstalledToolVersion(ctx, release.Name, execPath)
	if err != nil {
		return dto.ExternalTool{}, err
	}
	return service.saveInstalledTool(ctx, release.Name, execPath, resolvedVersion, baseDir, version)
}

func (service *ExternalToolsService) installCatalogArchiveRelease(ctx context.Context, release softwareupdate.ToolRelease) (dto.ExternalTool, error) {
	baseDir, err := externalToolsBaseDir()
	if err != nil {
		return dto.ExternalTool{}, err
	}
	version := release.TargetVersion()
	toolDir := filepath.Join(baseDir, string(release.Name), version)
	if err := os.MkdirAll(toolDir, 0o755); err != nil {
		return dto.ExternalTool{}, err
	}
	archivePath := filepath.Join(toolDir, fmt.Sprintf("download-%d%s", time.Now().UnixNano(), archiveSuffixForAsset(release.Asset)))
	defer os.Remove(archivePath)

	if err := downloadFromSourcesWithProgress(ctx, release.Asset.DownloadURLs(), archivePath, func(progress int) {
		mapped := mapProgress(progress, downloadProgressStart, downloadProgressEnd)
		service.setInstallState(release.Name, installStageDownloading, mapped, "")
	}); err != nil {
		return dto.ExternalTool{}, err
	}

	binaries := append([]string(nil), release.Asset.Binaries...)
	if len(binaries) == 0 {
		primary := strings.TrimSpace(release.Asset.PrimaryExecutableName())
		if primary != "" {
			binaries = []string{primary}
		}
	}
	if len(binaries) == 0 {
		return dto.ExternalTool{}, fmt.Errorf("missing binaries for %s release", release.Name)
	}

	var extracted map[string]string
	switch strings.ToLower(strings.TrimSpace(release.Asset.ArtifactType)) {
	case "zip":
		extracted, err = extractZipExecutables(archivePath, toolDir, binaries, func(progress int) {
			mapped := mapProgress(progress, extractProgressStart, extractProgressEnd)
			service.setInstallState(release.Name, installStageExtracting, mapped, "")
		})
	case "tar.xz":
		extracted, err = extractTarXZExecutables(archivePath, toolDir, binaries, func(progress int) {
			mapped := mapProgress(progress, extractProgressStart, extractProgressEnd)
			service.setInstallState(release.Name, installStageExtracting, mapped, "")
		})
	default:
		err = fmt.Errorf("unsupported artifact type %s for %s", release.Asset.ArtifactType, release.Name)
	}
	if err != nil {
		return dto.ExternalTool{}, err
	}

	for _, binary := range binaries {
		if path := extracted[binary]; strings.TrimSpace(path) != "" {
			if err := markExecutable(path); err != nil {
				return dto.ExternalTool{}, err
			}
		}
	}

	service.setInstallState(release.Name, installStageVerifying, verifyProgressStart, "")
	execPath := extracted[strings.TrimSpace(release.Asset.PrimaryExecutableName())]
	if strings.TrimSpace(execPath) == "" {
		execPath = extracted[binaries[0]]
	}
	if strings.TrimSpace(execPath) == "" {
		return dto.ExternalTool{}, fmt.Errorf("primary executable not found for %s", release.Name)
	}

	resolvedVersion, err := resolveInstalledCatalogVersion(ctx, release.Name, execPath)
	if err != nil {
		return dto.ExternalTool{}, err
	}
	return service.saveInstalledTool(ctx, release.Name, execPath, resolvedVersion, baseDir, version)
}

func resolveInstalledCatalogVersion(ctx context.Context, name externaltools.ToolName, execPath string) (string, error) {
	if name == externaltools.ToolFFmpeg {
		return validateFFmpegInstallation(ctx, execPath)
	}
	return resolveVersion(ctx, name, execPath)
}

func (service *ExternalToolsService) saveInstalledTool(ctx context.Context, name externaltools.ToolName, execPath string, version string, baseDir string, cleanupVersion string) (dto.ExternalTool, error) {
	now := service.now()
	tool, err := externaltools.NewExternalTool(externaltools.ExternalToolParams{
		Name:        string(name),
		ExecPath:    execPath,
		Version:     strings.TrimSpace(version),
		Status:      string(externaltools.StatusInstalled),
		InstalledAt: &now,
		UpdatedAt:   &now,
	})
	if err != nil {
		return dto.ExternalTool{}, err
	}
	if err := service.repo.Save(ctx, tool); err != nil {
		service.setInstallState(name, installStageError, verifyProgressEnd, err.Error())
		return dto.ExternalTool{}, err
	}
	service.setInstallState(name, installStageDone, verifyProgressEnd, "")
	_ = cleanupOldToolVersions(baseDir, name, cleanupVersion)
	return toExternalToolDTO(tool), nil
}

func archiveSuffixForAsset(asset softwareupdate.Asset) string {
	name := strings.ToLower(strings.TrimSpace(asset.ArtifactName))
	switch {
	case strings.HasSuffix(name, ".tar.xz"):
		return ".tar.xz"
	case strings.HasSuffix(name, ".zip"):
		return ".zip"
	}
	switch strings.ToLower(strings.TrimSpace(asset.ArtifactType)) {
	case "tar.xz":
		return ".tar.xz"
	default:
		return ".zip"
	}
}

func (service *ExternalToolsService) installYTDLP(ctx context.Context, version string) (dto.ExternalTool, error) {
	if version == "" || version == "latest" {
		resolved, err := getLatestGitHubTag(ctx, "yt-dlp", "yt-dlp")
		if err != nil {
			service.setInstallState(externaltools.ToolYTDLP, installStageError, downloadProgressStart, err.Error())
			return dto.ExternalTool{}, err
		}
		version = resolved
	}
	filename := ytdlpFilename(runtime.GOOS)
	downloadURL := fmt.Sprintf("https://github.com/yt-dlp/yt-dlp/releases/download/%s/%s", version, filename)
	baseDir, err := externalToolsBaseDir()
	if err != nil {
		service.setInstallState(externaltools.ToolYTDLP, installStageError, downloadProgressStart, err.Error())
		return dto.ExternalTool{}, err
	}
	toolDir := filepath.Join(baseDir, string(externaltools.ToolYTDLP), version)
	execName := executableName(externaltools.ToolYTDLP)
	execPath := filepath.Join(toolDir, execName)
	if err := downloadFileWithProgress(ctx, downloadURL, execPath, func(progress int) {
		mapped := mapProgress(progress, downloadProgressStart, downloadProgressEnd)
		service.setInstallState(externaltools.ToolYTDLP, installStageDownloading, mapped, "")
	}); err != nil {
		service.setInstallState(externaltools.ToolYTDLP, installStageError, downloadProgressStart, err.Error())
		return dto.ExternalTool{}, err
	}
	if err := validateDownloadedExecutable(execPath); err != nil {
		if shouldApplyGitHubProxy(downloadURL) {
			_ = os.Remove(execPath)
			if retryErr := downloadFileWithProgressDirect(ctx, downloadURL, execPath, func(progress int) {
				mapped := mapProgress(progress, downloadProgressStart, downloadProgressEnd)
				service.setInstallState(externaltools.ToolYTDLP, installStageDownloading, mapped, "")
			}); retryErr != nil {
				service.setInstallState(externaltools.ToolYTDLP, installStageError, downloadProgressStart, retryErr.Error())
				return dto.ExternalTool{}, retryErr
			}
			if retryErr := validateDownloadedExecutable(execPath); retryErr != nil {
				service.setInstallState(externaltools.ToolYTDLP, installStageError, downloadProgressStart, retryErr.Error())
				return dto.ExternalTool{}, retryErr
			}
		} else {
			service.setInstallState(externaltools.ToolYTDLP, installStageError, downloadProgressStart, err.Error())
			return dto.ExternalTool{}, err
		}
	}
	if err := markExecutable(execPath); err != nil {
		service.setInstallState(externaltools.ToolYTDLP, installStageError, downloadProgressEnd, err.Error())
		return dto.ExternalTool{}, err
	}
	service.setInstallState(externaltools.ToolYTDLP, installStageVerifying, verifyProgressStart, "")
	if _, err := resolveVersion(ctx, externaltools.ToolYTDLP, execPath); err != nil {
		service.setInstallState(externaltools.ToolYTDLP, installStageError, verifyProgressStart, err.Error())
		return dto.ExternalTool{}, err
	}
	now := service.now()
	tool, err := externaltools.NewExternalTool(externaltools.ExternalToolParams{
		Name:        string(externaltools.ToolYTDLP),
		ExecPath:    execPath,
		Version:     version,
		Status:      string(externaltools.StatusInstalled),
		InstalledAt: &now,
		UpdatedAt:   &now,
	})
	if err != nil {
		return dto.ExternalTool{}, err
	}
	if err := service.repo.Save(ctx, tool); err != nil {
		service.setInstallState(externaltools.ToolYTDLP, installStageError, verifyProgressEnd, err.Error())
		return dto.ExternalTool{}, err
	}
	service.setInstallState(externaltools.ToolYTDLP, installStageDone, verifyProgressEnd, "")
	_ = cleanupOldToolVersions(baseDir, externaltools.ToolYTDLP, version)
	return toExternalToolDTO(tool), nil
}

func (service *ExternalToolsService) installBun(ctx context.Context, version string) (dto.ExternalTool, error) {
	tag := normalizeBunReleaseTag(version)
	if tag == "" || tag == "latest" {
		resolved, err := getLatestGitHubTag(ctx, "oven-sh", "bun")
		if err != nil {
			service.setInstallState(externaltools.ToolBun, installStageError, downloadProgressStart, err.Error())
			return dto.ExternalTool{}, err
		}
		tag = resolved
	}
	filename, err := bunArchiveName(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		service.setInstallState(externaltools.ToolBun, installStageError, downloadProgressStart, err.Error())
		return dto.ExternalTool{}, err
	}
	downloadURL := fmt.Sprintf("https://github.com/oven-sh/bun/releases/download/%s/%s", tag, filename)
	baseDir, err := externalToolsBaseDir()
	if err != nil {
		service.setInstallState(externaltools.ToolBun, installStageError, downloadProgressStart, err.Error())
		return dto.ExternalTool{}, err
	}
	toolDir := filepath.Join(baseDir, string(externaltools.ToolBun), tag)
	execName := executableName(externaltools.ToolBun)
	execPath := filepath.Join(toolDir, execName)
	if err := downloadAndExtractWithProgress(
		ctx,
		downloadURL,
		toolDir,
		execName,
		func(progress int) {
			mapped := mapProgress(progress, downloadProgressStart, downloadProgressEnd)
			service.setInstallState(externaltools.ToolBun, installStageDownloading, mapped, "")
		},
		func(progress int) {
			mapped := mapProgress(progress, extractProgressStart, extractProgressEnd)
			service.setInstallState(externaltools.ToolBun, installStageExtracting, mapped, "")
		},
	); err != nil {
		service.setInstallState(externaltools.ToolBun, installStageError, downloadProgressStart, err.Error())
		return dto.ExternalTool{}, err
	}
	if err := markExecutable(execPath); err != nil {
		service.setInstallState(externaltools.ToolBun, installStageError, extractProgressEnd, err.Error())
		return dto.ExternalTool{}, err
	}
	service.setInstallState(externaltools.ToolBun, installStageVerifying, verifyProgressStart, "")
	resolvedVersion, err := resolveVersion(ctx, externaltools.ToolBun, execPath)
	if err != nil {
		service.setInstallState(externaltools.ToolBun, installStageError, verifyProgressStart, err.Error())
		return dto.ExternalTool{}, err
	}
	now := service.now()
	tool, err := externaltools.NewExternalTool(externaltools.ExternalToolParams{
		Name:        string(externaltools.ToolBun),
		ExecPath:    execPath,
		Version:     resolvedVersion,
		Status:      string(externaltools.StatusInstalled),
		InstalledAt: &now,
		UpdatedAt:   &now,
	})
	if err != nil {
		return dto.ExternalTool{}, err
	}
	if err := service.repo.Save(ctx, tool); err != nil {
		service.setInstallState(externaltools.ToolBun, installStageError, verifyProgressEnd, err.Error())
		return dto.ExternalTool{}, err
	}
	service.setInstallState(externaltools.ToolBun, installStageDone, verifyProgressEnd, "")
	_ = cleanupOldToolVersions(baseDir, externaltools.ToolBun, tag)
	return toExternalToolDTO(tool), nil
}

func (service *ExternalToolsService) installFFmpeg(ctx context.Context, version string) (dto.ExternalTool, error) {
	switch runtime.GOOS {
	case "darwin":
		return service.installFFmpegDarwin(ctx, version)
	case "windows":
		return service.installFFmpegWindows(ctx, version)
	default:
		service.setInstallState(externaltools.ToolFFmpeg, installStageError, downloadProgressStart, "ffmpeg auto-install only supported on macOS and Windows")
		return dto.ExternalTool{}, fmt.Errorf("ffmpeg auto-install is only supported on macOS and Windows for now")
	}
}

func (service *ExternalToolsService) installFFmpegDarwin(ctx context.Context, version string) (dto.ExternalTool, error) {
	if version == "" || version == "latest" {
		resolved, err := getLatestFFmpegVersion(ctx)
		if err != nil {
			service.setInstallState(externaltools.ToolFFmpeg, installStageError, downloadProgressStart, err.Error())
			return dto.ExternalTool{}, err
		}
		version = resolved
	}
	downloadURL := fmt.Sprintf("https://evermeet.cx/ffmpeg/ffmpeg-%s.zip", version)
	baseDir, err := externalToolsBaseDir()
	if err != nil {
		service.setInstallState(externaltools.ToolFFmpeg, installStageError, downloadProgressStart, err.Error())
		return dto.ExternalTool{}, err
	}
	toolDir := filepath.Join(baseDir, string(externaltools.ToolFFmpeg), version)
	execName := executableName(externaltools.ToolFFmpeg)
	execPath := filepath.Join(toolDir, execName)
	ffprobeName := executableNameForBinary("ffprobe")
	ffprobePath := filepath.Join(toolDir, ffprobeName)
	if err := downloadAndExtractWithProgress(
		ctx,
		downloadURL,
		toolDir,
		execName,
		func(progress int) {
			mapped := mapProgress(progress, downloadProgressStart, 60)
			service.setInstallState(externaltools.ToolFFmpeg, installStageDownloading, mapped, "")
		},
		func(progress int) {
			mapped := mapProgress(progress, 60, 75)
			service.setInstallState(externaltools.ToolFFmpeg, installStageExtracting, mapped, "")
		},
	); err != nil {
		service.setInstallState(externaltools.ToolFFmpeg, installStageError, downloadProgressStart, err.Error())
		return dto.ExternalTool{}, err
	}
	if err := markExecutable(execPath); err != nil {
		service.setInstallState(externaltools.ToolFFmpeg, installStageError, extractProgressEnd, err.Error())
		return dto.ExternalTool{}, err
	}
	ffprobeURL := fmt.Sprintf("https://evermeet.cx/ffmpeg/ffprobe-%s.zip", version)
	if err := downloadAndExtractWithProgress(
		ctx,
		ffprobeURL,
		toolDir,
		ffprobeName,
		func(progress int) {
			mapped := mapProgress(progress, 75, 88)
			service.setInstallState(externaltools.ToolFFmpeg, installStageDownloading, mapped, "")
		},
		func(progress int) {
			mapped := mapProgress(progress, 88, extractProgressEnd)
			service.setInstallState(externaltools.ToolFFmpeg, installStageExtracting, mapped, "")
		},
	); err != nil {
		service.setInstallState(externaltools.ToolFFmpeg, installStageError, downloadProgressStart, err.Error())
		return dto.ExternalTool{}, err
	}
	if err := markExecutable(ffprobePath); err != nil {
		service.setInstallState(externaltools.ToolFFmpeg, installStageError, extractProgressEnd, err.Error())
		return dto.ExternalTool{}, err
	}
	service.setInstallState(externaltools.ToolFFmpeg, installStageVerifying, verifyProgressStart, "")
	if _, err := validateFFmpegInstallation(ctx, execPath); err != nil {
		service.setInstallState(externaltools.ToolFFmpeg, installStageError, verifyProgressStart, err.Error())
		return dto.ExternalTool{}, err
	}
	now := service.now()
	tool, err := externaltools.NewExternalTool(externaltools.ExternalToolParams{
		Name:        string(externaltools.ToolFFmpeg),
		ExecPath:    execPath,
		Version:     version,
		Status:      string(externaltools.StatusInstalled),
		InstalledAt: &now,
		UpdatedAt:   &now,
	})
	if err != nil {
		return dto.ExternalTool{}, err
	}
	if err := service.repo.Save(ctx, tool); err != nil {
		service.setInstallState(externaltools.ToolFFmpeg, installStageError, verifyProgressEnd, err.Error())
		return dto.ExternalTool{}, err
	}
	service.setInstallState(externaltools.ToolFFmpeg, installStageDone, verifyProgressEnd, "")
	_ = cleanupOldToolVersions(baseDir, externaltools.ToolFFmpeg, version)
	return toExternalToolDTO(tool), nil
}

func (service *ExternalToolsService) installFFmpegWindows(ctx context.Context, version string) (dto.ExternalTool, error) {
	if version == "" || strings.EqualFold(version, "latest") {
		resolved, err := getLatestFFmpegVersion(ctx)
		if err != nil {
			service.setInstallState(externaltools.ToolFFmpeg, installStageError, downloadProgressStart, err.Error())
			return dto.ExternalTool{}, err
		}
		version = resolved
	}
	normalizedVersion := normalizeFFmpegVersion(version)
	downloadURL, err := buildWindowsFFmpegDownloadURL(normalizedVersion)
	if err != nil {
		service.setInstallState(externaltools.ToolFFmpeg, installStageError, downloadProgressStart, err.Error())
		return dto.ExternalTool{}, err
	}
	baseDir, err := externalToolsBaseDir()
	if err != nil {
		service.setInstallState(externaltools.ToolFFmpeg, installStageError, downloadProgressStart, err.Error())
		return dto.ExternalTool{}, err
	}
	toolDir := filepath.Join(baseDir, string(externaltools.ToolFFmpeg), normalizedVersion)
	if err := os.MkdirAll(toolDir, 0o755); err != nil {
		service.setInstallState(externaltools.ToolFFmpeg, installStageError, downloadProgressStart, err.Error())
		return dto.ExternalTool{}, err
	}
	archivePath := filepath.Join(toolDir, fmt.Sprintf("download-%d.zip", time.Now().UnixNano()))
	defer os.Remove(archivePath)
	if err := downloadFileWithProgress(ctx, downloadURL, archivePath, func(progress int) {
		mapped := mapProgress(progress, downloadProgressStart, downloadProgressEnd)
		service.setInstallState(externaltools.ToolFFmpeg, installStageDownloading, mapped, "")
	}); err != nil {
		service.setInstallState(externaltools.ToolFFmpeg, installStageError, downloadProgressStart, err.Error())
		return dto.ExternalTool{}, err
	}
	if !isZipFile(archivePath) && shouldApplyGitHubProxy(downloadURL) {
		_ = os.Remove(archivePath)
		if err := downloadFileWithProgressDirect(ctx, downloadURL, archivePath, func(progress int) {
			mapped := mapProgress(progress, downloadProgressStart, downloadProgressEnd)
			service.setInstallState(externaltools.ToolFFmpeg, installStageDownloading, mapped, "")
		}); err != nil {
			service.setInstallState(externaltools.ToolFFmpeg, installStageError, downloadProgressStart, err.Error())
			return dto.ExternalTool{}, err
		}
	}
	execName := executableName(externaltools.ToolFFmpeg)
	ffprobeName := executableNameForBinary("ffprobe")
	extracted, err := extractZipExecutables(archivePath, toolDir, []string{execName, ffprobeName}, func(progress int) {
		mapped := mapProgress(progress, extractProgressStart, extractProgressEnd)
		service.setInstallState(externaltools.ToolFFmpeg, installStageExtracting, mapped, "")
	})
	if err != nil {
		service.setInstallState(externaltools.ToolFFmpeg, installStageError, extractProgressStart, err.Error())
		return dto.ExternalTool{}, err
	}
	execPath := extracted[execName]
	if execPath == "" {
		service.setInstallState(externaltools.ToolFFmpeg, installStageError, extractProgressEnd, "ffmpeg executable not found in archive")
		return dto.ExternalTool{}, fmt.Errorf("ffmpeg executable not found in archive")
	}
	service.setInstallState(externaltools.ToolFFmpeg, installStageVerifying, verifyProgressStart, "")
	if _, err := validateFFmpegInstallation(ctx, execPath); err != nil {
		service.setInstallState(externaltools.ToolFFmpeg, installStageError, verifyProgressStart, err.Error())
		return dto.ExternalTool{}, err
	}
	now := service.now()
	tool, err := externaltools.NewExternalTool(externaltools.ExternalToolParams{
		Name:        string(externaltools.ToolFFmpeg),
		ExecPath:    execPath,
		Version:     normalizedVersion,
		Status:      string(externaltools.StatusInstalled),
		InstalledAt: &now,
		UpdatedAt:   &now,
	})
	if err != nil {
		return dto.ExternalTool{}, err
	}
	if err := service.repo.Save(ctx, tool); err != nil {
		service.setInstallState(externaltools.ToolFFmpeg, installStageError, verifyProgressEnd, err.Error())
		return dto.ExternalTool{}, err
	}
	service.setInstallState(externaltools.ToolFFmpeg, installStageDone, verifyProgressEnd, "")
	_ = cleanupOldToolVersions(baseDir, externaltools.ToolFFmpeg, normalizedVersion)
	return toExternalToolDTO(tool), nil
}

func (service *ExternalToolsService) installNPMRegistryTool(ctx context.Context, name externaltools.ToolName, source externalToolSource, version string, manager string) (dto.ExternalTool, error) {
	if source.Kind != sourceKindNPMRegistry {
		service.setInstallState(name, installStageError, downloadProgressStart, "unsupported source")
		return dto.ExternalTool{}, fmt.Errorf("unsupported source for tool %s", name)
	}
	manager = strings.ToLower(strings.TrimSpace(manager))
	if manager == "" {
		manager = strings.ToLower(strings.TrimSpace(source.Manager))
	}
	if manager == "" {
		manager = toolManagerNPM
	}
	if manager != toolManagerNPM && manager != toolManagerBun {
		service.setInstallState(name, installStageError, downloadProgressStart, fmt.Sprintf("manager %s not supported", manager))
		return dto.ExternalTool{}, fmt.Errorf("manager %s not supported for %s", manager, name)
	}
	if strings.TrimSpace(version) == "" || strings.EqualFold(strings.TrimSpace(version), "latest") {
		resolved, err := getLatestNPMVersion(ctx, source.SourceRef)
		if err != nil {
			service.setInstallState(name, installStageError, downloadProgressStart, err.Error())
			return dto.ExternalTool{}, err
		}
		version = resolved
	}
	baseDir, err := externalToolsBaseDir()
	if err != nil {
		service.setInstallState(name, installStageError, downloadProgressStart, err.Error())
		return dto.ExternalTool{}, err
	}
	toolDir := filepath.Join(baseDir, string(name), version)
	if err := os.MkdirAll(toolDir, 0o755); err != nil {
		service.setInstallState(name, installStageError, downloadProgressStart, err.Error())
		return dto.ExternalTool{}, err
	}
	service.setInstallState(name, installStageDownloading, downloadProgressStart, "")
	execPath := ""
	switch manager {
	case toolManagerBun:
		execPath, err = service.installBunManagedPackage(ctx, name, source.SourceRef, version, toolDir)
	case toolManagerNPM:
		if err = npmInstallPackage(ctx, source.SourceRef, version, toolDir); err == nil {
			execPath = filepath.Join(toolDir, "node_modules", ".bin", npmExecutableName(name))
		}
	}
	if err != nil {
		service.setInstallState(name, installStageError, downloadProgressStart, err.Error())
		return dto.ExternalTool{}, err
	}
	service.setInstallState(name, installStageVerifying, verifyProgressStart, "")
	resolvedVersion, err := resolveVersion(ctx, name, execPath)
	if err != nil {
		service.setInstallState(name, installStageError, verifyProgressStart, err.Error())
		return dto.ExternalTool{}, err
	}
	now := service.now()
	tool, err := externaltools.NewExternalTool(externaltools.ExternalToolParams{
		Name:        string(name),
		ExecPath:    execPath,
		Version:     resolvedVersion,
		Status:      string(externaltools.StatusInstalled),
		InstalledAt: &now,
		UpdatedAt:   &now,
	})
	if err != nil {
		return dto.ExternalTool{}, err
	}
	if err := service.repo.Save(ctx, tool); err != nil {
		service.setInstallState(name, installStageError, verifyProgressEnd, err.Error())
		return dto.ExternalTool{}, err
	}
	service.setInstallState(name, installStageDone, verifyProgressEnd, "")
	_ = cleanupOldToolVersions(baseDir, name, version)
	return toExternalToolDTO(tool), nil
}

func executableNameForBinary(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return ""
	}
	if runtime.GOOS == "windows" {
		return trimmed + ".exe"
	}
	return trimmed
}

func ffprobePathForFFmpegExec(execPath string) string {
	trimmed := strings.TrimSpace(execPath)
	if trimmed == "" {
		return executableNameForBinary("ffprobe")
	}
	return filepath.Join(filepath.Dir(trimmed), executableNameForBinary("ffprobe"))
}

func resolveInstalledToolVersion(ctx context.Context, name externaltools.ToolName, execPath string) (string, error) {
	if name == externaltools.ToolFFmpeg {
		version, err := validateFFmpegInstallation(ctx, execPath)
		if err != nil {
			return "", err
		}
		if managedVersion := managedToolVersionFromPath(name, execPath); managedVersion != "" {
			return managedVersion, nil
		}
		return version, nil
	}
	if managedVersion := managedToolVersionFromPath(name, execPath); managedVersion != "" {
		if _, err := resolveVersion(ctx, name, execPath); err != nil {
			return "", err
		}
		return managedVersion, nil
	}
	return resolveVersion(ctx, name, execPath)
}

func validateFFmpegInstallation(ctx context.Context, execPath string) (string, error) {
	trimmedExecPath := strings.TrimSpace(execPath)
	if trimmedExecPath == "" || !pathExists(trimmedExecPath) {
		return "", fmt.Errorf("ffmpeg is not installed")
	}
	ffprobePath := ffprobePathForFFmpegExec(trimmedExecPath)
	if !pathExists(ffprobePath) {
		return "", fmt.Errorf("ffprobe is not installed")
	}
	version, err := resolveVersion(ctx, externaltools.ToolFFmpeg, trimmedExecPath)
	if err != nil {
		return "", err
	}
	if _, err := resolveVersion(ctx, externaltools.ToolFFmpeg, ffprobePath); err != nil {
		return "", err
	}
	return version, nil
}

func normalizeFFmpegVersion(version string) string {
	trimmed := strings.TrimSpace(version)
	trimmed = strings.TrimPrefix(trimmed, "v")
	trimmed = strings.TrimPrefix(trimmed, "V")
	return strings.TrimSpace(trimmed)
}

func ffmpegReleaseTag(version string) string {
	normalized := normalizeFFmpegVersion(version)
	if normalized == "" {
		return ""
	}
	return "v" + normalized
}

func windowsFFmpegArchiveName(version string) string {
	normalized := normalizeFFmpegVersion(version)
	archSuffix := "win64"
	if runtime.GOARCH == "arm64" {
		archSuffix = "winarm64"
	}
	return fmt.Sprintf("jellyfin-ffmpeg_%s_portable_%s-clang-gpl", normalized, archSuffix)
}

func buildWindowsFFmpegDownloadURL(version string) (string, error) {
	normalized := normalizeFFmpegVersion(version)
	if normalized == "" {
		return "", fmt.Errorf("ffmpeg version is required")
	}
	return fmt.Sprintf(
		"https://github.com/jellyfin/jellyfin-ffmpeg/releases/download/%s/%s.zip",
		ffmpegReleaseTag(normalized),
		windowsFFmpegArchiveName(normalized),
	), nil
}

func toExternalToolDTO(tool externaltools.ExternalTool) dto.ExternalTool {
	installedAt := ""
	if tool.InstalledAt != nil {
		installedAt = tool.InstalledAt.Format(time.RFC3339)
	}
	version := strings.TrimSpace(tool.Version)
	if managedVersion := managedToolVersionFromPath(tool.Name, tool.ExecPath); managedVersion != "" {
		version = managedVersion
	}
	toolKind, sourceKind, sourceRef, manager := toolSourceMetadata(tool.Name)
	return dto.ExternalTool{
		Name:        string(tool.Name),
		Kind:        toolKind,
		ExecPath:    tool.ExecPath,
		Version:     version,
		Status:      string(tool.Status),
		SourceKind:  sourceKind,
		SourceRef:   sourceRef,
		Manager:     manager,
		InstalledAt: installedAt,
		UpdatedAt:   tool.UpdatedAt.Format(time.RFC3339),
	}
}

func resolveToolUpdate(ctx context.Context, name externaltools.ToolName) (string, string, string, error) {
	source, err := resolveToolSource(name)
	if err != nil {
		return "", "", "", err
	}
	switch source.Kind {
	case sourceKindGitHubRelease:
		switch name {
		case externaltools.ToolYTDLP:
			release, err := getLatestGitHubRelease(ctx, "yt-dlp", "yt-dlp")
			if err != nil {
				return "", "", "", err
			}
			notes := strings.TrimSpace(release.Body)
			notesURL := ""
			if notes != "" {
				notesURL = release.HTMLURL
			}
			return release.TagName, notes, notesURL, nil
		case externaltools.ToolBun:
			release, err := getLatestGitHubRelease(ctx, "oven-sh", "bun")
			if err != nil {
				return "", "", "", err
			}
			notes := strings.TrimSpace(release.Body)
			notesURL := ""
			if notes != "" {
				notesURL = release.HTMLURL
			}
			return normalizeBunVersion(release.TagName), notes, notesURL, nil
		case externaltools.ToolFFmpeg:
			if runtime.GOOS == "windows" {
				release, err := getLatestGitHubRelease(ctx, "jellyfin", "jellyfin-ffmpeg")
				if err != nil {
					return "", "", "", err
				}
				notes := strings.TrimSpace(release.Body)
				notesURL := ""
				if notes != "" {
					notesURL = release.HTMLURL
				}
				return normalizeFFmpegVersion(release.TagName), notes, notesURL, nil
			}
			version, err := getLatestFFmpegVersion(ctx)
			if err != nil {
				return "", "", "", err
			}
			return version, "", "", nil
		default:
			return "", "", "", externaltools.ErrInvalidTool
		}
	case sourceKindNPMRegistry:
		latest, err := getLatestNPMVersion(ctx, source.SourceRef)
		if err != nil {
			return "", "", "", err
		}
		notesURL := fmt.Sprintf("https://www.npmjs.com/package/%s/v/%s", source.SourceRef, latest)
		return latest, "", notesURL, nil
	case sourceKindRuntime:
		return "", "", "", nil
	default:
		return "", "", "", fmt.Errorf("unsupported source for tool %s", name)
	}
}

func resolveToolSource(name externaltools.ToolName) (externalToolSource, error) {
	source, ok := toolSources[name]
	if !ok {
		return externalToolSource{}, externaltools.ErrInvalidTool
	}
	return source, nil
}

func toolSourceMetadata(name externaltools.ToolName) (string, string, string, string) {
	source, err := resolveToolSource(name)
	if err != nil {
		return "", "", "", ""
	}
	return source.ToolKind, source.Kind, source.SourceRef, source.Manager
}

func npmExecutableName(name externaltools.ToolName) string {
	if runtime.GOOS == "windows" {
		return string(name) + ".cmd"
	}
	return string(name)
}

func npmInstallPackage(ctx context.Context, packageName string, version string, installDir string) error {
	pkg := strings.TrimSpace(packageName)
	if pkg == "" {
		return fmt.Errorf("package name is required")
	}
	ref := pkg
	if strings.TrimSpace(version) != "" {
		ref = fmt.Sprintf("%s@%s", pkg, strings.TrimSpace(version))
	}
	command := exec.CommandContext(ctx, "npm", "install", "--no-audit", "--no-fund", "--prefix", installDir, ref)
	configureCommand(command)
	output, err := command.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = err.Error()
		}
		return fmt.Errorf("npm install failed: %s", message)
	}
	return nil
}

func (service *ExternalToolsService) installBunManagedPackage(
	ctx context.Context,
	name externaltools.ToolName,
	packageName string,
	version string,
	installDir string,
) (string, error) {
	bunExecPath, err := service.resolvePackageManagerExecPath(ctx, toolManagerBun)
	if err != nil {
		return "", err
	}
	if err := ensureBunPackageManifest(installDir, name); err != nil {
		return "", err
	}
	if err := bunInstallPackage(ctx, bunExecPath, packageName, version, installDir); err != nil {
		return "", err
	}
	binScriptPath, err := resolveInstalledPackageBinPath(installDir, packageName, string(name))
	if err != nil {
		return "", err
	}
	return writeBunPackageWrapper(installDir, name, bunExecPath, binScriptPath)
}

func (service *ExternalToolsService) resolvePackageManagerExecPath(ctx context.Context, manager string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(manager)) {
	case toolManagerBun:
		if execPath, err := service.ResolveExecPath(ctx, externaltools.ToolBun); err == nil && pathExists(execPath) {
			return execPath, nil
		}
		installed, err := service.installBun(ctx, "latest")
		if err != nil {
			return "", err
		}
		if strings.TrimSpace(installed.ExecPath) == "" {
			return "", fmt.Errorf("bun exec path not resolved")
		}
		return installed.ExecPath, nil
	case toolManagerNPM:
		execPath, err := exec.LookPath("npm")
		if err != nil {
			return "", fmt.Errorf("npm not found")
		}
		return execPath, nil
	default:
		return "", fmt.Errorf("manager %s not supported", manager)
	}
}

func bunInstallPackage(ctx context.Context, bunExecPath string, packageName string, version string, installDir string) error {
	pkg := strings.TrimSpace(packageName)
	if pkg == "" {
		return fmt.Errorf("package name is required")
	}
	ref := pkg
	if strings.TrimSpace(version) != "" {
		ref = fmt.Sprintf("%s@%s", pkg, strings.TrimSpace(version))
	}
	command := exec.CommandContext(ctx, bunExecPath, "add", "--cwd", installDir, "--exact", ref)
	configureCommand(command)
	output, err := command.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = err.Error()
		}
		return fmt.Errorf("bun add failed: %s", message)
	}
	return nil
}

func ensureBunPackageManifest(installDir string, name externaltools.ToolName) error {
	manifestPath := filepath.Join(installDir, "package.json")
	if pathExists(manifestPath) {
		return nil
	}
	manifest := fmt.Sprintf("{\n  \"name\": \"dreamcreator-external-%s\",\n  \"private\": true\n}\n", sanitizePackageName(string(name)))
	return os.WriteFile(manifestPath, []byte(manifest), 0o644)
}

func sanitizePackageName(value string) string {
	replacer := strings.NewReplacer("/", "-", "\\", "-", "@", "", " ", "-")
	return replacer.Replace(strings.TrimSpace(value))
}

func resolveInstalledPackageBinPath(installDir string, packageName string, binaryName string) (string, error) {
	manifestPath := filepath.Join(nodeModulesPackageDir(installDir, packageName), "package.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return "", err
	}
	var manifest struct {
		Bin any `json:"bin"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return "", err
	}
	resolveCandidate := func(relativePath string) (string, error) {
		trimmed := strings.TrimSpace(relativePath)
		if trimmed == "" {
			return "", fmt.Errorf("package %s bin entry is empty", packageName)
		}
		candidate := filepath.Join(nodeModulesPackageDir(installDir, packageName), filepath.FromSlash(trimmed))
		if !pathExists(candidate) {
			return "", fmt.Errorf("package %s bin %s not found", packageName, trimmed)
		}
		return candidate, nil
	}
	switch raw := manifest.Bin.(type) {
	case string:
		return resolveCandidate(raw)
	case map[string]any:
		if selected, ok := raw[binaryName].(string); ok {
			return resolveCandidate(selected)
		}
		if len(raw) == 1 {
			for _, value := range raw {
				if selected, ok := value.(string); ok {
					return resolveCandidate(selected)
				}
			}
		}
	}
	return "", fmt.Errorf("package %s bin entry missing", packageName)
}

func nodeModulesPackageDir(installDir string, packageName string) string {
	parts := append([]string{installDir, "node_modules"}, strings.Split(strings.TrimSpace(packageName), "/")...)
	return filepath.Join(parts...)
}

func writeBunPackageWrapper(installDir string, name externaltools.ToolName, bunExecPath string, scriptPath string) (string, error) {
	execPath := filepath.Join(installDir, npmExecutableName(name))
	var content string
	if runtime.GOOS == "windows" {
		content = fmt.Sprintf("@echo off\r\n\"%s\" \"%s\" %%*\r\n", escapeCmdArgument(bunExecPath), escapeCmdArgument(scriptPath))
	} else {
		content = fmt.Sprintf("#!/bin/sh\nexec \"%s\" \"%s\" \"$@\"\n", escapeShellDoubleQuotes(bunExecPath), escapeShellDoubleQuotes(scriptPath))
	}
	if err := os.WriteFile(execPath, []byte(content), 0o755); err != nil {
		return "", err
	}
	if err := markExecutable(execPath); err != nil {
		return "", err
	}
	return execPath, nil
}

func escapeShellDoubleQuotes(value string) string {
	return strings.ReplaceAll(value, `"`, `\"`)
}

func escapeCmdArgument(value string) string {
	return strings.ReplaceAll(value, `"`, `""`)
}

func externalToolsBaseDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(configDir, "dreamcreator", "external-tools")
	if err := os.MkdirAll(path, 0o755); err != nil {
		return "", err
	}
	return path, nil
}

func executableName(name externaltools.ToolName) string {
	switch runtime.GOOS {
	case "windows":
		return fmt.Sprintf("%s.exe", name)
	default:
		return string(name)
	}
}

func ytdlpFilename(osType string) string {
	switch osType {
	case "windows":
		return "yt-dlp.exe"
	case "darwin":
		return "yt-dlp_macos"
	default:
		return "yt-dlp"
	}
}

func bunArchiveName(osType, arch string) (string, error) {
	switch osType {
	case "windows":
		switch arch {
		case "amd64":
			return "bun-windows-x64.zip", nil
		default:
			return "", fmt.Errorf("bun auto-install unsupported on %s/%s", osType, arch)
		}
	case "darwin":
		switch arch {
		case "arm64":
			return "bun-darwin-aarch64.zip", nil
		default:
			return "bun-darwin-x64.zip", nil
		}
	default:
		switch arch {
		case "arm64":
			return "bun-linux-aarch64.zip", nil
		default:
			return "bun-linux-x64.zip", nil
		}
	}
}

func normalizeBunReleaseTag(version string) string {
	trimmed := strings.TrimSpace(version)
	if trimmed == "" {
		return ""
	}
	if strings.EqualFold(trimmed, "latest") {
		return "latest"
	}
	normalized := normalizeBunVersion(trimmed)
	if normalized == "" {
		return ""
	}
	return "bun-v" + normalized
}

func normalizeBunVersion(version string) string {
	trimmed := strings.TrimSpace(version)
	trimmed = strings.TrimPrefix(trimmed, "bun-v")
	trimmed = strings.TrimPrefix(trimmed, "bun-V")
	trimmed = strings.TrimPrefix(trimmed, "v")
	trimmed = strings.TrimPrefix(trimmed, "V")
	return strings.TrimSpace(trimmed)
}

func downloadFile(ctx context.Context, url string, destPath string) error {
	return downloadFileWithProgress(ctx, url, destPath, nil)
}

func shouldApplyGitHubProxy(raw string) bool {
	if raw == "" {
		return false
	}
	if strings.HasPrefix(raw, "https://ghproxy.com/") {
		return false
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return false
	}
	host := strings.ToLower(parsed.Host)
	return host == "github.com" || host == "raw.githubusercontent.com"
}

func applyGitHubProxy(raw string) string {
	if !shouldApplyGitHubProxy(raw) {
		return raw
	}
	return "https://ghproxy.com/" + raw
}

func downloadFileWithProgress(ctx context.Context, url string, destPath string, progress func(int)) error {
	err := downloadFileWithProgressInternal(ctx, url, destPath, progress, true)
	if err == nil || !shouldApplyGitHubProxy(url) {
		return err
	}
	return downloadFileWithProgressInternal(ctx, url, destPath, progress, false)
}

func downloadFileWithProgressDirect(ctx context.Context, url string, destPath string, progress func(int)) error {
	return downloadFileWithProgressInternal(ctx, url, destPath, progress, false)
}

func downloadFromSourcesWithProgress(ctx context.Context, urls []string, destPath string, progress func(int)) error {
	if len(urls) == 0 {
		return errors.New("download url is empty")
	}
	var lastErr error
	for _, rawURL := range urls {
		url := strings.TrimSpace(rawURL)
		if url == "" {
			continue
		}
		if err := os.Remove(destPath); err != nil && !os.IsNotExist(err) {
			return err
		}
		lastErr = downloadFileWithProgressDirect(ctx, url, destPath, progress)
		if lastErr == nil {
			return nil
		}
	}
	if lastErr == nil {
		lastErr = errors.New("download url is empty")
	}
	return lastErr
}

func downloadFileWithProgressInternal(ctx context.Context, url string, destPath string, progress func(int), useProxy bool) error {
	if url == "" {
		return errors.New("download url is empty")
	}
	if useProxy {
		url = applyGitHubProxy(url)
	}
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return err
	}
	client := &http.Client{Timeout: defaultDownloadTimeout}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("download failed: %s", resp.Status)
	}
	tmpPath := destPath + "." + uuid.NewString() + ".tmp"
	out, err := os.Create(tmpPath)
	if err != nil {
		return err
	}
	defer out.Close()

	total := resp.ContentLength
	var written int64
	buf := make([]byte, 32*1024)
	lastReport := time.Now()

	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, err := out.Write(buf[:n]); err != nil {
				_ = os.Remove(tmpPath)
				return err
			}
			written += int64(n)
		}
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			_ = os.Remove(tmpPath)
			return readErr
		}
		if progress != nil && (time.Since(lastReport) > 200*time.Millisecond || written == total) {
			progress(percent(written, total))
			lastReport = time.Now()
		}
	}

	if total > 0 && written != total {
		_ = out.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("download incomplete: expected %d bytes, got %d", total, written)
	}
	if progress != nil {
		progress(100)
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, destPath); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return nil
}

func downloadAndExtract(ctx context.Context, url, destDir, execName string) error {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}
	if !strings.HasSuffix(url, ".zip") {
		return downloadFile(ctx, url, filepath.Join(destDir, execName))
	}
	archivePath := filepath.Join(destDir, fmt.Sprintf("download-%d.zip", time.Now().UnixNano()))
	if err := downloadFile(ctx, url, archivePath); err != nil {
		return err
	}
	defer os.Remove(archivePath)
	extractedPath, err := extractZipExecutable(archivePath, destDir, execName, nil)
	if err != nil {
		return err
	}
	targetPath := filepath.Join(destDir, execName)
	if extractedPath != targetPath {
		if err := os.Rename(extractedPath, targetPath); err != nil {
			return err
		}
	}
	return nil
}

func cleanupOldToolVersions(baseDir string, name externaltools.ToolName, keepVersion string) error {
	if strings.TrimSpace(baseDir) == "" || strings.TrimSpace(keepVersion) == "" {
		return nil
	}
	root := filepath.Join(baseDir, string(name))
	entries, err := os.ReadDir(root)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == keepVersion {
			continue
		}
		_ = os.RemoveAll(filepath.Join(root, entry.Name()))
	}
	return nil
}

func downloadAndExtractWithProgress(
	ctx context.Context,
	url,
	destDir,
	execName string,
	downloadProgress func(int),
	extractProgress func(int),
) error {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}
	if !strings.HasSuffix(url, ".zip") {
		return downloadFileWithProgress(ctx, url, filepath.Join(destDir, execName), downloadProgress)
	}
	archivePath := filepath.Join(destDir, fmt.Sprintf("download-%d.zip", time.Now().UnixNano()))
	if err := downloadFileWithProgress(ctx, url, archivePath, downloadProgress); err != nil {
		return err
	}
	defer os.Remove(archivePath)
	if !isZipFile(archivePath) && shouldApplyGitHubProxy(url) {
		_ = os.Remove(archivePath)
		if err := downloadFileWithProgressDirect(ctx, url, archivePath, downloadProgress); err != nil {
			return err
		}
	}
	if extractProgress != nil {
		extractProgress(0)
	}
	extractedPath, err := extractZipExecutable(archivePath, destDir, execName, extractProgress)
	if err != nil {
		return err
	}
	targetPath := filepath.Join(destDir, execName)
	if extractedPath != targetPath {
		if err := os.Rename(extractedPath, targetPath); err != nil {
			return err
		}
	}
	return nil
}

func isZipFile(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()
	header := make([]byte, 4)
	if _, err := io.ReadFull(file, header); err != nil {
		return false
	}
	return header[0] == 'P' && header[1] == 'K' &&
		((header[2] == 3 && header[3] == 4) ||
			(header[2] == 5 && header[3] == 6) ||
			(header[2] == 7 && header[3] == 8))
}

func validateDownloadedExecutable(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	header := make([]byte, 512)
	n, readErr := io.ReadFull(file, header)
	if readErr != nil && readErr != io.ErrUnexpectedEOF {
		return readErr
	}
	header = header[:n]
	if len(header) < 4 {
		return fmt.Errorf("downloaded file is too small")
	}
	if looksLikeHTML(header) {
		return fmt.Errorf("downloaded file is HTML, likely blocked or redirected")
	}
	if looksLikeZip(header) {
		return fmt.Errorf("downloaded file is archive, expected executable")
	}
	switch runtime.GOOS {
	case "windows":
		if header[0] == 'M' && header[1] == 'Z' {
			return nil
		}
		return fmt.Errorf("downloaded file is not a Windows executable")
	case "darwin":
		if looksLikeMachO(header) {
			return nil
		}
		return fmt.Errorf("downloaded file is not a macOS executable")
	default:
		if looksLikeELF(header) {
			return nil
		}
		return fmt.Errorf("downloaded file is not a Linux executable")
	}
}

func looksLikeHTML(header []byte) bool {
	lower := strings.ToLower(string(header))
	return strings.Contains(lower, "<!doctype") ||
		strings.Contains(lower, "<html") ||
		strings.Contains(lower, "<head") ||
		strings.Contains(lower, "<body")
}

func looksLikeZip(header []byte) bool {
	if len(header) < 4 {
		return false
	}
	return header[0] == 'P' && header[1] == 'K' &&
		((header[2] == 3 && header[3] == 4) ||
			(header[2] == 5 && header[3] == 6) ||
			(header[2] == 7 && header[3] == 8))
}

func looksLikeELF(header []byte) bool {
	return len(header) >= 4 && header[0] == 0x7f && header[1] == 'E' && header[2] == 'L' && header[3] == 'F'
}

func looksLikeMachO(header []byte) bool {
	if len(header) < 4 {
		return false
	}
	magicBE := binary.BigEndian.Uint32(header[:4])
	magicLE := binary.LittleEndian.Uint32(header[:4])
	switch magicBE {
	case 0xFEEDFACE, 0xFEEDFACF, 0xCAFEBABE:
		return true
	}
	switch magicLE {
	case 0xCEFAEDFE, 0xCFFAEDFE, 0xBEBAFECA:
		return true
	}
	return false
}

func extractZipExecutable(archivePath, destDir, execName string, progress func(int)) (string, error) {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return "", err
	}
	defer reader.Close()
	var candidate string
	total := len(reader.File)
	for i, file := range reader.File {
		path := filepath.Join(destDir, file.Name)
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(path, 0o755); err != nil {
				return "", err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return "", err
		}
		src, err := file.Open()
		if err != nil {
			return "", err
		}
		dst, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, file.Mode())
		if err != nil {
			_ = src.Close()
			return "", err
		}
		if _, err := io.Copy(dst, src); err != nil {
			_ = dst.Close()
			_ = src.Close()
			return "", err
		}
		_ = dst.Close()
		_ = src.Close()
		if strings.EqualFold(filepath.Base(path), execName) {
			candidate = path
		}
		if progress != nil {
			progress(percent(int64(i+1), int64(total)))
		}
	}
	if candidate == "" {
		return "", fmt.Errorf("executable %s not found in archive", execName)
	}
	return candidate, nil
}

func extractZipExecutables(archivePath, destDir string, execNames []string, progress func(int)) (map[string]string, error) {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	targets := make(map[string]string, len(execNames))
	for _, execName := range execNames {
		trimmed := strings.TrimSpace(execName)
		if trimmed == "" {
			continue
		}
		targets[strings.ToLower(trimmed)] = trimmed
	}
	found := make(map[string]string, len(targets))
	total := len(reader.File)

	for i, file := range reader.File {
		path := filepath.Join(destDir, file.Name)
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(path, 0o755); err != nil {
				return nil, err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, err
		}
		src, err := file.Open()
		if err != nil {
			return nil, err
		}
		dst, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, file.Mode())
		if err != nil {
			_ = src.Close()
			return nil, err
		}
		if _, err := io.Copy(dst, src); err != nil {
			_ = dst.Close()
			_ = src.Close()
			return nil, err
		}
		_ = dst.Close()
		_ = src.Close()
		if originalName, ok := targets[strings.ToLower(filepath.Base(path))]; ok {
			found[originalName] = path
		}
		if progress != nil {
			progress(percent(int64(i+1), int64(total)))
		}
	}

	for _, execName := range execNames {
		trimmed := strings.TrimSpace(execName)
		if trimmed == "" {
			continue
		}
		if found[trimmed] == "" {
			return nil, fmt.Errorf("executable %s not found in archive", trimmed)
		}
	}

	return found, nil
}

func extractTarXZExecutables(archivePath, destDir string, execNames []string, progress func(int)) (map[string]string, error) {
	tarPath, err := exec.LookPath("tar")
	if err != nil {
		return nil, fmt.Errorf("tar.xz archives are not supported on this system")
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return nil, err
	}

	command := exec.Command(tarPath, "-xJf", archivePath, "-C", destDir)
	configureCommand(command)
	output, err := command.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("extract tar.xz archive: %w %s", err, strings.TrimSpace(string(output)))
	}
	if progress != nil {
		progress(100)
	}

	targets := make(map[string]string, len(execNames))
	for _, execName := range execNames {
		trimmed := strings.TrimSpace(execName)
		if trimmed == "" {
			continue
		}
		targets[strings.ToLower(trimmed)] = trimmed
	}
	found := make(map[string]string, len(targets))
	walkErr := filepath.WalkDir(destDir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if originalName, ok := targets[strings.ToLower(entry.Name())]; ok && found[originalName] == "" {
			found[originalName] = path
		}
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}
	for _, execName := range execNames {
		trimmed := strings.TrimSpace(execName)
		if trimmed == "" {
			continue
		}
		if found[trimmed] == "" {
			return nil, fmt.Errorf("executable %s not found in archive", trimmed)
		}
	}
	return found, nil
}

func markExecutable(path string) error {
	if runtime.GOOS == "windows" {
		return nil
	}
	if err := os.Chmod(path, 0o755); err != nil {
		return err
	}
	if runtime.GOOS == "darwin" {
		_ = exec.Command("xattr", "-dr", "com.apple.quarantine", path).Run()
	}
	return nil
}

func resolveVersion(ctx context.Context, name externaltools.ToolName, execPath string) (string, error) {
	var args []string
	switch name {
	case externaltools.ToolFFmpeg:
		args = []string{"-version"}
	case externaltools.ToolClawHub:
		args = []string{"--cli-version"}
	default:
		args = []string{"--version"}
	}
	command := exec.CommandContext(ctx, execPath, args...)
	if runtime.GOOS == "windows" && strings.HasSuffix(strings.ToLower(execPath), ".cmd") {
		command = exec.CommandContext(ctx, "cmd", append([]string{"/c", execPath}, args...)...)
	}
	configureCommand(command)
	output, err := command.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			return "", err
		}
		return "", fmt.Errorf("%w: %s", err, message)
	}
	text := strings.TrimSpace(string(output))
	if text == "" {
		return "", fmt.Errorf("empty version output")
	}
	switch name {
	case externaltools.ToolFFmpeg:
		return parseFFmpegVersion(text)
	case externaltools.ToolClawHub:
		return parseClawHubVersion(text)
	default:
		return strings.Fields(text)[0], nil
	}
}

func parseFFmpegVersion(output string) (string, error) {
	lines := strings.Split(output, "\n")
	if len(lines) == 0 {
		return "", fmt.Errorf("ffmpeg version output empty")
	}
	fields := strings.Fields(lines[0])
	for i, field := range fields {
		if strings.EqualFold(field, "version") && i+1 < len(fields) {
			return strings.TrimSpace(fields[i+1]), nil
		}
	}
	return "", fmt.Errorf("ffmpeg version not found")
}

func parseClawHubVersion(output string) (string, error) {
	for _, token := range strings.Fields(output) {
		candidate := strings.Trim(strings.TrimSpace(token), ",;:()[]{}")
		if semverTokenPattern.MatchString(candidate) {
			return strings.TrimPrefix(candidate, "v"), nil
		}
	}
	return "", fmt.Errorf("clawhub version not found")
}

func percent(written int64, total int64) int {
	if total <= 0 {
		return 0
	}
	p := int(float64(written) / float64(total) * 100)
	if p > 100 {
		return 100
	}
	if p < 0 {
		return 0
	}
	return p
}

func mapProgress(progress int, start, end int) int {
	if progress < 0 {
		progress = 0
	}
	if progress > 100 {
		progress = 100
	}
	if start >= end {
		return start
	}
	return start + int(float64(progress)*(float64(end-start))/100.0)
}

func getLatestGitHubTag(ctx context.Context, owner, repo string) (string, error) {
	release, err := getLatestGitHubRelease(ctx, owner, repo)
	if err != nil {
		return "", err
	}
	if release.TagName == "" {
		return "", fmt.Errorf("latest release tag is empty")
	}
	return release.TagName, nil
}

type githubRelease struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
	Body    string `json:"body"`
}

func getLatestGitHubRelease(ctx context.Context, owner, repo string) (githubRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)
	client := &http.Client{Timeout: defaultDownloadTimeout}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return githubRelease{}, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return githubRelease{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return githubRelease{}, fmt.Errorf("failed to fetch latest release: %s", resp.Status)
	}
	var payload githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return githubRelease{}, err
	}
	if payload.TagName == "" {
		return githubRelease{}, fmt.Errorf("latest release tag is empty")
	}
	return payload, nil
}

func getLatestNPMVersion(ctx context.Context, packageName string) (string, error) {
	name := strings.TrimSpace(packageName)
	if name == "" {
		return "", fmt.Errorf("npm package name is required")
	}
	endpoint := fmt.Sprintf("https://registry.npmjs.org/%s", url.PathEscape(name))
	client := &http.Client{Timeout: defaultDownloadTimeout}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch npm package metadata: %s", resp.Status)
	}
	var payload struct {
		DistTags map[string]string `json:"dist-tags"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", err
	}
	latest := strings.TrimSpace(payload.DistTags["latest"])
	if latest == "" {
		return "", fmt.Errorf("latest npm version not found")
	}
	return latest, nil
}

func getLatestFFmpegVersion(ctx context.Context) (string, error) {
	if runtime.GOOS == "windows" {
		release, err := getLatestGitHubRelease(ctx, "jellyfin", "jellyfin-ffmpeg")
		if err != nil {
			return "", err
		}
		return normalizeFFmpegVersion(release.TagName), nil
	}
	url := "https://evermeet.cx/ffmpeg/info/ffmpeg/snapshot"
	client := &http.Client{Timeout: defaultDownloadTimeout}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch ffmpeg version: %s", resp.Status)
	}
	var payload struct {
		Version string `json:"version"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", err
	}
	if payload.Version == "" {
		return "", fmt.Errorf("ffmpeg version not found")
	}
	return payload.Version, nil
}

func pathExists(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}
