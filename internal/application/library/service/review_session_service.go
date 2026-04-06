package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"dreamcreator/internal/application/library/dto"
	"dreamcreator/internal/domain/library"
)

func (service *LibraryService) createSubtitleRevision(
	ctx context.Context,
	file library.LibraryFile,
	format string,
	content string,
	sourceKind string,
	operationID string,
	reviewSessionID string,
) (library.SubtitleRevision, error) {
	if service == nil || service.revisions == nil {
		return library.SubtitleRevision{}, fmt.Errorf("subtitle revision repository unavailable")
	}
	now := service.now()
	item, err := library.NewSubtitleRevision(library.SubtitleRevisionParams{
		ID:              uuid.NewString(),
		LibraryID:       file.LibraryID,
		FileID:          file.ID,
		Format:          detectSubtitleFormat(format, file.Storage.LocalPath, format),
		Content:         content,
		SourceKind:      sourceKind,
		SourceOperation: strings.TrimSpace(operationID),
		ReviewSessionID: strings.TrimSpace(reviewSessionID),
		CreatedAt:       &now,
	})
	if err != nil {
		return library.SubtitleRevision{}, err
	}
	if err := service.revisions.Save(ctx, item); err != nil {
		return library.SubtitleRevision{}, err
	}
	return item, nil
}

func (service *LibraryService) createPendingSubtitleReviewSession(
	ctx context.Context,
	file library.LibraryFile,
	kind string,
	operationID string,
	sourceRevision library.SubtitleRevision,
	candidateRevision library.SubtitleRevision,
	suggestions []library.SubtitleReviewSuggestion,
) (library.SubtitleReviewSession, error) {
	if service == nil || service.reviews == nil {
		return library.SubtitleReviewSession{}, fmt.Errorf("subtitle review repository unavailable")
	}
	now := service.now()
	item, err := library.NewSubtitleReviewSession(library.SubtitleReviewSessionParams{
		ID:                  uuid.NewString(),
		LibraryID:           file.LibraryID,
		FileID:              file.ID,
		Kind:                kind,
		Status:              "pending",
		OperationID:         strings.TrimSpace(operationID),
		SourceRevisionID:    sourceRevision.ID,
		CandidateRevisionID: candidateRevision.ID,
		ChangedCueCount:     len(suggestions),
		Suggestions:         suggestions,
		CreatedAt:           &now,
		UpdatedAt:           &now,
	})
	if err != nil {
		return library.SubtitleReviewSession{}, err
	}
	if err := service.reviews.Save(ctx, item); err != nil {
		return library.SubtitleReviewSession{}, err
	}
	return item, nil
}

func (service *LibraryService) GetSubtitleReviewSession(
	ctx context.Context,
	request dto.GetSubtitleReviewSessionRequest,
) (dto.SubtitleReviewSessionDetailDTO, error) {
	if service == nil || service.reviews == nil || service.revisions == nil {
		return dto.SubtitleReviewSessionDetailDTO{}, fmt.Errorf("subtitle review unavailable")
	}
	session, err := service.reviews.Get(ctx, strings.TrimSpace(request.SessionID))
	if err != nil {
		return dto.SubtitleReviewSessionDetailDTO{}, err
	}
	return service.buildSubtitleReviewSessionDetailDTO(ctx, session)
}

func (service *LibraryService) buildSubtitleReviewSessionDetailDTO(
	ctx context.Context,
	session library.SubtitleReviewSession,
) (dto.SubtitleReviewSessionDetailDTO, error) {
	sourceRevision, err := service.revisions.Get(ctx, session.SourceRevisionID)
	if err != nil {
		return dto.SubtitleReviewSessionDetailDTO{}, err
	}
	candidateRevision, err := service.revisions.Get(ctx, session.CandidateRevisionID)
	if err != nil {
		return dto.SubtitleReviewSessionDetailDTO{}, err
	}
	file, err := service.files.Get(ctx, session.FileID)
	if err != nil {
		return dto.SubtitleReviewSessionDetailDTO{}, err
	}
	sourceDocument := parseSubtitleDocument(sourceRevision.Content, detectSubtitleFormat(sourceRevision.Format, file.Storage.LocalPath, sourceRevision.Format))
	candidateDocument := parseSubtitleDocument(candidateRevision.Content, detectSubtitleFormat(candidateRevision.Format, file.Storage.LocalPath, candidateRevision.Format))
	return dto.SubtitleReviewSessionDetailDTO{
		SessionID:           session.ID,
		LibraryID:           session.LibraryID,
		FileID:              session.FileID,
		Kind:                session.Kind,
		Status:              session.Status,
		SourceRevisionID:    session.SourceRevisionID,
		CandidateRevisionID: session.CandidateRevisionID,
		AppliedRevisionID:   session.AppliedRevisionID,
		ChangedCueCount:     session.ChangedCueCount,
		SourceDocument:      sourceDocument,
		CandidateDocument:   candidateDocument,
		Suggestions:         toSubtitleReviewSuggestionDTOs(session.Suggestions),
	}, nil
}

func (service *LibraryService) ApplySubtitleReviewSession(
	ctx context.Context,
	request dto.ApplySubtitleReviewSessionRequest,
) (dto.ApplySubtitleReviewSessionResult, error) {
	if service == nil || service.reviews == nil || service.revisions == nil || service.subtitles == nil {
		return dto.ApplySubtitleReviewSessionResult{}, fmt.Errorf("subtitle review unavailable")
	}
	session, err := service.reviews.Get(ctx, strings.TrimSpace(request.SessionID))
	if err != nil {
		return dto.ApplySubtitleReviewSessionResult{}, err
	}
	if session.Status != "pending" {
		return dto.ApplySubtitleReviewSessionResult{}, fmt.Errorf("review session is not pending")
	}
	file, err := service.files.Get(ctx, session.FileID)
	if err != nil {
		return dto.ApplySubtitleReviewSessionResult{}, err
	}
	document, err := service.subtitles.GetByFileID(ctx, file.ID)
	if err != nil {
		return dto.ApplySubtitleReviewSessionResult{}, err
	}
	sourceRevision, err := service.revisions.Get(ctx, session.SourceRevisionID)
	if err != nil {
		return dto.ApplySubtitleReviewSessionResult{}, err
	}
	candidateRevision, err := service.revisions.Get(ctx, session.CandidateRevisionID)
	if err != nil {
		return dto.ApplySubtitleReviewSessionResult{}, err
	}
	sourceDocument := parseSubtitleDocument(sourceRevision.Content, detectSubtitleFormat(sourceRevision.Format, file.Storage.LocalPath, sourceRevision.Format))
	candidateDocument := parseSubtitleDocument(candidateRevision.Content, detectSubtitleFormat(candidateRevision.Format, file.Storage.LocalPath, candidateRevision.Format))
	acceptAll := len(request.Decisions) == 0
	accepted := make(map[int]bool, len(request.Decisions))
	for _, decision := range request.Decisions {
		if strings.EqualFold(strings.TrimSpace(decision.Action), "accept") {
			accepted[decision.CueIndex] = true
		}
	}
	candidateByIndex := make(map[int]dto.SubtitleCue, len(candidateDocument.Cues))
	for _, cue := range candidateDocument.Cues {
		candidateByIndex[cue.Index] = cue
	}
	appliedCues := make([]dto.SubtitleCue, 0, len(sourceDocument.Cues))
	appliedChangeCount := 0
	for _, cue := range sourceDocument.Cues {
		next := cue
		if candidateCue, ok := candidateByIndex[cue.Index]; ok && (acceptAll || accepted[cue.Index]) {
			if cue.Text != candidateCue.Text {
				next.Text = candidateCue.Text
				appliedChangeCount++
			}
		}
		appliedCues = append(appliedCues, next)
	}
	appliedDocument := dto.SubtitleDocument{
		Format:   detectSubtitleFormat(document.Format, file.Storage.LocalPath, sourceRevision.Format),
		Cues:     appliedCues,
		Metadata: cloneSubtitleMetadata(sourceDocument.Metadata),
	}
	appliedContent := renderSubtitleContent(appliedDocument, appliedDocument.Format)
	document.WorkingContent = appliedContent
	document.Format = appliedDocument.Format
	document.UpdatedAt = service.now()
	if err := service.subtitles.Save(ctx, document); err != nil {
		return dto.ApplySubtitleReviewSessionResult{}, err
	}
	file.UpdatedAt = service.now()
	if file.Media == nil {
		file.Media = libraryMediaInfo(appliedDocument.Format, int64(len(appliedContent)))
	} else {
		file.Media.Format = appliedDocument.Format
		sizeValue := int64(len(appliedContent))
		file.Media.SizeBytes = &sizeValue
	}
	if err := service.files.Save(ctx, file); err != nil {
		return dto.ApplySubtitleReviewSessionResult{}, err
	}
	appliedRevision, err := service.createSubtitleRevision(
		ctx,
		file,
		appliedDocument.Format,
		appliedContent,
		"review_apply",
		session.OperationID,
		session.ID,
	)
	if err != nil {
		return dto.ApplySubtitleReviewSessionResult{}, err
	}
	session.Status = "applied"
	session.AppliedRevisionID = appliedRevision.ID
	session.UpdatedAt = service.now()
	if err := service.reviews.Save(ctx, session); err != nil {
		return dto.ApplySubtitleReviewSessionResult{}, err
	}
	_ = service.touchLibrary(ctx, file.LibraryID, service.now())
	service.publishFileUpdate(service.mustBuildFileDTO(ctx, file))
	service.publishWorkspaceProjectUpdate(file.LibraryID)
	return dto.ApplySubtitleReviewSessionResult{
		SessionID:         session.ID,
		FileID:            file.ID,
		AppliedRevisionID: appliedRevision.ID,
		ChangedCueCount:   appliedChangeCount,
	}, nil
}

func (service *LibraryService) DiscardSubtitleReviewSession(
	ctx context.Context,
	request dto.DiscardSubtitleReviewSessionRequest,
) (dto.DiscardSubtitleReviewSessionResult, error) {
	if service == nil || service.reviews == nil {
		return dto.DiscardSubtitleReviewSessionResult{}, fmt.Errorf("subtitle review unavailable")
	}
	session, err := service.reviews.Get(ctx, strings.TrimSpace(request.SessionID))
	if err != nil {
		return dto.DiscardSubtitleReviewSessionResult{}, err
	}
	session.Status = "discarded"
	session.UpdatedAt = service.now()
	if err := service.reviews.Save(ctx, session); err != nil {
		return dto.DiscardSubtitleReviewSessionResult{}, err
	}
	service.publishWorkspaceProjectUpdate(session.LibraryID)
	return dto.DiscardSubtitleReviewSessionResult{SessionID: session.ID, Status: session.Status}, nil
}

func toSubtitleReviewSuggestionDTOs(items []library.SubtitleReviewSuggestion) []dto.SubtitleReviewSuggestionDTO {
	result := make([]dto.SubtitleReviewSuggestionDTO, 0, len(items))
	for _, item := range items {
		result = append(result, dto.SubtitleReviewSuggestionDTO{
			CueIndex:      item.CueIndex,
			OriginalText:  item.OriginalText,
			SuggestedText: item.SuggestedText,
			Categories:    append([]string(nil), item.Categories...),
			Reason:        item.Reason,
			SourceCode:    item.SourceCode,
			Severity:      item.Severity,
		})
	}
	return result
}
