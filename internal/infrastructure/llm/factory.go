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
}

func NewChatModelFactory() *ChatModelFactory {
	return &ChatModelFactory{
		httpClient: &http.Client{Timeout: defaultHTTPClientTimeout},
	}
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
		BaseURL:    baseURL,
		APIKey:     apiKey,
		Model:      modelName,
		ChatPath:   defaultChatPath,
		HTTPClient: factory.httpClient,
		Headers:    map[string]string{},
	})
}
