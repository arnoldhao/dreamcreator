package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"dreamcreator/internal/application/chatevent"
	gatewayevents "dreamcreator/internal/application/gateway/events"
	runtimedto "dreamcreator/internal/application/gateway/runtime/dto"
	"dreamcreator/internal/application/thread/dto"
	"dreamcreator/internal/domain/thread"
)

type threadRepositoryStub struct {
	item     thread.Thread
	saveCall int
}

func (repo *threadRepositoryStub) List(context.Context, bool) ([]thread.Thread, error) {
	return []thread.Thread{repo.item}, nil
}

func (repo *threadRepositoryStub) ListPurgeCandidates(context.Context, time.Time, int) ([]thread.Thread, error) {
	return nil, nil
}

func (repo *threadRepositoryStub) Get(_ context.Context, id string) (thread.Thread, error) {
	if id != repo.item.ID {
		return thread.Thread{}, thread.ErrThreadNotFound
	}
	return repo.item, nil
}

func (repo *threadRepositoryStub) Save(_ context.Context, item thread.Thread) error {
	repo.item = item
	repo.saveCall++
	return nil
}

func (repo *threadRepositoryStub) SoftDelete(context.Context, string, *time.Time, *time.Time) error {
	return nil
}

func (repo *threadRepositoryStub) Restore(context.Context, string) error {
	return nil
}

func (repo *threadRepositoryStub) Purge(context.Context, string) error {
	return nil
}

func (repo *threadRepositoryStub) SetStatus(context.Context, string, thread.Status, time.Time) error {
	return nil
}

type messageRepositoryStub struct {
	items []thread.ThreadMessage
}

func (repo *messageRepositoryStub) ListByThread(_ context.Context, threadID string, _ int) ([]thread.ThreadMessage, error) {
	result := make([]thread.ThreadMessage, 0, len(repo.items))
	for _, item := range repo.items {
		if item.ThreadID == threadID {
			result = append(result, item)
		}
	}
	return result, nil
}

func (repo *messageRepositoryStub) Append(context.Context, thread.ThreadMessage) error {
	return nil
}

func (repo *messageRepositoryStub) DeleteByThread(context.Context, string) error {
	return nil
}

type titleRuntimeStub struct {
	request   runtimedto.RuntimeRunRequest
	result    runtimedto.RuntimeRunResult
	err       error
	callCount int
}

func (stub *titleRuntimeStub) RunOneShot(_ context.Context, request runtimedto.RuntimeRunRequest) (runtimedto.RuntimeRunResult, error) {
	stub.request = request
	stub.callCount++
	if stub.err != nil {
		return runtimedto.RuntimeRunResult{}, stub.err
	}
	return stub.result, nil
}

func newThreadForTitleTest(t *testing.T, id string, title string, titleIsDefault bool) thread.Thread {
	t.Helper()
	now := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
	titleChangedBy := thread.TitleChangedBy("")
	if !titleIsDefault {
		titleChangedBy = thread.ThreadTitleChangedByUser
	}
	item, err := thread.NewThread(thread.ThreadParams{
		ID:             id,
		AssistantID:    "assistant-1",
		Title:          title,
		TitleIsDefault: titleIsDefault,
		TitleChangedBy: titleChangedBy,
		Status:         thread.ThreadStatusRegular,
		CreatedAt:      &now,
		UpdatedAt:      &now,
	})
	if err != nil {
		t.Fatalf("new thread: %v", err)
	}
	return item
}

func TestGenerateThreadTitle_WithoutRuntimeKeepsDefaultTitle(t *testing.T) {
	t.Parallel()

	threadRepo := &threadRepositoryStub{
		item: newThreadForTitleTest(t, "session-1", defaultThreadTitle, true),
	}
	messageRepo := &messageRepositoryStub{
		items: []thread.ThreadMessage{
			{
				ID:        "msg-1",
				ThreadID:  "session-1",
				Role:      "user",
				PartsJSON: `[{"type":"text","text":"  这是   自动   命名   测试  "}]`,
			},
		},
	}
	service := &ThreadService{
		threads:  threadRepo,
		messages: messageRepo,
		now: func() time.Time {
			return time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
		},
	}

	resp, err := service.GenerateThreadTitle(context.Background(), dto.GenerateThreadTitleRequest{
		ThreadID: "session-1",
	})
	if err != nil {
		t.Fatalf("generate thread title: %v", err)
	}

	if resp.Title != defaultThreadTitle {
		t.Fatalf("unexpected title: %q", resp.Title)
	}
	if resp.Updated {
		t.Fatalf("title should not update when one-shot runtime is unavailable")
	}
	if resp.TitleChangedBy != "" {
		t.Fatalf("titleChangedBy should stay empty when one-shot runtime is unavailable, got %q", resp.TitleChangedBy)
	}
	if threadRepo.saveCall != 0 {
		t.Fatalf("expected no save, got %d", threadRepo.saveCall)
	}
}

func TestGenerateThreadTitle_ManualTitleIsImmutable(t *testing.T) {
	t.Parallel()

	threadRepo := &threadRepositoryStub{
		item: newThreadForTitleTest(t, "session-2", "手动标题", false),
	}
	service := &ThreadService{
		threads:  threadRepo,
		messages: &messageRepositoryStub{},
	}

	resp, err := service.GenerateThreadTitle(context.Background(), dto.GenerateThreadTitleRequest{
		ThreadID: "session-2",
	})
	if err != nil {
		t.Fatalf("generate thread title: %v", err)
	}

	if resp.Title != "手动标题" {
		t.Fatalf("title should stay unchanged, got %q", resp.Title)
	}
	if resp.Updated {
		t.Fatalf("manual title should not be overwritten")
	}
	if resp.TitleChangedBy != string(thread.ThreadTitleChangedByUser) {
		t.Fatalf("manual title should be marked changed by user, got %q", resp.TitleChangedBy)
	}
	if threadRepo.saveCall != 0 {
		t.Fatalf("manual title should not trigger save, got %d", threadRepo.saveCall)
	}
}

func TestGenerateThreadTitle_UsesRuntimeOneShotWhenAvailable(t *testing.T) {
	t.Parallel()

	threadRepo := &threadRepositoryStub{
		item: newThreadForTitleTest(t, "session-3", defaultThreadTitle, true),
	}
	messageRepo := &messageRepositoryStub{
		items: []thread.ThreadMessage{
			{
				ID:       "msg-1",
				ThreadID: "session-3",
				Role:     "user",
				Content:  "帮我总结今天关于 Web Fetch 的改动",
			},
			{
				ID:       "msg-2",
				ThreadID: "session-3",
				Role:     "assistant",
				Content:  "我们新增了 search engine 并调整了 cookies 选择策略",
			},
		},
	}
	runtimeStub := &titleRuntimeStub{
		result: runtimedto.RuntimeRunResult{
			AssistantMessage: runtimedto.Message{Content: "Web Fetch 配置优化总结"},
		},
	}
	service := &ThreadService{
		threads:  threadRepo,
		messages: messageRepo,
		runtime:  runtimeStub,
		now: func() time.Time {
			return time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
		},
	}

	resp, err := service.GenerateThreadTitle(context.Background(), dto.GenerateThreadTitleRequest{
		ThreadID: "session-3",
	})
	if err != nil {
		t.Fatalf("generate thread title: %v", err)
	}

	if runtimeStub.callCount != 1 {
		t.Fatalf("expected runtime call once, got %d", runtimeStub.callCount)
	}
	if runtimeStub.request.Tools.Mode != "disabled" {
		t.Fatalf("expected runtime tools disabled, got %q", runtimeStub.request.Tools.Mode)
	}
	if runtimeStub.request.RunKind != "one-shot" {
		t.Fatalf("expected one-shot run kind, got %q", runtimeStub.request.RunKind)
	}
	if runtimeStub.request.Thinking.Mode != titleRunThinkingLevel {
		t.Fatalf("unexpected thinking level: %q", runtimeStub.request.Thinking.Mode)
	}
	if value, _ := runtimeStub.request.Metadata["maxTokens"].(int); value != titleRunMaxTokens {
		t.Fatalf("unexpected maxTokens: %v", runtimeStub.request.Metadata["maxTokens"])
	}
	if value, _ := runtimeStub.request.Metadata["runLane"].(string); value != "subagent" {
		t.Fatalf("runLane should be subagent, got %v", runtimeStub.request.Metadata["runLane"])
	}
	if value, _ := runtimeStub.request.Metadata["oneShotKind"].(string); value != "title_generation" {
		t.Fatalf("oneShotKind should be title_generation, got %v", runtimeStub.request.Metadata["oneShotKind"])
	}
	if value, _ := runtimeStub.request.Metadata["extraSystemPrompt"].(string); !strings.Contains(value, "Do not copy any user message verbatim") {
		t.Fatalf("expected extra system prompt, got %q", value)
	}
	if resp.Title != "Web Fetch 配置优化总结" {
		t.Fatalf("unexpected title: %q", resp.Title)
	}
	if !resp.Updated {
		t.Fatalf("expected title update")
	}
	if resp.TitleIsDefault {
		t.Fatalf("auto generated title should no longer be default")
	}
	if resp.TitleChangedBy != string(thread.ThreadTitleChangedBySummary) {
		t.Fatalf("auto generated title should be marked changed by summary, got %q", resp.TitleChangedBy)
	}
}

func TestGenerateThreadTitle_RuntimeReturnsPlaceholderKeepsDefaultTitle(t *testing.T) {
	t.Parallel()

	threadRepo := &threadRepositoryStub{
		item: newThreadForTitleTest(t, "session-4", defaultThreadTitle, true),
	}
	messageRepo := &messageRepositoryStub{
		items: []thread.ThreadMessage{{
			ID:       "msg-1",
			ThreadID: "session-4",
			Role:     "user",
			Content:  "随便聊聊",
		}},
	}
	runtimeStub := &titleRuntimeStub{
		result: runtimedto.RuntimeRunResult{
			AssistantMessage: runtimedto.Message{Content: defaultThreadTitle},
		},
	}
	service := &ThreadService{
		threads:  threadRepo,
		messages: messageRepo,
		runtime:  runtimeStub,
		now: func() time.Time {
			return time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
		},
	}

	resp, err := service.GenerateThreadTitle(context.Background(), dto.GenerateThreadTitleRequest{
		ThreadID: "session-4",
	})
	if err != nil {
		t.Fatalf("generate thread title: %v", err)
	}

	if resp.Title != defaultThreadTitle {
		t.Fatalf("unexpected title: %q", resp.Title)
	}
	if !resp.TitleIsDefault {
		t.Fatalf("default title should remain unchanged")
	}
	if resp.TitleChangedBy != "" {
		t.Fatalf("default title should keep empty titleChangedBy, got %q", resp.TitleChangedBy)
	}
	if resp.Updated {
		t.Fatalf("placeholder title should not update thread title")
	}
	if threadRepo.saveCall != 0 {
		t.Fatalf("placeholder title should not trigger save")
	}
}

func TestGenerateThreadTitle_UsesRequestMessagesAsSource(t *testing.T) {
	t.Parallel()

	threadRepo := &threadRepositoryStub{
		item: newThreadForTitleTest(t, "session-5", defaultThreadTitle, true),
	}
	runtimeStub := &titleRuntimeStub{
		result: runtimedto.RuntimeRunResult{
			AssistantMessage: runtimedto.Message{Content: "请求前置标题"},
		},
	}
	service := &ThreadService{
		threads:  threadRepo,
		messages: &messageRepositoryStub{},
		runtime:  runtimeStub,
		now: func() time.Time {
			return time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
		},
	}

	resp, err := service.GenerateThreadTitle(context.Background(), dto.GenerateThreadTitleRequest{
		ThreadID: "session-5",
		Messages: []dto.GenerateThreadTitleMessage{
			{Role: "user", Content: "先做 thread name"},
			{Role: "assistant", Content: "好的"},
		},
	})
	if err != nil {
		t.Fatalf("generate thread title: %v", err)
	}
	if !resp.Updated {
		t.Fatalf("expected title update")
	}
	if !strings.Contains(runtimeStub.request.Input.Messages[0].Content, "先做 thread name") {
		t.Fatalf("prompt should include request messages, got %q", runtimeStub.request.Input.Messages[0].Content)
	}
}

func TestGenerateThreadTitle_OnlyUsesLatestThreeRoundsAsContext(t *testing.T) {
	t.Parallel()

	threadRepo := &threadRepositoryStub{
		item: newThreadForTitleTest(t, "session-7", defaultThreadTitle, true),
	}
	messageRepo := &messageRepositoryStub{
		items: []thread.ThreadMessage{
			{ID: "msg-1", ThreadID: "session-7", Role: "user", Content: "第一轮用户"},
			{ID: "msg-2", ThreadID: "session-7", Role: "assistant", Content: "第一轮助手"},
			{ID: "msg-3", ThreadID: "session-7", Role: "user", Content: "第二轮用户"},
			{ID: "msg-4", ThreadID: "session-7", Role: "assistant", Content: "第二轮助手"},
			{ID: "msg-5", ThreadID: "session-7", Role: "user", Content: "第三轮用户"},
			{ID: "msg-6", ThreadID: "session-7", Role: "assistant", Content: "第三轮助手"},
			{ID: "msg-7", ThreadID: "session-7", Role: "user", Content: "第四轮用户"},
			{ID: "msg-8", ThreadID: "session-7", Role: "assistant", Content: "第四轮助手"},
		},
	}
	runtimeStub := &titleRuntimeStub{
		result: runtimedto.RuntimeRunResult{
			AssistantMessage: runtimedto.Message{Content: "最近三轮总结"},
		},
	}
	service := &ThreadService{
		threads:  threadRepo,
		messages: messageRepo,
		runtime:  runtimeStub,
		now: func() time.Time {
			return time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
		},
	}

	resp, err := service.GenerateThreadTitle(context.Background(), dto.GenerateThreadTitleRequest{
		ThreadID: "session-7",
	})
	if err != nil {
		t.Fatalf("generate thread title: %v", err)
	}
	if !resp.Updated {
		t.Fatalf("expected title update")
	}
	prompt := runtimeStub.request.Input.Messages[0].Content
	if strings.Contains(prompt, "第一轮用户") || strings.Contains(prompt, "第一轮助手") {
		t.Fatalf("prompt should not include messages older than the latest three rounds, got %q", prompt)
	}
	for _, expected := range []string{"第二轮用户", "第二轮助手", "第三轮用户", "第三轮助手", "第四轮用户", "第四轮助手"} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("prompt should include %q, got %q", expected, prompt)
		}
	}
}

func TestGenerateThreadTitle_RejectsLatestUserRequestAsTitle(t *testing.T) {
	t.Parallel()

	threadRepo := &threadRepositoryStub{
		item: newThreadForTitleTest(t, "session-6", defaultThreadTitle, true),
	}
	runtimeStub := &titleRuntimeStub{
		result: runtimedto.RuntimeRunResult{
			AssistantMessage: runtimedto.Message{Content: "帮我做一个东京三日游计划"},
		},
	}
	service := &ThreadService{
		threads:  threadRepo,
		messages: &messageRepositoryStub{},
		runtime:  runtimeStub,
		now: func() time.Time {
			return time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
		},
	}

	resp, err := service.GenerateThreadTitle(context.Background(), dto.GenerateThreadTitleRequest{
		ThreadID: "session-6",
		Messages: []dto.GenerateThreadTitleMessage{
			{Role: "user", Content: "帮我做一个东京三日游计划"},
		},
	})
	if err != nil {
		t.Fatalf("generate thread title: %v", err)
	}

	if resp.Title != defaultThreadTitle {
		t.Fatalf("unexpected title: %q", resp.Title)
	}
	if !resp.TitleIsDefault {
		t.Fatalf("title should remain default when runtime copies latest user request")
	}
	if resp.Updated {
		t.Fatalf("copied user request should not update thread title")
	}
	if threadRepo.saveCall != 0 {
		t.Fatalf("copied user request should not trigger save")
	}
}

func TestGenerateThreadTitle_PublishesGatewayEventsForEmptyOneShotResult(t *testing.T) {
	t.Parallel()

	threadRepo := &threadRepositoryStub{
		item: newThreadForTitleTest(t, "session-debug", defaultThreadTitle, true),
	}
	messageRepo := &messageRepositoryStub{
		items: []thread.ThreadMessage{
			{ID: "msg-1", ThreadID: "session-debug", Role: "user", Content: "帮我看看这个 reasoning model 为什么没生成标题"},
		},
	}
	runtimeStub := &titleRuntimeStub{
		result: runtimedto.RuntimeRunResult{
			Status:       "completed",
			FinishReason: "stop",
			AssistantMessage: runtimedto.Message{
				Role: "assistant",
				Parts: []chatevent.MessagePart{{
					Type: "reasoning",
					Text: "先分析标题生成问题",
				}},
			},
		},
	}
	broker := gatewayevents.NewBroker(nil)
	records := make(chan gatewayevents.Record, 8)
	unsubscribe := broker.Subscribe(gatewayevents.Filter{SessionID: "session-debug"}, func(record gatewayevents.Record) {
		records <- record
	})
	defer unsubscribe()

	service := &ThreadService{
		threads:       threadRepo,
		messages:      messageRepo,
		runtime:       runtimeStub,
		gatewayEvents: broker,
		now: func() time.Time {
			return time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
		},
	}

	resp, err := service.GenerateThreadTitle(context.Background(), dto.GenerateThreadTitleRequest{
		ThreadID: "session-debug",
	})
	if err != nil {
		t.Fatalf("generate thread title: %v", err)
	}
	if resp.Updated {
		t.Fatalf("empty one-shot result should not update title")
	}

	gotTypes := make([]string, 0, 2)
	var emptyPayload threadTitleDebugPayload
	for len(records) > 0 {
		record := <-records
		gotTypes = append(gotTypes, record.Envelope.Type)
		if record.Envelope.Type == threadTitleEventEmpty {
			if err := json.Unmarshal(record.Payload, &emptyPayload); err != nil {
				t.Fatalf("decode empty payload: %v", err)
			}
		}
	}
	if len(gotTypes) == 0 {
		t.Fatal("expected gateway events to be published")
	}
	if gotTypes[0] != threadTitleEventStarted {
		t.Fatalf("expected first event %q, got %q", threadTitleEventStarted, gotTypes[0])
	}
	if gotTypes[len(gotTypes)-1] != threadTitleEventEmpty {
		t.Fatalf("expected last event %q, got %q", threadTitleEventEmpty, gotTypes[len(gotTypes)-1])
	}
	if emptyPayload.AssistantMessage == nil {
		t.Fatal("expected empty payload to include assistant message")
	}
	if len(emptyPayload.AssistantMessage.Parts) != 1 || emptyPayload.AssistantMessage.Parts[0].Type != "reasoning" {
		t.Fatalf("expected reasoning part in empty payload, got %#v", emptyPayload.AssistantMessage.Parts)
	}
	if emptyPayload.ContentChars != 0 {
		t.Fatalf("expected empty content chars, got %d", emptyPayload.ContentChars)
	}
}

func TestGenerateThreadTitle_PublishesGatewayEventsForRuntimeFailure(t *testing.T) {
	t.Parallel()

	threadRepo := &threadRepositoryStub{
		item: newThreadForTitleTest(t, "session-failed", defaultThreadTitle, true),
	}
	messageRepo := &messageRepositoryStub{
		items: []thread.ThreadMessage{
			{ID: "msg-1", ThreadID: "session-failed", Role: "user", Content: "生成一个标题"},
		},
	}
	runtimeStub := &titleRuntimeStub{
		err: errors.New("upstream timeout"),
	}
	broker := gatewayevents.NewBroker(nil)
	records := make(chan gatewayevents.Record, 8)
	unsubscribe := broker.Subscribe(gatewayevents.Filter{SessionID: "session-failed"}, func(record gatewayevents.Record) {
		records <- record
	})
	defer unsubscribe()

	service := &ThreadService{
		threads:       threadRepo,
		messages:      messageRepo,
		runtime:       runtimeStub,
		gatewayEvents: broker,
		now: func() time.Time {
			return time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
		},
	}

	resp, err := service.GenerateThreadTitle(context.Background(), dto.GenerateThreadTitleRequest{
		ThreadID: "session-failed",
	})
	if err != nil {
		t.Fatalf("generate thread title: %v", err)
	}
	if resp.Updated {
		t.Fatalf("failed one-shot should not update title")
	}

	gotTypes := make([]string, 0, 2)
	var failedPayload threadTitleDebugPayload
	for len(records) > 0 {
		record := <-records
		gotTypes = append(gotTypes, record.Envelope.Type)
		if record.Envelope.Type == threadTitleEventFailed {
			if err := json.Unmarshal(record.Payload, &failedPayload); err != nil {
				t.Fatalf("decode failed payload: %v", err)
			}
		}
	}
	if len(gotTypes) == 0 {
		t.Fatal("expected gateway events to be published")
	}
	if gotTypes[0] != threadTitleEventStarted {
		t.Fatalf("expected first event %q, got %q", threadTitleEventStarted, gotTypes[0])
	}
	if gotTypes[len(gotTypes)-1] != threadTitleEventFailed {
		t.Fatalf("expected last event %q, got %q", threadTitleEventFailed, gotTypes[len(gotTypes)-1])
	}
	if failedPayload.Error != "upstream timeout" {
		t.Fatalf("expected failed payload error, got %q", failedPayload.Error)
	}
}
