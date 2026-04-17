package service

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"dreamcreator/internal/application/externaltools/dto"
	"dreamcreator/internal/domain/externaltools"
)

type memoryRepo struct {
	items map[string]externaltools.ExternalTool
}

func newMemoryRepo() *memoryRepo {
	return &memoryRepo{items: make(map[string]externaltools.ExternalTool)}
}

func (repo *memoryRepo) List(_ context.Context) ([]externaltools.ExternalTool, error) {
	result := make([]externaltools.ExternalTool, 0, len(repo.items))
	for _, item := range repo.items {
		result = append(result, item)
	}
	return result, nil
}

func (repo *memoryRepo) Get(_ context.Context, name string) (externaltools.ExternalTool, error) {
	item, ok := repo.items[name]
	if !ok {
		return externaltools.ExternalTool{}, externaltools.ErrToolNotFound
	}
	return item, nil
}

func (repo *memoryRepo) Save(_ context.Context, tool externaltools.ExternalTool) error {
	repo.items[string(tool.Name)] = tool
	return nil
}

func (repo *memoryRepo) Delete(_ context.Context, name string) error {
	delete(repo.items, name)
	return nil
}

func TestEnsureDefaultsIncludesBunAndClawHub(t *testing.T) {
	t.Parallel()

	repo := newMemoryRepo()
	service := NewExternalToolsService(repo, nil, "")
	if err := service.EnsureDefaults(context.Background()); err != nil {
		t.Fatalf("ensure defaults failed: %v", err)
	}
	for _, name := range []externaltools.ToolName{
		externaltools.ToolYTDLP,
		externaltools.ToolFFmpeg,
		externaltools.ToolBun,
		externaltools.ToolClawHub,
	} {
		if _, err := repo.Get(context.Background(), string(name)); err != nil {
			t.Fatalf("expected default tool %s: %v", name, err)
		}
	}
}

func TestToolReadinessReasons(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := newMemoryRepo()
	service := NewExternalToolsService(repo, nil, "")
	if err := service.EnsureDefaults(ctx); err != nil {
		t.Fatalf("ensure defaults failed: %v", err)
	}

	ready, reason, err := service.ToolReadiness(ctx, externaltools.ToolClawHub)
	if err != nil {
		t.Fatalf("tool readiness failed: %v", err)
	}
	if ready {
		t.Fatalf("expected clawhub to be not ready by default")
	}
	if reason != "not_installed" {
		t.Fatalf("unexpected reason: %s", reason)
	}

	now := time.Now()
	missingPathTool, err := externaltools.NewExternalTool(externaltools.ExternalToolParams{
		Name:      string(externaltools.ToolClawHub),
		ExecPath:  filepath.Join(t.TempDir(), "missing-clawhub"),
		Version:   "1.2.3",
		Status:    string(externaltools.StatusInstalled),
		UpdatedAt: &now,
	})
	if err != nil {
		t.Fatalf("new external tool failed: %v", err)
	}
	if err := repo.Save(ctx, missingPathTool); err != nil {
		t.Fatalf("save tool failed: %v", err)
	}

	ready, reason, err = service.ToolReadiness(ctx, externaltools.ToolClawHub)
	if err != nil {
		t.Fatalf("tool readiness failed: %v", err)
	}
	if ready {
		t.Fatalf("expected clawhub to be not ready when path is missing")
	}
	if reason != "exec_not_found" {
		t.Fatalf("unexpected reason: %s", reason)
	}

	execDir := t.TempDir()
	execPath := filepath.Join(execDir, "clawhub")
	if err := os.WriteFile(execPath, []byte("#!/bin/sh\necho clawhub\n"), 0o755); err != nil {
		t.Fatalf("write exec file failed: %v", err)
	}
	readyTool, err := externaltools.NewExternalTool(externaltools.ExternalToolParams{
		Name:      string(externaltools.ToolClawHub),
		ExecPath:  execPath,
		Version:   "1.2.3",
		Status:    string(externaltools.StatusInstalled),
		UpdatedAt: &now,
	})
	if err != nil {
		t.Fatalf("new external tool failed: %v", err)
	}
	if err := repo.Save(ctx, readyTool); err != nil {
		t.Fatalf("save tool failed: %v", err)
	}
	ready, reason, err = service.ToolReadiness(ctx, externaltools.ToolClawHub)
	if err != nil {
		t.Fatalf("tool readiness failed: %v", err)
	}
	if !ready {
		t.Fatalf("expected clawhub to be ready")
	}
	if reason != "" {
		t.Fatalf("expected empty reason, got: %s", reason)
	}
}

func TestParseClawHubVersionFromUsageText(t *testing.T) {
	t.Parallel()

	version, err := parseClawHubVersion("ClawHub CLI v0.7.0 (89bd7bf6)")
	if err != nil {
		t.Fatalf("parse clawhub version failed: %v", err)
	}
	if version != "0.7.0" {
		t.Fatalf("unexpected version: %s", version)
	}
}

func TestListToolsMarksFFmpegInvalidWhenFFprobeMissing(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := newMemoryRepo()
	service := NewExternalToolsService(repo, nil, "")
	now := time.Now()

	execDir := t.TempDir()
	ffmpegPath := filepath.Join(execDir, executableNameForBinary("ffmpeg"))
	if err := os.WriteFile(ffmpegPath, []byte("stub"), 0o755); err != nil {
		t.Fatalf("write ffmpeg stub failed: %v", err)
	}
	tool, err := externaltools.NewExternalTool(externaltools.ExternalToolParams{
		Name:      string(externaltools.ToolFFmpeg),
		ExecPath:  ffmpegPath,
		Version:   "7.1.1",
		Status:    string(externaltools.StatusInstalled),
		UpdatedAt: &now,
	})
	if err != nil {
		t.Fatalf("new external tool failed: %v", err)
	}
	if err := repo.Save(ctx, tool); err != nil {
		t.Fatalf("save tool failed: %v", err)
	}

	items, err := service.ListTools(ctx)
	if err != nil {
		t.Fatalf("list tools failed: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(items))
	}
	if items[0].Status != string(externaltools.StatusInvalid) {
		t.Fatalf("expected ffmpeg to be invalid when ffprobe is missing, got %s", items[0].Status)
	}
}

func TestSetToolPathFFmpegRequiresFFprobe(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := newMemoryRepo()
	service := NewExternalToolsService(repo, nil, "")

	execDir := t.TempDir()
	ffmpegPath := filepath.Join(execDir, executableNameForBinary("ffmpeg"))
	ffprobePath := filepath.Join(execDir, executableNameForBinary("ffprobe"))
	ffmpegScript := "#!/bin/sh\nprintf 'ffmpeg version 7.1.1\\n'"
	ffprobeScript := "#!/bin/sh\nprintf 'ffprobe version 7.1.1\\n'"

	if err := os.WriteFile(ffmpegPath, []byte(ffmpegScript), 0o755); err != nil {
		t.Fatalf("write ffmpeg stub failed: %v", err)
	}

	result, err := service.SetToolPath(ctx, dto.SetExternalToolPathRequest{
		Name:     string(externaltools.ToolFFmpeg),
		ExecPath: ffmpegPath,
	})
	if err != nil {
		t.Fatalf("set tool path failed: %v", err)
	}
	if result.Status != string(externaltools.StatusInvalid) {
		t.Fatalf("expected invalid status without ffprobe, got %s", result.Status)
	}

	if err := os.WriteFile(ffprobePath, []byte(ffprobeScript), 0o755); err != nil {
		t.Fatalf("write ffprobe stub failed: %v", err)
	}

	result, err = service.SetToolPath(ctx, dto.SetExternalToolPathRequest{
		Name:     string(externaltools.ToolFFmpeg),
		ExecPath: ffmpegPath,
	})
	if err != nil {
		t.Fatalf("set tool path failed: %v", err)
	}
	if result.Status != string(externaltools.StatusInstalled) {
		t.Fatalf("expected installed status with ffprobe present, got %s", result.Status)
	}
	if strings.TrimSpace(result.Version) != "7.1.1" {
		t.Fatalf("unexpected ffmpeg version: %s", result.Version)
	}
}

func TestBuildWindowsFFmpegDownloadURL(t *testing.T) {
	t.Parallel()

	url, err := buildWindowsFFmpegDownloadURL("v7.1.1-3")
	if err != nil {
		t.Fatalf("build download url failed: %v", err)
	}
	archSuffix := "win64"
	if runtime.GOARCH == "arm64" {
		archSuffix = "winarm64"
	}
	expected := "https://github.com/jellyfin/jellyfin-ffmpeg/releases/download/v7.1.1-3/jellyfin-ffmpeg_7.1.1-3_portable_" + archSuffix + "-clang-gpl.zip"
	if url != expected {
		t.Fatalf("unexpected url: %s", url)
	}
}

func TestManagedToolVersionFromPathUsesManagedVersionDirectory(t *testing.T) {
	t.Parallel()

	path := filepath.Join("/Users/test/Library/Application Support/dreamcreator/external-tools", "ffmpeg", "7.1.3-5", "bin", executableNameForBinary("ffmpeg"))
	version := managedToolVersionFromPath(externaltools.ToolFFmpeg, path)
	if version != "7.1.3-5" {
		t.Fatalf("unexpected managed version: %s", version)
	}
}

func TestToExternalToolDTOUsesManagedVersionFromPath(t *testing.T) {
	t.Parallel()

	now := time.Now()
	tool, err := externaltools.NewExternalTool(externaltools.ExternalToolParams{
		Name:      string(externaltools.ToolFFmpeg),
		ExecPath:  filepath.Join("/tmp/dreamcreator/external-tools", "ffmpeg", "7.1.3-5", "bin", executableNameForBinary("ffmpeg")),
		Version:   "7.1.3-jellyfin",
		Status:    string(externaltools.StatusInstalled),
		UpdatedAt: &now,
	})
	if err != nil {
		t.Fatalf("new external tool failed: %v", err)
	}

	dto := toExternalToolDTO(tool)
	if dto.Version != "7.1.3-5" {
		t.Fatalf("unexpected dto version: %s", dto.Version)
	}
}

func TestToExternalToolDTOIncludesSourceMetadata(t *testing.T) {
	t.Parallel()

	now := time.Now()
	item, err := externaltools.NewExternalTool(externaltools.ExternalToolParams{
		Name:      string(externaltools.ToolClawHub),
		ExecPath:  "/tmp/clawhub",
		Version:   "1.2.3",
		Status:    string(externaltools.StatusInstalled),
		UpdatedAt: &now,
	})
	if err != nil {
		t.Fatalf("new external tool failed: %v", err)
	}
	result := toExternalToolDTO(item)
	if result.Kind != string(externaltools.KindBin) {
		t.Fatalf("expected bin kind, got %q", result.Kind)
	}
	if result.SourceKind != sourceKindNPMRegistry {
		t.Fatalf("expected npm registry source kind, got %q", result.SourceKind)
	}
	if result.SourceRef != "clawhub" {
		t.Fatalf("expected clawhub source ref, got %q", result.SourceRef)
	}
	if result.Manager != toolManagerBun {
		t.Fatalf("expected bun manager, got %q", result.Manager)
	}
}
