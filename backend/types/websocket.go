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
	Download  DownloadRequest  `json:"download"`
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

type TestProxyRequest struct {
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

type TestProxyResult struct {
	ID      string `json:"id"`
	Done    bool   `json:"done"`
	URL     string `json:"url"`
	Success bool   `json:"success"`
	Latency int    `json:"latency"`
	Error   string `json:"error"`
}

func (s TestProxyResult) WSResponseMessage() string {
	resp := WSResponseMessage{
		Event:   string(consts.WS_EVENT_TEST_PROXY_RESULT),
		Payload: s,
	}

	responseJSON, _ := json.Marshal(resp)
	return string(responseJSON)
}

type DownloadRequest struct {
	ID      string `json:"id"`
	Stream  string `json:"stream"`
	Caption string `json:"caption"`
}

type DownloadResponse struct {
	ID       string                `json:"id"`       // all task uniq id
	Status   consts.DownloadStatus `json:"status"`   // task status
	Total    int64                 `json:"total"`    // total task count
	Finished int64                 `json:"finished"` // finished task count
	Speed    string                `json:"speed"`    // task speed
	DataType ExtractorDataType     `json:"dataType"` // task data type
	Progress float64               `json:"progress"` // task progress
	Error    string                `json:"error"`    // task error
}

func (s DownloadResponse) WSResponseMessage() string {
	resp := WSResponseMessage{
		Event:   string(consts.WS_EVENT_DOWNLOAD_UPDATE),
		Payload: s,
	}

	responseJSON, _ := json.Marshal(resp)
	return string(responseJSON)
}
