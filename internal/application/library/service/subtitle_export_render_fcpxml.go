package service

import (
	"encoding/xml"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"dreamcreator/internal/application/library/dto"
)

func renderFCPXMLFromSegments(segments []subtitleCueSegment, config *dto.SubtitleExportConfig, styleDocumentContent string) string {
	fcpConfig := dto.SubtitleFCPXMLExportConfig{}
	if config != nil && config.FCPXML != nil {
		fcpConfig = *config.FCPXML
	}
	frameDuration := normalizeFCPXMLFrameDuration(fcpConfig.FrameDuration)
	frameGrid := newFCPXMLFrameGrid(frameDuration)
	normalizedFrameDuration := frameGrid.formatFrames(1)
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
	baseStartMS := startSeconds * 1000
	baseStartFrames := frameGrid.roundMillisecondsToFrames(baseStartMS)
	styleDocument := resolveSubtitleExportStyleDocument(styleDocumentContent, subtitleExportStyleDocumentOptions{})
	_, stylePlayResY := resolveSubtitleExportPlayRes(styleDocument)
	primaryStyle, secondaryStyle, hasSecondaryStyle := resolveSubtitleExportStylePair(styleDocument)
	useSecondaryStyle := subtitleSegmentsUseSecondaryStyle(segments) && hasSecondaryStyle
	totalDurationFrames := int64(1)
	titles := make([]fcpxmlTitle, 0, len(segments))
	hasPrimaryStyleDef := false
	hasSecondaryStyleDef := false
	for _, segment := range segments {
		startFrames, durationFrames := frameGrid.roundMillisecondsRangeToFrames(segment.StartMS, segment.EndMS)
		if useSecondaryStyle && segment.HasSecondary {
			if strings.TrimSpace(segment.PrimaryText) != "" {
				title := buildFCPXMLSubtitleTitle(segment.PrimaryText, primaryStyle, defaultLane, "ts1", !hasPrimaryStyleDef, frameGrid, stylePlayResY, startFrames, durationFrames)
				titles = append(titles, title)
				hasPrimaryStyleDef = true
			}
			if strings.TrimSpace(segment.SecondaryText) != "" {
				title := buildFCPXMLSubtitleTitle(segment.SecondaryText, secondaryStyle, defaultLane+1, "ts2", !hasSecondaryStyleDef, frameGrid, stylePlayResY, startFrames, durationFrames)
				titles = append(titles, title)
				hasSecondaryStyleDef = true
			}
		} else if strings.TrimSpace(segment.Text) != "" {
			title := buildFCPXMLSubtitleTitle(segment.Text, primaryStyle, defaultLane, "ts1", !hasPrimaryStyleDef, frameGrid, stylePlayResY, startFrames, durationFrames)
			titles = append(titles, title)
			hasPrimaryStyleDef = true
		}
		endFrames := startFrames + durationFrames
		if endFrames > totalDurationFrames {
			totalDurationFrames = endFrames
		}
	}
	doc := fcpxmlRoot{
		Version: version,
		Resources: fcpxmlResources{
			Formats: []fcpxmlFormat{{
				ID:            "r1",
				Name:          fmt.Sprintf("FFVideoFormat%dx%d_%s", width, height, sanitizeFCPXMLFormatToken(normalizedFrameDuration)),
				FrameDuration: normalizedFrameDuration,
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
						Duration:    frameGrid.formatFrames(totalDurationFrames),
						Format:      "r1",
						TCStart:     frameGrid.formatFrames(baseStartFrames),
						TCFormat:    "NDF",
						AudioLayout: "stereo",
						AudioRate:   "48k",
						Spine: fcpxmlSpine{
							Gap: fcpxmlGap{
								Name:     "Gap",
								Offset:   "0s",
								Duration: frameGrid.formatFrames(totalDurationFrames),
								Start:    "0s",
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

func buildFCPXMLSubtitleTitle(
	text string,
	style subtitleExportStyle,
	lane int,
	styleRef string,
	includeStyleDef bool,
	frameGrid fcpxmlFrameGrid,
	stylePlayResY int,
	startFrames int64,
	durationFrames int64,
) fcpxmlTitle {
	title := fcpxmlTitle{
		Name:     text,
		Lane:     maxInt(1, lane),
		Offset:   frameGrid.formatFrames(startFrames),
		Ref:      "r2",
		Duration: frameGrid.formatFrames(durationFrames),
		Start:    frameGrid.formatFrames(startFrames),
		Params:   resolveFCPXMLBasicTitleParams(text, style, stylePlayResY),
		Text: &fcpxmlText{
			TextStyle: []fcpxmlTextStyle{{
				Ref:     styleRef,
				Content: text,
			}},
		},
	}
	if includeStyleDef {
		title.TextStyleDef = []fcpxmlTextStyleDef{{
			ID: styleRef,
			TextStyle: &fcpxmlTextStyleAttr{
				Font:            style.FontName,
				FontSize:        formatSubtitleExportFCPXMLScalar(style.FontSize),
				FontFace:        resolveFCPXMLFontFace(style),
				FontColor:       formatSubtitleExportFCPXMLColor(style.PrimaryColor),
				BackgroundColor: resolveFCPXMLBackgroundColor(style),
				Alignment:       resolveSubtitleExportTextAlign(style.Alignment),
				Bold:            boolToFCPXMLFlag(style.Bold),
				Italic:          boolToFCPXMLFlag(style.Italic),
				StrokeColor:     resolveFCPXMLStrokeColor(style),
				StrokeWidth:     resolveFCPXMLStrokeWidth(style),
				ShadowColor:     resolveFCPXMLShadowColor(style),
				ShadowOffset:    resolveFCPXMLShadowOffset(style),
				Kerning:         resolveFCPXMLKerning(style),
				Underline:       boolToFCPXMLFlag(style.Underline),
			},
		}}
	}
	return title
}

func boolToFCPXMLFlag(value bool) int {
	if value {
		return 1
	}
	return 0
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

type fcpxmlFrameGrid struct {
	FrameNumerator   int64
	FrameDenominator int64
	Timebase         int64
}

func newFCPXMLFrameGrid(frameDuration string) fcpxmlFrameGrid {
	numerator, denominator, ok := parseFCPXMLRationalTime(frameDuration)
	if !ok {
		numerator, denominator, _ = parseFCPXMLRationalTime(defaultFCPXMLFrameDuration)
	}
	requiredDivisor := denominator / gcdInt64(numerator, denominator)
	return fcpxmlFrameGrid{
		FrameNumerator:   numerator,
		FrameDenominator: denominator,
		Timebase:         lcmInt64(defaultFCPXMLTimebase, requiredDivisor),
	}
}

func (grid fcpxmlFrameGrid) roundMillisecondsToFrames(milliseconds int64) int64 {
	if milliseconds <= 0 || grid.FrameNumerator <= 0 || grid.FrameDenominator <= 0 {
		return 0
	}
	numerator := milliseconds * grid.FrameDenominator
	denominator := int64(1000) * grid.FrameNumerator
	return divideAndRoundInt64(numerator, denominator)
}

func (grid fcpxmlFrameGrid) roundMillisecondsRangeToFrames(startMS int64, endMS int64) (int64, int64) {
	startFrames := grid.roundMillisecondsToFrames(startMS)
	endFrames := grid.roundMillisecondsToFrames(maxInt64(startMS+1, endMS))
	if endFrames <= startFrames {
		endFrames = startFrames + 1
	}
	return startFrames, endFrames - startFrames
}

func (grid fcpxmlFrameGrid) framesForTimeValue(value string) (int64, bool) {
	if grid.FrameNumerator <= 0 || grid.FrameDenominator <= 0 {
		return 0, false
	}
	numerator, denominator, ok := parseFCPXMLTimeValue(value)
	if !ok || numerator < 0 || denominator <= 0 {
		return 0, false
	}
	return divideAndRoundInt64(numerator*grid.FrameDenominator, denominator*grid.FrameNumerator), true
}

func (grid fcpxmlFrameGrid) snapTimeValue(value string, fallbackFrames int64) string {
	if frames, ok := grid.framesForTimeValue(value); ok {
		return grid.formatFrames(frames)
	}
	return grid.formatFrames(fallbackFrames)
}

func (grid fcpxmlFrameGrid) formatFrames(frames int64) string {
	if frames <= 0 {
		return "0s"
	}
	if grid.Timebase <= 0 {
		grid.Timebase = defaultFCPXMLTimebase
	}
	numerator := frames * grid.FrameNumerator * grid.Timebase / grid.FrameDenominator
	return fmt.Sprintf("%d/%ds", numerator, grid.Timebase)
}

func divideAndRoundInt64(numerator int64, denominator int64) int64 {
	if denominator <= 0 {
		return 0
	}
	if numerator <= 0 {
		return 0
	}
	return (numerator + denominator/2) / denominator
}

func resolveFCPXMLFontFace(style subtitleExportStyle) string {
	if face := strings.TrimSpace(style.FontFace); face != "" {
		return face
	}
	switch {
	case style.Bold && style.Italic:
		return "Bold Italic"
	case style.Bold:
		return "Bold"
	case style.Italic:
		return "Italic"
	default:
		return "Regular"
	}
}

func formatSubtitleExportFCPXMLColor(color subtitleExportColor) string {
	return strings.Join([]string{
		formatSubtitleExportFCPXMLColorComponent(color.Red),
		formatSubtitleExportFCPXMLColorComponent(color.Green),
		formatSubtitleExportFCPXMLColorComponent(color.Blue),
		formatSubtitleExportFCPXMLColorComponent(color.Alpha),
	}, " ")
}

func formatSubtitleExportFCPXMLColorComponent(value uint8) string {
	component := float64(value) / 255
	return strings.TrimRight(strings.TrimRight(strconv.FormatFloat(component, 'f', 6, 64), "0"), ".")
}

func formatSubtitleExportFCPXMLScalar(value float64) string {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return "0"
	}
	rounded := math.Round(value*1000) / 1000
	if rounded == 0 {
		return "0"
	}
	return strings.TrimRight(strings.TrimRight(strconv.FormatFloat(rounded, 'f', 3, 64), "0"), ".")
}

func resolveFCPXMLBasicTitleParams(text string, style subtitleExportStyle, playResY int) []fcpxmlParam {
	params := make([]fcpxmlParam, 0, 4)
	if position := resolveFCPXMLBasicTitlePositionValue(text, style, playResY); position != "" {
		params = append(params, fcpxmlParam{
			Name:  "Position",
			Key:   "9999/999166631/999166633/1/100/101",
			Value: position,
		})
	}
	return append(params, resolveFCPXMLBasicTitleParamsFromAlignment(resolveSubtitleExportTextAlign(style.Alignment))...)
}

func resolveFCPXMLBasicTitleParamsFromAlignment(alignment string) []fcpxmlParam {
	value := resolveFCPXMLBasicTitleAlignmentValue(alignment)
	return []fcpxmlParam{
		{Name: "Flatten", Key: "9999/999166631/999166633/2/351", Value: "1"},
		{Name: "Alignment", Key: "9999/999166631/999166633/2/354/3142713059/401", Value: value},
		{Name: "Alignment", Key: "9999/999166631/999166633/2/354/999169573/401", Value: value},
	}
}

func resolveFCPXMLBasicTitleAlignmentValue(alignment string) string {
	switch strings.ToLower(strings.TrimSpace(alignment)) {
	case "left":
		return "0 (Left)"
	case "right":
		return "2 (Right)"
	default:
		return "1 (Center)"
	}
}

func resolveFCPXMLBasicTitlePositionValue(text string, style subtitleExportStyle, playResY int) string {
	resolvedPlayResY := playResY
	if resolvedPlayResY <= 0 {
		resolvedPlayResY = defaultSubtitleExportResolutionY
	}

	linePercent := 50.0
	correctionPx := resolveFCPXMLBasicTitleCorrectionPx(text, style)
	switch style.Alignment {
	case 7, 8, 9:
		linePercent = clampSubtitleExportPercent(subtitleExportPercent(style.MarginV, resolvedPlayResY), 0, 100)
	case 1, 2, 3:
		linePercent = clampSubtitleExportPercent(100-subtitleExportPercent(style.MarginV, resolvedPlayResY), 0, 100)
	}
	// Apple's bundled Basic Title template uses a fixed 1920x1080 canvas, so its
	// published Position parameter stays in template space instead of scaling with
	// the exported sequence resolution.
	positionY := ((50.0 - linePercent) / 100.0) * float64(fcpxmlBasicTitleTemplateHeight)
	switch style.Alignment {
	case 7, 8, 9:
		positionY -= float64(correctionPx)
	case 1, 2, 3:
		positionY += float64(correctionPx)
	}
	positionY = math.Round(positionY*1000) / 1000
	if math.Abs(positionY) < 0.001 {
		return ""
	}
	return fmt.Sprintf("0 %s", formatSubtitleExportFCPXMLScalar(positionY))
}

func resolveFCPXMLBasicTitleCorrectionPx(text string, style subtitleExportStyle) int {
	lineCount := countSubtitleExportTextLines(text)
	scaleY := style.ScaleY
	if scaleY <= 0 {
		scaleY = 100
	}
	lineHeight := maxFloat64(1, style.FontSize*(scaleY/100)*fcpxmlBasicTitleFontVisualFactor)
	blockHeight := float64(lineCount) * lineHeight
	if style.BorderStyle == 1 && style.Outline > 0 {
		blockHeight += style.Outline * 2
	}
	if style.BorderStyle != 3 && style.Shadow > 0 {
		blockHeight += style.Shadow
	}
	return maxInt(0, int(math.Floor(blockHeight/2)))
}

func countSubtitleExportTextLines(text string) int {
	normalized := normalizeSubtitleText(text)
	if normalized == "" {
		return 1
	}
	return maxInt(1, len(strings.Split(normalized, "\n")))
}

func resolveFCPXMLBackgroundColor(style subtitleExportStyle) string {
	if style.BorderStyle != 3 || style.BackColor.Alpha == 0 {
		return ""
	}
	return formatSubtitleExportFCPXMLColor(style.BackColor)
}

func resolveFCPXMLStrokeColor(style subtitleExportStyle) string {
	if style.BorderStyle != 1 || style.Outline <= 0 || style.OutlineColor.Alpha == 0 {
		return ""
	}
	return formatSubtitleExportFCPXMLColor(style.OutlineColor)
}

func resolveFCPXMLStrokeWidth(style subtitleExportStyle) string {
	if style.BorderStyle != 1 || style.Outline <= 0 || style.OutlineColor.Alpha == 0 {
		return ""
	}
	return formatSubtitleExportFCPXMLScalar(style.Outline)
}

func resolveFCPXMLShadowColor(style subtitleExportStyle) string {
	if style.BorderStyle == 3 || style.Shadow <= 0 || style.BackColor.Alpha == 0 {
		return ""
	}
	return formatSubtitleExportFCPXMLColor(style.BackColor)
}

func resolveFCPXMLShadowOffset(style subtitleExportStyle) string {
	if style.BorderStyle == 3 || style.Shadow <= 0 || style.BackColor.Alpha == 0 {
		return ""
	}
	return fmt.Sprintf("%s %s", formatSubtitleExportFCPXMLScalar(style.Shadow), formatSubtitleExportFCPXMLScalar(315))
}

func resolveFCPXMLKerning(style subtitleExportStyle) string {
	if style.Spacing == 0 {
		return ""
	}
	return formatSubtitleExportFCPXMLScalar(style.Spacing)
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

func parseFCPXMLTimeValue(value string) (int64, int64, bool) {
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
		if errNum != nil || errDen != nil || numerator < 0 || denominator <= 0 {
			return 0, 0, false
		}
		return numerator, denominator, true
	}
	seconds, err := strconv.ParseInt(core, 10, 64)
	if err != nil || seconds < 0 {
		return 0, 0, false
	}
	return seconds, 1, true
}

func parseFCPXMLRationalTime(value string) (int64, int64, bool) {
	numerator, denominator, ok := parseFCPXMLTimeValue(value)
	if !ok || numerator <= 0 || denominator <= 0 {
		return 0, 0, false
	}
	return numerator, denominator, true
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

func lcmInt64(left int64, right int64) int64 {
	if left <= 0 {
		return right
	}
	if right <= 0 {
		return left
	}
	return left / gcdInt64(left, right) * right
}

func serviceTimestampNow() string {
	return time.Now().UTC().Format("2006-01-02 15:04:05 -0700")
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
	Params       []fcpxmlParam        `xml:"param,omitempty"`
	Text         *fcpxmlText          `xml:"text,omitempty"`
	TextStyleDef []fcpxmlTextStyleDef `xml:"text-style-def,omitempty"`
}

type fcpxmlParam struct {
	Name  string `xml:"name,attr"`
	Key   string `xml:"key,attr,omitempty"`
	Value string `xml:"value,attr,omitempty"`
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
	Font            string `xml:"font,attr,omitempty"`
	FontSize        string `xml:"fontSize,attr,omitempty"`
	FontFace        string `xml:"fontFace,attr,omitempty"`
	FontColor       string `xml:"fontColor,attr,omitempty"`
	BackgroundColor string `xml:"backgroundColor,attr,omitempty"`
	Alignment       string `xml:"alignment,attr,omitempty"`
	Bold            int    `xml:"bold,attr,omitempty"`
	Italic          int    `xml:"italic,attr,omitempty"`
	StrokeColor     string `xml:"strokeColor,attr,omitempty"`
	StrokeWidth     string `xml:"strokeWidth,attr,omitempty"`
	ShadowColor     string `xml:"shadowColor,attr,omitempty"`
	ShadowOffset    string `xml:"shadowOffset,attr,omitempty"`
	Kerning         string `xml:"kerning,attr,omitempty"`
	Underline       int    `xml:"underline,attr,omitempty"`
}
