package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	librarydto "dreamcreator/internal/application/library/dto"
	libraryservice "dreamcreator/internal/application/library/service"
)

func runLibraryActionTool(library *libraryservice.LibraryService, action string) func(ctx context.Context, args string) (string, error) {
	return func(ctx context.Context, args string) (string, error) {
		if library == nil {
			return "", errors.New("library service unavailable")
		}
		inputJSON := strings.TrimSpace(args)
		if inputJSON == "" {
			return "", errors.New("library tool args required")
		}
		request := librarydto.LibraryToolRequest{
			Action:    action,
			InputJSON: inputJSON,
		}
		result, err := library.HandleToolRequest(ctx, request)
		if err != nil {
			return "", err
		}
		return marshalResult(result), nil
	}
}

func runLibraryGroupTool(library *libraryservice.LibraryService, group string) func(ctx context.Context, args string) (string, error) {
	return func(ctx context.Context, args string) (string, error) {
		if library == nil {
			return "", errors.New("library service unavailable")
		}
		payload := map[string]any{}
		trimmedArgs := strings.TrimSpace(args)
		if trimmedArgs != "" {
			if err := json.Unmarshal([]byte(trimmedArgs), &payload); err != nil {
				return "", err
			}
		}
		action := strings.ToLower(strings.TrimSpace(getString(payload, "action", "type")))
		if action == "" {
			action = resolveLibraryDefaultAction(group, payload)
		}
		if action == "" {
			return "", fmt.Errorf("tool action is required for %s", group)
		}
		fullAction := action
		prefix := strings.ToLower(strings.TrimSpace(group))
		if prefix != "" && !strings.HasPrefix(fullAction, prefix+".") {
			fullAction = prefix + "." + fullAction
		}

		input := map[string]any{}
		if raw, ok := payload["params"]; ok {
			if typed, ok := raw.(map[string]any); ok {
				for key, value := range typed {
					input[key] = value
				}
			}
		}
		for key, value := range payload {
			normalized := strings.ToLower(strings.TrimSpace(key))
			if normalized == "action" || normalized == "type" || normalized == "params" {
				continue
			}
			input[key] = value
		}
		inputJSON := "{}"
		if len(input) > 0 {
			encoded, err := json.Marshal(input)
			if err != nil {
				return "", err
			}
			inputJSON = string(encoded)
		}
		result, err := library.HandleToolRequest(ctx, librarydto.LibraryToolRequest{
			Action:    fullAction,
			InputJSON: inputJSON,
		})
		if err != nil {
			return "", err
		}
		return marshalResult(result), nil
	}
}

func resolveLibraryDefaultAction(group string, payload map[string]any) string {
	switch strings.ToLower(strings.TrimSpace(group)) {
	case "library":
		return "overview"
	default:
		return ""
	}
}
