package subtitles

import (
	"CanMe/backend/pkg/textmetrics"
	"CanMe/backend/types"
)

// QualityAssessor 字幕质量评估器
type QualityAssessorImpl struct {
	textCalculator *textmetrics.Calculator
}

// NewQualityAssessor 创建新的质量评估器
func NewQualityAssessor() QualityAssessor {
	return &QualityAssessorImpl{
		textCalculator: textmetrics.NewCalculator(),
	}
}

func (qa *QualityAssessorImpl) AssessSegmentQuality(segment *types.SubtitleSegment) *types.SubtitleSegment {
	if len(segment.GuidelineStandard) < 1 || len(segment.Languages) < 1 {
		return segment
	}

	for langCode, seg := range segment.Languages {
		// 获取该语言的标准，如果没有则跳过该语言
		standard, exists := segment.GuidelineStandard[langCode]
		if !exists || standard == "" {
			continue // 跳过该语言，继续处理其他语言
		}

		// 计算持续时间
		duration := segment.EndTime.Time - segment.StartTime.Time

		// 创建新的LanguageContent副本
		updatedSegment := seg
		updatedSegment.SubtitleGuideline = qa.AssessSubtitleQuality(
			seg.Text,
			duration.Seconds(),
			standard,
			segment.IsKidsContent,
		)

		// 更新map中的值
		segment.Languages[langCode] = updatedSegment
	}

	return segment
}

// AssessSubtitleQuality 评估字幕质量
func (qa *QualityAssessorImpl) AssessSubtitleQuality(
	text string,
	duration float64,
	standard types.GuideLineStandard,
	isKidsContent bool,
) *types.SubtitleGuideline {
	// 获取推荐的阅读速度
	maxCPS, maxWPM := qa.textCalculator.GetReadingSpeed(text, isKidsContent)

	// 计算实际指标
	charCount := qa.textCalculator.CountCharacters(text)
	wordCount := qa.textCalculator.CountWords(text)
	maxLineLength := qa.textCalculator.CountMaxLineLength(text)

	// 计算速度
	currentCPS := int(float64(charCount) / duration)
	currentWPM := int(float64(wordCount) / duration * 60)

	return &types.SubtitleGuideline{
		CPS: &types.Guideline{
			Current: currentCPS,
			Level:   qa.evaluateLevel(currentCPS, maxCPS),
		},
		WPM: &types.Guideline{
			Current: currentWPM,
			Level:   qa.evaluateLevel(currentWPM, maxWPM),
		},
		CPL: &types.Guideline{
			Current: maxLineLength,
			Level:   qa.evaluateLevel(maxLineLength, 42), // Netflix标准
		},
	}
}

// evaluateLevel 评估级别（0=正常，1=警告，2=超标）
func (qa *QualityAssessorImpl) evaluateLevel(current, threshold int) int {
	if current <= threshold {
		return 0 // 正常
	} else if current <= int(float64(threshold)*1.2) {
		return 1 // 轻微超标
	}
	return 2 // 严重超标
}
