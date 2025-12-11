package openai

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"

	oa "github.com/sashabaranov/go-openai"
)

// Wrapper of community SDK (go-openai) with BaseURL and custom http.Client support.
type Client struct {
	baseURL string
	apiKey  string
	httpc   *http.Client
	cli     *oa.Client
}

func NewClient(baseURL, apiKey string, httpc *http.Client) *Client {
	cfg := oa.DefaultConfig(apiKey)
	if httpc != nil {
		cfg.HTTPClient = httpc
	}
	if strings.TrimSpace(baseURL) != "" {
		cfg.BaseURL = strings.TrimRight(baseURL, "/")
	}
	return &Client{baseURL: cfg.BaseURL, apiKey: apiKey, httpc: httpc, cli: oa.NewClientWithConfig(cfg)}
}

func (c *Client) ListModels(ctx context.Context) ([]string, error) {
	resp, err := c.cli.ListModels(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(resp.Models))
	for _, m := range resp.Models {
		out = append(out, m.ID)
	}
	if len(out) == 0 {
		out = []string{"default"}
	}
	return out, nil
}

// ChatMessage mirrors provider.Service's message shape
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func (c *Client) ChatCompletions(ctx context.Context, model string, messages []ChatMessage, temperature float64) (string, error) {
	// map to go-openai messages
	m := make([]oa.ChatCompletionMessage, 0, len(messages))
	for _, x := range messages {
		m = append(m, oa.ChatCompletionMessage{Role: x.Role, Content: x.Content})
	}
	req := oa.ChatCompletionRequest{Model: model, Messages: m, Temperature: float32(temperature)}
	resp, err := c.cli.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", err
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("chat completion returned no choices")
	}
	out := resp.Choices[0].Message.Content
	if strings.TrimSpace(out) == "" {
		return "", fmt.Errorf("chat completion returned empty content")
	}
	return out, nil
}

// ChatCompletionsWithOpts supports additional OpenAI parameters like top_p, max_tokens and JSON response format.
func (c *Client) ChatCompletionsWithOpts(ctx context.Context, model string, messages []ChatMessage, temperature float64, topP float64, maxTokens int, jsonMode bool) (string, error) {
	m := make([]oa.ChatCompletionMessage, 0, len(messages))
	for _, x := range messages {
		m = append(m, oa.ChatCompletionMessage{Role: x.Role, Content: x.Content})
	}
	req := oa.ChatCompletionRequest{Model: model, Messages: m}
	if temperature > 0 {
		req.Temperature = float32(temperature)
	}
	if topP > 0 {
		req.TopP = float32(topP)
	}
	if maxTokens > 0 {
		req.MaxTokens = maxTokens
	}
	if jsonMode {
		rf := oa.ChatCompletionResponseFormat{Type: oa.ChatCompletionResponseFormatTypeJSONObject}
		req.ResponseFormat = &rf
	}
	resp, err := c.cli.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", err
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("chat completion returned no choices")
	}
	out := resp.Choices[0].Message.Content
	if strings.TrimSpace(out) == "" {
		return "", fmt.Errorf("chat completion returned empty content")
	}
	return out, nil
}

// ChatCompletionsWithOptsUsage returns both content and token usage (if provided by backend).
// Internally it uses streaming so callers can optionally observe deltas via onDelta.
func (c *Client) ChatCompletionsWithOptsUsage(
	ctx context.Context,
	model string,
	messages []ChatMessage,
	temperature float64,
	topP float64,
	maxTokens int,
	jsonMode bool,
	onDelta func(string) error,
) (string, int, int, int, error) {
	// For usage-aware requests we prefer streaming with usage included,
	// so higher layers can benefit from SSE-style behaviour without
	// changing their call patterns.
	m := make([]oa.ChatCompletionMessage, 0, len(messages))
	for _, x := range messages {
		m = append(m, oa.ChatCompletionMessage{Role: x.Role, Content: x.Content})
	}
	req := oa.ChatCompletionRequest{Model: model, Messages: m}
	if temperature > 0 {
		req.Temperature = float32(temperature)
	}
	if topP > 0 {
		req.TopP = float32(topP)
	}
	if maxTokens > 0 {
		req.MaxTokens = maxTokens
	}
	if jsonMode {
		rf := oa.ChatCompletionResponseFormat{Type: oa.ChatCompletionResponseFormatTypeJSONObject}
		req.ResponseFormat = &rf
	}
	// Ask backend to include usage in the final stream chunk if supported.
	req.StreamOptions = &oa.StreamOptions{IncludeUsage: true}

	stream, err := c.cli.CreateChatCompletionStream(ctx, req)
	if err != nil {
		return "", 0, 0, 0, err
	}
	defer stream.Close()

	var builder strings.Builder
	var usage *oa.Usage

	for {
		resp, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", 0, 0, 0, err
		}
		if len(resp.Choices) > 0 {
			delta := resp.Choices[0].Delta.Content
			if delta != "" {
				builder.WriteString(delta)
				if onDelta != nil {
					if e := onDelta(delta); e != nil {
						return "", 0, 0, 0, e
					}
				}
			}
		}
		// When StreamOptions.IncludeUsage is true, the last chunk will
		// carry the aggregated usage for the whole request.
		if resp.Usage != nil {
			usage = resp.Usage
		}
	}

	out := builder.String()
	if strings.TrimSpace(out) == "" {
		return "", 0, 0, 0, fmt.Errorf("chat completion returned empty content")
	}

	pt, ct, tt := 0, 0, 0
	if usage != nil {
		pt = usage.PromptTokens
		ct = usage.CompletionTokens
		tt = usage.TotalTokens
	}
	return out, pt, ct, tt, nil
}

// ChatCompletionsStream streams deltas via callback; callback receives partial text chunks.
func (c *Client) ChatCompletionsStream(ctx context.Context, model string, messages []ChatMessage, temperature float64, onDelta func(string) error) error {
	m := make([]oa.ChatCompletionMessage, 0, len(messages))
	for _, x := range messages {
		m = append(m, oa.ChatCompletionMessage{Role: x.Role, Content: x.Content})
	}
	req := oa.ChatCompletionRequest{Model: model, Messages: m, Temperature: float32(temperature), Stream: true}
	stream, err := c.cli.CreateChatCompletionStream(ctx, req)
	if err != nil {
		return err
	}
	defer stream.Close()
	for {
		resp, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		if len(resp.Choices) > 0 {
			delta := resp.Choices[0].Delta.Content
			if delta != "" && onDelta != nil {
				if e := onDelta(delta); e != nil {
					return e
				}
			}
		}
	}
}

// CreateEmbeddings returns vectors for each input
func (c *Client) CreateEmbeddings(ctx context.Context, model string, inputs []string) ([][]float32, error) {
	if len(inputs) == 0 {
		return [][]float32{}, nil
	}
	req := oa.EmbeddingRequest{Model: oa.EmbeddingModel(model), Input: inputs}
	resp, err := c.cli.CreateEmbeddings(ctx, req)
	if err != nil {
		return nil, err
	}
	out := make([][]float32, 0, len(resp.Data))
	for _, d := range resp.Data {
		out = append(out, d.Embedding)
	}
	return out, nil
}

// CreateImageBase64 generates images (b64 JSON) and returns decoded bytes per image
func (c *Client) CreateImageBase64(ctx context.Context, model, prompt, size string, n int) ([][]byte, error) {
	if n <= 0 {
		n = 1
	}
	// go-openai Images API ignores model for legacy endpoints; set if supported by backend
	req := oa.ImageRequest{Prompt: prompt, N: n, Size: size, ResponseFormat: oa.CreateImageResponseFormatB64JSON}
	resp, err := c.cli.CreateImage(ctx, req)
	if err != nil {
		return nil, err
	}
	out := make([][]byte, 0, len(resp.Data))
	for _, d := range resp.Data {
		if b, e := base64.StdEncoding.DecodeString(d.B64JSON); e == nil {
			out = append(out, b)
		}
	}
	return out, nil
}
