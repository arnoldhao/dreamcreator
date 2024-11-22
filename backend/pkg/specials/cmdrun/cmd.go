//go:build !windows

package cmdrun

import (
	"os/exec"
)

func hideConsoleWindow(cmd *exec.Cmd) {
}
