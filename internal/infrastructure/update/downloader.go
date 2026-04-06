package update

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type HTTPDownloader struct {
	client *http.Client
}

func NewHTTPDownloader(client *http.Client) *HTTPDownloader {
	return &HTTPDownloader{client: client}
}

func (downloader *HTTPDownloader) Download(ctx context.Context, url string, progress func(int)) (string, error) {
	if downloader == nil || downloader.client == nil {
		return "", fmt.Errorf("http client not configured")
	}
	if url == "" {
		return "", fmt.Errorf("download url is empty")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	resp, err := downloader.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return "", fmt.Errorf("download failed: http %d", resp.StatusCode)
	}

	tempDir, err := os.MkdirTemp("", "dreamcreator-update-*")
	if err != nil {
		return "", err
	}
	dest := filepath.Join(tempDir, filepath.Base(url))

	file, err := os.Create(dest)
	if err != nil {
		return "", err
	}
	defer file.Close()

	total := resp.ContentLength
	var written int64
	buf := make([]byte, 32*1024)
	lastReport := time.Now()

	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, err := file.Write(buf[:n]); err != nil {
				return "", err
			}
			written += int64(n)
		}
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			return "", readErr
		}
		if progress != nil && (time.Since(lastReport) > 200*time.Millisecond || written == total) {
			progress(percent(written, total))
			lastReport = time.Now()
		}
	}

	if progress != nil {
		progress(100)
	}
	return dest, nil
}

func percent(written int64, total int64) int {
	if total <= 0 {
		return 0
	}
	p := int(float64(written) / float64(total) * 100)
	if p > 100 {
		return 100
	}
	if p < 0 {
		return 0
	}
	return p
}
