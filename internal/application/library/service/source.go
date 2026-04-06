package service

import (
	"fmt"
	"strconv"
	"strings"
)

func getString(values map[string]any, keys ...string) string {
	if values == nil {
		return ""
	}
	for _, key := range keys {
		raw, ok := values[key]
		if !ok {
			continue
		}
		switch value := raw.(type) {
		case string:
			trimmed := strings.TrimSpace(value)
			if trimmed != "" {
				return trimmed
			}
		}
	}
	return ""
}

func getStringSlice(values map[string]any, keys ...string) []string {
	if values == nil {
		return nil
	}
	for _, key := range keys {
		raw, ok := values[key]
		if !ok {
			continue
		}
		switch value := raw.(type) {
		case []string:
			result := make([]string, 0, len(value))
			for _, item := range value {
				if trimmed := strings.TrimSpace(item); trimmed != "" {
					result = append(result, trimmed)
				}
			}
			return result
		case []any:
			result := make([]string, 0, len(value))
			for _, item := range value {
				if text, ok := item.(string); ok {
					if trimmed := strings.TrimSpace(text); trimmed != "" {
						result = append(result, trimmed)
					}
				}
			}
			return result
		}
	}
	return nil
}

func getInt64(values map[string]any, key string) (int64, error) {
	if values == nil {
		return 0, fmt.Errorf("missing key %s", key)
	}
	raw, ok := values[key]
	if !ok {
		return 0, fmt.Errorf("missing key %s", key)
	}
	switch value := raw.(type) {
	case float64:
		return int64(value), nil
	case int64:
		return value, nil
	case int:
		return int64(value), nil
	case string:
		parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
		if err != nil {
			return 0, err
		}
		return parsed, nil
	default:
		return 0, fmt.Errorf("unsupported value for %s", key)
	}
}
