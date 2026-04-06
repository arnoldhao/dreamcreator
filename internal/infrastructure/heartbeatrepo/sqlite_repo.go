package heartbeatrepo

import (
	"context"
	"database/sql"
	"dreamcreator/internal/infrastructure/persistence/sqlitedto"
	"errors"
	"strings"
	"time"

	"github.com/uptrace/bun"

	"dreamcreator/internal/application/gateway/heartbeat"
)

type SQLiteHeartbeatStore struct {
	db *bun.DB
}

type heartbeatEventRow = sqlitedto.HeartbeatEventRow

func NewSQLiteHeartbeatStore(db *bun.DB) *SQLiteHeartbeatStore {
	return &SQLiteHeartbeatStore{db: db}
}

func (store *SQLiteHeartbeatStore) Save(ctx context.Context, event heartbeat.Event) error {
	if store == nil || store.db == nil {
		return errors.New("heartbeat store unavailable")
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}
	row := heartbeatEventRow{
		ID:          strings.TrimSpace(event.ID),
		SessionKey:  strings.TrimSpace(event.SessionKey),
		ThreadID:    strings.TrimSpace(event.ThreadID),
		Status:      strings.TrimSpace(string(event.Status)),
		Message:     strings.TrimSpace(event.Message),
		ErrorText:   strings.TrimSpace(event.Error),
		ContentHash: strings.TrimSpace(event.ContentHash),
		Reason:      strings.TrimSpace(event.Reason),
		Source:      strings.TrimSpace(event.Source),
		RunID:       strings.TrimSpace(event.RunID),
		CreatedAt:   event.CreatedAt,
	}
	_, err := store.db.NewInsert().Model(&row).
		On("CONFLICT(id) DO UPDATE").
		Set("session_key = EXCLUDED.session_key").
		Set("thread_id = EXCLUDED.thread_id").
		Set("status = EXCLUDED.status").
		Set("message = EXCLUDED.message").
		Set("error = EXCLUDED.error").
		Set("content_hash = EXCLUDED.content_hash").
		Set("reason = EXCLUDED.reason").
		Set("source = EXCLUDED.source").
		Set("run_id = EXCLUDED.run_id").
		Set("created_at = EXCLUDED.created_at").
		Exec(ctx)
	return err
}

func (store *SQLiteHeartbeatStore) Last(ctx context.Context, sessionKey string) (heartbeat.Event, error) {
	if store == nil || store.db == nil {
		return heartbeat.Event{}, errors.New("heartbeat store unavailable")
	}
	key := strings.TrimSpace(sessionKey)
	if key == "" {
		return heartbeat.Event{}, errors.New("session key is required")
	}
	var row heartbeatEventRow
	if err := store.db.NewSelect().Model(&row).
		Where("session_key = ?", key).
		Order("created_at DESC").
		Limit(1).
		Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return heartbeat.Event{}, heartbeat.ErrEventNotFound
		}
		return heartbeat.Event{}, err
	}
	return toHeartbeatEvent(row), nil
}

func (store *SQLiteHeartbeatStore) HasDuplicate(ctx context.Context, sessionKey string, contentHash string, since time.Time) (bool, error) {
	if store == nil || store.db == nil {
		return false, errors.New("heartbeat store unavailable")
	}
	key := strings.TrimSpace(sessionKey)
	hash := strings.TrimSpace(contentHash)
	if key == "" || hash == "" {
		return false, nil
	}
	count, err := store.db.NewSelect().Model((*heartbeatEventRow)(nil)).
		Where("session_key = ?", key).
		Where("content_hash = ?", hash).
		Where("created_at >= ?", since).
		Count(ctx)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func toHeartbeatEvent(row heartbeatEventRow) heartbeat.Event {
	return heartbeat.Event{
		ID:          row.ID,
		SessionKey:  row.SessionKey,
		ThreadID:    row.ThreadID,
		Status:      heartbeat.EventStatus(strings.TrimSpace(row.Status)),
		Message:     row.Message,
		Error:       row.ErrorText,
		ContentHash: row.ContentHash,
		Reason:      strings.TrimSpace(row.Reason),
		Source:      strings.TrimSpace(row.Source),
		RunID:       strings.TrimSpace(row.RunID),
		CreatedAt:   row.CreatedAt,
	}
}
