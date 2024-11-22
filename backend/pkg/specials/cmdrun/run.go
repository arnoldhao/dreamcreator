package cmdrun

import (
	"os/exec"
)

// RunCommand runs a command with hidden console window
func RunCommand(name string, arg ...string) *exec.Cmd {
	cmd := exec.Command(name, arg...)
	hideConsoleWindow(cmd)
	return cmd
}
