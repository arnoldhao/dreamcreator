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
