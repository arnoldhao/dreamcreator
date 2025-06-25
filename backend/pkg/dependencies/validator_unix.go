//go:build !windows

package dependencies

import (
	"context"
	"os/exec"
)

// createHiddenCommand 创建命令（Unix/Linux/macOS）
func createHiddenCommand(ctx context.Context, name string, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, name, args...)
}
