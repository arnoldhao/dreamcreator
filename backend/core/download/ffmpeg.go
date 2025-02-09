package download

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"CanMe/backend/pkg/specials/cmdrun"
)

func findFFmpeg() string {
	fileName := "ffmpeg"
	if runtime.GOOS == "windows" {
		fileName += ".exe"
	}

	execPath, err := os.Executable()
	if err == nil {
		appDir := filepath.Dir(execPath)
		if runtime.GOOS == "darwin" {
			// if macOS .app checkContents/MacOS directory
			if strings.Contains(appDir, "Contents/MacOS") {
				ffmpegPath := filepath.Join(filepath.Dir(filepath.Dir(appDir)), "Resources", fileName)
				if _, err := os.Stat(ffmpegPath); err == nil {
					return ffmpegPath
				}
			}
		}
		// check app directory
		ffmpegPath := filepath.Join(appDir, fileName)
		if _, err := os.Stat(ffmpegPath); err == nil {
			return ffmpegPath
		}
	}

	// get system PATH
	pathEnv := os.Getenv("PATH")
	if runtime.GOOS == "darwin" {
		// default macOS PATH
		pathEnv = "/usr/local/bin:/usr/bin:/bin:/opt/homebrew/bin:" + pathEnv
	}

	// find ffmpeg
	for _, dir := range filepath.SplitList(pathEnv) {
		path := filepath.Join(dir, fileName)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return fileName
}

func runMuxParts(cmd *exec.Cmd, parts []string) (err error) {
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err = cmd.Run(); err != nil {
		return fmt.Errorf("failed to mux parts: %v, %s", err, stderr.String())
	}

	for _, part := range parts {
		os.Remove(part)
	}

	return nil
}

func ffmpegVersion() (version []byte, err error) {
	cmd := cmdrun.RunCommand(findFFmpeg(), "-version")
	return cmd.Output()
}
