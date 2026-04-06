package telegram

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"dreamcreator/internal/application/agentruntime"
	runtimedto "dreamcreator/internal/application/gateway/runtime/dto"
	"dreamcreator/internal/presentation/channels"
)

type Adapter struct {
	runtime Runtime
}

type Runtime interface {
	Run(ctx context.Context, request runtimedto.RuntimeRunRequest) (runtimedto.RuntimeRunResult, error)
}

func NewAdapter(runtime Runtime) *Adapter {
	return &Adapter{runtime: runtime}
}

func (adapter *Adapter) Name() string {
	return "telegram"
}

type inboundRequest struct {
	ThreadID   string         `json:"threadId"`
	AgentID    string         `json:"agentId,omitempty"`
	UserID     string         `json:"userId,omitempty"`
	Message    string         `json:"message"`
	System     string         `json:"system,omitempty"`
	ToolsAuto  *bool          `json:"toolsAuto,omitempty"`
	ToolsList  []string       `json:"toolsList,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

type outboundResponse struct {
	RunID    string                     `json:"runId"`
	ThreadID string                     `json:"threadId"`
	Reply    string                     `json:"reply,omitempty"`
	Events   []channels.OutboundMessage `json:"events"`
}

func (adapter *Adapter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if adapter == nil || adapter.runtime == nil {
		http.Error(w, "telegram adapter is not configured", http.StatusServiceUnavailable)
		return
	}
	if r.Method == http.MethodOptions {
		setCORS(w, r)
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	setCORS(w, r)
	defer r.Body.Close()

	var request inboundRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	request.ThreadID = strings.TrimSpace(request.ThreadID)
	request.Message = strings.TrimSpace(request.Message)
	if request.ThreadID == "" || request.Message == "" {
		http.Error(w, "threadId/message are required", http.StatusBadRequest)
		return
	}

	runID := uuid.NewString()
	messages := make([]runtimedto.Message, 0, 2)
	if system := strings.TrimSpace(request.System); system != "" {
		messages = append(messages, runtimedto.Message{Role: "system", Content: system})
	}
	messages = append(messages, runtimedto.Message{Role: "user", Content: request.Message})
	runtimeRequest := runtimedto.RuntimeRunRequest{
		RunID:      runID,
		SessionID:  request.ThreadID,
		SessionKey: request.ThreadID,
		AgentID:    strings.TrimSpace(request.AgentID),
		Input: runtimedto.RuntimeInput{
			Messages: messages,
		},
		Tools: runtimedto.ToolExecutionConfig{
			Mode:      resolveToolsMode(request.ToolsAuto),
			AllowList: request.ToolsList,
		},
		Metadata: map[string]any{
			"channel": "telegram",
			"userId":  strings.TrimSpace(request.UserID),
		},
	}
	for key, value := range request.Metadata {
		runtimeRequest.Metadata[key] = value
	}
	result, err := adapter.runtime.Run(r.Context(), runtimeRequest)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	reply := strings.TrimSpace(result.AssistantMessage.Content)
	events := buildOutboundEvents(runID, request.ThreadID, reply)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(outboundResponse{
		RunID:    runID,
		ThreadID: request.ThreadID,
		Reply:    reply,
		Events:   events,
	})
}

func buildOutboundEvents(runID string, threadID string, reply string) []channels.OutboundMessage {
	outbound := make([]channels.OutboundMessage, 0, 2)
	if reply != "" {
		outbound = append(outbound, channels.OutboundMessage{
			Channel:  "telegram",
			ThreadID: threadID,
			RunID:    runID,
			Text:     reply,
			Event: agentruntime.Event{
				Type:     agentruntime.EventTextDelta,
				RunID:    runID,
				ThreadID: threadID,
				Delta:    reply,
			},
		})
	}
	outbound = append(outbound, channels.OutboundMessage{
		Channel:  "telegram",
		ThreadID: threadID,
		RunID:    runID,
		Event: agentruntime.Event{
			Type:     agentruntime.EventRunEnd,
			RunID:    runID,
			ThreadID: threadID,
		},
	})
	return outbound
}

func resolveToolsMode(value *bool) string {
	if value == nil {
		return "auto"
	}
	if *value {
		return "auto"
	}
	return "off"
}

func setCORS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	allowHeaders := strings.TrimSpace(r.Header.Get("Access-Control-Request-Headers"))
	if allowHeaders == "" {
		allowHeaders = "Content-Type, Authorization, User-Agent"
	}
	w.Header().Set("Access-Control-Allow-Headers", allowHeaders)
}
