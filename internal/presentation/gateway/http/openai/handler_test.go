package openai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	runtimedto "dreamcreator/internal/application/gateway/runtime/dto"
)

type stubRuntime struct {
	run func(ctx context.Context, request runtimedto.RuntimeRunRequest) (runtimedto.RuntimeRunResult, error)
}

func (stub stubRuntime) Run(ctx context.Context, request runtimedto.RuntimeRunRequest) (runtimedto.RuntimeRunResult, error) {
	return stub.run(ctx, request)
}

func collectSSEData(body string) []string {
	lines := strings.Split(body, "\n")
	data := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "data: ") {
			data = append(data, strings.TrimPrefix(line, "data: "))
		}
	}
	return data
}

func TestContractReplay(t *testing.T) {
	t.Run("non-stream", func(t *testing.T) {
		runtime := stubRuntime{run: func(ctx context.Context, request runtimedto.RuntimeRunRequest) (runtimedto.RuntimeRunResult, error) {
			if request.SessionID != "session-1" {
				t.Fatalf("expected sessionID session-1, got %q", request.SessionID)
			}
			if request.Model == nil || request.Model.ProviderID != "test" || request.Model.Name != "mock" {
				t.Fatalf("unexpected model ref: %+v", request.Model)
			}
			if len(request.Input.Messages) != 1 || request.Input.Messages[0].Content != "hello" {
				t.Fatalf("unexpected messages: %+v", request.Input.Messages)
			}
			return runtimedto.RuntimeRunResult{
				Status: "completed",
				AssistantMessage: runtimedto.Message{Role: "assistant", Content: "hi"},
			}, nil
		}}

		handler := NewHandler(runtime)
		payload := ChatCompletionRequest{
			Model: "test:mock",
			Messages: []OpenAIMessage{{
				Role:    "user",
				Content: "hello",
			}},
			Stream: false,
			User:   "session-1",
		}
		body, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal request: %v", err)
		}
		req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(string(body)))
		req.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", recorder.Code)
		}
		var response ChatCompletionResponse
		if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if !strings.HasPrefix(response.ID, "chatcmpl_") {
			t.Fatalf("unexpected response id: %q", response.ID)
		}
		if response.Object != "chat.completion" || len(response.Choices) != 1 {
			t.Fatalf("unexpected response: %+v", response)
		}
		if response.Choices[0].Message.Content != "hi" || response.Choices[0].FinishReason != "stop" {
			t.Fatalf("unexpected choice: %+v", response.Choices[0])
		}
	})

	t.Run("stream", func(t *testing.T) {
		runtime := stubRuntime{run: func(ctx context.Context, request runtimedto.RuntimeRunRequest) (runtimedto.RuntimeRunResult, error) {
			return runtimedto.RuntimeRunResult{
				Status: "completed",
				AssistantMessage: runtimedto.Message{Role: "assistant", Content: "streaming"},
			}, nil
		}}
		handler := NewHandler(runtime)
		payload := ChatCompletionRequest{
			Model: "test:mock",
			Messages: []OpenAIMessage{{
				Role:    "user",
				Content: "hello",
			}},
			Stream: true,
			User:   "session-1",
		}
		body, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal request: %v", err)
		}
		req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(string(body)))
		req.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", recorder.Code)
		}
		dataLines := collectSSEData(recorder.Body.String())
		if len(dataLines) < 3 {
			t.Fatalf("expected at least 3 SSE data lines, got %d", len(dataLines))
		}
		if dataLines[len(dataLines)-1] != "[DONE]" {
			t.Fatalf("expected [DONE] marker, got %q", dataLines[len(dataLines)-1])
		}
		var chunk ChatCompletionChunk
		if err := json.Unmarshal([]byte(dataLines[0]), &chunk); err != nil {
			t.Fatalf("decode first chunk: %v", err)
		}
		if !strings.HasPrefix(chunk.ID, "chatcmpl_") || chunk.Object != "chat.completion.chunk" {
			t.Fatalf("unexpected chunk: %+v", chunk)
		}
	})

	t.Run("tool-call", func(t *testing.T) {
		runtime := stubRuntime{run: func(ctx context.Context, request runtimedto.RuntimeRunRequest) (runtimedto.RuntimeRunResult, error) {
			return runtimedto.RuntimeRunResult{
				Status: "completed",
				AssistantMessage: runtimedto.Message{Role: "assistant", Content: "tool output"},
			}, nil
		}}
		handler := NewHandler(runtime)
		payload := ChatCompletionRequest{
			Model: "test:mock",
			Messages: []OpenAIMessage{{
				Role:    "user",
				Content: "call tool",
			}},
			Stream: false,
			User:   "session-1",
		}
		body, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal request: %v", err)
		}
		req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(string(body)))
		req.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", recorder.Code)
		}
		var response ChatCompletionResponse
		if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if response.Choices[0].Message.Content != "tool output" {
			t.Fatalf("unexpected tool response: %+v", response.Choices[0])
		}
	})

	t.Run("error", func(t *testing.T) {
		runtime := stubRuntime{run: func(ctx context.Context, request runtimedto.RuntimeRunRequest) (runtimedto.RuntimeRunResult, error) {
			return runtimedto.RuntimeRunResult{}, nil
		}}
		handler := NewHandler(runtime)
		payload := ChatCompletionRequest{
			Model:    "test:mock",
			Messages: nil,
			Stream:   false,
			User:     "session-1",
		}
		body, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal request: %v", err)
		}
		req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(string(body)))
		req.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", recorder.Code)
		}
	})
}
