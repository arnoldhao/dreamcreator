package assistantrepo

import (
	"context"
	"database/sql"
	"dreamcreator/internal/infrastructure/persistence/sqlitedto"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/uptrace/bun"

	"dreamcreator/internal/domain/assistant"
)

type SQLiteAssistantRepository struct {
	db *bun.DB
}

type assistantRow = sqlitedto.AssistantRow

func NewSQLiteAssistantRepository(db *bun.DB) *SQLiteAssistantRepository {
	return &SQLiteAssistantRepository{db: db}
}

func (repo *SQLiteAssistantRepository) List(ctx context.Context, includeDisabled bool) ([]assistant.Assistant, error) {
	rows := make([]assistantRow, 0)
	query := repo.db.NewSelect().Model(&rows).Order("is_default DESC", "updated_at DESC")
	if !includeDisabled {
		query = query.Where("enabled = 1")
	}
	if err := query.Scan(ctx); err != nil {
		return nil, err
	}
	result := make([]assistant.Assistant, 0, len(rows))
	for _, row := range rows {
		item, err := rowToAssistant(row)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, nil
}

func (repo *SQLiteAssistantRepository) Get(ctx context.Context, id string) (assistant.Assistant, error) {
	row := new(assistantRow)
	if err := repo.db.NewSelect().Model(row).Where("id = ?", id).Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return assistant.Assistant{}, assistant.ErrAssistantNotFound
		}
		return assistant.Assistant{}, err
	}
	return rowToAssistant(*row)
}

func (repo *SQLiteAssistantRepository) Save(ctx context.Context, item assistant.Assistant) error {
	createdAt, updatedAt := normalizeTimes(item.CreatedAt, item.UpdatedAt)
	identityJSON, err := marshalJSON(item.Identity)
	if err != nil {
		return err
	}
	avatarJSON, err := marshalJSON(item.Avatar)
	if err != nil {
		return err
	}
	userJSON, err := marshalJSON(item.User)
	if err != nil {
		return err
	}
	modelJSON, err := marshalJSON(item.Model)
	if err != nil {
		return err
	}
	toolsJSON, err := marshalJSON(item.Tools)
	if err != nil {
		return err
	}
	skillsJSON, err := marshalJSON(item.Skills)
	if err != nil {
		return err
	}
	callJSON, err := marshalJSON(item.Call)
	if err != nil {
		return err
	}
	memoryJSON, err := marshalJSON(item.Memory)
	if err != nil {
		return err
	}
	row := assistantRow{
		ID:           item.ID,
		IdentityJSON: nullString(identityJSON),
		AvatarJSON:   nullString(avatarJSON),
		UserJSON:     nullString(userJSON),
		ModelJSON:    nullString(modelJSON),
		ToolsJSON:    nullString(toolsJSON),
		SkillsJSON:   nullString(skillsJSON),
		CallJSON:     nullString(callJSON),
		MemoryJSON:   nullString(memoryJSON),
		Builtin:      item.Builtin,
		Deletable:    item.Deletable,
		Enabled:      item.Enabled,
		IsDefault:    item.IsDefault,
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
	}
	_, err = repo.db.NewInsert().Model(&row).
		On("CONFLICT(id) DO UPDATE").
		Set("identity_json = EXCLUDED.identity_json").
		Set("avatar_json = EXCLUDED.avatar_json").
		Set("user_json = EXCLUDED.user_json").
		Set("model_json = EXCLUDED.model_json").
		Set("tools_json = EXCLUDED.tools_json").
		Set("skills_json = EXCLUDED.skills_json").
		Set("call_json = EXCLUDED.call_json").
		Set("memory_json = EXCLUDED.memory_json").
		Set("builtin = EXCLUDED.builtin").
		Set("deletable = EXCLUDED.deletable").
		Set("enabled = EXCLUDED.enabled").
		Set("is_default = EXCLUDED.is_default").
		Set("updated_at = EXCLUDED.updated_at").
		Exec(ctx)
	return err
}

func (repo *SQLiteAssistantRepository) Delete(ctx context.Context, id string) error {
	_, err := repo.db.NewDelete().Model((*assistantRow)(nil)).Where("id = ?", id).Exec(ctx)
	return err
}

func (repo *SQLiteAssistantRepository) SetDefault(ctx context.Context, id string) error {
	trimmed := strings.TrimSpace(id)
	if trimmed == "" {
		return assistant.ErrInvalidAssistantID
	}
	tx, err := repo.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if _, err := tx.NewUpdate().Model((*assistantRow)(nil)).
		Set("is_default = 0").
		Where("is_default = 1").
		Exec(ctx); err != nil {
		_ = tx.Rollback()
		return err
	}
	now := time.Now()
	res, err := tx.NewUpdate().Model((*assistantRow)(nil)).
		Set("is_default = 1").
		Set("updated_at = ?", now).
		Where("id = ?", trimmed).
		Exec(ctx)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		_ = tx.Rollback()
		return assistant.ErrAssistantNotFound
	}
	return tx.Commit()
}

func rowToAssistant(row assistantRow) (assistant.Assistant, error) {
	identity, err := parseAssistantIdentity(row.IdentityJSON)
	if err != nil {
		return assistant.Assistant{}, err
	}
	avatar, err := parseAssistantAvatar(row.AvatarJSON)
	if err != nil {
		return assistant.Assistant{}, err
	}
	user, err := parseAssistantUser(row.UserJSON)
	if err != nil {
		return assistant.Assistant{}, err
	}
	model, err := parseAssistantModel(row.ModelJSON)
	if err != nil {
		return assistant.Assistant{}, err
	}
	tools, err := parseAssistantTools(row.ToolsJSON)
	if err != nil {
		return assistant.Assistant{}, err
	}
	skills, err := parseAssistantSkills(row.SkillsJSON)
	if err != nil {
		return assistant.Assistant{}, err
	}
	call, err := parseAssistantCall(row.CallJSON)
	if err != nil {
		return assistant.Assistant{}, err
	}
	memory, err := parseAssistantMemory(row.MemoryJSON)
	if err != nil {
		return assistant.Assistant{}, err
	}
	return assistant.NewAssistant(assistant.AssistantParams{
		ID:        row.ID,
		Builtin:   &row.Builtin,
		Deletable: &row.Deletable,
		Identity:  identity,
		Avatar:    avatar,
		User:      user,
		Model:     model,
		Tools:     tools,
		Skills:    skills,
		Call:      call,
		Memory:    memory,
		Enabled:   &row.Enabled,
		IsDefault: &row.IsDefault,
		CreatedAt: &row.CreatedAt,
		UpdatedAt: &row.UpdatedAt,
	})
}

func parseAssistantIdentity(raw sql.NullString) (assistant.AssistantIdentity, error) {
	if !raw.Valid || strings.TrimSpace(raw.String) == "" {
		return assistant.AssistantIdentity{}, nil
	}
	var identity assistant.AssistantIdentity
	if err := json.Unmarshal([]byte(raw.String), &identity); err != nil {
		return assistant.AssistantIdentity{}, err
	}
	return identity, nil
}

func parseAssistantAvatar(raw sql.NullString) (assistant.AssistantAvatar, error) {
	if !raw.Valid || strings.TrimSpace(raw.String) == "" {
		return assistant.AssistantAvatar{}, nil
	}
	var avatar assistant.AssistantAvatar
	if err := json.Unmarshal([]byte(raw.String), &avatar); err != nil {
		return assistant.AssistantAvatar{}, err
	}
	return avatar, nil
}

func parseAssistantUser(raw sql.NullString) (assistant.AssistantUser, error) {
	if !raw.Valid || strings.TrimSpace(raw.String) == "" {
		return assistant.AssistantUser{}, nil
	}
	var user assistant.AssistantUser
	if err := json.Unmarshal([]byte(raw.String), &user); err != nil {
		return assistant.AssistantUser{}, err
	}
	return user, nil
}

func parseAssistantModel(raw sql.NullString) (assistant.AssistantModel, error) {
	if !raw.Valid || strings.TrimSpace(raw.String) == "" {
		return assistant.AssistantModel{}, nil
	}
	var model assistant.AssistantModel
	if err := json.Unmarshal([]byte(raw.String), &model); err != nil {
		return assistant.AssistantModel{}, err
	}
	return model, nil
}

func parseAssistantTools(raw sql.NullString) (assistant.AssistantTools, error) {
	if !raw.Valid || strings.TrimSpace(raw.String) == "" {
		return assistant.AssistantTools{}, nil
	}
	var tools assistant.AssistantTools
	if err := json.Unmarshal([]byte(raw.String), &tools); err != nil {
		return assistant.AssistantTools{}, err
	}
	return tools, nil
}

func parseAssistantCall(raw sql.NullString) (assistant.AssistantCall, error) {
	if !raw.Valid || strings.TrimSpace(raw.String) == "" {
		return assistant.AssistantCall{}, nil
	}
	var call assistant.AssistantCall
	if err := json.Unmarshal([]byte(raw.String), &call); err != nil {
		return assistant.AssistantCall{}, err
	}
	return call, nil
}

func parseAssistantSkills(raw sql.NullString) (assistant.AssistantSkills, error) {
	if !raw.Valid || strings.TrimSpace(raw.String) == "" {
		return assistant.AssistantSkills{}, nil
	}
	var skills assistant.AssistantSkills
	if err := json.Unmarshal([]byte(raw.String), &skills); err != nil {
		return assistant.AssistantSkills{}, err
	}
	return skills, nil
}

func parseAssistantMemory(raw sql.NullString) (assistant.AssistantMemory, error) {
	if !raw.Valid || strings.TrimSpace(raw.String) == "" {
		return assistant.AssistantMemory{}, nil
	}
	var memory assistant.AssistantMemory
	if err := json.Unmarshal([]byte(raw.String), &memory); err != nil {
		return assistant.AssistantMemory{}, err
	}
	return memory, nil
}

func marshalJSON(value any) (string, error) {
	bytes, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
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
	if !value.Valid {
		return ""
	}
	return value.String
}
