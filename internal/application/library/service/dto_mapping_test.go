package service

import (
	"encoding/json"
	"testing"
	"time"

	"dreamcreator/internal/application/library/dto"
	"dreamcreator/internal/domain/library"
)

func TestToOperationDTOBuildsDownloadRequestPreview(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC)
	inputJSON, err := json.Marshal(dto.CreateYTDLPJobRequest{
		URL:          "https://example.com/watch?v=1",
		Caller:       "library.page",
		Extractor:    "youtube",
		Author:       "creator",
		ThumbnailURL: "https://example.com/thumb.jpg",
	})
	if err != nil {
		t.Fatalf("marshal input: %v", err)
	}

	item, err := library.NewLibraryOperation(library.LibraryOperationParams{
		ID:          "op-1",
		LibraryID:   "lib-1",
		Kind:        "download",
		Status:      string(library.OperationStatusSucceeded),
		DisplayName: "Example",
		InputJSON:   string(inputJSON),
		OutputJSON:  "{}",
		Meta: library.OperationMeta{
			Platform: "YouTube",
			Uploader: "Creator Name",
		},
		CreatedAt:  &now,
		StartedAt:  &now,
		FinishedAt: &now,
	})
	if err != nil {
		t.Fatalf("new operation: %v", err)
	}

	got := toOperationDTO(item)
	if got.Request == nil {
		t.Fatalf("expected request preview")
	}
	if got.Request.URL != "https://example.com/watch?v=1" {
		t.Fatalf("expected url in preview, got %q", got.Request.URL)
	}
	if got.Request.Caller != "library.page" {
		t.Fatalf("expected caller in preview, got %q", got.Request.Caller)
	}
	if got.Request.Extractor != "YouTube" {
		t.Fatalf("expected meta extractor to win, got %q", got.Request.Extractor)
	}
	if got.Request.Author != "Creator Name" {
		t.Fatalf("expected meta author to win, got %q", got.Request.Author)
	}
	if got.Request.ThumbnailURL != "https://example.com/thumb.jpg" {
		t.Fatalf("expected thumbnail url in preview, got %q", got.Request.ThumbnailURL)
	}
}

func TestToOperationListItemDTOIncludesLibraryProjectionFields(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC)
	finished := now.Add(2 * time.Minute)
	durationMs := int64((2 * time.Minute).Milliseconds())
	item, err := library.NewLibraryOperation(library.LibraryOperationParams{
		ID:           "op-1",
		LibraryID:    "lib-1",
		Kind:         "download",
		Status:       string(library.OperationStatusSucceeded),
		DisplayName:  "Example",
		InputJSON:    "{}",
		OutputJSON:   "{}",
		SourceDomain: "youtube.com",
		SourceIcon:   "https://example.com/icon.png",
		Metrics:      library.OperationMetrics{FileCount: 1, DurationMs: &durationMs},
		CreatedAt:    &now,
		StartedAt:    &now,
		FinishedAt:   &finished,
	})
	if err != nil {
		t.Fatalf("new operation: %v", err)
	}

	got := toOperationListItemDTO(item, "Library A")
	if got.LibraryName != "Library A" {
		t.Fatalf("expected library name, got %q", got.LibraryName)
	}
	if got.SourceIcon != "https://example.com/icon.png" {
		t.Fatalf("expected source icon, got %q", got.SourceIcon)
	}
	if got.FinishedAt != finished.Format(time.RFC3339) {
		t.Fatalf("expected finishedAt, got %q", got.FinishedAt)
	}
	if got.Metrics.DurationMs == nil || *got.Metrics.DurationMs != durationMs {
		t.Fatalf("expected durationMs in metrics, got %#v", got.Metrics.DurationMs)
	}
}

func TestToOperationDTOIncludesPrimaryOutputFlag(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC)
	item, err := library.NewLibraryOperation(library.LibraryOperationParams{
		ID:          "op-1",
		LibraryID:   "lib-1",
		Kind:        "download",
		Status:      string(library.OperationStatusSucceeded),
		DisplayName: "Example",
		InputJSON:   "{}",
		OutputJSON:  "{}",
		OutputFiles: []library.OperationOutputFile{
			{FileID: "file-1", Kind: "video", Format: "mp4", IsPrimary: true},
			{FileID: "file-2", Kind: "subtitle", Format: "srt"},
		},
		CreatedAt: &now,
	})
	if err != nil {
		t.Fatalf("new operation: %v", err)
	}

	got := toOperationDTO(item)
	if len(got.OutputFiles) != 2 {
		t.Fatalf("expected 2 output files, got %d", len(got.OutputFiles))
	}
	if !got.OutputFiles[0].IsPrimary {
		t.Fatalf("expected first output to keep isPrimary=true")
	}
	if got.OutputFiles[1].IsPrimary {
		t.Fatalf("expected second output to keep isPrimary=false")
	}
}

func TestToOperationDTOSanitizesTerminalProgressMessage(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 23, 10, 0, 0, 0, time.UTC)
	progressPercent := 42
	progressCurrent := int64(42)
	progressTotal := int64(100)
	item, err := library.NewLibraryOperation(library.LibraryOperationParams{
		ID:          "op-1",
		LibraryID:   "lib-1",
		Kind:        "download",
		Status:      string(library.OperationStatusFailed),
		DisplayName: "Example",
		InputJSON:   "{}",
		OutputJSON:  "{}",
		Progress: &library.OperationProgress{
			Stage:     progressText("library.status.failed"),
			Percent:   &progressPercent,
			Current:   &progressCurrent,
			Total:     &progressTotal,
			Speed:     "1.2 MiB/s",
			Message:   "HTTP 403 Forbidden: login required",
			UpdatedAt: now.Format(time.RFC3339),
		},
		ErrorCode:    "auth_required",
		ErrorMessage: "HTTP 403 Forbidden: login required",
		CreatedAt:    &now,
		StartedAt:    &now,
		FinishedAt:   &now,
	})
	if err != nil {
		t.Fatalf("new operation: %v", err)
	}

	got := toOperationDTO(item)
	if got.Progress == nil {
		t.Fatalf("expected progress dto")
	}
	if got.Progress.Message != progressText("library.progressDetail.downloadFailed") {
		t.Fatalf("expected sanitized terminal progress message, got %q", got.Progress.Message)
	}
	if got.Progress.Speed != "1.2 MiB/s" {
		t.Fatalf("expected speed to be preserved, got %q", got.Progress.Speed)
	}
}

func TestToModuleConfigDTOIncludesTranslateLanguages(t *testing.T) {
	t.Parallel()

	config := library.DefaultModuleConfig()
	config.TranslateLanguages.Custom = []library.TranslateLanguage{
		{Code: "yue", Label: "Cantonese", Aliases: []string{"cantonese", "zh-yue"}},
	}

	got := toModuleConfigDTO(config)
	if len(got.TranslateLanguages.Builtin) == 0 {
		t.Fatalf("expected builtin translate languages")
	}
	if len(got.TranslateLanguages.Custom) != 1 {
		t.Fatalf("expected 1 custom translate language, got %d", len(got.TranslateLanguages.Custom))
	}
	if got.TranslateLanguages.Custom[0].Code != "yue" {
		t.Fatalf("expected custom code yue, got %q", got.TranslateLanguages.Custom[0].Code)
	}
	if got.TranslateLanguages.Custom[0].Label != "Cantonese" {
		t.Fatalf("expected custom label Cantonese, got %q", got.TranslateLanguages.Custom[0].Label)
	}
}

func TestToModuleConfigDTOIncludesLanguageAssetsAndRuntime(t *testing.T) {
	t.Parallel()

	config := library.DefaultModuleConfig()
	config.LanguageAssets.GlossaryProfiles = []library.GlossaryProfile{{
		ID:             "brand-core",
		Name:           "Brand Core",
		SourceLanguage: "en",
		TargetLanguage: "zh-CN",
		Terms: []library.GlossaryTerm{{
			Source: "DreamCreator",
			Target: "梦创作",
			Note:   "Keep product naming consistent.",
		}},
	}}
	config.LanguageAssets.PromptProfiles = []library.PromptProfile{{
		ID:       "proofread-tight",
		Name:     "Proofread Tight",
		Category: "proofread",
		Prompt:   "Keep tone formal and concise.",
	}}
	config.TaskRuntime.Translate.StructuredOutputMode = "json_schema"
	config.TaskRuntime.Translate.ThinkingMode = "off"
	config.TaskRuntime.Translate.MaxTokensCeiling = 8192

	got := toModuleConfigDTO(config)
	if len(got.LanguageAssets.GlossaryProfiles) != 1 {
		t.Fatalf("expected 1 glossary profile, got %d", len(got.LanguageAssets.GlossaryProfiles))
	}
	if got.LanguageAssets.GlossaryProfiles[0].Terms[0].Target != "梦创作" {
		t.Fatalf("expected glossary target to round-trip, got %q", got.LanguageAssets.GlossaryProfiles[0].Terms[0].Target)
	}
	if got.TaskRuntime.Translate.StructuredOutputMode != "json_schema" {
		t.Fatalf("expected translate structured output mode to round-trip, got %q", got.TaskRuntime.Translate.StructuredOutputMode)
	}
	if got.TaskRuntime.Translate.MaxTokensCeiling != 8192 {
		t.Fatalf("expected translate runtime max token ceiling 8192, got %d", got.TaskRuntime.Translate.MaxTokensCeiling)
	}
}

func TestToDomainModuleConfigNormalizesLanguageAssetsAndRuntime(t *testing.T) {
	t.Parallel()

	config := dto.LibraryModuleConfigDTO{
		Workspace: dto.LibraryWorkspaceConfigDTO{FastReadLatestState: true},
		TranslateLanguages: dto.LibraryTranslateLanguagesConfigDTO{
			Builtin: toTranslateLanguageDTOs(library.DefaultModuleConfig().TranslateLanguages.Builtin),
			Custom:  nil,
		},
		LanguageAssets: dto.LibraryLanguageAssetsConfigDTO{
			GlossaryProfiles: []dto.LibraryGlossaryProfileDTO{{
				Name:           "Brand Core",
				Category:       "all",
				SourceLanguage: "all",
				TargetLanguage: "all",
				Terms: []dto.LibraryGlossaryTermDTO{{
					Source: "DreamCreator",
					Target: "梦创作",
				}},
			}},
			PromptProfiles: []dto.LibraryPromptProfileDTO{{
				Name:     "Translate Style",
				Category: "all",
				Prompt:   "Prefer concise subtitle wording.",
			}},
		},
		TaskRuntime: dto.LibraryTaskRuntimeConfigDTO{
			Translate: dto.LibraryTaskRuntimeSettingsDTO{
				StructuredOutputMode: "json_schema",
				ThinkingMode:         "off",
				MaxTokensFloor:       2048,
				MaxTokensCeiling:     8192,
				RetryTokenStep:       1024,
			},
			Proofread: dto.LibraryTaskRuntimeSettingsDTO{
				StructuredOutputMode: "prompt_only",
				ThinkingMode:         "minimal",
				MaxTokensFloor:       1024,
				MaxTokensCeiling:     4096,
				RetryTokenStep:       512,
			},
		},
	}

	got := toDomainModuleConfig(config)
	if len(got.LanguageAssets.GlossaryProfiles) != 1 {
		t.Fatalf("expected 1 glossary profile, got %d", len(got.LanguageAssets.GlossaryProfiles))
	}
	if got.LanguageAssets.GlossaryProfiles[0].ID == "" {
		t.Fatalf("expected glossary profile id to be normalized")
	}
	if got.LanguageAssets.GlossaryProfiles[0].Category != "all" {
		t.Fatalf("expected glossary category all, got %q", got.LanguageAssets.GlossaryProfiles[0].Category)
	}
	if got.LanguageAssets.GlossaryProfiles[0].SourceLanguage != "all" {
		t.Fatalf("expected glossary source language all, got %q", got.LanguageAssets.GlossaryProfiles[0].SourceLanguage)
	}
	if got.LanguageAssets.GlossaryProfiles[0].TargetLanguage != "all" {
		t.Fatalf("expected glossary target language all, got %q", got.LanguageAssets.GlossaryProfiles[0].TargetLanguage)
	}
	if got.LanguageAssets.PromptProfiles[0].Category != "all" {
		t.Fatalf("expected prompt category all, got %q", got.LanguageAssets.PromptProfiles[0].Category)
	}
	if got.TaskRuntime.Translate.StructuredOutputMode != "json_schema" {
		t.Fatalf("expected normalized translate structured output mode, got %q", got.TaskRuntime.Translate.StructuredOutputMode)
	}
	if got.TaskRuntime.Proofread.ThinkingMode != "on" {
		t.Fatalf("expected normalized proofread thinking mode on, got %q", got.TaskRuntime.Proofread.ThinkingMode)
	}
	if got.TaskRuntime.Proofread.RetryTokenStep != 512 {
		t.Fatalf("expected normalized proofread retry token step 512, got %d", got.TaskRuntime.Proofread.RetryTokenStep)
	}
}

func TestToModuleConfigDTOExposesSubtitleStyleDefaults(t *testing.T) {
	t.Parallel()

	config := library.DefaultModuleConfig()

	got := toModuleConfigDTO(config)
	if got.SubtitleStyles.Defaults.MonoStyleID == "" {
		t.Fatalf("expected subtitle style default mono style id, got %#v", got.SubtitleStyles.Defaults)
	}
	if got.SubtitleStyles.Defaults.BilingualStyleID == "" {
		t.Fatalf("expected subtitle style default bilingual style id, got %#v", got.SubtitleStyles.Defaults)
	}
	if got.SubtitleStyles.Defaults.SubtitleExportPresetID == "" {
		t.Fatalf("expected subtitle style default export preset id, got %#v", got.SubtitleStyles.Defaults)
	}
	if len(got.SubtitleStyles.MonoStyles) == 0 || !got.SubtitleStyles.MonoStyles[0].BuiltIn {
		t.Fatalf("expected built-in mono styles to be exposed in dto, got %#v", got.SubtitleStyles.MonoStyles)
	}
	if len(got.SubtitleStyles.BilingualStyles) == 0 || !got.SubtitleStyles.BilingualStyles[0].BuiltIn {
		t.Fatalf("expected built-in bilingual styles to be exposed in dto, got %#v", got.SubtitleStyles.BilingualStyles)
	}
	if len(got.SubtitleStyles.SubtitleExportPresets) == 0 {
		t.Fatalf("expected subtitle style export presets")
	}
}

func TestToModuleConfigDTOPreservesSubtitleStyleFonts(t *testing.T) {
	t.Parallel()

	config := library.DefaultModuleConfig()
	config.SubtitleStyles.Fonts = []library.SubtitleStyleFont{{
		ID:           "font-1",
		Family:       "Arial",
		Source:       "system",
		SystemFamily: "Arial",
		Enabled:      true,
	}}

	got := toModuleConfigDTO(config)
	if len(got.SubtitleStyles.Fonts) != 1 {
		t.Fatalf("expected subtitle style font mappings, got %#v", got.SubtitleStyles.Fonts)
	}
	if got.SubtitleStyles.Fonts[0].Family != "Arial" || got.SubtitleStyles.Fonts[0].SystemFamily != "Arial" {
		t.Fatalf("expected preserved font mapping, got %#v", got.SubtitleStyles.Fonts[0])
	}
}

func TestToModuleConfigDTOPreservesSubtitleStyleSources(t *testing.T) {
	t.Parallel()

	config := library.DefaultModuleConfig()
	config.SubtitleStyles.Sources = []library.SubtitleStyleSource{{
		ID:       "fontget-google-fonts",
		Name:     "Google Fonts",
		Kind:     "font",
		Provider: "fontget",
		URL:      "https://example.com/google-fonts.json",
		Prefix:   "google",
		Filename: "google-fonts.json",
		Priority: 1,
		BuiltIn:  true,
		Manifest: library.SubtitleStyleSourceManifest{
			SourceInfo: library.SubtitleStyleSourceManifestInfo{
				Name:       "Google Fonts",
				TotalFonts: 1200,
			},
			Fonts: map[string]library.SubtitleStyleSourceManifestFont{
				"roboto": {
					Name:   "Roboto",
					Family: "Roboto",
				},
			},
		},
		Enabled:   true,
		FontCount: 1200,
	}}

	got := toModuleConfigDTO(config)
	if len(got.SubtitleStyles.Sources) != 1 {
		t.Fatalf("expected subtitle style sources, got %#v", got.SubtitleStyles.Sources)
	}
	if got.SubtitleStyles.Sources[0].URL != "https://example.com/google-fonts.json" {
		t.Fatalf("expected source url to survive mapping, got %#v", got.SubtitleStyles.Sources[0])
	}
	if got.SubtitleStyles.Sources[0].Prefix != "google" || got.SubtitleStyles.Sources[0].Priority != 1 || !got.SubtitleStyles.Sources[0].BuiltIn {
		t.Fatalf("expected source config fields to survive mapping, got %#v", got.SubtitleStyles.Sources[0])
	}
	if got.SubtitleStyles.Sources[0].FontCount != 1200 {
		t.Fatalf("expected source font count to survive mapping, got %#v", got.SubtitleStyles.Sources[0])
	}
	if got.SubtitleStyles.Sources[0].RemoteFontManifest.SourceInfo.TotalFonts != 1200 {
		t.Fatalf("expected source manifest info to survive mapping, got %#v", got.SubtitleStyles.Sources[0])
	}
	if got.SubtitleStyles.Sources[0].RemoteFontManifest.Fonts["roboto"].Family != "Roboto" {
		t.Fatalf("expected source manifest fonts to survive mapping, got %#v", got.SubtitleStyles.Sources[0])
	}
}

func TestToModuleConfigDTOPreservesSubtitleExportPresets(t *testing.T) {
	t.Parallel()

	config := library.DefaultModuleConfig()
	config.SubtitleStyles.SubtitleExportPresets = []library.SubtitleExportPreset{{
		ID:            "fcpxml-custom",
		Name:          "FCPXML 1001/30000s",
		Description:   "Custom timeline profile",
		TargetFormat:  "fcpxml",
		MediaStrategy: "fixed",
		Config: library.SubtitleExportConfig{
			FCPXML: &library.SubtitleFCPXMLExportConfig{
				FrameDuration: "1001/30000s",
				Width:         1280,
				Height:        720,
			},
		},
	}}
	config.SubtitleStyles.Defaults.SubtitleExportPresetID = "fcpxml-custom"

	got := toModuleConfigDTO(config)
	if len(got.SubtitleStyles.SubtitleExportPresets) != 1 {
		t.Fatalf("expected subtitle export presets, got %#v", got.SubtitleStyles.SubtitleExportPresets)
	}
	profile := got.SubtitleStyles.SubtitleExportPresets[0]
	if profile.Format != "fcpxml" {
		t.Fatalf("expected profile format field to survive mapping, got %#v", profile)
	}
	if profile.TargetFormat != "fcpxml" || profile.Config.FCPXML == nil {
		t.Fatalf("expected profile format/config to survive mapping, got %#v", profile)
	}
	if profile.Config.FCPXML.FrameDuration != "1001/30000s" || profile.Config.FCPXML.Width != 1280 || profile.Config.FCPXML.Height != 720 {
		t.Fatalf("expected fcpxml profile config to survive mapping, got %#v", profile.Config.FCPXML)
	}
	if profile.MediaStrategy != "fixed" {
		t.Fatalf("expected media strategy fixed to survive mapping, got %#v", profile)
	}
	if got.SubtitleStyles.Defaults.SubtitleExportPresetID != "fcpxml-custom" {
		t.Fatalf("expected default export preset id to survive mapping, got %#v", got.SubtitleStyles.Defaults)
	}
}

func TestToDomainModuleConfigNormalizesSubtitleStyleConfigPayload(t *testing.T) {
	t.Parallel()

	config := dto.LibraryModuleConfigDTO{
		SubtitleStyles: dto.LibrarySubtitleStyleConfigDTO{
			MonoStyles: []dto.LibraryMonoStyleDTO{{
				ID:              "custom-mono-style",
				Name:            "Custom Mono Style",
				BuiltIn:         false,
				BasePlayResX:    1920,
				BasePlayResY:    1080,
				BaseAspectRatio: "16:9",
				Style: dto.AssStyleSpecDTO{
					Fontname:      "Arial",
					Fontsize:      48,
					PrimaryColour: "&H00FFFFFF",
					OutlineColour: "&H00111111",
					BackColour:    "&H80000000",
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
			}},
			BilingualStyles: []dto.LibraryBilingualStyleDTO{{
				ID:              "custom-bilingual-style",
				Name:            "Custom Bilingual Style",
				BuiltIn:         false,
				BasePlayResX:    1920,
				BasePlayResY:    1080,
				BaseAspectRatio: "16:9",
				Primary: dto.LibraryMonoStyleSnapshotDTO{
					SourceMonoStyleID:   "custom-mono-style",
					SourceMonoStyleName: "Custom Mono Style",
					Name:                "Primary",
					BasePlayResX:        1920,
					BasePlayResY:        1080,
					BaseAspectRatio:     "16:9",
					Style: dto.AssStyleSpecDTO{
						Fontname:      "Arial",
						Fontsize:      48,
						PrimaryColour: "&H00FFFFFF",
						OutlineColour: "&H00111111",
						BackColour:    "&H80000000",
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
					SourceMonoStyleID:   "custom-mono-style",
					SourceMonoStyleName: "Custom Mono Style",
					Name:                "Secondary",
					BasePlayResX:        1920,
					BasePlayResY:        1080,
					BaseAspectRatio:     "16:9",
					Style: dto.AssStyleSpecDTO{
						Fontname:      "Arial",
						Fontsize:      40,
						PrimaryColour: "&H00E8E8E8",
						OutlineColour: "&H00111111",
						BackColour:    "&H80000000",
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
					Gap:         20,
					BlockAnchor: 2,
				},
			}},
			Fonts: []dto.LibrarySubtitleStyleFontDTO{{
				ID:           "font-1",
				Family:       "Arial",
				Source:       "system",
				SystemFamily: "Arial",
				Enabled:      true,
			}},
			Sources: []dto.LibrarySubtitleStyleSourceDTO{{
				ID:        "fontget-google-fonts",
				Name:      "Google Fonts",
				Kind:      "font",
				Provider:  "fontget",
				URL:       "https://example.com/google-fonts.json",
				Prefix:    "google",
				Filename:  "google-fonts.json",
				Priority:  1,
				Enabled:   true,
				FontCount: 42,
				RemoteFontManifest: dto.LibraryRemoteFontManifestDTO{
					SourceInfo: dto.LibraryRemoteFontManifestInfoDTO{
						Name:       "Google Fonts",
						TotalFonts: 42,
					},
					Fonts: map[string]dto.LibraryRemoteFontManifestFontDTO{
						"roboto": {
							Name:   "Roboto",
							Family: "Roboto",
						},
					},
				},
			}},
			Defaults: dto.LibrarySubtitleStyleDefaultsDTO{
				MonoStyleID:            "custom-mono-style",
				BilingualStyleID:       "custom-bilingual-style",
				SubtitleExportPresetID: "custom-export-profile",
			},
			SubtitleExportPresets: []dto.LibrarySubtitleExportPresetDTO{{
				ID:            "custom-export-profile",
				Name:          "Custom FCPXML",
				Format:        "xml",
				TargetFormat:  "xml",
				MediaStrategy: "fixed",
				Config: dto.SubtitleExportConfig{
					FCPXML: &dto.SubtitleFCPXMLExportConfig{
						FrameDuration: "1/25s",
						Width:         1280,
						Height:        720,
					},
				},
			}},
		},
	}

	got := toDomainModuleConfig(config)
	if got.SubtitleStyles.Defaults.MonoStyleID != "custom-mono-style" {
		t.Fatalf("expected normalized default mono style id, got %#v", got.SubtitleStyles.Defaults)
	}
	if got.SubtitleStyles.Defaults.BilingualStyleID != "custom-bilingual-style" {
		t.Fatalf("expected normalized default bilingual style id, got %#v", got.SubtitleStyles.Defaults)
	}
	if got.SubtitleStyles.Defaults.SubtitleExportPresetID != "custom-export-profile" {
		t.Fatalf("expected normalized default export preset id, got %#v", got.SubtitleStyles.Defaults)
	}
	if len(got.SubtitleStyles.Fonts) != 1 {
		t.Fatalf("expected subtitle style font mappings to survive normalization, got %#v", got.SubtitleStyles.Fonts)
	}
	if len(got.SubtitleStyles.Sources) == 0 {
		t.Fatalf("expected subtitle style sources to survive normalization, got %#v", got.SubtitleStyles.Sources)
	}
	if got.SubtitleStyles.Sources[0].Manifest.SourceInfo.TotalFonts != 42 {
		t.Fatalf("expected source manifest info to survive normalization, got %#v", got.SubtitleStyles.Sources[0])
	}
	if got.SubtitleStyles.Sources[0].Manifest.Fonts["roboto"].Family != "Roboto" {
		t.Fatalf("expected source manifest fonts to survive normalization, got %#v", got.SubtitleStyles.Sources[0])
	}
	if got.SubtitleStyles.Fonts[0].Family != "Arial" || got.SubtitleStyles.Fonts[0].SystemFamily != "Arial" {
		t.Fatalf("expected normalized subtitle style font mapping, got %#v", got.SubtitleStyles.Fonts[0])
	}
	if len(got.SubtitleStyles.Sources) != 3 {
		t.Fatalf("expected normalized subtitle style sources to keep built-ins only, got %#v", got.SubtitleStyles.Sources)
	}
	if len(got.SubtitleStyles.SubtitleExportPresets) < 2 {
		t.Fatalf("expected subtitle export presets to include custom and built-ins, got %#v", got.SubtitleStyles.SubtitleExportPresets)
	}
	customProfile := library.SubtitleExportPreset{}
	for _, profile := range got.SubtitleStyles.SubtitleExportPresets {
		if profile.ID == "custom-export-profile" {
			customProfile = profile
			break
		}
	}
	if customProfile.ID == "" {
		t.Fatalf("expected custom export preset to survive normalization, got %#v", got.SubtitleStyles.SubtitleExportPresets)
	}
	if customProfile.TargetFormat != "itt" {
		t.Fatalf("expected xml profile format to normalize to itt, got %#v", customProfile)
	}
	if customProfile.Config.FCPXML == nil {
		t.Fatalf("expected profile config to survive normalization, got %#v", customProfile)
	}
	if customProfile.MediaStrategy != "fixed" {
		t.Fatalf("expected normalized media strategy fixed, got %#v", customProfile)
	}
	var googleSource library.SubtitleStyleSource
	for _, source := range got.SubtitleStyles.Sources {
		if source.ID == "fontget-google-fonts" {
			googleSource = source
			break
		}
	}
	if googleSource.ID == "" {
		t.Fatalf("expected built-in google font source to remain, got %#v", got.SubtitleStyles.Sources)
	}
	if googleSource.Enabled != true {
		t.Fatalf("expected built-in font source to stay enabled, got %#v", googleSource)
	}
}

func TestToDomainModuleConfigUsesFormatWhenTargetFormatMissing(t *testing.T) {
	t.Parallel()

	config := dto.LibraryModuleConfigDTO{
		SubtitleStyles: dto.LibrarySubtitleStyleConfigDTO{
			SubtitleExportPresets: []dto.LibrarySubtitleExportPresetDTO{{
				ID:     "custom-ass-profile",
				Name:   "Custom ASS",
				Format: "ass",
				Config: dto.SubtitleExportConfig{
					ASS: &dto.SubtitleASSExportConfig{
						PlayResX: 1280,
						PlayResY: 720,
					},
				},
			}},
		},
	}

	got := toDomainModuleConfig(config)
	var customProfile library.SubtitleExportPreset
	for _, profile := range got.SubtitleStyles.SubtitleExportPresets {
		if profile.ID == "custom-ass-profile" {
			customProfile = profile
			break
		}
	}
	if customProfile.ID == "" {
		t.Fatalf("expected custom export preset to survive normalization, got %#v", got.SubtitleStyles.SubtitleExportPresets)
	}
	if customProfile.TargetFormat != "ass" {
		t.Fatalf("expected format field to backfill target format, got %#v", customProfile)
	}
}
