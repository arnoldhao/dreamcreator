package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"dreamcreator/internal/application/agentruntime"
	"dreamcreator/internal/application/chatevent"
	appevents "dreamcreator/internal/application/events"
	gatewayevents "dreamcreator/internal/application/gateway/events"
	runtimedto "dreamcreator/internal/application/gateway/runtime/dto"
	memorydto "dreamcreator/internal/application/memory/dto"
	"dreamcreator/internal/application/runtimeconfig"
	appsession "dreamcreator/internal/application/session"
	"dreamcreator/internal/application/thread/dto"
	domainassistant "dreamcreator/internal/domain/assistant"
	domainproviders "dreamcreator/internal/domain/providers"
	domainsession "dreamcreator/internal/domain/session"
	"dreamcreator/internal/domain/thread"
)

const (
	defaultPurgeDelay          = 7 * 24 * time.Hour
	defaultThreadTitle         = "New Chat"
	titleRunTimeout            = runtimeconfig.DefaultAuxiliaryLLMTimeout
	titleRunThinkingLevel      = "off"
	titleRunTemperature        = 0.2
	titleRunMaxTokens          = 256
	titleRunSourceRoundLimit   = 3
	titleRunSourceCharsPerLine = 240
	threadTitleEventTopic      = "thread.title"
	threadTitleEventStarted    = "thread.title.oneshot.started"
	threadTitleEventSkipped    = "thread.title.oneshot.skipped"
	threadTitleEventFailed     = "thread.title.oneshot.failed"
	threadTitleEventEmpty      = "thread.title.oneshot.empty"
	threadTitleEventCompleted  = "thread.title.oneshot.completed"
)

type ThreadService struct {
	threads       thread.Repository
	messages      thread.MessageRepository
	runs          thread.RunRepository
	runEvents     thread.RunEventRepository
	sessions      SessionService
	assistants    AssistantRepository
	models        ModelRepository
	runtime       ThreadTitleRuntime
	memory        ThreadMemoryLifecycle
	gatewayEvents *gatewayevents.Broker
	now           func() time.Time
	newID         func() string
}

type SessionService interface {
	CreateSession(ctx context.Context, request appsession.CreateSessionRequest) (domainsession.Entry, error)
	UpdateTitle(ctx context.Context, sessionID, title string) error
	SetStatus(ctx context.Context, sessionID string, status domainsession.Status) error
	Get(ctx context.Context, sessionID string) (domainsession.Entry, error)
}

type AssistantRepository interface {
	Get(ctx context.Context, id string) (domainassistant.Assistant, error)
}

type ModelRepository interface {
	ListByProvider(ctx context.Context, providerID string) ([]domainproviders.Model, error)
}

type ThreadTitleRuntime interface {
	RunOneShot(ctx context.Context, request runtimedto.RuntimeRunRequest) (runtimedto.RuntimeRunResult, error)
}

type ThreadMemoryLifecycle interface {
	HandleSessionLifecycle(ctx context.Context, request memorydto.SessionLifecycleRequest) error
}

func NewThreadService(
	threadRepo thread.Repository,
	messageRepo thread.MessageRepository,
	runRepo thread.RunRepository,
	runEventRepo thread.RunEventRepository,
	sessionService SessionService,
	assistantRepo AssistantRepository,
	modelRepo ModelRepository,
) *ThreadService {
	return &ThreadService{
		threads:    threadRepo,
		messages:   messageRepo,
		runs:       runRepo,
		runEvents:  runEventRepo,
		sessions:   sessionService,
		assistants: assistantRepo,
		models:     modelRepo,
		now:        time.Now,
		newID:      uuid.NewString,
	}
}

func (service *ThreadService) SetTitleRuntime(runtime ThreadTitleRuntime) {
	if service == nil {
		return
	}
	service.runtime = runtime
}

func (service *ThreadService) SetMemoryLifecycle(memory ThreadMemoryLifecycle) {
	if service == nil {
		return
	}
	service.memory = memory
}

func (service *ThreadService) SetGatewayEventBroker(events *gatewayevents.Broker) {
	if service == nil {
		return
	}
	service.gatewayEvents = events
}

func (service *ThreadService) NewThread(ctx context.Context, request dto.NewThreadRequest) (dto.NewThreadResponse, error) {
	now := service.now()
	title := strings.TrimSpace(request.Title)
	titleIsDefault := request.IsDefaultTitle
	titleChangedBy := thread.TitleChangedBy("")
	assistantID := strings.TrimSpace(request.AssistantID)
	if title == "" {
		title = defaultThreadTitle
		titleIsDefault = true
	}
	if !titleIsDefault {
		titleChangedBy = thread.ThreadTitleChangedByUser
	}
	if assistantID == "" {
		return dto.NewThreadResponse{}, errors.New("assistant id is required")
	}

	threadID := ""
	if service.sessions != nil {
		sessionEntry, err := service.sessions.CreateSession(ctx, appsession.CreateSessionRequest{
			AssistantID: assistantID,
			Title:       title,
			KeyParts: domainsession.KeyParts{
				Channel:   "aui",
				PrimaryID: service.newID(),
				ThreadRef: "",
			},
			Origin: domainsession.Origin{
				Channel:   "aui",
				ThreadRef: "",
			},
		})
		if err != nil {
			return dto.NewThreadResponse{}, err
		}
		threadID = sessionEntry.SessionID
	} else {
		threadID = service.newID()
	}

	item, err := thread.NewThread(thread.ThreadParams{
		ID:                threadID,
		AssistantID:       assistantID,
		Title:             title,
		TitleIsDefault:    titleIsDefault,
		TitleChangedBy:    titleChangedBy,
		Status:            thread.ThreadStatusRegular,
		CreatedAt:         &now,
		UpdatedAt:         &now,
		LastInteractiveAt: &now,
	})
	if err != nil {
		return dto.NewThreadResponse{}, err
	}
	if err := service.threads.Save(ctx, item); err != nil {
		return dto.NewThreadResponse{}, err
	}

	return dto.NewThreadResponse{
		ThreadID:    threadID,
		AssistantID: assistantID,
	}, nil
}

func (service *ThreadService) RenameThread(ctx context.Context, request dto.RenameThreadRequest) error {
	threadID := strings.TrimSpace(request.ThreadID)
	if threadID == "" {
		return errors.New("thread id is required")
	}
	title := strings.TrimSpace(request.Title)
	if title == "" {
		return errors.New("title is required")
	}

	item, err := service.threads.Get(ctx, threadID)
	if err != nil {
		return err
	}
	item.Title = title
	item.TitleIsDefault = false
	item.TitleChangedBy = thread.ThreadTitleChangedByUser
	item.UpdatedAt = service.now()
	if err := service.threads.Save(ctx, item); err != nil {
		return err
	}
	if service.sessions != nil {
		_ = service.sessions.UpdateTitle(ctx, threadID, title)
	}
	return nil
}

func (service *ThreadService) GenerateThreadTitle(ctx context.Context, request dto.GenerateThreadTitleRequest) (dto.GenerateThreadTitleResponse, error) {
	threadID := strings.TrimSpace(request.ThreadID)
	if threadID == "" {
		return dto.GenerateThreadTitleResponse{}, errors.New("thread id is required")
	}
	item, err := service.threads.Get(ctx, threadID)
	if err != nil {
		return dto.GenerateThreadTitleResponse{}, err
	}
	currentTitle := strings.TrimSpace(item.Title)
	if currentTitle == "" {
		currentTitle = defaultThreadTitle
	}
	if !item.TitleIsDefault {
		titleChangedBy := item.TitleChangedBy
		if titleChangedBy == "" {
			titleChangedBy = thread.ThreadTitleChangedByUser
		}
		zap.L().Debug("thread title one-shot skipped: manual title is immutable",
			zap.String("threadID", threadID),
			zap.String("title", currentTitle),
		)
		return dto.GenerateThreadTitleResponse{
			ThreadID:       threadID,
			Title:          currentTitle,
			TitleIsDefault: false,
			TitleChangedBy: string(titleChangedBy),
			Updated:        false,
		}, nil
	}
	sourceLines := service.collectThreadTitleSourceLinesFromInputMessages(request.Messages)
	nextTitle := service.generateThreadTitleByRuntime(ctx, item, sourceLines)
	if nextTitle == "" {
		zap.L().Debug("thread title one-shot returned empty title; keep current title",
			zap.String("threadID", threadID),
			zap.String("currentTitle", currentTitle),
		)
		return dto.GenerateThreadTitleResponse{
			ThreadID:       threadID,
			Title:          currentTitle,
			TitleIsDefault: item.TitleIsDefault,
			TitleChangedBy: string(item.TitleChangedBy),
			Updated:        false,
		}, nil
	}
	if service.shouldRejectGeneratedThreadTitle(ctx, threadID, request, nextTitle) {
		zap.L().Debug("thread title one-shot rejected generated title; keep current title",
			zap.String("threadID", threadID),
			zap.String("generatedTitle", nextTitle),
			zap.String("currentTitle", currentTitle),
		)
		return dto.GenerateThreadTitleResponse{
			ThreadID:       threadID,
			Title:          currentTitle,
			TitleIsDefault: item.TitleIsDefault,
			TitleChangedBy: string(item.TitleChangedBy),
			Updated:        false,
		}, nil
	}
	if generatedTitlesEquivalent(nextTitle, currentTitle) {
		zap.L().Debug("thread title one-shot returned unchanged title; keep default state",
			zap.String("threadID", threadID),
			zap.String("title", nextTitle),
		)
		return dto.GenerateThreadTitleResponse{
			ThreadID:       threadID,
			Title:          currentTitle,
			TitleIsDefault: item.TitleIsDefault,
			TitleChangedBy: string(item.TitleChangedBy),
			Updated:        false,
		}, nil
	}

	latest, err := service.threads.Get(ctx, threadID)
	if err != nil {
		return dto.GenerateThreadTitleResponse{}, err
	}
	latestTitle := strings.TrimSpace(latest.Title)
	if latestTitle == "" {
		latestTitle = defaultThreadTitle
	}
	if !latest.TitleIsDefault {
		titleChangedBy := latest.TitleChangedBy
		if titleChangedBy == "" {
			titleChangedBy = thread.ThreadTitleChangedByUser
		}
		zap.L().Debug("thread title one-shot skipped update: title no longer default",
			zap.String("threadID", threadID),
			zap.String("title", latestTitle),
		)
		return dto.GenerateThreadTitleResponse{
			ThreadID:       threadID,
			Title:          latestTitle,
			TitleIsDefault: false,
			TitleChangedBy: string(titleChangedBy),
			Updated:        false,
		}, nil
	}

	updated := true
	latest.Title = nextTitle
	latest.TitleIsDefault = false
	latest.TitleChangedBy = thread.ThreadTitleChangedBySummary
	latest.UpdatedAt = service.now()
	if err := service.threads.Save(ctx, latest); err != nil {
		return dto.GenerateThreadTitleResponse{}, err
	}
	if service.sessions != nil {
		_ = service.sessions.UpdateTitle(ctx, threadID, nextTitle)
	}
	zap.L().Debug("thread title one-shot updated thread metadata",
		zap.String("threadID", threadID),
		zap.String("title", nextTitle),
		zap.Bool("titleChanged", true),
		zap.String("titleChangedBy", string(latest.TitleChangedBy)),
	)
	return dto.GenerateThreadTitleResponse{
		ThreadID:       threadID,
		Title:          nextTitle,
		TitleIsDefault: false,
		TitleChangedBy: string(latest.TitleChangedBy),
		Updated:        updated,
	}, nil
}

func (service *ThreadService) generateThreadTitleByRuntime(ctx context.Context, item thread.Thread, sourceLines []string) string {
	threadID := strings.TrimSpace(item.ID)
	assistantID := strings.TrimSpace(item.AssistantID)
	if service == nil {
		zap.L().Debug("thread title one-shot skipped: nil thread service")
		return ""
	}
	if service.runtime == nil {
		service.publishThreadTitleOneShotEvent(ctx, threadID, threadTitleEventSkipped, threadTitleDebugPayload{
			ThreadID:    threadID,
			AssistantID: assistantID,
			Reason:      "runtime_unavailable",
		})
		zap.L().Debug("thread title one-shot skipped: runtime unavailable",
			zap.String("threadID", threadID),
			zap.String("assistantID", assistantID),
		)
		return ""
	}
	if service.messages == nil && len(sourceLines) == 0 {
		service.publishThreadTitleOneShotEvent(ctx, threadID, threadTitleEventSkipped, threadTitleDebugPayload{
			ThreadID:    threadID,
			AssistantID: assistantID,
			Reason:      "message_repository_unavailable",
		})
		zap.L().Debug("thread title one-shot skipped: message repository unavailable",
			zap.String("threadID", threadID),
			zap.String("assistantID", assistantID),
		)
		return ""
	}
	if threadID == "" {
		service.publishThreadTitleOneShotEvent(ctx, threadID, threadTitleEventSkipped, threadTitleDebugPayload{
			ThreadID:    threadID,
			AssistantID: assistantID,
			Reason:      "empty_thread_id",
		})
		zap.L().Debug("thread title one-shot skipped: empty thread id")
		return ""
	}
	if len(sourceLines) == 0 {
		sourceLines = service.collectThreadTitleSourceLines(ctx, threadID)
	}
	if len(sourceLines) == 0 {
		service.publishThreadTitleOneShotEvent(ctx, threadID, threadTitleEventSkipped, threadTitleDebugPayload{
			ThreadID:    threadID,
			AssistantID: assistantID,
			Reason:      "no_dialogue_source_lines",
		})
		zap.L().Debug("thread title one-shot skipped: no dialogue source lines",
			zap.String("threadID", threadID),
			zap.String("assistantID", assistantID),
		)
		return ""
	}
	systemPrompt, userPrompt := buildThreadTitlePrompts(sourceLines)
	if userPrompt == "" {
		service.publishThreadTitleOneShotEvent(ctx, threadID, threadTitleEventSkipped, threadTitleDebugPayload{
			ThreadID:    threadID,
			AssistantID: assistantID,
			Reason:      "empty_prompt_after_source_extraction",
		})
		zap.L().Debug("thread title one-shot skipped: empty prompt after source extraction",
			zap.String("threadID", threadID),
			zap.String("assistantID", assistantID),
		)
		return ""
	}
	modelRef := service.resolveThreadTitleAssistantModelRef(ctx, assistantID)

	zap.L().Debug("thread title one-shot started",
		zap.String("threadID", threadID),
		zap.String("assistantID", assistantID),
		zap.Int("sourceLineCount", len(sourceLines)),
		zap.Int("promptChars", len([]rune(userPrompt))),
	)
	service.publishThreadTitleOneShotEvent(ctx, threadID, threadTitleEventStarted, threadTitleDebugPayload{
		ThreadID:        threadID,
		AssistantID:     assistantID,
		ModelRef:        modelRef,
		SourceLineCount: len(sourceLines),
		SourceLines:     append([]string(nil), sourceLines...),
		PromptChars:     len([]rune(userPrompt)),
		TimeoutMS:       titleRunTimeout.Milliseconds(),
		Thinking:        titleRunThinkingLevel,
		Temperature:     titleRunTemperature,
		MaxTokens:       titleRunMaxTokens,
	})

	runCtx, cancel := context.WithTimeout(ctx, titleRunTimeout)
	defer cancel()
	request := runtimedto.RuntimeRunRequest{
		SessionID:   threadID,
		AssistantID: assistantID,
		RunKind:     "one-shot",
		PromptMode:  domainassistant.PromptModeNone,
		Input: runtimedto.RuntimeInput{
			Messages: []runtimedto.Message{{
				Role:    "user",
				Content: userPrompt,
			}},
		},
		Thinking: runtimedto.ThinkingConfig{
			Mode: titleRunThinkingLevel,
		},
		Tools: runtimedto.ToolExecutionConfig{
			Mode: "disabled",
		},
		Metadata: map[string]any{
			"channel":           "aui",
			"useQueue":          false,
			"runLane":           "subagent",
			"oneShotKind":       "title_generation",
			"temperature":       titleRunTemperature,
			"maxTokens":         titleRunMaxTokens,
			"extraSystemPrompt": systemPrompt,
		},
	}
	result, err := service.runtime.RunOneShot(runCtx, request)
	if err != nil {
		service.publishThreadTitleOneShotEvent(ctx, threadID, threadTitleEventFailed, threadTitleDebugPayload{
			ThreadID:        threadID,
			AssistantID:     assistantID,
			ModelRef:        modelRef,
			SourceLineCount: len(sourceLines),
			PromptChars:     len([]rune(userPrompt)),
			TimeoutMS:       titleRunTimeout.Milliseconds(),
			Thinking:        titleRunThinkingLevel,
			Temperature:     titleRunTemperature,
			MaxTokens:       titleRunMaxTokens,
			Error:           err.Error(),
		})
		zap.L().Warn("thread title one-shot runtime call failed",
			zap.String("threadID", threadID),
			zap.String("assistantID", assistantID),
			zap.Error(err),
		)
		return ""
	}
	title := resolveGeneratedTitleFromOneShotResult(result.AssistantMessage)
	if title == "" {
		service.publishThreadTitleOneShotEvent(ctx, threadID, threadTitleEventEmpty, threadTitleDebugPayload{
			ThreadID:         threadID,
			AssistantID:      assistantID,
			ModelRef:         threadTitleResultModelRef(modelRef, result.Model),
			SourceLineCount:  len(sourceLines),
			PromptChars:      len([]rune(userPrompt)),
			TimeoutMS:        titleRunTimeout.Milliseconds(),
			Thinking:         titleRunThinkingLevel,
			Temperature:      titleRunTemperature,
			MaxTokens:        titleRunMaxTokens,
			Status:           strings.TrimSpace(result.Status),
			FinishReason:     strings.TrimSpace(result.FinishReason),
			ContentChars:     len([]rune(strings.TrimSpace(result.AssistantMessage.Content))),
			PartsCount:       len(result.AssistantMessage.Parts),
			AssistantMessage: cloneRuntimeMessage(result.AssistantMessage),
		})
		zap.L().Debug("thread title one-shot runtime returned empty content",
			zap.String("threadID", threadID),
			zap.String("assistantID", assistantID),
			zap.String("status", strings.TrimSpace(result.Status)),
			zap.String("finishReason", strings.TrimSpace(result.FinishReason)),
			zap.Int("contentChars", len([]rune(strings.TrimSpace(result.AssistantMessage.Content)))),
		)
		return ""
	}
	service.publishThreadTitleOneShotEvent(ctx, threadID, threadTitleEventCompleted, threadTitleDebugPayload{
		ThreadID:         threadID,
		AssistantID:      assistantID,
		ModelRef:         threadTitleResultModelRef(modelRef, result.Model),
		SourceLineCount:  len(sourceLines),
		PromptChars:      len([]rune(userPrompt)),
		TimeoutMS:        titleRunTimeout.Milliseconds(),
		Thinking:         titleRunThinkingLevel,
		Temperature:      titleRunTemperature,
		MaxTokens:        titleRunMaxTokens,
		Status:           strings.TrimSpace(result.Status),
		FinishReason:     strings.TrimSpace(result.FinishReason),
		Title:            title,
		ContentChars:     len([]rune(strings.TrimSpace(result.AssistantMessage.Content))),
		PartsCount:       len(result.AssistantMessage.Parts),
		AssistantMessage: cloneRuntimeMessage(result.AssistantMessage),
	})
	zap.L().Debug("thread title one-shot runtime completed",
		zap.String("threadID", threadID),
		zap.String("assistantID", assistantID),
		zap.String("title", title),
		zap.String("status", strings.TrimSpace(result.Status)),
		zap.String("finishReason", strings.TrimSpace(result.FinishReason)),
	)
	return title
}

func resolveGeneratedTitleFromOneShotResult(message runtimedto.Message) string {
	if title := normalizeGeneratedTitle(message.Content); title != "" {
		return title
	}
	for _, part := range message.Parts {
		if !strings.EqualFold(strings.TrimSpace(part.Type), "text") {
			continue
		}
		if title := normalizeGeneratedTitle(part.Text); title != "" {
			return title
		}
	}
	return ""
}

type threadTitleDebugPayload struct {
	ThreadID         string              `json:"threadId"`
	AssistantID      string              `json:"assistantId,omitempty"`
	ModelRef         string              `json:"modelRef,omitempty"`
	Reason           string              `json:"reason,omitempty"`
	SourceLineCount  int                 `json:"sourceLineCount,omitempty"`
	SourceLines      []string            `json:"sourceLines,omitempty"`
	PromptChars      int                 `json:"promptChars,omitempty"`
	TimeoutMS        int64               `json:"timeoutMs,omitempty"`
	Thinking         string              `json:"thinking,omitempty"`
	Temperature      float32             `json:"temperature,omitempty"`
	MaxTokens        int                 `json:"maxTokens,omitempty"`
	Status           string              `json:"status,omitempty"`
	FinishReason     string              `json:"finishReason,omitempty"`
	Error            string              `json:"error,omitempty"`
	Title            string              `json:"title,omitempty"`
	ContentChars     int                 `json:"contentChars,omitempty"`
	PartsCount       int                 `json:"partsCount,omitempty"`
	AssistantMessage *runtimedto.Message `json:"assistantMessage,omitempty"`
}

func (service *ThreadService) publishThreadTitleOneShotEvent(ctx context.Context, threadID string, eventType string, payload threadTitleDebugPayload) {
	if service == nil || service.gatewayEvents == nil {
		return
	}
	envelope := appevents.NewGatewayEventEnvelope(threadTitleEventTopic, eventType)
	envelope.SessionID = strings.TrimSpace(threadID)
	envelope.SessionKey = strings.TrimSpace(threadID)
	now := time.Now()
	if service.now != nil {
		now = service.now()
	}
	envelope.Timestamp = now
	_, _ = service.gatewayEvents.Publish(ctx, envelope, payload)
}

func (service *ThreadService) resolveThreadTitleAssistantModelRef(ctx context.Context, assistantID string) string {
	if service == nil || service.assistants == nil {
		return ""
	}
	assistantID = strings.TrimSpace(assistantID)
	if assistantID == "" {
		return ""
	}
	item, err := service.assistants.Get(ctx, assistantID)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(item.Model.Agent.Primary)
}

func threadTitleResultModelRef(fallback string, selection *runtimedto.ModelSelection) string {
	if selection == nil {
		return strings.TrimSpace(fallback)
	}
	providerID := strings.TrimSpace(selection.ProviderID)
	modelName := strings.TrimSpace(selection.Name)
	if providerID == "" || modelName == "" {
		return strings.TrimSpace(fallback)
	}
	return providerID + "/" + modelName
}

func cloneRuntimeMessage(message runtimedto.Message) *runtimedto.Message {
	cloned := message
	if len(message.Parts) > 0 {
		cloned.Parts = append([]chatevent.MessagePart(nil), message.Parts...)
	}
	return &cloned
}

func (service *ThreadService) collectThreadTitleSourceLines(ctx context.Context, threadID string) []string {
	items, err := service.messages.ListByThread(ctx, threadID, 0)
	if err != nil || len(items) == 0 {
		return nil
	}
	sourceItems := make([]threadTitleSourceItem, 0, len(items))
	for _, message := range items {
		role := strings.ToLower(strings.TrimSpace(message.Role))
		if role != "user" && role != "assistant" {
			continue
		}
		text := strings.TrimSpace(extractThreadTitleText(message))
		if text == "" {
			continue
		}
		text = strings.Join(strings.Fields(text), " ")
		text = truncateRunes(text, titleRunSourceCharsPerLine)
		if text == "" {
			continue
		}
		line := buildThreadTitleSourceLine(role, text)
		if line == "" {
			continue
		}
		sourceItems = append(sourceItems, threadTitleSourceItem{Role: role, Text: text})
	}
	return buildThreadTitleSourceLines(trimThreadTitleSourceItemsToRecentRounds(sourceItems, titleRunSourceRoundLimit))
}

func (service *ThreadService) collectThreadTitleSourceLinesFromInputMessages(messages []dto.GenerateThreadTitleMessage) []string {
	if len(messages) == 0 {
		return nil
	}
	sourceItems := make([]threadTitleSourceItem, 0, len(messages))
	for _, message := range messages {
		role := strings.ToLower(strings.TrimSpace(message.Role))
		if role != "user" && role != "assistant" {
			continue
		}
		text := strings.TrimSpace(extractThreadTitleTextFromInputMessage(message))
		if text == "" {
			continue
		}
		text = strings.Join(strings.Fields(text), " ")
		text = truncateRunes(text, titleRunSourceCharsPerLine)
		if text == "" {
			continue
		}
		line := buildThreadTitleSourceLine(role, text)
		if line == "" {
			continue
		}
		sourceItems = append(sourceItems, threadTitleSourceItem{Role: role, Text: text})
	}
	return buildThreadTitleSourceLines(trimThreadTitleSourceItemsToRecentRounds(sourceItems, titleRunSourceRoundLimit))
}

type threadTitleSourceItem struct {
	Role string
	Text string
}

func trimThreadTitleSourceItemsToRecentRounds(items []threadTitleSourceItem, roundLimit int) []threadTitleSourceItem {
	if len(items) == 0 || roundLimit <= 0 {
		return nil
	}
	collected := make([]threadTitleSourceItem, 0, minInt(len(items), roundLimit*2))
	userTurns := 0
	for i := len(items) - 1; i >= 0; i-- {
		item := items[i]
		if item.Role != "user" && item.Role != "assistant" {
			continue
		}
		if strings.TrimSpace(item.Text) == "" {
			continue
		}
		collected = append(collected, item)
		if item.Role == "user" {
			userTurns++
			if userTurns >= roundLimit {
				break
			}
		}
	}
	for left, right := 0, len(collected)-1; left < right; left, right = left+1, right-1 {
		collected[left], collected[right] = collected[right], collected[left]
	}
	return collected
}

func buildThreadTitleSourceLines(items []threadTitleSourceItem) []string {
	if len(items) == 0 {
		return nil
	}
	lines := make([]string, 0, len(items))
	for _, item := range items {
		line := buildThreadTitleSourceLine(item.Role, item.Text)
		if line == "" {
			continue
		}
		lines = append(lines, line)
	}
	if len(lines) == 0 {
		return nil
	}
	return lines
}

func minInt(left int, right int) int {
	if left < right {
		return left
	}
	return right
}

func buildThreadTitleSourceLine(role string, text string) string {
	switch role {
	case "user":
		return "User: " + text
	case "assistant":
		return "Assistant: " + text
	default:
		return ""
	}
}

func buildThreadTitlePrompts(lines []string) (string, string) {
	if len(lines) == 0 {
		return "", ""
	}
	systemPrompt := strings.Join([]string{
		"You are a conversation title generator.",
		"Generate one short title from the dialogue below.",
		"Requirements:",
		"1) Output title only. No explanation.",
		"2) Use the dominant language of the conversation.",
		"3) Keep it concise (about 4-12 words in English or roughly up to 18 characters in CJK languages).",
		"4) Do not use quotes, colons, or ending punctuation.",
		"5) Do not copy any user message verbatim, especially the latest user request.",
		"6) If the best candidate would just repeat the user's wording or the placeholder title, return an empty response.",
	}, "\n")
	userPrompt := fmt.Sprintf(
		"Dialogue:\n%s",
		strings.Join(lines, "\n"),
	)
	return systemPrompt, userPrompt
}

func truncateRunes(value string, max int) string {
	if max <= 0 {
		return ""
	}
	runes := []rune(strings.TrimSpace(value))
	if len(runes) <= max {
		return strings.TrimSpace(value)
	}
	return strings.TrimSpace(string(runes[:max]))
}

func (service *ThreadService) SetThreadStatus(ctx context.Context, request dto.SetThreadStatusRequest) error {
	threadID := strings.TrimSpace(request.ThreadID)
	if threadID == "" {
		return errors.New("thread id is required")
	}
	status := request.Status
	if status != thread.ThreadStatusRegular && status != thread.ThreadStatusArchived {
		return errors.New("status is invalid")
	}
	if err := service.threads.SetStatus(ctx, threadID, status, service.now()); err != nil {
		return err
	}
	if service.sessions != nil {
		sessionStatus := domainsession.StatusActive
		if status == thread.ThreadStatusArchived {
			sessionStatus = domainsession.StatusArchived
		}
		_ = service.sessions.SetStatus(ctx, threadID, sessionStatus)
	}
	if status == thread.ThreadStatusArchived {
		service.handleSessionLifecycle(ctx, threadID, memorydto.SessionLifecycleArchived)
	}
	return nil
}

func (service *ThreadService) ListThreads(ctx context.Context, includeDeleted bool) ([]dto.Thread, error) {
	items, err := service.threads.List(ctx, includeDeleted)
	if err != nil {
		return nil, err
	}
	result := make([]dto.Thread, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item.AgentID) != "" {
			continue
		}
		result = append(result, toDTO(item))
	}
	return result, nil
}

func (service *ThreadService) GetThread(ctx context.Context, threadID string) (dto.Thread, error) {
	threadID = strings.TrimSpace(threadID)
	if threadID == "" {
		return dto.Thread{}, errors.New("thread id is required")
	}
	item, err := service.threads.Get(ctx, threadID)
	if err != nil {
		return dto.Thread{}, err
	}
	return toDTO(item), nil
}

func (service *ThreadService) GetContextTokensSnapshot(ctx context.Context, threadID string) (dto.ContextTokensSnapshot, error) {
	threadID = strings.TrimSpace(threadID)
	if threadID == "" {
		return dto.ContextTokensSnapshot{}, errors.New("thread id is required")
	}
	if service.sessions != nil {
		entry, err := service.sessions.Get(ctx, threadID)
		if err != nil {
			if !errors.Is(err, appsession.ErrSessionNotFound) {
				return dto.ContextTokensSnapshot{}, err
			}
		} else if hasSessionContextSnapshot(entry) {
			promptTokens := entry.ContextPromptTokens
			totalTokens := entry.ContextTotalTokens
			// Older sessions may carry an empty snapshot (0 tokens with only updated_at set).
			// Backfill from persisted thread history so UI can show non-zero usage on thread switch.
			if promptTokens <= 0 && totalTokens <= 0 {
				if estimated := service.estimateThreadContextTokens(ctx, threadID); estimated > 0 {
					promptTokens = estimated
					totalTokens = estimated
				}
			}
			contextWindowTokens := entry.ContextWindowTokens
			// Model can be switched between runs; resolve the current thread model window
			// on read so UI percentage follows the latest selection immediately.
			if resolvedWindowTokens := service.resolveThreadContextWindowTokens(ctx, threadID); resolvedWindowTokens > 0 {
				contextWindowTokens = resolvedWindowTokens
			}
			return dto.ContextTokensSnapshot{
				PromptTokens:        promptTokens,
				TotalTokens:         totalTokens,
				ContextWindowTokens: contextWindowTokens,
				UpdatedAt:           formatTimeValue(entry.ContextUpdatedAt),
				Fresh:               entry.ContextFresh,
			}, nil
		}
	}

	estimatedTokens := service.estimateThreadContextTokens(ctx, threadID)
	contextWindowTokens := service.resolveThreadContextWindowTokens(ctx, threadID)
	if estimatedTokens <= 0 && contextWindowTokens <= 0 {
		return dto.ContextTokensSnapshot{}, nil
	}
	return dto.ContextTokensSnapshot{
		PromptTokens:        estimatedTokens,
		TotalTokens:         estimatedTokens,
		ContextWindowTokens: contextWindowTokens,
		Fresh:               false,
	}, nil
}

func (service *ThreadService) ListMessages(ctx context.Context, threadID string, limit int) ([]dto.Message, error) {
	threadID = strings.TrimSpace(threadID)
	if threadID == "" {
		return nil, errors.New("thread id is required")
	}
	if limit <= 0 {
		limit = 200
	}
	items, err := service.messages.ListByThread(ctx, threadID, limit)
	if err != nil {
		return nil, err
	}
	result := make([]dto.Message, 0, len(items))
	for _, item := range items {
		result = append(result, dto.Message{
			ID:           item.ID,
			Kind:         string(item.Kind),
			Role:         item.Role,
			Content:      item.Content,
			PartsJSON:    item.PartsJSON,
			PartsVersion: detectPartsVersion(item.PartsJSON),
			CreatedAt:    formatTimeValue(item.CreatedAt),
		})
	}
	if service.runs != nil {
		runs, err := service.runs.ListActiveByThread(ctx, threadID)
		if err != nil && !errors.Is(err, thread.ErrRunNotFound) {
			return nil, err
		}
		if len(runs) > 0 {
			run := runs[0]
			if run.Status == thread.RunStatusActive && strings.TrimSpace(run.ContentPartial) != "" {
				partialMessage := dto.Message{
					ID:           run.AssistantMessageID,
					Role:         "assistant",
					Content:      run.ContentPartial,
					PartsJSON:    `[{"type":"text","text":` + quoteJSONString(run.ContentPartial) + `}]`,
					PartsVersion: 1,
					CreatedAt:    formatTimeValue(run.CreatedAt),
				}
				found := false
				for i := range result {
					if result[i].ID == partialMessage.ID {
						result[i] = partialMessage
						found = true
						break
					}
				}
				if !found {
					result = append(result, partialMessage)
				}
			}
		}
	}
	return result, nil
}

func (service *ThreadService) ListThreadRunEvents(ctx context.Context, request dto.ListThreadRunEventsRequest) ([]dto.ThreadRunEvent, error) {
	threadID := strings.TrimSpace(request.ThreadID)
	if threadID == "" {
		return nil, errors.New("thread id is required")
	}
	if service.runEvents == nil {
		return nil, errors.New("run event repository unavailable")
	}
	limit := request.Limit
	if limit <= 0 {
		limit = 200
	}
	items, err := service.runEvents.ListByThread(
		ctx,
		threadID,
		request.AfterID,
		limit,
		strings.TrimSpace(request.EventTypePrefix),
	)
	if err != nil {
		return nil, err
	}
	result := make([]dto.ThreadRunEvent, 0, len(items))
	for _, item := range items {
		result = append(result, dto.ThreadRunEvent{
			ID:          item.ID,
			RunID:       item.RunID,
			ThreadID:    item.ThreadID,
			EventType:   item.EventType,
			PayloadJSON: item.PayloadJSON,
			CreatedAt:   formatTimeValue(item.CreatedAt),
		})
	}
	return result, nil
}

func (service *ThreadService) AppendMessage(ctx context.Context, request dto.AppendMessageRequest) error {
	threadID := strings.TrimSpace(request.ThreadID)
	if threadID == "" {
		return errors.New("thread id is required")
	}
	messageID := strings.TrimSpace(request.ID)
	role := strings.TrimSpace(request.Role)
	if role == "" {
		return errors.New("role is required")
	}
	content := strings.TrimSpace(request.Content)
	partsJSON := normalizeThreadMessagePartsJSON(request.Parts, content)
	if content == "" && (partsJSON == "" || partsJSON == "[]") {
		return errors.New("message is empty")
	}
	now := service.now()
	if messageID == "" {
		messageID = service.newID()
	}
	msg, err := thread.NewThreadMessage(thread.ThreadMessageParams{
		ID:        messageID,
		ThreadID:  threadID,
		Kind:      thread.MessageKind(strings.TrimSpace(request.Kind)),
		Role:      role,
		Content:   content,
		PartsJSON: partsJSON,
		CreatedAt: &now,
	})
	if err != nil {
		return err
	}
	if err := service.messages.Append(ctx, msg); err != nil {
		return err
	}
	threadItem, err := service.threads.Get(ctx, threadID)
	if err != nil {
		return err
	}
	threadItem.UpdatedAt = now
	if msg.Kind != thread.ThreadMessageKindNotice {
		threadItem.LastInteractiveAt = now
	}
	return service.threads.Save(ctx, threadItem)
}

func (service *ThreadService) SoftDeleteThread(ctx context.Context, id string) error {
	service.handleSessionLifecycle(ctx, id, memorydto.SessionLifecycleDeleted)
	now := service.now()
	purgeAfter := now.Add(defaultPurgeDelay)
	return service.threads.SoftDelete(ctx, id, &now, &purgeAfter)
}

func (service *ThreadService) RestoreThread(ctx context.Context, id string) error {
	return service.threads.Restore(ctx, id)
}

func (service *ThreadService) PurgeThread(ctx context.Context, id string) error {
	shouldTrigger := true
	if service != nil && service.threads != nil {
		if item, err := service.threads.Get(ctx, strings.TrimSpace(id)); err == nil {
			shouldTrigger = item.DeletedAt == nil
		}
	}
	if shouldTrigger {
		service.handleSessionLifecycle(ctx, id, memorydto.SessionLifecycleDeleted)
	}
	return service.threads.Purge(ctx, id)
}

func (service *ThreadService) PurgeExpired(ctx context.Context, limit int) (int, error) {
	if limit <= 0 {
		limit = 100
	}
	now := service.now()
	candidates, err := service.threads.ListPurgeCandidates(ctx, now, limit)
	if err != nil {
		return 0, err
	}
	purged := 0
	for _, item := range candidates {
		if err := service.PurgeThread(ctx, item.ID); err != nil {
			return purged, err
		}
		purged++
	}
	return purged, nil
}

func normalizeThreadMessagePartsJSON(parts []chatevent.MessagePart, fallbackContent string) string {
	if len(parts) == 0 {
		content := strings.TrimSpace(fallbackContent)
		if content == "" {
			return "[]"
		}
		data, err := json.Marshal([]chatevent.MessagePart{{
			Type: "text",
			Text: content,
		}})
		if err != nil {
			return "[]"
		}
		return string(data)
	}
	data, err := json.Marshal(parts)
	if err != nil {
		return "[]"
	}
	if len(data) == 0 {
		return "[]"
	}
	return string(data)
}

func toDTO(item thread.Thread) dto.Thread {
	return dto.Thread{
		ID:                item.ID,
		AssistantID:       item.AssistantID,
		Title:             item.Title,
		TitleIsDefault:    item.TitleIsDefault,
		TitleChangedBy:    string(item.TitleChangedBy),
		Status:            item.Status,
		CreatedAt:         item.CreatedAt.Format(time.RFC3339),
		UpdatedAt:         item.UpdatedAt.Format(time.RFC3339),
		LastInteractiveAt: item.LastInteractiveAt.Format(time.RFC3339),
		DeletedAt:         formatTime(item.DeletedAt),
		PurgeAfter:        formatTime(item.PurgeAfter),
	}
}

func formatTime(value *time.Time) string {
	if value == nil || value.IsZero() {
		return ""
	}
	return value.Format(time.RFC3339)
}

func formatTimeValue(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format(time.RFC3339)
}

func (service *ThreadService) handleSessionLifecycle(ctx context.Context, threadID string, event memorydto.SessionLifecycleEvent) {
	if service == nil || service.memory == nil || service.threads == nil {
		return
	}
	threadID = strings.TrimSpace(threadID)
	if threadID == "" {
		return
	}
	item, err := service.threads.Get(ctx, threadID)
	if err != nil {
		zap.L().Warn("thread lifecycle memory hook skipped: thread lookup failed",
			zap.String("threadID", threadID),
			zap.String("event", string(event)),
			zap.Error(err),
		)
		return
	}
	channel := ""
	accountID := ""
	userID := ""
	groupID := ""
	if service.sessions != nil {
		if sessionEntry, sessionErr := service.sessions.Get(ctx, threadID); sessionErr == nil {
			channel = strings.TrimSpace(sessionEntry.Origin.Channel)
			accountID = strings.TrimSpace(sessionEntry.Origin.AccountID)
			chatType := strings.ToLower(strings.TrimSpace(sessionEntry.Origin.ChatType))
			peerID := strings.TrimSpace(sessionEntry.Origin.PeerID)
			switch chatType {
			case "group", "supergroup", "room", "channel":
				groupID = peerID
			case "direct", "private", "user", "dm":
				userID = peerID
			}
		}
	}
	hookCtx, cancel := context.WithTimeout(ctx, runtimeconfig.DefaultAuxiliaryLLMTimeout)
	defer cancel()
	if err := service.memory.HandleSessionLifecycle(hookCtx, memorydto.SessionLifecycleRequest{
		Identity: memorydto.MemoryIdentity{
			AssistantID: strings.TrimSpace(item.AssistantID),
			ThreadID:    threadID,
			Channel:     channel,
			AccountID:   accountID,
			UserID:      userID,
			GroupID:     groupID,
		},
		Event: event,
	}); err != nil {
		zap.L().Warn("thread lifecycle memory hook failed",
			zap.String("threadID", threadID),
			zap.String("assistantID", strings.TrimSpace(item.AssistantID)),
			zap.String("event", string(event)),
			zap.Error(err),
		)
	}
}

func quoteJSONString(value string) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		return `""`
	}
	return string(encoded)
}

func detectPartsVersion(partsJSON string) int {
	trimmed := strings.TrimSpace(partsJSON)
	if trimmed == "" || trimmed == "[]" {
		return 0
	}
	return 1
}

func (service *ThreadService) shouldRejectGeneratedThreadTitle(ctx context.Context, threadID string, request dto.GenerateThreadTitleRequest, title string) bool {
	normalizedTitle := normalizeGeneratedTitle(title)
	if normalizedTitle == "" {
		return true
	}
	if generatedTitlesEquivalent(normalizedTitle, defaultThreadTitle) {
		return true
	}
	if latestUserTitle := service.resolveLatestUserTitle(ctx, threadID, request.Messages); latestUserTitle != "" && generatedTitlesEquivalent(normalizedTitle, latestUserTitle) {
		return true
	}
	return false
}

func (service *ThreadService) resolveLatestUserTitle(ctx context.Context, threadID string, messages []dto.GenerateThreadTitleMessage) string {
	if title := resolveLatestUserTitleFromInputMessages(messages); title != "" {
		return title
	}
	return service.resolveLatestUserTitleFromPersistedMessages(ctx, threadID)
}

func resolveLatestUserTitleFromInputMessages(messages []dto.GenerateThreadTitleMessage) string {
	if len(messages) == 0 {
		return ""
	}
	for i := len(messages) - 1; i >= 0; i-- {
		item := messages[i]
		if strings.ToLower(strings.TrimSpace(item.Role)) != "user" {
			continue
		}
		title := normalizeGeneratedTitle(extractThreadTitleTextFromInputMessage(item))
		if title != "" {
			return title
		}
	}
	return ""
}

func (service *ThreadService) resolveLatestUserTitleFromPersistedMessages(ctx context.Context, threadID string) string {
	if service == nil || service.messages == nil {
		return ""
	}
	items, err := service.messages.ListByThread(ctx, threadID, 200)
	if err != nil || len(items) == 0 {
		return ""
	}
	for i := len(items) - 1; i >= 0; i-- {
		item := items[i]
		if strings.ToLower(strings.TrimSpace(item.Role)) != "user" {
			continue
		}
		title := normalizeGeneratedTitle(extractThreadTitleText(item))
		if title != "" {
			return title
		}
	}
	return ""
}

func extractThreadTitleText(item thread.ThreadMessage) string {
	partsJSON := strings.TrimSpace(item.PartsJSON)
	if partsJSON != "" && partsJSON != "[]" {
		var parts []chatevent.MessagePart
		if err := json.Unmarshal([]byte(partsJSON), &parts); err == nil && len(parts) > 0 {
			text := strings.TrimSpace(joinMessagePartsText(parts))
			if text != "" {
				return text
			}
		}
	}
	return strings.TrimSpace(item.Content)
}

func extractThreadTitleTextFromInputMessage(item dto.GenerateThreadTitleMessage) string {
	if len(item.Parts) > 0 {
		text := strings.TrimSpace(joinMessagePartsText(item.Parts))
		if text != "" {
			return text
		}
	}
	return strings.TrimSpace(item.Content)
}

func joinMessagePartsText(parts []chatevent.MessagePart) string {
	if len(parts) == 0 {
		return ""
	}
	var builder strings.Builder
	for _, part := range parts {
		if strings.TrimSpace(part.Type) != "text" {
			continue
		}
		builder.WriteString(part.Text)
	}
	return builder.String()
}

func normalizeGeneratedTitle(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	compact := strings.Join(strings.Fields(trimmed), " ")
	if compact == "" {
		return ""
	}
	const maxChars = 80
	runes := []rune(compact)
	if len(runes) <= maxChars {
		return compact
	}
	return string(runes[:maxChars]) + "…"
}

func generatedTitlesEquivalent(left string, right string) bool {
	leftNormalized := normalizeGeneratedTitleComparison(left)
	rightNormalized := normalizeGeneratedTitleComparison(right)
	return leftNormalized != "" && leftNormalized == rightNormalized
}

func normalizeGeneratedTitleComparison(raw string) string {
	normalized := normalizeGeneratedTitle(raw)
	if normalized == "" {
		return ""
	}
	normalized = strings.TrimSpace(strings.Trim(normalized, `"'“”‘’`))
	normalized = strings.TrimSpace(strings.TrimRight(normalized, ".,!?;:，。！？；："))
	if normalized == "" {
		return ""
	}
	return strings.ToLower(normalized)
}

func hasSessionContextSnapshot(entry domainsession.Entry) bool {
	return entry.ContextPromptTokens > 0 ||
		entry.ContextTotalTokens > 0 ||
		entry.ContextWindowTokens > 0
}

func (service *ThreadService) estimateThreadContextTokens(ctx context.Context, threadID string) int {
	if service == nil || service.messages == nil {
		return 0
	}
	items, err := service.messages.ListByThread(ctx, threadID, 0)
	if err != nil || len(items) == 0 {
		return 0
	}
	total := 0
	for _, item := range items {
		text := extractThreadMessageEstimateText(item)
		if text == "" {
			continue
		}
		total += 4 // Message envelope overhead aligned with runtime estimator.
		total += agentruntime.EstimateTextTokens(text)
	}
	if total < 0 {
		return 0
	}
	return total
}

func extractThreadMessageEstimateText(item thread.ThreadMessage) string {
	partsText := extractPartsEstimateText(item.PartsJSON)
	if partsText != "" {
		return partsText
	}
	content := strings.TrimSpace(item.Content)
	if content != "" {
		return content
	}
	partsJSON := strings.TrimSpace(item.PartsJSON)
	if partsJSON == "" || partsJSON == "[]" {
		return ""
	}
	return partsJSON
}

func extractPartsEstimateText(partsJSON string) string {
	trimmed := strings.TrimSpace(partsJSON)
	if trimmed == "" || trimmed == "[]" {
		return ""
	}
	var parts []chatevent.MessagePart
	if err := json.Unmarshal([]byte(trimmed), &parts); err != nil {
		return ""
	}
	if len(parts) == 0 {
		return ""
	}
	segments := make([]string, 0, len(parts)*5)
	appendSegment := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		segments = append(segments, value)
	}
	for _, part := range parts {
		appendSegment(part.Type)
		appendSegment(part.ParentID)
		appendSegment(part.Text)
		appendSegment(part.State)
		appendSegment(part.ToolCallID)
		appendSegment(part.ToolName)
		appendSegment(part.ErrorText)
		appendSegment(strings.TrimSpace(string(part.Input)))
		appendSegment(strings.TrimSpace(string(part.Output)))
		appendSegment(strings.TrimSpace(string(part.Data)))
	}
	return strings.Join(segments, "\n")
}

func (service *ThreadService) resolveThreadContextWindowTokens(ctx context.Context, threadID string) int {
	if service == nil || service.threads == nil || service.assistants == nil || service.models == nil {
		return 0
	}
	threadItem, err := service.threads.Get(ctx, threadID)
	if err != nil {
		return 0
	}
	assistantID := strings.TrimSpace(threadItem.AssistantID)
	if assistantID == "" {
		return 0
	}
	assistantItem, err := service.assistants.Get(ctx, assistantID)
	if err != nil {
		return 0
	}
	providerID, modelName, err := parseAssistantModelRef(assistantItem.Model.Agent.Primary)
	if err != nil {
		return 0
	}
	models, err := service.models.ListByProvider(ctx, providerID)
	if err != nil {
		return 0
	}
	selected := 0
	for _, model := range models {
		if !strings.EqualFold(strings.TrimSpace(model.Name), modelName) {
			continue
		}
		candidate := 0
		if model.ContextWindow != nil && *model.ContextWindow > 0 {
			candidate = *model.ContextWindow
		} else {
			candidate = extractContextWindowFromCapabilities(model.CapabilitiesJSON)
		}
		if candidate <= 0 {
			continue
		}
		if selected <= 0 || candidate < selected {
			selected = candidate
		}
	}
	return selected
}

func parseAssistantModelRef(value string) (string, string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", "", errors.New("model ref is empty")
	}
	for _, sep := range []string{"/", ":"} {
		if !strings.Contains(trimmed, sep) {
			continue
		}
		parts := strings.SplitN(trimmed, sep, 2)
		providerID := strings.TrimSpace(parts[0])
		modelName := strings.TrimSpace(parts[1])
		if providerID == "" || modelName == "" {
			return "", "", errors.New("model ref must include provider prefix")
		}
		return providerID, modelName, nil
	}
	return "", "", errors.New("model ref must include provider prefix")
}

func extractContextWindowFromCapabilities(raw string) int {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0
	}
	var payload any
	if err := json.Unmarshal([]byte(trimmed), &payload); err != nil {
		return 0
	}
	max := 0
	visitCapabilityValue(payload, func(key string, value any) {
		if !isContextWindowCapabilityKey(key) {
			return
		}
		candidate := capabilityToInt(value)
		if candidate > max {
			max = candidate
		}
	})
	return max
}

func visitCapabilityValue(value any, handle func(key string, value any)) {
	switch typed := value.(type) {
	case map[string]any:
		for key, next := range typed {
			handle(key, next)
			visitCapabilityValue(next, handle)
		}
	case []any:
		for _, next := range typed {
			visitCapabilityValue(next, handle)
		}
	}
}

func isContextWindowCapabilityKey(key string) bool {
	lower := strings.ToLower(strings.TrimSpace(key))
	if lower == "" {
		return false
	}
	if lower == "context" || strings.Contains(lower, "context_window") {
		return true
	}
	if strings.Contains(lower, "context") && strings.Contains(lower, "length") {
		return true
	}
	if strings.Contains(lower, "context") && strings.Contains(lower, "tokens") {
		return true
	}
	return false
}

func capabilityToInt(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case json.Number:
		if parsed, err := typed.Int64(); err == nil {
			return int(parsed)
		}
	case string:
		trimmed := strings.ToLower(strings.TrimSpace(typed))
		if trimmed == "" {
			return 0
		}
		if strings.HasSuffix(trimmed, "k") {
			base := strings.TrimSpace(strings.TrimSuffix(trimmed, "k"))
			if parsed, err := strconv.ParseFloat(base, 64); err == nil {
				return int(math.Round(parsed * 1000))
			}
		}
		if parsed, err := strconv.Atoi(trimmed); err == nil {
			return parsed
		}
		if parsed, err := strconv.ParseFloat(trimmed, 64); err == nil {
			return int(math.Round(parsed))
		}
	}
	return 0
}
