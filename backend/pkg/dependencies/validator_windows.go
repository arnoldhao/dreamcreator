//go:build windows

package dependencies

import (
	"context"
	"os/exec"

	"golang.org/x/sys/windows"
)

// createHiddenCommand 创建隐藏窗口的命令（Windows专用）
func createHiddenCommand(ctx context.Context, name string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.SysProcAttr = &windows.SysProcAttr{
		HideWindow:    true,
		CreationFlags: windows.CREATE_NO_WINDOW,
	}
	return cmd
}
