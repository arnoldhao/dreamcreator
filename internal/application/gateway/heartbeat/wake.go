package heartbeat

import (
	"context"
	"sort"
	"strings"
	"time"
)

const (
	defaultWakeCoalesce = 250 * time.Millisecond
	defaultWakeRetry    = 1 * time.Second
)

type wakeTimerKind string

const (
	wakeTimerNormal wakeTimerKind = "normal"
	wakeTimerRetry  wakeTimerKind = "retry"
)

type pendingWake struct {
	input       TriggerInput
	priority    int
	requestedAt time.Time
}

func (service *Service) enqueueWake(_ context.Context, input TriggerInput) TriggerResult {
	if service == nil {
		return TriggerResult{
			Accepted:       false,
			ExecutedStatus: TriggerExecutionSkipped,
			Reason:         "unavailable",
		}
	}
	normalizedReason := normalizeWakeReason(input.Reason)
	input.Reason = normalizedReason
	requestedAt := service.now()

	service.wakeMu.Lock()
	service.queuePendingWakeLocked(input, requestedAt)
	service.scheduleWakeLocked(defaultWakeCoalesce, wakeTimerNormal)
	service.wakeMu.Unlock()

	return TriggerResult{
		Accepted:       true,
		ExecutedStatus: TriggerExecutionQueued,
		Reason:         normalizedReason,
	}
}

func (service *Service) queuePendingWakeLocked(input TriggerInput, requestedAt time.Time) {
	key := wakeTargetKey(input)
	next := pendingWake{
		input:       input,
		priority:    resolveWakePriority(input.Reason),
		requestedAt: requestedAt,
	}
	previous, exists := service.wakePending[key]
	if !exists {
		service.wakePending[key] = next
		return
	}
	if next.priority > previous.priority {
		service.wakePending[key] = next
		return
	}
	if next.priority == previous.priority && (next.requestedAt.After(previous.requestedAt) || next.requestedAt.Equal(previous.requestedAt)) {
		service.wakePending[key] = next
	}
}

func (service *Service) scheduleWakeLocked(delay time.Duration, kind wakeTimerKind) {
	effectiveDelay := delay
	if effectiveDelay < 0 {
		effectiveDelay = 0
	}
	dueAt := service.now().Add(effectiveDelay)
	if service.wakeTimer != nil {
		if service.wakeTimerKind == wakeTimerRetry {
			return
		}
		if !service.wakeDueAt.IsZero() && (service.wakeDueAt.Before(dueAt) || service.wakeDueAt.Equal(dueAt)) {
			return
		}
		service.wakeTimer.Stop()
		service.wakeTimer = nil
		service.wakeDueAt = time.Time{}
		service.wakeTimerKind = ""
	}

	service.wakeDueAt = dueAt
	service.wakeTimerKind = kind
	service.wakeTimer = time.AfterFunc(effectiveDelay, func() {
		service.processWakeTimer(effectiveDelay, kind)
	})
}

func (service *Service) processWakeTimer(delay time.Duration, kind wakeTimerKind) {
	service.wakeMu.Lock()
	service.wakeTimer = nil
	service.wakeDueAt = time.Time{}
	service.wakeTimerKind = ""
	if service.wakeRunning {
		service.wakeScheduled = true
		service.scheduleWakeLocked(delay, kind)
		service.wakeMu.Unlock()
		return
	}

	pendingBatch := make([]pendingWake, 0, len(service.wakePending))
	for _, pending := range service.wakePending {
		pendingBatch = append(pendingBatch, pending)
	}
	service.wakePending = make(map[string]pendingWake)
	service.wakeRunning = true
	service.wakeMu.Unlock()

	sort.Slice(pendingBatch, func(i, j int) bool {
		if pendingBatch[i].priority == pendingBatch[j].priority {
			return pendingBatch[i].requestedAt.Before(pendingBatch[j].requestedAt)
		}
		return pendingBatch[i].priority > pendingBatch[j].priority
	})

	for _, pending := range pendingBatch {
		result := service.run(context.Background(), pending.input)
		if result.ExecutedStatus == TriggerExecutionSkipped && strings.EqualFold(strings.TrimSpace(result.Reason), "requests-in-flight") {
			retryInput := pending.input
			retryInput.Reason = normalizeWakeReason(retryInput.Reason)
			service.wakeMu.Lock()
			service.queuePendingWakeLocked(retryInput, service.now())
			service.scheduleWakeLocked(defaultWakeRetry, wakeTimerRetry)
			service.wakeMu.Unlock()
		}
	}

	service.wakeMu.Lock()
	service.wakeRunning = false
	needsReschedule := len(service.wakePending) > 0 || service.wakeScheduled
	service.wakeScheduled = false
	if needsReschedule {
		service.scheduleWakeLocked(delay, wakeTimerNormal)
	}
	service.wakeMu.Unlock()
}

func wakeTargetKey(input TriggerInput) string {
	key := strings.TrimSpace(input.SessionKey)
	if key == "" {
		return "__default__"
	}
	return key
}

func normalizeWakeReason(reason string) string {
	trimmed := strings.TrimSpace(reason)
	if trimmed == "" {
		return "manual"
	}
	return trimmed
}

func resolveWakePriority(reason string) int {
	normalized := strings.ToLower(strings.TrimSpace(reason))
	switch {
	case strings.Contains(normalized, "retry"):
		return 0
	case normalized == "interval":
		return 1
	case isForceReason(normalized):
		return 3
	default:
		return 2
	}
}
