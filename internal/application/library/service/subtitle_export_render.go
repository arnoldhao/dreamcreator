package service

import (
	"fmt"
	"html"
	"math"
	"strconv"
	"strings"

	"dreamcreator/internal/application/library/dto"
)

const (
	defaultSubtitleExportResolutionX = 1920
	defaultSubtitleExportResolutionY = 1080
	defaultITTFrameRate              = 30
	defaultITTFrameRateMultiplier    = "1 1"
	defaultFCPXMLFrameDuration       = "1/30s"
	defaultFCPXMLTimebase            = int64(60000)
	defaultFCPXMLVersion             = "1.11"
	defaultFCPXMLColorSpace          = "1-1-1 (Rec. 709)"
	defaultFCPXMLStartSeconds        = int64(0)
	fcpxmlBasicTitleTemplateHeight   = 1080
	fcpxmlBasicTitleFontVisualFactor = 0.70
	subtitleExportDisplayModeKey     = "dreamcreatorExportDisplayMode"
	subtitleExportPrimaryTextsKey    = "dreamcreatorExportPrimaryTexts"
	subtitleExportSecondaryTextsKey  = "dreamcreatorExportSecondaryTexts"
)

type subtitleCueSegment struct {
	Index         int
	StartMS       int64
	EndMS         int64
	Text          string
	PrimaryText   string
	SecondaryText string
	HasSecondary  bool
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
		return renderVTTFromSegments(segments, config, styleDocumentContent)
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
	displayMode := normalizeSubtitleExportDisplayMode(subtitleExportMetadataString(document.Metadata, subtitleExportDisplayModeKey))
	primaryTexts := subtitleExportMetadataStringList(document.Metadata, subtitleExportPrimaryTextsKey)
	secondaryTexts := subtitleExportMetadataStringList(document.Metadata, subtitleExportSecondaryTextsKey)
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
		primaryText := normalizeSubtitleText(cue.Text)
		secondaryText := ""
		if displayMode == "bilingual" {
			if index < len(primaryTexts) && strings.TrimSpace(primaryTexts[index]) != "" {
				primaryText = normalizeSubtitleText(primaryTexts[index])
			}
			if index < len(secondaryTexts) && strings.TrimSpace(secondaryTexts[index]) != "" {
				secondaryText = normalizeSubtitleText(secondaryTexts[index])
			}
		}
		text := normalizeSubtitleText(cue.Text)
		if text == "" {
			text = joinSubtitleExportText(primaryText, secondaryText)
		}
		segments = append(segments, subtitleCueSegment{
			Index:         index + 1,
			StartMS:       startMS,
			EndMS:         endMS,
			Text:          text,
			PrimaryText:   primaryText,
			SecondaryText: secondaryText,
			HasSecondary:  strings.TrimSpace(secondaryText) != "",
		})
	}
	return segments
}

func subtitleExportMetadataString(metadata map[string]any, key string) string {
	if len(metadata) == 0 {
		return ""
	}
	value, ok := metadata[key]
	if !ok {
		return ""
	}
	text, _ := value.(string)
	return strings.TrimSpace(text)
}

func subtitleExportMetadataStringList(metadata map[string]any, key string) []string {
	if len(metadata) == 0 {
		return nil
	}
	value, ok := metadata[key]
	if !ok {
		return nil
	}
	switch typed := value.(type) {
	case []string:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			result = append(result, normalizeSubtitleText(item))
		}
		return result
	case []any:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			text, _ := item.(string)
			result = append(result, normalizeSubtitleText(text))
		}
		return result
	default:
		return nil
	}
}

func joinSubtitleExportText(primary string, secondary string) string {
	parts := make([]string, 0, 2)
	if strings.TrimSpace(primary) != "" {
		parts = append(parts, normalizeSubtitleText(primary))
	}
	if strings.TrimSpace(secondary) != "" {
		parts = append(parts, normalizeSubtitleText(secondary))
	}
	return strings.Join(parts, "\n")
}

func normalizeSubtitleExportDisplayMode(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "dual", "bilingual":
		return "bilingual"
	default:
		return "mono"
	}
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

func renderVTTFromSegments(segments []subtitleCueSegment, config *dto.SubtitleExportConfig, styleDocumentContent string) string {
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
	styleDocument := resolveSubtitleExportStyleDocument(styleDocumentContent, subtitleExportStyleDocumentOptions{})
	primaryStyle, secondaryStyle, hasSecondaryStyle := resolveSubtitleExportStylePair(styleDocument)
	hasStyleDocument := strings.TrimSpace(styleDocumentContent) != ""
	if hasStyleDocument {
		builder.WriteString("\nSTYLE\n::cue {\n  white-space: pre-line;\n}\n")
		if subtitleSegmentsUseSecondaryStyle(segments) && hasSecondaryStyle {
			builder.WriteString(strings.Join(buildSubtitleExportVTTStyleBlock("primary", primaryStyle), "\n"))
			builder.WriteString("\n")
			builder.WriteString(strings.Join(buildSubtitleExportVTTStyleBlock("secondary", secondaryStyle), "\n"))
			builder.WriteString("\n")
		} else {
			builder.WriteString(strings.Join(buildSubtitleExportVTTStyleBlock("mono", primaryStyle), "\n"))
			builder.WriteString("\n")
		}
	}
	builder.WriteString("\n")
	firstCueWritten := false
	for _, segment := range segments {
		if hasStyleDocument && segment.HasSecondary && hasSecondaryStyle {
			if strings.TrimSpace(segment.PrimaryText) != "" {
				if firstCueWritten {
					builder.WriteString("\n")
				}
				writeStyledVTTCue(&builder, segment, "primary", segment.PrimaryText, primaryStyle, styleDocument.PlayResX, styleDocument.PlayResY)
				firstCueWritten = true
			}
			if strings.TrimSpace(segment.SecondaryText) != "" {
				if firstCueWritten {
					builder.WriteString("\n")
				}
				writeStyledVTTCue(&builder, segment, "secondary", segment.SecondaryText, secondaryStyle, styleDocument.PlayResX, styleDocument.PlayResY)
				firstCueWritten = true
			}
			continue
		}
		if firstCueWritten {
			builder.WriteString("\n")
		}
		if hasStyleDocument {
			writeStyledVTTCue(&builder, segment, "mono", segment.Text, primaryStyle, styleDocument.PlayResX, styleDocument.PlayResY)
		} else {
			builder.WriteString(fmt.Sprintf(
				"%s --> %s\n%s\n",
				formatVTTTimestamp(segment.StartMS),
				formatVTTTimestamp(segment.EndMS),
				segment.Text,
			))
		}
		firstCueWritten = true
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
	primaryStyleName, secondaryStyleName, hasSecondaryStyle := resolveSubtitleExportStyleNames(styleDocument)
	lines := append([]string{}, styleDocument.Lines...)
	if len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) != "" {
		lines = append(lines, "")
	}
	lines = append(lines, "[Events]", fmt.Sprintf("Format: %s", styleDocument.EventFormat))
	for _, segment := range segments {
		if segment.HasSecondary && hasSecondaryStyle {
			if strings.TrimSpace(segment.PrimaryText) != "" {
				lines = append(lines, fmt.Sprintf(
					"Dialogue: 0,%s,%s,%s,,0,0,0,,%s",
					formatASSTimestamp(segment.StartMS),
					formatASSTimestamp(segment.EndMS),
					primaryStyleName,
					escapeASSText(segment.PrimaryText),
				))
			}
			if strings.TrimSpace(segment.SecondaryText) != "" {
				lines = append(lines, fmt.Sprintf(
					"Dialogue: 0,%s,%s,%s,,0,0,0,,%s",
					formatASSTimestamp(segment.StartMS),
					formatASSTimestamp(segment.EndMS),
					secondaryStyleName,
					escapeASSText(segment.SecondaryText),
				))
			}
			continue
		}
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
	primaryStyle, secondaryStyle, hasSecondaryStyle := resolveSubtitleExportStylePair(styleDocument)
	useSecondaryStyle := subtitleSegmentsUseSecondaryStyle(segments) && hasSecondaryStyle
	styleName := firstNonEmpty(strings.TrimSpace(primaryStyle.Name), "Default")
	secondaryStyleName := firstNonEmpty(strings.TrimSpace(secondaryStyle.Name), styleName)
	lines := []string{
		"[Script Info]",
		fmt.Sprintf("Title: %s", title),
		"ScriptType: v4.00",
		fmt.Sprintf("PlayResX: %d", playResX),
		fmt.Sprintf("PlayResY: %d", playResY),
		"",
		"[V4 Styles]",
		"Format: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, TertiaryColour, BackColour, Bold, Italic, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, AlphaLevel, Encoding",
	}
	lines = append(lines,
		fmt.Sprintf(
			"Style: %s,%s,%s,%s,%s,0,%s,%d,%d,%d,%s,%s,%d,%d,%d,%d,0,1",
			styleName,
			resolveSubtitleExportASSFontName(primaryStyle),
			formatSubtitleExportFloat(primaryStyle.FontSize),
			formatSubtitleExportLegacySSAColor(primaryStyle.PrimaryColor),
			formatSubtitleExportLegacySSAColor(primaryStyle.PrimaryColor),
			formatSubtitleExportLegacySSAColor(primaryStyle.BackColor),
			boolToSSAFlag(resolveSubtitleExportASSBold(primaryStyle)),
			boolToSSAFlag(resolveSubtitleExportASSItalic(primaryStyle)),
			maxInt(1, primaryStyle.BorderStyle),
			formatSubtitleExportFloat(primaryStyle.Outline),
			formatSubtitleExportFloat(primaryStyle.Shadow),
			maxInt(1, primaryStyle.Alignment),
			maxInt(0, primaryStyle.MarginL),
			maxInt(0, primaryStyle.MarginR),
			maxInt(0, primaryStyle.MarginV),
		),
	)
	if useSecondaryStyle {
		lines = append(lines, fmt.Sprintf(
			"Style: %s,%s,%s,%s,%s,0,%s,%d,%d,%d,%s,%s,%d,%d,%d,%d,0,1",
			secondaryStyleName,
			resolveSubtitleExportASSFontName(secondaryStyle),
			formatSubtitleExportFloat(secondaryStyle.FontSize),
			formatSubtitleExportLegacySSAColor(secondaryStyle.PrimaryColor),
			formatSubtitleExportLegacySSAColor(secondaryStyle.PrimaryColor),
			formatSubtitleExportLegacySSAColor(secondaryStyle.BackColor),
			boolToSSAFlag(resolveSubtitleExportASSBold(secondaryStyle)),
			boolToSSAFlag(resolveSubtitleExportASSItalic(secondaryStyle)),
			maxInt(1, secondaryStyle.BorderStyle),
			formatSubtitleExportFloat(secondaryStyle.Outline),
			formatSubtitleExportFloat(secondaryStyle.Shadow),
			maxInt(1, secondaryStyle.Alignment),
			maxInt(0, secondaryStyle.MarginL),
			maxInt(0, secondaryStyle.MarginR),
			maxInt(0, secondaryStyle.MarginV),
		))
	}
	lines = append(lines,
		"",
		"[Events]",
		"Format: Marked, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text",
	)
	for _, segment := range segments {
		if segment.HasSecondary && useSecondaryStyle {
			if strings.TrimSpace(segment.PrimaryText) != "" {
				lines = append(lines, fmt.Sprintf(
					"Dialogue: Marked=0,%s,%s,%s,,0,0,0,,%s",
					formatASSTimestamp(segment.StartMS),
					formatASSTimestamp(segment.EndMS),
					styleName,
					escapeASSText(segment.PrimaryText),
				))
			}
			if strings.TrimSpace(segment.SecondaryText) != "" {
				lines = append(lines, fmt.Sprintf(
					"Dialogue: Marked=0,%s,%s,%s,,0,0,0,,%s",
					formatASSTimestamp(segment.StartMS),
					formatASSTimestamp(segment.EndMS),
					secondaryStyleName,
					escapeASSText(segment.SecondaryText),
				))
			}
			continue
		}
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
	primaryStyle, secondaryStyle, hasSecondaryStyle := resolveSubtitleExportStylePair(styleDocument)
	useSecondaryStyle := subtitleSegmentsUseSecondaryStyle(segments) && hasSecondaryStyle
	var builder strings.Builder
	builder.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	builder.WriteString(`<tt xmlns="http://www.w3.org/ns/ttml" xmlns:ttp="http://www.w3.org/ns/ttml#parameter" xmlns:tts="http://www.w3.org/ns/ttml#styling" xml:lang="`)
	builder.WriteString(html.EscapeString(language))
	builder.WriteString(`" ttp:timeBase="media" ttp:frameRate="`)
	builder.WriteString(strconv.Itoa(frameRate))
	builder.WriteString(`" ttp:frameRateMultiplier="`)
	builder.WriteString(frameRateMultiplier)
	builder.WriteString(`">` + "\n")
	builder.WriteString("  <head>\n    <styling>\n")
	writeITTStyleDefinition(&builder, "s1", primaryStyle)
	if useSecondaryStyle {
		writeITTStyleDefinition(&builder, "s2", secondaryStyle)
	}
	builder.WriteString("    </styling>\n  </head>\n")
	if useSecondaryStyle {
		builder.WriteString("  <body>\n")
	} else {
		builder.WriteString("  <body style=\"s1\">\n")
	}
	builder.WriteString("    <div>\n")
	for _, segment := range segments {
		if useSecondaryStyle && segment.HasSecondary {
			if strings.TrimSpace(segment.PrimaryText) != "" {
				writeITTParagraph(&builder, "s1", segment.StartMS, segment.EndMS, segment.PrimaryText, true)
			}
			if strings.TrimSpace(segment.SecondaryText) != "" {
				writeITTParagraph(&builder, "s2", segment.StartMS, segment.EndMS, segment.SecondaryText, true)
			}
			continue
		}
		writeITTParagraph(&builder, "s1", segment.StartMS, segment.EndMS, segment.Text, useSecondaryStyle)
	}
	builder.WriteString("    </div>\n  </body>\n</tt>\n")
	return builder.String()
}

func resolveSubtitleExportStylePair(document subtitleExportStyleDocument) (subtitleExportStyle, subtitleExportStyle, bool) {
	primary := resolvePrimarySubtitleExportStyle(document)
	secondary, ok := resolveSecondarySubtitleExportStyle(document, primary)
	if !ok {
		return primary, subtitleExportStyle{}, false
	}
	return primary, secondary, true
}

func resolveSubtitleExportStyleNames(document subtitleExportStyleDocument) (string, string, bool) {
	primary, secondary, hasSecondary := resolveSubtitleExportStylePair(document)
	primaryName := firstNonEmpty(strings.TrimSpace(primary.Name), "Default")
	secondaryName := firstNonEmpty(strings.TrimSpace(secondary.Name), primaryName)
	return primaryName, secondaryName, hasSecondary
}

func subtitleSegmentsUseSecondaryStyle(segments []subtitleCueSegment) bool {
	for _, segment := range segments {
		if segment.HasSecondary && strings.TrimSpace(segment.SecondaryText) != "" {
			return true
		}
	}
	return false
}

func resolveSubtitleExportPlayRes(document subtitleExportStyleDocument) (int, int) {
	playResX := document.PlayResX
	if playResX <= 0 {
		playResX = defaultSubtitleExportResolutionX
	}
	playResY := document.PlayResY
	if playResY <= 0 {
		playResY = defaultSubtitleExportResolutionY
	}
	return playResX, playResY
}

func buildSubtitleExportVTTStyleBlock(className string, style subtitleExportStyle) []string {
	textDecoration := "none"
	if style.Underline || style.StrikeOut {
		parts := make([]string, 0, 2)
		if style.Underline {
			parts = append(parts, "underline")
		}
		if style.StrikeOut {
			parts = append(parts, "line-through")
		}
		textDecoration = strings.Join(parts, " ")
	}
	backgroundColor := "transparent"
	if style.BorderStyle == 3 && style.BackColor.Alpha > 0 {
		backgroundColor = formatSubtitleExportVTTRGBA(style.BackColor, "transparent")
	}
	paddingX := "0px"
	paddingY := "0px"
	if style.BorderStyle == 3 {
		paddingX = "8px"
		paddingY = "4px"
	}
	return []string{
		fmt.Sprintf("::cue(.%s) {", className),
		fmt.Sprintf("  font-family: %s;", formatSubtitleExportVTTFontFamily(style.FontName)),
		fmt.Sprintf("  font-size: %s;", formatSubtitleExportVTTLength(maxFloat64(1, style.FontSize))),
		fmt.Sprintf("  line-height: %s;", formatSubtitleExportVTTLength(maxFloat64(1, style.FontSize*1.2))),
		fmt.Sprintf("  font-weight: %s;", resolveSubtitleExportVTTFontWeight(style)),
		fmt.Sprintf("  font-style: %s;", boolToString(style.Italic, "italic", "normal")),
		fmt.Sprintf("  text-decoration: %s;", textDecoration),
		fmt.Sprintf("  letter-spacing: %s;", formatSubtitleExportVTTLength(style.Spacing)),
		fmt.Sprintf("  color: %s;", formatSubtitleExportVTTRGBA(style.PrimaryColor, "rgba(255, 255, 255, 1)")),
		fmt.Sprintf("  background-color: %s;", backgroundColor),
		fmt.Sprintf("  padding: %s %s;", paddingY, paddingX),
		fmt.Sprintf("  text-shadow: %s;", buildSubtitleExportVTTTextShadow(style)),
		"}",
	}
}

func writeStyledVTTCue(builder *strings.Builder, segment subtitleCueSegment, className string, text string, style subtitleExportStyle, playResX int, playResY int) {
	resolvedPlayResX := playResX
	if resolvedPlayResX <= 0 {
		resolvedPlayResX = defaultSubtitleExportResolutionX
	}
	resolvedPlayResY := playResY
	if resolvedPlayResY <= 0 {
		resolvedPlayResY = defaultSubtitleExportResolutionY
	}
	builder.WriteString(formatVTTCueTiming(
		formatVTTTimestamp(segment.StartMS),
		formatVTTTimestamp(segment.EndMS),
		buildSubtitleExportVTTCueSettings(style, resolvedPlayResX, resolvedPlayResY),
	))
	builder.WriteString("\n")
	builder.WriteString(fmt.Sprintf("<c.%s>%s</c>\n", className, escapeVTTStyledCueText(text)))
}

func formatSubtitleExportVTTFontFamily(value string) string {
	safe := strings.TrimSpace(value)
	if safe == "" {
		return "sans-serif"
	}
	escaped := strings.ReplaceAll(safe, `"`, `\"`)
	return fmt.Sprintf("\"%s\", sans-serif", escaped)
}

func formatSubtitleExportVTTLength(value float64) string {
	return fmt.Sprintf("%spx", formatSubtitleExportVTTRaw(value))
}

func formatSubtitleExportVTTRaw(value float64) string {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return "0"
	}
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.2f", value), "0"), ".")
}

func formatSubtitleExportVTTRGBA(color subtitleExportColor, fallback string) string {
	if color.Alpha == 0 && color.Red == 0 && color.Green == 0 && color.Blue == 0 {
		return fallback
	}
	alpha := float64(color.Alpha) / 255
	return fmt.Sprintf(
		"rgba(%d, %d, %d, %s)",
		color.Red,
		color.Green,
		color.Blue,
		strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.3f", alpha), "0"), "."),
	)
}

func buildSubtitleExportVTTTextShadow(style subtitleExportStyle) string {
	outline := maxFloat64(0, style.Outline)
	shadow := maxFloat64(0, style.Shadow)
	layers := make([]string, 0, 10)
	if outline > 0 && style.OutlineColor.Alpha > 0 {
		offsets := [][2]float64{
			{-outline, 0},
			{outline, 0},
			{0, -outline},
			{0, outline},
			{-outline, -outline},
			{outline, -outline},
			{-outline, outline},
			{outline, outline},
		}
		outlineColor := formatSubtitleExportVTTRGBA(style.OutlineColor, "rgba(0, 0, 0, 0.85)")
		for _, offset := range offsets {
			layers = append(layers, fmt.Sprintf("%spx %spx 0 %s", formatSubtitleExportVTTRaw(offset[0]), formatSubtitleExportVTTRaw(offset[1]), outlineColor))
		}
	}
	if shadow > 0 && style.BackColor.Alpha > 0 {
		blur := maxFloat64(1, shadow*1.6)
		shadowColor := formatSubtitleExportVTTRGBA(style.BackColor, "rgba(0, 0, 0, 0.45)")
		layers = append(layers, fmt.Sprintf("%spx %spx %spx %s", formatSubtitleExportVTTRaw(shadow), formatSubtitleExportVTTRaw(shadow), formatSubtitleExportVTTRaw(blur), shadowColor))
	}
	if len(layers) == 0 {
		return "none"
	}
	return strings.Join(layers, ", ")
}

func buildSubtitleExportVTTCueSettings(style subtitleExportStyle, playResX int, playResY int) string {
	anchorHorizontal := "center"
	switch style.Alignment {
	case 1, 4, 7:
		anchorHorizontal = "start"
	case 3, 6, 9:
		anchorHorizontal = "end"
	}
	anchorVertical := "bottom"
	switch style.Alignment {
	case 7, 8, 9:
		anchorVertical = "top"
	case 4, 5, 6:
		anchorVertical = "middle"
	}
	leftPercent := subtitleExportPercent(style.MarginL, playResX)
	rightPercent := subtitleExportPercent(style.MarginR, playResX)
	sizePercent := clampSubtitleExportPercent(100-leftPercent-rightPercent, 10, 100)
	positionPercent := 50.0
	switch anchorHorizontal {
	case "start":
		positionPercent = clampSubtitleExportPercent(leftPercent, 0, 100)
	case "end":
		positionPercent = clampSubtitleExportPercent(100-rightPercent, 0, 100)
	default:
		positionPercent = clampSubtitleExportPercent(leftPercent+sizePercent/2, 0, 100)
	}
	linePercent := 50.0
	switch anchorVertical {
	case "top":
		linePercent = clampSubtitleExportPercent(subtitleExportPercent(style.MarginV, playResY), 0, 100)
	case "bottom":
		linePercent = clampSubtitleExportPercent(100-subtitleExportPercent(style.MarginV, playResY), 0, 100)
	}
	return fmt.Sprintf(
		"line:%s position:%s size:%s align:%s",
		formatSubtitleExportVTTPercent(linePercent),
		formatSubtitleExportVTTPercent(positionPercent),
		formatSubtitleExportVTTPercent(sizePercent),
		anchorHorizontal,
	)
}

func subtitleExportPercent(value int, total int) float64 {
	if total <= 0 {
		return 0
	}
	return (float64(maxInt(0, value)) / float64(total)) * 100
}

func clampSubtitleExportPercent(value float64, minValue float64, maxValue float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return minValue
	}
	return math.Min(maxValue, math.Max(minValue, value))
}

func formatSubtitleExportVTTPercent(value float64) string {
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.2f", value), "0"), ".") + "%"
}

func escapeVTTStyledCueText(value string) string {
	normalized := normalizeSubtitleText(value)
	normalized = strings.ReplaceAll(normalized, "&", "&amp;")
	normalized = strings.ReplaceAll(normalized, "<", "&lt;")
	normalized = strings.ReplaceAll(normalized, ">", "&gt;")
	return normalized
}

func writeITTStyleDefinition(builder *strings.Builder, id string, style subtitleExportStyle) {
	builder.WriteString(`      <style xml:id="`)
	builder.WriteString(html.EscapeString(id))
	builder.WriteString(`"`)
	if fontFamily := resolveSubtitleExportTTMLFontFamily(style); fontFamily != "" {
		builder.WriteString(` tts:fontFamily="`)
		builder.WriteString(html.EscapeString(fontFamily))
		builder.WriteString(`"`)
	}
	if style.FontSize > 0 {
		builder.WriteString(` tts:fontSize="`)
		builder.WriteString(formatSubtitleExportFloat(style.FontSize))
		builder.WriteString(`px"`)
	}
	if resolveSubtitleExportFontWeight(style) >= 700 || style.Bold {
		builder.WriteString(` tts:fontWeight="bold"`)
	}
	if style.Italic {
		builder.WriteString(` tts:fontStyle="italic"`)
	}
	if style.PrimaryColor.Alpha > 0 {
		builder.WriteString(` tts:color="`)
		builder.WriteString(formatSubtitleExportHexColor(style.PrimaryColor))
		builder.WriteString(`"`)
	}
	if style.BorderStyle == 3 && style.BackColor.Alpha > 0 {
		builder.WriteString(` tts:backgroundColor="`)
		builder.WriteString(formatSubtitleExportHexColor(style.BackColor))
		builder.WriteString(`"`)
	}
	if textOutline := formatSubtitleExportTTMLTextOutline(style); textOutline != "" {
		builder.WriteString(` tts:textOutline="`)
		builder.WriteString(html.EscapeString(textOutline))
		builder.WriteString(`"`)
	}
	builder.WriteString(` tts:textAlign="`)
	builder.WriteString(resolveSubtitleExportTextAlign(style.Alignment))
	builder.WriteString(`" tts:displayAlign="`)
	builder.WriteString(resolveSubtitleExportDisplayAlign(style.Alignment))
	builder.WriteString(`"/>` + "\n")
}

func writeITTParagraph(builder *strings.Builder, styleID string, startMS int64, endMS int64, text string, includeStyle bool) {
	builder.WriteString(`      <p`)
	if includeStyle {
		builder.WriteString(` style="`)
		builder.WriteString(html.EscapeString(styleID))
		builder.WriteString(`"`)
	}
	builder.WriteString(` begin="`)
	builder.WriteString(formatVTTTimestamp(startMS))
	builder.WriteString(`" end="`)
	builder.WriteString(formatVTTTimestamp(endMS))
	builder.WriteString(`">`)
	writeTTMLParagraphText(builder, text)
	builder.WriteString("</p>\n")
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

func resolveSubtitleExportASSFontName(style subtitleExportStyle) string {
	return resolveASSCompatibleFontName(style.FontName, "Arial")
}

func resolveSubtitleExportTTMLFontFamily(style subtitleExportStyle) string {
	if postScript := strings.TrimSpace(style.FontPostScriptName); postScript != "" {
		return postScript
	}
	return strings.TrimSpace(style.FontName)
}

func resolveSubtitleExportFontWeight(style subtitleExportStyle) int {
	if style.FontWeight > 0 {
		return style.FontWeight
	}
	if style.Bold {
		return 700
	}
	return 400
}

func resolveSubtitleExportVTTFontWeight(style subtitleExportStyle) string {
	return strconv.Itoa(resolveSubtitleExportFontWeight(style))
}

func resolveSubtitleExportASSBold(style subtitleExportStyle) bool {
	if style.Bold {
		return true
	}
	return resolveSubtitleExportASSWeight(style) >= 600
}

func resolveSubtitleExportASSItalic(style subtitleExportStyle) bool {
	if style.Italic {
		return true
	}
	return assFontFaceImpliesItalic(style.FontFace)
}

func resolveSubtitleExportASSWeight(style subtitleExportStyle) int {
	if style.FontWeight > 0 {
		return style.FontWeight
	}
	return deriveSubtitleExportFontWeight(style.FontFace, style.FontPostScriptName, style.FontName, style.Bold)
}

func resolveASSCompatibleFontName(fontName string, fallback string) string {
	trimmedName := strings.TrimSpace(fontName)
	if trimmedName == "" {
		return fallback
	}
	return trimmedName
}

func assFontFaceImpliesItalic(fontFace string) bool {
	normalized := normalizeASSFontIdentityValue(fontFace)
	return strings.Contains(normalized, "italic") || strings.Contains(normalized, "oblique")
}

func normalizeASSFontIdentityValue(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = strings.ReplaceAll(normalized, "-", " ")
	normalized = strings.Join(strings.Fields(normalized), " ")
	return normalized
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
