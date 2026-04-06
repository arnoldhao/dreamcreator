package externaltoolsrepo

import (
	"context"
	"database/sql"
	"dreamcreator/internal/infrastructure/persistence/sqlitedto"
	"errors"
	"strings"
	"time"

	"github.com/uptrace/bun"

	"dreamcreator/internal/domain/externaltools"
)

type SQLiteExternalToolRepository struct {
	db *bun.DB
}

type externalToolRow = sqlitedto.ExternalToolRow

func NewSQLiteExternalToolRepository(db *bun.DB) *SQLiteExternalToolRepository {
	return &SQLiteExternalToolRepository{db: db}
}

func (repo *SQLiteExternalToolRepository) List(ctx context.Context) ([]externaltools.ExternalTool, error) {
	rows := make([]externalToolRow, 0)
	if err := repo.db.NewSelect().Model(&rows).Order("name ASC").Scan(ctx); err != nil {
		return nil, err
	}
	result := make([]externaltools.ExternalTool, 0, len(rows))
	for _, row := range rows {
		item, err := externaltools.NewExternalTool(externaltools.ExternalToolParams{
			Name:        row.Name,
			ExecPath:    stringOrEmpty(row.ExecPath),
			Version:     stringOrEmpty(row.Version),
			Status:      stringOrEmpty(row.Status),
			InstalledAt: timeOrNil(row.InstalledAt),
			UpdatedAt:   &row.UpdatedAt,
		})
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, nil
}

func (repo *SQLiteExternalToolRepository) Get(ctx context.Context, name string) (externaltools.ExternalTool, error) {
	row := new(externalToolRow)
	if err := repo.db.NewSelect().Model(row).Where("name = ?", name).Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return externaltools.ExternalTool{}, externaltools.ErrToolNotFound
		}
		return externaltools.ExternalTool{}, err
	}
	return externaltools.NewExternalTool(externaltools.ExternalToolParams{
		Name:        row.Name,
		ExecPath:    stringOrEmpty(row.ExecPath),
		Version:     stringOrEmpty(row.Version),
		Status:      stringOrEmpty(row.Status),
		InstalledAt: timeOrNil(row.InstalledAt),
		UpdatedAt:   &row.UpdatedAt,
	})
}

func (repo *SQLiteExternalToolRepository) Save(ctx context.Context, tool externaltools.ExternalTool) error {
	updatedAt := tool.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = time.Now()
	}
	row := externalToolRow{
		Name:        string(tool.Name),
		ExecPath:    nullString(tool.ExecPath),
		Version:     nullString(tool.Version),
		Status:      nullString(string(tool.Status)),
		InstalledAt: nullTime(tool.InstalledAt),
		UpdatedAt:   updatedAt,
	}
	_, err := repo.db.NewInsert().Model(&row).
		On("CONFLICT(name) DO UPDATE").
		Set("exec_path = EXCLUDED.exec_path").
		Set("version = EXCLUDED.version").
		Set("status = EXCLUDED.status").
		Set("installed_at = EXCLUDED.installed_at").
		Set("updated_at = EXCLUDED.updated_at").
		Exec(ctx)
	return err
}

func (repo *SQLiteExternalToolRepository) Delete(ctx context.Context, name string) error {
	_, err := repo.db.NewDelete().Model((*externalToolRow)(nil)).Where("name = ?", name).Exec(ctx)
	return err
}

func nullString(value string) sql.NullString {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: trimmed, Valid: true}
}

func stringOrEmpty(value sql.NullString) string {
	if value.Valid {
		return value.String
	}
	return ""
}

func nullTime(value *time.Time) sql.NullTime {
	if value == nil || value.IsZero() {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *value, Valid: true}
}

func timeOrNil(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	return &value.Time
}
