package telegram

import (
	"context"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	runtimedto "dreamcreator/internal/application/gateway/runtime/dto"
	telegramapi "dreamcreator/internal/infrastructure/telegram"
)

const (
	telegramDraftStreamMinInitialChars = 30
	telegramDraftStreamMinThrottle     = 250 * time.Millisecond
	telegramDraftStreamDefaultThrottle = time.Second
)

type draftStreamLoop struct {
	throttle   time.Duration
	isStopped  func() bool
	sendOrEdit func(text string) bool

	mu           sync.Mutex
	pendingText  string
	lastSentAt   time.Time
	inFlight     bool
	inFlightDone chan struct{}
	timer        *time.Timer
	stopped      bool
}

func newDraftStreamLoop(throttle time.Duration, isStopped func() bool, sendOrEdit func(text string) bool) *draftStreamLoop {
	if throttle < telegramDraftStreamMinThrottle {
		throttle = telegramDraftStreamMinThrottle
	}
	return &draftStreamLoop{
		throttle:   throttle,
		isStopped:  isStopped,
		sendOrEdit: sendOrEdit,
	}
}

func (loop *draftStreamLoop) Update(text string) {
	if loop == nil {
		return
	}
	loop.mu.Lock()
	if loop.stopped {
		loop.mu.Unlock()
		return
	}
	loop.pendingText = text
	if loop.inFlight {
		loop.scheduleLocked()
		loop.mu.Unlock()
		return
	}
	if loop.timer == nil && time.Since(loop.lastSentAt) >= loop.throttle {
		loop.startFlushLocked()
		loop.mu.Unlock()
		go loop.flushLoop()
		return
	}
	loop.scheduleLocked()
	loop.mu.Unlock()
}

func (loop *draftStreamLoop) Flush() {
	if loop == nil {
		return
	}
	for {
		loop.mu.Lock()
		if loop.stopped {
			loop.mu.Unlock()
			return
		}
		if loop.inFlight {
			done := loop.inFlightDone
			loop.mu.Unlock()
			if done != nil {
				<-done
			}
			continue
		}
		loop.startFlushLocked()
		loop.mu.Unlock()
		loop.flushLoop()
		return
	}
}

func (loop *draftStreamLoop) Stop() {
	if loop == nil {
		return
	}
	loop.mu.Lock()
	loop.stopped = true
	loop.pendingText = ""
	if loop.timer != nil {
		loop.timer.Stop()
		loop.timer = nil
	}
	loop.mu.Unlock()
}

func (loop *draftStreamLoop) ResetPending() {
	if loop == nil {
		return
	}
	loop.mu.Lock()
	loop.pendingText = ""
	loop.mu.Unlock()
}

func (loop *draftStreamLoop) WaitForInFlight() {
	if loop == nil {
		return
	}
	loop.mu.Lock()
	done := loop.inFlightDone
	loop.mu.Unlock()
	if done != nil {
		<-done
	}
}

func (loop *draftStreamLoop) scheduleLocked() {
	if loop.timer != nil {
		return
	}
	delay := loop.throttle - time.Since(loop.lastSentAt)
	if delay < 0 {
		delay = 0
	}
	loop.timer = time.AfterFunc(delay, func() {
		loop.mu.Lock()
		loop.timer = nil
		if loop.stopped || loop.inFlight {
			loop.mu.Unlock()
			return
		}
		loop.startFlushLocked()
		loop.mu.Unlock()
		loop.flushLoop()
	})
}

func (loop *draftStreamLoop) startFlushLocked() {
	loop.inFlight = true
	loop.inFlightDone = make(chan struct{})
	if loop.timer != nil {
		loop.timer.Stop()
		loop.timer = nil
	}
}

func (loop *draftStreamLoop) finishFlush() {
	loop.mu.Lock()
	done := loop.inFlightDone
	loop.inFlightDone = nil
	loop.inFlight = false
	loop.mu.Unlock()
	if done != nil {
		close(done)
	}
}

func (loop *draftStreamLoop) flushLoop() {
	for {
		if loop == nil {
			return
		}
		if loop.isStopped != nil && loop.isStopped() {
			loop.finishFlush()
			return
		}
		loop.mu.Lock()
		if loop.timer != nil {
			loop.timer.Stop()
			loop.timer = nil
		}
		text := loop.pendingText
		loop.pendingText = ""
		loop.mu.Unlock()

		if strings.TrimSpace(text) == "" {
			loop.finishFlush()
			return
		}

		sent := loop.sendOrEdit(text)
		if !sent {
			loop.mu.Lock()
			if loop.pendingText == "" {
				loop.pendingText = text
			}
			loop.mu.Unlock()
			loop.finishFlush()
			return
		}

		loop.mu.Lock()
		loop.lastSentAt = time.Now()
		pending := loop.pendingText
		loop.mu.Unlock()
		if pending == "" {
			loop.finishFlush()
			return
		}
	}
}

type telegramDraftStream struct {
	ctx        context.Context
	service    *BotService
	state      *telegramAccountState
	chatID     int64
	threadID   int
	replyTo    int
	maxChars   int
	minInitial int
	throttle   time.Duration

	mu        sync.Mutex
	messageID int
	lastSent  string
	stopped   bool
	isFinal   bool
	loop      *draftStreamLoop
}

func newTelegramDraftStream(
	ctx context.Context,
	service *BotService,
	state *telegramAccountState,
	chatID int64,
	threadID int,
	replyTo int,
	maxChars int,
	minInitial int,
	throttle time.Duration,
) *telegramDraftStream {
	if maxChars <= 0 {
		maxChars = 4096
	}
	stream := &telegramDraftStream{
		ctx:        ctx,
		service:    service,
		state:      state,
		chatID:     chatID,
		threadID:   threadID,
		replyTo:    replyTo,
		maxChars:   maxChars,
		minInitial: minInitial,
		throttle:   throttle,
	}
	stream.loop = newDraftStreamLoop(throttle, stream.isStopped, stream.sendOrEdit)
	return stream
}

func (stream *telegramDraftStream) Update(text string) {
	if stream == nil || stream.loop == nil {
		return
	}
	stream.mu.Lock()
	if stream.stopped || stream.isFinal {
		stream.mu.Unlock()
		return
	}
	stream.mu.Unlock()
	stream.loop.Update(text)
}

func (stream *telegramDraftStream) Flush() {
	if stream == nil || stream.loop == nil {
		return
	}
	stream.loop.Flush()
}

func (stream *telegramDraftStream) Stop() {
	if stream == nil || stream.loop == nil {
		return
	}
	stream.mu.Lock()
	stream.isFinal = true
	stream.mu.Unlock()
	stream.loop.Flush()
}

func (stream *telegramDraftStream) Clear() {
	if stream == nil || stream.loop == nil {
		return
	}
	stream.mu.Lock()
	stream.stopped = true
	messageID := stream.messageID
	stream.messageID = 0
	stream.mu.Unlock()
	stream.loop.Stop()
	stream.loop.WaitForInFlight()
	if messageID <= 0 {
		return
	}
	ctx := stream.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	_ = stream.service.deleteDraftMessage(ctx, stream.state, stream.chatID, messageID)
}

func (stream *telegramDraftStream) ForceNewMessage() {
	if stream == nil {
		return
	}
	stream.mu.Lock()
	stream.messageID = 0
	stream.lastSent = ""
	stream.mu.Unlock()
	if stream.loop != nil {
		stream.loop.ResetPending()
	}
}

func (stream *telegramDraftStream) MessageID() int {
	if stream == nil {
		return 0
	}
	stream.mu.Lock()
	defer stream.mu.Unlock()
	return stream.messageID
}

func (stream *telegramDraftStream) AdoptMessage(messageID int) {
	if stream == nil || messageID <= 0 {
		return
	}
	stream.mu.Lock()
	if stream.messageID == 0 {
		stream.messageID = messageID
	}
	stream.mu.Unlock()
}

func (stream *telegramDraftStream) isStopped() bool {
	if stream == nil {
		return true
	}
	stream.mu.Lock()
	defer stream.mu.Unlock()
	return stream.stopped
}

func (stream *telegramDraftStream) sendOrEdit(text string) bool {
	if stream == nil {
		return false
	}
	stream.mu.Lock()
	stopped := stream.stopped
	isFinal := stream.isFinal
	stream.mu.Unlock()
	if stopped && !isFinal {
		return false
	}
	trimmed := strings.TrimRightFunc(text, unicode.IsSpace)
	if strings.TrimSpace(trimmed) == "" {
		return false
	}
	chunk, ok, tooLong := renderDraftChunk(trimmed, stream.maxChars)
	if !ok {
		return false
	}
	if tooLong {
		stream.mu.Lock()
		stream.stopped = true
		stream.mu.Unlock()
		return false
	}
	stream.mu.Lock()
	messageID := stream.messageID
	lastSent := stream.lastSent
	stream.mu.Unlock()
	if chunk.HTML == lastSent {
		return true
	}
	if messageID == 0 && stream.minInitial > 0 && !isFinal {
		if utf8.RuneCountInString(chunk.HTML) < stream.minInitial {
			return false
		}
	}
	ctx := stream.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	if messageID > 0 {
		editedID, err := stream.service.editPlaceholder(ctx, stream.state, stream.chatID, messageID, chunk, false)
		if err == nil {
			stream.service.recordOutboundSuccess(stream.state, int64(editedID))
			stream.mu.Lock()
			stream.messageID = editedID
			stream.lastSent = chunk.HTML
			stream.mu.Unlock()
			return true
		}
		stream.mu.Lock()
		stream.stopped = true
		stream.mu.Unlock()
		return false
	}
	sentID, err := stream.service.sendDraftMessage(ctx, stream.state, stream.chatID, stream.threadID, stream.replyTo, chunk)
	if err == nil {
		stream.service.recordOutboundSuccess(stream.state, int64(sentID))
		stream.mu.Lock()
		stream.messageID = sentID
		stream.lastSent = chunk.HTML
		stream.mu.Unlock()
		return true
	}
	stream.mu.Lock()
	stream.stopped = true
	stream.mu.Unlock()
	return false
}

func renderDraftChunk(text string, limit int) (telegramapi.FormattedChunk, bool, bool) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return telegramapi.FormattedChunk{}, false, false
	}
	if limit <= 0 {
		limit = 4096
	}
	htmlText := telegramapi.RenderTelegramHTML(trimmed)
	if htmlText == "" {
		return telegramapi.FormattedChunk{}, false, false
	}
	if utf8.RuneCountInString(htmlText) > limit {
		return telegramapi.FormattedChunk{}, true, true
	}
	return telegramapi.FormattedChunk{HTML: htmlText, Text: telegramapi.PlainTextFromHTML(htmlText)}, true, false
}

type telegramFenceSpan struct {
	start    int
	end      int
	openLine string
	marker   string
	indent   string
}

type telegramFenceSplit struct {
	closeFenceLine  string
	reopenFenceLine string
}

type telegramBreakResult struct {
	index      int
	fenceSplit *telegramFenceSplit
}

type telegramDraftChunker struct {
	minChars        int
	maxChars        int
	breakPreference string
	buffer          []rune
}

func newTelegramDraftChunker(cfg TelegramDraftChunkConfig) *telegramDraftChunker {
	minChars := cfg.MinChars
	maxChars := cfg.MaxChars
	if minChars <= 0 {
		minChars = 200
	}
	if maxChars <= 0 {
		maxChars = 800
	}
	if maxChars < minChars {
		maxChars = minChars
	}
	breakPref := strings.ToLower(strings.TrimSpace(cfg.BreakPreference))
	switch breakPref {
	case "newline", "sentence":
	default:
		breakPref = "paragraph"
	}
	return &telegramDraftChunker{
		minChars:        minChars,
		maxChars:        maxChars,
		breakPreference: breakPref,
	}
}

func (chunker *telegramDraftChunker) Append(delta string) []string {
	if chunker == nil || delta == "" {
		return nil
	}
	chunker.buffer = append(chunker.buffer, []rune(delta)...)
	return chunker.drain(false)
}

func (chunker *telegramDraftChunker) Flush() []string {
	if chunker == nil {
		return nil
	}
	return chunker.drain(true)
}

func (chunker *telegramDraftChunker) Reset() {
	if chunker == nil {
		return
	}
	chunker.buffer = nil
}

func (chunker *telegramDraftChunker) drain(force bool) []string {
	if chunker == nil || len(chunker.buffer) == 0 {
		return nil
	}
	minChars := max(1, chunker.minChars)
	maxChars := max(minChars, chunker.maxChars)
	result := make([]string, 0, 2)
	if force && len(chunker.buffer) <= maxChars {
		chunk := string(chunker.buffer)
		if strings.TrimSpace(chunk) != "" {
			result = append(result, chunk)
		}
		chunker.buffer = nil
		return result
	}
	if len(chunker.buffer) < minChars && !force {
		return nil
	}
	for len(chunker.buffer) >= minChars || (force && len(chunker.buffer) > 0) {
		var breakResult telegramBreakResult
		if force && len(chunker.buffer) <= maxChars {
			breakResult = chunker.pickSoftBreakIndex(chunker.buffer, 1)
		} else {
			breakResult = chunker.pickBreakIndex(chunker.buffer, map[bool]int{true: 1, false: 0}[force])
		}
		if breakResult.index <= 0 {
			if force {
				chunk := string(chunker.buffer)
				if strings.TrimSpace(chunk) != "" {
					result = append(result, chunk)
				}
				chunker.buffer = nil
			}
			return result
		}
		if !chunker.emitBreakResult(breakResult, &result) {
			continue
		}
		if len(chunker.buffer) < minChars && !force {
			return result
		}
		if len(chunker.buffer) < maxChars && !force {
			return result
		}
	}
	return result
}

func (chunker *telegramDraftChunker) emitBreakResult(breakResult telegramBreakResult, result *[]string) bool {
	if chunker == nil || breakResult.index <= 0 {
		return false
	}
	rawChunkRunes := chunker.buffer[:breakResult.index]
	rawChunk := string(rawChunkRunes)
	if strings.TrimSpace(rawChunk) == "" {
		next := chunker.buffer[breakResult.index:]
		next = stripLeadingNewlines(next)
		next = stripLeadingWhitespace(next)
		chunker.buffer = next
		return false
	}

	nextBuffer := string(chunker.buffer[breakResult.index:])
	if breakResult.fenceSplit != nil {
		closeFence := breakResult.fenceSplit.closeFenceLine
		if strings.HasSuffix(rawChunk, "\n") {
			rawChunk = rawChunk + closeFence + "\n"
		} else {
			rawChunk = rawChunk + "\n" + closeFence + "\n"
		}
		reopenFence := breakResult.fenceSplit.reopenFenceLine
		if !strings.HasSuffix(reopenFence, "\n") {
			reopenFence += "\n"
		}
		nextBuffer = reopenFence + nextBuffer
	}
	*result = append(*result, rawChunk)
	if breakResult.fenceSplit != nil {
		chunker.buffer = []rune(nextBuffer)
		return true
	}

	nextStart := breakResult.index
	if breakResult.index < len(chunker.buffer) && unicode.IsSpace(chunker.buffer[breakResult.index]) {
		nextStart = breakResult.index + 1
	}
	next := chunker.buffer[nextStart:]
	next = stripLeadingNewlines(next)
	chunker.buffer = next
	return true
}

func (chunker *telegramDraftChunker) pickSoftBreakIndex(buffer []rune, minCharsOverride int) telegramBreakResult {
	minChars := max(1, minCharsOverride)
	if len(buffer) < minChars {
		return telegramBreakResult{index: -1}
	}
	fenceSpans := parseTelegramFenceSpans(buffer)
	preference := chunker.breakPreference
	if preference == "paragraph" {
		if idx := findSafeParagraphBreakIndex(buffer, fenceSpans, minChars, false); idx != -1 {
			return telegramBreakResult{index: idx}
		}
	}
	if preference == "paragraph" || preference == "newline" {
		if idx := findSafeNewlineBreakIndex(buffer, fenceSpans, minChars, false); idx != -1 {
			return telegramBreakResult{index: idx}
		}
	}
	if preference != "newline" {
		if idx := findSafeSentenceBreakIndex(buffer, fenceSpans, minChars); idx != -1 {
			return telegramBreakResult{index: idx}
		}
	}
	return telegramBreakResult{index: -1}
}

func (chunker *telegramDraftChunker) pickBreakIndex(buffer []rune, minCharsOverride int) telegramBreakResult {
	minChars := chunker.minChars
	if minCharsOverride > 0 {
		minChars = minCharsOverride
	}
	minChars = max(1, minChars)
	maxChars := max(minChars, chunker.maxChars)
	if len(buffer) < minChars {
		return telegramBreakResult{index: -1}
	}
	windowEnd := len(buffer)
	if windowEnd > maxChars {
		windowEnd = maxChars
	}
	window := buffer[:windowEnd]
	fenceSpans := parseTelegramFenceSpans(buffer)
	preference := chunker.breakPreference
	if preference == "paragraph" {
		if idx := findSafeParagraphBreakIndex(window, fenceSpans, minChars, true); idx != -1 {
			return telegramBreakResult{index: idx}
		}
	}
	if preference == "paragraph" || preference == "newline" {
		if idx := findSafeNewlineBreakIndex(window, fenceSpans, minChars, true); idx != -1 {
			return telegramBreakResult{index: idx}
		}
	}
	if preference != "newline" {
		if idx := findSafeSentenceBreakIndex(window, fenceSpans, minChars); idx != -1 {
			return telegramBreakResult{index: idx}
		}
	}
	if preference == "newline" && len(buffer) < maxChars {
		return telegramBreakResult{index: -1}
	}
	for i := len(window) - 1; i >= minChars; i-- {
		if unicode.IsSpace(window[i]) && isSafeTelegramFenceBreak(fenceSpans, i) {
			return telegramBreakResult{index: i}
		}
	}
	if len(buffer) >= maxChars {
		if isSafeTelegramFenceBreak(fenceSpans, maxChars) {
			return telegramBreakResult{index: maxChars}
		}
		if fence := findTelegramFenceSpanAt(fenceSpans, maxChars); fence != nil {
			return telegramBreakResult{
				index: maxChars,
				fenceSplit: &telegramFenceSplit{
					closeFenceLine:  fence.indent + fence.marker,
					reopenFenceLine: fence.openLine,
				},
			}
		}
		return telegramBreakResult{index: maxChars}
	}
	return telegramBreakResult{index: -1}
}

func parseTelegramFenceSpans(buffer []rune) []telegramFenceSpan {
	spans := make([]telegramFenceSpan, 0)
	var open *struct {
		start      int
		markerChar rune
		markerLen  int
		openLine   string
		marker     string
		indent     string
	}
	offset := 0
	for offset <= len(buffer) {
		lineEnd := offset
		for lineEnd < len(buffer) && buffer[lineEnd] != '\n' {
			lineEnd++
		}
		line := string(buffer[offset:lineEnd])
		indentLen := 0
		for indentLen < len(line) && indentLen < 3 && line[indentLen] == ' ' {
			indentLen++
		}
		rest := line[indentLen:]
		if len(rest) >= 3 {
			markerChar := rune(rest[0])
			if markerChar == '`' || markerChar == '~' {
				count := 0
				for count < len(rest) && rest[count] == byte(markerChar) {
					count++
				}
				if count >= 3 {
					marker := rest[:count]
					indent := rest[:0]
					if indentLen > 0 {
						indent = line[:indentLen]
					}
					if open == nil {
						open = &struct {
							start      int
							markerChar rune
							markerLen  int
							openLine   string
							marker     string
							indent     string
						}{
							start:      offset,
							markerChar: markerChar,
							markerLen:  count,
							openLine:   line,
							marker:     marker,
							indent:     indent,
						}
					} else if open.markerChar == markerChar && count >= open.markerLen {
						spans = append(spans, telegramFenceSpan{
							start:    open.start,
							end:      lineEnd,
							openLine: open.openLine,
							marker:   open.marker,
							indent:   open.indent,
						})
						open = nil
					}
				}
			}
		}
		if lineEnd >= len(buffer) {
			break
		}
		offset = lineEnd + 1
	}
	if open != nil {
		spans = append(spans, telegramFenceSpan{
			start:    open.start,
			end:      len(buffer),
			openLine: open.openLine,
			marker:   open.marker,
			indent:   open.indent,
		})
	}
	return spans
}

func findTelegramFenceSpanAt(spans []telegramFenceSpan, index int) *telegramFenceSpan {
	for i := range spans {
		span := &spans[i]
		if index > span.start && index < span.end {
			return span
		}
	}
	return nil
}

func isSafeTelegramFenceBreak(spans []telegramFenceSpan, index int) bool {
	return findTelegramFenceSpanAt(spans, index) == nil
}

func findSafeSentenceBreakIndex(buffer []rune, spans []telegramFenceSpan, minChars int) int {
	idx := -1
	for i, r := range buffer {
		if r != '.' && r != '!' && r != '?' {
			continue
		}
		next := i + 1
		if next < len(buffer) && !unicode.IsSpace(buffer[next]) {
			continue
		}
		if next < minChars {
			continue
		}
		if isSafeTelegramFenceBreak(spans, next) {
			idx = next
		}
	}
	if idx >= minChars {
		return idx
	}
	return -1
}

func findSafeParagraphBreakIndex(buffer []rune, spans []telegramFenceSpan, minChars int, reverse bool) int {
	idx := -1
	for i := 0; i < len(buffer)-1; i++ {
		if buffer[i] != '\n' {
			continue
		}
		j := i + 1
		for j < len(buffer) && (buffer[j] == ' ' || buffer[j] == '\t') {
			j++
		}
		if j < len(buffer) && buffer[j] == '\n' {
			if i < minChars {
				continue
			}
			if !isSafeTelegramFenceBreak(spans, i) {
				continue
			}
			if !reverse {
				return i
			}
			idx = i
		}
	}
	return idx
}

func findSafeNewlineBreakIndex(buffer []rune, spans []telegramFenceSpan, minChars int, reverse bool) int {
	idx := -1
	for i := 0; i < len(buffer); i++ {
		if buffer[i] != '\n' {
			continue
		}
		if i < minChars {
			continue
		}
		if !isSafeTelegramFenceBreak(spans, i) {
			continue
		}
		if !reverse {
			return i
		}
		idx = i
	}
	return idx
}

func stripLeadingNewlines(value []rune) []rune {
	idx := 0
	for idx < len(value) && value[idx] == '\n' {
		idx++
	}
	if idx == 0 {
		return value
	}
	return value[idx:]
}

func stripLeadingWhitespace(value []rune) []rune {
	idx := 0
	for idx < len(value) && unicode.IsSpace(value[idx]) {
		idx++
	}
	if idx == 0 {
		return value
	}
	return value[idx:]
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

type telegramDraftStreamer struct {
	stream      *telegramDraftStream
	mode        string
	chunker     *telegramDraftChunker
	draftText   strings.Builder
	lastPreview string
	stopped     bool
}

func newTelegramDraftStreamer(stream *telegramDraftStream, mode string, chunkCfg TelegramDraftChunkConfig) *telegramDraftStreamer {
	resolvedMode := resolveTelegramStreamMode(mode)
	streamer := &telegramDraftStreamer{
		stream: stream,
		mode:   resolvedMode,
	}
	if resolvedMode == "block" {
		streamer.chunker = newTelegramDraftChunker(chunkCfg)
	}
	return streamer
}

func (streamer *telegramDraftStreamer) HandleEvent(event runtimedto.RuntimeStreamEvent) {
	if streamer == nil || streamer.stopped {
		return
	}
	switch event.Type {
	case runtimedto.RuntimeStreamEventDelta:
		streamer.handleDelta(event.Delta)
	case runtimedto.RuntimeStreamEventEnd:
		streamer.Flush()
	case runtimedto.RuntimeStreamEventError:
		streamer.stopped = true
	}
}

func (streamer *telegramDraftStreamer) handleDelta(delta string) {
	if streamer == nil || streamer.stopped || delta == "" {
		return
	}
	if streamer.mode == "block" && streamer.chunker != nil {
		chunks := streamer.chunker.Append(delta)
		for _, chunk := range chunks {
			streamer.draftText.WriteString(chunk)
			streamer.updatePreview(streamer.draftText.String())
		}
		return
	}
	streamer.draftText.WriteString(delta)
	streamer.updatePreview(streamer.draftText.String())
}

func (streamer *telegramDraftStreamer) Flush() {
	if streamer == nil || streamer.stopped {
		return
	}
	if streamer.mode == "block" && streamer.chunker != nil {
		chunks := streamer.chunker.Flush()
		for _, chunk := range chunks {
			streamer.draftText.WriteString(chunk)
		}
	}
	streamer.updatePreview(streamer.draftText.String())
	if streamer.stream != nil {
		streamer.stream.Flush()
	}
}

func (streamer *telegramDraftStreamer) updatePreview(text string) {
	if streamer == nil || streamer.stream == nil {
		return
	}
	if streamer.lastPreview != "" && len(text) < len(streamer.lastPreview) && strings.HasPrefix(streamer.lastPreview, text) {
		return
	}
	streamer.lastPreview = text
	streamer.stream.Update(text)
}
