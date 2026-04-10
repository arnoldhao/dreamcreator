package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"dreamcreator/internal/application/library/dto"
	"dreamcreator/internal/domain/library"
	"go.uber.org/zap"
)

const (
	subtitleStylePreviewPrimaryText   = "DreamCreator 追创作 0214"
	subtitleStylePreviewSecondaryText = "DreamCreator 追创作 0214"
	subtitleStylePreviewDurationMS    = 60_000

	dcsspFormatName    = "dcssp"
	dcsspSchemaVersion = 1
)

type parsedASSStyleImportDocument struct {
	Format     string
	PlayResX   int
	PlayResY   int
	StyleNames []string
	Styles     map[string]library.AssStyleSpec
}

type previewVTTRenderOptions struct {
	FontMappings  []dto.LibrarySubtitleStyleFontDTO
	PrimaryText   string
	SecondaryText string
	PreviewWidth  int
	PreviewHeight int
}

func (service *LibraryService) GenerateSubtitleStylePreviewASS(_ context.Context, request dto.GenerateSubtitleStylePreviewASSRequest) (dto.GenerateSubtitleStylePreviewASSResult, error) {
	zap.L().Info("subtitle style preview ass request",
		zap.String("type", strings.ToLower(strings.TrimSpace(request.Type))),
		zap.Bool("hasMono", request.Mono != nil),
		zap.Bool("hasBilingual", request.Bilingual != nil),
		zap.Int("fontMappingCount", len(request.FontMappings)),
	)

	switch strings.ToLower(strings.TrimSpace(request.Type)) {
	case "bilingual":
		if request.Bilingual == nil {
			return dto.GenerateSubtitleStylePreviewASSResult{}, fmt.Errorf("bilingual style is required")
		}
		style, err := normalizePreviewBilingualStyle(*request.Bilingual)
		if err != nil {
			return dto.GenerateSubtitleStylePreviewASSResult{}, err
		}
		style = mapPreviewBilingualStyleFontMappings(style, request.FontMappings)
		assContent := buildBilingualStylePreviewASS(style)
		zap.L().Info("subtitle style preview ass generated",
			zap.String("type", "bilingual"),
			zap.String("name", strings.TrimSpace(style.Name)),
			zap.Int("playResX", style.BasePlayResX),
			zap.Int("playResY", style.BasePlayResY),
			zap.String("aspectRatio", style.BaseAspectRatio),
			zap.Int64("previewDurationMS", subtitleStylePreviewDurationMS),
			zap.Int("assLength", len(assContent)),
		)
		return dto.GenerateSubtitleStylePreviewASSResult{
			ASSContent: assContent,
		}, nil
	default:
		if request.Mono == nil {
			return dto.GenerateSubtitleStylePreviewASSResult{}, fmt.Errorf("mono style is required")
		}
		style, err := normalizePreviewMonoStyle(*request.Mono)
		if err != nil {
			return dto.GenerateSubtitleStylePreviewASSResult{}, err
		}
		style = mapPreviewMonoStyleFontMappings(style, request.FontMappings)
		assContent := buildMonoStylePreviewASS(style)
		zap.L().Info("subtitle style preview ass generated",
			zap.String("type", "mono"),
			zap.String("name", strings.TrimSpace(style.Name)),
			zap.Int("playResX", style.BasePlayResX),
			zap.Int("playResY", style.BasePlayResY),
			zap.String("aspectRatio", style.BaseAspectRatio),
			zap.Int64("previewDurationMS", subtitleStylePreviewDurationMS),
			zap.Int("assLength", len(assContent)),
		)
		return dto.GenerateSubtitleStylePreviewASSResult{
			ASSContent: assContent,
		}, nil
	}
}

func (service *LibraryService) GenerateSubtitleStylePreviewVTT(_ context.Context, request dto.GenerateSubtitleStylePreviewVTTRequest) (dto.GenerateSubtitleStylePreviewVTTResult, error) {
	options := previewVTTRenderOptions{
		FontMappings:  append([]dto.LibrarySubtitleStyleFontDTO(nil), request.FontMappings...),
		PrimaryText:   firstNonEmpty(strings.TrimSpace(request.PrimaryText), subtitleStylePreviewPrimaryText),
		SecondaryText: firstNonEmpty(strings.TrimSpace(request.SecondaryText), subtitleStylePreviewSecondaryText),
		PreviewWidth:  request.PreviewWidth,
		PreviewHeight: request.PreviewHeight,
	}

	switch strings.ToLower(strings.TrimSpace(request.Type)) {
	case "bilingual":
		if request.Bilingual == nil {
			return dto.GenerateSubtitleStylePreviewVTTResult{}, fmt.Errorf("bilingual style is required")
		}
		style, err := normalizePreviewBilingualStyle(*request.Bilingual)
		if err != nil {
			return dto.GenerateSubtitleStylePreviewVTTResult{}, err
		}
		vttContent := buildBilingualStylePreviewVTT(style, options)
		return dto.GenerateSubtitleStylePreviewVTTResult{
			VTTContent: vttContent,
		}, nil
	default:
		if request.Mono == nil {
			return dto.GenerateSubtitleStylePreviewVTTResult{}, fmt.Errorf("mono style is required")
		}
		style, err := normalizePreviewMonoStyle(*request.Mono)
		if err != nil {
			return dto.GenerateSubtitleStylePreviewVTTResult{}, err
		}
		vttContent := buildMonoStylePreviewVTT(style, options)
		return dto.GenerateSubtitleStylePreviewVTTResult{
			VTTContent: vttContent,
		}, nil
	}
}

func (service *LibraryService) ParseSubtitleStyleImport(_ context.Context, request dto.ParseSubtitleStyleImportRequest) (dto.ParseSubtitleStyleImportResult, error) {
	format := detectSubtitleStyleImportFormat(request.Format, request.Filename, request.Content)
	switch format {
	case dcsspFormatName:
		return parseDCSSPSubtitleStyleImport(request.Content)
	case "ass", "ssa":
		return parseASSSubtitleStyleImport(request.Content, format)
	default:
		return dto.ParseSubtitleStyleImportResult{}, fmt.Errorf("unsupported subtitle style import format")
	}
}

func (service *LibraryService) ExportSubtitleStylePreset(_ context.Context, request dto.ExportSubtitleStylePresetRequest) (dto.ExportSubtitleStylePresetResult, error) {
	directoryPath := strings.TrimSpace(request.DirectoryPath)
	if directoryPath == "" {
		return dto.ExportSubtitleStylePresetResult{}, fmt.Errorf("directory path is required")
	}
	stat, err := os.Stat(directoryPath)
	if err != nil {
		return dto.ExportSubtitleStylePresetResult{}, err
	}
	if !stat.IsDir() {
		return dto.ExportSubtitleStylePresetResult{}, fmt.Errorf("directory path must be a folder")
	}

	file, err := buildExportableDCSSP(request)
	if err != nil {
		return dto.ExportSubtitleStylePresetResult{}, err
	}

	fileName := sanitizeSubtitleStyleExportFileName(file.Name, file.Type)
	exportPath := filepath.Join(directoryPath, fileName)
	payload, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return dto.ExportSubtitleStylePresetResult{}, err
	}
	payload = append(payload, '\n')
	if err := os.WriteFile(exportPath, payload, 0o644); err != nil {
		return dto.ExportSubtitleStylePresetResult{}, err
	}
	return dto.ExportSubtitleStylePresetResult{
		ExportPath: exportPath,
		FileName:   fileName,
	}, nil
}

func normalizePreviewMonoStyle(value dto.LibraryMonoStyleDTO) (library.MonoStyle, error) {
	config := library.DefaultModuleConfig()
	config.SubtitleStyles.MonoStyles = []library.MonoStyle{toMonoStyles([]dto.LibraryMonoStyleDTO{value})[0]}
	config.SubtitleStyles.BilingualStyles = nil
	config = library.NormalizeModuleConfig(config)
	if len(config.SubtitleStyles.MonoStyles) == 0 {
		return library.MonoStyle{}, fmt.Errorf("mono style is invalid")
	}
	return config.SubtitleStyles.MonoStyles[0], nil
}

func normalizePreviewBilingualStyle(value dto.LibraryBilingualStyleDTO) (library.BilingualStyle, error) {
	config := library.DefaultModuleConfig()
	config.SubtitleStyles.MonoStyles = nil
	config.SubtitleStyles.BilingualStyles = []library.BilingualStyle{toBilingualStyles([]dto.LibraryBilingualStyleDTO{value})[0]}
	config = library.NormalizeModuleConfig(config)
	if len(config.SubtitleStyles.BilingualStyles) == 0 {
		return library.BilingualStyle{}, fmt.Errorf("bilingual style is invalid")
	}
	return config.SubtitleStyles.BilingualStyles[0], nil
}

func buildMonoStylePreviewASS(style library.MonoStyle) string {
	lines := []string{
		"[Script Info]",
		fmt.Sprintf("Title: %s Preview", firstNonEmpty(strings.TrimSpace(style.Name), "Mono Style")),
		"ScriptType: v4.00+",
		"WrapStyle: 0",
		"ScaledBorderAndShadow: yes",
		fmt.Sprintf("PlayResX: %d", style.BasePlayResX),
		fmt.Sprintf("PlayResY: %d", style.BasePlayResY),
	}
	lines = append(lines,
		buildASSStyleMetadataLines([]previewASSStyleMetadata{
			{Name: "Primary", Style: style.Style},
		})...,
	)
	lines = append(lines,
		"",
		"[V4+ Styles]",
		"Format: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour, Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, Encoding",
		formatASSStyleLine("Primary", style.Style),
		"",
		"[Events]",
		"Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text",
		buildPreviewDialogueLine("Primary", 0, subtitleStylePreviewDurationMS, subtitleStylePreviewPrimaryText),
		"",
	)
	return strings.Join(lines, "\n")
}

func buildBilingualStylePreviewASS(style library.BilingualStyle) string {
	primaryStyle, secondaryStyle := resolveBilingualPreviewStylePair(style)
	lines := []string{
		"[Script Info]",
		fmt.Sprintf("Title: %s Preview", firstNonEmpty(strings.TrimSpace(style.Name), "Bilingual Style")),
		"ScriptType: v4.00+",
		"WrapStyle: 0",
		"ScaledBorderAndShadow: yes",
		fmt.Sprintf("PlayResX: %d", style.BasePlayResX),
		fmt.Sprintf("PlayResY: %d", style.BasePlayResY),
	}
	lines = append(lines,
		buildASSStyleMetadataLines([]previewASSStyleMetadata{
			{Name: "Primary", Style: primaryStyle},
			{Name: "Secondary", Style: secondaryStyle},
		})...,
	)
	lines = append(lines,
		"",
		"[V4+ Styles]",
		"Format: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour, Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, Encoding",
		formatASSStyleLine("Primary", primaryStyle),
		formatASSStyleLine("Secondary", secondaryStyle),
		"",
		"[Events]",
		"Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text",
		buildPreviewDialogueLine("Primary", 0, subtitleStylePreviewDurationMS, subtitleStylePreviewPrimaryText),
		buildPreviewDialogueLine("Secondary", 0, subtitleStylePreviewDurationMS, subtitleStylePreviewSecondaryText),
		"",
	)
	return strings.Join(lines, "\n")
}

func buildMonoStylePreviewVTT(style library.MonoStyle, options previewVTTRenderOptions) string {
	primaryText := escapeVTTPreviewText(firstNonEmpty(strings.TrimSpace(options.PrimaryText), subtitleStylePreviewPrimaryText))
	lines := []string{
		"WEBVTT",
		"",
		"STYLE",
		"::cue {",
		"  white-space: pre-line;",
		"}",
	}
	lines = append(lines, buildVTTStyleBlock("mono", style.Style, style.BasePlayResX, style.BasePlayResY, options)...)
	lines = append(lines,
		"",
		fmt.Sprintf(
			"%s --> %s %s",
			formatVTTTimestamp(0),
			formatVTTTimestamp(subtitleStylePreviewDurationMS),
			buildVTTCueSettings(style.Style, style.BasePlayResX, style.BasePlayResY),
		),
		fmt.Sprintf("<c.mono>%s</c>", primaryText),
		"",
	)
	return strings.Join(lines, "\n")
}

func buildBilingualStylePreviewVTT(style library.BilingualStyle, options previewVTTRenderOptions) string {
	primaryStyle, secondaryStyle := resolveBilingualPreviewStylePair(style)
	primaryText := escapeVTTPreviewText(firstNonEmpty(strings.TrimSpace(options.PrimaryText), subtitleStylePreviewPrimaryText))
	secondaryText := escapeVTTPreviewText(firstNonEmpty(strings.TrimSpace(options.SecondaryText), subtitleStylePreviewSecondaryText))
	lines := []string{
		"WEBVTT",
		"",
		"STYLE",
		"::cue {",
		"  white-space: pre-line;",
		"}",
	}
	lines = append(lines, buildVTTStyleBlock("primary", primaryStyle, style.BasePlayResX, style.BasePlayResY, options)...)
	lines = append(lines, buildVTTStyleBlock("secondary", secondaryStyle, style.BasePlayResX, style.BasePlayResY, options)...)
	lines = append(lines,
		"",
		fmt.Sprintf(
			"%s --> %s %s",
			formatVTTTimestamp(0),
			formatVTTTimestamp(subtitleStylePreviewDurationMS),
			buildVTTCueSettings(primaryStyle, style.BasePlayResX, style.BasePlayResY),
		),
		fmt.Sprintf("<c.primary>%s</c>", primaryText),
		"",
		fmt.Sprintf(
			"%s --> %s %s",
			formatVTTTimestamp(0),
			formatVTTTimestamp(subtitleStylePreviewDurationMS),
			buildVTTCueSettings(secondaryStyle, style.BasePlayResX, style.BasePlayResY),
		),
		fmt.Sprintf("<c.secondary>%s</c>", secondaryText),
		"",
	)
	return strings.Join(lines, "\n")
}

func buildVTTStyleBlock(
	className string,
	style library.AssStyleSpec,
	basePlayResX int,
	basePlayResY int,
	options previewVTTRenderOptions,
) []string {
	scale := resolvePreviewVTTScale(basePlayResX, basePlayResY, options.PreviewWidth, options.PreviewHeight)
	fontFamily := resolvePreviewVTTFontFamily(style.Fontname, options.FontMappings)
	fontSize := maxFloat64(1, style.Fontsize*scale.Uniform)
	letterSpacing := style.Spacing * scale.X
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
	if style.BorderStyle == 3 {
		backgroundColor = formatPreviewVTTColor(style.BackColour, "rgba(0, 0, 0, 0)")
	}
	paddingX := 0.0
	paddingY := 0.0
	if style.BorderStyle == 3 {
		paddingX = maxFloat64(4, fontSize*0.28)
		paddingY = maxFloat64(2, fontSize*0.16)
	}
	lines := []string{
		fmt.Sprintf("::cue(.%s) {", className),
		fmt.Sprintf("  font-family: %s;", formatPreviewVTTFontFamily(fontFamily)),
		fmt.Sprintf("  font-size: %s;", formatPreviewVTTLength(fontSize)),
		fmt.Sprintf("  line-height: %s;", formatPreviewVTTLength(fontSize*1.2)),
		fmt.Sprintf("  font-weight: %s;", resolvePreviewVTTFontWeight(style)),
		fmt.Sprintf("  font-style: %s;", boolToString(style.Italic, "italic", "normal")),
		fmt.Sprintf("  text-decoration: %s;", textDecoration),
		fmt.Sprintf("  letter-spacing: %s;", formatPreviewVTTLength(letterSpacing)),
		fmt.Sprintf("  color: %s;", formatPreviewVTTColor(style.PrimaryColour, "rgba(255, 255, 255, 1)")),
		fmt.Sprintf("  background-color: %s;", backgroundColor),
		fmt.Sprintf("  padding: %s %s;", formatPreviewVTTLength(paddingY), formatPreviewVTTLength(paddingX)),
		fmt.Sprintf("  text-shadow: %s;", buildPreviewVTTTextShadow(style, scale.Uniform)),
		"}",
	}
	return lines
}

type previewVTTScale struct {
	Uniform float64
	X       float64
	Y       float64
}

func resolvePreviewVTTScale(basePlayResX int, basePlayResY int, previewWidth int, previewHeight int) previewVTTScale {
	if basePlayResX <= 0 || basePlayResY <= 0 || previewWidth <= 0 || previewHeight <= 0 {
		return previewVTTScale{Uniform: 1, X: 1, Y: 1}
	}
	xScale := float64(previewWidth) / float64(maxInt(1, basePlayResX))
	yScale := float64(previewHeight) / float64(maxInt(1, basePlayResY))
	if math.IsNaN(xScale) || math.IsInf(xScale, 0) || xScale <= 0 {
		xScale = 1
	}
	if math.IsNaN(yScale) || math.IsInf(yScale, 0) || yScale <= 0 {
		yScale = 1
	}
	return previewVTTScale{
		Uniform: math.Max(0.01, math.Min(xScale, yScale)),
		X:       math.Max(0.01, xScale),
		Y:       math.Max(0.01, yScale),
	}
}

func resolvePreviewVTTFontFamily(fontName string, mappings []dto.LibrarySubtitleStyleFontDTO) string {
	normalized := normalizePreviewVTTFontFamilyKey(fontName)
	for _, item := range mappings {
		if !item.Enabled {
			continue
		}
		if normalizePreviewVTTFontFamilyKey(item.Family) != normalized {
			continue
		}
		if systemFamily := strings.TrimSpace(item.SystemFamily); systemFamily != "" {
			return systemFamily
		}
	}
	return firstNonEmpty(strings.TrimSpace(fontName), "sans-serif")
}

func normalizePreviewVTTFontFamilyKey(value string) string {
	return strings.ReplaceAll(strings.ReplaceAll(strings.ToLower(strings.TrimSpace(value)), "_", ""), "-", "")
}

func formatPreviewVTTFontFamily(value string) string {
	safe := strings.TrimSpace(value)
	if safe == "" {
		return "sans-serif"
	}
	escaped := strings.ReplaceAll(safe, `"`, `\"`)
	return fmt.Sprintf("\"%s\", sans-serif", escaped)
}

func buildVTTCueSettings(style library.AssStyleSpec, basePlayResX int, basePlayResY int) string {
	anchorHorizontal := "center"
	anchorVertical := "bottom"
	switch style.Alignment {
	case 1, 4, 7:
		anchorHorizontal = "start"
	case 3, 6, 9:
		anchorHorizontal = "end"
	default:
		anchorHorizontal = "center"
	}
	switch style.Alignment {
	case 7, 8, 9:
		anchorVertical = "top"
	case 4, 5, 6:
		anchorVertical = "middle"
	default:
		anchorVertical = "bottom"
	}

	leftPercent := resolvePreviewPercent(style.MarginL, basePlayResX)
	rightPercent := resolvePreviewPercent(style.MarginR, basePlayResX)
	sizePercent := clampPreviewPercent(100-leftPercent-rightPercent, 10, 100)
	positionPercent := 50.0
	switch anchorHorizontal {
	case "start":
		positionPercent = clampPreviewPercent(leftPercent, 0, 100)
	case "end":
		positionPercent = clampPreviewPercent(100-rightPercent, 0, 100)
	default:
		positionPercent = clampPreviewPercent(leftPercent+sizePercent/2, 0, 100)
	}

	linePercent := 50.0
	switch anchorVertical {
	case "top":
		linePercent = clampPreviewPercent(resolvePreviewPercent(style.MarginV, basePlayResY), 0, 100)
	case "bottom":
		linePercent = clampPreviewPercent(100-resolvePreviewPercent(style.MarginV, basePlayResY), 0, 100)
	default:
		linePercent = 50
	}

	return fmt.Sprintf(
		"line:%s position:%s size:%s align:%s",
		formatPreviewVTTPercent(linePercent),
		formatPreviewVTTPercent(positionPercent),
		formatPreviewVTTPercent(sizePercent),
		anchorHorizontal,
	)
}

func resolvePreviewPercent(value int, total int) float64 {
	if total <= 0 {
		return 0
	}
	return (float64(maxInt(0, value)) / float64(total)) * 100
}

func clampPreviewPercent(value float64, minValue float64, maxValue float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return minValue
	}
	return math.Min(maxValue, math.Max(minValue, value))
}

func formatPreviewVTTPercent(value float64) string {
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.2f", value), "0"), ".") + "%"
}

func formatPreviewVTTColor(value string, fallback string) string {
	color, ok := parseSubtitleExportASSColor(value)
	if !ok {
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

func buildPreviewVTTTextShadow(style library.AssStyleSpec, scale float64) string {
	outline := maxFloat64(0, style.Outline*scale)
	shadow := maxFloat64(0, style.Shadow*scale)
	outlineColor := formatPreviewVTTColor(style.OutlineColour, "rgba(0, 0, 0, 0.85)")
	shadowColor := formatPreviewVTTColor(style.BackColour, "rgba(0, 0, 0, 0.45)")
	layers := make([]string, 0, 10)
	if outline > 0 {
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
		for _, offset := range offsets {
			layers = append(layers, fmt.Sprintf("%spx %spx 0 %s", formatPreviewVTTRaw(offset[0]), formatPreviewVTTRaw(offset[1]), outlineColor))
		}
	}
	if shadow > 0 {
		blur := maxFloat64(1, shadow*1.6)
		layers = append(layers, fmt.Sprintf("%spx %spx %spx %s", formatPreviewVTTRaw(shadow), formatPreviewVTTRaw(shadow), formatPreviewVTTRaw(blur), shadowColor))
	}
	if len(layers) == 0 {
		return "none"
	}
	return strings.Join(layers, ", ")
}

func formatPreviewVTTLength(value float64) string {
	return fmt.Sprintf("%spx", formatPreviewVTTRaw(value))
}

func formatPreviewVTTRaw(value float64) string {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return "0"
	}
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.2f", value), "0"), ".")
}

func escapeVTTPreviewText(value string) string {
	normalized := strings.ReplaceAll(strings.ReplaceAll(strings.TrimSpace(value), "\r\n", "\n"), "\r", "\n")
	normalized = strings.ReplaceAll(normalized, "&", "&amp;")
	normalized = strings.ReplaceAll(normalized, "<", "&lt;")
	normalized = strings.ReplaceAll(normalized, ">", "&gt;")
	return normalized
}

func boolToString(value bool, trueValue string, falseValue string) string {
	if value {
		return trueValue
	}
	return falseValue
}

func maxFloat64(a float64, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func resolveBilingualPreviewStylePair(style library.BilingualStyle) (library.AssStyleSpec, library.AssStyleSpec) {
	anchor := style.Layout.BlockAnchor
	if anchor < 1 || anchor > 9 {
		anchor = 2
	}
	gap := style.Layout.Gap
	if math.IsNaN(gap) || math.IsInf(gap, 0) || gap < 0 {
		gap = 24
	}
	primary := style.Primary.Style
	secondary := style.Secondary.Style
	primaryOffset := int(math.Round(gap + secondary.Fontsize))
	secondaryOffset := int(math.Round(gap + primary.Fontsize))

	switch anchor {
	case 4, 5, 6:
		// For middle anchors, two separate active cues may collide and be auto-relocated by
		// media-captions. Convert to explicit top-anchored offsets around center so the visual
		// block position is stable and matches "middle" semantics.
		topAnchor := anchor + 3
		primary.Alignment = topAnchor
		secondary.Alignment = topAnchor

		playResY := style.BasePlayResY
		if playResY <= 0 {
			playResY = 1080
		}
		blockHeight := primary.Fontsize + secondary.Fontsize + gap
		baseTop := int(math.Round(float64(playResY)/2 - blockHeight/2))
		if baseTop < 0 {
			baseTop = 0
		}

		primary.MarginV = baseTop
		secondary.MarginV = baseTop + secondaryOffset
	case 7, 8, 9:
		primary.Alignment = anchor
		secondary.Alignment = anchor
		secondary.MarginV += secondaryOffset
	default:
		primary.Alignment = anchor
		secondary.Alignment = anchor
		primary.MarginV += primaryOffset
	}
	return primary, secondary
}

func formatASSStyleLine(name string, style library.AssStyleSpec) string {
	return fmt.Sprintf(
		"Style: %s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%d,%s,%s,%d,%d,%d,%d,%d",
		name,
		resolveASSStyleFontName(style),
		formatSubtitleExportFloat(style.Fontsize),
		firstNonEmpty(style.PrimaryColour, "&H00FFFFFF"),
		firstNonEmpty(style.SecondaryColour, "&H00FFFFFF"),
		firstNonEmpty(style.OutlineColour, "&H00111111"),
		firstNonEmpty(style.BackColour, "&HFF111111"),
		formatASSBool(resolveASSStyleBold(style)),
		formatASSBool(resolveASSStyleItalic(style)),
		formatASSBool(style.Underline),
		formatASSBool(style.StrikeOut),
		formatSubtitleExportFloat(style.ScaleX),
		formatSubtitleExportFloat(style.ScaleY),
		formatSubtitleExportFloat(style.Spacing),
		formatSubtitleExportFloat(style.Angle),
		style.BorderStyle,
		formatSubtitleExportFloat(style.Outline),
		formatSubtitleExportFloat(style.Shadow),
		style.Alignment,
		style.MarginL,
		style.MarginR,
		style.MarginV,
		style.Encoding,
	)
}

func formatASSBool(value bool) string {
	if value {
		return "-1"
	}
	return "0"
}

type previewASSStyleMetadata struct {
	Name  string
	Style library.AssStyleSpec
}

func buildASSStyleMetadataLines(values []previewASSStyleMetadata) []string {
	lines := make([]string, 0, len(values)*4)
	for _, value := range values {
		styleName := strings.TrimSpace(value.Name)
		if styleName == "" {
			continue
		}
		fontFamily := strings.TrimSpace(value.Style.Fontname)
		if fontFamily != "" {
			lines = append(lines, fmt.Sprintf("; DCStyle.%s.FontFamily: %s", styleName, fontFamily))
		}
		fontFace := strings.TrimSpace(value.Style.FontFace)
		if fontFace != "" {
			lines = append(lines, fmt.Sprintf("; DCStyle.%s.FontFace: %s", styleName, fontFace))
		}
		if value.Style.FontWeight > 0 {
			lines = append(lines, fmt.Sprintf("; DCStyle.%s.FontWeight: %d", styleName, value.Style.FontWeight))
		}
		if postScript := strings.TrimSpace(value.Style.FontPostScriptName); postScript != "" {
			lines = append(lines, fmt.Sprintf("; DCStyle.%s.FontPostScriptName: %s", styleName, postScript))
		}
	}
	return lines
}

func mapPreviewMonoStyleFontMappings(style library.MonoStyle, mappings []dto.LibrarySubtitleStyleFontDTO) library.MonoStyle {
	style.Style = applyPreviewFontMappingsToStyle(style.Style, mappings)
	return style
}

func mapPreviewBilingualStyleFontMappings(style library.BilingualStyle, mappings []dto.LibrarySubtitleStyleFontDTO) library.BilingualStyle {
	style.Primary.Style = applyPreviewFontMappingsToStyle(style.Primary.Style, mappings)
	style.Secondary.Style = applyPreviewFontMappingsToStyle(style.Secondary.Style, mappings)
	return style
}

func applyPreviewFontMappingsToStyle(style library.AssStyleSpec, mappings []dto.LibrarySubtitleStyleFontDTO) library.AssStyleSpec {
	normalized := normalizePreviewVTTFontFamilyKey(style.Fontname)
	if normalized == "" {
		return style
	}
	for _, item := range mappings {
		if !item.Enabled {
			continue
		}
		if normalizePreviewVTTFontFamilyKey(item.Family) != normalized {
			continue
		}
		systemFamily := strings.TrimSpace(item.SystemFamily)
		if systemFamily == "" {
			return style
		}
		style.Fontname = systemFamily
		style.FontPostScriptName = ""
		return style
	}
	return style
}

func resolvePreviewVTTFontWeight(style library.AssStyleSpec) string {
	if style.FontWeight > 0 {
		return strconv.Itoa(style.FontWeight)
	}
	if style.Bold {
		return "700"
	}
	return "400"
}

func resolveASSStyleFontName(style library.AssStyleSpec) string {
	return resolveASSCompatibleFontName(style.Fontname, "Arial")
}

func resolveASSStyleBold(style library.AssStyleSpec) bool {
	if style.Bold {
		return true
	}
	return resolveASSStyleWeight(style) >= 600
}

func resolveASSStyleItalic(style library.AssStyleSpec) bool {
	if style.Italic {
		return true
	}
	return assFontFaceImpliesItalic(style.FontFace)
}

func resolveASSStyleWeight(style library.AssStyleSpec) int {
	if style.FontWeight > 0 {
		return style.FontWeight
	}
	return deriveSubtitleExportFontWeight(style.FontFace, style.FontPostScriptName, style.Fontname, style.Bold)
}

func buildPreviewDialogueLine(styleName string, startMS int64, endMS int64, text string) string {
	return fmt.Sprintf(
		"Dialogue: 0,%s,%s,%s,,0,0,0,,%s",
		formatAssPreviewTime(startMS),
		formatAssPreviewTime(endMS),
		styleName,
		escapeASSPreviewText(text),
	)
}

func formatAssPreviewTime(value int64) string {
	safe := maxInt64(0, value)
	hours := safe / 3_600_000
	minutes := (safe % 3_600_000) / 60_000
	seconds := (safe % 60_000) / 1000
	centiseconds := (safe % 1000) / 10
	return fmt.Sprintf("%d:%02d:%02d.%02d", hours, minutes, seconds, centiseconds)
}

func escapeASSPreviewText(value string) string {
	normalized := strings.ReplaceAll(strings.ReplaceAll(strings.TrimSpace(value), "\r\n", "\n"), "\r", "\n")
	lines := strings.Split(normalized, "\n")
	for index, line := range lines {
		line = strings.ReplaceAll(line, `\`, `\\`)
		line = strings.ReplaceAll(line, "{", `\{`)
		line = strings.ReplaceAll(line, "}", `\}`)
		lines[index] = line
	}
	return strings.Join(lines, `\N`)
}

func detectSubtitleStyleImportFormat(format string, filename string, content string) string {
	normalized := strings.ToLower(strings.TrimSpace(format))
	switch normalized {
	case "ass", "ssa", dcsspFormatName:
		return normalized
	}
	extension := strings.ToLower(strings.TrimPrefix(filepath.Ext(strings.TrimSpace(filename)), "."))
	switch extension {
	case "ass", "ssa", dcsspFormatName:
		return extension
	}
	trimmed := strings.TrimSpace(content)
	if strings.HasPrefix(trimmed, "{") && strings.Contains(strings.ToLower(trimmed), `"format"`) && strings.Contains(strings.ToLower(trimmed), dcsspFormatName) {
		return dcsspFormatName
	}
	if strings.Contains(strings.ToLower(trimmed), "[v4 styles]") {
		return "ssa"
	}
	return "ass"
}

func parseASSSubtitleStyleImport(content string, importFormat string) (dto.ParseSubtitleStyleImportResult, error) {
	document, err := parseASSStyleImportDocument(content)
	if err != nil {
		return dto.ParseSubtitleStyleImportResult{}, err
	}
	if len(document.StyleNames) == 0 {
		return dto.ParseSubtitleStyleImportResult{}, fmt.Errorf("no ASS styles found")
	}
	detectedRatio := library.ResolveSubtitleStyleAspectRatio(document.PlayResX, document.PlayResY)
	targetPlayResX, targetPlayResY := library.ResolveSubtitleStyleBaseResolution(detectedRatio)
	scale := resolveASSImportScale(document.PlayResX, document.PlayResY, targetPlayResX, targetPlayResY)
	config := library.DefaultModuleConfig()
	config.SubtitleStyles.MonoStyles = make([]library.MonoStyle, 0, len(document.StyleNames))
	for index, styleName := range document.StyleNames {
		styleSpec, ok := document.Styles[strings.ToLower(strings.TrimSpace(styleName))]
		if !ok {
			continue
		}
		config.SubtitleStyles.MonoStyles = append(config.SubtitleStyles.MonoStyles, library.MonoStyle{
			ID:                 fmt.Sprintf("mono-style-%d", index+1),
			Name:               strings.TrimSpace(styleName),
			BasePlayResX:       targetPlayResX,
			BasePlayResY:       targetPlayResY,
			BaseAspectRatio:    detectedRatio,
			SourceAssStyleName: strings.TrimSpace(styleName),
			Style:              scaleImportedASSStyleSpec(styleSpec, scale),
		})
	}
	config = library.NormalizeModuleConfig(config)
	return dto.ParseSubtitleStyleImportResult{
		ImportFormat:       importFormat,
		MonoStyles:         toMonoStyleDTOs(config.SubtitleStyles.MonoStyles),
		DetectedRatio:      detectedRatio,
		NormalizedPlayResX: targetPlayResX,
		NormalizedPlayResY: targetPlayResY,
	}, nil
}

func parseDCSSPSubtitleStyleImport(content string) (dto.ParseSubtitleStyleImportResult, error) {
	var file dto.DCSSPFileDTO
	if err := json.Unmarshal([]byte(content), &file); err != nil {
		return dto.ParseSubtitleStyleImportResult{}, err
	}
	if strings.ToLower(strings.TrimSpace(file.Format)) != dcsspFormatName {
		return dto.ParseSubtitleStyleImportResult{}, fmt.Errorf("invalid dcssp format")
	}
	if file.SchemaVersion == 0 {
		file.SchemaVersion = dcsspSchemaVersion
	}
	switch strings.ToLower(strings.TrimSpace(file.Type)) {
	case "mono":
		var payload dto.DCSSPMonoPayloadDTO
		if err := json.Unmarshal(file.Payload, &payload); err != nil {
			return dto.ParseSubtitleStyleImportResult{}, err
		}
		style, err := normalizePreviewMonoStyle(dto.LibraryMonoStyleDTO{
			ID:              file.ID,
			Name:            file.Name,
			BasePlayResX:    payload.BasePlayResX,
			BasePlayResY:    payload.BasePlayResY,
			BaseAspectRatio: payload.BaseAspectRatio,
			Style:           payload.Style,
		})
		if err != nil {
			return dto.ParseSubtitleStyleImportResult{}, err
		}
		return dto.ParseSubtitleStyleImportResult{
			ImportFormat:       dcsspFormatName,
			DCSSP:              &file,
			MonoStyles:         toMonoStyleDTOs([]library.MonoStyle{style}),
			DetectedRatio:      style.BaseAspectRatio,
			NormalizedPlayResX: style.BasePlayResX,
			NormalizedPlayResY: style.BasePlayResY,
		}, nil
	case "bilingual":
		var payload dto.DCSSPBilingualPayloadDTO
		if err := json.Unmarshal(file.Payload, &payload); err != nil {
			return dto.ParseSubtitleStyleImportResult{}, err
		}
		style, err := normalizePreviewBilingualStyle(dto.LibraryBilingualStyleDTO{
			ID:              file.ID,
			Name:            file.Name,
			BasePlayResX:    payload.BasePlayResX,
			BasePlayResY:    payload.BasePlayResY,
			BaseAspectRatio: payload.BaseAspectRatio,
			Primary:         payload.Primary,
			Secondary:       payload.Secondary,
			Layout:          payload.Layout,
		})
		if err != nil {
			return dto.ParseSubtitleStyleImportResult{}, err
		}
		bilingualStyles := toBilingualStyleDTOs([]library.BilingualStyle{style})
		return dto.ParseSubtitleStyleImportResult{
			ImportFormat:       dcsspFormatName,
			DCSSP:              &file,
			BilingualStyle:     &bilingualStyles[0],
			DetectedRatio:      style.BaseAspectRatio,
			NormalizedPlayResX: style.BasePlayResX,
			NormalizedPlayResY: style.BasePlayResY,
		}, nil
	default:
		return dto.ParseSubtitleStyleImportResult{}, fmt.Errorf("unsupported dcssp type")
	}
}

func parseASSStyleImportDocument(content string) (parsedASSStyleImportDocument, error) {
	normalized := normalizeSubtitleExportStyleDocumentContent(content)
	if normalized == "" {
		return parsedASSStyleImportDocument{}, fmt.Errorf("subtitle style content is empty")
	}
	lines := strings.Split(normalized, "\n")
	result := parsedASSStyleImportDocument{
		Format: "ass",
		Styles: make(map[string]library.AssStyleSpec),
	}
	styleMetadata := make(map[string]subtitleStyleFontMetadata)
	currentSection := ""
	styleFormat := append([]string(nil), defaultSubtitleExportStyleFormat...)
	for _, rawLine := range lines {
		trimmed := strings.TrimSpace(rawLine)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			currentSection = strings.ToLower(trimmed)
			if currentSection == "[v4 styles]" {
				result.Format = "ssa"
			}
			continue
		}
		if currentSection == "[script info]" {
			key, value, ok := splitSubtitleExportStyleKeyValue(trimmed)
			if !ok {
				continue
			}
			collectSubtitleStyleFontMetadata(key, value, styleMetadata)
			switch strings.ToLower(strings.TrimSpace(key)) {
			case "playresx":
				result.PlayResX = parseASSImportInt(value)
			case "playresy":
				result.PlayResY = parseASSImportInt(value)
			case "scripttype":
				if strings.EqualFold(strings.TrimSpace(value), "v4.00") {
					result.Format = "ssa"
				}
			}
			continue
		}
		if currentSection != "[v4+ styles]" && currentSection != "[v4 styles]" {
			continue
		}
		key, value, ok := splitSubtitleExportStyleKeyValue(trimmed)
		if !ok {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(key)) {
		case "format":
			if nextFormat := parseSubtitleExportStyleFormat(value); len(nextFormat) > 0 {
				styleFormat = nextFormat
			}
		case "style":
			name, spec := parseASSImportStyleDefinition(styleFormat, value)
			if name == "" {
				continue
			}
			lowered := strings.ToLower(strings.TrimSpace(name))
			if _, exists := result.Styles[lowered]; exists {
				continue
			}
			result.StyleNames = append(result.StyleNames, name)
			result.Styles[lowered] = spec
		}
	}
	for styleName, metadata := range styleMetadata {
		style, ok := result.Styles[styleName]
		if !ok {
			continue
		}
		result.Styles[styleName] = applyASSStyleFontMetadata(style, metadata)
	}
	return result, nil
}

func parseASSImportStyleDefinition(format []string, value string) (string, library.AssStyleSpec) {
	fields := splitSubtitleExportStyleFields(value, len(format))
	name := findSubtitleExportStyleField(format, fields, "name")
	spec := library.AssStyleSpec{
		Fontname:        findSubtitleExportStyleField(format, fields, "fontname"),
		Fontsize:        parseASSImportFloat(findSubtitleExportStyleField(format, fields, "fontsize")),
		PrimaryColour:   findSubtitleExportStyleField(format, fields, "primarycolour"),
		SecondaryColour: findSubtitleExportStyleField(format, fields, "secondarycolour"),
		OutlineColour:   firstNonEmpty(findSubtitleExportStyleField(format, fields, "outlinecolour"), findSubtitleExportStyleField(format, fields, "tertiarycolour")),
		BackColour:      findSubtitleExportStyleField(format, fields, "backcolour"),
		Bold:            parseSubtitleExportStyleBool(findSubtitleExportStyleField(format, fields, "bold")),
		Italic:          parseSubtitleExportStyleBool(findSubtitleExportStyleField(format, fields, "italic")),
		Underline:       parseSubtitleExportStyleBool(findSubtitleExportStyleField(format, fields, "underline")),
		StrikeOut:       parseSubtitleExportStyleBool(findSubtitleExportStyleField(format, fields, "strikeout")),
		ScaleX:          parseASSImportFloat(findSubtitleExportStyleField(format, fields, "scalex")),
		ScaleY:          parseASSImportFloat(findSubtitleExportStyleField(format, fields, "scaley")),
		Spacing:         parseASSImportFloat(findSubtitleExportStyleField(format, fields, "spacing")),
		Angle:           parseASSImportFloat(findSubtitleExportStyleField(format, fields, "angle")),
		BorderStyle:     parseASSImportInt(findSubtitleExportStyleField(format, fields, "borderstyle")),
		Outline:         parseASSImportFloat(findSubtitleExportStyleField(format, fields, "outline")),
		Shadow:          parseASSImportFloat(findSubtitleExportStyleField(format, fields, "shadow")),
		Alignment:       parseASSImportInt(findSubtitleExportStyleField(format, fields, "alignment")),
		MarginL:         parseASSImportInt(findSubtitleExportStyleField(format, fields, "marginl")),
		MarginR:         parseASSImportInt(findSubtitleExportStyleField(format, fields, "marginr")),
		MarginV:         parseASSImportInt(findSubtitleExportStyleField(format, fields, "marginv")),
		Encoding:        parseASSImportInt(findSubtitleExportStyleField(format, fields, "encoding")),
	}
	return strings.TrimSpace(name), spec
}

func applyASSStyleFontMetadata(style library.AssStyleSpec, metadata subtitleStyleFontMetadata) library.AssStyleSpec {
	if strings.TrimSpace(metadata.FontFamily) != "" {
		style.Fontname = strings.TrimSpace(metadata.FontFamily)
	}
	if strings.TrimSpace(metadata.FontFace) != "" {
		style.FontFace = strings.TrimSpace(metadata.FontFace)
	}
	if metadata.FontWeight > 0 {
		style.FontWeight = metadata.FontWeight
	}
	if strings.TrimSpace(metadata.FontPostScriptName) != "" {
		style.FontPostScriptName = strings.TrimSpace(metadata.FontPostScriptName)
	}
	return style
}

func parseASSImportFloat(value string) float64 {
	parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil || math.IsNaN(parsed) || math.IsInf(parsed, 0) {
		return 0
	}
	return parsed
}

func parseASSImportInt(value string) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0
	}
	return parsed
}

func resolveASSImportScale(sourceX int, sourceY int, targetX int, targetY int) float64 {
	if sourceX <= 0 || sourceY <= 0 || targetX <= 0 || targetY <= 0 {
		return 1
	}
	scaleX := float64(targetX) / float64(sourceX)
	scaleY := float64(targetY) / float64(sourceY)
	if scaleX <= 0 || scaleY <= 0 {
		return 1
	}
	delta := math.Abs(scaleX - scaleY)
	maxScale := math.Max(scaleX, scaleY)
	if maxScale <= 0 {
		return 1
	}
	if delta/maxScale <= 0.02 {
		return (scaleX + scaleY) / 2
	}
	return math.Min(scaleX, scaleY)
}

func scaleImportedASSStyleSpec(value library.AssStyleSpec, scale float64) library.AssStyleSpec {
	if math.IsNaN(scale) || math.IsInf(scale, 0) || scale <= 0 {
		scale = 1
	}
	result := value
	result.Fontsize = scaleImportedASSFontSize(result.Fontsize, scale)
	result.Outline = scaleImportedASSFloat(result.Outline, scale)
	result.Shadow = scaleImportedASSFloat(result.Shadow, scale)
	result.MarginL = scaleImportedASSInt(result.MarginL, scale)
	result.MarginR = scaleImportedASSInt(result.MarginR, scale)
	result.MarginV = scaleImportedASSInt(result.MarginV, scale)
	if result.Spacing != 0 {
		result.Spacing = scaleImportedASSFloat(result.Spacing, scale)
	}
	return result
}

func scaleImportedASSFontSize(value float64, scale float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0
	}
	return math.Round(value * scale)
}

func scaleImportedASSFloat(value float64, scale float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0
	}
	return value * scale
}

func scaleImportedASSInt(value int, scale float64) int {
	if value == 0 {
		return 0
	}
	return int(math.Round(float64(value) * scale))
}

func buildExportableDCSSP(request dto.ExportSubtitleStylePresetRequest) (dto.DCSSPFileDTO, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	switch strings.ToLower(strings.TrimSpace(request.Type)) {
	case "bilingual":
		if request.Bilingual == nil {
			return dto.DCSSPFileDTO{}, fmt.Errorf("bilingual style is required")
		}
		style, err := normalizePreviewBilingualStyle(*request.Bilingual)
		if err != nil {
			return dto.DCSSPFileDTO{}, err
		}
		payload, err := json.Marshal(dto.DCSSPBilingualPayloadDTO{
			BasePlayResX:    style.BasePlayResX,
			BasePlayResY:    style.BasePlayResY,
			BaseAspectRatio: style.BaseAspectRatio,
			Primary:         toMonoStyleSnapshotDTO(style.Primary),
			Secondary:       toMonoStyleSnapshotDTO(style.Secondary),
			Layout:          toBilingualLayoutDTO(style.Layout),
		})
		if err != nil {
			return dto.DCSSPFileDTO{}, err
		}
		return dto.DCSSPFileDTO{
			Format:        dcsspFormatName,
			SchemaVersion: dcsspSchemaVersion,
			Type:          "bilingual",
			ID:            style.ID,
			Name:          style.Name,
			CreatedAt:     now,
			UpdatedAt:     now,
			Payload:       payload,
		}, nil
	default:
		if request.Mono == nil {
			return dto.DCSSPFileDTO{}, fmt.Errorf("mono style is required")
		}
		style, err := normalizePreviewMonoStyle(*request.Mono)
		if err != nil {
			return dto.DCSSPFileDTO{}, err
		}
		payload, err := json.Marshal(dto.DCSSPMonoPayloadDTO{
			BasePlayResX:    style.BasePlayResX,
			BasePlayResY:    style.BasePlayResY,
			BaseAspectRatio: style.BaseAspectRatio,
			Style:           toAssStyleSpecDTO(style.Style),
		})
		if err != nil {
			return dto.DCSSPFileDTO{}, err
		}
		return dto.DCSSPFileDTO{
			Format:        dcsspFormatName,
			SchemaVersion: dcsspSchemaVersion,
			Type:          "mono",
			ID:            style.ID,
			Name:          style.Name,
			CreatedAt:     now,
			UpdatedAt:     now,
			Payload:       payload,
		}, nil
	}
}

func sanitizeSubtitleStyleExportFileName(name string, styleType string) string {
	base := strings.TrimSpace(name)
	if base == "" {
		base = strings.TrimSpace(styleType)
	}
	if base == "" {
		base = "subtitle-style"
	}
	var builder strings.Builder
	lastHyphen := false
	for _, r := range strings.ToLower(base) {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
			lastHyphen = false
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
			lastHyphen = false
		case !lastHyphen:
			builder.WriteRune('-')
			lastHyphen = true
		}
	}
	normalized := strings.Trim(builder.String(), "-")
	if normalized == "" {
		normalized = "subtitle-style"
	}
	return normalized + ".dcssp"
}
