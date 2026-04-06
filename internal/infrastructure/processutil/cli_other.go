//go:build !windows

package processutil

import "os/exec"

// ConfigureCLI is a no-op outside Windows.
func ConfigureCLI(cmd *exec.Cmd) {}

