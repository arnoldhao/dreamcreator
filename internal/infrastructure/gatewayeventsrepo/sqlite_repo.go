package gatewayeventsrepo

import (
	"context"
	"database/sql"
	"dreamcreator/internal/infrastructure/persistence/sqlitedto"
	"errors"
	"strings"
	"time"

	"github.com/uptrace/bun"

	gatewayevents "dreamcreator/internal/application/gateway/events"
)

type SQLiteEventStore struct {
	db *bun.DB
}

type gatewayEventRow = sqlitedto.GatewayEventRow

func NewSQLiteEventStore(db *bun.DB) *SQLiteEventStore {
	return &SQLiteEventStore{db: db}
}

func (store *SQLiteEventStore) Append(ctx context.Context, record gatewayevents.Record) (gatewayevents.Record, error) {
	if store == nil || store.db == nil {
		return gatewayevents.Record{}, errors.New("event store unavailable")
	}
	envelope := record.Envelope
	if envelope.EventID == "" {
		envelope.EventID = time.Now().Format(time.RFC3339Nano)
	}
	if envelope.Timestamp.IsZero() {
		envelope.Timestamp = time.Now()
	}
	row := gatewayEventRow{
		ID:          envelope.EventID,
		EventType:   strings.TrimSpace(envelope.Type),
		SessionID:   nullString(envelope.SessionID),
		SessionKey:  nullString(envelope.SessionKey),
		PayloadJSON: string(record.Payload),
		CreatedAt:   envelope.Timestamp,
	}
	_, err := store.db.NewInsert().Model(&row).
		On("CONFLICT(id) DO UPDATE").
		Set("event_type = EXCLUDED.event_type").
		Set("session_id = EXCLUDED.session_id").
		Set("session_key = EXCLUDED.session_key").
		Set("payload_json = EXCLUDED.payload_json").
		Set("created_at = EXCLUDED.created_at").
		Exec(ctx)
	if err != nil {
		return gatewayevents.Record{}, err
	}
	record.Envelope = envelope
	return record, nil
}

func (store *SQLiteEventStore) Query(ctx context.Context, filter gatewayevents.Filter) ([]gatewayevents.Record, error) {
	if store == nil || store.db == nil {
		return nil, errors.New("event store unavailable")
	}
	rows := make([]gatewayEventRow, 0)
	query := store.db.NewSelect().Model(&rows).Order("created_at DESC")
	if strings.TrimSpace(filter.SessionID) != "" {
		query = query.Where("session_id = ?", filter.SessionID)
	}
	if strings.TrimSpace(filter.SessionKey) != "" {
		query = query.Where("session_key = ?", filter.SessionKey)
	}
	if strings.TrimSpace(filter.Type) != "" {
		query = query.Where("event_type = ?", filter.Type)
	}
	if filter.Limit > 0 {
		query = query.Limit(filter.Limit)
	}
	if err := query.Scan(ctx); err != nil {
		return nil, err
	}
	result := make([]gatewayevents.Record, 0, len(rows))
	for _, row := range rows {
		envelope := gatewayevents.Envelope{
			EventID:    row.ID,
			Type:       row.EventType,
			SessionID:  stringOrEmpty(row.SessionID),
			SessionKey: stringOrEmpty(row.SessionKey),
			Timestamp:  row.CreatedAt,
		}
		record := gatewayevents.Record{
			Envelope: envelope,
			Payload:  []byte(row.PayloadJSON),
		}
		result = append(result, record)
	}
	return result, nil
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
