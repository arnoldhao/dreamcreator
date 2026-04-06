package heartbeat

import (
	"context"
	"strings"
	"testing"
	"time"

	runtimedto "dreamcreator/internal/application/gateway/runtime/dto"
	appnotice "dreamcreator/internal/application/notice"
	settingsservice "dreamcreator/internal/application/settings/service"
	threaddto "dreamcreator/internal/application/thread/dto"
	domainnotice "dreamcreator/internal/domain/notice"
	"dreamcreator/internal/domain/session"
	domainsettings "dreamcreator/internal/domain/settings"
	"dreamcreator/internal/domain/thread"
)

type deliveryTestRuntimeRunner struct {
	requests []runtimedto.RuntimeRunRequest
	result   runtimedto.RuntimeRunResult
	err      error
}

func (runner *deliveryTestRuntimeRunner) Run(_ context.Context, request runtimedto.RuntimeRunRequest) (runtimedto.RuntimeRunResult, error) {
	runner.requests = append(runner.requests, request)
	return runner.result, runner.err
}

type deliveryTestThreadWriter struct {
	requests []threaddto.AppendMessageRequest
}

func (writer *deliveryTestThreadWriter) AppendMessage(_ context.Context, request threaddto.AppendMessageRequest) error {
	writer.requests = append(writer.requests, request)
	return nil
}

type deliveryTestNoticePublisher struct {
	inputs []appnotice.CreateNoticeInput
}

func (publisher *deliveryTestNoticePublisher) Create(_ context.Context, input appnotice.CreateNoticeInput) (domainnotice.Notice, error) {
	publisher.inputs = append(publisher.inputs, input)
	return domainnotice.Notice{ID: "notice-1"}, nil
}

func TestRun_EventDrivenNeverDoesNotReplyInThread(t *testing.T) {
	t.Parallel()

	service, runtimeRunner, writer, notices, sessionKey := newHeartbeatDeliveryTestService(
		t,
		"never",
		`{"code":"heartbeat.exec_attention","severity":"error","params":{"detail":"Command failed: permission denied"}}`,
	)
	service.systemEvents.Enqueue(SystemEventInput{
		SessionKey: sessionKey,
		Text:       "exec finished: command failed",
		ContextKey: "exec:run-1",
		RunID:      "run-1",
		Source:     "exec",
	})

	result := service.run(context.Background(), TriggerInput{
		Reason:     "exec.completed",
		SessionKey: sessionKey,
		Force:      true,
	})

	if result.ExecutedStatus != TriggerExecutionRan {
		t.Fatalf("expected ran result, got %+v", result)
	}
	if len(runtimeRunner.requests) != 1 {
		t.Fatalf("expected one runtime request, got %d", len(runtimeRunner.requests))
	}
	prompt := runtimeRunner.requests[0].Input.Messages[0].Content
	if !containsAll(prompt,
		"Do not relay anything to the user or send chat messages.",
		"return only compact JSON with code, severity, params, and optional action.",
	) {
		t.Fatalf("expected non-relay heartbeat prompt, got %q", prompt)
	}
	if len(writer.requests) != 0 {
		t.Fatalf("expected no thread reply when mode=never, got %d writes", len(writer.requests))
	}
	if len(notices.inputs) != 1 {
		t.Fatalf("expected one notice, got %d", len(notices.inputs))
	}
	if notices.inputs[0].Code != "heartbeat.exec_attention" {
		t.Fatalf("expected exec attention notice, got %q", notices.inputs[0].Code)
	}
}

func TestRun_EventDrivenInlineRepliesInThread(t *testing.T) {
	t.Parallel()

	service, runtimeRunner, writer, notices, sessionKey := newHeartbeatDeliveryTestService(
		t,
		"inline",
		`{"code":"heartbeat.exec_attention","severity":"error","params":{"detail":"Command failed: permission denied"}}`,
	)
	service.systemEvents.Enqueue(SystemEventInput{
		SessionKey: sessionKey,
		Text:       "exec finished: command failed",
		ContextKey: "exec:run-1",
		RunID:      "run-1",
		Source:     "exec",
	})

	result := service.run(context.Background(), TriggerInput{
		Reason:     "exec.completed",
		SessionKey: sessionKey,
		Force:      true,
	})

	if result.ExecutedStatus != TriggerExecutionRan {
		t.Fatalf("expected ran result, got %+v", result)
	}
	if len(runtimeRunner.requests) != 1 {
		t.Fatalf("expected one runtime request, got %d", len(runtimeRunner.requests))
	}
	prompt := runtimeRunner.requests[0].Input.Messages[0].Content
	if !containsAll(prompt, "Please relay the command output to the user in a helpful way.") {
		t.Fatalf("expected relay heartbeat prompt, got %q", prompt)
	}
	if len(writer.requests) != 1 {
		t.Fatalf("expected one thread reply when mode=inline, got %d writes", len(writer.requests))
	}
	if writer.requests[0].Content != "Command failed: permission denied" {
		t.Fatalf("unexpected thread reply content: %q", writer.requests[0].Content)
	}
	if writer.requests[0].ThreadID == "" {
		t.Fatal("expected thread id to be populated")
	}
	if len(notices.inputs) != 1 {
		t.Fatalf("expected one notice, got %d", len(notices.inputs))
	}
}

func newHeartbeatDeliveryTestService(
	t *testing.T,
	threadReplyMode string,
	runtimeContent string,
) (*Service, *deliveryTestRuntimeRunner, *deliveryTestThreadWriter, *deliveryTestNoticePublisher, string) {
	t.Helper()

	now := time.Date(2026, time.April, 4, 9, 0, 0, 0, time.UTC)
	threadItem, err := thread.NewThread(thread.ThreadParams{
		ID:          "thread-delivery",
		AssistantID: "assistant-1",
		Title:       "Thread",
		CreatedAt:   &now,
		UpdatedAt:   &now,
	})
	if err != nil {
		t.Fatalf("new thread: %v", err)
	}
	sessionKey, err := session.BuildSessionKey(session.KeyParts{
		Channel:   "web",
		PrimaryID: threadItem.ID,
		ThreadRef: threadItem.ID,
	})
	if err != nil {
		t.Fatalf("build session key: %v", err)
	}

	heartbeatParams := &domainsettings.GatewayHeartbeatSettingsParams{
		Enabled:      boolPtr(true),
		RunSession:   stringPtr(sessionKey),
		Every:        stringPtr("30m"),
		EveryMinutes: intPtr(30),
		Prompt:       stringPtr(""),
		PromptAppend: stringPtr(""),
		Periodic: &domainsettings.GatewayHeartbeatPeriodicSettingsParams{
			Enabled: boolPtr(true),
			Every:   stringPtr("30m"),
		},
		Delivery: &domainsettings.GatewayHeartbeatDeliverySettingsParams{
			Periodic: &domainsettings.GatewayHeartbeatSurfacePolicyParams{
				Center: boolPtr(true),
			},
			EventDriven: &domainsettings.GatewayHeartbeatSurfacePolicyParams{
				Center: boolPtr(true),
			},
			ThreadReplyMode: stringPtr(threadReplyMode),
		},
	}
	defaults, err := domainsettings.NewSettings(domainsettings.SettingsParams{
		Appearance: string(domainsettings.AppearanceLight),
		Language:   domainsettings.DefaultLanguage.String(),
		Gateway: domainsettings.GatewaySettingsParams{
			Heartbeat: heartbeatParams,
		},
	})
	if err != nil {
		t.Fatalf("new settings: %v", err)
	}

	settingsRepo := &runtimeTestSettingsRepo{current: defaults}
	threadRepo := &runtimeTestThreadRepo{item: threadItem}
	runtimeRunner := &deliveryTestRuntimeRunner{
		result: runtimedto.RuntimeRunResult{
			AssistantMessage: runtimedto.Message{
				Role:    "assistant",
				Content: runtimeContent,
			},
		},
	}
	writer := &deliveryTestThreadWriter{}
	notices := &deliveryTestNoticePublisher{}
	service := NewService(
		settingsservice.NewSettingsService(settingsRepo, nil, defaults),
		threadRepo,
		nil,
		runtimeRunner,
		StoreOptions{},
		nil,
		notices,
	)
	service.writer = writer
	service.now = func() time.Time { return now }
	service.newID = func() string { return "hb-delivery" }
	return service, runtimeRunner, writer, notices, sessionKey
}

func containsAll(value string, expected ...string) bool {
	for _, part := range expected {
		if part == "" {
			continue
		}
		if !strings.Contains(value, part) {
			return false
		}
	}
	return true
}
