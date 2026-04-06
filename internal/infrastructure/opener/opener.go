package opener

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

func OpenDirectory(path string) error {
	if path == "" {
		return fmt.Errorf("directory is empty")
	}
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("access directory: %w", err)
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", path)
	case "windows":
		cmd = exec.Command("explorer", path)
	default:
		cmd = exec.Command("xdg-open", path)
	}

	return cmd.Start()
}
