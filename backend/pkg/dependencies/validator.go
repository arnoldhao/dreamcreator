package dependencies

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"time"

	"CanMe/backend/pkg/logger"
	"CanMe/backend/types"

	"go.uber.org/zap"
)

// validator 验证器实现
type validator struct {
	pushEvent PushEvent
}

// NewValidator 创建验证器
func NewValidator(pushEvent PushEvent) Validator {
	return &validator{
		pushEvent: pushEvent,
	}
}

// ValidateFile 验证文件是否存在且可读
func (v *validator) ValidateFile(filePath string) error {
	if filePath == "" {
		return fmt.Errorf("file path is empty")
	}

	// 检查文件是否存在
	info, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file does not exist: %s", filePath)
		}
		return fmt.Errorf("failed to stat file: %w", err)
	}

	// 检查是否为文件
	if info.IsDir() {
		return fmt.Errorf("path is a directory, not a file: %s", filePath)
	}

	// 检查文件大小
	if info.Size() == 0 {
		return fmt.Errorf("file is empty: %s", filePath)
	}

	// 检查可执行权限（Unix系统）
	if runtime.GOOS != "windows" {
		if info.Mode()&0111 == 0 {
			return fmt.Errorf("file is not executable: %s", filePath)
		}
	}

	return nil
}

// ValidateExecutable 验证可执行文件并检查版本
func (v *validator) ValidateExecutable(ctx context.Context, execPath string, expectedVersion string, depType types.DependencyType) error {
	if v.pushEvent != nil {
		v.pushEvent.PublishInstallEvent(string(depType), types.DependenciesValidating, 0)
	}

	defer func() {
		if v.pushEvent != nil {
			v.pushEvent.PublishInstallEvent(string(depType), types.DependenciesValidating, 100)
		}
	}()

	// 首先验证文件
	if err := v.ValidateFile(execPath); err != nil {
		return err
	}

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	// 尝试执行版本命令
	cmd := createHiddenCommand(ctx, execPath, "-version")
	output, err := cmd.Output()
	if err != nil {
		// 尝试其他版本命令格式
		cmd = createHiddenCommand(ctx, execPath, "--version")
		output, err = cmd.Output()
		if err != nil {
			return fmt.Errorf("failed to get version: %w", err)
		}
	}

	versionStr := string(output)
	logger.Info("Version output", zap.String("exec", execPath), zap.String("output", versionStr))

	// 如果指定了期望版本，进行验证
	if expectedVersion != "" {
		// 去掉v前缀
		if strings.HasPrefix(expectedVersion, "v") {
			expectedVersion = expectedVersion[1:]
		}

		// 处理可能包含后缀的版本（如 "1-6"）
		if strings.Contains(expectedVersion, "-") {
			expectedVersion = strings.Split(expectedVersion, "-")[0]
		}

		// 去掉versionStr后缀换行符
		if strings.HasSuffix(versionStr, "\n") {
			versionStr = strings.TrimSuffix(versionStr, "\n")
		}

		if !strings.Contains(strings.ToLower(versionStr), strings.ToLower(expectedVersion)) {
			return fmt.Errorf("version mismatch: expected %s, got %s", expectedVersion, versionStr)
		}
	}

	return nil
}

// ValidateChecksum 验证文件校验和
func (v *validator) ValidateChecksum(filePath, expectedChecksum string) error {
	if expectedChecksum == "" {
		return nil // 如果没有提供校验和，跳过验证
	}

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return fmt.Errorf("failed to calculate checksum: %w", err)
	}

	actualChecksum := fmt.Sprintf("%x", hash.Sum(nil))
	if actualChecksum != expectedChecksum {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedChecksum, actualChecksum)
	}

	return nil
}
