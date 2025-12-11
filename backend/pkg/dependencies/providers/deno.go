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

// denoProvider Deno依赖提供者
type denoProvider struct {
	*BaseProvider
}

// NewDenoProvider 创建Deno提供者
func NewDenoProvider(eventBus events.EventBus) dependencies.DependencyProvider {
	// store in persistent per-user directory to avoid system cleaning
	cacheDir := filepath.Join(persistentDepsRoot(), "deno")
	return &denoProvider{
		BaseProvider: NewBaseProvider("deno", types.DependencyDeno, cacheDir, eventBus),
	}
}

func (p *denoProvider) GetType() types.DependencyType {
	return types.DependencyDeno
}

// Download 下载并安装 Deno
func (p *denoProvider) Download(ctx context.Context, manager dependencies.Manager, config types.DownloadConfig, progress dependencies.ProgressCallback) (*types.DependencyInfo, error) {
	version := config.Version
	if version == "" || version == "latest" {
		var err error
		version, err = p.GetLatestVersionWithMirror(ctx, manager, config.Mirror)
		if err != nil {
			return nil, fmt.Errorf("failed to get latest deno version: %w", err)
		}
	}

	// 获取下载URL
	downloadURL, err := p.GetDownloadURLWithMirror(version, config.Mirror)
	if err != nil {
		return nil, err
	}

	// 下载并解压或直接写入可执行文件
	execPath, err := p.DownloadAndExtract(ctx, manager, downloadURL, version, progress)
	if err != nil {
		return nil, err
	}

	// 设置执行权限（Unix系统）
	if runtime.GOOS != "windows" {
		if err := os.Chmod(execPath, 0o755); err != nil {
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
		Type:      types.DependencyDeno,
		Name:      "Deno",
		Path:      filepath.Dir(execPath),
		ExecPath:  execPath,
		Version:   version,
		Available: true,
		LastCheck: time.Now(),
	}

	return info, nil
}

// Validate 验证Deno是否有效
func (p *denoProvider) Validate(ctx context.Context, execPath string) error {
	return p.ValidateExecutable(ctx, execPath, "", types.DependencyDeno)
}

func (p *denoProvider) ValidateVersion(ctx context.Context, execPath, expectedVersion string) error {
	return p.ValidateExecutable(ctx, execPath, expectedVersion, types.DependencyDeno)
}

// GetDownloadURLWithMirror 使用指定镜像获取下载URL
func (p *denoProvider) GetDownloadURLWithMirror(version string, mirror string) (string, error) {
	if version == "" || version == "latest" {
		version = "latest"
	}

	osType := runtime.GOOS
	arch := runtime.GOARCH

	downloadURL, err := consts.BuildDenoDownloadURL(mirror, version, osType, arch)
	if err != nil {
		// 如果指定镜像不存在，回退到默认镜像
		defaultMirror := consts.GetRecommendedMirror("deno", osType)
		if mirror != defaultMirror {
			return consts.BuildDenoDownloadURL(defaultMirror, version, osType, arch)
		}
		return "", err
	}

	return downloadURL, nil
}

func (p *denoProvider) GetAPIURLWithMirror() (string, error) {
	return consts.GetDenoAPIURL()
}

func (p *denoProvider) GetLatestVersionWithMirror(ctx context.Context, manager dependencies.Manager, mirror string) (string, error) {
	url, err := p.GetAPIURLWithMirror()
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
		return "", fmt.Errorf("failed to check deno version: %s", resp.Status)
	}

	var release struct {
		TagName string `json:"tag_name"`
		Name    string `json:"name"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}

	if release.TagName == "" {
		return "", fmt.Errorf("empty tag_name received from GitHub API for deno")
	}

	return release.TagName, nil
}
