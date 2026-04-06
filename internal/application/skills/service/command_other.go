//go:build !windows

package service

import "os/exec"

func configurePackageCommand(cmd *exec.Cmd) {}
