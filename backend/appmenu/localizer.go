package appmenu

import (
	"embed"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

//go:embed i18n/*.json
var i18nFS embed.FS

type Localizer struct {
	mu        sync.RWMutex
	locale    string
	messages  map[string]string
	fallback  map[string]string
	available map[string]struct{}
}

func NewLocalizer(preferredLocale string) *Localizer {
	l := &Localizer{
		locale:    "en-us",
		messages:  map[string]string{},
		fallback:  map[string]string{},
		available: map[string]struct{}{},
	}

	for _, p := range mustGlobI18nFiles() {
		base := strings.TrimSuffix(filepath.Base(p), ".json")
		l.available[base] = struct{}{}
	}

	l.fallback = loadMessages("en-us")
	l.SetLocale(preferredLocale)
	return l
}

func (l *Localizer) SetLocale(locale string) {
	normalized := normalizeLocale(locale)
	if normalized == "" || normalized == "auto" {
		normalized = normalizeLocale(systemLocale())
	}

	best := l.bestMatch(normalized)
	msg := loadMessages(best)

	l.mu.Lock()
	l.locale = best
	l.messages = msg
	l.mu.Unlock()
}

func (l *Localizer) Locale() string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.locale
}

func (l *Localizer) T(key string) string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if v, ok := l.messages[key]; ok && v != "" {
		return v
	}
	if v, ok := l.fallback[key]; ok && v != "" {
		return v
	}
	return key
}

func (l *Localizer) Format(key string, vars map[string]string) string {
	s := l.T(key)
	if len(vars) == 0 {
		return s
	}
	for k, v := range vars {
		s = strings.ReplaceAll(s, "{"+k+"}", v)
	}
	return s
}

func (l *Localizer) bestMatch(locale string) string {
	if locale == "" {
		return "en-us"
	}
	if _, ok := l.available[locale]; ok {
		return locale
	}

	// Try language-only match (e.g. "en" -> "en-us", "zh" -> "zh-cn").
	lang := locale
	if i := strings.IndexByte(locale, '-'); i > 0 {
		lang = locale[:i]
	}
	if _, ok := l.available[lang]; ok {
		return lang
	}

	switch lang {
	case "zh":
		if _, ok := l.available["zh-cn"]; ok {
			return "zh-cn"
		}
	case "en":
		if _, ok := l.available["en-us"]; ok {
			return "en-us"
		}
	}

	// Pick a deterministic fallback.
	if _, ok := l.available["en-us"]; ok {
		return "en-us"
	}
	return "en-us"
}

func normalizeLocale(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	v = strings.ToLower(v)
	v = strings.ReplaceAll(v, "_", "-")

	// LANG env often has encoding suffix: zh_CN.UTF-8
	if i := strings.IndexByte(v, '.'); i > 0 {
		v = v[:i]
	}
	return v
}

func systemLocale() string {
	for _, k := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v
		}
	}
	return ""
}

func mustGlobI18nFiles() []string {
	entries, err := i18nFS.ReadDir("i18n")
	if err != nil {
		return nil
	}
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		out = append(out, filepath.Join("i18n", e.Name()))
	}
	return out
}

func loadMessages(locale string) map[string]string {
	if locale == "" {
		locale = "en-us"
	}
	b, err := i18nFS.ReadFile(filepath.Join("i18n", locale+".json"))
	if err != nil {
		return map[string]string{}
	}
	var m map[string]string
	if err := json.Unmarshal(b, &m); err != nil {
		return map[string]string{}
	}
	return m
}
