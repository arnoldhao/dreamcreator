package http

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	defaultPreviewASSTTL        = 30 * time.Minute
	defaultPreviewASSMaxEntries = 64
)

type LibraryPreviewASSHandler struct {
	mu         sync.RWMutex
	entries    map[string]previewASSEntry
	ttl        time.Duration
	maxEntries int
	now        func() time.Time
}

type previewASSEntry struct {
	Content   string
	UpdatedAt time.Time
}

type previewASSUpsertRequest struct {
	Content string `json:"content"`
}

type previewASSUpsertResponse struct {
	ID string `json:"id"`
}

func NewLibraryPreviewASSHandler() *LibraryPreviewASSHandler {
	return &LibraryPreviewASSHandler{
		entries:    make(map[string]previewASSEntry),
		ttl:        defaultPreviewASSTTL,
		maxEntries: defaultPreviewASSMaxEntries,
		now:        time.Now,
	}
}

func (handler *LibraryPreviewASSHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		setCORSHeaders(w, r)
		w.WriteHeader(http.StatusNoContent)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/library/preview-ass")
	if path == "" || path == "/" {
		handler.handleCollection(w, r)
		return
	}

	segments := strings.Split(strings.Trim(path, "/"), "/")
	if len(segments) == 0 || strings.TrimSpace(segments[0]) == "" {
		http.NotFound(w, r)
		return
	}

	id := strings.TrimSpace(segments[0])
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		handler.handleItemRead(w, r, id)
	case http.MethodDelete:
		handler.handleItemDelete(w, r, id)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (handler *LibraryPreviewASSHandler) handleCollection(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w, r)
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload previewASSUpsertRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<20)).Decode(&payload); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	content := strings.TrimSpace(payload.Content)
	if content == "" {
		http.Error(w, "content is required", http.StatusBadRequest)
		return
	}

	id := handler.put(content)
	writeJSON(w, previewASSUpsertResponse{ID: id})
}

func (handler *LibraryPreviewASSHandler) handleItemRead(w http.ResponseWriter, r *http.Request, id string) {
	setCORSHeaders(w, r)
	content, ok := handler.get(id)
	if !ok {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store, max-age=0")
	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}
	_, _ = w.Write([]byte(content))
}

func (handler *LibraryPreviewASSHandler) handleItemDelete(w http.ResponseWriter, r *http.Request, id string) {
	setCORSHeaders(w, r)
	handler.delete(id)
	w.WriteHeader(http.StatusNoContent)
}

func (handler *LibraryPreviewASSHandler) put(content string) string {
	if handler == nil {
		return ""
	}
	now := handler.resolveNow()
	handler.mu.Lock()
	defer handler.mu.Unlock()
	handler.pruneLocked(now)
	id := uuid.NewString()
	handler.entries[id] = previewASSEntry{
		Content:   content,
		UpdatedAt: now,
	}
	return id
}

func (handler *LibraryPreviewASSHandler) get(id string) (string, bool) {
	if handler == nil || strings.TrimSpace(id) == "" {
		return "", false
	}
	now := handler.resolveNow()
	handler.mu.Lock()
	defer handler.mu.Unlock()
	handler.pruneLocked(now)
	entry, ok := handler.entries[id]
	if !ok {
		return "", false
	}
	entry.UpdatedAt = now
	handler.entries[id] = entry
	return entry.Content, true
}

func (handler *LibraryPreviewASSHandler) delete(id string) {
	if handler == nil || strings.TrimSpace(id) == "" {
		return
	}
	handler.mu.Lock()
	defer handler.mu.Unlock()
	delete(handler.entries, id)
}

func (handler *LibraryPreviewASSHandler) resolveNow() time.Time {
	if handler != nil && handler.now != nil {
		return handler.now()
	}
	return time.Now()
}

func (handler *LibraryPreviewASSHandler) pruneLocked(now time.Time) {
	if handler == nil {
		return
	}
	if handler.ttl > 0 {
		for id, entry := range handler.entries {
			if now.Sub(entry.UpdatedAt) > handler.ttl {
				delete(handler.entries, id)
			}
		}
	}
	if handler.maxEntries <= 0 || len(handler.entries) <= handler.maxEntries {
		return
	}
	for len(handler.entries) > handler.maxEntries {
		oldestID := ""
		var oldestTime time.Time
		for id, entry := range handler.entries {
			if oldestID == "" || entry.UpdatedAt.Before(oldestTime) {
				oldestID = id
				oldestTime = entry.UpdatedAt
			}
		}
		if oldestID == "" {
			return
		}
		delete(handler.entries, oldestID)
	}
}
