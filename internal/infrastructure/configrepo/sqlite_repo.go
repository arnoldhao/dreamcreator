package configrepo

import (
	"context"
	"dreamcreator/internal/infrastructure/persistence/sqlitedto"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/uptrace/bun"

	gatewayconfig "dreamcreator/internal/application/gateway/config"
)

type SQLiteRevisionStore struct {
	db *bun.DB
}

type revisionRow = sqlitedto.RevisionRow

func NewSQLiteRevisionStore(db *bun.DB) *SQLiteRevisionStore {
	return &SQLiteRevisionStore{db: db}
}

func (store *SQLiteRevisionStore) Record(ctx context.Context, revision gatewayconfig.ConfigRevision) error {
	if store == nil || store.db == nil {
		return errors.New("config revision store unavailable")
	}
	payload := map[string]any{
		"config": revision.Config,
		"plan":   revision.Plan,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	row := revisionRow{
		ID:          revision.ID,
		Version:     strings.TrimSpace(intToString(revision.Version)),
		PayloadJSON: string(data),
		CreatedAt:   revision.CreatedAt,
	}
	if row.CreatedAt.IsZero() {
		row.CreatedAt = time.Now()
	}
	_, err = store.db.NewInsert().Model(&row).
		On("CONFLICT(id) DO UPDATE").
		Set("version = EXCLUDED.version").
		Set("payload_json = EXCLUDED.payload_json").
		Set("created_at = EXCLUDED.created_at").
		Exec(ctx)
	return err
}

func (store *SQLiteRevisionStore) Get(ctx context.Context, id string) (gatewayconfig.ConfigRevision, error) {
	if store == nil || store.db == nil {
		return gatewayconfig.ConfigRevision{}, errors.New("config revision store unavailable")
	}
	row := new(revisionRow)
	if err := store.db.NewSelect().Model(row).Where("id = ?", id).Scan(ctx); err != nil {
		return gatewayconfig.ConfigRevision{}, err
	}
	var payload map[string]any
	_ = json.Unmarshal([]byte(row.PayloadJSON), &payload)
	return gatewayconfig.ConfigRevision{
		ID:        row.ID,
		Version:   parseInt(row.Version),
		Config:    payload["config"],
		Plan:      parsePlan(payload["plan"]),
		CreatedAt: row.CreatedAt,
	}, nil
}

func parseInt(value string) int {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0
	}
	var result int
	_, _ = fmt.Sscanf(trimmed, "%d", &result)
	return result
}

func intToString(value int) string {
	return fmt.Sprintf("%d", value)
}

func parsePlan(value any) gatewayconfig.ReloadPlan {
	if value == nil {
		return gatewayconfig.ReloadPlan{}
	}
	data, err := json.Marshal(value)
	if err != nil {
		return gatewayconfig.ReloadPlan{}
	}
	var plan gatewayconfig.ReloadPlan
	if err := json.Unmarshal(data, &plan); err != nil {
		return gatewayconfig.ReloadPlan{}
	}
	return plan
}
