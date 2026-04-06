package service

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestMemoryAvatarCache_DownloadAllowsOctetStream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write([]byte{0xff, 0xd8, 0xff, 0xe0, 0x00, 0x10, 'J', 'F', 'I', 'F', 0x00})
	}))
	defer server.Close()

	now := time.Now().UTC()
	cache := newMemoryAvatarCache(server.Client(), func() time.Time { return now })
	targetPath := filepath.Join(t.TempDir(), "avatar.jpg")
	if err := cache.downloadToPath(context.Background(), server.URL+"/avatar.jpg", targetPath); err != nil {
		t.Fatalf("download avatar: %v", err)
	}
	info, err := os.Stat(targetPath)
	if err != nil {
		t.Fatalf("stat avatar: %v", err)
	}
	if info.Size() <= 0 {
		t.Fatalf("expected non-empty avatar file")
	}
}

func TestMemoryAvatarCache_MaterializeAsyncSkipsDownload(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", "")
	var requestCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write([]byte{0xff, 0xd8, 0xff, 0xd9})
	}))
	defer server.Close()

	now := time.Now().UTC()
	cache := newMemoryAvatarCache(server.Client(), func() time.Time { return now })
	result, err := cache.materialize(context.Background(), memoryAvatarCacheRequest{
		Channel:       "telegram",
		AccountID:     "main",
		PrincipalType: "user",
		PrincipalID:   "u-1",
		SourceURL:     server.URL + "/avatar.jpg",
		AsyncDownload: true,
	})
	if err != nil {
		t.Fatalf("materialize async: %v", err)
	}
	if !result.Pending {
		t.Fatalf("expected pending result for async materialize")
	}
	if got := atomic.LoadInt32(&requestCount); got != 0 {
		t.Fatalf("expected no download during async materialize, got %d requests", got)
	}
	if _, statErr := os.Stat(result.LocalPath); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("expected no avatar file yet, stat error: %v", statErr)
	}
}

func TestMemoryAvatarCache_UsesFlattenedChannelPath(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", "")
	cache := newMemoryAvatarCache(http.DefaultClient, time.Now)
	key := buildMemoryAvatarKey("telegram", "main", "user", "u-1", "https://example.com/avatar.jpg")
	if strings.TrimSpace(key) == "" {
		t.Fatalf("expected avatar key")
	}
	metaPath, localPath, err := cache.resolvePaths(key, "https://example.com/avatar.jpg")
	if err != nil {
		t.Fatalf("resolve paths: %v", err)
	}
	baseDir, err := memoryAvatarBaseDir()
	if err != nil {
		t.Fatalf("resolve base dir: %v", err)
	}
	relativeMetaPath, err := filepath.Rel(baseDir, metaPath)
	if err != nil {
		t.Fatalf("resolve relative meta path: %v", err)
	}
	segments := strings.Split(filepath.ToSlash(relativeMetaPath), "/")
	if len(segments) != 2 {
		t.Fatalf("expected flattened channel path with 2 segments, got %v", segments)
	}
	if segments[0] != "telegram" {
		t.Fatalf("expected channel segment telegram, got %s", segments[0])
	}
	if !strings.HasSuffix(metaPath, ".json") {
		t.Fatalf("expected json meta path, got %s", metaPath)
	}
	if !strings.Contains(filepath.ToSlash(localPath), "/telegram/") {
		t.Fatalf("expected local path to include channel folder, got %s", localPath)
	}
}
