package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"dreamcreator/internal/application/assistant/dto"
	"dreamcreator/internal/domain/assistant"
)

const (
	autoLocaleEndpointTimeout   = 1200 * time.Millisecond
	autoLocaleCacheSuccessTTL   = 12 * time.Hour
	autoLocaleCacheFailureTTL   = 90 * time.Second
	autoLocaleResponseLimitByte = 64 * 1024
)

var (
	autoLocaleIPWhoEndpoint   = "https://ipwho.is/?fields=success,country,region,city"
	autoLocaleIPAPICoEndpoint = "https://ipapi.co/json/"
)

type autoLocaleCache struct {
	locationValue string
	expiresAt     time.Time
	inFlight      chan struct{}
}

func (service *AssistantService) RefreshAssistantUserLocale(ctx context.Context, assistantID string) (dto.Assistant, error) {
	id := strings.TrimSpace(assistantID)
	if id == "" {
		return dto.Assistant{}, assistant.ErrInvalidAssistantID
	}
	item, err := service.repo.Get(ctx, id)
	if err != nil {
		return dto.Assistant{}, err
	}
	updatedUser, changed := service.populateAutoUserLocale(ctx, item.User)
	if changed {
		item.User = updatedUser
		item.UpdatedAt = service.currentTime()
		if err := service.repo.Save(ctx, item); err != nil {
			return dto.Assistant{}, err
		}
	}
	return toDTO(item), nil
}

func (service *AssistantService) populateAutoUserLocale(ctx context.Context, user assistant.AssistantUser) (assistant.AssistantUser, bool) {
	updated := user
	changed := false

	languageValue := strings.TrimSpace(service.resolveCurrentLanguage(ctx))
	timezoneValue := normalizeSystemTimezone(service.currentTime().Location().String())
	locationValue := ""
	if shouldResolveAutoLocation(updated.Location) {
		locationValue = strings.TrimSpace(service.resolveCurrentLocation(ctx))
	}

	var localeChanged bool
	updated.Language, localeChanged = applyAutoLocaleCurrent(updated.Language, languageValue)
	changed = changed || localeChanged
	updated.Timezone, localeChanged = applyAutoLocaleCurrent(updated.Timezone, timezoneValue)
	changed = changed || localeChanged
	updated.Location, localeChanged = applyAutoLocaleCurrent(updated.Location, locationValue)
	changed = changed || localeChanged

	return updated, changed
}

func (service *AssistantService) resolveCurrentLanguage(ctx context.Context) string {
	if service == nil || service.settings == nil {
		return ""
	}
	current, err := service.settings.GetSettings(ctx)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(current.Language)
}

func (service *AssistantService) resolveCurrentLocation(ctx context.Context) string {
	if service == nil {
		return ""
	}
	if service.resolveLocation != nil {
		return strings.TrimSpace(service.resolveLocation(ctx))
	}

	now := service.currentTime()
	service.cacheMu.Lock()
	if now.Before(service.cache.expiresAt) {
		value := service.cache.locationValue
		service.cacheMu.Unlock()
		return value
	}
	if service.cache.inFlight != nil {
		done := service.cache.inFlight
		service.cacheMu.Unlock()
		select {
		case <-done:
		case <-ctx.Done():
			return ""
		}
		service.cacheMu.Lock()
		value := service.cache.locationValue
		service.cacheMu.Unlock()
		return value
	}
	service.cache.inFlight = make(chan struct{})
	done := service.cache.inFlight
	service.cacheMu.Unlock()

	value := strings.TrimSpace(service.resolveLocationFromIP(ctx))
	ttl := autoLocaleCacheFailureTTL
	if value != "" {
		ttl = autoLocaleCacheSuccessTTL
	}

	service.cacheMu.Lock()
	service.cache.locationValue = value
	service.cache.expiresAt = service.currentTime().Add(ttl)
	if service.cache.inFlight == done {
		close(service.cache.inFlight)
		service.cache.inFlight = nil
	}
	service.cacheMu.Unlock()
	return value
}

func (service *AssistantService) resolveLocationFromIP(ctx context.Context) string {
	if value := strings.TrimSpace(service.resolveFromIPWho(ctx)); value != "" {
		return value
	}
	if value := strings.TrimSpace(service.resolveFromIPAPICo(ctx)); value != "" {
		return value
	}
	return ""
}

func (service *AssistantService) resolveFromIPWho(ctx context.Context) string {
	payload := struct {
		Success bool   `json:"success"`
		City    string `json:"city"`
		Region  string `json:"region"`
		Country string `json:"country"`
	}{}
	if err := service.decodeLocationPayload(ctx, autoLocaleIPWhoEndpoint, &payload); err != nil {
		return ""
	}
	if !payload.Success {
		return ""
	}
	return formatAutoLocation(payload.City, payload.Region, payload.Country)
}

func (service *AssistantService) resolveFromIPAPICo(ctx context.Context) string {
	payload := struct {
		Error       bool   `json:"error"`
		City        string `json:"city"`
		Region      string `json:"region"`
		CountryName string `json:"country_name"`
		Country     string `json:"country"`
	}{}
	if err := service.decodeLocationPayload(ctx, autoLocaleIPAPICoEndpoint, &payload); err != nil {
		return ""
	}
	if payload.Error {
		return ""
	}
	country := payload.CountryName
	if strings.TrimSpace(country) == "" {
		country = payload.Country
	}
	return formatAutoLocation(payload.City, payload.Region, country)
}

func (service *AssistantService) decodeLocationPayload(ctx context.Context, endpoint string, target any) error {
	url := strings.TrimSpace(endpoint)
	if url == "" {
		return errors.New("endpoint is required")
	}
	requestCtx, cancel := context.WithTimeout(ctx, autoLocaleEndpointTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(requestCtx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")

	response, err := service.autoLocaleHTTPClient().Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, autoLocaleResponseLimitByte))
		return errors.New("location endpoint returned non-success status")
	}

	decoder := json.NewDecoder(io.LimitReader(response.Body, autoLocaleResponseLimitByte))
	return decoder.Decode(target)
}

func (service *AssistantService) autoLocaleHTTPClient() *http.Client {
	if service != nil && service.httpClient != nil {
		return service.httpClient
	}
	return &http.Client{Timeout: 3 * time.Second}
}

func (service *AssistantService) currentTime() time.Time {
	if service != nil && service.now != nil {
		return service.now()
	}
	return time.Now()
}

func shouldResolveAutoLocation(locale assistant.UserLocale) bool {
	mode := strings.ToLower(strings.TrimSpace(locale.Mode))
	if mode == "manual" {
		return false
	}
	return localeCurrentMissing(locale.Current)
}

func localeCurrentMissing(value string) bool {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return true
	}
	return strings.EqualFold(trimmed, "unknown")
}

func applyAutoLocaleCurrent(locale assistant.UserLocale, autoValue string) (assistant.UserLocale, bool) {
	mode := strings.ToLower(strings.TrimSpace(locale.Mode))
	if mode == "manual" {
		return locale, false
	}
	if mode != "auto" {
		mode = "auto"
	}
	changed := false
	if locale.Mode != mode {
		locale.Mode = mode
		changed = true
	}
	trimmedCurrent := strings.TrimSpace(locale.Current)
	if locale.Current != trimmedCurrent {
		locale.Current = trimmedCurrent
		changed = true
	}
	trimmedAuto := strings.TrimSpace(autoValue)
	if trimmedAuto == "" || trimmedCurrent == trimmedAuto {
		return locale, changed
	}
	locale.Current = trimmedAuto
	return locale, true
}

func normalizeSystemTimezone(value string) string {
	trimmed := strings.TrimSpace(value)
	if strings.EqualFold(trimmed, "local") {
		return ""
	}
	return trimmed
}

func formatAutoLocation(city string, region string, country string) string {
	parts := []string{
		normalizeAutoLocationPart(city),
		normalizeAutoLocationPart(region),
		normalizeAutoLocationPart(country),
	}
	seen := make(map[string]struct{}, len(parts))
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		key := strings.ToLower(part)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, part)
	}
	return strings.Join(result, ", ")
}

func normalizeAutoLocationPart(value string) string {
	trimmed := strings.TrimSpace(value)
	switch strings.ToLower(trimmed) {
	case "", "null", "unknown", "undefined":
		return ""
	default:
		return trimmed
	}
}
