package dependencies

import (
	"CanMe/backend/pkg/proxy"
	"CanMe/backend/types"
	"context"
	"net/http"
)

// ProgressCallback 下载进度回调
type ProgressCallback func(read, total int64, percentage float64)

// DependencyProvider 依赖提供者接口
type DependencyProvider interface {
	// GetType 获取依赖类型
	GetType() types.DependencyType
	// Download 下载依赖
	Download(ctx context.Context, manager Manager, config types.DownloadConfig, progress ProgressCallback) (*types.DependencyInfo, error)
	// Validate 验证依赖是否有效
	Validate(ctx context.Context, execPath string) error
	// ValidateVersion 验证依赖版本是否有效
	ValidateVersion(ctx context.Context, execPath, expectedVersion string) error
	// GetDownloadURLWithMirror 获取下载URL
	GetDownloadURLWithMirror(version string, mirror string) (string, error)
	// GetLatestVersionWithMirror 获取最新版本
	GetLatestVersionWithMirror(ctx context.Context, manager Manager, mirror string) (string, error)
}

// Manager 依赖管理器接口
type Manager interface {
	// InitializeDefaultDependencies 初始化默认依赖信息
	InitializeDefaultDependencies() error
	// Register 注册依赖提供者
	Register(provider DependencyProvider)
	// GetProvider 获取依赖提供者
	GetProvider(depType types.DependencyType) DependencyProvider
	// GetDownloader 获取下载器
	GetDownloader() Downloader
	// GetHTTPClient 获取代理http客户端
	GetHTTPClient() *http.Client
	// Get 获取依赖信息
	Get(ctx context.Context, depType types.DependencyType) (*types.DependencyInfo, error)
	// Install 安装依赖
	Install(ctx context.Context, depType types.DependencyType, config types.DownloadConfig) (*types.DependencyInfo, error)
	// UpdateWithMirror 使用指定镜像更新依赖
	UpdateWithMirror(ctx context.Context, depType types.DependencyType, config types.DownloadConfig) (*types.DependencyInfo, error)
	// CheckUpdates 检查所有依赖的更新
	CheckUpdates(ctx context.Context) (map[types.DependencyType]*types.DependencyInfo, error)
	// List 列出所有依赖
	List(ctx context.Context) (map[types.DependencyType]*types.DependencyInfo, error)
	// DependenciesReady 检查所有依赖是否已准备好
	DependenciesReady(ctx context.Context) (bool, error)
	// ValidateDependencies 验证所有依赖可用性
	ValidateDependencies(ctx context.Context) error
}

// Downloader 下载器接口
type Downloader interface {
	Download(ctx context.Context, url, destPath string, progress ProgressCallback) error
	SetProxyManager(manager proxy.ProxyManager)
}

// Validator 验证器接口
type Validator interface {
	ValidateFile(filePath string) error
	ValidateExecutable(ctx context.Context, execPath string, expectedVersion string, depType types.DependencyType) error
	ValidateChecksum(filePath, expectedChecksum string) error
}

// PushEvent WS接口
type PushEvent interface {
	PublishInstallEvent(depType string, stage types.DtTaskStage, percentage float64)
}
