package ollama

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/ollama/ollama/api"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type Service struct {
	Model  string
	Client *api.Client
}

func New(model string, baseURL *url.URL) *Service {
	return &Service{Model: model, Client: api.NewClient(baseURL, http.DefaultClient)}
}

func (s *Service) Completion(ctx context.Context, prompt string) (text string, err error) {
	return
}

func (s *Service) ChatCompletion(ctx context.Context, systemPrompt, userPrompt string) (text string, err error) {
	return
}

func (s *Service) ChatCompletionStream(ctx context.Context, systemPrompt, userPrompt string) (<-chan string, <-chan error) {
	stream := true
	req := api.GenerateRequest{
		Model:  s.Model,
		Prompt: userPrompt,
		System: systemPrompt,
		Stream: &stream,
	}

	textChan := make(chan string)
	errChan := make(chan error, 1)

	go func() {
		defer close(textChan)
		defer close(errChan)
		err := s.Client.Generate(ctx, &req, func(resp api.GenerateResponse) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case textChan <- resp.Response:
				return nil
			}
		})

		if err != nil {
			runtime.LogError(ctx, fmt.Sprintf("ollama generate error: %v", err))
			errChan <- err
		}
	}()

	return textChan, errChan
}
