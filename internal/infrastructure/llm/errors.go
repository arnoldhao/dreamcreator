package llm

import (
	"encoding/json"
	"fmt"
	"strings"
)

type HTTPStatusError struct {
	Code    int
	Message string
	Body    string
}

func (err *HTTPStatusError) Error() string {
	message := strings.TrimSpace(err.Message)
	if message == "" {
		message = "request failed"
	}
	if err.Code > 0 {
		return fmt.Sprintf("llm request failed (%d): %s", err.Code, message)
	}
	return fmt.Sprintf("llm request failed: %s", message)
}

func (err *HTTPStatusError) StatusCode() int {
	return err.Code
}

func (err *HTTPStatusError) SafeMessage() string {
	return strings.TrimSpace(err.Message)
}

func extractHTTPErrorMessage(body []byte) string {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return ""
	}
	var payload struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
		Message string `json:"message"`
		Detail  string `json:"detail"`
	}
	if json.Unmarshal(body, &payload) == nil {
		if message := strings.TrimSpace(payload.Error.Message); message != "" {
			return message
		}
		if message := strings.TrimSpace(payload.Message); message != "" {
			return message
		}
		if message := strings.TrimSpace(payload.Detail); message != "" {
			return message
		}
	}

	var generic map[string]any
	if json.Unmarshal(body, &generic) == nil {
		if message := findMessageInPayload(generic); message != "" {
			return message
		}
		return ""
	}

	if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
		return ""
	}
	return trimmed
}

func findMessageInPayload(payload map[string]any) string {
	if payload == nil {
		return ""
	}
	for _, key := range []string{"message", "detail", "error"} {
		value, ok := payload[key]
		if !ok || value == nil {
			continue
		}
		switch typed := value.(type) {
		case string:
			if message := strings.TrimSpace(typed); message != "" {
				return message
			}
		case map[string]any:
			if message, ok := typed["message"].(string); ok {
				if trimmed := strings.TrimSpace(message); trimmed != "" {
					return trimmed
				}
			}
		}
	}
	return ""
}
