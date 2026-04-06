package tools

import (
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

const (
	defaultMediaPathAppDirName = "dreamcreator"
)

func resolveDefaultMediaLocalRoots() []string {
	roots := []string{
		filepath.Join(os.TempDir(), defaultMediaPathAppDirName),
	}
	configDir, err := os.UserConfigDir()
	if err != nil {
		return normalizePathRoots(roots)
	}
	appDir := filepath.Join(configDir, defaultMediaPathAppDirName)
	roots = append(
		roots,
		filepath.Join(appDir, "media"),
		filepath.Join(appDir, "agents"),
		filepath.Join(appDir, "workspace"),
		filepath.Join(appDir, "workspaces"),
		filepath.Join(appDir, "sandboxes"),
	)
	return normalizePathRoots(roots)
}

func normalizePathRoots(roots []string) []string {
	if len(roots) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	result := make([]string, 0, len(roots))
	for _, root := range roots {
		trimmed := strings.TrimSpace(root)
		if trimmed == "" {
			continue
		}
		cleaned := filepath.Clean(trimmed)
		abs, err := filepath.Abs(cleaned)
		if err == nil {
			cleaned = abs
		}
		if cleaned == "" {
			continue
		}
		if _, exists := seen[cleaned]; exists {
			continue
		}
		seen[cleaned] = struct{}{}
		result = append(result, cleaned)
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func resolveInboundPath(raw string, roots []string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", errors.New("path is required")
	}
	if strings.HasPrefix(strings.ToLower(value), "file://") {
		parsed, err := url.Parse(value)
		if err != nil {
			return "", err
		}
		value = parsed.Path
		if value == "" {
			value = strings.TrimPrefix(raw, "file://")
		}
	}
	if strings.HasPrefix(value, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if value == "~" {
			value = home
		} else {
			value = filepath.Join(home, strings.TrimPrefix(value, "~/"))
		}
	}
	cleaned := filepath.Clean(value)
	abs, err := filepath.Abs(cleaned)
	if err == nil {
		cleaned = abs
	}
	normalizedRoots := normalizePathRoots(roots)
	if len(normalizedRoots) == 0 {
		normalizedRoots = resolveDefaultMediaLocalRoots()
	}
	if !pathWithinAnyRoot(cleaned, normalizedRoots) {
		return "", errors.New("path outside allowed roots")
	}
	canonical, err := filepath.EvalSymlinks(cleaned)
	if err == nil && canonical != "" {
		if !pathWithinAnyRoot(canonical, normalizedRoots) {
			return "", errors.New("path outside allowed roots")
		}
		return canonical, nil
	}
	return cleaned, nil
}

func pathWithinAnyRoot(path string, roots []string) bool {
	normalizedPath := strings.TrimSpace(path)
	if normalizedPath == "" {
		return false
	}
	for _, root := range roots {
		if isSubpath(root, normalizedPath) {
			return true
		}
	}
	return false
}
