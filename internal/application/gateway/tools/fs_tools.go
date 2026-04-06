package tools

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

type readResult struct {
	Path       string `json:"path"`
	Content    string `json:"content"`
	Truncated  bool   `json:"truncated,omitempty"`
	Size       int    `json:"size"`
	TotalBytes int64  `json:"totalBytes,omitempty"`
}

func runReadTool(ctx context.Context, args string) (string, error) {
	_ = ctx
	payload, err := parseToolArgs(args)
	if err != nil {
		return "", err
	}
	path, err := resolvePath(payload)
	if err != nil {
		return "", err
	}
	maxChars, _ := getIntArg(payload, "maxChars", "max_chars", "limit")
	data, truncated, err := readFileLimited(path, maxChars)
	if err != nil {
		return "", err
	}
	info, _ := os.Stat(path)
	result := readResult{
		Path:      path,
		Content:   string(data),
		Truncated: truncated,
		Size:      len(data),
	}
	if info != nil {
		result.TotalBytes = info.Size()
	}
	return marshalResult(result), nil
}

type writeResult struct {
	Path         string `json:"path"`
	BytesWritten int    `json:"bytesWritten"`
}

func runWriteTool(ctx context.Context, args string) (string, error) {
	_ = ctx
	payload, err := parseToolArgs(args)
	if err != nil {
		return "", err
	}
	path, err := resolvePath(payload)
	if err != nil {
		return "", err
	}
	content := getStringArg(payload, "content", "input", "text")
	appendMode, _ := getBoolArg(payload, "append")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	flags := os.O_CREATE | os.O_WRONLY
	if appendMode {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
	}
	file, err := os.OpenFile(path, flags, 0o644)
	if err != nil {
		return "", err
	}
	defer file.Close()
	n, err := file.WriteString(content)
	if err != nil {
		return "", err
	}
	return marshalResult(writeResult{Path: path, BytesWritten: n}), nil
}

type editResult struct {
	Path     string `json:"path"`
	Replaced int    `json:"replaced"`
}

func runEditTool(ctx context.Context, args string) (string, error) {
	_ = ctx
	payload, err := parseToolArgs(args)
	if err != nil {
		return "", err
	}
	path, err := resolvePath(payload)
	if err != nil {
		return "", err
	}
	oldText := getStringArg(payload, "oldText", "old", "find")
	newText := getStringArg(payload, "newText", "new", "replace")
	if oldText == "" {
		return "", errors.New("oldText is required")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	content := string(data)
	replaceAll, _ := getBoolArg(payload, "replaceAll", "all")
	replaced := 0
	if replaceAll {
		replaced = strings.Count(content, oldText)
		if replaced == 0 {
			return "", errors.New("text not found")
		}
		content = strings.ReplaceAll(content, oldText, newText)
	} else {
		index := strings.Index(content, oldText)
		if index < 0 {
			return "", errors.New("text not found")
		}
		replaced = 1
		content = strings.Replace(content, oldText, newText, 1)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return "", err
	}
	return marshalResult(editResult{Path: path, Replaced: replaced}), nil
}

func runApplyPatchTool(ctx context.Context, args string) (string, error) {
	_ = ctx
	payload, err := parseToolArgs(args)
	if err != nil {
		return "", err
	}
	patch := getStringArg(payload, "patch", "diff")
	if patch == "" {
		return "", errors.New("patch is required")
	}
	root := getStringArg(payload, "rootPath", "root", "cwd")
	if strings.TrimSpace(root) == "" {
		root = "."
	}
	cmdArgs := []string{"apply", "--whitespace=nowarn", "-"}
	exitCode, stdout, stderr, err := runCommandWithInput(ctx, root, "git", cmdArgs, patch, nil, 0)
	if err != nil {
		return "", err
	}
	if exitCode != 0 {
		message := strings.TrimSpace(stderr)
		if message == "" {
			message = strings.TrimSpace(stdout)
		}
		if message == "" {
			message = "git apply failed"
		}
		return "", errors.New(message)
	}
	return marshalResult(map[string]any{"ok": true}), nil
}
