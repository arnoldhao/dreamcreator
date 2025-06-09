package subtitles

import (
	"strings"
	"unicode"
)

// TextProcessorImpl implements the TextProcessor interface
type TextProcessorImpl struct{}

// NewTextProcessor 创建新的文本处理器
func NewTextProcessor() TextProcessor {
	return &TextProcessorImpl{}
}

// RemoveEmptyLines 去除空白行
func (tp *TextProcessorImpl) RemoveEmptyLines(text string) string {
	lines := strings.Split(text, "\n")
	var nonEmptyLines []string

	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			nonEmptyLines = append(nonEmptyLines, line)
		}
	}

	return strings.Join(nonEmptyLines, "\n")
}

// TrimWhitespace 修剪每行的空白字符
func (tp *TextProcessorImpl) TrimWhitespace(text string) string {
	lines := strings.Split(text, "\n")
	var trimmedLines []string

	for _, line := range lines {
		trimmedLines = append(trimmedLines, strings.TrimSpace(line))
	}

	return strings.Join(trimmedLines, "\n")
}

func (tp *TextProcessorImpl) FixEncoding(text string) string {
	// 替换常见的编码错误

	return text
}

// NormalizeLineBreaks 标准化换行符
func (tp *TextProcessorImpl) NormalizeLineBreaks(text string) string {
	// 统一换行符为 \n
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	// 去除多余的连续换行符（保留最多两个连续换行符）
	for strings.Contains(text, "\n\n\n") {
		text = strings.ReplaceAll(text, "\n\n\n", "\n\n")
	}

	return text
}

// FixCommonTextErrors 修正常见的文本错误
func (tp *TextProcessorImpl) FixCommonTextErrors(text string) string {
	// 修正常见的字符问题
	replacements := map[string]string{
		"’": "'",
		"‘": "'",
		"“": "\"",
		"”": "\"",
		"—": "-",
		"–": "-",
		"…": "...",
		" ": " ", // 替换为普通空格
	}

	for old, new := range replacements {
		text = strings.ReplaceAll(text, old, new)
	}
	// 修复连续的标点符号
	text = strings.ReplaceAll(text, "..", ".")
	text = strings.ReplaceAll(text, "!!", "!")
	text = strings.ReplaceAll(text, "??", "?")
	// 修复连续的空格
	text = strings.ReplaceAll(text, "  ", " ")
	return text
}

// RemovePunctuation 去除标点符号
func (tp *TextProcessorImpl) RemovePunctuation(text string) string {
	var result strings.Builder
	for _, char := range text {
		if !unicode.IsPunct(char) {
			result.WriteRune(char)
		}
	}
	return result.String()
}
