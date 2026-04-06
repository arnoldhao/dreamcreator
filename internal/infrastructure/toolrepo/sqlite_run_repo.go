package toolrepo

import (
	"context"
	"database/sql"
	"dreamcreator/internal/infrastructure/persistence/sqlitedto"
	"errors"

	"github.com/uptrace/bun"

	"dreamcreator/internal/domain/tools"
)

type SQLiteToolRunRepository struct {
	db *bun.DB
}

type toolRunRow = sqlitedto.ToolRunRow

func NewSQLiteToolRunRepository(db *bun.DB) *SQLiteToolRunRepository {
	return &SQLiteToolRunRepository{db: db}
}

func (repo *SQLiteToolRunRepository) FindByKey(ctx context.Context, runID string, toolName string, inputHash string) (tools.ToolRun, error) {
	if repo == nil || repo.db == nil {
		return tools.ToolRun{}, tools.ErrToolRunNotFound
	}
	row := new(toolRunRow)
	err := repo.db.NewSelect().
		Model(row).
		Where("run_id = ?", runID).
		Where("tool_name = ?", toolName).
		Where("input_hash = ?", inputHash).
		Order("created_at DESC").
		Limit(1).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return tools.ToolRun{}, tools.ErrToolRunNotFound
		}
		return tools.ToolRun{}, err
	}
	return toToolRun(row)
}

func (repo *SQLiteToolRunRepository) Create(ctx context.Context, run tools.ToolRun) error {
	if repo == nil || repo.db == nil {
		return tools.ErrNotImplemented
	}
	row := fromToolRun(run)
	_, err := repo.db.NewInsert().Model(&row).Exec(ctx)
	return err
}

func (repo *SQLiteToolRunRepository) Update(ctx context.Context, run tools.ToolRun) error {
	if repo == nil || repo.db == nil {
		return tools.ErrNotImplemented
	}
	row := fromToolRun(run)
	_, err := repo.db.NewUpdate().Model(&row).Where("id = ?", run.ID).Exec(ctx)
	return err
}

func toToolRun(row *toolRunRow) (tools.ToolRun, error) {
	if row == nil {
		return tools.ToolRun{}, tools.ErrToolRunNotFound
	}
	return tools.NewToolRun(toToolRunParams(row))
}

func toToolRunParams(row *toolRunRow) tools.ToolRunParams {
	if row == nil {
		return tools.ToolRunParams{}
	}
	return tools.ToolRunParams{
		ID:         row.ID,
		RunID:      row.RunID,
		ToolCallID: row.ToolCallID,
		ToolName:   row.ToolName,
		InputHash:  row.InputHash,
		InputJSON:  row.InputJSON,
		OutputJSON: row.OutputJSON,
		ErrorText:  row.ErrorText,
		JobID:      row.JobID,
		Status:     row.Status,
		CreatedAt:  &row.CreatedAt,
		StartedAt:  row.StartedAt,
		FinishedAt: row.FinishedAt,
	}
}

func fromToolRun(run tools.ToolRun) toolRunRow {
	return toolRunRow{
		ID:         run.ID,
		RunID:      run.RunID,
		ToolCallID: run.ToolCallID,
		ToolName:   run.ToolName,
		InputHash:  run.InputHash,
		InputJSON:  run.InputJSON,
		OutputJSON: run.OutputJSON,
		ErrorText:  run.ErrorText,
		JobID:      run.JobID,
		Status:     string(run.Status),
		CreatedAt:  run.CreatedAt,
		StartedAt:  run.StartedAt,
		FinishedAt: run.FinishedAt,
	}
}
