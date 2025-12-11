package dependencies

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"dreamcreator/backend/consts"
	"dreamcreator/backend/embedded"
	"dreamcreator/backend/pkg/events"
	"dreamcreator/backend/pkg/logger"
	"dreamcreator/backend/pkg/proxy"
	"dreamcreator/backend/storage"
	"dreamcreator/backend/types"
	"dreamcreator/backend/utils"

	"go.uber.org/zap"
)

// manager 依赖管理器实现
type manager struct {
	providers    map[types.DependencyType]DependencyProvider
	memoryCache  map[types.DependencyType]*types.DependencyInfo
	downloader   Downloader
	validator    Validator
	mu           sync.RWMutex
	proxyManager proxy.ProxyManager
	boltStorage  *storage.BoltStorage
	pushEvent    PushEvent
}

// NewManager 创建新的依赖管理器
func NewManager(eventBus events.EventBus, proxyManager proxy.ProxyManager, boltStorage *storage.BoltStorage) Manager {
	// 默认30分钟
	downloader := NewHTTPDownloader(30*time.Minute, proxyManager)
	// push event
	pushEvent := NewPushEvent(eventBus)

	return &manager{
		providers:    make(map[types.DependencyType]DependencyProvider),
		memoryCache:  make(map[types.DependencyType]*types.DependencyInfo),
		downloader:   downloader,
		validator:    NewValidator(pushEvent),
		proxyManager: proxyManager,
		boltStorage:  boltStorage,
		pushEvent:    pushEvent,
	}
}

// InitializeDefaultDependencies 初始化默认依赖信息
func (m *manager) InitializeDefaultDependencies() error {
	// 没有持久化存储时跳过初始化（开发/测试环境允许不初始化）
	if m.boltStorage == nil {
		logger.Warn("Skip initializing dependencies: bolt storage is nil")
		return nil
	}
	// 分别检查每个依赖是否存在，如果不存在则初始化

	// 检查 yt-dlp 依赖
	ytdlpInfo, err := m.boltStorage.GetDependency(types.DependencyYTDLP)
	if err != nil || ytdlpInfo == nil || m.validator.ValidateFile(ytdlpInfo.ExecPath) != nil {
		err = m.initializeFromEmbedded(types.DependencyYTDLP)
		if err != nil {
			logger.Error("Failed to initialize yt-dlp dependency", zap.Error(err))
			return err
		}
	}

	// 检查 ffmpeg 依赖
	ffmpegInfo, err := m.boltStorage.GetDependency(types.DependencyFFmpeg)
	if err != nil || ffmpegInfo == nil || m.validator.ValidateFile(ffmpegInfo.ExecPath) != nil {
		err = m.initializeFromEmbedded(types.DependencyFFmpeg)
		if err != nil {
			logger.Error("Failed to initialize ffmpeg dependency", zap.Error(err))
			return err
		}
	}

	// 检查 Deno 依赖
	denoInfo, err := m.boltStorage.GetDependency(types.DependencyDeno)
	if err != nil || denoInfo == nil || m.validator.ValidateFile(denoInfo.ExecPath) != nil {
		err = m.initializeFromEmbedded(types.DependencyDeno)
		if err != nil {
			logger.Error("Failed to initialize deno dependency", zap.Error(err))
			return err
		}
	}

	return nil
}

// Register 注册依赖提供者
func (m *manager) Register(provider DependencyProvider) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.providers[provider.GetType()] = provider
	logger.Info("Dependency provider registered", zap.String("type", string(provider.GetType())))
}

// GetProvider 获取依赖提供者
func (m *manager) GetProvider(depType types.DependencyType) DependencyProvider {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.providers[depType]
}

// GetDownloader 获取下载器
func (m *manager) GetDownloader() Downloader {
	return m.downloader
}

func (m *manager) GetHTTPClient() *http.Client {
	return m.proxyManager.GetHTTPClient()
}

// CleanUnused 清理所有未被当前依赖信息使用的旧版本缓存，仅保留当前版本相关目录
func (m *manager) CleanUnused(ctx context.Context) (*types.DependencyCleanResult, error) {
	if m.boltStorage == nil {
		return nil, errors.New("bolt storage not initialized")
	}

	// 从存储中获取当前依赖信息，确定每种依赖当前在用的路径
	stored, err := m.boltStorage.ListAllDependencies()
	if err != nil {
		return nil, fmt.Errorf("failed to list dependencies for cleaning: %w", err)
	}

	currentPaths := make(map[types.DependencyType]string)
	for _, dep := range stored {
		if dep == nil || dep.Path == "" {
			continue
		}
		// 归一化路径，后续用前缀判断保护当前版本目录及其父目录
		currentPaths[dep.Type] = filepath.Clean(dep.Path)
	}

	root := persistentDepsRoot()
	result := &types.DependencyCleanResult{}
	statsByType := make(map[types.DependencyType]*types.DependencyCleanStats)

	// 目前支持的依赖类型列表
	depTypes := []types.DependencyType{
		types.DependencyYTDLP,
		types.DependencyFFmpeg,
		types.DependencyDeno,
	}

	for _, depType := range depTypes {
		depRoot := filepath.Join(root, string(depType))

		info, err := os.Stat(depRoot)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			logger.Warn("CleanUnused: failed to stat dependency root",
				zap.String("type", string(depType)),
				zap.String("root", depRoot),
				zap.Error(err))
			continue
		}
		if !info.IsDir() {
			continue
		}

		entries, err := os.ReadDir(depRoot)
		if err != nil {
			logger.Warn("CleanUnused: failed to read dependency root",
				zap.String("type", string(depType)),
				zap.String("root", depRoot),
				zap.Error(err))
			continue
		}

		currentPath, hasCurrent := currentPaths[depType]
		if hasCurrent {
			currentPath = filepath.Clean(currentPath)
		}

		for _, entry := range entries {
			entryPath := filepath.Join(depRoot, entry.Name())

			// 保留当前版本所在目录及其父目录
			if hasCurrent && currentPath != "" {
				if entryPath == currentPath || strings.HasPrefix(currentPath, entryPath+string(os.PathSeparator)) {
					continue
				}
			}

			// 计算要删除路径的大小
			size, sizeErr := calcPathSize(entryPath)
			if sizeErr != nil {
				logger.Warn("CleanUnused: failed to calculate path size",
					zap.String("type", string(depType)),
					zap.String("path", entryPath),
					zap.Error(sizeErr))
				// 即使计算失败，也尝试删除，避免长时间残留垃圾
			}

			if err := os.RemoveAll(entryPath); err != nil {
				logger.Warn("CleanUnused: failed to remove path",
					zap.String("type", string(depType)),
					zap.String("path", entryPath),
					zap.Error(err))
				continue
			}

			if sizeErr == nil {
				result.TotalFreedBytes += size
				stats, ok := statsByType[depType]
				if !ok {
					stats = &types.DependencyCleanStats{
						Type:         depType,
						RemovedPaths: make([]string, 0),
					}
					statsByType[depType] = stats
				}
				stats.FreedBytes += size
				stats.RemovedPaths = append(stats.RemovedPaths, entryPath)
			}
		}
	}

	for _, st := range statsByType {
		result.Stats = append(result.Stats, *st)
	}

	return result, nil
}

// Get 获取依赖信息
func (m *manager) Get(ctx context.Context, depType types.DependencyType) (*types.DependencyInfo, error) {
	var lastErr error

	// 优先检查内存缓存（用于无持久化场景）
	m.mu.RLock()
	if cached := m.memoryCache[depType]; cached != nil {
		m.mu.RUnlock()
		return cached, nil
	}
	m.mu.RUnlock()

	// 优先尝试从 bbolt 存储读取
	if m.boltStorage != nil {
		if stored, err := m.boltStorage.GetDependency(depType); err == nil {
			// 校验已缓存依赖是否仍然有效
			if validateErr := m.validator.ValidateFile(stored.ExecPath); validateErr == nil {
				return stored, nil
			} else {
				logger.Warn("Dependency cached entry invalid, fallback to provider",
					zap.String("type", string(depType)),
					zap.Error(validateErr))
				lastErr = validateErr
			}
		} else {
			logger.Warn("Failed to get dependency from storage",
				zap.String("type", string(depType)),
				zap.Error(err))
			lastErr = err
		}
	} else {
		lastErr = errors.New("bolt storage is nil")
	}

	provider := m.GetProvider(depType)
	if provider == nil {
		if lastErr != nil {
			return nil, lastErr
		}
		return nil, fmt.Errorf("dependency provider not found: %s", depType)
	}

	type checker interface {
		Check(ctx context.Context, manager Manager) (*types.DependencyInfo, error)
	}

	var info *types.DependencyInfo
	if c, ok := provider.(checker); ok {
		var err error
		info, err = c.Check(ctx, m)
		if err != nil {
			if lastErr != nil {
				return nil, lastErr
			}
			return nil, err
		}
	} else {
		if lastErr != nil {
			return nil, lastErr
		}
		return nil, fmt.Errorf("dependency provider %s does not support status check", depType)
	}

	if m.boltStorage != nil {
		if err := m.boltStorage.SaveDependency(info); err != nil {
			logger.Warn("Failed to persist dependency info", zap.Error(err))
			m.mu.Lock()
			m.memoryCache[depType] = info
			m.mu.Unlock()
		}
	} else {
		m.mu.Lock()
		m.memoryCache[depType] = info
		m.mu.Unlock()
	}

	return info, nil
}

// 修改Install方法
func (m *manager) Install(ctx context.Context, depType types.DependencyType, config types.DownloadConfig) (*types.DependencyInfo, error) {
	m.mu.RLock()
	provider, exists := m.providers[depType]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("dependency provider not found: %s", depType)
	}

	// 发布开始安装事件：1.初始化
	if m.pushEvent != nil {
		m.pushEvent.PublishInstallEvent(string(depType), types.DependenciesPreparing, 0)
	}

	// 下载依赖
	info, err := provider.Download(ctx, m, config, func(read, total int64, percentage float64) {
		// 发布下载进度事件：2.下载中
		m.pushEvent.PublishInstallEvent(string(depType), types.DependenciesDownloading, percentage)
	})

	if err != nil {
		// 发布下载进度事件：3.下载失败（下载失败）
		logger.Warn("Manager Install Download Error",
			zap.Error(err),
			zap.Any("info", info),
			zap.Any("config", config),
		)
		m.pushEvent.PublishInstallEvent(string(depType), types.DependenciesInstallFailed, 0)
		return nil, err
	} else {
		// 保存到 bbolt 存储
		info.LastCheck = time.Now()
		if m.boltStorage != nil {
			if err := m.boltStorage.SaveDependency(info); err != nil {
				// 发布下载进度事件：3.下载失败(保存失败，仍然发布失败事件)
				m.pushEvent.PublishInstallEvent(string(depType), types.DependenciesInstallFailed, 0)
				return nil, err
			} else {
				// 发布下载进度事件：4.下载成功
				m.pushEvent.PublishInstallEvent(string(depType), types.DependenciesInstallCompleted, 100)

				// 安装成功后尝试清理旧版本缓存，仅保留当前版本
				if _, cleanErr := m.CleanUnused(ctx); cleanErr != nil {
					logger.Warn("Failed to clean unused dependencies after install",
						zap.String("type", string(depType)),
						zap.Error(cleanErr))
				}
			}
		}

	}

	return info, nil
}

// List 列出所有依赖 - 优先从 bbolt 读取
func (m *manager) List(ctx context.Context) (map[types.DependencyType]*types.DependencyInfo, error) {
	// 尝试从 bbolt 存储读取
	if m.boltStorage != nil {
		results := make(map[types.DependencyType]*types.DependencyInfo)
		if stored, err := m.boltStorage.ListAllDependencies(); err == nil {
			for _, dep := range stored {
				results[dep.Type] = dep
			}

			// 直接返回
			return results, nil
		} else {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("list: error, bolt storage is nil")
	}
}

// UpdateWithMirror 使用指定镜像更新依赖
func (m *manager) UpdateWithMirror(ctx context.Context, depType types.DependencyType, config types.DownloadConfig) (*types.DependencyInfo, error) {
	m.mu.RLock()
	provider, exists := m.providers[depType]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("dependency provider not found: %s", depType)
	}

	// 获取最新版本
	latestVersion, err := provider.GetLatestVersionWithMirror(ctx, m, config.Mirror)
	if err != nil {
		return nil, err
	}

	// 使用指定镜像下载最新版本
	updateConfig := types.DownloadConfig{
		Version:     latestVersion,
		Mirror:      config.Mirror,
		ForceUpdate: true,
		Timeout:     config.Timeout,
	}

	return m.Install(ctx, depType, updateConfig)
}

// CheckUpdates 检查所有依赖的更新
func (m *manager) CheckUpdates(ctx context.Context) (map[types.DependencyType]*types.DependencyInfo, error) {
	m.mu.RLock()
	providers := make(map[types.DependencyType]DependencyProvider)
	for k, v := range m.providers {
		providers[k] = v
	}
	m.mu.RUnlock()

	results := make(map[types.DependencyType]*types.DependencyInfo)

	for depType, provider := range providers {
		if m.boltStorage != nil {
			info, err := m.boltStorage.GetDependency(depType)
			if err != nil {
				logger.Warn("Failed to get dependency from storage",
					zap.String("type", string(depType)),
					zap.Error(err))
				// Ensure non-nil info to safely record check result
				info = &types.DependencyInfo{Type: depType}
			}

			// 检查是否有更新
			latestVersion, err := provider.GetLatestVersionWithMirror(ctx, m, "")
			if err == nil {
				info.LatestVersion = latestVersion
				info.LastCheckAttempted = true
				info.LastCheckSuccess = true
				info.LastCheckError = ""
				info.LastCheckErrorCode = ""

				// 当前版本不为空才进行更新检查
				if info.Version != "" {
					// compare with current version
					current, err := utils.ParseVersion(info.Version)
					if err == nil {
						latest, err := utils.ParseVersion(info.LatestVersion)
						if err == nil {
							if current.Compare(latest) < 0 {
								info.NeedUpdate = true
							} else {
								info.NeedUpdate = false
							}
						}
					} else {
						info.NeedUpdate = true
						logger.Warn("Failed to parse version",
							zap.String("type", string(depType)),
							zap.Error(err))
					}
				}
			} else {
				logger.Warn("Failed to get latest version",
					zap.String("type", string(depType)),
					zap.Error(err))
				// Record failed check details back to storage for frontend to display
				if info == nil {
					info = &types.DependencyInfo{Type: depType}
				}
				info.LastCheckAttempted = true
				info.LastCheckSuccess = false
				info.LastCheckError = err.Error()
				info.LastCheckErrorCode = classifyCheckError(err)
			}

			// 存储至Bbolt
			if err := m.boltStorage.SaveDependency(info); err != nil {
				logger.Warn("Failed to save dependency to storage",
					zap.String("type", string(depType)),
					zap.Error(err))
			}

			results[depType] = info
		}

	}

	return results, nil
}

// classifyCheckError 提取适合前端 i18n 展示的错误码
func classifyCheckError(err error) string {
	if err == nil {
		return ""
	}
	var ne net.Error
	if errors.As(err, &ne) {
		if ne.Timeout() {
			return "timeout"
		}
		return "network"
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "status") || strings.Contains(msg, "failed to check") {
		return "http_error"
	}
	if strings.Contains(msg, "decode") || strings.Contains(msg, "invalid character") || strings.Contains(msg, "eof") {
		return "parse_error"
	}
	if strings.Contains(msg, "network") || strings.Contains(msg, "dial tcp") || strings.Contains(msg, "connection") {
		return "network"
	}
	return "unknown"
}

func (m *manager) DependenciesReady(ctx context.Context) (bool, error) {
	if m.boltStorage == nil {
		return false, errors.New("bolt storage not initialized")
	}

	stored, err := m.boltStorage.ListAllDependencies()
	if err != nil {
		return false, fmt.Errorf("failed to list dependencies: %w", err)
	}

	for _, dep := range stored {
		if !dep.Available {
			return false, fmt.Errorf("dependency %s is not available", dep.Name)
		}

		if err := m.validator.ValidateFile(dep.ExecPath); err != nil {
			return false, fmt.Errorf("dependency %s validation failed: %w", dep.Name, err)
		}
	}

	return true, nil
}

func (m *manager) ValidateDependencies(ctx context.Context) error {
	if m.boltStorage == nil {
		return errors.New("bolt storage not initialized")
	}

	stored, err := m.boltStorage.ListAllDependencies()
	if err != nil {
		return fmt.Errorf("failed to list dependencies: %w", err)
	}

	// 使用普通的 WaitGroup，不要因为单个验证失败而取消其他验证
	var wg sync.WaitGroup
	var mu sync.Mutex
	var validationErrors []error

	for _, dep := range stored {
		// 捕获循环变量
		dependency := dep
		wg.Add(1)
		go func() {
			defer wg.Done()

			// 创建依赖的副本以避免并发修改
			updatedDep := dependency

			if err := m.validator.ValidateExecutable(ctx, dependency.ExecPath, dependency.Version, dependency.Type); err != nil {
				updatedDep.Available = false
				mu.Lock()
				validationErrors = append(validationErrors, fmt.Errorf("dependency %s validation failed: %w", dependency.Name, err))
				mu.Unlock()
			} else {
				updatedDep.Available = true
			}

			// 保存更新后的依赖状态
			if saveErr := m.boltStorage.SaveDependency(updatedDep); saveErr != nil {
				mu.Lock()
				validationErrors = append(validationErrors, fmt.Errorf("failed to save dependency %s: %w", dependency.Name, saveErr))
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	// 如果有验证错误，返回合并的错误信息
	if len(validationErrors) > 0 {
		return fmt.Errorf("validation completed with %d errors: %v", len(validationErrors), validationErrors)
	}

	return nil
}

// QuickValidate 仅快速验证：检查本地可执行文件是否存在且可执行，不做版本比对或远程请求
func (m *manager) QuickValidate(ctx context.Context) (map[types.DependencyType]*types.DependencyInfo, error) {
	if m.boltStorage == nil {
		return nil, errors.New("bolt storage not initialized")
	}

	stored, err := m.boltStorage.ListAllDependencies()
	if err != nil {
		return nil, fmt.Errorf("failed to list dependencies: %w", err)
	}

	results := make(map[types.DependencyType]*types.DependencyInfo)

	for _, dep := range stored {
		// copy
		updated := *dep
		if err := m.validator.ValidateFile(dep.ExecPath); err != nil {
			updated.Available = false
		} else {
			updated.Available = true
		}
		// persist
		if err := m.boltStorage.SaveDependency(&updated); err != nil {
			// keep going but record into results
		}
		results[updated.Type] = &updated
	}

	return results, nil
}

// RepairDependency 修复依赖
func (m *manager) RepairDependency(ctx context.Context, depType types.DependencyType) error {
	return m.initializeFromEmbedded(depType)
}

func (m *manager) initializeFromEmbedded(depType types.DependencyType) error {
	if depType == types.DependencyYTDLP {
		return m.initializeYTDLP()
	} else if depType == types.DependencyFFmpeg {
		return m.initializeFFmpeg()
	} else if depType == types.DependencyDeno {
		return m.initializeDeno()
	} else {
		return fmt.Errorf("unknown dependency type: %s", depType)
	}
}

func (m *manager) initializeYTDLP() error {
	fileByte, version, err := getEmbeddedBinary(types.DependencyYTDLP)
	if err != nil {
		return err
	}

	// exec path
	execPath, err := execPath(types.DependencyYTDLP, version)
	if err != nil {
		return err
	}

	// write
	dest, err := os.OpenFile(execPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer dest.Close()
	reader := bytes.NewReader(fileByte)
	_, err = io.Copy(dest, reader)
	if err != nil {
		return err
	}

	// return
	info := &types.DependencyInfo{
		Type:      types.DependencyYTDLP,
		Name:      "YT-DLP",
		Path:      filepath.Dir(execPath),
		ExecPath:  execPath,
		Version:   version,
		Available: true,
		LastCheck: time.Now(),
	}

	// 校验文件
	if err := m.validator.ValidateFile(execPath); err != nil {
		return err
	}

	// 保存到存储
	if err := m.boltStorage.SaveDependency(info); err != nil {
		return fmt.Errorf("failed to save default YTDLP dependency: %w", err)
	}

	return nil
}

func (m *manager) initializeFFmpeg() error {
	fileByte, version, err := getEmbeddedBinary(types.DependencyFFmpeg)
	if err != nil {
		return err
	}

	// exec path
	execPath, err := execPath(types.DependencyFFmpeg, version)
	if err != nil {
		return err
	}

	// write
	dest, err := os.OpenFile(execPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer dest.Close()
	reader := bytes.NewReader(fileByte)
	_, err = io.Copy(dest, reader)
	if err != nil {
		return err
	}

	// return
	info := &types.DependencyInfo{
		Type:      types.DependencyFFmpeg,
		Name:      "FFmpeg",
		Path:      filepath.Dir(execPath),
		ExecPath:  execPath,
		Version:   version,
		Available: true,
		LastCheck: time.Now(),
	}

	// 校验文件
	if err := m.validator.ValidateFile(execPath); err != nil {
		return err
	}

	// 保存到存储
	if err := m.boltStorage.SaveDependency(info); err != nil {
		return fmt.Errorf("failed to save default FFmpeg dependency: %w", err)
	}

	return nil
}

func (m *manager) initializeDeno() error {
	fileByte, version, err := getEmbeddedBinary(types.DependencyDeno)
	if err != nil {
		return err
	}

	// exec path
	execPath, err := execPath(types.DependencyDeno, version)
	if err != nil {
		return err
	}

	// write
	dest, err := os.OpenFile(execPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	defer dest.Close()
	reader := bytes.NewReader(fileByte)
	if _, err = io.Copy(dest, reader); err != nil {
		return err
	}

	info := &types.DependencyInfo{
		Type:      types.DependencyDeno,
		Name:      "Deno",
		Path:      filepath.Dir(execPath),
		ExecPath:  execPath,
		Version:   version,
		Available: true,
		LastCheck: time.Now(),
	}

	// 校验文件
	if err := m.validator.ValidateFile(execPath); err != nil {
		return err
	}

	// 保存到存储
	if err := m.boltStorage.SaveDependency(info); err != nil {
		return fmt.Errorf("failed to save default Deno dependency: %w", err)
	}

	return nil
}

func getEmbeddedBinary(depType types.DependencyType) (fileByte []byte, version string, err error) {
	binariesFS := embedded.GetEmbeddedBinaries() // 从embedded包获取二进制文件系统
	version, err = embedded.GetEmbeddedBinaryVersion(depType)
	if err != nil {
		return nil, "", err
	}
	fileName := fmt.Sprintf("binaries/%s_%s_%s_%s", depType, version, runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		fileName += ".exe"
	}
	// 检查文件是否存在
	if _, err := binariesFS.Open(fileName); err != nil {
		return nil, "", fmt.Errorf("embedded binary not found: %s", fileName)
	}

	fileByte, err = binariesFS.ReadFile(fileName)
	if err != nil {
		return nil, "", err
	}

	return fileByte, version, nil
}

func persistentDepsRoot() string {
	base, err := os.UserConfigDir()
	if err != nil || base == "" {
		base = os.TempDir()
	}
	root := filepath.Join(base, consts.AppDataDirName(), "deps")
	_ = os.MkdirAll(root, 0o755)
	return root
}

func execPath(depType types.DependencyType, version string) (string, error) {
	cacheDir := filepath.Join(persistentDepsRoot(), string(depType))
	execName := string(depType)
	if runtime.GOOS == "windows" {
		execName += ".exe"
	}

	execPath := filepath.Join(cacheDir, fmt.Sprintf("%s-%s", string(depType), version), execName)
	if err := os.MkdirAll(filepath.Dir(execPath), 0755); err != nil {
		return "", err
	}

	return execPath, nil
}

// calcPathSize 计算给定路径（文件或目录）的总大小
func calcPathSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}

	// 单个文件直接返回大小
	if !info.IsDir() {
		return info.Size(), nil
	}

	var total int64
	// 目录则递归计算所有文件大小
	err = filepath.Walk(path, func(_ string, fi os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if fi.IsDir() {
			return nil
		}
		total += fi.Size()
		return nil
	})
	if err != nil {
		return 0, err
	}
	return total, nil
}
