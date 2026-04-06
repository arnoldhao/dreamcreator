package tools

import (
	"encoding/json"
	"reflect"

	librarydto "dreamcreator/internal/application/library/dto"
	tooldto "dreamcreator/internal/application/tools/dto"
)

type libraryMethodDefinition struct {
	name       string
	inputType  reflect.Type
	outputType reflect.Type
}

var libraryMethodDefinitions = []libraryMethodDefinition{
	{
		name:       "overview",
		inputType:  reflect.TypeOf(struct{}{}),
		outputType: reflect.TypeOf([]librarydto.LibraryDTO{}),
	},
	{
		name:       "files",
		inputType:  reflect.TypeOf(librarydto.GetLibraryRequest{}),
		outputType: reflect.TypeOf(librarydto.LibraryDTO{}),
	},
	{
		name:       "operations",
		inputType:  reflect.TypeOf(librarydto.ListOperationsRequest{}),
		outputType: reflect.TypeOf([]librarydto.OperationListItemDTO{}),
	},
	{
		name:       "records",
		inputType:  reflect.TypeOf(librarydto.ListLibraryHistoryRequest{}),
		outputType: reflect.TypeOf([]librarydto.LibraryHistoryRecordDTO{}),
	},
	{
		name:       "operation_status",
		inputType:  reflect.TypeOf(librarydto.GetOperationRequest{}),
		outputType: reflect.TypeOf(librarydto.LibraryOperationDTO{}),
	},
}

func libraryMethodSpecs() []tooldto.ToolMethodSpec {
	specs := make([]tooldto.ToolMethodSpec, 0, len(libraryMethodDefinitions))
	for _, definition := range libraryMethodDefinitions {
		specs = append(specs, buildLibraryMethodSpec(definition))
	}
	return specs
}

func buildLibraryMethodSpec(definition libraryMethodDefinition) tooldto.ToolMethodSpec {
	return tooldto.ToolMethodSpec{
		Name:          definition.name,
		InputSchema:   buildGatewayMethodInputSchema(definition.name, definition.inputType),
		OutputSchema:  buildLibraryMethodOutputSchema(),
		InputExample:  buildGatewayMethodInputExample(definition.name, definition.inputType),
		OutputExample: buildLibraryMethodOutputExample(definition),
	}
}

func buildLibraryMethodOutputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"status": map[string]any{
				"type":  "string",
				"const": "ok",
			},
			"operationId": map[string]any{
				"type": "string",
			},
			"outputJson": map[string]any{
				"type":        "string",
				"description": "JSON-encoded result payload.",
			},
		},
		"required": []string{"status", "outputJson"},
	}
}

func buildLibraryMethodOutputExample(definition libraryMethodDefinition) map[string]any {
	result := map[string]any{
		"status":     "ok",
		"outputJson": marshalLibraryMethodOutput(definition.outputType),
	}
	if definition.outputType == reflect.TypeOf(librarydto.LibraryOperationDTO{}) {
		result["operationId"] = ""
	}
	return result
}

func marshalLibraryMethodOutput(outputType reflect.Type) string {
	value := buildToolTypeEmptyValue(outputType)
	if value == nil {
		return "null"
	}
	data, err := json.Marshal(value)
	if err != nil {
		return "null"
	}
	return string(data)
}

func libraryToolSchema() string {
	actions := make([]string, 0, len(libraryMethodDefinitions))
	properties := map[string]any{
		"action": map[string]any{
			"type": "string",
		},
		"type": map[string]any{
			"type": "string",
		},
		"params": map[string]any{
			"type": "object",
		},
	}
	for _, definition := range libraryMethodDefinitions {
		actions = append(actions, definition.name)
		paramsSchema, _ := buildToolTypeSchema(definition.inputType).(map[string]any)
		paramProperties, _ := paramsSchema["properties"].(map[string]any)
		for key, value := range paramProperties {
			properties[key] = value
		}
	}
	return schemaJSON(map[string]any{
		"type":       "object",
		"properties": mergeLibraryActionEnum(properties, actions),
		"required":   []string{"action"},
	})
}

func mergeLibraryActionEnum(properties map[string]any, actions []string) map[string]any {
	cloned := make(map[string]any, len(properties))
	for key, value := range properties {
		cloned[key] = value
	}
	cloned["action"] = map[string]any{
		"type": "string",
		"enum": append([]string(nil), actions...),
	}
	cloned["type"] = map[string]any{
		"type": "string",
		"enum": append([]string(nil), actions...),
	}
	return cloned
}
