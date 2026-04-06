package agentruntime

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/schema"
)

var (
	ErrToolNameRequired = errors.New("tool name is required")
	ErrToolArgsInvalid  = errors.New("tool args must be a json object")
)

type ToolValidator interface {
	Validate(call schema.ToolCall) error
}

type JSONToolValidator struct{}

func (JSONToolValidator) Validate(call schema.ToolCall) error {
	name := strings.TrimSpace(call.Function.Name)
	if name == "" {
		return ErrToolNameRequired
	}
	args := strings.TrimSpace(call.Function.Arguments)
	if args == "" {
		args = "{}"
	}
	var decoded any
	if err := json.Unmarshal([]byte(args), &decoded); err != nil {
		return fmt.Errorf("%w: %v", ErrToolArgsInvalid, err)
	}
	if _, ok := decoded.(map[string]any); !ok {
		return ErrToolArgsInvalid
	}
	return nil
}
