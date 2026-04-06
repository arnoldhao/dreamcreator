package providersync

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const modelsDevLogoBaseURL = "https://models.dev/logos"

const defaultLogoSVG = `<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 40 40" fill="none">
  <rect width="40" height="40" rx="8" fill="#E5E7EB"/>
  <circle cx="20" cy="20" r="10" fill="#9CA3AF"/>
</svg>`

type ModelsDevLogoCache struct {
	baseDir        string
	httpClient     *http.Client
	defaultDataURI string
}

func NewModelsDevLogoCache() *ModelsDevLogoCache {
	baseDir := ""
	if cacheDir, err := os.UserCacheDir(); err == nil && cacheDir != "" {
		baseDir = filepath.Join(cacheDir, "dreamcreator", "modelsdev", "logos")
	}
	return &ModelsDevLogoCache{
		baseDir:        baseDir,
		httpClient:     &http.Client{Timeout: 10 * time.Second},
		defaultDataURI: svgToDataURI([]byte(defaultLogoSVG)),
	}
}

func (cache *ModelsDevLogoCache) ResolveProviderLogo(ctx context.Context, providerID string) (string, error) {
	if cache == nil {
		return "", nil
	}
	trimmed := strings.ToLower(strings.TrimSpace(providerID))
	if trimmed == "" {
		return cache.defaultDataURI, nil
	}

	if cache.baseDir != "" {
		path := cache.logoPath(trimmed)
		if data, err := os.ReadFile(path); err == nil {
			return svgToDataURI(data), nil
		}
	}

	data, err := cache.fetchLogo(ctx, trimmed)
	if err != nil {
		return cache.defaultDataURI, nil
	}

	if cache.baseDir != "" {
		if err := os.MkdirAll(cache.baseDir, 0o755); err == nil {
			_ = os.WriteFile(cache.logoPath(trimmed), data, 0o644)
		}
	}
	return svgToDataURI(data), nil
}

func (cache *ModelsDevLogoCache) fetchLogo(ctx context.Context, providerID string) ([]byte, error) {
	requestURL := fmt.Sprintf("%s/%s.svg", modelsDevLogoBaseURL, url.PathEscape(providerID))
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, err
	}
	client := cache.httpClient
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusBadRequest {
		body, _ := io.ReadAll(response.Body)
		return nil, fmt.Errorf("models.dev logo request failed: %s", strings.TrimSpace(string(body)))
	}

	return io.ReadAll(response.Body)
}

func (cache *ModelsDevLogoCache) logoPath(providerID string) string {
	return filepath.Join(cache.baseDir, fmt.Sprintf("%s.svg", sanitizeLogoKey(providerID)))
}

func sanitizeLogoKey(value string) string {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	if trimmed == "" {
		return "default"
	}
	var builder strings.Builder
	for _, runeValue := range trimmed {
		if (runeValue >= 'a' && runeValue <= 'z') || (runeValue >= '0' && runeValue <= '9') || runeValue == '-' || runeValue == '_' {
			builder.WriteRune(runeValue)
		} else {
			builder.WriteByte('_')
		}
	}
	return builder.String()
}

func svgToDataURI(svg []byte) string {
	if len(svg) == 0 {
		return ""
	}
	encoded := base64.StdEncoding.EncodeToString(svg)
	return "data:image/svg+xml;base64," + encoded
}
