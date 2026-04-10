package library

import (
	"reflect"
	"testing"
)

func TestDefaultSubtitleStyleConfigIncludesBuiltInFontSources(t *testing.T) {
	t.Parallel()

	config := defaultSubtitleStyleConfig()
	if len(config.MonoStyles) != 2 {
		t.Fatalf("expected 2 built-in mono styles, got %#v", config.MonoStyles)
	}
	if len(config.BilingualStyles) != 1 {
		t.Fatalf("expected 1 built-in bilingual style, got %#v", config.BilingualStyles)
	}
	if config.MonoStyles[0].ID != "builtin-subtitle-mono-primary-1080p" {
		t.Fatalf("expected primary built-in mono style first, got %#v", config.MonoStyles)
	}
	if !config.MonoStyles[0].BuiltIn || !config.MonoStyles[1].BuiltIn {
		t.Fatalf("expected built-in mono styles to be flagged built-in, got %#v", config.MonoStyles)
	}
	if config.MonoStyles[1].ID != "builtin-subtitle-mono-secondary-1080p" {
		t.Fatalf("expected secondary built-in mono style second, got %#v", config.MonoStyles)
	}
	if config.BilingualStyles[0].ID != "builtin-subtitle-bilingual-1080p" {
		t.Fatalf("expected built-in bilingual style, got %#v", config.BilingualStyles)
	}
	if !config.BilingualStyles[0].BuiltIn {
		t.Fatalf("expected built-in bilingual style to be flagged built-in, got %#v", config.BilingualStyles)
	}
	if config.BilingualStyles[0].Primary.SourceMonoStyleID != config.MonoStyles[0].ID {
		t.Fatalf("expected bilingual primary snapshot to reference primary mono style, got %#v", config.BilingualStyles[0])
	}
	if config.BilingualStyles[0].Secondary.SourceMonoStyleID != config.MonoStyles[1].ID {
		t.Fatalf("expected bilingual secondary snapshot to reference secondary mono style, got %#v", config.BilingualStyles[0])
	}
	if config.Defaults.MonoStyleID != config.MonoStyles[0].ID {
		t.Fatalf("expected default mono style id to point at primary built-in style, got %#v", config.Defaults)
	}
	if config.Defaults.BilingualStyleID != config.BilingualStyles[0].ID {
		t.Fatalf("expected default bilingual style id to point at built-in bilingual style, got %#v", config.Defaults)
	}

	if len(config.Sources) < 3 {
		t.Fatalf("expected built-in font sources, got %#v", config.Sources)
	}

	names := map[string]bool{}
	for _, source := range config.Sources {
		if source.Kind != "font" {
			continue
		}
		names[source.Name] = true
	}

	for _, expected := range []string{"Google Fonts", "Nerd Fonts", "Font Squirrel"} {
		if !names[expected] {
			t.Fatalf("expected built-in source %q, got %#v", expected, config.Sources)
		}
	}

	if len(config.SubtitleExportPresets) == 0 {
		t.Fatalf("expected built-in subtitle export presets")
	}
	if config.Defaults.SubtitleExportPresetID == "" {
		t.Fatalf("expected default subtitle export preset id")
	}
	if config.Defaults.SubtitleExportPresetID != "builtin-subtitle-export-preset-srt-auto" {
		t.Fatalf("expected default subtitle export preset to use srt auto, got %q", config.Defaults.SubtitleExportPresetID)
	}
	assCount := 0
	ittCount := 0
	fcpxmlCount := 0
	for _, profile := range config.SubtitleExportPresets {
		if profile.TargetFormat == "ass" {
			assCount++
		}
		if profile.TargetFormat == "itt" {
			ittCount++
		}
		if profile.TargetFormat == "fcpxml" {
			fcpxmlCount++
		}
	}
	if assCount < 3 {
		t.Fatalf("expected built-in ass profiles to include auto + fixed presets, got %#v", config.SubtitleExportPresets)
	}
	if ittCount < 5 {
		t.Fatalf("expected built-in itt profiles to include auto + fixed presets, got %#v", config.SubtitleExportPresets)
	}
	if fcpxmlCount < 5 {
		t.Fatalf("expected built-in fcpxml profiles to include auto + fixed presets, got %#v", config.SubtitleExportPresets)
	}
}

func TestNormalizeSubtitleExportPresetsMapsLegacyASSPresetIDs(t *testing.T) {
	t.Parallel()

	presets := normalizeSubtitleExportPresets([]SubtitleExportPreset{
		{
			ID:            "builtin-subtitle-export-preset-ass-4k60",
			Name:          "ASS · 4K · 60fps",
			TargetFormat:  "ass",
			MediaStrategy: subtitleExportPresetMediaStrategyFixed,
		},
		{
			ID:            "builtin-subtitle-export-preset-ass-1080p30",
			Name:          "ASS · 1080p · 30fps",
			TargetFormat:  "ass",
			MediaStrategy: subtitleExportPresetMediaStrategyFixed,
		},
	}, defaultSubtitleExportPresets())

	idSet := make(map[string]struct{}, len(presets))
	for _, preset := range presets {
		idSet[preset.ID] = struct{}{}
	}

	if _, ok := idSet["builtin-subtitle-export-preset-ass-4k"]; !ok {
		t.Fatalf("expected legacy 4k ass preset id to normalize to builtin-subtitle-export-preset-ass-4k, got %#v", presets)
	}
	if _, ok := idSet["builtin-subtitle-export-preset-ass-1080p"]; !ok {
		t.Fatalf("expected legacy 1080p ass preset id to normalize to builtin-subtitle-export-preset-ass-1080p, got %#v", presets)
	}
	if _, ok := idSet["builtin-subtitle-export-preset-ass-4k60"]; ok {
		t.Fatalf("expected legacy 4k60 ass preset id to be removed, got %#v", presets)
	}
	if _, ok := idSet["builtin-subtitle-export-preset-ass-1080p30"]; ok {
		t.Fatalf("expected legacy 1080p30 ass preset id to be removed, got %#v", presets)
	}
}

func TestNormalizeSubtitleStyleDefaultSubtitleExportPresetIDMapsLegacyASSPresetIDs(t *testing.T) {
	t.Parallel()

	profiles := defaultSubtitleExportPresets()

	if got := normalizeSubtitleStyleDefaultSubtitleExportPresetID(
		"builtin-subtitle-export-preset-ass-4k60",
		defaultSubtitleExportPresetID,
		profiles,
	); got != "builtin-subtitle-export-preset-ass-4k" {
		t.Fatalf("expected legacy 4k ass default id to map to new preset, got %q", got)
	}
	if got := normalizeSubtitleStyleDefaultSubtitleExportPresetID(
		"builtin-subtitle-export-preset-ass-1080p30",
		defaultSubtitleExportPresetID,
		profiles,
	); got != "builtin-subtitle-export-preset-ass-1080p" {
		t.Fatalf("expected legacy 1080p ass default id to map to new preset, got %q", got)
	}
}

func TestNormalizeSubtitleStyleConfigRefreshesPersistedBuiltInSubtitleExportPresets(t *testing.T) {
	t.Parallel()

	defaults := defaultSubtitleExportPresets()
	var builtInPreset SubtitleExportPreset
	for _, preset := range defaults {
		if preset.ID == "builtin-subtitle-export-preset-fcpxml-auto" {
			builtInPreset = preset
			break
		}
	}
	if builtInPreset.ID == "" {
		t.Fatalf("expected built-in fcpxml preset to exist, got %#v", defaults)
	}

	config := normalizeSubtitleStyleConfig(SubtitleStyleConfig{
		SubtitleExportPresets: []SubtitleExportPreset{
			{
				ID:            "builtin-subtitle-export-preset-fcpxml-auto",
				Name:          "FCPXML · Auto",
				Description:   "persisted legacy copy",
				TargetFormat:  "fcpxml",
				MediaStrategy: subtitleExportPresetMediaStrategyAuto,
				Config: SubtitleExportConfig{
					FCPXML: &SubtitleFCPXMLExportConfig{
						FrameDuration:        "1/24s",
						Width:                1920,
						Height:               1080,
						ColorSpace:           "9-9-9",
						Version:              "1.10",
						DefaultLane:          8,
						StartTimecodeSeconds: 3600,
					},
				},
			},
			{
				ID:            "custom-subtitle-export-preset-fcpxml",
				Name:          "Custom FCPXML",
				Description:   "custom preset should remain user-controlled",
				TargetFormat:  "fcpxml",
				MediaStrategy: subtitleExportPresetMediaStrategyFixed,
				Config: SubtitleExportConfig{
					FCPXML: &SubtitleFCPXMLExportConfig{
						FrameDuration:        "1/24s",
						Width:                1920,
						Height:               1080,
						ColorSpace:           "9-9-9",
						Version:              "1.10",
						DefaultLane:          8,
						StartTimecodeSeconds: 3600,
					},
				},
			},
		},
		Defaults: SubtitleStyleDefaults{
			SubtitleExportPresetID: "builtin-subtitle-export-preset-fcpxml-auto",
		},
	})

	var refreshedBuiltIn SubtitleExportPreset
	var customPreset SubtitleExportPreset
	for _, preset := range config.SubtitleExportPresets {
		switch preset.ID {
		case builtInPreset.ID:
			refreshedBuiltIn = preset
		case "custom-subtitle-export-preset-fcpxml":
			customPreset = preset
		}
	}

	if refreshedBuiltIn.ID == "" {
		t.Fatalf("expected built-in fcpxml preset to remain present, got %#v", config.SubtitleExportPresets)
	}
	if !reflect.DeepEqual(refreshedBuiltIn, builtInPreset) {
		t.Fatalf("expected built-in fcpxml preset to be refreshed from code, got %#v want %#v", refreshedBuiltIn, builtInPreset)
	}
	if customPreset.ID == "" {
		t.Fatalf("expected custom fcpxml preset to remain present, got %#v", config.SubtitleExportPresets)
	}
	if customPreset.Config.FCPXML == nil || customPreset.Config.FCPXML.StartTimecodeSeconds != 3600 {
		t.Fatalf("expected custom fcpxml preset config to remain user-controlled, got %#v", customPreset)
	}
}

func TestNormalizeSubtitleStyleSourcesMergesBuiltInFontSources(t *testing.T) {
	t.Parallel()

	sources := normalizeSubtitleStyleSources([]SubtitleStyleSource{
		{
			ID:       "fontget-google-fonts",
			Name:     "Google Fonts Custom",
			Kind:     "font",
			Provider: "fontget",
			URL:      "https://example.com/custom-google.json",
			Prefix:   "google-custom",
			Filename: "custom-google.json",
			Priority: 999,
			BuiltIn:  false,
			Enabled:  false,
			Manifest: SubtitleStyleSourceManifest{
				SourceInfo: SubtitleStyleSourceManifestInfo{
					Name:       "Google Fonts",
					TotalFonts: 321,
				},
				Fonts: map[string]SubtitleStyleSourceManifestFont{
					"roboto": {
						Name:   "Roboto",
						Family: "Roboto",
					},
				},
			},
			FontCount: 321,
		},
		{
			ID:       "custom-font-source",
			Name:     "Studio Fonts",
			Kind:     "font",
			Provider: "fontget",
			URL:      "https://example.com/studio-fonts.json",
			Prefix:   "studio",
			Filename: "studio-fonts.json",
			Priority: 120,
			Enabled:  true,
		},
	})

	if len(sources) != 3 {
		t.Fatalf("expected built-in sources only, got %#v", sources)
	}

	var googleSource SubtitleStyleSource
	for _, source := range sources {
		if source.ID == "fontget-google-fonts" {
			googleSource = source
		}
	}

	if googleSource.ID == "" {
		t.Fatalf("expected built-in google source to remain present")
	}
	if googleSource.URL != "https://raw.githubusercontent.com/Graphixa/FontGet-Sources/main/sources/google-fonts.json" {
		t.Fatalf("expected built-in google source url to be restored, got %q", googleSource.URL)
	}
	if googleSource.BuiltIn != true {
		t.Fatalf("expected built-in google source flag, got %#v", googleSource)
	}
	if googleSource.Enabled != true {
		t.Fatalf("expected built-in enable state to be forced on, got %#v", googleSource)
	}
	if googleSource.FontCount != 321 {
		t.Fatalf("expected built-in metadata to survive normalization, got %#v", googleSource)
	}
	if googleSource.Manifest.SourceInfo.TotalFonts != 321 {
		t.Fatalf("expected built-in manifest info to survive normalization, got %#v", googleSource)
	}
	if googleSource.Manifest.Fonts["roboto"].Family != "Roboto" {
		t.Fatalf("expected built-in manifest fonts to survive normalization, got %#v", googleSource)
	}
}

func TestNormalizeSubtitleStyleSourcesRejectsLegacyGitHubFontFields(t *testing.T) {
	t.Parallel()

	sources := normalizeSubtitleStyleSources([]SubtitleStyleSource{{
		ID:           "legacy-font-source",
		Name:         "Legacy Font Source",
		Kind:         "font",
		Provider:     "github",
		Owner:        "example",
		Repo:         "fonts",
		Ref:          "main",
		ManifestPath: "fonts.json",
		Enabled:      true,
	}})

	if len(sources) != 3 {
		t.Fatalf("expected built-in font sources only, got %#v", sources)
	}

	for _, source := range sources {
		if source.ID == "legacy-font-source" {
			t.Fatalf("expected legacy font source to be removed, got %#v", sources)
		}
	}
}

func TestNormalizeSubtitleStyleConfigNormalizesSubtitleExportPresetDefaults(t *testing.T) {
	t.Parallel()

	config := normalizeSubtitleStyleConfig(SubtitleStyleConfig{
		SubtitleExportPresets: []SubtitleExportPreset{
			{
				ID:           "profile-xml",
				Name:         "XML profile",
				TargetFormat: "xml",
			},
			{
				ID:           "profile-ass",
				Name:         "ASS profile",
				TargetFormat: "ass",
			},
		},
		Defaults: SubtitleStyleDefaults{
			SubtitleExportPresetID: "profile-missing",
		},
	})

	if len(config.SubtitleExportPresets) < 3 {
		t.Fatalf("expected normalized subtitle export presets to include custom and built-ins, got %#v", config.SubtitleExportPresets)
	}
	var xmlProfile SubtitleExportPreset
	for _, profile := range config.SubtitleExportPresets {
		if profile.ID == "profile-xml" {
			xmlProfile = profile
			break
		}
	}
	if xmlProfile.ID == "" {
		t.Fatalf("expected xml profile to remain after normalization, got %#v", config.SubtitleExportPresets)
	}
	if xmlProfile.TargetFormat != "itt" {
		t.Fatalf("expected xml target format to normalize to itt, got %#v", xmlProfile)
	}
	if config.Defaults.SubtitleExportPresetID == "profile-missing" || config.Defaults.SubtitleExportPresetID == "" {
		t.Fatalf("expected default subtitle export preset to fallback from missing preset, got %#v", config.Defaults)
	}
}

func TestNormalizeSubtitleStyleConfigFallsBackToBuiltInStylePresets(t *testing.T) {
	t.Parallel()

	config := normalizeSubtitleStyleConfig(SubtitleStyleConfig{})

	if len(config.MonoStyles) != 2 {
		t.Fatalf("expected built-in mono styles fallback, got %#v", config.MonoStyles)
	}
	if len(config.BilingualStyles) != 1 {
		t.Fatalf("expected built-in bilingual style fallback, got %#v", config.BilingualStyles)
	}

	primary := config.MonoStyles[0]
	secondary := config.MonoStyles[1]
	bilingual := config.BilingualStyles[0]

	if primary.BasePlayResX != 1920 || primary.BasePlayResY != 1080 || primary.BaseAspectRatio != SubtitleStyleAspectRatio16By9 {
		t.Fatalf("expected primary mono style to target 1920x1080 16:9, got %#v", primary)
	}
	if primary.Style.Fontname != "PingFang SC" || primary.Style.FontFace != "Semibold" || primary.Style.FontWeight != 600 {
		t.Fatalf("expected primary mono style to use the bundled PingFang primary preset, got %#v", primary.Style)
	}
	if primary.Style.FontPostScriptName != "PingFangSC-Semibold" || primary.Style.Fontsize != 52 {
		t.Fatalf("expected primary mono style font metadata to match workspace-derived preset, got %#v", primary.Style)
	}
	if primary.Style.Alignment != 2 || primary.Style.MarginV != 72 || primary.Style.Outline != 0 || primary.Style.Shadow != 5 {
		t.Fatalf("expected primary mono style bottom-center shadow preset, got %#v", primary.Style)
	}
	if secondary.Style.Fontname != "PingFang SC" || secondary.Style.FontFace != "Regular" || secondary.Style.FontWeight != 400 {
		t.Fatalf("expected secondary mono style to use matching PingFang companion preset, got %#v", secondary.Style)
	}
	if secondary.Style.FontPostScriptName != "PingFangSC-Regular" || secondary.Style.Fontsize != 40 {
		t.Fatalf("expected secondary mono style to use smaller companion size, got %#v", secondary.Style)
	}
	if secondary.Style.MarginV != 72 || secondary.Style.Outline != 0 || secondary.Style.Shadow != 5 {
		t.Fatalf("expected secondary mono style to share the same shadow positioning, got %#v", secondary.Style)
	}
	if bilingual.Layout.BlockAnchor != 2 || bilingual.Layout.Gap != 20 {
		t.Fatalf("expected built-in bilingual layout to compose bottom-center mono styles, got %#v", bilingual.Layout)
	}
	if bilingual.Primary.SourceMonoStyleID != primary.ID || bilingual.Secondary.SourceMonoStyleID != secondary.ID {
		t.Fatalf("expected built-in bilingual style to reference built-in mono styles, got %#v", bilingual)
	}
	if bilingual.Primary.Style.FontPostScriptName != primary.Style.FontPostScriptName || bilingual.Secondary.Style.FontPostScriptName != secondary.Style.FontPostScriptName {
		t.Fatalf("expected built-in bilingual snapshots to embed the built-in mono style presets, got %#v", bilingual)
	}
	if bilingual.Primary.Style.MarginV != 72 || bilingual.Secondary.Style.MarginV != 72 {
		t.Fatalf("expected built-in bilingual snapshots to carry the updated 72px vertical margins, got %#v", bilingual)
	}
	if config.Defaults.MonoStyleID != primary.ID || config.Defaults.BilingualStyleID != bilingual.ID {
		t.Fatalf("expected built-in style defaults to resolve to built-in ids, got %#v", config.Defaults)
	}
}

func TestNormalizeSubtitleStyleConfigRefreshesPersistedBuiltInStylePresets(t *testing.T) {
	t.Parallel()

	config := normalizeSubtitleStyleConfig(SubtitleStyleConfig{
		MonoStyles: []MonoStyle{
			{
				ID:                 "builtin-subtitle-mono-primary-1080p",
				Name:               "1080p Primary",
				BuiltIn:            true,
				BasePlayResX:       1920,
				BasePlayResY:       1080,
				BaseAspectRatio:    SubtitleStyleAspectRatio16By9,
				SourceAssStyleName: "Primary",
				Style: AssStyleSpec{
					Fontname:        "Arial",
					Fontsize:        56,
					PrimaryColour:   "&H00FFFFFF",
					SecondaryColour: "&H00FFFFFF",
					OutlineColour:   "&H00101010",
					BackColour:      "&H80000000",
					BorderStyle:     1,
					Outline:         2.6,
					Shadow:          0.8,
					Alignment:       2,
					MarginL:         72,
					MarginR:         72,
					MarginV:         56,
					Encoding:        1,
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
					Fontname:        "Arial",
					Fontsize:        40,
					PrimaryColour:   "&H00E8E8E8",
					SecondaryColour: "&H00E8E8E8",
					OutlineColour:   "&H00101010",
					BackColour:      "&H80000000",
					BorderStyle:     1,
					Outline:         2.2,
					Shadow:          0.6,
					Alignment:       2,
					MarginL:         72,
					MarginR:         72,
					MarginV:         56,
					Encoding:        1,
				},
			},
		},
		BilingualStyles: []BilingualStyle{{
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
					Fontname: "Arial",
					Fontsize: 56,
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
					Fontname: "Arial",
					Fontsize: 40,
				},
			},
			Layout: BilingualLayout{
				Gap:         20,
				BlockAnchor: 2,
			},
		}},
		Defaults: SubtitleStyleDefaults{
			MonoStyleID:      "builtin-subtitle-mono-primary-1080p",
			BilingualStyleID: "builtin-subtitle-bilingual-1080p",
		},
	})

	primary := config.MonoStyles[0]
	secondary := config.MonoStyles[1]
	bilingual := config.BilingualStyles[0]

	if primary.Style.Fontname != "PingFang SC" || primary.Style.FontPostScriptName != "PingFangSC-Semibold" || primary.Style.Outline != 0 || primary.Style.Shadow != 5 || primary.Style.MarginV != 72 {
		t.Fatalf("expected persisted built-in primary style to refresh to latest preset, got %#v", primary.Style)
	}
	if secondary.Style.Fontname != "PingFang SC" || secondary.Style.FontPostScriptName != "PingFangSC-Regular" || secondary.Style.Outline != 0 || secondary.Style.Shadow != 5 || secondary.Style.MarginV != 72 {
		t.Fatalf("expected persisted built-in secondary style to refresh to latest preset, got %#v", secondary.Style)
	}
	if bilingual.Primary.Style.FontPostScriptName != "PingFangSC-Semibold" || bilingual.Secondary.Style.FontPostScriptName != "PingFangSC-Regular" {
		t.Fatalf("expected persisted built-in bilingual style to refresh to latest snapshots, got %#v", bilingual)
	}
	if bilingual.Primary.Style.MarginV != 72 || bilingual.Secondary.Style.MarginV != 72 {
		t.Fatalf("expected persisted built-in bilingual style to refresh the updated vertical margins, got %#v", bilingual)
	}
}

func TestNormalizeSubtitleStyleConfigNormalizesSubtitleStyleDefaults(t *testing.T) {
	t.Parallel()

	config := normalizeSubtitleStyleConfig(SubtitleStyleConfig{
		MonoStyles: []MonoStyle{{
			ID:              "custom-mono",
			Name:            "Custom Mono",
			BasePlayResX:    1920,
			BasePlayResY:    1080,
			BaseAspectRatio: SubtitleStyleAspectRatio16By9,
			Style: AssStyleSpec{
				Fontname: "Arial",
				Fontsize: 48,
			},
		}},
		BilingualStyles: []BilingualStyle{{
			ID:              "custom-bilingual",
			Name:            "Custom Bilingual",
			BasePlayResX:    1920,
			BasePlayResY:    1080,
			BaseAspectRatio: SubtitleStyleAspectRatio16By9,
			Primary: MonoStyleSnapshot{
				SourceMonoStyleID: "custom-mono",
				Name:              "Primary",
				BasePlayResX:      1920,
				BasePlayResY:      1080,
				BaseAspectRatio:   SubtitleStyleAspectRatio16By9,
				Style: AssStyleSpec{
					Fontname: "Arial",
					Fontsize: 48,
				},
			},
			Secondary: MonoStyleSnapshot{
				SourceMonoStyleID: "custom-mono",
				Name:              "Secondary",
				BasePlayResX:      1920,
				BasePlayResY:      1080,
				BaseAspectRatio:   SubtitleStyleAspectRatio16By9,
				Style: AssStyleSpec{
					Fontname: "Arial",
					Fontsize: 40,
				},
			},
		}},
		Defaults: SubtitleStyleDefaults{
			MonoStyleID:      "missing-mono",
			BilingualStyleID: "missing-bilingual",
		},
	})

	if config.Defaults.MonoStyleID != "custom-mono" {
		t.Fatalf("expected default mono style id to fallback to available mono style, got %#v", config.Defaults)
	}
	if config.Defaults.BilingualStyleID != "custom-bilingual" {
		t.Fatalf("expected default bilingual style id to fallback to available bilingual style, got %#v", config.Defaults)
	}
}
