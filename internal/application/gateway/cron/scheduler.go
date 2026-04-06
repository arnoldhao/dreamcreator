package cron

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	automation "dreamcreator/internal/application/gateway/automation"
)

const defaultPendingDeliveryTimeout = 90 * time.Second

type Store interface {
	SaveCronJob(ctx context.Context, job CronJob) error
	ListCronJobs(ctx context.Context) ([]CronJob, error)
	DeleteCronJob(ctx context.Context, jobID string) error
	SaveCronRun(ctx context.Context, run CronRunRecord) error
	GetCronRun(ctx context.Context, runID string) (CronRunRecord, bool, error)
	ListCronRuns(ctx context.Context, query ListRunsQuery) (ListRunsResult, error)
	SaveCronRunEvent(ctx context.Context, event CronRunEvent) error
	ListCronRunEvents(ctx context.Context, query ListRunEventsQuery) (ListRunEventsResult, error)
}

type AnnouncementRequest struct {
	RunID       string
	JobID       string
	JobName     string
	AssistantID string
	SessionKey  string
	Channel     string
	Message     string
}

type AnnouncementSender func(ctx context.Context, request AnnouncementRequest) error

type WebhookRequest struct {
	RunID       string
	JobID       string
	JobName     string
	AssistantID string
	SessionKey  string
	URL         string
	Status      string
	Summary     string
	Error       string
	Payload     map[string]any
}

type WebhookSender func(ctx context.Context, request WebhookRequest) error

type IsolatedExecutionRequest struct {
	RunID          string
	JobID          string
	JobName        string
	AssistantID    string
	SessionKey     string
	Message        string
	Model          string
	Thinking       string
	TimeoutSeconds int
	LightContext   bool
	Delivery       *CronDelivery
}

type IsolatedExecutionResult struct {
	Status     string
	Error      string
	Summary    string
	SessionKey string
	Model      string
	Provider   string
	UsageJSON  string
}

type IsolatedExecutor func(ctx context.Context, request IsolatedExecutionRequest) (IsolatedExecutionResult, error)

type HeartbeatDeliveryEvent struct {
	RunID      string
	Source     string
	Status     string
	Message    string
	Error      string
	SessionKey string
}

type pendingRunDelivery struct {
	RunID       string
	JobID       string
	JobName     string
	AssistantID string
	Status      string
	Model       string
	Provider    string
	UsageJSON   string
	StartedAt   time.Time
	EndedAt     time.Time
	PendingAt   time.Time
	SessionKey  string
	Channel     string
}

type MainSystemEventRequest struct {
	SessionKey string
	Text       string
	ContextKey string
	RunID      string
}

type MainSystemEventEnqueuer func(ctx context.Context, request MainSystemEventRequest) bool

type WakeTriggerRequest struct {
	Reason     string
	SessionKey string
	Force      bool
	Source     string
	RunID      string
}

type WakeTriggerResult struct {
	Accepted       bool
	ExecutedStatus string
	Reason         string
}

type WakeTrigger func(ctx context.Context, request WakeTriggerRequest) WakeTriggerResult

type RunRealtimeEvent struct {
	RunID          string    `json:"runId"`
	JobID          string    `json:"jobId,omitempty"`
	Stage          string    `json:"stage,omitempty"`
	Status         string    `json:"status,omitempty"`
	DeliveryStatus string    `json:"deliveryStatus,omitempty"`
	Source         string    `json:"source,omitempty"`
	CreatedAt      time.Time `json:"createdAt"`
}

type RunRealtimeNotifier func(ctx context.Context, event RunRealtimeEvent)
type AssistantIDResolver func(ctx context.Context) string

type Scheduler struct {
	mu                     sync.RWMutex
	enabled                bool
	jobs                   map[string]CronJob
	nextRuns               map[string]time.Time
	retries                map[string]int
	pendingDeliveries      map[string]pendingRunDelivery
	store                  Store
	engine                 *automation.Engine
	now                    func() time.Time
	announceSender         AnnouncementSender
	webhookSender          WebhookSender
	mainSystemEventEnqueue MainSystemEventEnqueuer
	wakeTrigger            WakeTrigger
	isolatedExecutor       IsolatedExecutor
	runRealtimeNotifier    RunRealtimeNotifier
	assistantIDResolver    AssistantIDResolver
}

func NewScheduler(store Store, engine *automation.Engine) *Scheduler {
	return &Scheduler{
		enabled:           true,
		jobs:              make(map[string]CronJob),
		nextRuns:          make(map[string]time.Time),
		retries:           make(map[string]int),
		pendingDeliveries: make(map[string]pendingRunDelivery),
		store:             store,
		engine:            engine,
		now:               time.Now,
	}
}

func (scheduler *Scheduler) Start(ctx context.Context) func() {
	if scheduler == nil {
		return func() {}
	}
	scheduler.loadJobs(ctx)
	scheduler.recoverPendingRuns(ctx)
	stop := make(chan struct{})
	ticker := time.NewTicker(1 * time.Second)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-stop:
				return
			case <-ticker.C:
				scheduler.tick(ctx)
			}
		}
	}()
	return func() { close(stop) }
}

func (scheduler *Scheduler) recoverPendingRuns(ctx context.Context) {
	if scheduler == nil || scheduler.store == nil {
		return
	}
	result, err := scheduler.store.ListCronRuns(ctx, ListRunsQuery{
		Statuses:         []string{"running"},
		DeliveryStatuses: []string{"pending"},
		SortDir:          "asc",
		Limit:            500,
		Offset:           0,
	})
	if err != nil {
		return
	}
	if len(result.Items) == 0 {
		return
	}
	now := scheduler.now()
	for _, item := range result.Items {
		run := item
		if !strings.EqualFold(strings.TrimSpace(run.Status), "running") || !strings.EqualFold(strings.TrimSpace(run.DeliveryStatus), "pending") {
			continue
		}
		run.Status = "failed"
		run.Error = "delivery pending lost after scheduler restart"
		run.DeliveryStatus = "failed"
		run.DeliveryError = "delivery pending lost after scheduler restart"
		if run.EndedAt.IsZero() {
			run.EndedAt = now
		}
		run.LatestStage = "delivery_failed"
		run.Summary = summarizeRun(run)
		scheduler.appendRunEvent(ctx, buildRunEvent(run, "delivery_failed", "failed", "", run.DeliveryError, "", "scheduler", map[string]any{
			"reason": "scheduler_recover",
		}))
		scheduler.appendRunEvent(ctx, buildRunEvent(run, "failed", "failed", "", run.Error, "", "scheduler", map[string]any{
			"reason": "scheduler_recover",
		}))
		scheduler.persistDeliveryResult(ctx, run)
		scheduler.updateRunState(ctx, run)
	}
}

func (scheduler *Scheduler) IsEnabled() bool {
	if scheduler == nil {
		return false
	}
	scheduler.mu.RLock()
	defer scheduler.mu.RUnlock()
	return scheduler.enabled
}

func (scheduler *Scheduler) SetEnabled(enabled bool) {
	if scheduler == nil {
		return
	}
	scheduler.mu.Lock()
	scheduler.enabled = enabled
	scheduler.mu.Unlock()
}

func (scheduler *Scheduler) SetAnnouncementSender(sender AnnouncementSender) {
	if scheduler == nil {
		return
	}
	scheduler.mu.Lock()
	scheduler.announceSender = sender
	scheduler.mu.Unlock()
}

func (scheduler *Scheduler) SetWebhookSender(sender WebhookSender) {
	if scheduler == nil {
		return
	}
	scheduler.mu.Lock()
	scheduler.webhookSender = sender
	scheduler.mu.Unlock()
}

func (scheduler *Scheduler) SetMainSystemEventEnqueuer(enqueuer MainSystemEventEnqueuer) {
	if scheduler == nil {
		return
	}
	scheduler.mu.Lock()
	scheduler.mainSystemEventEnqueue = enqueuer
	scheduler.mu.Unlock()
}

func (scheduler *Scheduler) SetWakeTrigger(trigger WakeTrigger) {
	if scheduler == nil {
		return
	}
	scheduler.mu.Lock()
	scheduler.wakeTrigger = trigger
	scheduler.mu.Unlock()
}

func (scheduler *Scheduler) SetIsolatedExecutor(executor IsolatedExecutor) {
	if scheduler == nil {
		return
	}
	scheduler.mu.Lock()
	scheduler.isolatedExecutor = executor
	scheduler.mu.Unlock()
}

func (scheduler *Scheduler) SetRunRealtimeNotifier(notifier RunRealtimeNotifier) {
	if scheduler == nil {
		return
	}
	scheduler.mu.Lock()
	scheduler.runRealtimeNotifier = notifier
	scheduler.mu.Unlock()
}

func (scheduler *Scheduler) SetAssistantIDResolver(resolver AssistantIDResolver) {
	if scheduler == nil {
		return
	}
	scheduler.mu.Lock()
	scheduler.assistantIDResolver = resolver
	scheduler.mu.Unlock()
}

func (scheduler *Scheduler) assistantResolver() AssistantIDResolver {
	if scheduler == nil {
		return nil
	}
	scheduler.mu.RLock()
	defer scheduler.mu.RUnlock()
	return scheduler.assistantIDResolver
}

func (scheduler *Scheduler) resolveAssistantID(ctx context.Context, fallback string) string {
	resolver := scheduler.assistantResolver()
	if resolver != nil {
		if resolved := strings.TrimSpace(resolver(ctx)); resolved != "" {
			return resolved
		}
	}
	return strings.TrimSpace(fallback)
}

func (scheduler *Scheduler) Register(ctx context.Context, job CronJob) (CronJob, error) {
	return scheduler.Upsert(ctx, job)
}

func (scheduler *Scheduler) Upsert(ctx context.Context, job CronJob) (CronJob, error) {
	if scheduler == nil {
		return CronJob{}, errors.New("scheduler unavailable")
	}

	now := scheduler.now()
	job = normalizeJob(job)
	job.AssistantID = scheduler.resolveAssistantID(ctx, job.AssistantID)
	if err := validateJobSemantics(job); err != nil {
		return CronJob{}, err
	}
	if err := validateJobSchedule(job, now); err != nil {
		return CronJob{}, err
	}

	scheduler.mu.Lock()
	if existing, ok := scheduler.jobs[job.JobID]; ok {
		if job.CreatedAt.IsZero() {
			job.CreatedAt = existing.CreatedAt
		}
		job.State = mergeJobState(existing.State, job.State)
	}
	if job.CreatedAt.IsZero() {
		job.CreatedAt = now
	}
	job.UpdatedAt = now
	next := time.Time{}
	if job.Enabled {
		next, _ = nextRunForJob(job, now)
	}
	job = syncNextRunState(job, next)
	scheduler.jobs[job.JobID] = job
	scheduler.nextRuns[job.JobID] = next
	scheduler.mu.Unlock()

	if scheduler.store != nil {
		_ = scheduler.store.SaveCronJob(ctx, job)
	}
	scheduler.emitRunRealtimeEvent(ctx, RunRealtimeEvent{
		JobID:     strings.TrimSpace(job.JobID),
		Stage:     "job_upserted",
		Status:    normalizeRunStatus(job.State.LastRunStatus),
		Source:    "scheduler",
		CreatedAt: now,
	})
	return job, nil
}

func (scheduler *Scheduler) List(_ context.Context) ([]CronJob, error) {
	if scheduler == nil {
		return nil, errors.New("scheduler unavailable")
	}
	scheduler.mu.RLock()
	defer scheduler.mu.RUnlock()
	result := make([]CronJob, 0, len(scheduler.jobs))
	for jobID, job := range scheduler.jobs {
		if next, ok := scheduler.nextRuns[jobID]; ok {
			job = syncNextRunState(job, next)
		}
		result = append(result, job)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].UpdatedAt.Equal(result[j].UpdatedAt) {
			return result[i].JobID < result[j].JobID
		}
		return result[i].UpdatedAt.After(result[j].UpdatedAt)
	})
	return result, nil
}

func (scheduler *Scheduler) Delete(ctx context.Context, jobID string) error {
	if scheduler == nil {
		return errors.New("scheduler unavailable")
	}
	trimmed := strings.TrimSpace(jobID)
	if trimmed == "" {
		return errors.New("job id is required")
	}

	scheduler.mu.Lock()
	if _, ok := scheduler.jobs[trimmed]; !ok {
		scheduler.mu.Unlock()
		return errors.New("job not found")
	}
	delete(scheduler.jobs, trimmed)
	delete(scheduler.nextRuns, trimmed)
	delete(scheduler.retries, trimmed)
	scheduler.mu.Unlock()

	if scheduler.store != nil {
		if err := scheduler.store.DeleteCronJob(ctx, trimmed); err != nil {
			return err
		}
	}
	scheduler.emitRunRealtimeEvent(ctx, RunRealtimeEvent{
		JobID:     trimmed,
		Stage:     "job_deleted",
		Source:    "scheduler",
		CreatedAt: scheduler.now(),
	})
	return nil
}

func (scheduler *Scheduler) Status() Status {
	if scheduler == nil {
		return Status{}
	}
	scheduler.mu.RLock()
	defer scheduler.mu.RUnlock()

	nextWake := time.Time{}
	for jobID, next := range scheduler.nextRuns {
		if next.IsZero() {
			continue
		}
		job := scheduler.jobs[jobID]
		if !job.Enabled {
			continue
		}
		if nextWake.IsZero() || next.Before(nextWake) {
			nextWake = next
		}
	}
	status := Status{
		Enabled: scheduler.enabled,
		Jobs:    len(scheduler.jobs),
	}
	if !nextWake.IsZero() {
		status.NextWakeAt = nextWake.UTC().Format(time.RFC3339)
		status.NextWakeAtMs = nextWake.UnixMilli()
	}
	return status
}

func (scheduler *Scheduler) RunJob(ctx context.Context, jobID string) (CronRunRecord, error) {
	return scheduler.RunJobWithMode(ctx, jobID, "force")
}

func (scheduler *Scheduler) RunJobWithMode(ctx context.Context, jobID string, mode string) (CronRunRecord, error) {
	if scheduler == nil {
		return CronRunRecord{}, errors.New("scheduler unavailable")
	}
	normalizedMode, err := normalizeRunMode(mode)
	if err != nil {
		return CronRunRecord{}, err
	}
	trimmedID := strings.TrimSpace(jobID)
	job, ok := scheduler.lookup(trimmedID)
	if !ok {
		return CronRunRecord{}, errors.New("job not found")
	}
	if normalizedMode == "due" {
		next := scheduler.nextRun(trimmedID)
		now := scheduler.now()
		if next.IsZero() || now.Before(next) {
			return CronRunRecord{
				RunID:     uuid.NewString(),
				JobID:     trimmedID,
				JobName:   strings.TrimSpace(job.Name),
				Status:    "skipped",
				Error:     "job is not due",
				StartedAt: now,
				EndedAt:   now,
			}, nil
		}
	}
	return scheduler.executeJob(ctx, job)
}

func (scheduler *Scheduler) Wake(ctx context.Context, mode string, text string, sessionKey string) (WakeResult, error) {
	if scheduler == nil {
		return WakeResult{}, errors.New("scheduler unavailable")
	}
	normalizedMode := strings.ToLower(strings.TrimSpace(mode))
	if normalizedMode == "" {
		normalizedMode = "next-heartbeat"
	}
	if normalizedMode != "next-heartbeat" && normalizedMode != "now" {
		return WakeResult{}, errors.New("mode must be now or next-heartbeat")
	}

	targetSessionKey := strings.TrimSpace(sessionKey)
	trimmedText := strings.TrimSpace(text)
	result := WakeResult{
		Mode:       normalizedMode,
		Text:       trimmedText,
		SessionKey: targetSessionKey,
	}

	if enqueuer := scheduler.mainSystemEventEnqueuer(); enqueuer != nil {
		resolvedSessionKey := normalizeMainSessionKeyForHeartbeat(targetSessionKey)
		wakeRunID := ""
		if trimmedText != "" {
			wakeRunID = uuid.NewString()
			queued := enqueuer(ctx, MainSystemEventRequest{
				SessionKey: resolvedSessionKey,
				Text:       trimmedText,
				ContextKey: "cron:wake",
				RunID:      wakeRunID,
			})
			if !queued {
				return result, errors.New("failed to enqueue wake text as system event")
			}
		}
		if trigger := scheduler.wakeTriggerFunc(); trigger != nil {
			wakeResult := trigger(ctx, buildWakeTriggerRequest(normalizedMode, resolvedSessionKey, "cron", wakeRunID))
			result.OK = wakeResult.Accepted
			result.Accepted = wakeResult.Accepted
			result.SessionKey = resolvedSessionKey
			if !wakeResult.Accepted {
				if reason := strings.TrimSpace(wakeResult.Reason); reason != "" {
					return result, fmt.Errorf("heartbeat wake rejected: %s", reason)
				}
				return result, errors.New("heartbeat wake rejected")
			}
			return result, nil
		}
		result.OK = true
		result.Accepted = true
		result.SessionKey = resolvedSessionKey
		return result, nil
	}

	if targetSessionKey == "" {
		targetSessionKey = "cron/default"
	}
	actionType := "cron.wake"
	if normalizedMode == "now" {
		actionType = "cron.wake.now"
	}
	if scheduler.engine == nil {
		return result, errors.New("automation engine unavailable")
	}
	_, err := scheduler.engine.Trigger(ctx, automation.AutomationAction{
		Type:       actionType,
		SessionKey: targetSessionKey,
		Payload: map[string]any{
			"mode": normalizedMode,
			"text": trimmedText,
		},
	})
	if err != nil {
		return result, err
	}
	result.OK = true
	result.Accepted = true
	return result, nil
}

func (scheduler *Scheduler) ListRuns(ctx context.Context, query ListRunsQuery) (ListRunsResult, error) {
	if scheduler == nil {
		return ListRunsResult{}, errors.New("scheduler unavailable")
	}
	if scheduler.store == nil {
		return ListRunsResult{}, nil
	}
	return scheduler.store.ListCronRuns(ctx, query)
}

func (scheduler *Scheduler) RunDetail(ctx context.Context, runID string, eventsLimit int) (RunDetail, error) {
	if scheduler == nil {
		return RunDetail{}, errors.New("scheduler unavailable")
	}
	if scheduler.store == nil {
		return RunDetail{}, errors.New("cron store unavailable")
	}
	trimmedRunID := strings.TrimSpace(runID)
	if trimmedRunID == "" {
		return RunDetail{}, errors.New("runId is required")
	}
	run, ok, err := scheduler.store.GetCronRun(ctx, trimmedRunID)
	if err != nil {
		return RunDetail{}, err
	}
	if !ok {
		return RunDetail{}, errors.New("run not found")
	}
	limit := eventsLimit
	if limit <= 0 {
		limit = 200
	}
	eventsResult, err := scheduler.store.ListCronRunEvents(ctx, ListRunEventsQuery{
		RunID:   trimmedRunID,
		SortDir: "asc",
		Limit:   limit,
		Offset:  0,
	})
	if err != nil {
		return RunDetail{}, err
	}
	return RunDetail{
		Run:         run,
		Events:      eventsResult.Items,
		EventsTotal: eventsResult.Total,
	}, nil
}

func (scheduler *Scheduler) ListRunEvents(ctx context.Context, query ListRunEventsQuery) (ListRunEventsResult, error) {
	if scheduler == nil {
		return ListRunEventsResult{}, errors.New("scheduler unavailable")
	}
	if scheduler.store == nil {
		return ListRunEventsResult{}, errors.New("cron store unavailable")
	}
	if strings.TrimSpace(query.RunID) == "" {
		return ListRunEventsResult{}, errors.New("runId is required")
	}
	return scheduler.store.ListCronRunEvents(ctx, query)
}

func (scheduler *Scheduler) loadJobs(ctx context.Context) {
	if scheduler == nil || scheduler.store == nil {
		return
	}
	jobs, err := scheduler.store.ListCronJobs(ctx)
	if err != nil {
		return
	}
	now := scheduler.now()
	scheduler.mu.Lock()
	for _, raw := range jobs {
		job := normalizeJob(raw)
		if err := validateJobSemantics(job); err != nil {
			continue
		}
		if err := validateJobSchedule(job, now); err != nil {
			continue
		}
		next := time.Time{}
		if job.Enabled {
			next, _ = nextRunForJob(job, now)
		}
		job = syncNextRunState(job, next)
		scheduler.jobs[job.JobID] = job
		scheduler.nextRuns[job.JobID] = next
	}
	scheduler.mu.Unlock()
}

func (scheduler *Scheduler) tick(ctx context.Context) {
	now := scheduler.now()
	scheduler.finalizeExpiredPendingDeliveries(ctx, now)
	scheduler.mu.RLock()
	enabled := scheduler.enabled
	scheduler.mu.RUnlock()
	if !enabled {
		return
	}
	jobs := scheduler.snapshot()
	for _, job := range jobs {
		if !job.Enabled {
			continue
		}
		next := scheduler.nextRun(job.JobID)
		if next.IsZero() {
			scheduler.scheduleNext(job, now)
			next = scheduler.nextRun(job.JobID)
			if next.IsZero() {
				continue
			}
		}
		if now.After(next) || now.Equal(next) {
			_, _ = scheduler.executeJob(ctx, job)
		}
	}
}

func (scheduler *Scheduler) executeJob(ctx context.Context, job CronJob) (CronRunRecord, error) {
	if scheduler == nil {
		return CronRunRecord{}, errors.New("scheduler unavailable")
	}
	job.AssistantID = scheduler.resolveAssistantID(ctx, job.AssistantID)
	started := scheduler.now()
	scheduler.markJobRunning(ctx, job.JobID, started)
	providerID, modelName := splitModelRef(strings.TrimSpace(job.PayloadSpec.Model))
	run := CronRunRecord{
		RunID:      uuid.NewString(),
		JobID:      job.JobID,
		JobName:    job.Name,
		Status:     "running",
		Model:      modelName,
		Provider:   providerID,
		StartedAt:  started,
		SessionKey: strings.TrimSpace(job.SessionKey),
	}
	if scheduler.store != nil {
		_ = scheduler.store.SaveCronRun(ctx, run)
	}
	scheduler.appendRunEvent(ctx, buildRunEvent(run, "started", "running", "", "", "", "scheduler", nil))

	if handled, err := scheduler.tryExecuteIsolated(ctx, job, &run); handled {
		scheduler.applyRunDelivery(ctx, job, &run)
		if scheduler.store != nil {
			_ = scheduler.store.SaveCronRun(ctx, run)
		}
		switch strings.ToLower(strings.TrimSpace(run.Status)) {
		case "failed", "error":
			scheduler.appendRunEvent(ctx, buildRunEvent(run, "action_failed", "failed", "", strings.TrimSpace(run.Error), "", "scheduler", map[string]any{
				"sessionTarget": strings.TrimSpace(job.SessionTarget),
				"payloadKind":   normalizePayloadKind(job.PayloadSpec.Kind),
				"source":        "isolated_executor",
			}))
			scheduler.appendRunEvent(ctx, buildRunEvent(run, "failed", "failed", "", strings.TrimSpace(run.Error), "", "scheduler", nil))
			scheduler.updateRunState(ctx, run)
			scheduler.scheduleRetry(job)
			if err != nil {
				return run, err
			}
			if strings.TrimSpace(run.Error) == "" {
				return run, errors.New("isolated execution failed")
			}
			return run, errors.New(strings.TrimSpace(run.Error))
		case "skipped":
			scheduler.appendRunEvent(ctx, buildRunEvent(run, "skipped", "skipped", run.Summary, "", "", "scheduler", map[string]any{
				"sessionTarget": strings.TrimSpace(job.SessionTarget),
				"payloadKind":   normalizePayloadKind(job.PayloadSpec.Kind),
				"source":        "isolated_executor",
			}))
			scheduler.updateRunState(ctx, run)
			scheduler.resetRetry(job.JobID)
			scheduler.scheduleAfterSuccess(ctx, job)
			return run, nil
		default:
			scheduler.appendRunEvent(ctx, buildRunEvent(run, "completed", "completed", run.Summary, "", "", "scheduler", map[string]any{
				"sessionTarget": strings.TrimSpace(job.SessionTarget),
				"payloadKind":   normalizePayloadKind(job.PayloadSpec.Kind),
				"source":        "isolated_executor",
			}))
			scheduler.updateRunState(ctx, run)
			scheduler.resetRetry(job.JobID)
			scheduler.scheduleAfterSuccess(ctx, job)
			return run, nil
		}
	}

	err := scheduler.triggerAction(ctx, job, strings.TrimSpace(run.RunID))
	if err != nil {
		run.Status = "failed"
		run.Error = err.Error()
		run.Summary = strings.TrimSpace(run.Error)
		run.LatestStage = "failed"
		run.EndedAt = scheduler.now()
		scheduler.applyRunDelivery(ctx, job, &run)
		if scheduler.store != nil {
			_ = scheduler.store.SaveCronRun(ctx, run)
		}
		scheduler.appendRunEvent(ctx, buildRunEvent(run, "action_failed", "failed", "", run.Error, "", "scheduler", map[string]any{
			"sessionTarget": strings.TrimSpace(job.SessionTarget),
			"payloadKind":   normalizePayloadKind(job.PayloadSpec.Kind),
		}))
		scheduler.appendRunEvent(ctx, buildRunEvent(run, "failed", "failed", "", run.Error, "", "scheduler", nil))
		scheduler.updateRunState(ctx, run)
		scheduler.scheduleRetry(job)
		return run, err
	}
	run.LatestStage = "action_accepted"
	scheduler.appendRunEvent(ctx, buildRunEvent(run, "action_accepted", "running", "", "", "", "scheduler", map[string]any{
		"sessionTarget": strings.TrimSpace(job.SessionTarget),
		"payloadKind":   normalizePayloadKind(job.PayloadSpec.Kind),
		"wakeMode":      normalizeWakeMode(strings.TrimSpace(job.WakeMode)),
	}))

	if scheduler.shouldFinalizeViaHeartbeat(job) {
		scheduler.applyRunDelivery(ctx, job, &run)
		if scheduler.store != nil {
			_ = scheduler.store.SaveCronRun(ctx, run)
		}
		scheduler.resetRetry(job.JobID)
		scheduler.scheduleAfterSuccess(ctx, job)
		return run, nil
	}

	run.Status = "completed"
	run.Summary = summarizeRun(run)
	run.LatestStage = "completed"
	run.EndedAt = scheduler.now()
	scheduler.applyRunDelivery(ctx, job, &run)
	if scheduler.store != nil {
		_ = scheduler.store.SaveCronRun(ctx, run)
	}
	scheduler.appendRunEvent(ctx, buildRunEvent(run, "completed", "completed", run.Summary, "", "", "scheduler", nil))
	scheduler.updateRunState(ctx, run)
	scheduler.resetRetry(job.JobID)
	scheduler.scheduleAfterSuccess(ctx, job)
	return run, nil
}

func (scheduler *Scheduler) tryExecuteIsolated(ctx context.Context, job CronJob, run *CronRunRecord) (bool, error) {
	if scheduler == nil || run == nil {
		return false, nil
	}
	if !strings.EqualFold(strings.TrimSpace(job.SessionTarget), "isolated") {
		return false, nil
	}
	if normalizePayloadKind(job.PayloadSpec.Kind) != "agentTurn" {
		return false, nil
	}
	executor := scheduler.isolatedExecutorFunc()
	if executor == nil {
		return false, nil
	}

	request := IsolatedExecutionRequest{
		RunID:          strings.TrimSpace(run.RunID),
		JobID:          strings.TrimSpace(job.JobID),
		JobName:        strings.TrimSpace(job.Name),
		AssistantID:    strings.TrimSpace(job.AssistantID),
		SessionKey:     strings.TrimSpace(job.SessionKey),
		Message:        strings.TrimSpace(job.PayloadSpec.Message),
		Model:          strings.TrimSpace(job.PayloadSpec.Model),
		Thinking:       strings.TrimSpace(job.PayloadSpec.Thinking),
		TimeoutSeconds: job.PayloadSpec.TimeoutSeconds,
		LightContext:   job.PayloadSpec.LightContext,
		Delivery:       normalizeDelivery(job.Delivery),
	}

	result, err := executor(ctx, request)
	run.EndedAt = scheduler.now()
	if err != nil {
		run.Status = "failed"
		run.Error = strings.TrimSpace(err.Error())
		run.Summary = summarizeRun(*run)
		run.LatestStage = "failed"
		return true, err
	}

	status := normalizeDirectRunStatus(result.Status)
	run.Status = status
	if text := strings.TrimSpace(result.Error); text != "" {
		run.Error = text
	}
	if text := strings.TrimSpace(result.Summary); text != "" {
		run.Summary = text
	}
	if text := strings.TrimSpace(result.SessionKey); text != "" {
		run.SessionKey = text
	}
	if text := strings.TrimSpace(result.Model); text != "" {
		run.Model = text
	}
	if text := strings.TrimSpace(result.Provider); text != "" {
		run.Provider = text
	}
	if text := strings.TrimSpace(result.UsageJSON); text != "" {
		run.UsageJSON = text
	}

	switch status {
	case "completed":
		if strings.TrimSpace(run.Summary) == "" {
			run.Summary = summarizeRun(*run)
		}
		run.LatestStage = "completed"
		return true, nil
	case "skipped":
		if strings.TrimSpace(run.Summary) == "" {
			run.Summary = summarizeRun(*run)
		}
		run.LatestStage = "skipped"
		return true, nil
	default:
		if strings.TrimSpace(run.Error) == "" {
			run.Error = "isolated execution failed"
		}
		run.Summary = summarizeRun(*run)
		run.LatestStage = "failed"
		return true, errors.New(strings.TrimSpace(run.Error))
	}
}

func normalizeDirectRunStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "ok", "completed", "success":
		return "completed"
	case "skipped":
		return "skipped"
	case "failed", "error":
		return "failed"
	default:
		return "completed"
	}
}

func (scheduler *Scheduler) triggerAction(ctx context.Context, job CronJob, runID string) error {
	payloadKind := normalizePayloadKind(job.PayloadSpec.Kind)
	if payloadKind == "systemEvent" {
		if err := scheduler.triggerMainSystemEvent(ctx, job, runID); err == nil {
			return nil
		} else if scheduler.mainSystemEventEnqueuer() != nil {
			return err
		}
	}

	if scheduler.engine == nil {
		return errors.New("automation engine unavailable")
	}
	actionType := "cron.system_event"
	if payloadKind == "agentTurn" {
		actionType = "cron.agent_turn"
	}
	sessionKey := strings.TrimSpace(job.SessionKey)
	if sessionKey == "" {
		if strings.EqualFold(strings.TrimSpace(job.SessionTarget), "main") {
			sessionKey = "cron/main"
		} else {
			sessionKey = "cron/isolated"
		}
	}
	action := automation.AutomationAction{
		Type:       actionType,
		SessionKey: sessionKey,
		Payload:    buildActionPayload(job),
	}
	_, err := scheduler.engine.Trigger(ctx, action)
	return err
}

func (scheduler *Scheduler) applyRunDelivery(ctx context.Context, job CronJob, run *CronRunRecord) {
	if scheduler == nil || run == nil {
		return
	}
	delivery := normalizeDelivery(job.Delivery)
	if delivery == nil {
		return
	}
	switch strings.ToLower(strings.TrimSpace(delivery.Mode)) {
	case "none", "":
		// No-op for primary delivery mode.
	case "announce":
		scheduler.applyAnnounceDelivery(ctx, job, run, delivery)
	case "webhook":
		scheduler.applyWebhookDelivery(ctx, job, run, delivery)
	default:
		run.DeliveryStatus = "failed"
		run.DeliveryError = "delivery.mode must be one of: none, announce, webhook"
		run.LatestStage = "delivery_failed"
		run.Summary = summarizeRun(*run)
		scheduler.appendRunEvent(ctx, buildRunEvent(*run, "delivery_failed", "failed", "", run.DeliveryError, "", "scheduler", nil))
	}
	if strings.EqualFold(strings.TrimSpace(run.Status), "failed") || strings.EqualFold(strings.TrimSpace(run.Status), "error") {
		scheduler.applyFailureDestination(ctx, job, run, delivery)
	}
}

func (scheduler *Scheduler) applyAnnounceDelivery(ctx context.Context, job CronJob, run *CronRunRecord, delivery *CronDelivery) {
	if scheduler == nil || run == nil || delivery == nil {
		return
	}
	channel := normalizeAnnounceChannel(delivery.Channel)
	if !isValidAnnounceChannel(channel) {
		run.DeliveryStatus = "failed"
		run.DeliveryError = "delivery.channel must be one of: default, app, telegram"
		run.LatestStage = "delivery_failed"
		run.Summary = summarizeRun(*run)
		scheduler.appendRunEvent(ctx, buildRunEvent(*run, "delivery_failed", "failed", "", run.DeliveryError, channel, "scheduler", nil))
		return
	}
	if normalizePayloadKind(job.PayloadSpec.Kind) == "systemEvent" && scheduler.shouldFinalizeViaHeartbeat(job) {
		runStatus := strings.ToLower(strings.TrimSpace(run.Status))
		if runStatus != "completed" && runStatus != "running" {
			run.DeliveryStatus = "failed"
			run.DeliveryError = "cron run did not reach heartbeat execution"
			run.LatestStage = "delivery_failed"
			run.Summary = summarizeRun(*run)
			scheduler.appendRunEvent(ctx, buildRunEvent(*run, "delivery_failed", "failed", "", run.DeliveryError, channel, "scheduler", nil))
			return
		}
		scheduler.trackPendingRunDelivery(job, *run, channel)
		run.DeliveryStatus = "pending"
		run.DeliveryError = ""
		run.LatestStage = "delivery_pending"
		run.Summary = summarizeRun(*run)
		scheduler.appendRunEvent(ctx, buildRunEvent(*run, "delivery_pending", "pending", "", "", channel, "scheduler", map[string]any{
			"deliveryMode": "announce",
		}))
		return
	}
	sender := scheduler.announcementSender()
	if sender == nil {
		run.DeliveryStatus = "failed"
		run.DeliveryError = "announce sender unavailable"
		run.LatestStage = "delivery_failed"
		run.Summary = summarizeRun(*run)
		scheduler.appendRunEvent(ctx, buildRunEvent(*run, "delivery_failed", "failed", "", run.DeliveryError, channel, "scheduler", nil))
		return
	}
	message := resolveDeliveryMessage(job, *run, false)
	if message == "" {
		run.DeliveryStatus = "failed"
		run.DeliveryError = "delivery message is empty"
		run.LatestStage = "delivery_failed"
		run.Summary = summarizeRun(*run)
		scheduler.appendRunEvent(ctx, buildRunEvent(*run, "delivery_failed", "failed", "", run.DeliveryError, channel, "scheduler", nil))
		return
	}
	if err := sender(ctx, AnnouncementRequest{
		RunID:       strings.TrimSpace(run.RunID),
		JobID:       strings.TrimSpace(run.JobID),
		JobName:     strings.TrimSpace(run.JobName),
		AssistantID: strings.TrimSpace(job.AssistantID),
		SessionKey:  strings.TrimSpace(run.SessionKey),
		Channel:     channel,
		Message:     message,
	}); err != nil {
		run.DeliveryStatus = "failed"
		run.DeliveryError = strings.TrimSpace(err.Error())
		run.LatestStage = "delivery_failed"
		run.Summary = summarizeRun(*run)
		scheduler.appendRunEvent(ctx, buildRunEvent(*run, "delivery_failed", "failed", "", run.DeliveryError, channel, "scheduler", nil))
		return
	}
	run.DeliveryStatus = "delivered"
	run.DeliveryError = ""
	run.LatestStage = "delivery_delivered"
	if strings.TrimSpace(run.Summary) == "" {
		run.Summary = strings.TrimSpace(message)
	}
	scheduler.appendRunEvent(ctx, buildRunEvent(*run, "delivery_delivered", "ok", strings.TrimSpace(message), "", channel, "scheduler", map[string]any{
		"deliveryMode": "announce",
	}))
}

func (scheduler *Scheduler) applyWebhookDelivery(ctx context.Context, job CronJob, run *CronRunRecord, delivery *CronDelivery) {
	if scheduler == nil || run == nil || delivery == nil {
		return
	}
	target := strings.TrimSpace(delivery.To)
	lower := strings.ToLower(target)
	if target == "" || (!strings.HasPrefix(lower, "http://") && !strings.HasPrefix(lower, "https://")) {
		run.DeliveryStatus = "failed"
		run.DeliveryError = "delivery.to must start with http:// or https:// when delivery.mode=webhook"
		run.LatestStage = "delivery_failed"
		run.Summary = summarizeRun(*run)
		scheduler.appendRunEvent(ctx, buildRunEvent(*run, "delivery_failed", "failed", "", run.DeliveryError, "", "scheduler", nil))
		return
	}
	sender := scheduler.webhookSenderFunc()
	if sender == nil {
		run.DeliveryStatus = "failed"
		run.DeliveryError = "webhook sender unavailable"
		run.LatestStage = "delivery_failed"
		run.Summary = summarizeRun(*run)
		scheduler.appendRunEvent(ctx, buildRunEvent(*run, "delivery_failed", "failed", "", run.DeliveryError, "", "scheduler", nil))
		return
	}
	message := resolveDeliveryMessage(job, *run, false)
	payload := map[string]any{
		"runId":       strings.TrimSpace(run.RunID),
		"jobId":       strings.TrimSpace(run.JobID),
		"jobName":     strings.TrimSpace(run.JobName),
		"assistantId": strings.TrimSpace(job.AssistantID),
		"status":      strings.TrimSpace(run.Status),
		"summary":     strings.TrimSpace(run.Summary),
		"error":       strings.TrimSpace(run.Error),
		"message":     message,
		"sessionKey":  strings.TrimSpace(run.SessionKey),
		"model":       strings.TrimSpace(run.Model),
		"provider":    strings.TrimSpace(run.Provider),
		"usageJson":   strings.TrimSpace(run.UsageJSON),
	}
	if err := sender(ctx, WebhookRequest{
		RunID:       strings.TrimSpace(run.RunID),
		JobID:       strings.TrimSpace(run.JobID),
		JobName:     strings.TrimSpace(run.JobName),
		AssistantID: strings.TrimSpace(job.AssistantID),
		SessionKey:  strings.TrimSpace(run.SessionKey),
		URL:         target,
		Status:      strings.TrimSpace(run.Status),
		Summary:     strings.TrimSpace(run.Summary),
		Error:       strings.TrimSpace(run.Error),
		Payload:     payload,
	}); err != nil {
		run.DeliveryStatus = "failed"
		run.DeliveryError = strings.TrimSpace(err.Error())
		run.LatestStage = "delivery_failed"
		run.Summary = summarizeRun(*run)
		scheduler.appendRunEvent(ctx, buildRunEvent(*run, "delivery_failed", "failed", "", run.DeliveryError, "", "scheduler", nil))
		return
	}
	run.DeliveryStatus = "delivered"
	run.DeliveryError = ""
	run.LatestStage = "delivery_delivered"
	if strings.TrimSpace(run.Summary) == "" {
		run.Summary = strings.TrimSpace(message)
	}
	scheduler.appendRunEvent(ctx, buildRunEvent(*run, "delivery_delivered", "ok", strings.TrimSpace(message), "", "", "scheduler", map[string]any{
		"deliveryMode": "webhook",
	}))
}

func (scheduler *Scheduler) applyFailureDestination(ctx context.Context, job CronJob, run *CronRunRecord, delivery *CronDelivery) {
	if scheduler == nil || run == nil || delivery == nil || delivery.FailureDestination == nil {
		return
	}
	if delivery.BestEffort {
		return
	}
	target := delivery.FailureDestination
	mode := strings.ToLower(strings.TrimSpace(target.Mode))
	if mode == "" {
		mode = "announce"
	}
	message := resolveDeliveryMessage(job, *run, true)
	if message == "" {
		message = "cron run failed"
	}
	switch mode {
	case "announce":
		channel := normalizeAnnounceChannel(target.Channel)
		if !isValidAnnounceChannel(channel) {
			scheduler.appendRunEvent(ctx, buildRunEvent(*run, "failure_destination_failed", "failed", "", "delivery.failureDestination.channel must be one of: default, app, telegram", channel, "scheduler", nil))
			if strings.TrimSpace(run.DeliveryStatus) == "" {
				run.DeliveryStatus = "failed"
				run.DeliveryError = "invalid failure destination announce channel"
			}
			return
		}
		sender := scheduler.announcementSender()
		if sender == nil {
			scheduler.appendRunEvent(ctx, buildRunEvent(*run, "failure_destination_failed", "failed", "", "announce sender unavailable", channel, "scheduler", nil))
			if strings.TrimSpace(run.DeliveryStatus) == "" {
				run.DeliveryStatus = "failed"
				run.DeliveryError = "announce sender unavailable"
			}
			return
		}
		if err := sender(ctx, AnnouncementRequest{
			RunID:       strings.TrimSpace(run.RunID),
			JobID:       strings.TrimSpace(run.JobID),
			JobName:     strings.TrimSpace(run.JobName),
			AssistantID: strings.TrimSpace(job.AssistantID),
			SessionKey:  strings.TrimSpace(run.SessionKey),
			Channel:     channel,
			Message:     message,
		}); err != nil {
			scheduler.appendRunEvent(ctx, buildRunEvent(*run, "failure_destination_failed", "failed", "", strings.TrimSpace(err.Error()), channel, "scheduler", nil))
			if strings.TrimSpace(run.DeliveryStatus) == "" {
				run.DeliveryStatus = "failed"
				run.DeliveryError = strings.TrimSpace(err.Error())
			}
			return
		}
		scheduler.appendRunEvent(ctx, buildRunEvent(*run, "failure_destination_delivered", "ok", message, "", channel, "scheduler", map[string]any{
			"mode": "announce",
		}))
		if strings.TrimSpace(run.DeliveryStatus) == "" {
			run.DeliveryStatus = "delivered"
			run.DeliveryError = ""
		}
	case "webhook":
		targetURL := strings.TrimSpace(target.To)
		lower := strings.ToLower(targetURL)
		if targetURL == "" || (!strings.HasPrefix(lower, "http://") && !strings.HasPrefix(lower, "https://")) {
			scheduler.appendRunEvent(ctx, buildRunEvent(*run, "failure_destination_failed", "failed", "", "delivery.failureDestination.to must start with http:// or https://", "", "scheduler", nil))
			if strings.TrimSpace(run.DeliveryStatus) == "" {
				run.DeliveryStatus = "failed"
				run.DeliveryError = "invalid failure destination webhook url"
			}
			return
		}
		sender := scheduler.webhookSenderFunc()
		if sender == nil {
			scheduler.appendRunEvent(ctx, buildRunEvent(*run, "failure_destination_failed", "failed", "", "webhook sender unavailable", "", "scheduler", nil))
			if strings.TrimSpace(run.DeliveryStatus) == "" {
				run.DeliveryStatus = "failed"
				run.DeliveryError = "webhook sender unavailable"
			}
			return
		}
		payload := map[string]any{
			"kind":        "failureDestination",
			"runId":       strings.TrimSpace(run.RunID),
			"jobId":       strings.TrimSpace(run.JobID),
			"jobName":     strings.TrimSpace(run.JobName),
			"assistantId": strings.TrimSpace(job.AssistantID),
			"status":      strings.TrimSpace(run.Status),
			"summary":     strings.TrimSpace(run.Summary),
			"error":       strings.TrimSpace(run.Error),
			"message":     message,
			"sessionKey":  strings.TrimSpace(run.SessionKey),
		}
		if err := sender(ctx, WebhookRequest{
			RunID:       strings.TrimSpace(run.RunID),
			JobID:       strings.TrimSpace(run.JobID),
			JobName:     strings.TrimSpace(run.JobName),
			AssistantID: strings.TrimSpace(job.AssistantID),
			SessionKey:  strings.TrimSpace(run.SessionKey),
			URL:         targetURL,
			Status:      strings.TrimSpace(run.Status),
			Summary:     strings.TrimSpace(run.Summary),
			Error:       strings.TrimSpace(run.Error),
			Payload:     payload,
		}); err != nil {
			scheduler.appendRunEvent(ctx, buildRunEvent(*run, "failure_destination_failed", "failed", "", strings.TrimSpace(err.Error()), "", "scheduler", nil))
			if strings.TrimSpace(run.DeliveryStatus) == "" {
				run.DeliveryStatus = "failed"
				run.DeliveryError = strings.TrimSpace(err.Error())
			}
			return
		}
		scheduler.appendRunEvent(ctx, buildRunEvent(*run, "failure_destination_delivered", "ok", message, "", "", "scheduler", map[string]any{
			"mode": "webhook",
		}))
		if strings.TrimSpace(run.DeliveryStatus) == "" {
			run.DeliveryStatus = "delivered"
			run.DeliveryError = ""
		}
	default:
		scheduler.appendRunEvent(ctx, buildRunEvent(*run, "failure_destination_failed", "failed", "", "delivery.failureDestination.mode must be one of: announce, webhook", "", "scheduler", nil))
		if strings.TrimSpace(run.DeliveryStatus) == "" {
			run.DeliveryStatus = "failed"
			run.DeliveryError = "invalid failure destination mode"
		}
	}
	run.Summary = summarizeRun(*run)
}

func resolveDeliveryMessage(job CronJob, run CronRunRecord, preferError bool) string {
	if preferError {
		if text := strings.TrimSpace(run.Error); text != "" {
			return text
		}
	}
	if text := strings.TrimSpace(run.Summary); text != "" {
		return text
	}
	if text := strings.TrimSpace(run.Error); text != "" {
		return text
	}
	if normalizePayloadKind(job.PayloadSpec.Kind) == "systemEvent" {
		return strings.TrimSpace(job.PayloadSpec.Text)
	}
	if text := strings.TrimSpace(job.PayloadSpec.Message); text != "" {
		return text
	}
	return ""
}

func (scheduler *Scheduler) shouldFinalizeViaHeartbeat(job CronJob) bool {
	if normalizePayloadKind(job.PayloadSpec.Kind) != "systemEvent" {
		return false
	}
	delivery := normalizeDelivery(job.Delivery)
	if delivery == nil || strings.ToLower(strings.TrimSpace(delivery.Mode)) != "announce" {
		return false
	}
	return scheduler.mainSystemEventEnqueuer() != nil
}

func (scheduler *Scheduler) announcementSender() AnnouncementSender {
	if scheduler == nil {
		return nil
	}
	scheduler.mu.RLock()
	defer scheduler.mu.RUnlock()
	return scheduler.announceSender
}

func (scheduler *Scheduler) webhookSenderFunc() WebhookSender {
	if scheduler == nil {
		return nil
	}
	scheduler.mu.RLock()
	defer scheduler.mu.RUnlock()
	return scheduler.webhookSender
}

func (scheduler *Scheduler) mainSystemEventEnqueuer() MainSystemEventEnqueuer {
	if scheduler == nil {
		return nil
	}
	scheduler.mu.RLock()
	defer scheduler.mu.RUnlock()
	return scheduler.mainSystemEventEnqueue
}

func (scheduler *Scheduler) wakeTriggerFunc() WakeTrigger {
	if scheduler == nil {
		return nil
	}
	scheduler.mu.RLock()
	defer scheduler.mu.RUnlock()
	return scheduler.wakeTrigger
}

func (scheduler *Scheduler) isolatedExecutorFunc() IsolatedExecutor {
	if scheduler == nil {
		return nil
	}
	scheduler.mu.RLock()
	defer scheduler.mu.RUnlock()
	return scheduler.isolatedExecutor
}

func (scheduler *Scheduler) runRealtimeNotifierFunc() RunRealtimeNotifier {
	if scheduler == nil {
		return nil
	}
	scheduler.mu.RLock()
	defer scheduler.mu.RUnlock()
	return scheduler.runRealtimeNotifier
}

func (scheduler *Scheduler) triggerMainSystemEvent(ctx context.Context, job CronJob, runID string) error {
	enqueuer := scheduler.mainSystemEventEnqueuer()
	if enqueuer == nil {
		return errors.New("main system event enqueuer unavailable")
	}
	eventText := strings.TrimSpace(job.PayloadSpec.Text)
	if eventText == "" {
		return errors.New("payload.text is required when payload.kind=systemEvent")
	}
	sessionKey := normalizeMainSessionKeyForHeartbeat(strings.TrimSpace(job.SessionKey))
	contextKey := "cron:" + strings.TrimSpace(job.JobID)
	if strings.TrimSpace(job.JobID) == "" {
		contextKey = "cron:event"
	}
	resolvedRunID := strings.TrimSpace(runID)
	if resolvedRunID == "" {
		resolvedRunID = uuid.NewString()
	}
	queued := enqueuer(ctx, MainSystemEventRequest{
		SessionKey: sessionKey,
		Text:       eventText,
		ContextKey: contextKey,
		RunID:      resolvedRunID,
	})
	if !queued {
		return errors.New("failed to enqueue cron system event")
	}
	if trigger := scheduler.wakeTriggerFunc(); trigger != nil {
		wakeResult := trigger(ctx, buildWakeTriggerRequest(job.WakeMode, sessionKey, "cron", resolvedRunID))
		if !wakeResult.Accepted {
			if reason := strings.TrimSpace(wakeResult.Reason); reason != "" {
				return fmt.Errorf("heartbeat wake rejected: %s", reason)
			}
			return errors.New("heartbeat wake rejected")
		}
	}
	return nil
}

func buildWakeTriggerRequest(wakeMode string, sessionKey string, source string, runID string) WakeTriggerRequest {
	normalizedMode := strings.ToLower(strings.TrimSpace(wakeMode))
	force := false
	reason := "cron.wake.next-heartbeat"
	if normalizedMode == "now" {
		force = true
		reason = "cron.wake.now"
	}
	return WakeTriggerRequest{
		Reason:     reason,
		SessionKey: strings.TrimSpace(sessionKey),
		Force:      force,
		Source:     strings.TrimSpace(source),
		RunID:      strings.TrimSpace(runID),
	}
}

func normalizeMainSessionKeyForHeartbeat(sessionKey string) string {
	trimmed := strings.TrimSpace(sessionKey)
	if trimmed == "" {
		return ""
	}
	if strings.EqualFold(trimmed, "cron/main") {
		return ""
	}
	return trimmed
}

func isSyntheticCronSessionKey(sessionKey string) bool {
	switch strings.ToLower(strings.TrimSpace(sessionKey)) {
	case "cron/main", "cron/isolated", "cron/default":
		return true
	default:
		return false
	}
}

func DecodeHeartbeatDeliveryEvent(payload []byte) (HeartbeatDeliveryEvent, bool) {
	if len(payload) == 0 {
		return HeartbeatDeliveryEvent{}, false
	}
	var envelope struct {
		Status     string `json:"status"`
		Message    string `json:"message"`
		Error      string `json:"error"`
		SessionKey string `json:"sessionKey"`
		Context    struct {
			Source string `json:"source"`
			RunID  string `json:"runId"`
		} `json:"context"`
	}
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return HeartbeatDeliveryEvent{}, false
	}
	event := HeartbeatDeliveryEvent{
		RunID:      strings.TrimSpace(envelope.Context.RunID),
		Source:     strings.TrimSpace(envelope.Context.Source),
		Status:     strings.TrimSpace(envelope.Status),
		Message:    strings.TrimSpace(envelope.Message),
		Error:      strings.TrimSpace(envelope.Error),
		SessionKey: strings.TrimSpace(envelope.SessionKey),
	}
	if event.RunID == "" {
		return HeartbeatDeliveryEvent{}, false
	}
	return event, true
}

func (scheduler *Scheduler) HandleHeartbeatDeliveryEvent(ctx context.Context, event HeartbeatDeliveryEvent) {
	if scheduler == nil {
		return
	}
	if !strings.EqualFold(strings.TrimSpace(event.Source), "cron") {
		return
	}
	runID := strings.TrimSpace(event.RunID)
	if runID == "" {
		return
	}
	scheduler.mu.Lock()
	pending, ok := scheduler.pendingDeliveries[runID]
	if ok {
		delete(scheduler.pendingDeliveries, runID)
	}
	scheduler.mu.Unlock()
	if !ok {
		return
	}

	run := CronRunRecord{
		RunID:      pending.RunID,
		JobID:      pending.JobID,
		JobName:    pending.JobName,
		Status:     pending.Status,
		Model:      pending.Model,
		Provider:   pending.Provider,
		UsageJSON:  pending.UsageJSON,
		StartedAt:  pending.StartedAt,
		EndedAt:    pending.EndedAt,
		SessionKey: pending.SessionKey,
	}
	if (strings.TrimSpace(run.Model) == "" || strings.TrimSpace(run.Provider) == "" || strings.TrimSpace(run.UsageJSON) == "") && scheduler.store != nil {
		if persisted, ok, err := scheduler.store.GetCronRun(ctx, runID); err == nil && ok {
			if strings.TrimSpace(run.Model) == "" {
				run.Model = strings.TrimSpace(persisted.Model)
			}
			if strings.TrimSpace(run.Provider) == "" {
				run.Provider = strings.TrimSpace(persisted.Provider)
			}
			if strings.TrimSpace(run.UsageJSON) == "" {
				run.UsageJSON = strings.TrimSpace(persisted.UsageJSON)
			}
		}
	}
	status := strings.ToLower(strings.TrimSpace(event.Status))
	message := strings.TrimSpace(event.Message)
	if status == "failed" && message == "" {
		message = strings.TrimSpace(event.Error)
	}
	scheduler.appendRunEvent(ctx, buildRunEvent(run, "heartbeat_received", status, message, strings.TrimSpace(event.Error), strings.TrimSpace(pending.Channel), "heartbeat", map[string]any{
		"source": strings.TrimSpace(event.Source),
	}))
	switch status {
	case "sent":
		run.Status = "completed"
		run.Error = ""
		run.Summary = message
	case "failed":
		run.Status = "failed"
		run.Error = message
		run.Summary = strings.TrimSpace(message)
	default:
		run.Status = "failed"
		run.Error = "heartbeat status is not deliverable: " + strings.TrimSpace(event.Status)
		run.DeliveryStatus = "failed"
		run.DeliveryError = run.Error
		run.Summary = summarizeRun(run)
		run.LatestStage = "failed"
		if run.EndedAt.IsZero() {
			run.EndedAt = scheduler.now()
		}
		scheduler.appendRunEvent(ctx, buildRunEvent(run, "failed", "failed", "", run.Error, strings.TrimSpace(pending.Channel), "scheduler", nil))
		scheduler.persistDeliveryResult(ctx, run)
		scheduler.updateRunState(ctx, run)
		return
	}
	if run.EndedAt.IsZero() {
		run.EndedAt = scheduler.now()
	}
	resolvedSessionKey := strings.TrimSpace(pending.SessionKey)
	if isSyntheticCronSessionKey(resolvedSessionKey) {
		resolvedSessionKey = ""
	}
	if resolvedSessionKey == "" {
		resolvedSessionKey = strings.TrimSpace(event.SessionKey)
	}
	run.SessionKey = resolvedSessionKey
	if message == "" {
		run.DeliveryStatus = "failed"
		run.DeliveryError = "heartbeat result is empty"
		run.Summary = summarizeRun(run)
		run.LatestStage = "delivery_failed"
		scheduler.appendRunEvent(ctx, buildRunEvent(run, "delivery_failed", "failed", "", run.DeliveryError, strings.TrimSpace(pending.Channel), "scheduler", nil))
		scheduler.persistDeliveryResult(ctx, run)
		scheduler.updateRunState(ctx, run)
		return
	}

	sender := scheduler.announcementSender()
	if sender == nil {
		run.DeliveryStatus = "failed"
		run.DeliveryError = "announce sender unavailable"
		run.Summary = summarizeRun(run)
		run.LatestStage = "delivery_failed"
		scheduler.appendRunEvent(ctx, buildRunEvent(run, "delivery_failed", "failed", "", run.DeliveryError, strings.TrimSpace(pending.Channel), "scheduler", nil))
		scheduler.persistDeliveryResult(ctx, run)
		scheduler.updateRunState(ctx, run)
		return
	}
	run.LatestStage = "delivery_attempted"
	scheduler.appendRunEvent(ctx, buildRunEvent(run, "delivery_attempted", "running", message, "", strings.TrimSpace(pending.Channel), "scheduler", nil))
	err := sender(ctx, AnnouncementRequest{
		RunID:       pending.RunID,
		JobID:       pending.JobID,
		JobName:     pending.JobName,
		AssistantID: pending.AssistantID,
		SessionKey:  resolvedSessionKey,
		Channel:     strings.TrimSpace(pending.Channel),
		Message:     message,
	})
	if err != nil {
		run.DeliveryStatus = "failed"
		run.DeliveryError = strings.TrimSpace(err.Error())
		run.Summary = summarizeRun(run)
		run.LatestStage = "delivery_failed"
		scheduler.appendRunEvent(ctx, buildRunEvent(run, "delivery_failed", "failed", "", run.DeliveryError, strings.TrimSpace(pending.Channel), "scheduler", nil))
		scheduler.persistDeliveryResult(ctx, run)
		scheduler.updateRunState(ctx, run)
		return
	}
	run.DeliveryStatus = "delivered"
	run.DeliveryError = ""
	if strings.TrimSpace(run.Summary) == "" {
		run.Summary = strings.TrimSpace(message)
	}
	run.LatestStage = "delivery_delivered"
	scheduler.appendRunEvent(ctx, buildRunEvent(run, "delivery_delivered", "ok", strings.TrimSpace(message), "", strings.TrimSpace(pending.Channel), "scheduler", nil))
	if strings.EqualFold(strings.TrimSpace(run.Status), "completed") {
		scheduler.appendRunEvent(ctx, buildRunEvent(run, "completed", "completed", run.Summary, "", strings.TrimSpace(pending.Channel), "scheduler", nil))
	} else {
		scheduler.appendRunEvent(ctx, buildRunEvent(run, "failed", "failed", "", strings.TrimSpace(run.Error), strings.TrimSpace(pending.Channel), "scheduler", nil))
	}
	scheduler.persistDeliveryResult(ctx, run)
	scheduler.updateRunState(ctx, run)
}

func (scheduler *Scheduler) trackPendingRunDelivery(job CronJob, run CronRunRecord, channel string) {
	runID := strings.TrimSpace(run.RunID)
	if runID == "" {
		return
	}
	entry := pendingRunDelivery{
		RunID:       runID,
		JobID:       strings.TrimSpace(run.JobID),
		JobName:     strings.TrimSpace(run.JobName),
		AssistantID: strings.TrimSpace(job.AssistantID),
		Status:      strings.TrimSpace(run.Status),
		Model:       strings.TrimSpace(run.Model),
		Provider:    strings.TrimSpace(run.Provider),
		UsageJSON:   strings.TrimSpace(run.UsageJSON),
		StartedAt:   run.StartedAt,
		EndedAt:     run.EndedAt,
		PendingAt:   scheduler.now(),
		SessionKey:  strings.TrimSpace(run.SessionKey),
		Channel:     normalizeAnnounceChannel(channel),
	}
	scheduler.mu.Lock()
	scheduler.pendingDeliveries[runID] = entry
	scheduler.mu.Unlock()
}

func (scheduler *Scheduler) finalizeExpiredPendingDeliveries(ctx context.Context, now time.Time) {
	if scheduler == nil {
		return
	}
	expired := make([]pendingRunDelivery, 0)
	scheduler.mu.Lock()
	for runID, pending := range scheduler.pendingDeliveries {
		pendingAt := pending.PendingAt
		if pendingAt.IsZero() {
			pendingAt = pending.StartedAt
		}
		if pendingAt.IsZero() || now.Sub(pendingAt) < defaultPendingDeliveryTimeout {
			continue
		}
		expired = append(expired, pending)
		delete(scheduler.pendingDeliveries, runID)
	}
	scheduler.mu.Unlock()
	for _, pending := range expired {
		scheduler.markPendingDeliveryTimedOut(ctx, pending, now)
	}
}

func (scheduler *Scheduler) markPendingDeliveryTimedOut(ctx context.Context, pending pendingRunDelivery, now time.Time) {
	if scheduler == nil {
		return
	}
	timeoutErr := "delivery timeout waiting for heartbeat result"
	run := CronRunRecord{
		RunID:          strings.TrimSpace(pending.RunID),
		JobID:          strings.TrimSpace(pending.JobID),
		JobName:        strings.TrimSpace(pending.JobName),
		Status:         "failed",
		Model:          strings.TrimSpace(pending.Model),
		Provider:       strings.TrimSpace(pending.Provider),
		UsageJSON:      strings.TrimSpace(pending.UsageJSON),
		Error:          timeoutErr,
		Summary:        timeoutErr,
		DeliveryStatus: "failed",
		DeliveryError:  timeoutErr,
		StartedAt:      pending.StartedAt,
		EndedAt:        now,
		SessionKey:     strings.TrimSpace(pending.SessionKey),
		LatestStage:    "delivery_failed",
	}
	scheduler.appendRunEvent(ctx, buildRunEvent(run, "heartbeat_timeout", "failed", "", timeoutErr, strings.TrimSpace(pending.Channel), "scheduler", map[string]any{
		"timeoutMs": defaultPendingDeliveryTimeout.Milliseconds(),
	}))
	scheduler.appendRunEvent(ctx, buildRunEvent(run, "delivery_failed", "failed", "", timeoutErr, strings.TrimSpace(pending.Channel), "scheduler", nil))
	scheduler.appendRunEvent(ctx, buildRunEvent(run, "failed", "failed", "", timeoutErr, strings.TrimSpace(pending.Channel), "scheduler", nil))
	scheduler.persistDeliveryResult(ctx, run)
	scheduler.updateRunState(ctx, run)
}

func (scheduler *Scheduler) appendRunEvent(ctx context.Context, event CronRunEvent) {
	if scheduler == nil || scheduler.store == nil {
		return
	}
	stage := strings.TrimSpace(event.Stage)
	if stage == "" {
		return
	}
	if strings.TrimSpace(event.EventID) == "" {
		event.EventID = uuid.NewString()
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = scheduler.now()
	}
	if err := scheduler.store.SaveCronRunEvent(ctx, event); err != nil {
		return
	}
	scheduler.emitRunRealtimeEvent(ctx, RunRealtimeEvent{
		RunID:     strings.TrimSpace(event.RunID),
		JobID:     strings.TrimSpace(event.JobID),
		Stage:     strings.TrimSpace(event.Stage),
		Status:    strings.TrimSpace(event.Status),
		Source:    strings.TrimSpace(event.Source),
		CreatedAt: event.CreatedAt,
	})
}

func (scheduler *Scheduler) emitRunRealtimeEvent(ctx context.Context, event RunRealtimeEvent) {
	if scheduler == nil {
		return
	}
	notifier := scheduler.runRealtimeNotifierFunc()
	if notifier == nil {
		return
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = scheduler.now()
	}
	notifier(ctx, event)
}

func buildRunEvent(run CronRunRecord, stage string, status string, message string, errText string, channel string, source string, meta map[string]any) CronRunEvent {
	resolvedStatus := strings.TrimSpace(status)
	if resolvedStatus == "" {
		resolvedStatus = strings.TrimSpace(run.Status)
	}
	return CronRunEvent{
		EventID:    uuid.NewString(),
		RunID:      strings.TrimSpace(run.RunID),
		JobID:      strings.TrimSpace(run.JobID),
		JobName:    strings.TrimSpace(run.JobName),
		Stage:      strings.TrimSpace(stage),
		Status:     resolvedStatus,
		Message:    strings.TrimSpace(message),
		Error:      strings.TrimSpace(errText),
		Channel:    strings.TrimSpace(channel),
		SessionKey: strings.TrimSpace(run.SessionKey),
		Source:     strings.TrimSpace(source),
		Meta:       meta,
	}
}

func summarizeRun(run CronRunRecord) string {
	if text := strings.TrimSpace(run.Summary); text != "" {
		return text
	}
	if text := strings.TrimSpace(run.DeliveryError); text != "" {
		return text
	}
	if text := strings.TrimSpace(run.Error); text != "" {
		return text
	}
	if strings.EqualFold(strings.TrimSpace(run.DeliveryStatus), "pending") {
		return "delivery pending"
	}
	status := strings.TrimSpace(run.Status)
	if status == "" {
		return ""
	}
	return "status: " + status
}

func (scheduler *Scheduler) persistDeliveryResult(ctx context.Context, run CronRunRecord) {
	if scheduler == nil {
		return
	}
	run.Summary = summarizeRun(run)
	if strings.TrimSpace(run.LatestStage) == "" {
		if strings.TrimSpace(run.DeliveryStatus) == "delivered" {
			run.LatestStage = "delivery_delivered"
		} else if strings.TrimSpace(run.DeliveryStatus) == "failed" {
			run.LatestStage = "delivery_failed"
		}
	}
	if scheduler.store != nil {
		_ = scheduler.store.SaveCronRun(ctx, run)
	}
	jobID := strings.TrimSpace(run.JobID)
	if jobID == "" {
		return
	}
	scheduler.mu.Lock()
	job, ok := scheduler.jobs[jobID]
	if ok {
		job.State.LastDeliveryStatus = strings.TrimSpace(run.DeliveryStatus)
		job.State.LastDeliveryError = strings.TrimSpace(run.DeliveryError)
		delivered := strings.EqualFold(job.State.LastDeliveryStatus, "delivered") || strings.EqualFold(job.State.LastDeliveryStatus, "ok")
		job.State.LastDelivered = boolPtr(delivered)
		job.UpdatedAt = scheduler.now()
		scheduler.jobs[jobID] = job
	}
	scheduler.mu.Unlock()
	if ok && scheduler.store != nil {
		_ = scheduler.store.SaveCronJob(ctx, job)
	}
}

func (scheduler *Scheduler) markJobRunning(ctx context.Context, jobID string, startedAt time.Time) {
	if scheduler == nil {
		return
	}
	scheduler.mu.Lock()
	job, ok := scheduler.jobs[jobID]
	if ok {
		job.State.RunningAtMs = startedAt.UnixMilli()
		job.UpdatedAt = scheduler.now()
		scheduler.jobs[jobID] = job
	}
	scheduler.mu.Unlock()
	if ok && scheduler.store != nil {
		_ = scheduler.store.SaveCronJob(ctx, job)
	}
	if ok {
		scheduler.emitRunRealtimeEvent(ctx, RunRealtimeEvent{
			JobID:     strings.TrimSpace(jobID),
			Stage:     "job_running",
			Status:    "running",
			Source:    "scheduler",
			CreatedAt: startedAt,
		})
	}
}

func (scheduler *Scheduler) updateRunState(ctx context.Context, run CronRunRecord) {
	jobID := strings.TrimSpace(run.JobID)
	if jobID == "" {
		return
	}
	scheduler.mu.Lock()
	job, ok := scheduler.jobs[jobID]
	if ok {
		lastRunAt := run.EndedAt
		if lastRunAt.IsZero() {
			lastRunAt = run.StartedAt
		}
		job.State.LastRunAtMs = lastRunAt.UnixMilli()
		job.State.RunningAtMs = 0
		job.State.LastRunStatus = normalizeRunStatus(run.Status)
		job.State.LastDurationMs = maxDurationMs(run.EndedAt.Sub(run.StartedAt))
		job.State.LastError = strings.TrimSpace(run.Error)
		if deliveryStatus := strings.TrimSpace(run.DeliveryStatus); deliveryStatus != "" {
			job.State.LastDeliveryStatus = deliveryStatus
			job.State.LastDeliveryError = strings.TrimSpace(run.DeliveryError)
			delivered := strings.EqualFold(deliveryStatus, "delivered") || strings.EqualFold(deliveryStatus, "ok")
			job.State.LastDelivered = boolPtr(delivered)
		}
		if job.State.LastRunStatus == "error" {
			job.State.ConsecutiveErrors++
		} else if job.State.LastRunStatus == "ok" {
			job.State.ConsecutiveErrors = 0
		}
		job.UpdatedAt = scheduler.now()
		scheduler.jobs[jobID] = job
	}
	scheduler.mu.Unlock()
	if ok && scheduler.store != nil {
		_ = scheduler.store.SaveCronJob(ctx, job)
	}
	if ok {
		scheduler.emitRunRealtimeEvent(ctx, RunRealtimeEvent{
			RunID:          strings.TrimSpace(run.RunID),
			JobID:          strings.TrimSpace(run.JobID),
			Stage:          "job_state_updated",
			Status:         normalizeRunStatus(run.Status),
			DeliveryStatus: strings.TrimSpace(run.DeliveryStatus),
			Source:         "scheduler",
			CreatedAt:      scheduler.now(),
		})
	}
}

func (scheduler *Scheduler) scheduleAfterSuccess(ctx context.Context, job CronJob) {
	if job.DeleteAfterRun {
		_ = scheduler.Delete(ctx, job.JobID)
		return
	}
	switch normalizeScheduleType(job) {
	case "at":
		scheduler.mu.Lock()
		current, ok := scheduler.jobs[job.JobID]
		if ok {
			current.Enabled = false
			current.UpdatedAt = scheduler.now()
			next := time.Time{}
			current = syncNextRunState(current, next)
			scheduler.jobs[job.JobID] = current
			scheduler.nextRuns[job.JobID] = next
			job = current
		}
		scheduler.mu.Unlock()
		if ok && scheduler.store != nil {
			_ = scheduler.store.SaveCronJob(ctx, job)
		}
		if ok {
			scheduler.emitRunRealtimeEvent(ctx, RunRealtimeEvent{
				JobID:     strings.TrimSpace(job.JobID),
				Stage:     "job_rescheduled",
				Status:    "completed",
				Source:    "scheduler",
				CreatedAt: scheduler.now(),
			})
		}
	default:
		scheduler.scheduleNext(job, scheduler.now())
		scheduler.emitRunRealtimeEvent(ctx, RunRealtimeEvent{
			JobID:     strings.TrimSpace(job.JobID),
			Stage:     "job_rescheduled",
			Status:    "completed",
			Source:    "scheduler",
			CreatedAt: scheduler.now(),
		})
	}
}

func (scheduler *Scheduler) scheduleNext(job CronJob, base time.Time) {
	if scheduler == nil {
		return
	}
	next := time.Time{}
	if job.Enabled {
		if calculated, err := nextRunForJob(job, base); err == nil {
			next = calculated
		}
	}
	scheduler.mu.Lock()
	current, ok := scheduler.jobs[job.JobID]
	if ok {
		current = syncNextRunState(current, next)
		scheduler.jobs[job.JobID] = current
	}
	scheduler.nextRuns[job.JobID] = next
	scheduler.mu.Unlock()
}

func (scheduler *Scheduler) scheduleRetry(job CronJob) {
	if scheduler == nil {
		return
	}
	persist := false
	var snapshot CronJob
	scheduler.mu.Lock()
	current, ok := scheduler.jobs[job.JobID]
	if !ok || !current.Enabled {
		scheduler.mu.Unlock()
		return
	}
	count := scheduler.retries[job.JobID]
	if count < 0 {
		count = 0
	}
	if count >= 5 {
		scheduler.mu.Unlock()
		return
	}
	count++
	scheduler.retries[job.JobID] = count
	delay := time.Duration(1<<count) * time.Minute
	next := scheduler.now().Add(delay)
	current = syncNextRunState(current, next)
	scheduler.jobs[job.JobID] = current
	scheduler.nextRuns[job.JobID] = next
	persist = true
	snapshot = current
	scheduler.mu.Unlock()
	if persist && scheduler.store != nil {
		_ = scheduler.store.SaveCronJob(context.Background(), snapshot)
	}
}

func (scheduler *Scheduler) resetRetry(jobID string) {
	scheduler.mu.Lock()
	delete(scheduler.retries, jobID)
	scheduler.mu.Unlock()
}

func (scheduler *Scheduler) lookup(jobID string) (CronJob, bool) {
	scheduler.mu.RLock()
	defer scheduler.mu.RUnlock()
	job, ok := scheduler.jobs[jobID]
	return job, ok
}

func (scheduler *Scheduler) snapshot() []CronJob {
	scheduler.mu.RLock()
	defer scheduler.mu.RUnlock()
	result := make([]CronJob, 0, len(scheduler.jobs))
	for _, job := range scheduler.jobs {
		result = append(result, job)
	}
	return result
}

func (scheduler *Scheduler) nextRun(jobID string) time.Time {
	scheduler.mu.RLock()
	defer scheduler.mu.RUnlock()
	return scheduler.nextRuns[jobID]
}

func normalizeJob(job CronJob) CronJob {
	job.ID = strings.TrimSpace(job.ID)
	job.JobID = strings.TrimSpace(job.JobID)
	if job.JobID == "" {
		job.JobID = job.ID
	}
	if job.JobID == "" {
		job.JobID = uuid.NewString()
	}
	job.ID = job.JobID
	job.Name = strings.TrimSpace(job.Name)
	if job.Name == "" {
		job.Name = job.JobID
	}
	job.Description = strings.TrimSpace(job.Description)
	job.AssistantID = strings.TrimSpace(job.AssistantID)
	job.SessionTarget = normalizeSessionTarget(job.SessionTarget)
	job.WakeMode = normalizeWakeMode(job.WakeMode)
	if job.WakeMode == "" {
		job.WakeMode = "next-heartbeat"
	}
	job.Schedule = normalizeSchedule(job.Schedule)
	job.PayloadSpec = normalizePayload(job.PayloadSpec)
	if job.SessionTarget == "" {
		if strings.EqualFold(job.PayloadSpec.Kind, "agentTurn") {
			job.SessionTarget = "isolated"
		} else {
			job.SessionTarget = "main"
		}
	}
	if job.Delivery != nil {
		job.Delivery = normalizeDelivery(job.Delivery)
	}
	job.SessionKey = strings.TrimSpace(job.SessionKey)
	if job.SessionKey == "" {
		if strings.EqualFold(strings.TrimSpace(job.SessionTarget), "main") {
			job.SessionKey = "cron/main"
		} else {
			job.SessionKey = "cron/isolated"
		}
	}
	return job
}

func normalizeScheduleType(job CronJob) string {
	return strings.ToLower(strings.TrimSpace(job.Schedule.Kind))
}

func validateJobSchedule(job CronJob, now time.Time) error {
	switch normalizeScheduleType(job) {
	case "cron":
		expr := strings.TrimSpace(job.Schedule.Expr)
		if expr == "" {
			return errors.New("cron expression is required")
		}
		_, err := nextRunForCron(expr, job.Schedule.TZ, now)
		return err
	case "every":
		_, err := parseEveryDurationForJob(job)
		return err
	case "at":
		atTime, err := parseAtTime(job.Schedule.At, job.Schedule.TZ)
		if err != nil {
			return err
		}
		if atTime.Before(now.Add(-1 * time.Minute)) {
			return errors.New("at schedule is in the past")
		}
		return nil
	default:
		return fmt.Errorf("unsupported schedule type: %s", strings.TrimSpace(job.Schedule.Kind))
	}
}

func validateJobSemantics(job CronJob) error {
	target := strings.ToLower(strings.TrimSpace(job.SessionTarget))
	payloadKind := strings.ToLower(strings.TrimSpace(job.PayloadSpec.Kind))
	switch target {
	case "main":
		if payloadKind != "systemevent" {
			return errors.New("sessionTarget=main requires payload.kind=systemEvent (hint: use payload.text with systemEvent)")
		}
	case "isolated":
		if payloadKind != "agentturn" {
			return errors.New("sessionTarget=isolated requires payload.kind=agentTurn (hint: use payload.message with agentTurn)")
		}
	default:
		return errors.New("sessionTarget must be one of: main, isolated")
	}
	if job.Delivery == nil {
		return nil
	}
	mode := strings.ToLower(strings.TrimSpace(job.Delivery.Mode))
	switch mode {
	case "none":
		// no-op
	case "announce":
		channel := normalizeAnnounceChannel(job.Delivery.Channel)
		if !isValidAnnounceChannel(channel) {
			return errors.New("delivery.channel must be one of: default, app, telegram when delivery.mode=announce")
		}
	case "webhook":
		targetURL := strings.ToLower(strings.TrimSpace(job.Delivery.To))
		if targetURL == "" {
			return errors.New("delivery.to is required when delivery.mode=webhook")
		}
		if !strings.HasPrefix(targetURL, "http://") && !strings.HasPrefix(targetURL, "https://") {
			return errors.New("delivery.to must start with http:// or https:// when delivery.mode=webhook")
		}
	default:
		return errors.New("delivery.mode must be one of: none, announce, webhook")
	}
	if err := validateFailureDestinationSemantics(job.Delivery.FailureDestination); err != nil {
		return err
	}
	return nil
}

func validateFailureDestinationSemantics(destination *CronFailureDestination) error {
	if destination == nil {
		return nil
	}
	mode := strings.ToLower(strings.TrimSpace(destination.Mode))
	if mode == "" {
		mode = "announce"
	}
	switch mode {
	case "announce":
		channel := normalizeAnnounceChannel(destination.Channel)
		if !isValidAnnounceChannel(channel) {
			return errors.New("delivery.failureDestination.channel must be one of: default, app, telegram when delivery.failureDestination.mode=announce")
		}
		return nil
	case "webhook":
		targetURL := strings.ToLower(strings.TrimSpace(destination.To))
		if targetURL == "" {
			return errors.New("delivery.failureDestination.to is required when delivery.failureDestination.mode=webhook")
		}
		if !strings.HasPrefix(targetURL, "http://") && !strings.HasPrefix(targetURL, "https://") {
			return errors.New("delivery.failureDestination.to must start with http:// or https:// when delivery.failureDestination.mode=webhook")
		}
		return nil
	default:
		return errors.New("delivery.failureDestination.mode must be one of: announce, webhook")
	}
}

func nextRunForJob(job CronJob, base time.Time) (time.Time, error) {
	switch normalizeScheduleType(job) {
	case "cron":
		next, err := nextRunForCron(strings.TrimSpace(job.Schedule.Expr), strings.TrimSpace(job.Schedule.TZ), base)
		if err != nil {
			return time.Time{}, err
		}
		if job.Schedule.StaggerMs > 0 {
			next = next.Add(time.Duration(job.Schedule.StaggerMs) * time.Millisecond)
		}
		return next, nil
	case "every":
		duration, err := parseEveryDurationForJob(job)
		if err != nil {
			return time.Time{}, err
		}
		return base.Add(duration).Truncate(time.Second), nil
	case "at":
		atTime, err := parseAtTime(strings.TrimSpace(job.Schedule.At), strings.TrimSpace(job.Schedule.TZ))
		if err != nil {
			return time.Time{}, err
		}
		if !atTime.After(base) {
			return time.Time{}, errors.New("at schedule has no future run")
		}
		return atTime, nil
	default:
		return time.Time{}, fmt.Errorf("unsupported schedule type: %s", strings.TrimSpace(job.Schedule.Kind))
	}
}

func mergeJobState(existing CronJobState, incoming CronJobState) CronJobState {
	merged := incoming
	if merged.NextRunAtMs == 0 {
		merged.NextRunAtMs = existing.NextRunAtMs
	}
	if merged.RunningAtMs == 0 {
		merged.RunningAtMs = existing.RunningAtMs
	}
	if merged.LastRunAtMs == 0 {
		merged.LastRunAtMs = existing.LastRunAtMs
	}
	if strings.TrimSpace(merged.LastRunStatus) == "" {
		merged.LastRunStatus = existing.LastRunStatus
	}
	if strings.TrimSpace(merged.LastError) == "" {
		merged.LastError = existing.LastError
	}
	if merged.LastDurationMs == 0 {
		merged.LastDurationMs = existing.LastDurationMs
	}
	if merged.ConsecutiveErrors == 0 {
		merged.ConsecutiveErrors = existing.ConsecutiveErrors
	}
	if merged.ScheduleErrorCount == 0 {
		merged.ScheduleErrorCount = existing.ScheduleErrorCount
	}
	if strings.TrimSpace(merged.LastDeliveryStatus) == "" {
		merged.LastDeliveryStatus = existing.LastDeliveryStatus
	}
	if strings.TrimSpace(merged.LastDeliveryError) == "" {
		merged.LastDeliveryError = existing.LastDeliveryError
	}
	if merged.LastDelivered == nil {
		merged.LastDelivered = existing.LastDelivered
	}
	return merged
}

func syncNextRunState(job CronJob, next time.Time) CronJob {
	if next.IsZero() {
		job.State.NextRunAtMs = 0
		return job
	}
	job.State.NextRunAtMs = next.UnixMilli()
	return job
}

func normalizeSessionTarget(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "main":
		return "main"
	case "isolated":
		return "isolated"
	default:
		return ""
	}
}

func normalizeWakeMode(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "now":
		return "now"
	case "next-heartbeat":
		return "next-heartbeat"
	default:
		return ""
	}
}

func normalizeSchedule(schedule CronSchedule) CronSchedule {
	schedule.Kind = strings.ToLower(strings.TrimSpace(schedule.Kind))
	schedule.Expr = strings.TrimSpace(schedule.Expr)
	schedule.At = strings.TrimSpace(schedule.At)
	schedule.TZ = strings.TrimSpace(schedule.TZ)
	if schedule.StaggerMs < 0 {
		schedule.StaggerMs = 0
	}
	return schedule
}

func normalizePayload(spec CronPayload) CronPayload {
	normalized := spec
	normalized.Kind = normalizePayloadKind(normalized.Kind)
	normalized.Text = strings.TrimSpace(normalized.Text)
	normalized.Message = strings.TrimSpace(normalized.Message)
	normalized.Model = strings.TrimSpace(normalized.Model)
	normalized.Thinking = strings.TrimSpace(normalized.Thinking)
	if normalized.TimeoutSeconds < 0 {
		normalized.TimeoutSeconds = 0
	}
	return normalized
}

func normalizePayloadKind(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "agentturn":
		return "agentTurn"
	case "systemevent":
		return "systemEvent"
	default:
		return ""
	}
}

func normalizeDelivery(delivery *CronDelivery) *CronDelivery {
	if delivery == nil {
		return nil
	}
	result := *delivery
	result.Mode = strings.ToLower(strings.TrimSpace(result.Mode))
	result.Channel = strings.TrimSpace(result.Channel)
	if result.Mode == "announce" {
		result.Channel = normalizeAnnounceChannel(result.Channel)
	}
	result.To = strings.TrimSpace(result.To)
	result.AccountID = strings.TrimSpace(result.AccountID)
	if result.FailureDestination != nil {
		failure := *result.FailureDestination
		failure.Mode = strings.ToLower(strings.TrimSpace(failure.Mode))
		if failure.Mode == "" {
			failure.Mode = "announce"
		}
		failureChannel := strings.TrimSpace(failure.Channel)
		if failure.Mode == "announce" {
			failure.Channel = normalizeAnnounceChannel(failureChannel)
		} else {
			failure.Channel = failureChannel
		}
		failure.To = strings.TrimSpace(failure.To)
		failure.AccountID = strings.TrimSpace(failure.AccountID)
		result.FailureDestination = &failure
	}
	if result.Mode == "none" && result.Channel == "" && result.To == "" && result.AccountID == "" && !result.BestEffort && result.FailureDestination == nil {
		return nil
	}
	return &result
}

func normalizeAnnounceChannel(channel string) string {
	normalized := strings.ToLower(strings.TrimSpace(channel))
	switch normalized {
	case "", "default":
		return "default"
	case "app", "aui":
		return "app"
	case "telegram":
		return "telegram"
	default:
		return normalized
	}
}

func isValidAnnounceChannel(channel string) bool {
	switch normalizeAnnounceChannel(channel) {
	case "default", "app", "telegram":
		return true
	default:
		return false
	}
}

func buildActionPayload(job CronJob) any {
	payload := map[string]any{
		"jobId":         job.JobID,
		"name":          job.Name,
		"schedule":      job.Schedule,
		"sessionTarget": job.SessionTarget,
		"wakeMode":      job.WakeMode,
		"payload":       job.PayloadSpec,
	}
	if job.Description != "" {
		payload["description"] = job.Description
	}
	if job.AssistantID != "" {
		payload["assistantId"] = job.AssistantID
	}
	if job.DeleteAfterRun {
		payload["deleteAfterRun"] = true
	}
	if job.Delivery != nil {
		payload["delivery"] = job.Delivery
	}
	if job.SessionKey != "" {
		payload["sessionKey"] = job.SessionKey
	}
	return payload
}

func normalizeRunStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "ok", "completed", "success":
		return "ok"
	case "failed", "error":
		return "error"
	case "skipped":
		return "skipped"
	default:
		return "skipped"
	}
}

func splitModelRef(value string) (string, string) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", ""
	}
	if strings.Contains(trimmed, "/") {
		parts := strings.SplitN(trimmed, "/", 2)
		return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	}
	if strings.Contains(trimmed, ":") {
		parts := strings.SplitN(trimmed, ":", 2)
		return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	}
	return "", trimmed
}

func maxDurationMs(duration time.Duration) int64 {
	if duration <= 0 {
		return 0
	}
	return duration.Milliseconds()
}

func boolPtr(value bool) *bool {
	resolved := value
	return &resolved
}

func parseEveryDurationForJob(job CronJob) (time.Duration, error) {
	if job.Schedule.EveryMs > 0 {
		return time.Duration(job.Schedule.EveryMs) * time.Millisecond, nil
	}
	return 0, errors.New("schedule.everyMs is required when schedule.kind=every")
}

func nextRunForCron(expr string, timezone string, base time.Time) (time.Time, error) {
	spec, err := parseCronExpr(expr)
	if err != nil {
		return time.Time{}, err
	}
	loc := time.Local
	if tz := strings.TrimSpace(timezone); tz != "" {
		if resolved, err := time.LoadLocation(tz); err == nil {
			loc = resolved
		}
	}
	current := base.In(loc).Truncate(time.Minute)
	for i := 0; i < 525600; i++ {
		current = current.Add(time.Minute)
		if matchesCron(spec, current) {
			return current, nil
		}
	}
	return time.Time{}, errors.New("cron schedule not found")
}

func parseAtTime(value string, timezone string) (time.Time, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return time.Time{}, errors.New("at schedule is required")
	}
	if parsed, err := time.Parse(time.RFC3339Nano, trimmed); err == nil {
		return parsed, nil
	}
	if parsed, err := time.Parse(time.RFC3339, trimmed); err == nil {
		return parsed, nil
	}
	location := time.Local
	if tz := strings.TrimSpace(timezone); tz != "" {
		resolved, err := time.LoadLocation(tz)
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid timezone: %s", timezone)
		}
		location = resolved
	}
	layouts := []string{
		"2006-01-02T15:04:05",
		"2006-01-02T15:04",
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
	}
	for _, layout := range layouts {
		if parsed, err := time.ParseInLocation(layout, trimmed, location); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid at schedule: %s", value)
}

type cronSpec struct {
	minutes  map[int]bool
	hours    map[int]bool
	days     map[int]bool
	months   map[int]bool
	weekdays map[int]bool
}

func parseCronExpr(expr string) (cronSpec, error) {
	parts := strings.Fields(strings.TrimSpace(expr))
	if len(parts) != 5 {
		return cronSpec{}, fmt.Errorf("invalid cron expression: %s", expr)
	}
	minutes, err := parseCronField(parts[0], 0, 59)
	if err != nil {
		return cronSpec{}, err
	}
	hours, err := parseCronField(parts[1], 0, 23)
	if err != nil {
		return cronSpec{}, err
	}
	days, err := parseCronField(parts[2], 1, 31)
	if err != nil {
		return cronSpec{}, err
	}
	months, err := parseCronField(parts[3], 1, 12)
	if err != nil {
		return cronSpec{}, err
	}
	weekdays, err := parseCronField(parts[4], 0, 6)
	if err != nil {
		return cronSpec{}, err
	}
	return cronSpec{
		minutes:  minutes,
		hours:    hours,
		days:     days,
		months:   months,
		weekdays: weekdays,
	}, nil
}

func matchesCron(spec cronSpec, t time.Time) bool {
	if !spec.minutes[t.Minute()] {
		return false
	}
	if !spec.hours[t.Hour()] {
		return false
	}
	if !spec.days[t.Day()] {
		return false
	}
	if !spec.months[int(t.Month())] {
		return false
	}
	weekday := int(t.Weekday())
	if !spec.weekdays[weekday] {
		return false
	}
	return true
}

func parseCronField(field string, min int, max int) (map[int]bool, error) {
	if field == "" {
		return nil, errors.New("cron field missing")
	}
	values := make(map[int]bool)
	for _, part := range strings.Split(field, ",") {
		part = strings.TrimSpace(part)
		if part == "*" {
			for i := min; i <= max; i++ {
				values[i] = true
			}
			continue
		}
		step := 1
		rangePart := part
		if strings.Contains(part, "/") {
			chunks := strings.Split(part, "/")
			if len(chunks) != 2 {
				return nil, fmt.Errorf("invalid cron step: %s", part)
			}
			rangePart = chunks[0]
			if parsed, err := parseCronNumber(chunks[1]); err == nil && parsed > 0 {
				step = parsed
			} else {
				return nil, fmt.Errorf("invalid cron step: %s", part)
			}
		}
		if rangePart == "*" {
			for i := min; i <= max; i += step {
				values[i] = true
			}
			continue
		}
		if strings.Contains(rangePart, "-") {
			bounds := strings.Split(rangePart, "-")
			if len(bounds) != 2 {
				return nil, fmt.Errorf("invalid cron range: %s", part)
			}
			start, err := parseCronNumber(bounds[0])
			if err != nil {
				return nil, err
			}
			end, err := parseCronNumber(bounds[1])
			if err != nil {
				return nil, err
			}
			if start < min || end > max || start > end {
				return nil, fmt.Errorf("cron range out of bounds: %s", part)
			}
			for i := start; i <= end; i += step {
				values[i] = true
			}
			continue
		}
		value, err := parseCronNumber(rangePart)
		if err != nil {
			return nil, err
		}
		if value < min || value > max {
			return nil, fmt.Errorf("cron value out of bounds: %d", value)
		}
		values[value] = true
	}
	if len(values) == 0 {
		return nil, errors.New("cron field has no values")
	}
	return values, nil
}

func parseCronNumber(value string) (int, error) {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0, err
	}
	return parsed, nil
}
