package websockets

import (
	"CanMe/backend/consts"
	"CanMe/backend/types"
	"encoding/json"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// handleTranslation handle translation
func (s *Service) handleTranslation(event consts.WSRequestEventType, request any) {
	var data types.TranslateRequest
	if jsonBytes, err := json.Marshal(request); err == nil {
		if err := json.Unmarshal(jsonBytes, &data); err != nil {
			runtime.LogErrorf(s.ctx, "Failed to parse message from %s: %v", event, err)
			return
		}
	} else {
		runtime.LogErrorf(s.ctx, "Failed to parse message from %s: %v", event, err)
		return
	}

	switch event {
	case consts.EVENT_TRANSLATION_START:
		err := s.iis.translate.AddTranslation(data.ID, data.OriginalSubtitleID, data.Language)
		if err != nil {
			runtime.LogError(s.ctx, "translation_start error: "+err.Error())
			s.send <- types.WSResponse{
				Namespace: consts.NAMESPACE_TRANSLATION,
				Event:     consts.EVENT_TRANSLATION_ERROR,
				Data: types.TranslateResponse{
					ID:      data.ID,
					Error:   true,
					Message: err.Error(),
				},
			}
		}
	}
}

func (s *Service) handleOllama(event consts.WSRequestEventType, request any) {
	var data types.OllamaRequest
	if jsonBytes, err := json.Marshal(request); err == nil {
		if err := json.Unmarshal(jsonBytes, &data); err != nil {
			runtime.LogErrorf(s.ctx, "Failed to parse message from %s: %v", event, err)
			return
		}
	} else {
		runtime.LogErrorf(s.ctx, "Failed to parse message from %s: %v", event, err)
		return
	}

	switch event {
	case consts.EVENT_OLLAMA_PULL:
		err := s.iis.ollama.Pull(data.ID, data.Model)
		if err != nil {
			runtime.LogErrorf(s.ctx, "Ollama pull error: %v", err)
			s.send <- types.WSResponse{
				Namespace: consts.NAMESPACE_OLLAMA,
				Event:     consts.EVENT_OLLAMA_PULL_ERROR,
				Data: types.OllamaResponse{
					ID:      data.ID,
					Error:   true,
					Message: err.Error(),
				},
			}
		}
	}
}

func (s *Service) handleChat(event consts.WSRequestEventType, request any) {
	// todo
}

func (s *Service) handleProxy(event consts.WSRequestEventType, request any) {
	var data types.TestProxyRequest
	if jsonBytes, err := json.Marshal(request); err == nil {
		if err := json.Unmarshal(jsonBytes, &data); err != nil {
			runtime.LogErrorf(s.ctx, "Failed to parse message from %s: %v", event, err)
			return
		}
	} else {
		runtime.LogErrorf(s.ctx, "Failed to parse message from %s: %v", event, err)
		return
	}

	switch event {
	case consts.EVENT_PROXY_TEST:
		err := s.iis.preference.TestProxy(data.ID)
		if err != nil {
			runtime.LogErrorf(s.ctx, "TestProxy error: %v", err)
			s.send <- types.WSResponse{
				Namespace: consts.NAMESPACE_PROXY,
				Event:     consts.EVENT_PROXY_TEST_RESULT,
				Data: types.TestProxyResult{
					ID:      data.ID,
					Done:    true,
					Success: false,
					Error:   err.Error(),
				},
			}
		}
	}
}
