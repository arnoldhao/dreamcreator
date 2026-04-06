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

type SQLiteProviderSecretRepository struct {
	db *bun.DB
}

type providerSecretRow = sqlitedto.ProviderSecretRow

func NewSQLiteProviderSecretRepository(db *bun.DB) *SQLiteProviderSecretRepository {
	return &SQLiteProviderSecretRepository{db: db}
}

func (repo *SQLiteProviderSecretRepository) GetByProviderID(ctx context.Context, providerID string) (providers.ProviderSecret, error) {
	row := new(providerSecretRow)
	if err := repo.db.NewSelect().Model(row).
		Where("provider_id = ?", providerID).
		Order("created_at DESC").
		Limit(1).
		Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return providers.ProviderSecret{}, providers.ErrProviderSecretNotFound
		}
		return providers.ProviderSecret{}, err
	}

	return providers.NewProviderSecret(providers.ProviderSecretParams{
		ID:         row.ID,
		ProviderID: row.ProviderID,
		APIKey:     stringOrEmpty(row.KeyRef),
		OrgRef:     stringOrEmpty(row.OrgRef),
		CreatedAt:  &row.CreatedAt,
	})
}

func (repo *SQLiteProviderSecretRepository) Save(ctx context.Context, secret providers.ProviderSecret) error {
	providerID := strings.TrimSpace(secret.ProviderID)
	if providerID == "" {
		return providers.ErrInvalidSecret
	}

	createdAt := secret.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}

	row := providerSecretRow{
		ID:         secret.ID,
		ProviderID: providerID,
		KeyRef:     nullString(secret.APIKey),
		OrgRef:     nullString(secret.OrgRef),
		CreatedAt:  createdAt,
	}
	if row.ID == "" {
		row.ID = providerID
	}

	tx, err := repo.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err = tx.NewDelete().Model((*providerSecretRow)(nil)).Where("provider_id = ?", providerID).Exec(ctx); err != nil {
		return err
	}
	if _, err = tx.NewInsert().Model(&row).Exec(ctx); err != nil {
		return err
	}
	return tx.Commit()
}

func (repo *SQLiteProviderSecretRepository) DeleteByProviderID(ctx context.Context, providerID string) error {
	_, err := repo.db.NewDelete().Model((*providerSecretRow)(nil)).Where("provider_id = ?", providerID).Exec(ctx)
	return err
}
