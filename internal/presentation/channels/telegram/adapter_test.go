package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	runtimedto "dreamcreator/internal/application/gateway/runtime/dto"
)

type stubRuntime struct {
	run func(ctx context.Context, request runtimedto.RuntimeRunRequest) (runtimedto.RuntimeRunResult, error)
}

func (stub stubRuntime) Run(ctx context.Context, request runtimedto.RuntimeRunRequest) (runtimedto.RuntimeRunResult, error) {
	return stub.run(ctx, request)
}

func TestTelegramAdapterServeHTTP(t *testing.T) {
	called := false
	adapter := NewAdapter(stubRuntime{run: func(ctx context.Context, request runtimedto.RuntimeRunRequest) (runtimedto.RuntimeRunResult, error) {
		called = true
		if request.SessionID != "thread-1" {
			t.Fatalf("expected session id thread-1, got %q", request.SessionID)
		}
		if len(request.Input.Messages) != 1 || request.Input.Messages[0].Content != "ping" {
			t.Fatalf("unexpected messages: %+v", request.Input.Messages)
		}
		return runtimedto.RuntimeRunResult{
			Status: "completed",
			AssistantMessage: runtimedto.Message{
				Role:    "assistant",
				Content: "pong",
			},
		}, nil
	}})
	body := map[string]any{
		"threadId": "thread-1",
		"message":  "ping",
	}
	encoded, _ := json.Marshal(body)
	request := httptest.NewRequest(http.MethodPost, "/api/channels/telegram", bytes.NewReader(encoded))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	adapter.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	if !called {
		t.Fatalf("expected runtime to be called")
	}
	var response outboundResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if response.Reply != "pong" {
		t.Fatalf("expected reply pong, got %q", response.Reply)
	}
	if response.RunID == "" {
		t.Fatalf("expected run id")
	}
	if len(response.Events) < 1 {
		t.Fatalf("expected outbound events")
	}
}
