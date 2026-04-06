package workspacerepo

import (
	"context"
	"database/sql"
	"dreamcreator/internal/infrastructure/persistence/sqlitedto"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/uptrace/bun"

	"dreamcreator/internal/domain/assistant"
	"dreamcreator/internal/domain/workspace"
)

type SQLiteWorkspaceRepository struct {
	db *bun.DB
}

type globalWorkspaceRow = sqlitedto.GlobalWorkspaceRow

type assistantWorkspaceRow = sqlitedto.WorkspaceRow

type assistantWorkspaceSnapshotRow = sqlitedto.WorkspaceSnapshotRow

func NewSQLiteWorkspaceRepository(db *bun.DB) *SQLiteWorkspaceRepository {
	return &SQLiteWorkspaceRepository{db: db}
}

func (repo *SQLiteWorkspaceRepository) GetGlobal(ctx context.Context) (workspace.GlobalWorkspace, error) {
	row := new(globalWorkspaceRow)
	if err := repo.db.NewSelect().Model(row).Where("id = 1").Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return workspace.GlobalWorkspace{}, workspace.ErrWorkspaceNotFound
		}
		return workspace.GlobalWorkspace{}, err
	}

	return workspace.NewGlobalWorkspace(workspace.GlobalWorkspaceParams{
		ID:                       row.ID,
		DefaultExecutorModelJSON: stringOrEmpty(row.DefaultExecutorModelJSON),
		DefaultMemoryJSON:        stringOrEmpty(row.DefaultMemoryJSON),
		DefaultPersona:           stringOrEmpty(row.DefaultPersona),
		CreatedAt:                &row.CreatedAt,
		UpdatedAt:                &row.UpdatedAt,
	})
}

func (repo *SQLiteWorkspaceRepository) SaveGlobal(ctx context.Context, workspace workspace.GlobalWorkspace) error {
	createdAt, updatedAt := normalizeTimes(workspace.CreatedAt, workspace.UpdatedAt)
	row := globalWorkspaceRow{
		ID:                       workspace.ID,
		DefaultExecutorModelJSON: nullString(workspace.DefaultExecutorModelJSON),
		DefaultMemoryJSON:        nullString(workspace.DefaultMemoryJSON),
		DefaultPersona:           nullString(workspace.DefaultPersona),
		CreatedAt:                createdAt,
		UpdatedAt:                updatedAt,
	}

	_, err := repo.db.NewInsert().Model(&row).
		On("CONFLICT(id) DO UPDATE").
		Set("default_executor_model_json = EXCLUDED.default_executor_model_json").
		Set("default_memory_json = EXCLUDED.default_memory_json").
		Set("default_persona = EXCLUDED.default_persona").
		Set("updated_at = EXCLUDED.updated_at").
		Exec(ctx)
	return err
}

func (repo *SQLiteWorkspaceRepository) GetAssistantWorkspace(ctx context.Context, assistantID string) (workspace.AssistantWorkspace, error) {
	row := new(assistantWorkspaceRow)
	if err := repo.db.NewSelect().Model(row).Where("assistant_id = ?", strings.TrimSpace(assistantID)).Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return workspace.AssistantWorkspace{}, workspace.ErrWorkspaceNotFound
		}
		return workspace.AssistantWorkspace{}, err
	}
	return rowToAssistantWorkspace(*row)
}

func (repo *SQLiteWorkspaceRepository) SaveAssistantWorkspace(ctx context.Context, workspaceItem workspace.AssistantWorkspace) error {
	row, err := toAssistantWorkspaceRow(workspaceItem)
	if err != nil {
		return err
	}
	_, err = repo.db.NewInsert().Model(&row).
		On("CONFLICT(assistant_id) DO UPDATE").
		Set("version = EXCLUDED.version").
		Set("identity_json = EXCLUDED.identity_json").
		Set("persona_text = EXCLUDED.persona_text").
		Set("user_profile_json = EXCLUDED.user_profile_json").
		Set("tooling_json = EXCLUDED.tooling_json").
		Set("memory_json = EXCLUDED.memory_json").
		Set("memory_config_json = EXCLUDED.memory_config_json").
		Set("extra_files_json = EXCLUDED.extra_files_json").
		Set("prompt_mode_default = EXCLUDED.prompt_mode_default").
		Set("updated_at = EXCLUDED.updated_at").
		Exec(ctx)
	return err
}

func (repo *SQLiteWorkspaceRepository) UpdateAssistantWorkspace(ctx context.Context, workspaceItem workspace.AssistantWorkspace, expectedVersion int64) error {
	if expectedVersion <= 0 {
		return workspace.ErrWorkspaceVersionConflict
	}
	row, err := toAssistantWorkspaceRow(workspaceItem)
	if err != nil {
		return err
	}
	res, err := repo.db.NewUpdate().Model((*assistantWorkspaceRow)(nil)).
		Set("version = ?", row.Version).
		Set("identity_json = ?", row.IdentityJSON).
		Set("persona_text = ?", row.PersonaText).
		Set("user_profile_json = ?", row.UserProfileJSON).
		Set("tooling_json = ?", row.ToolingJSON).
		Set("memory_json = ?", row.MemoryJSON).
		Set("memory_config_json = ?", row.MemoryConfigJSON).
		Set("extra_files_json = ?", row.ExtraFilesJSON).
		Set("prompt_mode_default = ?", row.PromptModeDefault).
		Set("updated_at = ?", row.UpdatedAt).
		Where("assistant_id = ?", row.AssistantID).
		Where("version = ?", expectedVersion).
		Exec(ctx)
	if err != nil {
		return err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return workspace.ErrWorkspaceVersionConflict
	}
	return nil
}

func (repo *SQLiteWorkspaceRepository) GetAssistantWorkspaceSnapshot(ctx context.Context, assistantID string, version int64) (workspace.AssistantWorkspaceSnapshot, error) {
	row := new(assistantWorkspaceSnapshotRow)
	versionKey := strconv.FormatInt(version, 10)
	if err := repo.db.NewSelect().
		Model(row).
		Where("assistant_id = ?", assistantID).
		Where("workspace_version = ?", versionKey).
		OrderExpr("created_at DESC").
		Limit(1).
		Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return workspace.AssistantWorkspaceSnapshot{}, workspace.ErrWorkspaceNotFound
		}
		return workspace.AssistantWorkspaceSnapshot{}, err
	}
	logicalFiles, err := decodeLogicalFiles(row.LogicalFilesJSON)
	if err != nil {
		return workspace.AssistantWorkspaceSnapshot{}, err
	}
	versionValue, _ := strconv.ParseInt(strings.TrimSpace(row.WorkspaceVersion), 10, 64)
	promptMode := strings.TrimSpace(row.PromptModeDefault.String)
	toolHints, err := decodeStringList(row.ToolHintsJSON)
	if err != nil {
		return workspace.AssistantWorkspaceSnapshot{}, err
	}
	skillHints, err := decodeStringList(row.SkillHintsJSON)
	if err != nil {
		return workspace.AssistantWorkspaceSnapshot{}, err
	}
	generatedAt := row.CreatedAt
	if row.GeneratedAt.Valid {
		generatedAt = row.GeneratedAt.Time
	}
	return workspace.AssistantWorkspaceSnapshot{
		ID:                row.ID,
		AssistantID:       row.AssistantID,
		WorkspaceVersion:  versionValue,
		LogicalFiles:      logicalFiles,
		PromptModeDefault: promptMode,
		ToolHints:         toolHints,
		SkillHints:        skillHints,
		GeneratedAt:       generatedAt,
		CreatedAt:         row.CreatedAt,
	}, nil
}

func (repo *SQLiteWorkspaceRepository) SaveAssistantWorkspaceSnapshot(ctx context.Context, snapshot workspace.AssistantWorkspaceSnapshot) error {
	logicalFilesJSON, err := encodeLogicalFiles(snapshot.LogicalFiles)
	if err != nil {
		return err
	}
	toolHintsJSON, err := encodeStringList(snapshot.ToolHints)
	if err != nil {
		return err
	}
	skillHintsJSON, err := encodeStringList(snapshot.SkillHints)
	if err != nil {
		return err
	}
	promptMode := nullString(snapshot.PromptModeDefault)
	row := assistantWorkspaceSnapshotRow{
		ID:                snapshot.ID,
		AssistantID:       snapshot.AssistantID,
		WorkspaceVersion:  strconv.FormatInt(snapshot.WorkspaceVersion, 10),
		LogicalFilesJSON:  logicalFilesJSON,
		PromptModeDefault: promptMode,
		ToolHintsJSON:     toolHintsJSON,
		SkillHintsJSON:    skillHintsJSON,
		GeneratedAt:       nullTime(&snapshot.GeneratedAt),
		CreatedAt:         snapshot.CreatedAt,
	}
	if row.CreatedAt.IsZero() {
		row.CreatedAt = time.Now()
	}
	_, err = repo.db.NewInsert().Model(&row).
		On("CONFLICT(id) DO UPDATE").
		Set("assistant_id = EXCLUDED.assistant_id").
		Set("workspace_version = EXCLUDED.workspace_version").
		Set("logical_files_json = EXCLUDED.logical_files_json").
		Set("prompt_mode_default = EXCLUDED.prompt_mode_default").
		Set("tool_hints_json = EXCLUDED.tool_hints_json").
		Set("skill_hints_json = EXCLUDED.skill_hints_json").
		Set("generated_at = EXCLUDED.generated_at").
		Set("created_at = EXCLUDED.created_at").
		Exec(ctx)
	return err
}

func rowToAssistantWorkspace(row assistantWorkspaceRow) (workspace.AssistantWorkspace, error) {
	identity := assistant.AssistantIdentity{}
	if err := decodeJSON(row.IdentityJSON, &identity); err != nil {
		return workspace.AssistantWorkspace{}, err
	}
	userProfile := assistant.AssistantUser{}
	if err := decodeJSON(row.UserProfileJSON, &userProfile); err != nil {
		return workspace.AssistantWorkspace{}, err
	}
	tooling := assistant.AssistantCall{}
	if err := decodeJSON(row.ToolingJSON, &tooling); err != nil {
		return workspace.AssistantWorkspace{}, err
	}
	memory := assistant.AssistantMemory{}
	if err := decodeJSON(row.MemoryJSON, &memory); err != nil {
		return workspace.AssistantWorkspace{}, err
	}
	extraFiles, err := decodeLogicalFiles(row.ExtraFilesJSON)
	if err != nil {
		return workspace.AssistantWorkspace{}, err
	}
	return workspace.NewAssistantWorkspace(workspace.AssistantWorkspaceParams{
		AssistantID:       row.AssistantID,
		Version:           row.Version,
		Identity:          identity,
		Persona:           stringOrEmpty(row.PersonaText),
		UserProfile:       userProfile,
		Tooling:           tooling,
		Memory:            memory,
		MemoryJSON:        stringOrEmpty(row.MemoryConfigJSON),
		ExtraFiles:        extraFiles,
		PromptModeDefault: stringOrEmpty(row.PromptModeDefault),
		CreatedAt:         &row.CreatedAt,
		UpdatedAt:         &row.UpdatedAt,
	})
}

func toAssistantWorkspaceRow(workspaceItem workspace.AssistantWorkspace) (assistantWorkspaceRow, error) {
	identityJSON, err := encodeJSON(workspaceItem.Identity)
	if err != nil {
		return assistantWorkspaceRow{}, err
	}
	userProfileJSON, err := encodeJSON(workspaceItem.UserProfile)
	if err != nil {
		return assistantWorkspaceRow{}, err
	}
	toolingJSON, err := encodeJSON(workspaceItem.Tooling)
	if err != nil {
		return assistantWorkspaceRow{}, err
	}
	memoryJSON, err := encodeJSON(workspaceItem.Memory)
	if err != nil {
		return assistantWorkspaceRow{}, err
	}
	extraFilesJSON, err := encodeLogicalFiles(workspaceItem.ExtraFiles)
	if err != nil {
		return assistantWorkspaceRow{}, err
	}
	createdAt := workspaceItem.CreatedAt
	updatedAt := workspaceItem.UpdatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	if updatedAt.IsZero() {
		updatedAt = createdAt
	}
	return assistantWorkspaceRow{
		AssistantID:       strings.TrimSpace(workspaceItem.AssistantID),
		Version:           workspaceItem.Version,
		IdentityJSON:      identityJSON,
		PersonaText:       nullString(workspaceItem.Persona),
		UserProfileJSON:   userProfileJSON,
		ToolingJSON:       toolingJSON,
		MemoryJSON:        memoryJSON,
		MemoryConfigJSON:  nullString(workspaceItem.MemoryJSON),
		ExtraFilesJSON:    extraFilesJSON,
		PromptModeDefault: nullString(workspaceItem.PromptModeDefault),
		CreatedAt:         createdAt,
		UpdatedAt:         updatedAt,
	}, nil
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

func encodeLogicalFiles(files []workspace.WorkspaceLogicalFile) (sql.NullString, error) {
	if len(files) == 0 {
		return sql.NullString{}, nil
	}
	encoded, err := json.Marshal(files)
	if err != nil {
		return sql.NullString{}, err
	}
	return nullString(string(encoded)), nil
}

func decodeLogicalFiles(value sql.NullString) ([]workspace.WorkspaceLogicalFile, error) {
	if !value.Valid || strings.TrimSpace(value.String) == "" {
		return nil, nil
	}
	var files []workspace.WorkspaceLogicalFile
	if err := json.Unmarshal([]byte(value.String), &files); err != nil {
		return nil, err
	}
	return files, nil
}

func encodeStringList(values []string) (sql.NullString, error) {
	clean := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		clean = append(clean, trimmed)
	}
	if len(clean) == 0 {
		return sql.NullString{}, nil
	}
	encoded, err := json.Marshal(clean)
	if err != nil {
		return sql.NullString{}, err
	}
	return nullString(string(encoded)), nil
}

func decodeStringList(value sql.NullString) ([]string, error) {
	if !value.Valid || strings.TrimSpace(value.String) == "" {
		return nil, nil
	}
	var entries []string
	if err := json.Unmarshal([]byte(value.String), &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

func encodeJSON(value any) (sql.NullString, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return sql.NullString{}, err
	}
	trimmed := strings.TrimSpace(string(encoded))
	if trimmed == "" || trimmed == "null" {
		return sql.NullString{}, nil
	}
	return sql.NullString{String: trimmed, Valid: true}, nil
}

func decodeJSON(value sql.NullString, target any) error {
	if !value.Valid || strings.TrimSpace(value.String) == "" {
		return nil
	}
	return json.Unmarshal([]byte(value.String), target)
}

func nullTime(value *time.Time) sql.NullTime {
	if value == nil || value.IsZero() {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *value, Valid: true}
}

func timeOrNil(value sql.NullTime) *time.Time {
	if value.Valid {
		return &value.Time
	}
	return nil
}
