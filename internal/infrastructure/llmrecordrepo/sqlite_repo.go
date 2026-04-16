package llmrecordrepo

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/uptrace/bun"

	"dreamcreator/internal/application/llmrecord"
	"dreamcreator/internal/infrastructure/persistence/sqlitedto"
)

type SQLiteRepository struct {
	db *bun.DB
}

type llmCallRecordRow = sqlitedto.LLMCallRecordRow

func NewSQLiteRepository(db *bun.DB) *SQLiteRepository {
	return &SQLiteRepository{db: db}
}

func (repo *SQLiteRepository) Insert(ctx context.Context, record llmrecord.Record) error {
	row := toRow(record)
	_, err := repo.db.NewInsert().Model(&row).Exec(ctx)
	return err
}

func (repo *SQLiteRepository) Update(ctx context.Context, record llmrecord.Record) error {
	row := toRow(record)
	_, err := repo.db.NewUpdate().Model(&row).
		Column(
			"status",
			"finish_reason",
			"error_text",
			"input_tokens",
			"output_tokens",
			"total_tokens",
			"context_prompt_tokens",
			"context_total_tokens",
			"context_window_tokens",
			"response_payload_json",
			"payload_truncated",
			"finished_at",
			"duration_ms",
		).
		WherePK().
		Exec(ctx)
	return err
}

func (repo *SQLiteRepository) Get(ctx context.Context, id string) (llmrecord.Record, error) {
	row := new(llmCallRecordRow)
	if err := repo.db.NewSelect().Model(row).Where("id = ?", strings.TrimSpace(id)).Limit(1).Scan(ctx); err != nil {
		return llmrecord.Record{}, err
	}
	return fromRow(*row), nil
}

func (repo *SQLiteRepository) List(ctx context.Context, filter llmrecord.QueryFilter) ([]llmrecord.Record, error) {
	rows := make([]llmCallRecordRow, 0)
	query := repo.db.NewSelect().Model(&rows).OrderExpr("started_at DESC, id DESC")
	if trimmed := strings.TrimSpace(filter.ThreadID); trimmed != "" {
		query = query.Where("thread_id = ?", trimmed)
	}
	if trimmed := strings.TrimSpace(filter.RunID); trimmed != "" {
		query = query.Where("run_id = ?", trimmed)
	}
	if trimmed := strings.TrimSpace(filter.ProviderID); trimmed != "" {
		query = query.Where("provider_id = ?", trimmed)
	}
	if trimmed := strings.TrimSpace(filter.ModelName); trimmed != "" {
		query = query.Where("model_name = ?", trimmed)
	}
	if trimmed := strings.TrimSpace(filter.RequestSource); trimmed != "" {
		query = query.Where("request_source = ?", trimmed)
	}
	if trimmed := strings.TrimSpace(filter.Status); trimmed != "" {
		query = query.Where("status = ?", trimmed)
	}
	if !filter.StartAt.IsZero() {
		query = query.Where("started_at >= ?", filter.StartAt.UTC())
	}
	if !filter.EndAt.IsZero() {
		query = query.Where("started_at <= ?", filter.EndAt.UTC())
	}
	if filter.Limit > 0 {
		query = query.Limit(filter.Limit)
	}
	if err := query.Scan(ctx); err != nil {
		return nil, err
	}
	result := make([]llmrecord.Record, 0, len(rows))
	for _, row := range rows {
		result = append(result, fromRow(row))
	}
	return result, nil
}

func toRow(record llmrecord.Record) llmCallRecordRow {
	return llmCallRecordRow{
		ID:                  strings.TrimSpace(record.ID),
		SessionID:           nullString(record.SessionID),
		ThreadID:            nullString(record.ThreadID),
		RunID:               nullString(record.RunID),
		ProviderID:          nullString(record.ProviderID),
		ModelName:           nullString(record.ModelName),
		RequestSource:       nullString(record.RequestSource),
		Operation:           nullString(record.Operation),
		Status:              strings.TrimSpace(record.Status),
		FinishReason:        nullString(record.FinishReason),
		ErrorText:           nullString(record.ErrorText),
		InputTokens:         nullInt64(int64(record.InputTokens)),
		OutputTokens:        nullInt64(int64(record.OutputTokens)),
		TotalTokens:         nullInt64(int64(record.TotalTokens)),
		ContextPromptTokens: nullInt64(int64(record.ContextPromptTokens)),
		ContextTotalTokens:  nullInt64(int64(record.ContextTotalTokens)),
		ContextWindowTokens: nullInt64(int64(record.ContextWindowTokens)),
		RequestPayloadJSON:  nullString(record.RequestPayloadJSON),
		ResponsePayloadJSON: nullString(record.ResponsePayloadJSON),
		PayloadTruncated:    record.PayloadTruncated,
		StartedAt:           record.StartedAt.UTC(),
		FinishedAt:          nullTime(record.FinishedAt),
		DurationMS:          nullInt64(record.DurationMS),
	}
}

func fromRow(row llmCallRecordRow) llmrecord.Record {
	return llmrecord.Record{
		ID:                  row.ID,
		SessionID:           stringOrEmpty(row.SessionID),
		ThreadID:            stringOrEmpty(row.ThreadID),
		RunID:               stringOrEmpty(row.RunID),
		ProviderID:          stringOrEmpty(row.ProviderID),
		ModelName:           stringOrEmpty(row.ModelName),
		RequestSource:       stringOrEmpty(row.RequestSource),
		Operation:           stringOrEmpty(row.Operation),
		Status:              row.Status,
		FinishReason:        stringOrEmpty(row.FinishReason),
		ErrorText:           stringOrEmpty(row.ErrorText),
		InputTokens:         int64OrZero(row.InputTokens),
		OutputTokens:        int64OrZero(row.OutputTokens),
		TotalTokens:         int64OrZero(row.TotalTokens),
		ContextPromptTokens: int64OrZero(row.ContextPromptTokens),
		ContextTotalTokens:  int64OrZero(row.ContextTotalTokens),
		ContextWindowTokens: int64OrZero(row.ContextWindowTokens),
		RequestPayloadJSON:  stringOrEmpty(row.RequestPayloadJSON),
		ResponsePayloadJSON: stringOrEmpty(row.ResponsePayloadJSON),
		PayloadTruncated:    row.PayloadTruncated,
		StartedAt:           row.StartedAt,
		FinishedAt:          timeOrZero(row.FinishedAt),
		DurationMS:          int64ValueOrZero(row.DurationMS),
	}
}

func nullString(value string) sql.NullString {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: trimmed, Valid: true}
}

func nullInt64(value int64) sql.NullInt64 {
	if value <= 0 {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: value, Valid: true}
}

func nullTime(value time.Time) sql.NullTime {
	if value.IsZero() {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: value.UTC(), Valid: true}
}

func stringOrEmpty(value sql.NullString) string {
	if !value.Valid {
		return ""
	}
	return strings.TrimSpace(value.String)
}

func int64OrZero(value sql.NullInt64) int {
	if !value.Valid || value.Int64 <= 0 {
		return 0
	}
	return int(value.Int64)
}

func int64ValueOrZero(value sql.NullInt64) int64 {
	if !value.Valid || value.Int64 <= 0 {
		return 0
	}
	return value.Int64
}

func timeOrZero(value sql.NullTime) time.Time {
	if !value.Valid {
		return time.Time{}
	}
	return value.Time
}
