package dependencies

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHTTPDownloader(t *testing.T) {
	timeout := 30 * time.Second
	// Fix: Pass nil proxyManager for testing
	downloader := NewHTTPDownloader(timeout, nil)
	assert.NotNil(t, downloader)
	assert.IsType(t, &httpDownloader{}, downloader)

	hd := downloader.(*httpDownloader)
	assert.Equal(t, timeout, hd.client.Timeout)
}

func TestHTTPDownloaderDownload(t *testing.T) {
	// 创建测试服务器
	testContent := "test file content"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(testContent)))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(testContent))
	}))
	defer server.Close()

	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "downloader_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	destPath := filepath.Join(tempDir, "test_file.txt")
	// Fix: Pass nil proxyManager for testing
	downloader := NewHTTPDownloader(30*time.Second, nil)

	// 测试下载
	ctx := context.Background()
	var progressCalled bool
	progressCallback := func(read, total int64, percentage float64) {
		progressCalled = true
		assert.True(t, read <= total)
		assert.True(t, percentage >= 0 && percentage <= 100)
	}

	err = downloader.Download(ctx, server.URL, destPath, progressCallback)
	assert.NoError(t, err)
	assert.True(t, progressCalled)

	// 验证文件内容
	content, err := os.ReadFile(destPath)
	require.NoError(t, err)
	assert.Equal(t, testContent, string(content))
}

func TestHTTPDownloaderDownloadError(t *testing.T) {
	// 测试404错误
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	tempDir, err := os.MkdirTemp("", "downloader_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	destPath := filepath.Join(tempDir, "test_file.txt")
	// Fix: Pass nil proxyManager for testing
	downloader := NewHTTPDownloader(30*time.Second, nil)

	ctx := context.Background()
	err = downloader.Download(ctx, server.URL, destPath, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "download failed with status: 404")
}

func TestHTTPDownloaderDownloadTimeout(t *testing.T) {
	// 创建慢响应服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.Write([]byte("slow response"))
	}))
	defer server.Close()

	tempDir, err := os.MkdirTemp("", "downloader_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	destPath := filepath.Join(tempDir, "test_file.txt")
	// 设置很短的超时时间
	// Fix: Pass nil proxyManager for testing
	downloader := NewHTTPDownloader(100*time.Millisecond, nil)

	ctx := context.Background()
	err = downloader.Download(ctx, server.URL, destPath, nil)
	assert.Error(t, err)
}

func TestHTTPDownloaderDownloadContextCancel(t *testing.T) {
	// 创建慢响应服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.Write([]byte("slow response"))
	}))
	defer server.Close()

	tempDir, err := os.MkdirTemp("", "downloader_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	destPath := filepath.Join(tempDir, "test_file.txt")
	// Fix: Pass nil proxyManager for testing
	downloader := NewHTTPDownloader(30*time.Second, nil)

	// 创建可取消的上下文
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	err = downloader.Download(ctx, server.URL, destPath, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")
}

func TestProgressReader(t *testing.T) {
	content := "test content for progress reader"
	reader := strings.NewReader(content)
	totalSize := int64(len(content))

	var progressUpdates []float64
	progressCallback := func(read, total int64, percentage float64) {
		progressUpdates = append(progressUpdates, percentage)
		assert.Equal(t, totalSize, total)
		assert.True(t, read <= total)
	}

	pr := &progressReader{
		reader:   reader,
		total:    totalSize,
		callback: progressCallback,
	}

	// 读取所有内容
	result, err := io.ReadAll(pr)
	assert.NoError(t, err)
	assert.Equal(t, content, string(result))
	assert.NotEmpty(t, progressUpdates)
	assert.Equal(t, 100.0, progressUpdates[len(progressUpdates)-1])
}
