package agentrepo

import (
	"context"
	"database/sql"
	"dreamcreator/internal/infrastructure/persistence/sqlitedto"
	"errors"
	"time"

	"github.com/uptrace/bun"

	"dreamcreator/internal/domain/agent"
)

type SQLiteAgentRepository struct {
	db *bun.DB
}

type agentRow = sqlitedto.AgentRow

func NewSQLiteAgentRepository(db *bun.DB) *SQLiteAgentRepository {
	return &SQLiteAgentRepository{db: db}
}

func (repo *SQLiteAgentRepository) List(ctx context.Context, includeDisabled bool) ([]agent.Agent, error) {
	rows := make([]agentRow, 0)
	query := repo.db.NewSelect().Model(&rows).Where("deleted_at IS NULL").Order("updated_at DESC")
	if !includeDisabled {
		query = query.Where("enabled = 1")
	}
	if err := query.Scan(ctx); err != nil {
		return nil, err
	}

	result := make([]agent.Agent, 0, len(rows))
	for _, row := range rows {
		item, err := agent.NewAgent(agent.AgentParams{
			ID:          row.ID,
			Name:        row.Name,
			Description: stringOrEmpty(row.Description),
			Enabled:     &row.Enabled,
			ThreadID:    row.ThreadID,
			CreatedAt:   &row.CreatedAt,
			UpdatedAt:   &row.UpdatedAt,
			DeletedAt:   timeOrNil(row.DeletedAt),
		})
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, nil
}

func (repo *SQLiteAgentRepository) Get(ctx context.Context, id string) (agent.Agent, error) {
	row := new(agentRow)
	if err := repo.db.NewSelect().Model(row).Where("id = ?", id).Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return agent.Agent{}, agent.ErrAgentNotFound
		}
		return agent.Agent{}, err
	}
	return agent.NewAgent(agent.AgentParams{
		ID:          row.ID,
		Name:        row.Name,
		Description: stringOrEmpty(row.Description),
		Enabled:     &row.Enabled,
		ThreadID:    row.ThreadID,
		CreatedAt:   &row.CreatedAt,
		UpdatedAt:   &row.UpdatedAt,
		DeletedAt:   timeOrNil(row.DeletedAt),
	})
}

func (repo *SQLiteAgentRepository) Save(ctx context.Context, item agent.Agent) error {
	createdAt, updatedAt := normalizeTimes(item.CreatedAt, item.UpdatedAt)
	row := agentRow{
		ID:          item.ID,
		Name:        item.Name,
		Description: nullString(item.Description),
		Enabled:     item.Enabled,
		ThreadID:    item.ThreadID,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
		DeletedAt:   nullTime(item.DeletedAt),
	}

	_, err := repo.db.NewInsert().Model(&row).
		On("CONFLICT(id) DO UPDATE").
		Set("name = EXCLUDED.name").
		Set("description = EXCLUDED.description").
		Set("enabled = EXCLUDED.enabled").
		Set("thread_id = EXCLUDED.thread_id").
		Set("updated_at = EXCLUDED.updated_at").
		Set("deleted_at = EXCLUDED.deleted_at").
		Exec(ctx)
	return err
}

func (repo *SQLiteAgentRepository) SoftDelete(ctx context.Context, id string, deletedAt *time.Time) error {
	_, err := repo.db.NewUpdate().Model((*agentRow)(nil)).
		Set("deleted_at = ?", deletedAt).
		Where("id = ?", id).
		Exec(ctx)
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
