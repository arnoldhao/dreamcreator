package telegram

import (
	"bytes"
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	ghtml "github.com/yuin/goldmark/renderer/html"
)

type FormattedChunk struct {
	HTML string
	Text string
}

var telegramMarkdown = goldmark.New(
	goldmark.WithExtensions(
		extension.GFM,
	),
	goldmark.WithRendererOptions(
		ghtml.WithXHTML(),
	),
)

var (
	brTagPattern             = regexp.MustCompile(`(?i)<br\s*/?>\s*`)
	headingOpenPattern       = regexp.MustCompile(`(?i)<h[1-6][^>]*>`)
	headingClosePattern      = regexp.MustCompile(`(?i)</h[1-6]>`)
	strongOpenPattern        = regexp.MustCompile(`(?i)<strong[^>]*>`)
	strongClosePattern       = regexp.MustCompile(`(?i)</strong>`)
	emOpenPattern            = regexp.MustCompile(`(?i)<em[^>]*>`)
	emClosePattern           = regexp.MustCompile(`(?i)</em>`)
	delOpenPattern           = regexp.MustCompile(`(?i)<del[^>]*>`)
	delClosePattern          = regexp.MustCompile(`(?i)</del>`)
	strikeOpenPattern        = regexp.MustCompile(`(?i)<strike[^>]*>`)
	strikeClosePattern       = regexp.MustCompile(`(?i)</strike>`)
	preOpenPattern           = regexp.MustCompile(`(?i)<pre[^>]*>`)
	insOpenPattern           = regexp.MustCompile(`(?i)<ins[^>]*>`)
	insClosePattern          = regexp.MustCompile(`(?i)</ins>`)
	pOpenPattern             = regexp.MustCompile(`(?i)<p\b[^>]*>`)
	pClosePattern            = regexp.MustCompile(`(?i)</p>`)
	hrPattern                = regexp.MustCompile(`(?i)<hr\s*/?>`)
	imgPattern               = regexp.MustCompile(`(?i)<img\b[^>]*>`)
	anchorHrefPattern        = regexp.MustCompile(`(?i)href\s*=\s*"([^"]+)"`)
	tagPattern               = regexp.MustCompile(`(?i)</?\s*([a-z0-9-]+)\b[^>]*?>`)
	htmlTagTokenPattern      = regexp.MustCompile(`(?s)<[^>]+>`)
	autoLinkedAnchorPattern  = regexp.MustCompile(`(?i)<a\s+href="https?:\\/\\/([^"]+)"[^>]*>([^<]+)</a>`)
	htmlTagPattern           = regexp.MustCompile(`(?i)(</?)([a-z][a-z0-9-]*)\b[^>]*?>`)
	codeClassPattern         = regexp.MustCompile(`(?i)class\s*=\s*"([^"]+)"`)
	classAttrPattern         = regexp.MustCompile(`(?i)class\s*=\s*("([^"]*)"|'([^']*)')`)
	tgEmojiIDPattern         = regexp.MustCompile(`(?i)emoji-id\s*=\s*"([0-9]+)"`)
	multiNewlinePattern      = regexp.MustCompile(`\n{3,}`)
	rawHTMLOmittedPattern    = regexp.MustCompile(`(?i)<!--\s*raw html omitted\s*-->`)
	spoilerPattern           = regexp.MustCompile(`\|\|(.+?)\|\|`)
	orderedListStartPattern  = regexp.MustCompile(`(?i)\bstart\s*=\s*"?(\d+)"?`)
	blockquoteContentPattern = regexp.MustCompile(`(?is)<blockquote(\s+expandable)?>(.*?)</blockquote>`)
	blockquoteExpandableRE   = regexp.MustCompile(`(?i)\bexpandable\b`)
	checkboxInputPattern     = regexp.MustCompile(`(?i)<input\b[^>]*\btype\s*=\s*"?checkbox"?[^>]*>`)
	checkedAttrPattern       = regexp.MustCompile(`(?i)\bchecked\b`)
	listItemLinePattern      = regexp.MustCompile(`^\s*(?:\d+\.\s|[•◦]\s)`)
)

var allowedTelegramTags = map[string]struct{}{
	"b":          {},
	"i":          {},
	"u":          {},
	"s":          {},
	"code":       {},
	"pre":        {},
	"a":          {},
	"blockquote": {},
	"tg-spoiler": {},
	"tg-emoji":   {},
}

var (
	fileExtensionsWithTLD = []string{"md", "go", "py", "pl", "sh", "am", "at", "be", "cc"}
	fileRefPattern        *regexp.Regexp
	orphanedTLDPattern    *regexp.Regexp
)

func init() {
	extPattern := strings.Join(escapeRegexList(fileExtensionsWithTLD), "|")
	fileRefPattern = regexp.MustCompile(`(^|[^a-zA-Z0-9_\\/-])([a-zA-Z0-9_.\\./-]+\.(?:` + extPattern + `))($|[^a-zA-Z0-9_\\/-])`)
	orphanedTLDPattern = regexp.MustCompile(`([^a-zA-Z0-9]|^)([A-Za-z]\.(?:` + extPattern + `))([^a-zA-Z0-9/]|$)`)
}

func RenderTelegramHTML(markdown string) string {
	trimmed := strings.TrimSpace(markdown)
	if trimmed == "" {
		return ""
	}
	var buf bytes.Buffer
	if err := telegramMarkdown.Convert([]byte(trimmed), &buf); err != nil {
		return html.EscapeString(trimmed)
	}
	return normalizeTelegramHTML(buf.String())
}

func MarkdownToTelegramChunks(markdown string, limit int) []FormattedChunk {
	htmlText := RenderTelegramHTML(markdown)
	return SplitTelegramHTML(htmlText, limit)
}

func SplitTelegramHTML(htmlText string, limit int) []FormattedChunk {
	trimmed := strings.TrimSpace(htmlText)
	if trimmed == "" {
		return nil
	}
	if limit <= 0 {
		limit = 3800
	}
	if utf8.RuneCountInString(trimmed) <= limit {
		return []FormattedChunk{{HTML: trimmed, Text: PlainTextFromHTML(trimmed)}}
	}
	tokens := tokenizeHTML(trimmed)
	chunks := make([]FormattedChunk, 0, (len(trimmed)/limit)+1)
	var buf strings.Builder
	runeCount := 0
	openTags := make([]openTag, 0)

	flush := func() {
		if buf.Len() == 0 {
			return
		}
		closing := buildClosingTags(openTags)
		buf.WriteString(closing)
		htmlChunk := buf.String()
		chunks = append(chunks, FormattedChunk{HTML: htmlChunk, Text: PlainTextFromHTML(htmlChunk)})
		buf.Reset()
		runeCount = 0
		for _, tag := range openTags {
			buf.WriteString(tag.open)
			runeCount += utf8.RuneCountInString(tag.open)
		}
	}

	appendTag := func(tag string) {
		buf.WriteString(tag)
		runeCount += utf8.RuneCountInString(tag)
	}

	appendText := func(text string) {
		buf.WriteString(text)
		runeCount += utf8.RuneCountInString(text)
	}

	for _, token := range tokens {
		switch token.kind {
		case tokenTag:
			if runeCount+utf8.RuneCountInString(token.value) > limit {
				flush()
			}
			appendTag(token.value)
			if token.selfClosing {
				continue
			}
			if token.closing {
				openTags = popOpenTag(openTags, token.tagName)
			} else {
				openTags = append(openTags, openTag{name: token.tagName, open: token.value})
			}
		case tokenText:
			remaining := token.value
			for remaining != "" {
				available := limit - runeCount
				if available <= 0 {
					flush()
					available = limit - runeCount
					if available <= 0 {
						break
					}
				}
				if utf8.RuneCountInString(remaining) <= available {
					appendText(remaining)
					break
				}
				cut := safeTextCut(remaining, available)
				if cut == 0 {
					flush()
					available = limit - runeCount
					if available <= 0 {
						break
					}
					cut = safeTextCut(remaining, available)
					if cut == 0 {
						cut = cutByRunes(remaining, available)
					}
				}
				appendText(remaining[:cut])
				flush()
				remaining = remaining[cut:]
			}
		}
	}

	flush()
	return chunks
}

func PlainTextFromHTML(htmlText string) string {
	if htmlText == "" {
		return ""
	}
	clean := tagPattern.ReplaceAllString(htmlText, "")
	return strings.TrimSpace(html.UnescapeString(clean))
}

type tokenKind int

const (
	tokenText tokenKind = iota
	tokenTag
)

type htmlToken struct {
	kind        tokenKind
	value       string
	tagName     string
	closing     bool
	selfClosing bool
}

type openTag struct {
	name string
	open string
}

type listState struct {
	kind  string
	index int
}

type htmlWriter struct {
	builder  strings.Builder
	lastByte byte
}

func (w *htmlWriter) WriteString(value string) {
	if value == "" {
		return
	}
	w.builder.WriteString(value)
	w.lastByte = value[len(value)-1]
}

func (w *htmlWriter) EnsureNewline() {
	if w.lastByte == '\n' {
		return
	}
	w.builder.WriteString("\n")
	w.lastByte = '\n'
}

func (w *htmlWriter) String() string {
	return w.builder.String()
}

func tokenizeHTML(input string) []htmlToken {
	matches := htmlTagTokenPattern.FindAllStringIndex(input, -1)
	if len(matches) == 0 {
		return []htmlToken{{kind: tokenText, value: input}}
	}
	tokens := make([]htmlToken, 0, len(matches)*2)
	last := 0
	for _, match := range matches {
		start, end := match[0], match[1]
		if start > last {
			tokens = append(tokens, htmlToken{kind: tokenText, value: input[last:start]})
		}
		tagText := input[start:end]
		tokens = append(tokens, parseTagToken(tagText))
		last = end
	}
	if last < len(input) {
		tokens = append(tokens, htmlToken{kind: tokenText, value: input[last:]})
	}
	return tokens
}

func parseTagToken(tagText string) htmlToken {
	trimmed := strings.TrimSpace(tagText)
	match := tagPattern.FindStringSubmatch(trimmed)
	if len(match) < 2 {
		return htmlToken{kind: tokenText, value: trimmed}
	}
	name := strings.ToLower(match[1])
	closing := strings.HasPrefix(trimmed, "</")
	selfClosing := strings.HasSuffix(trimmed, "/>")
	return htmlToken{kind: tokenTag, value: trimmed, tagName: name, closing: closing, selfClosing: selfClosing}
}

func normalizeListBlocks(input string) string {
	if input == "" {
		return ""
	}
	tokens := tokenizeHTML(input)
	var out htmlWriter
	listStack := make([]listState, 0)
	for _, token := range tokens {
		if token.kind == tokenText {
			out.WriteString(token.value)
			continue
		}
		name := token.tagName
		if name == "" {
			out.WriteString(token.value)
			continue
		}
		if token.closing {
			switch name {
			case "ul", "ol":
				if len(listStack) > 0 {
					listStack = listStack[:len(listStack)-1]
				}
				if len(listStack) == 0 {
					out.EnsureNewline()
				}
				continue
			case "li":
				out.EnsureNewline()
				continue
			case "p":
				if len(listStack) > 0 {
					continue
				}
			}
			out.WriteString(token.value)
			continue
		}

		switch name {
		case "ul", "ol":
			startIndex := 0
			if name == "ol" {
				start := parseOrderedListStart(token.value)
				if start < 1 {
					start = 1
				}
				startIndex = start - 1
			}
			listStack = append(listStack, listState{kind: name, index: startIndex})
			continue
		case "li":
			if len(listStack) > 0 {
				depth := len(listStack)
				top := &listStack[len(listStack)-1]
				out.WriteString(indentForDepth(depth))
				if top.kind == "ol" {
					top.index++
					out.WriteString(fmt.Sprintf("%d. ", top.index))
				} else {
					out.WriteString(bulletForDepth(depth))
				}
			}
			continue
		case "p":
			if len(listStack) > 0 {
				continue
			}
		}
		out.WriteString(token.value)
	}
	return out.String()
}

func indentForDepth(depth int) string {
	if depth <= 1 {
		return ""
	}
	return strings.Repeat("  ", depth-1)
}

func bulletForDepth(depth int) string {
	if depth <= 1 {
		return "\u2022 "
	}
	return "\u25E6 "
}

func parseOrderedListStart(tag string) int {
	match := orderedListStartPattern.FindStringSubmatch(tag)
	if len(match) < 2 {
		return 1
	}
	value, err := strconv.Atoi(strings.TrimSpace(match[1]))
	if err != nil || value < 1 {
		return 1
	}
	return value
}

func popOpenTag(tags []openTag, name string) []openTag {
	if len(tags) == 0 || name == "" {
		return tags
	}
	for i := len(tags) - 1; i >= 0; i-- {
		if tags[i].name == name {
			return tags[:i]
		}
	}
	return tags
}

func buildClosingTags(tags []openTag) string {
	if len(tags) == 0 {
		return ""
	}
	var buf strings.Builder
	for i := len(tags) - 1; i >= 0; i-- {
		buf.WriteString("</")
		buf.WriteString(tags[i].name)
		buf.WriteString(">")
	}
	return buf.String()
}

func safeTextCut(text string, limit int) int {
	if limit <= 0 {
		return 0
	}
	count := 0
	inEntity := false
	lastSafe := 0
	for idx, r := range text {
		if !inEntity && r == '&' {
			inEntity = true
		} else if inEntity && r == ';' {
			inEntity = false
		}
		count++
		if count > limit {
			break
		}
		if !inEntity {
			lastSafe = idx + utf8.RuneLen(r)
		}
	}
	if count <= limit {
		return len(text)
	}
	return lastSafe
}

func cutByRunes(text string, limit int) int {
	if limit <= 0 {
		return 0
	}
	count := 0
	for idx, r := range text {
		count++
		if count == limit {
			return idx + utf8.RuneLen(r)
		}
	}
	return len(text)
}

func normalizeTelegramHTML(input string) string {
	if input == "" {
		return ""
	}
	output := strings.ReplaceAll(input, "\r\n", "\n")
	output = rawHTMLOmittedPattern.ReplaceAllString(output, "")
	output = brTagPattern.ReplaceAllString(output, "\n")
	output = headingOpenPattern.ReplaceAllString(output, "")
	output = headingClosePattern.ReplaceAllString(output, "\n")
	output = strongOpenPattern.ReplaceAllString(output, "<b>")
	output = strongClosePattern.ReplaceAllString(output, "</b>")
	output = emOpenPattern.ReplaceAllString(output, "<i>")
	output = emClosePattern.ReplaceAllString(output, "</i>")
	output = delOpenPattern.ReplaceAllString(output, "<s>")
	output = delClosePattern.ReplaceAllString(output, "</s>")
	output = strikeOpenPattern.ReplaceAllString(output, "<s>")
	output = strikeClosePattern.ReplaceAllString(output, "</s>")
	output = insOpenPattern.ReplaceAllString(output, "<u>")
	output = insClosePattern.ReplaceAllString(output, "</u>")
	output = normalizeSpanSpoilerTags(output)
	output = normalizeCodeTags(output)
	output = normalizeTaskListCheckboxes(output)
	output = preOpenPattern.ReplaceAllString(output, "<pre>")
	output = normalizeListBlocks(output)
	output = normalizeTables(output)
	output = pOpenPattern.ReplaceAllString(output, "")
	output = pClosePattern.ReplaceAllString(output, "\n\n")
	output = hrPattern.ReplaceAllString(output, "\n---\n")
	output = imgPattern.ReplaceAllString(output, "[image]")
	output = sanitizeAnchors(output)
	output = stripUnsupportedTags(output)
	output = normalizeBlockquoteSpacing(output)
	output = normalizeSpoilerTags(output)
	output = wrapFileReferencesInHTML(output)
	output = normalizeListItemSpacing(output)
	output = multiNewlinePattern.ReplaceAllString(output, "\n\n")
	return strings.TrimSpace(output)
}

func normalizeBlockquoteSpacing(input string) string {
	if input == "" {
		return ""
	}
	output := blockquoteContentPattern.ReplaceAllStringFunc(input, func(block string) string {
		match := blockquoteContentPattern.FindStringSubmatch(block)
		if len(match) < 3 {
			return block
		}
		content := strings.Trim(match[2], "\n")
		if strings.TrimSpace(match[1]) != "" {
			return "<blockquote expandable>" + content + "</blockquote>"
		}
		return "<blockquote>" + content + "</blockquote>"
	})
	return output
}

func normalizeTaskListCheckboxes(input string) string {
	if input == "" {
		return ""
	}
	return checkboxInputPattern.ReplaceAllStringFunc(input, func(tag string) string {
		if checkedAttrPattern.MatchString(tag) {
			return "[x] "
		}
		return "[ ] "
	})
}

func normalizeSpanSpoilerTags(input string) string {
	if input == "" {
		return ""
	}
	tokens := tokenizeHTML(input)
	var result strings.Builder
	spanStack := make([]bool, 0, 4)
	for _, token := range tokens {
		if token.kind != tokenTag || token.tagName != "span" {
			result.WriteString(token.value)
			continue
		}
		if token.closing {
			emitClose := false
			if len(spanStack) > 0 {
				emitClose = spanStack[len(spanStack)-1]
				spanStack = spanStack[:len(spanStack)-1]
			}
			if emitClose {
				result.WriteString("</tg-spoiler>")
			}
			continue
		}
		classValue := extractTagClass(token.value)
		isSpoiler := hasClassToken(classValue, "tg-spoiler")
		spanStack = append(spanStack, isSpoiler)
		if isSpoiler {
			result.WriteString("<tg-spoiler>")
		}
	}
	return result.String()
}

func normalizeSpoilerTags(input string) string {
	if input == "" {
		return ""
	}
	codeDepth := 0
	preDepth := 0
	var result strings.Builder
	lastIndex := 0

	matches := htmlTagPattern.FindAllStringSubmatchIndex(input, -1)
	for _, match := range matches {
		tagStart := match[0]
		tagEnd := match[1]
		if tagStart < lastIndex {
			continue
		}
		textBefore := input[lastIndex:tagStart]
		result.WriteString(replaceSpoilersInText(textBefore, codeDepth, preDepth))

		tagText := input[tagStart:tagEnd]
		isClosing := strings.HasPrefix(strings.TrimSpace(tagText), "</")
		name := strings.ToLower(input[match[4]:match[5]])
		switch name {
		case "code":
			if isClosing {
				if codeDepth > 0 {
					codeDepth--
				}
			} else {
				codeDepth++
			}
		case "pre":
			if isClosing {
				if preDepth > 0 {
					preDepth--
				}
			} else {
				preDepth++
			}
		}
		result.WriteString(tagText)
		lastIndex = tagEnd
	}
	if lastIndex < len(input) {
		result.WriteString(replaceSpoilersInText(input[lastIndex:], codeDepth, preDepth))
	}
	return result.String()
}

func replaceSpoilersInText(text string, codeDepth int, preDepth int) string {
	if text == "" || codeDepth > 0 || preDepth > 0 {
		return text
	}
	return spoilerPattern.ReplaceAllString(text, "<tg-spoiler>$1</tg-spoiler>")
}

func normalizeListItemSpacing(input string) string {
	if input == "" {
		return ""
	}
	lines := strings.Split(input, "\n")
	result := make([]string, 0, len(lines))
	for idx, line := range lines {
		if strings.TrimSpace(line) == "" {
			prevLine := ""
			nextLine := ""
			if len(result) > 0 {
				prevLine = strings.TrimSpace(result[len(result)-1])
			}
			if idx+1 < len(lines) {
				nextLine = strings.TrimSpace(lines[idx+1])
			}
			if listItemLinePattern.MatchString(prevLine) && listItemLinePattern.MatchString(nextLine) {
				continue
			}
		}
		result = append(result, line)
	}
	return strings.Join(result, "\n")
}

func normalizeCodeTags(input string) string {
	if input == "" {
		return ""
	}
	tokens := tokenizeHTML(input)
	var result strings.Builder
	for _, token := range tokens {
		if token.kind != tokenTag || token.tagName != "code" {
			result.WriteString(token.value)
			continue
		}
		if token.closing {
			result.WriteString("</code>")
			continue
		}
		className := ""
		match := codeClassPattern.FindStringSubmatch(token.value)
		if len(match) >= 2 {
			className = strings.TrimSpace(match[1])
		}
		if strings.HasPrefix(strings.ToLower(className), "language-") {
			result.WriteString(`<code class="` + html.EscapeString(className) + `">`)
			continue
		}
		result.WriteString("<code>")
	}
	return result.String()
}

func sanitizeAnchors(input string) string {
	if input == "" {
		return ""
	}
	tokens := tokenizeHTML(input)
	var result strings.Builder
	anchorStack := make([]bool, 0, 2)
	for _, token := range tokens {
		if token.kind != tokenTag || token.tagName != "a" {
			result.WriteString(token.value)
			continue
		}
		if token.closing {
			allowClose := false
			if len(anchorStack) > 0 {
				allowClose = anchorStack[len(anchorStack)-1]
				anchorStack = anchorStack[:len(anchorStack)-1]
			}
			if allowClose {
				result.WriteString("</a>")
			}
			continue
		}
		match := anchorHrefPattern.FindStringSubmatch(token.value)
		if len(match) < 2 {
			anchorStack = append(anchorStack, false)
			continue
		}
		hrefValue := strings.TrimSpace(html.UnescapeString(match[1]))
		if !isAllowedTelegramHref(hrefValue) {
			anchorStack = append(anchorStack, false)
			continue
		}
		anchorStack = append(anchorStack, true)
		result.WriteString(`<a href="` + html.EscapeString(hrefValue) + `">`)
	}
	return result.String()
}

func isAllowedTelegramHref(hrefValue string) bool {
	trimmed := strings.TrimSpace(hrefValue)
	if trimmed == "" {
		return false
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(parsed.Scheme)) {
	case "http", "https", "tg":
		return true
	default:
		return false
	}
}

func stripUnsupportedTags(input string) string {
	if input == "" {
		return ""
	}
	return tagPattern.ReplaceAllStringFunc(input, func(tag string) string {
		match := tagPattern.FindStringSubmatch(tag)
		if len(match) < 2 {
			return ""
		}
		name := strings.ToLower(match[1])
		if _, ok := allowedTelegramTags[name]; !ok {
			return ""
		}
		if name == "a" {
			return tag
		}
		if name == "blockquote" {
			if strings.HasPrefix(strings.TrimSpace(tag), "</") {
				return "</blockquote>"
			}
			if blockquoteExpandableRE.MatchString(tag) {
				return "<blockquote expandable>"
			}
			return "<blockquote>"
		}
		if name == "code" {
			if strings.HasPrefix(strings.TrimSpace(tag), "</") {
				return "</code>"
			}
			match := codeClassPattern.FindStringSubmatch(tag)
			if len(match) >= 2 {
				className := strings.TrimSpace(match[1])
				if strings.HasPrefix(strings.ToLower(className), "language-") {
					return `<code class="` + html.EscapeString(className) + `">`
				}
			}
			return "<code>"
		}
		if name == "tg-emoji" {
			if strings.HasPrefix(strings.TrimSpace(tag), "</") {
				return "</tg-emoji>"
			}
			match := tgEmojiIDPattern.FindStringSubmatch(tag)
			if len(match) < 2 {
				return ""
			}
			emojiID := strings.TrimSpace(match[1])
			if emojiID == "" {
				return ""
			}
			return `<tg-emoji emoji-id="` + emojiID + `">`
		}
		if strings.HasPrefix(strings.TrimSpace(tag), "</") {
			return "</" + name + ">"
		}
		return "<" + name + ">"
	})
}

func extractTagClass(tag string) string {
	match := classAttrPattern.FindStringSubmatch(tag)
	if len(match) < 4 {
		return ""
	}
	if strings.TrimSpace(match[2]) != "" {
		return strings.TrimSpace(match[2])
	}
	return strings.TrimSpace(match[3])
}

func hasClassToken(classValue string, token string) bool {
	if strings.TrimSpace(classValue) == "" || strings.TrimSpace(token) == "" {
		return false
	}
	for _, item := range strings.Fields(strings.ToLower(classValue)) {
		if item == strings.ToLower(strings.TrimSpace(token)) {
			return true
		}
	}
	return false
}

type tableRow struct {
	Cells  []string
	Header bool
}

func normalizeTables(input string) string {
	if input == "" {
		return ""
	}
	tokens := tokenizeHTML(input)
	var result strings.Builder
	var tableBuf strings.Builder
	tableDepth := 0
	for _, token := range tokens {
		if token.kind == tokenTag && token.tagName == "table" {
			if token.closing {
				if tableDepth > 0 {
					tableDepth--
					if tableDepth == 0 {
						result.WriteString(renderTableAsTelegramPre(tableBuf.String()))
						tableBuf.Reset()
					}
					continue
				}
			} else {
				tableDepth++
				if tableDepth == 1 {
					tableBuf.Reset()
					continue
				}
			}
		}
		if tableDepth > 0 {
			tableBuf.WriteString(token.value)
			continue
		}
		result.WriteString(token.value)
	}
	if tableDepth > 0 && tableBuf.Len() > 0 {
		result.WriteString(tableBuf.String())
	}
	return result.String()
}

func renderTableAsTelegramPre(inner string) string {
	rows := parseTableRows(inner)
	if len(rows) == 0 {
		plain := strings.TrimSpace(PlainTextFromHTML(inner))
		if plain == "" {
			return ""
		}
		return "<pre>" + html.EscapeString(plain) + "</pre>"
	}
	lines := renderTableLines(rows)
	if len(lines) == 0 {
		return ""
	}
	return "<pre>" + html.EscapeString(strings.Join(lines, "\n")) + "</pre>"
}

func parseTableRows(inner string) []tableRow {
	tokens := tokenizeHTML(inner)
	rows := make([]tableRow, 0, 4)
	current := tableRow{Cells: make([]string, 0, 4)}
	inRow := false
	inCell := false
	cellIsHeader := false
	var cellBuf strings.Builder

	flushCell := func() {
		if !inRow || !inCell {
			return
		}
		current.Cells = append(current.Cells, normalizeTableCellText(cellBuf.String()))
		if cellIsHeader {
			current.Header = true
		}
		cellBuf.Reset()
		inCell = false
		cellIsHeader = false
	}
	flushRow := func() {
		if !inRow {
			return
		}
		flushCell()
		if len(current.Cells) > 0 {
			rows = append(rows, current)
		}
		current = tableRow{Cells: make([]string, 0, 4)}
		inRow = false
	}

	for _, token := range tokens {
		if token.kind == tokenText {
			if inCell {
				cellBuf.WriteString(token.value)
			}
			continue
		}
		switch token.tagName {
		case "tr":
			if token.closing {
				flushRow()
			} else {
				flushRow()
				inRow = true
			}
		case "th", "td":
			if !inRow {
				continue
			}
			if token.closing {
				flushCell()
				continue
			}
			flushCell()
			inCell = true
			cellIsHeader = token.tagName == "th"
			cellBuf.Reset()
		case "br":
			if inCell && !token.closing {
				cellBuf.WriteString("\n")
			}
		default:
			if inCell {
				cellBuf.WriteString(token.value)
			}
		}
	}
	flushRow()
	return rows
}

func normalizeTableCellText(value string) string {
	plain := strings.TrimSpace(PlainTextFromHTML(value))
	if plain == "" {
		return ""
	}
	lines := strings.Split(strings.ReplaceAll(plain, "\r\n", "\n"), "\n")
	normalized := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.Join(strings.Fields(strings.TrimSpace(line)), " ")
		if trimmed == "" {
			continue
		}
		normalized = append(normalized, trimmed)
	}
	return strings.Join(normalized, " / ")
}

func renderTableLines(rows []tableRow) []string {
	if len(rows) == 0 {
		return nil
	}
	colCount := 0
	for _, row := range rows {
		if len(row.Cells) > colCount {
			colCount = len(row.Cells)
		}
	}
	if colCount == 0 {
		return nil
	}
	widths := make([]int, colCount)
	for _, row := range rows {
		for col := 0; col < colCount; col++ {
			cell := ""
			if col < len(row.Cells) {
				cell = row.Cells[col]
			}
			cellWidth := utf8.RuneCountInString(cell)
			if cellWidth > widths[col] {
				widths[col] = cellWidth
			}
		}
	}
	for idx := range widths {
		if widths[idx] < 1 {
			widths[idx] = 1
		}
	}
	lines := make([]string, 0, len(rows)+1)
	headerWritten := false
	for idx, row := range rows {
		lines = append(lines, renderTableLine(row.Cells, widths))
		if row.Header && !headerWritten {
			lines = append(lines, renderTableSeparator(widths))
			headerWritten = true
		}
		if idx == 0 && !headerWritten && len(rows) > 1 {
			// No explicit header markers; keep content-only table.
		}
	}
	return lines
}

func renderTableLine(cells []string, widths []int) string {
	var line strings.Builder
	line.WriteString("|")
	for idx, width := range widths {
		cell := ""
		if idx < len(cells) {
			cell = cells[idx]
		}
		line.WriteString(" ")
		line.WriteString(padRightRunes(cell, width))
		line.WriteString(" |")
	}
	return line.String()
}

func renderTableSeparator(widths []int) string {
	var line strings.Builder
	line.WriteString("|")
	for _, width := range widths {
		dashCount := width
		if dashCount < 3 {
			dashCount = 3
		}
		line.WriteString(" ")
		line.WriteString(strings.Repeat("-", dashCount))
		line.WriteString(" |")
	}
	return line.String()
}

func padRightRunes(value string, width int) string {
	current := utf8.RuneCountInString(value)
	if current >= width {
		return value
	}
	return value + strings.Repeat(" ", width-current)
}

func wrapFileReferencesInHTML(input string) string {
	if input == "" {
		return ""
	}
	deLinked := autoLinkedAnchorPattern.ReplaceAllStringFunc(input, func(match string) string {
		sub := autoLinkedAnchorPattern.FindStringSubmatch(match)
		if len(sub) < 3 {
			return match
		}
		hrefLabel := sub[1]
		label := sub[2]
		if !strings.EqualFold(hrefLabel, label) {
			return match
		}
		if !isAutoLinkedFileRef("http://"+label, label) {
			return match
		}
		return "<code>" + html.EscapeString(label) + "</code>"
	})

	codeDepth := 0
	preDepth := 0
	anchorDepth := 0
	var result strings.Builder
	lastIndex := 0

	matches := htmlTagPattern.FindAllStringSubmatchIndex(deLinked, -1)
	for _, match := range matches {
		tagStart := match[0]
		tagEnd := match[1]
		if tagStart < lastIndex {
			continue
		}
		textBefore := deLinked[lastIndex:tagStart]
		result.WriteString(wrapSegmentFileRefs(textBefore, codeDepth, preDepth, anchorDepth))

		tagText := deLinked[tagStart:tagEnd]
		isClosing := strings.HasPrefix(tagText, "</")
		name := strings.ToLower(deLinked[match[4]:match[5]])
		switch name {
		case "code":
			if isClosing {
				if codeDepth > 0 {
					codeDepth--
				}
			} else {
				codeDepth++
			}
		case "pre":
			if isClosing {
				if preDepth > 0 {
					preDepth--
				}
			} else {
				preDepth++
			}
		case "a":
			if isClosing {
				if anchorDepth > 0 {
					anchorDepth--
				}
			} else {
				anchorDepth++
			}
		}
		result.WriteString(tagText)
		lastIndex = tagEnd
	}

	if lastIndex < len(deLinked) {
		remaining := deLinked[lastIndex:]
		result.WriteString(wrapSegmentFileRefs(remaining, codeDepth, preDepth, anchorDepth))
	}

	return result.String()
}

func wrapSegmentFileRefs(text string, codeDepth, preDepth, anchorDepth int) string {
	if text == "" || codeDepth > 0 || preDepth > 0 || anchorDepth > 0 {
		return text
	}
	wrapped := fileRefPattern.ReplaceAllStringFunc(text, func(match string) string {
		sub := fileRefPattern.FindStringSubmatch(match)
		if len(sub) < 4 {
			return match
		}
		prefix := sub[1]
		filename := sub[2]
		suffix := sub[3]
		if strings.HasPrefix(filename, "//") {
			return match
		}
		lowerPrefix := strings.ToLower(prefix)
		if strings.HasSuffix(lowerPrefix, "http://") || strings.HasSuffix(lowerPrefix, "https://") {
			return match
		}
		return prefix + "<code>" + html.EscapeString(filename) + "</code>" + suffix
	})
	return orphanedTLDPattern.ReplaceAllStringFunc(wrapped, func(match string) string {
		sub := orphanedTLDPattern.FindStringSubmatch(match)
		if len(sub) < 4 {
			return match
		}
		prefix := sub[1]
		tld := sub[2]
		suffix := sub[3]
		if prefix == ">" {
			return match
		}
		return prefix + "<code>" + html.EscapeString(tld) + "</code>" + suffix
	})
}

func isAutoLinkedFileRef(href string, label string) bool {
	stripped := strings.TrimPrefix(strings.TrimPrefix(strings.ToLower(href), "http://"), "https://")
	if stripped != strings.ToLower(label) {
		return false
	}
	dotIndex := strings.LastIndex(label, ".")
	if dotIndex < 1 {
		return false
	}
	ext := strings.ToLower(label[dotIndex+1:])
	if !contains(fileExtensionsWithTLD, ext) {
		return false
	}
	segments := strings.Split(label, "/")
	if len(segments) > 1 {
		for i := 0; i < len(segments)-1; i++ {
			if strings.Contains(segments[i], ".") {
				return false
			}
		}
	}
	return true
}

func escapeRegexList(items []string) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		result = append(result, regexp.QuoteMeta(item))
	}
	return result
}

func contains(items []string, value string) bool {
	for _, item := range items {
		if item == value {
			return true
		}
	}
	return false
}
