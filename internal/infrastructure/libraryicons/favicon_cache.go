package libraryicons

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
	"sync"
	"time"
)

const faviconBaseURL = "https://www.google.com/s2/favicons"

type FaviconCache struct {
	baseDir    string
	httpClient *http.Client
	mu         sync.Mutex
	memory     map[string]string
	missing    map[string]struct{}
}

func NewFaviconCache() *FaviconCache {
	baseDir := ""
	if cacheDir, err := os.UserCacheDir(); err == nil && cacheDir != "" {
		baseDir = filepath.Join(cacheDir, "dreamcreator", "library", "favicons")
	}
	return &FaviconCache{
		baseDir:    baseDir,
		httpClient: &http.Client{Timeout: 2 * time.Second},
		memory:     map[string]string{},
		missing:    map[string]struct{}{},
	}
}

func (cache *FaviconCache) ResolveDomainIcon(ctx context.Context, domain string) (string, error) {
	if cache == nil {
		return "", nil
	}
	normalized := normalizeDomain(domain)
	if normalized == "" {
		return "", nil
	}

	cache.mu.Lock()
	if icon, ok := cache.memory[normalized]; ok {
		cache.mu.Unlock()
		return icon, nil
	}
	if _, ok := cache.missing[normalized]; ok {
		cache.mu.Unlock()
		return "", nil
	}
	cache.mu.Unlock()

	if cache.baseDir != "" {
		path := cache.iconPath(normalized)
		if data, err := os.ReadFile(path); err == nil {
			icon := dataToDataURI(data)
			cache.storeIcon(normalized, icon)
			return icon, nil
		}
	}

	data, err := cache.fetchFavicon(ctx, normalized)
	if err != nil {
		cache.markMissing(normalized)
		return "", err
	}

	if cache.baseDir != "" {
		if err := os.MkdirAll(cache.baseDir, 0o755); err == nil {
			_ = os.WriteFile(cache.iconPath(normalized), data, 0o644)
		}
	}
	icon := dataToDataURI(data)
	cache.storeIcon(normalized, icon)
	return icon, nil
}

func (cache *FaviconCache) ResolveDomainIconCached(ctx context.Context, domain string) (string, bool) {
	if cache == nil {
		return "", false
	}
	normalized := normalizeDomain(domain)
	if normalized == "" {
		return "", false
	}

	cache.mu.Lock()
	if icon, ok := cache.memory[normalized]; ok {
		cache.mu.Unlock()
		return icon, true
	}
	if _, ok := cache.missing[normalized]; ok {
		cache.mu.Unlock()
		return "", false
	}
	cache.mu.Unlock()

	if cache.baseDir != "" {
		path := cache.iconPath(normalized)
		if data, err := os.ReadFile(path); err == nil {
			icon := dataToDataURI(data)
			cache.storeIcon(normalized, icon)
			return icon, true
		}
	}

	return "", false
}

func (cache *FaviconCache) fetchFavicon(ctx context.Context, domain string) ([]byte, error) {
	requestURL := fmt.Sprintf("%s?domain=%s&sz=64", faviconBaseURL, url.QueryEscape(domain))
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, err
	}
	client := cache.httpClient
	if client == nil {
		client = &http.Client{Timeout: 2 * time.Second}
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusBadRequest {
		body, _ := io.ReadAll(response.Body)
		return nil, fmt.Errorf("favicon request failed: %s", strings.TrimSpace(string(body)))
	}
	return io.ReadAll(response.Body)
}

func (cache *FaviconCache) iconPath(domain string) string {
	return filepath.Join(cache.baseDir, fmt.Sprintf("%s.png", sanitizeDomainKey(domain)))
}

func (cache *FaviconCache) storeIcon(domain string, icon string) {
	cache.mu.Lock()
	cache.memory[domain] = icon
	cache.mu.Unlock()
}

func (cache *FaviconCache) markMissing(domain string) {
	cache.mu.Lock()
	cache.missing[domain] = struct{}{}
	cache.mu.Unlock()
}

func normalizeDomain(domain string) string {
	return strings.ToLower(strings.TrimSpace(domain))
}

func sanitizeDomainKey(value string) string {
	trimmed := normalizeDomain(value)
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

func dataToDataURI(payload []byte) string {
	if len(payload) == 0 {
		return ""
	}
	encoded := base64.StdEncoding.EncodeToString(payload)
	return "data:image/png;base64," + encoded
}
