package service

import (
	"encoding/xml"
	"fmt"
	"html"
	"regexp"
	"strings"

	"dreamcreator/internal/application/library/dto"
)

const (
	subtitleSourceContentMetadataKey = "sourceContent"
	subtitleSourceFormatMetadataKey  = "sourceFormat"
)

var (
	ittHeadPattern = regexp.MustCompile(`(?is)<head\b[^>]*>.*?</head>`)
	ittBodyPattern = regexp.MustCompile(`(?is)<body\b([^>]*)>(.*)</body>`)
	ittDivPattern  = regexp.MustCompile(`(?is)<div\b([^>]*)>(.*)</div>`)
)

type parsedVTTCue struct {
	Identifier string
	Settings   string
	Start      string
	End        string
	Text       string
}

type parsedVTTDocument struct {
	Header string
	Blocks [][]string
	Cues   []parsedVTTCue
}

type assEventEntry struct {
	Kind     string
	Values   []string
	RawLine  string
	Dialogue bool
}

type parsedASSDocument struct {
	HeaderLines []string
	FooterLines []string
	EventFormat []string
	Events      []assEventEntry
}

func normalizeSubtitleFormat(value string) string {
	switch normalizeTranscodeFormat(value) {
	case "webvtt":
		return "vtt"
	case "dfxp", "ttml", "xml":
		return "itt"
	default:
		return normalizeTranscodeFormat(value)
	}
}

func cloneSubtitleMetadata(source map[string]any) map[string]any {
	if len(source) == 0 {
		return map[string]any{}
	}
	result := make(map[string]any, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}

func subtitleDocumentWithSource(content string, format string, cues []dto.SubtitleCue, metadata map[string]any) dto.SubtitleDocument {
	result := dto.SubtitleDocument{
		Format:   normalizeSubtitleFormat(format),
		Cues:     cues,
		Metadata: cloneSubtitleMetadata(metadata),
	}
	if result.Metadata == nil {
		result.Metadata = map[string]any{}
	}
	result.Metadata[subtitleSourceContentMetadataKey] = content
	result.Metadata[subtitleSourceFormatMetadataKey] = result.Format
	return result
}

func subtitleDocumentSourceContent(document dto.SubtitleDocument) string {
	if document.Metadata == nil {
		return ""
	}
	value, ok := document.Metadata[subtitleSourceContentMetadataKey]
	if !ok {
		return ""
	}
	content, _ := value.(string)
	return content
}

func parseSubtitleDocument(content string, format string) dto.SubtitleDocument {
	normalized := normalizeSubtitleFormat(format)
	switch normalized {
	case "vtt":
		return subtitleDocumentWithSource(content, normalized, parseVTTSubtitleCues(content), nil)
	case "ass", "ssa":
		return subtitleDocumentWithSource(content, normalized, parseASSSubtitleCues(content), nil)
	case "itt":
		return subtitleDocumentWithSource(content, normalized, parseITTSubtitleCues(content), nil)
	case "fcpxml":
		return subtitleDocumentWithSource(content, normalized, parseFCPXMLSubtitleCues(content), nil)
	default:
		return subtitleDocumentWithSource(content, normalized, parseSRTSubtitleCues(content), nil)
	}
}

func parseSRTSubtitleCues(content string) []dto.SubtitleCue {
	normalized := normalizeSubtitleNewlines(content)
	if strings.TrimSpace(normalized) == "" {
		return nil
	}
	blocks := splitSubtitleBlocks(normalized)
	cues := make([]dto.SubtitleCue, 0, len(blocks))
	for _, block := range blocks {
		lines := blockLines(block)
		if len(lines) == 0 {
			continue
		}
		timeLineIndex := -1
		if len(lines) > 1 && strings.Contains(lines[1], "-->") {
			timeLineIndex = 1
		} else if strings.Contains(lines[0], "-->") {
			timeLineIndex = 0
		}
		start := ""
		end := ""
		textStart := 0
		if timeLineIndex >= 0 {
			start, end = parseCueTimingLine(lines[timeLineIndex])
			textStart = timeLineIndex + 1
		}
		text := strings.Join(lines[textStart:], "\n")
		if strings.TrimSpace(text) == "" {
			text = strings.Join(lines, "\n")
		}
		cues = append(cues, dto.SubtitleCue{
			Index: len(cues) + 1,
			Start: start,
			End:   end,
			Text:  text,
		})
	}
	return cues
}

func parseVTTSubtitleCues(content string) []dto.SubtitleCue {
	document := parseVTTDocument(content)
	cues := make([]dto.SubtitleCue, 0, len(document.Cues))
	for index, cue := range document.Cues {
		cues = append(cues, dto.SubtitleCue{
			Index: index + 1,
			Start: cue.Start,
			End:   cue.End,
			Text:  cue.Text,
		})
	}
	return cues
}

func parseVTTDocument(content string) parsedVTTDocument {
	normalized := normalizeSubtitleNewlines(content)
	lines := strings.Split(normalized, "\n")
	document := parsedVTTDocument{Header: "WEBVTT"}
	if len(lines) == 0 {
		return document
	}
	if strings.HasPrefix(lines[0], "\uFEFF") {
		lines[0] = strings.TrimPrefix(lines[0], "\uFEFF")
	}
	startIndex := 0
	if strings.HasPrefix(strings.TrimSpace(lines[0]), "WEBVTT") {
		document.Header = strings.TrimSpace(lines[0])
		startIndex = 1
	}
	blocks := splitSubtitleBlocks(strings.Join(lines[startIndex:], "\n"))
	document.Blocks = make([][]string, 0, len(blocks))
	for _, block := range blocks {
		blockLines := blockLines(block)
		if len(blockLines) == 0 {
			continue
		}
		document.Blocks = append(document.Blocks, blockLines)
		if isVTTNonCueBlock(blockLines[0]) {
			continue
		}
		identifier := ""
		timingLine := blockLines[0]
		textStart := 1
		if len(blockLines) > 1 && !strings.Contains(blockLines[0], "-->") && strings.Contains(blockLines[1], "-->") {
			identifier = strings.TrimSpace(blockLines[0])
			timingLine = blockLines[1]
			textStart = 2
		}
		start, end, settings := parseVTTCueTimingLine(timingLine)
		if start == "" || end == "" {
			continue
		}
		document.Cues = append(document.Cues, parsedVTTCue{
			Identifier: identifier,
			Settings:   settings,
			Start:      start,
			End:        end,
			Text:       strings.Join(blockLines[textStart:], "\n"),
		})
	}
	return document
}

func parseASSSubtitleCues(content string) []dto.SubtitleCue {
	document := parseASSDocument(content)
	cues := make([]dto.SubtitleCue, 0, len(document.Events))
	for _, entry := range document.Events {
		if !entry.Dialogue {
			continue
		}
		start := assEventValue(document.EventFormat, entry.Values, "start")
		end := assEventValue(document.EventFormat, entry.Values, "end")
		text := unescapeASSText(assEventValue(document.EventFormat, entry.Values, "text"))
		cues = append(cues, dto.SubtitleCue{
			Index: len(cues) + 1,
			Start: start,
			End:   end,
			Text:  text,
		})
	}
	return cues
}

func parseASSDocument(content string) parsedASSDocument {
	lines := strings.Split(normalizeSubtitleNewlines(content), "\n")
	result := parsedASSDocument{}
	inEvents := false
	sawEventLine := false
	for _, rawLine := range lines {
		trimmed := strings.TrimSpace(rawLine)
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			inEvents = strings.EqualFold(trimmed, "[Events]")
			if !sawEventLine {
				result.HeaderLines = append(result.HeaderLines, rawLine)
			} else {
				result.FooterLines = append(result.FooterLines, rawLine)
			}
			continue
		}
		if inEvents {
			if key, value, ok := splitSubtitleExportStyleKeyValue(trimmed); ok && strings.EqualFold(key, "Format") {
				result.EventFormat = parseSubtitleExportStyleFormat(value)
				if !sawEventLine {
					result.HeaderLines = append(result.HeaderLines, rawLine)
				} else {
					result.FooterLines = append(result.FooterLines, rawLine)
				}
				continue
			}
			if kind, values, ok := parseASSEventLine(trimmed, len(result.EventFormat)); ok {
				sawEventLine = true
				result.Events = append(result.Events, assEventEntry{
					Kind:     kind,
					Values:   values,
					RawLine:  rawLine,
					Dialogue: strings.EqualFold(kind, "Dialogue"),
				})
				continue
			}
		}
		if !sawEventLine {
			result.HeaderLines = append(result.HeaderLines, rawLine)
		} else {
			result.FooterLines = append(result.FooterLines, rawLine)
		}
	}
	if len(result.EventFormat) == 0 {
		result.EventFormat = parseSubtitleExportStyleFormat(defaultSubtitleExportEventFormat)
	}
	return result
}

func parseITTSubtitleCues(content string) []dto.SubtitleCue {
	paragraphs := parseITTCueParagraphs(content)
	cues := make([]dto.SubtitleCue, 0, len(paragraphs))
	for _, paragraph := range paragraphs {
		if paragraph.Start == "" && paragraph.End == "" && strings.TrimSpace(paragraph.Text) == "" {
			continue
		}
		cues = append(cues, dto.SubtitleCue{
			Index: len(cues) + 1,
			Start: paragraph.Start,
			End:   paragraph.End,
			Text:  paragraph.Text,
		})
	}
	return cues
}

type ittParagraph struct {
	Start string
	End   string
	Text  string
}

func parseITTCueParagraphs(content string) []ittParagraph {
	decoder := xml.NewDecoder(strings.NewReader(content))
	result := make([]ittParagraph, 0, 32)
	for {
		token, err := decoder.Token()
		if err != nil {
			break
		}
		startElement, ok := token.(xml.StartElement)
		if !ok || startElement.Name.Local != "p" {
			continue
		}
		var paragraph struct {
			InnerXML string `xml:",innerxml"`
			Attrs    []xml.Attr
		}
		if err := decoder.DecodeElement(&paragraph, &startElement); err != nil {
			continue
		}
		result = append(result, ittParagraph{
			Start: xmlAttributeValue(startElement.Attr, "begin"),
			End:   xmlAttributeValue(startElement.Attr, "end"),
			Text:  ttmlInnerXMLToText(paragraph.InnerXML),
		})
	}
	return result
}

func parseFCPXMLSubtitleCues(content string) []dto.SubtitleCue {
	decoder := xml.NewDecoder(strings.NewReader(content))
	result := make([]dto.SubtitleCue, 0, 32)
	for {
		token, err := decoder.Token()
		if err != nil {
			break
		}
		startElement, ok := token.(xml.StartElement)
		if !ok || startElement.Name.Local != "title" {
			continue
		}
		var title fcpxmlTitle
		if err := decoder.DecodeElement(&title, &startElement); err != nil {
			continue
		}
		startMS, okStart := parseTimestampToMilliseconds(title.Offset)
		durationMS, okDuration := parseTimestampToMilliseconds(title.Duration)
		if !okStart {
			startMS = 0
		}
		if !okDuration || durationMS <= 0 {
			durationMS = 1000
		}
		result = append(result, dto.SubtitleCue{
			Index: len(result) + 1,
			Start: formatVTTTimestamp(startMS),
			End:   formatVTTTimestamp(startMS + durationMS),
			Text:  fcpxmlTitleText(title),
		})
	}
	return result
}

func renderSubtitleContentPreservingSource(
	document dto.SubtitleDocument,
	targetFormat string,
	config *dto.SubtitleExportConfig,
	styleDocumentContent string,
	originalContent string,
) (string, bool) {
	format := normalizeSubtitleFormat(targetFormat)
	if strings.TrimSpace(originalContent) == "" {
		return "", false
	}
	switch format {
	case "srt":
		return renderSRTFromSegments(normalizeSubtitleSegments(document)), true
	case "vtt":
		return renderVTTFromSource(document, originalContent), true
	case "ass":
		return renderASSFromSource(document, originalContent, false), true
	case "ssa":
		return renderASSFromSource(document, originalContent, true), true
	case "itt":
		if content, ok := renderITTFromSource(document, originalContent); ok {
			return content, true
		}
	case "fcpxml":
		if content, ok := renderFCPXMLFromSource(document, originalContent); ok {
			return content, true
		}
	}
	return "", false
}

func renderVTTFromSource(document dto.SubtitleDocument, source string) string {
	parsed := parseVTTDocument(source)
	var builder strings.Builder
	builder.WriteString(firstNonEmpty(strings.TrimSpace(parsed.Header), "WEBVTT"))
	builder.WriteString("\n\n")
	for _, block := range parsed.Blocks {
		if len(block) == 0 || !isVTTNonCueBlock(block[0]) {
			continue
		}
		builder.WriteString(strings.Join(block, "\n"))
		builder.WriteString("\n\n")
	}
	for index, cue := range document.Cues {
		if index > 0 {
			builder.WriteString("\n")
		}
		if index < len(parsed.Cues) && strings.TrimSpace(parsed.Cues[index].Identifier) != "" {
			builder.WriteString(parsed.Cues[index].Identifier)
			builder.WriteString("\n")
		}
		settings := ""
		if index < len(parsed.Cues) {
			settings = strings.TrimSpace(parsed.Cues[index].Settings)
		}
		builder.WriteString(formatVTTCueTiming(cue.Start, cue.End, settings))
		builder.WriteString("\n")
		builder.WriteString(normalizeSubtitleText(cue.Text))
		builder.WriteString("\n")
	}
	return strings.TrimRight(builder.String(), "\n") + "\n"
}

func renderASSFromSource(document dto.SubtitleDocument, source string, forceSSA bool) string {
	parsed := parseASSDocument(source)
	format := parsed.EventFormat
	if len(format) == 0 {
		format = parseSubtitleExportStyleFormat(defaultSubtitleExportEventFormat)
	}
	lines := append([]string{}, parsed.HeaderLines...)
	dialogueIndex := 0
	for _, entry := range parsed.Events {
		if !entry.Dialogue {
			lines = append(lines, entry.RawLine)
			continue
		}
		if dialogueIndex >= len(document.Cues) {
			dialogueIndex++
			continue
		}
		lines = append(lines, renderASSEventLine(format, entry, document.Cues[dialogueIndex], forceSSA))
		dialogueIndex++
	}
	for dialogueIndex < len(document.Cues) {
		lines = append(lines, renderASSEventLine(format, defaultASSDialogueEntry(forceSSA), document.Cues[dialogueIndex], forceSSA))
		dialogueIndex++
	}
	lines = append(lines, parsed.FooterLines...)
	return strings.TrimRight(strings.Join(lines, "\n"), "\n") + "\n"
}

func renderITTFromSource(document dto.SubtitleDocument, source string) (string, bool) {
	content := normalizeSubtitleNewlines(source)
	if !strings.Contains(strings.ToLower(content), "<tt") {
		return "", false
	}
	rootOpen := ""
	rootClose := "</tt>"
	if index := strings.Index(strings.ToLower(content), "<tt"); index >= 0 {
		closeIndex := strings.Index(content[index:], ">")
		if closeIndex >= 0 {
			rootOpen = content[index : index+closeIndex+1]
		}
	}
	if rootOpen == "" {
		return "", false
	}
	headBlock := strings.TrimSpace(ittHeadPattern.FindString(content))
	bodyMatch := ittBodyPattern.FindStringSubmatch(content)
	bodyAttrs := ""
	divAttrs := ""
	if len(bodyMatch) > 1 {
		bodyAttrs = bodyMatch[1]
		divMatch := ittDivPattern.FindStringSubmatch(bodyMatch[2])
		if len(divMatch) > 1 {
			divAttrs = divMatch[1]
		}
	}
	var builder strings.Builder
	builder.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	builder.WriteString("\n")
	builder.WriteString(rootOpen)
	builder.WriteString("\n")
	if headBlock != "" {
		builder.WriteString("  ")
		builder.WriteString(headBlock)
		builder.WriteString("\n")
	}
	builder.WriteString("  <body")
	builder.WriteString(bodyAttrs)
	builder.WriteString(">\n")
	builder.WriteString("    <div")
	builder.WriteString(divAttrs)
	builder.WriteString(">\n")
	for _, cue := range document.Cues {
		builder.WriteString(`      <p begin="`)
		builder.WriteString(firstNonEmpty(strings.TrimSpace(cue.Start), "00:00:00.000"))
		builder.WriteString(`" end="`)
		builder.WriteString(firstNonEmpty(strings.TrimSpace(cue.End), "00:00:01.000"))
		builder.WriteString(`">`)
		builder.WriteString(ttmlTextToInnerXML(cue.Text))
		builder.WriteString("</p>\n")
	}
	builder.WriteString("    </div>\n")
	builder.WriteString("  </body>\n")
	builder.WriteString(rootClose)
	builder.WriteString("\n")
	return builder.String(), true
}

func renderFCPXMLFromSource(document dto.SubtitleDocument, source string) (string, bool) {
	var parsed fcpxmlRoot
	if err := xml.Unmarshal([]byte(source), &parsed); err != nil {
		return "", false
	}
	if len(parsed.Library.Events) == 0 || len(parsed.Library.Events[0].Projects) == 0 {
		return "", false
	}
	sequence := &parsed.Library.Events[0].Projects[0].Sequence
	gap := &sequence.Spine.Gap
	existingTitles := gap.Titles
	nextTitles := make([]fcpxmlTitle, 0, len(document.Cues))
	for index, cue := range document.Cues {
		base := fcpxmlTitle{
			Name:     cue.Text,
			Lane:     1,
			Ref:      "r2",
			Duration: formatFCPXMLDuration(maxInt64(1, cueDurationMS(cue))),
		}
		if len(existingTitles) > 0 {
			base = existingTitles[minInt(index, len(existingTitles)-1)]
		}
		base.Offset = formatFCPXMLDuration(cueStartMS(cue))
		base.Duration = formatFCPXMLDuration(maxInt64(1, cueDurationMS(cue)))
		if strings.TrimSpace(base.Start) == "" {
			base.Start = "3600s"
		}
		base.Name = cue.Text
		if strings.TrimSpace(fcpxmlTitleText(base)) != strings.TrimSpace(cue.Text) {
			ref := "ts1"
			if base.Text != nil && len(base.Text.TextStyle) > 0 && strings.TrimSpace(base.Text.TextStyle[0].Ref) != "" {
				ref = base.Text.TextStyle[0].Ref
			}
			base.Text = &fcpxmlText{TextStyle: []fcpxmlTextStyle{{Ref: ref, Content: cue.Text}}}
		}
		nextTitles = append(nextTitles, base)
	}
	gap.Titles = nextTitles
	if len(document.Cues) > 0 {
		totalDuration := cueEndMS(document.Cues[len(document.Cues)-1])
		sequence.Duration = formatFCPXMLDuration(totalDuration)
		gap.Duration = formatFCPXMLDuration(totalDuration)
	}
	xmlData, err := xml.MarshalIndent(parsed, "", "  ")
	if err != nil {
		return "", false
	}
	return "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<!DOCTYPE fcpxml>\n\n" + string(xmlData) + "\n", true
}

func validateSubtitleDocument(document dto.SubtitleDocument) dto.SubtitleValidateResult {
	issues := make([]dto.SubtitleValidateIssue, 0)
	if len(document.Cues) == 0 {
		issues = append(issues, dto.SubtitleValidateIssue{
			Severity: "error",
			Code:     "empty_content",
			Message:  "subtitle content is empty",
		})
		return dto.SubtitleValidateResult{Valid: false, IssueCount: len(issues), Issues: issues}
	}
	var previousEnd int64
	for index, cue := range document.Cues {
		if strings.TrimSpace(cue.Text) == "" {
			issues = append(issues, dto.SubtitleValidateIssue{
				Severity: "warning",
				Code:     "empty_text",
				Message:  "subtitle cue text is empty",
				CueIndex: cue.Index,
			})
		}
		startMS, okStart := parseTimestampToMilliseconds(cue.Start)
		endMS, okEnd := parseTimestampToMilliseconds(cue.End)
		if !okStart || !okEnd {
			issues = append(issues, dto.SubtitleValidateIssue{
				Severity: "error",
				Code:     "invalid_timing",
				Message:  "subtitle cue timing is invalid",
				CueIndex: cue.Index,
			})
			continue
		}
		if endMS <= startMS {
			issues = append(issues, dto.SubtitleValidateIssue{
				Severity: "error",
				Code:     "non_positive_duration",
				Message:  "subtitle cue end time must be after start time",
				CueIndex: cue.Index,
			})
		}
		if index > 0 && startMS < previousEnd {
			issues = append(issues, dto.SubtitleValidateIssue{
				Severity: "warning",
				Code:     "overlap",
				Message:  "subtitle cue overlaps with the previous cue",
				CueIndex: cue.Index,
			})
		}
		previousEnd = endMS
	}
	return dto.SubtitleValidateResult{
		Valid:      len(issues) == 0,
		IssueCount: len(issues),
		Issues:     issues,
	}
}

func normalizeSubtitleDocumentText(document dto.SubtitleDocument) (dto.SubtitleDocument, int) {
	result := document
	result.Metadata = cloneSubtitleMetadata(document.Metadata)
	if len(document.Cues) == 0 {
		return result, 0
	}
	result.Cues = make([]dto.SubtitleCue, 0, len(document.Cues))
	changes := 0
	for _, cue := range document.Cues {
		normalizedText := normalizeCueTextWhitespace(cue.Text)
		if normalizedText != cue.Text {
			changes++
		}
		nextCue := cue
		nextCue.Text = normalizedText
		result.Cues = append(result.Cues, nextCue)
	}
	return result, changes
}

func normalizeCueTextWhitespace(value string) string {
	lines := strings.Split(normalizeSubtitleNewlines(value), "\n")
	for index, line := range lines {
		lines[index] = strings.Join(strings.Fields(line), " ")
	}
	return strings.TrimRight(strings.Join(lines, "\n"), "\n")
}

func parseCueTimingLine(line string) (string, string) {
	parts := strings.SplitN(line, "-->", 2)
	if len(parts) != 2 {
		return "", ""
	}
	start := strings.TrimSpace(parts[0])
	end := strings.TrimSpace(parts[1])
	if fields := strings.Fields(end); len(fields) > 0 {
		end = fields[0]
	}
	return start, end
}

func parseVTTCueTimingLine(line string) (string, string, string) {
	parts := strings.SplitN(line, "-->", 2)
	if len(parts) != 2 {
		return "", "", ""
	}
	start := strings.TrimSpace(parts[0])
	right := strings.Fields(strings.TrimSpace(parts[1]))
	if len(right) == 0 {
		return "", "", ""
	}
	end := right[0]
	settings := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(parts[1]), end))
	return start, end, settings
}

func formatVTTCueTiming(start string, end string, settings string) string {
	line := fmt.Sprintf("%s --> %s", firstNonEmpty(strings.TrimSpace(start), "00:00:00.000"), firstNonEmpty(strings.TrimSpace(end), "00:00:01.000"))
	if strings.TrimSpace(settings) != "" {
		line += " " + strings.TrimSpace(settings)
	}
	return line
}

func splitSubtitleBlocks(content string) []string {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return nil
	}
	return regexp.MustCompile(`\n{2,}`).Split(trimmed, -1)
}

func blockLines(block string) []string {
	lines := strings.Split(strings.TrimSpace(block), "\n")
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		result = append(result, line)
	}
	return result
}

func isVTTNonCueBlock(firstLine string) bool {
	trimmed := strings.TrimSpace(firstLine)
	switch {
	case strings.HasPrefix(trimmed, "NOTE"):
		return true
	case strings.EqualFold(trimmed, "STYLE"):
		return true
	case strings.EqualFold(trimmed, "REGION"):
		return true
	default:
		return false
	}
}

func parseASSEventLine(line string, expected int) (string, []string, bool) {
	colonIndex := strings.Index(line, ":")
	if colonIndex <= 0 {
		return "", nil, false
	}
	kind := strings.TrimSpace(line[:colonIndex])
	if !strings.EqualFold(kind, "Dialogue") && !strings.EqualFold(kind, "Comment") {
		return "", nil, false
	}
	fields := splitSubtitleExportStyleFields(line[colonIndex+1:], expected)
	return kind, fields, true
}

func assEventValue(format []string, values []string, name string) string {
	return findSubtitleExportStyleField(format, values, name)
}

func renderASSEventLine(format []string, entry assEventEntry, cue dto.SubtitleCue, forceSSA bool) string {
	values := append([]string{}, entry.Values...)
	if len(values) < len(format) {
		padding := make([]string, len(format)-len(values))
		values = append(values, padding...)
	}
	setASSField(format, values, "start", firstNonEmpty(strings.TrimSpace(cue.Start), assEventValue(format, entry.Values, "start")))
	setASSField(format, values, "end", firstNonEmpty(strings.TrimSpace(cue.End), assEventValue(format, entry.Values, "end")))
	text := normalizeSubtitleText(cue.Text)
	if strings.TrimSpace(text) == "" {
		text = normalizeSubtitleText(unescapeASSText(assEventValue(format, entry.Values, "text")))
	}
	setASSField(format, values, "text", escapeASSText(text))
	kind := entry.Kind
	if strings.TrimSpace(kind) == "" {
		kind = "Dialogue"
	}
	return fmt.Sprintf("%s: %s", kind, strings.Join(values, ","))
}

func defaultASSDialogueEntry(forceSSA bool) assEventEntry {
	values := []string{"0", "0:00:00.00", "0:00:01.00", "Default", "", "0", "0", "0", "", ""}
	if forceSSA {
		values[0] = "Marked=0"
	}
	return assEventEntry{
		Kind:     "Dialogue",
		Values:   values,
		Dialogue: true,
	}
}

func setASSField(format []string, values []string, name string, value string) {
	normalizedName := strings.ToLower(strings.TrimSpace(name))
	for index, field := range format {
		if field != normalizedName || index >= len(values) {
			continue
		}
		values[index] = value
		return
	}
}

func unescapeASSText(text string) string {
	replaced := strings.ReplaceAll(strings.ReplaceAll(text, `\N`, "\n"), `\n`, "\n")
	replaced = strings.ReplaceAll(replaced, `\h`, " ")
	return replaced
}

func xmlAttributeValue(attrs []xml.Attr, local string) string {
	for _, attr := range attrs {
		if attr.Name.Local == local {
			return strings.TrimSpace(attr.Value)
		}
	}
	return ""
}

func ttmlInnerXMLToText(inner string) string {
	if strings.TrimSpace(inner) == "" {
		return ""
	}
	decoder := xml.NewDecoder(strings.NewReader("<root>" + inner + "</root>"))
	var builder strings.Builder
	for {
		token, err := decoder.Token()
		if err != nil {
			break
		}
		switch typed := token.(type) {
		case xml.CharData:
			builder.WriteString(string(typed))
		case xml.StartElement:
			if typed.Name.Local == "br" {
				builder.WriteString("\n")
			}
		}
	}
	return builder.String()
}

func ttmlTextToInnerXML(text string) string {
	lines := strings.Split(normalizeSubtitleText(text), "\n")
	escaped := make([]string, 0, len(lines))
	for _, line := range lines {
		escaped = append(escaped, html.EscapeString(line))
	}
	return strings.Join(escaped, "<br/>")
}

func fcpxmlTitleText(title fcpxmlTitle) string {
	if title.Text == nil || len(title.Text.TextStyle) == 0 {
		return strings.TrimSpace(title.Name)
	}
	var builder strings.Builder
	for _, style := range title.Text.TextStyle {
		builder.WriteString(style.Content)
	}
	return builder.String()
}

func normalizeSubtitleNewlines(content string) string {
	return strings.ReplaceAll(strings.ReplaceAll(content, "\r\n", "\n"), "\r", "\n")
}

func normalizeSubtitleText(text string) string {
	return strings.TrimRight(normalizeSubtitleNewlines(text), "\n")
}

func cueStartMS(cue dto.SubtitleCue) int64 {
	value, ok := parseTimestampToMilliseconds(cue.Start)
	if !ok {
		return 0
	}
	return value
}

func cueEndMS(cue dto.SubtitleCue) int64 {
	value, ok := parseTimestampToMilliseconds(cue.End)
	if !ok {
		return cueStartMS(cue) + 1000
	}
	return value
}

func cueDurationMS(cue dto.SubtitleCue) int64 {
	return maxInt64(1, cueEndMS(cue)-cueStartMS(cue))
}
