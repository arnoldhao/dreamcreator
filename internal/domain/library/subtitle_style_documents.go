package library

import (
	"fmt"
	"strings"
)

type SubtitleStyleDocument struct {
	ID          string
	Name        string
	Description string
	Source      string
	SourceRef   string
	Version     string
	Enabled     bool
	Format      string
	Content     string
}

func defaultBuiltinSubtitleStyleDocuments() []SubtitleStyleDocument {
	return nil
}

func newBuiltinSubtitleStyleDocument(id string, name string, description string, sourceRef string, content string) SubtitleStyleDocument {
	return SubtitleStyleDocument{
		ID:          id,
		Name:        name,
		Description: description,
		Source:      "builtin",
		SourceRef:   sourceRef,
		Version:     "1",
		Enabled:     true,
		Format:      "ass",
		Content:     normalizeSubtitleStyleDocumentContent(content),
	}
}

func normalizeSubtitleStyleDocuments(values []SubtitleStyleDocument, fallback []SubtitleStyleDocument) []SubtitleStyleDocument {
	builtinByID := make(map[string]SubtitleStyleDocument, len(fallback))
	result := make([]SubtitleStyleDocument, 0, len(fallback)+len(values))
	for _, document := range fallback {
		normalized := normalizeSubtitleStyleDocument(document, document)
		builtinByID[normalized.ID] = normalized
		result = append(result, normalized)
	}
	seen := make(map[string]struct{}, len(result))
	for _, document := range result {
		seen[document.ID] = struct{}{}
	}
	for index, value := range values {
		id := normalizeAssetID(value.ID, value.Name, fmt.Sprintf("subtitle-ass-%d", index+1))
		if id == "" {
			continue
		}
		if builtin, exists := builtinByID[id]; exists {
			if _, already := seen[builtin.ID]; !already {
				result = append(result, builtin)
				seen[builtin.ID] = struct{}{}
			}
			continue
		}
		normalized := normalizeSubtitleStyleDocument(value, SubtitleStyleDocument{})
		if normalized.ID == "" {
			continue
		}
		if _, exists := seen[normalized.ID]; exists {
			continue
		}
		seen[normalized.ID] = struct{}{}
		result = append(result, normalized)
	}
	return result
}

func normalizeSubtitleStyleDocument(value SubtitleStyleDocument, fallback SubtitleStyleDocument) SubtitleStyleDocument {
	id := normalizeAssetID(value.ID, value.Name, fallback.ID)
	if id == "" {
		return SubtitleStyleDocument{}
	}
	name := strings.TrimSpace(value.Name)
	if name == "" {
		name = firstNonEmpty(strings.TrimSpace(fallback.Name), id)
	}
	source := normalizeSubtitleStyleDocumentSource(firstNonEmpty(value.Source, fallback.Source))
	if source == "" {
		source = "library"
	}
	format := normalizeSubtitleStyleDocumentFormat(firstNonEmpty(value.Format, fallback.Format))
	if format == "" {
		format = "ass"
	}
	content := normalizeSubtitleStyleDocumentContent(firstNonEmpty(value.Content, fallback.Content))
	if content == "" {
		content = normalizeSubtitleStyleDocumentContent(strings.Join([]string{
			"[Script Info]",
			"ScriptType: v4.00+",
			"",
			"[V4+ Styles]",
			"Format: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour, Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, Encoding",
			"",
			"[Events]",
			"Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text",
			"",
		}, "\n"))
	}
	enabled := value.Enabled
	if source == "builtin" {
		enabled = true
	}
	return SubtitleStyleDocument{
		ID:          id,
		Name:        name,
		Description: strings.TrimSpace(firstNonEmpty(value.Description, fallback.Description)),
		Source:      source,
		SourceRef:   strings.TrimSpace(firstNonEmpty(value.SourceRef, fallback.SourceRef)),
		Version:     strings.TrimSpace(firstNonEmpty(value.Version, fallback.Version, "1")),
		Enabled:     enabled,
		Format:      format,
		Content:     content,
	}
}

func normalizeSubtitleStyleDocumentFormat(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "ass", "ssa":
		return "ass"
	default:
		return ""
	}
}

func normalizeSubtitleStyleDocumentContent(value string) string {
	normalized := strings.ReplaceAll(value, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	normalized = strings.TrimSpace(strings.TrimPrefix(normalized, "\uFEFF"))
	if normalized == "" {
		return ""
	}
	return normalized + "\n"
}
