package types

type TextProcessingOptions struct {
	RemoveEmptyLines    bool              `json:"remove_empty_lines"`
	TrimWhitespace      bool              `json:"trim_whitespace"`
	NormalizeLineBreaks bool              `json:"normalize_line_breaks"`
	FixEncoding         bool              `json:"fix_encoding"`
	FixCommonErrors     bool              `json:"fix_common_errors"`
	ValidateGuidelines  bool              `json:"validate_guidelines"`
	IsKidsContent       bool              `json:"is_kids_content"`
	GuidelineStandard   GuideLineStandard `json:"guideline_standard"`
}
