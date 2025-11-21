package anthropic

import (
    "context"
    "errors"
    "net/http"
    "strings"

    an "github.com/liushuangls/go-anthropic/v2"
)

// Community SDK-backed client (go-anthropic v2)
type Client struct {
    baseURL string
    apiKey  string
    httpc   *http.Client
    cli     *an.Client
}

type ContentBlock struct { Type string; Text string }
type Message struct { Role string; Content []ContentBlock }

func NewClient(baseURL, apiKey string, httpc *http.Client) *Client {
    // Respect custom BaseURL and HTTP client (proxy) via SDK options
    opts := make([]an.ClientOption, 0, 2)
    if strings.TrimSpace(baseURL) != "" {
        opts = append(opts, an.WithBaseURL(strings.TrimRight(baseURL, "/")))
    }
    if httpc != nil {
        opts = append(opts, an.WithHTTPClient(httpc))
    }
    cli := an.NewClient(apiKey, opts...)
    return &Client{baseURL: baseURL, apiKey: apiKey, httpc: httpc, cli: cli}
}

// ChatMessages sends a non-streaming Messages request and returns first text block.
// maxTokens: if <=0, defaults to 1024.
func (c *Client) ChatMessages(ctx context.Context, model, system string, msgs []Message, temperature float64, maxTokens int) (string, error) {
    // Map wrapper to SDK messages; join blocks as plain text per message
    sm := make([]an.Message, 0, len(msgs))
    for _, m := range msgs {
        txt := ""
        for _, b := range m.Content { if b.Text != "" { if txt == "" { txt = b.Text } else { txt += "\n" + b.Text } } }
        if txt == "" { continue }
        if m.Role == "assistant" {
            sm = append(sm, an.NewAssistantTextMessage(txt))
        } else {
            sm = append(sm, an.NewUserTextMessage(txt))
        }
    }
    if maxTokens <= 0 { maxTokens = 1024 }
    req := an.MessagesRequest{ Model: an.Model(model), Messages: sm, MaxTokens: maxTokens }
    if system != "" { req.System = system }
    if temperature > 0 { t := float32(temperature); req.Temperature = &t }
    resp, err := c.cli.CreateMessages(ctx, req)
    if err != nil { return "", err }
    // Use SDK helper to safely read first text content
    return resp.GetFirstContentText(), nil
}

// --- Extended client surface to align with provider-level capabilities ---

// ListModels returns a static set of common Anthropic models (API does not expose a models endpoint).
func (c *Client) ListModels(ctx context.Context) ([]string, error) {
    return []string{
        "claude-3-5-sonnet-20241022",
        "claude-3-5-haiku-20241022",
        "claude-3-opus-20240229",
        "claude-3-sonnet-20240229",
        "claude-3-haiku-20240307",
    }, nil
}

// ChatMessagesStream: streaming not implemented in this wrapper. Use non-streaming ChatMessages or extend if needed.
func (c *Client) ChatMessagesStream(ctx context.Context, model, system string, msgs []Message, temperature float64, onDelta func(string) error) error {
    return errors.New("anthropic streaming not implemented")
}

// CreateEmbeddings: Anthropic currently does not expose embeddings API.
func (c *Client) CreateEmbeddings(ctx context.Context, model string, inputs []string) ([][]float32, error) {
    return nil, errors.New("anthropic embeddings not supported")
}

// CreateImageBase64: Anthropic does not provide image generation in Messages API.
func (c *Client) CreateImageBase64(ctx context.Context, model, prompt, size string, n int) ([][]byte, error) {
    return nil, errors.New("anthropic images not supported")
}
// ChatMessagesWithUsage returns content and token usage if available.
func (c *Client) ChatMessagesWithUsage(ctx context.Context, model, system string, msgs []Message, temperature float64, maxTokens int) (string, int, int, int, error) {
    sm := make([]an.Message, 0, len(msgs))
    for _, m := range msgs {
        txt := ""
        for _, b := range m.Content { if b.Text != "" { if txt == "" { txt = b.Text } else { txt += "\n" + b.Text } } }
        if txt == "" { continue }
        if m.Role == "assistant" {
            sm = append(sm, an.NewAssistantTextMessage(txt))
        } else {
            sm = append(sm, an.NewUserTextMessage(txt))
        }
    }
    if maxTokens <= 0 { maxTokens = 1024 }
    req := an.MessagesRequest{ Model: an.Model(model), Messages: sm, MaxTokens: maxTokens }
    if system != "" { req.System = system }
    if temperature > 0 { t := float32(temperature); req.Temperature = &t }
    resp, err := c.cli.CreateMessages(ctx, req)
    if err != nil { return "", 0, 0, 0, err }
    out := resp.GetFirstContentText()
    // SDK v2 exposes Usage as a struct with InputTokens/OutputTokens
    pt, ct := resp.Usage.InputTokens, resp.Usage.OutputTokens
    tt := pt + ct
    return out, pt, ct, tt, nil
}
