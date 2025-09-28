package dependencies

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"dreamcreator/backend/pkg/logger"
	"dreamcreator/backend/pkg/proxy"

	"go.uber.org/zap"
)

// httpDownloader HTTP下载器实现
type httpDownloader struct {
	client       *http.Client
	proxyManager proxy.ProxyManager
}

// NewHTTPDownloader 创建HTTP下载器
func NewHTTPDownloader(timeout time.Duration, proxyManager proxy.ProxyManager) Downloader {
	return &httpDownloader{
		client: &http.Client{
			Timeout: timeout,
		},
		proxyManager: proxyManager,
	}
}

// Download 下载文件
func (d *httpDownloader) Download(ctx context.Context, url, destPath string, progress ProgressCallback) error {
	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// 每次下载时直接获取最新的HTTP客户端
	client := d.client
	if d.proxyManager != nil {
		client = d.proxyManager.GetHTTPClient()
	}

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	// 确保目标目录存在
	targetDir := filepath.Dir(destPath)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// 创建临时文件
	tempPath := destPath + ".tmp"
	out, err := os.Create(tempPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer func() {
		out.Close()
		// 只在下载失败时清理临时文件
		if _, err := os.Stat(tempPath); err == nil {
			if removeErr := os.Remove(tempPath); removeErr != nil {
				logger.Warn("Failed to remove temporary file",
					zap.Error(removeErr),
					zap.String("temp_path", tempPath))
			}
		}
	}()

	// 获取文件大小
	totalSize := resp.ContentLength

	// 创建进度读取器
	var reader io.Reader = resp.Body
	if progress != nil && totalSize > 0 {
		reader = &progressReader{
			reader:   resp.Body,
			total:    totalSize,
			callback: progress,
		}
	}

	// 复制数据
	_, err = io.Copy(out, reader)
	if err != nil {
		return fmt.Errorf("failed to copy data: %w", err)
	}

	// 关闭文件
	out.Close()

	// 移动到最终位置
	if err := os.Rename(tempPath, destPath); err != nil {
		return fmt.Errorf("failed to move file: %w", err)
	}

	return nil
}

// SetProxyClient 设置代理客户端
func (d *httpDownloader) SetProxyManager(manager proxy.ProxyManager) {
	if manager != nil {
		d.proxyManager = manager
	}
}

// progressReader 带进度的读取器
type progressReader struct {
	reader   io.Reader
	total    int64
	read     int64
	callback ProgressCallback
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	pr.read += int64(n)

	if pr.callback != nil {
		percentage := float64(pr.read) / float64(pr.total) * 100
		pr.callback(pr.read, pr.total, percentage)
	}

	return n, err
}
