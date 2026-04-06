package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"dreamcreator/internal/application/gateway/httpfacade"
	runtimedto "dreamcreator/internal/application/gateway/runtime/dto"
)

type ChatCompletionRequest struct {
	Model    string          `json:"model"`
	Messages []OpenAIMessage `json:"messages"`
	Stream   bool            `json:"stream"`
	User     string          `json:"user,omitempty"`
}

type OpenAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatCompletionResponse struct {
	ID      string                 `json:"id"`
	Object  string                 `json:"object"`
	Created int64                  `json:"created"`
	Model   string                 `json:"model"`
	Choices []ChatCompletionChoice `json:"choices"`
}

type ChatCompletionChoice struct {
	Index        int           `json:"index"`
	Message      OpenAIMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

type ChatCompletionChunk struct {
	ID      string                      `json:"id"`
	Object  string                      `json:"object"`
	Created int64                       `json:"created"`
	Model   string                      `json:"model"`
	Choices []ChatCompletionChunkChoice `json:"choices"`
}

type ChatCompletionChunkChoice struct {
	Index        int                `json:"index"`
	Delta        OpenAIMessageDelta `json:"delta"`
	FinishReason *string            `json:"finish_reason,omitempty"`
}

type OpenAIMessageDelta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

type Handler struct {
	runtime Runtime
}

type Runtime interface {
	Run(ctx context.Context, request runtimedto.RuntimeRunRequest) (runtimedto.RuntimeRunResult, error)
}

func NewHandler(runtime Runtime) *Handler {
	return &Handler{runtime: runtime}
}

func (handler *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		setCORSHeaders(w, r)
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	setCORSHeaders(w, r)
	if handler == nil || handler.runtime == nil {
		http.Error(w, "runtime unavailable", http.StatusServiceUnavailable)
		return
	}
	var request ChatCompletionRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	var modelRef httpfacade.ModelRef
	hasModelRef := false
	if strings.TrimSpace(request.Model) != "" {
		parsed, err := httpfacade.ParseModelRef(request.Model)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		modelRef = parsed
		hasModelRef = true
	}
	sessionKey := httpfacade.ResolveSessionKey(request.User, r.Header)
	threadID := strings.TrimSpace(r.Header.Get("X-Thread-Id"))
	if threadID == "" {
		threadID = sessionKey
	}
	if strings.TrimSpace(threadID) == "" {
		http.Error(w, "thread id is required", http.StatusBadRequest)
		return
	}
	messages := make([]runtimedto.Message, 0, len(request.Messages))
	for _, message := range request.Messages {
		if strings.TrimSpace(message.Content) == "" {
			continue
		}
		messages = append(messages, runtimedto.Message{
			Role:    strings.TrimSpace(message.Role),
			Content: message.Content,
		})
	}
	if len(messages) == 0 {
		http.Error(w, "messages are required", http.StatusBadRequest)
		return
	}
	turn := httpfacade.BuildRuntimeRequest(messages, modelRef, sessionKey, "")
	turn.SessionID = threadID
	if turn.Metadata == nil {
		turn.Metadata = make(map[string]any)
	}
	turn.Metadata["usageSource"] = "relay"
	if hasModelRef {
		turn.Model = &runtimedto.ModelSelection{
			ProviderID: modelRef.ProviderID,
			Name:       modelRef.ModelName,
		}
	}
	if request.Stream {
		handler.streamResponse(w, r, turn, request.Model)
		return
	}
	result, err := handler.runtime.Run(r.Context(), turn)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	modelLabel := strings.TrimSpace(request.Model)
	if modelLabel == "" && result.Model != nil {
		modelLabel = fmt.Sprintf("%s/%s", strings.TrimSpace(result.Model.ProviderID), strings.TrimSpace(result.Model.Name))
	}
	created := time.Now().Unix()
	response := ChatCompletionResponse{
		ID:      "chatcmpl_" + uuid.NewString(),
		Object:  "chat.completion",
		Created: created,
		Model:   modelLabel,
		Choices: []ChatCompletionChoice{{
			Index: 0,
			Message: OpenAIMessage{
				Role:    "assistant",
				Content: result.AssistantMessage.Content,
			},
			FinishReason: "stop",
		}},
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

func (handler *Handler) streamResponse(w http.ResponseWriter, r *http.Request, turn runtimedto.RuntimeRunRequest, model string) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}
	result, err := handler.runtime.Run(r.Context(), turn)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	modelLabel := strings.TrimSpace(model)
	if modelLabel == "" && result.Model != nil {
		modelLabel = fmt.Sprintf("%s/%s", strings.TrimSpace(result.Model.ProviderID), strings.TrimSpace(result.Model.Name))
	}
	id := "chatcmpl_" + uuid.NewString()
	created := time.Now().Unix()
	chunk := ChatCompletionChunk{
		ID:      id,
		Object:  "chat.completion.chunk",
		Created: created,
		Model:   modelLabel,
		Choices: []ChatCompletionChunkChoice{{
			Index: 0,
			Delta: OpenAIMessageDelta{
				Role:    "assistant",
				Content: result.AssistantMessage.Content,
			},
		}},
	}
	writeSSE(w, flusher, chunk)
	finish := "stop"
	writeSSE(w, flusher, ChatCompletionChunk{
		ID:      id,
		Object:  "chat.completion.chunk",
		Created: created,
		Model:   modelLabel,
		Choices: []ChatCompletionChunkChoice{{
			Index:        0,
			Delta:        OpenAIMessageDelta{},
			FinishReason: &finish,
		}},
	})
	fmt.Fprint(w, "data: [DONE]\n\n")
	flusher.Flush()
}

func writeSSE(w http.ResponseWriter, flusher http.Flusher, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()
}

func setCORSHeaders(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	allowHeaders := strings.TrimSpace(r.Header.Get("Access-Control-Request-Headers"))
	if allowHeaders == "" {
		allowHeaders = "Content-Type, Authorization, User-Agent"
	}
	w.Header().Set("Access-Control-Allow-Headers", allowHeaders)
}
