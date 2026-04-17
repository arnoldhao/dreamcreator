package library

import (
	"fmt"
	"strings"
	"time"
	"unicode"
)

type CreateMeta struct {
	Source             string
	TriggerOperationID string
	ImportBatchID      string
	Actor              string
}

type RetentionConfig struct {
	WorkspaceStatesMax int
	FileEventsMax      int
	HistoryMax         int
	OperationLogsMax   int
}

type WorkspaceConfig struct {
	FastReadLatestState bool
}

type SubtitleStorageConfig struct {
	DownloadPolicy string
	ImportPolicy   string
}

type TranslateLanguage struct {
	Code    string
	Label   string
	Aliases []string
}

type TranslateLanguagesConfig struct {
	Builtin []TranslateLanguage
	Custom  []TranslateLanguage
}

type GlossaryTerm struct {
	Source string
	Target string
	Note   string
}

type GlossaryProfile struct {
	ID             string
	Name           string
	Category       string
	Description    string
	SourceLanguage string
	TargetLanguage string
	Terms          []GlossaryTerm
}

type PromptProfile struct {
	ID          string
	Name        string
	Category    string
	Description string
	Prompt      string
}

type LanguageAssetsConfig struct {
	GlossaryProfiles []GlossaryProfile
	PromptProfiles   []PromptProfile
}

type LanguageTaskRuntimeSettings struct {
	StructuredOutputMode string
	ThinkingMode         string
	MaxTokensFloor       int
	MaxTokensCeiling     int
	RetryTokenStep       int
}

type LanguageTaskRuntimeConfig struct {
	Translate LanguageTaskRuntimeSettings
	Proofread LanguageTaskRuntimeSettings
}

type ModuleConfig struct {
	Retention          RetentionConfig
	Workspace          WorkspaceConfig
	SubtitleStorage    SubtitleStorageConfig
	TranslateLanguages TranslateLanguagesConfig
	LanguageAssets     LanguageAssetsConfig
	SubtitleStyles     SubtitleStyleConfig
	TaskRuntime        LanguageTaskRuntimeConfig
}

type Library struct {
	ID        string
	Name      string
	CreatedBy CreateMeta
	CreatedAt time.Time
	UpdatedAt time.Time
}

type LibraryParams struct {
	ID        string
	Name      string
	CreatedBy CreateMeta
	CreatedAt *time.Time
	UpdatedAt *time.Time
}

func DefaultModuleConfig() ModuleConfig {
	return ModuleConfig{
		Retention: RetentionConfig{
			WorkspaceStatesMax: 20,
			FileEventsMax:      200,
			HistoryMax:         200,
			OperationLogsMax:   50,
		},
		Workspace: WorkspaceConfig{FastReadLatestState: true},
		SubtitleStorage: SubtitleStorageConfig{
			DownloadPolicy: "db_document_delete_source",
			ImportPolicy:   "db_document_keep_source",
		},
		TranslateLanguages: TranslateLanguagesConfig{
			Builtin: defaultBuiltinTranslateLanguages(),
			Custom:  nil,
		},
		LanguageAssets: LanguageAssetsConfig{
			GlossaryProfiles: nil,
			PromptProfiles:   nil,
		},
		SubtitleStyles: defaultSubtitleStyleConfig(),
		TaskRuntime:    defaultLanguageTaskRuntimeConfig(),
	}
}

func NormalizeModuleConfig(config ModuleConfig) ModuleConfig {
	result := config
	if result.Retention.WorkspaceStatesMax <= 0 {
		result = DefaultModuleConfig()
	} else {
		result.TranslateLanguages = normalizeTranslateLanguagesConfig(result.TranslateLanguages)
		result.LanguageAssets = normalizeLanguageAssetsConfig(result.LanguageAssets)
		result.SubtitleStyles = normalizeSubtitleStyleConfig(result.SubtitleStyles)
	}
	if strings.TrimSpace(result.SubtitleStorage.DownloadPolicy) == "" {
		result.SubtitleStorage.DownloadPolicy = DefaultModuleConfig().SubtitleStorage.DownloadPolicy
	}
	if strings.TrimSpace(result.SubtitleStorage.ImportPolicy) == "" {
		result.SubtitleStorage.ImportPolicy = DefaultModuleConfig().SubtitleStorage.ImportPolicy
	}
	result.TaskRuntime = normalizeLanguageTaskRuntimeConfig(result.TaskRuntime)
	return result
}

func NewLibrary(params LibraryParams) (Library, error) {
	id := strings.TrimSpace(params.ID)
	name := strings.TrimSpace(params.Name)
	if id == "" || name == "" {
		return Library{}, ErrInvalidLibrary
	}
	createdAt := time.Now().UTC()
	if params.CreatedAt != nil && !params.CreatedAt.IsZero() {
		createdAt = params.CreatedAt.UTC()
	}
	updatedAt := createdAt
	if params.UpdatedAt != nil && !params.UpdatedAt.IsZero() {
		updatedAt = params.UpdatedAt.UTC()
	}
	meta := params.CreatedBy
	meta.Source = strings.TrimSpace(meta.Source)
	meta.TriggerOperationID = strings.TrimSpace(meta.TriggerOperationID)
	meta.ImportBatchID = strings.TrimSpace(meta.ImportBatchID)
	meta.Actor = strings.TrimSpace(meta.Actor)
	return Library{
		ID:        id,
		Name:      name,
		CreatedBy: meta,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}, nil
}

func defaultBuiltinTranslateLanguages() []TranslateLanguage {
	return []TranslateLanguage{
		{Code: "en", Label: "English", Aliases: []string{"eng", "english"}},
		{Code: "zh-CN", Label: "Simplified Chinese", Aliases: []string{"zh", "zho", "chi", "cmn", "chinese", "mandarin", "zh-cn", "zh_cn", "zh-hans", "zh_hans", "simplified chinese", "simplified-chinese"}},
		{Code: "zh-TW", Label: "Traditional Chinese", Aliases: []string{"zh-tw", "zh_tw", "zh-hk", "zh_hk", "zh-hant", "zh_hant", "traditional chinese", "traditional-chinese"}},
		{Code: "ja", Label: "Japanese", Aliases: []string{"jpn", "japanese"}},
		{Code: "ko", Label: "Korean", Aliases: []string{"kor", "korean"}},
		{Code: "es", Label: "Spanish", Aliases: []string{"spa", "spanish", "espanol"}},
		{Code: "fr", Label: "French", Aliases: []string{"fra", "fre", "french"}},
		{Code: "de", Label: "German", Aliases: []string{"deu", "ger", "german"}},
		{Code: "pt", Label: "Portuguese", Aliases: []string{"por", "portuguese"}},
		{Code: "pt-BR", Label: "Brazilian Portuguese", Aliases: []string{"pt-br", "pt_br", "brazilian portuguese", "brazilian-portuguese"}},
		{Code: "it", Label: "Italian", Aliases: []string{"ita", "italian"}},
		{Code: "ru", Label: "Russian", Aliases: []string{"rus", "russian"}},
		{Code: "ar", Label: "Arabic", Aliases: []string{"ara", "arabic"}},
		{Code: "hi", Label: "Hindi", Aliases: []string{"hin", "hindi"}},
		{Code: "id", Label: "Indonesian", Aliases: []string{"ind", "indonesian", "bahasa indonesia"}},
		{Code: "vi", Label: "Vietnamese", Aliases: []string{"vie", "vietnamese"}},
		{Code: "th", Label: "Thai", Aliases: []string{"tha", "thai"}},
		{Code: "tr", Label: "Turkish", Aliases: []string{"tur", "turkish"}},
	}
}

func normalizeTranslateLanguagesConfig(config TranslateLanguagesConfig) TranslateLanguagesConfig {
	result := TranslateLanguagesConfig{
		Builtin: normalizeTranslateLanguageList(config.Builtin, defaultBuiltinTranslateLanguages()),
	}
	if len(result.Builtin) == 0 {
		result.Builtin = defaultBuiltinTranslateLanguages()
	}
	result.Custom = normalizeTranslateLanguageList(config.Custom, nil)
	return result
}

func normalizeTranslateLanguageList(values []TranslateLanguage, fallback []TranslateLanguage) []TranslateLanguage {
	source := values
	if len(source) == 0 && len(fallback) > 0 {
		source = fallback
	}
	result := make([]TranslateLanguage, 0, len(source))
	seen := make(map[string]struct{}, len(source))
	for _, value := range source {
		code := normalizeTranslateLanguageCode(value.Code)
		if code == "" {
			continue
		}
		if _, exists := seen[strings.ToLower(code)]; exists {
			continue
		}
		seen[strings.ToLower(code)] = struct{}{}
		label := strings.TrimSpace(value.Label)
		if label == "" {
			label = code
		}
		aliases := normalizeTranslateLanguageAliases(value.Aliases, code, label)
		result = append(result, TranslateLanguage{
			Code:    code,
			Label:   label,
			Aliases: aliases,
		})
	}
	return result
}

func normalizeTranslateLanguageAliases(values []string, code string, label string) []string {
	result := make([]string, 0, len(values)+2)
	seen := map[string]struct{}{}
	for _, candidate := range append([]string{code, label}, values...) {
		trimmed := strings.TrimSpace(candidate)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func normalizeTranslateLanguageCode(value string) string {
	trimmed := strings.TrimSpace(strings.ReplaceAll(value, "_", "-"))
	if trimmed == "" {
		return ""
	}
	lower := strings.ToLower(trimmed)
	switch lower {
	case "eng", "english":
		return "en"
	case "zho", "chi", "chinese", "zh", "cmn", "mandarin", "zh-cn", "zh-hans":
		return "zh-CN"
	case "zh-tw", "zh-hant", "zh-hk":
		return "zh-TW"
	case "jpn", "japanese":
		return "ja"
	case "kor", "korean":
		return "ko"
	case "spa", "spanish", "espanol":
		return "es"
	case "fra", "fre", "french":
		return "fr"
	case "deu", "ger", "german":
		return "de"
	case "por", "portuguese":
		return "pt"
	case "pt-br":
		return "pt-BR"
	case "ita", "italian":
		return "it"
	case "rus", "russian":
		return "ru"
	case "ara", "arabic":
		return "ar"
	case "hin", "hindi":
		return "hi"
	case "ind", "indonesian":
		return "id"
	case "vie", "vietnamese":
		return "vi"
	case "tha", "thai":
		return "th"
	case "tur", "turkish":
		return "tr"
	}
	if len(lower) == 2 {
		return lower
	}
	if len(lower) == 5 && lower[2] == '-' {
		return lower[:2] + "-" + strings.ToUpper(lower[3:])
	}
	return trimmed
}

func normalizeLanguageAssetsConfig(config LanguageAssetsConfig) LanguageAssetsConfig {
	return LanguageAssetsConfig{
		GlossaryProfiles: normalizeGlossaryProfiles(config.GlossaryProfiles),
		PromptProfiles:   normalizePromptProfiles(config.PromptProfiles),
	}
}

func defaultLanguageTaskRuntimeConfig() LanguageTaskRuntimeConfig {
	defaults := LanguageTaskRuntimeSettings{
		StructuredOutputMode: "auto",
		ThinkingMode:         "off",
		MaxTokensFloor:       2048,
		MaxTokensCeiling:     6144,
		RetryTokenStep:       1024,
	}
	return LanguageTaskRuntimeConfig{
		Translate: defaults,
		Proofread: defaults,
	}
}

func normalizeLanguageTaskRuntimeConfig(config LanguageTaskRuntimeConfig) LanguageTaskRuntimeConfig {
	defaults := defaultLanguageTaskRuntimeConfig()
	return LanguageTaskRuntimeConfig{
		Translate: normalizeLanguageTaskRuntimeSettings(config.Translate, defaults.Translate),
		Proofread: normalizeLanguageTaskRuntimeSettings(config.Proofread, defaults.Proofread),
	}
}

func normalizeLanguageTaskRuntimeSettings(value LanguageTaskRuntimeSettings, fallback LanguageTaskRuntimeSettings) LanguageTaskRuntimeSettings {
	structuredOutputMode := normalizeLanguageTaskStructuredOutputMode(value.StructuredOutputMode)
	if structuredOutputMode == "" {
		structuredOutputMode = fallback.StructuredOutputMode
	}
	thinkingMode := normalizeLanguageTaskThinkingMode(value.ThinkingMode)
	if thinkingMode == "" {
		thinkingMode = fallback.ThinkingMode
	}
	maxTokensFloor := value.MaxTokensFloor
	if maxTokensFloor <= 0 {
		maxTokensFloor = fallback.MaxTokensFloor
	}
	maxTokensCeiling := value.MaxTokensCeiling
	if maxTokensCeiling <= 0 {
		maxTokensCeiling = fallback.MaxTokensCeiling
	}
	if maxTokensCeiling < maxTokensFloor {
		maxTokensCeiling = maxTokensFloor
	}
	retryTokenStep := value.RetryTokenStep
	if retryTokenStep <= 0 {
		retryTokenStep = fallback.RetryTokenStep
	}
	return LanguageTaskRuntimeSettings{
		StructuredOutputMode: structuredOutputMode,
		ThinkingMode:         thinkingMode,
		MaxTokensFloor:       maxTokensFloor,
		MaxTokensCeiling:     maxTokensCeiling,
		RetryTokenStep:       retryTokenStep,
	}
}

func normalizeLanguageTaskStructuredOutputMode(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "auto":
		return "auto"
	case "json_schema", "json-schema", "jsonschema", "schema":
		return "json_schema"
	case "prompt_only", "prompt-only", "promptonly", "prompt":
		return "prompt_only"
	case "off", "none", "disabled", "disable":
		return "prompt_only"
	default:
		return ""
	}
}

func normalizeLanguageTaskThinkingMode(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "off", "none", "disabled", "disable", "false", "0":
		return "off"
	case "minimal", "min",
		"low", "on", "enabled", "enable", "true", "1",
		"medium", "med",
		"high", "max",
		"xhigh", "x-high", "x_high", "extra-high", "extra_high", "extra high":
		return "on"
	default:
		return ""
	}
}

func normalizeGlossaryProfiles(values []GlossaryProfile) []GlossaryProfile {
	result := make([]GlossaryProfile, 0, len(values))
	seen := map[string]struct{}{}
	for index, value := range values {
		if !glossaryProfileHasContent(value) {
			continue
		}
		id := normalizeAssetID(value.ID, value.Name, fmt.Sprintf("glossary-%d", index+1))
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		name := strings.TrimSpace(value.Name)
		if name == "" {
			name = fmt.Sprintf("Glossary %d", index+1)
		}
		result = append(result, GlossaryProfile{
			ID:             id,
			Name:           name,
			Category:       normalizeGlossaryCategory(value.Category),
			Description:    strings.TrimSpace(value.Description),
			SourceLanguage: normalizeGlossaryLanguageScope(value.SourceLanguage),
			TargetLanguage: normalizeGlossaryLanguageScope(value.TargetLanguage),
			Terms:          normalizeGlossaryTerms(value.Terms),
		})
	}
	return result
}

func normalizeGlossaryTerms(values []GlossaryTerm) []GlossaryTerm {
	result := make([]GlossaryTerm, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		source := strings.TrimSpace(value.Source)
		target := strings.TrimSpace(value.Target)
		if source == "" || target == "" {
			continue
		}
		key := strings.ToLower(source) + "->" + strings.ToLower(target)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, GlossaryTerm{
			Source: source,
			Target: target,
			Note:   strings.TrimSpace(value.Note),
		})
	}
	return result
}

func normalizePromptProfiles(values []PromptProfile) []PromptProfile {
	result := make([]PromptProfile, 0, len(values))
	seen := map[string]struct{}{}
	for index, value := range values {
		if !promptProfileHasContent(value) {
			continue
		}
		id := normalizeAssetID(value.ID, value.Name, fmt.Sprintf("prompt-%d", index+1))
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		name := strings.TrimSpace(value.Name)
		if name == "" {
			name = fmt.Sprintf("Prompt %d", index+1)
		}
		result = append(result, PromptProfile{
			ID:          id,
			Name:        name,
			Category:    normalizePromptCategory(value.Category),
			Description: strings.TrimSpace(value.Description),
			Prompt:      strings.TrimSpace(value.Prompt),
		})
	}
	return result
}

func glossaryProfileHasContent(value GlossaryProfile) bool {
	if strings.TrimSpace(value.Name) != "" ||
		strings.TrimSpace(value.Description) != "" ||
		strings.TrimSpace(value.Category) != "" ||
		strings.TrimSpace(value.SourceLanguage) != "" ||
		strings.TrimSpace(value.TargetLanguage) != "" {
		return true
	}
	for _, term := range value.Terms {
		if strings.TrimSpace(term.Source) != "" || strings.TrimSpace(term.Target) != "" || strings.TrimSpace(term.Note) != "" {
			return true
		}
	}
	return false
}

func promptProfileHasContent(value PromptProfile) bool {
	return strings.TrimSpace(value.Name) != "" ||
		strings.TrimSpace(value.Description) != "" ||
		strings.TrimSpace(value.Prompt) != "" ||
		strings.TrimSpace(value.Category) != ""
}

func normalizePromptCategory(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "all", "glossary", "translate", "proofread":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "all"
	}
}

func normalizeGlossaryCategory(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "all", "translate", "proofread":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "all"
	}
}

func normalizeGlossaryLanguageScope(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || strings.EqualFold(trimmed, "all") {
		return "all"
	}
	normalized := normalizeTranslateLanguageCode(trimmed)
	if normalized == "" {
		return "all"
	}
	return normalized
}

func glossaryProfileIDSet(values []GlossaryProfile) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		result[value.ID] = struct{}{}
	}
	return result
}

func promptProfileIDSet(values []PromptProfile) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		result[value.ID] = struct{}{}
	}
	return result
}

func normalizeSelectedIDs(values []string, allowed map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, exists := allowed[trimmed]; !exists {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func normalizeSingleSelectedID(value string, allowed map[string]struct{}) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	if _, exists := allowed[trimmed]; !exists {
		return ""
	}
	return trimmed
}

func normalizeAssetID(explicit string, name string, fallback string) string {
	candidate := strings.TrimSpace(explicit)
	if candidate == "" {
		candidate = strings.TrimSpace(name)
	}
	if candidate == "" {
		candidate = fallback
	}
	var builder strings.Builder
	lastHyphen := false
	for _, char := range strings.ToLower(candidate) {
		switch {
		case unicode.IsLetter(char) || unicode.IsDigit(char):
			builder.WriteRune(char)
			lastHyphen = false
		case !lastHyphen:
			builder.WriteRune('-')
			lastHyphen = true
		}
	}
	normalized := strings.Trim(builder.String(), "-")
	if normalized == "" {
		return fallback
	}
	return normalized
}
