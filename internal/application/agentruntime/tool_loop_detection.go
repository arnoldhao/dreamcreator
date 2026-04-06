package agentruntime

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/cloudwego/eino/schema"
)

type ToolLoopDetectors struct {
	GenericRepeat       bool `json:"genericRepeat"`
	KnownPollNoProgress bool `json:"knownPollNoProgress"`
	PingPong            bool `json:"pingPong"`
}

type ToolLoopDetectionConfig struct {
	Enabled                       bool              `json:"enabled"`
	HistorySize                   int               `json:"historySize"`
	WarnThreshold                 int               `json:"warnThreshold"`
	CriticalThreshold             int               `json:"criticalThreshold"`
	GlobalCircuitBreakerThreshold int               `json:"globalCircuitBreakerThreshold"`
	Detectors                     ToolLoopDetectors `json:"detectors"`
}

const (
	defaultToolLoopHistorySize                   = 30
	defaultToolLoopWarnThreshold                 = 10
	defaultToolLoopCriticalThreshold             = 20
	defaultToolLoopGlobalCircuitBreakerThreshold = 30
)

func normalizeToolLoopConfig(config ToolLoopDetectionConfig) ToolLoopDetectionConfig {
	historySize := normalizePositiveInt(config.HistorySize, defaultToolLoopHistorySize)
	warnThreshold := normalizePositiveInt(config.WarnThreshold, defaultToolLoopWarnThreshold)
	criticalThreshold := normalizePositiveInt(config.CriticalThreshold, defaultToolLoopCriticalThreshold)
	globalThreshold := normalizePositiveInt(config.GlobalCircuitBreakerThreshold, defaultToolLoopGlobalCircuitBreakerThreshold)

	if criticalThreshold <= warnThreshold {
		criticalThreshold = warnThreshold + 1
	}
	if globalThreshold <= criticalThreshold {
		globalThreshold = criticalThreshold + 1
	}

	return ToolLoopDetectionConfig{
		Enabled:                       config.Enabled,
		HistorySize:                   historySize,
		WarnThreshold:                 warnThreshold,
		CriticalThreshold:             criticalThreshold,
		GlobalCircuitBreakerThreshold: globalThreshold,
		Detectors: ToolLoopDetectors{
			GenericRepeat:       config.Detectors.GenericRepeat,
			KnownPollNoProgress: config.Detectors.KnownPollNoProgress,
			PingPong:            config.Detectors.PingPong,
		},
	}
}

func normalizePositiveInt(value int, fallback int) int {
	if value <= 0 {
		return fallback
	}
	return value
}

type ToolLoopDetectionResult struct {
	Stuck          bool
	Level          string
	Detector       string
	Count          int
	Message        string
	ToolName       string
	PairedToolName string
	WarningKey     string
}

type toolCallRecord struct {
	toolName   string
	argsHash   string
	toolCallID string
	resultHash string
	timestamp  time.Time
}

type ToolLoopDetector struct {
	config     ToolLoopDetectionConfig
	history    []toolCallRecord
	lastResult ToolLoopDetectionResult
}

func NewToolLoopDetector(config ToolLoopDetectionConfig) *ToolLoopDetector {
	normalized := normalizeToolLoopConfig(config)
	if !normalized.Enabled {
		return nil
	}
	return &ToolLoopDetector{config: normalized}
}

func (detector *ToolLoopDetector) ObserveCalls(calls []schema.ToolCall) ToolLoopDetectionResult {
	if detector == nil || len(calls) == 0 {
		return ToolLoopDetectionResult{}
	}
	var picked ToolLoopDetectionResult
	for _, call := range calls {
		name := strings.TrimSpace(call.Function.Name)
		if name == "" {
			continue
		}
		params := parseJSONValue(call.Function.Arguments)
		result := detector.detect(name, params)
		detector.recordCall(name, params, call.ID)
		picked = pickLoopResult(picked, result)
		if picked.Stuck && picked.Level == "critical" {
			break
		}
	}
	detector.lastResult = picked
	return picked
}

func (detector *ToolLoopDetector) RecordOutcome(toolName string, args string, toolCallID string, output string, err error) {
	if detector == nil {
		return
	}
	name := strings.TrimSpace(toolName)
	if name == "" {
		return
	}
	params := parseJSONValue(args)
	var result any
	if err == nil {
		result = parseJSONValue(output)
	}
	resultHash := hashToolOutcome(name, params, result, err)
	if resultHash == "" {
		return
	}
	detector.attachOutcome(name, params, toolCallID, resultHash)
}

func (detector *ToolLoopDetector) detect(toolName string, params any) ToolLoopDetectionResult {
	config := detector.config
	if !config.Enabled {
		return ToolLoopDetectionResult{}
	}
	history := detector.history
	argsHash := hashToolCall(toolName, params)
	noProgressCount, latestResultHash := getNoProgressStreak(history, toolName, argsHash)
	knownPoll := isKnownPollToolCall(toolName, params)
	pingPong := getPingPongStreak(history, argsHash)

	if noProgressCount >= config.GlobalCircuitBreakerThreshold {
		return ToolLoopDetectionResult{
			Stuck:      true,
			Level:      "critical",
			Detector:   "global_circuit_breaker",
			Count:      noProgressCount,
			ToolName:   toolName,
			Message:    "CRITICAL: identical no-progress tool outcomes repeated; blocking to prevent runaway loop.",
			WarningKey: "global:" + toolName + ":" + argsHash + ":" + latestResultHash,
		}
	}

	if knownPoll && config.Detectors.KnownPollNoProgress && noProgressCount >= config.CriticalThreshold {
		return ToolLoopDetectionResult{
			Stuck:      true,
			Level:      "critical",
			Detector:   "known_poll_no_progress",
			Count:      noProgressCount,
			ToolName:   toolName,
			Message:    "CRITICAL: repeated polling with no progress; blocking to prevent runaway loop.",
			WarningKey: "poll:" + toolName + ":" + argsHash + ":" + latestResultHash,
		}
	}
	if knownPoll && config.Detectors.KnownPollNoProgress && noProgressCount >= config.WarnThreshold {
		return ToolLoopDetectionResult{
			Stuck:      true,
			Level:      "warning",
			Detector:   "known_poll_no_progress",
			Count:      noProgressCount,
			ToolName:   toolName,
			Message:    "WARNING: polling with identical arguments is not making progress.",
			WarningKey: "poll:" + toolName + ":" + argsHash + ":" + latestResultHash,
		}
	}

	pingPongKey := "pingpong:" + argsHash
	if pingPong.pairedSignature != "" {
		pingPongKey = "pingpong:" + canonicalPairKey(argsHash, pingPong.pairedSignature)
	}
	if config.Detectors.PingPong && pingPong.count >= config.CriticalThreshold && pingPong.noProgressEvidence {
		return ToolLoopDetectionResult{
			Stuck:          true,
			Level:          "critical",
			Detector:       "ping_pong",
			Count:          pingPong.count,
			ToolName:       toolName,
			PairedToolName: pingPong.pairedToolName,
			Message:        "CRITICAL: alternating tool calls with no progress; blocking to prevent runaway loop.",
			WarningKey:     pingPongKey,
		}
	}
	if config.Detectors.PingPong && pingPong.count >= config.WarnThreshold {
		return ToolLoopDetectionResult{
			Stuck:          true,
			Level:          "warning",
			Detector:       "ping_pong",
			Count:          pingPong.count,
			ToolName:       toolName,
			PairedToolName: pingPong.pairedToolName,
			Message:        "WARNING: alternating tool calls detected; consider stopping retries.",
			WarningKey:     pingPongKey,
		}
	}

	if !knownPoll && config.Detectors.GenericRepeat {
		recentCount := 0
		for _, item := range history {
			if item.toolName == toolName && item.argsHash == argsHash {
				recentCount++
			}
		}
		if recentCount >= config.WarnThreshold {
			return ToolLoopDetectionResult{
				Stuck:      true,
				Level:      "warning",
				Detector:   "generic_repeat",
				Count:      recentCount,
				ToolName:   toolName,
				Message:    "WARNING: identical tool call repeated; consider stopping retries.",
				WarningKey: "generic:" + toolName + ":" + argsHash,
			}
		}
	}

	return ToolLoopDetectionResult{}
}

func pickLoopResult(current ToolLoopDetectionResult, next ToolLoopDetectionResult) ToolLoopDetectionResult {
	if !next.Stuck {
		return current
	}
	if !current.Stuck {
		return next
	}
	if current.Level == "critical" && next.Level != "critical" {
		return current
	}
	if next.Level == "critical" && current.Level != "critical" {
		return next
	}
	if next.Count > current.Count {
		return next
	}
	return current
}

func (detector *ToolLoopDetector) recordCall(toolName string, params any, toolCallID string) {
	if detector == nil {
		return
	}
	argsHash := hashToolCall(toolName, params)
	detector.history = append(detector.history, toolCallRecord{
		toolName:   toolName,
		argsHash:   argsHash,
		toolCallID: strings.TrimSpace(toolCallID),
		timestamp:  time.Now(),
	})
	if maxSize := detector.config.HistorySize; maxSize > 0 && len(detector.history) > maxSize {
		detector.history = detector.history[len(detector.history)-maxSize:]
	}
}

func (detector *ToolLoopDetector) attachOutcome(toolName string, params any, toolCallID string, resultHash string) {
	if detector == nil || resultHash == "" {
		return
	}
	argsHash := hashToolCall(toolName, params)
	id := strings.TrimSpace(toolCallID)
	for i := len(detector.history) - 1; i >= 0; i-- {
		record := detector.history[i]
		if id != "" && record.toolCallID != id {
			continue
		}
		if record.toolName != toolName || record.argsHash != argsHash {
			continue
		}
		if record.resultHash != "" {
			continue
		}
		detector.history[i].resultHash = resultHash
		return
	}
	detector.history = append(detector.history, toolCallRecord{
		toolName:   toolName,
		argsHash:   argsHash,
		toolCallID: id,
		resultHash: resultHash,
		timestamp:  time.Now(),
	})
	if maxSize := detector.config.HistorySize; maxSize > 0 && len(detector.history) > maxSize {
		detector.history = detector.history[len(detector.history)-maxSize:]
	}
}

func getNoProgressStreak(history []toolCallRecord, toolName string, argsHash string) (int, string) {
	streak := 0
	latestHash := ""
	for i := len(history) - 1; i >= 0; i-- {
		record := history[i]
		if record.toolName != toolName || record.argsHash != argsHash {
			continue
		}
		if record.resultHash == "" {
			continue
		}
		if latestHash == "" {
			latestHash = record.resultHash
			streak = 1
			continue
		}
		if record.resultHash != latestHash {
			break
		}
		streak++
	}
	return streak, latestHash
}

type pingPongStreak struct {
	count              int
	pairedToolName     string
	pairedSignature    string
	noProgressEvidence bool
}

func getPingPongStreak(history []toolCallRecord, currentSignature string) pingPongStreak {
	if len(history) < 2 {
		return pingPongStreak{}
	}
	last := history[len(history)-1]
	var otherSignature string
	var otherToolName string
	for i := len(history) - 2; i >= 0; i-- {
		call := history[i]
		if call.argsHash != last.argsHash {
			otherSignature = call.argsHash
			otherToolName = call.toolName
			break
		}
	}
	if otherSignature == "" || otherToolName == "" {
		return pingPongStreak{}
	}
	alternatingTailCount := 0
	for i := len(history) - 1; i >= 0; i-- {
		call := history[i]
		expected := last.argsHash
		if alternatingTailCount%2 == 1 {
			expected = otherSignature
		}
		if call.argsHash != expected {
			break
		}
		alternatingTailCount++
	}
	if alternatingTailCount < 2 {
		return pingPongStreak{}
	}
	expectedCurrent := otherSignature
	if currentSignature != expectedCurrent {
		return pingPongStreak{}
	}
	tailStart := len(history) - alternatingTailCount
	if tailStart < 0 {
		tailStart = 0
	}
	firstHashA := ""
	firstHashB := ""
	noProgress := true
	for i := tailStart; i < len(history); i++ {
		call := history[i]
		if call.resultHash == "" {
			noProgress = false
			break
		}
		if call.argsHash == last.argsHash {
			if firstHashA == "" {
				firstHashA = call.resultHash
			} else if firstHashA != call.resultHash {
				noProgress = false
				break
			}
			continue
		}
		if call.argsHash == otherSignature {
			if firstHashB == "" {
				firstHashB = call.resultHash
			} else if firstHashB != call.resultHash {
				noProgress = false
				break
			}
			continue
		}
		noProgress = false
		break
	}
	if firstHashA == "" || firstHashB == "" {
		noProgress = false
	}
	return pingPongStreak{
		count:              alternatingTailCount + 1,
		pairedToolName:     last.toolName,
		pairedSignature:    last.argsHash,
		noProgressEvidence: noProgress,
	}
}

func canonicalPairKey(a string, b string) string {
	pair := []string{a, b}
	sort.Strings(pair)
	return pair[0] + "|" + pair[1]
}

func isKnownPollToolCall(toolName string, params any) bool {
	if toolName == "command_status" {
		return true
	}
	if toolName != "process" {
		return false
	}
	typed, ok := params.(map[string]any)
	if !ok {
		return false
	}
	action, _ := typed["action"].(string)
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "poll", "log":
		return true
	default:
		return false
	}
}

func hashToolCall(toolName string, params any) string {
	return toolName + ":" + digestStable(params)
}

func hashToolOutcome(toolName string, params any, result any, err error) string {
	if err != nil {
		return "error:" + digestStable(formatErrorForHash(err))
	}
	if result == nil {
		return ""
	}
	if typed, ok := result.(map[string]any); ok {
		details, _ := typed["details"].(map[string]any)
		text := extractTextContent(typed)
		if isKnownPollToolCall(toolName, params) && toolName == "process" {
			action := ""
			if paramMap, ok := params.(map[string]any); ok {
				if raw, ok := paramMap["action"].(string); ok {
					action = strings.ToLower(strings.TrimSpace(raw))
				}
			}
			switch action {
			case "poll":
				return digestStable(map[string]any{
					"action":     action,
					"status":     pickMapValue(details, "status"),
					"exitCode":   pickMapValue(details, "exitCode"),
					"exitSignal": pickMapValue(details, "exitSignal"),
					"aggregated": pickMapValue(details, "aggregated"),
					"text":       text,
				})
			case "log":
				return digestStable(map[string]any{
					"action":     action,
					"status":     pickMapValue(details, "status"),
					"totalLines": pickMapValue(details, "totalLines"),
					"totalChars": pickMapValue(details, "totalChars"),
					"truncated":  pickMapValue(details, "truncated"),
					"exitCode":   pickMapValue(details, "exitCode"),
					"exitSignal": pickMapValue(details, "exitSignal"),
					"text":       text,
				})
			}
		}
		return digestStable(map[string]any{
			"details": details,
			"text":    text,
		})
	}
	return digestStable(result)
}

func extractTextContent(result map[string]any) string {
	raw, ok := result["content"]
	if !ok {
		return ""
	}
	content, ok := raw.([]any)
	if !ok {
		return ""
	}
	lines := make([]string, 0)
	for _, entry := range content {
		item, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		kind, _ := item["type"].(string)
		if strings.TrimSpace(kind) != "text" {
			continue
		}
		text, ok := item["text"].(string)
		if !ok {
			continue
		}
		lines = append(lines, text)
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func formatErrorForHash(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func pickMapValue(values map[string]any, key string) any {
	if values == nil {
		return nil
	}
	if raw, ok := values[key]; ok {
		return raw
	}
	return nil
}

func digestStable(value any) string {
	serialized := stableStringifyFallback(value)
	sum := sha256.Sum256([]byte(serialized))
	return hex.EncodeToString(sum[:])
}

func stableStringifyFallback(value any) string {
	serialized, err := stableStringify(value)
	if err == nil {
		return serialized
	}
	if value == nil {
		return "null"
	}
	switch typed := value.(type) {
	case string:
		return typed
	case []byte:
		return string(typed)
	case bool:
		if typed {
			return "true"
		}
		return "false"
	case int:
		return strconv.Itoa(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case error:
		return typed.Error()
	default:
		return "unknown"
	}
}

func stableStringify(value any) (string, error) {
	switch typed := value.(type) {
	case nil:
		return "null", nil
	case string:
		encoded, _ := json.Marshal(typed)
		return string(encoded), nil
	case []byte:
		encoded, _ := json.Marshal(string(typed))
		return string(encoded), nil
	case bool:
		if typed {
			return "true", nil
		}
		return "false", nil
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64), nil
	case float32:
		return strconv.FormatFloat(float64(typed), 'f', -1, 32), nil
	case int:
		return strconv.Itoa(typed), nil
	case int64:
		return strconv.FormatInt(typed, 10), nil
	case json.Number:
		return typed.String(), nil
	case []any:
		builder := strings.Builder{}
		builder.WriteString("[")
		for i, item := range typed {
			if i > 0 {
				builder.WriteString(",")
			}
			encoded, err := stableStringify(item)
			if err != nil {
				return "", err
			}
			builder.WriteString(encoded)
		}
		builder.WriteString("]")
		return builder.String(), nil
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		builder := strings.Builder{}
		builder.WriteString("{")
		for i, key := range keys {
			if i > 0 {
				builder.WriteString(",")
			}
			encodedKey, _ := json.Marshal(key)
			builder.Write(encodedKey)
			builder.WriteString(":")
			encodedValue, err := stableStringify(typed[key])
			if err != nil {
				return "", err
			}
			builder.WriteString(encodedValue)
		}
		builder.WriteString("}")
		return builder.String(), nil
	default:
		encoded, err := json.Marshal(typed)
		if err != nil {
			return "", err
		}
		return string(encoded), nil
	}
}

func parseJSONValue(raw string) any {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	var payload any
	if err := json.Unmarshal([]byte(trimmed), &payload); err == nil {
		return payload
	}
	return trimmed
}
