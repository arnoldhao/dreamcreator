package tools

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"dreamcreator/internal/application/library/dto"
	"dreamcreator/internal/application/library/service"
	tooldto "dreamcreator/internal/application/tools/dto"
	toolservice "dreamcreator/internal/application/tools/service"
)

const libraryAdapterToolName = "library_adapter"

type LibraryAdapter struct {
	library *service.LibraryService
}

func NewLibraryAdapter(library *service.LibraryService) *LibraryAdapter {
	return &LibraryAdapter{library: library}
}

func (adapter *LibraryAdapter) ToolSpec() tooldto.ToolSpec {
	return tooldto.ToolSpec{
		ID:          libraryAdapterToolName,
		Name:        libraryAdapterToolName,
		Description: "library task adapter",
		Kind:        "local",
		Enabled:     true,
	}
}

func (adapter *LibraryAdapter) Execute(ctx context.Context, args string) (string, error) {
	if adapter == nil || adapter.library == nil {
		return "", errors.New("library service unavailable")
	}
	request, err := parseLibraryToolRequest(args)
	if err != nil {
		return "", err
	}
	result, err := adapter.library.HandleToolRequest(ctx, request)
	if err != nil {
		return "", err
	}
	encoded, _ := json.Marshal(result)
	return string(encoded), nil
}

func RegisterLibraryAdapter(ctx context.Context, toolSvc *toolservice.ToolService, executor *RegistryExecutor, library *service.LibraryService) {
	if toolSvc == nil || executor == nil {
		return
	}
	adapter := NewLibraryAdapter(library)
	_, _ = toolSvc.RegisterTool(ctx, tooldto.RegisterToolRequest{Spec: adapter.ToolSpec()})
	executor.Register(libraryAdapterToolName, adapter.Execute)
}

func parseLibraryToolRequest(args string) (dto.LibraryToolRequest, error) {
	if strings.TrimSpace(args) == "" {
		return dto.LibraryToolRequest{}, errors.New("library tool args required")
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(args), &payload); err != nil {
		return dto.LibraryToolRequest{}, err
	}
	action := getString(payload, "action", "type")
	inputValue := payload["input"]
	if inputValue == nil {
		inputValue = payload["payload"]
	}
	inputJSON := ""
	if raw, ok := payload["inputJson"]; ok && inputValue == nil {
		if typed, ok := raw.(string); ok {
			inputJSON = typed
		}
	}
	if inputValue != nil {
		if typed, ok := inputValue.(string); ok {
			inputJSON = typed
		} else if data, err := json.Marshal(inputValue); err == nil {
			inputJSON = string(data)
		}
	}
	sessionKey := getString(payload, "sessionKey", "session_key")
	runID := getString(payload, "runId", "runID")
	if inputJSON != "" {
		var inputMap map[string]any
		if err := json.Unmarshal([]byte(inputJSON), &inputMap); err == nil && inputMap != nil {
			ensureInputField(inputMap, "sessionKey", sessionKey)
			ensureInputField(inputMap, "runId", runID)
			if data, err := json.Marshal(inputMap); err == nil {
				inputJSON = string(data)
			}
		}
	}
	return dto.LibraryToolRequest{Action: action, InputJSON: inputJSON}, nil
}

func ensureInputField(target map[string]any, key string, value string) {
	if target == nil || strings.TrimSpace(value) == "" {
		return
	}
	if raw, ok := target[key]; ok {
		if str, ok := raw.(string); ok && strings.TrimSpace(str) != "" {
			return
		}
	}
	target[key] = strings.TrimSpace(value)
}

func getString(values map[string]any, keys ...string) string {
	if values == nil {
		return ""
	}
	for _, key := range keys {
		if raw, ok := values[key]; ok {
			if str, ok := raw.(string); ok {
				str = strings.TrimSpace(str)
				if str != "" {
					return str
				}
			}
		}
	}
	return ""
}
