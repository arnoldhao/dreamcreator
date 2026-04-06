package usagerepo

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	gatewayusage "dreamcreator/internal/application/gateway/usage"
	"dreamcreator/internal/infrastructure/persistence/sqlitedto"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type SQLiteUsageLedgerRepository struct {
	db *bun.DB
}

type usageEventRow = sqlitedto.UsageEventRow
type usageLedgerEntryRow = sqlitedto.UsageLedgerEntryRow
type usagePricingVersionRow = sqlitedto.UsagePricingVersionRow

func NewSQLiteUsageLedgerRepository(db *bun.DB) *SQLiteUsageLedgerRepository {
	return &SQLiteUsageLedgerRepository{db: db}
}

func (repo *SQLiteUsageLedgerRepository) UpsertEvent(ctx context.Context, event gatewayusage.UsageEvent) (gatewayusage.UsageEvent, error) {
	now := time.Now().UTC()
	row := usageEventRow{
		ID:                strings.TrimSpace(event.ID),
		RequestID:         strings.TrimSpace(event.RequestID),
		StepID:            strings.TrimSpace(event.StepID),
		ProviderID:        strings.TrimSpace(event.ProviderID),
		ModelName:         strings.TrimSpace(event.ModelName),
		Category:          nullString(event.Category),
		Channel:           nullString(event.Channel),
		RequestSource:     strings.TrimSpace(event.RequestSource),
		UsageStatus:       nullString(event.UsageStatus),
		InputTokens:       nullInt64(int64(event.InputTokens)),
		OutputTokens:      nullInt64(int64(event.OutputTokens)),
		TotalTokens:       nullInt64(int64(event.TotalTokens)),
		CachedInputTokens: nullInt64(int64(event.CachedInputTokens)),
		ReasoningTokens:   nullInt64(int64(event.ReasoningTokens)),
		AudioInputTokens:  nullInt64(int64(event.AudioInputTokens)),
		AudioOutputTokens: nullInt64(int64(event.AudioOutputTokens)),
		RawUsageJSON:      nullString(event.RawUsageJSON),
		OccurredAt:        event.OccurredAt.UTC(),
		CreatedAt:         event.CreatedAt.UTC(),
		UpdatedAt:         event.UpdatedAt.UTC(),
	}
	if row.ID == "" {
		row.ID = uuid.NewString()
	}
	if row.RequestID == "" {
		row.RequestID = row.ID
	}
	if row.StepID == "" {
		row.StepID = "run"
	}
	if row.ProviderID == "" {
		row.ProviderID = "unknown"
	}
	if row.ModelName == "" {
		row.ModelName = "unknown"
	}
	if row.RequestSource == "" {
		row.RequestSource = gatewayusage.RequestSourceUnknown
	}
	if row.OccurredAt.IsZero() {
		row.OccurredAt = now
	}
	if row.CreatedAt.IsZero() {
		row.CreatedAt = now
	}
	if row.UpdatedAt.IsZero() {
		row.UpdatedAt = row.CreatedAt
	}
	_, err := repo.db.NewInsert().Model(&row).
		On("CONFLICT (request_id, step_id, provider_id, model_name) DO UPDATE").
		Set("category = EXCLUDED.category").
		Set("channel = EXCLUDED.channel").
		Set("request_source = EXCLUDED.request_source").
		Set("usage_status = EXCLUDED.usage_status").
		Set("input_tokens = EXCLUDED.input_tokens").
		Set("output_tokens = EXCLUDED.output_tokens").
		Set("total_tokens = EXCLUDED.total_tokens").
		Set("cached_input_tokens = EXCLUDED.cached_input_tokens").
		Set("reasoning_tokens = EXCLUDED.reasoning_tokens").
		Set("audio_input_tokens = EXCLUDED.audio_input_tokens").
		Set("audio_output_tokens = EXCLUDED.audio_output_tokens").
		Set("raw_usage_json = EXCLUDED.raw_usage_json").
		Set("occurred_at = EXCLUDED.occurred_at").
		Set("updated_at = EXCLUDED.updated_at").
		Exec(ctx)
	if err != nil {
		return gatewayusage.UsageEvent{}, err
	}
	stored := usageEventRow{}
	if err := repo.db.NewSelect().Model(&stored).
		Where("request_id = ?", row.RequestID).
		Where("step_id = ?", row.StepID).
		Where("provider_id = ?", row.ProviderID).
		Where("model_name = ?", row.ModelName).
		Limit(1).
		Scan(ctx); err != nil {
		return gatewayusage.UsageEvent{}, err
	}
	return mapUsageEvent(stored), nil
}

func (repo *SQLiteUsageLedgerRepository) UpsertLedger(ctx context.Context, entry gatewayusage.LedgerEntry) error {
	now := time.Now().UTC()
	row := usageLedgerEntryRow{
		ID:                    strings.TrimSpace(entry.ID),
		EventID:               strings.TrimSpace(entry.EventID),
		RequestID:             strings.TrimSpace(entry.RequestID),
		Category:              strings.TrimSpace(entry.Category),
		ProviderID:            strings.TrimSpace(entry.ProviderID),
		ModelName:             strings.TrimSpace(entry.ModelName),
		Channel:               nullString(entry.Channel),
		RequestSource:         strings.TrimSpace(entry.RequestSource),
		CostBasis:             strings.TrimSpace(entry.CostBasis),
		PricingVersionID:      strings.TrimSpace(entry.PricingVersionID),
		Units:                 nullInt64(int64(entry.Units)),
		InputTokens:           nullInt64(int64(entry.InputTokens)),
		OutputTokens:          nullInt64(int64(entry.OutputTokens)),
		CachedInputTokens:     nullInt64(int64(entry.CachedInputTokens)),
		ReasoningTokens:       nullInt64(int64(entry.ReasoningTokens)),
		InputCostMicros:       nullInt64(entry.InputCostMicros),
		OutputCostMicros:      nullInt64(entry.OutputCostMicros),
		CachedInputCostMicros: nullInt64(entry.CachedInputCostMicros),
		ReasoningCostMicros:   nullInt64(entry.ReasoningCostMicros),
		RequestCostMicros:     nullInt64(entry.RequestCostMicros),
		TotalCostMicros:       nullInt64(entry.CostMicros),
		CreatedAt:             entry.CreatedAt.UTC(),
	}
	if row.ID == "" {
		row.ID = uuid.NewString()
	}
	if row.EventID == "" {
		return errors.New("usage ledger event_id is required")
	}
	if row.RequestID == "" {
		row.RequestID = row.ID
	}
	if row.ProviderID == "" {
		row.ProviderID = "unknown"
	}
	if row.ModelName == "" {
		row.ModelName = "unknown"
	}
	if row.Category == "" {
		row.Category = gatewayusage.CategoryTokens
	}
	if row.RequestSource == "" {
		row.RequestSource = gatewayusage.RequestSourceUnknown
	}
	if row.CostBasis == "" {
		row.CostBasis = gatewayusage.CostBasisEstimated
	}
	if row.CreatedAt.IsZero() {
		row.CreatedAt = now
	}
	_, err := repo.db.NewInsert().Model(&row).
		On("CONFLICT (event_id, pricing_version_id, cost_basis, category) DO UPDATE").
		Set("request_id = EXCLUDED.request_id").
		Set("provider_id = EXCLUDED.provider_id").
		Set("model_name = EXCLUDED.model_name").
		Set("channel = EXCLUDED.channel").
		Set("request_source = EXCLUDED.request_source").
		Set("units = EXCLUDED.units").
		Set("input_tokens = EXCLUDED.input_tokens").
		Set("output_tokens = EXCLUDED.output_tokens").
		Set("cached_input_tokens = EXCLUDED.cached_input_tokens").
		Set("reasoning_tokens = EXCLUDED.reasoning_tokens").
		Set("input_cost_micros = EXCLUDED.input_cost_micros").
		Set("output_cost_micros = EXCLUDED.output_cost_micros").
		Set("cached_input_cost_micros = EXCLUDED.cached_input_cost_micros").
		Set("reasoning_cost_micros = EXCLUDED.reasoning_cost_micros").
		Set("request_cost_micros = EXCLUDED.request_cost_micros").
		Set("total_cost_micros = EXCLUDED.total_cost_micros").
		Set("created_at = EXCLUDED.created_at").
		Exec(ctx)
	return err
}

func (repo *SQLiteUsageLedgerRepository) ListLedger(ctx context.Context, filter gatewayusage.QueryFilter) ([]gatewayusage.LedgerEntry, error) {
	rows := make([]usageLedgerEntryRow, 0)
	query := repo.db.NewSelect().Model(&rows)
	if !filter.StartAt.IsZero() {
		query = query.Where("created_at >= ?", filter.StartAt.UTC())
	}
	if !filter.EndAt.IsZero() {
		query = query.Where("created_at <= ?", filter.EndAt.UTC())
	}
	if trimmed := strings.TrimSpace(filter.ProviderID); trimmed != "" {
		query = query.Where("provider_id = ?", trimmed)
	}
	if trimmed := strings.TrimSpace(filter.ModelName); trimmed != "" {
		query = query.Where("model_name = ?", trimmed)
	}
	if trimmed := strings.TrimSpace(filter.Channel); trimmed != "" {
		query = query.Where("channel = ?", trimmed)
	}
	if trimmed := strings.TrimSpace(filter.Category); trimmed != "" {
		query = query.Where("category = ?", trimmed)
	}
	if trimmed := strings.TrimSpace(filter.RequestSource); trimmed != "" {
		query = query.Where("request_source = ?", trimmed)
	}
	if trimmed := strings.TrimSpace(filter.CostBasis); trimmed != "" {
		query = query.Where("cost_basis = ?", trimmed)
	}
	if err := query.Order("created_at DESC").Scan(ctx); err != nil {
		return nil, err
	}
	result := make([]gatewayusage.LedgerEntry, 0, len(rows))
	for _, row := range rows {
		result = append(result, gatewayusage.LedgerEntry{
			ID:                    row.ID,
			EventID:               row.EventID,
			RequestID:             row.RequestID,
			Category:              row.Category,
			ProviderID:            row.ProviderID,
			ModelName:             row.ModelName,
			Channel:               stringOrEmpty(row.Channel),
			RequestSource:         row.RequestSource,
			CostBasis:             row.CostBasis,
			PricingVersionID:      row.PricingVersionID,
			Units:                 intOrZero(row.Units),
			InputTokens:           intOrZero(row.InputTokens),
			OutputTokens:          intOrZero(row.OutputTokens),
			CachedInputTokens:     intOrZero(row.CachedInputTokens),
			ReasoningTokens:       intOrZero(row.ReasoningTokens),
			InputCostMicros:       int64OrZero(row.InputCostMicros),
			OutputCostMicros:      int64OrZero(row.OutputCostMicros),
			CachedInputCostMicros: int64OrZero(row.CachedInputCostMicros),
			ReasoningCostMicros:   int64OrZero(row.ReasoningCostMicros),
			RequestCostMicros:     int64OrZero(row.RequestCostMicros),
			CostMicros:            int64OrZero(row.TotalCostMicros),
			CreatedAt:             row.CreatedAt,
		})
	}
	return result, nil
}

func (repo *SQLiteUsageLedgerRepository) ResolvePricingVersion(ctx context.Context, providerID string, modelName string, at time.Time) (gatewayusage.PricingVersion, bool, error) {
	providerID = strings.TrimSpace(providerID)
	modelName = strings.TrimSpace(modelName)
	if providerID == "" || modelName == "" {
		return gatewayusage.PricingVersion{}, false, nil
	}
	if at.IsZero() {
		at = time.Now()
	}
	row := usagePricingVersionRow{}
	err := repo.db.NewSelect().
		Model(&row).
		Where("provider_id = ?", providerID).
		Where("model_name = ?", modelName).
		Where("is_active = 1").
		Where("effective_from <= ?", at.UTC()).
		Where("(effective_to IS NULL OR effective_to > ?)", at.UTC()).
		Order("effective_from DESC").
		Limit(1).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return gatewayusage.PricingVersion{}, false, nil
		}
		return gatewayusage.PricingVersion{}, false, err
	}
	return mapPricingVersion(row), true, nil
}

func (repo *SQLiteUsageLedgerRepository) ListPricingVersions(ctx context.Context, filter gatewayusage.PricingVersionFilter) ([]gatewayusage.PricingVersion, error) {
	rows := make([]usagePricingVersionRow, 0)
	query := repo.db.NewSelect().Model(&rows)
	if trimmed := strings.TrimSpace(filter.ProviderID); trimmed != "" {
		query = query.Where("provider_id = ?", trimmed)
	}
	if trimmed := strings.TrimSpace(filter.ModelName); trimmed != "" {
		query = query.Where("model_name = ?", trimmed)
	}
	if trimmed := strings.TrimSpace(filter.Source); trimmed != "" {
		query = query.Where("source = ?", trimmed)
	}
	if filter.ActiveOnly {
		query = query.Where("is_active = 1")
	}
	if err := query.Order("effective_from DESC", "updated_at DESC").Scan(ctx); err != nil {
		return nil, err
	}
	result := make([]gatewayusage.PricingVersion, 0, len(rows))
	for _, row := range rows {
		result = append(result, mapPricingVersion(row))
	}
	return result, nil
}

func (repo *SQLiteUsageLedgerRepository) UpsertPricingVersion(ctx context.Context, version gatewayusage.PricingVersion) (gatewayusage.PricingVersion, error) {
	now := time.Now().UTC()
	row := usagePricingVersionRow{
		ID:                    strings.TrimSpace(version.ID),
		ProviderID:            strings.TrimSpace(version.ProviderID),
		ModelName:             strings.TrimSpace(version.ModelName),
		Currency:              strings.TrimSpace(version.Currency),
		InputPerMillion:       version.InputPerMillion,
		OutputPerMillion:      version.OutputPerMillion,
		CachedInputPerMillion: version.CachedInputPerMillion,
		ReasoningPerMillion:   version.ReasoningPerMillion,
		AudioInputPerMillion:  version.AudioInputPerMillion,
		AudioOutputPerMillion: version.AudioOutputPerMillion,
		PerRequest:            version.PerRequest,
		Source:                strings.TrimSpace(version.Source),
		EffectiveFrom:         version.EffectiveFrom.UTC(),
		EffectiveTo:           nullTime(version.EffectiveTo),
		IsActive:              version.IsActive,
		UpdatedBy:             nullString(version.UpdatedBy),
		CreatedAt:             version.CreatedAt.UTC(),
		UpdatedAt:             version.UpdatedAt.UTC(),
	}
	if row.ID == "" {
		row.ID = uuid.NewString()
	}
	if row.ProviderID == "" {
		return gatewayusage.PricingVersion{}, errors.New("provider_id is required")
	}
	if row.ModelName == "" {
		return gatewayusage.PricingVersion{}, errors.New("model_name is required")
	}
	if row.Currency == "" {
		row.Currency = "USD"
	}
	if row.Source == "" {
		row.Source = "manual"
	}
	if row.EffectiveFrom.IsZero() {
		row.EffectiveFrom = now
	}
	if row.CreatedAt.IsZero() {
		row.CreatedAt = now
	}
	if row.UpdatedAt.IsZero() {
		row.UpdatedAt = now
	}
	_, err := repo.db.NewInsert().Model(&row).
		On("CONFLICT (id) DO UPDATE").
		Set("provider_id = EXCLUDED.provider_id").
		Set("model_name = EXCLUDED.model_name").
		Set("currency = EXCLUDED.currency").
		Set("input_per_million = EXCLUDED.input_per_million").
		Set("output_per_million = EXCLUDED.output_per_million").
		Set("cached_input_per_million = EXCLUDED.cached_input_per_million").
		Set("reasoning_per_million = EXCLUDED.reasoning_per_million").
		Set("audio_input_per_million = EXCLUDED.audio_input_per_million").
		Set("audio_output_per_million = EXCLUDED.audio_output_per_million").
		Set("per_request = EXCLUDED.per_request").
		Set("source = EXCLUDED.source").
		Set("effective_from = EXCLUDED.effective_from").
		Set("effective_to = EXCLUDED.effective_to").
		Set("is_active = EXCLUDED.is_active").
		Set("updated_by = EXCLUDED.updated_by").
		Set("updated_at = EXCLUDED.updated_at").
		Exec(ctx)
	if err != nil {
		return gatewayusage.PricingVersion{}, err
	}
	if row.IsActive {
		if _, err := repo.db.NewUpdate().
			Model((*usagePricingVersionRow)(nil)).
			Set("is_active = 0").
			Where("provider_id = ?", row.ProviderID).
			Where("model_name = ?", row.ModelName).
			Where("id <> ?", row.ID).
			Exec(ctx); err != nil {
			return gatewayusage.PricingVersion{}, err
		}
	}
	updated := usagePricingVersionRow{}
	if err := repo.db.NewSelect().Model(&updated).Where("id = ?", row.ID).Limit(1).Scan(ctx); err != nil {
		return gatewayusage.PricingVersion{}, err
	}
	return mapPricingVersion(updated), nil
}

func (repo *SQLiteUsageLedgerRepository) DeletePricingVersion(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return errors.New("id is required")
	}
	count, err := repo.db.NewSelect().
		Model((*usageLedgerEntryRow)(nil)).
		Where("pricing_version_id = ?", id).
		Count(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return errors.New("pricing version is referenced by usage ledger")
	}
	_, err = repo.db.NewDelete().Model((*usagePricingVersionRow)(nil)).Where("id = ?", id).Exec(ctx)
	return err
}

func (repo *SQLiteUsageLedgerRepository) ActivatePricingVersion(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return errors.New("id is required")
	}
	target := usagePricingVersionRow{}
	if err := repo.db.NewSelect().Model(&target).Where("id = ?", id).Limit(1).Scan(ctx); err != nil {
		return err
	}
	tx, err := repo.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if _, err := tx.NewUpdate().Model((*usagePricingVersionRow)(nil)).
		Set("is_active = 0").
		Set("updated_at = ?", time.Now().UTC()).
		Where("provider_id = ?", target.ProviderID).
		Where("model_name = ?", target.ModelName).
		Exec(ctx); err != nil {
		_ = tx.Rollback()
		return err
	}
	if _, err := tx.NewUpdate().Model((*usagePricingVersionRow)(nil)).
		Set("is_active = 1").
		Set("updated_at = ?", time.Now().UTC()).
		Where("id = ?", id).
		Exec(ctx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func mapUsageEvent(row usageEventRow) gatewayusage.UsageEvent {
	return gatewayusage.UsageEvent{
		ID:                row.ID,
		RequestID:         row.RequestID,
		StepID:            row.StepID,
		ProviderID:        row.ProviderID,
		ModelName:         row.ModelName,
		Category:          stringOrEmpty(row.Category),
		Channel:           stringOrEmpty(row.Channel),
		RequestSource:     row.RequestSource,
		UsageStatus:       stringOrEmpty(row.UsageStatus),
		InputTokens:       intOrZero(row.InputTokens),
		OutputTokens:      intOrZero(row.OutputTokens),
		TotalTokens:       intOrZero(row.TotalTokens),
		CachedInputTokens: intOrZero(row.CachedInputTokens),
		ReasoningTokens:   intOrZero(row.ReasoningTokens),
		AudioInputTokens:  intOrZero(row.AudioInputTokens),
		AudioOutputTokens: intOrZero(row.AudioOutputTokens),
		RawUsageJSON:      stringOrEmpty(row.RawUsageJSON),
		OccurredAt:        row.OccurredAt,
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
	}
}

func mapPricingVersion(row usagePricingVersionRow) gatewayusage.PricingVersion {
	var effectiveTo *time.Time
	if row.EffectiveTo.Valid {
		value := row.EffectiveTo.Time
		effectiveTo = &value
	}
	return gatewayusage.PricingVersion{
		ID:                    row.ID,
		ProviderID:            row.ProviderID,
		ModelName:             row.ModelName,
		Currency:              row.Currency,
		InputPerMillion:       row.InputPerMillion,
		OutputPerMillion:      row.OutputPerMillion,
		CachedInputPerMillion: row.CachedInputPerMillion,
		ReasoningPerMillion:   row.ReasoningPerMillion,
		AudioInputPerMillion:  row.AudioInputPerMillion,
		AudioOutputPerMillion: row.AudioOutputPerMillion,
		PerRequest:            row.PerRequest,
		Source:                row.Source,
		EffectiveFrom:         row.EffectiveFrom,
		EffectiveTo:           effectiveTo,
		IsActive:              row.IsActive,
		UpdatedBy:             stringOrEmpty(row.UpdatedBy),
		CreatedAt:             row.CreatedAt,
		UpdatedAt:             row.UpdatedAt,
	}
}

func nullString(value string) sql.NullString {
	if strings.TrimSpace(value) == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: strings.TrimSpace(value), Valid: true}
}

func nullInt64(value int64) sql.NullInt64 {
	if value == 0 {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: value, Valid: true}
}

func nullTime(value *time.Time) sql.NullTime {
	if value == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: value.UTC(), Valid: true}
}

func stringOrEmpty(value sql.NullString) string {
	if !value.Valid {
		return ""
	}
	return value.String
}

func intOrZero(value sql.NullInt64) int {
	if !value.Valid {
		return 0
	}
	return int(value.Int64)
}

func int64OrZero(value sql.NullInt64) int64 {
	if !value.Valid {
		return 0
	}
	return value.Int64
}
