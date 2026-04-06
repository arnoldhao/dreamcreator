package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	gatewayconfig "dreamcreator/internal/application/gateway/config"
)

type GatewayConfigToolService interface {
	Get(ctx context.Context, request gatewayconfig.ConfigGetRequest) (gatewayconfig.ConfigGetResponse, error)
	Set(ctx context.Context, request gatewayconfig.ConfigSetRequest) (gatewayconfig.ConfigSetResponse, error)
	Patch(ctx context.Context, request gatewayconfig.ConfigPatchRequest) (gatewayconfig.ConfigPatchResponse, error)
	Apply(ctx context.Context, request gatewayconfig.ConfigApplyRequest) (gatewayconfig.ConfigApplyResponse, error)
	Schema(ctx context.Context, request gatewayconfig.ConfigSchemaRequest) (gatewayconfig.ConfigSchemaResponse, error)
}

type gatewayActionDefinition struct {
	name       string
	inputType  reflect.Type
	outputType reflect.Type
	execute    func(ctx context.Context, params map[string]any, configService GatewayConfigToolService) (any, error)
}

type gatewayPingResult struct {
	Status string `json:"status"`
}

var gatewayActionDefinitions = []gatewayActionDefinition{
	{
		name:       "gateway.ping",
		inputType:  reflect.TypeOf(struct{}{}),
		outputType: reflect.TypeOf(gatewayPingResult{}),
		execute: func(_ context.Context, _ map[string]any, _ GatewayConfigToolService) (any, error) {
			return gatewayPingResult{Status: "ok"}, nil
		},
	},
	newGatewayConfigActionDefinition[gatewayconfig.ConfigGetRequest, gatewayconfig.ConfigGetResponse](
		"config.get",
		func(ctx context.Context, configService GatewayConfigToolService, request gatewayconfig.ConfigGetRequest) (gatewayconfig.ConfigGetResponse, error) {
			return configService.Get(ctx, request)
		},
	),
	newGatewayConfigActionDefinition[gatewayconfig.ConfigSetRequest, gatewayconfig.ConfigSetResponse](
		"config.set",
		func(ctx context.Context, configService GatewayConfigToolService, request gatewayconfig.ConfigSetRequest) (gatewayconfig.ConfigSetResponse, error) {
			return configService.Set(ctx, request)
		},
	),
	newGatewayConfigActionDefinition[gatewayconfig.ConfigPatchRequest, gatewayconfig.ConfigPatchResponse](
		"config.patch",
		func(ctx context.Context, configService GatewayConfigToolService, request gatewayconfig.ConfigPatchRequest) (gatewayconfig.ConfigPatchResponse, error) {
			return configService.Patch(ctx, request)
		},
	),
	newGatewayConfigActionDefinition[gatewayconfig.ConfigApplyRequest, gatewayconfig.ConfigApplyResponse](
		"config.apply",
		func(ctx context.Context, configService GatewayConfigToolService, request gatewayconfig.ConfigApplyRequest) (gatewayconfig.ConfigApplyResponse, error) {
			return configService.Apply(ctx, request)
		},
	),
	newGatewayConfigActionDefinition[gatewayconfig.ConfigSchemaRequest, gatewayconfig.ConfigSchemaResponse](
		"config.schema",
		func(ctx context.Context, configService GatewayConfigToolService, request gatewayconfig.ConfigSchemaRequest) (gatewayconfig.ConfigSchemaResponse, error) {
			return configService.Schema(ctx, request)
		},
	),
}

var gatewayToolActions = collectGatewayActionNames(gatewayActionDefinitions)
var gatewayActionsByName = indexGatewayActionDefinitions(gatewayActionDefinitions)

func newGatewayConfigActionDefinition[Req any, Resp any](
	name string,
	executor func(ctx context.Context, configService GatewayConfigToolService, request Req) (Resp, error),
) gatewayActionDefinition {
	return gatewayActionDefinition{
		name:       name,
		inputType:  reflect.TypeOf(*new(Req)),
		outputType: reflect.TypeOf(*new(Resp)),
		execute: func(ctx context.Context, params map[string]any, configService GatewayConfigToolService) (any, error) {
			if configService == nil {
				return nil, errors.New("gateway config service unavailable")
			}
			request := *new(Req)
			if err := decodeGatewayActionParams(name, params, &request); err != nil {
				return nil, err
			}
			response, err := executor(ctx, configService, request)
			if err != nil {
				return nil, err
			}
			return response, nil
		},
	}
}

func collectGatewayActionNames(definitions []gatewayActionDefinition) []string {
	names := make([]string, 0, len(definitions))
	for _, definition := range definitions {
		name := strings.TrimSpace(definition.name)
		if name == "" {
			continue
		}
		names = append(names, name)
	}
	return names
}

func indexGatewayActionDefinitions(definitions []gatewayActionDefinition) map[string]gatewayActionDefinition {
	index := make(map[string]gatewayActionDefinition, len(definitions))
	for _, definition := range definitions {
		name := strings.ToLower(strings.TrimSpace(definition.name))
		if name == "" {
			continue
		}
		index[name] = definition
	}
	return index
}

func runGatewayTool(
	settings SettingsReader,
	configService GatewayConfigToolService,
) func(ctx context.Context, args string) (string, error) {
	return func(ctx context.Context, args string) (string, error) {
		if err := ensureGatewayControlPlaneEnabled(ctx, settings); err != nil {
			return "", err
		}
		payload, err := parseToolArgs(args)
		if err != nil {
			return "", err
		}
		action, params := resolveGatewayActionAndParams(payload)
		if action == "" {
			return "", errors.New("gateway action is required")
		}
		return runGatewayAction(ctx, action, params, configService)
	}
}

func ensureGatewayControlPlaneEnabled(ctx context.Context, settings SettingsReader) error {
	if settings == nil {
		return nil
	}
	current, err := settings.GetSettings(ctx)
	if err != nil {
		return fmt.Errorf("gateway settings unavailable: %w", err)
	}
	if !current.Gateway.ControlPlaneEnabled {
		return errors.New("gateway control plane disabled")
	}
	return nil
}

func resolveGatewayActionAndParams(payload toolArgs) (string, map[string]any) {
	action := canonicalGatewayAction(getStringArg(payload, "action", "method", "type"))
	params := getMapArg(payload, "params")
	if params != nil {
		return action, cloneAnyMap(params)
	}
	if len(payload) == 0 {
		return action, nil
	}
	flattened := make(map[string]any, len(payload))
	for key, value := range payload {
		switch strings.ToLower(strings.TrimSpace(key)) {
		case "action", "method", "type", "params":
			continue
		default:
			flattened[key] = value
		}
	}
	if len(flattened) == 0 {
		return action, nil
	}
	return action, flattened
}

func canonicalGatewayAction(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch normalized {
	case "ping", "status":
		return "gateway.ping"
	default:
		return normalized
	}
}

func runGatewayAction(
	ctx context.Context,
	action string,
	params map[string]any,
	configService GatewayConfigToolService,
) (string, error) {
	definition, ok := gatewayActionsByName[action]
	if !ok {
		return "", errors.New("unsupported gateway action: " + action)
	}
	result, err := definition.execute(ctx, params, configService)
	if err != nil {
		return "", err
	}
	return marshalGatewayToolResponse(definition.name, result), nil
}

func decodeGatewayActionParams(action string, params map[string]any, target any) error {
	if len(params) == 0 {
		return nil
	}
	payload, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("invalid %s params: %w", action, err)
	}
	if err := json.Unmarshal(payload, target); err != nil {
		return fmt.Errorf("invalid %s params: %w", action, err)
	}
	return nil
}

func marshalGatewayToolResponse(action string, result any) string {
	return marshalResult(map[string]any{
		"ok":     true,
		"action": action,
		"result": result,
	})
}
