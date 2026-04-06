package httpfacade

import (
	"errors"
	"net/http"
	"strings"

	runtimedto "dreamcreator/internal/application/gateway/runtime/dto"
)

type ModelRef struct {
	ProviderID string
	ModelName  string
}

func ParseModelRef(model string) (ModelRef, error) {
	trimmed := strings.TrimSpace(model)
	if trimmed == "" {
		return ModelRef{}, errors.New("model is required")
	}
	if strings.Contains(trimmed, ":") {
		parts := strings.SplitN(trimmed, ":", 2)
		if len(parts) == 2 {
			return ModelRef{ProviderID: strings.TrimSpace(parts[0]), ModelName: strings.TrimSpace(parts[1])}, nil
		}
	}
	if strings.Contains(trimmed, "/") {
		parts := strings.SplitN(trimmed, "/", 2)
		if len(parts) == 2 {
			return ModelRef{ProviderID: strings.TrimSpace(parts[0]), ModelName: strings.TrimSpace(parts[1])}, nil
		}
	}
	return ModelRef{}, errors.New("model must include provider prefix")
}

func ResolveSessionKey(user string, header http.Header) string {
	if header != nil {
		if raw := strings.TrimSpace(header.Get("X-Session-Key")); raw != "" {
			return raw
		}
		if raw := strings.TrimSpace(header.Get("X-Session-Id")); raw != "" {
			return raw
		}
	}
	return strings.TrimSpace(user)
}

func BuildRuntimeRequest(messages []runtimedto.Message, model ModelRef, sessionKey string, agentID string) runtimedto.RuntimeRunRequest {
	var selection *runtimedto.ModelSelection
	if strings.TrimSpace(model.ProviderID) != "" && strings.TrimSpace(model.ModelName) != "" {
		selection = &runtimedto.ModelSelection{
			ProviderID: model.ProviderID,
			Name:       model.ModelName,
		}
	}
	return runtimedto.RuntimeRunRequest{
		SessionKey: strings.TrimSpace(sessionKey),
		AgentID:    strings.TrimSpace(agentID),
		Input: runtimedto.RuntimeInput{
			Messages: messages,
		},
		Model: selection,
	}
}
