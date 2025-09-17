//go:build windows

package systems

import (
    "os/exec"
    "syscall"
)

// winOpenCmd returns a command that opens the given path with the system default
// application while suppressing any console window on Windows.
func winOpenCmd(path string) *exec.Cmd {
    // Use `cmd /c start "" <path>` to respect file associations.
    cmd := exec.Command("cmd", "/c", "start", "", path)
    // Hide the transient console window when launching.
    cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
    return cmd
}

