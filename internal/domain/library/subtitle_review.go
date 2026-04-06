package library

import (
	"strings"
	"time"
)

type SubtitleRevision struct {
	ID              string
	LibraryID       string
	FileID          string
	Format          string
	Content         string
	SourceKind      string
	SourceOperation string
	ReviewSessionID string
	CreatedAt       time.Time
}

type SubtitleRevisionParams struct {
	ID              string
	LibraryID       string
	FileID          string
	Format          string
	Content         string
	SourceKind      string
	SourceOperation string
	ReviewSessionID string
	CreatedAt       *time.Time
}

func NewSubtitleRevision(params SubtitleRevisionParams) (SubtitleRevision, error) {
	id := strings.TrimSpace(params.ID)
	libraryID := strings.TrimSpace(params.LibraryID)
	fileID := strings.TrimSpace(params.FileID)
	format := strings.TrimSpace(params.Format)
	sourceKind := strings.TrimSpace(params.SourceKind)
	if id == "" || libraryID == "" || fileID == "" || format == "" || sourceKind == "" {
		return SubtitleRevision{}, ErrInvalidSubtitleRevision
	}
	switch sourceKind {
	case "snapshot", "proofread_candidate", "qa_candidate", "review_apply":
	default:
		return SubtitleRevision{}, ErrInvalidSubtitleRevision
	}
	createdAt := time.Now().UTC()
	if params.CreatedAt != nil && !params.CreatedAt.IsZero() {
		createdAt = params.CreatedAt.UTC()
	}
	return SubtitleRevision{
		ID:              id,
		LibraryID:       libraryID,
		FileID:          fileID,
		Format:          format,
		Content:         params.Content,
		SourceKind:      sourceKind,
		SourceOperation: strings.TrimSpace(params.SourceOperation),
		ReviewSessionID: strings.TrimSpace(params.ReviewSessionID),
		CreatedAt:       createdAt,
	}, nil
}

type SubtitleReviewSuggestion struct {
	CueIndex      int
	OriginalText  string
	SuggestedText string
	Categories    []string
	Reason        string
	SourceCode    string
	Severity      string
}

type SubtitleReviewSession struct {
	ID                  string
	LibraryID           string
	FileID              string
	Kind                string
	Status              string
	OperationID         string
	SourceRevisionID    string
	CandidateRevisionID string
	AppliedRevisionID   string
	ChangedCueCount     int
	Suggestions         []SubtitleReviewSuggestion
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type SubtitleReviewSessionParams struct {
	ID                  string
	LibraryID           string
	FileID              string
	Kind                string
	Status              string
	OperationID         string
	SourceRevisionID    string
	CandidateRevisionID string
	AppliedRevisionID   string
	ChangedCueCount     int
	Suggestions         []SubtitleReviewSuggestion
	CreatedAt           *time.Time
	UpdatedAt           *time.Time
}

func NewSubtitleReviewSession(params SubtitleReviewSessionParams) (SubtitleReviewSession, error) {
	id := strings.TrimSpace(params.ID)
	libraryID := strings.TrimSpace(params.LibraryID)
	fileID := strings.TrimSpace(params.FileID)
	kind := strings.TrimSpace(params.Kind)
	status := strings.TrimSpace(params.Status)
	sourceRevisionID := strings.TrimSpace(params.SourceRevisionID)
	candidateRevisionID := strings.TrimSpace(params.CandidateRevisionID)
	if id == "" || libraryID == "" || fileID == "" || kind == "" || status == "" || sourceRevisionID == "" || candidateRevisionID == "" {
		return SubtitleReviewSession{}, ErrInvalidSubtitleReviewSession
	}
	switch kind {
	case "proofread", "qa":
	default:
		return SubtitleReviewSession{}, ErrInvalidSubtitleReviewSession
	}
	switch status {
	case "pending", "applied", "discarded":
	default:
		return SubtitleReviewSession{}, ErrInvalidSubtitleReviewSession
	}
	createdAt := time.Now().UTC()
	if params.CreatedAt != nil && !params.CreatedAt.IsZero() {
		createdAt = params.CreatedAt.UTC()
	}
	updatedAt := createdAt
	if params.UpdatedAt != nil && !params.UpdatedAt.IsZero() {
		updatedAt = params.UpdatedAt.UTC()
	}
	suggestions := append([]SubtitleReviewSuggestion(nil), params.Suggestions...)
	return SubtitleReviewSession{
		ID:                  id,
		LibraryID:           libraryID,
		FileID:              fileID,
		Kind:                kind,
		Status:              status,
		OperationID:         strings.TrimSpace(params.OperationID),
		SourceRevisionID:    sourceRevisionID,
		CandidateRevisionID: candidateRevisionID,
		AppliedRevisionID:   strings.TrimSpace(params.AppliedRevisionID),
		ChangedCueCount:     maxInt(params.ChangedCueCount, 0),
		Suggestions:         suggestions,
		CreatedAt:           createdAt,
		UpdatedAt:           updatedAt,
	}, nil
}

func maxInt(value int, minimum int) int {
	if value < minimum {
		return minimum
	}
	return value
}
