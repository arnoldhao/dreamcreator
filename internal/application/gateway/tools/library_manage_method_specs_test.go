package tools

import (
	"context"
	"encoding/json"
	"testing"

	librarydto "dreamcreator/internal/application/library/dto"
)

func TestLibraryManageMethodSpecsMatchDefinitions(t *testing.T) {
	specs := libraryManageMethodSpecs()
	if len(specs) != len(libraryManageMethodDefinitions) {
		t.Fatalf("expected %d method specs, got %d", len(libraryManageMethodDefinitions), len(specs))
	}

	byName := make(map[string]struct{}, len(specs))
	for _, spec := range specs {
		byName[spec.Name] = struct{}{}

		inputExample, ok := spec.InputExample.(map[string]any)
		if !ok {
			t.Fatalf("expected input example object for %s", spec.Name)
		}
		if inputExample["action"] != spec.Name {
			t.Fatalf("expected input action %q for %s, got %#v", spec.Name, spec.Name, inputExample["action"])
		}

		outputExample, ok := spec.OutputExample.(map[string]any)
		if !ok {
			t.Fatalf("expected output example object for %s", spec.Name)
		}
		if outputExample["ok"] != true {
			t.Fatalf("expected ok=true for %s", spec.Name)
		}
	}

	for _, definition := range libraryManageMethodDefinitions {
		if _, ok := byName[definition.name]; !ok {
			t.Fatalf("missing method spec for action %q", definition.name)
		}
	}
}

func TestLibraryManageToolSchemaCoversAllActions(t *testing.T) {
	var parsed map[string]any
	if err := json.Unmarshal([]byte(libraryManageToolSchema()), &parsed); err != nil {
		t.Fatalf("unmarshal schema: %v", err)
	}

	if parsed["type"] != "object" {
		t.Fatalf("expected top-level object schema, got %#v", parsed["type"])
	}
	properties, ok := parsed["properties"].(map[string]any)
	if !ok {
		t.Fatalf("expected properties object")
	}
	actionProp, ok := properties["action"].(map[string]any)
	if !ok {
		t.Fatalf("expected action property schema")
	}
	actionEnum, ok := actionProp["enum"].([]any)
	if !ok {
		t.Fatalf("expected action enum")
	}
	if len(actionEnum) != len(libraryManageMethodDefinitions) {
		t.Fatalf("expected %d action enum values, got %d", len(libraryManageMethodDefinitions), len(actionEnum))
	}
}

type libraryManageServiceStub struct {
	prepareRequest          librarydto.PrepareYTDLPDownloadRequest
	createDownloadRequest   librarydto.CreateYTDLPJobRequest
	prepareResponse         librarydto.PrepareYTDLPDownloadResponse
	createDownloadResponse  librarydto.LibraryOperationDTO
	cancelRequest           librarydto.CancelOperationRequest
	cancelOperationResponse librarydto.LibraryOperationDTO
}

func (stub *libraryManageServiceStub) PrepareYTDLPDownload(_ context.Context, request librarydto.PrepareYTDLPDownloadRequest) (librarydto.PrepareYTDLPDownloadResponse, error) {
	stub.prepareRequest = request
	return stub.prepareResponse, nil
}

func (stub *libraryManageServiceStub) ParseYTDLPDownload(context.Context, librarydto.ParseYTDLPDownloadRequest) (librarydto.ParseYTDLPDownloadResponse, error) {
	return librarydto.ParseYTDLPDownloadResponse{}, nil
}

func (stub *libraryManageServiceStub) CreateYTDLPJob(_ context.Context, request librarydto.CreateYTDLPJobRequest) (librarydto.LibraryOperationDTO, error) {
	stub.createDownloadRequest = request
	return stub.createDownloadResponse, nil
}

func (stub *libraryManageServiceStub) RetryYTDLPOperation(context.Context, librarydto.RetryYTDLPOperationRequest) (librarydto.LibraryOperationDTO, error) {
	return librarydto.LibraryOperationDTO{}, nil
}

func (stub *libraryManageServiceStub) CreateTranscodeJob(context.Context, librarydto.CreateTranscodeJobRequest) (librarydto.LibraryOperationDTO, error) {
	return librarydto.LibraryOperationDTO{}, nil
}

func (stub *libraryManageServiceStub) CreateSubtitleTranslateJob(context.Context, librarydto.SubtitleTranslateRequest) (librarydto.LibraryOperationDTO, error) {
	return librarydto.LibraryOperationDTO{}, nil
}

func (stub *libraryManageServiceStub) CreateSubtitleProofreadJob(context.Context, librarydto.SubtitleProofreadRequest) (librarydto.LibraryOperationDTO, error) {
	return librarydto.LibraryOperationDTO{}, nil
}

func (stub *libraryManageServiceStub) CreateSubtitleQAReviewJob(context.Context, librarydto.SubtitleQAReviewRequest) (librarydto.LibraryOperationDTO, error) {
	return librarydto.LibraryOperationDTO{}, nil
}

func (stub *libraryManageServiceStub) CancelOperation(_ context.Context, request librarydto.CancelOperationRequest) (librarydto.LibraryOperationDTO, error) {
	stub.cancelRequest = request
	return stub.cancelOperationResponse, nil
}

func (stub *libraryManageServiceStub) ResumeOperation(context.Context, librarydto.ResumeOperationRequest) (librarydto.LibraryOperationDTO, error) {
	return librarydto.LibraryOperationDTO{}, nil
}

func TestRunLibraryManageToolReturnsAsyncAcceptedResult(t *testing.T) {
	t.Parallel()

	stub := &libraryManageServiceStub{
		createDownloadResponse: librarydto.LibraryOperationDTO{
			ID:        "op_download_123",
			LibraryID: "lib_123",
			Kind:      "download",
			Status:    "queued",
		},
	}
	handler := runLibraryManageTool(stub)
	output, err := handler(WithRuntimeContext(context.Background(), "session-main", "run-parent"), `{"action":"download.create","url":"https://example.com/video"}`)
	if err != nil {
		t.Fatalf("run library_manage: %v", err)
	}
	if stub.createDownloadRequest.SessionKey != "session-main" {
		t.Fatalf("expected sessionKey injection, got %q", stub.createDownloadRequest.SessionKey)
	}
	if stub.createDownloadRequest.RunID != "run-parent" {
		t.Fatalf("expected runId injection, got %q", stub.createDownloadRequest.RunID)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	if payload["action"] != "download.create" {
		t.Fatalf("expected action download.create, got %#v", payload["action"])
	}
	if payload["async"] != true {
		t.Fatalf("expected async=true, got %#v", payload["async"])
	}
	result, ok := payload["result"].(map[string]any)
	if !ok {
		t.Fatalf("expected result object")
	}
	if result["operationId"] != "op_download_123" {
		t.Fatalf("expected operationId, got %#v", result["operationId"])
	}
}

func TestRunLibraryManageToolPrepareReturnsSyncResult(t *testing.T) {
	t.Parallel()

	stub := &libraryManageServiceStub{
		prepareResponse: librarydto.PrepareYTDLPDownloadResponse{
			URL:                "https://example.com/video",
			Domain:             "example.com",
			ConnectorAvailable: true,
		},
	}
	handler := runLibraryManageTool(stub)
	output, err := handler(context.Background(), `{"action":"download.prepare","url":"https://example.com/video"}`)
	if err != nil {
		t.Fatalf("run library_manage: %v", err)
	}
	if stub.prepareRequest.URL != "https://example.com/video" {
		t.Fatalf("expected prepare request url, got %q", stub.prepareRequest.URL)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	if payload["async"] != false {
		t.Fatalf("expected async=false, got %#v", payload["async"])
	}
}
