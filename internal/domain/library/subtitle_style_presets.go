package library

import (
	"fmt"
	"math"
	"strings"
)

const (
	SubtitleStyleAspectRatio16By9  = "16:9"
	SubtitleStyleAspectRatio16By10 = "16:10"
	SubtitleStyleAspectRatio4By3   = "4:3"
	SubtitleStyleAspectRatio1By1   = "1:1"
	SubtitleStyleAspectRatio9By16  = "9:16"
)

type AssStyleSpec struct {
	Fontname           string
	FontFace           string
	FontWeight         int
	FontPostScriptName string
	Fontsize           float64
	PrimaryColour      string
	SecondaryColour    string
	OutlineColour      string
	BackColour         string
	Bold               bool
	Italic             bool
	Underline          bool
	StrikeOut          bool
	ScaleX             float64
	ScaleY             float64
	Spacing            float64
	Angle              float64
	BorderStyle        int
	Outline            float64
	Shadow             float64
	Alignment          int
	MarginL            int
	MarginR            int
	MarginV            int
	Encoding           int
}

type MonoStyle struct {
	ID                 string
	Name               string
	BuiltIn            bool
	BasePlayResX       int
	BasePlayResY       int
	BaseAspectRatio    string
	SourceAssStyleName string
	Style              AssStyleSpec
}

type MonoStyleSnapshot struct {
	SourceMonoStyleID   string
	SourceMonoStyleName string
	Name                string
	BasePlayResX        int
	BasePlayResY        int
	BaseAspectRatio     string
	Style               AssStyleSpec
}

type BilingualLayout struct {
	Gap         float64
	BlockAnchor int
}

type BilingualStyle struct {
	ID              string
	Name            string
	BuiltIn         bool
	BasePlayResX    int
	BasePlayResY    int
	BaseAspectRatio string
	Primary         MonoStyleSnapshot
	Secondary       MonoStyleSnapshot
	Layout          BilingualLayout
}

func NormalizeSubtitleStyleAspectRatio(value string) string {
	switch strings.TrimSpace(value) {
	case SubtitleStyleAspectRatio16By10:
		return SubtitleStyleAspectRatio16By10
	case SubtitleStyleAspectRatio4By3:
		return SubtitleStyleAspectRatio4By3
	case SubtitleStyleAspectRatio1By1:
		return SubtitleStyleAspectRatio1By1
	case SubtitleStyleAspectRatio9By16:
		return SubtitleStyleAspectRatio9By16
	case SubtitleStyleAspectRatio16By9:
		return SubtitleStyleAspectRatio16By9
	default:
		return SubtitleStyleAspectRatio16By9
	}
}

func ResolveSubtitleStyleAspectRatio(playResX int, playResY int) string {
	if playResX <= 0 || playResY <= 0 {
		return SubtitleStyleAspectRatio16By9
	}
	ratio := float64(playResX) / float64(playResY)
	best := SubtitleStyleAspectRatio16By9
	bestDistance := math.Abs(ratio - (16.0 / 9.0))
	candidates := []struct {
		ratio string
		value float64
	}{
		{ratio: SubtitleStyleAspectRatio16By10, value: 16.0 / 10.0},
		{ratio: SubtitleStyleAspectRatio4By3, value: 4.0 / 3.0},
		{ratio: SubtitleStyleAspectRatio1By1, value: 1.0},
		{ratio: SubtitleStyleAspectRatio9By16, value: 9.0 / 16.0},
	}
	for _, candidate := range candidates {
		distance := math.Abs(ratio - candidate.value)
		if distance < bestDistance {
			best = candidate.ratio
			bestDistance = distance
		}
	}
	return best
}

func ResolveSubtitleStyleBaseResolution(aspectRatio string) (int, int) {
	switch NormalizeSubtitleStyleAspectRatio(aspectRatio) {
	case SubtitleStyleAspectRatio16By10:
		return 1920, 1200
	case SubtitleStyleAspectRatio4By3:
		return 1440, 1080
	case SubtitleStyleAspectRatio1By1:
		return 1080, 1080
	case SubtitleStyleAspectRatio9By16:
		return 1080, 1920
	default:
		return 1920, 1080
	}
}

func normalizeAssStyleSpec(value AssStyleSpec) AssStyleSpec {
	defaults := AssStyleSpec{
		Fontname:        "Arial",
		FontFace:        "Regular",
		FontWeight:      400,
		Fontsize:        48,
		PrimaryColour:   "&H00FFFFFF",
		SecondaryColour: "&H00FFFFFF",
		OutlineColour:   "&H00111111",
		BackColour:      "&HFF111111",
		ScaleX:          100,
		ScaleY:          100,
		BorderStyle:     1,
		Alignment:       2,
		MarginL:         72,
		MarginR:         72,
		MarginV:         56,
		Encoding:        1,
	}
	result := value
	if strings.TrimSpace(result.Fontname) == "" {
		result.Fontname = defaults.Fontname
	}
	result.FontPostScriptName = strings.TrimSpace(result.FontPostScriptName)
	if result.FontPostScriptName == "" && strings.Contains(result.Fontname, "-") {
		result.FontPostScriptName = strings.TrimSpace(result.Fontname)
	}
	result.FontFace = normalizeSubtitleStyleFontFace(
		firstNonEmpty(
			strings.TrimSpace(result.FontFace),
			deriveSubtitleStyleFontFace(result.FontWeight, result.Bold, result.Italic, result.FontPostScriptName),
		),
	)
	result.FontWeight = normalizeSubtitleStyleFontWeight(result.FontWeight, result.FontFace, result.FontPostScriptName, result.Bold)
	if !result.Bold && result.FontWeight >= 700 {
		result.Bold = true
	}
	if !result.Italic && subtitleStyleFontFaceImpliesItalic(result.FontFace) {
		result.Italic = true
	}
	if !isFinitePositive(result.Fontsize) {
		result.Fontsize = defaults.Fontsize
	}
	result.PrimaryColour = firstNonEmpty(result.PrimaryColour, defaults.PrimaryColour)
	result.SecondaryColour = firstNonEmpty(result.SecondaryColour, defaults.SecondaryColour)
	result.OutlineColour = firstNonEmpty(result.OutlineColour, defaults.OutlineColour)
	result.BackColour = firstNonEmpty(result.BackColour, defaults.BackColour)
	if !isFinitePositive(result.ScaleX) {
		result.ScaleX = defaults.ScaleX
	}
	if !isFinitePositive(result.ScaleY) {
		result.ScaleY = defaults.ScaleY
	}
	if !isFiniteNumber(result.Spacing) {
		result.Spacing = 0
	}
	if !isFiniteNumber(result.Angle) {
		result.Angle = 0
	}
	if result.BorderStyle != 3 {
		result.BorderStyle = defaults.BorderStyle
	}
	if !isFiniteNonNegative(result.Outline) {
		result.Outline = 0
	}
	if !isFiniteNonNegative(result.Shadow) {
		result.Shadow = 0
	}
	if result.Alignment < 1 || result.Alignment > 9 {
		result.Alignment = defaults.Alignment
	}
	if result.MarginL < 0 {
		result.MarginL = defaults.MarginL
	}
	if result.MarginR < 0 {
		result.MarginR = defaults.MarginR
	}
	if result.MarginV < 0 {
		result.MarginV = defaults.MarginV
	}
	if result.Encoding < 0 {
		result.Encoding = defaults.Encoding
	}
	return result
}

func normalizeMonoStyles(values []MonoStyle) []MonoStyle {
	result := make([]MonoStyle, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for index, value := range values {
		id := normalizeAssetID(value.ID, value.Name, firstNonEmpty(value.SourceAssStyleName, fmt.Sprintf("mono-style-%d", index+1)))
		if id == "" {
			id = normalizeAssetID("", "", "mono-style")
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		aspectRatio := NormalizeSubtitleStyleAspectRatio(firstNonEmpty(value.BaseAspectRatio, ResolveSubtitleStyleAspectRatio(value.BasePlayResX, value.BasePlayResY)))
		basePlayResX, basePlayResY := ResolveSubtitleStyleBaseResolution(aspectRatio)
		if value.BasePlayResX > 0 {
			basePlayResX = value.BasePlayResX
		}
		if value.BasePlayResY > 0 {
			basePlayResY = value.BasePlayResY
		}
		if basePlayResX <= 0 || basePlayResY <= 0 {
			basePlayResX, basePlayResY = ResolveSubtitleStyleBaseResolution(aspectRatio)
		}
		result = append(result, MonoStyle{
			ID:                 id,
			Name:               firstNonEmpty(value.Name, value.SourceAssStyleName, "Mono Style"),
			BuiltIn:            value.BuiltIn,
			BasePlayResX:       basePlayResX,
			BasePlayResY:       basePlayResY,
			BaseAspectRatio:    aspectRatio,
			SourceAssStyleName: strings.TrimSpace(value.SourceAssStyleName),
			Style:              normalizeAssStyleSpec(value.Style),
		})
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func normalizeMonoStyleSnapshot(value MonoStyleSnapshot, fallback MonoStyleSnapshot) MonoStyleSnapshot {
	aspectRatio := NormalizeSubtitleStyleAspectRatio(firstNonEmpty(value.BaseAspectRatio, fallback.BaseAspectRatio, ResolveSubtitleStyleAspectRatio(value.BasePlayResX, value.BasePlayResY)))
	basePlayResX, basePlayResY := ResolveSubtitleStyleBaseResolution(aspectRatio)
	if value.BasePlayResX > 0 {
		basePlayResX = value.BasePlayResX
	} else if fallback.BasePlayResX > 0 {
		basePlayResX = fallback.BasePlayResX
	}
	if value.BasePlayResY > 0 {
		basePlayResY = value.BasePlayResY
	} else if fallback.BasePlayResY > 0 {
		basePlayResY = fallback.BasePlayResY
	}
	if basePlayResX <= 0 || basePlayResY <= 0 {
		basePlayResX, basePlayResY = ResolveSubtitleStyleBaseResolution(aspectRatio)
	}
	return MonoStyleSnapshot{
		SourceMonoStyleID:   strings.TrimSpace(firstNonEmpty(value.SourceMonoStyleID, fallback.SourceMonoStyleID)),
		SourceMonoStyleName: strings.TrimSpace(firstNonEmpty(value.SourceMonoStyleName, fallback.SourceMonoStyleName)),
		Name:                firstNonEmpty(value.Name, fallback.Name, value.SourceMonoStyleName, fallback.SourceMonoStyleName, "Mono Snapshot"),
		BasePlayResX:        basePlayResX,
		BasePlayResY:        basePlayResY,
		BaseAspectRatio:     aspectRatio,
		Style:               normalizeAssStyleSpec(mergeAssStyleSpec(fallback.Style, value.Style)),
	}
}

func normalizeBilingualLayout(value BilingualLayout) BilingualLayout {
	result := value
	if !isFiniteNonNegative(result.Gap) {
		result.Gap = 24
	}
	if result.BlockAnchor < 1 || result.BlockAnchor > 9 {
		result.BlockAnchor = 2
	}
	return result
}

func normalizeBilingualStyles(values []BilingualStyle) []BilingualStyle {
	result := make([]BilingualStyle, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		id := normalizeAssetID(value.ID, value.Name, "bilingual-style")
		if id == "" {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		aspectRatio := NormalizeSubtitleStyleAspectRatio(firstNonEmpty(value.BaseAspectRatio, ResolveSubtitleStyleAspectRatio(value.BasePlayResX, value.BasePlayResY)))
		basePlayResX, basePlayResY := ResolveSubtitleStyleBaseResolution(aspectRatio)
		if value.BasePlayResX > 0 {
			basePlayResX = value.BasePlayResX
		}
		if value.BasePlayResY > 0 {
			basePlayResY = value.BasePlayResY
		}
		fallbackSnapshot := MonoStyleSnapshot{
			BasePlayResX:    basePlayResX,
			BasePlayResY:    basePlayResY,
			BaseAspectRatio: aspectRatio,
		}
		result = append(result, BilingualStyle{
			ID:              id,
			Name:            firstNonEmpty(value.Name, "Bilingual Style"),
			BuiltIn:         value.BuiltIn,
			BasePlayResX:    basePlayResX,
			BasePlayResY:    basePlayResY,
			BaseAspectRatio: aspectRatio,
			Primary:         normalizeMonoStyleSnapshot(value.Primary, fallbackSnapshot),
			Secondary:       normalizeMonoStyleSnapshot(value.Secondary, fallbackSnapshot),
			Layout:          normalizeBilingualLayout(value.Layout),
		})
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func mergeAssStyleSpec(base AssStyleSpec, override AssStyleSpec) AssStyleSpec {
	result := base
	if strings.TrimSpace(override.Fontname) != "" {
		result.Fontname = override.Fontname
	}
	fontIdentityChanged := strings.TrimSpace(override.Fontname) != "" ||
		strings.TrimSpace(override.FontFace) != "" ||
		strings.TrimSpace(override.FontPostScriptName) != "" ||
		override.FontWeight > 0
	if fontIdentityChanged {
		result.FontFace = strings.TrimSpace(override.FontFace)
		result.FontPostScriptName = strings.TrimSpace(override.FontPostScriptName)
		result.FontWeight = override.FontWeight
	}
	if isFinitePositive(override.Fontsize) {
		result.Fontsize = override.Fontsize
	}
	if strings.TrimSpace(override.PrimaryColour) != "" {
		result.PrimaryColour = override.PrimaryColour
	}
	if strings.TrimSpace(override.SecondaryColour) != "" {
		result.SecondaryColour = override.SecondaryColour
	}
	if strings.TrimSpace(override.OutlineColour) != "" {
		result.OutlineColour = override.OutlineColour
	}
	if strings.TrimSpace(override.BackColour) != "" {
		result.BackColour = override.BackColour
	}
	result.Bold = override.Bold
	result.Italic = override.Italic
	result.Underline = override.Underline
	result.StrikeOut = override.StrikeOut
	if isFinitePositive(override.ScaleX) {
		result.ScaleX = override.ScaleX
	}
	if isFinitePositive(override.ScaleY) {
		result.ScaleY = override.ScaleY
	}
	if isFiniteNumber(override.Spacing) {
		result.Spacing = override.Spacing
	}
	if isFiniteNumber(override.Angle) {
		result.Angle = override.Angle
	}
	if override.BorderStyle != 0 {
		result.BorderStyle = override.BorderStyle
	}
	if isFiniteNonNegative(override.Outline) {
		result.Outline = override.Outline
	}
	if isFiniteNonNegative(override.Shadow) {
		result.Shadow = override.Shadow
	}
	if override.Alignment >= 1 && override.Alignment <= 9 {
		result.Alignment = override.Alignment
	}
	if override.MarginL >= 0 {
		result.MarginL = override.MarginL
	}
	if override.MarginR >= 0 {
		result.MarginR = override.MarginR
	}
	if override.MarginV >= 0 {
		result.MarginV = override.MarginV
	}
	if override.Encoding >= 0 {
		result.Encoding = override.Encoding
	}
	return result
}

func isFiniteNumber(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func isFinitePositive(value float64) bool {
	return isFiniteNumber(value) && value > 0
}

func isFiniteNonNegative(value float64) bool {
	return isFiniteNumber(value) && value >= 0
}

func normalizeSubtitleStyleFontFace(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "Regular"
	}
	return trimmed
}

func normalizeSubtitleStyleFontWeight(value int, face string, postScriptName string, bold bool) int {
	if value > 0 {
		return value
	}
	if derived := deriveSubtitleStyleFontWeightFromFace(face); derived > 0 {
		return derived
	}
	if derived := deriveSubtitleStyleFontWeightFromFace(deriveSubtitleStyleFontFaceFromPostScriptName(postScriptName)); derived > 0 {
		return derived
	}
	if bold {
		return 700
	}
	return 400
}

func deriveSubtitleStyleFontFace(weight int, bold bool, italic bool, postScriptName string) string {
	if derived := deriveSubtitleStyleFontFaceFromPostScriptName(postScriptName); derived != "" {
		return derived
	}
	switch {
	case italic && weight >= 700:
		return "Bold Italic"
	case italic:
		return "Italic"
	case weight >= 700 || bold:
		return "Bold"
	default:
		return "Regular"
	}
}

func subtitleStyleFontFaceImpliesItalic(face string) bool {
	normalized := strings.ToLower(strings.TrimSpace(face))
	return strings.Contains(normalized, "italic") || strings.Contains(normalized, "oblique")
}

func deriveSubtitleStyleFontWeightFromFace(face string) int {
	normalized := strings.ToLower(strings.TrimSpace(face))
	switch {
	case normalized == "":
		return 0
	case strings.Contains(normalized, "thin"), strings.Contains(normalized, "hairline"):
		return 100
	case strings.Contains(normalized, "ultralight"), strings.Contains(normalized, "extra light"), strings.Contains(normalized, "extralight"):
		return 200
	case strings.Contains(normalized, "light"):
		return 300
	case strings.Contains(normalized, "semibold"), strings.Contains(normalized, "demibold"), strings.Contains(normalized, "demi bold"), strings.Contains(normalized, "demi"):
		return 600
	case strings.Contains(normalized, "extrabold"), strings.Contains(normalized, "extra bold"), strings.Contains(normalized, "ultrabold"), strings.Contains(normalized, "ultra bold"):
		return 800
	case strings.Contains(normalized, "black"), strings.Contains(normalized, "heavy"):
		return 900
	case strings.Contains(normalized, "medium"):
		return 500
	case strings.Contains(normalized, "bold"):
		return 700
	case strings.Contains(normalized, "regular"), strings.Contains(normalized, "normal"), strings.Contains(normalized, "roman"), strings.Contains(normalized, "book"):
		return 400
	default:
		return 400
	}
}

func deriveSubtitleStyleFontFaceFromPostScriptName(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	if separatorIndex := strings.LastIndex(trimmed, "-"); separatorIndex >= 0 && separatorIndex < len(trimmed)-1 {
		return strings.TrimSpace(trimmed[separatorIndex+1:])
	}
	return ""
}
