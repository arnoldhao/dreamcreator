package telegram

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"dreamcreator/internal/application/chatevent"
	gatewayapprovals "dreamcreator/internal/application/gateway/approvals"
	runtimedto "dreamcreator/internal/application/gateway/runtime/dto"
	settingsdto "dreamcreator/internal/application/settings/dto"
	skillsdto "dreamcreator/internal/application/skills/dto"
	domainproviders "dreamcreator/internal/domain/providers"
	domainsession "dreamcreator/internal/domain/session"
	telegramapi "github.com/mymmrac/telego"
)

func TestParseExecApprovalCommand(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		input          string
		wantID         string
		wantDecision   string
		wantHandled    bool
		wantUsageError bool
	}{
		{
			name:         "approve default decision",
			input:        "/approve req-1",
			wantID:       "req-1",
			wantDecision: "approve",
			wantHandled:  true,
		},
		{
			name:         "deny alias",
			input:        "/approve req-2 reject",
			wantID:       "req-2",
			wantDecision: "deny",
			wantHandled:  true,
		},
		{
			name:           "missing id",
			input:          "/approve",
			wantHandled:    true,
			wantUsageError: true,
		},
		{
			name:        "non approval command",
			input:       "/start",
			wantHandled: false,
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gotID, gotDecision, gotHandled, gotUsage := parseExecApprovalCommand(tc.input)
			if gotHandled != tc.wantHandled {
				t.Fatalf("handled mismatch: got %v want %v", gotHandled, tc.wantHandled)
			}
			if gotID != tc.wantID {
				t.Fatalf("id mismatch: got %q want %q", gotID, tc.wantID)
			}
			if gotDecision != tc.wantDecision {
				t.Fatalf("decision mismatch: got %q want %q", gotDecision, tc.wantDecision)
			}
			if (gotUsage != "") != tc.wantUsageError {
				t.Fatalf("usage error mismatch: got %q want usage error=%v", gotUsage, tc.wantUsageError)
			}
		})
	}
}

func TestIsCallbackApprovalCommand(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		update telegramapi.Update
		text   string
		want   bool
	}{
		{
			name: "callback approval command",
			update: telegramapi.Update{
				CallbackQuery: &telegramapi.CallbackQuery{
					ID:   "cb-1",
					Data: "/approve req-1 approve",
				},
			},
			text: "/approve req-1 approve",
			want: true,
		},
		{
			name: "callback non approval command",
			update: telegramapi.Update{
				CallbackQuery: &telegramapi.CallbackQuery{
					ID:   "cb-2",
					Data: "/help",
				},
			},
			text: "/help",
			want: false,
		},
		{
			name: "normal message approval command",
			update: telegramapi.Update{
				Message: &telegramapi.Message{
					MessageID: 1,
					Text:      "/approve req-1 approve",
				},
			},
			text: "/approve req-1 approve",
			want: false,
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := isCallbackApprovalCommand(tc.update, tc.text)
			if got != tc.want {
				t.Fatalf("callback approval detect mismatch: got %v want %v", got, tc.want)
			}
		})
	}
}

func TestIsExecApprovalUpdate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		update telegramapi.Update
		want   bool
	}{
		{
			name: "callback approval command",
			update: telegramapi.Update{
				CallbackQuery: &telegramapi.CallbackQuery{
					ID:   "cb-1",
					Data: "/approve req-1 approve",
				},
			},
			want: true,
		},
		{
			name: "message approval command",
			update: telegramapi.Update{
				Message: &telegramapi.Message{
					MessageID: 1,
					Text:      "/approve req-1 deny",
				},
			},
			want: true,
		},
		{
			name: "normal command",
			update: telegramapi.Update{
				Message: &telegramapi.Message{
					MessageID: 2,
					Text:      "/models",
				},
			},
			want: false,
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := isExecApprovalUpdate(tc.update)
			if got != tc.want {
				t.Fatalf("approval update mismatch: got %v want %v", got, tc.want)
			}
		})
	}
}

func TestResolveTelegramPollingWorkerCount(t *testing.T) {
	t.Parallel()
	cases := map[int]int{
		-1: 2,
		0:  2,
		1:  2,
		2:  2,
		3:  3,
	}
	for input, want := range cases {
		if got := resolveTelegramPollingWorkerCount(input); got != want {
			t.Fatalf("worker count mismatch for %d: got %d want %d", input, got, want)
		}
	}
}

func TestShouldSuppressTelegramResolvedForward(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		reason string
		want   bool
	}{
		{name: "telegram with sender", reason: "telegram:test-user-001", want: true},
		{name: "telegram plain", reason: "telegram", want: true},
		{name: "upper telegram", reason: "TeLeGrAm:1", want: true},
		{name: "gateway reason", reason: "gateway:web", want: false},
		{name: "empty reason", reason: "", want: false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := shouldSuppressTelegramResolvedForward(tc.reason)
			if got != tc.want {
				t.Fatalf("suppress mismatch for %q: got %v want %v", tc.reason, got, tc.want)
			}
		})
	}
}

func TestTelegramStatusCardSwapAndClear(t *testing.T) {
	t.Parallel()
	service := NewBotService(nil, nil, nil)

	if previous := service.swapTelegramStatusCard("acct-1", -1001, 0, 11); previous != 0 {
		t.Fatalf("expected no previous status card, got %d", previous)
	}
	if previous := service.swapTelegramStatusCard("acct-1", -1001, 0, 22); previous != 11 {
		t.Fatalf("expected previous status card 11, got %d", previous)
	}
	// Clearing an outdated message id should keep the current mapping intact.
	service.clearTelegramStatusCard("acct-1", -1001, 0, 11)
	if previous := service.swapTelegramStatusCard("acct-1", -1001, 0, 33); previous != 22 {
		t.Fatalf("expected previous status card 22 after stale clear, got %d", previous)
	}
	// Topic/thread isolation: same chat but different thread should not collide.
	if previous := service.swapTelegramStatusCard("acct-1", -1001, 5, 44); previous != 0 {
		t.Fatalf("expected no previous status card for different thread, got %d", previous)
	}
}

func TestTelegramRunStatusCardSwapAndClear(t *testing.T) {
	t.Parallel()
	service := NewBotService(nil, nil, nil)

	if previous := service.swapTelegramRunStatusCard("acct-1", -1001, 0, 11); previous != 0 {
		t.Fatalf("expected no previous run status card, got %d", previous)
	}
	if previous := service.swapTelegramRunStatusCard("acct-1", -1001, 0, 22); previous != 11 {
		t.Fatalf("expected previous run status card 11, got %d", previous)
	}
	// Clearing an outdated message id should keep the current mapping intact.
	service.clearTelegramRunStatusCard("acct-1", -1001, 0, 11)
	if previous := service.swapTelegramRunStatusCard("acct-1", -1001, 0, 33); previous != 22 {
		t.Fatalf("expected previous run status card 22 after stale clear, got %d", previous)
	}
	// Topic/thread isolation: same chat but different thread should not collide.
	if previous := service.swapTelegramRunStatusCard("acct-1", -1001, 5, 44); previous != 0 {
		t.Fatalf("expected no previous run status card for different thread, got %d", previous)
	}
}

func TestAllowMessage_ApprovalCommandBypassesMentionCheck(t *testing.T) {
	t.Parallel()
	state := &telegramAccountState{
		config: TelegramAccountConfig{
			GroupPolicy: GroupPolicyOpen,
		},
	}
	message := &telegramapi.Message{
		Chat: telegramapi.Chat{
			ID:   -10001,
			Type: "group",
		},
		From: &telegramapi.User{ID: 42},
	}

	allowed, reason, _ := allowMessage(state, message, "/approve req-1 approve", "", false)
	if allowed {
		t.Fatalf("expected command to be blocked by mention policy")
	}
	if reason != "group_requires_mention" {
		t.Fatalf("unexpected deny reason: %q", reason)
	}

	allowed, reason, _ = allowMessage(state, message, "/approve req-1 approve", "", true)
	if !allowed {
		t.Fatalf("expected approval command to bypass mention policy, reason=%q", reason)
	}
}

func TestHandleExecApprovalCommand_ResolvesWithoutMessage(t *testing.T) {
	t.Parallel()

	service := NewBotService(nil, nil, nil)
	resolver := &execApprovalResolverStub{}
	service.SetApprovalResolver(resolver)

	handled := service.handleExecApprovalCommand(
		context.Background(),
		&telegramAccountState{},
		nil,
		"/approve req-1 approve",
		"42",
	)
	if !handled {
		t.Fatal("expected approval command to be handled")
	}
	if !resolver.called {
		t.Fatal("expected resolver to be called even when callback message is unavailable")
	}
	if resolver.id != "req-1" {
		t.Fatalf("unexpected approval id: %q", resolver.id)
	}
	if resolver.decision != "approve" {
		t.Fatalf("unexpected decision: %q", resolver.decision)
	}
	if resolver.reason != "telegram:42" {
		t.Fatalf("unexpected reason: %q", resolver.reason)
	}
}

func TestHandleUpdate_CallbackApprovalRunsAsync(t *testing.T) {
	t.Parallel()

	service := NewBotService(nil, nil, nil)
	resolver := &blockingExecApprovalResolver{
		called:  make(chan struct{}, 1),
		release: make(chan struct{}),
	}
	service.SetApprovalResolver(resolver)
	state := &telegramAccountState{
		config: TelegramAccountConfig{
			AccountID: "acct-1",
		},
	}
	update := telegramapi.Update{
		CallbackQuery: &telegramapi.CallbackQuery{
			ID:   "cb-1",
			Data: "/approve req-1 approve",
			From: telegramapi.User{ID: 1001},
			Message: &telegramapi.InaccessibleMessage{
				MessageID: 123,
				Chat: telegramapi.Chat{
					ID:   -100001,
					Type: "supergroup",
				},
			},
		},
	}

	done := make(chan error, 1)
	go func() {
		done <- service.handleUpdate(context.Background(), state, update)
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("handleUpdate returned error: %v", err)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected callback approval update to return quickly")
	}

	select {
	case <-resolver.called:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected resolver to be invoked asynchronously")
	}
	close(resolver.release)
}

func TestFirstMessage_CallbackInaccessibleMessageFallback(t *testing.T) {
	t.Parallel()

	update := telegramapi.Update{
		CallbackQuery: &telegramapi.CallbackQuery{
			ID: "cb-1",
			Message: &telegramapi.InaccessibleMessage{
				MessageID: 321,
				Chat: telegramapi.Chat{
					ID:   -10012345,
					Type: "supergroup",
				},
			},
			Data: "/approve req-1 approve",
		},
	}

	message := firstMessage(update)
	if message == nil {
		t.Fatal("expected fallback message for inaccessible callback query message")
	}
	if message.MessageID != 321 {
		t.Fatalf("unexpected message id: %d", message.MessageID)
	}
	if message.Chat.ID != -10012345 {
		t.Fatalf("unexpected chat id: %d", message.Chat.ID)
	}
}

func TestParseTelegramSessionKey(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		key           string
		wantAccountID string
		wantChatID    int64
		wantThreadID  int
		wantParseOK   bool
	}{
		{
			name:          "with thread",
			key:           "telegram:account-a:group:-100123:thread:77",
			wantAccountID: "account-a",
			wantChatID:    -100123,
			wantThreadID:  77,
			wantParseOK:   true,
		},
		{
			name:          "without thread",
			key:           "telegram:account-b:private:42",
			wantAccountID: "account-b",
			wantChatID:    42,
			wantThreadID:  0,
			wantParseOK:   true,
		},
		{
			name:        "invalid key",
			key:         "discord:abc",
			wantParseOK: false,
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			accountID, chatID, threadID, ok := parseTelegramSessionKey(tc.key)
			if ok != tc.wantParseOK {
				t.Fatalf("parse result mismatch: got %v want %v", ok, tc.wantParseOK)
			}
			if !ok {
				return
			}
			if accountID != tc.wantAccountID {
				t.Fatalf("account mismatch: got %q want %q", accountID, tc.wantAccountID)
			}
			if chatID != tc.wantChatID {
				t.Fatalf("chat id mismatch: got %d want %d", chatID, tc.wantChatID)
			}
			if threadID != tc.wantThreadID {
				t.Fatalf("thread id mismatch: got %d want %d", threadID, tc.wantThreadID)
			}
		})
	}
}

func TestParseTelegramInboundCommand(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		input     string
		wantName  string
		wantArgs  string
		wantFound bool
	}{
		{
			name:      "basic command",
			input:     "/help",
			wantName:  "help",
			wantFound: true,
		},
		{
			name:      "command with bot suffix and args",
			input:     "/stop@dream_bot run-1",
			wantName:  "stop",
			wantArgs:  "run-1",
			wantFound: true,
		},
		{
			name:      "normal text",
			input:     "hello",
			wantFound: false,
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			command, found := parseTelegramInboundCommand(tc.input)
			if found != tc.wantFound {
				t.Fatalf("found mismatch: got %v want %v", found, tc.wantFound)
			}
			if !found {
				return
			}
			if command.Name != tc.wantName {
				t.Fatalf("name mismatch: got %q want %q", command.Name, tc.wantName)
			}
			if command.Args != tc.wantArgs {
				t.Fatalf("args mismatch: got %q want %q", command.Args, tc.wantArgs)
			}
		})
	}
}

func TestBuildTelegramToolStatusText(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		toolName string
		wantPart string
	}{
		{
			name:     "default tool",
			toolName: "",
			wantPart: "Running tool... 🛠️",
		},
		{
			name:     "web tool",
			toolName: "browser.navigate",
			wantPart: "🌐",
		},
		{
			name:     "coding tool",
			toolName: "exec_command",
			wantPart: "💻",
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := buildTelegramToolStatusText(tc.toolName)
			if !strings.Contains(got, tc.wantPart) {
				t.Fatalf("unexpected status text %q, expected to contain %q", got, tc.wantPart)
			}
		})
	}
}

func TestResolveTelegramSessionChannel(t *testing.T) {
	t.Parallel()
	v2Key, err := domainsession.BuildSessionKey(domainsession.KeyParts{
		Channel:   "aui",
		PrimaryID: "thread-1",
		ThreadRef: "thread-1",
	})
	if err != nil {
		t.Fatalf("build v2 key: %v", err)
	}
	tests := []struct {
		name      string
		sessionID string
		want      string
	}{
		{
			name:      "telegram legacy",
			sessionID: "telegram:acct:private:1001",
			want:      "telegram",
		},
		{
			name:      "v2 session",
			sessionID: v2Key,
			want:      "aui",
		},
		{
			name:      "app thread id",
			sessionID: "thread-123",
			want:      "app",
		},
		{
			name:      "unknown format",
			sessionID: "foo:bar:baz",
			want:      "unknown",
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := resolveTelegramSessionChannel(tc.sessionID); got != tc.want {
				t.Fatalf("channel mismatch: got %q want %q", got, tc.want)
			}
		})
	}
}

func TestResolveTelegramRuntimeSessionKey_PrefersSessionOverride(t *testing.T) {
	t.Parallel()
	service := NewBotService(nil, nil, nil)
	base := "telegram:acct:private:1001"
	withConversation := telegramSessionCommandState{ConversationID: "conv-a"}
	if got := service.resolveTelegramRuntimeSessionKey(base, withConversation); got != base+":conv:conv-a" {
		t.Fatalf("unexpected conversation session key: %q", got)
	}
	withOverride := telegramSessionCommandState{
		ConversationID:  "conv-a",
		SessionOverride: "thread-123",
	}
	if got := service.resolveTelegramRuntimeSessionKey(base, withOverride); got != "thread-123" {
		t.Fatalf("unexpected override session key: %q", got)
	}
}

func TestTelegramSessionSelectToken_Stable(t *testing.T) {
	t.Parallel()
	input := "v2::-::aui::-::thread-1::-::thread-1"
	first := telegramSessionSelectToken(input)
	second := telegramSessionSelectToken(input)
	if first == "" || second == "" {
		t.Fatalf("expected non-empty token")
	}
	if first != second {
		t.Fatalf("expected stable token, got %q and %q", first, second)
	}
}

func TestResolveTelegramSessionByToken(t *testing.T) {
	t.Parallel()
	items := []telegramSessionThread{
		{ThreadID: "thread-a", Title: "A"},
		{ThreadID: "thread-b", Title: "B"},
	}
	target := telegramSessionSelectToken("thread-b")
	got, ok, collision := resolveTelegramSessionByToken(items, target)
	if collision {
		t.Fatalf("did not expect collision")
	}
	if !ok {
		t.Fatalf("expected token to resolve")
	}
	if got.ThreadID != "thread-b" {
		t.Fatalf("unexpected resolved thread: %q", got.ThreadID)
	}
}

func TestWithTelegramEditTarget_ContextRoundTrip(t *testing.T) {
	t.Parallel()
	ctx := withTelegramEditTarget(context.Background(), 1001, 42)
	target, ok := telegramEditTargetFromContext(ctx)
	if !ok {
		t.Fatalf("expected edit target")
	}
	if target.ChatID != 1001 || target.MessageID != 42 {
		t.Fatalf("unexpected target: %+v", target)
	}
}

func TestAbortActiveRun_LocalCancel(t *testing.T) {
	t.Parallel()
	service := NewBotService(nil, nil, nil)
	called := false
	service.registerActiveRun("acct", "telegram:acct:private:1", "run-1", func() {
		called = true
	})

	aborted, runID, err := service.abortActiveRun(context.Background(), "acct", "telegram:acct:private:1", "", "test")
	if err != nil {
		t.Fatalf("abort failed: %v", err)
	}
	if !aborted {
		t.Fatalf("expected local run to abort")
	}
	if runID != "run-1" {
		t.Fatalf("unexpected run id: %q", runID)
	}
	if !called {
		t.Fatalf("expected cancel function to be called")
	}
}

func TestNormalizeTelegramQueueMode_Aliases(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"steer":         "steer",
		"steer-backlog": "steer",
		"interrupt":     "followup",
		"followup":      "followup",
		"collect":       "collect",
	}
	for input, want := range cases {
		if got := normalizeTelegramQueueMode(input); got != want {
			t.Fatalf("queue mode mismatch for %q: got %q want %q", input, got, want)
		}
	}
}

func TestDecorateTelegramReply_ReasoningAndUsage(t *testing.T) {
	t.Parallel()
	reply := decorateTelegramReply(
		"Final answer",
		runtimedto.RuntimeRunResult{
			Status: "success",
			Usage: runtimedto.RuntimeUsage{
				PromptTokens:     10,
				CompletionTokens: 5,
				TotalTokens:      15,
			},
			AssistantMessage: runtimedto.Message{
				Parts: []chatevent.MessagePart{
					{Type: "reasoning", Text: "internal thought"},
				},
			},
		},
		telegramSessionCommandState{
			ReasoningMode: "on",
			UsageMode:     "tokens",
			VerboseMode:   "on",
		},
	)
	if reply == "" {
		t.Fatalf("expected decorated reply")
	}
	if !containsAll(reply, []string{"Final answer", "Reasoning:", "Usage:", "Runtime status:"}) {
		t.Fatalf("decorated reply missing expected sections: %q", reply)
	}
}

func TestBuildTelegramModelRef_AvoidsDuplicatePrefix(t *testing.T) {
	t.Parallel()
	if got := buildTelegramModelRef("openai", "gpt-4o-mini"); got != "openai/gpt-4o-mini" {
		t.Fatalf("unexpected model ref: %q", got)
	}
	if got := buildTelegramModelRef("openai", "openai/gpt-4o-mini"); got != "openai/gpt-4o-mini" {
		t.Fatalf("unexpected prefixed model ref: %q", got)
	}
}

func TestBuildTelegramModelDisplayRef_UsesProviderName(t *testing.T) {
	t.Parallel()
	item := telegramModelRef{
		ProviderID:   "custom-provider",
		ProviderName: "My Custom",
		ModelName:    "model-a",
	}
	got := buildTelegramModelDisplayRef(item)
	if !strings.Contains(got, "My Custom/model-a") {
		t.Fatalf("unexpected display ref: %q", got)
	}
	if strings.Contains(got, "(custom-provider)") {
		t.Fatalf("did not expect provider id suffix in display ref: %q", got)
	}
}

func TestResolveModelSelection_AcceptsProviderName(t *testing.T) {
	t.Parallel()
	providerID := "provider-custom-a"
	providerName := "Acme AI"
	modelName := "model-a"
	service := NewBotService(nil, nil, nil)
	service.SetModelRepositories(
		providerRepoStub{
			items: []domainproviders.Provider{
				{ID: providerID, Name: providerName, Enabled: true},
			},
		},
		modelRepoStub{
			items: map[string][]domainproviders.Model{
				providerID: {
					{ProviderID: providerID, Name: modelName, Enabled: true},
				},
			},
		},
	)

	resolved, err := service.resolveModelSelection(context.Background(), telegramSessionCommandState{}, providerName+"/"+modelName)
	if err != nil {
		t.Fatalf("resolve model selection failed: %v", err)
	}
	if resolved.ProviderID != providerID {
		t.Fatalf("unexpected provider id: got %q want %q", resolved.ProviderID, providerID)
	}
	if resolved.ProviderName != providerName {
		t.Fatalf("unexpected provider name: got %q", resolved.ProviderName)
	}
}

func TestDescribeEffectiveModel_UsesProviderNameForSessionOverride(t *testing.T) {
	t.Parallel()
	providerID := "provider-custom-a"
	providerName := "Acme AI"
	modelName := "model-a"
	service := NewBotService(nil, nil, nil)
	service.SetModelRepositories(
		providerRepoStub{
			items: []domainproviders.Provider{
				{ID: providerID, Name: providerName, Enabled: true},
			},
		},
		modelRepoStub{
			items: map[string][]domainproviders.Model{
				providerID: {
					{ProviderID: providerID, Name: modelName, Enabled: true},
				},
			},
		},
	)

	got := service.describeEffectiveModel(context.Background(), telegramSessionCommandState{
		ModelProvider: providerID,
		ModelName:     modelName,
	})
	if got != "Acme AI/model-a (session override)" {
		t.Fatalf("unexpected model display: got %q", got)
	}
}

func TestParseTelegramModelsPage(t *testing.T) {
	t.Parallel()
	cases := map[string]int{
		"":         1,
		"2":        2,
		"page 3":   3,
		"page=4":   4,
		"unknown":  1,
		"page=bad": 1,
	}
	for input, want := range cases {
		if got := parseTelegramModelsPage(input); got != want {
			t.Fatalf("page parse mismatch for %q: got %d want %d", input, got, want)
		}
	}
}

func TestBuildTelegramRunFailureText_Default(t *testing.T) {
	t.Parallel()
	got := buildTelegramRunFailureText(nil, nil)
	if got != "Request failed. Please retry." {
		t.Fatalf("unexpected default error text: %q", got)
	}
}

func TestBuildTelegramRunFailureText_RedactsToken(t *testing.T) {
	t.Parallel()
	state := &telegramAccountState{
		config: TelegramAccountConfig{
			BotToken: "123456:abcdefSECRETtoken",
		},
	}
	got := buildTelegramRunFailureText(state, errors.New("upstream rejected bot123456:abcdefSECRETtoken request"))
	if !strings.Contains(got, "Request failed: ") {
		t.Fatalf("expected prefixed message, got %q", got)
	}
	if strings.Contains(got, "abcdefSECRETtoken") {
		t.Fatalf("expected token to be redacted, got %q", got)
	}
}

func TestIsTelegramRegisteredCustomCommandInSettings(t *testing.T) {
	t.Parallel()
	current := settingsdto.Settings{
		Gateway: settingsdto.GatewaySettings{
			ControlPlaneEnabled: true,
		},
		Channels: map[string]any{
			"telegram": map[string]any{
				"customCommands": []map[string]any{
					{
						"command":     "my-cmd",
						"description": "Custom command",
					},
					{
						"command":     "restart",
						"description": "Conflicts with native",
					},
					{
						"command":     "",
						"description": "Missing",
					},
				},
			},
		},
	}
	if !isTelegramRegisteredCustomCommandInSettings(current, "my_cmd") {
		t.Fatalf("expected valid custom command to be recognized")
	}
	if isTelegramRegisteredCustomCommandInSettings(current, "restart") {
		t.Fatalf("did not expect reserved native command to be treated as custom")
	}
	if isTelegramRegisteredCustomCommandInSettings(current, "missing") {
		t.Fatalf("did not expect unknown command to be treated as custom")
	}
}

func TestIsTelegramRegisteredCommandInSettings_RecognizesSkillCommands(t *testing.T) {
	t.Parallel()
	service := NewBotService(nil, nil, nil)
	service.SetSkillPromptResolver(skillPromptResolverStub{
		response: skillsdto.ResolveSkillPromptResponse{
			Items: []skillsdto.SkillPromptItem{
				{Name: "focus-mode"},
			},
		},
	})
	current := settingsdto.Settings{
		AgentModelProviderID: "provider-a",
		Gateway: settingsdto.GatewaySettings{
			ControlPlaneEnabled: true,
		},
		Channels: map[string]any{
			"telegram": map[string]any{
				"commands": map[string]any{
					"native":       true,
					"nativeSkills": true,
				},
			},
		},
	}
	if !service.isTelegramRegisteredCommandInSettings(context.Background(), current, "focus_mode") {
		t.Fatalf("expected skill command to be recognized as registered")
	}
	if service.isTelegramRegisteredCommandInSettings(context.Background(), current, "missing_skill") {
		t.Fatalf("did not expect unknown skill command to be recognized")
	}
}

type skillPromptResolverStub struct {
	response skillsdto.ResolveSkillPromptResponse
	err      error
}

func (stub skillPromptResolverStub) ResolveSkillPromptItems(_ context.Context, _ skillsdto.ResolveSkillPromptRequest) (skillsdto.ResolveSkillPromptResponse, error) {
	return stub.response, stub.err
}

type execApprovalResolverStub struct {
	called   bool
	id       string
	decision string
	reason   string
	err      error
}

func (stub *execApprovalResolverStub) Resolve(_ context.Context, id string, decision string, reason string) (gatewayapprovals.Request, error) {
	stub.called = true
	stub.id = id
	stub.decision = decision
	stub.reason = reason
	if stub.err != nil {
		return gatewayapprovals.Request{}, stub.err
	}
	return gatewayapprovals.Request{ID: id}, nil
}

type blockingExecApprovalResolver struct {
	called  chan struct{}
	release chan struct{}
}

func (stub *blockingExecApprovalResolver) Resolve(ctx context.Context, id string, decision string, reason string) (gatewayapprovals.Request, error) {
	if stub.called != nil {
		select {
		case stub.called <- struct{}{}:
		default:
		}
	}
	if stub.release != nil {
		select {
		case <-stub.release:
		case <-ctx.Done():
			return gatewayapprovals.Request{}, ctx.Err()
		}
	}
	return gatewayapprovals.Request{ID: id}, nil
}

func containsAll(text string, fragments []string) bool {
	for _, fragment := range fragments {
		if fragment == "" {
			continue
		}
		if !strings.Contains(text, fragment) {
			return false
		}
	}
	return true
}

type providerRepoStub struct {
	items []domainproviders.Provider
	err   error
}

func (stub providerRepoStub) List(_ context.Context) ([]domainproviders.Provider, error) {
	if stub.err != nil {
		return nil, stub.err
	}
	result := make([]domainproviders.Provider, len(stub.items))
	copy(result, stub.items)
	return result, nil
}

type modelRepoStub struct {
	items map[string][]domainproviders.Model
	err   error
}

func (stub modelRepoStub) ListByProvider(_ context.Context, providerID string) ([]domainproviders.Model, error) {
	if stub.err != nil {
		return nil, stub.err
	}
	items := stub.items[strings.TrimSpace(providerID)]
	result := make([]domainproviders.Model, len(items))
	copy(result, items)
	return result, nil
}
