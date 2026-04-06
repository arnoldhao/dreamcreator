package tools

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"dreamcreator/internal/infrastructure/processutil"
)

const defaultExecTimeoutSeconds = 60

type execResult struct {
	Stdout   string `json:"stdout,omitempty"`
	Stderr   string `json:"stderr,omitempty"`
	ExitCode int    `json:"exitCode"`
}

func runExecTool(ctx context.Context, args string) (string, error) {
	payload, err := parseToolArgs(args)
	if err != nil {
		return "", err
	}
	timeoutSeconds, ok := getIntArg(payload, "timeoutSeconds", "timeout", "timeoutSec")
	if !ok || timeoutSeconds <= 0 {
		timeoutSeconds = defaultExecTimeoutSeconds
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
	defer cancel()
	command, cmdArgs, err := resolveCommand(payload)
	if err != nil {
		return "", err
	}
	cwd := getStringArg(payload, "cwd", "workingDir", "workingDirectory")
	env := getMapArg(payload, "env")
	exitCode, stdout, stderr, err := runCommandWithInput(ctx, cwd, command, cmdArgs, "", env, 0)
	if err != nil {
		return "", err
	}
	result := execResult{
		Stdout:   stdout,
		Stderr:   stderr,
		ExitCode: exitCode,
	}
	return marshalResult(result), nil
}

type processResult struct {
	PID     int    `json:"pid"`
	Command string `json:"command"`
}

func runProcessTool(ctx context.Context, args string) (string, error) {
	payload, err := parseToolArgs(args)
	if err != nil {
		return "", err
	}
	command, cmdArgs, err := resolveCommand(payload)
	if err != nil {
		return "", err
	}
	cwd := getStringArg(payload, "cwd", "workingDir", "workingDirectory")
	cmd := exec.CommandContext(ctx, command, cmdArgs...)
	processutil.ConfigureCLI(cmd)
	if strings.TrimSpace(cwd) != "" {
		cmd.Dir = cwd
	}
	if err := cmd.Start(); err != nil {
		return "", err
	}
	return marshalResult(processResult{PID: cmd.Process.Pid, Command: command}), nil
}

func resolveCommand(args toolArgs) (string, []string, error) {
	if raw, ok := args["cmd"]; ok {
		switch typed := raw.(type) {
		case []string:
			if len(typed) == 0 {
				return "", nil, errors.New("cmd is empty")
			}
			return typed[0], typed[1:], nil
		case []any:
			items := make([]string, 0, len(typed))
			for _, entry := range typed {
				if value, ok := entry.(string); ok && strings.TrimSpace(value) != "" {
					items = append(items, strings.TrimSpace(value))
				}
			}
			if len(items) == 0 {
				return "", nil, errors.New("cmd is empty")
			}
			return items[0], items[1:], nil
		}
	}
	command := getStringArg(args, "command", "cmdline", "cmd")
	if command == "" {
		return "", nil, errors.New("command is required")
	}
	if runtime.GOOS == "windows" {
		return "cmd", []string{"/c", command}, nil
	}
	return "bash", []string{"-lc", command}, nil
}

func runCommandWithInput(ctx context.Context, cwd string, command string, cmdArgs []string, input string, env map[string]any, maxOutput int) (int, string, string, error) {
	cmd := exec.CommandContext(ctx, command, cmdArgs...)
	processutil.ConfigureCLI(cmd)
	if strings.TrimSpace(cwd) != "" {
		cmd.Dir = cwd
	}
	if len(env) > 0 {
		merged := append([]string{}, cmd.Environ()...)
		for key, value := range env {
			if strings.TrimSpace(key) == "" {
				continue
			}
			merged = append(merged, key+"="+strings.TrimSpace(toString(value)))
		}
		cmd.Env = merged
	}
	if input != "" {
		cmd.Stdin = strings.NewReader(input)
	}
	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf
	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
			err = nil
		} else if errors.Is(err, context.DeadlineExceeded) {
			return -1, stdoutBuf.String(), stderrBuf.String(), err
		} else {
			return -1, stdoutBuf.String(), stderrBuf.String(), err
		}
	}
	stdout := stdoutBuf.String()
	stderr := stderrBuf.String()
	if maxOutput > 0 {
		stdout = truncateString(stdout, maxOutput)
		stderr = truncateString(stderr, maxOutput)
	}
	return exitCode, stdout, stderr, err
}

func toString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case []byte:
		return string(typed)
	default:
		return ""
	}
}

func truncateString(value string, max int) string {
	if max <= 0 || len(value) <= max {
		return value
	}
	return value[:max]
}
