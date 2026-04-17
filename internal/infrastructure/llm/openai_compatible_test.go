package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cloudwego/eino/schema"

	domainproviders "dreamcreator/internal/domain/providers"
)

func TestOpenAICompatibleThinkingEffort(t *testing.T) {
	t.Parallel()

	captureRequest := func(t *testing.T, config OpenAICompatibleConfig, params RuntimeParams) map[string]any {
		t.Helper()
		var captured map[string]any
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer r.Body.Close()
			if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
				t.Fatalf("decode request: %v", err)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}]}`))
		}))
		defer server.Close()

		config.BaseURL = server.URL
		model, err := NewOpenAICompatibleChatModel(config)
		if err != nil {
			t.Fatalf("new model: %v", err)
		}
		ctx := WithRuntimeParams(context.Background(), params)
		if _, err := model.Generate(ctx, []*schema.Message{{Role: schema.User, Content: "hi"}}); err != nil {
			t.Fatalf("generate: %v", err)
		}
		return captured
	}

	t.Run("custom openai compatible provider injects reasoning_effort", func(t *testing.T) {
		t.Parallel()
		captured := captureRequest(t, OpenAICompatibleConfig{
			Model:                 "gpt-5.2",
			ProviderID:            "custom-openai",
			ProviderType:          domainproviders.ProviderTypeOpenAI,
			ProviderCompatibility: domainproviders.ProviderCompatibilityOpenAI,
		}, RuntimeParams{ThinkingLevel: "high"})

		if got, _ := captured["reasoning_effort"].(string); got != "high" {
			t.Fatalf("expected reasoning_effort=high, got %q", got)
		}
	})

	t.Run("openai gpt-5.1 off maps to none", func(t *testing.T) {
		t.Parallel()
		captured := captureRequest(t, OpenAICompatibleConfig{
			Model:                 "gpt-5.1",
			ProviderID:            "openai",
			ProviderType:          domainproviders.ProviderTypeOpenAI,
			ProviderCompatibility: domainproviders.ProviderCompatibilityOpenAI,
		}, RuntimeParams{ThinkingLevel: "off"})

		if got, _ := captured["reasoning_effort"].(string); got != "none" {
			t.Fatalf("expected reasoning_effort=none, got %q", got)
		}
	})

	t.Run("legacy gpt-5 omits none because off is unsupported", func(t *testing.T) {
		t.Parallel()
		captured := captureRequest(t, OpenAICompatibleConfig{
			Model:                 "gpt-5",
			ProviderID:            "openai",
			ProviderType:          domainproviders.ProviderTypeOpenAI,
			ProviderCompatibility: domainproviders.ProviderCompatibilityOpenAI,
		}, RuntimeParams{ThinkingLevel: "off"})

		if _, exists := captured["reasoning_effort"]; exists {
			t.Fatalf("expected legacy gpt-5 to omit reasoning_effort for off, got %#v", captured["reasoning_effort"])
		}
	})

	t.Run("custom google compatible provider omits none for unsupported models", func(t *testing.T) {
		t.Parallel()
		captured := captureRequest(t, OpenAICompatibleConfig{
			Model:                 "gemini-3-flash-preview",
			ProviderID:            "custom-google",
			ProviderType:          domainproviders.ProviderTypeOpenAI,
			ProviderCompatibility: domainproviders.ProviderCompatibilityGoogle,
		}, RuntimeParams{ThinkingLevel: "off"})

		if _, exists := captured["reasoning_effort"]; exists {
			t.Fatalf("google-compatible gemini 3 should omit reasoning_effort for off, got %#v", captured["reasoning_effort"])
		}
	})

	t.Run("google-compatible gemini 2.5 flash maps off to none", func(t *testing.T) {
		t.Parallel()
		captured := captureRequest(t, OpenAICompatibleConfig{
			Model:                 "gemini-2.5-flash",
			ProviderID:            "google",
			ProviderType:          domainproviders.ProviderTypeOpenAI,
			ProviderCompatibility: domainproviders.ProviderCompatibilityGoogle,
		}, RuntimeParams{ThinkingLevel: "off"})

		if got, _ := captured["reasoning_effort"].(string); got != "none" {
			t.Fatalf("expected reasoning_effort=none, got %q", got)
		}
	})

	t.Run("custom openrouter compatible provider injects reasoning object", func(t *testing.T) {
		t.Parallel()
		captured := captureRequest(t, OpenAICompatibleConfig{
			Model:                 "openai/gpt-5.2",
			ProviderID:            "custom-openrouter",
			ProviderType:          domainproviders.ProviderTypeOpenAI,
			ProviderCompatibility: domainproviders.ProviderCompatibilityOpenRouter,
		}, RuntimeParams{ThinkingLevel: "medium"})

		reasoning, _ := captured["reasoning"].(map[string]any)
		if got, _ := reasoning["effort"].(string); got != "medium" {
			t.Fatalf("expected reasoning.effort=medium, got %#v", reasoning["effort"])
		}
		if _, exists := captured["reasoning_effort"]; exists {
			t.Fatalf("openrouter should not use reasoning_effort")
		}
	})

	t.Run("custom deepseek compatible provider injects thinking object", func(t *testing.T) {
		t.Parallel()
		captured := captureRequest(t, OpenAICompatibleConfig{
			Model:                 "deepseek-chat",
			ProviderID:            "custom-deepseek",
			ProviderType:          domainproviders.ProviderTypeOpenAI,
			ProviderCompatibility: domainproviders.ProviderCompatibilityDeepSeek,
		}, RuntimeParams{ThinkingLevel: "high"})

		thinking, _ := captured["thinking"].(map[string]any)
		if got, _ := thinking["type"].(string); got != "enabled" {
			t.Fatalf("expected thinking.type=enabled, got %#v", thinking["type"])
		}
		if _, exists := captured["reasoning_effort"]; exists {
			t.Fatalf("deepseek should not use reasoning_effort")
		}
	})

	t.Run("custom deepseek compatible provider disables thinking explicitly", func(t *testing.T) {
		t.Parallel()
		captured := captureRequest(t, OpenAICompatibleConfig{
			Model:                 "deepseek-chat",
			ProviderID:            "custom-deepseek",
			ProviderType:          domainproviders.ProviderTypeOpenAI,
			ProviderCompatibility: domainproviders.ProviderCompatibilityDeepSeek,
		}, RuntimeParams{ThinkingLevel: "off"})

		thinking, _ := captured["thinking"].(map[string]any)
		if got, _ := thinking["type"].(string); got != "disabled" {
			t.Fatalf("expected thinking.type=disabled, got %#v", thinking["type"])
		}
	})

	t.Run("anthropic compatibility uses thinking budget and omits reasoning_effort", func(t *testing.T) {
		t.Parallel()
		captured := captureRequest(t, OpenAICompatibleConfig{
			Model:                 "claude-sonnet-4-6",
			ProviderID:            "anthropic",
			ProviderType:          domainproviders.ProviderTypeAnthropic,
			ProviderCompatibility: domainproviders.ProviderCompatibilityAnthropic,
		}, RuntimeParams{ThinkingLevel: "medium"})

		thinking, _ := captured["thinking"].(map[string]any)
		if got, _ := thinking["type"].(string); got != "enabled" {
			t.Fatalf("expected thinking.type=enabled, got %#v", thinking["type"])
		}
		if got, _ := thinking["budget_tokens"].(float64); got != 4096 {
			t.Fatalf("expected budget_tokens=4096, got %#v", thinking["budget_tokens"])
		}
		if got, _ := captured["temperature"].(float64); got != 1 {
			t.Fatalf("expected temperature=1 for anthropic thinking, got %#v", captured["temperature"])
		}
		if _, exists := captured["reasoning_effort"]; exists {
			t.Fatalf("anthropic should not use reasoning_effort")
		}
	})

	t.Run("anthropic compatibility omits off because disable is undocumented in compatibility mode", func(t *testing.T) {
		t.Parallel()
		captured := captureRequest(t, OpenAICompatibleConfig{
			Model:                 "claude-sonnet-4-5",
			ProviderID:            "anthropic-custom",
			ProviderType:          domainproviders.ProviderTypeAnthropic,
			ProviderCompatibility: domainproviders.ProviderCompatibilityAnthropic,
		}, RuntimeParams{ThinkingLevel: "off"})

		if _, exists := captured["thinking"]; exists {
			t.Fatalf("expected anthropic off to omit thinking in compatibility mode, got %#v", captured["thinking"])
		}
		if _, exists := captured["reasoning_effort"]; exists {
			t.Fatalf("expected anthropic off to omit reasoning_effort, got %#v", captured["reasoning_effort"])
		}
	})

	t.Run("glm models use thinking enabled toggle", func(t *testing.T) {
		t.Parallel()
		captured := captureRequest(t, OpenAICompatibleConfig{
			Model:                 "glm-4.6",
			ProviderID:            "custom-glm",
			ProviderType:          domainproviders.ProviderTypeOpenAI,
			ProviderCompatibility: domainproviders.ProviderCompatibilityOpenAI,
		}, RuntimeParams{ThinkingLevel: "high"})

		thinking, _ := captured["thinking"].(map[string]any)
		if got, _ := thinking["type"].(string); got != "enabled" {
			t.Fatalf("expected thinking.type=enabled, got %#v", thinking["type"])
		}
	})

	t.Run("kimi k2.5 disables thinking with thinking object", func(t *testing.T) {
		t.Parallel()
		captured := captureRequest(t, OpenAICompatibleConfig{
			Model:                 "kimi-k2.5",
			ProviderID:            "custom-kimi",
			ProviderType:          domainproviders.ProviderTypeOpenAI,
			ProviderCompatibility: domainproviders.ProviderCompatibilityOpenAI,
		}, RuntimeParams{ThinkingLevel: "off"})

		thinking, _ := captured["thinking"].(map[string]any)
		if got, _ := thinking["type"].(string); got != "disabled" {
			t.Fatalf("expected thinking.type=disabled, got %#v", thinking["type"])
		}
	})

	t.Run("qwen models use enable_thinking flag", func(t *testing.T) {
		t.Parallel()
		captured := captureRequest(t, OpenAICompatibleConfig{
			Model:                 "qwen-plus-2025-04-28",
			ProviderID:            "custom-qwen",
			ProviderType:          domainproviders.ProviderTypeOpenAI,
			ProviderCompatibility: domainproviders.ProviderCompatibilityOpenAI,
		}, RuntimeParams{ThinkingLevel: "off"})

		if got, ok := captured["enable_thinking"].(bool); !ok || got {
			t.Fatalf("expected enable_thinking=false, got %#v", captured["enable_thinking"])
		}
	})

	t.Run("xai reasoning models omit reasoning control fields", func(t *testing.T) {
		t.Parallel()
		captured := captureRequest(t, OpenAICompatibleConfig{
			Model:                 "grok-4.20-reasoning",
			ProviderID:            "custom-xai",
			ProviderType:          domainproviders.ProviderTypeOpenAI,
			ProviderCompatibility: domainproviders.ProviderCompatibilityOpenAI,
		}, RuntimeParams{ThinkingLevel: "high"})

		if _, exists := captured["reasoning_effort"]; exists {
			t.Fatalf("expected xai reasoning model to omit reasoning_effort, got %#v", captured["reasoning_effort"])
		}
		if _, exists := captured["thinking"]; exists {
			t.Fatalf("expected xai reasoning model to omit thinking, got %#v", captured["thinking"])
		}
	})
}

func TestOpenAICompatibleStructuredOutputAddsResponseFormat(t *testing.T) {
	t.Parallel()

	var captured map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"{\"items\":[]}"},"finish_reason":"stop"}]}`))
	}))
	defer server.Close()

	model, err := NewOpenAICompatibleChatModel(OpenAICompatibleConfig{
		BaseURL: server.URL,
		Model:   "gpt-5",
	})
	if err != nil {
		t.Fatalf("new model: %v", err)
	}

	ctx := WithRuntimeParams(context.Background(), RuntimeParams{
		StructuredOutput: StructuredOutputConfig{
			Mode: "json_schema",
			Name: "subtitle_chunk",
			Schema: map[string]any{
				"type": "object",
			},
			Strict: true,
		},
	})
	if _, err := model.Generate(ctx, []*schema.Message{{Role: schema.User, Content: "hi"}}); err != nil {
		t.Fatalf("generate: %v", err)
	}

	responseFormat, _ := captured["response_format"].(map[string]any)
	if got, _ := responseFormat["type"].(string); got != "json_schema" {
		t.Fatalf("expected response_format.type=json_schema, got %#v", responseFormat["type"])
	}
	jsonSchema, _ := responseFormat["json_schema"].(map[string]any)
	if got, _ := jsonSchema["name"].(string); got != "subtitle_chunk" {
		t.Fatalf("expected json_schema.name=subtitle_chunk, got %#v", jsonSchema["name"])
	}
}

func TestOpenAICompatibleStreamStructuredOutputAutoFallback(t *testing.T) {
	t.Parallel()

	var (
		requestCount int
		captured     []map[string]any
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		captured = append(captured, payload)
		requestCount++
		if requestCount == 1 {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":{"message":"response_format json_schema is not supported"}}`))
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher, _ := w.(http.Flusher)
		if flusher != nil {
			flusher.Flush()
		}
		_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"hello\"},\"index\":0}]}\n\n")
		if flusher != nil {
			flusher.Flush()
		}
		_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
		if flusher != nil {
			flusher.Flush()
		}
	}))
	defer server.Close()

	model, err := NewOpenAICompatibleChatModel(OpenAICompatibleConfig{
		BaseURL: server.URL,
		Model:   "gpt-5",
	})
	if err != nil {
		t.Fatalf("new model: %v", err)
	}

	ctx := WithRuntimeParams(context.Background(), RuntimeParams{
		StructuredOutput: StructuredOutputConfig{
			Mode: "auto",
			Name: "subtitle_chunk",
			Schema: map[string]any{
				"type": "object",
			},
			Strict: true,
		},
	})
	stream, err := model.Stream(ctx, []*schema.Message{{Role: schema.User, Content: "hi"}})
	if err != nil {
		t.Fatalf("stream: %v", err)
	}
	defer stream.Close()

	var content strings.Builder
	for {
		message, recvErr := stream.Recv()
		if recvErr != nil {
			if errors.Is(recvErr, io.EOF) {
				break
			}
			t.Fatalf("recv: %v", recvErr)
		}
		if message != nil {
			content.WriteString(message.Content)
		}
	}

	if requestCount != 2 {
		t.Fatalf("expected 2 requests, got %d", requestCount)
	}
	if _, ok := captured[0]["response_format"]; !ok {
		t.Fatal("expected first request to include response_format")
	}
	if _, ok := captured[1]["response_format"]; ok {
		t.Fatal("expected second request to omit response_format after fallback")
	}
	if content.String() != "hello" {
		t.Fatalf("expected streamed content hello, got %q", content.String())
	}
}

func TestOpenAICompatibleStream_IgnoresClientTotalTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher, _ := w.(http.Flusher)
		if flusher != nil {
			flusher.Flush()
		}
		time.Sleep(80 * time.Millisecond)
		_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"hello\"},\"index\":0}]}\n\n")
		if flusher != nil {
			flusher.Flush()
		}
		_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
		if flusher != nil {
			flusher.Flush()
		}
	}))
	defer server.Close()

	model, err := NewOpenAICompatibleChatModel(OpenAICompatibleConfig{
		BaseURL:    server.URL,
		Model:      "gpt-5",
		HTTPClient: &http.Client{Timeout: 20 * time.Millisecond},
	})
	if err != nil {
		t.Fatalf("new model: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	stream, err := model.Stream(ctx, []*schema.Message{{Role: schema.User, Content: "hi"}})
	if err != nil {
		t.Fatalf("stream: %v", err)
	}
	defer stream.Close()

	var content strings.Builder
	for {
		msg, recvErr := stream.Recv()
		if recvErr != nil {
			if errors.Is(recvErr, io.EOF) {
				break
			}
			t.Fatalf("recv: %v", recvErr)
		}
		if msg != nil && msg.Content != "" {
			content.WriteString(msg.Content)
		}
	}

	if content.String() != "hello" {
		t.Fatalf("expected streamed content 'hello', got %q", content.String())
	}
}

func TestOpenAICompatibleStream_IdleTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher, _ := w.(http.Flusher)
		if flusher != nil {
			flusher.Flush()
		}
		time.Sleep(120 * time.Millisecond)
	}))
	defer server.Close()

	model, err := NewOpenAICompatibleChatModel(OpenAICompatibleConfig{
		BaseURL:           server.URL,
		Model:             "gpt-5",
		HTTPClient:        &http.Client{Timeout: time.Second},
		StreamIdleTimeout: 30 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("new model: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	stream, err := model.Stream(ctx, []*schema.Message{{Role: schema.User, Content: "hi"}})
	if err != nil {
		t.Fatalf("stream: %v", err)
	}
	defer stream.Close()

	_, recvErr := stream.Recv()
	if recvErr == nil {
		t.Fatalf("expected idle timeout error")
	}
	if !strings.Contains(strings.ToLower(recvErr.Error()), "stream idle timeout") {
		t.Fatalf("expected stream idle timeout error, got %v", recvErr)
	}
}

func TestEmitStreamChunk_EmitsUsageWhenChoicesAreEmpty(t *testing.T) {
	t.Parallel()

	reader, writer := schema.Pipe[*schema.Message](8)
	if err := emitStreamChunk(`{"choices":[],"usage":{"prompt_tokens":12,"completion_tokens":8,"total_tokens":20}}`, writer); err != nil {
		t.Fatalf("emit stream chunk: %v", err)
	}
	writer.Close()
	defer reader.Close()

	message, err := reader.Recv()
	if err != nil {
		t.Fatalf("recv: %v", err)
	}
	if message == nil || message.ResponseMeta == nil || message.ResponseMeta.Usage == nil {
		t.Fatalf("expected usage response meta message")
	}
	if message.ResponseMeta.Usage.PromptTokens != 12 {
		t.Fatalf("expected prompt tokens 12, got %d", message.ResponseMeta.Usage.PromptTokens)
	}
	if message.ResponseMeta.Usage.CompletionTokens != 8 {
		t.Fatalf("expected completion tokens 8, got %d", message.ResponseMeta.Usage.CompletionTokens)
	}
	if message.ResponseMeta.Usage.TotalTokens != 20 {
		t.Fatalf("expected total tokens 20, got %d", message.ResponseMeta.Usage.TotalTokens)
	}
}

func TestOpenAICompatibleGenerate_RecordsCall(t *testing.T) {
	t.Parallel()

	recorder := &stubCallRecorder{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":11,"completion_tokens":7,"total_tokens":18}}`))
	}))
	defer server.Close()

	model, err := NewOpenAICompatibleChatModel(OpenAICompatibleConfig{
		BaseURL:  server.URL,
		Model:    "gpt-5",
		Recorder: recorder,
	})
	if err != nil {
		t.Fatalf("new model: %v", err)
	}

	ctx := WithRuntimeParams(context.Background(), RuntimeParams{
		ProviderID:    "openai",
		ModelName:     "gpt-5",
		SessionID:     "thread-1",
		ThreadID:      "thread-1",
		RunID:         "run-1",
		RequestSource: "dialogue",
		Operation:     "runtime.run",
	})
	if _, err := model.Generate(ctx, []*schema.Message{{Role: schema.User, Content: "hi"}}); err != nil {
		t.Fatalf("generate: %v", err)
	}

	calls := recorder.snapshot()
	if len(calls) != 1 {
		t.Fatalf("expected 1 recorded call, got %d", len(calls))
	}
	call := calls[0]
	if call.start.ThreadID != "thread-1" || call.start.RunID != "run-1" {
		t.Fatalf("unexpected correlation fields: %#v", call.start)
	}
	if !strings.Contains(call.start.RequestPayload, `"messages"`) {
		t.Fatalf("expected request payload to be recorded, got %q", call.start.RequestPayload)
	}
	if call.finish.Status != CallRecordStatusCompleted {
		t.Fatalf("expected completed status, got %q", call.finish.Status)
	}
	if call.finish.TotalTokens != 18 {
		t.Fatalf("expected total tokens 18, got %d", call.finish.TotalTokens)
	}
	if call.finish.FinishReason != "stop" {
		t.Fatalf("expected finish reason stop, got %q", call.finish.FinishReason)
	}
}

func TestOpenAICompatibleStream_FallbackRecordsEachHTTPCall(t *testing.T) {
	t.Parallel()

	recorder := &stubCallRecorder{}
	var requestCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount == 1 {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":{"message":"response_format json_schema is not supported"}}`))
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		if flusher != nil {
			flusher.Flush()
		}
		_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"hello\"},\"index\":0}]}\n\n")
		if flusher != nil {
			flusher.Flush()
		}
		_, _ = fmt.Fprint(w, "data: {\"choices\":[],\"usage\":{\"prompt_tokens\":12,\"completion_tokens\":8,\"total_tokens\":20}}\n\n")
		if flusher != nil {
			flusher.Flush()
		}
		_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
		if flusher != nil {
			flusher.Flush()
		}
	}))
	defer server.Close()

	model, err := NewOpenAICompatibleChatModel(OpenAICompatibleConfig{
		BaseURL:  server.URL,
		Model:    "gpt-5",
		Recorder: recorder,
	})
	if err != nil {
		t.Fatalf("new model: %v", err)
	}

	ctx := WithRuntimeParams(context.Background(), RuntimeParams{
		StructuredOutput: StructuredOutputConfig{
			Mode: "auto",
			Name: "subtitle_chunk",
			Schema: map[string]any{
				"type": "object",
			},
			Strict: true,
		},
		ProviderID:    "openai",
		ModelName:     "gpt-5",
		SessionID:     "thread-2",
		ThreadID:      "thread-2",
		RunID:         "run-2",
		RequestSource: "dialogue",
		Operation:     "runtime.run",
	})
	stream, err := model.Stream(ctx, []*schema.Message{{Role: schema.User, Content: "hi"}})
	if err != nil {
		t.Fatalf("stream: %v", err)
	}
	defer stream.Close()
	for {
		_, recvErr := stream.Recv()
		if recvErr != nil {
			if errors.Is(recvErr, io.EOF) {
				break
			}
			t.Fatalf("recv: %v", recvErr)
		}
	}

	calls := recorder.snapshot()
	if len(calls) != 2 {
		t.Fatalf("expected 2 recorded calls, got %d", len(calls))
	}
	if calls[0].finish.Status != CallRecordStatusError {
		t.Fatalf("expected first call to be recorded as error, got %q", calls[0].finish.Status)
	}
	if calls[1].finish.Status != CallRecordStatusCompleted {
		t.Fatalf("expected second call to be recorded as completed, got %q", calls[1].finish.Status)
	}
	if calls[1].finish.TotalTokens != 20 {
		t.Fatalf("expected second call total tokens 20, got %d", calls[1].finish.TotalTokens)
	}
}

type stubCallRecorder struct {
	mu       sync.Mutex
	nextID   int
	starts   map[string]CallRecordStart
	finishes map[string]CallRecordFinish
	order    []string
}

type recordedCall struct {
	id     string
	start  CallRecordStart
	finish CallRecordFinish
}

func (recorder *stubCallRecorder) StartLLMCall(_ context.Context, record CallRecordStart) (string, error) {
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	if recorder.starts == nil {
		recorder.starts = make(map[string]CallRecordStart)
		recorder.finishes = make(map[string]CallRecordFinish)
	}
	recorder.nextID++
	id := fmt.Sprintf("call-%d", recorder.nextID)
	recorder.starts[id] = record
	recorder.order = append(recorder.order, id)
	return id, nil
}

func (recorder *stubCallRecorder) FinishLLMCall(_ context.Context, record CallRecordFinish) error {
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	if recorder.finishes == nil {
		recorder.finishes = make(map[string]CallRecordFinish)
	}
	recorder.finishes[record.ID] = record
	return nil
}

func (recorder *stubCallRecorder) snapshot() []recordedCall {
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	result := make([]recordedCall, 0, len(recorder.order))
	for _, id := range recorder.order {
		result = append(result, recordedCall{
			id:     id,
			start:  recorder.starts[id],
			finish: recorder.finishes[id],
		})
	}
	return result
}
