package noticerepo

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	domainnotice "dreamcreator/internal/domain/notice"
	"dreamcreator/internal/infrastructure/persistence/sqlitedto"

	"github.com/uptrace/bun"
)

type SQLiteStore struct {
	db *bun.DB
}

type noticeRow = sqlitedto.NoticeRow

func NewSQLiteStore(db *bun.DB) *SQLiteStore {
	return &SQLiteStore{db: db}
}

func (store *SQLiteStore) Save(ctx context.Context, item domainnotice.Notice) error {
	if store == nil || store.db == nil {
		return errors.New("notice store unavailable")
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = time.Now()
	}
	if item.UpdatedAt.IsZero() {
		item.UpdatedAt = item.CreatedAt
	}
	if item.LastOccurredAt.IsZero() {
		item.LastOccurredAt = item.UpdatedAt
	}
	row, err := toNoticeRow(item)
	if err != nil {
		return err
	}
	_, err = store.db.NewInsert().Model(&row).
		On("CONFLICT(id) DO UPDATE").
		Set("kind = EXCLUDED.kind").
		Set("category = EXCLUDED.category").
		Set("code = EXCLUDED.code").
		Set("severity = EXCLUDED.severity").
		Set("status = EXCLUDED.status").
		Set("i18n_json = EXCLUDED.i18n_json").
		Set("source_json = EXCLUDED.source_json").
		Set("action_json = EXCLUDED.action_json").
		Set("surfaces_json = EXCLUDED.surfaces_json").
		Set("dedup_key = EXCLUDED.dedup_key").
		Set("occurrence_count = EXCLUDED.occurrence_count").
		Set("metadata_json = EXCLUDED.metadata_json").
		Set("updated_at = EXCLUDED.updated_at").
		Set("last_occurred_at = EXCLUDED.last_occurred_at").
		Set("read_at = EXCLUDED.read_at").
		Set("archived_at = EXCLUDED.archived_at").
		Set("expires_at = EXCLUDED.expires_at").
		Exec(ctx)
	return err
}

func (store *SQLiteStore) Get(ctx context.Context, id string) (domainnotice.Notice, error) {
	if store == nil || store.db == nil {
		return domainnotice.Notice{}, errors.New("notice store unavailable")
	}
	var row noticeRow
	if err := store.db.NewSelect().Model(&row).
		Where("id = ?", strings.TrimSpace(id)).
		Limit(1).
		Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domainnotice.Notice{}, domainnotice.ErrNoticeNotFound
		}
		return domainnotice.Notice{}, err
	}
	return toNotice(row)
}

func (store *SQLiteStore) List(ctx context.Context, filter domainnotice.ListFilter) ([]domainnotice.Notice, error) {
	if store == nil || store.db == nil {
		return nil, errors.New("notice store unavailable")
	}
	rows := make([]noticeRow, 0)
	query := store.db.NewSelect().Model(&rows).
		Order("last_occurred_at DESC")
	if len(filter.Statuses) > 0 {
		query = query.Where("status IN (?)", bun.In(stringSliceStatuses(filter.Statuses)))
	}
	if len(filter.Kinds) > 0 {
		query = query.Where("kind IN (?)", bun.In(stringSliceKinds(filter.Kinds)))
	}
	if len(filter.Categories) > 0 {
		query = query.Where("category IN (?)", bun.In(stringSliceCategories(filter.Categories)))
	}
	if len(filter.Severities) > 0 {
		query = query.Where("severity IN (?)", bun.In(stringSliceSeverities(filter.Severities)))
	}
	if surface := strings.TrimSpace(string(filter.Surface)); surface != "" {
		query = query.Where("surfaces_json LIKE ?", `%`+surface+`%`)
	}
	if search := strings.TrimSpace(filter.Query); search != "" {
		pattern := "%" + search + "%"
		query = query.Where("(code LIKE ? OR i18n_json LIKE ? OR source_json LIKE ?)", pattern, pattern, pattern)
	}
	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}
	query = query.Limit(limit)
	if err := query.Scan(ctx); err != nil {
		return nil, err
	}
	result := make([]domainnotice.Notice, 0, len(rows))
	for _, row := range rows {
		item, err := toNotice(row)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, nil
}

func (store *SQLiteStore) CountUnread(ctx context.Context, surface domainnotice.Surface) (int, error) {
	if store == nil || store.db == nil {
		return 0, errors.New("notice store unavailable")
	}
	query := store.db.NewSelect().Model((*noticeRow)(nil)).
		Where("status = ?", string(domainnotice.StatusUnread))
	if trimmed := strings.TrimSpace(string(surface)); trimmed != "" {
		query = query.Where("surfaces_json LIKE ?", `%`+trimmed+`%`)
	}
	return query.Count(ctx)
}

func (store *SQLiteStore) FindByDedupKey(ctx context.Context, dedupKey string) (domainnotice.Notice, error) {
	if store == nil || store.db == nil {
		return domainnotice.Notice{}, errors.New("notice store unavailable")
	}
	trimmed := strings.TrimSpace(dedupKey)
	if trimmed == "" {
		return domainnotice.Notice{}, domainnotice.ErrNoticeNotFound
	}
	var row noticeRow
	if err := store.db.NewSelect().Model(&row).
		Where("dedup_key = ?", trimmed).
		Order("last_occurred_at DESC").
		Limit(1).
		Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domainnotice.Notice{}, domainnotice.ErrNoticeNotFound
		}
		return domainnotice.Notice{}, err
	}
	return toNotice(row)
}

func (store *SQLiteStore) MarkRead(ctx context.Context, ids []string, read bool, at time.Time) error {
	if store == nil || store.db == nil {
		return errors.New("notice store unavailable")
	}
	if len(ids) == 0 {
		return nil
	}
	status := domainnotice.StatusRead
	readAt := sql.NullTime{Time: at, Valid: true}
	if !read {
		status = domainnotice.StatusUnread
		readAt = sql.NullTime{}
	}
	_, err := store.db.NewUpdate().Model((*noticeRow)(nil)).
		Set("status = ?", string(status)).
		Set("read_at = ?", readAt).
		Set("updated_at = ?", at).
		Where("id IN (?)", bun.In(ids)).
		Exec(ctx)
	return err
}

func (store *SQLiteStore) Archive(ctx context.Context, ids []string, archived bool, at time.Time) error {
	if store == nil || store.db == nil {
		return errors.New("notice store unavailable")
	}
	if len(ids) == 0 {
		return nil
	}
	status := domainnotice.StatusRead
	archivedAt := sql.NullTime{}
	if archived {
		status = domainnotice.StatusArchived
		archivedAt = sql.NullTime{Time: at, Valid: true}
	}
	_, err := store.db.NewUpdate().Model((*noticeRow)(nil)).
		Set("status = ?", string(status)).
		Set("archived_at = ?", archivedAt).
		Set("updated_at = ?", at).
		Where("id IN (?)", bun.In(ids)).
		Exec(ctx)
	return err
}

func (store *SQLiteStore) MarkAllRead(ctx context.Context, surface domainnotice.Surface, at time.Time) error {
	if store == nil || store.db == nil {
		return errors.New("notice store unavailable")
	}
	query := store.db.NewUpdate().Model((*noticeRow)(nil)).
		Set("status = ?", string(domainnotice.StatusRead)).
		Set("read_at = ?", sql.NullTime{Time: at, Valid: true}).
		Set("updated_at = ?", at).
		Where("status = ?", string(domainnotice.StatusUnread))
	if trimmed := strings.TrimSpace(string(surface)); trimmed != "" {
		query = query.Where("surfaces_json LIKE ?", `%`+trimmed+`%`)
	}
	_, err := query.Exec(ctx)
	return err
}

func toNoticeRow(item domainnotice.Notice) (noticeRow, error) {
	i18nJSON, err := json.Marshal(item.I18n)
	if err != nil {
		return noticeRow{}, err
	}
	sourceJSON, err := json.Marshal(item.Source)
	if err != nil {
		return noticeRow{}, err
	}
	actionJSON, err := json.Marshal(item.Action)
	if err != nil {
		return noticeRow{}, err
	}
	surfacesJSON, err := json.Marshal(item.Surfaces)
	if err != nil {
		return noticeRow{}, err
	}
	metadataJSON, err := json.Marshal(item.Metadata)
	if err != nil {
		return noticeRow{}, err
	}
	return noticeRow{
		ID:              item.ID,
		Kind:            string(item.Kind),
		Category:        string(item.Category),
		Code:            item.Code,
		Severity:        string(item.Severity),
		Status:          string(item.Status),
		I18nJSON:        string(i18nJSON),
		SourceJSON:      string(sourceJSON),
		ActionJSON:      string(actionJSON),
		SurfacesJSON:    string(surfacesJSON),
		DedupKey:        item.DedupKey,
		OccurrenceCount: item.OccurrenceCount,
		MetadataJSON:    string(metadataJSON),
		CreatedAt:       item.CreatedAt,
		UpdatedAt:       item.UpdatedAt,
		LastOccurredAt:  item.LastOccurredAt,
		ReadAt:          nullTime(item.ReadAt),
		ArchivedAt:      nullTime(item.ArchivedAt),
		ExpiresAt:       nullTime(item.ExpiresAt),
	}, nil
}

func toNotice(row noticeRow) (domainnotice.Notice, error) {
	item := domainnotice.Notice{
		ID:              row.ID,
		Kind:            domainnotice.Kind(strings.TrimSpace(row.Kind)),
		Category:        domainnotice.Category(strings.TrimSpace(row.Category)),
		Code:            strings.TrimSpace(row.Code),
		Severity:        domainnotice.Severity(strings.TrimSpace(row.Severity)),
		Status:          domainnotice.Status(strings.TrimSpace(row.Status)),
		DedupKey:        strings.TrimSpace(row.DedupKey),
		OccurrenceCount: row.OccurrenceCount,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
		LastOccurredAt:  row.LastOccurredAt,
		ReadAt:          ptrTime(row.ReadAt),
		ArchivedAt:      ptrTime(row.ArchivedAt),
		ExpiresAt:       ptrTime(row.ExpiresAt),
	}
	if err := decodeJSON(row.I18nJSON, &item.I18n); err != nil {
		return domainnotice.Notice{}, err
	}
	if err := decodeJSON(row.SourceJSON, &item.Source); err != nil {
		return domainnotice.Notice{}, err
	}
	if err := decodeJSON(row.ActionJSON, &item.Action); err != nil {
		return domainnotice.Notice{}, err
	}
	if err := decodeJSON(row.SurfacesJSON, &item.Surfaces); err != nil {
		return domainnotice.Notice{}, err
	}
	if err := decodeJSON(row.MetadataJSON, &item.Metadata); err != nil {
		return domainnotice.Notice{}, err
	}
	if item.Metadata == nil {
		item.Metadata = map[string]any{}
	}
	return item, nil
}

func decodeJSON(value string, target any) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return json.Unmarshal([]byte(trimmed), target)
}

func nullTime(value *time.Time) sql.NullTime {
	if value == nil || value.IsZero() {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *value, Valid: true}
}

func ptrTime(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	result := value.Time
	return &result
}

func stringSliceStatuses(values []domainnotice.Status) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, string(value))
	}
	return result
}

func stringSliceKinds(values []domainnotice.Kind) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, string(value))
	}
	return result
}

func stringSliceCategories(values []domainnotice.Category) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, string(value))
	}
	return result
}

func stringSliceSeverities(values []domainnotice.Severity) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, string(value))
	}
	return result
}
