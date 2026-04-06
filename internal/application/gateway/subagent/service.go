package subagent

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	gatewayevents "dreamcreator/internal/application/gateway/events"
	gatewayheartbeat "dreamcreator/internal/application/gateway/heartbeat"
	gatewayqueue "dreamcreator/internal/application/gateway/queue"
	runtimedto "dreamcreator/internal/application/gateway/runtime/dto"
	settingsservice "dreamcreator/internal/application/settings/service"
	subagentservice "dreamcreator/internal/application/subagent/service"
	threaddto "dreamcreator/internal/application/thread/dto"
	domainsession "dreamcreator/internal/domain/session"
	domainsettings "dreamcreator/internal/domain/settings"
	domainthread "dreamcreator/internal/domain/thread"
	"github.com/google/uuid"
)

const defaultSubagentMaxSpawnDepth = 1
const (
	defaultAnnounceRetryMax     = 3
	defaultAnnounceRetryBackoff = 500 * time.Millisecond
	maxAnnounceRetryBackoff     = 4 * time.Second
	defaultArchiveAfterMinutes  = 60
)

var subagentToolDenyAlways = []string{
	"gateway",
	"agents_list",
	"whatsapp_login",
	"session_status",
	"cron",
	"memory_recall",
	"memory_list",
	"sessions_send",
}

var subagentToolDenyLeaf = []string{
	"sessions_list",
	"sessions_history",
	"sessions_spawn",
}

type AnnounceEvent struct {
	RunID            string `json:"runId"`
	AnnounceID       string `json:"announceId,omitempty"`
	ParentSessionKey string `json:"parentSessionKey,omitempty"`
	ParentRunID      string `json:"parentRunId,omitempty"`
	AgentID          string `json:"agentId,omitempty"`
	ChildSessionKey  string `json:"childSessionKey,omitempty"`
	ChildSessionID   string `json:"childSessionId,omitempty"`
	Transcript       string `json:"transcript,omitempty"`
	Status           string `json:"status"`
	Summary          string `json:"summary,omitempty"`
	Error            string `json:"error,omitempty"`
	Result           string `json:"result,omitempty"`
	Notes            string `json:"notes,omitempty"`
	Runtime          string `json:"runtime,omitempty"`
	Usage            any    `json:"usage,omitempty"`
}

type announceQueueItem struct {
	record   subagentservice.RunRecord
	id       string
	attempt  int
	enqueued time.Time
}

type GatewayService struct {
	spawner   *subagentservice.Spawner
	queue     *gatewayqueue.Manager
	store     RunStore
	events    *gatewayevents.Broker
	runtime   RuntimeRunner
	aborts    Aborter
	controls  ControlRegistry
	threads   ThreadWriter
	settings  *settingsservice.SettingsService
	heartbeat HeartbeatSink
	now       func() time.Time

	announceMu       sync.Mutex
	announceQueue    []announceQueueItem
	announceDraining bool
	announced        map[string]struct{}

	archiveMu     sync.Mutex
	archiveTimers map[string]*time.Timer

	isolatedSessionEnabled         bool
	announceStandardizationEnabled bool
	lifecycleEnhancementEnabled    bool
}

type RunStore interface {
	Save(ctx context.Context, record subagentservice.RunRecord) error
	Get(ctx context.Context, runID string) (subagentservice.RunRecord, error)
	ListByParent(ctx context.Context, parentSessionKey string) ([]subagentservice.RunRecord, error)
	ListPendingAnnounce(ctx context.Context) ([]subagentservice.RunRecord, error)
}

type RuntimeRunner interface {
	Run(ctx context.Context, request runtimedto.RuntimeRunRequest) (runtimedto.RuntimeRunResult, error)
}

type Aborter interface {
	Abort(runID string, reason string) bool
}

type ControlRegistry interface {
	Steer(runID string, message string) bool
	FollowUp(runID string, message string) bool
}

type ThreadWriter interface {
	AppendMessage(ctx context.Context, request threaddto.AppendMessageRequest) error
}

type ThreadLifecycleWriter interface {
	SetThreadStatus(ctx context.Context, request threaddto.SetThreadStatusRequest) error
	SoftDeleteThread(ctx context.Context, id string) error
}

type HeartbeatSink interface {
	EnqueueSystemEvent(ctx context.Context, input gatewayheartbeat.SystemEventInput) bool
	TriggerWithInput(ctx context.Context, input gatewayheartbeat.TriggerInput) bool
}

func NewGatewayService(
	spawner *subagentservice.Spawner,
	queue *gatewayqueue.Manager,
	store RunStore,
	events *gatewayevents.Broker,
	runtime RuntimeRunner,
	aborts Aborter,
	controls ControlRegistry,
	threadWriter ThreadWriter,
	settings *settingsservice.SettingsService,
	heartbeat HeartbeatSink,
) *GatewayService {
	if spawner == nil {
		spawner = subagentservice.NewSpawner()
	}
	gateway := &GatewayService{
		spawner:                        spawner,
		queue:                          queue,
		store:                          store,
		events:                         events,
		runtime:                        runtime,
		aborts:                         aborts,
		controls:                       controls,
		threads:                        threadWriter,
		settings:                       settings,
		heartbeat:                      heartbeat,
		now:                            time.Now,
		announced:                      make(map[string]struct{}),
		archiveTimers:                  make(map[string]*time.Timer),
		isolatedSessionEnabled:         envBool("DREAMCREATOR_SUBAGENT_ISOLATED_SESSION_ENABLED", true),
		announceStandardizationEnabled: envBool("DREAMCREATOR_SUBAGENT_ANNOUNCE_STANDARDIZATION_ENABLED", true),
		lifecycleEnhancementEnabled:    envBool("DREAMCREATOR_SUBAGENT_LIFECYCLE_ENABLED", true),
	}
	go gateway.recoverPendingAnnounces(context.Background())
	return gateway
}

func (gateway *GatewayService) Spawn(ctx context.Context, request subagentservice.SpawnRequest) (subagentservice.RunRecord, error) {
	payload := normalizeSubagentPayload(request.Payload)
	request.Payload = payload
	request.ParentSessionKey = strings.TrimSpace(request.ParentSessionKey)
	request.ParentRunID = strings.TrimSpace(request.ParentRunID)
	request.AgentID = strings.TrimSpace(request.AgentID)
	if request.Task == "" {
		request.Task, _ = extractTask(payload)
	}
	request.Task = strings.TrimSpace(request.Task)
	if request.Task == "" {
		return subagentservice.RunRecord{}, errors.New("task is required")
	}
	if request.Label == "" {
		if value, ok := extractPayloadString(payload, "label"); ok {
			request.Label = value
		}
	}
	request.Label = strings.TrimSpace(request.Label)
	if request.Model == "" {
		if value, ok := extractPayloadString(payload, "model"); ok {
			request.Model = value
		}
	}
	if request.Thinking == "" {
		if value, ok := extractPayloadString(payload, "thinking"); ok {
			request.Thinking = value
		}
	}
	if request.CallerModel == "" {
		if value, ok := extractPayloadString(payload, "callerModel"); ok {
			request.CallerModel = value
		}
	}
	if request.CallerThinking == "" {
		if value, ok := extractPayloadString(payload, "callerThinking"); ok {
			request.CallerThinking = value
		}
	}
	if request.RunTimeoutSeconds <= 0 {
		if value, ok := extractPayloadInt(payload, "runTimeoutSeconds"); ok {
			request.RunTimeoutSeconds = value
		} else if value, ok := extractPayloadInt(payload, "timeoutSeconds"); ok {
			request.RunTimeoutSeconds = value
		}
	}
	if request.RunTimeoutSeconds < 0 {
		request.RunTimeoutSeconds = 0
	}
	if request.CleanupPolicy == "" {
		if value, ok := extractPayloadString(payload, "cleanup"); ok {
			request.CleanupPolicy = subagentservice.ParseCleanupPolicy(value)
		}
	}
	if request.CleanupPolicy == "" {
		request.CleanupPolicy = subagentservice.CleanupKeep
	}
	if !gateway.lifecycleEnhancementEnabled {
		request.RunTimeoutSeconds = 0
		request.CleanupPolicy = subagentservice.CleanupKeep
	}
	if request.ParentSessionKey == "" {
		return subagentservice.RunRecord{}, errors.New("sessionKey is required")
	}

	childSessionID := strings.TrimSpace(request.ChildSessionID)
	if childSessionID == "" {
		childSessionID = uuid.NewString()
	}
	childSessionKey := strings.TrimSpace(request.ChildSessionKey)
	if childSessionKey == "" {
		childSessionKey = buildChildSessionKey(request.AgentID, childSessionID)
	}
	if !gateway.isolatedSessionEnabled {
		childSessionKey = request.ParentSessionKey
		if parentID := resolveParentThreadID(request.ParentSessionKey); parentID != "" {
			childSessionID = parentID
		}
	}
	request.ChildSessionID = childSessionID
	request.ChildSessionKey = childSessionKey

	limits := gateway.resolveSubagentLimits(ctx)
	maxDepth := limits.maxDepth
	if value := resolveSubagentMaxDepth(payload); value > 0 {
		maxDepth = value
	}
	if maxDepth <= 0 {
		maxDepth = defaultSubagentMaxSpawnDepth
	}
	depth := resolveSubagentDepth(payload)
	if depth <= 0 {
		depth = gateway.resolveDepthFromParent(ctx, request.ParentRunID)
	}
	if depth <= 0 {
		depth = 1
	}
	if maxDepth > 0 && depth > maxDepth {
		return subagentservice.RunRecord{}, errors.New("subagent depth exceeds maxDepth")
	}
	if limits.maxChildren > 0 {
		if count := gateway.countActiveChildren(ctx, request.ParentSessionKey, request.ParentRunID); count >= limits.maxChildren {
			return subagentservice.RunRecord{}, errors.New("subagent maxChildren exceeded")
		}
	}
	if limits.maxConcurrent > 0 {
		if count := gateway.countActiveBySession(ctx, request.ParentSessionKey); count >= limits.maxConcurrent {
			return subagentservice.RunRecord{}, errors.New("subagent maxConcurrent exceeded")
		}
	}
	if payload != nil {
		payload["subagentDepth"] = depth
		payload["subagentMaxDepth"] = maxDepth
		payload["task"] = request.Task
		if request.Label != "" {
			payload["label"] = request.Label
		}
		if request.Model != "" {
			payload["model"] = request.Model
		}
		if request.Thinking != "" {
			payload["thinking"] = request.Thinking
		}
		if request.CallerModel != "" {
			payload["callerModel"] = request.CallerModel
		}
		if request.CallerThinking != "" {
			payload["callerThinking"] = request.CallerThinking
		}
		if request.RunTimeoutSeconds > 0 {
			payload["runTimeoutSeconds"] = request.RunTimeoutSeconds
		}
		payload["cleanup"] = string(request.CleanupPolicy)
		payload["childSessionKey"] = request.ChildSessionKey
		payload["childSessionId"] = request.ChildSessionID
	}
	record, err := gateway.spawner.Spawn(ctx, request)
	if err != nil {
		return subagentservice.RunRecord{}, err
	}
	if gateway.store != nil {
		_ = gateway.store.Save(ctx, record)
	}
	gateway.enqueueSpawn(ctx, record)
	go gateway.runSubagent(record, request)
	return record, nil
}

func (gateway *GatewayService) Get(ctx context.Context, runID string) (subagentservice.RunRecord, error) {
	if gateway == nil {
		return subagentservice.RunRecord{}, subagentservice.ErrSubagentRunNotFound
	}
	if gateway.store != nil {
		if record, err := gateway.store.Get(ctx, runID); err == nil {
			return record, nil
		}
	}
	return gateway.spawner.Get(ctx, runID)
}

func (gateway *GatewayService) ListByParent(ctx context.Context, parentSessionKey string) ([]subagentservice.RunRecord, error) {
	if gateway == nil {
		return nil, subagentservice.ErrSubagentRunNotFound
	}
	parentSessionKey = strings.TrimSpace(parentSessionKey)
	if parentSessionKey == "" {
		return nil, subagentservice.ErrSubagentRunNotFound
	}
	if gateway.store != nil {
		if records, err := gateway.store.ListByParent(ctx, parentSessionKey); err == nil {
			return records, nil
		}
	}
	return []subagentservice.RunRecord{}, nil
}

func (gateway *GatewayService) Kill(ctx context.Context, runID string) error {
	if gateway == nil {
		return subagentservice.ErrSubagentRunNotFound
	}
	trimmed := strings.TrimSpace(runID)
	if trimmed == "" {
		return subagentservice.ErrSubagentRunNotFound
	}
	if gateway.aborts == nil || !gateway.aborts.Abort(trimmed, "subagent_killed") {
		return errors.New("subagent not running")
	}
	record, err := gateway.Get(ctx, trimmed)
	if err == nil {
		record.Status = subagentservice.RunStatusAborted
		record.Error = "aborted"
		record.Notes = "aborted by kill"
		record.UpdatedAt = gateway.now()
		if gateway.store != nil {
			_ = gateway.store.Save(ctx, record)
		}
		if gateway.spawner != nil {
			gateway.spawner.Update(record)
		}
	}
	return nil
}

func (gateway *GatewayService) KillAll(ctx context.Context, parentSessionKey string) (int, error) {
	if gateway == nil {
		return 0, subagentservice.ErrSubagentRunNotFound
	}
	if strings.TrimSpace(parentSessionKey) == "" {
		return 0, subagentservice.ErrSubagentRunNotFound
	}
	records, err := gateway.ListByParent(ctx, parentSessionKey)
	if err != nil {
		return 0, err
	}
	stopped := 0
	for _, record := range records {
		if record.Status != subagentservice.RunStatusRunning {
			continue
		}
		if gateway.aborts != nil && gateway.aborts.Abort(record.RunID, "subagent_killed") {
			stopped++
		}
	}
	return stopped, nil
}

func (gateway *GatewayService) Steer(ctx context.Context, runID string, message string) error {
	if gateway == nil {
		return subagentservice.ErrSubagentRunNotFound
	}
	trimmed := strings.TrimSpace(runID)
	if trimmed == "" {
		return subagentservice.ErrSubagentRunNotFound
	}
	if gateway.controls == nil || !gateway.controls.Steer(trimmed, message) {
		return errors.New("subagent not running")
	}
	return nil
}

func (gateway *GatewayService) Send(ctx context.Context, runID string, message string) error {
	if gateway == nil {
		return subagentservice.ErrSubagentRunNotFound
	}
	trimmed := strings.TrimSpace(runID)
	if trimmed == "" {
		return subagentservice.ErrSubagentRunNotFound
	}
	if gateway.controls == nil || !gateway.controls.FollowUp(trimmed, message) {
		return errors.New("subagent not running")
	}
	return nil
}

func (gateway *GatewayService) publishAnnounce(ctx context.Context, record subagentservice.RunRecord) {
	if gateway == nil || gateway.events == nil {
		return
	}
	announceID := buildAnnounceID(record)
	if announceID == "" {
		return
	}
	if record.AnnounceSentAt != nil && strings.TrimSpace(record.AnnounceKey) == announceID {
		gateway.markAnnounced(announceID)
		return
	}
	if gateway.isAnnounced(announceID) {
		return
	}
	item := announceQueueItem{
		record:   record,
		id:       announceID,
		enqueued: gateway.now(),
	}
	gateway.enqueueAnnounce(item)
}

func buildAnnounceID(record subagentservice.RunRecord) string {
	runID := strings.TrimSpace(record.RunID)
	if runID == "" {
		return ""
	}
	status := strings.TrimSpace(string(record.Status))
	if status == "" {
		status = "unknown"
	}
	return "announce:" + runID + ":" + status
}

func (gateway *GatewayService) enqueueAnnounce(item announceQueueItem) {
	if gateway == nil {
		return
	}
	gateway.announceMu.Lock()
	for _, pending := range gateway.announceQueue {
		if pending.id == item.id {
			gateway.announceMu.Unlock()
			return
		}
	}
	gateway.announceQueue = append(gateway.announceQueue, item)
	if gateway.announceDraining {
		gateway.announceMu.Unlock()
		return
	}
	gateway.announceDraining = true
	gateway.announceMu.Unlock()
	go gateway.drainAnnounceQueue()
}

func (gateway *GatewayService) drainAnnounceQueue() {
	if gateway == nil {
		return
	}
	for {
		gateway.announceMu.Lock()
		if len(gateway.announceQueue) == 0 {
			gateway.announceDraining = false
			gateway.announceMu.Unlock()
			return
		}
		item := gateway.announceQueue[0]
		gateway.announceQueue = gateway.announceQueue[1:]
		gateway.announceMu.Unlock()

		if gateway.isAnnounced(item.id) {
			continue
		}

		publishErr := gateway.publishAnnounceNow(context.Background(), item)
		if publishErr == nil {
			gateway.markAnnounceDelivered(item)
			continue
		}

		item.attempt++
		gateway.markAnnounceAttempt(item, false)
		if item.attempt >= defaultAnnounceRetryMax {
			continue
		}
		time.Sleep(resolveAnnounceRetryBackoff(item.attempt))
		gateway.enqueueAnnounce(item)
	}
}

func resolveAnnounceRetryBackoff(attempt int) time.Duration {
	if attempt <= 0 {
		return defaultAnnounceRetryBackoff
	}
	delay := defaultAnnounceRetryBackoff * time.Duration(1<<(attempt-1))
	if delay > maxAnnounceRetryBackoff {
		return maxAnnounceRetryBackoff
	}
	return delay
}

func (gateway *GatewayService) publishAnnounceNow(ctx context.Context, item announceQueueItem) error {
	if gateway == nil || gateway.events == nil {
		return nil
	}
	result := strings.TrimSpace(item.record.Result)
	if result == "" {
		result = strings.TrimSpace(item.record.Summary)
	}
	notes := strings.TrimSpace(item.record.Notes)
	if notes == "" {
		notes = strings.TrimSpace(item.record.Error)
	}
	payload := AnnounceEvent{
		RunID:            item.record.RunID,
		AnnounceID:       item.id,
		ParentSessionKey: item.record.ParentSessionKey,
		ParentRunID:      item.record.ParentRunID,
		AgentID:          item.record.AgentID,
		ChildSessionKey:  item.record.ChildSessionKey,
		ChildSessionID:   item.record.ChildSessionID,
		Transcript:       strings.TrimSpace(item.record.TranscriptPath),
		Status:           announceStatus(item.record.Status),
	}
	if gateway.announceStandardizationEnabled {
		payload.Result = result
		payload.Notes = notes
		payload.Runtime = formatRuntimeText(item.record.RuntimeMs)
		payload.Usage = map[string]int{
			"promptTokens":     item.record.Usage.PromptTokens,
			"completionTokens": item.record.Usage.CompletionTokens,
			"totalTokens":      item.record.Usage.TotalTokens,
		}
	} else {
		payload.Status = string(item.record.Status)
		payload.Summary = strings.TrimSpace(item.record.Summary)
		payload.Error = strings.TrimSpace(item.record.Error)
	}
	envelope := gatewayevents.Envelope{
		Type:       "subagent.announced",
		Topic:      "subagent",
		SessionKey: strings.TrimSpace(item.record.ParentSessionKey),
		RunID:      strings.TrimSpace(item.record.RunID),
		Timestamp:  gateway.now(),
	}
	if envelope.SessionKey != "" {
		if parts, _, err := domainsession.NormalizeSessionKey(envelope.SessionKey); err == nil {
			envelope.SessionID = strings.TrimSpace(parts.ThreadRef)
		}
	}
	_, err := gateway.events.Publish(ctx, envelope, payload)
	return err
}

func (gateway *GatewayService) markAnnounceAttempt(item announceQueueItem, delivered bool) {
	if gateway == nil {
		return
	}
	if delivered {
		gateway.markAnnounced(item.id)
	}
	if gateway.store == nil {
		return
	}
	record := item.record
	record.AnnounceKey = item.id
	attempts := item.attempt
	if attempts <= 0 {
		attempts = 1
	}
	record.AnnounceAttempts = attempts
	now := gateway.now()
	record.UpdatedAt = now
	if delivered {
		record.AnnounceSentAt = &now
	}
	_ = gateway.store.Save(context.Background(), record)
}

func (gateway *GatewayService) markAnnounceDelivered(item announceQueueItem) {
	gateway.markAnnounceAttempt(item, true)
}

func (gateway *GatewayService) markAnnounced(announceID string) {
	if gateway == nil || strings.TrimSpace(announceID) == "" {
		return
	}
	gateway.announceMu.Lock()
	gateway.announced[strings.TrimSpace(announceID)] = struct{}{}
	gateway.announceMu.Unlock()
}

func (gateway *GatewayService) isAnnounced(announceID string) bool {
	if gateway == nil || strings.TrimSpace(announceID) == "" {
		return false
	}
	key := strings.TrimSpace(announceID)
	gateway.announceMu.Lock()
	defer gateway.announceMu.Unlock()
	_, ok := gateway.announced[key]
	return ok
}

func (gateway *GatewayService) recoverPendingAnnounces(ctx context.Context) {
	if gateway == nil || gateway.store == nil {
		return
	}
	records, err := gateway.store.ListPendingAnnounce(ctx)
	if err != nil || len(records) == 0 {
		return
	}
	for _, record := range records {
		if strings.TrimSpace(record.RunID) == "" {
			continue
		}
		gateway.publishAnnounce(ctx, record)
	}
}

func (gateway *GatewayService) enqueueSpawn(ctx context.Context, record subagentservice.RunRecord) {
	if gateway == nil || gateway.queue == nil {
		return
	}
	sessionKey := strings.TrimSpace(record.ParentSessionKey)
	if sessionKey == "" {
		return
	}
	_, _, _ = gateway.queue.Enqueue(ctx, gatewayqueue.EnqueueRequest{
		SessionKey: sessionKey,
		Mode:       string(domainsession.QueueModeFollowup),
		Payload: map[string]any{
			"type":             "subagent.spawn",
			"runId":            record.RunID,
			"parentSessionKey": record.ParentSessionKey,
			"parentRunId":      record.ParentRunID,
			"agentId":          record.AgentID,
		},
	})
}

func (gateway *GatewayService) runSubagent(record subagentservice.RunRecord, request subagentservice.SpawnRequest) {
	if gateway == nil || gateway.runtime == nil {
		return
	}
	ctx := context.Background()
	task := strings.TrimSpace(request.Task)
	contextText := ""
	if task == "" {
		task, contextText = extractTask(request.Payload)
	}
	if task == "" {
		task = "subagent task"
	}
	record.Task = task
	record.Label = strings.TrimSpace(request.Label)
	record.CallerModel = strings.TrimSpace(request.CallerModel)
	record.CallerThinking = strings.TrimSpace(request.CallerThinking)

	content := task
	if contextText != "" {
		content = content + "\n\nContext:\n" + contextText
	}

	explicitModel := strings.TrimSpace(request.Model)
	if explicitModel == "" {
		if value, ok := extractPayloadString(request.Payload, "model"); ok {
			explicitModel = value
		}
	}
	explicitThinking := strings.TrimSpace(request.Thinking)
	if explicitThinking == "" {
		if value, ok := extractPayloadString(request.Payload, "thinking"); ok {
			explicitThinking = value
		}
	}

	providerID := ""
	modelName := ""
	defaults := gateway.resolveSubagentDefaults(ctx)
	modelNotes := make([]string, 0, 1)
	if explicitModel != "" {
		overrideProvider, overrideModel := splitModelRef(explicitModel)
		if overrideProvider != "" && overrideModel != "" {
			providerID = overrideProvider
			modelName = overrideModel
			record.Model = explicitModel
		} else {
			modelNotes = append(modelNotes, "model override 无效，已回退默认模型")
		}
	} else {
		if providerID != "" && modelName != "" {
			record.Model = providerID + "/" + modelName
		} else if defaults.model != "" {
			overrideProvider, overrideModel := splitModelRef(defaults.model)
			if overrideProvider != "" && overrideModel != "" {
				providerID = overrideProvider
				modelName = overrideModel
				record.Model = defaults.model
			}
		} else if request.CallerModel != "" {
			overrideProvider, overrideModel := splitModelRef(request.CallerModel)
			if overrideProvider != "" && overrideModel != "" {
				providerID = overrideProvider
				modelName = overrideModel
				record.Model = request.CallerModel
			}
		}
	}
	useQueue := true
	if value, ok := extractPayloadBool(request.Payload, "useQueue"); ok {
		useQueue = value
	}
	metadata := map[string]any{
		"isSubagent":       true,
		"persistRun":       true,
		"persistMessages":  true,
		"persistEvents":    false,
		"persistUsage":     true,
		"useQueue":         useQueue,
		"parentSessionKey": strings.TrimSpace(record.ParentSessionKey),
		"parentRunId":      strings.TrimSpace(record.ParentRunID),
		"childSessionKey":  strings.TrimSpace(record.ChildSessionKey),
		"childSessionId":   strings.TrimSpace(record.ChildSessionID),
	}
	if lane, ok := extractPayloadString(request.Payload, "queueLane"); ok {
		metadata["queueLane"] = lane
	}
	depth := resolveSubagentDepth(request.Payload)
	maxDepth := resolveSubagentMaxDepth(request.Payload)
	metadata["subagentDepth"] = depth
	metadata["subagentMaxDepth"] = maxDepth
	toolConfig := gateway.resolveSubagentTools(ctx, request.Payload, depth, maxDepth)
	runTimeoutSeconds := request.RunTimeoutSeconds
	if runTimeoutSeconds <= 0 {
		if value, ok := extractPayloadInt(request.Payload, "runTimeoutSeconds"); ok {
			runTimeoutSeconds = value
		} else if value, ok := extractPayloadInt(request.Payload, "timeoutSeconds"); ok {
			runTimeoutSeconds = value
		}
	}
	if gateway.lifecycleEnhancementEnabled && runTimeoutSeconds > 0 {
		metadata["runTimeoutSeconds"] = runTimeoutSeconds
		record.RunTimeoutSeconds = runTimeoutSeconds
	}
	if value, ok := extractPayloadInt(request.Payload, "maxSteps"); ok {
		metadata["maxSteps"] = value
	}
	thinking := ""
	if explicitThinking != "" {
		thinking = explicitThinking
	} else if defaults.thinking != "" {
		thinking = defaults.thinking
	} else if request.CallerThinking != "" {
		thinking = request.CallerThinking
	}
	if thinking != "" {
		metadata["thinking"] = thinking
		record.Thinking = thinking
	}
	if len(modelNotes) > 0 {
		record.Notes = strings.Join(modelNotes, "; ")
	}
	promptMode := resolveSubagentPromptMode(request.Payload)
	runRequest := runtimedto.RuntimeRunRequest{
		RunID:      record.RunID,
		SessionID:  strings.TrimSpace(record.ChildSessionID),
		SessionKey: strings.TrimSpace(record.ChildSessionKey),
		AgentID:    strings.TrimSpace(record.AgentID),
		PromptMode: promptMode,
		Input: runtimedto.RuntimeInput{
			Messages: []runtimedto.Message{{
				Role:    "user",
				Content: content,
			}},
		},
		Tools:    toolConfig,
		Metadata: metadata,
	}
	if strings.TrimSpace(providerID) != "" && strings.TrimSpace(modelName) != "" {
		runRequest.Model = &runtimedto.ModelSelection{
			ProviderID: providerID,
			Name:       modelName,
		}
	}
	result, err := gateway.runtime.Run(ctx, runRequest)
	summary := ""
	if result.AssistantMessage.Content != "" {
		summary = strings.TrimSpace(result.AssistantMessage.Content)
	}
	if summary == "" {
		summary = strings.TrimSpace(result.Error)
	}
	if summary == "" && err != nil {
		summary = "subagent run failed"
	}
	gateway.finishRun(ctx, record, summary, result, err)
}

func (gateway *GatewayService) finishRun(ctx context.Context, record subagentservice.RunRecord, summary string, runtimeResult runtimedto.RuntimeRunResult, err error) {
	now := gateway.now()
	runtimeMs := now.Sub(record.CreatedAt).Milliseconds()
	if runtimeMs < 0 {
		runtimeMs = 0
	}
	status, notes := classifyRunOutcome(record, runtimeMs, err)
	if !gateway.lifecycleEnhancementEnabled {
		status = subagentservice.RunStatusSuccess
		notes = ""
		if err != nil {
			status = subagentservice.RunStatusFailed
			notes = strings.TrimSpace(err.Error())
		}
	}
	if existingNotes := strings.TrimSpace(record.Notes); existingNotes != "" {
		if notes == "" {
			notes = existingNotes
		} else if !strings.Contains(notes, existingNotes) {
			notes = existingNotes + "; " + notes
		}
	}
	record.Status = status
	record.Result = strings.TrimSpace(summary)
	record.Summary = record.Result
	record.RuntimeMs = runtimeMs
	record.Usage = subagentservice.RunUsage{
		PromptTokens:     runtimeResult.Usage.PromptTokens,
		CompletionTokens: runtimeResult.Usage.CompletionTokens,
		TotalTokens:      runtimeResult.Usage.TotalTokens,
	}
	record.Notes = strings.TrimSpace(notes)
	if err != nil {
		record.Error = strings.TrimSpace(err.Error())
	} else if runtimeResult.Error != "" {
		record.Error = strings.TrimSpace(runtimeResult.Error)
	} else {
		record.Error = ""
	}
	record.UpdatedAt = now
	record.FinishedAt = &now
	if gateway.store != nil {
		_ = gateway.store.Save(ctx, record)
	}
	if gateway.spawner != nil {
		gateway.spawner.Update(record)
	}
	gateway.publishAnnounce(ctx, record)
	gateway.publishHeartbeatResult(ctx, record)
	if gateway.lifecycleEnhancementEnabled {
		gateway.scheduleArchive(record)
	}
}

func (gateway *GatewayService) publishHeartbeatResult(ctx context.Context, record subagentservice.RunRecord) {
	if gateway == nil || gateway.heartbeat == nil {
		return
	}
	parentSessionKey := strings.TrimSpace(record.ParentSessionKey)
	if parentSessionKey == "" {
		return
	}
	contextKey := "subagent:success"
	if isSubagentFailureStatus(record.Status) {
		contextKey = "subagent:failed"
	}
	queued := gateway.heartbeat.EnqueueSystemEvent(ctx, gatewayheartbeat.SystemEventInput{
		SessionKey: parentSessionKey,
		Text:       formatSubagentHeartbeatText(record),
		ContextKey: contextKey,
		RunID:      strings.TrimSpace(record.RunID),
		Source:     "subagent",
	})
	if !queued || !isSubagentFailureStatus(record.Status) {
		return
	}
	gateway.heartbeat.TriggerWithInput(ctx, gatewayheartbeat.TriggerInput{
		Reason:     "subagent-event",
		SessionKey: parentSessionKey,
		Force:      true,
	})
}

func resolveParentThreadID(sessionKey string) string {
	sessionKey = strings.TrimSpace(sessionKey)
	if sessionKey == "" {
		return ""
	}
	if parts, normalized, err := domainsession.NormalizeSessionKey(sessionKey); err == nil {
		if parts.ThreadRef != "" {
			return strings.TrimSpace(parts.ThreadRef)
		}
		return strings.TrimSpace(normalized)
	}
	return sessionKey
}

func resolveSubagentPromptMode(payload any) string {
	if value, ok := extractPayloadString(payload, "promptMode"); ok {
		return value
	}
	return "minimal"
}

type subagentToolPolicy struct {
	allow     []string
	alsoAllow []string
	deny      []string
}

func (gateway *GatewayService) resolveSubagentTools(ctx context.Context, payload any, depth int, maxDepth int) runtimedto.ToolExecutionConfig {
	if depth <= 0 {
		depth = 1
	}
	if maxDepth <= 0 {
		maxDepth = defaultSubagentMaxSpawnDepth
	}
	settingsPolicy := gateway.resolveSubagentToolsPolicy(ctx)
	globalPolicy := parseToolPolicy(payload, "globalToolsPolicy")
	channelPolicy := parseToolPolicy(payload, "channelToolsPolicy")
	providerPolicy := parseToolPolicy(payload, "providerToolsPolicy")
	subagentPolicy := parseToolPolicy(payload, "subagentToolsPolicy")
	if override, ok := parseInlineSubagentToolPolicy(payload); ok {
		subagentPolicy = mergeSubagentToolPolicies(subagentPolicy, override)
	}

	effective := subagentToolPolicy{}
	effective = mergeSubagentToolPolicies(effective, globalPolicy)
	effective = mergeSubagentToolPolicies(effective, channelPolicy)
	effective = mergeSubagentToolPolicies(effective, providerPolicy)
	effective = mergeSubagentToolPolicies(effective, settingsPolicy)
	effective = mergeSubagentToolPolicies(effective, subagentPolicy)
	effective.deny = mergeUniqueStrings(effective.deny, resolveSubagentDenyList(depth, maxDepth))

	config := runtimedto.ToolExecutionConfig{
		DenyList: effective.deny,
	}
	if len(effective.allow) > 0 {
		config.AllowList = effective.allow
	}
	return config
}

func resolveSubagentDepth(payload any) int {
	if value, ok := extractPayloadInt(payload, "subagentDepth"); ok && value > 0 {
		return value
	}
	if value, ok := extractPayloadInt(payload, "depth"); ok && value > 0 {
		return value
	}
	return 1
}

func resolveSubagentMaxDepth(payload any) int {
	if value, ok := extractPayloadInt(payload, "subagentMaxDepth"); ok && value > 0 {
		return value
	}
	if value, ok := extractPayloadInt(payload, "maxSpawnDepth"); ok && value > 0 {
		return value
	}
	return defaultSubagentMaxSpawnDepth
}

func resolveSubagentDenyList(depth int, maxDepth int) []string {
	if depth >= maxDepth {
		return append(append([]string{}, subagentToolDenyAlways...), subagentToolDenyLeaf...)
	}
	return append([]string{}, subagentToolDenyAlways...)
}

func (gateway *GatewayService) resolveSubagentToolsPolicy(ctx context.Context) subagentToolPolicy {
	policy := subagentToolPolicy{}
	if gateway == nil || gateway.settings == nil {
		return policy
	}
	current, err := gateway.settings.GetSettings(ctx)
	if err != nil {
		return policy
	}
	policy.allow = normalizeStringList(current.Gateway.Subagents.Tools.Allow)
	policy.alsoAllow = normalizeStringList(current.Gateway.Subagents.Tools.AlsoAllow)
	policy.deny = normalizeStringList(current.Gateway.Subagents.Tools.Deny)
	return policy
}

func parseInlineSubagentToolPolicy(payload any) (subagentToolPolicy, bool) {
	raw, ok := extractPayloadMap(payload, "tools")
	if !ok {
		return subagentToolPolicy{}, false
	}
	return subagentToolPolicy{
		allow:     extractStringSlice(raw, "allow"),
		alsoAllow: extractStringSlice(raw, "alsoAllow"),
		deny:      extractStringSlice(raw, "deny"),
	}, true
}

func parseToolPolicy(payload any, key string) subagentToolPolicy {
	raw, ok := extractPayloadMap(payload, key)
	if !ok {
		return subagentToolPolicy{}
	}
	return subagentToolPolicy{
		allow:     extractStringSlice(raw, "allow"),
		alsoAllow: extractStringSlice(raw, "alsoAllow"),
		deny:      extractStringSlice(raw, "deny"),
	}
}

func extractPayloadMap(payload any, key string) (map[string]any, bool) {
	if payload == nil || strings.TrimSpace(key) == "" {
		return nil, false
	}
	source, ok := payload.(map[string]any)
	if !ok {
		return nil, false
	}
	value, exists := source[key]
	if !exists {
		return nil, false
	}
	typed, ok := value.(map[string]any)
	return typed, ok
}

func extractStringSlice(source map[string]any, key string) []string {
	value, ok := source[key]
	if !ok {
		return nil
	}
	result := make([]string, 0)
	switch typed := value.(type) {
	case []string:
		for _, entry := range typed {
			if trimmed := strings.TrimSpace(entry); trimmed != "" {
				result = append(result, trimmed)
			}
		}
	case []any:
		for _, entry := range typed {
			text, textOK := entry.(string)
			if !textOK {
				continue
			}
			if trimmed := strings.TrimSpace(text); trimmed != "" {
				result = append(result, trimmed)
			}
		}
	}
	return normalizeStringList(result)
}

func mergeSubagentToolPolicies(base subagentToolPolicy, next subagentToolPolicy) subagentToolPolicy {
	merged := base
	if len(next.allow) > 0 {
		merged.allow = normalizeStringList(next.allow)
	}
	if len(next.alsoAllow) > 0 {
		if len(merged.allow) == 0 {
			merged.allow = normalizeStringList(next.alsoAllow)
		} else {
			merged.allow = mergeUniqueStrings(merged.allow, next.alsoAllow)
		}
	}
	if len(next.deny) > 0 {
		merged.deny = mergeUniqueStrings(merged.deny, next.deny)
	}
	return merged
}

func splitModelRef(value string) (string, string) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", ""
	}
	parts := strings.SplitN(trimmed, "/", 2)
	if len(parts) == 2 {
		return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	}
	return "", trimmed
}

func extractTask(payload any) (string, string) {
	if payload == nil {
		return "", ""
	}
	if raw, ok := payload.(string); ok {
		return strings.TrimSpace(raw), ""
	}
	task := ""
	contextText := ""
	for _, key := range []string{"task", "message", "prompt", "input"} {
		if value, ok := extractPayloadString(payload, key); ok {
			task = value
			break
		}
	}
	if value, ok := extractPayloadString(payload, "context"); ok {
		contextText = value
	} else if value, ok := extractPayloadString(payload, "details"); ok {
		contextText = value
	}
	return task, contextText
}

func extractPayloadString(payload any, key string) (string, bool) {
	if payload == nil || key == "" {
		return "", false
	}
	switch typed := payload.(type) {
	case map[string]any:
		value, ok := typed[key]
		if !ok {
			return "", false
		}
		return coerceString(value)
	case map[string]string:
		value, ok := typed[key]
		if !ok {
			return "", false
		}
		value = strings.TrimSpace(value)
		if value == "" {
			return "", false
		}
		return value, true
	default:
		return "", false
	}
}

func extractPayloadInt(payload any, key string) (int, bool) {
	if payload == nil || key == "" {
		return 0, false
	}
	switch typed := payload.(type) {
	case map[string]any:
		value, ok := typed[key]
		if !ok {
			return 0, false
		}
		return coerceInt(value)
	case map[string]string:
		value, ok := typed[key]
		if !ok {
			return 0, false
		}
		parsed, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

func extractPayloadBool(payload any, key string) (bool, bool) {
	if payload == nil || key == "" {
		return false, false
	}
	switch typed := payload.(type) {
	case map[string]any:
		value, ok := typed[key]
		if !ok {
			return false, false
		}
		switch flag := value.(type) {
		case bool:
			return flag, true
		case string:
			trimmed := strings.TrimSpace(strings.ToLower(flag))
			if trimmed == "true" || trimmed == "1" || trimmed == "yes" {
				return true, true
			}
			if trimmed == "false" || trimmed == "0" || trimmed == "no" {
				return false, true
			}
		}
	case map[string]string:
		value, ok := typed[key]
		if !ok {
			return false, false
		}
		trimmed := strings.TrimSpace(strings.ToLower(value))
		if trimmed == "true" || trimmed == "1" || trimmed == "yes" {
			return true, true
		}
		if trimmed == "false" || trimmed == "0" || trimmed == "no" {
			return false, true
		}
	}
	return false, false
}

func coerceString(value any) (string, bool) {
	switch typed := value.(type) {
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed == "" {
			return "", false
		}
		return trimmed, true
	case []byte:
		trimmed := strings.TrimSpace(string(typed))
		if trimmed == "" {
			return "", false
		}
		return trimmed, true
	default:
		return "", false
	}
}

func coerceInt(value any) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, true
	case int64:
		return int(typed), true
	case float64:
		return int(typed), true
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed == "" {
			return 0, false
		}
		parsed, err := strconv.Atoi(trimmed)
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

type subagentLimits struct {
	maxDepth      int
	maxChildren   int
	maxConcurrent int
}

type subagentDefaults struct {
	model    string
	thinking string
}

func (gateway *GatewayService) resolveSubagentLimits(ctx context.Context) subagentLimits {
	limits := subagentLimits{
		maxDepth:      domainsettings.DefaultGatewaySubagentMaxDepth,
		maxChildren:   domainsettings.DefaultGatewaySubagentMaxChildren,
		maxConcurrent: domainsettings.DefaultGatewaySubagentMaxConcurrent,
	}
	if gateway == nil || gateway.settings == nil {
		return limits
	}
	current, err := gateway.settings.GetSettings(ctx)
	if err != nil {
		return limits
	}
	if current.Gateway.Subagents.MaxDepth > 0 {
		limits.maxDepth = current.Gateway.Subagents.MaxDepth
	}
	if current.Gateway.Subagents.MaxChildren > 0 {
		limits.maxChildren = current.Gateway.Subagents.MaxChildren
	}
	if current.Gateway.Subagents.MaxConcurrent > 0 {
		limits.maxConcurrent = current.Gateway.Subagents.MaxConcurrent
	}
	return limits
}

func (gateway *GatewayService) resolveSubagentDefaults(ctx context.Context) subagentDefaults {
	defaults := subagentDefaults{}
	if gateway == nil || gateway.settings == nil {
		return defaults
	}
	current, err := gateway.settings.GetSettings(ctx)
	if err != nil {
		return defaults
	}
	defaults.model = strings.TrimSpace(current.Gateway.Subagents.Model)
	defaults.thinking = strings.TrimSpace(current.Gateway.Subagents.Thinking)
	return defaults
}

func normalizeSubagentPayload(payload any) map[string]any {
	if payload == nil {
		return map[string]any{}
	}
	if typed, ok := payload.(map[string]any); ok {
		return typed
	}
	if typed, ok := payload.(map[string]string); ok {
		result := make(map[string]any, len(typed))
		for key, value := range typed {
			result[key] = value
		}
		return result
	}
	if text, ok := payload.(string); ok {
		return map[string]any{"task": strings.TrimSpace(text)}
	}
	return map[string]any{}
}

func (gateway *GatewayService) resolveDepthFromParent(ctx context.Context, parentRunID string) int {
	parentRunID = strings.TrimSpace(parentRunID)
	if parentRunID == "" || gateway == nil || gateway.store == nil {
		return 1
	}
	depth := 2
	current := parentRunID
	for i := 0; i < 16; i++ {
		record, err := gateway.store.Get(ctx, current)
		if err != nil {
			break
		}
		next := strings.TrimSpace(record.ParentRunID)
		if next == "" {
			break
		}
		depth++
		current = next
	}
	return depth
}

func (gateway *GatewayService) countActiveChildren(ctx context.Context, parentSessionKey string, parentRunID string) int {
	if gateway == nil || gateway.store == nil {
		return 0
	}
	parentSessionKey = strings.TrimSpace(parentSessionKey)
	if parentSessionKey == "" {
		return 0
	}
	records, err := gateway.store.ListByParent(ctx, parentSessionKey)
	if err != nil {
		return 0
	}
	count := 0
	for _, record := range records {
		if record.Status != subagentservice.RunStatusRunning {
			continue
		}
		if parentRunID != "" {
			if strings.TrimSpace(record.ParentRunID) != strings.TrimSpace(parentRunID) {
				continue
			}
		} else if strings.TrimSpace(record.ParentRunID) != "" {
			continue
		}
		count++
	}
	return count
}

func (gateway *GatewayService) countActiveBySession(ctx context.Context, parentSessionKey string) int {
	if gateway == nil || gateway.store == nil {
		return 0
	}
	parentSessionKey = strings.TrimSpace(parentSessionKey)
	if parentSessionKey == "" {
		return 0
	}
	records, err := gateway.store.ListByParent(ctx, parentSessionKey)
	if err != nil {
		return 0
	}
	count := 0
	for _, record := range records {
		if record.Status == subagentservice.RunStatusRunning {
			count++
		}
	}
	return count
}

func normalizeStringList(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func mergeUniqueStrings(base []string, extra []string) []string {
	if len(base) == 0 && len(extra) == 0 {
		return nil
	}
	merged := make([]string, 0, len(base)+len(extra))
	seen := make(map[string]struct{}, len(base)+len(extra))
	for _, value := range base {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		merged = append(merged, trimmed)
	}
	for _, value := range extra {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		merged = append(merged, trimmed)
	}
	if len(merged) == 0 {
		return nil
	}
	return merged
}

func envBool(key string, fallback bool) bool {
	raw := strings.TrimSpace(strings.ToLower(os.Getenv(strings.TrimSpace(key))))
	switch raw {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func formatSubagentHeartbeatText(record subagentservice.RunRecord) string {
	result := strings.TrimSpace(record.Result)
	if result == "" {
		result = strings.TrimSpace(record.Summary)
	}
	if result == "" {
		result = "(not available)"
	}
	notes := strings.TrimSpace(record.Notes)
	if notes == "" {
		notes = strings.TrimSpace(record.Error)
	}
	usageText := fmt.Sprintf("input=%d output=%d total=%d", record.Usage.PromptTokens, record.Usage.CompletionTokens, record.Usage.TotalTokens)
	status := announceStatus(record.Status)
	message := fmt.Sprintf(
		"Subagent completion\nStatus: %s\nResult: %s\nRuntime: %s\nUsage: %s\nSessionKey: %s\nSessionID: %s",
		status,
		result,
		formatRuntimeText(record.RuntimeMs),
		usageText,
		strings.TrimSpace(record.ChildSessionKey),
		strings.TrimSpace(record.ChildSessionID),
	)
	if notes != "" {
		message += "\nNotes: " + notes
	}
	if transcript := strings.TrimSpace(record.TranscriptPath); transcript != "" {
		message += "\nTranscript: " + transcript
	}
	return message
}

func isSubagentFailureStatus(status subagentservice.RunStatus) bool {
	switch status {
	case subagentservice.RunStatusFailed, subagentservice.RunStatusTimeout, subagentservice.RunStatusAborted:
		return true
	default:
		return false
	}
}

func announceStatus(status subagentservice.RunStatus) string {
	switch status {
	case subagentservice.RunStatusSuccess:
		return "success"
	case subagentservice.RunStatusTimeout:
		return "timeout"
	case subagentservice.RunStatusAborted:
		return "aborted"
	case subagentservice.RunStatusFailed:
		return "error"
	case subagentservice.RunStatusRunning:
		return "running"
	default:
		return "unknown"
	}
}

func formatRuntimeText(runtimeMs int64) string {
	if runtimeMs <= 0 {
		return "0s"
	}
	return (time.Duration(runtimeMs) * time.Millisecond).String()
}

func classifyRunOutcome(record subagentservice.RunRecord, runtimeMs int64, err error) (subagentservice.RunStatus, string) {
	if err == nil {
		return subagentservice.RunStatusSuccess, ""
	}
	text := strings.ToLower(strings.TrimSpace(err.Error()))
	if text == "" {
		return subagentservice.RunStatusFailed, ""
	}
	if strings.Contains(text, "deadline exceeded") || strings.Contains(text, "timeout") {
		return subagentservice.RunStatusTimeout, fmt.Sprintf("timed out after %ds", maxInt(record.RunTimeoutSeconds, int(runtimeMs/1000)))
	}
	if strings.Contains(text, "canceled") || strings.Contains(text, "abort") || strings.Contains(text, "killed") {
		if record.RunTimeoutSeconds > 0 && runtimeMs >= int64(record.RunTimeoutSeconds)*int64(time.Second/time.Millisecond) {
			return subagentservice.RunStatusTimeout, fmt.Sprintf("timed out after %ds", record.RunTimeoutSeconds)
		}
		return subagentservice.RunStatusAborted, "aborted"
	}
	return subagentservice.RunStatusFailed, strings.TrimSpace(err.Error())
}

func maxInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

func buildChildSessionKey(agentID string, sessionID string) string {
	normalizedAgentID := strings.TrimSpace(agentID)
	if normalizedAgentID == "" {
		normalizedAgentID = "default"
	}
	normalizedSessionID := strings.TrimSpace(sessionID)
	if normalizedSessionID == "" {
		normalizedSessionID = uuid.NewString()
	}
	return fmt.Sprintf("agent:%s:subagent:%s", normalizedAgentID, normalizedSessionID)
}

func (gateway *GatewayService) scheduleArchive(record subagentservice.RunRecord) {
	if gateway == nil {
		return
	}
	if record.Status == subagentservice.RunStatusRunning {
		return
	}
	delay := time.Duration(defaultArchiveAfterMinutes) * time.Minute
	if record.CleanupPolicy == subagentservice.CleanupDelete {
		delay = 0
	}
	gateway.archiveMu.Lock()
	if existing, ok := gateway.archiveTimers[record.RunID]; ok {
		existing.Stop()
		delete(gateway.archiveTimers, record.RunID)
	}
	runID := strings.TrimSpace(record.RunID)
	if runID == "" {
		gateway.archiveMu.Unlock()
		return
	}
	timer := time.AfterFunc(delay, func() {
		gateway.archiveRun(context.Background(), record)
		gateway.archiveMu.Lock()
		delete(gateway.archiveTimers, runID)
		gateway.archiveMu.Unlock()
	})
	gateway.archiveTimers[runID] = timer
	gateway.archiveMu.Unlock()
}

func (gateway *GatewayService) archiveRun(ctx context.Context, record subagentservice.RunRecord) {
	if gateway == nil {
		return
	}
	threadID := strings.TrimSpace(record.ChildSessionID)
	if threadID == "" {
		return
	}
	lifecycle, ok := gateway.threads.(ThreadLifecycleWriter)
	if !ok || lifecycle == nil {
		return
	}
	_ = lifecycle.SetThreadStatus(ctx, threaddto.SetThreadStatusRequest{
		ThreadID: threadID,
		Status:   domainthread.ThreadStatusArchived,
	})
	_ = lifecycle.SoftDeleteThread(ctx, threadID)
	now := gateway.now()
	record.ArchivedAt = &now
	record.UpdatedAt = now
	if gateway.store != nil {
		_ = gateway.store.Save(ctx, record)
	}
	if gateway.spawner != nil {
		gateway.spawner.Update(record)
	}
}
