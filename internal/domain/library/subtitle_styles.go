package library

import (
	"fmt"
	"path"
	"sort"
	"strconv"
	"strings"
)

type SubtitleStyleConfig struct {
	MonoStyles            []MonoStyle
	BilingualStyles       []BilingualStyle
	Sources               []SubtitleStyleSource
	Fonts                 []SubtitleStyleFont
	SubtitleExportPresets []SubtitleExportPreset
	Defaults              SubtitleStyleDefaults
}

type SubtitleStyleDefaults struct {
	MonoStyleID            string
	BilingualStyleID       string
	SubtitleExportPresetID string
}

type SubtitleStyleSource struct {
	ID           string
	Name         string
	Kind         string
	Provider     string
	URL          string
	Prefix       string
	Filename     string
	Priority     int
	BuiltIn      bool
	Owner        string
	Repo         string
	Ref          string
	ManifestPath string
	Manifest     SubtitleStyleSourceManifest
	Enabled      bool
	FontCount    int
	SyncStatus   string
	LastSyncedAt string
	LastError    string
}

type SubtitleStyleSourceManifest struct {
	SourceInfo SubtitleStyleSourceManifestInfo
	Fonts      map[string]SubtitleStyleSourceManifestFont
}

type SubtitleStyleSourceManifestInfo struct {
	Name        string
	Description string
	URL         string
	APIEndpoint string
	Version     string
	LastUpdated string
	TotalFonts  int
}

type SubtitleStyleSourceManifestFont struct {
	Name          string
	Family        string
	License       string
	LicenseURL    string
	Designer      string
	Foundry       string
	Version       string
	Description   string
	Categories    []string
	Tags          []string
	Popularity    int
	LastModified  string
	MetadataURL   string
	SourceURL     string
	Variants      []SubtitleStyleSourceManifestVariant
	UnicodeRanges []string
	Languages     []string
	SampleText    string
}

type SubtitleStyleSourceManifestVariant struct {
	Name    string
	Weight  int
	Style   string
	Subsets []string
	Files   map[string]string
}

type SubtitleStyleFont struct {
	ID           string
	Family       string
	Source       string
	SystemFamily string
	Enabled      bool
}

type SubtitleExportPreset struct {
	ID            string
	Name          string
	Description   string
	TargetFormat  string
	MediaStrategy string
	Config        SubtitleExportConfig
}

type SubtitleExportConfig struct {
	SRT    *SubtitleSRTExportConfig
	VTT    *SubtitleVTTExportConfig
	ASS    *SubtitleASSExportConfig
	ITT    *SubtitleITTExportConfig
	FCPXML *SubtitleFCPXMLExportConfig
}

type SubtitleSRTExportConfig struct {
	Encoding string
}

type SubtitleVTTExportConfig struct {
	Kind     string
	Language string
}

type SubtitleASSExportConfig struct {
	PlayResX int
	PlayResY int
	Title    string
}

type SubtitleITTExportConfig struct {
	FrameRate           int
	FrameRateMultiplier string
	Language            string
}

type SubtitleFCPXMLExportConfig struct {
	FrameDuration        string
	Width                int
	Height               int
	ColorSpace           string
	Version              string
	LibraryName          string
	EventName            string
	ProjectName          string
	DefaultLane          int
	StartTimecodeSeconds int64
}

const (
	defaultSubtitleExportPresetID = "builtin-subtitle-export-preset-srt-auto"

	subtitleStyleSourceProviderGitHub  = "github"
	subtitleStyleSourceProviderFontGet = "fontget"

	subtitleExportPresetMediaStrategyAuto  = "auto"
	subtitleExportPresetMediaStrategyFixed = "fixed"

	defaultSubtitleFontSourcePriorityStart = 100
)

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
			Fontname:        "Arial",
			Fontsize:        56,
			PrimaryColour:   "&H00FFFFFF",
			SecondaryColour: "&H00FFFFFF",
			OutlineColour:   "&H00101010",
			BackColour:      "&H80000000",
			Bold:            false,
			Italic:          false,
			Underline:       false,
			StrikeOut:       false,
			ScaleX:          100,
			ScaleY:          100,
			Spacing:         0,
			Angle:           0,
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
			Bold:            false,
			Italic:          false,
			Underline:       false,
			StrikeOut:       false,
			ScaleX:          100,
			ScaleY:          100,
			Spacing:         0,
			Angle:           0,
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
				Fontname:        "Arial",
				Fontsize:        56,
				PrimaryColour:   "&H00FFFFFF",
				SecondaryColour: "&H00FFFFFF",
				OutlineColour:   "&H00101010",
				BackColour:      "&H80000000",
				Bold:            false,
				Italic:          false,
				Underline:       false,
				StrikeOut:       false,
				ScaleX:          100,
				ScaleY:          100,
				Spacing:         0,
				Angle:           0,
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
		Secondary: MonoStyleSnapshot{
			SourceMonoStyleID:   "builtin-subtitle-mono-secondary-1080p",
			SourceMonoStyleName: "1080p Secondary",
			Name:                "Secondary",
			BasePlayResX:        1920,
			BasePlayResY:        1080,
			BaseAspectRatio:     SubtitleStyleAspectRatio16By9,
			Style: AssStyleSpec{
				Fontname:        "Arial",
				Fontsize:        40,
				PrimaryColour:   "&H00E8E8E8",
				SecondaryColour: "&H00E8E8E8",
				OutlineColour:   "&H00101010",
				BackColour:      "&H80000000",
				Bold:            false,
				Italic:          false,
				Underline:       false,
				StrikeOut:       false,
				ScaleX:          100,
				ScaleY:          100,
				Spacing:         0,
				Angle:           0,
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
		Layout: BilingualLayout{
			Gap:         20,
			BlockAnchor: 2,
		},
	},
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
		ID:            "builtin-subtitle-export-preset-ass-4k60",
		Name:          "ASS · 4K · 60fps",
		Description:   "ASS output forced to 3840x2160 for 60fps delivery.",
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
		ID:            "builtin-subtitle-export-preset-ass-4k30",
		Name:          "ASS · 4K · 30fps",
		Description:   "ASS output forced to 3840x2160 for 30fps delivery.",
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
		ID:            "builtin-subtitle-export-preset-ass-1080p60",
		Name:          "ASS · 1080p · 60fps",
		Description:   "ASS output forced to 1920x1080 for 60fps delivery.",
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
		ID:            "builtin-subtitle-export-preset-ass-1080p30",
		Name:          "ASS · 1080p · 30fps",
		Description:   "ASS output forced to 1920x1080 for 30fps delivery.",
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
				StartTimecodeSeconds: 3600,
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
				StartTimecodeSeconds: 3600,
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
				StartTimecodeSeconds: 3600,
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
				StartTimecodeSeconds: 3600,
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
				StartTimecodeSeconds: 3600,
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

func normalizeSubtitleStyleConfig(config SubtitleStyleConfig) SubtitleStyleConfig {
	defaults := defaultSubtitleStyleConfig()
	monoStyles := normalizeMonoStyles(config.MonoStyles)
	if len(monoStyles) == 0 {
		monoStyles = defaults.MonoStyles
	}
	bilingualStyles := normalizeBilingualStyles(config.BilingualStyles)
	if len(bilingualStyles) == 0 {
		bilingualStyles = defaults.BilingualStyles
	}
	monoStyleID := normalizeSubtitleStyleDefaultMonoStyleID(
		config.Defaults.MonoStyleID,
		defaults.Defaults.MonoStyleID,
		monoStyles,
	)
	bilingualStyleID := normalizeSubtitleStyleDefaultBilingualStyleID(
		config.Defaults.BilingualStyleID,
		defaults.Defaults.BilingualStyleID,
		bilingualStyles,
	)
	subtitleExportPresets := normalizeSubtitleExportPresets(config.SubtitleExportPresets, defaults.SubtitleExportPresets)
	subtitleExportPresetID := normalizeSubtitleStyleDefaultSubtitleExportPresetID(
		config.Defaults.SubtitleExportPresetID,
		defaults.Defaults.SubtitleExportPresetID,
		subtitleExportPresets,
	)
	return SubtitleStyleConfig{
		MonoStyles:            monoStyles,
		BilingualStyles:       bilingualStyles,
		Sources:               normalizeSubtitleStyleSources(config.Sources),
		Fonts:                 normalizeSubtitleStyleFonts(config.Fonts),
		SubtitleExportPresets: subtitleExportPresets,
		Defaults: SubtitleStyleDefaults{
			MonoStyleID:            monoStyleID,
			BilingualStyleID:       bilingualStyleID,
			SubtitleExportPresetID: subtitleExportPresetID,
		},
	}
}

func normalizeSubtitleStyleDefaultMonoStyleID(
	selectedID string,
	fallbackID string,
	styles []MonoStyle,
) string {
	idSet := make(map[string]struct{}, len(styles))
	for _, style := range styles {
		idSet[style.ID] = struct{}{}
	}
	if id := normalizeSingleSelectedID(selectedID, idSet); id != "" {
		return id
	}
	if id := normalizeSingleSelectedID(fallbackID, idSet); id != "" {
		return id
	}
	for _, style := range styles {
		return style.ID
	}
	return ""
}

func normalizeSubtitleStyleDefaultBilingualStyleID(
	selectedID string,
	fallbackID string,
	styles []BilingualStyle,
) string {
	idSet := make(map[string]struct{}, len(styles))
	for _, style := range styles {
		idSet[style.ID] = struct{}{}
	}
	if id := normalizeSingleSelectedID(selectedID, idSet); id != "" {
		return id
	}
	if id := normalizeSingleSelectedID(fallbackID, idSet); id != "" {
		return id
	}
	for _, style := range styles {
		return style.ID
	}
	return ""
}

func normalizeSubtitleStyleSources(values []SubtitleStyleSource) []SubtitleStyleSource {
	result := make([]SubtitleStyleSource, 0, len(values)+len(defaultBuiltInSubtitleStyleFontSources))
	seen := make(map[string]struct{}, len(values)+len(defaultBuiltInSubtitleStyleFontSources))
	normalizedByID := make(map[string]SubtitleStyleSource, len(values)+len(defaultBuiltInSubtitleStyleFontSources))

	for index, value := range values {
		id := normalizeAssetID(value.ID, value.Name, fmt.Sprintf("subtitle-style-source-%d", index+1))
		if id == "" {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		name := strings.TrimSpace(value.Name)
		if name == "" {
			name = fmt.Sprintf("Remote source %d", index+1)
		}
		provider := normalizeSubtitleStyleSourceProvider(value.Provider)
		kind := normalizeSubtitleStyleSourceKind(value.Kind)
		if kind == "" {
			kind = "style"
		}
		if kind == "font" {
			if _, exists := builtInSubtitleStyleFontSourceIDs[id]; !exists {
				continue
			}
			provider = subtitleStyleSourceProviderFontGet
		} else if provider == "" {
			provider = subtitleStyleSourceProviderGitHub
		}
		manifestPathFallback := "manifest.json"
		if kind == "font" {
			manifestPathFallback = "fonts.json"
		}
		syncStatus := normalizeSubtitleStyleSourceSyncStatus(value.SyncStatus)
		if syncStatus == "" {
			syncStatus = "idle"
		}
		filename := strings.TrimSpace(value.Filename)
		if filename == "" {
			switch provider {
			case subtitleStyleSourceProviderFontGet:
				filename = deriveSubtitleStyleSourceFilename(value.URL, name, "fonts.json")
			default:
				filename = path.Base(strings.TrimSpace(firstNonEmpty(value.ManifestPath, manifestPathFallback)))
			}
		}
		priority := value.Priority
		if priority <= 0 {
			priority = defaultSubtitleFontSourcePriorityStart + index + 1
		}
		fontCount := value.FontCount
		if fontCount < 0 {
			fontCount = 0
		}
		manifest := SubtitleStyleSourceManifest{}
		if kind == "font" {
			manifest = normalizeSubtitleStyleSourceManifest(value.Manifest)
			if fontCount <= 0 && manifest.SourceInfo.TotalFonts > 0 {
				fontCount = manifest.SourceInfo.TotalFonts
			}
		}
		resolvedURL := strings.TrimSpace(value.URL)
		owner := strings.TrimSpace(value.Owner)
		repo := strings.TrimSpace(value.Repo)
		ref := strings.TrimSpace(firstNonEmpty(value.Ref, "main"))
		manifestPath := strings.TrimSpace(firstNonEmpty(value.ManifestPath, manifestPathFallback))
		if kind == "font" {
			owner = ""
			repo = ""
			ref = ""
			manifestPath = ""
		}
		normalized := SubtitleStyleSource{
			ID:           id,
			Name:         name,
			Kind:         kind,
			Provider:     provider,
			URL:          resolvedURL,
			Prefix:       normalizeSubtitleStyleSourcePrefix(value.Prefix, name),
			Filename:     filename,
			Priority:     priority,
			BuiltIn:      value.BuiltIn,
			Owner:        owner,
			Repo:         repo,
			Ref:          ref,
			ManifestPath: manifestPath,
			Manifest:     manifest,
			Enabled:      value.Enabled,
			FontCount:    fontCount,
			SyncStatus:   syncStatus,
			LastSyncedAt: strings.TrimSpace(value.LastSyncedAt),
			LastError:    strings.TrimSpace(value.LastError),
		}
		normalizedByID[id] = normalized
		result = append(result, normalized)
	}

	for _, builtIn := range defaultBuiltInSubtitleStyleFontSources {
		existing, exists := normalizedByID[builtIn.ID]
		if exists {
			existing.Name = builtIn.Name
			existing.Kind = builtIn.Kind
			existing.Provider = builtIn.Provider
			existing.URL = builtIn.URL
			existing.Prefix = builtIn.Prefix
			existing.Filename = builtIn.Filename
			existing.Priority = builtIn.Priority
			existing.BuiltIn = true
			existing.Owner = ""
			existing.Repo = ""
			existing.Ref = ""
			existing.ManifestPath = ""
			existing.Manifest = normalizeSubtitleStyleSourceManifest(existing.Manifest)
			existing.Enabled = true
			if existing.FontCount <= 0 && existing.Manifest.SourceInfo.TotalFonts > 0 {
				existing.FontCount = existing.Manifest.SourceInfo.TotalFonts
			}
			if existing.FontCount < 0 {
				existing.FontCount = 0
			}
			normalizedByID[builtIn.ID] = existing
			continue
		}
		normalizedByID[builtIn.ID] = builtIn
		result = append(result, builtIn)
	}

	sort.SliceStable(result, func(left, right int) bool {
		leftSource := normalizedByID[result[left].ID]
		rightSource := normalizedByID[result[right].ID]
		if leftSource.BuiltIn != rightSource.BuiltIn {
			return leftSource.BuiltIn
		}
		if leftSource.Priority != rightSource.Priority {
			return leftSource.Priority < rightSource.Priority
		}
		return strings.ToLower(leftSource.Name) < strings.ToLower(rightSource.Name)
	})

	for index := range result {
		result[index] = normalizedByID[result[index].ID]
	}
	return result
}

func normalizeSubtitleStyleFonts(values []SubtitleStyleFont) []SubtitleStyleFont {
	result := make([]SubtitleStyleFont, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for index, value := range values {
		id := normalizeAssetID(value.ID, value.Family, fmt.Sprintf("subtitle-style-font-%d", index+1))
		if id == "" {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}

		family := strings.TrimSpace(firstNonEmpty(value.Family, value.SystemFamily))
		systemFamily := strings.TrimSpace(firstNonEmpty(value.SystemFamily, value.Family))
		if family == "" || systemFamily == "" {
			continue
		}

		source := normalizeSubtitleStyleFontSource(value.Source)
		if source == "" {
			source = "system"
		}

		result = append(result, SubtitleStyleFont{
			ID:           id,
			Family:       family,
			Source:       source,
			SystemFamily: systemFamily,
			Enabled:      value.Enabled,
		})
	}
	return result
}

func normalizeSubtitleExportPresets(
	values []SubtitleExportPreset,
	fallback []SubtitleExportPreset,
) []SubtitleExportPreset {
	result := make([]SubtitleExportPreset, 0, len(values)+len(fallback))
	seen := make(map[string]struct{}, len(values)+len(fallback))
	normalizedByID := make(map[string]SubtitleExportPreset, len(values)+len(fallback))

	source := values
	if len(source) == 0 {
		source = fallback
	}

	for index, value := range source {
		id := normalizeAssetID(value.ID, value.Name, fmt.Sprintf("subtitle-export-preset-%d", index+1))
		if id == "" {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}

		name := strings.TrimSpace(value.Name)
		if name == "" {
			name = fmt.Sprintf("Subtitle Export Preset %d", index+1)
		}
		targetFormat := normalizeSubtitleExportTargetFormat(value.TargetFormat)
		if targetFormat == "" {
			targetFormat = "ass"
		}
		normalized := SubtitleExportPreset{
			ID:            id,
			Name:          name,
			Description:   strings.TrimSpace(value.Description),
			TargetFormat:  targetFormat,
			MediaStrategy: normalizeSubtitleExportPresetMediaStrategy(value.MediaStrategy),
			Config:        normalizeSubtitleExportConfig(value.Config),
		}
		normalizedByID[id] = normalized
		result = append(result, normalized)
	}

	for _, builtIn := range fallback {
		existing, exists := normalizedByID[builtIn.ID]
		if exists {
			existing.Name = builtIn.Name
			existing.Description = builtIn.Description
			existing.TargetFormat = builtIn.TargetFormat
			existing.MediaStrategy = builtIn.MediaStrategy
			if existing.Config == (SubtitleExportConfig{}) {
				existing.Config = builtIn.Config
			}
			normalizedByID[builtIn.ID] = existing
			continue
		}
		normalizedByID[builtIn.ID] = builtIn
		result = append(result, builtIn)
	}

	for index := range result {
		result[index] = normalizedByID[result[index].ID]
	}
	return result
}

func normalizeSubtitleStyleDefaultSubtitleExportPresetID(
	selectedID string,
	fallbackID string,
	profiles []SubtitleExportPreset,
) string {
	idSet := make(map[string]struct{}, len(profiles))
	for _, profile := range profiles {
		idSet[profile.ID] = struct{}{}
	}
	if id := normalizeSingleSelectedID(selectedID, idSet); id != "" {
		return id
	}
	if id := normalizeSingleSelectedID(fallbackID, idSet); id != "" {
		return id
	}
	for _, profile := range profiles {
		return profile.ID
	}
	return ""
}

func normalizeSubtitleExportTargetFormat(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "srt":
		return "srt"
	case "vtt":
		return "vtt"
	case "ass", "ssa":
		return "ass"
	case "itt", "ttml", "xml":
		return "itt"
	case "fcpxml":
		return "fcpxml"
	default:
		return ""
	}
}

func normalizeSubtitleExportPresetMediaStrategy(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case subtitleExportPresetMediaStrategyFixed:
		return subtitleExportPresetMediaStrategyFixed
	default:
		return subtitleExportPresetMediaStrategyAuto
	}
}

func normalizeSubtitleExportConfig(value SubtitleExportConfig) SubtitleExportConfig {
	var result SubtitleExportConfig

	if value.SRT != nil {
		encoding := strings.TrimSpace(value.SRT.Encoding)
		if encoding != "" {
			result.SRT = &SubtitleSRTExportConfig{Encoding: encoding}
		}
	}

	if value.VTT != nil {
		kind := strings.TrimSpace(value.VTT.Kind)
		language := strings.TrimSpace(value.VTT.Language)
		if kind != "" || language != "" {
			result.VTT = &SubtitleVTTExportConfig{
				Kind:     kind,
				Language: language,
			}
		}
	}

	if value.ASS != nil {
		playResX := value.ASS.PlayResX
		playResY := value.ASS.PlayResY
		if playResX < 0 {
			playResX = 0
		}
		if playResY < 0 {
			playResY = 0
		}
		title := strings.TrimSpace(value.ASS.Title)
		if playResX > 0 || playResY > 0 || title != "" {
			result.ASS = &SubtitleASSExportConfig{
				PlayResX: playResX,
				PlayResY: playResY,
				Title:    title,
			}
		}
	}

	if value.ITT != nil {
		frameRate := value.ITT.FrameRate
		if frameRate < 0 {
			frameRate = 0
		}
		frameRateMultiplier := normalizeITTFrameRateMultiplier(value.ITT.FrameRateMultiplier)
		if frameRate <= 0 {
			frameRateMultiplier = ""
		}
		language := strings.TrimSpace(value.ITT.Language)
		if frameRate > 0 || frameRateMultiplier != "" || language != "" {
			result.ITT = &SubtitleITTExportConfig{
				FrameRate:           frameRate,
				FrameRateMultiplier: frameRateMultiplier,
				Language:            language,
			}
		}
	}

	if value.FCPXML != nil {
		frameDuration := strings.TrimSpace(value.FCPXML.FrameDuration)
		width := value.FCPXML.Width
		if width < 0 {
			width = 0
		}
		height := value.FCPXML.Height
		if height < 0 {
			height = 0
		}
		defaultLane := value.FCPXML.DefaultLane
		startTimecodeSeconds := value.FCPXML.StartTimecodeSeconds
		if startTimecodeSeconds < 0 {
			startTimecodeSeconds = 0
		}
		colorSpace := strings.TrimSpace(value.FCPXML.ColorSpace)
		version := strings.TrimSpace(value.FCPXML.Version)
		libraryName := strings.TrimSpace(value.FCPXML.LibraryName)
		eventName := strings.TrimSpace(value.FCPXML.EventName)
		projectName := strings.TrimSpace(value.FCPXML.ProjectName)
		if frameDuration != "" || width > 0 || height > 0 || colorSpace != "" || version != "" || libraryName != "" || eventName != "" || projectName != "" || defaultLane != 0 || startTimecodeSeconds > 0 {
			result.FCPXML = &SubtitleFCPXMLExportConfig{
				FrameDuration:        frameDuration,
				Width:                width,
				Height:               height,
				ColorSpace:           colorSpace,
				Version:              version,
				LibraryName:          libraryName,
				EventName:            eventName,
				ProjectName:          projectName,
				DefaultLane:          defaultLane,
				StartTimecodeSeconds: startTimecodeSeconds,
			}
		}
	}

	return result
}

func normalizeITTFrameRateMultiplier(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	normalized := strings.ReplaceAll(trimmed, "/", " ")
	parts := strings.Fields(normalized)
	if len(parts) != 2 {
		return ""
	}
	numerator, errNum := strconv.ParseInt(parts[0], 10, 64)
	denominator, errDen := strconv.ParseInt(parts[1], 10, 64)
	if errNum != nil || errDen != nil || numerator <= 0 || denominator <= 0 {
		return ""
	}
	return fmt.Sprintf("%d %d", numerator, denominator)
}

func normalizeSubtitleStyleDocumentSource(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "builtin":
		return "builtin"
	case "library", "local":
		return "library"
	case "remote":
		return "remote"
	default:
		return ""
	}
}

func normalizeSubtitleStyleFontSource(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "system":
		return "system"
	default:
		return ""
	}
}

func normalizeSubtitleStyleSourceProvider(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case subtitleStyleSourceProviderFontGet:
		return subtitleStyleSourceProviderFontGet
	case subtitleStyleSourceProviderGitHub:
		return subtitleStyleSourceProviderGitHub
	default:
		return ""
	}
}

func normalizeSubtitleStyleSourceKind(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "font":
		return "font"
	case "style":
		return "style"
	default:
		return ""
	}
}

func normalizeSubtitleStyleSourceSyncStatus(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "idle":
		return "idle"
	case "synced":
		return "synced"
	case "error":
		return "error"
	default:
		return ""
	}
}

func normalizeSubtitleStyleSourceManifest(value SubtitleStyleSourceManifest) SubtitleStyleSourceManifest {
	info := SubtitleStyleSourceManifestInfo{
		Name:        strings.TrimSpace(value.SourceInfo.Name),
		Description: strings.TrimSpace(value.SourceInfo.Description),
		URL:         strings.TrimSpace(value.SourceInfo.URL),
		APIEndpoint: strings.TrimSpace(value.SourceInfo.APIEndpoint),
		Version:     strings.TrimSpace(value.SourceInfo.Version),
		LastUpdated: strings.TrimSpace(value.SourceInfo.LastUpdated),
		TotalFonts:  value.SourceInfo.TotalFonts,
	}
	if info.TotalFonts < 0 {
		info.TotalFonts = 0
	}

	fonts := make(map[string]SubtitleStyleSourceManifestFont, len(value.Fonts))
	for rawID, font := range value.Fonts {
		id := strings.TrimSpace(rawID)
		if id == "" {
			continue
		}
		fonts[id] = normalizeSubtitleStyleSourceManifestFont(font)
	}
	if info.TotalFonts <= 0 && len(fonts) > 0 {
		info.TotalFonts = len(fonts)
	}
	if info == (SubtitleStyleSourceManifestInfo{}) && len(fonts) == 0 {
		return SubtitleStyleSourceManifest{}
	}
	return SubtitleStyleSourceManifest{
		SourceInfo: info,
		Fonts:      fonts,
	}
}

func normalizeSubtitleStyleSourceManifestFont(value SubtitleStyleSourceManifestFont) SubtitleStyleSourceManifestFont {
	variants := make([]SubtitleStyleSourceManifestVariant, 0, len(value.Variants))
	for _, variant := range value.Variants {
		files := make(map[string]string, len(variant.Files))
		for fileType, rawURL := range variant.Files {
			normalizedType := strings.TrimSpace(fileType)
			normalizedURL := strings.TrimSpace(rawURL)
			if normalizedType == "" || normalizedURL == "" {
				continue
			}
			files[normalizedType] = normalizedURL
		}
		variants = append(variants, SubtitleStyleSourceManifestVariant{
			Name:    strings.TrimSpace(variant.Name),
			Weight:  variant.Weight,
			Style:   strings.TrimSpace(variant.Style),
			Subsets: normalizeManifestStringList(variant.Subsets),
			Files:   files,
		})
	}
	return SubtitleStyleSourceManifestFont{
		Name:          strings.TrimSpace(value.Name),
		Family:        strings.TrimSpace(value.Family),
		License:       strings.TrimSpace(value.License),
		LicenseURL:    strings.TrimSpace(value.LicenseURL),
		Designer:      strings.TrimSpace(value.Designer),
		Foundry:       strings.TrimSpace(value.Foundry),
		Version:       strings.TrimSpace(value.Version),
		Description:   strings.TrimSpace(value.Description),
		Categories:    normalizeManifestStringList(value.Categories),
		Tags:          normalizeManifestStringList(value.Tags),
		Popularity:    value.Popularity,
		LastModified:  strings.TrimSpace(value.LastModified),
		MetadataURL:   strings.TrimSpace(value.MetadataURL),
		SourceURL:     strings.TrimSpace(value.SourceURL),
		Variants:      variants,
		UnicodeRanges: normalizeManifestStringList(value.UnicodeRanges),
		Languages:     normalizeManifestStringList(value.Languages),
		SampleText:    strings.TrimSpace(value.SampleText),
	}
}

func normalizeManifestStringList(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		result = append(result, trimmed)
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func normalizeSubtitleStyleSourcePrefix(value string, fallbackName string) string {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	if trimmed == "" {
		trimmed = strings.ToLower(strings.TrimSpace(fallbackName))
	}
	trimmed = strings.ReplaceAll(trimmed, "_", "-")
	trimmed = strings.Join(strings.Fields(trimmed), "-")
	var builder strings.Builder
	lastHyphen := false
	for _, r := range trimmed {
		isLower := r >= 'a' && r <= 'z'
		isDigit := r >= '0' && r <= '9'
		if isLower || isDigit {
			builder.WriteRune(r)
			lastHyphen = false
			continue
		}
		if r == '-' && !lastHyphen && builder.Len() > 0 {
			builder.WriteRune(r)
			lastHyphen = true
		}
	}
	result := strings.Trim(builder.String(), "-")
	return result
}

func deriveSubtitleStyleSourceFilename(rawURL string, name string, fallback string) string {
	trimmedURL := strings.TrimSpace(rawURL)
	if trimmedURL != "" {
		base := path.Base(trimmedURL)
		if base != "" && base != "." && base != "/" {
			return base
		}
	}
	prefix := normalizeSubtitleStyleSourcePrefix("", name)
	if prefix == "" {
		return fallback
	}
	return prefix + ".json"
}
