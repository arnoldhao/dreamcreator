package telegram

import (
	"encoding/json"
	"net/http"
	"strings"

	telegramservice "dreamcreator/internal/application/channels/telegram"
	telegramapi "dreamcreator/internal/infrastructure/telegram"
)

type WebhookHandler struct {
	service *telegramservice.BotService
}

func NewWebhookHandler(service *telegramservice.BotService) *WebhookHandler {
	return &WebhookHandler{service: service}
}

func (handler *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if handler == nil || handler.service == nil {
		http.Error(w, "telegram service unavailable", http.StatusServiceUnavailable)
		return
	}
	var update telegramapi.Update
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	accountID := resolveAccountIDFromPath(r.URL.Path)
	secret := strings.TrimSpace(r.Header.Get("X-Telegram-Bot-Api-Secret-Token"))
	if resolved, err := handler.service.ResolveWebhookAccountID(accountID, secret); err == nil {
		accountID = resolved
	} else {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := handler.service.HandleWebhook(r.Context(), accountID, update); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func resolveAccountIDFromPath(path string) string {
	trimmed := strings.TrimSuffix(strings.TrimSpace(path), "/")
	if trimmed == "" {
		return ""
	}
	parts := strings.Split(trimmed, "/")
	if len(parts) == 0 {
		return ""
	}
	last := parts[len(parts)-1]
	if last == "telegram" {
		return ""
	}
	return last
}
