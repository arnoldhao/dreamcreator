package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	threaddto "dreamcreator/internal/application/thread/dto"
	threadservice "dreamcreator/internal/application/thread/service"
	"dreamcreator/internal/domain/thread"
)

type ThreadAPIHandler struct {
	threads *threadservice.ThreadService
}

func NewThreadAPIHandler(threads *threadservice.ThreadService) *ThreadAPIHandler {
	return &ThreadAPIHandler{threads: threads}
}

func (handler *ThreadAPIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		setCORSHeaders(w, r)
		w.WriteHeader(http.StatusNoContent)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/threads")
	if path == "" || path == "/" {
		handler.handleCollection(w, r)
		return
	}

	segments := strings.Split(strings.Trim(path, "/"), "/")
	if len(segments) == 0 || segments[0] == "" {
		http.NotFound(w, r)
		return
	}
	threadID := segments[0]
	action := ""
	if len(segments) > 1 {
		action = segments[1]
	}

	switch action {
	case "":
		handler.handleThread(w, r, threadID)
	case "rename":
		handler.handleRename(w, r, threadID)
	case "status":
		handler.handleStatus(w, r, threadID)
	case "restore":
		handler.handleRestore(w, r, threadID)
	case "purge":
		handler.handlePurge(w, r, threadID)
	default:
		http.NotFound(w, r)
	}
}

func (handler *ThreadAPIHandler) handleCollection(w http.ResponseWriter, r *http.Request) {
	if handler.threads == nil {
		http.Error(w, "thread service is not configured", http.StatusServiceUnavailable)
		return
	}
	setCORSHeaders(w, r)
	switch r.Method {
	case http.MethodGet:
		includeDeleted := parseBoolQuery(r, "includeDeleted")
		items, err := handler.threads.ListThreads(r.Context(), includeDeleted)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, items)
	case http.MethodPost:
		var request threaddto.NewThreadRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		result, err := handler.threads.NewThread(r.Context(), request)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, result)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (handler *ThreadAPIHandler) handleThread(w http.ResponseWriter, r *http.Request, threadID string) {
	if handler.threads == nil {
		http.Error(w, "thread service is not configured", http.StatusServiceUnavailable)
		return
	}
	setCORSHeaders(w, r)
	switch r.Method {
	case http.MethodDelete:
		if err := handler.threads.SoftDeleteThread(r.Context(), threadID); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	case http.MethodGet:
		items, err := handler.threads.ListThreads(r.Context(), true)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		for _, item := range items {
			if item.ID == threadID {
				writeJSON(w, item)
				return
			}
		}
		http.NotFound(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (handler *ThreadAPIHandler) handleRename(w http.ResponseWriter, r *http.Request, threadID string) {
	if handler.threads == nil {
		http.Error(w, "thread service is not configured", http.StatusServiceUnavailable)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	setCORSHeaders(w, r)
	var payload struct {
		Title string `json:"title"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if err := handler.threads.RenameThread(r.Context(), threaddto.RenameThreadRequest{ThreadID: threadID, Title: payload.Title}); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (handler *ThreadAPIHandler) handleStatus(w http.ResponseWriter, r *http.Request, threadID string) {
	if handler.threads == nil {
		http.Error(w, "thread service is not configured", http.StatusServiceUnavailable)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	setCORSHeaders(w, r)
	var payload struct {
		Status thread.Status `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if err := handler.threads.SetThreadStatus(r.Context(), threaddto.SetThreadStatusRequest{ThreadID: threadID, Status: payload.Status}); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (handler *ThreadAPIHandler) handleRestore(w http.ResponseWriter, r *http.Request, threadID string) {
	if handler.threads == nil {
		http.Error(w, "thread service is not configured", http.StatusServiceUnavailable)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	setCORSHeaders(w, r)
	if err := handler.threads.RestoreThread(r.Context(), threadID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (handler *ThreadAPIHandler) handlePurge(w http.ResponseWriter, r *http.Request, threadID string) {
	if handler.threads == nil {
		http.Error(w, "thread service is not configured", http.StatusServiceUnavailable)
		return
	}
	if r.Method != http.MethodDelete && r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	setCORSHeaders(w, r)
	if err := handler.threads.PurgeThread(r.Context(), threadID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func parseBoolQuery(r *http.Request, key string) bool {
	value := strings.TrimSpace(r.URL.Query().Get(key))
	if value == "" {
		return false
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false
	}
	return parsed
}
