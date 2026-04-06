package heartbeat

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"dreamcreator/internal/application/chatevent"
	"dreamcreator/internal/application/gateway/events"
	"dreamcreator/internal/application/gateway/runtime/dto"
	appnotice "dreamcreator/internal/application/notice"
	settingsdto "dreamcreator/internal/application/settings/dto"
	settingsservice "dreamcreator/internal/application/settings/service"
	threaddto "dreamcreator/internal/application/thread/dto"
	threadservice "dreamcreator/internal/application/thread/service"
	workspacedto "dreamcreator/internal/application/workspace/dto"
	domainnotice "dreamcreator/internal/domain/notice"
	"dreamcreator/internal/domain/session"
	"dreamcreator/internal/domain/thread"
)

type RuntimeRunner interface {
	Run(ctx context.Context, request dto.RuntimeRunRequest) (dto.RuntimeRunResult, error)
}

const (
	heartbeatToken         = "HEARTBEAT_OK"
	defaultHeartbeatPrompt = "Review only pending heartbeat checklist items and pending system events. Do not infer or repeat old chat tasks. If nothing needs attention, reply exactly HEARTBEAT_OK. If something needs attention, return only compact JSON with code, severity, params, and optional action."
)

type ThreadWriter interface {
	AppendMessage(ctx context.Context, request threaddto.AppendMessageRequest) error
}

type BusyCheckFunc func(ctx context.Context) (bool, string)

type WorkspaceSnapshotResolver interface {
	ResolveRuntimeSnapshot(ctx context.Context, request workspacedto.ResolveRuntimeSnapshotRequest) (workspacedto.RuntimeSnapshot, error)
}

type ChannelReadyCheckFunc func(ctx context.Context, channelID string, accountID string) (bool, string)

type NoticePublisher interface {
	Create(ctx context.Context, input appnotice.CreateNoticeInput) (domainnotice.Notice, error)
}

type StoreOptions struct {
	EventStore        EventStore
	BusyCheck         BusyCheckFunc
	WorkspaceResolver WorkspaceSnapshotResolver
	ChannelReadyCheck ChannelReadyCheckFunc
}

type Service struct {
	settings *settingsservice.SettingsService
	threads  thread.Repository
	writer   ThreadWriter
	runtime  RuntimeRunner
	events   *events.Broker
	notices  NoticePublisher

	eventStore           EventStore
	persistentEventStore EventStore

	busyCheck         BusyCheckFunc
	workspaceResolver WorkspaceSnapshotResolver
	channelReadyCheck ChannelReadyCheckFunc

	now          func() time.Time
	newID        func() string
	systemEvents *SystemEventQueue

	mu              sync.Mutex
	running         bool
	lastTick        time.Time
	enabledOverride *bool

	wakeMu        sync.Mutex
	wakePending   map[string]pendingWake
	wakeRunning   bool
	wakeScheduled bool
	wakeTimer     *time.Timer
	wakeDueAt     time.Time
	wakeTimerKind wakeTimerKind
}

func NewService(
	settings *settingsservice.SettingsService,
	threads thread.Repository,
	writer *threadservice.ThreadService,
	runtime RuntimeRunner,
	stores StoreOptions,
	events *events.Broker,
	notices NoticePublisher,
) *Service {
	return &Service{
		settings:             settings,
		threads:              threads,
		writer:               writer,
		runtime:              runtime,
		events:               events,
		notices:              notices,
		eventStore:           NewMemoryEventStore(),
		persistentEventStore: stores.EventStore,
		busyCheck:            stores.BusyCheck,
		workspaceResolver:    stores.WorkspaceResolver,
		channelReadyCheck:    stores.ChannelReadyCheck,
		now:                  time.Now,
		newID:                func() string { return hex.EncodeToString([]byte(time.Now().Format("20060102150405.000000000"))) },
		systemEvents:         NewSystemEventQueue(),
		wakePending:          make(map[string]pendingWake),
	}
}

func (service *Service) Tick(ctx context.Context) {
	if service == nil {
		return
	}
	_ = service.TriggerWithResult(ctx, TriggerInput{Reason: "interval"})
}

func (service *Service) Trigger(ctx context.Context, reason string) {
	if service == nil {
		return
	}
	_ = service.TriggerWithResult(ctx, TriggerInput{Reason: reason, Force: true})
}

func (service *Service) TriggerWithInput(ctx context.Context, input TriggerInput) bool {
	if service == nil {
		return false
	}
	return service.TriggerWithResult(ctx, input).Accepted
}

func (service *Service) TriggerWithResult(ctx context.Context, input TriggerInput) TriggerResult {
	if service == nil {
		return TriggerResult{
			Accepted:       false,
			ExecutedStatus: TriggerExecutionSkipped,
			Reason:         "unavailable",
		}
	}
	return service.enqueueWake(ctx, input)
}

func (service *Service) SetEnabled(enabled bool) {
	if service == nil {
		return
	}
	service.mu.Lock()
	service.enabledOverride = &enabled
	service.mu.Unlock()
}

func (service *Service) ClearEnabledOverride() {
	if service == nil {
		return
	}
	service.mu.Lock()
	service.enabledOverride = nil
	service.mu.Unlock()
}

func (service *Service) SetChannelReadyCheck(check ChannelReadyCheckFunc) {
	if service == nil {
		return
	}
	service.mu.Lock()
	service.channelReadyCheck = check
	service.mu.Unlock()
}

func (service *Service) Last(ctx context.Context, sessionKey string) (Event, error) {
	if service == nil {
		return Event{}, errors.New("heartbeat store unavailable")
	}
	key := strings.TrimSpace(sessionKey)
	if key == "" {
		return Event{}, errors.New("session key is required")
	}
	event, err := service.eventStore.Last(ctx, key)
	if err == nil {
		return event, nil
	}
	if service.persistentEventStore == nil {
		return Event{}, err
	}
	persistent, persistentErr := service.persistentEventStore.Last(ctx, key)
	if persistentErr != nil {
		return Event{}, persistentErr
	}
	_ = service.eventStore.Save(ctx, persistent)
	return persistent, nil
}

func (service *Service) GetSpec(ctx context.Context) (*Spec, error) {
	if service == nil {
		return nil, errors.New("heartbeat checklist unavailable")
	}
	if service.settings == nil {
		return nil, errors.New("heartbeat checklist unavailable")
	}
	current, err := service.settings.GetSettings(ctx)
	if err != nil {
		return nil, err
	}
	spec, ok := specFromGatewayChecklist(current.Gateway.Heartbeat.Checklist)
	if !ok {
		return nil, nil
	}
	return &spec, nil
}

func (service *Service) EnqueueSystemEvent(_ context.Context, input SystemEventInput) bool {
	if service == nil || service.systemEvents == nil {
		return false
	}
	return service.systemEvents.Enqueue(input)
}

func (service *Service) run(ctx context.Context, input TriggerInput) TriggerResult {
	if service == nil {
		return TriggerResult{
			Accepted:       false,
			ExecutedStatus: TriggerExecutionSkipped,
			Reason:         "unavailable",
		}
	}
	cfg, ok := service.loadConfig(ctx)
	if !ok {
		return TriggerResult{
			Accepted:       true,
			ExecutedStatus: TriggerExecutionSkipped,
			Reason:         "config_unavailable",
		}
	}
	if !service.isEnabled(cfg) || !cfg.Enabled {
		return TriggerResult{
			Accepted:       true,
			ExecutedStatus: TriggerExecutionSkipped,
			Reason:         "disabled",
		}
	}
	reason := strings.TrimSpace(input.Reason)
	force := input.Force
	if isForceReason(reason) {
		force = true
	}
	targetSession := strings.TrimSpace(input.SessionKey)
	if targetSession == "" {
		targetSession = strings.TrimSpace(cfg.RunSession)
	}
	if targetSession == "" {
		targetSession = strings.TrimSpace(cfg.Session)
	}
	if targetSession == "" {
		service.publishEvent(ctx, Event{
			ID:        service.newID(),
			Status:    StatusSkipped,
			Message:   "run_session_unset",
			Reason:    reason,
			CreatedAt: service.now(),
		})
		return TriggerResult{
			Accepted:       true,
			ExecutedStatus: TriggerExecutionSkipped,
			Reason:         "run_session_unset",
		}
	}

	threadItem, sessionKey, err := service.resolveThreadBySessionKey(ctx, targetSession)
	if err != nil || threadItem.ID == "" {
		service.publishEvent(ctx, Event{
			ID:        service.newID(),
			Status:    StatusFailed,
			Error:     errString(err, "heartbeat target unavailable"),
			Reason:    reason,
			CreatedAt: service.now(),
			Indicator: IndicatorError,
		})
		return TriggerResult{
			Accepted:       true,
			ExecutedStatus: TriggerExecutionFailed,
			Reason:         "target_unavailable",
		}
	}
	if service.systemEvents != nil && service.systemEvents.Has(sessionKey) {
		force = true
	}
	if routeReady, routeReason := service.isRouteReady(ctx, sessionKey, cfg); !routeReady {
		service.publishEvent(ctx, Event{
			ID:         service.newID(),
			SessionKey: sessionKey,
			ThreadID:   threadItem.ID,
			Status:     StatusSkipped,
			Message:    routeReason,
			Reason:     reason,
			CreatedAt:  service.now(),
		})
		return TriggerResult{
			Accepted:       true,
			ExecutedStatus: TriggerExecutionSkipped,
			Reason:         routeReason,
		}
	}

	if !force && !service.withinActiveHours(cfg.ActiveHours) {
		service.publishEvent(ctx, Event{
			ID:         service.newID(),
			SessionKey: sessionKey,
			ThreadID:   threadItem.ID,
			Status:     StatusSkipped,
			Message:    "outside_active_hours",
			Reason:     reason,
			CreatedAt:  service.now(),
		})
		return TriggerResult{
			Accepted:       true,
			ExecutedStatus: TriggerExecutionSkipped,
			Reason:         "outside_active_hours",
		}
	}
	if busy, busyReason := service.isBusy(ctx); busy {
		if busyReason == "" {
			busyReason = "requests-in-flight"
		}
		service.publishEvent(ctx, Event{
			ID:         service.newID(),
			SessionKey: sessionKey,
			ThreadID:   threadItem.ID,
			Status:     StatusSkipped,
			Message:    busyReason,
			Reason:     reason,
			Source:     strings.TrimSpace(input.Source),
			RunID:      strings.TrimSpace(input.RunID),
			CreatedAt:  service.now(),
		})
		return TriggerResult{
			Accepted:       true,
			ExecutedStatus: TriggerExecutionSkipped,
			Reason:         busyReason,
		}
	}
	started, skipReason := service.beginRun(cfg, force)
	if !started {
		return TriggerResult{
			Accepted:       true,
			ExecutedStatus: TriggerExecutionSkipped,
			Reason:         skipReason,
		}
	}
	defer service.finishRun()

	prompt := service.resolveHeartbeatPrompt(ctx, cfg)
	pending := []SystemEvent{}
	if service.systemEvents != nil {
		pending = service.systemEvents.Drain(sessionKey)
	}
	if len(pending) > 0 {
		force = true
	}
	eventDriven := len(pending) > 0 || isCronReason(reason) || isExecReason(reason) || strings.Contains(strings.ToLower(reason), "subagent")
	if shouldShortCircuitHeartbeatOK(cfg, pending, eventDriven) {
		status := StatusOKEmpty
		service.publishEvent(ctx, Event{
			ID:         service.newID(),
			SessionKey: sessionKey,
			ThreadID:   threadItem.ID,
			Status:     status,
			Reason:     reason,
			CreatedAt:  service.now(),
			Silent:     true,
			Indicator:  resolveIndicatorStatus(status, true),
		})
		return TriggerResult{
			Accepted:       true,
			ExecutedStatus: TriggerExecutionRan,
			Reason:         string(status),
		}
	}
	replyInline := shouldReplyInThread(cfg)
	canRelayToUser := eventDriven && replyInline
	basePrompt := resolveHeartbeatPrompt(prompt)
	hasExec := isExecReason(reason)
	cronReason := isCronReason(reason)
	cronEvents := make([]string, 0)
	hasCronEvents := false
	hasExecEvents := false
	for _, event := range pending {
		text := strings.TrimSpace(event.Text)
		if text == "" {
			continue
		}
		contextKey := strings.ToLower(strings.TrimSpace(event.ContextKey))
		if strings.HasPrefix(contextKey, "exec:") || strings.EqualFold(event.Source, "exec") || isExecCompletionEvent(text) {
			hasExecEvents = true
		}
		if cronReason || strings.HasPrefix(contextKey, "cron:") || strings.EqualFold(event.Source, "cron") {
			if isCronSystemEvent(text) {
				cronEvents = append(cronEvents, text)
				hasCronEvents = true
			}
		}
	}
	if hasExec || hasExecEvents {
		basePrompt = buildExecEventPrompt(canRelayToUser)
	} else if hasCronEvents {
		basePrompt = buildCronEventPrompt(cronEvents, canRelayToUser)
	}
	if len(pending) > 0 && (hasExec || hasExecEvents || !hasCronEvents) {
		basePrompt = appendSystemEvents(basePrompt, pending)
	}
	prompt = appendCurrentTime(basePrompt, service.now())

	response, err := service.runHeartbeat(ctx, threadItem, sessionKey, prompt, cfg)
	eventSource, eventRunID := summarizeEventContext(pending)
	if err != nil {
		service.publishNotice(ctx, threadItem, sessionKey, reason, eventSource, eventRunID, strings.TrimSpace(err.Error()), heartbeatAlert{
			Code:     "heartbeat.runtime_failed",
			Severity: domainnotice.SeverityError,
			Params: map[string]string{
				"detail": strings.TrimSpace(err.Error()),
			},
		}, cfg, eventDriven)
		service.publishEvent(ctx, Event{
			ID:         service.newID(),
			SessionKey: sessionKey,
			ThreadID:   threadItem.ID,
			Status:     StatusFailed,
			Error:      err.Error(),
			Reason:     reason,
			Source:     eventSource,
			RunID:      eventRunID,
			CreatedAt:  service.now(),
			Indicator:  IndicatorError,
		})
		return TriggerResult{
			Accepted:       true,
			ExecutedStatus: TriggerExecutionFailed,
			Reason:         strings.TrimSpace(err.Error()),
		}
	}

	parsed := parseHeartbeatResponse(response)
	contentHash := hashContent(parsed.Cleaned)
	if parsed.Ack {
		status := StatusOKToken
		if strings.TrimSpace(parsed.Cleaned) == "" {
			status = StatusOKEmpty
		}
		service.publishEvent(ctx, Event{
			ID:          service.newID(),
			SessionKey:  sessionKey,
			ThreadID:    threadItem.ID,
			Status:      status,
			Message:     parsed.Cleaned,
			ContentHash: contentHash,
			Reason:      reason,
			Source:      eventSource,
			RunID:       eventRunID,
			CreatedAt:   service.now(),
			Silent:      true,
			Indicator:   resolveIndicatorStatus(status, true),
		})
		return TriggerResult{
			Accepted:       true,
			ExecutedStatus: TriggerExecutionRan,
			Reason:         string(status),
		}
	}
	cleaned := strings.TrimSpace(parsed.Cleaned)
	if cleaned == "" {
		cleaned = "heartbeat attention required"
	}
	service.publishNotice(ctx, threadItem, sessionKey, reason, eventSource, eventRunID, cleaned, parsed.Alert, cfg, eventDriven)
	if replyInline && service.writer != nil {
		parts := buildHeartbeatAssistantMessageParts(cleaned, reason, eventSource, eventRunID, hasCronEvents, hasExecEvents)
		_ = service.writer.AppendMessage(ctx, threaddto.AppendMessageRequest{
			ThreadID: threadItem.ID,
			Role:     "assistant",
			Content:  cleaned,
			Parts:    parts,
		})
	}

	service.publishEvent(ctx, Event{
		ID:          service.newID(),
		SessionKey:  sessionKey,
		ThreadID:    threadItem.ID,
		Status:      StatusSent,
		Message:     cleaned,
		ContentHash: contentHash,
		Reason:      reason,
		Source:      eventSource,
		RunID:       eventRunID,
		CreatedAt:   service.now(),
		Indicator:   resolveIndicatorStatus(StatusSent, true),
	})
	return TriggerResult{
		Accepted:       true,
		ExecutedStatus: TriggerExecutionRan,
		Reason:         "sent",
	}
}

func (service *Service) loadConfig(ctx context.Context) (settingsdto.GatewayHeartbeatSettings, bool) {
	if service.settings == nil {
		return settingsdto.GatewayHeartbeatSettings{}, false
	}
	current, err := service.settings.GetSettings(ctx)
	if err != nil {
		return settingsdto.GatewayHeartbeatSettings{}, false
	}
	return current.Gateway.Heartbeat, true
}

func (service *Service) isEnabled(cfg settingsdto.GatewayHeartbeatSettings) bool {
	service.mu.Lock()
	defer service.mu.Unlock()
	if service.enabledOverride == nil {
		return true
	}
	return *service.enabledOverride
}

func (service *Service) beginRun(cfg settingsdto.GatewayHeartbeatSettings, force bool) (bool, string) {
	service.mu.Lock()
	defer service.mu.Unlock()
	if service.running {
		return false, "running"
	}
	now := service.now()
	interval := resolveHeartbeatInterval(cfg)
	if interval <= 0 && !force {
		return false, "interval_disabled"
	}
	if !service.lastTick.IsZero() && now.Sub(service.lastTick) < interval {
		if !force {
			return false, "not_due"
		}
	}
	service.running = true
	service.lastTick = now
	return true, ""
}

func (service *Service) finishRun() {
	if service == nil {
		return
	}
	service.mu.Lock()
	service.running = false
	service.mu.Unlock()
}

func (service *Service) isBusy(ctx context.Context) (bool, string) {
	if service == nil || service.busyCheck == nil {
		return false, ""
	}
	busy, reason := service.busyCheck(ctx)
	return busy, strings.TrimSpace(reason)
}

func (service *Service) shouldSkipForEmptyHeartbeatFile(ctx context.Context, threadItem thread.Thread) bool {
	if service == nil || service.workspaceResolver == nil {
		return false
	}
	assistantID := strings.TrimSpace(threadItem.AssistantID)
	if assistantID == "" {
		return false
	}
	snapshot, err := service.workspaceResolver.ResolveRuntimeSnapshot(ctx, workspacedto.ResolveRuntimeSnapshotRequest{
		AssistantID: assistantID,
		ThreadID:    strings.TrimSpace(threadItem.ID),
	})
	if err != nil {
		return false
	}
	rootPath := strings.TrimSpace(snapshot.RootPath)
	if rootPath == "" {
		return false
	}
	heartbeatPath := filepath.Join(rootPath, "HEARTBEAT.md")
	content, readErr := os.ReadFile(heartbeatPath)
	if readErr != nil {
		return false
	}
	return isHeartbeatContentEffectivelyEmpty(string(content))
}

func (service *Service) isRouteReady(ctx context.Context, sessionKey string, cfg settingsdto.GatewayHeartbeatSettings) (bool, string) {
	if service == nil {
		return true, ""
	}
	hasExplicitRoute := strings.TrimSpace(cfg.To) != "" || strings.TrimSpace(cfg.AccountID) != ""
	parts, _, err := session.NormalizeSessionKey(strings.TrimSpace(sessionKey))
	if err != nil {
		if hasExplicitRoute {
			return false, "route_unavailable"
		}
		return true, ""
	}
	channelID := strings.TrimSpace(parts.Channel)
	if channelID == "" {
		if hasExplicitRoute {
			return false, "route_unavailable"
		}
		return true, ""
	}
	check := service.currentChannelReadyCheck()
	if check != nil {
		accountID := strings.TrimSpace(cfg.AccountID)
		if accountID == "" {
			accountID = strings.TrimSpace(parts.AccountID)
		}
		ready, reason := check(ctx, channelID, accountID)
		if !ready {
			trimmedReason := strings.TrimSpace(reason)
			if trimmedReason == "" {
				trimmedReason = "channel_not_ready"
			}
			return false, trimmedReason
		}
	}
	return true, ""
}

func (service *Service) currentChannelReadyCheck() ChannelReadyCheckFunc {
	if service == nil {
		return nil
	}
	service.mu.Lock()
	defer service.mu.Unlock()
	return service.channelReadyCheck
}

func (service *Service) withinActiveHours(cfg settingsdto.GatewayHeartbeatActiveHours) bool {
	start := strings.TrimSpace(cfg.Start)
	end := strings.TrimSpace(cfg.End)
	if start == "" && end == "" {
		return true
	}
	location := time.Local
	if tz := strings.TrimSpace(cfg.Timezone); tz != "" {
		if loaded, err := time.LoadLocation(tz); err == nil {
			location = loaded
		}
	}
	now := service.now().In(location)
	startMinutes, startOk := parseClockMinutes(start, false)
	endMinutes, endOk := parseClockMinutes(end, true)
	if !startOk || !endOk {
		return true
	}
	if startMinutes == endMinutes {
		return false
	}

	nowMinutes := now.Hour()*60 + now.Minute()
	if endMinutes > startMinutes {
		return nowMinutes >= startMinutes && nowMinutes < endMinutes
	}
	return nowMinutes >= startMinutes || nowMinutes < endMinutes
}

func parseClockMinutes(value string, allow24 bool) (int, bool) {
	if strings.TrimSpace(value) == "" {
		return 0, false
	}
	if allow24 && strings.TrimSpace(value) == "24:00" {
		return 24 * 60, true
	}
	parsed, err := time.Parse("15:04", value)
	if err != nil {
		return 0, false
	}
	return parsed.Hour()*60 + parsed.Minute(), true
}

func (service *Service) resolveThreadBySessionKey(ctx context.Context, key string) (thread.Thread, string, error) {
	trimmed := strings.TrimSpace(key)
	if trimmed == "" {
		return thread.Thread{}, "", errors.New("session key is required")
	}
	threadID := trimmed
	sessionKey := trimmed
	if parts, normalized, err := session.NormalizeSessionKey(trimmed); err == nil {
		sessionKey = normalized
		if strings.TrimSpace(parts.ThreadRef) != "" {
			threadID = strings.TrimSpace(parts.ThreadRef)
		} else if strings.TrimSpace(parts.PrimaryID) != "" {
			threadID = strings.TrimSpace(parts.PrimaryID)
		} else {
			threadID = strings.TrimSpace(normalized)
		}
	}
	item, err := service.threads.Get(ctx, threadID)
	if err != nil {
		return thread.Thread{}, "", err
	}
	return item, sessionKey, nil
}

func (service *Service) resolveHeartbeatPrompt(_ context.Context, cfg settingsdto.GatewayHeartbeatSettings) string {
	spec, ok := specFromGatewayChecklist(cfg.Checklist)
	if ok {
		if prompt := formatSpecPrompt(spec, cfg); prompt != "" {
			return prompt
		}
	}
	if strings.TrimSpace(cfg.PromptAppend) != "" {
		return joinHeartbeatLines([]string{defaultHeartbeatPrompt, cfg.PromptAppend})
	}
	if strings.TrimSpace(cfg.Prompt) != "" {
		return joinHeartbeatLines([]string{defaultHeartbeatPrompt, cfg.Prompt})
	}
	return defaultHeartbeatPrompt
}

func shouldShortCircuitHeartbeatOK(cfg settingsdto.GatewayHeartbeatSettings, pending []SystemEvent, eventDriven bool) bool {
	if eventDriven || len(pending) > 0 {
		return false
	}
	if strings.TrimSpace(cfg.PromptAppend) != "" || strings.TrimSpace(cfg.Prompt) != "" {
		return false
	}
	spec, ok := specFromGatewayChecklist(cfg.Checklist)
	if !ok {
		return true
	}
	if strings.TrimSpace(spec.Notes) != "" {
		return false
	}
	for _, item := range spec.Items {
		if strings.TrimSpace(item.Text) == "" {
			continue
		}
		if !item.Done {
			return false
		}
	}
	return true
}

func formatSpecPrompt(spec Spec, cfg settingsdto.GatewayHeartbeatSettings) string {
	lines := []string{
		"Heartbeat checklist:",
		defaultHeartbeatPrompt,
	}
	if title := strings.TrimSpace(spec.Title); title != "" {
		lines = append(lines, "Title: "+title)
	}
	if len(spec.Items) > 0 {
		lines = append(lines, "Items:")
		for _, item := range spec.Items {
			text := strings.TrimSpace(item.Text)
			if text == "" {
				continue
			}
			status := "[ ]"
			if item.Done {
				status = "[x]"
			}
			line := fmt.Sprintf("- %s %s", status, text)
			if priority := strings.TrimSpace(item.Priority); priority != "" {
				line += fmt.Sprintf(" (priority: %s)", priority)
			}
			lines = append(lines, line)
		}
	}
	if notes := strings.TrimSpace(spec.Notes); notes != "" {
		lines = append(lines, "Notes:")
		lines = append(lines, notes)
	}
	if appendPrompt := strings.TrimSpace(cfg.PromptAppend); appendPrompt != "" {
		lines = append(lines, appendPrompt)
	} else if legacyPrompt := strings.TrimSpace(cfg.Prompt); legacyPrompt != "" {
		lines = append(lines, legacyPrompt)
	}
	return joinHeartbeatLines(lines)
}

func joinHeartbeatLines(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	cleaned := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		cleaned = append(cleaned, trimmed)
	}
	return strings.TrimSpace(strings.Join(cleaned, "\n"))
}

func (service *Service) runHeartbeat(ctx context.Context, threadItem thread.Thread, sessionKey string, prompt string, cfg settingsdto.GatewayHeartbeatSettings) (string, error) {
	if service.runtime == nil {
		return "", errors.New("runtime unavailable")
	}
	request := dto.RuntimeRunRequest{
		SessionID:   threadItem.ID,
		SessionKey:  sessionKey,
		AssistantID: strings.TrimSpace(threadItem.AssistantID),
		PromptMode:  "minimal",
		RunKind:     "heartbeat",
		Input: dto.RuntimeInput{
			Messages: []dto.Message{{
				Role:    "user",
				Content: prompt,
			}},
			ReplaceHistory: false,
		},
		Metadata: map[string]any{
			"persistRun":                false,
			"persistMessages":           false,
			"persistEvents":             false,
			"useQueue":                  false,
			"heartbeat":                 true,
			"runKind":                   "heartbeat",
			"to":                        strings.TrimSpace(cfg.To),
			"accountId":                 strings.TrimSpace(cfg.AccountID),
			"includeReasoning":          cfg.IncludeReasoning,
			"suppressToolErrorWarnings": cfg.SuppressToolErrorWarnings,
		},
	}
	if modelRef := strings.TrimSpace(cfg.Model); modelRef != "" {
		if providerID, modelName := splitModelRef(modelRef); providerID != "" && modelName != "" {
			request.Model = &dto.ModelSelection{
				ProviderID: providerID,
				Name:       modelName,
			}
		}
	}
	result, err := service.runtime.Run(ctx, request)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(result.AssistantMessage.Content), nil
}

func (service *Service) publishEvent(ctx context.Context, event Event) {
	if service == nil {
		return
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = service.now()
	}
	_ = service.eventStore.Save(ctx, event)
	if service.persistentEventStore != nil {
		_ = service.persistentEventStore.Save(ctx, event)
	}
	if service.events == nil {
		return
	}
	envelope := events.Envelope{
		Type:       "heartbeat.event",
		Topic:      "heartbeat",
		SessionKey: strings.TrimSpace(event.SessionKey),
		Timestamp:  event.CreatedAt,
	}
	if envelope.SessionKey != "" {
		if parts, _, err := session.NormalizeSessionKey(envelope.SessionKey); err == nil {
			envelope.SessionID = strings.TrimSpace(parts.ThreadRef)
		}
	}
	payload := map[string]any{
		"status":     event.Status,
		"message":    event.Message,
		"error":      event.Error,
		"sessionKey": event.SessionKey,
		"threadId":   event.ThreadID,
		"timestamp":  event.CreatedAt.Format(time.RFC3339),
		"silent":     event.Silent,
	}
	if indicator := strings.TrimSpace(string(event.Indicator)); indicator != "" {
		payload["indicatorType"] = indicator
	}
	contextPayload := map[string]any{}
	if reason := strings.TrimSpace(event.Reason); reason != "" {
		contextPayload["reason"] = reason
	}
	if source := strings.TrimSpace(event.Source); source != "" {
		contextPayload["source"] = source
	}
	if runID := strings.TrimSpace(event.RunID); runID != "" {
		contextPayload["runId"] = runID
	}
	if len(contextPayload) > 0 {
		payload["context"] = contextPayload
	}
	_, _ = service.events.Publish(ctx, envelope, payload)
}

func (service *Service) hasDuplicate(ctx context.Context, sessionKey string, contentHash string, since time.Time) (bool, error) {
	duplicate, err := service.eventStore.HasDuplicate(ctx, sessionKey, contentHash, since)
	if err == nil && duplicate {
		return true, nil
	}
	if service.persistentEventStore == nil {
		return duplicate, err
	}
	persistentDuplicate, persistentErr := service.persistentEventStore.HasDuplicate(ctx, sessionKey, contentHash, since)
	if persistentErr != nil {
		if err != nil {
			return false, err
		}
		return false, persistentErr
	}
	return duplicate || persistentDuplicate, nil
}

func summarizeEventContext(events []SystemEvent) (string, string) {
	source := ""
	runID := ""
	for i := len(events) - 1; i >= 0; i-- {
		if source == "" {
			source = strings.TrimSpace(events[i].Source)
		}
		if runID == "" {
			runID = strings.TrimSpace(events[i].RunID)
		}
		if source != "" && runID != "" {
			break
		}
	}
	return source, runID
}

func buildHeartbeatAssistantMessageParts(
	content string,
	reason string,
	source string,
	runID string,
	hasCronEvents bool,
	hasExecEvents bool,
) []chatevent.MessagePart {
	trimmedContent := strings.TrimSpace(content)
	if trimmedContent == "" {
		return nil
	}
	parentID := "system-notice"
	resolvedSource := resolveHeartbeatNoticeSource(source, reason, hasCronEvents, hasExecEvents)
	noticeData := map[string]any{
		"origin": "heartbeat",
		"source": resolvedSource,
		"kind":   resolveHeartbeatNoticeKind(resolvedSource),
	}
	if trimmedReason := strings.TrimSpace(reason); trimmedReason != "" {
		noticeData["reason"] = trimmedReason
	}
	if trimmedRunID := strings.TrimSpace(runID); trimmedRunID != "" {
		noticeData["runId"] = trimmedRunID
	}
	parts := make([]chatevent.MessagePart, 0, 2)
	if encoded, err := json.Marshal(map[string]any{
		"name": "system_notice",
		"data": noticeData,
	}); err == nil {
		parts = append(parts, chatevent.MessagePart{
			Type:     "data",
			ParentID: parentID,
			Data:     encoded,
		})
	}
	parts = append(parts, chatevent.MessagePart{
		Type:     "text",
		ParentID: parentID,
		Text:     trimmedContent,
	})
	return parts
}

func resolveHeartbeatNoticeSource(source string, reason string, hasCronEvents bool, hasExecEvents bool) string {
	normalizedSource := strings.ToLower(strings.TrimSpace(source))
	switch normalizedSource {
	case "cron", "exec", "subagent", "heartbeat", "system":
		return normalizedSource
	}
	normalizedReason := strings.ToLower(strings.TrimSpace(reason))
	if hasCronEvents || strings.Contains(normalizedReason, "cron") {
		return "cron"
	}
	if hasExecEvents || strings.Contains(normalizedReason, "exec") {
		return "exec"
	}
	if strings.Contains(normalizedReason, "subagent") {
		return "subagent"
	}
	if strings.Contains(normalizedReason, "heartbeat") {
		return "heartbeat"
	}
	return "system"
}

func resolveHeartbeatNoticeKind(source string) string {
	switch strings.ToLower(strings.TrimSpace(source)) {
	case "cron":
		return "cron_reminder"
	case "exec":
		return "exec_result"
	case "subagent":
		return "subagent_result"
	case "heartbeat":
		return "heartbeat_notice"
	default:
		return "system_notice"
	}
}

func (service *Service) publishNotice(
	ctx context.Context,
	threadItem thread.Thread,
	sessionKey string,
	reason string,
	eventSource string,
	eventRunID string,
	cleaned string,
	alert heartbeatAlert,
	cfg settingsdto.GatewayHeartbeatSettings,
	eventDriven bool,
) {
	if service == nil || service.notices == nil {
		return
	}
	input, ok := service.buildNoticeInput(threadItem, sessionKey, reason, eventSource, eventRunID, cleaned, alert, cfg, eventDriven)
	if !ok {
		return
	}
	_, _ = service.notices.Create(ctx, input)
}

func (service *Service) buildNoticeInput(
	threadItem thread.Thread,
	sessionKey string,
	reason string,
	eventSource string,
	eventRunID string,
	cleaned string,
	alert heartbeatAlert,
	cfg settingsdto.GatewayHeartbeatSettings,
	eventDriven bool,
) (appnotice.CreateNoticeInput, bool) {
	severity := alert.Severity
	if severity == "" {
		severity = domainnotice.SeverityWarning
	}
	surfaces := resolveHeartbeatNoticeSurfaces(severity, cfg, eventDriven)
	if len(surfaces) == 0 {
		return appnotice.CreateNoticeInput{}, false
	}
	code, category, i18nKeys := resolveHeartbeatNoticePresentation(alert.Code, eventSource, eventDriven)
	detail := strings.TrimSpace(cleaned)
	if detail == "" {
		detail = resolveHeartbeatAlertMessage(alert, "")
	}
	params := map[string]string{
		"detail": detail,
	}
	if trimmedReason := strings.TrimSpace(reason); trimmedReason != "" {
		params["reason"] = trimmedReason
	}
	for key, value := range alert.Params {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		params[trimmedKey] = value
	}
	action := alert.Action
	if strings.TrimSpace(action.Type) == "" || strings.TrimSpace(action.LabelKey) == "" {
		action = defaultHeartbeatNoticeAction(category, threadItem.ID)
	}
	return appnotice.CreateNoticeInput{
		Kind:     domainnotice.KindRuntimeEvent,
		Category: category,
		Code:     code,
		Severity: severity,
		I18n: &domainnotice.I18n{
			TitleKey:   i18nKeys.TitleKey,
			SummaryKey: i18nKeys.SummaryKey,
			BodyKey:    i18nKeys.BodyKey,
			Params:     params,
		},
		Source: domainnotice.Source{
			Producer:   "heartbeat",
			SessionKey: strings.TrimSpace(sessionKey),
			ThreadID:   strings.TrimSpace(threadItem.ID),
			RunID:      strings.TrimSpace(eventRunID),
			Metadata: map[string]string{
				"reason": strings.TrimSpace(reason),
				"source": strings.TrimSpace(eventSource),
			},
		},
		Action:   action,
		Surfaces: surfaces,
		DedupKey: strings.Join([]string{"heartbeat", strings.TrimSpace(sessionKey), code, hashContent(detail)}, ":"),
		Metadata: map[string]any{
			"eventDriven": eventDriven,
			"eventSource": strings.TrimSpace(eventSource),
			"rawCode":     strings.TrimSpace(alert.Code),
			"reason":      strings.TrimSpace(reason),
		},
	}, true
}

type heartbeatNoticeI18nKeys struct {
	TitleKey   string
	SummaryKey string
	BodyKey    string
}

func resolveHeartbeatNoticePresentation(rawCode string, eventSource string, eventDriven bool) (string, domainnotice.Category, heartbeatNoticeI18nKeys) {
	normalizedCode := strings.TrimSpace(rawCode)
	source := strings.ToLower(strings.TrimSpace(eventSource))
	category := domainnotice.CategoryHeartbeat
	keys := heartbeatNoticeI18nKeys{
		TitleKey:   "notifications.center.codes.heartbeatPeriodic.title",
		SummaryKey: "notifications.center.codes.heartbeatPeriodic.summary",
		BodyKey:    "notifications.center.codes.heartbeatPeriodic.body",
	}
	switch {
	case normalizedCode == "heartbeat.runtime_failed":
		keys = heartbeatNoticeI18nKeys{
			TitleKey:   "notifications.center.codes.heartbeatRuntimeFailed.title",
			SummaryKey: "notifications.center.codes.heartbeatRuntimeFailed.summary",
			BodyKey:    "notifications.center.codes.heartbeatRuntimeFailed.body",
		}
	case source == "cron" || strings.Contains(strings.ToLower(normalizedCode), "cron"):
		category = domainnotice.CategoryCron
		keys = heartbeatNoticeI18nKeys{
			TitleKey:   "notifications.center.codes.heartbeatCron.title",
			SummaryKey: "notifications.center.codes.heartbeatCron.summary",
			BodyKey:    "notifications.center.codes.heartbeatCron.body",
		}
	case source == "exec" || strings.Contains(strings.ToLower(normalizedCode), "exec"):
		category = domainnotice.CategoryExec
		keys = heartbeatNoticeI18nKeys{
			TitleKey:   "notifications.center.codes.heartbeatExec.title",
			SummaryKey: "notifications.center.codes.heartbeatExec.summary",
			BodyKey:    "notifications.center.codes.heartbeatExec.body",
		}
	case source == "subagent" || strings.Contains(strings.ToLower(normalizedCode), "subagent"):
		category = domainnotice.CategorySubagent
		keys = heartbeatNoticeI18nKeys{
			TitleKey:   "notifications.center.codes.heartbeatSubagent.title",
			SummaryKey: "notifications.center.codes.heartbeatSubagent.summary",
			BodyKey:    "notifications.center.codes.heartbeatSubagent.body",
		}
	case eventDriven:
		keys = heartbeatNoticeI18nKeys{
			TitleKey:   "notifications.center.codes.heartbeatEvent.title",
			SummaryKey: "notifications.center.codes.heartbeatEvent.summary",
			BodyKey:    "notifications.center.codes.heartbeatEvent.body",
		}
	}
	if normalizedCode == "" {
		switch category {
		case domainnotice.CategoryCron:
			normalizedCode = "heartbeat.cron_attention"
		case domainnotice.CategoryExec:
			normalizedCode = "heartbeat.exec_attention"
		case domainnotice.CategorySubagent:
			normalizedCode = "heartbeat.subagent_attention"
		default:
			if eventDriven {
				normalizedCode = "heartbeat.event_attention"
			} else {
				normalizedCode = "heartbeat.periodic_attention"
			}
		}
	}
	return normalizedCode, category, keys
}

func resolveHeartbeatNoticeSurfaces(severity domainnotice.Severity, cfg settingsdto.GatewayHeartbeatSettings, eventDriven bool) []domainnotice.Surface {
	policy := cfg.Delivery.Periodic
	if eventDriven {
		policy = cfg.Delivery.EventDriven
	}
	surfaces := make([]domainnotice.Surface, 0, 4)
	if policy.Center {
		surfaces = append(surfaces, domainnotice.SurfaceCenter)
	}
	if noticeSeverityMeetsMin(severity, policy.PopupMinSeverity) {
		surfaces = append(surfaces, domainnotice.SurfacePopup)
	}
	if noticeSeverityMeetsMin(severity, policy.ToastMinSeverity) {
		surfaces = append(surfaces, domainnotice.SurfaceToast)
	}
	if noticeSeverityMeetsMin(severity, policy.OSMinSeverity) {
		surfaces = append(surfaces, domainnotice.SurfaceOS)
	}
	return surfaces
}

func shouldReplyInThread(cfg settingsdto.GatewayHeartbeatSettings) bool {
	return resolveHeartbeatThreadReplyMode(cfg) == "inline"
}

func resolveHeartbeatThreadReplyMode(cfg settingsdto.GatewayHeartbeatSettings) string {
	switch strings.ToLower(strings.TrimSpace(cfg.Delivery.ThreadReplyMode)) {
	case "inline":
		return "inline"
	default:
		return "never"
	}
}

func defaultHeartbeatNoticeAction(category domainnotice.Category, threadID string) domainnotice.Action {
	if trimmedThreadID := strings.TrimSpace(threadID); trimmedThreadID != "" {
		return domainnotice.Action{
			Type:     "open_thread",
			LabelKey: "notifications.actions.openThread",
			Target:   trimmedThreadID,
		}
	}
	switch category {
	case domainnotice.CategoryCron:
		return domainnotice.Action{
			Type:     "open_route",
			LabelKey: "notifications.actions.openCron",
			Target:   "cron",
		}
	default:
		return domainnotice.Action{
			Type:     "open_route",
			LabelKey: "notifications.actions.openNotifications",
			Target:   "notifications",
		}
	}
}

func resolveIndicatorStatus(status EventStatus, enabled bool) EventIndicatorType {
	if !enabled {
		return ""
	}
	switch status {
	case StatusOKToken, StatusOKEmpty:
		return IndicatorOK
	case StatusSent:
		return IndicatorAlert
	case StatusFailed:
		return IndicatorError
	default:
		return ""
	}
}

func hashContent(content string) string {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(trimmed))
	return hex.EncodeToString(sum[:])
}

func errString(err error, fallback string) string {
	if err == nil {
		return fallback
	}
	return err.Error()
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

func specFromGatewayChecklist(checklist settingsdto.GatewayHeartbeatChecklist) (Spec, bool) {
	updatedAt := time.Time{}
	if raw := strings.TrimSpace(checklist.UpdatedAt); raw != "" {
		if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
			updatedAt = parsed
		}
	}
	spec := sanitizeSpec(Spec{
		Title:     checklist.Title,
		Items:     toSpecChecklistItems(checklist.Items),
		Notes:     checklist.Notes,
		Version:   checklist.Version,
		UpdatedAt: updatedAt,
	})
	if spec.Version < 0 {
		spec.Version = 0
	}
	empty := strings.TrimSpace(spec.Title) == "" &&
		len(spec.Items) == 0 &&
		strings.TrimSpace(spec.Notes) == "" &&
		spec.Version == 0 &&
		spec.UpdatedAt.IsZero()
	if empty {
		return Spec{}, false
	}
	return spec, true
}

func toSpecChecklistItems(items []settingsdto.GatewayHeartbeatChecklistItem) []ChecklistItem {
	if len(items) == 0 {
		return nil
	}
	result := make([]ChecklistItem, 0, len(items))
	for _, item := range items {
		result = append(result, ChecklistItem{
			ID:       item.ID,
			Text:     item.Text,
			Done:     item.Done,
			Priority: item.Priority,
		})
	}
	return result
}
