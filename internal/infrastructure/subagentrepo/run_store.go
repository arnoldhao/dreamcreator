package subagentrepo

import (
	"context"
	"database/sql"
	"dreamcreator/internal/infrastructure/persistence/sqlitedto"
	"errors"
	"strings"
	"time"

	"github.com/uptrace/bun"

	"dreamcreator/internal/application/subagent/service"
)

type SQLiteRunStore struct {
	db *bun.DB
}

type subagentRunRow = sqlitedto.SubagentRunRow

func NewSQLiteRunStore(db *bun.DB) *SQLiteRunStore {
	return &SQLiteRunStore{db: db}
}

func (store *SQLiteRunStore) Save(ctx context.Context, record service.RunRecord) error {
	if store == nil || store.db == nil {
		return errors.New("subagent run store unavailable")
	}
	createdAt := record.CreatedAt
	updatedAt := record.UpdatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	if updatedAt.IsZero() {
		updatedAt = createdAt
	}
	row := subagentRunRow{
		RunID:                 strings.TrimSpace(record.RunID),
		ParentSessionKey:      nullString(record.ParentSessionKey),
		ParentRunID:           nullString(record.ParentRunID),
		AgentID:               nullString(record.AgentID),
		ChildSessionKey:       nullString(record.ChildSessionKey),
		ChildSessionID:        nullString(record.ChildSessionID),
		Task:                  nullString(record.Task),
		Label:                 nullString(record.Label),
		Model:                 nullString(record.Model),
		Thinking:              nullString(record.Thinking),
		CallerModel:           nullString(record.CallerModel),
		CallerThinking:        nullString(record.CallerThinking),
		CleanupPolicy:         nullString(string(record.CleanupPolicy)),
		RunTimeoutSeconds:     nullInt(record.RunTimeoutSeconds),
		ResultText:            nullString(record.Result),
		Notes:                 nullString(record.Notes),
		RuntimeMs:             nullInt64(record.RuntimeMs),
		UsagePromptTokens:     nullInt(record.Usage.PromptTokens),
		UsageCompletionTokens: nullInt(record.Usage.CompletionTokens),
		UsageTotalTokens:      nullInt(record.Usage.TotalTokens),
		TranscriptPath:        nullString(record.TranscriptPath),
		Status:                nullString(string(record.Status)),
		Summary:               nullString(record.Summary),
		ErrorText:             nullString(record.Error),
		AnnounceKey:           nullString(record.AnnounceKey),
		AnnounceAttempts:      nullInt(record.AnnounceAttempts),
		AnnounceSentAt:        nullTime(record.AnnounceSentAt),
		FinishedAt:            nullTime(record.FinishedAt),
		ArchivedAt:            nullTime(record.ArchivedAt),
		CreatedAt:             createdAt,
		UpdatedAt:             updatedAt,
	}
	_, err := store.db.NewInsert().Model(&row).
		On("CONFLICT(run_id) DO UPDATE").
		Set("parent_session_key = EXCLUDED.parent_session_key").
		Set("parent_run_id = EXCLUDED.parent_run_id").
		Set("agent_id = EXCLUDED.agent_id").
		Set("child_session_key = EXCLUDED.child_session_key").
		Set("child_session_id = EXCLUDED.child_session_id").
		Set("task = EXCLUDED.task").
		Set("label = EXCLUDED.label").
		Set("model = EXCLUDED.model").
		Set("thinking = EXCLUDED.thinking").
		Set("caller_model = EXCLUDED.caller_model").
		Set("caller_thinking = EXCLUDED.caller_thinking").
		Set("cleanup_policy = EXCLUDED.cleanup_policy").
		Set("run_timeout_seconds = EXCLUDED.run_timeout_seconds").
		Set("result_text = EXCLUDED.result_text").
		Set("notes = EXCLUDED.notes").
		Set("runtime_ms = EXCLUDED.runtime_ms").
		Set("usage_prompt_tokens = EXCLUDED.usage_prompt_tokens").
		Set("usage_completion_tokens = EXCLUDED.usage_completion_tokens").
		Set("usage_total_tokens = EXCLUDED.usage_total_tokens").
		Set("transcript_path = EXCLUDED.transcript_path").
		Set("status = EXCLUDED.status").
		Set("summary = EXCLUDED.summary").
		Set("error_text = EXCLUDED.error_text").
		Set("announce_key = EXCLUDED.announce_key").
		Set("announce_attempts = EXCLUDED.announce_attempts").
		Set("announce_sent_at = EXCLUDED.announce_sent_at").
		Set("finished_at = EXCLUDED.finished_at").
		Set("archived_at = EXCLUDED.archived_at").
		Set("updated_at = EXCLUDED.updated_at").
		Exec(ctx)
	return err
}

func (store *SQLiteRunStore) Get(ctx context.Context, runID string) (service.RunRecord, error) {
	if store == nil || store.db == nil {
		return service.RunRecord{}, errors.New("subagent run store unavailable")
	}
	row := new(subagentRunRow)
	if err := store.db.NewSelect().Model(row).Where("run_id = ?", strings.TrimSpace(runID)).Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return service.RunRecord{}, service.ErrSubagentRunNotFound
		}
		return service.RunRecord{}, err
	}
	return rowToRunRecord(*row), nil
}

func (store *SQLiteRunStore) ListByParent(ctx context.Context, parentSessionKey string) ([]service.RunRecord, error) {
	if store == nil || store.db == nil {
		return nil, errors.New("subagent run store unavailable")
	}
	rows := make([]subagentRunRow, 0)
	if err := store.db.NewSelect().Model(&rows).
		Where("parent_session_key = ?", strings.TrimSpace(parentSessionKey)).
		Order("created_at DESC").
		Scan(ctx); err != nil {
		return nil, err
	}
	result := make([]service.RunRecord, 0, len(rows))
	for _, row := range rows {
		result = append(result, rowToRunRecord(row))
	}
	return result, nil
}

func (store *SQLiteRunStore) ListPendingAnnounce(ctx context.Context) ([]service.RunRecord, error) {
	if store == nil || store.db == nil {
		return nil, errors.New("subagent run store unavailable")
	}
	rows := make([]subagentRunRow, 0)
	if err := store.db.NewSelect().Model(&rows).
		Where("announce_sent_at IS NULL").
		Where("status IN (?)", bun.In([]string{
			string(service.RunStatusSuccess),
			string(service.RunStatusFailed),
			string(service.RunStatusTimeout),
			string(service.RunStatusAborted),
		})).
		Order("updated_at ASC").
		Scan(ctx); err != nil {
		return nil, err
	}
	result := make([]service.RunRecord, 0, len(rows))
	for _, row := range rows {
		result = append(result, rowToRunRecord(row))
	}
	return result, nil
}

func rowToRunRecord(row subagentRunRow) service.RunRecord {
	return service.RunRecord{
		RunID:             row.RunID,
		ParentSessionKey:  stringOrEmpty(row.ParentSessionKey),
		ParentRunID:       stringOrEmpty(row.ParentRunID),
		AgentID:           stringOrEmpty(row.AgentID),
		ChildSessionKey:   stringOrEmpty(row.ChildSessionKey),
		ChildSessionID:    stringOrEmpty(row.ChildSessionID),
		Task:              stringOrEmpty(row.Task),
		Label:             stringOrEmpty(row.Label),
		Model:             stringOrEmpty(row.Model),
		Thinking:          stringOrEmpty(row.Thinking),
		CallerModel:       stringOrEmpty(row.CallerModel),
		CallerThinking:    stringOrEmpty(row.CallerThinking),
		CleanupPolicy:     service.ParseCleanupPolicy(stringOrEmpty(row.CleanupPolicy)),
		RunTimeoutSeconds: intOrZero(row.RunTimeoutSeconds),
		Status:            service.NormalizeRunStatus(strings.TrimSpace(row.Status.String)),
		Result:            stringOrEmpty(row.ResultText),
		Notes:             stringOrEmpty(row.Notes),
		RuntimeMs:         int64OrZero(row.RuntimeMs),
		Usage: service.RunUsage{
			PromptTokens:     intOrZero(row.UsagePromptTokens),
			CompletionTokens: intOrZero(row.UsageCompletionTokens),
			TotalTokens:      intOrZero(row.UsageTotalTokens),
		},
		TranscriptPath:   stringOrEmpty(row.TranscriptPath),
		Summary:          stringOrEmpty(row.Summary),
		Error:            strings.TrimSpace(row.ErrorText.String),
		AnnounceKey:      strings.TrimSpace(row.AnnounceKey.String),
		AnnounceAttempts: intOrZero(row.AnnounceAttempts),
		AnnounceSentAt:   timePtr(row.AnnounceSentAt),
		FinishedAt:       timePtr(row.FinishedAt),
		ArchivedAt:       timePtr(row.ArchivedAt),
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
	}
}

func nullInt(value int) sql.NullInt64 {
	if value <= 0 {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(value), Valid: true}
}

func nullString(value string) sql.NullString {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: trimmed, Valid: true}
}

func stringOrEmpty(value sql.NullString) string {
	if !value.Valid {
		return ""
	}
	return strings.TrimSpace(value.String)
}

func intOrZero(value sql.NullInt64) int {
	if !value.Valid {
		return 0
	}
	return int(value.Int64)
}

func int64OrZero(value sql.NullInt64) int64 {
	if !value.Valid {
		return 0
	}
	return value.Int64
}

func nullTime(value *time.Time) sql.NullTime {
	if value == nil || value.IsZero() {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *value, Valid: true}
}

func timePtr(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	copy := value.Time
	return &copy
}

func nullInt64(value int64) sql.NullInt64 {
	if value <= 0 {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: value, Valid: true}
}
