package providersrepo

import (
	"context"
	"database/sql"
	"dreamcreator/internal/infrastructure/persistence/sqlitedto"
	"errors"
	"strings"
	"time"

	"github.com/uptrace/bun"

	"dreamcreator/internal/domain/providers"
)

type SQLiteProviderRepository struct {
	db *bun.DB
}

type SQLiteModelRepository struct {
	db *bun.DB
}

type providerRow = sqlitedto.ProviderRow

type modelRow = sqlitedto.ModelRow

func NewSQLiteProviderRepository(db *bun.DB) *SQLiteProviderRepository {
	return &SQLiteProviderRepository{db: db}
}

func NewSQLiteModelRepository(db *bun.DB) *SQLiteModelRepository {
	return &SQLiteModelRepository{db: db}
}

func (repo *SQLiteProviderRepository) List(ctx context.Context) ([]providers.Provider, error) {
	rows := make([]providerRow, 0)
	if err := repo.db.NewSelect().Model(&rows).Order("created_at ASC").Scan(ctx); err != nil {
		return nil, err
	}

	result := make([]providers.Provider, 0, len(rows))
	for _, row := range rows {
		provider, err := providers.NewProvider(providers.ProviderParams{
			ID:        row.ID,
			Name:      row.Name,
			Type:      row.Type,
			Endpoint:  stringOrEmpty(row.Endpoint),
			Enabled:   row.Enabled,
			Builtin:   row.Builtin,
			CreatedAt: &row.CreatedAt,
			UpdatedAt: &row.UpdatedAt,
		})
		if err != nil {
			return nil, err
		}
		result = append(result, provider)
	}
	return result, nil
}

func (repo *SQLiteProviderRepository) Get(ctx context.Context, id string) (providers.Provider, error) {
	row := new(providerRow)
	if err := repo.db.NewSelect().Model(row).Where("id = ?", id).Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return providers.Provider{}, providers.ErrProviderNotFound
		}
		return providers.Provider{}, err
	}

	return providers.NewProvider(providers.ProviderParams{
		ID:        row.ID,
		Name:      row.Name,
		Type:      row.Type,
		Endpoint:  stringOrEmpty(row.Endpoint),
		Enabled:   row.Enabled,
		Builtin:   row.Builtin,
		CreatedAt: &row.CreatedAt,
		UpdatedAt: &row.UpdatedAt,
	})
}

func (repo *SQLiteProviderRepository) Save(ctx context.Context, provider providers.Provider) error {
	createdAt, updatedAt := normalizeTimes(provider.CreatedAt, provider.UpdatedAt)
	row := providerRow{
		ID:        provider.ID,
		Name:      provider.Name,
		Type:      string(provider.Type),
		Endpoint:  nullString(provider.Endpoint),
		Enabled:   provider.Enabled,
		Builtin:   provider.Builtin,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}

	_, err := repo.db.NewInsert().Model(&row).
		On("CONFLICT(id) DO UPDATE").
		Set("name = EXCLUDED.name").
		Set("type = EXCLUDED.type").
		Set("endpoint = EXCLUDED.endpoint").
		Set("enabled = EXCLUDED.enabled").
		Set("is_builtin = EXCLUDED.is_builtin").
		Set("updated_at = EXCLUDED.updated_at").
		Exec(ctx)
	return err
}

func (repo *SQLiteProviderRepository) Delete(ctx context.Context, id string) error {
	_, err := repo.db.NewDelete().Model((*providerRow)(nil)).Where("id = ?", id).Exec(ctx)
	return err
}

func (repo *SQLiteModelRepository) ListByProvider(ctx context.Context, providerID string) ([]providers.Model, error) {
	rows := make([]modelRow, 0)
	if err := repo.db.NewSelect().Model(&rows).Where("provider_id = ?", providerID).Order("created_at ASC").Scan(ctx); err != nil {
		return nil, err
	}

	result := make([]providers.Model, 0, len(rows))
	for _, row := range rows {
		model, err := providers.NewModel(providers.ModelParams{
			ID:                row.ID,
			ProviderID:        row.ProviderID,
			Name:              row.Name,
			DisplayName:       stringOrEmpty(row.DisplayName),
			CapabilitiesJSON:  stringOrEmpty(row.CapabilitiesJSON),
			ContextWindow:     intPointerOrNil(row.ContextWindow),
			MaxOutputTokens:   intPointerOrNil(row.MaxOutputTokens),
			SupportsTools:     boolPointerOrNil(row.SupportsTools),
			SupportsReasoning: boolPointerOrNil(row.SupportsReasoning),
			SupportsVision:    boolPointerOrNil(row.SupportsVision),
			SupportsAudio:     boolPointerOrNil(row.SupportsAudio),
			SupportsVideo:     boolPointerOrNil(row.SupportsVideo),
			Enabled:           row.Enabled,
			ShowInUI:          row.ShowInUI,
			CreatedAt:         &row.CreatedAt,
			UpdatedAt:         &row.UpdatedAt,
		})
		if err != nil {
			return nil, err
		}
		result = append(result, model)
	}
	return result, nil
}

func (repo *SQLiteModelRepository) Get(ctx context.Context, id string) (providers.Model, error) {
	row := new(modelRow)
	if err := repo.db.NewSelect().Model(row).Where("id = ?", id).Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return providers.Model{}, providers.ErrModelNotFound
		}
		return providers.Model{}, err
	}

	return providers.NewModel(providers.ModelParams{
		ID:                row.ID,
		ProviderID:        row.ProviderID,
		Name:              row.Name,
		DisplayName:       stringOrEmpty(row.DisplayName),
		CapabilitiesJSON:  stringOrEmpty(row.CapabilitiesJSON),
		ContextWindow:     intPointerOrNil(row.ContextWindow),
		MaxOutputTokens:   intPointerOrNil(row.MaxOutputTokens),
		SupportsTools:     boolPointerOrNil(row.SupportsTools),
		SupportsReasoning: boolPointerOrNil(row.SupportsReasoning),
		SupportsVision:    boolPointerOrNil(row.SupportsVision),
		SupportsAudio:     boolPointerOrNil(row.SupportsAudio),
		SupportsVideo:     boolPointerOrNil(row.SupportsVideo),
		Enabled:           row.Enabled,
		ShowInUI:          row.ShowInUI,
		CreatedAt:         &row.CreatedAt,
		UpdatedAt:         &row.UpdatedAt,
	})
}

func (repo *SQLiteModelRepository) Save(ctx context.Context, model providers.Model) error {
	createdAt, updatedAt := normalizeTimes(model.CreatedAt, model.UpdatedAt)
	row := modelRow{
		ID:                model.ID,
		ProviderID:        model.ProviderID,
		Name:              model.Name,
		DisplayName:       nullString(model.DisplayName),
		CapabilitiesJSON:  nullString(model.CapabilitiesJSON),
		ContextWindow:     nullIntPointer(model.ContextWindow),
		MaxOutputTokens:   nullIntPointer(model.MaxOutputTokens),
		SupportsTools:     nullBoolPointer(model.SupportsTools),
		SupportsReasoning: nullBoolPointer(model.SupportsReasoning),
		SupportsVision:    nullBoolPointer(model.SupportsVision),
		SupportsAudio:     nullBoolPointer(model.SupportsAudio),
		SupportsVideo:     nullBoolPointer(model.SupportsVideo),
		Enabled:           model.Enabled,
		ShowInUI:          model.ShowInUI,
		CreatedAt:         createdAt,
		UpdatedAt:         updatedAt,
	}

	_, err := repo.db.NewInsert().Model(&row).
		On("CONFLICT(id) DO UPDATE").
		Set("provider_id = EXCLUDED.provider_id").
		Set("name = EXCLUDED.name").
		Set("display_name = EXCLUDED.display_name").
		Set("capabilities_json = EXCLUDED.capabilities_json").
		Set("context_window_tokens = EXCLUDED.context_window_tokens").
		Set("max_output_tokens = EXCLUDED.max_output_tokens").
		Set("supports_tools = EXCLUDED.supports_tools").
		Set("supports_reasoning = EXCLUDED.supports_reasoning").
		Set("supports_vision = EXCLUDED.supports_vision").
		Set("supports_audio = EXCLUDED.supports_audio").
		Set("supports_video = EXCLUDED.supports_video").
		Set("enabled = EXCLUDED.enabled").
		Set("show_in_ui = EXCLUDED.show_in_ui").
		Set("updated_at = EXCLUDED.updated_at").
		Exec(ctx)
	return err
}

func (repo *SQLiteModelRepository) ReplaceByProvider(ctx context.Context, providerID string, models []providers.Model) error {
	tx, err := repo.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err = tx.NewDelete().Model((*modelRow)(nil)).Where("provider_id = ?", providerID).Exec(ctx); err != nil {
		return err
	}

	for _, model := range models {
		createdAt, updatedAt := normalizeTimes(model.CreatedAt, model.UpdatedAt)
		row := modelRow{
			ID:                model.ID,
			ProviderID:        model.ProviderID,
			Name:              model.Name,
			DisplayName:       nullString(model.DisplayName),
			CapabilitiesJSON:  nullString(model.CapabilitiesJSON),
			ContextWindow:     nullIntPointer(model.ContextWindow),
			MaxOutputTokens:   nullIntPointer(model.MaxOutputTokens),
			SupportsTools:     nullBoolPointer(model.SupportsTools),
			SupportsReasoning: nullBoolPointer(model.SupportsReasoning),
			SupportsVision:    nullBoolPointer(model.SupportsVision),
			SupportsAudio:     nullBoolPointer(model.SupportsAudio),
			SupportsVideo:     nullBoolPointer(model.SupportsVideo),
			Enabled:           model.Enabled,
			ShowInUI:          model.ShowInUI,
			CreatedAt:         createdAt,
			UpdatedAt:         updatedAt,
		}
		if _, err = tx.NewInsert().Model(&row).Exec(ctx); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (repo *SQLiteModelRepository) Delete(ctx context.Context, id string) error {
	_, err := repo.db.NewDelete().Model((*modelRow)(nil)).Where("id = ?", id).Exec(ctx)
	return err
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

func nullIntPointer(value *int) sql.NullInt64 {
	if value == nil || *value <= 0 {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(*value), Valid: true}
}

func nullBoolPointer(value *bool) sql.NullBool {
	if value == nil {
		return sql.NullBool{}
	}
	return sql.NullBool{Bool: *value, Valid: true}
}

func intPointerOrNil(value sql.NullInt64) *int {
	if !value.Valid {
		return nil
	}
	parsed := int(value.Int64)
	if parsed <= 0 {
		return nil
	}
	return &parsed
}

func boolPointerOrNil(value sql.NullBool) *bool {
	if !value.Valid {
		return nil
	}
	parsed := value.Bool
	return &parsed
}
