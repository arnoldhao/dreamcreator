package provider

// TokenUsage represents model-reported token accounting for one request.
// Fields are best-effort and may be zero when the backend does not return usage.
type TokenUsage struct {
    PromptTokens     int `json:"prompt_tokens"`
    CompletionTokens int `json:"completion_tokens"`
    TotalTokens      int `json:"total_tokens"`
}

