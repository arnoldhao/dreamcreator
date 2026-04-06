package threadrepo

import (
	"context"
	"database/sql"
	"dreamcreator/internal/infrastructure/persistence/sqlitedto"
	"errors"
	"time"

	"github.com/uptrace/bun"

	"dreamcreator/internal/domain/thread"
)

type SQLiteThreadRepository struct {
	db *bun.DB
}

type SQLiteThreadMessageRepository struct {
	db *bun.DB
}

type SQLiteThreadRunRepository struct {
	db *bun.DB
}

type SQLiteThreadRunEventRepository struct {
	db *bun.DB
}

type threadRow = sqlitedto.ThreadRow

type threadMessageRow = sqlitedto.ThreadMessageRow

type threadRunRow = sqlitedto.ThreadRunRow

type threadRunEventRow = sqlitedto.ThreadRunEventRow

func NewSQLiteThreadRepository(db *bun.DB) *SQLiteThreadRepository {
	return &SQLiteThreadRepository{db: db}
}

func NewSQLiteThreadMessageRepository(db *bun.DB) *SQLiteThreadMessageRepository {
	return &SQLiteThreadMessageRepository{db: db}
}

func NewSQLiteThreadRunRepository(db *bun.DB) *SQLiteThreadRunRepository {
	return &SQLiteThreadRunRepository{db: db}
}

func NewSQLiteThreadRunEventRepository(db *bun.DB) *SQLiteThreadRunEventRepository {
	return &SQLiteThreadRunEventRepository{db: db}
}

func (repo *SQLiteThreadRepository) List(ctx context.Context, includeDeleted bool) ([]thread.Thread, error) {
	rows := make([]threadRow, 0)
	query := repo.db.NewSelect().Model(&rows).Order("last_interactive_at DESC", "updated_at DESC")
	if !includeDeleted {
		query = query.Where("deleted_at IS NULL")
	}
	if err := query.Scan(ctx); err != nil {
		return nil, err
	}

	result := make([]thread.Thread, 0, len(rows))
	for _, row := range rows {
		item, err := thread.NewThread(thread.ThreadParams{
			ID:                row.ID,
			AgentID:           row.AgentID,
			AssistantID:       row.AssistantID,
			Title:             stringOrEmpty(row.Title),
			TitleIsDefault:    row.TitleIsDefault,
			TitleChangedBy:    thread.TitleChangedBy(stringOrEmpty(row.TitleChangedBy)),
			Status:            thread.Status(row.Status),
			CreatedAt:         &row.CreatedAt,
			UpdatedAt:         &row.UpdatedAt,
			LastInteractiveAt: &row.LastInteractiveAt,
			DeletedAt:         timeOrNil(row.DeletedAt),
			PurgeAfter:        timeOrNil(row.PurgeAfter),
		})
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, nil
}

func (repo *SQLiteThreadRepository) ListPurgeCandidates(ctx context.Context, before time.Time, limit int) ([]thread.Thread, error) {
	rows := make([]threadRow, 0)
	query := repo.db.NewSelect().Model(&rows).
		Where("deleted_at IS NOT NULL").
		Where("purge_after IS NOT NULL").
		Where("purge_after <= ?", before).
		Order("purge_after ASC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if err := query.Scan(ctx); err != nil {
		return nil, err
	}

	result := make([]thread.Thread, 0, len(rows))
	for _, row := range rows {
		item, err := thread.NewThread(thread.ThreadParams{
			ID:                row.ID,
			AgentID:           row.AgentID,
			AssistantID:       row.AssistantID,
			Title:             stringOrEmpty(row.Title),
			TitleIsDefault:    row.TitleIsDefault,
			TitleChangedBy:    thread.TitleChangedBy(stringOrEmpty(row.TitleChangedBy)),
			Status:            thread.Status(row.Status),
			CreatedAt:         &row.CreatedAt,
			UpdatedAt:         &row.UpdatedAt,
			LastInteractiveAt: &row.LastInteractiveAt,
			DeletedAt:         timeOrNil(row.DeletedAt),
			PurgeAfter:        timeOrNil(row.PurgeAfter),
		})
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, nil
}

func (repo *SQLiteThreadRepository) Get(ctx context.Context, id string) (thread.Thread, error) {
	row := new(threadRow)
	if err := repo.db.NewSelect().Model(row).Where("id = ?", id).Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return thread.Thread{}, thread.ErrThreadNotFound
		}
		return thread.Thread{}, err
	}

	return thread.NewThread(thread.ThreadParams{
		ID:                row.ID,
		AgentID:           row.AgentID,
		AssistantID:       row.AssistantID,
		Title:             stringOrEmpty(row.Title),
		TitleIsDefault:    row.TitleIsDefault,
		TitleChangedBy:    thread.TitleChangedBy(stringOrEmpty(row.TitleChangedBy)),
		Status:            thread.Status(row.Status),
		CreatedAt:         &row.CreatedAt,
		UpdatedAt:         &row.UpdatedAt,
		LastInteractiveAt: &row.LastInteractiveAt,
		DeletedAt:         timeOrNil(row.DeletedAt),
		PurgeAfter:        timeOrNil(row.PurgeAfter),
	})
}

func (repo *SQLiteThreadRepository) Save(ctx context.Context, item thread.Thread) error {
	createdAt, updatedAt := normalizeTimes(item.CreatedAt, item.UpdatedAt)
	lastInteractiveAt := normalizeTime(item.LastInteractiveAt)
	row := threadRow{
		ID:                item.ID,
		AgentID:           item.AgentID,
		AssistantID:       item.AssistantID,
		Title:             nullString(item.Title),
		TitleIsDefault:    item.TitleIsDefault,
		TitleChangedBy:    nullString(string(item.TitleChangedBy)),
		Status:            string(item.Status),
		CreatedAt:         createdAt,
		UpdatedAt:         updatedAt,
		LastInteractiveAt: lastInteractiveAt,
		DeletedAt:         nullTime(item.DeletedAt),
		PurgeAfter:        nullTime(item.PurgeAfter),
	}

	_, err := repo.db.NewInsert().Model(&row).
		On("CONFLICT(id) DO UPDATE").
		Set("agent_id = EXCLUDED.agent_id").
		Set("assistant_id = EXCLUDED.assistant_id").
		Set("title = EXCLUDED.title").
		Set("title_is_default = EXCLUDED.title_is_default").
		Set("title_changed_by = EXCLUDED.title_changed_by").
		Set("status = EXCLUDED.status").
		Set("updated_at = EXCLUDED.updated_at").
		Set("last_interactive_at = EXCLUDED.last_interactive_at").
		Set("deleted_at = EXCLUDED.deleted_at").
		Set("purge_after = EXCLUDED.purge_after").
		Exec(ctx)
	return err
}

func (repo *SQLiteThreadRepository) SetStatus(ctx context.Context, id string, status thread.Status, updatedAt time.Time) error {
	_, err := repo.db.NewUpdate().Model((*threadRow)(nil)).
		Set("status = ?", string(status)).
		Set("updated_at = ?", updatedAt).
		Where("id = ?", id).
		Exec(ctx)
	return err
}

func (repo *SQLiteThreadRepository) SoftDelete(ctx context.Context, id string, deletedAt, purgeAfter *time.Time) error {
	_, err := repo.db.NewUpdate().Model((*threadRow)(nil)).
		Set("deleted_at = ?", deletedAt).
		Set("purge_after = ?", purgeAfter).
		Where("id = ?", id).
		Exec(ctx)
	return err
}

func (repo *SQLiteThreadRepository) Restore(ctx context.Context, id string) error {
	_, err := repo.db.NewUpdate().Model((*threadRow)(nil)).
		Set("deleted_at = NULL").
		Set("purge_after = NULL").
		Where("id = ?", id).
		Exec(ctx)
	return err
}

func (repo *SQLiteThreadRepository) Purge(ctx context.Context, id string) error {
	_, err := repo.db.NewDelete().Model((*threadRow)(nil)).Where("id = ?", id).Exec(ctx)
	return err
}

func (repo *SQLiteThreadMessageRepository) ListByThread(ctx context.Context, threadID string, limit int) ([]thread.ThreadMessage, error) {
	rows := make([]threadMessageRow, 0)
	query := repo.db.NewSelect().Model(&rows).Where("thread_id = ?", threadID).OrderExpr("rowid ASC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if err := query.Scan(ctx); err != nil {
		return nil, err
	}

	result := make([]thread.ThreadMessage, 0, len(rows))
	for _, row := range rows {
		msg, err := thread.NewThreadMessage(thread.ThreadMessageParams{
			ID:        row.ID,
			ThreadID:  row.ThreadID,
			Kind:      thread.MessageKind(row.Kind),
			Role:      row.Role,
			Content:   row.Content,
			PartsJSON: row.PartsJSON,
			CreatedAt: &row.CreatedAt,
		})
		if err != nil {
			return nil, err
		}
		result = append(result, msg)
	}
	return result, nil
}

func (repo *SQLiteThreadMessageRepository) Append(ctx context.Context, message thread.ThreadMessage) error {
	row := threadMessageRow{
		ID:        message.ID,
		ThreadID:  message.ThreadID,
		Kind:      string(message.Kind),
		Role:      message.Role,
		Content:   message.Content,
		PartsJSON: message.PartsJSON,
		CreatedAt: normalizeTime(message.CreatedAt),
	}
	_, err := repo.db.NewInsert().Model(&row).
		On("CONFLICT(id) DO UPDATE").
		Set("thread_id = EXCLUDED.thread_id").
		Set("kind = EXCLUDED.kind").
		Set("role = EXCLUDED.role").
		Set("content = EXCLUDED.content").
		Set("parts_json = EXCLUDED.parts_json").
		Exec(ctx)
	return err
}

func (repo *SQLiteThreadMessageRepository) DeleteByThread(ctx context.Context, threadID string) error {
	_, err := repo.db.NewDelete().Model((*threadMessageRow)(nil)).Where("thread_id = ?", threadID).Exec(ctx)
	return err
}

func (repo *SQLiteThreadRunRepository) Get(ctx context.Context, id string) (thread.ThreadRun, error) {
	row := new(threadRunRow)
	if err := repo.db.NewSelect().Model(row).Where("id = ?", id).Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return thread.ThreadRun{}, thread.ErrRunNotFound
		}
		return thread.ThreadRun{}, err
	}

	return thread.NewThreadRun(thread.ThreadRunParams{
		ID:                 row.ID,
		ThreadID:           row.ThreadID,
		AssistantMessageID: row.AssistantMessageID,
		UserMessageID:      row.UserMessageID,
		AgentID:            row.AgentID,
		Status:             thread.RunStatus(row.Status),
		ContentPartial:     row.ContentPartial,
		CreatedAt:          &row.CreatedAt,
		UpdatedAt:          &row.UpdatedAt,
	})
}

func (repo *SQLiteThreadRunRepository) ListActiveByThread(ctx context.Context, threadID string) ([]thread.ThreadRun, error) {
	rows := make([]threadRunRow, 0)
	query := repo.db.NewSelect().Model(&rows).
		Where("thread_id = ?", threadID).
		Where("status = ?", string(thread.RunStatusActive)).
		Order("created_at DESC")
	if err := query.Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, thread.ErrRunNotFound
		}
		return nil, err
	}
	if len(rows) == 0 {
		return nil, thread.ErrRunNotFound
	}

	result := make([]thread.ThreadRun, 0, len(rows))
	for _, row := range rows {
		item, err := thread.NewThreadRun(thread.ThreadRunParams{
			ID:                 row.ID,
			ThreadID:           row.ThreadID,
			AssistantMessageID: row.AssistantMessageID,
			UserMessageID:      row.UserMessageID,
			AgentID:            row.AgentID,
			Status:             thread.RunStatus(row.Status),
			ContentPartial:     row.ContentPartial,
			CreatedAt:          &row.CreatedAt,
			UpdatedAt:          &row.UpdatedAt,
		})
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, nil
}

func (repo *SQLiteThreadRunRepository) ListByAgentID(ctx context.Context, agentID string, limit int) ([]thread.ThreadRun, error) {
	rows := make([]threadRunRow, 0)
	query := repo.db.NewSelect().Model(&rows).
		Where("agent_id = ?", agentID).
		Order("created_at DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if err := query.Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, thread.ErrRunNotFound
		}
		return nil, err
	}
	if len(rows) == 0 {
		return nil, thread.ErrRunNotFound
	}

	result := make([]thread.ThreadRun, 0, len(rows))
	for _, row := range rows {
		item, err := thread.NewThreadRun(thread.ThreadRunParams{
			ID:                 row.ID,
			ThreadID:           row.ThreadID,
			AssistantMessageID: row.AssistantMessageID,
			UserMessageID:      row.UserMessageID,
			AgentID:            row.AgentID,
			Status:             thread.RunStatus(row.Status),
			ContentPartial:     row.ContentPartial,
			CreatedAt:          &row.CreatedAt,
			UpdatedAt:          &row.UpdatedAt,
		})
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, nil
}

func (repo *SQLiteThreadRunRepository) CountActive(ctx context.Context) (int, error) {
	return repo.db.NewSelect().Model((*threadRunRow)(nil)).
		Where("status = ?", string(thread.RunStatusActive)).
		Count(ctx)
}

func (repo *SQLiteThreadRunRepository) CountActiveSince(ctx context.Context, since time.Time) (int, error) {
	query := repo.db.NewSelect().Model((*threadRunRow)(nil)).
		Where("status = ?", string(thread.RunStatusActive))
	if !since.IsZero() {
		query = query.Where("updated_at >= ?", since)
	}
	return query.Count(ctx)
}

func (repo *SQLiteThreadRunRepository) Save(ctx context.Context, run thread.ThreadRun) error {
	createdAt, updatedAt := normalizeTimes(run.CreatedAt, run.UpdatedAt)
	row := threadRunRow{
		ID:                 run.ID,
		ThreadID:           run.ThreadID,
		AssistantMessageID: run.AssistantMessageID,
		UserMessageID:      run.UserMessageID,
		AgentID:            run.AgentID,
		Status:             string(run.Status),
		ContentPartial:     run.ContentPartial,
		CreatedAt:          createdAt,
		UpdatedAt:          updatedAt,
	}

	_, err := repo.db.NewInsert().Model(&row).
		On("CONFLICT(id) DO UPDATE").
		Set("thread_id = EXCLUDED.thread_id").
		Set("assistant_message_id = EXCLUDED.assistant_message_id").
		Set("user_message_id = EXCLUDED.user_message_id").
		Set("agent_id = EXCLUDED.agent_id").
		Set("status = EXCLUDED.status").
		Set("content_partial = EXCLUDED.content_partial").
		Set("updated_at = EXCLUDED.updated_at").
		Exec(ctx)
	return err
}

func (repo *SQLiteThreadRunEventRepository) Append(ctx context.Context, event thread.ThreadRunEvent) (thread.ThreadRunEvent, error) {
	row := threadRunEventRow{
		RunID:       event.RunID,
		ThreadID:    event.ThreadID,
		EventType:   event.EventType,
		PayloadJSON: event.PayloadJSON,
		CreatedAt:   normalizeTime(event.CreatedAt),
	}
	if event.ID > 0 {
		row.ID = event.ID
	}

	_, err := repo.db.NewInsert().Model(&row).Exec(ctx)
	if err != nil {
		return thread.ThreadRunEvent{}, err
	}
	return thread.NewThreadRunEvent(thread.ThreadRunEventParams{
		ID:          row.ID,
		RunID:       row.RunID,
		ThreadID:    row.ThreadID,
		EventType:   row.EventType,
		PayloadJSON: row.PayloadJSON,
		CreatedAt:   &row.CreatedAt,
	})
}

func (repo *SQLiteThreadRunEventRepository) ListAfter(ctx context.Context, runID string, afterID int64, limit int) ([]thread.ThreadRunEvent, error) {
	rows := make([]threadRunEventRow, 0)
	query := repo.db.NewSelect().Model(&rows).
		Where("run_id = ?", runID).
		Where("id > ?", afterID).
		Order("id ASC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if err := query.Scan(ctx); err != nil {
		return nil, err
	}

	result := make([]thread.ThreadRunEvent, 0, len(rows))
	for _, row := range rows {
		item, err := thread.NewThreadRunEvent(thread.ThreadRunEventParams{
			ID:          row.ID,
			RunID:       row.RunID,
			ThreadID:    row.ThreadID,
			EventType:   row.EventType,
			PayloadJSON: row.PayloadJSON,
			CreatedAt:   &row.CreatedAt,
		})
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, nil
}

func (repo *SQLiteThreadRunEventRepository) ListByThread(
	ctx context.Context,
	threadID string,
	afterID int64,
	limit int,
	eventTypePrefix string,
) ([]thread.ThreadRunEvent, error) {
	rows := make([]threadRunEventRow, 0)
	query := repo.db.NewSelect().Model(&rows).
		Where("thread_id = ?", threadID).
		Where("id > ?", afterID).
		Order("id ASC")
	if eventTypePrefix != "" {
		query = query.Where("event_name LIKE ?", eventTypePrefix+"%")
	}
	if limit > 0 {
		query = query.Limit(limit)
	}
	if err := query.Scan(ctx); err != nil {
		return nil, err
	}

	result := make([]thread.ThreadRunEvent, 0, len(rows))
	for _, row := range rows {
		item, err := thread.NewThreadRunEvent(thread.ThreadRunEventParams{
			ID:          row.ID,
			RunID:       row.RunID,
			ThreadID:    row.ThreadID,
			EventType:   row.EventType,
			PayloadJSON: row.PayloadJSON,
			CreatedAt:   &row.CreatedAt,
		})
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, nil
}

func (repo *SQLiteThreadRunEventRepository) GetLastEventID(ctx context.Context, runID string) (int64, error) {
	row := new(threadRunEventRow)
	if err := repo.db.NewSelect().Model(row).
		Where("run_id = ?", runID).
		Order("id DESC").
		Limit(1).
		Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return 0, err
	}
	return row.ID, nil
}

func normalizeTimes(createdAt time.Time, updatedAt time.Time) (time.Time, time.Time) {
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	if updatedAt.IsZero() {
		updatedAt = createdAt
	}
	return createdAt, updatedAt
}

func normalizeTime(value time.Time) time.Time {
	if value.IsZero() {
		return time.Now()
	}
	return value
}

func nullString(value string) sql.NullString {
	if value == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: value, Valid: true}
}

func nullTime(value *time.Time) sql.NullTime {
	if value == nil || value.IsZero() {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *value, Valid: true}
}

func stringOrEmpty(value sql.NullString) string {
	if !value.Valid {
		return ""
	}
	return value.String
}

func timeOrNil(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	copy := value.Time
	return &copy
}
