package web

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"dreamcreator/internal/application/gateway/channels"
	settingsservice "dreamcreator/internal/application/settings/service"
)

type WebhookHandler struct {
	registry *channels.Registry
	settings *settingsservice.SettingsService
}

func NewWebhookHandler(registry *channels.Registry, settings *settingsservice.SettingsService) *WebhookHandler {
	return &WebhookHandler{registry: registry, settings: settings}
}

func (handler *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		setCORSHeaders(w, r)
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	setCORSHeaders(w, r)
	if handler == nil || handler.registry == nil {
		http.Error(w, "registry unavailable", http.StatusServiceUnavailable)
		return
	}
	if handler.settings == nil {
		http.Error(w, "settings unavailable", http.StatusServiceUnavailable)
		return
	}
	if ok, status, errMsg := handler.authorize(r); !ok {
		http.Error(w, errMsg, status)
		return
	}
	var request channels.WebhookIngestRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(request.ChannelID) == "" {
		request.ChannelID = "web"
	}
	if strings.TrimSpace(request.EventID) == "" {
		request.EventID = time.Now().Format(time.RFC3339Nano)
	}
	result, err := handler.registry.IngestWebhook(r.Context(), request.ChannelID, request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

func (handler *WebhookHandler) authorize(r *http.Request) (bool, int, string) {
	settings, err := handler.settings.GetSettings(r.Context())
	if err != nil {
		return false, http.StatusServiceUnavailable, "settings unavailable"
	}
	config := resolveMap(settings.Channels["web"])
	enabled := resolveBool(config["enabled"], true)
	if !enabled {
		return false, http.StatusForbidden, "web channel disabled"
	}
	token := strings.TrimSpace(resolveString(config["bearerToken"]))
	if token == "" {
		return false, http.StatusUnauthorized, "web channel not configured"
	}
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	if authHeader == "" {
		return false, http.StatusUnauthorized, "authorization required"
	}
	lower := strings.ToLower(authHeader)
	if !strings.HasPrefix(lower, "bearer ") {
		return false, http.StatusUnauthorized, "invalid authorization"
	}
	value := strings.TrimSpace(authHeader[len("Bearer "):])
	if value != token {
		return false, http.StatusUnauthorized, "invalid token"
	}
	return true, http.StatusOK, ""
}

func resolveMap(value any) map[string]any {
	if typed, ok := value.(map[string]any); ok {
		return typed
	}
	return map[string]any{}
}

func resolveString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return ""
	}
}

func resolveBool(value any, fallback bool) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		trimmed := strings.ToLower(strings.TrimSpace(typed))
		if trimmed == "true" || trimmed == "1" || trimmed == "yes" || trimmed == "on" {
			return true
		}
		if trimmed == "false" || trimmed == "0" || trimmed == "no" || trimmed == "off" {
			return false
		}
	}
	return fallback
}

func setCORSHeaders(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	allowHeaders := strings.TrimSpace(r.Header.Get("Access-Control-Request-Headers"))
	if allowHeaders == "" {
		allowHeaders = "Content-Type, Authorization, User-Agent"
	}
	w.Header().Set("Access-Control-Allow-Headers", allowHeaders)
}
