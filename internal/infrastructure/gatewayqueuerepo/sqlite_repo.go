package gatewayqueuerepo

import (
	"context"
	"dreamcreator/internal/infrastructure/persistence/sqlitedto"
	"errors"
	"strings"
	"time"

	"github.com/uptrace/bun"

	"dreamcreator/internal/application/gateway/queue"
)

type SQLiteQueueStore struct {
	db *bun.DB
}

type queueRow = sqlitedto.QueueRow

func NewSQLiteQueueStore(db *bun.DB) *SQLiteQueueStore {
	return &SQLiteQueueStore{db: db}
}

func (store *SQLiteQueueStore) Save(ctx context.Context, ticket queue.Ticket) error {
	if store == nil || store.db == nil {
		return errors.New("queue store unavailable")
	}
	row := queueRow{
		TicketID:   ticket.TicketID,
		SessionKey: strings.TrimSpace(ticket.SessionKey),
		Lane:       strings.TrimSpace(ticket.Lane),
		Status:     strings.TrimSpace(ticket.Status),
		Position:   ticket.Position,
		CreatedAt:  time.Now(),
	}
	if row.CreatedAt.IsZero() {
		row.CreatedAt = time.Now()
	}
	_, err := store.db.NewInsert().Model(&row).
		On("CONFLICT(ticket_id) DO UPDATE").
		Set("session_key = EXCLUDED.session_key").
		Set("lane = EXCLUDED.lane").
		Set("status = EXCLUDED.status").
		Set("position = EXCLUDED.position").
		Set("created_at = EXCLUDED.created_at").
		Exec(ctx)
	return err
}
