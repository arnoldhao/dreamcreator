package service

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

const defaultSubtitleExportStyleDocumentContent = "[Script Info]\n" +
	"Title: DreamCreator Export\n" +
	"ScriptType: v4.00+\n" +
	"WrapStyle: 0\n" +
	"ScaledBorderAndShadow: yes\n" +
	"PlayResX: 1920\n" +
	"PlayResY: 1080\n" +
	"\n" +
	"[V4+ Styles]\n" +
	"Format: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour, Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, Encoding\n" +
	"Style: Default,Arial,48,&H00FFFFFF,&H00FFFFFF,&H00111111,&HFF111111,-1,0,0,0,100,100,0,0,1,2,0,2,72,72,56,1\n" +
	"\n" +
	"[Events]\n" +
	"Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text\n"

var (
	defaultSubtitleExportStyleFormat = []string{
		"name",
		"fontname",
		"fontsize",
		"primarycolour",
		"secondarycolour",
		"outlinecolour",
		"backcolour",
		"bold",
		"italic",
		"underline",
		"strikeout",
		"scalex",
		"scaley",
		"spacing",
		"angle",
		"borderstyle",
		"outline",
		"shadow",
		"alignment",
		"marginl",
		"marginr",
		"marginv",
		"encoding",
	}
	defaultSubtitleExportEventFormat = "Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text"
)

type subtitleExportStyleDocument struct {
	Lines       []string
	EventFormat string
	StyleNames  []string
	Styles      map[string]subtitleExportStyle
}

type subtitleExportStyle struct {
	Name         string
	FontName     string
	FontSize     float64
	PrimaryColor subtitleExportColor
	OutlineColor subtitleExportColor
	BackColor    subtitleExportColor
	Bold         bool
	Italic       bool
	BorderStyle  int
	Outline      float64
	Shadow       float64
	Alignment    int
	MarginL      int
	MarginR      int
	MarginV      int
}

type subtitleExportStyleDocumentOptions struct {
	Title    string
	PlayResX int
	PlayResY int
}

type subtitleExportColor struct {
	Red   uint8
	Green uint8
	Blue  uint8
	Alpha uint8
}

func resolveSubtitleExportStyleDocument(content string, options subtitleExportStyleDocumentOptions) subtitleExportStyleDocument {
	normalized := normalizeSubtitleExportStyleDocumentContent(content)
	if normalized == "" {
		normalized = defaultSubtitleExportStyleDocumentContent
	}
	result, hasScriptInfo, hasStyles := parseSubtitleExportStyleDocumentContent(normalized, options)
	if hasScriptInfo && hasStyles && len(result.StyleNames) > 0 {
		return result
	}
	fallback, _, _ := parseSubtitleExportStyleDocumentContent(defaultSubtitleExportStyleDocumentContent, options)
	return fallback
}

func parseSubtitleExportStyleDocumentContent(
	content string,
	options subtitleExportStyleDocumentOptions,
) (subtitleExportStyleDocument, bool, bool) {
	lines := strings.Split(normalizeSubtitleExportStyleDocumentContent(content), "\n")
	preservedSections := make([]string, 0, len(lines))
	styleNames := make([]string, 0, 4)
	styles := make(map[string]subtitleExportStyle)
	styleFormat := append([]string(nil), defaultSubtitleExportStyleFormat...)
	eventFormat := defaultSubtitleExportEventFormat
	currentSection := ""
	hasScriptInfo := false
	hasStyles := false
	titleInserted := strings.TrimSpace(options.Title) == ""
	playResXInserted := options.PlayResX <= 0
	playResYInserted := options.PlayResY <= 0

	for _, rawLine := range lines {
		trimmed := strings.TrimSpace(rawLine)
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			currentSection = strings.ToLower(trimmed)
			if currentSection != "[events]" {
				if len(preservedSections) > 0 && preservedSections[len(preservedSections)-1] != "" {
					preservedSections = append(preservedSections, "")
				}
				preservedSections = append(preservedSections, trimmed)
				if currentSection == "[script info]" {
					hasScriptInfo = true
				}
				if currentSection == "[v4+ styles]" || currentSection == "[v4 styles]" {
					hasStyles = true
					styleFormat = append([]string(nil), defaultSubtitleExportStyleFormat...)
				}
			}
			continue
		}

		if currentSection == "[events]" {
			if key, value, ok := splitSubtitleExportStyleKeyValue(trimmed); ok && strings.EqualFold(key, "Format") {
				if normalizedEventFormat := strings.TrimSpace(value); normalizedEventFormat != "" {
					eventFormat = normalizedEventFormat
				}
			}
			continue
		}

		if currentSection == "[script info]" {
			if key, _, ok := splitSubtitleExportStyleKeyValue(trimmed); ok {
				switch strings.ToLower(key) {
				case "title":
					if !titleInserted {
						preservedSections = append(preservedSections, fmt.Sprintf("Title: %s", strings.TrimSpace(options.Title)))
						titleInserted = true
					}
					continue
				case "playresx":
					if !playResXInserted {
						preservedSections = append(preservedSections, fmt.Sprintf("PlayResX: %d", options.PlayResX))
						playResXInserted = true
					}
					continue
				case "playresy":
					if !playResYInserted {
						preservedSections = append(preservedSections, fmt.Sprintf("PlayResY: %d", options.PlayResY))
						playResYInserted = true
					}
					continue
				}
			}
		}

		if currentSection == "[v4+ styles]" || currentSection == "[v4 styles]" {
			if key, value, ok := splitSubtitleExportStyleKeyValue(trimmed); ok {
				switch strings.ToLower(key) {
				case "format":
					fields := parseSubtitleExportStyleFormat(value)
					if len(fields) > 0 {
						styleFormat = fields
					}
				case "style":
					style := parseSubtitleExportStyleDefinition(styleFormat, value)
					if strings.TrimSpace(style.Name) != "" {
						styleNames = append(styleNames, style.Name)
						styles[strings.ToLower(style.Name)] = style
					}
				}
			}
		}

		preservedSections = append(preservedSections, rawLine)
	}

	insertScriptInfoOverrides(&preservedSections, options, titleInserted, playResXInserted, playResYInserted)
	return subtitleExportStyleDocument{
		Lines:       normalizeSubtitleExportScriptInfoHeaders(preservedSections),
		EventFormat: eventFormat,
		StyleNames:  styleNames,
		Styles:      styles,
	}, hasScriptInfo, hasStyles
}

func insertScriptInfoOverrides(
	lines *[]string,
	options subtitleExportStyleDocumentOptions,
	titleInserted bool,
	playResXInserted bool,
	playResYInserted bool,
) {
	scriptInfoHeaderIndex := -1
	for index, line := range *lines {
		if strings.EqualFold(strings.TrimSpace(line), "[script info]") {
			scriptInfoHeaderIndex = index
			break
		}
	}
	if scriptInfoHeaderIndex < 0 {
		return
	}
	insertions := make([]string, 0, 3)
	if !titleInserted && strings.TrimSpace(options.Title) != "" {
		insertions = append(insertions, fmt.Sprintf("Title: %s", strings.TrimSpace(options.Title)))
	}
	if !playResXInserted && options.PlayResX > 0 {
		insertions = append(insertions, fmt.Sprintf("PlayResX: %d", options.PlayResX))
	}
	if !playResYInserted && options.PlayResY > 0 {
		insertions = append(insertions, fmt.Sprintf("PlayResY: %d", options.PlayResY))
	}
	if len(insertions) == 0 {
		return
	}
	next := append([]string{}, (*lines)[:scriptInfoHeaderIndex+1]...)
	next = append(next, insertions...)
	next = append(next, (*lines)[scriptInfoHeaderIndex+1:]...)
	*lines = next
}

func normalizeSubtitleExportStyleDocumentContent(content string) string {
	normalized := strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(content, "\r\n", "\n"), "\r", "\n"))
	if normalized == "" {
		return ""
	}
	return normalized + "\n"
}

func normalizeSubtitleExportScriptInfoHeaders(lines []string) []string {
	normalizedLines := make([]string, 0, len(lines))
	aggregatedScriptInfoLines := make([]string, 0, len(lines))
	firstScriptInfoInsertIndex := -1

	for index := 0; index < len(lines); index += 1 {
		line := lines[index]
		if !strings.EqualFold(strings.TrimSpace(line), "[script info]") {
			normalizedLines = append(normalizedLines, line)
			continue
		}
		if firstScriptInfoInsertIndex < 0 {
			firstScriptInfoInsertIndex = len(normalizedLines)
			normalizedLines = append(normalizedLines, "[Script Info]")
		}
		for index += 1; index < len(lines); index += 1 {
			candidate := lines[index]
			trimmedCandidate := strings.TrimSpace(candidate)
			if strings.HasPrefix(trimmedCandidate, "[") && strings.HasSuffix(trimmedCandidate, "]") {
				index -= 1
				break
			}
			aggregatedScriptInfoLines = append(aggregatedScriptInfoLines, candidate)
		}
	}

	if firstScriptInfoInsertIndex < 0 {
		return normalizedLines
	}

	deduplicatedScriptInfoLines := normalizeDuplicateSubtitleExportScriptInfoLines(aggregatedScriptInfoLines)
	next := append([]string{}, normalizedLines[:firstScriptInfoInsertIndex+1]...)
	next = append(next, deduplicatedScriptInfoLines...)
	next = append(next, normalizedLines[firstScriptInfoInsertIndex+1:]...)
	return next
}

func normalizeDuplicateSubtitleExportScriptInfoLines(scriptInfoLines []string) []string {
	normalizedLines := make([]string, 0, len(scriptInfoLines))
	lastHeaderLineIndex := make(map[string]int)

	for index, line := range scriptInfoLines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, ";") {
			continue
		}
		delimiterIndex := strings.Index(trimmed, ":")
		if delimiterIndex <= 0 {
			continue
		}
		headerName := strings.ToLower(strings.TrimSpace(trimmed[:delimiterIndex]))
		if headerName != "" {
			lastHeaderLineIndex[headerName] = index
		}
	}

	for index, line := range scriptInfoLines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, ";") {
			normalizedLines = append(normalizedLines, line)
			continue
		}
		delimiterIndex := strings.Index(trimmed, ":")
		if delimiterIndex <= 0 {
			normalizedLines = append(normalizedLines, line)
			continue
		}
		headerName := strings.ToLower(strings.TrimSpace(trimmed[:delimiterIndex]))
		if lastHeaderLineIndex[headerName] != index {
			continue
		}
		normalizedLines = append(normalizedLines, line)
	}

	return normalizedLines
}

func splitSubtitleExportStyleKeyValue(line string) (string, string, bool) {
	key, value, ok := strings.Cut(line, ":")
	if !ok {
		return "", "", false
	}
	return strings.TrimSpace(key), strings.TrimSpace(value), true
}

func parseSubtitleExportStyleFormat(value string) []string {
	fields := splitSubtitleExportStyleFields(value, 0)
	if len(fields) == 0 {
		return nil
	}
	result := make([]string, 0, len(fields))
	for _, field := range fields {
		normalized := strings.ToLower(strings.TrimSpace(field))
		if normalized != "" {
			result = append(result, normalized)
		}
	}
	return result
}

func splitSubtitleExportStyleFields(value string, expected int) []string {
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

func findSubtitleExportStyleField(format []string, values []string, name string) string {
	normalizedName := strings.ToLower(strings.TrimSpace(name))
	for index, field := range format {
		if field != normalizedName || index >= len(values) {
			continue
		}
		return strings.TrimSpace(values[index])
	}
	return ""
}

func parseSubtitleExportStyleDefinition(format []string, value string) subtitleExportStyle {
	fields := splitSubtitleExportStyleFields(value, len(format))
	style := subtitleExportStyle{
		Name:        findSubtitleExportStyleField(format, fields, "name"),
		FontName:    findSubtitleExportStyleField(format, fields, "fontname"),
		FontSize:    parseSubtitleExportStyleFloat(findSubtitleExportStyleField(format, fields, "fontsize"), 48),
		Bold:        parseSubtitleExportStyleBool(findSubtitleExportStyleField(format, fields, "bold")),
		Italic:      parseSubtitleExportStyleBool(findSubtitleExportStyleField(format, fields, "italic")),
		BorderStyle: parseSubtitleExportStyleInt(findSubtitleExportStyleField(format, fields, "borderstyle"), 1),
		Outline:     parseSubtitleExportStyleFloat(findSubtitleExportStyleField(format, fields, "outline"), 0),
		Shadow:      parseSubtitleExportStyleFloat(findSubtitleExportStyleField(format, fields, "shadow"), 0),
		Alignment:   parseSubtitleExportStyleInt(findSubtitleExportStyleField(format, fields, "alignment"), 2),
		MarginL:     parseSubtitleExportStyleInt(findSubtitleExportStyleField(format, fields, "marginl"), 72),
		MarginR:     parseSubtitleExportStyleInt(findSubtitleExportStyleField(format, fields, "marginr"), 72),
		MarginV:     parseSubtitleExportStyleInt(findSubtitleExportStyleField(format, fields, "marginv"), 56),
	}
	if fontName := strings.TrimSpace(style.FontName); fontName != "" {
		style.FontName = fontName
	} else {
		style.FontName = "Arial"
	}
	if color, ok := parseSubtitleExportASSColor(findSubtitleExportStyleField(format, fields, "primarycolour")); ok {
		style.PrimaryColor = color
	} else {
		style.PrimaryColor = subtitleExportColor{Red: 0xff, Green: 0xff, Blue: 0xff, Alpha: 0xff}
	}
	if color, ok := parseSubtitleExportASSColor(findSubtitleExportStyleField(format, fields, "outlinecolour")); ok {
		style.OutlineColor = color
	} else {
		style.OutlineColor = subtitleExportColor{Red: 0x11, Green: 0x11, Blue: 0x11, Alpha: 0xff}
	}
	if color, ok := parseSubtitleExportASSColor(findSubtitleExportStyleField(format, fields, "backcolour")); ok {
		style.BackColor = color
	}
	return style
}

func parseSubtitleExportStyleInt(value string, fallback int) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return fallback
	}
	return parsed
}

func parseSubtitleExportStyleFloat(value string, fallback float64) float64 {
	parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil || math.IsNaN(parsed) || math.IsInf(parsed, 0) {
		return fallback
	}
	return parsed
}

func parseSubtitleExportStyleBool(value string) bool {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	if trimmed == "" {
		return false
	}
	if trimmed == "true" || trimmed == "yes" {
		return true
	}
	parsed, err := strconv.Atoi(trimmed)
	if err != nil {
		return false
	}
	return parsed != 0
}

func pickSubtitleExportStyleName(styleNames []string, preferred []string, fallbackIndex int) string {
	lowered := make([]string, 0, len(styleNames))
	for _, styleName := range styleNames {
		lowered = append(lowered, strings.ToLower(strings.TrimSpace(styleName)))
	}
	for _, candidate := range preferred {
		normalized := strings.ToLower(strings.TrimSpace(candidate))
		for index, loweredName := range lowered {
			if loweredName == normalized {
				return styleNames[index]
			}
		}
	}
	if fallbackIndex >= 0 && fallbackIndex < len(styleNames) {
		return styleNames[fallbackIndex]
	}
	if len(styleNames) > 0 {
		return styleNames[0]
	}
	return "Default"
}

func resolvePrimarySubtitleExportStyle(document subtitleExportStyleDocument) subtitleExportStyle {
	styleName := pickSubtitleExportStyleName(document.StyleNames, []string{"Primary", "Default"}, 0)
	if style, ok := document.Styles[strings.ToLower(strings.TrimSpace(styleName))]; ok {
		return style
	}
	if style, ok := document.Styles["default"]; ok {
		return style
	}
	for _, styleName := range document.StyleNames {
		if style, ok := document.Styles[strings.ToLower(strings.TrimSpace(styleName))]; ok {
			return style
		}
	}
	return subtitleExportStyle{
		Name:         "Default",
		FontName:     "Arial",
		FontSize:     48,
		PrimaryColor: subtitleExportColor{Red: 0xff, Green: 0xff, Blue: 0xff, Alpha: 0xff},
		OutlineColor: subtitleExportColor{Red: 0x11, Green: 0x11, Blue: 0x11, Alpha: 0xff},
		BorderStyle:  1,
		Alignment:    2,
		MarginL:      72,
		MarginR:      72,
		MarginV:      56,
	}
}

func parseSubtitleExportASSColor(value string) (subtitleExportColor, bool) {
	trimmed := strings.TrimSpace(value)
	trimmed = strings.TrimSuffix(trimmed, "&")
	trimmed = strings.TrimPrefix(strings.ToLower(trimmed), "&h")
	if trimmed == "" {
		return subtitleExportColor{}, false
	}
	if len(trimmed) != 6 && len(trimmed) != 8 {
		return subtitleExportColor{}, false
	}
	parsed, err := strconv.ParseUint(trimmed, 16, 32)
	if err != nil {
		return subtitleExportColor{}, false
	}
	if len(trimmed) == 6 {
		return subtitleExportColor{
			Red:   uint8(parsed & 0xff),
			Green: uint8((parsed >> 8) & 0xff),
			Blue:  uint8((parsed >> 16) & 0xff),
			Alpha: 0xff,
		}, true
	}
	assAlpha := uint8((parsed >> 24) & 0xff)
	return subtitleExportColor{
		Red:   uint8(parsed & 0xff),
		Green: uint8((parsed >> 8) & 0xff),
		Blue:  uint8((parsed >> 16) & 0xff),
		Alpha: 0xff - assAlpha,
	}, true
}

func formatSubtitleExportHexColor(color subtitleExportColor) string {
	return fmt.Sprintf("#%02X%02X%02X", color.Red, color.Green, color.Blue)
}

func formatSubtitleExportLegacySSAColor(color subtitleExportColor) string {
	return fmt.Sprintf("&H00%02X%02X%02X", color.Blue, color.Green, color.Red)
}

func formatSubtitleExportTTMLTextOutline(style subtitleExportStyle) string {
	if style.BorderStyle != 1 || style.Outline <= 0 || style.OutlineColor.Alpha == 0 {
		return ""
	}
	return fmt.Sprintf("%s %spx", formatSubtitleExportHexColor(style.OutlineColor), formatSubtitleExportFloat(style.Outline))
}

func resolveSubtitleExportTextAlign(alignment int) string {
	switch alignment {
	case 1, 4, 7:
		return "left"
	case 3, 6, 9:
		return "right"
	default:
		return "center"
	}
}

func resolveSubtitleExportDisplayAlign(alignment int) string {
	switch alignment {
	case 7, 8, 9:
		return "before"
	case 4, 5, 6:
		return "center"
	default:
		return "after"
	}
}

func formatSubtitleExportFloat(value float64) string {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return "0"
	}
	return strconv.FormatFloat(value, 'f', -1, 64)
}
