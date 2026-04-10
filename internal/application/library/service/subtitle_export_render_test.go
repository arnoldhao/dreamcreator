package service

import (
	"strings"
	"testing"

	"dreamcreator/internal/application/library/dto"
)

const testSubtitleExportStyleDocument = `[Script Info]
Title: Test Subtitle Style
ScriptType: v4.00+
PlayResX: 1920
PlayResY: 1080

[V4+ Styles]
Format: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour, Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, Encoding
Style: Primary,Helvetica,60,&H00FFFFFF,&H00FFFFFF,&H00111111,&H00000000,-1,-1,0,0,100,100,0,0,1,2,0,3,72,72,56,1

[Events]
Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text
`

const testBilingualSubtitleExportStyleDocument = `[Script Info]
Title: Test Bilingual Subtitle Style
ScriptType: v4.00+
PlayResX: 1920
PlayResY: 1080

[V4+ Styles]
Format: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour, Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, Encoding
Style: Primary,Helvetica,60,&H00FFFFFF,&H00FFFFFF,&H00111111,&H00000000,-1,-1,0,0,100,100,0,0,1,2,0,3,72,72,56,1
Style: Secondary,Hiragino Sans,48,&H0000FFFF,&H0000FFFF,&H00111111,&H00000000,0,0,0,0,100,100,0,0,1,1,0,8,96,96,180,1

[Events]
Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text
`

const testRichFCPXMLSubtitleExportStyleDocument = `[Script Info]
Title: Rich FCPXML Subtitle Style
ScriptType: v4.00+
PlayResX: 1920
PlayResY: 1080

[V4+ Styles]
Format: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour, Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, Encoding
Style: Primary,Remem-Regular,58,&H00FFFFFF,&H00FFFFFF,&H00101010,&H80000000,0,0,-1,0,100,100,1.5555,0,1,2.6666,0.87654,2,72,72,56,1

[Events]
Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text
`

const testFCPXMLFontFaceMetadataStyleDocument = `[Script Info]
Title: FCPXML Font Face Metadata
ScriptType: v4.00+
PlayResX: 1920
PlayResY: 1080
; DCStyle.Primary.FontFamily: PingFang SC
; DCStyle.Primary.FontFace: Semibold
; DCStyle.Primary.FontWeight: 600
; DCStyle.Primary.FontPostScriptName: PingFangSC-Semibold

[V4+ Styles]
Format: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour, Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, Encoding
Style: Primary,PingFang SC,52,&H00FFFFFF,&H00FFFFFF,&H00111111,&H00000000,0,0,0,0,100,100,0,0,1,2,0,2,72,72,56,1

[Events]
Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text
`

func TestRenderSubtitleContentWithConfigFCPXML(t *testing.T) {
	t.Parallel()

	document := dto.SubtitleDocument{
		Format: "srt",
		Cues: []dto.SubtitleCue{
			{Index: 1, Start: "00:00:01,000", End: "00:00:02,500", Text: "Hello FCPXML"},
			{Index: 2, Start: "00:00:03,000", End: "00:00:04,250", Text: "Second line\nwrapped"},
		},
	}
	config := &dto.SubtitleExportConfig{
		FCPXML: &dto.SubtitleFCPXMLExportConfig{
			FrameDuration:        "1001/30000s",
			Width:                3840,
			Height:               2160,
			ColorSpace:           "Rec. 2020",
			Version:              "1.12",
			LibraryName:          "Demo/Library",
			EventName:            "DemoEvent",
			ProjectName:          "DemoProject",
			DefaultLane:          2,
			StartTimecodeSeconds: 7200,
		},
	}

	content := renderSubtitleContentWithConfig(document, "fcpxml", config, testSubtitleExportStyleDocument)
	root := parseRenderedFCPXML(t, content)
	if root.Version != "1.12" {
		t.Fatalf("expected configured fcpxml version, got %#v", root)
	}

	if len(root.Resources.Formats) != 1 {
		t.Fatalf("expected one fcpxml format resource, got %#v", root.Resources)
	}
	format := root.Resources.Formats[0]
	if format.Width != 3840 || format.Height != 2160 {
		t.Fatalf("expected configured resolution in fcpxml, got %#v", format)
	}
	frameGrid := newFCPXMLFrameGrid("1001/30000s")
	if format.FrameDuration != frameGrid.formatFrames(1) {
		t.Fatalf("expected configured frame duration, got %#v", format)
	}
	if format.ColorSpace != "Rec. 2020" {
		t.Fatalf("expected configured color space, got %#v", format)
	}

	if root.Library.Location != "file:///root/Movies/Demo_Library.fcpbundle" {
		t.Fatalf("expected sanitized library path, got %#v", root.Library)
	}
	if len(root.Library.Events) != 1 {
		t.Fatalf("expected one fcpxml event, got %#v", root.Library)
	}
	event := root.Library.Events[0]
	if event.Name != "DemoEvent" {
		t.Fatalf("expected configured event name, got %#v", event)
	}
	if len(event.Projects) != 1 {
		t.Fatalf("expected one fcpxml project, got %#v", event)
	}
	project := event.Projects[0]
	if project.Name != "DemoProject" {
		t.Fatalf("expected configured project name, got %#v", project)
	}

	baseStartFrames := frameGrid.roundMillisecondsToFrames(7200 * 1000)
	firstStartFrames, firstDurationFrames := frameGrid.roundMillisecondsRangeToFrames(1000, 2500)
	secondStartFrames, secondDurationFrames := frameGrid.roundMillisecondsRangeToFrames(3000, 4250)
	expectedSequenceDuration := frameGrid.formatFrames(maxInt64(
		firstStartFrames+firstDurationFrames,
		secondStartFrames+secondDurationFrames,
	))

	sequence := project.Sequence
	if sequence.Duration != expectedSequenceDuration || sequence.Format != "r1" {
		t.Fatalf("expected sequence duration/format to match rendered cues, got %#v", sequence)
	}
	if sequence.TCStart != frameGrid.formatFrames(baseStartFrames) {
		t.Fatalf("expected generated fcpxml sequence tcStart to match configured start timecode, got %#v", sequence)
	}
	requireFCPXMLTimeOnFrameBoundary(t, format.FrameDuration, sequence.TCStart)
	requireFCPXMLTimeOnFrameBoundary(t, format.FrameDuration, sequence.Duration)
	gap := sequence.Spine.Gap
	if gap.Offset != "0s" || gap.Start != "0s" || gap.Duration != expectedSequenceDuration {
		t.Fatalf("expected configured gap timing, got %#v", gap)
	}
	requireFCPXMLTimeOnFrameBoundary(t, format.FrameDuration, gap.Offset)
	requireFCPXMLTimeOnFrameBoundary(t, format.FrameDuration, gap.Duration)
	if len(gap.Titles) != 2 {
		t.Fatalf("expected two subtitle titles in fcpxml, got %#v", gap.Titles)
	}

	first := gap.Titles[0]
	if first.Lane != 2 || first.Offset != frameGrid.formatFrames(firstStartFrames) || first.Duration != frameGrid.formatFrames(firstDurationFrames) || first.Start != frameGrid.formatFrames(firstStartFrames) {
		t.Fatalf("expected first title timing/lane to match config, got %#v", first)
	}
	requireFCPXMLTimeOnFrameBoundary(t, format.FrameDuration, first.Offset)
	requireFCPXMLTimeOnFrameBoundary(t, format.FrameDuration, first.Duration)
	requireFCPXMLTimeOnFrameBoundary(t, format.FrameDuration, first.Start)
	if fcpxmlTitleText(first) != "Hello FCPXML" {
		t.Fatalf("expected first title text, got %#v", first)
	}
	if len(first.Params) != 4 {
		t.Fatalf("expected first title to include basic title params, got %#v", first.Params)
	}
	if first.Params[0].Name != "Position" || first.Params[0].Value != "0 -461" {
		t.Fatalf("expected first title to preserve bottom placement, got %#v", first.Params)
	}
	if first.Params[1].Name != "Flatten" || first.Params[1].Value != "1" {
		t.Fatalf("expected first basic title flatten param, got %#v", first.Params)
	}
	if first.Params[2].Name != "Alignment" || first.Params[2].Value != "2 (Right)" {
		t.Fatalf("expected first basic title alignment params, got %#v", first.Params)
	}
	if len(first.TextStyleDef) != 1 || first.TextStyleDef[0].TextStyle == nil {
		t.Fatalf("expected first title style definition, got %#v", first)
	}
	if style := first.TextStyleDef[0].TextStyle; style.Font != "Helvetica" || style.FontSize != "60" || style.FontFace != "Bold Italic" || style.FontColor != "1 1 1 1" || style.Alignment != "right" || style.Bold != 1 || style.Italic != 1 || style.StrokeColor != "0.066667 0.066667 0.066667 1" || style.StrokeWidth != "2" {
		t.Fatalf("expected fcpxml text style to inherit subtitle style font/emphasis, got %#v", style)
	}

	second := gap.Titles[1]
	if second.Lane != 2 || second.Offset != frameGrid.formatFrames(secondStartFrames) || second.Duration != frameGrid.formatFrames(secondDurationFrames) || second.Start != frameGrid.formatFrames(secondStartFrames) {
		t.Fatalf("expected second title timing/lane to match config, got %#v", second)
	}
	requireFCPXMLTimeOnFrameBoundary(t, format.FrameDuration, second.Offset)
	requireFCPXMLTimeOnFrameBoundary(t, format.FrameDuration, second.Duration)
	requireFCPXMLTimeOnFrameBoundary(t, format.FrameDuration, second.Start)
	if fcpxmlTitleText(second) != "Second line\nwrapped" {
		t.Fatalf("expected multiline title text to be preserved, got %#v", second)
	}
	if len(second.TextStyleDef) != 0 {
		t.Fatalf("expected only first title to carry shared text-style-def, got %#v", second)
	}
	if len(second.Params) == 0 || second.Params[0].Name != "Position" || second.Params[0].Value != "0 -440" {
		t.Fatalf("expected multiline title to apply a larger bottom correction, got %#v", second.Params)
	}

	roundTrip := parseSubtitleDocument(content, "fcpxml")
	if len(roundTrip.Cues) != 2 {
		t.Fatalf("expected round-trip fcpxml parse to return two cues, got %#v", roundTrip.Cues)
	}
	firstStartMS, _ := parseTimestampToMilliseconds(frameGrid.formatFrames(firstStartFrames))
	firstEndMS, _ := parseTimestampToMilliseconds(frameGrid.formatFrames(firstStartFrames + firstDurationFrames))
	secondStartMS, _ := parseTimestampToMilliseconds(frameGrid.formatFrames(secondStartFrames))
	secondEndMS, _ := parseTimestampToMilliseconds(frameGrid.formatFrames(secondStartFrames + secondDurationFrames))
	if roundTrip.Cues[0].Start != formatVTTTimestamp(firstStartMS) || roundTrip.Cues[0].End != formatVTTTimestamp(firstEndMS) {
		t.Fatalf("expected first round-trip cue timing to stay accurate, got %#v", roundTrip.Cues[0])
	}
	if roundTrip.Cues[1].Start != formatVTTTimestamp(secondStartMS) || roundTrip.Cues[1].End != formatVTTTimestamp(secondEndMS) || roundTrip.Cues[1].Text != "Second line\nwrapped" {
		t.Fatalf("expected second round-trip cue to stay accurate, got %#v", roundTrip.Cues[1])
	}
}

func TestRenderSubtitleContentWithConfigFCPXMLUsesLargerTimebaseWhen60000IsNotExact(t *testing.T) {
	t.Parallel()

	document := dto.SubtitleDocument{
		Format: "srt",
		Cues: []dto.SubtitleCue{
			{Index: 1, Start: "00:00:01,000", End: "00:00:02,000", Text: "23.976 cue"},
		},
	}
	content := renderSubtitleContentWithConfig(document, "fcpxml", &dto.SubtitleExportConfig{
		FCPXML: &dto.SubtitleFCPXMLExportConfig{
			FrameDuration: "1001/24000s",
		},
	}, testSubtitleExportStyleDocument)

	root := parseRenderedFCPXML(t, content)
	if len(root.Resources.Formats) != 1 {
		t.Fatalf("expected one fcpxml format resource, got %#v", root.Resources)
	}
	format := root.Resources.Formats[0]
	frameGrid := newFCPXMLFrameGrid("1001/24000s")
	if format.FrameDuration != frameGrid.formatFrames(1) {
		t.Fatalf("expected 23.976 frame duration to use the smallest exact shared timebase, got %#v", format)
	}
	if format.FrameDuration != "5005/120000s" {
		t.Fatalf("expected 23.976 frame duration to avoid lossy 60000 denominator, got %#v", format)
	}
	gap := root.Library.Events[0].Projects[0].Sequence.Spine.Gap
	if len(gap.Titles) != 1 {
		t.Fatalf("expected one title cue, got %#v", gap.Titles)
	}
	requireFCPXMLTimeOnFrameBoundary(t, format.FrameDuration, gap.Titles[0].Offset)
	requireFCPXMLTimeOnFrameBoundary(t, format.FrameDuration, gap.Titles[0].Duration)
	requireFCPXMLTimeOnFrameBoundary(t, format.FrameDuration, gap.Titles[0].Start)
}

func TestRenderSubtitleContentWithConfigVTT(t *testing.T) {
	t.Parallel()

	document := dto.SubtitleDocument{
		Format: "srt",
		Cues: []dto.SubtitleCue{
			{Index: 1, Start: "00:00:01,000", End: "00:00:02,000", Text: "Hello VTT"},
		},
	}
	config := &dto.SubtitleExportConfig{
		VTT: &dto.SubtitleVTTExportConfig{
			Kind:     "captions",
			Language: "en-US",
		},
	}

	content := renderSubtitleContentWithConfig(document, "vtt", config, testSubtitleExportStyleDocument)
	if !strings.Contains(content, "STYLE") {
		t.Fatalf("expected vtt export to include STYLE block, got %q", content)
	}
	if !strings.Contains(content, `::cue(.mono)`) {
		t.Fatalf("expected vtt export to include mono cue style selector, got %q", content)
	}
	if !strings.Contains(content, `font-family: "Helvetica", sans-serif;`) {
		t.Fatalf("expected vtt export to inherit font family, got %q", content)
	}
	parsed := parseVTTDocument(content)
	if len(parsed.Cues) != 1 {
		t.Fatalf("expected one vtt cue, got %#v", parsed.Cues)
	}
	if !strings.Contains(parsed.Cues[0].Settings, "align:end") {
		t.Fatalf("expected vtt cue settings to inherit alignment, got %#v", parsed.Cues[0])
	}
	if parsed.Cues[0].Text != "<c.mono>Hello VTT</c>" {
		t.Fatalf("expected vtt cue text to carry mono style class, got %#v", parsed.Cues[0])
	}
}

func TestRenderSubtitleContentWithConfigFCPXMLBilingual(t *testing.T) {
	t.Parallel()

	document := dto.SubtitleDocument{
		Format: "srt",
		Cues: []dto.SubtitleCue{
			{Index: 1, Start: "00:00:01,000", End: "00:00:02,500", Text: "Primary line\nSecondary line"},
		},
		Metadata: map[string]any{
			subtitleExportDisplayModeKey:    "dual",
			subtitleExportPrimaryTextsKey:   []string{"Primary line"},
			subtitleExportSecondaryTextsKey: []string{"Secondary line"},
		},
	}

	content := renderSubtitleContentWithConfig(document, "fcpxml", &dto.SubtitleExportConfig{
		FCPXML: &dto.SubtitleFCPXMLExportConfig{
			DefaultLane: 3,
		},
	}, testBilingualSubtitleExportStyleDocument)
	root := parseRenderedFCPXML(t, content)
	gap := root.Library.Events[0].Projects[0].Sequence.Spine.Gap
	if len(gap.Titles) != 2 {
		t.Fatalf("expected bilingual fcpxml export to render two title nodes, got %#v", gap.Titles)
	}
	if gap.Titles[0].Lane != 3 || gap.Titles[1].Lane != 4 {
		t.Fatalf("expected bilingual fcpxml export to allocate adjacent lanes, got %#v", gap.Titles)
	}
	if fcpxmlTitleText(gap.Titles[0]) != "Primary line" || fcpxmlTitleText(gap.Titles[1]) != "Secondary line" {
		t.Fatalf("expected bilingual fcpxml text to stay split by style, got %#v", gap.Titles)
	}
	if len(gap.Titles[0].TextStyleDef) != 1 || len(gap.Titles[1].TextStyleDef) != 1 {
		t.Fatalf("expected bilingual fcpxml export to emit distinct text-style definitions, got %#v", gap.Titles)
	}
	if style := gap.Titles[1].TextStyleDef[0].TextStyle; style == nil || style.Font != "Hiragino Sans" || style.FontSize != "48" || style.Alignment != "center" {
		t.Fatalf("expected bilingual fcpxml secondary style to inherit the secondary subtitle style, got %#v", gap.Titles[1].TextStyleDef)
	}
	if len(gap.Titles[1].Params) == 0 || gap.Titles[1].Params[0].Name != "Position" || gap.Titles[1].Params[0].Value != "0 343" {
		t.Fatalf("expected bilingual secondary style to preserve top placement, got %#v", gap.Titles[1].Params)
	}
}

func TestRenderSubtitleContentWithConfigFCPXMLRichStyleAttrs(t *testing.T) {
	t.Parallel()

	document := dto.SubtitleDocument{
		Format: "srt",
		Cues: []dto.SubtitleCue{
			{Index: 1, Start: "00:00:01,000", End: "00:00:02,000", Text: "Rich style"},
		},
	}

	content := renderSubtitleContentWithConfig(document, "fcpxml", &dto.SubtitleExportConfig{
		FCPXML: &dto.SubtitleFCPXMLExportConfig{
			Width:  1920,
			Height: 1080,
		},
	}, testRichFCPXMLSubtitleExportStyleDocument)

	root := parseRenderedFCPXML(t, content)
	title := root.Library.Events[0].Projects[0].Sequence.Spine.Gap.Titles[0]
	if len(title.Params) == 0 || title.Params[0].Name != "Position" || title.Params[0].Value != "0 -461" {
		t.Fatalf("expected rich style export to preserve bottom placement, got %#v", title.Params)
	}
	if len(title.TextStyleDef) != 1 || title.TextStyleDef[0].TextStyle == nil {
		t.Fatalf("expected rich style export to carry one style definition, got %#v", title)
	}
	style := title.TextStyleDef[0].TextStyle
	if style.Font != "Remem-Regular" || style.FontSize != "58" || style.FontColor != "1 1 1 1" {
		t.Fatalf("expected rich style export to inherit base font attrs, got %#v", style)
	}
	if style.StrokeColor != "0.062745 0.062745 0.062745 1" || style.StrokeWidth != "2.667" {
		t.Fatalf("expected rich style export to map outline to stroke attrs, got %#v", style)
	}
	if style.ShadowColor != "0 0 0 0.498039" || style.ShadowOffset != "0.877 315" {
		t.Fatalf("expected rich style export to map ass shadow attrs, got %#v", style)
	}
	if style.Kerning != "1.556" || style.Underline != 1 {
		t.Fatalf("expected rich style export to map spacing/underline, got %#v", style)
	}
}

func TestRenderSubtitleContentWithConfigFCPXMLUsesFontFaceMetadata(t *testing.T) {
	t.Parallel()

	document := dto.SubtitleDocument{
		Format: "srt",
		Cues: []dto.SubtitleCue{
			{Index: 1, Start: "00:00:01,000", End: "00:00:02,000", Text: "PingFang face"},
		},
	}

	content := renderSubtitleContentWithConfig(document, "fcpxml", &dto.SubtitleExportConfig{
		FCPXML: &dto.SubtitleFCPXMLExportConfig{
			Width:  1920,
			Height: 1080,
		},
	}, testFCPXMLFontFaceMetadataStyleDocument)

	root := parseRenderedFCPXML(t, content)
	style := root.Library.Events[0].Projects[0].Sequence.Spine.Gap.Titles[0].TextStyleDef[0].TextStyle
	if style == nil {
		t.Fatalf("expected font face metadata export to include text style definition")
	}
	if style.Font != "PingFang SC" || style.FontFace != "Semibold" || style.Bold != 0 {
		t.Fatalf("expected fcpxml export to preserve font family + font face metadata, got %#v", style)
	}
}

func TestRenderSubtitleContentWithConfigASSUsesFontFaceNameAndDerivedFlags(t *testing.T) {
	t.Parallel()

	document := dto.SubtitleDocument{
		Format: "srt",
		Cues: []dto.SubtitleCue{
			{Index: 1, Start: "00:00:01,000", End: "00:00:02,000", Text: "PingFang face"},
		},
	}

	content := renderSubtitleContentWithConfig(document, "ass", &dto.SubtitleExportConfig{
		ASS: &dto.SubtitleASSExportConfig{
			PlayResX: 1920,
			PlayResY: 1080,
		},
	}, testFCPXMLFontFaceMetadataStyleDocument)

	if got := requireASSStyleField(t, content, "[V4+ Styles]", "fontname"); got != "PingFang SC" {
		t.Fatalf("expected ass export to use font family name, got %q", got)
	}
	if got := requireASSStyleField(t, content, "[V4+ Styles]", "bold"); got != "-1" {
		t.Fatalf("expected semibold face to map to bold ass flag, got %q", got)
	}
	if !strings.Contains(content, "; DCStyle.Primary.FontFamily: PingFang SC") {
		t.Fatalf("expected ass export to keep private style metadata commented, got %q", content)
	}
}

func TestFormatSubtitleExportFCPXMLScalar(t *testing.T) {
	t.Parallel()

	testCases := map[float64]string{
		58:                 "58",
		2.6666:             "2.667",
		1.5555:             "1.556",
		315:                "315",
		-483.9999999999999: "-484",
	}
	for input, expected := range testCases {
		if got := formatSubtitleExportFCPXMLScalar(input); got != expected {
			t.Fatalf("expected FCPXML scalar %v to format as %q, got %q", input, expected, got)
		}
	}
}

func TestResolveFCPXMLBasicTitleCorrectionPx(t *testing.T) {
	t.Parallel()

	style := subtitleExportStyle{
		FontSize:    52,
		ScaleY:      100,
		BorderStyle: 1,
	}

	if got := resolveFCPXMLBasicTitleCorrectionPx("Single line", style); got != 18 {
		t.Fatalf("expected single-line correction to floor to 18px, got %d", got)
	}
	if got := resolveFCPXMLBasicTitleCorrectionPx("First line\nSecond line", style); got != 36 {
		t.Fatalf("expected multiline correction to scale with line count, got %d", got)
	}
}

func TestRenderSubtitleContentWithConfigASS(t *testing.T) {
	t.Parallel()

	document := dto.SubtitleDocument{
		Format: "vtt",
		Cues: []dto.SubtitleCue{
			{Index: 1, Start: "00:00:01.000", End: "00:00:02.500", Text: "Line A\nLine B"},
			{Index: 2, Start: "00:00:03.000", End: "00:00:04.000", Text: "Line C"},
		},
	}
	config := &dto.SubtitleExportConfig{
		ASS: &dto.SubtitleASSExportConfig{
			PlayResX: 1280,
			PlayResY: 720,
			Title:    "ASS Export",
		},
	}

	content := renderSubtitleContentWithConfig(document, "ass", config, testSubtitleExportStyleDocument)
	if got := requireASSSectionValue(t, content, "[Script Info]", "Title"); got != "ASS Export" {
		t.Fatalf("expected ass title config, got %q", got)
	}
	if got := requireASSSectionValue(t, content, "[Script Info]", "PlayResX"); got != "1280" {
		t.Fatalf("expected ass resolution config x, got %q", got)
	}
	if got := requireASSSectionValue(t, content, "[Script Info]", "PlayResY"); got != "720" {
		t.Fatalf("expected ass resolution config y, got %q", got)
	}
	if got := requireASSStyleField(t, content, "[V4+ Styles]", "fontname"); got != "Helvetica" {
		t.Fatalf("expected ass fontname to inherit subtitle style, got %q", got)
	}
	if got := requireASSStyleField(t, content, "[V4+ Styles]", "fontsize"); got != "40" {
		t.Fatalf("expected ass fontsize to scale to export resolution, got %q", got)
	}
	if got := requireASSStyleField(t, content, "[V4+ Styles]", "alignment"); got != "3" {
		t.Fatalf("expected ass alignment to inherit subtitle style, got %q", got)
	}
	if got := requireASSStyleField(t, content, "[V4+ Styles]", "bold"); got != "-1" {
		t.Fatalf("expected ass bold flag to inherit subtitle style, got %q", got)
	}
	if got := requireASSStyleField(t, content, "[V4+ Styles]", "italic"); got != "-1" {
		t.Fatalf("expected ass italic flag to inherit subtitle style, got %q", got)
	}
	if got := requireASSStyleField(t, content, "[V4+ Styles]", "marginl"); got != "48" {
		t.Fatalf("expected ass left margin to scale to export resolution, got %q", got)
	}
	if got := requireASSStyleField(t, content, "[V4+ Styles]", "marginv"); got != "37" {
		t.Fatalf("expected ass vertical margin to scale to export resolution, got %q", got)
	}

	parsed := parseASSDocument(content)
	if len(parsed.Events) != 2 {
		t.Fatalf("expected two ass dialogue events, got %#v", parsed.Events)
	}
	first := parsed.Events[0]
	if got := assEventValue(parsed.EventFormat, first.Values, "start"); got != "0:00:01.00" {
		t.Fatalf("expected first ass start timestamp, got %q", got)
	}
	if got := assEventValue(parsed.EventFormat, first.Values, "end"); got != "0:00:02.50" {
		t.Fatalf("expected first ass end timestamp, got %q", got)
	}
	if got := assEventValue(parsed.EventFormat, first.Values, "style"); got != "Primary" {
		t.Fatalf("expected first ass style name, got %q", got)
	}
	if got := assEventValue(parsed.EventFormat, first.Values, "text"); got != `Line A\NLine B` {
		t.Fatalf("expected ass multiline text to escape with \\N, got %q", got)
	}

	roundTrip := parseSubtitleDocument(content, "ass")
	if len(roundTrip.Cues) != 2 {
		t.Fatalf("expected ass round-trip parse to keep both cues, got %#v", roundTrip.Cues)
	}
	if roundTrip.Cues[0].Text != "Line A\nLine B" || roundTrip.Cues[1].Text != "Line C" {
		t.Fatalf("expected ass round-trip cue text to stay accurate, got %#v", roundTrip.Cues)
	}
}

func TestRenderSubtitleContentWithConfigSSA(t *testing.T) {
	t.Parallel()

	document := dto.SubtitleDocument{
		Format: "srt",
		Cues: []dto.SubtitleCue{
			{Index: 1, Start: "00:00:01,000", End: "00:00:02,000", Text: "SSA Line"},
		},
	}
	config := &dto.SubtitleExportConfig{
		ASS: &dto.SubtitleASSExportConfig{
			PlayResX: 1280,
			PlayResY: 720,
			Title:    "SSA Export",
		},
	}

	content := renderSubtitleContentWithConfig(document, "ssa", config, testSubtitleExportStyleDocument)
	if got := requireASSSectionValue(t, content, "[Script Info]", "Title"); got != "SSA Export" {
		t.Fatalf("expected ssa title config, got %q", got)
	}
	if got := requireASSSectionValue(t, content, "[Script Info]", "ScriptType"); got != "v4.00" {
		t.Fatalf("expected ssa script type, got %q", got)
	}
	if got := requireASSSectionValue(t, content, "[Script Info]", "PlayResX"); got != "1280" {
		t.Fatalf("expected ssa resolution config x, got %q", got)
	}
	if got := requireASSStyleField(t, content, "[V4 Styles]", "fontname"); got != "Helvetica" {
		t.Fatalf("expected ssa fontname to inherit subtitle style, got %q", got)
	}
	if got := requireASSStyleField(t, content, "[V4 Styles]", "fontsize"); got != "40" {
		t.Fatalf("expected ssa fontsize to scale to export resolution, got %q", got)
	}
	if got := requireASSStyleField(t, content, "[V4 Styles]", "alignment"); got != "3" {
		t.Fatalf("expected ssa alignment to inherit subtitle style, got %q", got)
	}
	if got := requireASSStyleField(t, content, "[V4 Styles]", "marginl"); got != "48" {
		t.Fatalf("expected ssa left margin to scale to export resolution, got %q", got)
	}
	if got := requireASSStyleField(t, content, "[V4 Styles]", "marginv"); got != "37" {
		t.Fatalf("expected ssa vertical margin to scale to export resolution, got %q", got)
	}

	parsed := parseASSDocument(content)
	if len(parsed.Events) != 1 {
		t.Fatalf("expected one ssa dialogue event, got %#v", parsed.Events)
	}
	first := parsed.Events[0]
	if got := assEventValue(parsed.EventFormat, first.Values, "marked"); got != "Marked=0" {
		t.Fatalf("expected ssa dialogue marker, got %q", got)
	}
	if got := assEventValue(parsed.EventFormat, first.Values, "start"); got != "0:00:01.00" {
		t.Fatalf("expected ssa start timestamp, got %q", got)
	}
	if got := assEventValue(parsed.EventFormat, first.Values, "end"); got != "0:00:02.00" {
		t.Fatalf("expected ssa end timestamp, got %q", got)
	}
	if got := assEventValue(parsed.EventFormat, first.Values, "style"); got != "Primary" {
		t.Fatalf("expected ssa style name, got %q", got)
	}
	if got := assEventValue(parsed.EventFormat, first.Values, "text"); got != "SSA Line" {
		t.Fatalf("expected ssa text, got %q", got)
	}
}

func TestRenderSubtitleContentWithConfigITT(t *testing.T) {
	t.Parallel()

	document := dto.SubtitleDocument{
		Format: "srt",
		Cues: []dto.SubtitleCue{
			{Index: 1, Start: "00:00:00,000", End: "00:00:01,000", Text: "Hello\nWorld"},
			{Index: 2, Start: "00:00:02,000", End: "00:00:03,500", Text: "Second"},
		},
	}
	config := &dto.SubtitleExportConfig{
		ITT: &dto.SubtitleITTExportConfig{
			FrameRate:           30,
			FrameRateMultiplier: "1000 1001",
			Language:            "zh-CN",
		},
	}

	content := renderSubtitleContentWithConfig(document, "itt", config, testSubtitleExportStyleDocument)
	rootAttrs, styleAttrs := parseRenderedITTAttrs(t, content)
	if got := xmlAttributeValue(rootAttrs, "lang"); got != "zh-CN" {
		t.Fatalf("expected itt language config, got %q", got)
	}
	if got := xmlAttributeValue(rootAttrs, "frameRate"); got != "30" {
		t.Fatalf("expected itt framerate config, got %q", got)
	}
	if got := xmlAttributeValue(rootAttrs, "frameRateMultiplier"); got != "1000 1001" {
		t.Fatalf("expected itt frameRateMultiplier config, got %q", got)
	}
	if got := xmlAttributeValue(styleAttrs, "fontFamily"); got != "Helvetica" {
		t.Fatalf("expected itt font family to inherit subtitle style, got %q", got)
	}
	if got := xmlAttributeValue(styleAttrs, "fontSize"); got != "60px" {
		t.Fatalf("expected itt font size to inherit subtitle style, got %q", got)
	}
	if got := xmlAttributeValue(styleAttrs, "textAlign"); got != "right" {
		t.Fatalf("expected itt text align to inherit subtitle style, got %q", got)
	}
	if got := xmlAttributeValue(styleAttrs, "displayAlign"); got != "after" {
		t.Fatalf("expected itt display align to inherit subtitle style, got %q", got)
	}
	if got := xmlAttributeValue(styleAttrs, "fontWeight"); got != "bold" {
		t.Fatalf("expected itt bold styling to inherit subtitle style, got %q", got)
	}
	if got := xmlAttributeValue(styleAttrs, "fontStyle"); got != "italic" {
		t.Fatalf("expected itt italic styling to inherit subtitle style, got %q", got)
	}

	paragraphs := parseITTCueParagraphs(content)
	if len(paragraphs) != 2 {
		t.Fatalf("expected two itt cue paragraphs, got %#v", paragraphs)
	}
	if paragraphs[0].Start != "00:00:00.000" || paragraphs[0].End != "00:00:01.000" || paragraphs[0].Text != "Hello\nWorld" {
		t.Fatalf("expected first itt cue paragraph, got %#v", paragraphs[0])
	}
	if paragraphs[1].Start != "00:00:02.000" || paragraphs[1].End != "00:00:03.500" || paragraphs[1].Text != "Second" {
		t.Fatalf("expected second itt cue paragraph, got %#v", paragraphs[1])
	}

	roundTrip := parseSubtitleDocument(content, "itt")
	if len(roundTrip.Cues) != 2 {
		t.Fatalf("expected itt round-trip parse to keep both cues, got %#v", roundTrip.Cues)
	}
	if roundTrip.Cues[0].Text != "Hello\nWorld" || roundTrip.Cues[1].Text != "Second" {
		t.Fatalf("expected itt round-trip cue text to stay accurate, got %#v", roundTrip.Cues)
	}
}
