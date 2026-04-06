package openresponses

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

type CreateResponseRequest struct {
	Model  string      `json:"model"`
	Input  interface{} `json:"input"`
	Stream bool        `json:"stream"`
	User   string      `json:"user,omitempty"`
}

type CreateResponseResult struct {
	ID      string               `json:"id"`
	Object  string               `json:"object"`
	Created int64                `json:"created"`
	Model   string               `json:"model"`
	Output  []ResponseOutputItem `json:"output"`
}

type ResponseOutputItem struct {
	Type    string                `json:"type"`
	Content []ResponseContentPart `json:"content"`
}

type ResponseContentPart struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type ResponseStreamEvent struct {
	Type         string `json:"type"`
	ResponseID   string `json:"response_id,omitempty"`
	OutputIndex  int    `json:"output_index,omitempty"`
	ContentIndex int    `json:"content_index,omitempty"`
	Delta        string `json:"delta,omitempty"`
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
	var request CreateResponseRequest
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
	inputText := resolveInputText(request.Input)
	if strings.TrimSpace(inputText) == "" {
		http.Error(w, "input is required", http.StatusBadRequest)
		return
	}
	turn := httpfacade.BuildRuntimeRequest([]runtimedto.Message{{
		Role:    "user",
		Content: inputText,
	}}, modelRef, sessionKey, "")
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
	response := CreateResponseResult{
		ID:      "resp_" + uuid.NewString(),
		Object:  "response",
		Created: time.Now().Unix(),
		Model:   modelLabel,
		Output: []ResponseOutputItem{{
			Type: "output_text",
			Content: []ResponseContentPart{{
				Type: "text",
				Text: result.AssistantMessage.Content,
			}},
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
	responseID := "resp_" + uuid.NewString()
	created := time.Now().Unix()
	_ = created
	writeSSE(w, flusher, ResponseStreamEvent{
		Type:         "response.output_text.delta",
		ResponseID:   responseID,
		OutputIndex:  0,
		ContentIndex: 0,
		Delta:        result.AssistantMessage.Content,
	})
	writeSSE(w, flusher, ResponseStreamEvent{
		Type:       "response.completed",
		ResponseID: responseID,
	})
	fmt.Fprint(w, "data: [DONE]\n\n")
	flusher.Flush()
}

func resolveInputText(input any) string {
	switch typed := input.(type) {
	case string:
		return typed
	case []any:
		for _, item := range typed {
			if text, ok := item.(string); ok && strings.TrimSpace(text) != "" {
				return text
			}
		}
	case map[string]any:
		if text, ok := typed["text"].(string); ok {
			return text
		}
	}
	return ""
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
