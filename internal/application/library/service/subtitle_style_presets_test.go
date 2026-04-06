package service

import (
	"context"
	"math"
	"os"
	"strings"
	"testing"

	"dreamcreator/internal/application/library/dto"
)

func TestParseSubtitleStyleImportNormalizesASSMonoStyles(t *testing.T) {
	service := &LibraryService{}
	content := strings.Join([]string{
		"[Script Info]",
		"ScriptType: v4.00+",
		"PlayResX: 1280",
		"PlayResY: 720",
		"",
		"[V4+ Styles]",
		"Format: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour, Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, Encoding",
		"Style: Primary,Arial,40,&H00FFFFFF,&H00FFFFFF,&H00111111,&HFF111111,-1,0,0,0,100,100,2,0,1,2,1,2,20,20,30,1",
		"",
		"[Events]",
		"Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text",
	}, "\n")

	result, err := service.ParseSubtitleStyleImport(context.Background(), dto.ParseSubtitleStyleImportRequest{
		Content: content,
		Format:  "ass",
	})
	if err != nil {
		t.Fatalf("ParseSubtitleStyleImport returned error: %v", err)
	}
	if result.DetectedRatio != "16:9" {
		t.Fatalf("expected detected ratio 16:9, got %q", result.DetectedRatio)
	}
	if result.NormalizedPlayResX != 1920 || result.NormalizedPlayResY != 1080 {
		t.Fatalf("expected normalized playres 1920x1080, got %dx%d", result.NormalizedPlayResX, result.NormalizedPlayResY)
	}
	if len(result.MonoStyles) != 1 {
		t.Fatalf("expected one mono style, got %d", len(result.MonoStyles))
	}
	style := result.MonoStyles[0]
	if style.BasePlayResX != 1920 || style.BasePlayResY != 1080 {
		t.Fatalf("expected mono base resolution 1920x1080, got %dx%d", style.BasePlayResX, style.BasePlayResY)
	}
	if style.Style.Fontsize != 60 {
		t.Fatalf("expected normalized fontsize 60, got %v", style.Style.Fontsize)
	}
	if style.Style.Outline != 3 {
		t.Fatalf("expected normalized outline 3, got %v", style.Style.Outline)
	}
	if style.Style.Shadow != 1.5 {
		t.Fatalf("expected normalized shadow 1.5, got %v", style.Style.Shadow)
	}
	if style.Style.MarginL != 30 || style.Style.MarginR != 30 || style.Style.MarginV != 45 {
		t.Fatalf("expected normalized margins 30/30/45, got %d/%d/%d", style.Style.MarginL, style.Style.MarginR, style.Style.MarginV)
	}
	if style.Style.Spacing != 3 {
		t.Fatalf("expected normalized spacing 3, got %v", style.Style.Spacing)
	}
}

func TestParseSubtitleStyleImportRoundsScaledFontSize(t *testing.T) {
	service := &LibraryService{}
	content := strings.Join([]string{
		"[Script Info]",
		"ScriptType: v4.00+",
		"PlayResX: 1280",
		"PlayResY: 720",
		"",
		"[V4+ Styles]",
		"Format: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour, Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, Encoding",
		"Style: Primary,Arial,39.7,&H00FFFFFF,&H00FFFFFF,&H00111111,&HFF111111,-1,0,0,0,100,100,0,0,1,2,1,2,20,20,30,1",
		"",
		"[Events]",
		"Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text",
	}, "\n")

	result, err := service.ParseSubtitleStyleImport(context.Background(), dto.ParseSubtitleStyleImportRequest{
		Content: content,
		Format:  "ass",
	})
	if err != nil {
		t.Fatalf("ParseSubtitleStyleImport returned error: %v", err)
	}
	if len(result.MonoStyles) != 1 {
		t.Fatalf("expected one mono style, got %d", len(result.MonoStyles))
	}
	if result.MonoStyles[0].Style.Fontsize != 60 {
		t.Fatalf("expected rounded fontsize 60, got %v", result.MonoStyles[0].Style.Fontsize)
	}
	if math.Mod(result.MonoStyles[0].Style.Fontsize, 1) != 0 {
		t.Fatalf("expected integer fontsize, got %v", result.MonoStyles[0].Style.Fontsize)
	}
}

func TestGenerateSubtitleStylePreviewASSBuildsBilingualDocument(t *testing.T) {
	service := &LibraryService{}
	result, err := service.GenerateSubtitleStylePreviewASS(context.Background(), dto.GenerateSubtitleStylePreviewASSRequest{
		Type: "bilingual",
		Bilingual: &dto.LibraryBilingualStyleDTO{
			ID:              "bilingual-test",
			Name:            "Bilingual Test",
			BasePlayResX:    1920,
			BasePlayResY:    1080,
			BaseAspectRatio: "16:9",
			Primary: dto.LibraryMonoStyleSnapshotDTO{
				Name:            "Primary",
				BasePlayResX:    1920,
				BasePlayResY:    1080,
				BaseAspectRatio: "16:9",
				Style: dto.AssStyleSpecDTO{
					Fontname:      "Arial",
					Fontsize:      52,
					PrimaryColour: "&H00FFFFFF",
					OutlineColour: "&H00111111",
					BackColour:    "&HFF111111",
					ScaleX:        100,
					ScaleY:        100,
					BorderStyle:   1,
					Outline:       2.2,
					Shadow:        0.8,
					Alignment:     2,
					MarginL:       72,
					MarginR:       72,
					MarginV:       60,
					Encoding:      1,
				},
			},
			Secondary: dto.LibraryMonoStyleSnapshotDTO{
				Name:            "Secondary",
				BasePlayResX:    1920,
				BasePlayResY:    1080,
				BaseAspectRatio: "16:9",
				Style: dto.AssStyleSpecDTO{
					Fontname:      "Arial",
					Fontsize:      38,
					PrimaryColour: "&H00EAE0D9",
					OutlineColour: "&H00111111",
					BackColour:    "&HFF111111",
					ScaleX:        100,
					ScaleY:        100,
					BorderStyle:   1,
					Outline:       1.6,
					Shadow:        0.6,
					Alignment:     2,
					MarginL:       72,
					MarginR:       72,
					MarginV:       20,
					Encoding:      1,
				},
			},
			Layout: dto.LibraryBilingualLayoutDTO{
				Gap:         24,
				BlockAnchor: 2,
			},
		},
	})
	if err != nil {
		t.Fatalf("GenerateSubtitleStylePreviewASS returned error: %v", err)
	}
	if !strings.Contains(result.ASSContent, "Style: Primary") {
		t.Fatalf("expected preview ASS to include primary style, got %q", result.ASSContent)
	}
	if !strings.Contains(result.ASSContent, "Style: Secondary") {
		t.Fatalf("expected preview ASS to include secondary style, got %q", result.ASSContent)
	}
	if !strings.Contains(result.ASSContent, "Dialogue: 0,0:00:00.00,0:01:00.00,Primary") {
		t.Fatalf("expected preview ASS to include primary dialogue line, got %q", result.ASSContent)
	}
	if !strings.Contains(result.ASSContent, "Dialogue: 0,0:00:00.00,0:01:00.00,Secondary") {
		t.Fatalf("expected preview ASS to include secondary dialogue line, got %q", result.ASSContent)
	}
}

func TestGenerateSubtitleStylePreviewASSCentersBilingualMiddleAnchor(t *testing.T) {
	service := &LibraryService{}
	result, err := service.GenerateSubtitleStylePreviewASS(context.Background(), dto.GenerateSubtitleStylePreviewASSRequest{
		Type: "bilingual",
		Bilingual: &dto.LibraryBilingualStyleDTO{
			ID:              "bilingual-middle-anchor",
			Name:            "Bilingual Middle Anchor",
			BasePlayResX:    1920,
			BasePlayResY:    1080,
			BaseAspectRatio: "16:9",
			Primary: dto.LibraryMonoStyleSnapshotDTO{
				Name:            "Primary",
				BasePlayResX:    1920,
				BasePlayResY:    1080,
				BaseAspectRatio: "16:9",
				Style: dto.AssStyleSpecDTO{
					Fontname:      "Arial",
					Fontsize:      52,
					PrimaryColour: "&H00FFFFFF",
					OutlineColour: "&H00111111",
					BackColour:    "&HFF111111",
					ScaleX:        100,
					ScaleY:        100,
					BorderStyle:   1,
					Outline:       2,
					Shadow:        1,
					Alignment:     2,
					MarginL:       72,
					MarginR:       72,
					MarginV:       56,
					Encoding:      1,
				},
			},
			Secondary: dto.LibraryMonoStyleSnapshotDTO{
				Name:            "Secondary",
				BasePlayResX:    1920,
				BasePlayResY:    1080,
				BaseAspectRatio: "16:9",
				Style: dto.AssStyleSpecDTO{
					Fontname:      "Arial",
					Fontsize:      38,
					PrimaryColour: "&H00EAE0D9",
					OutlineColour: "&H00111111",
					BackColour:    "&HFF111111",
					ScaleX:        100,
					ScaleY:        100,
					BorderStyle:   1,
					Outline:       2,
					Shadow:        1,
					Alignment:     2,
					MarginL:       72,
					MarginR:       72,
					MarginV:       56,
					Encoding:      1,
				},
			},
			Layout: dto.LibraryBilingualLayoutDTO{
				Gap:         24,
				BlockAnchor: 5,
			},
		},
	})
	if err != nil {
		t.Fatalf("GenerateSubtitleStylePreviewASS returned error: %v", err)
	}
	if !strings.Contains(result.ASSContent, ",8,72,72,483,1") {
		t.Fatalf("expected primary centered top-anchor margins in ASS, got %q", result.ASSContent)
	}
	if !strings.Contains(result.ASSContent, ",8,72,72,559,1") {
		t.Fatalf("expected secondary centered top-anchor margins in ASS, got %q", result.ASSContent)
	}
}

func TestGenerateSubtitleStylePreviewVTTBuildsBilingualDocument(t *testing.T) {
	service := &LibraryService{}
	result, err := service.GenerateSubtitleStylePreviewVTT(context.Background(), dto.GenerateSubtitleStylePreviewVTTRequest{
		Type: "bilingual",
		Bilingual: &dto.LibraryBilingualStyleDTO{
			ID:              "bilingual-test",
			Name:            "Bilingual Test",
			BasePlayResX:    1920,
			BasePlayResY:    1080,
			BaseAspectRatio: "16:9",
			Primary: dto.LibraryMonoStyleSnapshotDTO{
				Name:            "Primary",
				BasePlayResX:    1920,
				BasePlayResY:    1080,
				BaseAspectRatio: "16:9",
				Style: dto.AssStyleSpecDTO{
					Fontname:      "Arial",
					Fontsize:      52,
					PrimaryColour: "&H00FFFFFF",
					OutlineColour: "&H00111111",
					BackColour:    "&HFF111111",
					ScaleX:        100,
					ScaleY:        100,
					BorderStyle:   1,
					Outline:       2.2,
					Shadow:        0.8,
					Alignment:     2,
					MarginL:       72,
					MarginR:       72,
					MarginV:       60,
					Encoding:      1,
				},
			},
			Secondary: dto.LibraryMonoStyleSnapshotDTO{
				Name:            "Secondary",
				BasePlayResX:    1920,
				BasePlayResY:    1080,
				BaseAspectRatio: "16:9",
				Style: dto.AssStyleSpecDTO{
					Fontname:      "Arial",
					Fontsize:      38,
					PrimaryColour: "&H00EAE0D9",
					OutlineColour: "&H00111111",
					BackColour:    "&HFF111111",
					ScaleX:        100,
					ScaleY:        100,
					BorderStyle:   1,
					Outline:       1.6,
					Shadow:        0.6,
					Alignment:     2,
					MarginL:       72,
					MarginR:       72,
					MarginV:       20,
					Encoding:      1,
				},
			},
			Layout: dto.LibraryBilingualLayoutDTO{
				Gap:         24,
				BlockAnchor: 2,
			},
		},
		PreviewWidth:  960,
		PreviewHeight: 540,
		PrimaryText:   "Primary Preview",
		SecondaryText: "Secondary Preview",
	})
	if err != nil {
		t.Fatalf("GenerateSubtitleStylePreviewVTT returned error: %v", err)
	}
	if !strings.Contains(result.VTTContent, "WEBVTT") {
		t.Fatalf("expected preview VTT to include WEBVTT header, got %q", result.VTTContent)
	}
	if !strings.Contains(result.VTTContent, "::cue(.primary)") {
		t.Fatalf("expected preview VTT to include primary style block, got %q", result.VTTContent)
	}
	if !strings.Contains(result.VTTContent, "::cue(.secondary)") {
		t.Fatalf("expected preview VTT to include secondary style block, got %q", result.VTTContent)
	}
	if !strings.Contains(result.VTTContent, "<c.primary>Primary Preview</c>") {
		t.Fatalf("expected preview VTT to include primary cue text, got %q", result.VTTContent)
	}
	if !strings.Contains(result.VTTContent, "<c.secondary>Secondary Preview</c>") {
		t.Fatalf("expected preview VTT to include secondary cue text, got %q", result.VTTContent)
	}
}

func TestGenerateSubtitleStylePreviewASSAppliesFontMappings(t *testing.T) {
	service := &LibraryService{}
	result, err := service.GenerateSubtitleStylePreviewASS(context.Background(), dto.GenerateSubtitleStylePreviewASSRequest{
		Type: "mono",
		Mono: &dto.LibraryMonoStyleDTO{
			ID:              "mono-font-map",
			Name:            "Mono Font Map",
			BasePlayResX:    1920,
			BasePlayResY:    1080,
			BaseAspectRatio: "16:9",
			Style: dto.AssStyleSpecDTO{
				Fontname:      "Source Han Sans",
				Fontsize:      48,
				PrimaryColour: "&H00FFFFFF",
				OutlineColour: "&H00111111",
				BackColour:    "&HFF111111",
				ScaleX:        100,
				ScaleY:        100,
				BorderStyle:   1,
				Outline:       2,
				Shadow:        1,
				Alignment:     2,
				MarginL:       72,
				MarginR:       72,
				MarginV:       56,
				Encoding:      1,
			},
		},
		FontMappings: []dto.LibrarySubtitleStyleFontDTO{
			{
				ID:           "font-map-1",
				Family:       "Source Han Sans",
				SystemFamily: "PingFang SC",
				Enabled:      true,
			},
		},
	})
	if err != nil {
		t.Fatalf("GenerateSubtitleStylePreviewASS returned error: %v", err)
	}
	if !strings.Contains(result.ASSContent, "Style: Primary,PingFang SC,48,") {
		t.Fatalf("expected preview ASS to use mapped system font, got %q", result.ASSContent)
	}
	if strings.Contains(result.ASSContent, "Style: Primary,Source Han Sans,48,") {
		t.Fatalf("expected preview ASS to replace original font family, got %q", result.ASSContent)
	}
}

func TestExportSubtitleStylePresetWritesDCSSPFile(t *testing.T) {
	service := &LibraryService{}
	dir := t.TempDir()
	result, err := service.ExportSubtitleStylePreset(context.Background(), dto.ExportSubtitleStylePresetRequest{
		DirectoryPath: dir,
		Type:          "mono",
		Mono: &dto.LibraryMonoStyleDTO{
			ID:              "mono-export",
			Name:            "Mono Export",
			BasePlayResX:    1920,
			BasePlayResY:    1080,
			BaseAspectRatio: "16:9",
			Style: dto.AssStyleSpecDTO{
				Fontname:      "Arial",
				Fontsize:      48,
				PrimaryColour: "&H00FFFFFF",
				OutlineColour: "&H00111111",
				BackColour:    "&HFF111111",
				ScaleX:        100,
				ScaleY:        100,
				BorderStyle:   1,
				Outline:       2,
				Shadow:        1,
				Alignment:     2,
				MarginL:       72,
				MarginR:       72,
				MarginV:       56,
				Encoding:      1,
			},
		},
	})
	if err != nil {
		t.Fatalf("ExportSubtitleStylePreset returned error: %v", err)
	}
	if !strings.HasSuffix(result.FileName, ".dcssp") {
		t.Fatalf("expected dcssp file suffix, got %q", result.FileName)
	}
	content, err := os.ReadFile(result.ExportPath)
	if err != nil {
		t.Fatalf("read export file failed: %v", err)
	}
	text := string(content)
	if !strings.Contains(text, `"format": "dcssp"`) {
		t.Fatalf("expected export file to include dcssp format, got %q", text)
	}
	if !strings.Contains(text, `"type": "mono"`) {
		t.Fatalf("expected export file to include mono type, got %q", text)
	}
	if !strings.Contains(text, `"name": "Mono Export"`) {
		t.Fatalf("expected export file to include preset name, got %q", text)
	}
}
