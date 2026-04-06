package automationrepo

import (
	"context"
	"database/sql"
	"dreamcreator/internal/infrastructure/persistence/sqlitedto"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/uptrace/bun"

	automation "dreamcreator/internal/application/gateway/automation"
	"dreamcreator/internal/application/gateway/cron"
)

type SQLiteAutomationStore struct {
	db *bun.DB
}

type automationJobRow = sqlitedto.AutomationJobRow

type automationRunRow = sqlitedto.AutomationRunRow

type automationTriggerRow = sqlitedto.AutomationTriggerRow

type cronJobRow = sqlitedto.CronJobRow

type cronRunRow = sqlitedto.CronRunRow

type cronRunEventRow = sqlitedto.CronRunEventRow

func NewSQLiteAutomationStore(db *bun.DB) *SQLiteAutomationStore {
	return &SQLiteAutomationStore{db: db}
}

func (store *SQLiteAutomationStore) SaveJob(ctx context.Context, job automation.JobRecord) error {
	if store == nil || store.db == nil {
		return errors.New("automation store unavailable")
	}
	now := time.Now()
	if job.CreatedAt.IsZero() {
		job.CreatedAt = now
	}
	if job.UpdatedAt.IsZero() {
		job.UpdatedAt = job.CreatedAt
	}
	configJSON := ""
	if job.Config != nil {
		if data, err := json.Marshal(job.Config); err == nil {
			configJSON = string(data)
		}
	}
	row := automationJobRow{
		ID:         strings.TrimSpace(job.ID),
		Kind:       strings.TrimSpace(job.Kind),
		Status:     strings.TrimSpace(job.Status),
		ConfigJSON: configJSON,
		CreatedAt:  job.CreatedAt,
		UpdatedAt:  job.UpdatedAt,
	}
	_, err := store.db.NewInsert().Model(&row).
		On("CONFLICT(id) DO UPDATE").
		Set("kind = EXCLUDED.kind").
		Set("status = EXCLUDED.status").
		Set("config_json = EXCLUDED.config_json").
		Set("updated_at = EXCLUDED.updated_at").
		Exec(ctx)
	return err
}

func (store *SQLiteAutomationStore) SaveRun(ctx context.Context, run automation.RunRecord) error {
	if store == nil || store.db == nil {
		return errors.New("automation store unavailable")
	}
	startedAt := run.StartedAt
	if startedAt.IsZero() {
		startedAt = time.Now()
	}
	row := automationRunRow{
		ID:        strings.TrimSpace(run.ID),
		JobID:     strings.TrimSpace(run.JobID),
		Status:    strings.TrimSpace(run.Status),
		ErrorText: strings.TrimSpace(run.Error),
		StartedAt: startedAt,
		EndedAt:   run.EndedAt,
	}
	_, err := store.db.NewInsert().Model(&row).
		On("CONFLICT(id) DO UPDATE").
		Set("job_id = EXCLUDED.job_id").
		Set("status = EXCLUDED.status").
		Set("error = EXCLUDED.error").
		Set("started_at = EXCLUDED.started_at").
		Set("ended_at = EXCLUDED.ended_at").
		Exec(ctx)
	return err
}

func (store *SQLiteAutomationStore) SaveTriggerLog(ctx context.Context, log automation.TriggerLog) error {
	if store == nil || store.db == nil {
		return errors.New("automation store unavailable")
	}
	payload := ""
	if log.Payload != nil {
		if data, err := json.Marshal(log.Payload); err == nil {
			payload = string(data)
		}
	}
	row := automationTriggerRow{
		JobID:       strings.TrimSpace(log.JobID),
		EventID:     strings.TrimSpace(log.EventID),
		PayloadJSON: payload,
		CreatedAt:   time.Now(),
	}
	_, err := store.db.NewInsert().Model(&row).Exec(ctx)
	return err
}

func (store *SQLiteAutomationStore) SaveCronJob(ctx context.Context, job cron.CronJob) error {
	if store == nil || store.db == nil {
		return errors.New("automation store unavailable")
	}
	jobID := strings.TrimSpace(job.JobID)
	if jobID == "" {
		jobID = strings.TrimSpace(job.ID)
	}
	if jobID == "" {
		return errors.New("job id is required")
	}
	assistantID := strings.TrimSpace(job.AssistantID)
	if assistantID == "" {
		return errors.New("assistant id is required")
	}

	now := time.Now()
	createdAtMs := now.UnixMilli()
	if !job.CreatedAt.IsZero() {
		createdAtMs = job.CreatedAt.UnixMilli()
	}
	updatedAtMs := now.UnixMilli()

	scheduleJSON := "{}"
	if data, err := json.Marshal(job.Schedule); err == nil {
		scheduleJSON = string(data)
	}
	payloadJSON := "{}"
	if data, err := json.Marshal(job.PayloadSpec); err == nil {
		payloadJSON = string(data)
	}
	deliveryJSON := ""
	if job.Delivery != nil {
		if data, err := json.Marshal(job.Delivery); err == nil {
			deliveryJSON = string(data)
		}
	}
	stateJSON := "{}"
	if data, err := json.Marshal(job.State); err == nil {
		stateJSON = string(data)
	}

	row := cronJobRow{
		ID:             jobID,
		AssistantID:    assistantID,
		Name:           strings.TrimSpace(job.Name),
		Description:    strings.TrimSpace(job.Description),
		Enabled:        job.Enabled,
		DeleteAfterRun: job.DeleteAfterRun,
		ScheduleJSON:   scheduleJSON,
		PayloadJSON:    payloadJSON,
		DeliveryJSON:   deliveryJSON,
		SessionTarget:  strings.TrimSpace(job.SessionTarget),
		WakeMode:       strings.TrimSpace(job.WakeMode),
		SessionKey:     strings.TrimSpace(job.SessionKey),
		StateJSON:      stateJSON,
		CreatedAtMs:    createdAtMs,
		UpdatedAtMs:    updatedAtMs,
	}
	_, err := store.db.NewInsert().Model(&row).
		On("CONFLICT(id) DO UPDATE").
		Set("assistant_id = EXCLUDED.assistant_id").
		Set("name = EXCLUDED.name").
		Set("description = EXCLUDED.description").
		Set("enabled = EXCLUDED.enabled").
		Set("delete_after_run = EXCLUDED.delete_after_run").
		Set("schedule_json = EXCLUDED.schedule_json").
		Set("payload_json = EXCLUDED.payload_json").
		Set("delivery_json = EXCLUDED.delivery_json").
		Set("session_target = EXCLUDED.session_target").
		Set("wake_mode = EXCLUDED.wake_mode").
		Set("session_key = EXCLUDED.session_key").
		Set("state_json = EXCLUDED.state_json").
		Set("updated_at_ms = EXCLUDED.updated_at_ms").
		Exec(ctx)
	return err
}

func (store *SQLiteAutomationStore) ListCronJobs(ctx context.Context) ([]cron.CronJob, error) {
	if store == nil || store.db == nil {
		return nil, errors.New("automation store unavailable")
	}
	rows := make([]cronJobRow, 0)
	if err := store.db.NewSelect().Model(&rows).
		Order("updated_at_ms DESC").
		Scan(ctx); err != nil {
		return nil, err
	}
	result := make([]cron.CronJob, 0, len(rows))
	for _, row := range rows {
		var schedule cron.CronSchedule
		if strings.TrimSpace(row.ScheduleJSON) != "" {
			_ = json.Unmarshal([]byte(row.ScheduleJSON), &schedule)
		}
		var payload cron.CronPayload
		if strings.TrimSpace(row.PayloadJSON) != "" {
			_ = json.Unmarshal([]byte(row.PayloadJSON), &payload)
		}
		var delivery *cron.CronDelivery
		if strings.TrimSpace(row.DeliveryJSON) != "" {
			var decoded cron.CronDelivery
			if err := json.Unmarshal([]byte(row.DeliveryJSON), &decoded); err == nil {
				delivery = &decoded
			}
		}
		state := cron.CronJobState{}
		if strings.TrimSpace(row.StateJSON) != "" {
			_ = json.Unmarshal([]byte(row.StateJSON), &state)
		}

		job := cron.CronJob{
			ID:             row.ID,
			JobID:          row.ID,
			Name:           strings.TrimSpace(row.Name),
			Description:    strings.TrimSpace(row.Description),
			AssistantID:    strings.TrimSpace(row.AssistantID),
			Enabled:        row.Enabled,
			DeleteAfterRun: row.DeleteAfterRun,
			Schedule:       schedule,
			SessionTarget:  strings.TrimSpace(row.SessionTarget),
			WakeMode:       strings.TrimSpace(row.WakeMode),
			PayloadSpec:    payload,
			Delivery:       delivery,
			SessionKey:     strings.TrimSpace(row.SessionKey),
			State:          state,
		}
		if row.CreatedAtMs > 0 {
			job.CreatedAt = time.UnixMilli(row.CreatedAtMs)
		}
		if row.UpdatedAtMs > 0 {
			job.UpdatedAt = time.UnixMilli(row.UpdatedAtMs)
		}
		result = append(result, job)
	}
	return result, nil
}

func (store *SQLiteAutomationStore) DeleteCronJob(ctx context.Context, jobID string) error {
	if store == nil || store.db == nil {
		return errors.New("automation store unavailable")
	}
	_, err := store.db.NewDelete().
		Model((*cronJobRow)(nil)).
		Where("id = ?", strings.TrimSpace(jobID)).
		Exec(ctx)
	return err
}

func (store *SQLiteAutomationStore) SaveCronRun(ctx context.Context, run cron.CronRunRecord) error {
	if store == nil || store.db == nil {
		return errors.New("automation store unavailable")
	}
	runID := strings.TrimSpace(run.RunID)
	if runID == "" {
		return errors.New("run id is required")
	}
	jobID := strings.TrimSpace(run.JobID)
	if jobID == "" {
		return errors.New("job id is required")
	}
	startedAt := run.StartedAt
	if startedAt.IsZero() {
		startedAt = time.Now()
	}
	endedAtMs := int64(0)
	if !run.EndedAt.IsZero() {
		endedAtMs = run.EndedAt.UnixMilli()
	}
	durationMs := int64(0)
	if endedAtMs > 0 {
		durationMs = endedAtMs - startedAt.UnixMilli()
		if durationMs < 0 {
			durationMs = 0
		}
	}
	row := cronRunRow{
		RunID:          runID,
		JobID:          jobID,
		Status:         strings.TrimSpace(run.Status),
		ErrorText:      strings.TrimSpace(run.Error),
		Summary:        strings.TrimSpace(run.Summary),
		DeliveryStatus: strings.TrimSpace(run.DeliveryStatus),
		DeliveryError:  strings.TrimSpace(run.DeliveryError),
		SessionKey:     strings.TrimSpace(run.SessionKey),
		Model:          strings.TrimSpace(run.Model),
		Provider:       strings.TrimSpace(run.Provider),
		UsageJSON:      strings.TrimSpace(run.UsageJSON),
		RunAtMs:        startedAt.UnixMilli(),
		DurationMs:     durationMs,
		CreatedAtMs:    time.Now().UnixMilli(),
		EndedAtMs:      endedAtMs,
	}
	_, err := store.db.NewInsert().Model(&row).
		On("CONFLICT(run_id) DO UPDATE").
		Set("job_id = EXCLUDED.job_id").
		Set("status = EXCLUDED.status").
		Set("error = EXCLUDED.error").
		Set("summary = EXCLUDED.summary").
		Set("delivery_status = EXCLUDED.delivery_status").
		Set("delivery_error = EXCLUDED.delivery_error").
		Set("session_key = EXCLUDED.session_key").
		Set("model = EXCLUDED.model").
		Set("provider = EXCLUDED.provider").
		Set("usage_json = EXCLUDED.usage_json").
		Set("run_at_ms = EXCLUDED.run_at_ms").
		Set("duration_ms = EXCLUDED.duration_ms").
		Set("ended_at_ms = EXCLUDED.ended_at_ms").
		Exec(ctx)
	return err
}

func (store *SQLiteAutomationStore) GetCronRun(ctx context.Context, runID string) (cron.CronRunRecord, bool, error) {
	if store == nil || store.db == nil {
		return cron.CronRunRecord{}, false, errors.New("automation store unavailable")
	}
	trimmedRunID := strings.TrimSpace(runID)
	if trimmedRunID == "" {
		return cron.CronRunRecord{}, false, errors.New("run id is required")
	}
	var row cronRunRow
	err := store.db.NewSelect().
		Model(&row).
		Where("run_id = ?", trimmedRunID).
		Limit(1).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return cron.CronRunRecord{}, false, nil
		}
		return cron.CronRunRecord{}, false, err
	}
	startedAt := time.Time{}
	if row.RunAtMs > 0 {
		startedAt = time.UnixMilli(row.RunAtMs)
	}
	endedAt := time.Time{}
	if row.EndedAtMs > 0 {
		endedAt = time.UnixMilli(row.EndedAtMs)
	}
	return cron.CronRunRecord{
		RunID:          row.RunID,
		JobID:          row.JobID,
		Status:         row.Status,
		StartedAt:      startedAt,
		EndedAt:        endedAt,
		DeliveryStatus: row.DeliveryStatus,
		DeliveryError:  row.DeliveryError,
		Model:          row.Model,
		Provider:       row.Provider,
		SessionKey:     row.SessionKey,
		Summary:        row.Summary,
		UsageJSON:      row.UsageJSON,
		Error:          row.ErrorText,
	}, true, nil
}

func (store *SQLiteAutomationStore) ListCronRuns(ctx context.Context, query cron.ListRunsQuery) (cron.ListRunsResult, error) {
	if store == nil || store.db == nil {
		return cron.ListRunsResult{}, errors.New("automation store unavailable")
	}
	limit := query.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}
	offset := query.Offset
	if offset < 0 {
		offset = 0
	}
	jobID := strings.TrimSpace(query.JobID)
	status := strings.TrimSpace(query.Status)
	statuses := normalizeNonEmptyStrings(query.Statuses)
	if status != "" && strings.ToLower(status) != "all" && len(statuses) == 0 {
		statuses = []string{status}
	}
	deliveryStatus := strings.TrimSpace(query.DeliveryStatus)
	deliveryStatuses := normalizeNonEmptyStrings(query.DeliveryStatuses)
	if deliveryStatus != "" && strings.ToLower(deliveryStatus) != "all" && len(deliveryStatuses) == 0 {
		deliveryStatuses = []string{deliveryStatus}
	}
	searchQuery := strings.TrimSpace(query.Query)
	sortDir := strings.ToLower(strings.TrimSpace(query.SortDir))
	if sortDir != "asc" {
		sortDir = "desc"
	}

	buildQuery := func(model any) *bun.SelectQuery {
		selectQuery := store.db.NewSelect().Model(model)
		if jobID != "" {
			selectQuery = selectQuery.Where("job_id = ?", jobID)
		}
		if len(statuses) > 0 {
			selectQuery = selectQuery.Where("status IN (?)", bun.In(statuses))
		}
		if len(deliveryStatuses) > 0 {
			selectQuery = selectQuery.Where("delivery_status IN (?)", bun.In(deliveryStatuses))
		}
		if searchQuery != "" {
			like := "%" + strings.ToLower(searchQuery) + "%"
			selectQuery = selectQuery.Where("(LOWER(job_id) LIKE ? OR LOWER(error) LIKE ? OR LOWER(summary) LIKE ?)", like, like, like)
		}
		return selectQuery
	}

	total, err := buildQuery((*cronRunRow)(nil)).Count(ctx)
	if err != nil {
		return cron.ListRunsResult{}, err
	}

	rows := make([]cronRunRow, 0)
	if err := buildQuery(&rows).
		Order("run_at_ms " + strings.ToUpper(sortDir)).
		Limit(limit).
		Offset(offset).
		Scan(ctx); err != nil {
		return cron.ListRunsResult{}, err
	}

	items := make([]cron.CronRunRecord, 0, len(rows))
	for _, row := range rows {
		startedAt := time.Time{}
		if row.RunAtMs > 0 {
			startedAt = time.UnixMilli(row.RunAtMs)
		}
		endedAt := time.Time{}
		if row.EndedAtMs > 0 {
			endedAt = time.UnixMilli(row.EndedAtMs)
		}
		items = append(items, cron.CronRunRecord{
			RunID:          row.RunID,
			JobID:          row.JobID,
			Status:         row.Status,
			StartedAt:      startedAt,
			EndedAt:        endedAt,
			DeliveryStatus: row.DeliveryStatus,
			DeliveryError:  row.DeliveryError,
			Model:          row.Model,
			Provider:       row.Provider,
			SessionKey:     row.SessionKey,
			Summary:        row.Summary,
			UsageJSON:      row.UsageJSON,
			Error:          row.ErrorText,
		})
	}
	return cron.ListRunsResult{
		Items: items,
		Total: total,
	}, nil
}

func (store *SQLiteAutomationStore) SaveCronRunEvent(ctx context.Context, event cron.CronRunEvent) error {
	if store == nil || store.db == nil {
		return errors.New("automation store unavailable")
	}
	eventID := strings.TrimSpace(event.EventID)
	if eventID == "" {
		return errors.New("event id is required")
	}
	runID := strings.TrimSpace(event.RunID)
	if runID == "" {
		return errors.New("run id is required")
	}
	jobID := strings.TrimSpace(event.JobID)
	if jobID == "" {
		return errors.New("job id is required")
	}
	stage := strings.TrimSpace(event.Stage)
	if stage == "" {
		return errors.New("stage is required")
	}
	metaJSON := ""
	if event.Meta != nil {
		if data, err := json.Marshal(event.Meta); err == nil {
			metaJSON = string(data)
		}
	}
	createdAtMs := event.CreatedAt.UnixMilli()
	if createdAtMs <= 0 {
		createdAtMs = time.Now().UnixMilli()
	}
	row := cronRunEventRow{
		EventID:     eventID,
		RunID:       runID,
		JobID:       jobID,
		JobName:     strings.TrimSpace(event.JobName),
		Stage:       stage,
		Status:      strings.TrimSpace(event.Status),
		Message:     strings.TrimSpace(event.Message),
		ErrorText:   strings.TrimSpace(event.Error),
		Channel:     strings.TrimSpace(event.Channel),
		SessionKey:  strings.TrimSpace(event.SessionKey),
		Source:      strings.TrimSpace(event.Source),
		MetaJSON:    metaJSON,
		CreatedAtMs: createdAtMs,
	}
	_, err := store.db.NewInsert().
		Model(&row).
		On("CONFLICT(event_id) DO UPDATE").
		Set("run_id = EXCLUDED.run_id").
		Set("job_id = EXCLUDED.job_id").
		Set("job_name = EXCLUDED.job_name").
		Set("stage = EXCLUDED.stage").
		Set("status = EXCLUDED.status").
		Set("message = EXCLUDED.message").
		Set("error = EXCLUDED.error").
		Set("channel = EXCLUDED.channel").
		Set("session_key = EXCLUDED.session_key").
		Set("source = EXCLUDED.source").
		Set("meta_json = EXCLUDED.meta_json").
		Set("created_at_ms = EXCLUDED.created_at_ms").
		Exec(ctx)
	return err
}

func (store *SQLiteAutomationStore) ListCronRunEvents(ctx context.Context, query cron.ListRunEventsQuery) (cron.ListRunEventsResult, error) {
	if store == nil || store.db == nil {
		return cron.ListRunEventsResult{}, errors.New("automation store unavailable")
	}
	runID := strings.TrimSpace(query.RunID)
	if runID == "" {
		return cron.ListRunEventsResult{}, errors.New("run id is required")
	}
	limit := query.Limit
	if limit <= 0 {
		limit = 200
	}
	if limit > 1000 {
		limit = 1000
	}
	offset := query.Offset
	if offset < 0 {
		offset = 0
	}
	sortDir := strings.ToLower(strings.TrimSpace(query.SortDir))
	if sortDir != "asc" {
		sortDir = "desc"
	}
	baseQuery := store.db.NewSelect().Model((*cronRunEventRow)(nil)).Where("run_id = ?", runID)
	total, err := baseQuery.Count(ctx)
	if err != nil {
		return cron.ListRunEventsResult{}, err
	}
	rows := make([]cronRunEventRow, 0)
	if err := store.db.NewSelect().
		Model(&rows).
		Where("run_id = ?", runID).
		Order("created_at_ms " + strings.ToUpper(sortDir)).
		Limit(limit).
		Offset(offset).
		Scan(ctx); err != nil {
		return cron.ListRunEventsResult{}, err
	}
	items := make([]cron.CronRunEvent, 0, len(rows))
	for _, row := range rows {
		meta := map[string]any{}
		if strings.TrimSpace(row.MetaJSON) != "" {
			if err := json.Unmarshal([]byte(row.MetaJSON), &meta); err != nil {
				meta = map[string]any{
					"raw": row.MetaJSON,
				}
			}
		}
		createdAt := time.Time{}
		if row.CreatedAtMs > 0 {
			createdAt = time.UnixMilli(row.CreatedAtMs)
		}
		items = append(items, cron.CronRunEvent{
			EventID:    row.EventID,
			RunID:      row.RunID,
			JobID:      row.JobID,
			JobName:    row.JobName,
			Stage:      row.Stage,
			Status:     row.Status,
			Message:    row.Message,
			Error:      row.ErrorText,
			Channel:    row.Channel,
			SessionKey: row.SessionKey,
			Source:     row.Source,
			Meta:       meta,
			CreatedAt:  createdAt,
		})
	}
	return cron.ListRunEventsResult{
		Items: items,
		Total: total,
	}, nil
}

func normalizeNonEmptyStrings(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		result = append(result, trimmed)
	}
	return result
}
