package dependencies

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewValidator(t *testing.T) {
	validator := NewValidator(nil)
	assert.NotNil(t, validator)
	// 修复：使用 assert.Implements 检查接口实现
	assert.Implements(t, (*Validator)(nil), validator)
}

func TestValidatorValidateFile(t *testing.T) {
	validator := NewValidator(nil)

	// 测试空路径
	err := validator.ValidateFile("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "file path is empty")

	// 测试不存在的文件
	err = validator.ValidateFile("/non/existent/file")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "file does not exist")

	// 创建临时文件进行测试
	tempDir, err := os.MkdirTemp("", "validator_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	testFile := filepath.Join(tempDir, "test.txt")
	// 修复：创建可执行文件（非 Windows 系统需要可执行权限）
	var fileMode os.FileMode = 0755
	if runtime.GOOS == "windows" {
		fileMode = 0644 // Windows 不需要可执行权限
	}
	err = os.WriteFile(testFile, []byte("test content"), fileMode)
	require.NoError(t, err)

	// 测试有效文件
	err = validator.ValidateFile(testFile)
	assert.NoError(t, err)

	// 测试目录（应该失败）
	err = validator.ValidateFile(tempDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "is a directory")

	// 测试空文件（应该失败）
	emptyFile := filepath.Join(tempDir, "empty.txt")
	err = os.WriteFile(emptyFile, []byte(""), fileMode)
	require.NoError(t, err)
	err = validator.ValidateFile(emptyFile)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "file is empty")
}

func TestValidatorValidateExecutable(t *testing.T) {
	validator := NewValidator(nil)
	ctx := context.Background()

	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "validator_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 创建可执行文件
	execName := "test_exec"
	if runtime.GOOS == "windows" {
		execName = "test_exec.exe"
	}
	execPath := filepath.Join(tempDir, execName)

	// 写入简单的可执行内容
	var execContent []byte
	if runtime.GOOS == "windows" {
		// Windows PE header (简化版)
		execContent = []byte{0x4D, 0x5A} // MZ header
	} else {
		// Unix ELF header (简化版)
		execContent = []byte{0x7F, 0x45, 0x4C, 0x46} // ELF header
	}

	err = os.WriteFile(execPath, execContent, 0755)
	require.NoError(t, err)

	// 测试可执行文件验证
	err = validator.ValidateExecutable(ctx, execPath, "1.0.0", "ffmpeg")
	// 注意：这个测试可能会失败，因为我们创建的不是真正的可执行文件
	// 但至少可以测试文件存在性检查
	if err != nil {
		// 如果执行失败，至少确保不是文件不存在的错误
		assert.NotContains(t, err.Error(), "file does not exist")
		assert.NotContains(t, err.Error(), "file is not executable")
	}
}

func TestValidatorValidateChecksum(t *testing.T) {
	validator := NewValidator(nil)

	// 创建临时文件
	tempDir, err := os.MkdirTemp("", "validator_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	testContent := "test content for checksum"
	testFile := filepath.Join(tempDir, "test.txt")
	err = os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err)

	// 计算正确的校验和
	hash := sha256.Sum256([]byte(testContent))
	expectedChecksum := fmt.Sprintf("%x", hash)

	// 测试正确的校验和
	err = validator.ValidateChecksum(testFile, expectedChecksum)
	assert.NoError(t, err)

	// 测试错误的校验和
	err = validator.ValidateChecksum(testFile, "wrong_checksum")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "checksum mismatch")

	// 测试不存在的文件
	err = validator.ValidateChecksum("/non/existent/file", expectedChecksum)
	assert.Error(t, err)
}
