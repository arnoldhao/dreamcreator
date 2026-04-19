package tools

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type toolArgs map[string]any

func parseToolArgs(args string) (toolArgs, error) {
	trimmed := strings.TrimSpace(args)
	if trimmed == "" {
		return toolArgs{}, nil
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(trimmed), &payload); err != nil {
		return nil, err
	}
	return toolArgs(payload), nil
}

func getStringArg(args toolArgs, keys ...string) string {
	for _, key := range keys {
		if key == "" {
			continue
		}
		if raw, ok := args[key]; ok {
			switch typed := raw.(type) {
			case string:
				if value := strings.TrimSpace(typed); value != "" {
					return value
				}
			case []byte:
				if value := strings.TrimSpace(string(typed)); value != "" {
					return value
				}
			}
		}
	}
	return ""
}

func getBoolArg(args toolArgs, keys ...string) (bool, bool) {
	for _, key := range keys {
		if key == "" {
			continue
		}
		if raw, ok := args[key]; ok {
			switch typed := raw.(type) {
			case bool:
				return typed, true
			case string:
				value := strings.TrimSpace(strings.ToLower(typed))
				if value == "true" || value == "1" || value == "yes" {
					return true, true
				}
				if value == "false" || value == "0" || value == "no" {
					return false, true
				}
			}
		}
	}
	return false, false
}

func getIntArg(args toolArgs, keys ...string) (int, bool) {
	for _, key := range keys {
		if key == "" {
			continue
		}
		if raw, ok := args[key]; ok {
			switch typed := raw.(type) {
			case int:
				return typed, true
			case int64:
				return int(typed), true
			case float64:
				return int(typed), true
			case string:
				value := strings.TrimSpace(typed)
				if value == "" {
					continue
				}
				parsed, err := strconv.Atoi(value)
				if err == nil {
					return parsed, true
				}
			}
		}
	}
	return 0, false
}

func getMapArg(args toolArgs, keys ...string) map[string]any {
	for _, key := range keys {
		if key == "" {
			continue
		}
		if raw, ok := args[key]; ok {
			switch typed := raw.(type) {
			case map[string]any:
				return typed
			case map[string]string:
				result := make(map[string]any, len(typed))
				for itemKey, itemValue := range typed {
					result[itemKey] = itemValue
				}
				return result
			}
		}
	}
	return nil
}

func getStringSliceArg(args toolArgs, keys ...string) []string {
	for _, key := range keys {
		if key == "" {
			continue
		}
		if raw, ok := args[key]; ok {
			switch typed := raw.(type) {
			case []string:
				return normalizeStringSlice(typed)
			case []any:
				result := make([]string, 0, len(typed))
				for _, entry := range typed {
					if value, ok := entry.(string); ok && strings.TrimSpace(value) != "" {
						result = append(result, strings.TrimSpace(value))
					}
				}
				return result
			}
		}
	}
	return nil
}

func normalizeStringSlice(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		result = append(result, value)
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func resolvePath(args toolArgs) (string, error) {
	path := getStringArg(args, "path", "file", "filePath", "filepath")
	if path == "" {
		return "", errors.New("path is required")
	}
	root := getStringArg(args, "rootPath", "root", "baseDir", "basePath")
	if root == "" {
		return filepath.Clean(path), nil
	}
	if filepath.IsAbs(path) {
		cleaned := filepath.Clean(path)
		if !isSubpath(root, cleaned) {
			return "", errors.New("path escapes root")
		}
		return cleaned, nil
	}
	joined := filepath.Join(root, path)
	cleaned := filepath.Clean(joined)
	if !isSubpath(root, cleaned) {
		return "", errors.New("path escapes root")
	}
	return cleaned, nil
}

func isSubpath(root, target string) bool {
	root = filepath.Clean(root)
	target = filepath.Clean(target)
	if root == "." || root == "" {
		return true
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return false
	}
	if rootAbs == targetAbs {
		return true
	}
	rel, err := filepath.Rel(rootAbs, targetAbs)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	return !strings.HasPrefix(rel, "..")
}

func readFileLimited(path string, maxChars int) ([]byte, bool, error) {
	if maxChars <= 0 {
		data, err := os.ReadFile(path)
		return data, false, err
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, false, err
	}
	defer file.Close()
	buf := make([]byte, maxChars)
	n, err := io.ReadFull(file, buf)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return nil, false, err
	}
	truncated := false
	if n == maxChars {
		next := make([]byte, 1)
		if _, err := file.Read(next); err == nil {
			truncated = true
		}
	}
	return buf[:n], truncated, nil
}

func marshalResult(value any) string {
	if value == nil {
		return "null"
	}
	data, err := json.Marshal(value)
	if err != nil {
		return "null"
	}
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return "null"
	}
	return trimmed
}

func schemaJSON(value any) string {
	if value == nil {
		return ""
	}
	data, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(data)
}

func getNestedMap(root map[string]any, path ...string) map[string]any {
	current := root
	for _, key := range path {
		if current == nil {
			return nil
		}
		raw, ok := current[key]
		if !ok {
			return nil
		}
		next, ok := raw.(map[string]any)
		if !ok {
			return nil
		}
		current = next
	}
	return current
}

func getNestedString(root map[string]any, path ...string) string {
	if len(path) == 0 {
		return ""
	}
	last := path[len(path)-1]
	parent := getNestedMap(root, path[:len(path)-1]...)
	if parent == nil {
		return ""
	}
	value, ok := parent[last]
	if !ok {
		return ""
	}
	if str, ok := value.(string); ok {
		return strings.TrimSpace(str)
	}
	return ""
}

func getNestedBool(root map[string]any, path ...string) (bool, bool) {
	if len(path) == 0 {
		return false, false
	}
	last := path[len(path)-1]
	parent := getNestedMap(root, path[:len(path)-1]...)
	if parent == nil {
		return false, false
	}
	switch typed := parent[last].(type) {
	case bool:
		return typed, true
	case string:
		value := strings.TrimSpace(strings.ToLower(typed))
		if value == "true" || value == "1" || value == "yes" {
			return true, true
		}
		if value == "false" || value == "0" || value == "no" {
			return false, true
		}
	}
	return false, false
}

func getNestedInt(root map[string]any, path ...string) (int, bool) {
	if len(path) == 0 {
		return 0, false
	}
	last := path[len(path)-1]
	parent := getNestedMap(root, path[:len(path)-1]...)
	if parent == nil {
		return 0, false
	}
	switch typed := parent[last].(type) {
	case int:
		return typed, true
	case int64:
		return int(typed), true
	case float64:
		return int(typed), true
	case string:
		value := strings.TrimSpace(typed)
		if value == "" {
			return 0, false
		}
		parsed, err := strconv.Atoi(value)
		if err == nil {
			return parsed, true
		}
	}
	return 0, false
}

func containsString(values []string, needle string) bool {
	needle = strings.TrimSpace(needle)
	for _, value := range values {
		if strings.TrimSpace(value) == needle {
			return true
		}
	}
	return false
}

func trimToMaxChars(value string, maxChars int) string {
	value = strings.TrimSpace(value)
	if maxChars <= 0 || len(value) <= maxChars {
		return value
	}
	return value[:maxChars]
}
