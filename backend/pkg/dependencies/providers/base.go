package providers

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"dreamcreator/backend/pkg/dependencies"
	"dreamcreator/backend/pkg/events"
	"dreamcreator/backend/pkg/logger"
	"dreamcreator/backend/types"

	"go.uber.org/zap"
)

// BaseProvider 基础依赖提供者
type BaseProvider struct {
	name      string
	depType   types.DependencyType
	validator dependencies.Validator
	cacheDir  string
}

// NewBaseProvider 创建基础提供者
func NewBaseProvider(name string, depType types.DependencyType, cacheDir string, eventBus events.EventBus) *BaseProvider {
	return &BaseProvider{
		name:      name,
		depType:   depType,
		validator: dependencies.NewValidator(dependencies.NewPushEvent(eventBus)),
		cacheDir:  cacheDir,
	}
}

// GetType 获取依赖类型
func (bp *BaseProvider) GetType() types.DependencyType {
	return bp.depType
}

// GetCacheDir 获取缓存目录
func (bp *BaseProvider) GetCacheDir() string {
	return bp.cacheDir
}

// GetExecutablePath 获取可执行文件路径
func (bp *BaseProvider) GetExecutablePath(version string) string {
	execName := string(bp.depType)
	if runtime.GOOS == "windows" {
		execName += ".exe"
	}

	if version != "" {
		return filepath.Join(bp.cacheDir, fmt.Sprintf("%s-%s", string(bp.depType), version), execName)
	}
	return filepath.Join(bp.cacheDir, string(bp.depType), execName)
}

func (bp *BaseProvider) GetArchivePath(url, version string) string {
	archiveName := url[strings.LastIndex(url, "/")+1:]

	if version != "" {
		return filepath.Join(bp.cacheDir, fmt.Sprintf("%s-%s", string(bp.depType), version), archiveName)
	}
	return filepath.Join(bp.cacheDir, string(bp.depType), archiveName)
}

// DownloadAndExtract 下载并解压文件
func (bp *BaseProvider) DownloadAndExtract(ctx context.Context, manager dependencies.Manager, url, version string, progress dependencies.ProgressCallback) (string, error) {
	// 获取目标路径
	archivePath := bp.GetArchivePath(url, version)

	if err := os.MkdirAll(filepath.Dir(archivePath), 0755); err != nil {
		return "", err
	}

	// 检查文件是否已存在且有效
	if bp.isArchiveValid(archivePath) {
		logger.Info("Archive already exists and is valid, skipping download",
			zap.String("path", archivePath))
	} else {
		// 下载文件
		if err := manager.GetDownloader().Download(ctx, url, archivePath, progress); err != nil {
			logger.Warn("Failed to download archive",
				zap.String("url", url),
				zap.String("path", archivePath),
				zap.Error(err))
			return "", fmt.Errorf("failed to download: %w", err)
		}
	}

	// 如果是直接的可执行文件（如ytdlp），直接返回
	if strings.HasSuffix(archivePath, ".exe") || !strings.Contains(archivePath, ".") {
		return archivePath, nil
	}

	// 解压压缩文件
	execPath, err := bp.extractArchive(archivePath, version)
	if err != nil {
		return "", fmt.Errorf("failed to extract: %w", err)
	}

	// 清理压缩文件
	os.Remove(archivePath)

	return execPath, nil
}

// ValidateExecutable 验证可执行文件
func (bp *BaseProvider) ValidateExecutable(ctx context.Context, execPath, expectedVersion string, depType types.DependencyType) error {
	return bp.validator.ValidateExecutable(ctx, execPath, expectedVersion, depType)
}

// extractArchive 解压压缩文件
func (bp *BaseProvider) extractArchive(archivePath, version string) (string, error) {
	// 根据文件扩展名选择解压方法
	switch {
	case strings.HasSuffix(archivePath, ".zip"):
		return bp.extractZip(archivePath, version)
	default:
		return "", fmt.Errorf("unsupported archive format: %s", archivePath)
	}
}

// extractZip 解压ZIP文件
func (bp *BaseProvider) extractZip(archivePath, version string) (string, error) {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return "", err
	}
	defer r.Close()

	// 修改：使用不同的目录名避免冲突
	extractDir := filepath.Join(bp.cacheDir, fmt.Sprintf("%s-%s-extracted", bp.name, version))
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		return "", err
	}

	var execPath string
	var execPaths []string // 收集所有可执行文件

	for _, f := range r.File {
		path := filepath.Join(extractDir, f.Name)

		if f.FileInfo().IsDir() {
			os.MkdirAll(path, f.FileInfo().Mode())
			continue
		}

		// 创建文件
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return "", err
		}

		rc, err := f.Open()
		if err != nil {
			return "", err
		}

		outFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.FileInfo().Mode())
		if err != nil {
			rc.Close()
			return "", err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()
		if err != nil {
			return "", err
		}

		// 检查是否是可执行文件
		if bp.isExecutableFile(f.Name) {
			execPaths = append(execPaths, path)
		}
	}

	if len(execPaths) == 0 {
		return "", fmt.Errorf("executable not found in archive")
	}

	// 选择最合适的可执行文件
	execPath = bp.selectBestExecutable(execPaths)

	return execPath, nil
}

// selectBestExecutable 选择最合适的可执行文件
func (bp *BaseProvider) selectBestExecutable(execPaths []string) string {
	// 优先级规则：选择与provider名称最匹配的文件
	for _, path := range execPaths {
		fileName := filepath.Base(path)
		// 去除扩展名进行比较
		baseName := strings.TrimSuffix(fileName, filepath.Ext(fileName))
		if strings.EqualFold(baseName, bp.name) {
			return path
		}
	}

	// 如果没有完全匹配，返回第一个
	return execPaths[0]
}

// isExecutableFile 检查是否是可执行文件
func (bp *BaseProvider) isExecutableFile(filename string) bool {
	baseName := filepath.Base(filename)
	switch bp.depType {
	case types.DependencyFFmpeg:
		return baseName == "ffmpeg" || baseName == "ffmpeg.exe"
	case types.DependencyYTDLP:
		return baseName == "yt-dlp" || baseName == "yt-dlp.exe"
	case types.DependencyDeno:
		return baseName == "deno" || baseName == "deno.exe"
	default:
		return false
	}
}

// isArchiveValid 检查压缩文件是否存在且有效
func (bp *BaseProvider) isArchiveValid(archivePath string) bool {
	// 检查文件是否存在
	fileInfo, err := os.Stat(archivePath)
	if err != nil {
		return false
	}

	// 检查文件大小（避免空文件或损坏文件）
	if fileInfo.Size() == 0 {
		logger.Warn("Archive file is empty", zap.String("path", archivePath))
		return false
	}

	// 检查文件权限（确保可读）
	if fileInfo.Mode().Perm()&0400 == 0 {
		logger.Warn("Archive file is not readable",
			zap.String("path", archivePath),
			zap.String("mode", fileInfo.Mode().String()))
		return false
	}

	// 对于压缩文件，进行基本的格式验证
	if strings.HasSuffix(archivePath, ".zip") {
		return bp.isValidZipFile(archivePath)
	}

	// 对于其他格式，暂时只检查文件存在性和大小
	return true
}

// isValidZipFile 检查ZIP文件是否有效
func (bp *BaseProvider) isValidZipFile(zipPath string) bool {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		logger.Warn("Invalid ZIP file",
			zap.String("path", zipPath),
			zap.Error(err))
		return false
	}
	defer r.Close()

	// 检查ZIP文件是否包含文件
	if len(r.File) == 0 {
		logger.Warn("ZIP file is empty", zap.String("path", zipPath))
		return false
	}

	return true
}
