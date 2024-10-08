package types

type LLM struct {
	Num       int      `json:"num" yaml:"num"`             // show on main page
	Name      string   `json:"name" yaml:"name"`           // platform name: ollama, openai, deepseek etc.
	Region    string   `json:"region" yaml:"region"`       // ai region: Local, China, ExceptChina
	BaseURL   string   `json:"baseURL" yaml:"baseURL"`     // ai platform url: 127.0.0.1:11434, api.openai.com
	APIKey    string   `json:"APIKey" yaml:"APIKey"`       // api token
	Available bool     `json:"available" yaml:"available"` // llm available or not
	Icon      string   `json:"icon" yaml:"icon"`           // platform icon
	Show      bool     `json:"show" yaml:"show"`           // show on main page
	Models    []*Model `json:"models" yaml:"models"`       // all models
}

type Model struct {
	Num         int    `json:"num" yaml:"num"`                 // show on main page
	Name        string `json:"name" yaml:"name"`               // model name
	Available   bool   `json:"available" yaml:"available"`     // model available or not
	Description string `json:"description" yaml:"description"` // model description
}

type CurrentModel struct {
	Name  string `json:"name" yaml:"name"`   // current llm name
	Model string `json:"model" yaml:"model"` // current llm model
}
