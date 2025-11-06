package openai

import (
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "net/url"
    "strings"
)

// Minimal OpenAI-compatible client: supports baseURL + apiKey; ListModels for connectivity test.

type Client struct {
    baseURL string
    apiKey  string
    httpc   *http.Client
}

func NewClient(baseURL, apiKey string, httpc *http.Client) *Client {
    return &Client{baseURL: strings.TrimRight(baseURL, "/"), apiKey: apiKey, httpc: httpc}
}

func (c *Client) buildModelsURL() (string, error) {
    u, err := url.Parse(c.baseURL)
    if err != nil { return "", err }
    basePath := strings.TrimRight(u.Path, "/")
    if strings.HasSuffix(strings.ToLower(basePath), "/v1") {
        u.Path = basePath + "/models"
    } else {
        if basePath == "" { u.Path = "/v1/models" } else { u.Path = basePath + "/v1/models" }
    }
    return u.String(), nil
}

type listModelsResp struct {
    Data []struct{ ID string `json:"id"` } `json:"data"`
}

func (c *Client) ListModels(ctx context.Context) ([]string, error) {
    u, err := c.buildModelsURL()
    if err != nil { return nil, err }
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
    if err != nil { return nil, err }
    if c.apiKey != "" {
        req.Header.Set("Authorization", "Bearer "+c.apiKey)
    }
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Accept", "application/json")
    req.Header.Set("User-Agent", "DreamCreator-LLM/1.0")
    // Timeout safety: rely on provided http.Client timeout/transport
    resp, err := c.httpc.Do(req)
    if err != nil { return nil, err }
    defer resp.Body.Close()
    if resp.StatusCode/100 != 2 {
        b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
        return nil, fmt.Errorf("http %d: %s", resp.StatusCode, string(b))
    }
    var out listModelsResp
    dec := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)) // 1MB
    if err := dec.Decode(&out); err != nil {
        return nil, err
    }
    models := make([]string, 0, len(out.Data))
    for _, d := range out.Data { models = append(models, d.ID) }
    // Fallback: if empty list but 200 OK, return an example to pass connectivity
    if len(models) == 0 {
        models = []string{"default"}
    }
    return models, nil
}

// Optional ping (some providers don't allow /models): try GET baseURL or /v1
func (c *Client) Ping(ctx context.Context) error {
    u := c.baseURL
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
    if err != nil { return err }
    req.Header.Set("User-Agent", "DreamCreator-LLM/1.0")
    resp, err := c.httpc.Do(req)
    if err != nil { return err }
    defer resp.Body.Close()
    if resp.StatusCode/100 != 2 && resp.StatusCode != 404 { // 404 acceptable for ping
        return fmt.Errorf("ping http %d", resp.StatusCode)
    }
    return nil
}
