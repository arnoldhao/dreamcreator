package http

import (
	"context"
	"errors"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type MemoryAvatarPathResolver interface {
	ResolveAvatarPath(ctx context.Context, key string) (string, error)
}

type MemoryAvatarHandler struct {
	resolver MemoryAvatarPathResolver
}

func NewMemoryAvatarHandler(resolver MemoryAvatarPathResolver) *MemoryAvatarHandler {
	return &MemoryAvatarHandler{resolver: resolver}
}

func (handler *MemoryAvatarHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		setCORSHeaders(w, r)
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	setCORSHeaders(w, r)
	if handler == nil || handler.resolver == nil {
		http.Error(w, "avatar resolver unavailable", http.StatusServiceUnavailable)
		return
	}
	key := strings.TrimSpace(r.URL.Query().Get("key"))
	if key == "" {
		http.Error(w, "key is required", http.StatusBadRequest)
		return
	}
	avatarPath, err := handler.resolver.ResolveAvatarPath(r.Context(), key)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			http.Error(w, "avatar not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to resolve avatar", http.StatusInternalServerError)
		return
	}
	cleaned := filepath.Clean(strings.TrimSpace(avatarPath))
	if cleaned == "" {
		http.Error(w, "avatar not found", http.StatusNotFound)
		return
	}
	info, err := os.Stat(cleaned)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			http.Error(w, "avatar not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to read avatar", http.StatusInternalServerError)
		return
	}
	if info.IsDir() {
		http.Error(w, "avatar not found", http.StatusNotFound)
		return
	}
	file, err := os.Open(cleaned)
	if err != nil {
		http.Error(w, "failed to open avatar", http.StatusInternalServerError)
		return
	}
	defer file.Close()
	if contentType := mime.TypeByExtension(strings.ToLower(filepath.Ext(cleaned))); contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	w.Header().Set("Accept-Ranges", "bytes")
	http.ServeContent(w, r, info.Name(), info.ModTime(), file)
}
