package downtasks

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"CanMe/backend/consts"
	"CanMe/backend/pkg/logger"
	"CanMe/backend/types"

	"go.uber.org/zap"
)

// YtdlpConfig yt-dlp config
type YtdlpConfig struct {
	Version string
	BaseURL string
}

// getYtdlpPath get or install yt-dlp
func (s *Service) getYtdlpPath() (string, error) {
	config := YtdlpConfig{
		Version: consts.YTDLP_VERSION,
	}

	// check ytdlp
	available, path, execPath, err := ytdlpAvailable()
	if err != nil {
		logger.Error("Failed to check ytdlp", zap.Error(err))
	}

	// log info: directory/execPath
	logger.Info("ytdlp check result",
		zap.String("directory", path),
		zap.String("execPath", execPath),
		zap.Bool("available", available))

	// if installed, return
	if available {
		return execPath, nil
	}

	// report to event bus, begin to install
	s.eventBus.Broadcast(consts.TopicDowntasksInstalling, &types.DtProgress{
		Stage:     "installing",
		StageInfo: "0%",
	})

	// makesure directory exists
	if err := os.MkdirAll(path, 0o755); err != nil {
		return "", fmt.Errorf("Create directory failed: %w", err)
	}

	// build download url
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "https://gh-proxy.com/github.com/yt-dlp/yt-dlp/releases/download/" + config.Version
	}

	// get yt-dlp repo release name
	filename, err := releaseName()
	if err != nil {
		return "", err
	}

	// build download url
	downloadURL := fmt.Sprintf("%s/%s", baseURL, filename)
	// log info: downloadURL
	logger.Info("ytdlp download starting", zap.String("url", downloadURL))

	// download
	req, err := http.NewRequestWithContext(s.ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return "", fmt.Errorf("Create download request failed: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("Download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Download failed: HTTP %d", resp.StatusCode)
	}

	// create temp file
	tmpFile, err := os.CreateTemp(path, "yt-dlp-*")
	// log info: temp file
	logger.Info("ytdlp temp file created", zap.String("path", tmpFile.Name()))

	if err != nil {
		return "", fmt.Errorf("Create temp file failed: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	// write file, show download progress
	totalSize := resp.ContentLength
	var downloaded int64
	buf := make([]byte, 32*1024)
	lastProgress := 0

	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			if _, err := tmpFile.Write(buf[:n]); err != nil {
				tmpFile.Close()
				return "", fmt.Errorf("Write file failed: %w", err)
			}
			downloaded += int64(n)

			// update download progress
			if totalSize > 0 {
				progress := int(float64(downloaded) / float64(totalSize) * 100)
				if progress > lastProgress {
					s.eventBus.Broadcast(consts.TopicDowntasksInstalling, &types.DtProgress{
						Stage:      "installing",
						Percentage: float64(progress),
					})
					lastProgress = progress
				}
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			tmpFile.Close()
			return "", fmt.Errorf("Download file failed: %w", err)
		}
	}
	tmpFile.Close()

	// log info: set execute permission
	logger.Info("ytdlp setting permissions", zap.String("path", tmpFile.Name()))

	// set execute permission
	if err := os.Chmod(tmpFile.Name(), 0o755); err != nil {
		return "", fmt.Errorf("Set execute permission failed: %w", err)
	}

	// log info: move to final location
	logger.Info("ytdlp moving to final location",
		zap.String("from", tmpFile.Name()),
		zap.String("to", execPath))

	// move to final location
	if err := os.Rename(tmpFile.Name(), execPath); err != nil {
		// if rename failed (maybe because file already exists), try copy
		if err := copyFile(tmpFile.Name(), execPath); err != nil {
			return "", fmt.Errorf("Move file failed: %w", err)
		}
	}

	s.eventBus.Broadcast(consts.TopicDowntasksInstalling, &types.DtProgress{
		Stage:      "installed",
		Percentage: float64(100),
	})

	// log info: installed
	logger.Info("ytdlp installation completed", zap.String("path", execPath))

	return execPath, nil
}

// copyFile
func copyFile(src, dst string) error {
	// Open source file
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("Open source file failed: %w", err)
	}
	defer sourceFile.Close()

	// Create destination file
	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("Create destination file failed: %w", err)
	}
	defer destFile.Close()

	// Copy content
	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return fmt.Errorf("Copy file content failed: %w", err)
	}

	// Copy permissions
	sourceInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("Get source file info failed: %w", err)
	}

	if err := os.Chmod(dst, sourceInfo.Mode()); err != nil {
		return fmt.Errorf("Set destination file permissions failed: %w", err)
	}

	return nil
}

func ytdlpAvailable() (available bool, path, execPath string, err error) {
	// first check system PATH
	if runtime.GOOS == "windows" {
		// Windows: manually check PATH to avoid command window flash
		pathEnv := os.Getenv("PATH")
		paths := strings.Split(pathEnv, string(os.PathListSeparator))
		for _, p := range paths {
			exe := filepath.Join(p, "yt-dlp.exe")
			if info, err := os.Stat(exe); err == nil && !info.IsDir() {
				return true, p, exe, nil
			}
		}
	} else {
		// Unix-like systems: use LookPath
		if execPath, err := exec.LookPath("yt-dlp"); err == nil {
			return true, filepath.Dir(execPath), execPath, nil
		}
	}

	// if not found, check cache
	dir, err := os.UserCacheDir()
	dir = filepath.Join(dir, "go-ytdlp")
	execPath = filepath.Join(dir, "yt-dlp") // include dir and file
	if runtime.GOOS == "windows" {
		execPath += ".exe"
	}

	if err != nil {
		return false, dir, execPath, fmt.Errorf("Fail to get cached directory: %w", err)
	}

	// Check if file exists
	info, err := os.Stat(execPath)
	if err != nil {
		return false, dir, execPath, fmt.Errorf("yt-dlp not found in cache directory: %s", execPath)
	}

	// Windows: check file exists
	if runtime.GOOS == "windows" {
		return true, dir, execPath, nil
	}

	// Unix: check executable permission
	if info.Mode()&0111 != 0 {
		return true, dir, execPath, nil
	}

	return false, dir, execPath, fmt.Errorf("yt-dlp exists but not executable: %s", execPath)
}

func releaseName() (filename string, err error) {
	// Select filename based on OS
	switch runtime.GOOS {
	case "darwin":
		filename = "yt-dlp_macos"
	case "linux":
		switch runtime.GOARCH {
		case "amd64":
			filename = "yt-dlp_linux"
		case "arm64":
			filename = "yt-dlp_linux_aarch64"
		default:
			err = fmt.Errorf("Unsupported Linux architecture: %s", runtime.GOARCH)
		}
	case "windows":
		filename = "yt-dlp.exe"
	default:
		err = fmt.Errorf("Unsupported OS: %s", runtime.GOOS)
	}

	return
}

func ffmpegAvailable() (available bool, path, execPath string, err error) {
	// first check system PATH
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		// Windows/macOS: manually check PATH to avoid command window flash and sandbox issues
		pathEnv := os.Getenv("PATH")
		paths := strings.Split(pathEnv, string(os.PathListSeparator))
		execName := "ffmpeg"
		if runtime.GOOS == "windows" {
			execName = "ffmpeg.exe"
		}
		for _, p := range paths {
			exe := filepath.Join(p, execName)
			if info, err := os.Stat(exe); err == nil && !info.IsDir() {
				if runtime.GOOS != "windows" {
					// Check if the file is executable on Unix-like systems
					if info.Mode()&0111 != 0 {
						return true, p, exe, nil
					}
				} else {
					return true, p, exe, nil
				}
			}
		}
	} else {
		// Other Unix-like systems: use LookPath
		if execPath, err := exec.LookPath("ffmpeg"); err == nil {
			return true, filepath.Dir(execPath), execPath, nil
		}
	}

	// do not manage ffmpeg right now
	return false, "", "", fmt.Errorf("ffmpeg not found in system PATH")
}

func ffmpegVersion(execPath string, force bool) (string, error) {
	// default return click to check in windows, because exec command calling will flash command window
	if runtime.GOOS == "windows" && !force {
		return "click to check", nil
	}

	cmd := exec.Command(execPath, "-version")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("Get ffmpeg version failed: %w", err)
	}

	return string(output), nil
}
