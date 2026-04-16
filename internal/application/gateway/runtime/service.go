package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"dreamcreator/internal/application/agentruntime"
	assistantdto "dreamcreator/internal/application/assistant/dto"
	assistantservice "dreamcreator/internal/application/assistant/service"
	appevents "dreamcreator/internal/application/events"
	"dreamcreator/internal/application/gateway/queue"
	"dreamcreator/internal/application/gateway/runtime/dto"
	gatewaytools "dreamcreator/internal/application/gateway/tools"
	gatewayusage "dreamcreator/internal/application/gateway/usage"
	memorydto "dreamcreator/internal/application/memory/dto"
	appsession "dreamcreator/internal/application/session"
	settingsdto "dreamcreator/internal/application/settings/dto"
	settingsservice "dreamcreator/internal/application/settings/service"
	skillsservice "dreamcreator/internal/application/skills/service"
	threaddto "dreamcreator/internal/application/thread/dto"
	tooldto "dreamcreator/internal/application/tools/dto"
	workspacedto "dreamcreator/internal/application/workspace/dto"
	workspaceservice "dreamcreator/internal/application/workspace/service"
	"dreamcreator/internal/domain/providers"
	"dreamcreator/internal/domain/thread"
	"dreamcreator/internal/infrastructure/llm"
)

var (
	ErrInvalidRequest = errors.New("invalid runtime request")
)

const (
	defaultRuntimeThreadTitle = "New Chat"
	usageSourceDialogue       = "dialogue"
	usageSourceRelay          = "relay"
	usageSourceOneShot        = "one-shot"
	usageSourceUnknown        = "unknown"
)

type RunEventSink interface {
	Publish(ctx context.Context, event thread.ThreadRunEvent, sessionKey string)
}

type AbortRegistryStore interface {
	Register(runID string, cancel context.CancelFunc)
	Unregister(runID string)
	Abort(runID string, reason string) bool
}

type ThreadTitleGenerator interface {
	GenerateThreadTitle(ctx context.Context, request threaddto.GenerateThreadTitleRequest) (threaddto.GenerateThreadTitleResponse, error)
}

type MemoryLifecycle interface {
	BuildRecallContext(ctx context.Context, request memorydto.BeforeAgentStartRequest) (memorydto.BeforeAgentStartResult, error)
	HandleAgentEnd(ctx context.Context, request memorydto.AgentEndRequest) error
}

type Telemetry interface {
	TrackUserChatCompleted(ctx context.Context, runID string)
}

type Service struct {
	providers               providers.ProviderRepository
	models                  providers.ModelRepository
	secrets                 providers.SecretRepository
	threads                 thread.Repository
	messages                thread.MessageRepository
	runs                    thread.RunRepository
	runEvents               thread.RunEventRepository
	sessions                *appsession.Service
	queue                   *queue.Manager
	tools                   *gatewaytools.Service
	assistants              *assistantservice.AssistantService
	assistantSnapshots      *assistantservice.AssistantSnapshotResolver
	workspaces              *workspaceservice.WorkspaceService
	skills                  *skillsservice.SkillsService
	settings                *settingsservice.SettingsService
	usage                   *gatewayusage.Service
	events                  RunEventSink
	eventBus                appevents.Bus
	aborts                  AbortRegistryStore
	controls                *ControlRegistry
	threadTitles            ThreadTitleGenerator
	memory                  MemoryLifecycle
	telemetry               Telemetry
	titleGenerationMu       sync.Mutex
	titleGenerationInFlight map[string]struct{}
	chatFactory             *llm.ChatModelFactory
	now                     func() time.Time
	newID                   func() string
}

func NewService(
	providerRepo providers.ProviderRepository,
	modelRepo providers.ModelRepository,
	secretRepo providers.SecretRepository,
	threadRepo thread.Repository,
	messageRepo thread.MessageRepository,
	runRepo thread.RunRepository,
	runEventRepo thread.RunEventRepository,
	sessionService *appsession.Service,
	queueManager *queue.Manager,
	toolsService *gatewaytools.Service,
	assistantService *assistantservice.AssistantService,
	assistantSnapshots *assistantservice.AssistantSnapshotResolver,
	workspaceService *workspaceservice.WorkspaceService,
	skillsService *skillsservice.SkillsService,
	settings *settingsservice.SettingsService,
	usage *gatewayusage.Service,
	eventBus appevents.Bus,
	eventSink RunEventSink,
	abortRegistry AbortRegistryStore,
	controlRegistry *ControlRegistry,
	threadTitleGenerator ThreadTitleGenerator,
	memoryLifecycle MemoryLifecycle,
	telemetry Telemetry,
) *Service {
	return &Service{
		providers:               providerRepo,
		models:                  modelRepo,
		secrets:                 secretRepo,
		threads:                 threadRepo,
		messages:                messageRepo,
		runs:                    runRepo,
		runEvents:               runEventRepo,
		sessions:                sessionService,
		queue:                   queueManager,
		tools:                   toolsService,
		assistants:              assistantService,
		assistantSnapshots:      assistantSnapshots,
		workspaces:              workspaceService,
		skills:                  skillsService,
		settings:                settings,
		usage:                   usage,
		events:                  eventSink,
		eventBus:                eventBus,
		aborts:                  abortRegistry,
		controls:                controlRegistry,
		threadTitles:            threadTitleGenerator,
		memory:                  memoryLifecycle,
		telemetry:               telemetry,
		titleGenerationInFlight: make(map[string]struct{}),
		chatFactory:             llm.NewChatModelFactory(),
		now:                     time.Now,
		newID:                   uuid.NewString,
	}
}

func (service *Service) SetLLMCallRecorder(recorder llm.CallRecorder) {
	if service == nil || service.chatFactory == nil {
		return
	}
	service.chatFactory.SetCallRecorder(recorder)
}

func (service *Service) Start(ctx context.Context, request dto.RuntimeRunRequest) (dto.RuntimeStartResponse, error) {
	if service == nil {
		return dto.RuntimeStartResponse{}, ErrInvalidRequest
	}
	if len(request.Input.Messages) == 0 {
		return dto.RuntimeStartResponse{}, errors.New("runtime input messages required")
	}
	if err := service.ensureAssistantReady(ctx, request, assistantdto.AssistantSnapshot{}); err != nil {
		return dto.RuntimeStartResponse{}, err
	}
	runID := strings.TrimSpace(request.RunID)
	if runID == "" {
		runID = service.newID()
	}
	request.RunID = runID
	go func() {
		_, _ = service.Run(context.Background(), request)
	}()
	return dto.RuntimeStartResponse{RunID: runID, Status: "queued"}, nil
}

func (service *Service) Run(ctx context.Context, request dto.RuntimeRunRequest) (dto.RuntimeRunResult, error) {
	return service.runWithStream(ctx, request, nil)
}

func (service *Service) RunStream(ctx context.Context, request dto.RuntimeRunRequest, callback dto.RuntimeStreamCallback) (dto.RuntimeRunResult, error) {
	return service.runWithStream(ctx, request, callback)
}

func (service *Service) runWithStream(ctx context.Context, request dto.RuntimeRunRequest, callback dto.RuntimeStreamCallback) (dto.RuntimeRunResult, error) {
	if service == nil {
		return dto.RuntimeRunResult{}, ErrInvalidRequest
	}
	sessionID, sessionKey, err := service.resolveSession(request)
	if err != nil {
		return dto.RuntimeRunResult{}, err
	}
	if len(request.Input.Messages) == 0 {
		return dto.RuntimeRunResult{}, errors.New("runtime input messages required")
	}

	flags := resolveRunFlags(request.Metadata)
	runKind := resolveRunKind(request, flags)
	titleGenerationRun, _ := resolveMetadataBool(request.Metadata, "titleGeneration")
	gatewaySettings := settingsdto.GatewaySettings{}
	appLanguage := ""
	if service.settings != nil {
		if current, err := service.settings.GetSettings(ctx); err == nil {
			gatewaySettings = current.Gateway
			appLanguage = strings.TrimSpace(current.Language)
		}
	}
	service.updateQueuePolicy(gatewaySettings)

	assistantSnapshot := assistantdto.AssistantSnapshot{}
	if service.assistantSnapshots != nil {
		snapshot, err := service.assistantSnapshots.ResolveAssistantSnapshot(ctx, assistantdto.ResolveAssistantSnapshotRequest{
			ThreadID:    sessionID,
			AssistantID: strings.TrimSpace(request.AssistantID),
		})
		if err != nil {
			return dto.RuntimeRunResult{}, err
		}
		assistantSnapshot = snapshot
	}
	if assistantSnapshot.AssistantID == "" {
		assistantSnapshot.AssistantID = strings.TrimSpace(request.AssistantID)
	}
	assistantSnapshot.User = service.resolveRuntimeUserLocale(assistantSnapshot.AssistantID, assistantSnapshot.User, appLanguage)
	if err := service.ensureAssistantReady(ctx, request, assistantSnapshot); err != nil {
		return dto.RuntimeRunResult{}, err
	}

	threadItem := thread.Thread{}
	threadJustCreated := false
	if service.threads != nil {
		item, err := service.threads.Get(ctx, sessionID)
		if err != nil {
			if errors.Is(err, thread.ErrThreadNotFound) {
				now := service.now()
				created, createErr := thread.NewThread(thread.ThreadParams{
					ID:             sessionID,
					AssistantID:    strings.TrimSpace(assistantSnapshot.AssistantID),
					Title:          defaultRuntimeThreadTitle,
					TitleIsDefault: true,
					Status:         thread.ThreadStatusRegular,
					CreatedAt:      &now,
					UpdatedAt:      &now,
				})
				if createErr != nil {
					return dto.RuntimeRunResult{}, createErr
				}
				if saveErr := service.threads.Save(ctx, created); saveErr != nil {
					return dto.RuntimeRunResult{}, saveErr
				}
				service.emitThreadUpdated(ctx, sessionID, "upsert", "new-thread")
				item = created
				threadJustCreated = true
			} else {
				return dto.RuntimeRunResult{}, err
			}
		}
		if strings.TrimSpace(assistantSnapshot.AssistantID) != "" && strings.TrimSpace(item.AssistantID) == "" {
			item.AssistantID = strings.TrimSpace(assistantSnapshot.AssistantID)
			item.UpdatedAt = service.now()
			_ = service.threads.Save(ctx, item)
		}
		threadItem = item
	}
	if !titleGenerationRun && !flags.IsSubagent {
		if shouldScheduleThreadTitleGenerationAtRequestStart(threadItem, request.Input.Messages) {
			service.scheduleThreadTitleGenerationAtRequestStart(threadItem, request.Input.Messages)
		}
		defer service.scheduleThreadTitleGenerationAfterRun(sessionID)
	}

	workspaceSnapshot := workspacedto.RuntimeSnapshot{}
	assistantID := strings.TrimSpace(assistantSnapshot.AssistantID)
	service.persistSession(ctx, sessionID, sessionKey, strings.TrimSpace(request.AgentID), assistantID, request.Metadata)
	if threadJustCreated && service.sessions != nil {
		_ = service.sessions.UpdateTitle(ctx, sessionID, strings.TrimSpace(threadItem.Title))
	}
	isAgentRun := strings.TrimSpace(threadItem.AgentID) != "" || strings.TrimSpace(request.AgentID) != ""
	if service.workspaces != nil && assistantID != "" && !isAgentRun {
		useWorkspaceSnapshot := true
		resolveRequest := workspacedto.ResolveRuntimeSnapshotRequest{
			AssistantID:             assistantID,
			ThreadID:                sessionID,
			IncludeWorkspaceContext: useWorkspaceSnapshot,
		}
		if useWorkspaceSnapshot {
			resolveRequest.ForRunID = strings.TrimSpace(request.RunID)
		}
		snapshot, err := service.workspaces.ResolveRuntimeSnapshot(ctx, resolveRequest)
		if err != nil {
			return dto.RuntimeRunResult{}, err
		}
		workspaceSnapshot = snapshot
	}

	promptMode := resolvePromptMode(request.PromptMode, workspaceSnapshot.WorkspaceContext.PromptMode, flags.IsSubagent)
	extraSystemPrompt := resolveExtraSystemPrompt(request.Metadata)
	if service.memory != nil && !flags.IsSubagent && !titleGenerationRun {
		recallQuery := resolveMemoryRecallQuery(request.Input.Messages)
		if recallQuery != "" {
			memoryIdentity := memorydto.MemoryIdentity{
				AssistantID: assistantID,
				ThreadID:    sessionID,
				Channel:     resolveMetadataString(request.Metadata, "channel"),
				AccountID:   resolveMetadataString(request.Metadata, "accountId"),
				UserID:      resolveMemoryUserID(request.Metadata),
				GroupID:     resolveMemoryGroupID(request.Metadata),
			}
			recallResult, recallErr := service.memory.BuildRecallContext(ctx, memorydto.BeforeAgentStartRequest{
				Identity: memoryIdentity,
				Query:    recallQuery,
			})
			if recallErr != nil {
				zap.L().Warn("runtime memory before_agent_start failed",
					zap.String("sessionID", sessionID),
					zap.String("assistantID", assistantID),
					zap.Error(recallErr),
				)
			} else if block := strings.TrimSpace(recallResult.InjectedContext); block != "" {
				if strings.TrimSpace(extraSystemPrompt) != "" {
					extraSystemPrompt = strings.TrimSpace(extraSystemPrompt) + "\n\n" + block
				} else {
					extraSystemPrompt = block
				}
			}
		}
	}
	resolvedModel, chatModel, err := service.resolveRunModel(ctx, request.Model, assistantSnapshot.Model)
	if err != nil {
		return dto.RuntimeRunResult{}, err
	}
	thinkingLevel := resolveThinkingLevel(request.Thinking, request.Metadata)
	toolConfig := mergeToolExecutionConfig(request.Tools, assistantSnapshot.Call, flags.IsSubagent)

	var availableToolSpecs []tooldto.ToolSpec
	if service.tools != nil {
		availableToolSpecs = service.tools.ListTools(ctx)
	}
	toolSpecs := service.filterToolSpecs(availableToolSpecs, toolConfig, assistantSnapshot.Tools)
	skillItems := service.resolveSkillPromptItems(
		ctx,
		resolvedModel.ProviderID,
		assistantSnapshot.Call,
		assistantSnapshot.Skills,
		promptMode,
		flags.IsSubagent,
		workspaceSnapshot.RootPath,
	)
	skillItems = limitSkillPromptItems(skillItems, assistantSnapshot.Skills)
	channel := resolveMetadataString(request.Metadata, "channel")
	usageSource := resolveUsageSource(request.Metadata, channel)
	promptDoc, promptReport, promptSections := buildPromptDocument(promptBuildInput{
		Mode:              promptMode,
		RunKind:           runKind,
		Assistant:         assistantSnapshot,
		Workspace:         workspaceSnapshot,
		Tools:             toolSpecs,
		Skills:            skillItems,
		HeartbeatPrompt:   strings.TrimSpace(gatewaySettings.Heartbeat.Prompt),
		IsSubagent:        flags.IsSubagent,
		ExtraSystemPrompt: extraSystemPrompt,
		Runtime: runtimePromptInfo{
			SessionID:     sessionID,
			SessionKey:    sessionKey,
			Channel:       channel,
			RunID:         strings.TrimSpace(request.RunID),
			WorkspaceID:   workspaceSnapshot.AssistantID,
			WorkspaceRoot: workspaceSnapshot.RootPath,
		},
	})
	systemPrompt := strings.TrimSpace(promptDoc.Content)

	run, err := service.startRun(ctx, sessionID, request.AgentID, request.RunID, flags.PersistRun)
	if err != nil {
		return dto.RuntimeRunResult{}, err
	}

	runCtx, cancel := context.WithCancel(ctx)
	runCtx = llm.WithRuntimeParams(runCtx, llm.RuntimeParams{
		SessionID:        sessionID,
		ThreadID:         sessionID,
		RunID:            run.ID,
		RequestSource:    usageSource,
		Operation:        resolveLLMOperation(runKind, flags.IsSubagent, titleGenerationRun),
		ProviderID:       resolvedModel.ProviderID,
		ModelName:        resolvedModel.ModelName,
		ThinkingLevel:    thinkingLevel,
		StructuredOutput: resolveStructuredOutputConfig(request.Metadata),
	})
	if service.aborts != nil {
		service.aborts.Register(run.ID, cancel)
		defer service.aborts.Unregister(run.ID)
	}
	defer cancel()

	if service.queue != nil && flags.UseQueue {
		ticket, _, err := service.queue.Enqueue(runCtx, queue.EnqueueRequest{
			SessionKey: sessionKey,
			Mode:       resolveQueueMode(request),
			Payload: map[string]any{
				"runId": run.ID,
			},
		})
		if err != nil {
			_ = service.failRun(runCtx, run, err)
			return dto.RuntimeRunResult{}, err
		}
		if ticket.TicketID != "" {
			if err := service.queue.Wait(runCtx, ticket.TicketID); err != nil {
				_ = service.failRun(runCtx, run, err)
				return dto.RuntimeRunResult{}, err
			}
			defer service.queue.Done(runCtx, ticket.TicketID)
		}
	}

	persistedIncomingUserMessage := false
	if flags.PersistMessages {
		persisted, err := service.persistIncomingMessages(runCtx, sessionID, request.Input.Messages, request.Input.ReplaceHistory)
		if err != nil {
			if flags.PersistRun {
				_ = service.failRun(runCtx, run, err)
			}
			return dto.RuntimeRunResult{}, err
		}
		persistedIncomingUserMessage = persisted
		if persistedIncomingUserMessage {
			service.emitThreadUpdated(runCtx, sessionID, "upsert", "append-message")
		}
	}
	runLane := resolveRunLane(request.Metadata, flags.IsSubagent)
	if service.queue != nil {
		if err := service.queue.AcquireLane(runCtx, runLane); err != nil {
			if flags.PersistRun {
				_ = service.failRun(runCtx, run, err)
			}
			return dto.RuntimeRunResult{}, err
		}
		defer service.queue.ReleaseLane(runLane)
	}

	streamFn := agentruntime.StreamFunction(chatModel.Stream)
	policyCtx := tooldto.ToolPolicyContext{
		SessionKey:      sessionKey,
		AgentID:         strings.TrimSpace(request.AgentID),
		ProviderID:      strings.TrimSpace(resolvedModel.ProviderID),
		Source:          "runtime",
		IsSubagent:      flags.IsSubagent,
		RequireSandbox:  toolConfig.RequireSandbox,
		RequireApproval: toolConfig.RequireApproval,
	}
	toolInfos, toolAdapters := service.resolveToolAdapters(runCtx, sessionKey, run.ID, toolConfig, assistantSnapshot.Tools, policyCtx)
	if len(toolInfos) > 0 {
		if toolModel, ok := resolveToolCallingModel(chatModel); ok {
			if bound, bindErr := toolModel.WithTools(toolInfos); bindErr == nil {
				streamFn = agentruntime.StreamFunction(bound.Stream)
			}
		}
	}

	controller := agentruntime.NewAgentController()
	timeout := resolveLoopTimeout(request.Metadata, flags.IsSubagent)
	if timeout > 0 {
		controller.SetTimeout(timeout)
	}
	if service.controls != nil {
		service.controls.Register(run.ID, controller)
		defer service.controls.Unregister(run.ID)
	}
	maxSteps := resolveLoopMaxSteps(request.Metadata, flags.IsSubagent)
	if maxSteps == 0 && gatewaySettings.Runtime.MaxSteps > 0 {
		maxSteps = gatewaySettings.Runtime.MaxSteps
	}
	toolLoopConfig := resolveToolLoopDetectionConfig(request.Metadata, gatewaySettings.Runtime.ToolLoopDetection)
	toolLoopDetector := agentruntime.NewToolLoopDetector(toolLoopConfig)

	emitEvent := func(event agentruntime.Event) {
		if !flags.PersistEvents {
			return
		}
		service.emitRuntimeEvent(runCtx, run, sessionKey, event)
	}
	inputMessages := dtoMessagesToSchema(request.Input.Messages)

	if flags.PersistEvents {
		service.emitPromptReport(
			runCtx,
			run,
			sessionKey,
			promptMode,
			gatewaySettings.Runtime.RecordPrompt,
			promptDoc,
			promptReport,
			promptSections,
			inputMessages,
			toolSpecs,
			skillItems,
		)
	}
	contextWindowTokens := service.resolveContextWindowTokens(runCtx, resolvedModel.ProviderID, resolvedModel.ModelName, request.Metadata)
	contextConfig := resolveContextGuardConfig(gatewaySettings.Runtime, contextWindowTokens)
	contextConfig.extraTokens = estimateToolSpecTokens(toolSpecs)
	contextLimit := contextConfig.contextWindowTokens
	guardState := &contextGuardState{}
	if service.sessions != nil {
		if sessionEntry, getErr := service.sessions.Get(runCtx, sessionID); getErr == nil {
			guardState.compactionCount = sessionEntry.CompactionCount
			guardState.memoryFlushCompactionCount = sessionEntry.MemoryFlushCompactionCount
		}
	}
	hooks := contextGuardHooks{
		Summarize: func(ctx context.Context, params compactionSummaryParams) (string, error) {
			return service.summarizeForCompaction(ctx, chatModel, params)
		},
		MemoryFlush: func(ctx context.Context, params memoryFlushParams) (string, error) {
			return service.runMemoryFlush(ctx, params, streamFn, toolAdapters)
		},
		PersistCompactionState: func(ctx context.Context, compactionCount int, memoryFlushCompactionCount int) error {
			if service.sessions == nil {
				return nil
			}
			return service.sessions.UpdateCompactionCounters(ctx, sessionID, compactionCount, memoryFlushCompactionCount)
		},
	}

	loop := &agentruntime.AgentLoop{
		StreamFunction: streamFn,
		TransformContext: func(loopCtx context.Context, state agentruntime.AgentState) (agentruntime.AgentState, error) {
			updated, report, err := applyContextGuard(loopCtx, state, contextConfig, guardState, hooks)
			if report.totalTokens > 0 {
				metadata := map[string]any{}
				if report.truncatedResults > 0 {
					metadata["truncatedToolResults"] = report.truncatedResults
				}
				if report.droppedMessages > 0 {
					metadata["compactionDropped"] = report.droppedMessages
					if contextConfig.compactionMode != "" {
						metadata["compactionMode"] = contextConfig.compactionMode
					}
				}
				if report.compactedResults > 0 {
					metadata["compactedToolResults"] = report.compactedResults
				}
				if report.memoryFlushed {
					metadata["memoryFlushed"] = true
				}
				if report.memoryFlushError != "" {
					metadata["memoryFlushError"] = report.memoryFlushError
				}
				if report.compactionTimedOut {
					metadata["compactionTimedOut"] = true
				}
				if contextConfig.warnTokens > 0 && contextConfig.contextWindowTokens > 0 && contextConfig.contextWindowTokens < contextConfig.warnTokens {
					metadata["warn"] = true
				}
				if err != nil {
					metadata["blocked"] = true
				}
				emitEvent(agentruntime.Event{
					Type: agentruntime.EventContextSnapshot,
					Step: state.CurrentLoopStep,
					ContextTokens: &agentruntime.ContextTokenSnapshot{
						PromptTokens:       report.promptTokens,
						TotalTokens:        report.totalTokens,
						ContextLimitTokens: contextLimit,
						WarnTokens:         contextConfig.warnTokens,
						HardTokens:         contextConfig.hardTokens,
					},
					Metadata: metadata,
				})
			}
			return updated, err
		},
		ConvertToLlm: func(_ context.Context, state agentruntime.AgentState) ([]*schema.Message, error) {
			return state.Messages, nil
		},
		ToolExecutor: &agentruntime.ToolExecutor{
			Validator: agentruntime.JSONToolValidator{},
			Tools:     toolAdapters,
			Emit: func(event agentruntime.Event) {
				emitEvent(event)
			},
		},
		Controller: controller,
		BuildOptions: func() []model.Option {
			return service.buildChatOptions(runCtx, resolvedModel.Config, request.Metadata)
		},
		Emit: func(event agentruntime.Event) {
			emitEvent(event)
		},
		MaxSteps:         maxSteps,
		ToolLoopDetector: toolLoopDetector,
	}

	stream, err := loop.RunStream(runCtx, agentruntime.AgentState{
		Messages:     inputMessages,
		SystemPrompt: systemPrompt,
		IsStreaming:  true,
	})
	if err != nil {
		if flags.PersistMessages {
			if persisted := service.persistAssistantFailureMessage(runCtx, sessionID, run.AssistantMessageID, err); persisted {
				service.emitThreadUpdated(runCtx, sessionID, "upsert", "append-message")
			}
		}
		if flags.PersistRun {
			_ = service.failRun(runCtx, run, err)
		}
		return dto.RuntimeRunResult{}, err
	}

	content, parts, finishReason, usage, err := consumeAgentLoopStream(stream, callback)
	if err != nil {
		if flags.PersistContextSnapshot {
			service.persistSessionContextSnapshot(runCtx, sessionID, usage)
		}
		if flags.PersistMessages {
			if persisted := service.persistAssistantFailureMessage(runCtx, sessionID, run.AssistantMessageID, err); persisted {
				service.emitThreadUpdated(runCtx, sessionID, "upsert", "append-message")
			}
		}
		if flags.PersistRun {
			_ = service.failRun(runCtx, run, err)
		}
		return dto.RuntimeRunResult{}, err
	}
	if flags.PersistContextSnapshot {
		service.persistSessionContextSnapshot(runCtx, sessionID, usage)
	}

	if flags.PersistMessages {
		if err := service.persistAssistantMessage(runCtx, sessionID, run.AssistantMessageID, content, parts); err != nil {
			if flags.PersistRun {
				_ = service.failRun(runCtx, run, err)
			}
			return dto.RuntimeRunResult{}, err
		}
		service.emitThreadUpdated(runCtx, sessionID, "upsert", "append-message")
	}

	if flags.PersistRun {
		run.Status = thread.RunStatusFinished
		run.UpdatedAt = service.now()
		if service.runs != nil {
			_ = service.runs.Save(runCtx, run)
		}
	}
	if flags.PersistUsage {
		service.ingestUsage(runCtx, usage, resolvedModel, channel, usageSource, run.ID)
	}
	if service.memory != nil && !flags.IsSubagent && !titleGenerationRun {
		memoryIdentity := memorydto.MemoryIdentity{
			AssistantID: assistantID,
			ThreadID:    sessionID,
			Channel:     resolveMetadataString(request.Metadata, "channel"),
			AccountID:   resolveMetadataString(request.Metadata, "accountId"),
			UserID:      resolveMemoryUserID(request.Metadata),
			GroupID:     resolveMemoryGroupID(request.Metadata),
		}
		hookRequest := memorydto.AgentEndRequest{
			Identity: memoryIdentity,
			RunID:    run.ID,
			Messages: buildMemoryLifecycleMessages(request.Input.Messages, content),
		}
		go func() {
			hookCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if err := service.memory.HandleAgentEnd(hookCtx, hookRequest); err != nil {
				zap.L().Warn("runtime memory agent_end failed",
					zap.String("sessionID", memoryIdentity.ThreadID),
					zap.String("assistantID", memoryIdentity.AssistantID),
					zap.String("runID", hookRequest.RunID),
					zap.Error(err),
				)
			}
		}()
	}
	if service.telemetry != nil && runKind == "user" && !flags.IsSubagent && !titleGenerationRun {
		service.telemetry.TrackUserChatCompleted(runCtx, run.ID)
	}

	return dto.RuntimeRunResult{
		Status: "completed",
		AssistantMessage: dto.Message{
			ID:      run.AssistantMessageID,
			Role:    "assistant",
			Content: content,
			Parts:   parts,
		},
		FinishReason: finishReason,
		Model: &dto.ModelSelection{
			ProviderID: resolvedModel.ProviderID,
			Name:       resolvedModel.ModelName,
		},
		FinishedAt:  service.now(),
		Usage:       usage,
		Error:       "",
		ErrorDetail: nil,
	}, nil
}

func resolveMetadataString(metadata map[string]any, key string) string {
	if metadata == nil || key == "" {
		return ""
	}
	value, ok := metadata[key]
	if !ok {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	default:
		return strings.TrimSpace(toString(value))
	}
}

func resolveMemoryUserID(metadata map[string]any) string {
	userID := resolveMetadataString(metadata, "userId")
	if strings.TrimSpace(userID) != "" {
		return strings.TrimSpace(userID)
	}
	peerKind := strings.ToLower(strings.TrimSpace(resolveMetadataString(metadata, "peerKind")))
	peerID := strings.TrimSpace(resolveMetadataString(metadata, "peerId"))
	if peerID == "" {
		return ""
	}
	switch peerKind {
	case "direct", "private", "user", "dm":
		return peerID
	default:
		return ""
	}
}

func resolveMemoryGroupID(metadata map[string]any) string {
	groupID := resolveMetadataString(metadata, "groupId")
	if strings.TrimSpace(groupID) != "" {
		return strings.TrimSpace(groupID)
	}
	peerKind := strings.ToLower(strings.TrimSpace(resolveMetadataString(metadata, "peerKind")))
	peerID := strings.TrimSpace(resolveMetadataString(metadata, "peerId"))
	if peerID == "" {
		peerID = strings.TrimSpace(resolveMetadataString(metadata, "chatId"))
	}
	switch peerKind {
	case "group", "supergroup", "room", "channel":
		return peerID
	default:
		return ""
	}
}

func toString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case []byte:
		return string(typed)
	default:
		encoded, _ := json.Marshal(typed)
		return string(encoded)
	}
}

func resolveQueueMode(request dto.RuntimeRunRequest) string {
	mode := resolveMetadataString(request.Metadata, "queueMode")
	if mode != "" {
		return mode
	}
	return strings.TrimSpace(request.Tools.Mode)
}
