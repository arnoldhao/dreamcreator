package telemetryrepo

import (
	"context"
	"database/sql"
	"errors"
	"time"

	apptelemetry "dreamcreator/internal/application/telemetry"
	"dreamcreator/internal/infrastructure/persistence/sqlitedto"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type SQLiteStateRepository struct {
	db           *bun.DB
	newInstallID func() string
}

type telemetryStateRow = sqlitedto.TelemetryStateRow

func NewSQLiteStateRepository(db *bun.DB) *SQLiteStateRepository {
	return &SQLiteStateRepository{
		db:           db,
		newInstallID: uuid.NewString,
	}
}

func (repo *SQLiteStateRepository) Ensure(ctx context.Context) (apptelemetry.State, error) {
	var state apptelemetry.State
	err := repo.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		row, err := repo.ensureRow(ctx, tx)
		if err != nil {
			return err
		}
		state = toState(row)
		return nil
	})
	return state, err
}

func (repo *SQLiteStateRepository) IncrementLaunchCount(ctx context.Context, at time.Time) (apptelemetry.State, error) {
	var state apptelemetry.State
	err := repo.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		row, err := repo.ensureRow(ctx, tx)
		if err != nil {
			return err
		}
		row.LaunchCount++
		row.UpdatedAt = at.UTC()
		if err := repo.saveRow(ctx, tx, row); err != nil {
			return err
		}
		state = toState(row)
		return nil
	})
	return state, err
}

func (repo *SQLiteStateRepository) MarkFirstProviderConfigured(ctx context.Context, at time.Time) (apptelemetry.State, bool, error) {
	return repo.markFirstTime(ctx, at, func(row *telemetryStateRow) *sql.NullTime {
		return &row.FirstProviderConfiguredAt
	})
}

func (repo *SQLiteStateRepository) MarkFirstChatCompleted(ctx context.Context, at time.Time) (apptelemetry.State, bool, error) {
	return repo.markFirstTime(ctx, at, func(row *telemetryStateRow) *sql.NullTime {
		return &row.FirstChatCompletedAt
	})
}

func (repo *SQLiteStateRepository) MarkFirstLibraryCompleted(ctx context.Context, at time.Time) (apptelemetry.State, bool, error) {
	return repo.markFirstTime(ctx, at, func(row *telemetryStateRow) *sql.NullTime {
		return &row.FirstLibraryCompletedAt
	})
}

func (repo *SQLiteStateRepository) markFirstTime(
	ctx context.Context,
	at time.Time,
	field func(row *telemetryStateRow) *sql.NullTime,
) (apptelemetry.State, bool, error) {
	var (
		state apptelemetry.State
		first bool
	)
	err := repo.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		row, err := repo.ensureRow(ctx, tx)
		if err != nil {
			return err
		}
		target := field(&row)
		if target != nil && !target.Valid {
			target.Valid = true
			target.Time = at.UTC()
			row.UpdatedAt = at.UTC()
			if err := repo.saveRow(ctx, tx, row); err != nil {
				return err
			}
			first = true
		}
		state = toState(row)
		return nil
	})
	return state, first, err
}

func (repo *SQLiteStateRepository) ensureRow(ctx context.Context, tx bun.Tx) (telemetryStateRow, error) {
	row := telemetryStateRow{}
	if err := tx.NewSelect().Model(&row).Where("id = 1").Scan(ctx); err == nil {
		return row, nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return telemetryStateRow{}, err
	}

	now := time.Now().UTC()
	row = telemetryStateRow{
		ID:               1,
		InstallID:        repo.newInstallID(),
		InstallCreatedAt: now,
		LaunchCount:      0,
		UpdatedAt:        now,
	}
	if err := repo.saveRow(ctx, tx, row); err != nil {
		return telemetryStateRow{}, err
	}
	return row, nil
}

func (repo *SQLiteStateRepository) saveRow(ctx context.Context, tx bun.Tx, row telemetryStateRow) error {
	_, err := tx.NewInsert().Model(&row).
		On("CONFLICT(id) DO UPDATE").
		Set("install_id = EXCLUDED.install_id").
		Set("install_created_at = EXCLUDED.install_created_at").
		Set("launch_count = EXCLUDED.launch_count").
		Set("first_provider_configured_at = EXCLUDED.first_provider_configured_at").
		Set("first_chat_completed_at = EXCLUDED.first_chat_completed_at").
		Set("first_library_completed_at = EXCLUDED.first_library_completed_at").
		Set("updated_at = EXCLUDED.updated_at").
		Exec(ctx)
	return err
}

func toState(row telemetryStateRow) apptelemetry.State {
	return apptelemetry.State{
		InstallID:                 row.InstallID,
		InstallCreatedAt:          row.InstallCreatedAt,
		LaunchCount:               row.LaunchCount,
		FirstProviderConfiguredAt: nullTimePtr(row.FirstProviderConfiguredAt),
		FirstChatCompletedAt:      nullTimePtr(row.FirstChatCompletedAt),
		FirstLibraryCompletedAt:   nullTimePtr(row.FirstLibraryCompletedAt),
	}
}

func nullTimePtr(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	timestamp := value.Time
	return &timestamp
}
