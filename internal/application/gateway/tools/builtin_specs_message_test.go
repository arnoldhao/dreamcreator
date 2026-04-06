package tools

import (
	"context"
	"encoding/json"
	"testing"
)

func TestSpecMessageDynamicSchemaTelegram(t *testing.T) {
	spec := specMessage(context.Background(), messageToolSettingsStub{settings: baseMessageToolSettings()})
	properties, actionEnum := decodeMessageToolSchema(t, spec.SchemaJSON)

	assertMessageSchemaHasAll(t, actionEnum, []string{
		"broadcast",
		"delete",
		"edit",
		"poll",
		"react",
		"reply",
		"send",
		"sendAttachment",
		"sendWithEffect",
		"thread-reply",
		"unsend",
	})
	if _, ok := properties["buttons"]; !ok {
		t.Fatalf("expected telegram schema to include buttons")
	}
	if _, ok := properties["card"]; ok {
		t.Fatalf("expected telegram schema to hide card")
	}
	if _, ok := properties["components"]; ok {
		t.Fatalf("expected telegram schema to hide components")
	}
}

func TestSpecMessageFallbackSchemaKeepsFullSurface(t *testing.T) {
	spec := specMessage(context.Background(), nil)
	properties, actionEnum := decodeMessageToolSchema(t, spec.SchemaJSON)

	assertMessageSchemaHasAll(t, actionEnum, []string{"send", "sendWithEffect", "set-presence"})
	if _, ok := properties["buttons"]; !ok {
		t.Fatalf("expected fallback schema to include buttons")
	}
	if _, ok := properties["card"]; !ok {
		t.Fatalf("expected fallback schema to include card")
	}
	if _, ok := properties["components"]; !ok {
		t.Fatalf("expected fallback schema to include components")
	}
}

func TestSpecMessageDynamicSchemaRespectsTelegramActionGates(t *testing.T) {
	settings := baseMessageToolSettings()
	telegramCfg, _ := settings.Channels["telegram"].(map[string]any)
	telegramCfg["actions"] = map[string]any{
		"sendMessage":   false,
		"polls":         true,
		"reactions":     false,
		"deleteMessage": false,
		"editMessage":   true,
	}
	spec := specMessage(context.Background(), messageToolSettingsStub{settings: settings})
	_, actionEnum := decodeMessageToolSchema(t, spec.SchemaJSON)

	assertMessageSchemaHasAll(t, actionEnum, []string{"poll", "edit"})
	assertMessageSchemaHasNone(t, actionEnum, []string{
		"send",
		"broadcast",
		"reply",
		"sendWithEffect",
		"sendAttachment",
		"thread-reply",
		"react",
		"delete",
		"unsend",
	})
}

func decodeMessageToolSchema(t *testing.T, raw string) (map[string]any, []string) {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("unmarshal schema: %v", err)
	}
	properties, _ := payload["properties"].(map[string]any)
	if properties == nil {
		t.Fatalf("schema properties missing")
	}
	actionSchema, _ := properties["action"].(map[string]any)
	if actionSchema == nil {
		t.Fatalf("schema action missing")
	}
	enumRaw, _ := actionSchema["enum"].([]any)
	actions := make([]string, 0, len(enumRaw))
	for _, item := range enumRaw {
		value, _ := item.(string)
		if value != "" {
			actions = append(actions, value)
		}
	}
	return properties, actions
}

func assertMessageSchemaHasAll(t *testing.T, actions []string, expected []string) {
	t.Helper()
	set := make(map[string]struct{}, len(actions))
	for _, action := range actions {
		set[action] = struct{}{}
	}
	for _, action := range expected {
		if _, ok := set[action]; !ok {
			t.Fatalf("expected action %q in schema enum, got %#v", action, actions)
		}
	}
}

func assertMessageSchemaHasNone(t *testing.T, actions []string, unexpected []string) {
	t.Helper()
	set := make(map[string]struct{}, len(actions))
	for _, action := range actions {
		set[action] = struct{}{}
	}
	for _, action := range unexpected {
		if _, ok := set[action]; ok {
			t.Fatalf("did not expect action %q in schema enum, got %#v", action, actions)
		}
	}
}
