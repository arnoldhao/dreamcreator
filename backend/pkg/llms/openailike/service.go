package openailike

import (
	"context"
	"errors"
	"io"

	"github.com/sashabaranov/go-openai"
)

type Service struct {
	Client *openai.Client
	Model  string
}

func New(token, baseURL, model string) *Service {
	conf := openai.DefaultConfig(token)
	conf.BaseURL = baseURL
	return &Service{Client: openai.NewClientWithConfig(conf), Model: model}
}

func (s *Service) Completion(ctx context.Context, prompt string) (text string, err error) {
	resp, err := s.Client.CreateCompletion(ctx, openai.CompletionRequest{
		Model:  s.Model,
		Prompt: prompt,
	})
	if err != nil {
		return
	}

	text = resp.Choices[0].Text
	return
}

func (s *Service) ChatCompletion(ctx context.Context, systemPrompt, userPrompt string) (text string, err error) {
	resp, err := s.Client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: s.Model,
		Messages: []openai.ChatCompletionMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
	})
	if err != nil {
		return
	}

	text = resp.Choices[0].Message.Content
	return
}

func (s *Service) ChatCompletionStream(ctx context.Context, systemPrompt, userPrompt string) (<-chan string, <-chan error) {
	textChan := make(chan string)
	errChan := make(chan error, 1) // use buffer to avoid blocking

	stream, err := s.Client.CreateChatCompletionStream(ctx, openai.ChatCompletionRequest{
		Model:     s.Model,
		MaxTokens: 100,
		Messages: []openai.ChatCompletionMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Stream: true,
	})

	if err != nil {
		go func() {
			errChan <- err
			close(textChan)
			close(errChan)
		}()
		return textChan, errChan
	}

	go func() {
		defer close(textChan)
		defer close(errChan)
		defer stream.Close()

		for {
			response, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				return
			}
			if err != nil {
				errChan <- err
				return
			}

			if len(response.Choices) > 0 {
				textChan <- response.Choices[0].Delta.Content
			}
		}
	}()

	return textChan, errChan
}
