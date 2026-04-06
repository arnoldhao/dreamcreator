package libraryrepo

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/uptrace/bun"

	"dreamcreator/internal/domain/library"
)

type SQLiteSubtitleRevisionRepository struct{ db *bun.DB }
type SQLiteSubtitleReviewSessionRepository struct{ db *bun.DB }

type subtitleRevisionRow struct {
	bun.BaseModel   `bun:"table:library_subtitle_revisions"`
	ID              string         `bun:"id,pk"`
	LibraryID       string         `bun:"library_id"`
	FileID          string         `bun:"file_id"`
	Format          string         `bun:"format"`
	Content         string         `bun:"content"`
	SourceKind      string         `bun:"source_kind"`
	SourceOperation sql.NullString `bun:"source_operation_id"`
	ReviewSessionID sql.NullString `bun:"review_session_id"`
	CreatedAt       time.Time      `bun:"created_at"`
}

type subtitleReviewSessionRow struct {
	bun.BaseModel       `bun:"table:library_subtitle_review_sessions"`
	ID                  string         `bun:"id,pk"`
	LibraryID           string         `bun:"library_id"`
	FileID              string         `bun:"file_id"`
	Kind                string         `bun:"kind"`
	Status              string         `bun:"status"`
	OperationID         sql.NullString `bun:"operation_id"`
	SourceRevisionID    string         `bun:"source_revision_id"`
	CandidateRevisionID string         `bun:"candidate_revision_id"`
	AppliedRevisionID   sql.NullString `bun:"applied_revision_id"`
	ChangedCueCount     int            `bun:"changed_cue_count"`
	SuggestionsJSON     string         `bun:"suggestions_json"`
	CreatedAt           time.Time      `bun:"created_at"`
	UpdatedAt           time.Time      `bun:"updated_at"`
}

func NewSQLiteSubtitleRevisionRepository(db *bun.DB) *SQLiteSubtitleRevisionRepository {
	return &SQLiteSubtitleRevisionRepository{db: db}
}

func NewSQLiteSubtitleReviewSessionRepository(db *bun.DB) *SQLiteSubtitleReviewSessionRepository {
	return &SQLiteSubtitleReviewSessionRepository{db: db}
}

func (repo *SQLiteSubtitleRevisionRepository) Get(ctx context.Context, id string) (library.SubtitleRevision, error) {
	row := new(subtitleRevisionRow)
	if err := repo.db.NewSelect().Model(row).Where("id = ?", strings.TrimSpace(id)).Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return library.SubtitleRevision{}, library.ErrSubtitleRevisionNotFound
		}
		return library.SubtitleRevision{}, err
	}
	return toDomainSubtitleRevision(*row)
}

func (repo *SQLiteSubtitleRevisionRepository) Save(ctx context.Context, item library.SubtitleRevision) error {
	row := subtitleRevisionRow{
		ID:              item.ID,
		LibraryID:       item.LibraryID,
		FileID:          item.FileID,
		Format:          item.Format,
		Content:         item.Content,
		SourceKind:      item.SourceKind,
		SourceOperation: nullString(item.SourceOperation),
		ReviewSessionID: nullString(item.ReviewSessionID),
		CreatedAt:       item.CreatedAt,
	}
	_, err := repo.db.NewInsert().Model(&row).
		On("CONFLICT(id) DO UPDATE").
		Set("library_id = EXCLUDED.library_id").
		Set("file_id = EXCLUDED.file_id").
		Set("format = EXCLUDED.format").
		Set("content = EXCLUDED.content").
		Set("source_kind = EXCLUDED.source_kind").
		Set("source_operation_id = EXCLUDED.source_operation_id").
		Set("review_session_id = EXCLUDED.review_session_id").
		Exec(ctx)
	return err
}

func (repo *SQLiteSubtitleReviewSessionRepository) Get(ctx context.Context, id string) (library.SubtitleReviewSession, error) {
	row := new(subtitleReviewSessionRow)
	if err := repo.db.NewSelect().Model(row).Where("id = ?", strings.TrimSpace(id)).Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return library.SubtitleReviewSession{}, library.ErrSubtitleReviewSessionNotFound
		}
		return library.SubtitleReviewSession{}, err
	}
	return toDomainSubtitleReviewSession(*row)
}

func (repo *SQLiteSubtitleReviewSessionRepository) ListByLibraryID(ctx context.Context, libraryID string) ([]library.SubtitleReviewSession, error) {
	rows := make([]subtitleReviewSessionRow, 0)
	if err := repo.db.NewSelect().
		Model(&rows).
		Where("library_id = ?", strings.TrimSpace(libraryID)).
		Order("updated_at DESC").
		Scan(ctx); err != nil {
		return nil, err
	}
	items := make([]library.SubtitleReviewSession, 0, len(rows))
	for _, row := range rows {
		item, err := toDomainSubtitleReviewSession(row)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (repo *SQLiteSubtitleReviewSessionRepository) Save(ctx context.Context, item library.SubtitleReviewSession) error {
	payload, err := json.Marshal(item.Suggestions)
	if err != nil {
		return err
	}
	row := subtitleReviewSessionRow{
		ID:                  item.ID,
		LibraryID:           item.LibraryID,
		FileID:              item.FileID,
		Kind:                item.Kind,
		Status:              item.Status,
		OperationID:         nullString(item.OperationID),
		SourceRevisionID:    item.SourceRevisionID,
		CandidateRevisionID: item.CandidateRevisionID,
		AppliedRevisionID:   nullString(item.AppliedRevisionID),
		ChangedCueCount:     item.ChangedCueCount,
		SuggestionsJSON:     string(payload),
		CreatedAt:           item.CreatedAt,
		UpdatedAt:           item.UpdatedAt,
	}
	_, err = repo.db.NewInsert().Model(&row).
		On("CONFLICT(id) DO UPDATE").
		Set("library_id = EXCLUDED.library_id").
		Set("file_id = EXCLUDED.file_id").
		Set("kind = EXCLUDED.kind").
		Set("status = EXCLUDED.status").
		Set("operation_id = EXCLUDED.operation_id").
		Set("source_revision_id = EXCLUDED.source_revision_id").
		Set("candidate_revision_id = EXCLUDED.candidate_revision_id").
		Set("applied_revision_id = EXCLUDED.applied_revision_id").
		Set("changed_cue_count = EXCLUDED.changed_cue_count").
		Set("suggestions_json = EXCLUDED.suggestions_json").
		Set("updated_at = EXCLUDED.updated_at").
		Exec(ctx)
	return err
}

func toDomainSubtitleRevision(row subtitleRevisionRow) (library.SubtitleRevision, error) {
	return library.NewSubtitleRevision(library.SubtitleRevisionParams{
		ID:              row.ID,
		LibraryID:       row.LibraryID,
		FileID:          row.FileID,
		Format:          row.Format,
		Content:         row.Content,
		SourceKind:      row.SourceKind,
		SourceOperation: stringOrEmpty(row.SourceOperation),
		ReviewSessionID: stringOrEmpty(row.ReviewSessionID),
		CreatedAt:       &row.CreatedAt,
	})
}

func toDomainSubtitleReviewSession(row subtitleReviewSessionRow) (library.SubtitleReviewSession, error) {
	suggestions := make([]library.SubtitleReviewSuggestion, 0)
	if strings.TrimSpace(row.SuggestionsJSON) != "" {
		if err := json.Unmarshal([]byte(row.SuggestionsJSON), &suggestions); err != nil {
			return library.SubtitleReviewSession{}, err
		}
	}
	return library.NewSubtitleReviewSession(library.SubtitleReviewSessionParams{
		ID:                  row.ID,
		LibraryID:           row.LibraryID,
		FileID:              row.FileID,
		Kind:                row.Kind,
		Status:              row.Status,
		OperationID:         stringOrEmpty(row.OperationID),
		SourceRevisionID:    row.SourceRevisionID,
		CandidateRevisionID: row.CandidateRevisionID,
		AppliedRevisionID:   stringOrEmpty(row.AppliedRevisionID),
		ChangedCueCount:     row.ChangedCueCount,
		Suggestions:         suggestions,
		CreatedAt:           &row.CreatedAt,
		UpdatedAt:           &row.UpdatedAt,
	})
}
