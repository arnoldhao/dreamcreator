package subtitles

import (
	"CanMe/backend/types"
)

// 核心服务接口
type QualityAssessor interface {
	AssessSegmentQuality(segment *types.SubtitleSegment) *types.SubtitleSegment
	AssessSubtitleQuality(text string, duration float64, standard types.GuideLineStandard, isKidsContent bool) *types.SubtitleGuideline
}

type TextProcessor interface {
	RemoveEmptyLines(text string) string
	TrimWhitespace(text string) string
	NormalizeLineBreaks(text string) string
	FixEncoding(text string) string
	FixCommonTextErrors(text string) string
	RemovePunctuation(text string) string
}

type LanguageDetector interface {
	DetectLanguage(text string) string
	DetectLanguageInt(text string) (int, string)
}

type FormatConverter interface {
    FromItt(filePath string, file []byte) (types.SubtitleProject, error)
    FromSRT(filePath string, file []byte) (types.SubtitleProject, error)
    FromVTT(filePath string, file []byte) (types.SubtitleProject, error)
    FromASS(filePath string, file []byte) (types.SubtitleProject, error)
    ToSRT(project *types.SubtitleProject, langCode string) ([]byte, error)
    ToVTT(project *types.SubtitleProject, langCode string) ([]byte, error)
    ToFCPXML(project *types.SubtitleProject, langCode string) ([]byte, error)
    ToASS(project *types.SubtitleProject, langCode string) ([]byte, error)
    ToITT(project *types.SubtitleProject, langCode string) ([]byte, error)
}
