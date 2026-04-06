package service

import (
	"path/filepath"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"dreamcreator/internal/domain/library"
)

var latinLanguageStopwords = map[string][]string{
	"en":    {"the", "and", "you", "that", "with", "this", "for", "are", "not", "have"},
	"es":    {"que", "los", "las", "por", "con", "para", "una", "pero", "como", "estoy"},
	"fr":    {"que", "les", "des", "pour", "avec", "une", "dans", "mais", "vous", "est"},
	"de":    {"und", "der", "die", "das", "nicht", "mit", "ist", "eine", "ich", "auf"},
	"pt":    {"que", "com", "para", "uma", "não", "você", "está", "por", "mas", "isso"},
	"pt-BR": {"que", "com", "para", "uma", "não", "você", "está", "por", "mas", "isso"},
	"it":    {"che", "con", "per", "una", "non", "sono", "come", "questo", "della", "degli"},
	"id":    {"yang", "dan", "untuk", "dengan", "ini", "tidak", "anda", "saya", "kami", "dari"},
	"vi":    {"khong", "toi", "ban", "voi", "nhung", "mot", "cho", "nay", "cua", "la"},
	"tr":    {"ve", "bir", "icin", "degil", "sen", "ben", "ama", "gibi", "bu", "olan"},
}

var simplifiedChineseMarkers = []rune{'这', '们', '国', '说', '时', '后', '语', '会'}
var traditionalChineseMarkers = []rune{'這', '們', '國', '說', '時', '後', '語', '會'}

func effectiveTranslateLanguages(config library.ModuleConfig) []library.TranslateLanguage {
	if len(config.TranslateLanguages.Builtin) == 0 && len(config.TranslateLanguages.Custom) == 0 {
		return library.DefaultModuleConfig().TranslateLanguages.Builtin
	}
	result := make([]library.TranslateLanguage, 0, len(config.TranslateLanguages.Builtin)+len(config.TranslateLanguages.Custom))
	seen := make(map[string]struct{}, len(config.TranslateLanguages.Builtin)+len(config.TranslateLanguages.Custom))
	for _, value := range append(append([]library.TranslateLanguage(nil), config.TranslateLanguages.Builtin...), config.TranslateLanguages.Custom...) {
		code := strings.TrimSpace(strings.ToLower(value.Code))
		if code == "" {
			continue
		}
		if _, exists := seen[code]; exists {
			continue
		}
		seen[code] = struct{}{}
		result = append(result, value)
	}
	return result
}

func detectSubtitleLanguage(item library.LibraryFile, content string, config library.ModuleConfig) string {
	catalog := effectiveTranslateLanguages(config)
	if language := detectSubtitleLanguageByAliases(item, catalog); language != "" {
		return language
	}
	if language := detectSubtitleLanguageByContent(content, catalog); language != "" {
		return language
	}
	if strings.TrimSpace(content) != "" {
		return "other"
	}
	return ""
}

func detectSubtitleLanguageByAliases(item library.LibraryFile, catalog []library.TranslateLanguage) string {
	if len(catalog) == 0 {
		return ""
	}
	candidates := []string{
		item.Name,
		item.Storage.LocalPath,
	}
	if item.Origin.Import != nil {
		candidates = append(candidates, item.Origin.Import.ImportPath)
	}
	bestCode := ""
	bestLength := 0
	for _, candidate := range candidates {
		comparable := subtitleComparableValue(candidate)
		if comparable == "" {
			continue
		}
		for _, language := range catalog {
			for _, alias := range subtitleLanguageAliases(language) {
				comparableAlias := subtitleComparableValue(alias)
				if comparableAlias == "" {
					continue
				}
				if comparable == comparableAlias || strings.Contains(" "+comparable+" ", " "+comparableAlias+" ") {
					if len(comparableAlias) > bestLength {
						bestCode = language.Code
						bestLength = len(comparableAlias)
					}
				}
			}
		}
	}
	return bestCode
}

func detectSubtitleLanguageByContent(content string, catalog []library.TranslateLanguage) string {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" || len(catalog) == 0 {
		return ""
	}
	available := make(map[string]library.TranslateLanguage, len(catalog))
	for _, language := range catalog {
		available[strings.ToLower(language.Code)] = language
	}
	if code := detectSubtitleScriptLanguage(trimmed, available); code != "" {
		return code
	}
	return detectSubtitleLatinLanguage(trimmed, available)
}

func detectSubtitleScriptLanguage(content string, available map[string]library.TranslateLanguage) string {
	var hasKana bool
	var hasHangul bool
	var hasArabic bool
	var hasDevanagari bool
	var hasThai bool
	var hasCyrillic bool
	var hanCount int
	var simplifiedCount int
	var traditionalCount int
	for _, value := range content {
		switch {
		case unicode.In(value, unicode.Hiragana, unicode.Katakana):
			hasKana = true
		case unicode.In(value, unicode.Hangul):
			hasHangul = true
		case unicode.In(value, unicode.Arabic):
			hasArabic = true
		case unicode.In(value, unicode.Devanagari):
			hasDevanagari = true
		case unicode.In(value, unicode.Thai):
			hasThai = true
		case unicode.In(value, unicode.Cyrillic):
			hasCyrillic = true
		case unicode.In(value, unicode.Han):
			hanCount++
			if containsRune(simplifiedChineseMarkers, value) {
				simplifiedCount++
			}
			if containsRune(traditionalChineseMarkers, value) {
				traditionalCount++
			}
		}
	}
	switch {
	case hasKana:
		return firstAvailableLanguage(available, "ja")
	case hasHangul:
		return firstAvailableLanguage(available, "ko")
	case hasArabic:
		return firstAvailableLanguage(available, "ar")
	case hasDevanagari:
		return firstAvailableLanguage(available, "hi")
	case hasThai:
		return firstAvailableLanguage(available, "th")
	case hasCyrillic:
		return firstAvailableLanguage(available, "ru")
	case hanCount > 0:
		if traditionalCount > simplifiedCount {
			if code := firstAvailableLanguage(available, "zh-TW", "zh-CN"); code != "" {
				return code
			}
		}
		return firstAvailableLanguage(available, "zh-CN", "zh-TW")
	default:
		return ""
	}
}

func detectSubtitleLatinLanguage(content string, available map[string]library.TranslateLanguage) string {
	tokens := tokenizeSubtitleContent(content)
	if len(tokens) == 0 {
		return ""
	}
	scores := make(map[string]int)
	for code := range available {
		stopwords, ok := latinLanguageStopwords[code]
		if !ok {
			continue
		}
		for _, token := range tokens {
			for _, stopword := range stopwords {
				if token == stopword {
					scores[code]++
				}
			}
		}
	}
	type scoredLanguage struct {
		code  string
		score int
	}
	ranked := make([]scoredLanguage, 0, len(scores))
	for code, score := range scores {
		if score <= 0 {
			continue
		}
		ranked = append(ranked, scoredLanguage{code: code, score: score})
	}
	sort.Slice(ranked, func(left int, right int) bool {
		if ranked[left].score == ranked[right].score {
			return ranked[left].code < ranked[right].code
		}
		return ranked[left].score > ranked[right].score
	})
	if len(ranked) == 0 {
		return ""
	}
	best := ranked[0]
	if best.score < 2 {
		return ""
	}
	if len(ranked) > 1 && ranked[1].score == best.score {
		return ""
	}
	return available[best.code].Code
}

func subtitleLanguageAliases(value library.TranslateLanguage) []string {
	result := make([]string, 0, len(value.Aliases)+2)
	result = append(result, value.Code, value.Label)
	result = append(result, value.Aliases...)
	return result
}

func subtitleComparableValue(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	if strings.Contains(trimmed, "/") || strings.Contains(trimmed, "\\") {
		base := filepath.Base(trimmed)
		trimmed = strings.TrimSuffix(base, filepath.Ext(base))
	}
	var builder strings.Builder
	builder.Grow(len(trimmed))
	lastSpace := true
	for _, current := range strings.ToLower(trimmed) {
		switch {
		case unicode.IsLetter(current), unicode.IsDigit(current):
			builder.WriteRune(current)
			lastSpace = false
		default:
			if !lastSpace {
				builder.WriteByte(' ')
				lastSpace = true
			}
		}
	}
	return strings.TrimSpace(builder.String())
}

func tokenizeSubtitleContent(content string) []string {
	normalized := strings.ToLower(strings.TrimSpace(content))
	if normalized == "" {
		return nil
	}
	var builder strings.Builder
	builder.Grow(len(normalized))
	for _, value := range normalized {
		switch {
		case unicode.IsLetter(value):
			builder.WriteRune(value)
		case value > utf8.RuneSelf:
			builder.WriteByte(' ')
		default:
			builder.WriteByte(' ')
		}
	}
	fields := strings.Fields(builder.String())
	result := make([]string, 0, len(fields))
	for _, field := range fields {
		ascii := stripDiacritics(field)
		if ascii == "" {
			continue
		}
		result = append(result, ascii)
	}
	return result
}

func stripDiacritics(value string) string {
	var builder strings.Builder
	builder.Grow(len(value))
	for _, current := range value {
		switch current {
		case 'á', 'à', 'â', 'ä', 'ã', 'å':
			builder.WriteByte('a')
		case 'ç':
			builder.WriteByte('c')
		case 'é', 'è', 'ê', 'ë':
			builder.WriteByte('e')
		case 'í', 'ì', 'î', 'ï':
			builder.WriteByte('i')
		case 'ñ':
			builder.WriteByte('n')
		case 'ó', 'ò', 'ô', 'ö', 'õ':
			builder.WriteByte('o')
		case 'ú', 'ù', 'û', 'ü':
			builder.WriteByte('u')
		case 'ý', 'ÿ':
			builder.WriteByte('y')
		case 'ğ':
			builder.WriteByte('g')
		case 'ş':
			builder.WriteByte('s')
		case 'ı':
			builder.WriteByte('i')
		case 'đ':
			builder.WriteByte('d')
		case 'ă':
			builder.WriteByte('a')
		case 'ơ':
			builder.WriteByte('o')
		case 'ư':
			builder.WriteByte('u')
		default:
			if current <= unicode.MaxASCII {
				builder.WriteRune(current)
			}
		}
	}
	return builder.String()
}

func firstAvailableLanguage(available map[string]library.TranslateLanguage, codes ...string) string {
	for _, code := range codes {
		if language, ok := available[strings.ToLower(code)]; ok {
			return language.Code
		}
	}
	return ""
}

func containsRune(values []rune, target rune) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
