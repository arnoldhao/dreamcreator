package service

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	assistantassets "dreamcreator/internal/assets/assistant"
	"dreamcreator/internal/domain/assistant"
)

const assistantBuiltinDirName = "builtin"

func ensureBuiltinAssets() error {
	dir, err := builtinAssetsDir(true)
	if err != nil {
		return err
	}
	if err := ensureBuiltinAssetFile(dir, assistantassets.BuiltinAvatarFile); err != nil {
		return err
	}
	if err := ensureBuiltinAssetFile(dir, assistantassets.BuiltinMotionFile); err != nil {
		return err
	}
	return nil
}

func builtinAssetsDir(ensure bool) (string, error) {
	baseDir, err := avatarBaseDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(baseDir, assistantBuiltinDirName)
	if ensure {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", err
		}
	}
	return dir, nil
}

func ensureBuiltinAssetFile(dir, name string) error {
	if strings.TrimSpace(name) == "" {
		return errors.New("builtin asset name is required")
	}
	path := filepath.Join(dir, name)
	if info, err := os.Stat(path); err == nil {
		if !info.IsDir() && info.Size() > 0 {
			return nil
		}
	}
	file, err := assistantassets.FS.Open(name)
	if err != nil {
		return err
	}
	defer file.Close()
	payload, err := io.ReadAll(file)
	if err != nil {
		return err
	}
	return os.WriteFile(path, payload, 0o644)
}

func ensureBuiltinAvatarRefs(avatar assistant.AssistantAvatar) (assistant.AssistantAvatar, bool, error) {
	changed := false
	updated := avatar
	if ref, refChanged, err := ensureBuiltinAvatarRef(updated.Avatar3D, assistantassets.BuiltinAvatarAssetID, assistantassets.BuiltinAvatarFile, assistantassets.BuiltinAvatarDisplayName); err != nil {
		return avatar, false, err
	} else if refChanged {
		updated.Avatar3D = ref
		changed = true
	}
	if ref, refChanged, err := ensureBuiltinAvatarRef(updated.Motion, assistantassets.BuiltinMotionAssetID, assistantassets.BuiltinMotionFile, assistantassets.BuiltinMotionDisplayName); err != nil {
		return avatar, false, err
	} else if refChanged {
		updated.Motion = ref
		changed = true
	}
	return updated, changed, nil
}

func ensureBuiltinAvatarRef(ref assistant.AssistantAvatarAssetRef, assetID, fileName, displayName string) (assistant.AssistantAvatarAssetRef, bool, error) {
	isBuiltin := strings.TrimSpace(ref.Source) == assistant.AvatarSourceBuiltin || strings.TrimSpace(ref.AssetID) == assetID
	if !isBuiltin {
		return ref, false, nil
	}
	if err := ensureBuiltinAssets(); err != nil {
		return ref, false, err
	}
	dir, err := builtinAssetsDir(false)
	if err != nil {
		return ref, false, err
	}
	path := filepath.Join(dir, fileName)
	changed := false
	if strings.TrimSpace(ref.Path) == "" || !fileExists(ref.Path) || filepath.Clean(ref.Path) != filepath.Clean(path) {
		ref.Path = path
		changed = true
	}
	if strings.TrimSpace(ref.Source) == "" {
		ref.Source = assistant.AvatarSourceBuiltin
		changed = true
	}
	if strings.TrimSpace(ref.AssetID) == "" {
		ref.AssetID = assetID
		changed = true
	}
	if strings.TrimSpace(ref.DisplayName) == "" && strings.TrimSpace(displayName) != "" {
		ref.DisplayName = displayName
		changed = true
	}
	return ref, changed, nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}
