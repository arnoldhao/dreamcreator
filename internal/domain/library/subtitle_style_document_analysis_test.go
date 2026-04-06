package library

import "testing"

func TestAnalyzeSubtitleStyleDocumentExtractsMetadataAndFeatures(t *testing.T) {
	t.Parallel()

	content := `[Script Info]
Title: Feature Demo
ScriptType: v4.00+
PlayResX: 1280
PlayResY: 720

[V4+ Styles]
Format: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour, Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, Encoding
Style: Default,Arial,48,&H00FFFFFF,&H00FFFFFF,&H00111111,&HFF111111,-1,0,0,0,100,100,0,0,1,2,0,2,72,72,56,1

[Events]
Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text
Dialogue: 0,0:00:00.00,0:00:02.00,Default,,0,0,0,,{\fnNoto Sans CJK SC\pos(640,660)\fad(150,150)}Hello
Comment: 0,0:00:02.00,0:00:04.00,Default,,0,0,0,,{\clip(0,0,100,100)\t(0,500,\alpha&H80&)\p1}m 0 0 l 10 10{\p0}
Dialogue: 0,0:00:04.00,0:00:06.00,Default,,0,0,0,,{\k20}World
`

	analysis := AnalyzeSubtitleStyleDocument(content)
	if analysis.DetectedFormat != "ass" {
		t.Fatalf("expected detected format ass, got %q", analysis.DetectedFormat)
	}
	if analysis.ScriptType != "v4.00+" {
		t.Fatalf("expected ScriptType v4.00+, got %q", analysis.ScriptType)
	}
	if analysis.PlayResX != 1280 || analysis.PlayResY != 720 {
		t.Fatalf("expected PlayRes 1280x720, got %dx%d", analysis.PlayResX, analysis.PlayResY)
	}
	if analysis.StyleCount != 1 {
		t.Fatalf("expected 1 style row, got %d", analysis.StyleCount)
	}
	if analysis.DialogueCount != 2 {
		t.Fatalf("expected 2 dialogue rows, got %d", analysis.DialogueCount)
	}
	if analysis.CommentCount != 1 {
		t.Fatalf("expected 1 comment row, got %d", analysis.CommentCount)
	}
	if len(analysis.StyleNames) != 1 || analysis.StyleNames[0] != "Default" {
		t.Fatalf("expected style name Default, got %#v", analysis.StyleNames)
	}
	assertStringSliceContains(t, analysis.Fonts, "Arial")
	assertStringSliceContains(t, analysis.Fonts, "Noto Sans CJK SC")
	assertStringSliceContains(t, analysis.FeatureFlags, "override-tags")
	assertStringSliceContains(t, analysis.FeatureFlags, "font-override")
	assertStringSliceContains(t, analysis.FeatureFlags, "positioning")
	assertStringSliceContains(t, analysis.FeatureFlags, "fade")
	assertStringSliceContains(t, analysis.FeatureFlags, "clipping")
	assertStringSliceContains(t, analysis.FeatureFlags, "transform")
	assertStringSliceContains(t, analysis.FeatureFlags, "vector-drawing")
	assertStringSliceContains(t, analysis.FeatureFlags, "karaoke")
	if len(analysis.ValidationIssues) != 0 {
		t.Fatalf("expected no validation issues, got %#v", analysis.ValidationIssues)
	}
}

func TestAnalyzeSubtitleStyleDocumentReportsMissingSections(t *testing.T) {
	t.Parallel()

	analysis := AnalyzeSubtitleStyleDocument(`[Events]
Dialogue: 0,0:00:00.00,0:00:01.00,Default,,0,0,0,,Hello
`)

	assertStringSliceContains(t, analysis.ValidationIssues, "missing [Script Info] section")
	assertStringSliceContains(t, analysis.ValidationIssues, "missing [V4+ Styles] or [V4 Styles] section")
	assertStringSliceContains(t, analysis.ValidationIssues, "missing ScriptType in [Script Info]")
	assertStringSliceContains(t, analysis.ValidationIssues, "missing PlayResX in [Script Info]")
	assertStringSliceContains(t, analysis.ValidationIssues, "missing PlayResY in [Script Info]")
}

func assertStringSliceContains(t *testing.T, values []string, want string) {
	t.Helper()
	for _, value := range values {
		if value == want {
			return
		}
	}
	t.Fatalf("expected %#v to contain %q", values, want)
}
