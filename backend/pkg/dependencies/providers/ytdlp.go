package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"CanMe/backend/consts"
	"CanMe/backend/pkg/dependencies"
	"CanMe/backend/pkg/events"
	"CanMe/backend/types"
)

// ytdlpProvider YTDLP依赖提供者
type ytdlpProvider struct {
	*BaseProvider
}

// NewYTDLPProvider 创建YTDLP提供者
func NewYTDLPProvider(eventBus events.EventBus) dependencies.DependencyProvider {
	cacheDir := filepath.Join(os.TempDir(), "canme", "yt-dlp")
	return &ytdlpProvider{
		BaseProvider: NewBaseProvider("YT-DLP", types.DependencyYTDLP, cacheDir, eventBus),
	}
}

func (p *ytdlpProvider) GetType() types.DependencyType {
	return types.DependencyYTDLP
}

func (p *ytdlpProvider) Download(ctx context.Context, manager dependencies.Manager, config types.DownloadConfig, progress dependencies.ProgressCallback) (*types.DependencyInfo, error) {
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

	// 获取目标路径
	execPath := p.GetExecutablePath(version)

	if err := os.MkdirAll(filepath.Dir(execPath), 0755); err != nil {
		return nil, err
	}

	// 下载文件
	if err := manager.GetDownloader().Download(ctx, downloadURL, execPath, progress); err != nil {
		return nil, err
	}

	// 设置执行权限（Unix系统）
	if runtime.GOOS != "windows" {
		if err := os.Chmod(execPath, 0755); err != nil {
			return nil, err
		}
	}

	// 验证下载的文件
	if err := p.ValidateVersion(ctx, execPath, version); err != nil {
		return nil, err
	}

	// 返回依赖信息
	info := &types.DependencyInfo{
		Type:      types.DependencyYTDLP,
		Name:      "YT-DLP",
		Path:      filepath.Dir(execPath),
		ExecPath:  execPath,
		Version:   version,
		Available: true,
		LastCheck: time.Now(),
	}

	return info, nil
}

// Validate 验证YTDLP是否有效
func (p *ytdlpProvider) Validate(ctx context.Context, execPath string) error {
	return p.ValidateExecutable(ctx, execPath, "", types.DependencyYTDLP)
}

func (p *ytdlpProvider) ValidateVersion(ctx context.Context, execPath, expectedVersion string) error {
	return p.ValidateExecutable(ctx, execPath, expectedVersion, types.DependencyYTDLP)
}

// GetDownloadURLWithMirror 使用指定镜像获取下载URL
func (p *ytdlpProvider) GetDownloadURLWithMirror(version string, mirror string) (string, error) {
	if version == "" || version == "latest" {
		version = "latest"
	}

	osType := runtime.GOOS

	// 使用配置中的URL模板构建下载链接
	downloadURL, err := consts.BuildYTDLPDownloadURL(mirror, version, osType)
	if err != nil {
		// 如果指定镜像不存在，回退到默认镜像
		defaultMirror := consts.GetRecommendedMirror("yt-dlp", osType)
		if mirror != defaultMirror {
			return consts.BuildYTDLPDownloadURL(defaultMirror, version, osType)
		}
		return "", err
	}

	return downloadURL, nil
}

func (p *ytdlpProvider) GetAPIURLWithMirror() (string, error) {
	return consts.GetYTDLPAPIURL()
}

func (p *ytdlpProvider) GetLatestVersionWithMirror(ctx context.Context, manager dependencies.Manager, mirror string) (string, error) {
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
		return "", fmt.Errorf("failed to check ytdlp version: %s", resp.Status)
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
