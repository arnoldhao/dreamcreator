package tools

import (
	"context"
	"encoding/json"
	"testing"

	gatewayconfig "dreamcreator/internal/application/gateway/config"
	settingsdto "dreamcreator/internal/application/settings/dto"
)

type gatewayToolSettingsStub struct {
	settings settingsdto.Settings
	err      error
}

func (stub gatewayToolSettingsStub) GetSettings(context.Context) (settingsdto.Settings, error) {
	if stub.err != nil {
		return settingsdto.Settings{}, stub.err
	}
	return stub.settings, nil
}

type gatewayToolConfigStub struct {
	getFn    func(ctx context.Context, request gatewayconfig.ConfigGetRequest) (gatewayconfig.ConfigGetResponse, error)
	setFn    func(ctx context.Context, request gatewayconfig.ConfigSetRequest) (gatewayconfig.ConfigSetResponse, error)
	patchFn  func(ctx context.Context, request gatewayconfig.ConfigPatchRequest) (gatewayconfig.ConfigPatchResponse, error)
	applyFn  func(ctx context.Context, request gatewayconfig.ConfigApplyRequest) (gatewayconfig.ConfigApplyResponse, error)
	schemaFn func(ctx context.Context, request gatewayconfig.ConfigSchemaRequest) (gatewayconfig.ConfigSchemaResponse, error)
}

func (stub gatewayToolConfigStub) Get(ctx context.Context, request gatewayconfig.ConfigGetRequest) (gatewayconfig.ConfigGetResponse, error) {
	if stub.getFn != nil {
		return stub.getFn(ctx, request)
	}
	return gatewayconfig.ConfigGetResponse{}, nil
}

func (stub gatewayToolConfigStub) Set(ctx context.Context, request gatewayconfig.ConfigSetRequest) (gatewayconfig.ConfigSetResponse, error) {
	if stub.setFn != nil {
		return stub.setFn(ctx, request)
	}
	return gatewayconfig.ConfigSetResponse{}, nil
}

func (stub gatewayToolConfigStub) Patch(ctx context.Context, request gatewayconfig.ConfigPatchRequest) (gatewayconfig.ConfigPatchResponse, error) {
	if stub.patchFn != nil {
		return stub.patchFn(ctx, request)
	}
	return gatewayconfig.ConfigPatchResponse{}, nil
}

func (stub gatewayToolConfigStub) Apply(ctx context.Context, request gatewayconfig.ConfigApplyRequest) (gatewayconfig.ConfigApplyResponse, error) {
	if stub.applyFn != nil {
		return stub.applyFn(ctx, request)
	}
	return gatewayconfig.ConfigApplyResponse{}, nil
}

func (stub gatewayToolConfigStub) Schema(ctx context.Context, request gatewayconfig.ConfigSchemaRequest) (gatewayconfig.ConfigSchemaResponse, error) {
	if stub.schemaFn != nil {
		return stub.schemaFn(ctx, request)
	}
	return gatewayconfig.ConfigSchemaResponse{}, nil
}

func TestRunGatewayToolRequiresControlPlaneEnabled(t *testing.T) {
	handler := runGatewayTool(gatewayToolSettingsStub{
		settings: settingsdto.Settings{
			Gateway: settingsdto.GatewaySettings{
				ControlPlaneEnabled: false,
			},
		},
	}, gatewayToolConfigStub{})

	_, err := handler(context.Background(), `{"action":"gateway.ping"}`)
	if err == nil {
		t.Fatalf("expected gateway disabled error")
	}
	if err.Error() != "gateway control plane disabled" {
		t.Fatalf("expected gateway disabled error, got %q", err.Error())
	}
}

func TestRunGatewayToolConfigGet(t *testing.T) {
	var capturedPath string
	handler := runGatewayTool(gatewayToolSettingsStub{
		settings: settingsdto.Settings{
			Gateway: settingsdto.GatewaySettings{
				ControlPlaneEnabled: true,
			},
		},
	}, gatewayToolConfigStub{
		getFn: func(_ context.Context, request gatewayconfig.ConfigGetRequest) (gatewayconfig.ConfigGetResponse, error) {
			capturedPath = request.Path
			return gatewayconfig.ConfigGetResponse{
				Config:  map[string]any{"ok": true},
				Version: 7,
			}, nil
		},
	})

	output, err := handler(context.Background(), `{"action":"config.get","path":"/gateway"}`)
	if err != nil {
		t.Fatalf("run gateway tool: %v", err)
	}
	if capturedPath != "/gateway" {
		t.Fatalf("expected config.get path '/gateway', got %q", capturedPath)
	}
	response := map[string]any{}
	if unmarshalErr := json.Unmarshal([]byte(output), &response); unmarshalErr != nil {
		t.Fatalf("unmarshal gateway result: %v", unmarshalErr)
	}
	if response["action"] != "config.get" {
		t.Fatalf("expected action config.get, got %#v", response["action"])
	}
	result, ok := response["result"].(map[string]any)
	if !ok {
		t.Fatalf("expected result object, got %#v", response["result"])
	}
	if result["version"] != float64(7) {
		t.Fatalf("expected version=7, got %#v", result["version"])
	}
}

func TestRunGatewayToolMethodAliasAndUnsupportedAction(t *testing.T) {
	handler := runGatewayTool(gatewayToolSettingsStub{
		settings: settingsdto.Settings{
			Gateway: settingsdto.GatewaySettings{
				ControlPlaneEnabled: true,
			},
		},
	}, gatewayToolConfigStub{})

	output, err := handler(context.Background(), `{"method":"ping"}`)
	if err != nil {
		t.Fatalf("run gateway ping alias: %v", err)
	}
	response := map[string]any{}
	if unmarshalErr := json.Unmarshal([]byte(output), &response); unmarshalErr != nil {
		t.Fatalf("unmarshal ping result: %v", unmarshalErr)
	}
	if response["action"] != "gateway.ping" {
		t.Fatalf("expected canonical action gateway.ping, got %#v", response["action"])
	}

	_, err = handler(context.Background(), `{"action":"update.run"}`)
	if err == nil {
		t.Fatalf("expected unsupported action error")
	}
	if err.Error() != "unsupported gateway action: update.run" {
		t.Fatalf("expected unsupported action error, got %q", err.Error())
	}
}
