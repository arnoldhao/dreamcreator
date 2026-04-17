package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	runtimedto "dreamcreator/internal/application/gateway/runtime/dto"
	"dreamcreator/internal/application/library/dto"
	"dreamcreator/internal/domain/library"
)

const subtitleProofreadChunkTimeout = 90 * time.Second

type subtitleProofreadOutput struct {
	DocumentID            string                  `json:"documentId,omitempty"`
	FileID                string                  `json:"fileId,omitempty"`
	ReviewSessionID       string                  `json:"reviewSessionId,omitempty"`
	SourceRevisionID      string                  `json:"sourceRevisionId,omitempty"`
	CandidateRevisionID   string                  `json:"candidateRevisionId,omitempty"`
	Status                string                  `json:"status"`
	Assistant             string                  `json:"assistantId,omitempty"`
	Language              string                  `json:"language,omitempty"`
	ChunkCount            int                     `json:"chunkCount"`
	CueCount              int                     `json:"cueCount"`
	ChangedCueCount       int                     `json:"changedCueCount,omitempty"`
	SourceHash            string                  `json:"sourceHash,omitempty"`
	Passes                []string                `json:"passes,omitempty"`
	GlossaryProfileIDs    []string                `json:"glossaryProfileIds,omitempty"`
	PromptProfileIDs      []string                `json:"promptProfileIds,omitempty"`
	InlinePromptHash      string                  `json:"inlinePromptHash,omitempty"`
	CompletedChunkCount   int                     `json:"completedChunkCount,omitempty"`
	FailedChunkCount      int                     `json:"failedChunkCount,omitempty"`
	RequestHash           string                  `json:"requestHash,omitempty"`
	PromptHash            string                  `json:"promptHash,omitempty"`
	ResumedFromCheckpoint bool                    `json:"resumedFromCheckpoint,omitempty"`
	Usage                 runtimedto.RuntimeUsage `json:"usage,omitempty"`
}

type subtitleProofreadConstraints struct {
	GlossaryProfiles []library.GlossaryProfile
	PromptProfiles   []library.PromptProfile
	InlinePrompt     string
}

type subtitleProofreadResponse struct {
	Items []subtitleProofreadItem `json:"items"`
}

type subtitleProofreadItem struct {
	Index      int      `json:"index"`
	Text       string   `json:"text"`
	Categories []string `json:"categories,omitempty"`
	Reason     string   `json:"reason,omitempty"`
}

type subtitleProofreadCheckpointState struct {
	ItemsByChunkIndex     map[int][]subtitleProofreadItem
	Usage                 runtimedto.RuntimeUsage
	CompletedChunkCount   int
	ResumedFromCheckpoint bool
}

func (service *LibraryService) runSubtitleProofreadOperation(ctx context.Context, operation library.LibraryOperation, request dto.SubtitleProofreadRequest) {
	request = normalizeSubtitleProofreadRequest(request)
	runCtx, cancel := context.WithCancel(ctx)
	service.registerOperationRun(operation.ID, cancel)
	defer func() {
		cancel()
		service.unregisterOperationRun(operation.ID)
	}()

	sourceFile, document, err := service.resolveSubtitleFileAndDocument(ctx, request.FileID, request.DocumentID, request.Path)
	if err != nil {
		service.failSubtitleProofreadOperation(ctx, operation, err, runtimedto.RuntimeUsage{}, subtitleTaskRunState{})
		return
	}

	sourceContent := strings.TrimSpace(document.WorkingContent)
	if sourceContent == "" {
		sourceContent = strings.TrimSpace(document.OriginalContent)
	}
	if sourceContent == "" {
		service.failSubtitleProofreadOperation(ctx, operation, errors.New("subtitle content is empty"), runtimedto.RuntimeUsage{}, subtitleTaskRunState{})
		return
	}

	outputFormat := detectSubtitleFormat(request.OutputFormat, sourceFile.Storage.LocalPath, document.Format)
	sourceDocument := parseSubtitleDocument(sourceContent, detectSubtitleFormat(document.Format, sourceFile.Storage.LocalPath, document.Format))
	if len(sourceDocument.Cues) == 0 {
		service.failSubtitleProofreadOperation(ctx, operation, errors.New("subtitle document has no cues"), runtimedto.RuntimeUsage{}, subtitleTaskRunState{})
		return
	}
	moduleConfig, err := service.getModuleConfig(ctx)
	if err != nil {
		service.failSubtitleProofreadOperation(ctx, operation, err, runtimedto.RuntimeUsage{}, subtitleTaskRunState{})
		return
	}
	constraints, err := service.resolveSubtitleProofreadConstraints(request, moduleConfig)
	if err != nil {
		service.failSubtitleProofreadOperation(ctx, operation, err, runtimedto.RuntimeUsage{}, subtitleTaskRunState{})
		return
	}
	runtimeConfig := resolveSubtitleTaskRuntimeSettings(moduleConfig.TaskRuntime.Proofread)

	chunks := buildSubtitleTranslateChunks(sourceDocument.Cues)
	sourceHash := hashSubtitleSource(sourceContent)
	requestHash := hashSubtitleProofreadRequest(request)
	promptHash := hashSubtitleProofreadPrompt(request, constraints)
	checkpointState, err := service.loadSubtitleProofreadChunkCheckpointState(ctx, operation.ID, chunks, requestHash, promptHash)
	if err != nil {
		service.failSubtitleProofreadOperation(ctx, operation, err, runtimedto.RuntimeUsage{}, subtitleTaskRunState{
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
	operation.DisplayName = buildSubtitleProofreadOutputName(sourceFile.Name)
	operation.Status = library.OperationStatusRunning
	now := service.now()
	operation.StartedAt = &now
	operation.Progress = buildOperationProgress(
		now,
		progressText("library.progress.preparing"),
		runState.CompletedChunkCount,
		len(chunks),
		progressText("library.progressDetail.preparingSubtitleProofread"),
	)
	totalUsage := checkpointState.Usage
	operation.OutputJSON = marshalJSON(buildSubtitleProofreadOutput(
		request,
		"running",
		len(chunks),
		len(sourceDocument.Cues),
		sourceHash,
		sourceFile.ID,
		sourceFile.Storage.DocumentID,
		"",
		"",
		"",
		0,
		totalUsage,
		runState,
	))
	if err := service.saveAndPublishOperation(ctx, operation); err != nil {
		return
	}

	proofreadCues := make([]dto.SubtitleCue, 0, len(sourceDocument.Cues))
	proofreadSuggestions := make([]library.SubtitleReviewSuggestion, 0)
	for _, chunk := range chunks {
		if restoredItems, ok := checkpointState.ItemsByChunkIndex[chunk.Sequence]; ok {
			proofreadCues = appendProofreadChunkItemsAsCues(proofreadCues, chunk, restoredItems)
			proofreadSuggestions = append(proofreadSuggestions, buildProofreadSuggestions(chunk, restoredItems)...)
			continue
		}
		if runCtx.Err() != nil || service.isSubtitleOperationCanceled(ctx, operation.ID) {
			_, _ = service.markSubtitleOperationCanceled(ctx, operation.ID)
			return
		}

		startedAt := service.now()
		if err := service.saveSubtitleProofreadChunkCheckpoint(
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
			service.failSubtitleProofreadOperation(ctx, operation, err, totalUsage, runState)
			return
		}

		chunkCtx, chunkCancel := context.WithTimeout(runCtx, subtitleProofreadChunkTimeout)
		proofreadItems, usage, attemptsUsed, err := service.proofreadSubtitleChunk(chunkCtx, request, chunk, constraints, runtimeConfig)
		chunkCancel()
		totalUsage = addRuntimeUsage(totalUsage, usage)
		finishedAt := service.now()
		if err != nil {
			checkpointStatus := library.OperationChunkStatusFailed
			if errors.Is(err, context.Canceled) || runCtx.Err() == context.Canceled || service.isSubtitleOperationCanceled(ctx, operation.ID) {
				checkpointStatus = library.OperationChunkStatusCanceled
			}
			if saveErr := service.saveSubtitleProofreadChunkCheckpoint(
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
				service.failSubtitleProofreadOperation(ctx, operation, saveErr, totalUsage, runState)
				return
			}
			if checkpointStatus == library.OperationChunkStatusCanceled {
				_, _ = service.markSubtitleOperationCanceled(ctx, operation.ID)
				return
			}
			runState.FailedChunkCount++
			service.failSubtitleProofreadOperation(
				ctx,
				operation,
				fmt.Errorf("chunk %d/%d failed: %w", chunk.Sequence, len(chunks), err),
				totalUsage,
				runState,
			)
			return
		}

		if err := service.saveSubtitleProofreadChunkCheckpoint(
			ctx,
			operation,
			chunk,
			library.OperationChunkStatusSucceeded,
			requestHash,
			promptHash,
			proofreadItems,
			usage,
			countUsedRetries(attemptsUsed),
			"",
			&startedAt,
			&finishedAt,
		); err != nil {
			service.failSubtitleProofreadOperation(ctx, operation, err, totalUsage, runState)
			return
		}

		runState.CompletedChunkCount++
		proofreadCues = appendProofreadChunkItemsAsCues(proofreadCues, chunk, proofreadItems)
		proofreadSuggestions = append(proofreadSuggestions, buildProofreadSuggestions(chunk, proofreadItems)...)
		if runCtx.Err() != nil || service.isSubtitleOperationCanceled(ctx, operation.ID) {
			_, _ = service.markSubtitleOperationCanceled(ctx, operation.ID)
			return
		}

		progressTime := service.now()
		operation.Progress = buildOperationProgress(
			progressTime,
			progressText("library.progress.proofreading"),
			runState.CompletedChunkCount,
			len(chunks),
			progressTextTemplate("library.progressDetail.proofreadChunk", map[string]string{
				"current": fmt.Sprintf("%d", runState.CompletedChunkCount),
				"total":   fmt.Sprintf("%d", len(chunks)),
			}),
		)
		operation.OutputJSON = marshalJSON(buildSubtitleProofreadOutput(
			request,
			"running",
			len(chunks),
			len(sourceDocument.Cues),
			sourceHash,
			sourceFile.ID,
			sourceFile.Storage.DocumentID,
			"",
			"",
			"",
			len(proofreadSuggestions),
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

	proofreadDocument := dto.SubtitleDocument{
		Format: outputFormat,
		Cues:   proofreadCues,
		Metadata: map[string]any{
			"proofread": true,
			"language":  strings.TrimSpace(request.Language),
		},
	}
	proofreadContent := renderSubtitleContent(proofreadDocument, outputFormat)
	finishedAt := service.now()
	var (
		reviewSessionID     string
		sourceRevisionID    string
		candidateRevisionID string
	)
	if len(proofreadSuggestions) > 0 {
		sourceRevision, err := service.createSubtitleRevision(
			ctx,
			sourceFile,
			outputFormat,
			sourceContent,
			"snapshot",
			operation.ID,
			"",
		)
		if err != nil {
			service.failSubtitleProofreadOperation(ctx, operation, err, totalUsage, runState)
			return
		}
		candidateRevision, err := service.createSubtitleRevision(
			ctx,
			sourceFile,
			outputFormat,
			proofreadContent,
			"proofread_candidate",
			operation.ID,
			"",
		)
		if err != nil {
			service.failSubtitleProofreadOperation(ctx, operation, err, totalUsage, runState)
			return
		}
		reviewSession, err := service.createPendingSubtitleReviewSession(
			ctx,
			sourceFile,
			"proofread",
			operation.ID,
			sourceRevision,
			candidateRevision,
			proofreadSuggestions,
		)
		if err != nil {
			service.failSubtitleProofreadOperation(ctx, operation, err, totalUsage, runState)
			return
		}
		reviewSessionID = reviewSession.ID
		sourceRevisionID = sourceRevision.ID
		candidateRevisionID = candidateRevision.ID
	}
	sourceFile.LatestOperationID = operation.ID
	sourceFile.UpdatedAt = finishedAt
	if err := service.files.Save(ctx, sourceFile); err != nil {
		service.failSubtitleProofreadOperation(ctx, operation, err, totalUsage, runState)
		return
	}

	operation.Status = library.OperationStatusSucceeded
	operation.FinishedAt = &finishedAt
	operation.OutputFiles = nil
	operation.Metrics = buildOperationMetricsForOperation(nil, operation.StartedAt, &finishedAt)
	operation.Progress = buildOperationProgress(
		finishedAt,
		progressText("library.status.succeeded"),
		len(chunks),
		len(chunks),
		progressText("library.progressDetail.subtitleProofreadCompleted"),
	)
	operation.OutputJSON = marshalJSON(buildSubtitleProofreadOutput(
		request,
		"completed",
		len(chunks),
		len(proofreadCues),
		sourceHash,
		sourceFile.ID,
		sourceFile.Storage.DocumentID,
		reviewSessionID,
		sourceRevisionID,
		candidateRevisionID,
		len(proofreadSuggestions),
		totalUsage,
		runState,
	))
	if err := service.operations.Save(ctx, operation); err != nil {
		service.failSubtitleProofreadOperation(ctx, operation, err, totalUsage, runState)
		return
	}

	history, err := library.NewHistoryRecord(library.HistoryRecordParams{
		ID:          uuid.NewString(),
		LibraryID:   sourceFile.LibraryID,
		Category:    "operation",
		Action:      "subtitle_proofread",
		DisplayName: operation.DisplayName,
		Status:      string(operation.Status),
		Source: library.HistoryRecordSource{
			Kind:  resolveHistorySourceKind(request.Source),
			RunID: strings.TrimSpace(request.RunID),
		},
		Refs:          library.HistoryRecordRefs{OperationID: operation.ID},
		Files:         operation.OutputFiles,
		Metrics:       operation.Metrics,
		OperationMeta: &library.OperationRecordMeta{Kind: "subtitle_proofread"},
		OccurredAt:    &finishedAt,
		CreatedAt:     &finishedAt,
		UpdatedAt:     &finishedAt,
	})
	if err != nil {
		service.failSubtitleProofreadOperation(ctx, operation, err, totalUsage, runState)
		return
	}
	if err := service.histories.Save(ctx, history); err != nil {
		service.failSubtitleProofreadOperation(ctx, operation, err, totalUsage, runState)
		return
	}
	if err := service.touchLibrary(ctx, sourceFile.LibraryID, finishedAt); err != nil {
		service.failSubtitleProofreadOperation(ctx, operation, err, totalUsage, runState)
		return
	}

	service.publishOperationUpdate(toOperationDTO(operation))
	service.publishHistoryUpdate(toHistoryDTO(history))
	service.publishFileUpdate(service.mustBuildFileDTO(ctx, sourceFile))
	service.publishWorkspaceProjectUpdate(sourceFile.LibraryID)
}

func (service *LibraryService) proofreadSubtitleChunk(
	ctx context.Context,
	request dto.SubtitleProofreadRequest,
	chunk subtitleTranslateChunk,
	constraints subtitleProofreadConstraints,
	runtimeConfig subtitleTaskRuntimeSettings,
) ([]subtitleProofreadItem, runtimedto.RuntimeUsage, int, error) {
	var (
		lastErr      error
		totalUsage   runtimedto.RuntimeUsage
		repairPrompt string
		attemptsUsed int
	)

	for attempt := 0; attempt < subtitleTranslateMaxRetries; attempt++ {
		attemptsUsed = attempt + 1
		systemPrompt, userPrompt := buildSubtitleProofreadPrompts(request, chunk, constraints, repairPrompt)
		result, err := service.runtime.RunOneShot(ctx, runtimedto.RuntimeRunRequest{
			AssistantID: strings.TrimSpace(request.AssistantID),
			RunKind:     "one-shot",
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
				"useQueue":          true,
				"runLane":           "subagent",
				"oneShotKind":       "subtitle_proofread",
				"temperature":       0.1,
				"maxTokens":         estimateSubtitleChunkMaxTokensWithRuntime(chunk, runtimeConfig, attempt),
				"extraSystemPrompt": systemPrompt,
				"structuredOutput":  buildSubtitleProofreadStructuredOutputMetadata("library_subtitle_proofread_chunk", runtimeConfig),
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
		items, parseErr := parseSubtitleProofreadItems(result.AssistantMessage.Content, chunk.Cues)
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
		lastErr = errors.New("subtitle proofread failed")
	}
	return nil, totalUsage, attemptsUsed, lastErr
}

func buildSubtitleProofreadPrompts(
	request dto.SubtitleProofreadRequest,
	chunk subtitleTranslateChunk,
	constraints subtitleProofreadConstraints,
	repairPrompt string,
) (string, string) {
	var systemBuilder strings.Builder
	systemBuilder.WriteString("You proofread subtitle cues and return corrected text in the same language.\n")
	systemBuilder.WriteString("Return only valid JSON with this shape: {\"items\":[{\"index\":1,\"text\":\"...\",\"categories\":[\"spelling\"],\"reason\":\"...\"}]}\n")
	systemBuilder.WriteString("Rules:\n")
	systemBuilder.WriteString("- Keep the same number of items, the same indexes, and the same order.\n")
	systemBuilder.WriteString("- Only revise cue text. Do not merge, split, drop, or invent cues.\n")
	systemBuilder.WriteString("- Preserve the original language. Do not translate into another language.\n")
	systemBuilder.WriteString("- Preserve inline formatting markers, speaker labels, and escaped line breaks when possible.\n")
	systemBuilder.WriteString("- Make conservative fixes only. If a cue is already correct, keep it unchanged.\n")
	systemBuilder.WriteString("- Do not output markdown, prose, or explanations.\n")
	if language := strings.TrimSpace(request.Language); language != "" {
		systemBuilder.WriteString(fmt.Sprintf("- The subtitle language is %s; keep the output in that language.\n", language))
	}
	if passPrompt := buildSubtitleProofreadPassPrompt(request); passPrompt != "" {
		systemBuilder.WriteString("\nRequested proofread scope:\n")
		systemBuilder.WriteString(passPrompt)
		systemBuilder.WriteString("\n")
	}
	if glossaryText := renderGlossaryPrompt(chunk, constraints.GlossaryProfiles); glossaryText != "" {
		systemBuilder.WriteString("\nStructured glossary constraints:\n")
		systemBuilder.WriteString(glossaryText)
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
	if repairPrompt != "" {
		systemBuilder.WriteString("\nRepair note:\n")
		systemBuilder.WriteString("- Repair the previous invalid output and still obey every rule above.\n")
	}

	payload := make([]map[string]any, 0, len(chunk.Cues))
	for _, cue := range chunk.Cues {
		payload = append(payload, map[string]any{
			"index": cue.Index,
			"text":  cue.Text,
		})
	}

	var userBuilder strings.Builder
	if language := strings.TrimSpace(request.Language); language != "" {
		userBuilder.WriteString(fmt.Sprintf("Subtitle language: %s\n", language))
	}
	passes := enabledProofreadPasses(request)
	if len(passes) > 0 {
		userBuilder.WriteString(fmt.Sprintf("Requested proofread passes: %s\n", strings.Join(passes, ", ")))
	} else {
		userBuilder.WriteString("Requested proofread passes: conservative general proofreading\n")
	}
	userBuilder.WriteString("Proofread the following subtitle cues and return corrected cue text only.\n")
	if repairPrompt != "" {
		userBuilder.WriteString(repairPrompt)
		userBuilder.WriteString("\n")
	}
	userBuilder.WriteString("For each changed cue, explain the reason briefly and assign one or more categories from: spelling, punctuation, terminology, fluency.\n")
	userBuilder.WriteString("If a cue stays unchanged, still return it with empty categories and an empty reason.\n")
	userBuilder.WriteString("Input JSON:\n")
	userBuilder.WriteString(marshalJSON(map[string]any{"items": payload}))
	return strings.TrimSpace(systemBuilder.String()), strings.TrimSpace(userBuilder.String())
}

func buildSubtitleProofreadStructuredOutputMetadata(schemaName string, runtimeConfig subtitleTaskRuntimeSettings) map[string]any {
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
						"required":             []string{"index", "text", "categories", "reason"},
						"properties": map[string]any{
							"index": map[string]any{"type": "integer"},
							"text":  map[string]any{"type": "string"},
							"categories": map[string]any{
								"type": "array",
								"items": map[string]any{
									"type": "string",
								},
							},
							"reason": map[string]any{"type": "string"},
						},
					},
				},
			},
		},
	}
}

func parseSubtitleProofreadItems(raw string, sourceCues []dto.SubtitleCue) ([]subtitleProofreadItem, error) {
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
	response := subtitleProofreadResponse{}
	if err := json.Unmarshal([]byte(content), &response); err != nil {
		return nil, fmt.Errorf("invalid json output: %w", err)
	}
	if len(response.Items) != len(sourceCues) {
		return nil, fmt.Errorf("expected %d proofread items, got %d", len(sourceCues), len(response.Items))
	}
	items := make([]subtitleProofreadItem, 0, len(response.Items))
	for index, item := range response.Items {
		expectedIndex := sourceCues[index].Index
		if item.Index != expectedIndex {
			return nil, fmt.Errorf("expected index %d at position %d, got %d", expectedIndex, index, item.Index)
		}
		if strings.TrimSpace(item.Text) == "" && strings.TrimSpace(sourceCues[index].Text) != "" {
			return nil, fmt.Errorf("proofread text for cue %d is empty", expectedIndex)
		}
		items = append(items, subtitleProofreadItem{
			Index:      item.Index,
			Text:       strings.TrimSpace(item.Text),
			Categories: uniqueTrimmedStrings(item.Categories),
			Reason:     strings.TrimSpace(item.Reason),
		})
	}
	return items, nil
}

func appendProofreadChunkItemsAsCues(
	target []dto.SubtitleCue,
	chunk subtitleTranslateChunk,
	items []subtitleProofreadItem,
) []dto.SubtitleCue {
	for itemIndex, proofread := range items {
		sourceCue := chunk.Cues[itemIndex]
		target = append(target, dto.SubtitleCue{
			Index: sourceCue.Index,
			Start: sourceCue.Start,
			End:   sourceCue.End,
			Text:  proofread.Text,
		})
	}
	return target
}

func buildProofreadSuggestions(chunk subtitleTranslateChunk, items []subtitleProofreadItem) []library.SubtitleReviewSuggestion {
	result := make([]library.SubtitleReviewSuggestion, 0)
	for itemIndex, proofread := range items {
		sourceCue := chunk.Cues[itemIndex]
		if sourceCue.Text == proofread.Text {
			continue
		}
		result = append(result, library.SubtitleReviewSuggestion{
			CueIndex:      sourceCue.Index,
			OriginalText:  sourceCue.Text,
			SuggestedText: proofread.Text,
			Categories:    append([]string(nil), proofread.Categories...),
			Reason:        proofread.Reason,
			SourceCode:    "proofread",
			Severity:      "warning",
		})
	}
	return result
}

func (service *LibraryService) loadSubtitleProofreadChunkCheckpointState(
	ctx context.Context,
	operationID string,
	chunks []subtitleTranslateChunk,
	requestHash string,
	promptHash string,
) (subtitleProofreadCheckpointState, error) {
	state := subtitleProofreadCheckpointState{ItemsByChunkIndex: make(map[int][]subtitleProofreadItem)}
	if service == nil || service.operationChunks == nil {
		return state, nil
	}
	rows, err := service.operationChunks.ListByOperationID(ctx, operationID)
	if err != nil {
		return state, err
	}
	chunkMap := make(map[int]subtitleTranslateChunk, len(chunks))
	for _, chunk := range chunks {
		chunkMap[chunk.Sequence] = chunk
	}
	for _, row := range rows {
		if row.Status != library.OperationChunkStatusSucceeded {
			continue
		}
		chunk, ok := chunkMap[row.ChunkIndex]
		if !ok {
			continue
		}
		if row.RequestHash != "" && requestHash != "" && row.RequestHash != requestHash {
			return state, fmt.Errorf("checkpoint request hash mismatch for chunk %d", row.ChunkIndex)
		}
		if row.PromptHash != "" && promptHash != "" && row.PromptHash != promptHash {
			return state, fmt.Errorf("checkpoint prompt hash mismatch for chunk %d", row.ChunkIndex)
		}
		expectedInputHash := hashSubtitleChunkInput(chunk)
		if row.InputHash != "" && row.InputHash != expectedInputHash {
			return state, fmt.Errorf("checkpoint source changed for chunk %d", row.ChunkIndex)
		}
		items, err := parseSubtitleProofreadItems(row.ResultJSON, chunk.Cues)
		if err != nil {
			return state, fmt.Errorf("checkpoint parse failed for chunk %d: %w", row.ChunkIndex, err)
		}
		state.ItemsByChunkIndex[row.ChunkIndex] = items
		state.Usage = addRuntimeUsage(state.Usage, parseChunkUsage(row.UsageJSON))
		state.CompletedChunkCount++
	}
	state.ResumedFromCheckpoint = state.CompletedChunkCount > 0
	return state, nil
}

func (service *LibraryService) saveSubtitleProofreadChunkCheckpoint(
	ctx context.Context,
	operation library.LibraryOperation,
	chunk subtitleTranslateChunk,
	status library.OperationChunkStatus,
	requestHash string,
	promptHash string,
	items []subtitleProofreadItem,
	usage runtimedto.RuntimeUsage,
	retryCount int,
	errorMessage string,
	startedAt *time.Time,
	finishedAt *time.Time,
) error {
	if service == nil || service.operationChunks == nil {
		return nil
	}
	now := service.now()
	resultJSON := ""
	responseHash := ""
	if len(items) > 0 {
		resultJSON = marshalJSON(subtitleProofreadResponse{Items: items})
		responseHash = hashStructuredPayload(items)
	}
	chunkItem, err := library.NewOperationChunk(library.OperationChunkParams{
		ID:           uuid.NewString(),
		OperationID:  operation.ID,
		LibraryID:    operation.LibraryID,
		ChunkIndex:   chunk.Sequence,
		Status:       string(status),
		SourceRange:  buildSubtitleChunkSourceRange(chunk),
		InputHash:    hashSubtitleChunkInput(chunk),
		RequestHash:  strings.TrimSpace(requestHash),
		PromptHash:   strings.TrimSpace(promptHash),
		ResponseHash: responseHash,
		ResultJSON:   resultJSON,
		UsageJSON:    marshalChunkUsage(usage),
		RetryCount:   retryCount,
		ErrorMessage: strings.TrimSpace(errorMessage),
		StartedAt:    startedAt,
		FinishedAt:   finishedAt,
		CreatedAt:    &now,
		UpdatedAt:    &now,
	})
	if err != nil {
		return err
	}
	return service.operationChunks.Save(ctx, chunkItem)
}

func buildSubtitleProofreadPassPrompt(request dto.SubtitleProofreadRequest) string {
	lines := make([]string, 0, 4)
	if request.Spelling {
		lines = append(lines, "- Fix spelling mistakes and obvious typos.")
	}
	if request.Punctuation {
		lines = append(lines, "- Normalize punctuation and capitalization only when it improves consistency.")
	}
	if request.Terminology {
		lines = append(lines, "- Keep terminology consistent across cues without rewriting meaning.")
	}
	if len(lines) == 0 {
		lines = append(lines, "- Apply a conservative proofreading pass while preserving the original wording whenever possible.")
	}
	return strings.Join(lines, "\n")
}

func (service *LibraryService) resolveSubtitleProofreadConstraints(
	request dto.SubtitleProofreadRequest,
	config library.ModuleConfig,
) (subtitleProofreadConstraints, error) {
	constraints := subtitleProofreadConstraints{InlinePrompt: strings.TrimSpace(request.InlinePrompt)}
	glossaryProfiles, err := pickGlossaryProfilesByID(config.LanguageAssets.GlossaryProfiles, request.GlossaryProfileIDs)
	if err != nil {
		return subtitleProofreadConstraints{}, err
	}
	constraints.GlossaryProfiles = glossaryProfiles
	promptProfiles, err := pickPromptProfilesByID(config.LanguageAssets.PromptProfiles, request.PromptProfileIDs)
	if err != nil {
		return subtitleProofreadConstraints{}, err
	}
	constraints.PromptProfiles = promptProfiles
	return constraints, nil
}

func (service *LibraryService) failSubtitleProofreadOperation(
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
	currentOperation.ErrorCode = "subtitle_proofread_failed"
	currentOperation.ErrorMessage = strings.TrimSpace(err.Error())
	currentOperation.FinishedAt = &now
	currentOperation.Progress = buildOperationProgress(
		now,
		progressText("library.status.failed"),
		runState.CompletedChunkCount,
		progressTotal(currentOperation.Progress),
		terminalProgressMessage(currentOperation.Kind, currentOperation.Status),
	)
	output := buildSubtitleProofreadFailedOutput(currentOperation.InputJSON, usage, runState)
	if existing, ok := parseSubtitleProofreadOutput(currentOperation.OutputJSON); ok {
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

func buildSubtitleProofreadOutput(
	request dto.SubtitleProofreadRequest,
	status string,
	chunkCount int,
	cueCount int,
	sourceHash string,
	fileID string,
	documentID string,
	reviewSessionID string,
	sourceRevisionID string,
	candidateRevisionID string,
	changedCueCount int,
	usage runtimedto.RuntimeUsage,
	runState subtitleTaskRunState,
) subtitleProofreadOutput {
	return subtitleProofreadOutput{
		DocumentID:            strings.TrimSpace(documentID),
		FileID:                strings.TrimSpace(fileID),
		ReviewSessionID:       strings.TrimSpace(reviewSessionID),
		SourceRevisionID:      strings.TrimSpace(sourceRevisionID),
		CandidateRevisionID:   strings.TrimSpace(candidateRevisionID),
		Status:                strings.TrimSpace(status),
		Assistant:             strings.TrimSpace(request.AssistantID),
		Language:              strings.TrimSpace(request.Language),
		ChunkCount:            chunkCount,
		CueCount:              cueCount,
		ChangedCueCount:       changedCueCount,
		SourceHash:            strings.TrimSpace(sourceHash),
		Passes:                enabledProofreadPasses(request),
		GlossaryProfileIDs:    append([]string(nil), request.GlossaryProfileIDs...),
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

func buildSubtitleProofreadFailedOutput(
	inputJSON string,
	usage runtimedto.RuntimeUsage,
	runState subtitleTaskRunState,
) subtitleProofreadOutput {
	request := extractSubtitleProofreadRequest(inputJSON)
	return buildSubtitleProofreadOutput(request, "failed", 0, 0, "", "", "", "", "", "", 0, usage, runState)
}

func buildSubtitleProofreadOutputName(name string) string {
	base := strings.TrimSpace(name)
	if base == "" {
		base = "Subtitle"
	}
	if strings.HasSuffix(strings.ToLower(base), "(proofread)") {
		return base
	}
	return fmt.Sprintf("%s (Proofread)", base)
}

func normalizeSubtitleProofreadRequest(request dto.SubtitleProofreadRequest) dto.SubtitleProofreadRequest {
	request.FileID = strings.TrimSpace(request.FileID)
	request.DocumentID = strings.TrimSpace(request.DocumentID)
	request.Path = strings.TrimSpace(request.Path)
	request.LibraryID = strings.TrimSpace(request.LibraryID)
	request.RootFileID = strings.TrimSpace(request.RootFileID)
	request.AssistantID = strings.TrimSpace(request.AssistantID)
	request.Language = strings.TrimSpace(request.Language)
	request.OutputFormat = strings.TrimSpace(request.OutputFormat)
	request.Source = strings.TrimSpace(request.Source)
	request.GlossaryProfileIDs = uniqueTrimmedStrings(request.GlossaryProfileIDs)
	request.PromptProfileIDs = uniqueTrimmedStrings(request.PromptProfileIDs)
	request.InlinePrompt = strings.TrimSpace(request.InlinePrompt)
	request.SessionKey = strings.TrimSpace(request.SessionKey)
	request.RunID = strings.TrimSpace(request.RunID)
	return request
}

func extractSubtitleProofreadRequest(inputJSON string) dto.SubtitleProofreadRequest {
	request := dto.SubtitleProofreadRequest{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(inputJSON)), &request); err != nil {
		return dto.SubtitleProofreadRequest{}
	}
	return normalizeSubtitleProofreadRequest(request)
}

func enabledProofreadPasses(request dto.SubtitleProofreadRequest) []string {
	passes := make([]string, 0, 3)
	if request.Spelling {
		passes = append(passes, "spelling")
	}
	if request.Punctuation {
		passes = append(passes, "punctuation")
	}
	if request.Terminology {
		passes = append(passes, "terminology")
	}
	return passes
}
