package llm

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/cloudwego/eino/components/model"

	"dreamcreator/internal/domain/providers"
)

type ChatModelFactory struct {
	httpClient *http.Client
	recorder   CallRecorder
}

func NewChatModelFactory() *ChatModelFactory {
	return &ChatModelFactory{
		httpClient: &http.Client{Timeout: defaultHTTPClientTimeout},
	}
}

func (factory *ChatModelFactory) SetCallRecorder(recorder CallRecorder) {
	if factory == nil {
		return
	}
	factory.recorder = recorder
}

func (factory *ChatModelFactory) NewChatModel(provider providers.Provider, apiKey string, modelName string) (model.BaseChatModel, error) {
	baseURL := strings.TrimSpace(provider.Endpoint)
	if baseURL == "" {
		return nil, fmt.Errorf("provider endpoint is required")
	}
	if strings.TrimSpace(modelName) == "" {
		return nil, fmt.Errorf("model name is required")
	}

	return NewOpenAICompatibleChatModel(OpenAICompatibleConfig{
		BaseURL:               baseURL,
		APIKey:                apiKey,
		Model:                 modelName,
		ProviderID:            provider.ID,
		ProviderType:          provider.Type,
		ProviderCompatibility: provider.Compatibility,
		ChatPath:              defaultChatPath,
		HTTPClient:            factory.httpClient,
		Headers:               map[string]string{},
		Recorder:              factory.recorder,
	})
}
