package subtitles

import (
	"math"
	"strings"
	"unicode"
)

// LanguageDetectorImpl 基于Unicode统计分析的语言检测器
type LanguageDetectorImpl struct{}

// NewLanguageDetector 创建语言检测器实例
func NewLanguageDetector() LanguageDetector {
	return &LanguageDetectorImpl{}
}

// DetectLanguage 检测文本语言
func (ld *LanguageDetectorImpl) DetectLanguage(text string) string {
	if len(strings.TrimSpace(text)) < 3 {
		return "unknown"
	}

	stats := analyzeUnicodeStats(text)
	return classifyLanguage(stats, text)
}

// DetectLanguageInt 返回语言代码和名称
func (ld *LanguageDetectorImpl) DetectLanguageInt(text string) (int, string) {
	lang := ld.DetectLanguage(text)
	return getLanguageCode(lang), lang
}

// UnicodeStats Unicode字符统计信息
type UnicodeStats struct {
	Han         float64 // 汉字比例
	Hiragana    float64 // 平假名比例
	Katakana    float64 // 片假名比例
	Hangul      float64 // 韩文比例
	Cyrillic    float64 // 西里尔字母比例
	Arabic      float64 // 阿拉伯字母比例
	Latin       float64 // 拉丁字母比例
	Thai        float64 // 泰文比例
	Devanagari  float64 // 天城文比例
	Hebrew      float64 // 希伯来文比例
	Greek       float64 // 希腊文比例
	Total       int     // 总字符数
	Simplified  float64 // 简体中文特征分数
	Traditional float64 // 繁体中文特征分数
}

// analyzeUnicodeStats 分析文本的Unicode字符统计
func analyzeUnicodeStats(text string) UnicodeStats {
	stats := UnicodeStats{}
	var counts struct {
		han, hiragana, katakana, hangul, cyrillic, arabic, latin, thai, devanagari, hebrew, greek int
		simplifiedScore, traditionalScore                                                         float64
	}

	for _, r := range text {
		if unicode.IsSpace(r) || unicode.IsPunct(r) || unicode.IsDigit(r) {
			continue
		}
		stats.Total++

		switch {
		case unicode.Is(unicode.Han, r):
			counts.han++
			// 简繁体特征分析
			if isSimplifiedChar(r) {
				counts.simplifiedScore += 1.0
			} else if isTraditionalChar(r) {
				counts.traditionalScore += 1.0
			}
		case unicode.Is(unicode.Hiragana, r):
			counts.hiragana++
		case unicode.Is(unicode.Katakana, r):
			counts.katakana++
		case unicode.Is(unicode.Hangul, r):
			counts.hangul++
		case unicode.Is(unicode.Cyrillic, r):
			counts.cyrillic++
		case unicode.Is(unicode.Arabic, r):
			counts.arabic++
		case unicode.Is(unicode.Latin, r):
			counts.latin++
		case unicode.Is(unicode.Thai, r):
			counts.thai++
		case unicode.Is(unicode.Devanagari, r):
			counts.devanagari++
		case unicode.Is(unicode.Hebrew, r):
			counts.hebrew++
		case unicode.Is(unicode.Greek, r):
			counts.greek++
		}
	}

	if stats.Total == 0 {
		return stats
	}

	// 计算比例
	total := float64(stats.Total)
	stats.Han = float64(counts.han) / total
	stats.Hiragana = float64(counts.hiragana) / total
	stats.Katakana = float64(counts.katakana) / total
	stats.Hangul = float64(counts.hangul) / total
	stats.Cyrillic = float64(counts.cyrillic) / total
	stats.Arabic = float64(counts.arabic) / total
	stats.Latin = float64(counts.latin) / total
	stats.Thai = float64(counts.thai) / total
	stats.Devanagari = float64(counts.devanagari) / total
	stats.Hebrew = float64(counts.hebrew) / total
	stats.Greek = float64(counts.greek) / total
	stats.Simplified = counts.simplifiedScore / float64(counts.han+1)
	stats.Traditional = counts.traditionalScore / float64(counts.han+1)

	return stats
}

// classifyLanguage 基于统计信息分类语言
func classifyLanguage(stats UnicodeStats, text string) string {
	const threshold = 0.1 // 10%阈值

	// 非拉丁文字系统优先检测
	if stats.Thai > threshold {
		return "Thai"
	}
	if stats.Devanagari > threshold {
		return "Hindi"
	}
	if stats.Hebrew > threshold {
		return "Hebrew"
	}
	if stats.Arabic > threshold {
		return "Arabic"
	}
	if stats.Hangul > threshold {
		return "Korean"
	}
	if stats.Hiragana > threshold || stats.Katakana > threshold {
		return "Japanese"
	}
	if stats.Han > threshold {
		// 简繁体中文区分
		if stats.Simplified > stats.Traditional {
			return "Chinese (Simplified)"
		} else if stats.Traditional > stats.Simplified {
			return "Chinese (Traditional)"
		}
		return "Chinese"
	}
	if stats.Cyrillic > threshold {
		return "Russian"
	}
	if stats.Greek > threshold {
		return "Greek"
	}

	// 拉丁文字系统使用字符频率分析
	if stats.Latin > threshold {
		return detectLatinLanguage(text)
	}

	return "English" // 默认
}

// detectLatinLanguage 检测拉丁文字语言
func detectLatinLanguage(text string) string {
	lowerText := strings.ToLower(text)
	charFreq := make(map[rune]int)

	// 统计字符频率
	for _, r := range lowerText {
		if r >= 'a' && r <= 'z' {
			charFreq[r]++
		}
	}

	// 计算语言特征分数
	scores := map[string]float64{
		"English":    calculateEnglishScore(charFreq, lowerText),
		"French":     calculateFrenchScore(charFreq, lowerText),
		"German":     calculateGermanScore(charFreq, lowerText),
		"Spanish":    calculateSpanishScore(charFreq, lowerText),
		"Italian":    calculateItalianScore(charFreq, lowerText),
		"Portuguese": calculatePortugueseScore(charFreq, lowerText),
		"Dutch":      calculateDutchScore(charFreq, lowerText),
		"Polish":     calculatePolishScore(charFreq, lowerText),
		"Czech":      calculateCzechScore(charFreq, lowerText),
		"Vietnamese": calculateVietnameseScore(charFreq, lowerText),
	}

	// 找出最高分数的语言
	maxScore := 0.0
	bestLang := "English"
	for lang, score := range scores {
		if score > maxScore {
			maxScore = score
			bestLang = lang
		}
	}

	// 提高阈值，但如果英语分数接近最高分，优先选择英语
	if maxScore < 1.5 || (bestLang != "English" && scores["English"] > maxScore*0.85) {
		return "English"
	}

	return bestLang
}

// calculateEnglishScore 改进版：充分利用字符频率统计
func calculateEnglishScore(charFreq map[rune]int, text string) float64 {
	score := 0.0
	totalChars := getTotalChars(charFreq)

	if totalChars == 0 {
		return 0.0
	}

	// 1. 利用字符频率进行英语特征检测
	englishFreq := map[rune]float64{
		'e': 12.7, 't': 9.1, 'a': 8.2, 'o': 7.5, 'i': 7.0, 'n': 6.7,
		's': 6.3, 'h': 6.1, 'r': 6.0, 'd': 4.3, 'l': 4.0, 'c': 2.8,
	}

	// 计算字符频率相似度得分
	freqScore := 0.0
	for char, expectedFreq := range englishFreq {
		if count, exists := charFreq[char]; exists {
			actualFreq := float64(count) / float64(totalChars) * 100
			// 计算频率差异，差异越小得分越高
			diff := math.Abs(actualFreq - expectedFreq)
			if diff < expectedFreq {
				freqScore += (expectedFreq - diff) / expectedFreq
			}
		}
	}
	score += freqScore / float64(len(englishFreq)) * 2.0

	// 2. 检查是否有非英语特征字符（利用charFreq）
	nonEnglishChars := []rune{'à', 'á', 'â', 'ã', 'ä', 'å', 'æ', 'ç', 'è', 'é', 'ê', 'ë', 'ì', 'í', 'î', 'ï', 'ñ', 'ò', 'ó', 'ô', 'õ', 'ö', 'ø', 'ù', 'ú', 'û', 'ü', 'ý', 'ÿ', 'ß'}
	for _, char := range nonEnglishChars {
		if count := charFreq[char]; count > 0 {
			// 根据非英语字符的频率进行减分
			score -= float64(count) / float64(totalChars) * 10.0
		}
	}

	// 3. 英语常见字母组合检测
	if charFreq['t'] > 0 && charFreq['h'] > 0 {
		score += 0.5 // th组合
	}
	if charFreq['i'] > 0 && charFreq['n'] > 0 && charFreq['g'] > 0 {
		score += 0.3 // ing可能性
	}

	// 4. 文本模式检测（作为补充）
	englishWords := []string{" the ", " and ", " of ", " to ", " in ", " is ", " it ", " for "}
	wordCount := 0
	for _, word := range englishWords {
		if strings.Contains(" "+text+" ", word) {
			wordCount++
		}
	}
	score += float64(wordCount) * 0.2

	return math.Max(0, score)
}

// calculateFrenchScore 改进版：结合字符频率和特征检测
func calculateFrenchScore(charFreq map[rune]int, text string) float64 {
	score := 0.0
	totalChars := getTotalChars(charFreq)

	if totalChars == 0 {
		return 0.0
	}

	// 1. 法语特征字符检测（利用charFreq精确计算）
	frenchChars := map[rune]float64{
		'à': 2.0, 'é': 3.0, 'è': 1.5, 'ê': 1.0, 'ë': 0.5,
		'ç': 1.0, 'ô': 0.8, 'û': 0.5, 'ù': 0.3, 'î': 0.5,
		'ï': 0.3, 'â': 0.5, 'ä': 0.3,
	}

	frenchCharScore := 0.0
	for char, weight := range frenchChars {
		if count := charFreq[char]; count > 0 {
			frenchCharScore += float64(count) / float64(totalChars) * 100 * weight
		}
	}
	score += frenchCharScore

	// 如果没有法语特征字符，大幅减分
	if frenchCharScore == 0 {
		score -= 2.0
	}

	// 2. 法语字母频率特征
	if totalChars > 0 {
		// 法语中'e'的频率很高（约17%）
		eFreq := float64(charFreq['e']) / float64(totalChars)
		if eFreq > 0.15 && eFreq < 0.20 {
			score += 1.0
		}
	}

	// 3. 法语词汇检测
	frenchWords := []string{" le ", " la ", " les ", " de ", " du ", " des ", " un ", " une ", " et ", " est "}
	wordCount := 0
	for _, word := range frenchWords {
		if strings.Contains(" "+text+" ", word) {
			wordCount++
		}
	}
	score += float64(wordCount) * 0.3

	return math.Max(0, score)
}

// calculateItalianScore 改进版：结合字符频率统计
func calculateItalianScore(charFreq map[rune]int, text string) float64 {
	score := 0.0
	totalChars := getTotalChars(charFreq)

	if totalChars == 0 {
		return 0.0
	}

	// 1. 意大利语特征字符检测
	italianChars := map[rune]float64{
		'à': 1.5, 'è': 2.0, 'é': 1.0, 'ì': 1.5, 'í': 0.5,
		'ò': 1.5, 'ó': 0.5, 'ù': 1.0, 'ú': 0.3,
	}

	italianCharScore := 0.0
	for char, weight := range italianChars {
		if count := charFreq[char]; count > 0 {
			italianCharScore += float64(count) / float64(totalChars) * 100 * weight
		}
	}
	score += italianCharScore

	// 如果没有意大利语特征字符，减分
	if italianCharScore == 0 {
		score -= 1.5
	}

	// 2. 意大利语元音频率特征（元音比例较高）
	vowels := []rune{'a', 'e', 'i', 'o', 'u'}
	vowelCount := 0
	for _, vowel := range vowels {
		vowelCount += charFreq[vowel]
	}
	if totalChars > 0 {
		vowelRatio := float64(vowelCount) / float64(totalChars)
		if vowelRatio > 0.45 { // 意大利语元音比例高
			score += 1.0
		}
	}

	// 3. 意大利语词汇检测
	italianWords := []string{" il ", " la ", " le ", " di ", " del ", " della ", " un ", " una ", " che ", " con "}
	wordCount := 0
	for _, word := range italianWords {
		if strings.Contains(" "+text+" ", word) {
			wordCount++
		}
	}
	score += float64(wordCount) * 0.3

	return math.Max(0, score)
}

// calculateGermanScore 计算德语特征分数
func calculateGermanScore(charFreq map[rune]int, text string) float64 {
	score := 0.0

	// 德语特征字符
	germanChars := []rune{'ä', 'ö', 'ü', 'ß'}
	for _, char := range germanChars {
		if charFreq[char] > 0 {
			score += 2.0 // 德语特征字符权重更高
		}
	}

	// 德语常见模式
	germanPatterns := []string{"sch", "tsch", "ung", "keit", "heit", "lich"}
	for _, pattern := range germanPatterns {
		if strings.Contains(text, pattern) {
			score += 0.7
		}
	}

	// 德语复合词特征（长单词）
	words := strings.Fields(text)
	longWords := 0
	for _, word := range words {
		if len(word) > 10 {
			longWords++
		}
	}
	if len(words) > 0 && float64(longWords)/float64(len(words)) > 0.1 {
		score += 1.0
	}

	return score
}

// calculateSpanishScore 计算西班牙语特征分数
func calculateSpanishScore(charFreq map[rune]int, text string) float64 {
	score := 0.0

	// 西班牙语特征字符
	spanishChars := []rune{'ñ', 'á', 'é', 'í', 'ó', 'ú', 'ü'}
	for _, char := range spanishChars {
		if charFreq[char] > 0 {
			score += 1.5
		}
	}

	// 西班牙语常见模式
	spanishPatterns := []string{"ción", "mente", "ando", "endo", "illo", "ello"}
	for _, pattern := range spanishPatterns {
		if strings.Contains(text, pattern) {
			score += 0.6
		}
	}

	// 西班牙语双 'l' 和 'rr' 特征
	if strings.Contains(text, "ll") || strings.Contains(text, "rr") {
		score += 0.8
	}

	return score
}

// calculatePortugueseScore 计算葡萄牙语特征分数
func calculatePortugueseScore(charFreq map[rune]int, text string) float64 {
	score := 0.0

	// 葡萄牙语特征字符
	portugueseChars := []rune{'ã', 'õ', 'á', 'à', 'â', 'é', 'ê', 'í', 'ó', 'ô', 'ú', 'ç'}
	for _, char := range portugueseChars {
		if charFreq[char] > 0 {
			score += 1.3
		}
	}

	// 葡萄牙语常见模式
	portuguesePatterns := []string{"ção", "mente", "ando", "endo", "inho", "inha"}
	for _, pattern := range portuguesePatterns {
		if strings.Contains(text, pattern) {
			score += 0.7
		}
	}

	// 葡萄牙语特有的鼻音化特征
	if strings.Contains(text, "ão") || strings.Contains(text, "ões") {
		score += 1.5
	}

	return score
}

// calculateDutchScore 计算荷兰语特征分数
func calculateDutchScore(charFreq map[rune]int, text string) float64 {
	score := 0.0

	// 荷兰语特征字符
	dutchChars := []rune{'ë', 'ï', 'ö', 'ü', 'é', 'è', 'ê', 'á', 'à', 'â'}
	for _, char := range dutchChars {
		if charFreq[char] > 0 {
			score += 1.0
		}
	}

	// 荷兰语常见模式
	dutchPatterns := []string{"ij", "tion", "heid", "lijk", "baar"}
	for _, pattern := range dutchPatterns {
		if strings.Contains(text, pattern) {
			score += 0.8
		}
	}

	return score
}

// calculatePolishScore 计算波兰语特征分数
func calculatePolishScore(charFreq map[rune]int, text string) float64 {
	score := 0.0

	// 波兰语特征字符
	polishChars := []rune{'ą', 'ć', 'ę', 'ł', 'ń', 'ó', 'ś', 'ź', 'ż'}
	for _, char := range polishChars {
		if charFreq[char] > 0 {
			score += 2.0 // 波兰语特征字符权重很高
		}
	}

	// 波兰语常见模式
	polishPatterns := []string{"ów", "ach", "ych", "ość"}
	for _, pattern := range polishPatterns {
		if strings.Contains(text, pattern) {
			score += 0.7
		}
	}

	return score
}

// calculateCzechScore 计算捷克语特征分数
func calculateCzechScore(charFreq map[rune]int, text string) float64 {
	score := 0.0

	// 捷克语特征字符
	czechChars := []rune{'á', 'č', 'ď', 'é', 'ě', 'í', 'ň', 'ó', 'ř', 'š', 'ť', 'ú', 'ů', 'ý', 'ž'}
	for _, char := range czechChars {
		if charFreq[char] > 0 {
			score += 1.5
		}
	}

	// 捷克语常见模式
	czechPatterns := []string{"ní", "ost", "ení", "ová"}
	for _, pattern := range czechPatterns {
		if strings.Contains(text, pattern) {
			score += 0.6
		}
	}

	return score
}

// calculateVietnameseScore 计算越南语特征分数
func calculateVietnameseScore(charFreq map[rune]int, text string) float64 {
	score := 0.0
	totalChars := getTotalChars(charFreq)

	if totalChars == 0 {
		return 0.0
	}

	// 1. 越南语特征字符检测
	vietnameseChars := map[rune]float64{
		'ă': 3.0, 'â': 2.5, 'đ': 4.0, 'ê': 2.0, 'ô': 2.0, 'ơ': 3.0, 'ư': 3.0,
		'à': 1.5, 'á': 1.5, 'ả': 1.0, 'ã': 1.0, 'ạ': 1.5,
		'è': 1.0, 'é': 1.0, 'ẻ': 0.8, 'ẽ': 0.8, 'ẹ': 1.0,
		'ì': 1.0, 'í': 1.0, 'ỉ': 0.8, 'ĩ': 0.8, 'ị': 1.0,
		'ò': 1.0, 'ó': 1.0, 'ỏ': 0.8, 'õ': 0.8, 'ọ': 1.0,
		'ù': 1.0, 'ú': 1.0, 'ủ': 0.8, 'ũ': 0.8, 'ụ': 1.0,
		'ỳ': 0.5, 'ý': 0.5, 'ỷ': 0.3, 'ỹ': 0.3, 'ỵ': 0.5,
	}

	vietnameseCharScore := 0.0
	for _, r := range text {
		if weight, exists := vietnameseChars[r]; exists {
			vietnameseCharScore += weight
		}
	}
	score += vietnameseCharScore

	// 如果没有越南语特征字符，大幅减分
	if vietnameseCharScore == 0 {
		score -= 5.0
	}

	// 2. 越南语常用词检测
	vietnameseWords := []string{
		"việt", "nam", "của", "và", "có", "trong", "được", "cho", "với", "này",
		"một", "những", "từ", "người", "thì", "sẽ", "đã", "không", "tại", "về",
	}

	wordCount := 0
	for _, word := range vietnameseWords {
		if strings.Contains(strings.ToLower(text), word) {
			wordCount++
		}
	}
	score += float64(wordCount) * 0.5

	// 3. 越南语音节特征（单音节词较多）
	words := strings.Fields(text)
	monosyllableCount := 0
	for _, word := range words {
		// 简单的音节计数（基于元音）
		vowelCount := 0
		for _, r := range strings.ToLower(word) {
			if r == 'a' || r == 'e' || r == 'i' || r == 'o' || r == 'u' || r == 'y' {
				vowelCount++
			}
		}
		if vowelCount <= 2 {
			monosyllableCount++
		}
	}

	if len(words) > 0 {
		monosyllableRatio := float64(monosyllableCount) / float64(len(words))
		if monosyllableRatio > 0.7 { // 越南语单音节词比例高
			score += 1.0
		}
	}

	return math.Max(0, score)
}

// getTotalChars 计算字符总数
func getTotalChars(charFreq map[rune]int) int {
	total := 0
	for _, count := range charFreq {
		total += count
	}
	return total
}

// isSimplifiedChar 判断是否为简体中文特有字符
func isSimplifiedChar(r rune) bool {
	// 简体中文常用字符范围
	return (r >= 0x4E00 && r <= 0x9FFF) && !isTraditionalChar(r)
}

// isTraditionalChar 判断是否为繁体中文特有字符
func isTraditionalChar(r rune) bool {
	// 繁体中文扩展区域和特殊字符
	return (r >= 0x3400 && r <= 0x4DBF) || // CJK扩展A
		(r >= 0x20000 && r <= 0x2A6DF) || // CJK扩展B
		(r >= 0x2A700 && r <= 0x2B73F) || // CJK扩展C
		(r >= 0x2B740 && r <= 0x2B81F) || // CJK扩展D
		(r >= 0x2B820 && r <= 0x2CEAF) || // CJK扩展E
		(r >= 0xF900 && r <= 0xFAFF) // CJK兼容汉字
}

// getLanguageCode 获取语言代码
func getLanguageCode(lang string) int {
	langCodes := map[string]int{
		"Chinese (Simplified)":  1,
		"Chinese (Traditional)": 2,
		"Chinese":               3,
		"English":               4,
		"Japanese":              5,
		"Korean":                6,
		"Russian":               7,
		"Arabic":                8,
		"Thai":                  9,
		"Hindi":                 10,
		"Hebrew":                11,
		"Greek":                 12,
		"French":                13,
		"German":                14,
		"Spanish":               15,
		"Italian":               16,
		"Portuguese":            17,
		"Dutch":                 18,
		"Polish":                19,
		"Czech":                 20,
		"Vietnamese":            21,
		// new languages
		"unknown": 0,
	}

	if code, exists := langCodes[lang]; exists {
		return code
	}
	return 0
}
