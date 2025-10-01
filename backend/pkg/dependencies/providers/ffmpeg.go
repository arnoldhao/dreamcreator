package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"dreamcreator/backend/consts"
	"dreamcreator/backend/pkg/dependencies"
	"dreamcreator/backend/pkg/events"
	"dreamcreator/backend/types"
)

// ffmpegProvider FFmpeg依赖提供者
type ffmpegProvider struct {
	*BaseProvider
}

// NewFFmpegProvider 创建FFmpeg提供者
func NewFFmpegProvider(eventBus events.EventBus) dependencies.DependencyProvider {
	// store in persistent per-user directory to avoid system cleaning
	cacheDir := filepath.Join(persistentDepsRoot(), "ffmpeg")
	return &ffmpegProvider{
		BaseProvider: NewBaseProvider("FFmpeg", types.DependencyFFmpeg, cacheDir, eventBus),
	}
}

// GetType 获取依赖类型
func (p *ffmpegProvider) GetType() types.DependencyType {
	return types.DependencyFFmpeg
}

// Download 下载FFmpeg
func (p *ffmpegProvider) Download(ctx context.Context, manager dependencies.Manager, config types.DownloadConfig, progress dependencies.ProgressCallback) (*types.DependencyInfo, error) {
	version := config.Version
	if version == "" || version == "latest" {
		var err error
		version, err = p.GetLatestVersionWithMirror(ctx, manager, config.Mirror)
		if err != nil {
			return nil, fmt.Errorf("failed to get latest version: %w", err)
		}
	}

	// 获取下载URL
	downloadURL, err := p.GetDownloadURLWithMirror(version, config.Mirror)
	if err != nil {
		return nil, err
	}

	// 使用 BaseProvider 的 DownloadAndExtract 方法
	execPath, err := p.DownloadAndExtract(ctx, manager, downloadURL, version, progress)
	if err != nil {
		return nil, err
	}

	// 设置执行权限（Unix系统）
	if runtime.GOOS != "windows" {
		if err := os.Chmod(execPath, 0755); err != nil {
			return nil, err
		}

		// Remove macOS quarantine to avoid Gatekeeper slow verification
		if runtime.GOOS == "darwin" {
			_ = exec.Command("xattr", "-dr", "com.apple.quarantine", execPath).Run()
			_ = exec.Command("xattr", "-dr", "com.apple.quarantine", filepath.Dir(execPath)).Run()
		}
	}

	// 验证下载的文件
	if err := p.ValidateVersion(ctx, execPath, version); err != nil {
		return nil, err
	}

	// 返回依赖信息
	info := &types.DependencyInfo{
		Type:      types.DependencyFFmpeg,
		Name:      "FFmpeg",
		Path:      filepath.Dir(execPath),
		ExecPath:  execPath,
		Version:   version,
		Available: true,
		LastCheck: time.Now(), // 添加这个字段
	}

	return info, nil
}

// Validate 验证FFmpeg是否有效
func (p *ffmpegProvider) Validate(ctx context.Context, execPath string) error {
	return p.ValidateExecutable(ctx, execPath, "", types.DependencyFFmpeg)
}

func (p *ffmpegProvider) ValidateVersion(ctx context.Context, execPath, expectedVersion string) error {
	return p.ValidateExecutable(ctx, execPath, expectedVersion, types.DependencyFFmpeg)
}

// GetDownloadURLWithMirror 使用指定镜像获取下载URL
func (p *ffmpegProvider) GetDownloadURLWithMirror(version string, mirror string) (string, error) {
	osType := runtime.GOOS
	arch := runtime.GOARCH

	return consts.BuildFFMPEGDownloadURL(mirror, version, osType, arch)
}

func (p *ffmpegProvider) GetAPIURLWithMirror(mirror string) (string, error) {
	return consts.GetFFMPEGAPIURL(mirror, runtime.GOOS, runtime.GOARCH)
}

// GetLatestVersion 获取最新版本
func (p *ffmpegProvider) GetLatestVersionWithMirror(ctx context.Context, manager dependencies.Manager, mirror string) (string, error) {
	if mirror == "" {
		osType := runtime.GOOS
		mirror = consts.DefaultMirrors["ffmpeg"][osType]
	}

	switch mirror {
	case "evermeet":
		return p.getLatestVersionEvermeet(ctx, manager)
	case "ghproxy":
		return p.getLatestVersionGHProxy(ctx, manager)
	default:
		return "latest", nil
	}
}

func (p *ffmpegProvider) getLatestVersionGHProxy(ctx context.Context, manager dependencies.Manager) (string, error) {
	url, err := p.GetAPIURLWithMirror("ghproxy")
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}

	resp, err := manager.GetHTTPClient().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to check ffmpeg version: %s", resp.Status)
	}

	var release struct {
		TagName string `json:"tag_name"`
		Name    string `json:"name"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}

	// 验证tag_name不为空
	if release.TagName == "" {
		return "", fmt.Errorf("empty tag_name received from GitHub API")
	}

	return release.TagName, nil
}

func (p *ffmpegProvider) getLatestVersionEvermeet(ctx context.Context, manager dependencies.Manager) (string, error) {
	url, err := p.GetAPIURLWithMirror("evermeet")
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "latest", err
	}

	resp, err := manager.GetHTTPClient().Do(req)
	if err != nil {
		return "latest", err
	}
	defer resp.Body.Close()

	var release struct {
		Version string `json:"version"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "latest", err
	}

	return release.Version, nil
}
