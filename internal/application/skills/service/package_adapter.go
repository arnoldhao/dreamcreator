package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

type SkillsPackageAdapter interface {
	Run(ctx context.Context, workspaceRoot string, timeout time.Duration, args ...string) ([]byte, error)
}

type clawHubPackageAdapter struct {
	service *SkillsService
}

func newClawHubPackageAdapter(service *SkillsService) SkillsPackageAdapter {
	return &clawHubPackageAdapter{service: service}
}

func (adapter *clawHubPackageAdapter) Run(
	ctx context.Context,
	workspaceRoot string,
	timeout time.Duration,
	args ...string,
) ([]byte, error) {
	if adapter == nil || adapter.service == nil {
		return nil, errors.New("skills package adapter unavailable")
	}
	execPath, err := adapter.service.resolveClawHubExecPath(ctx)
	if err != nil {
		return nil, err
	}
	if timeout <= 0 {
		timeout = skillsManageTimeout
	}
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var command *exec.Cmd
	invocationArgs := make([]string, 0, len(args)+3)
	workdir := resolveDreamCreatorAppRoot(workspaceRoot)
	if workdir != "" {
		if err := os.MkdirAll(workdir, 0o755); err != nil {
			return nil, err
		}
		invocationArgs = append(invocationArgs, "--workdir", workdir)
	}
	invocationArgs = append(invocationArgs, "--no-input")
	invocationArgs = append(invocationArgs, args...)

	if runtime.GOOS == "windows" && strings.HasSuffix(strings.ToLower(execPath), ".cmd") {
		command = exec.CommandContext(execCtx, "cmd", append([]string{"/c", execPath}, invocationArgs...)...)
	} else {
		command = exec.CommandContext(execCtx, execPath, invocationArgs...)
	}
	if workdir != "" {
		command.Dir = workdir
	}
	configurePackageCommand(command)
	command.Env = append(
		os.Environ(),
		"NO_COLOR=1",
		"FORCE_COLOR=0",
	)
	output, err := command.CombinedOutput()
	if err != nil {
		if execCtx.Err() != nil {
			return nil, execCtx.Err()
		}
		message := strings.TrimSpace(stripANSI(string(output)))
		if message == "" {
			message = err.Error()
		}
		if classified := classifyClawHubCommandError(args, message); classified != nil {
			return nil, classified
		}
		return nil, fmt.Errorf("clawhub %s failed: %s", strings.Join(args, " "), message)
	}
	return output, nil
}
