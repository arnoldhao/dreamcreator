package voicerepo

import (
	"context"
	"database/sql"
	"dreamcreator/internal/infrastructure/persistence/sqlitedto"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/uptrace/bun"

	gatewayvoice "dreamcreator/internal/application/gateway/voice"
)

type SQLiteVoiceConfigRepository struct {
	db *bun.DB
}

type SQLiteTTSJobRepository struct {
	db *bun.DB
}

type voiceConfigRow = sqlitedto.VoiceConfigRow

type ttsJobRow = sqlitedto.TtsJobRow

func NewSQLiteVoiceConfigRepository(db *bun.DB) *SQLiteVoiceConfigRepository {
	return &SQLiteVoiceConfigRepository{db: db}
}

func NewSQLiteTTSJobRepository(db *bun.DB) *SQLiteTTSJobRepository {
	return &SQLiteTTSJobRepository{db: db}
}

func (repo *SQLiteVoiceConfigRepository) Get(ctx context.Context) (gatewayvoice.VoiceConfig, error) {
	row := new(voiceConfigRow)
	err := repo.db.NewSelect().Model(row).Where("id = ?", "default").Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return gatewayvoice.DefaultConfig(), nil
		}
		return gatewayvoice.VoiceConfig{}, err
	}
	config := gatewayvoice.DefaultConfig()
	if row.Version > 0 {
		config.Version = row.Version
	}
	if row.UpdatedAt.IsZero() {
		config.UpdatedAt = time.Now()
	} else {
		config.UpdatedAt = row.UpdatedAt
	}
	if row.TriggersJSON.Valid {
		_ = json.Unmarshal([]byte(row.TriggersJSON.String), &config.Triggers)
	}
	if row.TTSConfigJSON.Valid {
		_ = json.Unmarshal([]byte(row.TTSConfigJSON.String), &config.TTS)
	}
	if row.TalkConfigJSON.Valid {
		_ = json.Unmarshal([]byte(row.TalkConfigJSON.String), &config.Talk)
	}
	return config, nil
}

func (repo *SQLiteVoiceConfigRepository) Save(ctx context.Context, config gatewayvoice.VoiceConfig) error {
	triggersJSON, _ := json.Marshal(config.Triggers)
	ttsJSON, _ := json.Marshal(config.TTS)
	talkJSON, _ := json.Marshal(config.Talk)
	updatedAt := config.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = time.Now()
	}
	row := voiceConfigRow{
		ID:             "default",
		Version:        config.Version,
		TriggersJSON:   nullString(string(triggersJSON)),
		TTSConfigJSON:  nullString(string(ttsJSON)),
		TalkConfigJSON: nullString(string(talkJSON)),
		UpdatedAt:      updatedAt,
	}
	_, err := repo.db.NewInsert().Model(&row).
		On("CONFLICT(id) DO UPDATE").
		Set("version = EXCLUDED.version").
		Set("triggers_json = EXCLUDED.triggers_json").
		Set("tts_config_json = EXCLUDED.tts_config_json").
		Set("talk_config_json = EXCLUDED.talk_config_json").
		Set("updated_at = EXCLUDED.updated_at").
		Exec(ctx)
	return err
}

func (repo *SQLiteTTSJobRepository) Save(ctx context.Context, job gatewayvoice.TTSJob) error {
	createdAt := job.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	row := ttsJobRow{
		ID:         job.ID,
		ProviderID: nullString(job.ProviderID),
		VoiceID:    nullString(job.VoiceID),
		ModelID:    nullString(job.ModelID),
		Format:     nullString(job.Format),
		Status:     nullString(job.Status),
		InputText:  nullString(job.InputText),
		OutputJSON: nullString(job.OutputJSON),
		CostMicros: nullInt64(job.CostMicros),
		CreatedAt:  createdAt,
	}
	_, err := repo.db.NewInsert().Model(&row).Exec(ctx)
	return err
}

func nullString(value string) sql.NullString {
	if strings.TrimSpace(value) == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: value, Valid: true}
}

func nullInt64(value int64) sql.NullInt64 {
	if value == 0 {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: value, Valid: true}
}
