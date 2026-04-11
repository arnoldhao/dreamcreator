package library

import "runtime"

var defaultBuiltInSubtitleStyleFontSources = []SubtitleStyleSource{
	{
		ID:       "fontget-google-fonts",
		Name:     "Google Fonts",
		Kind:     "font",
		Provider: subtitleStyleSourceProviderFontGet,
		URL:      "https://raw.githubusercontent.com/Graphixa/FontGet-Sources/main/sources/google-fonts.json",
		Prefix:   "google",
		Filename: "google-fonts.json",
		Priority: 1,
		BuiltIn:  true,
		Enabled:  true,
	},
	{
		ID:       "fontget-nerd-fonts",
		Name:     "Nerd Fonts",
		Kind:     "font",
		Provider: subtitleStyleSourceProviderFontGet,
		URL:      "https://raw.githubusercontent.com/Graphixa/FontGet-Sources/main/sources/nerd-fonts.json",
		Prefix:   "nerd",
		Filename: "nerd-fonts.json",
		Priority: 2,
		BuiltIn:  true,
		Enabled:  true,
	},
	{
		ID:       "fontget-font-squirrel",
		Name:     "Font Squirrel",
		Kind:     "font",
		Provider: subtitleStyleSourceProviderFontGet,
		URL:      "https://raw.githubusercontent.com/Graphixa/FontGet-Sources/main/sources/font-squirrel.json",
		Prefix:   "squirrel",
		Filename: "font-squirrel.json",
		Priority: 3,
		BuiltIn:  true,
		Enabled:  true,
	},
}

var builtInSubtitleStyleFontSourceIDs = map[string]struct{}{
	"fontget-google-fonts":  {},
	"fontget-nerd-fonts":    {},
	"fontget-font-squirrel": {},
}

func PreferredPlatformSubtitleFontFamily() string {
	switch runtime.GOOS {
	case "windows":
		return "Segoe UI"
	case "darwin":
		return "Helvetica"
	default:
		return "Helvetica"
	}
}

var defaultBuiltInMonoStyles = []MonoStyle{
	{
		ID:                 "builtin-subtitle-mono-primary-1080p",
		Name:               "1080p Primary",
		BuiltIn:            true,
		BasePlayResX:       1920,
		BasePlayResY:       1080,
		BaseAspectRatio:    SubtitleStyleAspectRatio16By9,
		SourceAssStyleName: "Primary",
		Style: AssStyleSpec{
			Fontname:           PreferredPlatformSubtitleFontFamily(),
			FontFace:           "Semibold",
			FontWeight:         600,
			FontPostScriptName: "",
			Fontsize:           52,
			PrimaryColour:      "&H00FFFFFF",
			SecondaryColour:    "&H00FFFFFF",
			OutlineColour:      "&H00101010",
			BackColour:         "&H80000000",
			Bold:               false,
			Italic:             false,
			Underline:          false,
			StrikeOut:          false,
			ScaleX:             100,
			ScaleY:             100,
			Spacing:            0,
			Angle:              0,
			BorderStyle:        1,
			Outline:            0,
			Shadow:             5,
			Alignment:          2,
			MarginL:            72,
			MarginR:            72,
			MarginV:            72,
			Encoding:           1,
		},
	},
	{
		ID:                 "builtin-subtitle-mono-secondary-1080p",
		Name:               "1080p Secondary",
		BuiltIn:            true,
		BasePlayResX:       1920,
		BasePlayResY:       1080,
		BaseAspectRatio:    SubtitleStyleAspectRatio16By9,
		SourceAssStyleName: "Secondary",
		Style: AssStyleSpec{
			Fontname:           PreferredPlatformSubtitleFontFamily(),
			FontFace:           "Regular",
			FontWeight:         400,
			FontPostScriptName: "",
			Fontsize:           40,
			PrimaryColour:      "&H00E8E8E8",
			SecondaryColour:    "&H00E8E8E8",
			OutlineColour:      "&H00101010",
			BackColour:         "&H80000000",
			Bold:               false,
			Italic:             false,
			Underline:          false,
			StrikeOut:          false,
			ScaleX:             100,
			ScaleY:             100,
			Spacing:            0,
			Angle:              0,
			BorderStyle:        1,
			Outline:            0,
			Shadow:             5,
			Alignment:          2,
			MarginL:            72,
			MarginR:            72,
			MarginV:            72,
			Encoding:           1,
		},
	},
}

var defaultBuiltInBilingualStyles = []BilingualStyle{
	{
		ID:              "builtin-subtitle-bilingual-1080p",
		Name:            "1080p Bilingual",
		BuiltIn:         true,
		BasePlayResX:    1920,
		BasePlayResY:    1080,
		BaseAspectRatio: SubtitleStyleAspectRatio16By9,
		Primary: MonoStyleSnapshot{
			SourceMonoStyleID:   "builtin-subtitle-mono-primary-1080p",
			SourceMonoStyleName: "1080p Primary",
			Name:                "Primary",
			BasePlayResX:        1920,
			BasePlayResY:        1080,
			BaseAspectRatio:     SubtitleStyleAspectRatio16By9,
			Style: AssStyleSpec{
				Fontname:           PreferredPlatformSubtitleFontFamily(),
				FontFace:           "Semibold",
				FontWeight:         600,
				FontPostScriptName: "",
				Fontsize:           52,
				PrimaryColour:      "&H00FFFFFF",
				SecondaryColour:    "&H00FFFFFF",
				OutlineColour:      "&H00101010",
				BackColour:         "&H80000000",
				Bold:               false,
				Italic:             false,
				Underline:          false,
				StrikeOut:          false,
				ScaleX:             100,
				ScaleY:             100,
				Spacing:            0,
				Angle:              0,
				BorderStyle:        1,
				Outline:            0,
				Shadow:             5,
				Alignment:          2,
				MarginL:            72,
				MarginR:            72,
				MarginV:            72,
				Encoding:           1,
			},
		},
		Secondary: MonoStyleSnapshot{
			SourceMonoStyleID:   "builtin-subtitle-mono-secondary-1080p",
			SourceMonoStyleName: "1080p Secondary",
			Name:                "Secondary",
			BasePlayResX:        1920,
			BasePlayResY:        1080,
			BaseAspectRatio:     SubtitleStyleAspectRatio16By9,
			Style: AssStyleSpec{
				Fontname:           PreferredPlatformSubtitleFontFamily(),
				FontFace:           "Regular",
				FontWeight:         400,
				FontPostScriptName: "",
				Fontsize:           40,
				PrimaryColour:      "&H00E8E8E8",
				SecondaryColour:    "&H00E8E8E8",
				OutlineColour:      "&H00101010",
				BackColour:         "&H80000000",
				Bold:               false,
				Italic:             false,
				Underline:          false,
				StrikeOut:          false,
				ScaleX:             100,
				ScaleY:             100,
				Spacing:            0,
				Angle:              0,
				BorderStyle:        1,
				Outline:            0,
				Shadow:             5,
				Alignment:          2,
				MarginL:            72,
				MarginR:            72,
				MarginV:            72,
				Encoding:           1,
			},
		},
		Layout: BilingualLayout{
			Gap:         20,
			BlockAnchor: 2,
		},
	},
}

var builtInSubtitleMonoStyleIDs = map[string]struct{}{
	"builtin-subtitle-mono-primary-1080p":   {},
	"builtin-subtitle-mono-secondary-1080p": {},
}

var builtInSubtitleBilingualStyleIDs = map[string]struct{}{
	"builtin-subtitle-bilingual-1080p": {},
}

var defaultBuiltInSubtitleExportPresets = []SubtitleExportPreset{
	{
		ID:            "builtin-subtitle-export-preset-srt-auto",
		Name:          "SRT · Auto",
		Description:   "SRT output with source-matched timing defaults.",
		TargetFormat:  "srt",
		MediaStrategy: subtitleExportPresetMediaStrategyAuto,
		Config: SubtitleExportConfig{
			SRT: &SubtitleSRTExportConfig{
				Encoding: "utf-8",
			},
		},
	},
	{
		ID:            "builtin-subtitle-export-preset-vtt-auto",
		Name:          "WebVTT · Auto",
		Description:   "WebVTT output for web playback and soft subtitle muxing.",
		TargetFormat:  "vtt",
		MediaStrategy: subtitleExportPresetMediaStrategyAuto,
		Config: SubtitleExportConfig{
			VTT: &SubtitleVTTExportConfig{
				Kind:     "subtitles",
				Language: "en-US",
			},
		},
	},
	{
		ID:            "builtin-subtitle-export-preset-ass-auto",
		Name:          "ASS · Auto",
		Description:   "ASS output with auto-matched script resolution.",
		TargetFormat:  "ass",
		MediaStrategy: subtitleExportPresetMediaStrategyAuto,
		Config: SubtitleExportConfig{
			ASS: &SubtitleASSExportConfig{
				Title: "DreamCreator Export",
			},
		},
	},
	{
		ID:            "builtin-subtitle-export-preset-ass-4k",
		Name:          "ASS · 4K",
		Description:   "ASS output forced to 3840x2160 delivery.",
		TargetFormat:  "ass",
		MediaStrategy: subtitleExportPresetMediaStrategyFixed,
		Config: SubtitleExportConfig{
			ASS: &SubtitleASSExportConfig{
				PlayResX: 3840,
				PlayResY: 2160,
				Title:    "DreamCreator Export",
			},
		},
	},
	{
		ID:            "builtin-subtitle-export-preset-ass-1080p",
		Name:          "ASS · 1080p",
		Description:   "ASS output forced to 1920x1080 delivery.",
		TargetFormat:  "ass",
		MediaStrategy: subtitleExportPresetMediaStrategyFixed,
		Config: SubtitleExportConfig{
			ASS: &SubtitleASSExportConfig{
				PlayResX: 1920,
				PlayResY: 1080,
				Title:    "DreamCreator Export",
			},
		},
	},
	{
		ID:            "builtin-subtitle-export-preset-itt-auto",
		Name:          "ITT · Auto",
		Description:   "ITT output with source-matched frame timing.",
		TargetFormat:  "itt",
		MediaStrategy: subtitleExportPresetMediaStrategyAuto,
		Config: SubtitleExportConfig{
			ITT: &SubtitleITTExportConfig{
				FrameRate:           30,
				FrameRateMultiplier: "1 1",
				Language:            "en-US",
			},
		},
	},
	{
		ID:            "builtin-subtitle-export-preset-itt-4k60",
		Name:          "ITT · 4K · 60fps",
		Description:   "ITT output forced to 60fps timing for 4K delivery.",
		TargetFormat:  "itt",
		MediaStrategy: subtitleExportPresetMediaStrategyFixed,
		Config: SubtitleExportConfig{
			ITT: &SubtitleITTExportConfig{
				FrameRate:           60,
				FrameRateMultiplier: "1 1",
				Language:            "en-US",
			},
		},
	},
	{
		ID:            "builtin-subtitle-export-preset-itt-4k30",
		Name:          "ITT · 4K · 30fps",
		Description:   "ITT output forced to 30fps timing for 4K delivery.",
		TargetFormat:  "itt",
		MediaStrategy: subtitleExportPresetMediaStrategyFixed,
		Config: SubtitleExportConfig{
			ITT: &SubtitleITTExportConfig{
				FrameRate:           30,
				FrameRateMultiplier: "1 1",
				Language:            "en-US",
			},
		},
	},
	{
		ID:            "builtin-subtitle-export-preset-itt-1080p60",
		Name:          "ITT · 1080p · 60fps",
		Description:   "ITT output forced to 60fps timing for 1080p delivery.",
		TargetFormat:  "itt",
		MediaStrategy: subtitleExportPresetMediaStrategyFixed,
		Config: SubtitleExportConfig{
			ITT: &SubtitleITTExportConfig{
				FrameRate:           60,
				FrameRateMultiplier: "1 1",
				Language:            "en-US",
			},
		},
	},
	{
		ID:            "builtin-subtitle-export-preset-itt-1080p30",
		Name:          "ITT · 1080p · 30fps",
		Description:   "ITT output forced to 30fps timing for 1080p delivery.",
		TargetFormat:  "itt",
		MediaStrategy: subtitleExportPresetMediaStrategyFixed,
		Config: SubtitleExportConfig{
			ITT: &SubtitleITTExportConfig{
				FrameRate:           30,
				FrameRateMultiplier: "1 1",
				Language:            "en-US",
			},
		},
	},
	{
		ID:            "builtin-subtitle-export-preset-fcpxml-auto",
		Name:          "FCPXML · Auto",
		Description:   "FCPXML timeline with auto-matched resolution and frame duration.",
		TargetFormat:  "fcpxml",
		MediaStrategy: subtitleExportPresetMediaStrategyAuto,
		Config: SubtitleExportConfig{
			FCPXML: &SubtitleFCPXMLExportConfig{
				ColorSpace:           "1-1-1 (Rec. 709)",
				Version:              "1.11",
				DefaultLane:          1,
				StartTimecodeSeconds: 0,
			},
		},
	},
	{
		ID:            "builtin-subtitle-export-preset-fcpxml-4k60",
		Name:          "FCPXML · 4K · 1/60s",
		Description:   "FCPXML timeline forced to 3840x2160 at 1/60s frame duration.",
		TargetFormat:  "fcpxml",
		MediaStrategy: subtitleExportPresetMediaStrategyFixed,
		Config: SubtitleExportConfig{
			FCPXML: &SubtitleFCPXMLExportConfig{
				FrameDuration:        "1/60s",
				Width:                3840,
				Height:               2160,
				ColorSpace:           "1-1-1 (Rec. 709)",
				Version:              "1.11",
				DefaultLane:          1,
				StartTimecodeSeconds: 0,
			},
		},
	},
	{
		ID:            "builtin-subtitle-export-preset-fcpxml-4k30",
		Name:          "FCPXML · 4K · 1/30s",
		Description:   "FCPXML timeline forced to 3840x2160 at 1/30s frame duration.",
		TargetFormat:  "fcpxml",
		MediaStrategy: subtitleExportPresetMediaStrategyFixed,
		Config: SubtitleExportConfig{
			FCPXML: &SubtitleFCPXMLExportConfig{
				FrameDuration:        "1/30s",
				Width:                3840,
				Height:               2160,
				ColorSpace:           "1-1-1 (Rec. 709)",
				Version:              "1.11",
				DefaultLane:          1,
				StartTimecodeSeconds: 0,
			},
		},
	},
	{
		ID:            "builtin-subtitle-export-preset-fcpxml-1080p60",
		Name:          "FCPXML · 1080p · 1/60s",
		Description:   "FCPXML timeline forced to 1920x1080 at 1/60s frame duration.",
		TargetFormat:  "fcpxml",
		MediaStrategy: subtitleExportPresetMediaStrategyFixed,
		Config: SubtitleExportConfig{
			FCPXML: &SubtitleFCPXMLExportConfig{
				FrameDuration:        "1/60s",
				Width:                1920,
				Height:               1080,
				ColorSpace:           "1-1-1 (Rec. 709)",
				Version:              "1.11",
				DefaultLane:          1,
				StartTimecodeSeconds: 0,
			},
		},
	},
	{
		ID:            "builtin-subtitle-export-preset-fcpxml-1080p30",
		Name:          "FCPXML · 1080p · 1/30s",
		Description:   "FCPXML timeline forced to 1920x1080 at 1/30s frame duration.",
		TargetFormat:  "fcpxml",
		MediaStrategy: subtitleExportPresetMediaStrategyFixed,
		Config: SubtitleExportConfig{
			FCPXML: &SubtitleFCPXMLExportConfig{
				FrameDuration:        "1/30s",
				Width:                1920,
				Height:               1080,
				ColorSpace:           "1-1-1 (Rec. 709)",
				Version:              "1.11",
				DefaultLane:          1,
				StartTimecodeSeconds: 0,
			},
		},
	},
}

func defaultSubtitleStyleConfig() SubtitleStyleConfig {
	return SubtitleStyleConfig{
		MonoStyles:            defaultSubtitleMonoStyles(),
		BilingualStyles:       defaultSubtitleBilingualStyles(),
		Sources:               defaultSubtitleStyleSources(),
		Fonts:                 nil,
		SubtitleExportPresets: defaultSubtitleExportPresets(),
		Defaults: SubtitleStyleDefaults{
			MonoStyleID:            "builtin-subtitle-mono-primary-1080p",
			BilingualStyleID:       "builtin-subtitle-bilingual-1080p",
			SubtitleExportPresetID: defaultSubtitleExportPresetID,
		},
	}
}

func defaultSubtitleStyleSources() []SubtitleStyleSource {
	result := make([]SubtitleStyleSource, 0, len(defaultBuiltInSubtitleStyleFontSources))
	result = append(result, defaultBuiltInSubtitleStyleFontSources...)
	return result
}

func defaultSubtitleMonoStyles() []MonoStyle {
	result := make([]MonoStyle, 0, len(defaultBuiltInMonoStyles))
	result = append(result, defaultBuiltInMonoStyles...)
	return result
}

func defaultSubtitleBilingualStyles() []BilingualStyle {
	result := make([]BilingualStyle, 0, len(defaultBuiltInBilingualStyles))
	result = append(result, defaultBuiltInBilingualStyles...)
	return result
}

func defaultSubtitleExportPresets() []SubtitleExportPreset {
	result := make([]SubtitleExportPreset, 0, len(defaultBuiltInSubtitleExportPresets))
	result = append(result, defaultBuiltInSubtitleExportPresets...)
	return result
}
