package tools

import (
	"reflect"
	"strings"
	"time"

	tooldto "dreamcreator/internal/application/tools/dto"
)

func gatewayMethodSpecs() []tooldto.ToolMethodSpec {
	specs := make([]tooldto.ToolMethodSpec, 0, len(gatewayActionDefinitions))
	for _, definition := range gatewayActionDefinitions {
		specs = append(specs, buildGatewayMethodSpec(definition))
	}
	return specs
}

func buildGatewayMethodSpec(definition gatewayActionDefinition) tooldto.ToolMethodSpec {
	action := definition.name
	outputSchema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"ok": map[string]any{
				"type": "boolean",
			},
			"action": map[string]any{
				"type":  "string",
				"const": action,
			},
			"result": buildToolTypeSchema(definition.outputType),
		},
		"required": []string{"ok", "action", "result"},
	}

	outputExample := map[string]any{
		"ok":     true,
		"action": action,
		"result": buildToolTypeEmptyValue(definition.outputType),
	}

	return tooldto.ToolMethodSpec{
		Name:          action,
		InputSchema:   buildGatewayMethodInputSchema(action, definition.inputType),
		OutputSchema:  outputSchema,
		InputExample:  buildGatewayMethodInputExample(action, definition.inputType),
		OutputExample: outputExample,
	}
}

func buildGatewayMethodInputSchema(action string, inputType reflect.Type) any {
	paramsSchema := buildToolTypeSchema(inputType)
	paramsStyle := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type":  "string",
				"const": action,
			},
			"params": paramsSchema,
		},
		"required": []string{"action"},
	}

	flatStyle := buildGatewayMethodFlatInputSchema(action, paramsSchema)
	if flatStyle == nil {
		return paramsStyle
	}

	return map[string]any{
		"oneOf": []any{paramsStyle, flatStyle},
	}
}

func buildGatewayMethodFlatInputSchema(action string, paramsSchema any) map[string]any {
	paramsObject, ok := paramsSchema.(map[string]any)
	if !ok {
		return nil
	}
	typ, _ := paramsObject["type"].(string)
	if typ != "object" {
		return nil
	}

	flatProperties := map[string]any{
		"action": map[string]any{
			"type":  "string",
			"const": action,
		},
	}

	paramProperties, _ := paramsObject["properties"].(map[string]any)
	for key, value := range paramProperties {
		flatProperties[key] = value
	}

	required := []string{"action"}
	required = append(required, readSchemaRequired(paramsObject["required"])...)

	flat := map[string]any{
		"type":       "object",
		"properties": flatProperties,
	}
	if len(required) > 0 {
		flat["required"] = required
	}
	return flat
}

func readSchemaRequired(value any) []string {
	if value == nil {
		return nil
	}
	if typed, ok := value.([]string); ok {
		return append([]string(nil), typed...)
	}
	raw, ok := value.([]any)
	if !ok {
		return nil
	}
	items := make([]string, 0, len(raw))
	for _, candidate := range raw {
		name, ok := candidate.(string)
		if !ok {
			continue
		}
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		items = append(items, name)
	}
	return items
}

func buildGatewayMethodInputExample(action string, inputType reflect.Type) map[string]any {
	example := map[string]any{
		"action": action,
	}
	paramsExample := buildToolTypeEmptyValue(inputType)
	paramsObject, ok := paramsExample.(map[string]any)
	if !ok || len(paramsObject) == 0 {
		return example
	}
	for key, value := range paramsObject {
		example[key] = value
	}
	return example
}

func buildToolTypeSchema(t reflect.Type) any {
	if t == nil {
		return map[string]any{}
	}
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.PkgPath() == "time" && t.Name() == "Time" {
		return map[string]any{
			"type":   "string",
			"format": "date-time",
		}
	}
	switch t.Kind() {
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	case reflect.String:
		return map[string]any{"type": "string"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return map[string]any{"type": "integer"}
	case reflect.Float32, reflect.Float64:
		return map[string]any{"type": "number"}
	case reflect.Slice, reflect.Array:
		return map[string]any{
			"type":  "array",
			"items": buildToolTypeSchema(t.Elem()),
		}
	case reflect.Map:
		return map[string]any{
			"type":                 "object",
			"additionalProperties": buildToolTypeSchema(t.Elem()),
		}
	case reflect.Struct:
		properties := map[string]any{}
		required := make([]string, 0, t.NumField())
		for index := 0; index < t.NumField(); index++ {
			field := t.Field(index)
			if field.PkgPath != "" {
				continue
			}
			name, omitempty, ignore := resolveJSONFieldInfo(field)
			if ignore {
				continue
			}
			properties[name] = buildToolTypeSchema(field.Type)
			if !omitempty {
				required = append(required, name)
			}
		}
		schema := map[string]any{
			"type":       "object",
			"properties": properties,
		}
		if len(required) > 0 {
			schema["required"] = required
		}
		return schema
	case reflect.Interface:
		return map[string]any{}
	default:
		return map[string]any{}
	}
}

func buildToolTypeEmptyValue(t reflect.Type) any {
	if t == nil {
		return map[string]any{}
	}
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.PkgPath() == "time" && t.Name() == "Time" {
		return time.Time{}.Format(time.RFC3339)
	}
	switch t.Kind() {
	case reflect.Bool:
		return false
	case reflect.String:
		return ""
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return 0
	case reflect.Float32, reflect.Float64:
		return 0
	case reflect.Slice, reflect.Array:
		return []any{}
	case reflect.Map:
		return map[string]any{}
	case reflect.Struct:
		value := map[string]any{}
		for index := 0; index < t.NumField(); index++ {
			field := t.Field(index)
			if field.PkgPath != "" {
				continue
			}
			name, _, ignore := resolveJSONFieldInfo(field)
			if ignore {
				continue
			}
			value[name] = buildToolTypeEmptyValue(field.Type)
		}
		return value
	case reflect.Interface:
		return nil
	default:
		return nil
	}
}

func resolveJSONFieldInfo(field reflect.StructField) (name string, omitempty bool, ignore bool) {
	tag := strings.TrimSpace(field.Tag.Get("json"))
	if tag == "-" {
		return "", false, true
	}
	if tag == "" {
		return field.Name, false, false
	}
	parts := strings.Split(tag, ",")
	name = strings.TrimSpace(parts[0])
	if name == "" {
		name = field.Name
	}
	for _, part := range parts[1:] {
		if strings.TrimSpace(part) == "omitempty" {
			omitempty = true
			break
		}
	}
	return name, omitempty, false
}
