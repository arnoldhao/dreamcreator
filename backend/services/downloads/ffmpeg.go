package downloads

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

func findFFmpeg() string {
	fileName := "ffmpeg"
	if runtime.GOOS == "windows" {
		fileName += ".exe"
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

func (wq *WorkQueue) muxParts(parts []string, mergedFilePath string) (err error) {
	cmd := []string{"-y"}

	for _, part := range parts {
		cmd = append(cmd, "-i", part)
	}

	cmd = append(cmd, "-c:v", "copy", "-c:a", "copy", mergedFilePath)
	return runMuxParts(exec.Command(findFFmpeg(), cmd...), parts)
}
