package diagnosticsrepo

import (
	"context"
	"dreamcreator/internal/infrastructure/persistence/sqlitedto"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/uptrace/bun"

	gatewayobs "dreamcreator/internal/application/gateway/observability"
)

type SQLiteReportStore struct {
	db *bun.DB
}

type reportRow = sqlitedto.DiagnosticReportRow

func NewSQLiteReportStore(db *bun.DB) *SQLiteReportStore {
	return &SQLiteReportStore{db: db}
}

func (store *SQLiteReportStore) Save(ctx context.Context, report gatewayobs.DiagnosticsReport) error {
	if store == nil || store.db == nil {
		return errors.New("diagnostic report store unavailable")
	}
	payload, err := json.Marshal(report)
	if err != nil {
		return err
	}
	id := strings.TrimSpace(report.GeneratedAt)
	if id == "" {
		id = time.Now().Format(time.RFC3339Nano)
	}
	row := reportRow{
		ID:          id,
		PayloadJSON: string(payload),
		CreatedAt:   time.Now(),
	}
	_, err = store.db.NewInsert().Model(&row).
		On("CONFLICT(id) DO UPDATE").
		Set("payload_json = EXCLUDED.payload_json").
		Set("created_at = EXCLUDED.created_at").
		Exec(ctx)
	return err
}
