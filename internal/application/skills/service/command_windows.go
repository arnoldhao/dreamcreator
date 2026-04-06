//go:build windows

package service

import (
	"os/exec"
	"syscall"
)

func configurePackageCommand(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
}
