package sessionrepo

import (
	"context"
	"database/sql"
	"dreamcreator/internal/infrastructure/persistence/sqlitedto"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/uptrace/bun"

	appsession "dreamcreator/internal/application/session"
	domainsession "dreamcreator/internal/domain/session"
)

type SQLiteGatewaySessionStore struct {
	db *bun.DB
}

type gatewaySessionRow = sqlitedto.GatewaySessionRow

func NewSQLiteGatewaySessionStore(db *bun.DB) *SQLiteGatewaySessionStore {
	return &SQLiteGatewaySessionStore{db: db}
}

func (store *SQLiteGatewaySessionStore) Save(ctx context.Context, entry domainsession.Entry) error {
	if store == nil || store.db == nil {
		return errors.New("session store unavailable")
	}
	createdAt := entry.CreatedAt
	updatedAt := entry.UpdatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	if updatedAt.IsZero() {
		updatedAt = createdAt
	}
	hasContextSnapshot := !entry.ContextUpdatedAt.IsZero() ||
		entry.ContextPromptTokens > 0 ||
		entry.ContextTotalTokens > 0 ||
		entry.ContextWindowTokens > 0
	row := gatewaySessionRow{
		SessionID:                  entry.SessionID,
		SessionKey:                 entry.SessionKey,
		AgentID:                    nullString(entry.AgentID),
		AssistantID:                nullString(entry.AssistantID),
		Title:                      nullString(entry.Title),
		Status:                     nullString(string(entry.Status)),
		OriginJSON:                 nullString(marshalOrigin(entry.Origin)),
		ContextPromptTokens:        nullInt(entry.ContextPromptTokens),
		ContextTotalTokens:         nullInt(entry.ContextTotalTokens),
		ContextWindowTokens:        nullInt(entry.ContextWindowTokens),
		ContextUpdatedAt:           nullTime(entry.ContextUpdatedAt),
		ContextFresh:               nullBool(entry.ContextFresh, hasContextSnapshot),
		CompactionCount:            nullInt(entry.CompactionCount),
		MemoryFlushCompactionCount: nullInt(entry.MemoryFlushCompactionCount),
		CreatedAt:                  createdAt,
		UpdatedAt:                  updatedAt,
	}
	_, err := store.db.NewInsert().Model(&row).
		On("CONFLICT(session_id) DO UPDATE").
		Set("session_key = EXCLUDED.session_key").
		Set("agent_id = EXCLUDED.agent_id").
		Set("assistant_id = EXCLUDED.assistant_id").
		Set("title = EXCLUDED.title").
		Set("status = EXCLUDED.status").
		Set("origin_json = EXCLUDED.origin_json").
		Set("context_prompt_tokens = EXCLUDED.context_prompt_tokens").
		Set("context_total_tokens = EXCLUDED.context_total_tokens").
		Set("context_window_tokens = EXCLUDED.context_window_tokens").
		Set("context_updated_at = EXCLUDED.context_updated_at").
		Set("context_fresh = EXCLUDED.context_fresh").
		Set("compaction_count = EXCLUDED.compaction_count").
		Set("memory_flush_compaction_count = EXCLUDED.memory_flush_compaction_count").
		Set("updated_at = EXCLUDED.updated_at").
		Exec(ctx)
	return err
}

func (store *SQLiteGatewaySessionStore) Get(ctx context.Context, sessionID string) (domainsession.Entry, error) {
	if store == nil || store.db == nil {
		return domainsession.Entry{}, errors.New("session store unavailable")
	}
	row := new(gatewaySessionRow)
	if err := store.db.NewSelect().Model(row).Where("session_id = ?", sessionID).Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domainsession.Entry{}, appsession.ErrSessionNotFound
		}
		return domainsession.Entry{}, err
	}
	return rowToEntry(*row), nil
}

func (store *SQLiteGatewaySessionStore) List(ctx context.Context) ([]domainsession.Entry, error) {
	if store == nil || store.db == nil {
		return nil, errors.New("session store unavailable")
	}
	rows := make([]gatewaySessionRow, 0)
	if err := store.db.NewSelect().Model(&rows).Order("updated_at DESC").Scan(ctx); err != nil {
		return nil, err
	}
	result := make([]domainsession.Entry, 0, len(rows))
	for _, row := range rows {
		result = append(result, rowToEntry(row))
	}
	return result, nil
}

func rowToEntry(row gatewaySessionRow) domainsession.Entry {
	origin := domainsession.Origin{}
	if row.OriginJSON.Valid {
		_ = json.Unmarshal([]byte(row.OriginJSON.String), &origin)
	}
	return domainsession.Entry{
		SessionID:                  row.SessionID,
		SessionKey:                 row.SessionKey,
		AgentID:                    stringOrEmpty(row.AgentID),
		AssistantID:                stringOrEmpty(row.AssistantID),
		Title:                      stringOrEmpty(row.Title),
		Status:                     domainsession.Status(strings.TrimSpace(row.Status.String)),
		Origin:                     origin,
		ContextPromptTokens:        intOrZero(row.ContextPromptTokens),
		ContextTotalTokens:         intOrZero(row.ContextTotalTokens),
		ContextWindowTokens:        intOrZero(row.ContextWindowTokens),
		ContextUpdatedAt:           timeOrZero(row.ContextUpdatedAt),
		ContextFresh:               boolOrFalse(row.ContextFresh),
		CompactionCount:            intOrZero(row.CompactionCount),
		MemoryFlushCompactionCount: intOrZero(row.MemoryFlushCompactionCount),
		CreatedAt:                  row.CreatedAt,
		UpdatedAt:                  row.UpdatedAt,
	}
}

func marshalOrigin(origin domainsession.Origin) string {
	data, err := json.Marshal(origin)
	if err != nil {
		return ""
	}
	return string(data)
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

func nullInt(value int) sql.NullInt64 {
	if value <= 0 {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(value), Valid: true}
}

func nullTime(value time.Time) sql.NullTime {
	if value.IsZero() {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: value, Valid: true}
}

func nullBool(value bool, valid bool) sql.NullBool {
	if !valid {
		return sql.NullBool{}
	}
	return sql.NullBool{Bool: value, Valid: true}
}

func intOrZero(value sql.NullInt64) int {
	if !value.Valid {
		return 0
	}
	return int(value.Int64)
}

func timeOrZero(value sql.NullTime) time.Time {
	if !value.Valid {
		return time.Time{}
	}
	return value.Time
}

func boolOrFalse(value sql.NullBool) bool {
	if !value.Valid {
		return false
	}
	return value.Bool
}
