package llms

import (
	"context"
)

type LLMs struct{}

type Interface interface {
	Completion(ctx context.Context, prompt string) (text string, err error)
	ChatCompletion(ctx context.Context, systemPrompt, userPrompt string) (text string, err error)
	ChatCompletionStream(ctx context.Context, systemPrompt, userPrompt string) (<-chan string, <-chan error)
}
