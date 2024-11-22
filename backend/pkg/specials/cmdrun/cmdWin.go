//go:build windows

package cmdrun

import (
	"os/exec"
	"runtime"
	"syscall"
)

func hideConsoleWindow(cmd *exec.Cmd) {
	if runtime.GOOS == "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{
			HideWindow: true,
		}
	}
}
