package tools

import (
	"context"
	"fmt"
	"strings"

	librarydto "dreamcreator/internal/application/library/dto"
	libraryservice "dreamcreator/internal/application/library/service"
)

type libraryManageService interface {
	PrepareYTDLPDownload(ctx context.Context, request librarydto.PrepareYTDLPDownloadRequest) (librarydto.PrepareYTDLPDownloadResponse, error)
	ParseYTDLPDownload(ctx context.Context, request librarydto.ParseYTDLPDownloadRequest) (librarydto.ParseYTDLPDownloadResponse, error)
	CreateYTDLPJob(ctx context.Context, request librarydto.CreateYTDLPJobRequest) (librarydto.LibraryOperationDTO, error)
	RetryYTDLPOperation(ctx context.Context, request librarydto.RetryYTDLPOperationRequest) (librarydto.LibraryOperationDTO, error)
	CreateTranscodeJob(ctx context.Context, request librarydto.CreateTranscodeJobRequest) (librarydto.LibraryOperationDTO, error)
	CreateSubtitleTranslateJob(ctx context.Context, request librarydto.SubtitleTranslateRequest) (librarydto.LibraryOperationDTO, error)
	CreateSubtitleProofreadJob(ctx context.Context, request librarydto.SubtitleProofreadRequest) (librarydto.LibraryOperationDTO, error)
	CreateSubtitleQAReviewJob(ctx context.Context, request librarydto.SubtitleQAReviewRequest) (librarydto.LibraryOperationDTO, error)
	CancelOperation(ctx context.Context, request librarydto.CancelOperationRequest) (librarydto.LibraryOperationDTO, error)
	ResumeOperation(ctx context.Context, request librarydto.ResumeOperationRequest) (librarydto.LibraryOperationDTO, error)
}

type libraryManageToolResult struct {
	Ok     bool   `json:"ok"`
	Action string `json:"action"`
	Async  bool   `json:"async"`
	Result any    `json:"result,omitempty"`
}

func runLibraryManageTool(library libraryManageService) func(ctx context.Context, args string) (string, error) {
	return func(ctx context.Context, args string) (string, error) {
		if library == nil {
			return "", fmt.Errorf("library service unavailable")
		}
		payload, err := parseToolArgs(args)
		if err != nil {
			return "", err
		}
		action, params := resolveLibraryManageActionAndParams(payload)
		if action == "" {
			return "", fmt.Errorf("library_manage action is required")
		}
		return runLibraryManageAction(ctx, library, action, params)
	}
}

func resolveLibraryManageActionAndParams(payload toolArgs) (string, map[string]any) {
	action := canonicalLibraryManageAction(getStringArg(payload, "action", "type"))
	params := getMapArg(payload, "params")
	if params != nil {
		return action, cloneAnyMap(params)
	}
	flattened := make(map[string]any, len(payload))
	for key, value := range payload {
		switch strings.ToLower(strings.TrimSpace(key)) {
		case "action", "type", "params":
			continue
		default:
			flattened[key] = value
		}
	}
	if len(flattened) == 0 {
		return action, nil
	}
	return action, flattened
}

func canonicalLibraryManageAction(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch normalized {
	case "download":
		return "download.create"
	case "transcode":
		return "transcode.create"
	case "translate":
		return "subtitle.translate.create"
	case "proofread":
		return "subtitle.proofread.create"
	case "qa", "review", "qa_review":
		return "subtitle.qa_review.create"
	case "cancel":
		return "operation.cancel"
	case "resume":
		return "operation.resume"
	default:
		return normalized
	}
}

func runLibraryManageAction(ctx context.Context, library libraryManageService, action string, params map[string]any) (string, error) {
	sessionKey, runID := RuntimeContextFromContext(ctx)
	enriched := cloneAnyMap(params)
	if enriched == nil {
		enriched = map[string]any{}
	}
	ensureLibraryManageRuntimeField(enriched, "sessionKey", sessionKey)
	ensureLibraryManageRuntimeField(enriched, "runId", runID)

	switch action {
	case "download.prepare":
		request := librarydto.PrepareYTDLPDownloadRequest{}
		if err := decodeGatewayActionParams(action, enriched, &request); err != nil {
			return "", err
		}
		result, err := library.PrepareYTDLPDownload(ctx, request)
		if err != nil {
			return "", err
		}
		return marshalLibraryManageResult(action, false, result), nil
	case "download.parse":
		request := librarydto.ParseYTDLPDownloadRequest{}
		if err := decodeGatewayActionParams(action, enriched, &request); err != nil {
			return "", err
		}
		result, err := library.ParseYTDLPDownload(ctx, request)
		if err != nil {
			return "", err
		}
		return marshalLibraryManageResult(action, false, result), nil
	case "download.create":
		request := librarydto.CreateYTDLPJobRequest{}
		if err := decodeGatewayActionParams(action, enriched, &request); err != nil {
			return "", err
		}
		operation, err := library.CreateYTDLPJob(ctx, request)
		if err != nil {
			return "", err
		}
		return marshalLibraryManageResult(action, true, buildLibraryManageAsyncAccepted(operation)), nil
	case "download.retry":
		request := librarydto.RetryYTDLPOperationRequest{}
		if err := decodeGatewayActionParams(action, enriched, &request); err != nil {
			return "", err
		}
		operation, err := library.RetryYTDLPOperation(ctx, request)
		if err != nil {
			return "", err
		}
		return marshalLibraryManageResult(action, true, buildLibraryManageAsyncAccepted(operation)), nil
	case "transcode.create":
		request := librarydto.CreateTranscodeJobRequest{}
		if err := decodeGatewayActionParams(action, enriched, &request); err != nil {
			return "", err
		}
		operation, err := library.CreateTranscodeJob(ctx, request)
		if err != nil {
			return "", err
		}
		return marshalLibraryManageResult(action, true, buildLibraryManageAsyncAccepted(operation)), nil
	case "subtitle.translate.create":
		request := librarydto.SubtitleTranslateRequest{}
		if err := decodeGatewayActionParams(action, enriched, &request); err != nil {
			return "", err
		}
		operation, err := library.CreateSubtitleTranslateJob(ctx, request)
		if err != nil {
			return "", err
		}
		return marshalLibraryManageResult(action, true, buildLibraryManageAsyncAccepted(operation)), nil
	case "subtitle.proofread.create":
		request := librarydto.SubtitleProofreadRequest{}
		if err := decodeGatewayActionParams(action, enriched, &request); err != nil {
			return "", err
		}
		operation, err := library.CreateSubtitleProofreadJob(ctx, request)
		if err != nil {
			return "", err
		}
		return marshalLibraryManageResult(action, true, buildLibraryManageAsyncAccepted(operation)), nil
	case "subtitle.qa_review.create":
		request := librarydto.SubtitleQAReviewRequest{}
		if err := decodeGatewayActionParams(action, enriched, &request); err != nil {
			return "", err
		}
		operation, err := library.CreateSubtitleQAReviewJob(ctx, request)
		if err != nil {
			return "", err
		}
		return marshalLibraryManageResult(action, true, buildLibraryManageAsyncAccepted(operation)), nil
	case "operation.cancel":
		request := librarydto.CancelOperationRequest{}
		if err := decodeGatewayActionParams(action, enriched, &request); err != nil {
			return "", err
		}
		operation, err := library.CancelOperation(ctx, request)
		if err != nil {
			return "", err
		}
		return marshalLibraryManageResult(action, false, operation), nil
	case "operation.resume":
		request := librarydto.ResumeOperationRequest{}
		if err := decodeGatewayActionParams(action, enriched, &request); err != nil {
			return "", err
		}
		operation, err := library.ResumeOperation(ctx, request)
		if err != nil {
			return "", err
		}
		return marshalLibraryManageResult(action, true, buildLibraryManageAsyncAccepted(operation)), nil
	default:
		return "", fmt.Errorf("unsupported library_manage action: %s", action)
	}
}

func ensureLibraryManageRuntimeField(target map[string]any, key string, value string) {
	if target == nil || strings.TrimSpace(value) == "" {
		return
	}
	if current := strings.TrimSpace(getStringArg(target, key)); current != "" {
		return
	}
	target[key] = strings.TrimSpace(value)
}

func buildLibraryManageAsyncAccepted(operation librarydto.LibraryOperationDTO) libraryManageAsyncAcceptedResult {
	return libraryManageAsyncAcceptedResult{
		OperationID: strings.TrimSpace(operation.ID),
		LibraryID:   strings.TrimSpace(operation.LibraryID),
		Kind:        strings.TrimSpace(operation.Kind),
		Status:      strings.TrimSpace(operation.Status),
		Message:     "Background job accepted. Do not wait or retry in this tool call; query progress later by operationId.",
		FollowUp: libraryManageFollowUp{
			Tool:        "library",
			Action:      "operation_status",
			OperationID: strings.TrimSpace(operation.ID),
			Guidance:    "Call library with action=operation_status to inspect progress, or action=operations to review the queue.",
		},
		Operation: operation,
	}
}

func marshalLibraryManageResult(action string, async bool, result any) string {
	return marshalResult(libraryManageToolResult{
		Ok:     true,
		Action: action,
		Async:  async,
		Result: result,
	})
}

var _ libraryManageService = (*libraryservice.LibraryService)(nil)
