package cron

import (
	"context"
	"strings"
	"testing"
	"time"
)

type schedulerTestStore struct {
	jobs   map[string]CronJob
	runs   map[string]CronRunRecord
	events []CronRunEvent
}

func newSchedulerTestStore() *schedulerTestStore {
	return &schedulerTestStore{
		jobs: make(map[string]CronJob),
		runs: make(map[string]CronRunRecord),
	}
}

func (store *schedulerTestStore) SaveCronJob(_ context.Context, job CronJob) error {
	store.jobs[job.JobID] = job
	return nil
}

func (store *schedulerTestStore) ListCronJobs(_ context.Context) ([]CronJob, error) {
	items := make([]CronJob, 0, len(store.jobs))
	for _, job := range store.jobs {
		items = append(items, job)
	}
	return items, nil
}

func (store *schedulerTestStore) DeleteCronJob(_ context.Context, jobID string) error {
	delete(store.jobs, jobID)
	return nil
}

func (store *schedulerTestStore) SaveCronRun(_ context.Context, run CronRunRecord) error {
	store.runs[run.RunID] = run
	return nil
}

func (store *schedulerTestStore) GetCronRun(_ context.Context, runID string) (CronRunRecord, bool, error) {
	run, ok := store.runs[runID]
	return run, ok, nil
}

func (store *schedulerTestStore) ListCronRuns(_ context.Context, query ListRunsQuery) (ListRunsResult, error) {
	matches := make([]CronRunRecord, 0)
	for _, run := range store.runs {
		if query.JobID != "" && !strings.EqualFold(strings.TrimSpace(run.JobID), strings.TrimSpace(query.JobID)) {
			continue
		}
		if query.Status != "" && !strings.EqualFold(strings.TrimSpace(run.Status), strings.TrimSpace(query.Status)) {
			continue
		}
		if len(query.Statuses) > 0 {
			matched := false
			for _, expected := range query.Statuses {
				if strings.EqualFold(strings.TrimSpace(run.Status), strings.TrimSpace(expected)) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}
		if query.DeliveryStatus != "" && !strings.EqualFold(strings.TrimSpace(run.DeliveryStatus), strings.TrimSpace(query.DeliveryStatus)) {
			continue
		}
		if len(query.DeliveryStatuses) > 0 {
			matched := false
			for _, expected := range query.DeliveryStatuses {
				if strings.EqualFold(strings.TrimSpace(run.DeliveryStatus), strings.TrimSpace(expected)) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}
		matches = append(matches, run)
	}
	total := len(matches)
	offset := query.Offset
	if offset < 0 {
		offset = 0
	}
	if offset > total {
		offset = total
	}
	limit := query.Limit
	if limit <= 0 || offset+limit > total {
		limit = total - offset
	}
	items := matches[offset : offset+limit]
	return ListRunsResult{
		Items: append([]CronRunRecord(nil), items...),
		Total: total,
	}, nil
}

func (store *schedulerTestStore) SaveCronRunEvent(_ context.Context, event CronRunEvent) error {
	store.events = append(store.events, event)
	return nil
}

func (store *schedulerTestStore) ListCronRunEvents(_ context.Context, query ListRunEventsQuery) (ListRunEventsResult, error) {
	items := make([]CronRunEvent, 0)
	for _, event := range store.events {
		if !strings.EqualFold(strings.TrimSpace(event.RunID), strings.TrimSpace(query.RunID)) {
			continue
		}
		items = append(items, event)
	}
	total := len(items)
	offset := query.Offset
	if offset < 0 {
		offset = 0
	}
	if offset > total {
		offset = total
	}
	limit := query.Limit
	if limit <= 0 || offset+limit > total {
		limit = total - offset
	}
	selected := items[offset : offset+limit]
	return ListRunEventsResult{
		Items: append([]CronRunEvent(nil), selected...),
		Total: total,
	}, nil
}

func TestAppendRunEventEmitsRealtimeNotifier(t *testing.T) {
	store := newSchedulerTestStore()
	scheduler := NewScheduler(store, nil)
	now := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
	scheduler.now = func() time.Time { return now }

	received := make([]RunRealtimeEvent, 0)
	scheduler.SetRunRealtimeNotifier(func(_ context.Context, event RunRealtimeEvent) {
		received = append(received, event)
	})

	run := CronRunRecord{
		RunID:     "run-1",
		JobID:     "job-1",
		JobName:   "job",
		Status:    "running",
		StartedAt: now,
	}
	scheduler.appendRunEvent(context.Background(), buildRunEvent(run, "started", "running", "", "", "telegram", "scheduler", nil))

	if len(received) != 1 {
		t.Fatalf("expected one realtime event, got %d", len(received))
	}
	if received[0].RunID != "run-1" {
		t.Fatalf("expected run id run-1, got %q", received[0].RunID)
	}
	if received[0].Stage != "started" {
		t.Fatalf("expected stage started, got %q", received[0].Stage)
	}
	if received[0].Status != "running" {
		t.Fatalf("expected status running, got %q", received[0].Status)
	}
	if received[0].Source != "scheduler" {
		t.Fatalf("expected source scheduler, got %q", received[0].Source)
	}
	if !received[0].CreatedAt.Equal(now) {
		t.Fatalf("expected createdAt %s, got %s", now.Format(time.RFC3339), received[0].CreatedAt.Format(time.RFC3339))
	}
}

func TestTriggerActionSystemEventUsesHeartbeatHooks(t *testing.T) {
	scheduler := NewScheduler(nil, nil)

	var capturedEvent MainSystemEventRequest
	var capturedWake WakeTriggerRequest
	scheduler.SetMainSystemEventEnqueuer(func(_ context.Context, request MainSystemEventRequest) bool {
		capturedEvent = request
		return true
	})
	scheduler.SetWakeTrigger(func(_ context.Context, request WakeTriggerRequest) WakeTriggerResult {
		capturedWake = request
		return WakeTriggerResult{Accepted: true, ExecutedStatus: "queued"}
	})

	err := scheduler.triggerAction(context.Background(), CronJob{
		JobID:         "job-1",
		AssistantID:   "assistant-1",
		SessionTarget: "main",
		SessionKey:    "cron/main",
		WakeMode:      "now",
		PayloadSpec: CronPayload{
			Kind: "systemEvent",
			Text: "cron reminder",
		},
	}, "run-1")
	if err != nil {
		t.Fatalf("triggerAction error: %v", err)
	}
	if capturedEvent.Text != "cron reminder" {
		t.Fatalf("expected system event text to be queued")
	}
	if capturedEvent.ContextKey != "cron:job-1" {
		t.Fatalf("expected context key cron:job-1, got %q", capturedEvent.ContextKey)
	}
	if capturedEvent.RunID != "run-1" {
		t.Fatalf("expected run id run-1, got %q", capturedEvent.RunID)
	}
	if capturedEvent.SessionKey != "" {
		t.Fatalf("expected default main session key to map to empty heartbeat target, got %q", capturedEvent.SessionKey)
	}
	if capturedWake.Reason != "cron.wake.now" {
		t.Fatalf("expected wake reason cron.wake.now, got %q", capturedWake.Reason)
	}
	if !capturedWake.Force {
		t.Fatalf("expected wake force=true for now mode")
	}
	if capturedWake.Source != "cron" {
		t.Fatalf("expected wake source cron, got %q", capturedWake.Source)
	}
	if capturedWake.RunID != "run-1" {
		t.Fatalf("expected wake run id run-1, got %q", capturedWake.RunID)
	}
	if capturedWake.SessionKey != "" {
		t.Fatalf("expected wake session key to follow normalized main target, got %q", capturedWake.SessionKey)
	}
}

func TestTriggerActionSystemEventReturnsErrorWhenEnqueueFails(t *testing.T) {
	scheduler := NewScheduler(nil, nil)
	scheduler.SetMainSystemEventEnqueuer(func(_ context.Context, _ MainSystemEventRequest) bool {
		return false
	})

	err := scheduler.triggerAction(context.Background(), CronJob{
		JobID:         "job-2",
		AssistantID:   "assistant-1",
		SessionTarget: "main",
		WakeMode:      "next-heartbeat",
		PayloadSpec: CronPayload{
			Kind: "systemEvent",
			Text: "cron reminder",
		},
	}, "run-2")
	if err == nil {
		t.Fatalf("expected enqueue failure error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "enqueue") {
		t.Fatalf("expected enqueue-related error, got %q", err.Error())
	}
}

func TestWakeUsesHeartbeatHooks(t *testing.T) {
	scheduler := NewScheduler(nil, nil)

	var capturedEvent MainSystemEventRequest
	var capturedWake WakeTriggerRequest
	scheduler.SetMainSystemEventEnqueuer(func(_ context.Context, request MainSystemEventRequest) bool {
		capturedEvent = request
		return true
	})
	scheduler.SetWakeTrigger(func(_ context.Context, request WakeTriggerRequest) WakeTriggerResult {
		capturedWake = request
		return WakeTriggerResult{Accepted: true, ExecutedStatus: "queued"}
	})

	result, err := scheduler.Wake(context.Background(), "now", "manual wake", "cron/main")
	if err != nil {
		t.Fatalf("wake error: %v", err)
	}
	if !result.OK || !result.Accepted {
		t.Fatalf("expected wake accepted")
	}
	if result.SessionKey != "" {
		t.Fatalf("expected heartbeat default target session key, got %q", result.SessionKey)
	}
	if capturedEvent.Text != "manual wake" {
		t.Fatalf("expected wake text to be enqueued")
	}
	if capturedEvent.ContextKey != "cron:wake" {
		t.Fatalf("expected wake context key cron:wake, got %q", capturedEvent.ContextKey)
	}
	if strings.TrimSpace(capturedEvent.RunID) == "" {
		t.Fatalf("expected wake run id for dedupe")
	}
	if capturedWake.Reason != "cron.wake.now" || !capturedWake.Force {
		t.Fatalf("expected wake trigger to use now mode semantics")
	}
	if capturedWake.Source != "cron" {
		t.Fatalf("expected wake source cron, got %q", capturedWake.Source)
	}
}

func TestApplyRunDeliveryQueuesPendingForHeartbeatResult(t *testing.T) {
	scheduler := NewScheduler(nil, nil)
	scheduler.SetMainSystemEventEnqueuer(func(_ context.Context, _ MainSystemEventRequest) bool {
		return true
	})
	run := CronRunRecord{
		RunID:      "run-announce-1",
		JobID:      "job-announce-1",
		JobName:    "cron-test",
		Status:     "completed",
		StartedAt:  time.UnixMilli(1000),
		EndedAt:    time.UnixMilli(2000),
		SessionKey: "v2::-::telegram::-::telegram:default:private:123::-::telegram:default:private:123",
	}
	job := CronJob{
		JobID:       "job-announce-1",
		Name:        "cron-test",
		AssistantID: "assistant-1",
		PayloadSpec: CronPayload{
			Kind: "systemEvent",
			Text: "prompt text should not be delivered directly",
		},
		Delivery: &CronDelivery{
			Mode:    "announce",
			Channel: "default",
		},
	}

	scheduler.applyRunDelivery(context.Background(), job, &run)
	if run.DeliveryStatus != "pending" {
		t.Fatalf("expected delivery status pending, got %q", run.DeliveryStatus)
	}
	if run.DeliveryError != "" {
		t.Fatalf("expected no delivery error, got %q", run.DeliveryError)
	}
	pending := scheduler.pendingDeliveries["run-announce-1"]
	if pending.Channel != "default" {
		t.Fatalf("expected pending delivery channel default, got %q", pending.Channel)
	}
}

func TestUpsertResolvesAssistantIDFromGatewayResolver(t *testing.T) {
	store := newSchedulerTestStore()
	scheduler := NewScheduler(store, nil)
	scheduler.SetAssistantIDResolver(func(_ context.Context) string {
		return "assistant-default"
	})

	job := CronJob{
		JobID:   "job-resolve-assistant",
		Name:    "resolve-assistant",
		Enabled: true,
		Schedule: CronSchedule{
			Kind:    "every",
			EveryMs: 60_000,
		},
		SessionTarget: "main",
		WakeMode:      "now",
		PayloadSpec: CronPayload{
			Kind: "systemEvent",
			Text: "ping",
		},
	}

	stored, err := scheduler.Upsert(context.Background(), job)
	if err != nil {
		t.Fatalf("upsert failed: %v", err)
	}
	if strings.TrimSpace(stored.AssistantID) != "assistant-default" {
		t.Fatalf("assistant id mismatch: got %q", stored.AssistantID)
	}
}

func TestExecuteJobUsesResolvedAssistantIDForPendingDelivery(t *testing.T) {
	store := newSchedulerTestStore()
	scheduler := NewScheduler(store, nil)
	currentAssistantID := "assistant-v1"
	scheduler.SetAssistantIDResolver(func(_ context.Context) string {
		return currentAssistantID
	})
	scheduler.SetMainSystemEventEnqueuer(func(_ context.Context, _ MainSystemEventRequest) bool {
		return true
	})
	scheduler.SetWakeTrigger(func(_ context.Context, _ WakeTriggerRequest) WakeTriggerResult {
		return WakeTriggerResult{Accepted: true, ExecutedStatus: "queued"}
	})

	job := CronJob{
		JobID:   "job-assistant-runtime",
		Name:    "assistant-runtime",
		Enabled: true,
		Schedule: CronSchedule{
			Kind:    "every",
			EveryMs: 60_000,
		},
		SessionTarget: "main",
		WakeMode:      "now",
		PayloadSpec: CronPayload{
			Kind: "systemEvent",
			Text: "ping",
		},
		Delivery: &CronDelivery{
			Mode:    "announce",
			Channel: "default",
		},
	}

	stored, err := scheduler.Upsert(context.Background(), job)
	if err != nil {
		t.Fatalf("upsert failed: %v", err)
	}
	if strings.TrimSpace(stored.AssistantID) != "assistant-v1" {
		t.Fatalf("expected assistant-v1 at upsert, got %q", stored.AssistantID)
	}

	currentAssistantID = "assistant-v2"
	run, err := scheduler.executeJob(context.Background(), stored)
	if err != nil {
		t.Fatalf("executeJob failed: %v", err)
	}

	scheduler.mu.RLock()
	pending, ok := scheduler.pendingDeliveries[strings.TrimSpace(run.RunID)]
	scheduler.mu.RUnlock()
	if !ok {
		t.Fatalf("expected pending delivery entry for run %q", run.RunID)
	}
	if strings.TrimSpace(pending.AssistantID) != "assistant-v2" {
		t.Fatalf("pending delivery assistant id mismatch: got %q", pending.AssistantID)
	}
}

func TestHandleHeartbeatDeliveryEventUsesResultMessage(t *testing.T) {
	scheduler := NewScheduler(nil, nil)
	scheduler.SetMainSystemEventEnqueuer(func(_ context.Context, _ MainSystemEventRequest) bool {
		return true
	})
	run := CronRunRecord{
		RunID:      "run-announce-2",
		JobID:      "job-announce-2",
		JobName:    "cron-city",
		Status:     "completed",
		StartedAt:  time.UnixMilli(1000),
		EndedAt:    time.UnixMilli(2000),
		SessionKey: "v2::-::telegram::-::telegram:default:private:123::-::telegram:default:private:123",
	}
	job := CronJob{
		JobID:       "job-announce-2",
		Name:        "cron-city",
		AssistantID: "assistant-2",
		PayloadSpec: CronPayload{
			Kind: "systemEvent",
			Text: "pick one random county-level city in China",
		},
		Delivery: &CronDelivery{
			Mode:    "announce",
			Channel: "default",
		},
	}
	scheduler.applyRunDelivery(context.Background(), job, &run)

	var delivered AnnouncementRequest
	scheduler.SetAnnouncementSender(func(_ context.Context, request AnnouncementRequest) error {
		delivered = request
		return nil
	})
	scheduler.HandleHeartbeatDeliveryEvent(context.Background(), HeartbeatDeliveryEvent{
		RunID:      "run-announce-2",
		Source:     "cron",
		Status:     "sent",
		Message:    "Fenghua",
		SessionKey: "v2::-::telegram::-::telegram:default:private:123::-::telegram:default:private:123",
	})

	if delivered.RunID != "run-announce-2" {
		t.Fatalf("expected run id run-announce-2, got %q", delivered.RunID)
	}
	if delivered.Message != "Fenghua" {
		t.Fatalf("expected heartbeat result message to be delivered, got %q", delivered.Message)
	}
	if delivered.JobID != "job-announce-2" {
		t.Fatalf("expected job id job-announce-2, got %q", delivered.JobID)
	}
	if delivered.Channel != "default" {
		t.Fatalf("expected delivery channel default, got %q", delivered.Channel)
	}
}

func TestHandleHeartbeatDeliveryEventPrefersPendingSessionKey(t *testing.T) {
	scheduler := NewScheduler(nil, nil)
	scheduler.SetMainSystemEventEnqueuer(func(_ context.Context, _ MainSystemEventRequest) bool {
		return true
	})
	pendingSessionKey := "v2::-::telegram::-::telegram:default:private:123::-::telegram:default:private:123"
	run := CronRunRecord{
		RunID:      "run-announce-3",
		JobID:      "job-announce-3",
		JobName:    "cron-route",
		Status:     "completed",
		StartedAt:  time.UnixMilli(1000),
		EndedAt:    time.UnixMilli(2000),
		SessionKey: pendingSessionKey,
	}
	job := CronJob{
		JobID:       "job-announce-3",
		Name:        "cron-route",
		AssistantID: "assistant-3",
		PayloadSpec: CronPayload{
			Kind: "systemEvent",
			Text: "route check",
		},
		Delivery: &CronDelivery{
			Mode:    "announce",
			Channel: "default",
		},
	}
	scheduler.applyRunDelivery(context.Background(), job, &run)

	var delivered AnnouncementRequest
	scheduler.SetAnnouncementSender(func(_ context.Context, request AnnouncementRequest) error {
		delivered = request
		return nil
	})
	scheduler.HandleHeartbeatDeliveryEvent(context.Background(), HeartbeatDeliveryEvent{
		RunID:      "run-announce-3",
		Source:     "cron",
		Status:     "sent",
		Message:    "ok",
		SessionKey: "v2::-::aui::-::thread-app::-::thread-app",
	})

	if delivered.SessionKey != pendingSessionKey {
		t.Fatalf("expected pending session key to be preferred, got %q", delivered.SessionKey)
	}
}

func TestHandleHeartbeatDeliveryEventFallsBackToHeartbeatSessionKeyForSyntheticPending(t *testing.T) {
	scheduler := NewScheduler(nil, nil)
	scheduler.SetMainSystemEventEnqueuer(func(_ context.Context, _ MainSystemEventRequest) bool {
		return true
	})
	run := CronRunRecord{
		RunID:      "run-announce-4",
		JobID:      "job-announce-4",
		JobName:    "cron-route-fallback",
		Status:     "completed",
		StartedAt:  time.UnixMilli(1000),
		EndedAt:    time.UnixMilli(2000),
		SessionKey: "cron/main",
	}
	job := CronJob{
		JobID:       "job-announce-4",
		Name:        "cron-route-fallback",
		AssistantID: "assistant-4",
		PayloadSpec: CronPayload{
			Kind: "systemEvent",
			Text: "route fallback check",
		},
		Delivery: &CronDelivery{
			Mode:    "announce",
			Channel: "default",
		},
	}
	scheduler.applyRunDelivery(context.Background(), job, &run)

	heartbeatSessionKey := "v2::-::telegram::-::telegram:default:private:456::-::telegram:default:private:456"
	var delivered AnnouncementRequest
	scheduler.SetAnnouncementSender(func(_ context.Context, request AnnouncementRequest) error {
		delivered = request
		return nil
	})
	scheduler.HandleHeartbeatDeliveryEvent(context.Background(), HeartbeatDeliveryEvent{
		RunID:      "run-announce-4",
		Source:     "cron",
		Status:     "sent",
		Message:    "Anji",
		SessionKey: heartbeatSessionKey,
	})

	if delivered.SessionKey != heartbeatSessionKey {
		t.Fatalf("expected heartbeat session key fallback, got %q", delivered.SessionKey)
	}
}

func TestApplyRunDeliverySupportsExplicitChannels(t *testing.T) {
	scheduler := NewScheduler(nil, nil)
	scheduler.SetMainSystemEventEnqueuer(func(_ context.Context, _ MainSystemEventRequest) bool {
		return true
	})
	baseRun := CronRunRecord{
		RunID:      "run-explicit-channel",
		JobID:      "job-explicit-channel",
		JobName:    "cron-channel",
		Status:     "completed",
		StartedAt:  time.UnixMilli(1000),
		EndedAt:    time.UnixMilli(2000),
		SessionKey: "v2::-::telegram::-::telegram:default:private:123::-::telegram:default:private:123",
	}
	cases := []struct {
		name    string
		channel string
		want    string
	}{
		{name: "default", channel: "default", want: "default"},
		{name: "app", channel: "app", want: "app"},
		{name: "telegram", channel: "telegram", want: "telegram"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			run := baseRun
			run.RunID = "run-explicit-channel-" + strings.ReplaceAll(tc.channel, " ", "-")
			job := CronJob{
				JobID:       "job-explicit-channel",
				Name:        "cron-channel",
				AssistantID: "assistant-1",
				PayloadSpec: CronPayload{
					Kind: "systemEvent",
					Text: "notify",
				},
				Delivery: &CronDelivery{
					Mode:    "announce",
					Channel: tc.channel,
				},
			}
			scheduler.applyRunDelivery(context.Background(), job, &run)
			if run.DeliveryStatus != "pending" {
				t.Fatalf("expected pending, got %q (err=%q)", run.DeliveryStatus, run.DeliveryError)
			}
			pending := scheduler.pendingDeliveries[run.RunID]
			if pending.Channel != tc.want {
				t.Fatalf("expected pending channel %q, got %q", tc.want, pending.Channel)
			}
		})
	}
}

func TestApplyRunDeliveryRejectsInvalidChannel(t *testing.T) {
	scheduler := NewScheduler(nil, nil)
	run := CronRunRecord{
		RunID:      "run-invalid-channel",
		JobID:      "job-invalid-channel",
		JobName:    "cron-invalid-channel",
		Status:     "completed",
		StartedAt:  time.UnixMilli(1000),
		EndedAt:    time.UnixMilli(2000),
		SessionKey: "v2::-::telegram::-::telegram:default:private:123::-::telegram:default:private:123",
	}
	job := CronJob{
		JobID:       "job-invalid-channel",
		Name:        "cron-invalid-channel",
		AssistantID: "assistant-1",
		PayloadSpec: CronPayload{
			Kind: "systemEvent",
			Text: "notify",
		},
		Delivery: &CronDelivery{
			Mode:    "announce",
			Channel: "web",
		},
	}
	scheduler.applyRunDelivery(context.Background(), job, &run)
	if run.DeliveryStatus != "failed" {
		t.Fatalf("expected failed, got %q", run.DeliveryStatus)
	}
	if !strings.Contains(strings.ToLower(run.DeliveryError), "delivery.channel") {
		t.Fatalf("expected delivery.channel error, got %q", run.DeliveryError)
	}
}

func TestExecuteJobKeepsRunningStatusForHeartbeatAnnounce(t *testing.T) {
	scheduler := NewScheduler(nil, nil)
	scheduler.SetMainSystemEventEnqueuer(func(_ context.Context, _ MainSystemEventRequest) bool {
		return true
	})
	scheduler.SetWakeTrigger(func(_ context.Context, _ WakeTriggerRequest) WakeTriggerResult {
		return WakeTriggerResult{Accepted: true, ExecutedStatus: "queued"}
	})

	job := CronJob{
		JobID:       "job-running-1",
		Name:        "cron-running",
		AssistantID: "assistant-1",
		Enabled:     true,
		Schedule: CronSchedule{
			Kind:    "every",
			EveryMs: 60_000,
		},
		SessionTarget: "main",
		WakeMode:      "now",
		PayloadSpec: CronPayload{
			Kind: "systemEvent",
			Text: "heartbeat execution",
		},
		Delivery: &CronDelivery{
			Mode:    "announce",
			Channel: "default",
		},
	}

	run, err := scheduler.executeJob(context.Background(), job)
	if err != nil {
		t.Fatalf("executeJob error: %v", err)
	}
	if run.Status != "running" {
		t.Fatalf("expected running status, got %q", run.Status)
	}
	if !run.EndedAt.IsZero() {
		t.Fatalf("expected zero endedAt while heartbeat result pending")
	}
	if run.DeliveryStatus != "pending" {
		t.Fatalf("expected delivery status pending, got %q", run.DeliveryStatus)
	}
	if _, ok := scheduler.pendingDeliveries[run.RunID]; !ok {
		t.Fatalf("expected pending delivery to be tracked")
	}
}

func TestHandleHeartbeatDeliveryEventUpdatesJobRunState(t *testing.T) {
	scheduler := NewScheduler(nil, nil)
	scheduler.SetMainSystemEventEnqueuer(func(_ context.Context, _ MainSystemEventRequest) bool {
		return true
	})
	scheduler.SetWakeTrigger(func(_ context.Context, _ WakeTriggerRequest) WakeTriggerResult {
		return WakeTriggerResult{Accepted: true, ExecutedStatus: "queued"}
	})

	job, err := scheduler.Upsert(context.Background(), CronJob{
		JobID:       "job-running-2",
		Name:        "cron-running-state",
		AssistantID: "assistant-2",
		Enabled:     true,
		Schedule: CronSchedule{
			Kind:    "every",
			EveryMs: 60_000,
		},
		SessionTarget: "main",
		WakeMode:      "now",
		PayloadSpec: CronPayload{
			Kind: "systemEvent",
			Text: "heartbeat state update",
		},
		Delivery: &CronDelivery{
			Mode:    "announce",
			Channel: "default",
		},
	})
	if err != nil {
		t.Fatalf("upsert job error: %v", err)
	}

	run, err := scheduler.executeJob(context.Background(), job)
	if err != nil {
		t.Fatalf("executeJob error: %v", err)
	}
	if run.Status != "running" {
		t.Fatalf("expected running status before heartbeat, got %q", run.Status)
	}

	scheduler.SetAnnouncementSender(func(_ context.Context, _ AnnouncementRequest) error {
		return nil
	})
	scheduler.HandleHeartbeatDeliveryEvent(context.Background(), HeartbeatDeliveryEvent{
		RunID:      run.RunID,
		Source:     "cron",
		Status:     "sent",
		Message:    "Shanghai",
		SessionKey: "v2::-::telegram::-::telegram:default:private:123::-::telegram:default:private:123",
	})

	updated, ok := scheduler.lookup(job.JobID)
	if !ok {
		t.Fatalf("expected updated job to exist")
	}
	finalStatus := strings.ToLower(strings.TrimSpace(updated.State.LastRunStatus))
	if finalStatus != "completed" && finalStatus != "ok" {
		t.Fatalf("expected last run status completed/ok, got %q", updated.State.LastRunStatus)
	}
	if updated.State.RunningAtMs != 0 {
		t.Fatalf("expected runningAtMs to be cleared, got %d", updated.State.RunningAtMs)
	}
}

func TestHandleHeartbeatDeliveryEventKeepsRunModelProvider(t *testing.T) {
	store := newSchedulerTestStore()
	scheduler := NewScheduler(store, nil)
	scheduler.SetMainSystemEventEnqueuer(func(_ context.Context, _ MainSystemEventRequest) bool {
		return true
	})
	scheduler.SetWakeTrigger(func(_ context.Context, _ WakeTriggerRequest) WakeTriggerResult {
		return WakeTriggerResult{Accepted: true, ExecutedStatus: "queued"}
	})

	job := CronJob{
		JobID:       "job-model-provider-1",
		Name:        "cron-model-provider",
		AssistantID: "assistant-model-provider",
		Enabled:     true,
		Schedule: CronSchedule{
			Kind:    "every",
			EveryMs: 60_000,
		},
		SessionTarget: "main",
		WakeMode:      "now",
		PayloadSpec: CronPayload{
			Kind:  "systemEvent",
			Text:  "capture model/provider",
			Model: "openai/gpt-4.1",
		},
		Delivery: &CronDelivery{
			Mode:    "announce",
			Channel: "default",
		},
	}

	run, err := scheduler.executeJob(context.Background(), job)
	if err != nil {
		t.Fatalf("executeJob error: %v", err)
	}
	if run.Provider != "openai" || run.Model != "gpt-4.1" {
		t.Fatalf("expected run model/provider openai/gpt-4.1, got %q/%q", run.Provider, run.Model)
	}

	scheduler.SetAnnouncementSender(func(_ context.Context, _ AnnouncementRequest) error {
		return nil
	})
	scheduler.HandleHeartbeatDeliveryEvent(context.Background(), HeartbeatDeliveryEvent{
		RunID:      run.RunID,
		Source:     "cron",
		Status:     "sent",
		Message:    "ok",
		SessionKey: "v2::-::telegram::-::telegram:default:private:123::-::telegram:default:private:123",
	})

	persisted, ok, err := store.GetCronRun(context.Background(), run.RunID)
	if err != nil {
		t.Fatalf("get run error: %v", err)
	}
	if !ok {
		t.Fatalf("expected persisted run")
	}
	if persisted.Provider != "openai" || persisted.Model != "gpt-4.1" {
		t.Fatalf("expected persisted model/provider openai/gpt-4.1, got %q/%q", persisted.Provider, persisted.Model)
	}
}

func TestFinalizeExpiredPendingDeliveriesMarksRunFailed(t *testing.T) {
	scheduler := NewScheduler(nil, nil)
	baseTime := time.UnixMilli(1_000_000)
	scheduler.now = func() time.Time { return baseTime }
	job, err := scheduler.Upsert(context.Background(), CronJob{
		JobID:       "job-timeout-1",
		Name:        "cron-timeout",
		AssistantID: "assistant-timeout",
		Enabled:     true,
		Schedule: CronSchedule{
			Kind:    "every",
			EveryMs: 60_000,
		},
		SessionTarget: "main",
		PayloadSpec: CronPayload{
			Kind: "systemEvent",
			Text: "timeout check",
		},
		Delivery: &CronDelivery{
			Mode:    "announce",
			Channel: "default",
		},
	})
	if err != nil {
		t.Fatalf("upsert job error: %v", err)
	}

	run := CronRunRecord{
		RunID:      "run-timeout-1",
		JobID:      job.JobID,
		JobName:    job.Name,
		Status:     "running",
		StartedAt:  baseTime,
		SessionKey: "v2::-::telegram::-::telegram:default:private:123::-::telegram:default:private:123",
	}
	scheduler.trackPendingRunDelivery(job, run, "default")
	scheduler.finalizeExpiredPendingDeliveries(context.Background(), baseTime.Add(defaultPendingDeliveryTimeout+time.Second))

	if _, ok := scheduler.pendingDeliveries[run.RunID]; ok {
		t.Fatalf("expected timed out pending delivery to be removed")
	}
	updated, ok := scheduler.lookup(job.JobID)
	if !ok {
		t.Fatalf("expected updated job to exist")
	}
	if strings.ToLower(strings.TrimSpace(updated.State.LastRunStatus)) != "error" {
		t.Fatalf("expected last run status error, got %q", updated.State.LastRunStatus)
	}
	if strings.ToLower(strings.TrimSpace(updated.State.LastDeliveryStatus)) != "failed" {
		t.Fatalf("expected last delivery status failed, got %q", updated.State.LastDeliveryStatus)
	}
}

func TestRecoverPendingRunsMarksOrphanedPendingAsFailed(t *testing.T) {
	store := newSchedulerTestStore()
	scheduler := NewScheduler(store, nil)
	baseTime := time.UnixMilli(2_000_000)
	scheduler.now = func() time.Time { return baseTime }
	if err := store.SaveCronRun(context.Background(), CronRunRecord{
		RunID:          "run-recover-1",
		JobID:          "job-recover-1",
		JobName:        "recover-job",
		Status:         "running",
		DeliveryStatus: "pending",
		StartedAt:      baseTime.Add(-2 * time.Minute),
	}); err != nil {
		t.Fatalf("save run error: %v", err)
	}

	scheduler.recoverPendingRuns(context.Background())

	run, ok, err := store.GetCronRun(context.Background(), "run-recover-1")
	if err != nil {
		t.Fatalf("get run error: %v", err)
	}
	if !ok {
		t.Fatalf("expected recovered run to exist")
	}
	if strings.ToLower(strings.TrimSpace(run.Status)) != "failed" {
		t.Fatalf("expected run status failed, got %q", run.Status)
	}
	if strings.ToLower(strings.TrimSpace(run.DeliveryStatus)) != "failed" {
		t.Fatalf("expected delivery status failed, got %q", run.DeliveryStatus)
	}
	if run.EndedAt.IsZero() {
		t.Fatalf("expected recovered run endedAt to be set")
	}
	events, err := store.ListCronRunEvents(context.Background(), ListRunEventsQuery{
		RunID: "run-recover-1",
		Limit: 10,
	})
	if err != nil {
		t.Fatalf("list run events error: %v", err)
	}
	if events.Total < 2 {
		t.Fatalf("expected at least 2 recovery events, got %d", events.Total)
	}
}

func TestExecuteJobUsesIsolatedExecutorForAgentTurn(t *testing.T) {
	scheduler := NewScheduler(nil, nil)
	var captured IsolatedExecutionRequest
	scheduler.SetIsolatedExecutor(func(_ context.Context, request IsolatedExecutionRequest) (IsolatedExecutionResult, error) {
		captured = request
		return IsolatedExecutionResult{
			Status:     "completed",
			Summary:    "isolated done",
			SessionKey: "cron/isolated/job-iso-1",
			Model:      "gpt-4.1",
			Provider:   "openai",
			UsageJSON:  `{"totalTokens":42}`,
		}, nil
	})

	run, err := scheduler.executeJob(context.Background(), CronJob{
		JobID:       "job-iso-1",
		Name:        "isolated-job",
		AssistantID: "assistant-iso",
		SessionKey:  "cron/isolated",
		Enabled:     true,
		Schedule: CronSchedule{
			Kind:    "every",
			EveryMs: 60_000,
		},
		SessionTarget: "isolated",
		PayloadSpec: CronPayload{
			Kind:           "agentTurn",
			Message:        "say hi",
			Model:          "openai/gpt-4.1",
			Thinking:       "low",
			TimeoutSeconds: 30,
		},
	})
	if err != nil {
		t.Fatalf("executeJob error: %v", err)
	}
	if run.Status != "completed" {
		t.Fatalf("expected completed status, got %q", run.Status)
	}
	if run.Summary != "isolated done" {
		t.Fatalf("unexpected summary: %q", run.Summary)
	}
	if run.Provider != "openai" || run.Model != "gpt-4.1" {
		t.Fatalf("unexpected model/provider: %q/%q", run.Provider, run.Model)
	}
	if captured.JobID != "job-iso-1" {
		t.Fatalf("unexpected isolated request job id: %q", captured.JobID)
	}
	if captured.Message != "say hi" {
		t.Fatalf("unexpected isolated request message: %q", captured.Message)
	}
	if captured.TimeoutSeconds != 30 {
		t.Fatalf("unexpected isolated timeout seconds: %d", captured.TimeoutSeconds)
	}
}

func TestApplyRunDeliveryWebhookDelivered(t *testing.T) {
	scheduler := NewScheduler(nil, nil)
	var captured WebhookRequest
	scheduler.SetWebhookSender(func(_ context.Context, request WebhookRequest) error {
		captured = request
		return nil
	})

	run := CronRunRecord{
		RunID:      "run-webhook-1",
		JobID:      "job-webhook-1",
		JobName:    "cron-webhook",
		Status:     "completed",
		Summary:    "all good",
		StartedAt:  time.UnixMilli(1000),
		EndedAt:    time.UnixMilli(2000),
		SessionKey: "cron/isolated",
	}
	job := CronJob{
		JobID:       "job-webhook-1",
		Name:        "cron-webhook",
		AssistantID: "assistant-1",
		PayloadSpec: CronPayload{
			Kind:    "agentTurn",
			Message: "notify",
		},
		Delivery: &CronDelivery{
			Mode: "webhook",
			To:   "https://example.com/webhook",
		},
	}

	scheduler.applyRunDelivery(context.Background(), job, &run)
	if run.DeliveryStatus != "delivered" {
		t.Fatalf("expected delivered, got %q (err=%q)", run.DeliveryStatus, run.DeliveryError)
	}
	if captured.URL != "https://example.com/webhook" {
		t.Fatalf("unexpected webhook url: %q", captured.URL)
	}
	if captured.Status != "completed" {
		t.Fatalf("unexpected webhook status: %q", captured.Status)
	}
}

func TestApplyRunDeliveryFailureDestinationAnnounceOnFailedRun(t *testing.T) {
	scheduler := NewScheduler(nil, nil)
	var delivered AnnouncementRequest
	scheduler.SetAnnouncementSender(func(_ context.Context, request AnnouncementRequest) error {
		delivered = request
		return nil
	})

	run := CronRunRecord{
		RunID:      "run-failure-destination-1",
		JobID:      "job-failure-destination-1",
		JobName:    "cron-failure-destination",
		Status:     "failed",
		Error:      "primary run failed",
		StartedAt:  time.UnixMilli(1000),
		EndedAt:    time.UnixMilli(2000),
		SessionKey: "cron/isolated",
	}
	job := CronJob{
		JobID:       "job-failure-destination-1",
		Name:        "cron-failure-destination",
		AssistantID: "assistant-1",
		PayloadSpec: CronPayload{
			Kind:    "agentTurn",
			Message: "notify on failed run",
		},
		Delivery: &CronDelivery{
			Mode: "none",
			FailureDestination: &CronFailureDestination{
				Mode:    "announce",
				Channel: "app",
			},
		},
	}

	scheduler.applyRunDelivery(context.Background(), job, &run)
	if delivered.Channel != "app" {
		t.Fatalf("expected failure destination channel app, got %q", delivered.Channel)
	}
	if delivered.Message != "primary run failed" {
		t.Fatalf("expected failure destination message from error, got %q", delivered.Message)
	}
	if run.DeliveryStatus != "delivered" {
		t.Fatalf("expected delivery status delivered after failure destination, got %q", run.DeliveryStatus)
	}
}

func TestApplyRunDeliverySkipsFailureDestinationWhenBestEffort(t *testing.T) {
	scheduler := NewScheduler(nil, nil)
	called := false
	scheduler.SetAnnouncementSender(func(_ context.Context, request AnnouncementRequest) error {
		called = true
		_ = request
		return nil
	})

	run := CronRunRecord{
		RunID:      "run-failure-destination-2",
		JobID:      "job-failure-destination-2",
		JobName:    "cron-failure-destination-best-effort",
		Status:     "failed",
		Error:      "primary run failed",
		StartedAt:  time.UnixMilli(1000),
		EndedAt:    time.UnixMilli(2000),
		SessionKey: "cron/isolated",
	}
	job := CronJob{
		JobID:       "job-failure-destination-2",
		Name:        "cron-failure-destination-best-effort",
		AssistantID: "assistant-1",
		PayloadSpec: CronPayload{
			Kind:    "agentTurn",
			Message: "notify on failed run",
		},
		Delivery: &CronDelivery{
			Mode:       "none",
			BestEffort: true,
			FailureDestination: &CronFailureDestination{
				Mode:    "announce",
				Channel: "app",
			},
		},
	}

	scheduler.applyRunDelivery(context.Background(), job, &run)
	if called {
		t.Fatalf("expected best-effort configuration to skip failure destination delivery")
	}
}
