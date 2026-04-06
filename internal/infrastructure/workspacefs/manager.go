package workspacefs

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

const (
	defaultAppDirName   = "dreamcreator"
	defaultWorkspaceDir = "workspaces"
)

type Manager struct {
	baseDir string
}

func DefaultBaseDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, defaultAppDirName, defaultWorkspaceDir), nil
}

func NewManager(baseDir string) (*Manager, error) {
	trimmed := strings.TrimSpace(baseDir)
	if trimmed == "" {
		return nil, errors.New("workspace base dir is required")
	}
	if err := os.MkdirAll(trimmed, 0o755); err != nil {
		return nil, err
	}
	return &Manager{baseDir: trimmed}, nil
}

func (manager *Manager) CreateWorkspace(_ context.Context, workspaceID string) (string, error) {
	root, err := manager.resolveWorkspacePath(workspaceID)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return "", err
	}
	return root, nil
}

func (manager *Manager) EnsureBootstrapFiles(_ context.Context, workspaceID, rootPath string) error {
	root, err := manager.resolveWorkspacePathWithFallback(workspaceID, rootPath)
	if err != nil {
		return err
	}
	return os.MkdirAll(root, 0o755)
}

func (manager *Manager) resolveWorkspacePath(workspaceID string) (string, error) {
	trimmed := strings.TrimSpace(workspaceID)
	if trimmed == "" {
		return "", errors.New("workspace id is required")
	}
	return filepath.Join(manager.baseDir, trimmed), nil
}

func (manager *Manager) resolveWorkspacePathWithFallback(workspaceID, rootPath string) (string, error) {
	trimmedRoot := strings.TrimSpace(rootPath)
	if trimmedRoot != "" {
		return trimmedRoot, nil
	}
	return manager.resolveWorkspacePath(workspaceID)
}
