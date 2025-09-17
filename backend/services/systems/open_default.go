//go:build !windows

package systems

import "os/exec"

// Non-Windows stub to satisfy references; never called on non-Windows.
func winOpenCmd(path string) *exec.Cmd { return exec.Command("sh", "-c", "true") }

