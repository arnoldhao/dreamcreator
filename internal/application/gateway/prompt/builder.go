package prompt

import (
	"strings"
	"time"
)

type Section struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Content  string `json:"content"`
	Tokens   int    `json:"tokens"`
	Truncated bool  `json:"truncated"`
}

type Document struct {
	Sections []Section `json:"sections"`
	Content  string    `json:"content"`
}

type SectionReport struct {
	ID        string `json:"id"`
	Tokens    int    `json:"tokens"`
	Truncated bool   `json:"truncated"`
	Reason    string `json:"reason,omitempty"`
}

type BuildReport struct {
	GeneratedAt time.Time       `json:"generatedAt"`
	Sections    []SectionReport `json:"sections"`
	Truncated   bool            `json:"truncated"`
}

type Builder struct {
	maxTokens int
}

func NewBuilder(maxTokens int) *Builder {
	return &Builder{maxTokens: maxTokens}
}

type SectionComposer struct {
	maxTokens int
}

func NewSectionComposer(maxTokens int) *SectionComposer {
	return &SectionComposer{maxTokens: maxTokens}
}

func (composer *SectionComposer) Compose(sections []Section) (Document, BuildReport) {
	builder := NewBuilder(composer.maxTokens)
	return builder.Build(sections)
}

type Truncator struct {
	maxTokens int
}

func NewTruncator(maxTokens int) *Truncator {
	return &Truncator{maxTokens: maxTokens}
}

func (truncator *Truncator) Apply(sections []Section) ([]Section, []SectionReport, bool) {
	builder := NewBuilder(truncator.maxTokens)
	document, report := builder.Build(sections)
	return document.Sections, report.Sections, report.Truncated
}

func (builder *Builder) Build(sections []Section) (Document, BuildReport) {
	maxTokens := builder.maxTokens
	if maxTokens <= 0 {
		maxTokens = 8192
	}
	contentParts := make([]string, 0, len(sections))
	reports := make([]SectionReport, 0, len(sections))
	used := 0
	truncated := false
	for _, section := range sections {
		trimmed := strings.TrimSpace(section.Content)
		tokens := section.Tokens
		if tokens <= 0 {
			tokens = len(strings.Fields(trimmed))
		}
		reason := ""
		if used+tokens > maxTokens {
			truncated = true
			section.Truncated = true
			reason = "token_budget_exceeded"
		}
		used += tokens
		contentParts = append(contentParts, trimmed)
		reports = append(reports, SectionReport{ID: section.ID, Tokens: tokens, Truncated: section.Truncated, Reason: reason})
	}
	return Document{Sections: sections, Content: strings.Join(contentParts, "\n")}, BuildReport{
		GeneratedAt: time.Now(),
		Sections:    reports,
		Truncated:   truncated,
	}
}
