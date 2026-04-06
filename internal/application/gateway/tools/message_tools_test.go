package tools

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"

	settingsdto "dreamcreator/internal/application/settings/dto"
)

type messageToolSettingsStub struct {
	settings settingsdto.Settings
	err      error
}

func (stub messageToolSettingsStub) GetSettings(context.Context) (settingsdto.Settings, error) {
	if stub.err != nil {
		return settingsdto.Settings{}, stub.err
	}
	return stub.settings, nil
}

func TestRunMessageToolSendDryRun(t *testing.T) {
	handler := runMessageTool(messageToolSettingsStub{settings: baseMessageToolSettings()})

	output, err := handler(context.Background(), `{"action":"send","target":"telegram:-100123:topic:7","message":"hello","dryRun":true}`)
	if err != nil {
		t.Fatalf("run message tool: %v", err)
	}
	payload := map[string]any{}
	if unmarshalErr := json.Unmarshal([]byte(output), &payload); unmarshalErr != nil {
		t.Fatalf("unmarshal result: %v", unmarshalErr)
	}
	if payload["ok"] != true {
		t.Fatalf("expected ok=true, got %#v", payload["ok"])
	}
	if payload["kind"] != "send" {
		t.Fatalf("expected kind send, got %#v", payload["kind"])
	}
	if payload["to"] != "telegram:-100123:topic:7" {
		t.Fatalf("unexpected target: %#v", payload["to"])
	}
	if payload["action"] != "send" {
		t.Fatalf("expected action send, got %#v", payload["action"])
	}
}

func TestRunMessageToolUsesRuntimeSessionTarget(t *testing.T) {
	handler := runMessageTool(messageToolSettingsStub{settings: baseMessageToolSettings()})
	ctx := WithRuntimeContext(context.Background(), "telegram:default:group:-100222:thread:99", "run-1")

	output, err := handler(ctx, `{"action":"send","message":"hello","dryRun":true}`)
	if err != nil {
		t.Fatalf("run message tool: %v", err)
	}
	payload := map[string]any{}
	if unmarshalErr := json.Unmarshal([]byte(output), &payload); unmarshalErr != nil {
		t.Fatalf("unmarshal result: %v", unmarshalErr)
	}
	if payload["to"] != "telegram:-100222:topic:99" {
		t.Fatalf("expected runtime target fallback, got %#v", payload["to"])
	}
}

func TestRunMessageToolBroadcastDryRun(t *testing.T) {
	handler := runMessageTool(messageToolSettingsStub{settings: baseMessageToolSettings()})

	output, err := handler(context.Background(), `{"action":"broadcast","targets":["telegram:-1001","telegram:-1002:topic:3"],"message":"ping","dryRun":true}`)
	if err != nil {
		t.Fatalf("run message tool: %v", err)
	}
	payload := map[string]any{}
	if unmarshalErr := json.Unmarshal([]byte(output), &payload); unmarshalErr != nil {
		t.Fatalf("unmarshal result: %v", unmarshalErr)
	}
	results, ok := payload["results"].([]any)
	if !ok {
		t.Fatalf("expected results array, got %#v", payload["results"])
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 broadcast results, got %d", len(results))
	}
}

func TestRunMessageToolActionDisabled(t *testing.T) {
	settings := baseMessageToolSettings()
	telegramCfg := settings.Channels["telegram"].(map[string]any)
	telegramCfg["actions"] = map[string]any{"sendMessage": false}
	handler := runMessageTool(messageToolSettingsStub{settings: settings})

	_, err := handler(context.Background(), `{"action":"send","target":"telegram:-1001","message":"x","dryRun":true}`)
	if err == nil {
		t.Fatalf("expected disabled action error")
	}
	if err.Error() != "telegram sendMessage is disabled" {
		t.Fatalf("unexpected error: %q", err.Error())
	}
}

func TestRunMessageToolUnsupportedAction(t *testing.T) {
	handler := runMessageTool(messageToolSettingsStub{settings: baseMessageToolSettings()})

	_, err := handler(context.Background(), `{"action":"channel-create","dryRun":true}`)
	if err == nil {
		t.Fatalf("expected unsupported action error")
	}
}

func TestResolveMessageToolMediaBufferDataURL(t *testing.T) {
	data := base64.StdEncoding.EncodeToString([]byte("hello"))
	media, err := resolveMessageToolMedia(toolArgs{
		"buffer": "data:image/png;base64," + data,
	})
	if err != nil {
		t.Fatalf("resolve media: %v", err)
	}
	if media == nil {
		t.Fatalf("expected media result")
	}
	if string(media.payload) != "hello" {
		t.Fatalf("unexpected payload: %q", string(media.payload))
	}
	if media.contentType != "image/png" {
		t.Fatalf("unexpected content type: %q", media.contentType)
	}
	if media.filename != "attachment.png" {
		t.Fatalf("unexpected filename: %q", media.filename)
	}
	if media.document {
		t.Fatalf("expected image buffer to use photo mode")
	}
}

func TestResolveMessageToolMediaPathInfersDocument(t *testing.T) {
	media, err := resolveMessageToolMedia(toolArgs{
		"media": "/tmp/report.pdf",
	})
	if err != nil {
		t.Fatalf("resolve media: %v", err)
	}
	if media == nil {
		t.Fatalf("expected media result")
	}
	if !media.document {
		t.Fatalf("expected document mode for pdf")
	}
	if media.filename != "report.pdf" {
		t.Fatalf("unexpected filename: %q", media.filename)
	}
}

func TestResolveMessageToolGatewayOptions(t *testing.T) {
	options := resolveMessageToolGatewayOptions(toolArgs{
		"gatewayUrl":   "https://example.com/bot/",
		"gatewayToken": "token-1",
		"timeoutMs":    2500,
	})
	if options.url != "https://example.com/bot" {
		t.Fatalf("unexpected url: %q", options.url)
	}
	if options.token != "token-1" {
		t.Fatalf("unexpected token: %q", options.token)
	}
	if int(options.timeout.Milliseconds()) != 2500 {
		t.Fatalf("unexpected timeout: %s", options.timeout)
	}
}

func TestParseMessageToolTelegramTargetUsernameAndThread(t *testing.T) {
	target, err := parseMessageToolTelegramTarget("telegram:@mychannel:topic:9")
	if err != nil {
		t.Fatalf("parse target: %v", err)
	}
	if target.chat != "@mychannel" {
		t.Fatalf("unexpected chat: %q", target.chat)
	}
	if target.chatID != 0 {
		t.Fatalf("expected non-numeric chat id")
	}
	if target.threadID != 9 {
		t.Fatalf("unexpected thread: %d", target.threadID)
	}
}

func TestParseMessageToolTelegramTargetTMeLink(t *testing.T) {
	target, err := parseMessageToolTelegramTarget("https://t.me/MyChannel")
	if err != nil {
		t.Fatalf("parse target: %v", err)
	}
	if target.chat != "@MyChannel" {
		t.Fatalf("unexpected chat: %q", target.chat)
	}
}

func TestResolveMessageToolTelegramSchemaActionsIncludesImplementedActions(t *testing.T) {
	actions := resolveMessageToolTelegramSchemaActions(baseMessageToolSettings())
	set := make(map[string]struct{}, len(actions))
	for _, action := range actions {
		set[action] = struct{}{}
	}
	expected := []string{
		"send",
		"broadcast",
		"reply",
		"sendWithEffect",
		"sendAttachment",
		"thread-reply",
		"poll",
		"react",
		"delete",
		"unsend",
		"edit",
	}
	for _, action := range expected {
		if _, ok := set[action]; !ok {
			t.Fatalf("expected action %q in schema profile, got %#v", action, actions)
		}
	}
}

func baseMessageToolSettings() settingsdto.Settings {
	return settingsdto.Settings{
		Channels: map[string]any{
			"telegram": map[string]any{
				"enabled":  true,
				"botToken": "123456:ABCDEF",
			},
		},
	}
}
