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

func TestRenderSubtitleContentWithConfigFCPXML(t *testing.T) {
	document := dto.SubtitleDocument{
		Format: "srt",
		Cues: []dto.SubtitleCue{
			{Index: 1, Start: "00:00:01,000", End: "00:00:02,500", Text: "Hello FCPXML"},
		},
	}
	config := &dto.SubtitleExportConfig{
		FCPXML: &dto.SubtitleFCPXMLExportConfig{
			FrameDuration: "1001/30000s",
			Width:         3840,
			Height:        2160,
			ColorSpace:    "Rec. 2020",
			ProjectName:   "DemoProject",
		},
	}

	content := renderSubtitleContentWithConfig(document, "fcpxml", config, testSubtitleExportStyleDocument)
	if !strings.Contains(content, "<fcpxml") {
		t.Fatalf("expected fcpxml root, got %q", content)
	}
	if !strings.Contains(content, `width="3840"`) || !strings.Contains(content, `height="2160"`) {
		t.Fatalf("expected configured resolution in fcpxml, got %q", content)
	}
	if !strings.Contains(content, `frameDuration="1001/30000s"`) {
		t.Fatalf("expected configured frame duration, got %q", content)
	}
	if !strings.Contains(content, `offset="3601s"`) || !strings.Contains(content, `duration="3/2s"`) {
		t.Fatalf("expected subtitle timing to use rational seconds, got %q", content)
	}
	if !strings.Contains(content, "DemoProject") {
		t.Fatalf("expected configured project name, got %q", content)
	}
	if !strings.Contains(content, "<title") {
		t.Fatalf("expected subtitle titles in fcpxml, got %q", content)
	}
	if !strings.Contains(content, `font="Helvetica"`) || !strings.Contains(content, `fontSize="60"`) {
		t.Fatalf("expected fcpxml text style to inherit subtitle style font, got %q", content)
	}
	if !strings.Contains(content, `alignment="right"`) || !strings.Contains(content, `bold="1"`) || !strings.Contains(content, `italic="1"`) {
		t.Fatalf("expected fcpxml text style to inherit subtitle style emphasis, got %q", content)
	}
}

func TestRenderSubtitleContentWithConfigASS(t *testing.T) {
	document := dto.SubtitleDocument{
		Format: "vtt",
		Cues: []dto.SubtitleCue{
			{Index: 1, Start: "00:00:01.000", End: "00:00:02.000", Text: "Line A"},
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
	if !strings.Contains(content, "PlayResX: 1280") || !strings.Contains(content, "PlayResY: 720") {
		t.Fatalf("expected ass resolution config, got %q", content)
	}
	if !strings.Contains(content, "Title: ASS Export") {
		t.Fatalf("expected ass title config, got %q", content)
	}
	if !strings.Contains(content, "Style: Primary,Helvetica,60") {
		t.Fatalf("expected ass export to preserve subtitle style definition, got %q", content)
	}
	if !strings.Contains(content, "Dialogue: 0,0:00:01.00,0:00:02.00,Primary") {
		t.Fatalf("expected ass dialogue line to use subtitle style, got %q", content)
	}
}

func TestRenderSubtitleContentWithConfigSSA(t *testing.T) {
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
	if !strings.Contains(content, "ScriptType: v4.00") {
		t.Fatalf("expected ssa script type, got %q", content)
	}
	if !strings.Contains(content, "[V4 Styles]") {
		t.Fatalf("expected ssa style section, got %q", content)
	}
	if !strings.Contains(content, "Dialogue: Marked=0,0:00:01.00,0:00:02.00,Primary") {
		t.Fatalf("expected ssa dialogue line, got %q", content)
	}
}

func TestRenderSubtitleContentWithConfigITT(t *testing.T) {
	document := dto.SubtitleDocument{
		Format: "srt",
		Cues: []dto.SubtitleCue{
			{Index: 1, Start: "00:00:00,000", End: "00:00:01,000", Text: "Hello"},
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
	if !strings.Contains(content, `xml:lang="zh-CN"`) {
		t.Fatalf("expected itt language config, got %q", content)
	}
	if !strings.Contains(content, `ttp:frameRate="30"`) {
		t.Fatalf("expected itt framerate config, got %q", content)
	}
	if !strings.Contains(content, `ttp:frameRateMultiplier="1000 1001"`) {
		t.Fatalf("expected itt frameRateMultiplier config, got %q", content)
	}
	if !strings.Contains(content, `tts:fontFamily="Helvetica"`) || !strings.Contains(content, `tts:fontSize="60px"`) {
		t.Fatalf("expected itt styling to inherit subtitle style font, got %q", content)
	}
	if !strings.Contains(content, `tts:textAlign="right"`) || !strings.Contains(content, `tts:fontWeight="bold"`) {
		t.Fatalf("expected itt styling to inherit subtitle style alignment/emphasis, got %q", content)
	}
	if !strings.Contains(content, "<p begin=\"00:00:00.000\" end=\"00:00:01.000\">Hello</p>") {
		t.Fatalf("expected itt cue paragraph, got %q", content)
	}
}
