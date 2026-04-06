package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"dreamcreator/internal/application/library/dto"
)

func (service *LibraryService) HandleToolRequest(ctx context.Context, request dto.LibraryToolRequest) (dto.LibraryToolResult, error) {
	action := strings.ToLower(strings.TrimSpace(request.Action))
	if action == "" {
		return dto.LibraryToolResult{Status: "error", Error: "action is required"}, fmt.Errorf("action is required")
	}
	payload := map[string]any{}
	if strings.TrimSpace(request.InputJSON) != "" {
		if err := json.Unmarshal([]byte(request.InputJSON), &payload); err != nil {
			return dto.LibraryToolResult{Status: "error", Error: err.Error()}, err
		}
	}
	var output interface{}
	var err error
	switch action {
	case "library.overview", "overview":
		output, err = service.ListLibraries(ctx)
	case "library.operations", "operations":
		output, err = service.ListOperations(ctx, dto.ListOperationsRequest{
			LibraryID: getString(payload, "libraryId", "libraryID"),
			Status:    getStringSlice(payload, "status", "statuses"),
			Kinds:     getStringSlice(payload, "kinds", "types"),
			Query:     getString(payload, "query"),
			Limit:     getInt(payload, "limit"),
			Offset:    getInt(payload, "offset"),
		})
	case "library.records", "records":
		output, err = service.ListLibraryHistory(ctx, dto.ListLibraryHistoryRequest{
			LibraryID:  getString(payload, "libraryId", "libraryID"),
			Categories: getStringSlice(payload, "categories"),
			Actions:    getStringSlice(payload, "actions"),
			Limit:      getInt(payload, "limit"),
			Offset:     getInt(payload, "offset"),
		})
	case "library.files", "files":
		libraryID := getString(payload, "libraryId", "libraryID")
		output, err = service.GetLibrary(ctx, dto.GetLibraryRequest{LibraryID: libraryID})
	case "library.operation_status", "operation_status", "status":
		operation, getErr := service.GetOperation(ctx, dto.GetOperationRequest{OperationID: getString(payload, "operationId", "operationID", "id")})
		output = operation
		err = getErr
	default:
		err = fmt.Errorf("unsupported library action: %s", action)
	}
	if err != nil {
		return dto.LibraryToolResult{Status: "error", Error: err.Error()}, err
	}
	encoded, marshalErr := json.Marshal(output)
	if marshalErr != nil {
		return dto.LibraryToolResult{Status: "error", Error: marshalErr.Error()}, marshalErr
	}
	result := dto.LibraryToolResult{Status: "ok", OutputJSON: string(encoded)}
	if operation, ok := output.(dto.LibraryOperationDTO); ok {
		result.OperationID = operation.ID
	}
	return result, nil
}
