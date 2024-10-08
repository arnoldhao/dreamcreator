package types

import (
	"CanMe/backend/consts"
	"encoding/json"
)

type WSRequestMessage struct {
	Event     string           `json:"event"`
	Translate TranslateRequest `json:"translate"`
	Ollama    OllamaRequest    `json:"ollama"`
	Chat      ChatRequest      `json:"chat"`
}

type TranslateRequest struct {
	ID                 string `json:"id"`
	OriginalSubtitleID string `json:"originalSubtileId"`
	Language           string `json:"language"`
}

type OllamaRequest struct {
	ID    string `json:"id"`
	Model string `json:"model"`
}

type ChatRequest struct {
	ID string `json:"id"`
}

type WSResponseMessage struct {
	Event   string      `json:"event"`
	Payload interface{} `json:"payload"`
}

type TranslateResponse struct {
	ID       string  `json:"id"`
	Content  string  `json:"content"`
	Progress float64 `json:"progress"`
	Status   string  `json:"status"`
	Error    bool    `json:"error"`
	Message  string  `json:"message"`
}

func (s TranslateResponse) WSResponseMessage() string {
	resp := WSResponseMessage{
		Event:   string(consts.TRANSLATION_UPDATE),
		Payload: s,
	}

	responseJSON, _ := json.Marshal(resp)
	return string(responseJSON)
}

type OllamaResponse struct {
	ID        string `json:"id"`
	Status    string `json:"status"`
	Digest    string `json:"digest,omitempty"`
	Total     int64  `json:"total,omitempty"`
	Completed int64  `json:"completed,omitempty"`
	Error     bool   `json:"error"`
	Message   string `json:"message"`
}

func (s OllamaResponse) WSResponseMessage() string {
	resp := WSResponseMessage{
		Event:   string(consts.OLLAMA_PULL_UPDATE),
		Payload: s,
	}

	responseJSON, _ := json.Marshal(resp)
	return string(responseJSON)
}

type ChatResponse struct {
	ID string `json:"id"`
}
