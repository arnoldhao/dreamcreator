package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	assistantassets "dreamcreator/internal/assets/assistant"
	"dreamcreator/internal/application/assistant/dto"
	"dreamcreator/internal/domain/assistant"
)

const (
	assistantAssetRootDirName = "objects"
	assistant3DAvatarDirName  = "3davatar"
	assistant3DMotionDirName  = "3davatar-motion"
	maxAvatarSourceBytes      = 100 * 1024 * 1024
)

var avatarKindPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,31}$`)
var avatarFileStemPattern = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)
var avatar3DExtToMime = map[string]string{
	".glb": "model/gltf-binary",
	".vrm": "model/gltf-binary",
}
var avatar3DMotionExtToMime = map[string]string{
	".vrma": "model/gltf-binary",
}

type avatarAssetMetadata struct {
	DisplayName string `json:"displayName,omitempty"`
}

func (service *AssistantService) ListAvatarAssets(_ context.Context, kind string) ([]dto.AssistantAvatarAsset, error) {
	normalized, err := normalizeAvatarKind(kind)
	if err != nil {
		return nil, err
	}

	dir, err := avatarKindDir(normalized, false)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []dto.AssistantAvatarAsset{}, nil
		}
		return nil, err
	}

	assets := make([]dto.AssistantAvatarAsset, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		entryName := entry.Name()
		ext := strings.ToLower(filepath.Ext(entryName))
		switch normalized {
		case "3davatar":
			if _, ok := avatar3DExtToMime[ext]; !ok {
				continue
			}
		case "vrma":
			if _, ok := avatar3DMotionExtToMime[ext]; !ok {
				continue
			}
		default:
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		path := filepath.Join(dir, entryName)
		displayName := readAvatarAssetDisplayName(path)
		assets = append(assets, dto.AssistantAvatarAsset{
			Kind:        normalized,
			Path:        path,
			Name:        entryName,
			DisplayName: displayName,
			UpdatedAt:   info.ModTime().Format(time.RFC3339),
			Source:      assistant.AvatarSourceUser,
		})
	}

	builtinAssets, err := listBuiltinAvatarAssets(normalized)
	if err != nil {
		return nil, err
	}
	assets = append(assets, builtinAssets...)

	sort.Slice(assets, func(i, j int) bool {
		return assets[i].UpdatedAt > assets[j].UpdatedAt
	})

	return assets, nil
}

func (service *AssistantService) ImportAvatarAsset(_ context.Context, request dto.ImportAssistantAvatarRequest) (dto.AssistantAvatarAsset, error) {
	normalized, err := normalizeAvatarKind(request.Kind)
	if err != nil {
		return dto.AssistantAvatarAsset{}, err
	}

	payload, err := decodeBase64Payload(request.ContentBase64)
	if err != nil {
		return dto.AssistantAvatarAsset{}, err
	}
	return storeAvatarPayload(normalized, request.FileName, payload)
}

func avatarBaseDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "dreamcreator", assistantAssetRootDirName), nil
}

func avatarKindDir(kind string, ensure bool) (string, error) {
	baseDir, err := avatarBaseDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(baseDir, resolveAvatarDirName(kind))
	if ensure {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", err
		}
	}
	return dir, nil
}

func normalizeAvatarKind(kind string) (string, error) {
	trimmed := strings.ToLower(strings.TrimSpace(kind))
	if trimmed == "" {
		return "", errors.New("avatar kind is required")
	}
	if !avatarKindPattern.MatchString(trimmed) {
		return "", errors.New("invalid avatar kind")
	}
	if trimmed != "3davatar" && trimmed != "vrma" {
		return "", errors.New("unsupported avatar kind")
	}
	return trimmed, nil
}

func decodeBase64Payload(raw string) ([]byte, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, errors.New("content is required")
	}
	lower := strings.ToLower(trimmed)
	if strings.HasPrefix(lower, "data:") {
		if parts := strings.SplitN(trimmed, ",", 2); len(parts) == 2 {
			trimmed = parts[1]
		}
	}
	decoded, err := base64.StdEncoding.DecodeString(trimmed)
	if err == nil {
		return decoded, nil
	}
	return base64.RawStdEncoding.DecodeString(trimmed)
}

func sanitizeFileName(name string) string {
	base := strings.TrimSpace(name)
	if base == "" {
		return ""
	}
	base = filepath.Base(base)
	if base == "." || base == string(filepath.Separator) {
		return ""
	}
	return base
}

func sanitizeFileStem(stem string) string {
	value := strings.TrimSpace(stem)
	if value == "" {
		return ""
	}
	value = avatarFileStemPattern.ReplaceAllString(value, "-")
	return strings.Trim(value, "-_")
}

func resolveAvatarDirName(kind string) string {
	switch kind {
	case "3davatar":
		return assistant3DAvatarDirName
	case "vrma":
		return assistant3DMotionDirName
	default:
		return kind
	}
}

func resolveAvatarMime(kind, path string) string {
	ext := strings.ToLower(filepath.Ext(strings.TrimSpace(path)))
	switch kind {
	case "3davatar":
		return avatar3DExtToMime[ext]
	case "vrma":
		return avatar3DMotionExtToMime[ext]
	default:
		return ""
	}
}

func listBuiltinAvatarAssets(kind string) ([]dto.AssistantAvatarAsset, error) {
	if err := ensureBuiltinAssets(); err != nil {
		return nil, err
	}
	dir, err := builtinAssetsDir(false)
	if err != nil {
		return nil, err
	}
	switch kind {
	case "3davatar":
		return []dto.AssistantAvatarAsset{builtinAvatarAsset(dir)}, nil
	case "vrma":
		return []dto.AssistantAvatarAsset{builtinMotionAsset(dir)}, nil
	default:
		return nil, nil
	}
}

func builtinAvatarAsset(dir string) dto.AssistantAvatarAsset {
	path := filepath.Join(dir, assistantassets.BuiltinAvatarFile)
	return builtinAssetFromPath("3davatar", path, assistantassets.BuiltinAvatarAssetID, assistantassets.BuiltinAvatarDisplayName)
}

func builtinMotionAsset(dir string) dto.AssistantAvatarAsset {
	path := filepath.Join(dir, assistantassets.BuiltinMotionFile)
	return builtinAssetFromPath("vrma", path, assistantassets.BuiltinMotionAssetID, assistantassets.BuiltinMotionDisplayName)
}

func builtinAssetFromPath(kind, path, assetID, fallbackName string) dto.AssistantAvatarAsset {
	info, err := os.Stat(path)
	updatedAt := ""
	if err == nil && !info.IsDir() {
		updatedAt = info.ModTime().Format(time.RFC3339)
	}
	displayName := readAvatarAssetDisplayName(path)
	if strings.TrimSpace(displayName) == "" {
		displayName = strings.TrimSpace(fallbackName)
	}
	return dto.AssistantAvatarAsset{
		Kind:        kind,
		Path:        path,
		Name:        filepath.Base(path),
		DisplayName: displayName,
		UpdatedAt:   updatedAt,
		Source:      assistant.AvatarSourceBuiltin,
		AssetID:     assetID,
	}
}

func validateAvatarSourcePath(kind, path string) error {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return errors.New("path is required")
	}
	ext := strings.ToLower(filepath.Ext(trimmed))
	switch kind {
	case "3davatar":
		if _, ok := avatar3DExtToMime[ext]; !ok {
			return errors.New("unsupported 3d avatar type")
		}
		return nil
	case "vrma":
		if _, ok := avatar3DMotionExtToMime[ext]; !ok {
			return errors.New("unsupported 3d motion type")
		}
		return nil
	default:
		return errors.New("unsupported avatar kind")
	}
}

func normalizeAvatarSourcePath(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return ""
	}
	value = decodeRepeatedly(value, 2)
	lower := strings.ToLower(value)
	if strings.HasPrefix(lower, "file://") {
		value = value[len("file://"):]
		lower = lower[len("file://"):]
	} else if strings.HasPrefix(lower, "file:") {
		value = value[len("file:"):]
		lower = lower[len("file:"):]
	}
	if len(value) >= 3 && value[0] == '/' && value[2] == ':' {
		value = value[1:]
	}
	return value
}

func avatarAssetMetaPath(assetPath string) string {
	return assetPath + ".meta.json"
}

func readAvatarAssetDisplayName(assetPath string) string {
	metaPath := avatarAssetMetaPath(assetPath)
	payload, err := os.ReadFile(metaPath)
	if err != nil {
		return ""
	}
	var meta avatarAssetMetadata
	if err := json.Unmarshal(payload, &meta); err != nil {
		return ""
	}
	return strings.TrimSpace(meta.DisplayName)
}

func writeAvatarAssetDisplayName(assetPath, displayName string) error {
	metaPath := avatarAssetMetaPath(assetPath)
	trimmed := strings.TrimSpace(displayName)
	if trimmed == "" {
		if err := os.Remove(metaPath); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	payload, err := json.Marshal(avatarAssetMetadata{DisplayName: trimmed})
	if err != nil {
		return err
	}
	return os.WriteFile(metaPath, payload, 0o644)
}

func decodeRepeatedly(value string, max int) string {
	current := value
	for i := 0; i < max; i++ {
		decoded, err := url.PathUnescape(current)
		if err != nil {
			return current
		}
		if decoded == current {
			break
		}
		current = decoded
	}
	return current
}

func (service *AssistantService) ImportAvatarAssetFromPath(_ context.Context, request dto.ImportAssistantAvatarFromPathRequest) (dto.AssistantAvatarAsset, error) {
	normalized, err := normalizeAvatarKind(request.Kind)
	if err != nil {
		return dto.AssistantAvatarAsset{}, err
	}
	normalizedPath := normalizeAvatarSourcePath(request.Path)
	if err := validateAvatarSourcePath(normalized, normalizedPath); err != nil {
		return dto.AssistantAvatarAsset{}, err
	}
	trimmed := strings.TrimSpace(normalizedPath)
	if trimmed == "" {
		return dto.AssistantAvatarAsset{}, errors.New("path is required")
	}
	info, err := os.Stat(trimmed)
	if err != nil {
		return dto.AssistantAvatarAsset{}, err
	}
	if info.IsDir() {
		return dto.AssistantAvatarAsset{}, errors.New("path is a directory")
	}
	payload, err := os.ReadFile(trimmed)
	if err != nil {
		return dto.AssistantAvatarAsset{}, err
	}
	return storeAvatarPayload(normalized, filepath.Base(trimmed), payload)
}

func (service *AssistantService) ReadAvatarSource(_ context.Context, request dto.ReadAssistantAvatarSourceRequest) (dto.ReadAssistantAvatarSourceResponse, error) {
	normalized, err := normalizeAvatarKind(request.Kind)
	if err != nil {
		return dto.ReadAssistantAvatarSourceResponse{}, err
	}
	normalizedPath := normalizeAvatarSourcePath(request.Path)
	if err := validateAvatarSourcePath(normalized, normalizedPath); err != nil {
		return dto.ReadAssistantAvatarSourceResponse{}, err
	}
	trimmed := strings.TrimSpace(normalizedPath)
	info, err := os.Stat(trimmed)
	if err != nil {
		return dto.ReadAssistantAvatarSourceResponse{}, err
	}
	if info.IsDir() {
		return dto.ReadAssistantAvatarSourceResponse{}, errors.New("path is a directory")
	}
	if info.Size() > maxAvatarSourceBytes {
		return dto.ReadAssistantAvatarSourceResponse{}, fmt.Errorf("file too large (max %d bytes)", maxAvatarSourceBytes)
	}
	payload, err := os.ReadFile(trimmed)
	if err != nil {
		return dto.ReadAssistantAvatarSourceResponse{}, err
	}
	mime := resolveAvatarMime(normalized, trimmed)
	if mime == "" {
		return dto.ReadAssistantAvatarSourceResponse{}, errors.New("unsupported file type")
	}
	return dto.ReadAssistantAvatarSourceResponse{
		ContentBase64: base64.StdEncoding.EncodeToString(payload),
		Mime:          mime,
		FileName:      filepath.Base(trimmed),
		SizeBytes:     info.Size(),
	}, nil
}

func (service *AssistantService) DeleteAvatarAsset(_ context.Context, request dto.DeleteAssistantAvatarAssetRequest) error {
	normalized, err := normalizeAvatarKind(request.Kind)
	if err != nil {
		return err
	}
	normalizedPath := normalizeAvatarSourcePath(request.Path)
	if err := validateAvatarSourcePath(normalized, normalizedPath); err != nil {
		return err
	}
	trimmed := strings.TrimSpace(normalizedPath)
	if trimmed == "" {
		return errors.New("path is required")
	}
	if isBuiltinAssetPath(trimmed) {
		return errors.New("builtin asset cannot be deleted")
	}
	kindDir, err := avatarKindDir(normalized, false)
	if err != nil {
		return err
	}
	absDir, err := filepath.Abs(kindDir)
	if err != nil {
		return err
	}
	absPath := trimmed
	if !filepath.IsAbs(absPath) {
		absPath, err = filepath.Abs(absPath)
		if err != nil {
			return err
		}
	}
	absDir = filepath.Clean(absDir)
	absPath = filepath.Clean(absPath)
	rel, err := filepath.Rel(absDir, absPath)
	if err != nil {
		return err
	}
	if rel == "." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." {
		return errors.New("path is outside avatar assets")
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return errors.New("path is a directory")
	}
	if err := os.Remove(absPath); err != nil {
		return err
	}
	if err := os.Remove(avatarAssetMetaPath(absPath)); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (service *AssistantService) UpdateAvatarAsset(_ context.Context, request dto.UpdateAssistantAvatarAssetRequest) (dto.AssistantAvatarAsset, error) {
	normalized, err := normalizeAvatarKind(request.Kind)
	if err != nil {
		return dto.AssistantAvatarAsset{}, err
	}
	normalizedPath := normalizeAvatarSourcePath(request.Path)
	if err := validateAvatarSourcePath(normalized, normalizedPath); err != nil {
		return dto.AssistantAvatarAsset{}, err
	}
	trimmed := strings.TrimSpace(normalizedPath)
	if trimmed == "" {
		return dto.AssistantAvatarAsset{}, errors.New("path is required")
	}
	kindDir, err := avatarKindDir(normalized, false)
	if err != nil {
		return dto.AssistantAvatarAsset{}, err
	}
	absDir, err := filepath.Abs(kindDir)
	if err != nil {
		return dto.AssistantAvatarAsset{}, err
	}
	absPath := trimmed
	if !filepath.IsAbs(absPath) {
		absPath, err = filepath.Abs(absPath)
		if err != nil {
			return dto.AssistantAvatarAsset{}, err
		}
	}
	absDir = filepath.Clean(absDir)
	absPath = filepath.Clean(absPath)
	rel, err := filepath.Rel(absDir, absPath)
	if err != nil {
		return dto.AssistantAvatarAsset{}, err
	}
	if rel == "." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." {
		return dto.AssistantAvatarAsset{}, errors.New("path is outside avatar assets")
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return dto.AssistantAvatarAsset{}, err
	}
	if info.IsDir() {
		return dto.AssistantAvatarAsset{}, errors.New("path is a directory")
	}
	if err := writeAvatarAssetDisplayName(absPath, request.DisplayName); err != nil {
		return dto.AssistantAvatarAsset{}, err
	}
	displayName := strings.TrimSpace(request.DisplayName)
	source, assetID := resolveAssetSourceAndID(absPath)
	return dto.AssistantAvatarAsset{
		Kind:        normalized,
		Path:        absPath,
		Name:        filepath.Base(absPath),
		DisplayName: displayName,
		UpdatedAt:   info.ModTime().Format(time.RFC3339),
		Source:      source,
		AssetID:     assetID,
	}, nil
}

func storeAvatarPayload(kind, fileName string, payload []byte) (dto.AssistantAvatarAsset, error) {
	dir, err := avatarKindDir(kind, true)
	if err != nil {
		return dto.AssistantAvatarAsset{}, err
	}

	baseName := sanitizeFileName(fileName)
	if baseName == "" {
		if kind == "vrma" {
			baseName = "3dmotion"
		} else {
			baseName = "3davatar"
		}
	}

	ext := strings.ToLower(filepath.Ext(baseName))
	if ext == "" {
		if kind == "vrma" {
			ext = ".vrma"
		} else {
			ext = ".glb"
		}
	}

	stem := strings.TrimSuffix(baseName, filepath.Ext(baseName))
	stem = sanitizeFileStem(stem)
	if stem == "" {
		if kind == "vrma" {
			stem = "3dmotion"
		} else {
			stem = "3davatar"
		}
	}

	targetName := fmt.Sprintf("%s-%s%s", stem, uuid.NewString(), ext)
	targetPath := filepath.Join(dir, targetName)
	if err := os.WriteFile(targetPath, payload, 0o644); err != nil {
		return dto.AssistantAvatarAsset{}, err
	}

	return dto.AssistantAvatarAsset{
		Kind:      kind,
		Path:      targetPath,
		Name:      targetName,
		UpdatedAt: time.Now().Format(time.RFC3339),
		Source:    assistant.AvatarSourceUser,
	}, nil
}

func resolveAssetSourceAndID(path string) (string, string) {
	normalized := strings.TrimSpace(path)
	if normalized == "" {
		return assistant.AvatarSourceUser, ""
	}
	builtinDir, err := builtinAssetsDir(false)
	if err == nil {
		absDir, dirErr := filepath.Abs(builtinDir)
		absPath, pathErr := filepath.Abs(normalized)
		if dirErr == nil && pathErr == nil {
			rel, err := filepath.Rel(absDir, absPath)
			if err == nil && rel != "." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != ".." {
				base := filepath.Base(absPath)
				switch base {
				case assistantassets.BuiltinAvatarFile:
					return assistant.AvatarSourceBuiltin, assistantassets.BuiltinAvatarAssetID
				case assistantassets.BuiltinMotionFile:
					return assistant.AvatarSourceBuiltin, assistantassets.BuiltinMotionAssetID
				default:
					return assistant.AvatarSourceBuiltin, ""
				}
			}
		}
	}
	return assistant.AvatarSourceUser, ""
}

func isBuiltinAssetPath(path string) bool {
	source, _ := resolveAssetSourceAndID(path)
	return source == assistant.AvatarSourceBuiltin
}
