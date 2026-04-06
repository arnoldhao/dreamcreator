package agentruntime

import (
	"encoding/json"
	"math"
	"strings"
	"unicode/utf8"

	"github.com/cloudwego/eino/schema"
)

const (
	defaultCharsPerToken = 4
	messageTokenOverhead = 4
	tokenSafetyMargin    = 1.2
)

func EstimateTextTokens(text string) int {
	if text == "" {
		return 0
	}
	asciiCount := 0
	nonAsciiCount := 0
	for _, r := range text {
		if r <= 0x7f {
			asciiCount++
		} else {
			nonAsciiCount++
		}
	}
	if asciiCount == 0 && nonAsciiCount == 0 {
		return 0
	}
	asciiTokens := int(math.Ceil(float64(asciiCount) / float64(defaultCharsPerToken)))
	return asciiTokens + nonAsciiCount
}

func applyTokenSafetyMargin(tokens int) int {
	if tokens <= 0 {
		return 0
	}
	return int(math.Ceil(float64(tokens) * tokenSafetyMargin))
}

func stripToolResultDetailsForEstimate(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return raw
	}
	var payload any
	if err := json.Unmarshal([]byte(trimmed), &payload); err != nil {
		return raw
	}
	sanitized, changed := stripDetailsRecursive(payload)
	if !changed {
		return raw
	}
	encoded, err := json.Marshal(sanitized)
	if err != nil {
		return raw
	}
	return string(encoded)
}

func stripDetailsRecursive(value any) (any, bool) {
	switch typed := value.(type) {
	case map[string]any:
		changed := false
		result := make(map[string]any, len(typed))
		for key, raw := range typed {
			if strings.EqualFold(strings.TrimSpace(key), "details") {
				changed = true
				continue
			}
			next, nestedChanged := stripDetailsRecursive(raw)
			if nestedChanged {
				changed = true
			}
			result[key] = next
		}
		if !changed {
			return value, false
		}
		return result, true
	case []any:
		changed := false
		result := make([]any, 0, len(typed))
		for _, raw := range typed {
			next, nestedChanged := stripDetailsRecursive(raw)
			if nestedChanged {
				changed = true
			}
			result = append(result, next)
		}
		if !changed {
			return value, false
		}
		return result, true
	default:
		return value, false
	}
}

func estimateMessageTokens(message *schema.Message, applySafety bool) int {
	if message == nil {
		return 0
	}
	total := messageTokenOverhead
	content := message.Content
	if message.Role == schema.Tool {
		content = stripToolResultDetailsForEstimate(content)
	}
	total += EstimateTextTokens(content)
	total += EstimateTextTokens(message.ReasoningContent)
	if len(message.ToolCalls) > 0 {
		for _, call := range message.ToolCalls {
			total += EstimateTextTokens(call.Function.Name)
			total += EstimateTextTokens(call.Function.Arguments)
			total += 2
		}
	}
	if total < 0 {
		return 0
	}
	if applySafety {
		return applyTokenSafetyMargin(total)
	}
	return total
}

func EstimateMessageTokens(message *schema.Message) int {
	return estimateMessageTokens(message, false)
}

func EstimateMessageTokensSafe(message *schema.Message) int {
	return estimateMessageTokens(message, true)
}

func EstimateMessagesTokens(messages []*schema.Message) int {
	if len(messages) == 0 {
		return 0
	}
	total := 0
	for _, message := range messages {
		total += EstimateMessageTokens(message)
	}
	return total
}

func EstimateMessagesTokensSafe(messages []*schema.Message) int {
	if len(messages) == 0 {
		return 0
	}
	total := 0
	for _, message := range messages {
		total += EstimateMessageTokensSafe(message)
	}
	return total
}

func EstimateBytesTokens(payload []byte) int {
	if len(payload) == 0 {
		return 0
	}
	return int(math.Ceil(float64(utf8.RuneCount(payload)) / float64(defaultCharsPerToken)))
}
