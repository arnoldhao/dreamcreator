package service

import (
	"encoding/xml"
	"fmt"
	"html"
	"math"
	"strconv"
	"strings"
	"time"

	"dreamcreator/internal/application/library/dto"
)

const (
	defaultSubtitleExportResolutionX = 1920
	defaultSubtitleExportResolutionY = 1080
	defaultITTFrameRate              = 30
	defaultITTFrameRateMultiplier    = "1 1"
	defaultFCPXMLFrameDuration       = "1/30s"
	defaultFCPXMLVersion             = "1.11"
	defaultFCPXMLColorSpace          = "1-1-1 (Rec. 709)"
	defaultFCPXMLStartSeconds        = int64(3600)
)

type subtitleCueSegment struct {
	Index   int
	StartMS int64
	EndMS   int64
	Text    string
}

func renderSubtitleContentWithConfig(
	document dto.SubtitleDocument,
	targetFormat string,
	config *dto.SubtitleExportConfig,
	styleDocumentContent string,
) string {
	format := canonicalSubtitleFormat(detectSubtitleFormat(targetFormat, "", document.Format))
	segments := normalizeSubtitleSegments(document)
	switch format {
	case "txt":
		return renderTXTFromSegments(segments)
	case "vtt":
		return renderVTTFromSegments(segments, config)
	case "ass":
		return renderASSFromSegments(segments, config, styleDocumentContent)
	case "ssa":
		return renderSSAFromSegments(segments, config, styleDocumentContent)
	case "itt":
		return renderITTFromSegments(segments, config, styleDocumentContent)
	case "fcpxml":
		return renderFCPXMLFromSegments(segments, config, styleDocumentContent)
	default:
		return renderSRTFromSegments(segments)
	}
}

func canonicalSubtitleFormat(format string) string {
	return normalizeSubtitleFormat(format)
}

func normalizeSubtitleSegments(document dto.SubtitleDocument) []subtitleCueSegment {
	if len(document.Cues) == 0 {
		return nil
	}
	segments := make([]subtitleCueSegment, 0, len(document.Cues))
	for index, cue := range document.Cues {
		startMS, okStart := parseTimestampToMilliseconds(cue.Start)
		if !okStart {
			startMS = int64(index * 2000)
		}
		endMS, okEnd := parseTimestampToMilliseconds(cue.End)
		if !okEnd || endMS <= startMS {
			endMS = startMS + 1000
		}
		segments = append(segments, subtitleCueSegment{
			Index:   index + 1,
			StartMS: startMS,
			EndMS:   endMS,
			Text:    strings.TrimSpace(cue.Text),
		})
	}
	return segments
}

func renderTXTFromSegments(segments []subtitleCueSegment) string {
	lines := make([]string, 0, len(segments))
	for _, segment := range segments {
		if segment.Text != "" {
			lines = append(lines, segment.Text)
		}
	}
	return strings.Join(lines, "\n")
}

func renderSRTFromSegments(segments []subtitleCueSegment) string {
	if len(segments) == 0 {
		return ""
	}
	blocks := make([]string, 0, len(segments))
	for _, segment := range segments {
		blocks = append(blocks, fmt.Sprintf(
			"%d\n%s --> %s\n%s",
			segment.Index,
			formatSRTTimestamp(segment.StartMS),
			formatSRTTimestamp(segment.EndMS),
			segment.Text,
		))
	}
	return strings.Join(blocks, "\n\n")
}

func renderVTTFromSegments(segments []subtitleCueSegment, config *dto.SubtitleExportConfig) string {
	if len(segments) == 0 {
		return "WEBVTT\n"
	}
	var builder strings.Builder
	builder.WriteString("WEBVTT\n")
	if config != nil && config.VTT != nil {
		kind := strings.TrimSpace(config.VTT.Kind)
		language := strings.TrimSpace(config.VTT.Language)
		if kind != "" || language != "" {
			builder.WriteString("\nNOTE")
			if kind != "" {
				builder.WriteString(" kind=")
				builder.WriteString(kind)
			}
			if language != "" {
				builder.WriteString(" language=")
				builder.WriteString(language)
			}
			builder.WriteString("\n")
		}
	}
	builder.WriteString("\n")
	for index, segment := range segments {
		if index > 0 {
			builder.WriteString("\n")
		}
		builder.WriteString(fmt.Sprintf(
			"%s --> %s\n%s\n",
			formatVTTTimestamp(segment.StartMS),
			formatVTTTimestamp(segment.EndMS),
			segment.Text,
		))
	}
	return strings.TrimRight(builder.String(), "\n") + "\n"
}

func renderASSFromSegments(segments []subtitleCueSegment, config *dto.SubtitleExportConfig, styleDocumentContent string) string {
	assConfig := dto.SubtitleASSExportConfig{}
	if config != nil && config.ASS != nil {
		assConfig = *config.ASS
	}
	playResX := assConfig.PlayResX
	if playResX <= 0 {
		playResX = defaultSubtitleExportResolutionX
	}
	playResY := assConfig.PlayResY
	if playResY <= 0 {
		playResY = defaultSubtitleExportResolutionY
	}
	title := strings.TrimSpace(assConfig.Title)
	if title == "" {
		title = "DreamCreator Export"
	}
	styleDocument := resolveSubtitleExportStyleDocument(
		styleDocumentContent,
		subtitleExportStyleDocumentOptions{
			Title:    title,
			PlayResX: playResX,
			PlayResY: playResY,
		},
	)
	primaryStyleName := pickSubtitleExportStyleName(styleDocument.StyleNames, []string{"Primary", "Default"}, 0)
	if strings.TrimSpace(primaryStyleName) == "" {
		primaryStyleName = "Default"
	}
	lines := append([]string{}, styleDocument.Lines...)
	if len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) != "" {
		lines = append(lines, "")
	}
	lines = append(lines, "[Events]", fmt.Sprintf("Format: %s", styleDocument.EventFormat))
	for _, segment := range segments {
		lines = append(lines, fmt.Sprintf(
			"Dialogue: 0,%s,%s,%s,,0,0,0,,%s",
			formatASSTimestamp(segment.StartMS),
			formatASSTimestamp(segment.EndMS),
			primaryStyleName,
			escapeASSText(segment.Text),
		))
	}
	return strings.Join(lines, "\n") + "\n"
}

func renderSSAFromSegments(segments []subtitleCueSegment, config *dto.SubtitleExportConfig, styleDocumentContent string) string {
	assConfig := dto.SubtitleASSExportConfig{}
	if config != nil && config.ASS != nil {
		assConfig = *config.ASS
	}
	playResX := assConfig.PlayResX
	if playResX <= 0 {
		playResX = defaultSubtitleExportResolutionX
	}
	playResY := assConfig.PlayResY
	if playResY <= 0 {
		playResY = defaultSubtitleExportResolutionY
	}
	title := strings.TrimSpace(assConfig.Title)
	if title == "" {
		title = "DreamCreator Export"
	}
	styleDocument := resolveSubtitleExportStyleDocument(
		styleDocumentContent,
		subtitleExportStyleDocumentOptions{
			Title:    title,
			PlayResX: playResX,
			PlayResY: playResY,
		},
	)
	primaryStyle := resolvePrimarySubtitleExportStyle(styleDocument)
	styleName := strings.TrimSpace(primaryStyle.Name)
	if styleName == "" {
		styleName = "Default"
	}
	lines := []string{
		"[Script Info]",
		fmt.Sprintf("Title: %s", title),
		"ScriptType: v4.00",
		fmt.Sprintf("PlayResX: %d", playResX),
		fmt.Sprintf("PlayResY: %d", playResY),
		"",
		"[V4 Styles]",
		"Format: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, TertiaryColour, BackColour, Bold, Italic, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, AlphaLevel, Encoding",
		fmt.Sprintf(
			"Style: %s,%s,%s,%s,%s,0,%s,%d,%d,%d,%s,%s,%d,%d,%d,%d,0,1",
			styleName,
			firstNonEmpty(strings.TrimSpace(primaryStyle.FontName), "Arial"),
			formatSubtitleExportFloat(primaryStyle.FontSize),
			formatSubtitleExportLegacySSAColor(primaryStyle.PrimaryColor),
			formatSubtitleExportLegacySSAColor(primaryStyle.PrimaryColor),
			formatSubtitleExportLegacySSAColor(primaryStyle.BackColor),
			boolToSSAFlag(primaryStyle.Bold),
			boolToSSAFlag(primaryStyle.Italic),
			maxInt(1, primaryStyle.BorderStyle),
			formatSubtitleExportFloat(primaryStyle.Outline),
			formatSubtitleExportFloat(primaryStyle.Shadow),
			maxInt(1, primaryStyle.Alignment),
			maxInt(0, primaryStyle.MarginL),
			maxInt(0, primaryStyle.MarginR),
			maxInt(0, primaryStyle.MarginV),
		),
		"",
		"[Events]",
		"Format: Marked, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text",
	}
	for _, segment := range segments {
		lines = append(lines, fmt.Sprintf(
			"Dialogue: Marked=0,%s,%s,%s,,0,0,0,,%s",
			formatASSTimestamp(segment.StartMS),
			formatASSTimestamp(segment.EndMS),
			styleName,
			escapeASSText(segment.Text),
		))
	}
	return strings.Join(lines, "\n") + "\n"
}

func renderITTFromSegments(segments []subtitleCueSegment, config *dto.SubtitleExportConfig, styleDocumentContent string) string {
	ittConfig := dto.SubtitleITTExportConfig{}
	if config != nil && config.ITT != nil {
		ittConfig = *config.ITT
	}
	frameRate := normalizeITTFrameRate(ittConfig.FrameRate)
	frameRateMultiplier := normalizeITTFrameRateMultiplier(ittConfig.FrameRateMultiplier)
	language := strings.TrimSpace(ittConfig.Language)
	if language == "" {
		language = "en-US"
	}
	styleDocument := resolveSubtitleExportStyleDocument(styleDocumentContent, subtitleExportStyleDocumentOptions{})
	primaryStyle := resolvePrimarySubtitleExportStyle(styleDocument)
	var builder strings.Builder
	builder.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	builder.WriteString(`<tt xmlns="http://www.w3.org/ns/ttml" xmlns:ttp="http://www.w3.org/ns/ttml#parameter" xmlns:tts="http://www.w3.org/ns/ttml#styling" xml:lang="`)
	builder.WriteString(html.EscapeString(language))
	builder.WriteString(`" ttp:timeBase="media" ttp:frameRate="`)
	builder.WriteString(strconv.Itoa(frameRate))
	builder.WriteString(`" ttp:frameRateMultiplier="`)
	builder.WriteString(frameRateMultiplier)
	builder.WriteString(`">` + "\n")
	builder.WriteString("  <head>\n    <styling>\n      <style xml:id=\"s1\"")
	if primaryStyle.FontName != "" {
		builder.WriteString(` tts:fontFamily="`)
		builder.WriteString(html.EscapeString(primaryStyle.FontName))
		builder.WriteString(`"`)
	}
	if primaryStyle.FontSize > 0 {
		builder.WriteString(` tts:fontSize="`)
		builder.WriteString(formatSubtitleExportFloat(primaryStyle.FontSize))
		builder.WriteString(`px"`)
	}
	if primaryStyle.Bold {
		builder.WriteString(` tts:fontWeight="bold"`)
	}
	if primaryStyle.Italic {
		builder.WriteString(` tts:fontStyle="italic"`)
	}
	if primaryStyle.PrimaryColor.Alpha > 0 {
		builder.WriteString(` tts:color="`)
		builder.WriteString(formatSubtitleExportHexColor(primaryStyle.PrimaryColor))
		builder.WriteString(`"`)
	}
	if primaryStyle.BorderStyle == 3 && primaryStyle.BackColor.Alpha > 0 {
		builder.WriteString(` tts:backgroundColor="`)
		builder.WriteString(formatSubtitleExportHexColor(primaryStyle.BackColor))
		builder.WriteString(`"`)
	}
	if textOutline := formatSubtitleExportTTMLTextOutline(primaryStyle); textOutline != "" {
		builder.WriteString(` tts:textOutline="`)
		builder.WriteString(html.EscapeString(textOutline))
		builder.WriteString(`"`)
	}
	builder.WriteString(` tts:textAlign="`)
	builder.WriteString(resolveSubtitleExportTextAlign(primaryStyle.Alignment))
	builder.WriteString(`" tts:displayAlign="`)
	builder.WriteString(resolveSubtitleExportDisplayAlign(primaryStyle.Alignment))
	builder.WriteString(`"/>` + "\n")
	builder.WriteString("    </styling>\n  </head>\n")
	builder.WriteString("  <body style=\"s1\">\n    <div>\n")
	for _, segment := range segments {
		builder.WriteString(`      <p begin="`)
		builder.WriteString(formatVTTTimestamp(segment.StartMS))
		builder.WriteString(`" end="`)
		builder.WriteString(formatVTTTimestamp(segment.EndMS))
		builder.WriteString(`">`)
		writeTTMLParagraphText(&builder, segment.Text)
		builder.WriteString("</p>\n")
	}
	builder.WriteString("    </div>\n  </body>\n</tt>\n")
	return builder.String()
}

func renderFCPXMLFromSegments(segments []subtitleCueSegment, config *dto.SubtitleExportConfig, styleDocumentContent string) string {
	fcpConfig := dto.SubtitleFCPXMLExportConfig{}
	if config != nil && config.FCPXML != nil {
		fcpConfig = *config.FCPXML
	}
	frameDuration := normalizeFCPXMLFrameDuration(fcpConfig.FrameDuration)
	width := fcpConfig.Width
	if width <= 0 {
		width = defaultSubtitleExportResolutionX
	}
	height := fcpConfig.Height
	if height <= 0 {
		height = defaultSubtitleExportResolutionY
	}
	colorSpace := strings.TrimSpace(fcpConfig.ColorSpace)
	if colorSpace == "" {
		colorSpace = defaultFCPXMLColorSpace
	}
	projectName := strings.TrimSpace(fcpConfig.ProjectName)
	if projectName == "" {
		projectName = "DreamCreator Project"
	}
	libraryName := strings.TrimSpace(fcpConfig.LibraryName)
	if libraryName == "" {
		libraryName = projectName + "_Library"
	}
	eventName := strings.TrimSpace(fcpConfig.EventName)
	if eventName == "" {
		eventName = projectName + "_Event"
	}
	version := strings.TrimSpace(fcpConfig.Version)
	if version == "" {
		version = defaultFCPXMLVersion
	}
	defaultLane := fcpConfig.DefaultLane
	if defaultLane == 0 {
		defaultLane = 1
	}
	startSeconds := fcpConfig.StartTimecodeSeconds
	if startSeconds <= 0 {
		startSeconds = defaultFCPXMLStartSeconds
	}
	styleDocument := resolveSubtitleExportStyleDocument(styleDocumentContent, subtitleExportStyleDocumentOptions{})
	primaryStyle := resolvePrimarySubtitleExportStyle(styleDocument)
	startMS := startSeconds * 1000
	totalDurationMS := int64(10000)
	if len(segments) > 0 {
		last := segments[len(segments)-1]
		if last.EndMS > 0 {
			totalDurationMS = last.EndMS
		}
	}
	titles := make([]fcpxmlTitle, 0, len(segments))
	for _, segment := range segments {
		if strings.TrimSpace(segment.Text) == "" {
			continue
		}
		title := fcpxmlTitle{
			Name:     segment.Text,
			Lane:     defaultLane,
			Offset:   formatFCPXMLDuration(startMS + segment.StartMS),
			Ref:      "r2",
			Duration: formatFCPXMLDuration(segment.EndMS - segment.StartMS),
			Start:    fmt.Sprintf("%ds", startSeconds),
			Text: &fcpxmlText{
				TextStyle: []fcpxmlTextStyle{{
					Ref:     "ts1",
					Content: segment.Text,
				}},
			},
		}
		if len(titles) == 0 {
			title.TextStyleDef = []fcpxmlTextStyleDef{{
				ID: "ts1",
				TextStyle: &fcpxmlTextStyleAttr{
					Font:      primaryStyle.FontName,
					FontSize:  formatSubtitleExportFloat(primaryStyle.FontSize),
					Alignment: resolveSubtitleExportTextAlign(primaryStyle.Alignment),
					Bold:      boolToFCPXMLFlag(primaryStyle.Bold),
					Italic:    boolToFCPXMLFlag(primaryStyle.Italic),
				},
			}}
		}
		titles = append(titles, title)
	}
	doc := fcpxmlRoot{
		Version: version,
		Resources: fcpxmlResources{
			Formats: []fcpxmlFormat{{
				ID:            "r1",
				Name:          fmt.Sprintf("FFVideoFormat%dx%d_%s", width, height, sanitizeFCPXMLFormatToken(frameDuration)),
				FrameDuration: frameDuration,
				Width:         width,
				Height:        height,
				ColorSpace:    colorSpace,
			}},
			Effects: []fcpxmlEffect{{
				ID:   "r2",
				Name: "Basic Title",
				UID:  ".../Titles.localized/Bumper:Opener.localized/Basic Title.localized/Basic Title.moti",
			}},
		},
		Library: fcpxmlLibrary{
			Location: fmt.Sprintf("file:///root/Movies/%s.fcpbundle", sanitizeFCPXMLName(libraryName)),
			Events: []fcpxmlEvent{{
				Name: eventName,
				UID:  "event-1",
				Projects: []fcpxmlProject{{
					Name:    projectName,
					UID:     "project-1",
					ModDate: serviceTimestampNow(),
					Sequence: fcpxmlSequence{
						Duration:    formatFCPXMLDuration(totalDurationMS),
						Format:      "r1",
						TCStart:     "0s",
						TCFormat:    "NDF",
						AudioLayout: "stereo",
						AudioRate:   "48k",
						Spine: fcpxmlSpine{
							Gap: fcpxmlGap{
								Name:     "Gap",
								Offset:   "0s",
								Duration: formatFCPXMLDuration(totalDurationMS),
								Start:    fmt.Sprintf("%ds", startSeconds),
								Titles:   titles,
							},
						},
					},
				}},
			}},
		},
	}
	xmlData, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		return renderSRTFromSegments(segments)
	}
	return "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<!DOCTYPE fcpxml>\n\n" + string(xmlData) + "\n"
}

func writeTTMLParagraphText(builder *strings.Builder, text string) {
	normalized := strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\r", "\n")
	lines := strings.Split(normalized, "\n")
	for index, line := range lines {
		if index > 0 {
			builder.WriteString("<br/>")
		}
		builder.WriteString(html.EscapeString(line))
	}
}

func boolToFCPXMLFlag(value bool) int {
	if value {
		return 1
	}
	return 0
}

func boolToSSAFlag(value bool) int {
	if value {
		return -1
	}
	return 0
}

func parseTimestampToMilliseconds(value string) (int64, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, false
	}
	if strings.HasSuffix(trimmed, "s") && strings.Contains(trimmed, "/") {
		fraction := strings.TrimSuffix(trimmed, "s")
		parts := strings.SplitN(fraction, "/", 2)
		if len(parts) != 2 {
			return 0, false
		}
		numerator, errNum := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
		denominator, errDen := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		if errNum != nil || errDen != nil || denominator <= 0 {
			return 0, false
		}
		return int64(math.Round((numerator / denominator) * 1000)), true
	}
	if strings.HasSuffix(trimmed, "s") {
		seconds, err := strconv.ParseFloat(strings.TrimSuffix(trimmed, "s"), 64)
		if err == nil && seconds >= 0 {
			return int64(math.Round(seconds * 1000)), true
		}
	}
	parts := strings.Split(trimmed, ":")
	if len(parts) != 3 {
		return 0, false
	}
	hours, errHour := strconv.Atoi(parts[0])
	minutes, errMinute := strconv.Atoi(parts[1])
	if errHour != nil || errMinute != nil || hours < 0 || minutes < 0 {
		return 0, false
	}
	secondPart := strings.TrimSpace(parts[2])
	separator := ""
	if strings.Contains(secondPart, ",") {
		separator = ","
	} else if strings.Contains(secondPart, ".") {
		separator = "."
	}
	seconds := 0
	milliseconds := 0
	if separator == "" {
		parsedSeconds, err := strconv.Atoi(secondPart)
		if err != nil || parsedSeconds < 0 {
			return 0, false
		}
		seconds = parsedSeconds
	} else {
		chunks := strings.SplitN(secondPart, separator, 2)
		parsedSeconds, err := strconv.Atoi(strings.TrimSpace(chunks[0]))
		if err != nil || parsedSeconds < 0 {
			return 0, false
		}
		seconds = parsedSeconds
		fraction := strings.TrimSpace(chunks[1])
		if fraction == "" {
			milliseconds = 0
		} else {
			for len(fraction) < 3 {
				fraction += "0"
			}
			if len(fraction) > 3 {
				fraction = fraction[:3]
			}
			parsedMillis, err := strconv.Atoi(fraction)
			if err != nil || parsedMillis < 0 {
				return 0, false
			}
			milliseconds = parsedMillis
		}
	}
	total := int64(hours)*3600*1000 + int64(minutes)*60*1000 + int64(seconds)*1000 + int64(milliseconds)
	return total, true
}

func formatSRTTimestamp(ms int64) string {
	ms = maxInt64(0, ms)
	totalSeconds := ms / 1000
	milliseconds := ms % 1000
	hours := totalSeconds / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60
	return fmt.Sprintf("%02d:%02d:%02d,%03d", hours, minutes, seconds, milliseconds)
}

func formatVTTTimestamp(ms int64) string {
	ms = maxInt64(0, ms)
	totalSeconds := ms / 1000
	milliseconds := ms % 1000
	hours := totalSeconds / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60
	return fmt.Sprintf("%02d:%02d:%02d.%03d", hours, minutes, seconds, milliseconds)
}

func formatASSTimestamp(ms int64) string {
	ms = maxInt64(0, ms)
	totalSeconds := ms / 1000
	centiseconds := (ms % 1000) / 10
	hours := totalSeconds / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60
	return fmt.Sprintf("%d:%02d:%02d.%02d", hours, minutes, seconds, centiseconds)
}

func escapeASSText(text string) string {
	escaped := strings.ReplaceAll(text, "\r\n", "\n")
	escaped = strings.ReplaceAll(escaped, "\r", "\n")
	escaped = strings.ReplaceAll(escaped, "\n", `\N`)
	return escaped
}

func formatFCPXMLDuration(ms int64) string {
	if ms <= 0 {
		return "0s"
	}
	numerator := ms
	denominator := int64(1000)
	divisor := gcdInt64(numerator, denominator)
	if divisor > 0 {
		numerator /= divisor
		denominator /= divisor
	}
	if denominator == 1 {
		return fmt.Sprintf("%ds", numerator)
	}
	return fmt.Sprintf("%d/%ds", numerator, denominator)
}

func sanitizeFCPXMLName(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "DreamCreator"
	}
	return strings.ReplaceAll(trimmed, "/", "_")
}

func sanitizeFCPXMLFormatToken(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "1_30s"
	}
	replacer := strings.NewReplacer("/", "_", " ", "_", ".", "_")
	return replacer.Replace(trimmed)
}

func normalizeFCPXMLFrameDuration(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return defaultFCPXMLFrameDuration
	}
	if _, _, ok := parseFCPXMLRationalTime(trimmed); ok {
		return trimmed
	}
	return defaultFCPXMLFrameDuration
}

func parseFCPXMLRationalTime(value string) (int64, int64, bool) {
	trimmed := strings.TrimSpace(value)
	if !strings.HasSuffix(trimmed, "s") {
		return 0, 0, false
	}
	core := strings.TrimSpace(strings.TrimSuffix(trimmed, "s"))
	if core == "" {
		return 0, 0, false
	}
	if strings.Contains(core, "/") {
		parts := strings.SplitN(core, "/", 2)
		numerator, errNum := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64)
		denominator, errDen := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
		if errNum != nil || errDen != nil || numerator <= 0 || denominator <= 0 {
			return 0, 0, false
		}
		return numerator, denominator, true
	}
	seconds, err := strconv.ParseInt(core, 10, 64)
	if err != nil || seconds <= 0 {
		return 0, 0, false
	}
	return seconds, 1, true
}

func normalizeITTFrameRate(value int) int {
	if value <= 0 {
		return defaultITTFrameRate
	}
	return value
}

func normalizeITTFrameRateMultiplier(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return defaultITTFrameRateMultiplier
	}
	normalized := strings.ReplaceAll(trimmed, "/", " ")
	parts := strings.Fields(normalized)
	if len(parts) != 2 {
		return defaultITTFrameRateMultiplier
	}
	numerator, errNum := strconv.ParseInt(parts[0], 10, 64)
	denominator, errDen := strconv.ParseInt(parts[1], 10, 64)
	if errNum != nil || errDen != nil || numerator <= 0 || denominator <= 0 {
		return defaultITTFrameRateMultiplier
	}
	return fmt.Sprintf("%d %d", numerator, denominator)
}

func gcdInt64(left int64, right int64) int64 {
	if left < 0 {
		left = -left
	}
	if right < 0 {
		right = -right
	}
	for right != 0 {
		left, right = right, left%right
	}
	if left == 0 {
		return 1
	}
	return left
}

func serviceTimestampNow() string {
	return time.Now().UTC().Format("2006-01-02 15:04:05 -0700")
}

func maxInt64(left int64, right int64) int64 {
	if left > right {
		return left
	}
	return right
}

func maxInt(left int, right int) int {
	if left > right {
		return left
	}
	return right
}

type fcpxmlRoot struct {
	XMLName   xml.Name        `xml:"fcpxml"`
	Version   string          `xml:"version,attr"`
	Resources fcpxmlResources `xml:"resources"`
	Library   fcpxmlLibrary   `xml:"library"`
}

type fcpxmlResources struct {
	Formats []fcpxmlFormat `xml:"format"`
	Effects []fcpxmlEffect `xml:"effect,omitempty"`
}

type fcpxmlFormat struct {
	ID            string `xml:"id,attr"`
	Name          string `xml:"name,attr,omitempty"`
	FrameDuration string `xml:"frameDuration,attr"`
	Width         int    `xml:"width,attr,omitempty"`
	Height        int    `xml:"height,attr,omitempty"`
	ColorSpace    string `xml:"colorSpace,attr,omitempty"`
}

type fcpxmlEffect struct {
	ID   string `xml:"id,attr"`
	Name string `xml:"name,attr"`
	UID  string `xml:"uid,attr"`
}

type fcpxmlLibrary struct {
	Location string        `xml:"location,attr,omitempty"`
	Events   []fcpxmlEvent `xml:"event"`
}

type fcpxmlEvent struct {
	Name     string          `xml:"name,attr"`
	UID      string          `xml:"uid,attr,omitempty"`
	Projects []fcpxmlProject `xml:"project"`
}

type fcpxmlProject struct {
	Name     string         `xml:"name,attr"`
	UID      string         `xml:"uid,attr,omitempty"`
	ModDate  string         `xml:"modDate,attr,omitempty"`
	Sequence fcpxmlSequence `xml:"sequence"`
}

type fcpxmlSequence struct {
	Duration    string      `xml:"duration,attr"`
	Format      string      `xml:"format,attr"`
	TCStart     string      `xml:"tcStart,attr,omitempty"`
	TCFormat    string      `xml:"tcFormat,attr,omitempty"`
	AudioLayout string      `xml:"audioLayout,attr,omitempty"`
	AudioRate   string      `xml:"audioRate,attr,omitempty"`
	Spine       fcpxmlSpine `xml:"spine"`
}

type fcpxmlSpine struct {
	Gap fcpxmlGap `xml:"gap"`
}

type fcpxmlGap struct {
	Name     string        `xml:"name,attr,omitempty"`
	Offset   string        `xml:"offset,attr,omitempty"`
	Duration string        `xml:"duration,attr,omitempty"`
	Start    string        `xml:"start,attr,omitempty"`
	Titles   []fcpxmlTitle `xml:"title,omitempty"`
}

type fcpxmlTitle struct {
	Name         string               `xml:"name,attr,omitempty"`
	Lane         int                  `xml:"lane,attr,omitempty"`
	Offset       string               `xml:"offset,attr,omitempty"`
	Ref          string               `xml:"ref,attr,omitempty"`
	Duration     string               `xml:"duration,attr,omitempty"`
	Start        string               `xml:"start,attr,omitempty"`
	Text         *fcpxmlText          `xml:"text,omitempty"`
	TextStyleDef []fcpxmlTextStyleDef `xml:"text-style-def,omitempty"`
}

type fcpxmlText struct {
	TextStyle []fcpxmlTextStyle `xml:"text-style,omitempty"`
}

type fcpxmlTextStyle struct {
	Ref     string `xml:"ref,attr,omitempty"`
	Content string `xml:",chardata"`
}

type fcpxmlTextStyleDef struct {
	ID        string               `xml:"id,attr,omitempty"`
	TextStyle *fcpxmlTextStyleAttr `xml:"text-style,omitempty"`
}

type fcpxmlTextStyleAttr struct {
	Font      string `xml:"font,attr,omitempty"`
	FontSize  string `xml:"fontSize,attr,omitempty"`
	Alignment string `xml:"alignment,attr,omitempty"`
	Bold      int    `xml:"bold,attr,omitempty"`
	Italic    int    `xml:"italic,attr,omitempty"`
}
