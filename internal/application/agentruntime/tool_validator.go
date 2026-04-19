package agentruntime

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/cloudwego/eino/schema"
)

var (
	ErrToolNameRequired = errors.New("tool name is required")
	ErrToolArgsInvalid  = errors.New("tool args must be a json object")
	ErrToolArgsSchema   = errors.New("tool args do not match schema")
)

type ToolValidator interface {
	Validate(call schema.ToolCall) error
}

type JSONToolValidator struct {
	Tools map[string]ToolDefinition
}

func (validator JSONToolValidator) Validate(call schema.ToolCall) error {
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
	if definition, ok := validator.Tools[name]; ok {
		if err := validateToolArgsAgainstSchema(decoded, strings.TrimSpace(definition.SchemaJSON)); err != nil {
			return err
		}
	}
	return nil
}

func validateToolArgsAgainstSchema(value any, schemaJSON string) error {
	if strings.TrimSpace(schemaJSON) == "" {
		return nil
	}
	var schema map[string]any
	if err := json.Unmarshal([]byte(schemaJSON), &schema); err != nil {
		return nil
	}
	return validateJSONSchemaValue(value, schema, "$")
}

func validateJSONSchemaValue(value any, schema map[string]any, path string) error {
	if len(schema) == 0 {
		return nil
	}
	if allOf, ok := schema["allOf"].([]any); ok {
		for _, item := range allOf {
			child, _ := item.(map[string]any)
			if err := validateJSONSchemaValue(value, child, path); err != nil {
				return err
			}
		}
	}
	if anyOf, ok := schema["anyOf"].([]any); ok {
		matched := false
		var lastErr error
		for _, item := range anyOf {
			child, _ := item.(map[string]any)
			err := validateJSONSchemaValue(value, child, path)
			if err == nil {
				matched = true
				break
			}
			lastErr = err
		}
		if !matched {
			if lastErr != nil {
				return lastErr
			}
			return fmt.Errorf("%w: %s must match at least one schema", ErrToolArgsSchema, path)
		}
	}
	if oneOf, ok := schema["oneOf"].([]any); ok {
		matches := 0
		var lastErr error
		for _, item := range oneOf {
			child, _ := item.(map[string]any)
			err := validateJSONSchemaValue(value, child, path)
			if err == nil {
				matches++
				continue
			}
			lastErr = err
		}
		if matches != 1 {
			if lastErr != nil && matches == 0 {
				return lastErr
			}
			return fmt.Errorf("%w: %s must match exactly one schema", ErrToolArgsSchema, path)
		}
	}
	if constValue, ok := schema["const"]; ok && !jsonSchemaValuesEqual(value, constValue) {
		return fmt.Errorf("%w: %s must equal %v", ErrToolArgsSchema, path, constValue)
	}
	if enumValues, ok := schema["enum"].([]any); ok && len(enumValues) > 0 {
		matched := false
		for _, enumValue := range enumValues {
			if jsonSchemaValuesEqual(value, enumValue) {
				matched = true
				break
			}
		}
		if !matched {
			return fmt.Errorf("%w: %s must be one of %v", ErrToolArgsSchema, path, enumValues)
		}
	}
	switch schemaType := schema["type"].(type) {
	case string:
		if err := validateJSONSchemaType(value, schema, path, schemaType); err != nil {
			return err
		}
	case []any:
		var lastErr error
		for _, raw := range schemaType {
			typeName, _ := raw.(string)
			if typeName == "" {
				continue
			}
			if err := validateJSONSchemaType(value, schema, path, typeName); err == nil {
				return nil
			} else {
				lastErr = err
			}
		}
		if lastErr != nil {
			return lastErr
		}
	}
	if _, hasProperties := schema["properties"]; hasProperties || schema["required"] != nil || schema["additionalProperties"] != nil {
		return validateJSONSchemaType(value, schema, path, "object")
	}
	if _, hasItems := schema["items"]; hasItems {
		return validateJSONSchemaType(value, schema, path, "array")
	}
	return nil
}

func validateJSONSchemaType(value any, schema map[string]any, path string, schemaType string) error {
	switch schemaType {
	case "object":
		obj, ok := value.(map[string]any)
		if !ok {
			return fmt.Errorf("%w: %s must be an object", ErrToolArgsSchema, path)
		}
		properties, _ := schema["properties"].(map[string]any)
		requiredFields := jsonSchemaStringSlice(schema["required"])
		for _, field := range requiredFields {
			if _, exists := obj[field]; !exists {
				return fmt.Errorf("%w: %s.%s is required", ErrToolArgsSchema, path, field)
			}
		}
		for key, raw := range obj {
			childPath := path + "." + key
			propertySchema, hasProperty := properties[key]
			if hasProperty {
				if child, ok := propertySchema.(map[string]any); ok {
					if err := validateJSONSchemaValue(raw, child, childPath); err != nil {
						return err
					}
				}
				continue
			}
			switch additional := schema["additionalProperties"].(type) {
			case bool:
				if !additional {
					return fmt.Errorf("%w: %s is not allowed", ErrToolArgsSchema, childPath)
				}
			case map[string]any:
				if err := validateJSONSchemaValue(raw, additional, childPath); err != nil {
					return err
				}
			}
		}
		return nil
	case "array":
		items, ok := value.([]any)
		if !ok {
			return fmt.Errorf("%w: %s must be an array", ErrToolArgsSchema, path)
		}
		if itemSchema, ok := schema["items"].(map[string]any); ok {
			for index, item := range items {
				if err := validateJSONSchemaValue(item, itemSchema, fmt.Sprintf("%s[%d]", path, index)); err != nil {
					return err
				}
			}
		}
		return nil
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("%w: %s must be a string", ErrToolArgsSchema, path)
		}
		return nil
	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("%w: %s must be a boolean", ErrToolArgsSchema, path)
		}
		return nil
	case "integer":
		number, ok := jsonSchemaAsFloat64(value)
		if !ok || float64(int64(number)) != number {
			return fmt.Errorf("%w: %s must be an integer", ErrToolArgsSchema, path)
		}
		if minimum, ok := jsonSchemaAsFloat64(schema["minimum"]); ok && number < minimum {
			return fmt.Errorf("%w: %s must be >= %v", ErrToolArgsSchema, path, minimum)
		}
		if maximum, ok := jsonSchemaAsFloat64(schema["maximum"]); ok && number > maximum {
			return fmt.Errorf("%w: %s must be <= %v", ErrToolArgsSchema, path, maximum)
		}
		return nil
	case "number":
		number, ok := jsonSchemaAsFloat64(value)
		if !ok {
			return fmt.Errorf("%w: %s must be a number", ErrToolArgsSchema, path)
		}
		if minimum, ok := jsonSchemaAsFloat64(schema["minimum"]); ok && number < minimum {
			return fmt.Errorf("%w: %s must be >= %v", ErrToolArgsSchema, path, minimum)
		}
		if maximum, ok := jsonSchemaAsFloat64(schema["maximum"]); ok && number > maximum {
			return fmt.Errorf("%w: %s must be <= %v", ErrToolArgsSchema, path, maximum)
		}
		return nil
	default:
		return nil
	}
}

func jsonSchemaStringSlice(value any) []string {
	switch typed := value.(type) {
	case []string:
		return append([]string(nil), typed...)
	case []any:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			if text, ok := item.(string); ok && strings.TrimSpace(text) != "" {
				result = append(result, text)
			}
		}
		return result
	default:
		return nil
	}
}

func jsonSchemaAsFloat64(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case int32:
		return float64(typed), true
	default:
		return 0, false
	}
}

func jsonSchemaValuesEqual(left any, right any) bool {
	leftNumber, leftIsNumber := jsonSchemaAsFloat64(left)
	rightNumber, rightIsNumber := jsonSchemaAsFloat64(right)
	if leftIsNumber && rightIsNumber {
		return leftNumber == rightNumber
	}
	return reflect.DeepEqual(left, right)
}
