package agentruntime

import (
	"errors"
	"testing"

	"github.com/cloudwego/eino/schema"
)

func TestJSONToolValidatorRejectsInvalidInput(t *testing.T) {
	validator := JSONToolValidator{}

	err := validator.Validate(schema.ToolCall{
		Function: schema.FunctionCall{
			Name:      "read",
			Arguments: "{invalid",
		},
	})
	if err == nil || !errors.Is(err, ErrToolArgsInvalid) {
		t.Fatalf("expected invalid args error, got %v", err)
	}
}

func TestJSONToolValidatorAcceptsObjectArgs(t *testing.T) {
	validator := JSONToolValidator{}
	err := validator.Validate(schema.ToolCall{
		Function: schema.FunctionCall{
			Name:      "read",
			Arguments: `{"path":"README.md"}`,
		},
	})
	if err != nil {
		t.Fatalf("expected valid args, got %v", err)
	}
}

func TestJSONToolValidatorRejectsSchemaViolation(t *testing.T) {
	validator := JSONToolValidator{
		Tools: map[string]ToolDefinition{
			"browser": {
				Name: "browser",
				SchemaJSON: `{
					"type":"object",
					"properties":{
						"action":{"type":"string","enum":["open","act"]},
						"url":{"type":"string"},
						"request":{
							"type":"object",
							"properties":{"kind":{"type":"string"}},
							"required":["kind"]
						}
					},
					"required":["action"],
					"allOf":[
						{
							"anyOf":[
								{"properties":{"action":{"const":"open"}},"required":["action","url"]},
								{"properties":{"action":{"const":"act"}},"required":["action","request"]}
							]
						}
					]
				}`,
			},
		},
	}

	err := validator.Validate(schema.ToolCall{
		Function: schema.FunctionCall{
			Name:      "browser",
			Arguments: `{"action":"open"}`,
		},
	})
	if err == nil || !errors.Is(err, ErrToolArgsSchema) {
		t.Fatalf("expected schema error for missing url, got %v", err)
	}
}

func TestJSONToolValidatorRejectsNestedRequiredSchemaViolation(t *testing.T) {
	validator := JSONToolValidator{
		Tools: map[string]ToolDefinition{
			"browser": {
				Name: "browser",
				SchemaJSON: `{
					"type":"object",
					"properties":{
						"action":{"type":"string","enum":["act"]},
						"request":{
							"type":"object",
							"properties":{"kind":{"type":"string"}},
							"required":["kind"]
						}
					},
					"required":["action","request"]
				}`,
			},
		},
	}

	err := validator.Validate(schema.ToolCall{
		Function: schema.FunctionCall{
			Name:      "browser",
			Arguments: `{"action":"act","request":{}}`,
		},
	})
	if err == nil || !errors.Is(err, ErrToolArgsSchema) {
		t.Fatalf("expected schema error for missing request.kind, got %v", err)
	}
}
