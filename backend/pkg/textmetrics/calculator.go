package textmetrics

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// Calculator 文本度量计算器
type Calculator struct{}

// NewCalculator 创建新的文本度量计算器
func NewCalculator() *Calculator {
	return &Calculator{}
}

// CountCharacters 计算字符数（不包括空格）- Netflix标准
func (c *Calculator) CountCharacters(text string) int {
	count := 0
	for _, r := range text {
		// Exclude spaces, tabs and line breaks for readability metrics
		if r != ' ' && r != '\t' && r != '\n' && r != '\r' {
			count++
		}
	}
	return count
}

// CountCharactersWithSpaces 计算字符数（包括空格）
func (c *Calculator) CountCharactersWithSpaces(text string) int {
	return len([]rune(text))
}

// CountCharactersBytes counts UTF-8 bytes (excluding spaces, tabs and line breaks).
// Useful for ideographic scripts where byte length is used as visual density proxy.
func (c *Calculator) CountCharactersBytes(text string) int {
	count := 0
	for _, r := range text {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			continue
		}
		count += utf8.RuneLen(r)
	}
	return count
}

// CountWords 计算单词数（用于WPM）
func (c *Calculator) CountWords(text string) int {
	// 检查是否包含表意文字
	hasIdeographic := false
	hasAlphabetic := false

	for _, r := range text {
		if unicode.IsLetter(r) {
			if unicode.In(r, unicode.Han, unicode.Hiragana, unicode.Katakana, unicode.Hangul) {
				hasIdeographic = true
			} else {
				hasAlphabetic = true
			}
		}
	}

	// 如果是混合文本，需要分别计算
	if hasIdeographic && hasAlphabetic {
		return c.countMixedLanguageWords(text)
	}

	// 纯表意文字
	if hasIdeographic {
		return c.CountCharacters(text)
	}

	// 纯拼音文字
	return len(strings.Fields(text))
}

// countMixedLanguageWords 计算混合语言文本的词数
func (c *Calculator) countMixedLanguageWords(text string) int {
	wordCount := 0
	inWord := false

	for _, r := range text {
		if unicode.In(r, unicode.Han, unicode.Hiragana, unicode.Katakana, unicode.Hangul) {
			// 表意文字，每个字符算一个词
			wordCount++
			inWord = false
		} else if unicode.IsLetter(r) {
			// 拼音文字，开始一个新词
			if !inWord {
				wordCount++
				inWord = true
			}
		} else {
			// 非字母字符，结束当前词
			inWord = false
		}
	}

	return wordCount
}

// CountMaxLineLength 计算最大行长度（字符数，不含空格）
func (c *Calculator) CountMaxLineLength(text string) int {
	lines := strings.Split(text, "\n")
	maxLength := 0
	for _, line := range lines {
		length := c.CountCharacters(line)
		if length > maxLength {
			maxLength = length
		}
	}
	return maxLength
}

// CountMaxLineLengthBytes returns maximum UTF-8 byte length per line (excluding spaces/tabs/line breaks).
func (c *Calculator) CountMaxLineLengthBytes(text string) int {
	lines := strings.Split(text, "\n")
	maxLength := 0
	for _, line := range lines {
		length := c.CountCharactersBytes(line)
		if length > maxLength {
			maxLength = length
		}
	}
	return maxLength
}

// isPrimarilyIdeographic 判断文本是否主要为表意文字
func (c *Calculator) isPrimarilyIdeographic(text string) bool {
	ideographicCount := 0
	totalLetters := 0

	for _, r := range text {
		if unicode.IsLetter(r) {
			totalLetters++
			if unicode.In(r, unicode.Han, unicode.Hiragana, unicode.Katakana, unicode.Hangul) {
				ideographicCount++
			}
		}
	}

	if totalLetters == 0 {
		return false
	}

	// 如果表意文字占比超过50%，认为是表意文字文本
	return float64(ideographicCount)/float64(totalLetters) > 0.5
}

// IsPrimarilyIdeographic exported helper for external callers
func (c *Calculator) IsPrimarilyIdeographic(text string) bool {
	return c.isPrimarilyIdeographic(text)
}

// GetReadingSpeed 获取推荐的阅读速度限制
func (c *Calculator) GetReadingSpeed(text string, isKidsContent bool) (maxCPS int, maxWPM int) {
	if isKidsContent {
		return 13, 130 // 儿童内容更慢的阅读速度
	}

	if c.isPrimarilyIdeographic(text) {
		// 表意文字阅读速度稍慢
		return 15, 140
	}

	// 拼音文字标准速度
	return 17, 160
}
