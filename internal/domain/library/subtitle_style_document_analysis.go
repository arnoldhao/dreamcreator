package library

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type SubtitleStyleDocumentAnalysis struct {
	DetectedFormat   string
	ScriptType       string
	PlayResX         int
	PlayResY         int
	StyleCount       int
	DialogueCount    int
	CommentCount     int
	StyleNames       []string
	Fonts            []string
	FeatureFlags     []string
	ValidationIssues []string
}

var (
	assFontOverridePattern  = regexp.MustCompile(`\\fn([^\\}]+)`)
	assKaraokePattern       = regexp.MustCompile(`\\(?:k|K|kf|ko)\d+`)
	assPositioningPattern   = regexp.MustCompile(`\\(?:pos|move|org)\b`)
	assTransformPattern     = regexp.MustCompile(`\\t\(`)
	assClipPattern          = regexp.MustCompile(`\\(?:clip|iclip)\b`)
	assFadePattern          = regexp.MustCompile(`\\(?:fad|fade)\b`)
	assVectorDrawingPattern = regexp.MustCompile(`\\p\d+`)
	defaultASSStyleFormat   = []string{"name", "fontname", "fontsize", "primarycolour", "secondarycolour", "outlinecolour", "backcolour", "bold", "italic", "underline", "strikeout", "scalex", "scaley", "spacing", "angle", "borderstyle", "outline", "shadow", "alignment", "marginl", "marginr", "marginv", "encoding"}
	defaultASSEventFormat   = []string{"layer", "start", "end", "style", "name", "marginl", "marginr", "marginv", "effect", "text"}
	defaultSSAEventFormat   = []string{"marked", "start", "end", "style", "name", "marginl", "marginr", "marginv", "effect", "text"}
)

func AnalyzeSubtitleStyleDocument(content string) SubtitleStyleDocumentAnalysis {
	lines := strings.Split(normalizeSubtitleStyleDocumentContent(content), "\n")
	currentSection := ""
	styleFormat := append([]string(nil), defaultASSStyleFormat...)
	eventFormat := append([]string(nil), defaultASSEventFormat...)
	detectedFormat := ""
	scriptType := ""
	playResX := 0
	playResY := 0
	styleCount := 0
	dialogueCount := 0
	commentCount := 0
	seenScriptInfo := false
	seenStyles := false
	seenEvents := false
	styleNames := make(map[string]struct{})
	fonts := make(map[string]struct{})
	featureFlags := make(map[string]struct{})
	issues := make([]string, 0, 8)
	issueSet := make(map[string]struct{})

	for index, rawLine := range lines {
		lineNo := index + 1
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			currentSection = strings.ToLower(line)
			switch currentSection {
			case "[script info]":
				seenScriptInfo = true
			case "[v4+ styles]":
				seenStyles = true
				detectedFormat = "ass"
				styleFormat = append([]string(nil), defaultASSStyleFormat...)
			case "[v4 styles]":
				seenStyles = true
				detectedFormat = "ssa"
				styleFormat = append([]string(nil), defaultASSStyleFormat...)
				eventFormat = append([]string(nil), defaultSSAEventFormat...)
			case "[events]":
				seenEvents = true
				if detectedFormat == "ssa" {
					eventFormat = append([]string(nil), defaultSSAEventFormat...)
				} else {
					eventFormat = append([]string(nil), defaultASSEventFormat...)
				}
			}
			continue
		}

		switch currentSection {
		case "[script info]":
			key, value, ok := splitASSKeyValue(line)
			if !ok {
				continue
			}
			switch strings.ToLower(key) {
			case "scripttype":
				scriptType = strings.TrimSpace(value)
				if detectedFormat == "" && strings.EqualFold(scriptType, "v4.00") {
					detectedFormat = "ssa"
					eventFormat = append([]string(nil), defaultSSAEventFormat...)
				}
			case "playresx":
				parsed, err := strconv.Atoi(strings.TrimSpace(value))
				if err != nil || parsed <= 0 {
					addASSValidationIssue(&issues, issueSet, lineNo, fmt.Sprintf("invalid PlayResX value %q", strings.TrimSpace(value)))
					continue
				}
				playResX = parsed
			case "playresy":
				parsed, err := strconv.Atoi(strings.TrimSpace(value))
				if err != nil || parsed <= 0 {
					addASSValidationIssue(&issues, issueSet, lineNo, fmt.Sprintf("invalid PlayResY value %q", strings.TrimSpace(value)))
					continue
				}
				playResY = parsed
			}
		case "[v4+ styles]", "[v4 styles]":
			key, value, ok := splitASSKeyValue(line)
			if !ok {
				continue
			}
			switch strings.ToLower(key) {
			case "format":
				fields := parseASSFormatFields(value)
				if len(fields) == 0 {
					addASSValidationIssue(&issues, issueSet, lineNo, "style Format line is empty")
					continue
				}
				styleFormat = fields
			case "style":
				styleCount++
				fields := splitASSFields(value, len(styleFormat))
				if len(fields) < len(styleFormat) {
					addASSValidationIssue(&issues, issueSet, lineNo, "style row has fewer fields than the current Format line")
				}
				if name := findASSField(styleFormat, fields, "name"); name != "" {
					styleNames[name] = struct{}{}
				}
				if fontName := findASSField(styleFormat, fields, "fontname"); fontName != "" {
					fonts[fontName] = struct{}{}
				}
			}
		case "[events]":
			key, value, ok := splitASSKeyValue(line)
			if !ok {
				continue
			}
			switch strings.ToLower(key) {
			case "format":
				fields := parseASSFormatFields(value)
				if len(fields) == 0 {
					addASSValidationIssue(&issues, issueSet, lineNo, "events Format line is empty")
					continue
				}
				eventFormat = fields
			case "dialogue", "comment":
				fields := splitASSFields(value, len(eventFormat))
				if len(fields) < len(eventFormat) {
					addASSValidationIssue(&issues, issueSet, lineNo, fmt.Sprintf("%s row has fewer fields than the current Format line", strings.ToLower(key)))
				}
				if strings.EqualFold(key, "dialogue") {
					dialogueCount++
				} else {
					commentCount++
				}
				analyzeASSEventText(findASSField(eventFormat, fields, "text"), fonts, featureFlags)
			}
		}
	}

	if detectedFormat == "" {
		if strings.EqualFold(scriptType, "v4.00") {
			detectedFormat = "ssa"
		} else {
			detectedFormat = "ass"
		}
	}
	if !seenScriptInfo {
		addASSValidationIssue(&issues, issueSet, 0, "missing [Script Info] section")
	}
	if !seenStyles {
		addASSValidationIssue(&issues, issueSet, 0, "missing [V4+ Styles] or [V4 Styles] section")
	}
	if !seenEvents {
		addASSValidationIssue(&issues, issueSet, 0, "missing [Events] section")
	}
	if strings.TrimSpace(scriptType) == "" {
		addASSValidationIssue(&issues, issueSet, 0, "missing ScriptType in [Script Info]")
	}
	if playResX <= 0 {
		addASSValidationIssue(&issues, issueSet, 0, "missing PlayResX in [Script Info]")
	}
	if playResY <= 0 {
		addASSValidationIssue(&issues, issueSet, 0, "missing PlayResY in [Script Info]")
	}

	return SubtitleStyleDocumentAnalysis{
		DetectedFormat:   detectedFormat,
		ScriptType:       scriptType,
		PlayResX:         playResX,
		PlayResY:         playResY,
		StyleCount:       styleCount,
		DialogueCount:    dialogueCount,
		CommentCount:     commentCount,
		StyleNames:       sortedASSKeys(styleNames),
		Fonts:            sortedASSKeys(fonts),
		FeatureFlags:     sortedASSKeys(featureFlags),
		ValidationIssues: issues,
	}
}

func splitASSKeyValue(line string) (string, string, bool) {
	key, value, ok := strings.Cut(line, ":")
	if !ok {
		return "", "", false
	}
	return strings.TrimSpace(key), strings.TrimSpace(value), true
}

func parseASSFormatFields(value string) []string {
	fields := splitASSFields(value, 0)
	result := make([]string, 0, len(fields))
	for _, field := range fields {
		normalized := strings.ToLower(strings.TrimSpace(field))
		if normalized == "" {
			continue
		}
		result = append(result, normalized)
	}
	return result
}

func splitASSFields(value string, expected int) []string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	var parts []string
	if expected > 1 {
		parts = strings.SplitN(trimmed, ",", expected)
	} else {
		parts = strings.Split(trimmed, ",")
	}
	for index, part := range parts {
		parts[index] = strings.TrimSpace(part)
	}
	return parts
}

func findASSField(format []string, values []string, name string) string {
	normalizedName := strings.ToLower(strings.TrimSpace(name))
	for index, field := range format {
		if field != normalizedName || index >= len(values) {
			continue
		}
		return strings.TrimSpace(values[index])
	}
	return ""
}

func analyzeASSEventText(text string, fonts map[string]struct{}, featureFlags map[string]struct{}) {
	if strings.TrimSpace(text) == "" {
		return
	}
	if strings.Contains(text, "{") && strings.Contains(text, "\\") {
		featureFlags["override-tags"] = struct{}{}
	}
	for _, match := range assFontOverridePattern.FindAllStringSubmatch(text, -1) {
		fontName := strings.TrimSpace(match[1])
		if fontName == "" {
			continue
		}
		fonts[fontName] = struct{}{}
		featureFlags["font-override"] = struct{}{}
	}
	if assPositioningPattern.MatchString(text) {
		featureFlags["positioning"] = struct{}{}
	}
	if assTransformPattern.MatchString(text) {
		featureFlags["transform"] = struct{}{}
	}
	if assKaraokePattern.MatchString(text) {
		featureFlags["karaoke"] = struct{}{}
	}
	if assVectorDrawingPattern.MatchString(text) {
		featureFlags["vector-drawing"] = struct{}{}
	}
	if assClipPattern.MatchString(text) {
		featureFlags["clipping"] = struct{}{}
	}
	if assFadePattern.MatchString(text) {
		featureFlags["fade"] = struct{}{}
	}
}

func addASSValidationIssue(issues *[]string, seen map[string]struct{}, lineNo int, message string) {
	message = strings.TrimSpace(message)
	if message == "" {
		return
	}
	if lineNo > 0 {
		message = fmt.Sprintf("Line %d: %s", lineNo, message)
	}
	if _, exists := seen[message]; exists {
		return
	}
	seen[message] = struct{}{}
	*issues = append(*issues, message)
}

func sortedASSKeys(values map[string]struct{}) []string {
	if len(values) == 0 {
		return nil
	}
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}
