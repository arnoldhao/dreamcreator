package runtime

import (
	"encoding/json"
	"net/url"
	"strconv"
	"strings"

	"dreamcreator/internal/application/chatevent"
)

type sourceItem struct {
	ID    string
	URL   string
	Title string
}

func collectSourceItemsFromToolOutput(raw json.RawMessage, toolName string, toolCallID string) []sourceItem {
	if len(raw) == 0 {
		return nil
	}

	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil
	}

	result := make([]sourceItem, 0, 4)
	seen := make(map[string]struct{})
	var visit func(value any, depth int)
	visit = func(value any, depth int) {
		if depth > 5 || value == nil {
			return
		}
		switch typed := value.(type) {
		case map[string]any:
			if item, ok := normalizeSourceItem(typed); ok {
				key := normalizeSourceKey(item.URL)
				if _, exists := seen[key]; !exists {
					seen[key] = struct{}{}
					if item.ID == "" {
						item.ID = buildSourceItemID(toolCallID, toolName, len(result)+1)
					}
					result = append(result, item)
				}
			}
			for _, key := range []string{"results", "sources", "items", "documents", "data", "value"} {
				child, ok := typed[key]
				if !ok {
					continue
				}
				visit(child, depth+1)
			}
		case []any:
			for _, item := range typed {
				visit(item, depth+1)
			}
		case string:
			trimmed := strings.TrimSpace(typed)
			if !looksLikeHTTPURL(trimmed) {
				return
			}
			key := normalizeSourceKey(trimmed)
			if _, exists := seen[key]; exists {
				return
			}
			seen[key] = struct{}{}
			result = append(result, sourceItem{
				ID:  buildSourceItemID(toolCallID, toolName, len(result)+1),
				URL: trimmed,
			})
		}
	}

	visit(decoded, 0)
	return result
}

func appendSourceMessageParts(
	parts []chatevent.MessagePart,
	sources []sourceItem,
	nextParentID func() string,
	marshalRaw func(value any) json.RawMessage,
) []chatevent.MessagePart {
	if len(sources) == 0 {
		return parts
	}
	parentID := ""
	if nextParentID != nil {
		parentID = nextParentID()
	}
	for _, source := range sources {
		urlValue := strings.TrimSpace(source.URL)
		if !looksLikeHTTPURL(urlValue) {
			continue
		}
		payload := map[string]any{
			"id":  strings.TrimSpace(source.ID),
			"url": urlValue,
		}
		if title := strings.TrimSpace(source.Title); title != "" {
			payload["title"] = title
		}
		var data json.RawMessage
		if marshalRaw != nil {
			data = marshalRaw(payload)
		}
		parts = append(parts, chatevent.MessagePart{
			Type:     "source",
			ParentID: parentID,
			Text:     urlValue,
			Data:     data,
		})
	}
	return parts
}

func normalizeSourceItem(raw map[string]any) (sourceItem, bool) {
	if len(raw) == 0 {
		return sourceItem{}, false
	}
	urlValue := firstString(raw, "url", "href", "link", "sourceUrl", "sourceURL", "uri")
	if !looksLikeHTTPURL(urlValue) {
		return sourceItem{}, false
	}
	return sourceItem{
		ID:    firstString(raw, "id", "sourceId", "sourceID", "ref"),
		URL:   strings.TrimSpace(urlValue),
		Title: firstString(raw, "title", "name", "label"),
	}, true
}

func firstString(raw map[string]any, keys ...string) string {
	for _, key := range keys {
		value, ok := raw[key]
		if !ok {
			continue
		}
		if typed, ok := value.(string); ok {
			trimmed := strings.TrimSpace(typed)
			if trimmed != "" {
				return trimmed
			}
		}
	}
	return ""
}

func buildSourceItemID(toolCallID string, toolName string, index int) string {
	base := strings.TrimSpace(toolCallID)
	if base == "" {
		base = strings.TrimSpace(toolName)
	}
	if base == "" {
		base = "source"
	}
	if index < 1 {
		index = 1
	}
	return base + "-" + strconv.Itoa(index)
}

func normalizeSourceKey(rawURL string) string {
	return strings.ToLower(strings.TrimSpace(rawURL))
}

func looksLikeHTTPURL(rawURL string) bool {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return false
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return false
	}
	scheme := strings.ToLower(strings.TrimSpace(parsed.Scheme))
	if scheme != "http" && scheme != "https" {
		return false
	}
	return strings.TrimSpace(parsed.Host) != ""
}
