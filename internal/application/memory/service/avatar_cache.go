package service

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	memoryAvatarDownloadTimeout = 8 * time.Second
	memoryAvatarMaxBytes        = 8 * 1024 * 1024
	memoryAvatarKeyVersion      = "v1"
)

var memoryAvatarSegmentPattern = regexp.MustCompile(`[^a-z0-9_-]+`)

type memoryAvatarCache struct {
	httpClient *http.Client
	now        func() time.Time
}

type memoryAvatarCacheRequest struct {
	Channel       string
	AccountID     string
	PrincipalType string
	PrincipalID   string
	SourceURL     string
	AvatarKey     string
	LocalPath     string
	AsyncDownload bool
}

type memoryAvatarCacheResult struct {
	LocalPath string
	SourceURL string
	AvatarKey string
	Pending   bool
}

type memoryAvatarMaterialized struct {
	DisplayURL    string
	SourceURL     string
	AvatarKey     string
	Pending       bool
	ShouldPersist bool
}

type memoryAvatarCacheMeta struct {
	Key           string `json:"key"`
	Channel       string `json:"channel,omitempty"`
	AccountID     string `json:"accountId,omitempty"`
	PrincipalType string `json:"principalType,omitempty"`
	PrincipalID   string `json:"principalId,omitempty"`
	SourceURL     string `json:"sourceUrl,omitempty"`
	LocalPath     string `json:"localPath,omitempty"`
	UpdatedAt     string `json:"updatedAt,omitempty"`
}

type memoryAvatarKeyParts struct {
	Channel       string
	AccountID     string
	PrincipalType string
	PrincipalHash string
}

func newMemoryAvatarCache(client *http.Client, now func() time.Time) *memoryAvatarCache {
	if client == nil {
		client = &http.Client{Timeout: defaultEmbeddingTimeout}
	}
	if now == nil {
		now = time.Now
	}
	return &memoryAvatarCache{
		httpClient: client,
		now:        now,
	}
}

func (service *MemoryService) materializePrincipalAvatar(
	ctx context.Context,
	channel string,
	accountID string,
	principalType string,
	principalID string,
	avatarURL string,
	avatarSourceURL string,
	avatarKey string,
) memoryAvatarMaterialized {
	return service.materializePrincipalAvatarWithMode(
		ctx,
		channel,
		accountID,
		principalType,
		principalID,
		avatarURL,
		avatarSourceURL,
		avatarKey,
		false,
	)
}

func (service *MemoryService) materializePrincipalAvatarWithMode(
	ctx context.Context,
	channel string,
	accountID string,
	principalType string,
	principalID string,
	avatarURL string,
	avatarSourceURL string,
	avatarKey string,
	asyncDownload bool,
) memoryAvatarMaterialized {
	displayURL := strings.TrimSpace(avatarURL)
	sourceURL := strings.TrimSpace(avatarSourceURL)
	localPath := ""
	if !isLikelyRemoteURL(displayURL) {
		localPath = displayURL
	}
	if sourceURL == "" && isLikelyRemoteURL(displayURL) {
		sourceURL = displayURL
	}
	result := memoryAvatarMaterialized{
		DisplayURL: displayURL,
		SourceURL:  sourceURL,
		AvatarKey:  strings.TrimSpace(avatarKey),
	}
	if sourceURL == "" {
		return result
	}
	if service == nil || service.avatarCache == nil {
		return result
	}
	cacheResult, _ := service.avatarCache.materialize(ctx, memoryAvatarCacheRequest{
		Channel:       channel,
		AccountID:     accountID,
		PrincipalType: principalType,
		PrincipalID:   principalID,
		SourceURL:     sourceURL,
		AvatarKey:     avatarKey,
		LocalPath:     localPath,
		AsyncDownload: asyncDownload,
	})
	if strings.TrimSpace(cacheResult.LocalPath) != "" {
		result.DisplayURL = strings.TrimSpace(cacheResult.LocalPath)
	}
	if strings.TrimSpace(cacheResult.SourceURL) != "" {
		result.SourceURL = strings.TrimSpace(cacheResult.SourceURL)
	}
	if strings.TrimSpace(cacheResult.AvatarKey) != "" {
		result.AvatarKey = strings.TrimSpace(cacheResult.AvatarKey)
	}
	result.Pending = cacheResult.Pending
	if result.DisplayURL == "" {
		result.DisplayURL = displayURL
	}
	// 只要能得到可持久化 metadata，就回写到会话来源信息；下载失败不阻塞主流程。
	result.ShouldPersist = result.DisplayURL != strings.TrimSpace(avatarURL) ||
		result.SourceURL != strings.TrimSpace(avatarSourceURL) ||
		result.AvatarKey != strings.TrimSpace(avatarKey)
	return result
}

func (cache *memoryAvatarCache) materialize(ctx context.Context, request memoryAvatarCacheRequest) (memoryAvatarCacheResult, error) {
	sourceURL := strings.TrimSpace(request.SourceURL)
	if sourceURL == "" {
		return memoryAvatarCacheResult{}, nil
	}
	key := buildMemoryAvatarKey(
		request.Channel,
		request.AccountID,
		request.PrincipalType,
		request.PrincipalID,
		sourceURL,
	)
	if key == "" {
		return memoryAvatarCacheResult{}, errors.New("avatar key is required")
	}
	metaPath, defaultLocalPath, err := cache.resolvePaths(key, sourceURL)
	if err != nil {
		return memoryAvatarCacheResult{}, err
	}
	localPath := defaultLocalPath
	result := memoryAvatarCacheResult{
		LocalPath: localPath,
		SourceURL: sourceURL,
		AvatarKey: key,
	}
	meta := memoryAvatarCacheMeta{
		Key:           key,
		Channel:       strings.TrimSpace(request.Channel),
		AccountID:     strings.TrimSpace(request.AccountID),
		PrincipalType: strings.TrimSpace(request.PrincipalType),
		PrincipalID:   strings.TrimSpace(request.PrincipalID),
		SourceURL:     sourceURL,
		LocalPath:     localPath,
		UpdatedAt:     cache.now().UTC().Format(time.RFC3339),
	}
	if err := cache.writeMeta(metaPath, meta); err != nil {
		return result, err
	}
	if localPath != "" && fileExists(localPath) {
		return result, nil
	}
	if localPath == "" {
		return result, os.ErrNotExist
	}
	if request.AsyncDownload {
		result.Pending = true
		return result, nil
	}
	if err := cache.downloadToPath(ctx, sourceURL, localPath); err != nil {
		return result, err
	}
	return result, nil
}

func (cache *memoryAvatarCache) resolvePathByKey(ctx context.Context, key string) (string, error) {
	_ = ctx
	trimmedKey := strings.TrimSpace(key)
	if trimmedKey == "" {
		return "", os.ErrNotExist
	}
	parts, err := parseMemoryAvatarKey(trimmedKey)
	if err != nil {
		return "", os.ErrNotExist
	}
	dirPath, err := memoryAvatarDir(parts)
	if err != nil {
		return "", err
	}
	primaryMetaPath := filepath.Join(dirPath, trimmedKey+".json")
	meta, metaPath, err := cache.readMetaForKey(primaryMetaPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", os.ErrNotExist
		}
		return "", err
	}
	sourceURL := strings.TrimSpace(meta.SourceURL)
	localPath := ""
	if sourceURL != "" {
		_, inferredPath, inferErr := cache.resolvePaths(trimmedKey, sourceURL)
		if inferErr == nil {
			localPath = inferredPath
		}
	}
	if localPath == "" {
		metaLocalPath := strings.TrimSpace(meta.LocalPath)
		if isMemoryAvatarPrimaryPath(metaLocalPath) {
			localPath = metaLocalPath
		}
	}
	if localPath != "" && fileExists(localPath) {
		if metaPath != primaryMetaPath || strings.TrimSpace(meta.LocalPath) != localPath {
			meta.LocalPath = localPath
			meta.UpdatedAt = cache.now().UTC().Format(time.RFC3339)
			_ = cache.writeMeta(primaryMetaPath, meta)
		}
		return localPath, nil
	}
	return "", os.ErrNotExist
}

func (cache *memoryAvatarCache) readMetaForKey(primaryMetaPath string) (memoryAvatarCacheMeta, string, error) {
	meta, err := cache.readMeta(primaryMetaPath)
	if err == nil {
		return meta, primaryMetaPath, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return memoryAvatarCacheMeta{}, "", err
	}
	return memoryAvatarCacheMeta{}, "", os.ErrNotExist
}

func (cache *memoryAvatarCache) resolvePaths(key string, sourceURL string) (string, string, error) {
	parts, err := parseMemoryAvatarKey(key)
	if err != nil {
		return "", "", err
	}
	dirPath, err := memoryAvatarDir(parts)
	if err != nil {
		return "", "", err
	}
	ext := resolveMemoryAvatarExt(sourceURL)
	localPath := filepath.Join(dirPath, key+ext)
	metaPath := filepath.Join(dirPath, key+".json")
	return metaPath, localPath, nil
}

func (cache *memoryAvatarCache) readMeta(metaPath string) (memoryAvatarCacheMeta, error) {
	payload, err := os.ReadFile(metaPath)
	if err != nil {
		return memoryAvatarCacheMeta{}, err
	}
	meta := memoryAvatarCacheMeta{}
	if err := json.Unmarshal(payload, &meta); err != nil {
		return memoryAvatarCacheMeta{}, err
	}
	return meta, nil
}

func (cache *memoryAvatarCache) writeMeta(metaPath string, meta memoryAvatarCacheMeta) error {
	if strings.TrimSpace(metaPath) == "" {
		return errors.New("meta path is required")
	}
	if err := os.MkdirAll(filepath.Dir(metaPath), 0o755); err != nil {
		return err
	}
	payload, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	return os.WriteFile(metaPath, payload, 0o644)
}

func (cache *memoryAvatarCache) downloadToPath(ctx context.Context, sourceURL string, targetPath string) error {
	if cache == nil || cache.httpClient == nil {
		return errors.New("avatar cache is unavailable")
	}
	sourceURL = strings.TrimSpace(sourceURL)
	targetPath = strings.TrimSpace(targetPath)
	if sourceURL == "" || targetPath == "" {
		return errors.New("avatar source and target path are required")
	}
	parsed, err := url.Parse(sourceURL)
	if err != nil {
		return err
	}
	scheme := strings.ToLower(strings.TrimSpace(parsed.Scheme))
	if scheme != "http" && scheme != "https" {
		return errors.New("unsupported avatar source")
	}
	lookupCtx := ctx
	if lookupCtx == nil {
		lookupCtx = context.Background()
	}
	timeoutCtx, cancel := context.WithTimeout(lookupCtx, memoryAvatarDownloadTimeout)
	defer cancel()
	request, err := http.NewRequestWithContext(timeoutCtx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return err
	}
	response, err := cache.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("avatar fetch status %d", response.StatusCode)
	}
	if contentType := strings.TrimSpace(response.Header.Get("Content-Type")); contentType != "" {
		mediaType, _, parseErr := mime.ParseMediaType(contentType)
		normalizedMediaType := strings.ToLower(strings.TrimSpace(mediaType))
		if parseErr == nil && normalizedMediaType != "" &&
			!strings.HasPrefix(normalizedMediaType, "image/") &&
			normalizedMediaType != "application/octet-stream" {
			return errors.New("avatar response is not an image")
		}
	}
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return err
	}
	tempPath := fmt.Sprintf("%s.tmp-%d", targetPath, time.Now().UnixNano())
	file, err := os.OpenFile(tempPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer func() {
		_ = file.Close()
		_ = os.Remove(tempPath)
	}()
	reader := &io.LimitedReader{R: response.Body, N: memoryAvatarMaxBytes + 1}
	written, err := io.Copy(file, reader)
	if err != nil {
		return err
	}
	if written <= 0 {
		return errors.New("avatar response is empty")
	}
	if reader.N <= 0 {
		return errors.New("avatar file is too large")
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(tempPath, targetPath)
}

func buildMemoryAvatarKey(channel string, accountID string, principalType string, principalID string, sourceURL string) string {
	principal := strings.TrimSpace(principalID)
	source := strings.TrimSpace(sourceURL)
	if principal == "" || source == "" {
		return ""
	}
	channelSegment := sanitizeMemoryAvatarSegment(channel, "unknown")
	accountSegment := sanitizeMemoryAvatarSegment(accountID, "default")
	principalTypeSegment := sanitizeMemoryAvatarSegment(principalType, "principal")
	principalHash := shortHash(principal)
	sourceHash := shortHash(source)
	if principalHash == "" || sourceHash == "" {
		return ""
	}
	return strings.Join([]string{
		memoryAvatarKeyVersion,
		channelSegment,
		accountSegment,
		principalTypeSegment,
		principalHash,
		sourceHash,
	}, "~")
}

func parseMemoryAvatarKey(key string) (memoryAvatarKeyParts, error) {
	parts := strings.Split(strings.TrimSpace(key), "~")
	if len(parts) != 6 || parts[0] != memoryAvatarKeyVersion {
		return memoryAvatarKeyParts{}, errors.New("invalid avatar key")
	}
	if parts[1] == "" || parts[2] == "" || parts[3] == "" || parts[4] == "" {
		return memoryAvatarKeyParts{}, errors.New("invalid avatar key")
	}
	return memoryAvatarKeyParts{
		Channel:       parts[1],
		AccountID:     parts[2],
		PrincipalType: parts[3],
		PrincipalHash: parts[4],
	}, nil
}

func memoryAvatarDir(parts memoryAvatarKeyParts) (string, error) {
	baseDir, err := memoryAvatarBaseDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(
		baseDir,
		parts.Channel,
	), nil
}

func memoryAvatarBaseDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil || strings.TrimSpace(configDir) == "" {
		cacheDir, cacheErr := os.UserCacheDir()
		if cacheErr != nil || strings.TrimSpace(cacheDir) == "" {
			if err != nil {
				return "", err
			}
			if cacheErr == nil {
				return "", os.ErrNotExist
			}
			return "", cacheErr
		}
		configDir = cacheDir
	}
	return filepath.Join(configDir, "dreamcreator", "objects", "avatar"), nil
}

func resolveMemoryAvatarExt(sourceURL string) string {
	parsed, err := url.Parse(strings.TrimSpace(sourceURL))
	if err == nil {
		ext := strings.ToLower(strings.TrimSpace(path.Ext(parsed.Path)))
		switch ext {
		case ".jpg", ".jpeg", ".png", ".webp", ".gif", ".bmp", ".avif":
			return ext
		}
	}
	return ".img"
}

func shortHash(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	sum := sha1.Sum([]byte(trimmed))
	encoded := hex.EncodeToString(sum[:])
	if len(encoded) >= 12 {
		return encoded[:12]
	}
	return encoded
}

func sanitizeMemoryAvatarSegment(value string, fallback string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = memoryAvatarSegmentPattern.ReplaceAllString(normalized, "-")
	normalized = strings.Trim(normalized, "-_")
	if normalized == "" {
		return fallback
	}
	if len(normalized) > 48 {
		return normalized[:48]
	}
	return normalized
}

func isLikelyRemoteURL(value string) bool {
	normalized := strings.ToLower(strings.TrimSpace(value))
	return strings.HasPrefix(normalized, "http://") || strings.HasPrefix(normalized, "https://")
}

func fileExists(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func isMemoryAvatarPrimaryPath(value string) bool {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return false
	}
	baseDir, err := memoryAvatarBaseDir()
	if err != nil || strings.TrimSpace(baseDir) == "" {
		return false
	}
	relative, relErr := filepath.Rel(baseDir, trimmed)
	if relErr != nil {
		return false
	}
	return relative == "." || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)))
}
