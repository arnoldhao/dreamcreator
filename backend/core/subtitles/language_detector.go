package subtitles

import (
	"github.com/pemistahl/lingua-go"
)

// LanguageDetectorImpl 语言检测器实现
type LanguageDetectorImpl struct {
	detector lingua.LanguageDetector
}

// NewLanguageDetector 创建语言检测器 - 使用相对距离
func NewLanguageDetector() LanguageDetector {
	detector := lingua.NewLanguageDetectorBuilder().
		FromLanguages(
			lingua.English,
			lingua.Chinese,
			lingua.Japanese,
			lingua.Korean,
			lingua.Spanish,
			lingua.French,
			// more languages...
		).
		WithMinimumRelativeDistance(0.9). // 设置最小相对距离
		Build()

	return &LanguageDetectorImpl{
		detector: detector,
	}
}

func (ld *LanguageDetectorImpl) DetectLanguage(text string) string {
	if text == "" {
		return "unknown"
	}

	detectedLang, exists := ld.detector.DetectLanguageOf(text)
	if !exists {
		return "unknown"
	}

	return detectedLang.String()
}

func (ld *LanguageDetectorImpl) DetectLanguageInt(text string) (lingua.Language, string) {
	if text == "" {
		return 0, "unknown"
	}

	detectedLang, exists := ld.detector.DetectLanguageOf(text)
	if !exists {
		return 0, "unknown"
	}

	return detectedLang, detectedLang.String()
}
