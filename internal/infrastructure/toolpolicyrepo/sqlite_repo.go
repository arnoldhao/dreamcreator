package toolpolicyrepo

import (
	"context"
	"dreamcreator/internal/infrastructure/persistence/sqlitedto"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/uptrace/bun"
)

type SQLitePolicyAuditStore struct {
	db *bun.DB
}

type auditRow = sqlitedto.AuditRow

func NewSQLitePolicyAuditStore(db *bun.DB) *SQLitePolicyAuditStore {
	return &SQLitePolicyAuditStore{db: db}
}

func (store *SQLitePolicyAuditStore) Save(ctx context.Context, toolID string, decision string, reason string, context any) error {
	if store == nil || store.db == nil {
		return errors.New("policy audit store unavailable")
	}
	payload := ""
	if context != nil {
		if data, err := json.Marshal(context); err == nil {
			payload = string(data)
		}
	}
	row := auditRow{
		ToolID:      strings.TrimSpace(toolID),
		Decision:    strings.TrimSpace(decision),
		Reason:      strings.TrimSpace(reason),
		ContextJSON: payload,
		CreatedAt:   time.Now(),
	}
	_, err := store.db.NewInsert().Model(&row).Exec(ctx)
	return err
}
