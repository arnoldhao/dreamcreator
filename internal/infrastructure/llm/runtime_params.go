package llm

import (
	"context"
	"encoding/json"
	"strings"
)

type StructuredOutputConfig struct {
	Mode   string
	Name   string
	Schema map[string]any
	Strict bool
}

type RuntimeParams struct {
	ProviderID       string
	ModelName        string
	ThinkingLevel    string
	StructuredOutput StructuredOutputConfig
}

type runtimeParamsContextKey struct{}

func WithRuntimeParams(ctx context.Context, params RuntimeParams) context.Context {
	normalized := RuntimeParams{
		ProviderID:       strings.TrimSpace(params.ProviderID),
		ModelName:        strings.TrimSpace(params.ModelName),
		ThinkingLevel:    strings.TrimSpace(params.ThinkingLevel),
		StructuredOutput: normalizeStructuredOutputConfig(params.StructuredOutput),
	}
	return context.WithValue(ctx, runtimeParamsContextKey{}, normalized)
}

func runtimeParamsFromContext(ctx context.Context) RuntimeParams {
	if ctx == nil {
		return RuntimeParams{}
	}
	value, ok := ctx.Value(runtimeParamsContextKey{}).(RuntimeParams)
	if !ok {
		return RuntimeParams{}
	}
	return value
}

func RuntimeParamsFromContext(ctx context.Context) RuntimeParams {
	return runtimeParamsFromContext(ctx)
}

func normalizeStructuredOutputConfig(config StructuredOutputConfig) StructuredOutputConfig {
	mode := normalizeStructuredOutputMode(config.Mode)
	name := strings.TrimSpace(config.Name)
	schema := cloneStructuredOutputSchema(config.Schema)
	if mode == "" {
		if name == "" && len(schema) == 0 {
			return StructuredOutputConfig{}
		}
		mode = "json_schema"
	}
	if mode == "prompt_only" {
		return StructuredOutputConfig{Mode: mode}
	}
	if name == "" || len(schema) == 0 {
		return StructuredOutputConfig{}
	}
	return StructuredOutputConfig{
		Mode:   mode,
		Name:   name,
		Schema: schema,
		Strict: config.Strict,
	}
}

func normalizeStructuredOutputMode(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "auto", "":
		return "auto"
	case "json_schema", "json-schema", "jsonschema", "schema":
		return "json_schema"
	case "prompt_only", "prompt-only", "promptonly", "prompt":
		return "prompt_only"
	case "off", "none", "disabled", "disable":
		return "prompt_only"
	default:
		return ""
	}
}

func cloneStructuredOutputSchema(schema map[string]any) map[string]any {
	if len(schema) == 0 {
		return nil
	}
	encoded, err := json.Marshal(schema)
	if err != nil {
		return nil
	}
	var cloned map[string]any
	if err := json.Unmarshal(encoded, &cloned); err != nil {
		return nil
	}
	return cloned
}

func (config StructuredOutputConfig) UsesJSONSchema() bool {
	switch normalizeStructuredOutputMode(config.Mode) {
	case "auto", "json_schema":
		return strings.TrimSpace(config.Name) != "" && len(config.Schema) > 0
	default:
		return false
	}
}

func (config StructuredOutputConfig) AllowsFallback() bool {
	return normalizeStructuredOutputMode(config.Mode) == "auto"
}
