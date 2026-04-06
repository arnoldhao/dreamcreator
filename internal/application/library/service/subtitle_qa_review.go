package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"dreamcreator/internal/application/library/dto"
	"dreamcreator/internal/domain/library"
)

type subtitleQAReviewOutput struct {
	FileID              string `json:"fileId,omitempty"`
	DocumentID          string `json:"documentId,omitempty"`
	ReviewSessionID     string `json:"reviewSessionId,omitempty"`
	SourceRevisionID    string `json:"sourceRevisionId,omitempty"`
	CandidateRevisionID string `json:"candidateRevisionId,omitempty"`
	Status              string `json:"status"`
	CueCount            int    `json:"cueCount"`
	ChangedCueCount     int    `json:"changedCueCount"`
	IssueCount          int    `json:"issueCount"`
}

func (service *LibraryService) CreateSubtitleQAReviewJob(ctx context.Context, request dto.SubtitleQAReviewRequest) (dto.LibraryOperationDTO, error) {
	request = normalizeSubtitleQAReviewRequest(request)
	sourceFile, _, err := service.resolveSubtitleFileAndDocument(ctx, request.FileID, request.DocumentID, request.Path)
	if err != nil {
		return dto.LibraryOperationDTO{}, err
	}
	if strings.TrimSpace(sourceFile.LibraryID) == "" {
		return dto.LibraryOperationDTO{}, fmt.Errorf("source file is not attached to a library")
	}
	now := service.now()
	operationID := uuid.NewString()
	inputJSON, err := json.Marshal(request)
	if err != nil {
		return dto.LibraryOperationDTO{}, err
	}
	operation, err := library.NewLibraryOperation(library.LibraryOperationParams{
		ID:          operationID,
		LibraryID:   sourceFile.LibraryID,
		Kind:        "subtitle_qa_review",
		Status:      string(library.OperationStatusQueued),
		DisplayName: buildSubtitleQAReviewOutputName(sourceFile.Name),
		Correlation: library.OperationCorrelation{RunID: strings.TrimSpace(request.RunID)},
		InputJSON:   string(inputJSON),
		OutputJSON:  marshalJSON(subtitleQAReviewOutput{Status: "queued"}),
		Progress: buildOperationProgress(
			now,
			progressText("library.status.queued"),
			0,
			1,
			progressText("library.progressDetail.subtitleQaReviewQueued"),
		),
		CreatedAt: &now,
	})
	if err != nil {
		return dto.LibraryOperationDTO{}, err
	}
	if err := service.operations.Save(ctx, operation); err != nil {
		return dto.LibraryOperationDTO{}, err
	}
	operationDTO := toOperationDTO(operation)
	service.publishOperationUpdate(operationDTO)
	go service.runSubtitleQAReviewOperation(context.Background(), operation, request)
	return operationDTO, nil
}

func (service *LibraryService) runSubtitleQAReviewOperation(ctx context.Context, operation library.LibraryOperation, request dto.SubtitleQAReviewRequest) {
	request = normalizeSubtitleQAReviewRequest(request)
	runCtx, cancel := context.WithCancel(ctx)
	service.registerOperationRun(operation.ID, cancel)
	defer func() {
		cancel()
		service.unregisterOperationRun(operation.ID)
	}()
	sourceFile, document, err := service.resolveSubtitleFileAndDocument(ctx, request.FileID, request.DocumentID, request.Path)
	if err != nil {
		service.failSubtitleQAReviewOperation(ctx, operation, err)
		return
	}
	sourceContent := strings.TrimSpace(document.WorkingContent)
	if sourceContent == "" {
		sourceContent = strings.TrimSpace(document.OriginalContent)
	}
	if sourceContent == "" {
		service.failSubtitleQAReviewOperation(ctx, operation, errors.New("subtitle content is empty"))
		return
	}
	sourceFormat := detectSubtitleFormat(document.Format, sourceFile.Storage.LocalPath, document.Format)
	sourceDocument := parseSubtitleDocument(sourceContent, sourceFormat)
	if len(sourceDocument.Cues) == 0 {
		service.failSubtitleQAReviewOperation(ctx, operation, errors.New("subtitle document has no cues"))
		return
	}
	operation.Status = library.OperationStatusRunning
	now := service.now()
	operation.StartedAt = &now
	operation.Progress = buildOperationProgress(
		now,
		progressText("library.progress.preparing"),
		0,
		1,
		progressText("library.progressDetail.preparingSubtitleQaReview"),
	)
	operation.OutputJSON = marshalJSON(subtitleQAReviewOutput{Status: "running", CueCount: len(sourceDocument.Cues)})
	if err := service.saveAndPublishOperation(ctx, operation); err != nil {
		return
	}

	candidateDocument := sourceDocument
	changeCount := 0
	normalized, changes := normalizeSubtitleDocumentText(sourceDocument)
	candidateDocument = normalized
	changeCount = changes
	validationResult := validateSubtitleDocument(sourceDocument)
	suggestions := buildSubtitleQAReviewSuggestions(sourceDocument, candidateDocument)
	finishedAt := service.now()

	var sessionID string
	var sourceRevisionID string
	var candidateRevisionID string
	if len(suggestions) > 0 {
		sourceRevision, err := service.createSubtitleRevision(
			runCtx,
			sourceFile,
			sourceFormat,
			sourceContent,
			"snapshot",
			operation.ID,
			"",
		)
		if err != nil {
			service.failSubtitleQAReviewOperation(ctx, operation, err)
			return
		}
		candidateContent := renderSubtitleContent(candidateDocument, sourceFormat)
		candidateRevision, err := service.createSubtitleRevision(
			runCtx,
			sourceFile,
			sourceFormat,
			candidateContent,
			"qa_candidate",
			operation.ID,
			"",
		)
		if err != nil {
			service.failSubtitleQAReviewOperation(ctx, operation, err)
			return
		}
		session, err := service.createPendingSubtitleReviewSession(
			runCtx,
			sourceFile,
			"qa",
			operation.ID,
			sourceRevision,
			candidateRevision,
			suggestions,
		)
		if err != nil {
			service.failSubtitleQAReviewOperation(ctx, operation, err)
			return
		}
		sessionID = session.ID
		sourceRevisionID = sourceRevision.ID
		candidateRevisionID = candidateRevision.ID
	}

	operation.Status = library.OperationStatusSucceeded
	operation.FinishedAt = &finishedAt
	operation.Metrics = buildOperationMetricsForOperation(nil, operation.StartedAt, &finishedAt)
	operation.Progress = buildOperationProgress(
		finishedAt,
		progressText("library.status.succeeded"),
		1,
		1,
		progressText("library.progressDetail.subtitleQaReviewCompleted"),
	)
	operation.OutputJSON = marshalJSON(subtitleQAReviewOutput{
		FileID:              sourceFile.ID,
		DocumentID:          sourceFile.Storage.DocumentID,
		ReviewSessionID:     sessionID,
		SourceRevisionID:    sourceRevisionID,
		CandidateRevisionID: candidateRevisionID,
		Status:              "completed",
		CueCount:            len(sourceDocument.Cues),
		ChangedCueCount:     changeCount,
		IssueCount:          validationResult.IssueCount,
	})
	if err := service.operations.Save(ctx, operation); err != nil {
		service.failSubtitleQAReviewOperation(ctx, operation, err)
		return
	}
	history, err := library.NewHistoryRecord(library.HistoryRecordParams{
		ID:          uuid.NewString(),
		LibraryID:   sourceFile.LibraryID,
		Category:    "operation",
		Action:      operation.Kind,
		DisplayName: operation.DisplayName,
		Status:      string(operation.Status),
		Source: library.HistoryRecordSource{
			Kind:  resolveHistorySourceKind(request.Source),
			RunID: strings.TrimSpace(request.RunID),
		},
		Refs:          library.HistoryRecordRefs{OperationID: operation.ID},
		Metrics:       operation.Metrics,
		OperationMeta: &library.OperationRecordMeta{Kind: operation.Kind},
		OccurredAt:    &finishedAt,
		CreatedAt:     &finishedAt,
		UpdatedAt:     &finishedAt,
	})
	if err != nil {
		service.failSubtitleQAReviewOperation(ctx, operation, err)
		return
	}
	if err := service.histories.Save(ctx, history); err != nil {
		service.failSubtitleQAReviewOperation(ctx, operation, err)
		return
	}
	_ = service.touchLibrary(ctx, sourceFile.LibraryID, finishedAt)
	service.publishOperationUpdate(toOperationDTO(operation))
	service.publishHistoryUpdate(toHistoryDTO(history))
	service.publishWorkspaceProjectUpdate(sourceFile.LibraryID)
}

func buildSubtitleQAReviewSuggestions(sourceDocument dto.SubtitleDocument, candidateDocument dto.SubtitleDocument) []library.SubtitleReviewSuggestion {
	candidateByIndex := make(map[int]dto.SubtitleCue, len(candidateDocument.Cues))
	for _, cue := range candidateDocument.Cues {
		candidateByIndex[cue.Index] = cue
	}
	result := make([]library.SubtitleReviewSuggestion, 0)
	for _, cue := range sourceDocument.Cues {
		candidateCue, ok := candidateByIndex[cue.Index]
		if !ok || cue.Text == candidateCue.Text {
			continue
		}
		result = append(result, library.SubtitleReviewSuggestion{
			CueIndex:      cue.Index,
			OriginalText:  cue.Text,
			SuggestedText: candidateCue.Text,
			Categories:    []string{"spacing"},
			Reason:        "Normalize whitespace and blank-line spacing.",
			SourceCode:    "whitespace",
			Severity:      "warning",
		})
	}
	return result
}

func (service *LibraryService) failSubtitleQAReviewOperation(ctx context.Context, operation library.LibraryOperation, err error) {
	if service == nil || service.operations == nil {
		return
	}
	now := service.now()
	operation.Status = library.OperationStatusFailed
	operation.ErrorCode = "subtitle_qa_review_failed"
	operation.ErrorMessage = strings.TrimSpace(err.Error())
	operation.FinishedAt = &now
	operation.Progress = buildOperationProgress(
		now,
		progressText("library.status.failed"),
		0,
		1,
		progressText("library.progressDetail.subtitleQaReviewFailed"),
	)
	operation.OutputJSON = marshalJSON(subtitleQAReviewOutput{Status: "failed"})
	if saveErr := service.operations.Save(ctx, operation); saveErr != nil {
		return
	}
	service.publishOperationUpdate(toOperationDTO(operation))
}

func buildSubtitleQAReviewOutputName(name string) string {
	base := strings.TrimSpace(name)
	if base == "" {
		base = "Subtitle"
	}
	if strings.HasSuffix(strings.ToLower(base), "(qa review)") {
		return base
	}
	return fmt.Sprintf("%s (QA Review)", base)
}

func normalizeSubtitleQAReviewRequest(request dto.SubtitleQAReviewRequest) dto.SubtitleQAReviewRequest {
	request.FileID = strings.TrimSpace(request.FileID)
	request.DocumentID = strings.TrimSpace(request.DocumentID)
	request.Path = strings.TrimSpace(request.Path)
	request.LibraryID = strings.TrimSpace(request.LibraryID)
	request.OutputFormat = strings.TrimSpace(request.OutputFormat)
	request.Source = strings.TrimSpace(request.Source)
	request.SessionKey = strings.TrimSpace(request.SessionKey)
	request.RunID = strings.TrimSpace(request.RunID)
	return request
}

func extractSubtitleQAReviewRequest(inputJSON string) dto.SubtitleQAReviewRequest {
	request := dto.SubtitleQAReviewRequest{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(inputJSON)), &request); err != nil {
		return dto.SubtitleQAReviewRequest{}
	}
	return normalizeSubtitleQAReviewRequest(request)
}
