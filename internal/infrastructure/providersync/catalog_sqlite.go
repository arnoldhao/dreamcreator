package providersync

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/uptrace/bun"
)

type SQLiteModelsDevCatalogRepository struct {
	db *bun.DB
}

type modelsDevCatalogRow struct {
	bun.BaseModel `bun:"table:models_dev_catalog"`

	ID                string         `bun:"id,pk"`
	ProviderKey       string         `bun:"provider_key"`
	ModelName         string         `bun:"model_name"`
	DisplayName       sql.NullString `bun:"display_name"`
	CapabilitiesJSON  sql.NullString `bun:"capabilities_json"`
	ContextWindow     sql.NullInt64  `bun:"context_window_tokens"`
	MaxOutputTokens   sql.NullInt64  `bun:"max_output_tokens"`
	SupportsTools     sql.NullBool   `bun:"supports_tools"`
	SupportsReasoning sql.NullBool   `bun:"supports_reasoning"`
	SupportsVision    sql.NullBool   `bun:"supports_vision"`
	SupportsAudio     sql.NullBool   `bun:"supports_audio"`
	SupportsVideo     sql.NullBool   `bun:"supports_video"`
	CreatedAt         time.Time      `bun:"created_at"`
	UpdatedAt         time.Time      `bun:"updated_at"`
}

func NewSQLiteModelsDevCatalogRepository(db *bun.DB) *SQLiteModelsDevCatalogRepository {
	return &SQLiteModelsDevCatalogRepository{db: db}
}

func (repo *SQLiteModelsDevCatalogRepository) Count(ctx context.Context) (int, error) {
	return repo.db.NewSelect().Model((*modelsDevCatalogRow)(nil)).Count(ctx)
}

func (repo *SQLiteModelsDevCatalogRepository) ReplaceAll(ctx context.Context, entries []ModelsDevCatalogEntry) error {
	tx, err := repo.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err = tx.NewDelete().Model((*modelsDevCatalogRow)(nil)).Where("1 = 1").Exec(ctx); err != nil {
		return err
	}

	for _, entry := range entries {
		createdAt, updatedAt := normalizeCatalogTimes(entry.CreatedAt, entry.UpdatedAt)
		row := modelsDevCatalogRow{
			ID:                strings.TrimSpace(entry.ID),
			ProviderKey:       strings.ToLower(strings.TrimSpace(entry.ProviderKey)),
			ModelName:         strings.TrimSpace(entry.ModelName),
			DisplayName:       nullCatalogString(entry.DisplayName),
			CapabilitiesJSON:  nullCatalogString(entry.CapabilitiesJSON),
			ContextWindow:     nullCatalogIntPointer(entry.ContextWindow),
			MaxOutputTokens:   nullCatalogIntPointer(entry.MaxOutputTokens),
			SupportsTools:     nullCatalogBoolPointer(entry.SupportsTools),
			SupportsReasoning: nullCatalogBoolPointer(entry.SupportsReasoning),
			SupportsVision:    nullCatalogBoolPointer(entry.SupportsVision),
			SupportsAudio:     nullCatalogBoolPointer(entry.SupportsAudio),
			SupportsVideo:     nullCatalogBoolPointer(entry.SupportsVideo),
			CreatedAt:         createdAt,
			UpdatedAt:         updatedAt,
		}
		if _, err = tx.NewInsert().Model(&row).Exec(ctx); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (repo *SQLiteModelsDevCatalogRepository) ListByProviderKeys(ctx context.Context, providerKeys []string) ([]ModelsDevCatalogEntry, error) {
	normalized := normalizeCatalogStrings(providerKeys)
	if len(normalized) == 0 {
		return []ModelsDevCatalogEntry{}, nil
	}
	rows := make([]modelsDevCatalogRow, 0)
	if err := repo.db.NewSelect().
		Model(&rows).
		Where("provider_key IN (?)", bun.In(normalized)).
		Order("provider_key ASC, model_name ASC").
		Scan(ctx); err != nil {
		return nil, err
	}
	return catalogRowsToEntries(rows), nil
}

func (repo *SQLiteModelsDevCatalogRepository) ListByModelNames(ctx context.Context, modelNames []string) ([]ModelsDevCatalogEntry, error) {
	normalized := normalizeCatalogStrings(modelNames)
	if len(normalized) == 0 {
		return []ModelsDevCatalogEntry{}, nil
	}
	rows := make([]modelsDevCatalogRow, 0)
	if err := repo.db.NewSelect().
		Model(&rows).
		Where("LOWER(model_name) IN (?)", bun.In(normalized)).
		Order("provider_key ASC, model_name ASC").
		Scan(ctx); err != nil {
		return nil, err
	}
	return catalogRowsToEntries(rows), nil
}

func catalogRowsToEntries(rows []modelsDevCatalogRow) []ModelsDevCatalogEntry {
	result := make([]ModelsDevCatalogEntry, 0, len(rows))
	for _, row := range rows {
		result = append(result, ModelsDevCatalogEntry{
			ID:                row.ID,
			ProviderKey:       row.ProviderKey,
			ModelName:         row.ModelName,
			DisplayName:       catalogStringOrEmpty(row.DisplayName),
			CapabilitiesJSON:  catalogStringOrEmpty(row.CapabilitiesJSON),
			ContextWindow:     catalogIntPointerOrNil(row.ContextWindow),
			MaxOutputTokens:   catalogIntPointerOrNil(row.MaxOutputTokens),
			SupportsTools:     catalogBoolPointerOrNil(row.SupportsTools),
			SupportsReasoning: catalogBoolPointerOrNil(row.SupportsReasoning),
			SupportsVision:    catalogBoolPointerOrNil(row.SupportsVision),
			SupportsAudio:     catalogBoolPointerOrNil(row.SupportsAudio),
			SupportsVideo:     catalogBoolPointerOrNil(row.SupportsVideo),
			CreatedAt:         row.CreatedAt,
			UpdatedAt:         row.UpdatedAt,
		})
	}
	return result
}

func normalizeCatalogTimes(createdAt time.Time, updatedAt time.Time) (time.Time, time.Time) {
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	if updatedAt.IsZero() {
		updatedAt = createdAt
	}
	return createdAt, updatedAt
}

func normalizeCatalogStrings(values []string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		normalized := strings.ToLower(strings.TrimSpace(value))
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		result = append(result, normalized)
	}
	return result
}

func nullCatalogString(value string) sql.NullString {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: trimmed, Valid: true}
}

func catalogStringOrEmpty(value sql.NullString) string {
	if value.Valid {
		return value.String
	}
	return ""
}

func nullCatalogIntPointer(value *int) sql.NullInt64 {
	if value == nil || *value <= 0 {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(*value), Valid: true}
}

func catalogIntPointerOrNil(value sql.NullInt64) *int {
	if !value.Valid {
		return nil
	}
	parsed := int(value.Int64)
	if parsed <= 0 {
		return nil
	}
	return &parsed
}

func nullCatalogBoolPointer(value *bool) sql.NullBool {
	if value == nil {
		return sql.NullBool{}
	}
	return sql.NullBool{Bool: *value, Valid: true}
}

func catalogBoolPointerOrNil(value sql.NullBool) *bool {
	if !value.Valid {
		return nil
	}
	parsed := value.Bool
	return &parsed
}
