package types

// TargetLanguage represents a user-managed target language option for AI translation
// Code should be a BCP-47 or short code (e.g., "en", "zh-CN"). Name is a friendly label.
type TargetLanguage struct {
    Code      string `json:"code"`
    Name      string `json:"name"`
    CreatedAt int64  `json:"created_at"`
    UpdatedAt int64  `json:"updated_at"`
}

