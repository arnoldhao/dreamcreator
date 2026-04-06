package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	runtimedto "dreamcreator/internal/application/gateway/runtime/dto"
	"dreamcreator/internal/application/library/dto"
	"dreamcreator/internal/domain/library"
)

const (
	subtitleTranslateChunkCueLimit  = 36
	subtitleTranslateChunkCharLimit = 2200
	subtitleTranslateMaxRetries     = 2
	subtitleTranslateChunkTimeout   = 90 * time.Second
	subtitleChunkMaxTokensFloor     = 2048
	subtitleChunkMaxTokensCeiling   = 6144
	subtitleChunkRetryTokenStep     = 1024
)

type subtitleTranslateChunk struct {
	Sequence int
	Cues     []dto.SubtitleCue
}

type subtitleTranslateResponse struct {
	Items []subtitleTranslateItem `json:"items"`
}

type subtitleTranslateItem struct {
	Index int    `json:"index"`
	Text  string `json:"text"`
}

type subtitleTranslateOutput struct {
	DocumentID            string                  `json:"documentId,omitempty"`
	FileID                string                  `json:"fileId,omitempty"`
	Status                string                  `json:"status"`
	Mode                  string                  `json:"mode,omitempty"`
	Assistant             string                  `json:"assistantId,omitempty"`
	Target                string                  `json:"targetLanguage"`
	ChunkCount            int                     `json:"chunkCount"`
	CueCount              int                     `json:"cueCount"`
	SourceHash            string                  `json:"sourceHash,omitempty"`
	GlossaryProfileIDs    []string                `json:"glossaryProfileIds,omitempty"`
	ReferenceTrackFileIDs []string                `json:"referenceTrackFileIds,omitempty"`
	PromptProfileIDs      []string                `json:"promptProfileIds,omitempty"`
	InlinePromptHash      string                  `json:"inlinePromptHash,omitempty"`
	CompletedChunkCount   int                     `json:"completedChunkCount,omitempty"`
	FailedChunkCount      int                     `json:"failedChunkCount,omitempty"`
	RequestHash           string                  `json:"requestHash,omitempty"`
	PromptHash            string                  `json:"promptHash,omitempty"`
	ResumedFromCheckpoint bool                    `json:"resumedFromCheckpoint,omitempty"`
	Usage                 runtimedto.RuntimeUsage `json:"usage,omitempty"`
}

type subtitleTranslateConstraints struct {
	GlossaryProfiles []library.GlossaryProfile
	PromptProfiles   []library.PromptProfile
	ReferenceTracks  []subtitleTranslateReferenceTrack
	InlinePrompt     string
}

type subtitleTaskRuntimeSettings struct {
	StructuredOutputMode string
	ThinkingMode         string
	MaxTokensFloor       int
	MaxTokensCeiling     int
	RetryTokenStep       int
}

type subtitleTranslateReferenceTrack struct {
	FileID      string
	DisplayName string
	CuesByIndex map[int]string
}

func (service *LibraryService) runSubtitleTranslateOperation(ctx context.Context, operation library.LibraryOperation, request dto.SubtitleTranslateRequest) {
	request = normalizeSubtitleTranslateRequest(request)
	if err := validateSubtitleTranslateSourceAndTarget(request); err != nil {
		service.failSubtitleTranslateOperation(ctx, operation, err, runtimedto.RuntimeUsage{}, subtitleTaskRunState{})
		return
	}
	runCtx, cancel := context.WithCancel(ctx)
	service.registerOperationRun(operation.ID, cancel)
	defer func() {
		cancel()
		service.unregisterOperationRun(operation.ID)
	}()

	sourceFile, document, err := service.resolveSubtitleFileAndDocument(ctx, request.FileID, request.DocumentID, request.Path)
	if err != nil {
		service.failSubtitleTranslateOperation(ctx, operation, err, runtimedto.RuntimeUsage{}, subtitleTaskRunState{})
		return
	}

	sourceContent := strings.TrimSpace(document.WorkingContent)
	if sourceContent == "" {
		sourceContent = strings.TrimSpace(document.OriginalContent)
	}
	if sourceContent == "" {
		service.failSubtitleTranslateOperation(ctx, operation, errors.New("subtitle content is empty"), runtimedto.RuntimeUsage{}, subtitleTaskRunState{})
		return
	}

	targetLanguage := request.TargetLanguage
	outputFormat := detectSubtitleFormat(request.OutputFormat, sourceFile.Storage.LocalPath, document.Format)
	sourceDocument := parseSubtitleDocument(sourceContent, detectSubtitleFormat(document.Format, sourceFile.Storage.LocalPath, document.Format))
	if len(sourceDocument.Cues) == 0 {
		service.failSubtitleTranslateOperation(ctx, operation, errors.New("subtitle document has no cues"), runtimedto.RuntimeUsage{}, subtitleTaskRunState{})
		return
	}
	moduleConfig, err := service.getModuleConfig(ctx)
	if err != nil {
		service.failSubtitleTranslateOperation(ctx, operation, err, runtimedto.RuntimeUsage{}, subtitleTaskRunState{})
		return
	}
	constraints, err := service.resolveSubtitleTranslateConstraints(ctx, request, moduleConfig, sourceFile.ID)
	if err != nil {
		service.failSubtitleTranslateOperation(ctx, operation, err, runtimedto.RuntimeUsage{}, subtitleTaskRunState{})
		return
	}
	runtimeConfig := resolveSubtitleTaskRuntimeSettings(moduleConfig.TaskRuntime.Translate)

	chunks := buildSubtitleTranslateChunks(sourceDocument.Cues)
	sourceHash := hashSubtitleSource(sourceContent)
	requestHash := hashSubtitleTranslateRequest(request)
	promptHash := hashSubtitleTranslatePrompt(request, constraints)
	checkpointState, err := service.loadSubtitleChunkCheckpointState(ctx, operation.ID, chunks, requestHash, promptHash)
	if err != nil {
		service.failSubtitleTranslateOperation(ctx, operation, err, runtimedto.RuntimeUsage{}, subtitleTaskRunState{
			RequestHash: requestHash,
			PromptHash:  promptHash,
		})
		return
	}
	runState := subtitleTaskRunState{
		CompletedChunkCount:   checkpointState.CompletedChunkCount,
		RequestHash:           requestHash,
		PromptHash:            promptHash,
		ResumedFromCheckpoint: checkpointState.ResumedFromCheckpoint,
	}
	operation.DisplayName = buildSubtitleTranslateOutputName(sourceFile.Name, targetLanguage)
	operation.Status = library.OperationStatusRunning
	now := service.now()
	operation.StartedAt = &now
	operation.Progress = buildOperationProgress(
		now,
		progressText("library.progress.preparing"),
		runState.CompletedChunkCount,
		len(chunks),
		progressText("library.progressDetail.preparingSubtitleTranslation"),
	)
	totalUsage := checkpointState.Usage
	operation.OutputJSON = marshalJSON(buildSubtitleTranslateOutput(
		request,
		"running",
		len(chunks),
		len(sourceDocument.Cues),
		sourceHash,
		"",
		"",
		totalUsage,
		runState,
	))
	if err := service.saveAndPublishOperation(ctx, operation); err != nil {
		return
	}

	translatedCues := make([]dto.SubtitleCue, 0, len(sourceDocument.Cues))
	for _, chunk := range chunks {
		if restoredItems, ok := checkpointState.ItemsByChunkIndex[chunk.Sequence]; ok {
			translatedCues = appendChunkItemsAsCues(translatedCues, chunk, restoredItems)
			continue
		}
		if runCtx.Err() != nil || service.isSubtitleOperationCanceled(ctx, operation.ID) {
			_, _ = service.markSubtitleOperationCanceled(ctx, operation.ID)
			return
		}

		startedAt := service.now()
		if err := service.saveSubtitleChunkCheckpoint(
			ctx,
			operation,
			chunk,
			library.OperationChunkStatusRunning,
			requestHash,
			promptHash,
			nil,
			runtimedto.RuntimeUsage{},
			0,
			"",
			&startedAt,
			nil,
		); err != nil {
			service.failSubtitleTranslateOperation(ctx, operation, err, totalUsage, runState)
			return
		}

		chunkCtx, chunkCancel := context.WithTimeout(runCtx, subtitleTranslateChunkTimeout)
		translatedItems, usage, attemptsUsed, err := service.translateSubtitleChunk(chunkCtx, request, chunk, constraints, runtimeConfig)
		chunkCancel()
		totalUsage = addRuntimeUsage(totalUsage, usage)
		finishedAt := service.now()
		if err != nil {
			checkpointStatus := library.OperationChunkStatusFailed
			if errors.Is(err, context.Canceled) || runCtx.Err() == context.Canceled || service.isSubtitleOperationCanceled(ctx, operation.ID) {
				checkpointStatus = library.OperationChunkStatusCanceled
			}
			if saveErr := service.saveSubtitleChunkCheckpoint(
				ctx,
				operation,
				chunk,
				checkpointStatus,
				requestHash,
				promptHash,
				nil,
				usage,
				countUsedRetries(attemptsUsed),
				err.Error(),
				&startedAt,
				&finishedAt,
			); saveErr != nil {
				service.failSubtitleTranslateOperation(ctx, operation, saveErr, totalUsage, runState)
				return
			}
			if checkpointStatus == library.OperationChunkStatusCanceled {
				_, _ = service.markSubtitleOperationCanceled(ctx, operation.ID)
				return
			}
			runState.FailedChunkCount++
			service.failSubtitleTranslateOperation(
				ctx,
				operation,
				fmt.Errorf("chunk %d/%d failed: %w", chunk.Sequence, len(chunks), err),
				totalUsage,
				runState,
			)
			return
		}

		if err := service.saveSubtitleChunkCheckpoint(
			ctx,
			operation,
			chunk,
			library.OperationChunkStatusSucceeded,
			requestHash,
			promptHash,
			translatedItems,
			usage,
			countUsedRetries(attemptsUsed),
			"",
			&startedAt,
			&finishedAt,
		); err != nil {
			service.failSubtitleTranslateOperation(ctx, operation, err, totalUsage, runState)
			return
		}

		runState.CompletedChunkCount++
		translatedCues = appendChunkItemsAsCues(translatedCues, chunk, translatedItems)
		if runCtx.Err() != nil || service.isSubtitleOperationCanceled(ctx, operation.ID) {
			_, _ = service.markSubtitleOperationCanceled(ctx, operation.ID)
			return
		}

		progressTime := service.now()
		operation.Progress = buildOperationProgress(
			progressTime,
			progressText("library.progress.translating"),
			runState.CompletedChunkCount,
			len(chunks),
			progressTextTemplate("library.progressDetail.translatedChunk", map[string]string{
				"current": fmt.Sprintf("%d", runState.CompletedChunkCount),
				"total":   fmt.Sprintf("%d", len(chunks)),
			}),
		)
		operation.OutputJSON = marshalJSON(buildSubtitleTranslateOutput(
			request,
			"running",
			len(chunks),
			len(sourceDocument.Cues),
			sourceHash,
			"",
			"",
			totalUsage,
			runState,
		))
		if err := service.saveAndPublishOperation(ctx, operation); err != nil {
			return
		}
	}

	if runCtx.Err() != nil || service.isSubtitleOperationCanceled(ctx, operation.ID) {
		_, _ = service.markSubtitleOperationCanceled(ctx, operation.ID)
		return
	}

	translatedDocument := dto.SubtitleDocument{
		Format:   outputFormat,
		Cues:     translatedCues,
		Metadata: map[string]any{"targetLanguage": targetLanguage},
	}
	translatedContent := renderSubtitleContent(translatedDocument, outputFormat)
	finishedAt := service.now()
	translatedFile, history, err := service.createDerivedSubtitleFile(ctx, derivedSubtitleParams{
		LibraryID:      sourceFile.LibraryID,
		RootFileID:     rootFileID(sourceFile),
		Name:           buildSubtitleTranslateOutputName(sourceFile.Name, targetLanguage),
		OperationID:    operation.ID,
		OperationKind:  "subtitle_translate",
		Format:         outputFormat,
		SourceMedia:    sourceFile.Media,
		OriginalSource: translatedContent,
		LocalPath:      strings.TrimSpace(sourceFile.Storage.LocalPath),
		OccurredAt:     finishedAt,
		HistorySource:  library.HistoryRecordSource{Kind: resolveHistorySourceKind(request.Source), RunID: strings.TrimSpace(request.RunID)},
	})
	if err != nil {
		service.failSubtitleTranslateOperation(ctx, operation, err, totalUsage, runState)
		return
	}

	translatedFile.LatestOperationID = operation.ID
	translatedFile.UpdatedAt = finishedAt
	if err := service.files.Save(ctx, translatedFile); err != nil {
		service.failSubtitleTranslateOperation(ctx, operation, err, totalUsage, runState)
		return
	}
	sourceFile.LatestOperationID = operation.ID
	sourceFile.UpdatedAt = finishedAt
	if err := service.files.Save(ctx, sourceFile); err != nil {
		service.failSubtitleTranslateOperation(ctx, operation, err, totalUsage, runState)
		return
	}

	operation.Status = library.OperationStatusSucceeded
	operation.FinishedAt = &finishedAt
	operation.OutputFiles = []library.OperationOutputFile{{
		FileID:    translatedFile.ID,
		Kind:      string(translatedFile.Kind),
		Format:    mediaFormatFromFile(translatedFile),
		SizeBytes: mediaSizeFromFile(translatedFile),
		IsPrimary: true,
		Deleted:   translatedFile.State.Deleted,
	}}
	operation.Metrics = buildOperationMetricsForOperation([]library.LibraryFile{translatedFile}, operation.StartedAt, &finishedAt)
	operation.Progress = buildOperationProgress(
		finishedAt,
		progressText("library.status.succeeded"),
		len(chunks),
		len(chunks),
		progressText("library.progressDetail.subtitleTranslationCompleted"),
	)
	operation.OutputJSON = marshalJSON(buildSubtitleTranslateOutput(
		request,
		"completed",
		len(chunks),
		len(translatedCues),
		sourceHash,
		translatedFile.ID,
		translatedFile.Storage.DocumentID,
		totalUsage,
		runState,
	))
	if err := service.operations.Save(ctx, operation); err != nil {
		service.failSubtitleTranslateOperation(ctx, operation, err, totalUsage, runState)
		return
	}

	history.Refs.OperationID = operation.ID
	history.Action = "subtitle_translate"
	history.Status = string(operation.Status)
	history.OperationMeta = &library.OperationRecordMeta{Kind: "subtitle_translate"}
	history.Files = operation.OutputFiles
	history.Metrics = operation.Metrics
	if err := service.histories.Save(ctx, history); err != nil {
		service.failSubtitleTranslateOperation(ctx, operation, err, totalUsage, runState)
		return
	}
	if err := service.touchLibrary(ctx, sourceFile.LibraryID, finishedAt); err != nil {
		service.failSubtitleTranslateOperation(ctx, operation, err, totalUsage, runState)
		return
	}

	service.publishOperationUpdate(toOperationDTO(operation))
	service.publishHistoryUpdate(toHistoryDTO(history))
	service.publishFileUpdate(service.mustBuildFileDTO(ctx, sourceFile))
	service.publishFileUpdate(service.mustBuildFileDTO(ctx, translatedFile))
}

func (service *LibraryService) translateSubtitleChunk(
	ctx context.Context,
	request dto.SubtitleTranslateRequest,
	chunk subtitleTranslateChunk,
	constraints subtitleTranslateConstraints,
	runtimeConfig subtitleTaskRuntimeSettings,
) ([]subtitleTranslateItem, runtimedto.RuntimeUsage, int, error) {
	var (
		lastErr      error
		totalUsage   runtimedto.RuntimeUsage
		repairPrompt string
		attemptsUsed int
	)

	for attempt := 0; attempt < subtitleTranslateMaxRetries; attempt++ {
		attemptsUsed = attempt + 1
		systemPrompt, userPrompt := buildSubtitleTranslatePrompts(request, chunk, constraints, repairPrompt)
		result, err := service.runtime.RunOneShot(ctx, runtimedto.RuntimeRunRequest{
			AssistantID: strings.TrimSpace(request.AssistantID),
			PromptMode:  "none",
			Input: runtimedto.RuntimeInput{
				Messages: []runtimedto.Message{{
					Role:    "user",
					Content: userPrompt,
				}},
			},
			Thinking: runtimedto.ThinkingConfig{Mode: runtimeConfig.ThinkingMode},
			Tools: runtimedto.ToolExecutionConfig{
				Mode: "disabled",
			},
			Metadata: map[string]any{
				"channel":           "library",
				"usageSource":       "one-shot",
				"useQueue":          true,
				"runLane":           "subagent",
				"subagent":          true,
				"temperature":       0.2,
				"maxTokens":         estimateSubtitleChunkMaxTokensWithRuntime(chunk, runtimeConfig, attempt),
				"extraSystemPrompt": systemPrompt,
				"structuredOutput":  buildSubtitleStructuredOutputMetadata("library_subtitle_translate_chunk", runtimeConfig),
			},
		})
		if err != nil {
			if errors.Is(err, context.Canceled) || ctx.Err() == context.Canceled {
				return nil, totalUsage, attemptsUsed, context.Canceled
			}
			lastErr = err
			continue
		}
		totalUsage = addRuntimeUsage(totalUsage, result.Usage)
		items, parseErr := parseSubtitleTranslateItems(result.AssistantMessage.Content, chunk.Cues)
		if parseErr != nil && isTokenLimitFinishReason(result.FinishReason) {
			parseErr = fmt.Errorf("model output truncated at max token limit: %w", parseErr)
		}
		if parseErr == nil {
			return items, totalUsage, attemptsUsed, nil
		}
		lastErr = parseErr
		repairPrompt = fmt.Sprintf("The previous output was invalid. Fix it and return valid JSON only. Validation error: %s\nPrevious output:\n%s", parseErr.Error(), result.AssistantMessage.Content)
	}

	if lastErr == nil {
		lastErr = errors.New("subtitle translation failed")
	}
	return nil, totalUsage, attemptsUsed, lastErr
}

func buildSubtitleTranslateChunks(cues []dto.SubtitleCue) []subtitleTranslateChunk {
	if len(cues) == 0 {
		return nil
	}
	chunks := make([]subtitleTranslateChunk, 0, int(math.Ceil(float64(len(cues))/float64(subtitleTranslateChunkCueLimit))))
	buffer := make([]dto.SubtitleCue, 0, subtitleTranslateChunkCueLimit)
	bufferChars := 0
	appendChunk := func() {
		if len(buffer) == 0 {
			return
		}
		chunks = append(chunks, subtitleTranslateChunk{
			Sequence: len(chunks) + 1,
			Cues:     append([]dto.SubtitleCue(nil), buffer...),
		})
		buffer = buffer[:0]
		bufferChars = 0
	}
	for _, cue := range cues {
		textChars := len([]rune(strings.TrimSpace(cue.Text)))
		if len(buffer) > 0 && (len(buffer) >= subtitleTranslateChunkCueLimit || bufferChars+textChars > subtitleTranslateChunkCharLimit) {
			appendChunk()
		}
		buffer = append(buffer, cue)
		bufferChars += textChars
	}
	appendChunk()
	return chunks
}

func buildSubtitleTranslatePrompts(
	request dto.SubtitleTranslateRequest,
	chunk subtitleTranslateChunk,
	constraints subtitleTranslateConstraints,
	repairPrompt string,
) (string, string) {
	targetLanguage := strings.TrimSpace(request.TargetLanguage)
	var systemBuilder strings.Builder
	systemBuilder.WriteString("You translate subtitle cues into the requested target language.\n")
	systemBuilder.WriteString("Return only valid JSON with this shape: {\"items\":[{\"index\":1,\"text\":\"...\"}]}\n")
	systemBuilder.WriteString("Rules:\n")
	systemBuilder.WriteString("- Keep the same number of items, the same indexes, and the same order.\n")
	systemBuilder.WriteString("- Translate only the cue text. Do not merge, split, drop, or invent cues.\n")
	systemBuilder.WriteString("- Preserve inline formatting markers, speaker labels, and escaped line breaks when possible.\n")
	systemBuilder.WriteString("- Keep subtitle wording concise and natural for on-screen reading.\n")
	systemBuilder.WriteString("- Do not output markdown, prose, or explanations.\n")
	if repairPrompt != "" {
		systemBuilder.WriteString("- Repair the previous invalid output and still obey every rule above.\n")
	}
	if glossaryText := renderGlossaryPrompt(chunk, constraints.GlossaryProfiles); glossaryText != "" {
		systemBuilder.WriteString("\nStructured glossary constraints:\n")
		systemBuilder.WriteString(glossaryText)
		systemBuilder.WriteString("\n")
	}
	if referenceText := renderReferenceGuidancePrompt(len(constraints.ReferenceTracks) > 0); referenceText != "" {
		systemBuilder.WriteString("\nReference guidance:\n")
		systemBuilder.WriteString(referenceText)
		systemBuilder.WriteString("\n")
	}
	if promptText := renderPromptProfilesPrompt(constraints.PromptProfiles); promptText != "" {
		systemBuilder.WriteString("\nReusable prompt profiles:\n")
		systemBuilder.WriteString(promptText)
		systemBuilder.WriteString("\n")
	}
	if inlinePrompt := strings.TrimSpace(constraints.InlinePrompt); inlinePrompt != "" {
		systemBuilder.WriteString("\nInline task note:\n")
		systemBuilder.WriteString(inlinePrompt)
		systemBuilder.WriteString("\n")
	}

	payload := make([]map[string]any, 0, len(chunk.Cues))
	for _, cue := range chunk.Cues {
		payload = append(payload, map[string]any{
			"index": cue.Index,
			"text":  cue.Text,
		})
	}

	var userBuilder strings.Builder
	userBuilder.WriteString(fmt.Sprintf("Target language: %s\n", targetLanguage))
	userBuilder.WriteString("Translate the following subtitle cues.\n")
	if repairPrompt != "" {
		userBuilder.WriteString(repairPrompt)
		userBuilder.WriteString("\n")
	}
	if referencePayload := buildReferenceTrackPromptPayload(chunk, constraints.ReferenceTracks); len(referencePayload) > 0 {
		userBuilder.WriteString("Reference subtitle cues (for guidance only; do not copy sentences verbatim):\n")
		userBuilder.WriteString(marshalJSON(map[string]any{"tracks": referencePayload}))
		userBuilder.WriteString("\n")
	}
	userBuilder.WriteString("Input JSON:\n")
	userBuilder.WriteString(marshalJSON(map[string]any{"items": payload}))
	return strings.TrimSpace(systemBuilder.String()), strings.TrimSpace(userBuilder.String())
}

func renderGlossaryPrompt(chunk subtitleTranslateChunk, profiles []library.GlossaryProfile) string {
	if len(profiles) == 0 {
		return ""
	}
	chunkText := strings.ToLower(joinChunkCueText(chunk.Cues))
	lines := make([]string, 0)
	for _, profile := range profiles {
		matchedTerms := make([]library.GlossaryTerm, 0)
		for _, term := range profile.Terms {
			source := strings.TrimSpace(term.Source)
			target := strings.TrimSpace(term.Target)
			if source == "" || target == "" {
				continue
			}
			if chunkText == "" || strings.Contains(chunkText, strings.ToLower(source)) {
				matchedTerms = append(matchedTerms, term)
			}
		}
		if len(matchedTerms) == 0 {
			continue
		}
		lines = append(lines, fmt.Sprintf("- Profile %q", profileDisplayName(profile.Name, profile.ID)))
		for _, term := range matchedTerms {
			line := fmt.Sprintf("  - %q -> %q", strings.TrimSpace(term.Source), strings.TrimSpace(term.Target))
			if note := strings.TrimSpace(term.Note); note != "" {
				line += fmt.Sprintf(" (%s)", note)
			}
			lines = append(lines, line)
		}
	}
	return strings.Join(lines, "\n")
}

func renderReferenceGuidancePrompt(hasReferenceTracks bool) string {
	if !hasReferenceTracks {
		return ""
	}
	return strings.Join([]string{
		"- Reference cues are optional guidance for terminology and tone only.",
		"- Absorb useful terminology from reference cues without copying full sentences.",
	}, "\n")
}

func renderPromptProfilesPrompt(profiles []library.PromptProfile) string {
	if len(profiles) == 0 {
		return ""
	}
	lines := make([]string, 0, len(profiles))
	for _, profile := range profiles {
		prompt := strings.TrimSpace(profile.Prompt)
		if prompt == "" {
			continue
		}
		lines = append(lines, fmt.Sprintf("- %s: %s", profileDisplayName(profile.Name, profile.ID), prompt))
	}
	return strings.Join(lines, "\n")
}

func buildReferenceTrackPromptPayload(
	chunk subtitleTranslateChunk,
	tracks []subtitleTranslateReferenceTrack,
) []map[string]any {
	if len(tracks) == 0 {
		return nil
	}
	result := make([]map[string]any, 0, len(tracks))
	for _, track := range tracks {
		items := make([]map[string]any, 0, len(chunk.Cues))
		for _, cue := range chunk.Cues {
			text := strings.TrimSpace(track.CuesByIndex[cue.Index])
			if text == "" {
				continue
			}
			items = append(items, map[string]any{
				"index": cue.Index,
				"text":  text,
			})
		}
		if len(items) == 0 {
			continue
		}
		result = append(result, map[string]any{
			"fileId": track.FileID,
			"name":   track.DisplayName,
			"items":  items,
		})
	}
	return result
}

func joinChunkCueText(cues []dto.SubtitleCue) string {
	values := make([]string, 0, len(cues))
	for _, cue := range cues {
		text := strings.TrimSpace(cue.Text)
		if text != "" {
			values = append(values, text)
		}
	}
	return strings.Join(values, "\n")
}

func parseSubtitleTranslateItems(raw string, sourceCues []dto.SubtitleCue) ([]subtitleTranslateItem, error) {
	content := strings.TrimSpace(raw)
	if content == "" {
		return nil, errors.New("model returned empty output")
	}
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)
	if start := strings.Index(content, "{"); start >= 0 {
		if end := strings.LastIndex(content, "}"); end > start {
			content = content[start : end+1]
		}
	}

	response := subtitleTranslateResponse{}
	if err := json.Unmarshal([]byte(content), &response); err != nil {
		return nil, fmt.Errorf("invalid json output: %w", err)
	}
	if len(response.Items) != len(sourceCues) {
		return nil, fmt.Errorf("expected %d translated items, got %d", len(sourceCues), len(response.Items))
	}

	items := make([]subtitleTranslateItem, 0, len(response.Items))
	for index, item := range response.Items {
		expectedIndex := sourceCues[index].Index
		if item.Index != expectedIndex {
			return nil, fmt.Errorf("expected index %d at position %d, got %d", expectedIndex, index, item.Index)
		}
		if strings.TrimSpace(item.Text) == "" && strings.TrimSpace(sourceCues[index].Text) != "" {
			return nil, fmt.Errorf("translated text for cue %d is empty", expectedIndex)
		}
		items = append(items, subtitleTranslateItem{
			Index: item.Index,
			Text:  strings.TrimSpace(item.Text),
		})
	}
	return items, nil
}

func (service *LibraryService) resolveSubtitleTranslateConstraints(
	ctx context.Context,
	request dto.SubtitleTranslateRequest,
	config library.ModuleConfig,
	sourceFileID string,
) (subtitleTranslateConstraints, error) {
	constraints := subtitleTranslateConstraints{
		InlinePrompt: strings.TrimSpace(request.InlinePrompt),
	}

	glossaryProfiles, err := pickGlossaryProfilesByID(config.LanguageAssets.GlossaryProfiles, request.GlossaryProfileIDs)
	if err != nil {
		return subtitleTranslateConstraints{}, err
	}
	constraints.GlossaryProfiles = glossaryProfiles

	promptProfiles, err := pickPromptProfilesByID(config.LanguageAssets.PromptProfiles, request.PromptProfileIDs)
	if err != nil {
		return subtitleTranslateConstraints{}, err
	}
	constraints.PromptProfiles = promptProfiles

	if len(request.ReferenceTrackFileIDs) == 0 {
		return constraints, nil
	}
	referenceTracks := make([]subtitleTranslateReferenceTrack, 0, len(request.ReferenceTrackFileIDs))
	for _, fileID := range request.ReferenceTrackFileIDs {
		if fileID == "" || fileID == sourceFileID {
			continue
		}
		track, err := service.resolveSubtitleTranslateReferenceTrack(ctx, fileID)
		if err != nil {
			return subtitleTranslateConstraints{}, err
		}
		referenceTracks = append(referenceTracks, track)
	}
	constraints.ReferenceTracks = referenceTracks
	return constraints, nil
}

func (service *LibraryService) resolveSubtitleTranslateReferenceTrack(
	ctx context.Context,
	fileID string,
) (subtitleTranslateReferenceTrack, error) {
	file, document, err := service.resolveSubtitleFileAndDocument(ctx, fileID, "", "")
	if err != nil {
		return subtitleTranslateReferenceTrack{}, err
	}
	content := strings.TrimSpace(document.WorkingContent)
	if content == "" {
		content = strings.TrimSpace(document.OriginalContent)
	}
	if content == "" {
		return subtitleTranslateReferenceTrack{}, fmt.Errorf("reference track %s has no subtitle content", fileID)
	}
	parsed := parseSubtitleDocument(content, detectSubtitleFormat(document.Format, file.Storage.LocalPath, document.Format))
	if len(parsed.Cues) == 0 {
		return subtitleTranslateReferenceTrack{}, fmt.Errorf("reference track %s has no cues", fileID)
	}
	cuesByIndex := make(map[int]string, len(parsed.Cues))
	for _, cue := range parsed.Cues {
		text := strings.TrimSpace(cue.Text)
		if text == "" {
			continue
		}
		cuesByIndex[cue.Index] = text
	}
	return subtitleTranslateReferenceTrack{
		FileID:      file.ID,
		DisplayName: strings.TrimSpace(file.Name),
		CuesByIndex: cuesByIndex,
	}, nil
}

func appendChunkItemsAsCues(
	target []dto.SubtitleCue,
	chunk subtitleTranslateChunk,
	items []subtitleTranslateItem,
) []dto.SubtitleCue {
	for itemIndex, translated := range items {
		sourceCue := chunk.Cues[itemIndex]
		target = append(target, dto.SubtitleCue{
			Index: sourceCue.Index,
			Start: sourceCue.Start,
			End:   sourceCue.End,
			Text:  translated.Text,
		})
	}
	return target
}

func countUsedRetries(attemptsUsed int) int {
	if attemptsUsed <= 1 {
		return 0
	}
	return attemptsUsed - 1
}

func (service *LibraryService) failSubtitleTranslateOperation(
	ctx context.Context,
	operation library.LibraryOperation,
	err error,
	usage runtimedto.RuntimeUsage,
	runState subtitleTaskRunState,
) {
	if service == nil || service.operations == nil {
		return
	}
	if errors.Is(err, context.Canceled) || service.isSubtitleOperationCanceled(ctx, operation.ID) {
		_, _ = service.markSubtitleOperationCanceled(ctx, operation.ID)
		return
	}
	currentOperation := operation
	if item, getErr := service.operations.Get(ctx, operation.ID); getErr == nil {
		currentOperation = item
		if currentOperation.Status == library.OperationStatusCanceled {
			_, _ = service.markSubtitleOperationCanceled(ctx, operation.ID)
			return
		}
	}
	now := service.now()
	currentOperation.Status = library.OperationStatusFailed
	currentOperation.ErrorCode = "subtitle_translate_failed"
	currentOperation.ErrorMessage = strings.TrimSpace(err.Error())
	currentOperation.FinishedAt = &now
	currentOperation.Progress = buildOperationProgress(
		now,
		progressText("library.status.failed"),
		runState.CompletedChunkCount,
		progressTotal(currentOperation.Progress),
		terminalProgressMessage(currentOperation.Kind, currentOperation.Status),
	)
	output := buildSubtitleTranslateFailedOutput(currentOperation.InputJSON, usage, runState)
	if existing, ok := parseSubtitleTranslateOutput(currentOperation.OutputJSON); ok {
		if output.ChunkCount == 0 {
			output.ChunkCount = existing.ChunkCount
		}
		if output.CueCount == 0 {
			output.CueCount = existing.CueCount
		}
		if output.SourceHash == "" {
			output.SourceHash = existing.SourceHash
		}
	}
	currentOperation.OutputJSON = marshalJSON(output)
	if err := service.operations.Save(ctx, currentOperation); err != nil {
		return
	}
	service.publishOperationUpdate(toOperationDTO(currentOperation))
}

func (service *LibraryService) saveAndPublishOperation(ctx context.Context, operation library.LibraryOperation) error {
	if err := service.operations.Save(ctx, operation); err != nil {
		return err
	}
	service.publishOperationUpdate(toOperationDTO(operation))
	service.trackCompletedOperation(ctx, operation)
	return nil
}

func buildSubtitleTranslateOutput(
	request dto.SubtitleTranslateRequest,
	status string,
	chunkCount int,
	cueCount int,
	sourceHash string,
	fileID string,
	documentID string,
	usage runtimedto.RuntimeUsage,
	runState subtitleTaskRunState,
) subtitleTranslateOutput {
	return subtitleTranslateOutput{
		DocumentID:            strings.TrimSpace(documentID),
		FileID:                strings.TrimSpace(fileID),
		Status:                strings.TrimSpace(status),
		Mode:                  normalizeSubtitleTranslateMode(request.Mode),
		Assistant:             strings.TrimSpace(request.AssistantID),
		Target:                strings.TrimSpace(request.TargetLanguage),
		ChunkCount:            chunkCount,
		CueCount:              cueCount,
		SourceHash:            strings.TrimSpace(sourceHash),
		GlossaryProfileIDs:    append([]string(nil), request.GlossaryProfileIDs...),
		ReferenceTrackFileIDs: append([]string(nil), request.ReferenceTrackFileIDs...),
		PromptProfileIDs:      append([]string(nil), request.PromptProfileIDs...),
		InlinePromptHash:      hashOptionalPrompt(request.InlinePrompt),
		CompletedChunkCount:   runState.CompletedChunkCount,
		FailedChunkCount:      runState.FailedChunkCount,
		RequestHash:           strings.TrimSpace(runState.RequestHash),
		PromptHash:            strings.TrimSpace(runState.PromptHash),
		ResumedFromCheckpoint: runState.ResumedFromCheckpoint,
		Usage:                 usage,
	}
}

func buildSubtitleTranslateFailedOutput(
	inputJSON string,
	usage runtimedto.RuntimeUsage,
	runState subtitleTaskRunState,
) subtitleTranslateOutput {
	request := extractSubtitleTranslateRequest(inputJSON)
	return buildSubtitleTranslateOutput(request, "failed", 0, 0, "", "", "", usage, runState)
}

func buildSubtitleTranslateOutputName(name string, targetLanguage string) string {
	base := strings.TrimSpace(name)
	if base == "" {
		base = "Subtitle"
	}
	target := strings.ToUpper(strings.TrimSpace(targetLanguage))
	if target == "" {
		return base
	}
	return fmt.Sprintf("%s (%s)", base, target)
}

func buildOperationProgress(now time.Time, stage string, current int, total int, message string) *library.OperationProgress {
	progress := &library.OperationProgress{
		Stage:     strings.TrimSpace(stage),
		Message:   strings.TrimSpace(message),
		UpdatedAt: now.UTC().Format(time.RFC3339),
	}
	if current > 0 {
		value := int64(current)
		progress.Current = &value
	}
	if total > 0 {
		value := int64(total)
		progress.Total = &value
		percent := int(math.Round((float64(current) / float64(total)) * 100))
		if percent < 0 {
			percent = 0
		}
		if percent > 100 {
			percent = 100
		}
		progress.Percent = &percent
	}
	return progress
}

func hashSubtitleSource(content string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(content)))
	return hex.EncodeToString(sum[:])
}

func hashOptionalPrompt(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(trimmed))
	return hex.EncodeToString(sum[:])
}

func estimateSubtitleChunkMaxTokens(chunk subtitleTranslateChunk, attempt int) int {
	return estimateSubtitleChunkMaxTokensWithRuntime(chunk, defaultSubtitleTaskRuntimeSettings(), attempt)
}

func estimateSubtitleChunkMaxTokensWithRuntime(chunk subtitleTranslateChunk, runtimeConfig subtitleTaskRuntimeSettings, attempt int) int {
	totalChars := 0
	for _, cue := range chunk.Cues {
		totalChars += len([]rune(strings.TrimSpace(cue.Text)))
	}
	maxTokens := totalChars*4 + len(chunk.Cues)*24 + 512
	if maxTokens < runtimeConfig.MaxTokensFloor {
		maxTokens = runtimeConfig.MaxTokensFloor
	}
	if attempt > 0 {
		maxTokens += attempt * runtimeConfig.RetryTokenStep
	}
	if maxTokens > runtimeConfig.MaxTokensCeiling {
		maxTokens = runtimeConfig.MaxTokensCeiling
	}
	return maxTokens
}

func defaultSubtitleTaskRuntimeSettings() subtitleTaskRuntimeSettings {
	return subtitleTaskRuntimeSettings{
		StructuredOutputMode: "auto",
		ThinkingMode:         "off",
		MaxTokensFloor:       subtitleChunkMaxTokensFloor,
		MaxTokensCeiling:     subtitleChunkMaxTokensCeiling,
		RetryTokenStep:       subtitleChunkRetryTokenStep,
	}
}

func resolveSubtitleTaskRuntimeSettings(config library.LanguageTaskRuntimeSettings) subtitleTaskRuntimeSettings {
	resolved := defaultSubtitleTaskRuntimeSettings()
	if mode := strings.TrimSpace(config.StructuredOutputMode); mode != "" {
		resolved.StructuredOutputMode = mode
	}
	if mode := strings.TrimSpace(config.ThinkingMode); mode != "" {
		resolved.ThinkingMode = mode
	}
	if config.MaxTokensFloor > 0 {
		resolved.MaxTokensFloor = config.MaxTokensFloor
	}
	if config.MaxTokensCeiling > 0 {
		resolved.MaxTokensCeiling = config.MaxTokensCeiling
	}
	if config.RetryTokenStep > 0 {
		resolved.RetryTokenStep = config.RetryTokenStep
	}
	if resolved.MaxTokensCeiling < resolved.MaxTokensFloor {
		resolved.MaxTokensCeiling = resolved.MaxTokensFloor
	}
	return resolved
}

func buildSubtitleStructuredOutputMetadata(schemaName string, runtimeConfig subtitleTaskRuntimeSettings) map[string]any {
	return map[string]any{
		"mode":   strings.TrimSpace(runtimeConfig.StructuredOutputMode),
		"name":   strings.TrimSpace(schemaName),
		"strict": true,
		"schema": map[string]any{
			"type":                 "object",
			"additionalProperties": false,
			"required":             []string{"items"},
			"properties": map[string]any{
				"items": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type":                 "object",
						"additionalProperties": false,
						"required":             []string{"index", "text"},
						"properties": map[string]any{
							"index": map[string]any{
								"type": "integer",
							},
							"text": map[string]any{
								"type": "string",
							},
						},
					},
				},
			},
		},
	}
}

func isTokenLimitFinishReason(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "length", "max_tokens":
		return true
	default:
		return false
	}
}

func addRuntimeUsage(current runtimedto.RuntimeUsage, next runtimedto.RuntimeUsage) runtimedto.RuntimeUsage {
	current.PromptTokens += next.PromptTokens
	current.CompletionTokens += next.CompletionTokens
	current.TotalTokens += next.TotalTokens
	current.ContextPromptTokens += next.ContextPromptTokens
	current.ContextTotalTokens += next.ContextTotalTokens
	if next.ContextWindowTokens > current.ContextWindowTokens {
		current.ContextWindowTokens = next.ContextWindowTokens
	}
	return current
}

func normalizeSubtitleTranslateRequest(request dto.SubtitleTranslateRequest) dto.SubtitleTranslateRequest {
	request.FileID = strings.TrimSpace(request.FileID)
	request.DocumentID = strings.TrimSpace(request.DocumentID)
	request.Path = strings.TrimSpace(request.Path)
	request.LibraryID = strings.TrimSpace(request.LibraryID)
	request.RootFileID = strings.TrimSpace(request.RootFileID)
	request.AssistantID = strings.TrimSpace(request.AssistantID)
	request.TargetLanguage = strings.TrimSpace(request.TargetLanguage)
	request.OutputFormat = strings.TrimSpace(request.OutputFormat)
	request.Mode = strings.TrimSpace(request.Mode)
	request.Source = strings.TrimSpace(request.Source)
	request.GlossaryProfileIDs = uniqueTrimmedStrings(request.GlossaryProfileIDs)
	request.ReferenceTrackFileIDs = uniqueTrimmedStrings(request.ReferenceTrackFileIDs)
	request.PromptProfileIDs = uniqueTrimmedStrings(request.PromptProfileIDs)
	request.InlinePrompt = strings.TrimSpace(request.InlinePrompt)
	request.SessionKey = strings.TrimSpace(request.SessionKey)
	request.RunID = strings.TrimSpace(request.RunID)
	return request
}

func validateSubtitleTranslateSourceAndTarget(request dto.SubtitleTranslateRequest) error {
	if request.FileID == "" && request.DocumentID == "" {
		return fmt.Errorf("source subtitle fileId or documentId is required")
	}
	if request.TargetLanguage == "" {
		return fmt.Errorf("targetLanguage is required")
	}
	return nil
}

func normalizeSubtitleTranslateMode(value string) string {
	mode := strings.ToLower(strings.TrimSpace(value))
	if mode == "" {
		return "translate"
	}
	return mode
}

func extractSubtitleTranslateRequest(inputJSON string) dto.SubtitleTranslateRequest {
	request := dto.SubtitleTranslateRequest{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(inputJSON)), &request); err != nil {
		return dto.SubtitleTranslateRequest{}
	}
	return normalizeSubtitleTranslateRequest(request)
}

func extractSubtitleTranslateTarget(inputJSON string) string {
	return extractSubtitleTranslateRequest(inputJSON).TargetLanguage
}

func extractSubtitleTranslateMode(inputJSON string) string {
	return extractSubtitleTranslateRequest(inputJSON).Mode
}

func pickGlossaryProfilesByID(values []library.GlossaryProfile, ids []string) ([]library.GlossaryProfile, error) {
	index := make(map[string]library.GlossaryProfile, len(values))
	for _, value := range values {
		index[strings.TrimSpace(value.ID)] = value
	}
	result := make([]library.GlossaryProfile, 0, len(ids))
	for _, id := range ids {
		profile, ok := index[id]
		if !ok {
			return nil, fmt.Errorf("glossary profile %q not found", id)
		}
		result = append(result, profile)
	}
	return result, nil
}

func pickPromptProfilesByID(values []library.PromptProfile, ids []string) ([]library.PromptProfile, error) {
	index := make(map[string]library.PromptProfile, len(values))
	for _, value := range values {
		index[strings.TrimSpace(value.ID)] = value
	}
	result := make([]library.PromptProfile, 0, len(ids))
	for _, id := range ids {
		profile, ok := index[id]
		if !ok {
			return nil, fmt.Errorf("prompt profile %q not found", id)
		}
		result = append(result, profile)
	}
	return result, nil
}

func uniqueTrimmedStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		normalized := strings.TrimSpace(value)
		if normalized == "" {
			continue
		}
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		result = append(result, normalized)
	}
	return result
}

func profileDisplayName(name string, id string) string {
	if trimmed := strings.TrimSpace(name); trimmed != "" {
		return trimmed
	}
	return strings.TrimSpace(id)
}
