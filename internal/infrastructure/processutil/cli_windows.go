//go:build windows

package processutil

import (
	"os/exec"
	"syscall"
)

// ConfigureCLI applies common Windows process attributes for CLI invocations.
func ConfigureCLI(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.HideWindow = true
}

