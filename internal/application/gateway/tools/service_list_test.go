package tools

import (
	"context"
	"testing"

	settingsdto "dreamcreator/internal/application/settings/dto"
	tooldto "dreamcreator/internal/application/tools/dto"
	toolservice "dreamcreator/internal/application/tools/service"
)

type mutableMessageToolSettingsStub struct {
	settings settingsdto.Settings
}

func (stub *mutableMessageToolSettingsStub) GetSettings(context.Context) (settingsdto.Settings, error) {
	return stub.settings, nil
}

func TestServiceListToolsRefreshesMessageSchemaFromLatestSettings(t *testing.T) {
	t.Parallel()

	settingsStub := &mutableMessageToolSettingsStub{
		settings: settingsdto.Settings{
			Channels: map[string]any{
				"telegram": map[string]any{
					"enabled": true,
				},
			},
		},
	}
	tools := toolservice.NewToolService()
	if _, err := tools.RegisterTool(context.Background(), tooldto.RegisterToolRequest{
		Spec: specMessageBase().toDTO(),
	}); err != nil {
		t.Fatalf("register message tool: %v", err)
	}
	service := NewService(tools, nil, nil, settingsStub, nil, nil)

	first := findToolByID(service.ListTools(context.Background()), "message")
	if first == nil {
		t.Fatalf("message tool not listed")
	}
	_, firstActions := decodeMessageToolSchema(t, first.SchemaJSON)
	assertMessageSchemaHasAll(t, firstActions, []string{"send", "broadcast"})
	if containsMessageAction(firstActions, "react") {
		t.Fatalf("did not expect react before token is configured")
	}

	settingsStub.settings.Channels["telegram"] = map[string]any{
		"enabled":  true,
		"botToken": "123456:ABCDEF",
	}
	second := findToolByID(service.ListTools(context.Background()), "message")
	if second == nil {
		t.Fatalf("message tool not listed after update")
	}
	_, secondActions := decodeMessageToolSchema(t, second.SchemaJSON)
	assertMessageSchemaHasAll(t, secondActions, []string{
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
	})
}

func findToolByID(specs []tooldto.ToolSpec, id string) *tooldto.ToolSpec {
	for i := range specs {
		if specs[i].ID == id {
			return &specs[i]
		}
	}
	return nil
}

func containsMessageAction(actions []string, expected string) bool {
	for _, action := range actions {
		if action == expected {
			return true
		}
	}
	return false
}
