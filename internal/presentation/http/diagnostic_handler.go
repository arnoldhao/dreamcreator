package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"dreamcreator/internal/application/gateway/observability"
)

func NewHealthHandler(obs *observability.Service) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			setCORSHeaders(w, r)
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		setCORSHeaders(w, r)
		if obs == nil {
			writeJSON(w, map[string]any{
				"status": "ok",
				"time":   time.Now().Format(time.RFC3339),
			})
			return
		}
		snapshot := obs.Health(r.Context())
		writeJSON(w, map[string]any{
			"status":   "ok",
			"snapshot": snapshot,
		})
	})
}

func NewStatusHandler(obs *observability.Service) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			setCORSHeaders(w, r)
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if obs == nil {
			http.Error(w, "observability is not configured", http.StatusServiceUnavailable)
			return
		}
		setCORSHeaders(w, r)
		report := obs.Status(r.Context())
		writeJSON(w, map[string]any{
			"status": "ok",
			"report": report,
		})
	})
}

func NewLogsTailHandler(obs *observability.Service) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			setCORSHeaders(w, r)
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if obs == nil {
			http.Error(w, "observability is not configured", http.StatusServiceUnavailable)
			return
		}
		setCORSHeaders(w, r)
		query := r.URL.Query()
		limit := 0
		if raw := query.Get("limit"); raw != "" {
			if parsed, err := strconv.Atoi(raw); err == nil {
				limit = parsed
			}
		}
		resp, err := obs.TailLogs(r.Context(), observability.LogsTailRequest{
			Level:     query.Get("level"),
			Component: query.Get("component"),
			From:      query.Get("from"),
			Limit:     limit,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, map[string]any{
			"status": "ok",
			"logs":   resp,
		})
	})
}

func writeJSON(w http.ResponseWriter, payload any) {
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	_ = encoder.Encode(payload)
}
