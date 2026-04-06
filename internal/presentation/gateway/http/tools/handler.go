package tools

import (
	"encoding/json"
	"net/http"
	"strings"

	toolgateway "dreamcreator/internal/application/gateway/tools"
	tooldto "dreamcreator/internal/application/tools/dto"
)

type Handler struct {
	service *toolgateway.Service
}

func NewHandler(service *toolgateway.Service) *Handler {
	return &Handler{service: service}
}

func (handler *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
	if handler == nil || handler.service == nil {
		http.Error(w, "tool service unavailable", http.StatusServiceUnavailable)
		return
	}
	var request tooldto.ToolsInvokeRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	request.Tool = strings.TrimSpace(request.Tool)
	if request.Tool == "" {
		http.Error(w, "tool is required", http.StatusBadRequest)
		return
	}
	response, err := handler.service.Invoke(r.Context(), request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
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
