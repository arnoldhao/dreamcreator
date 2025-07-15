package zhconvert

import "time"

// ConvertRequest API请求结构
type ConvertRequest struct {
	Text      string `json:"text"`
	Converter string `json:"converter"`
}

// ConvertResponse API响应结构
type ConvertResponse struct {
	Data struct {
		Converter    string   `json:"converter"`
		Text         string   `json:"text"`
		Diff         string   `json:"diff"`
		UseModules   []string `json:"usedModules"`
		JPTextStyles []string `json:"jpTextStyles"`
		TextFormat   string   `json:"textFormat"`
	} `json:"data"`
}

// Config 配置结构
type Config struct {
	APIBaseURL string        `json:"api_base_url"`
	Timeout    time.Duration `json:"timeout"`
	RetryCount int           `json:"retry_count"`
}

// DefaultConfig 默认配置
func DefaultConfig() *Config {
	return &Config{
		APIBaseURL: ZHCONVERT_API_URL,
		Timeout:    30 * time.Second,
		RetryCount: 3,
	}
}
