package approvalsrepo

import (
	"context"
	"database/sql"
	"dreamcreator/internal/infrastructure/persistence/sqlitedto"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/uptrace/bun"

	"dreamcreator/internal/application/gateway/approvals"
)

type SQLiteApprovalStore struct {
	db *bun.DB
}

type approvalRow = sqlitedto.ApprovalRow

func NewSQLiteApprovalStore(db *bun.DB) *SQLiteApprovalStore {
	return &SQLiteApprovalStore{db: db}
}

func (store *SQLiteApprovalStore) Save(ctx context.Context, request approvals.Request) error {
	if store == nil || store.db == nil {
		return errors.New("approval store unavailable")
	}
	payload, _ := json.Marshal(request)
	row := approvalRow{
		ID:          request.ID,
		RequestJSON: string(payload),
		Status:      string(request.Status),
		Decision:    nullString(request.Decision),
		CreatedAt:   request.RequestedAt,
		ResolvedAt:  nullTime(request.ResolvedAt),
	}
	_, err := store.db.NewInsert().Model(&row).
		On("CONFLICT(id) DO UPDATE").
		Set("request_json = EXCLUDED.request_json").
		Set("status = EXCLUDED.status").
		Set("decision = EXCLUDED.decision").
		Set("created_at = EXCLUDED.created_at").
		Set("resolved_at = EXCLUDED.resolved_at").
		Exec(ctx)
	return err
}

func (store *SQLiteApprovalStore) Update(ctx context.Context, request approvals.Request) error {
	return store.Save(ctx, request)
}

func (store *SQLiteApprovalStore) Get(ctx context.Context, id string) (approvals.Request, error) {
	if store == nil || store.db == nil {
		return approvals.Request{}, errors.New("approval store unavailable")
	}
	row := new(approvalRow)
	if err := store.db.NewSelect().Model(row).Where("id = ?", id).Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return approvals.Request{}, errors.New("approval not found")
		}
		return approvals.Request{}, err
	}
	return decodeApproval(row)
}

func decodeApproval(row *approvalRow) (approvals.Request, error) {
	if row == nil {
		return approvals.Request{}, errors.New("approval not found")
	}
	if strings.TrimSpace(row.RequestJSON) == "" {
		return approvals.Request{}, errors.New("approval payload missing")
	}
	var payload approvals.Request
	if err := json.Unmarshal([]byte(row.RequestJSON), &payload); err != nil {
		return approvals.Request{}, err
	}
	payload.Status = approvals.Status(strings.TrimSpace(row.Status))
	payload.Decision = stringOrEmpty(row.Decision)
	payload.RequestedAt = row.CreatedAt
	if row.ResolvedAt.Valid {
		payload.ResolvedAt = row.ResolvedAt.Time
	}
	return payload, nil
}

func nullString(value string) sql.NullString {
	if strings.TrimSpace(value) == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: value, Valid: true}
}

func stringOrEmpty(value sql.NullString) string {
	if !value.Valid {
		return ""
	}
	return value.String
}

func nullTime(value time.Time) sql.NullTime {
	if value.IsZero() {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: value, Valid: true}
}
