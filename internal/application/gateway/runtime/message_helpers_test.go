package runtime

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/cloudwego/eino/schema"

	"dreamcreator/internal/application/chatevent"
	runtimedto "dreamcreator/internal/application/gateway/runtime/dto"
	sessionapp "dreamcreator/internal/application/session"
	"dreamcreator/internal/domain/thread"
)

func TestDtoMessagesToSchema_IgnoresHistoricalAssistantToolParts(t *testing.T) {
	t.Parallel()

	result := dtoMessagesToSchema([]runtimedto.Message{
		{
			Role:    "user",
			Content: "我已经登陆了啊",
		},
		{
			Role: "assistant",
			Parts: []chatevent.MessagePart{
				{Type: "text", Text: "我刚刚检查了页面状态。"},
				{
					Type:       "tool-call",
					ToolCallID: "call-1",
					ToolName:   "browser",
					State:      "output-available",
					Input:      json.RawMessage(`{"action":"observe"}`),
					Output:     json.RawMessage(`{"items":[{"id":"login"}]}`),
				},
			},
		},
	})

	if len(result) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(result))
	}
	if result[0].Role != schema.User {
		t.Fatalf("expected first role user, got %q", result[0].Role)
	}
	if result[1].Role != schema.Assistant {
		t.Fatalf("expected second role assistant, got %q", result[1].Role)
	}
	if result[1].Content != "我刚刚检查了页面状态。" {
		t.Fatalf("unexpected assistant content: %q", result[1].Content)
	}
	if len(result[1].ToolCalls) != 0 {
		t.Fatalf("expected historical assistant tool calls to be dropped, got %d", len(result[1].ToolCalls))
	}
}

func TestDtoMessagesToSchema_UserAttachmentsIncludeWorkspacePaths(t *testing.T) {
	t.Parallel()

	result := dtoMessagesToSchema([]runtimedto.Message{
		{
			Role: "user",
			Parts: []chatevent.MessagePart{
				{Type: "text", Text: "请帮我看这个文件"},
				{
					Type: "file",
					Data: json.RawMessage(`{"path":"/tmp/workspace/secret/report.pdf","filename":"report.pdf"}`),
				},
			},
		},
	})

	if len(result) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result))
	}
	if !strings.Contains(result[0].Content, "/tmp/workspace/secret/report.pdf") {
		t.Fatalf("expected current input to preserve attachment path, got %q", result[0].Content)
	}
}

func TestStoredMessagesToSchema_UserAttachmentsDoNotLeakWorkspacePaths(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.April, 20, 12, 0, 0, 0, time.UTC)
	message, err := thread.NewThreadMessage(thread.ThreadMessageParams{
		ID:       "msg-user-attachment",
		ThreadID: "thread-attachment",
		Role:     "user",
		PartsJSON: `[{"type":"text","text":"请结合附件继续分析"},` +
			`{"type":"file","data":{"path":"/Users/arnold/Documents/private/tax-2025.pdf","filename":"tax-2025.pdf"}}]`,
		CreatedAt: &now,
	})
	if err != nil {
		t.Fatalf("new user message: %v", err)
	}

	result := storedMessagesToSchema([]thread.ThreadMessage{message})
	if len(result) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result))
	}
	if strings.Contains(result[0].Content, "/Users/arnold/Documents/private/tax-2025.pdf") {
		t.Fatalf("expected stored history to hide attachment path, got %q", result[0].Content)
	}
	if strings.Contains(result[0].Content, "tax-2025.pdf") {
		t.Fatalf("expected stored history to hide attachment filename, got %q", result[0].Content)
	}
	if !strings.Contains(result[0].Content, "Attachments were provided with this message.") {
		t.Fatalf("expected stored history to keep sanitized attachment hint, got %q", result[0].Content)
	}
}

type runtimeMessageRepositoryStub struct {
	items []thread.ThreadMessage
}

func (repo *runtimeMessageRepositoryStub) ListByThread(_ context.Context, threadID string, _ int) ([]thread.ThreadMessage, error) {
	result := make([]thread.ThreadMessage, 0, len(repo.items))
	for _, item := range repo.items {
		if item.ThreadID == threadID {
			result = append(result, item)
		}
	}
	return result, nil
}

func (repo *runtimeMessageRepositoryStub) Append(context.Context, thread.ThreadMessage) error {
	return nil
}

func (repo *runtimeMessageRepositoryStub) DeleteByThread(context.Context, string) error {
	return nil
}

func TestBuildPromptInputMessages_PrefersStoredThreadMessages(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.April, 20, 11, 0, 0, 0, time.UTC)
	userMessage, err := thread.NewThreadMessage(thread.ThreadMessageParams{
		ID:        "msg-user-1",
		ThreadID:  "thread-1",
		Role:      "user",
		Content:   "服务端已保存的用户消息",
		PartsJSON: `[{"type":"text","text":"服务端已保存的用户消息"}]`,
		CreatedAt: &now,
	})
	if err != nil {
		t.Fatalf("new user message: %v", err)
	}
	assistantMessage, err := thread.NewThreadMessage(thread.ThreadMessageParams{
		ID:       "msg-assistant-1",
		ThreadID: "thread-1",
		Role:     "assistant",
		Content:  "服务端已保存的 assistant 结论",
		PartsJSON: `[{"type":"text","text":"服务端已保存的 assistant 结论"},` +
			`{"type":"tool-call","toolCallId":"call-1","toolName":"browser","state":"output-available","output":{"items":[{"id":"login"}]}}]`,
		CreatedAt: &now,
	})
	if err != nil {
		t.Fatalf("new assistant message: %v", err)
	}
	service := &Service{
		messages: &runtimeMessageRepositoryStub{
			items: []thread.ThreadMessage{userMessage, assistantMessage},
		},
	}

	result, report, err := service.buildPromptInputMessages(context.Background(), "thread-1", []runtimedto.Message{{
		Role:    "user",
		Content: "客户端传来的不同内容",
	}}, true, promptContextBuildConfig{})
	if err != nil {
		t.Fatalf("build prompt input messages: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 prompt messages, got %d", len(result))
	}
	if result[0].Content != "服务端已保存的用户消息" {
		t.Fatalf("expected stored user message, got %q", result[0].Content)
	}
	if result[1].Content != "服务端已保存的 assistant 结论" {
		t.Fatalf("expected stored assistant message, got %q", result[1].Content)
	}
	if len(result[1].ToolCalls) != 0 {
		t.Fatalf("expected stored assistant tool calls to be dropped, got %d", len(result[1].ToolCalls))
	}
	if report.Source != "stored" || report.StoredMessageCount != 2 || report.BuiltMessageCount != 2 {
		t.Fatalf("unexpected prompt context report: %+v", report)
	}
}

func TestBuildPromptInputMessages_UsesPersistedCompactionState(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.April, 20, 11, 30, 0, 0, time.UTC)
	messages := []thread.ThreadMessage{
		mustThreadMessage(t, "msg-1", "thread-ctx-1", "user", "第一轮用户消息", now),
		mustThreadMessage(t, "msg-2", "thread-ctx-1", "assistant", "第一轮结论", now.Add(time.Second)),
		mustThreadMessage(t, "msg-3", "thread-ctx-1", "user", "第二轮用户消息", now.Add(2*time.Second)),
	}
	sessionService := sessionapp.NewService(sessionapp.NewInMemoryStore())
	if _, err := sessionService.CreateSession(context.Background(), sessionapp.CreateSessionRequest{
		SessionID: "thread-ctx-1",
	}); err != nil {
		t.Fatalf("create session: %v", err)
	}
	if err := sessionService.UpdateContextCompactionState(context.Background(), "thread-ctx-1", sessionapp.ContextCompactionStateUpdate{
		Summary:            "早期历史已概括。",
		FirstKeptMessageID: "msg-2",
		StrategyVersion:    persistedContextStrategyVersion,
		CompactedAt:        now,
	}); err != nil {
		t.Fatalf("update context compaction state: %v", err)
	}
	service := &Service{
		messages: &runtimeMessageRepositoryStub{items: messages},
		sessions: sessionService,
	}

	result, report, err := service.buildPromptInputMessages(context.Background(), "thread-ctx-1", nil, true, promptContextBuildConfig{})
	if err != nil {
		t.Fatalf("build prompt input messages: %v", err)
	}

	if len(result) != 3 {
		t.Fatalf("expected summary + 2 kept messages, got %d", len(result))
	}
	if result[0].Role != schema.System || result[0].Content != compactionSummaryPrefix+"早期历史已概括。" {
		t.Fatalf("unexpected summary message: role=%q content=%q", result[0].Role, result[0].Content)
	}
	if result[1].Content != "第一轮结论" || result[2].Content != "第二轮用户消息" {
		t.Fatalf("unexpected kept suffix: %q / %q", result[1].Content, result[2].Content)
	}
	if !report.UsedPersistedSummary || report.PersistedFirstKeptMessageID != "msg-2" {
		t.Fatalf("expected persisted summary report, got %+v", report)
	}
}

func TestBuildPromptInputMessages_ClearsStalePersistedCompactionState(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.April, 20, 11, 45, 0, 0, time.UTC)
	sessionService := sessionapp.NewService(sessionapp.NewInMemoryStore())
	if _, err := sessionService.CreateSession(context.Background(), sessionapp.CreateSessionRequest{
		SessionID: "thread-ctx-stale",
	}); err != nil {
		t.Fatalf("create session: %v", err)
	}
	if err := sessionService.UpdateContextCompactionState(context.Background(), "thread-ctx-stale", sessionapp.ContextCompactionStateUpdate{
		Summary:            "过期摘要",
		FirstKeptMessageID: "missing-message-id",
		StrategyVersion:    persistedContextStrategyVersion,
		CompactedAt:        now,
	}); err != nil {
		t.Fatalf("update context compaction state: %v", err)
	}
	service := &Service{
		messages: &runtimeMessageRepositoryStub{
			items: []thread.ThreadMessage{
				mustThreadMessage(t, "msg-1", "thread-ctx-stale", "user", "现存消息", now),
			},
		},
		sessions: sessionService,
	}

	result, report, err := service.buildPromptInputMessages(context.Background(), "thread-ctx-stale", nil, true, promptContextBuildConfig{})
	if err != nil {
		t.Fatalf("build prompt input messages: %v", err)
	}
	if len(result) != 1 || result[0].Content != "现存消息" {
		t.Fatalf("expected fallback to stored messages, got %+v", result)
	}
	stored, err := sessionService.Get(context.Background(), "thread-ctx-stale")
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	if stored.ContextSummary != "" || stored.ContextFirstKeptMessageID != "" {
		t.Fatalf("expected stale compaction state to be cleared, got summary=%q boundary=%q", stored.ContextSummary, stored.ContextFirstKeptMessageID)
	}
	if !report.ClearedStalePersistedSummary {
		t.Fatalf("expected stale persisted summary flag, got %+v", report)
	}
}

func mustThreadMessage(t *testing.T, id string, threadID string, role string, content string, createdAt time.Time) thread.ThreadMessage {
	t.Helper()
	message, err := thread.NewThreadMessage(thread.ThreadMessageParams{
		ID:        id,
		ThreadID:  threadID,
		Role:      role,
		Content:   content,
		PartsJSON: `[{"type":"text","text":"` + content + `"}]`,
		CreatedAt: &createdAt,
	})
	if err != nil {
		t.Fatalf("new thread message: %v", err)
	}
	return message
}

func TestBuildPromptInputMessages_AppliesTokenBudgetToRecentSuffix(t *testing.T) {
	t.Parallel()

	service := &Service{}

	result, report, err := service.buildPromptInputMessages(context.Background(), "thread-budget-1", []runtimedto.Message{
		{Role: "user", Content: strings.Repeat("a", 80)},
		{Role: "assistant", Content: strings.Repeat("b", 80)},
		{Role: "user", Content: "short tail"},
	}, false, promptContextBuildConfig{
		contextWindowTokens: 10,
	})
	if err != nil {
		t.Fatalf("build prompt input messages: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("expected only most recent message to fit budget, got %d", len(result))
	}
	if result[0].Content != "short tail" {
		t.Fatalf("expected latest message to be kept, got %q", result[0].Content)
	}
	if !report.BudgetApplied || report.Source != "incoming" {
		t.Fatalf("expected budget report on incoming messages, got %+v", report)
	}
}
