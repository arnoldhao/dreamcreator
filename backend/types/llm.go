package types

import "time"

// Unified LLM provider types

// ProviderType 标识底层协议兼容性或访问方式
// 保持与后端实现解耦，例如 openai_compat、remote 等
type ProviderType string

const (
    ProviderOpenAICompat   ProviderType = "openai_compat"
    ProviderAnthropicCompat ProviderType = "anthropic_compat"
    ProviderRemote         ProviderType = "remote"
)

// ProviderPolicy 仅用于管理/展示策略，不影响协议
// - preset_show: 默认展示，不能删除，名称不可编辑
// - preset_hidden: 默认不展示，删除时应当重置数据（清 api_key/models），保留条目
// - custom: 用户自建，默认不展示，允许删除，名称可编辑
type ProviderPolicy string

const (
    PolicyCustom       ProviderPolicy = "custom"
    PolicyPresetShow   ProviderPolicy = "preset_show"
    PolicyPresetHidden ProviderPolicy = "preset_hidden"
)

type RateLimit struct {
    RPM         int `json:"rpm"`
    RPS         int `json:"rps"`
    Burst       int `json:"burst"`
    Concurrency int `json:"concurrency"`
}

type Provider struct {
    ID        string         `json:"id"`
    Type      ProviderType   `json:"type"`
    Policy    ProviderPolicy `json:"policy"`
    Platform  string         `json:"platform"`       // generic | aws_bedrock | gcp_vertex | aliyun | azure_openai | ...
    Name      string         `json:"name"`
    BaseURL   string         `json:"base_url"`
    Region    string         `json:"region"`          // e.g., aws/aliyun region or logical area
    APIKey    string         `json:"api_key"`
    // GCP Vertex AI
    ProjectID      string `json:"project_id"`
    SAEmail        string `json:"sa_email"`
    SAPrivateKey   string `json:"sa_private_key"`
    Models    []string       `json:"models"`
    RateLimit RateLimit      `json:"rate_limit"`
    Enabled   bool           `json:"enabled"`
    // Reserved/extended fields
    AuthMethod       string         `json:"auth_method"`        // "api" (default), "oauth" (reserved for Anthropic etc.)
    APIVersion       string         `json:"api_version"`        // e.g., Azure API version
    InferenceSummary bool           `json:"inference_summary"`  // OpenAI inference-summary header toggle
    APIUsage         map[string]any `json:"api_usage"`          // reserved: usage/account info (e.g., OpenRouter)
    CreatedAt time.Time      `json:"created_at"`
    UpdatedAt time.Time      `json:"updated_at"`
}

// LLMProfile 统一的 LLM 配置文件类型
type LLMProfile struct {
    ID           string            `json:"id"`
    ProviderID   string            `json:"provider_id"`
    Model        string            `json:"model"`
    Temperature  float64           `json:"temperature"`
    TopP         float64           `json:"top_p"`
    JSONMode     bool              `json:"json_mode"`
    SysPromptTpl string            `json:"sys_prompt_tpl"`
    CostWeight   float64           `json:"cost_weight"`
    MaxTokens    int               `json:"max_tokens"`
    Metadata     map[string]string `json:"metadata"`
    CreatedAt    time.Time         `json:"created_at"`
    UpdatedAt    time.Time         `json:"updated_at"`
}

// --- Models meta (optional richer info than plain names) ---

// ModelKind 对应模型使用类型（UI 中的“类型”）
// chat: 对话/文本；image_gen: 文生图；embedding: 向量；audio/video: 语音/视频相关
type ModelKind string

const (
    ModelKindChat      ModelKind = "chat"
    ModelKindImageGen  ModelKind = "image_gen"
    ModelKindEmbedding ModelKind = "embedding"
    ModelKindAudio     ModelKind = "audio"
    ModelKindVideo     ModelKind = "video"
)

type ModelCapabilities struct {
    ImageInput  bool `json:"image_input"`
    ImageOutput bool `json:"image_output"`
    AudioInput  bool `json:"audio_input"`
    VideoInput  bool `json:"video_input"`
}

type ModelReasoning struct {
    IsReasoningModel   bool    `json:"is_reasoning_model"`
    ReasoningStrength  float64 `json:"reasoning_strength"`
    CanDisableReasoning bool   `json:"can_disable_reasoning"`
}

// ModelInfo 为单个模型的扩展信息（可选，来自供应商说明/探测/手动配置）
type ModelInfo struct {
    Name            string             `json:"name"`      // 机器名（如 gpt-4o-mini）
    Nickname        string             `json:"nickname"`  // 展示名（可选）
    Kind            ModelKind          `json:"kind"`
    ContextWindow   int                `json:"context_window"`
    MaxOutputTokens int                `json:"max_output_tokens"`
    Capabilities    ModelCapabilities  `json:"capabilities"`
    Reasoning       ModelReasoning     `json:"reasoning"`
    Extra           map[string]any     `json:"extra"`     // 供应商特有信息（价格/并发等）
}

// GlobalProfile is a global, model-agnostic template of LLM parameters.
// It is not bound to any provider or model; callers supply provider+model at use time.
type GlobalProfile struct {
    ID           string            `json:"id"`
    Name         string            `json:"name"`
    Temperature  float64           `json:"temperature"`
    TopP         float64           `json:"top_p"`
    JSONMode     bool              `json:"json_mode"`
    SysPromptTpl string            `json:"sys_prompt_tpl"`
    MaxTokens    int               `json:"max_tokens"`
	Metadata     map[string]string `json:"metadata"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

// --- Generic LLM conversations (for provider chats) ---

// LLMChatRole 标识对话参与方：应用侧或 LLM 提供方
type LLMChatRole string

const (
	LLMChatRoleApp      LLMChatRole = "app"
	LLMChatRoleProvider LLMChatRole = "provider"
)

// LLMConversationStatus 为会话整体状态
type LLMConversationStatus string

const (
	LLMConversationStatusRunning  LLMConversationStatus = "running"
	LLMConversationStatusFinished LLMConversationStatus = "finished"
	LLMConversationStatusFailed   LLMConversationStatus = "failed"
)

// LLMChatMessage 表示一次消息（如一次请求、一次回复或应用侧处理说明）
type LLMChatMessage struct {
	ID        string         `json:"id"`
	Role      LLMChatRole    `json:"role"`                // app | provider
	Kind      string         `json:"kind,omitempty"`      // request | response | meta | error
	Content   string         `json:"content"`             // 原始文本内容
	CreatedAt int64          `json:"created_at"`          // Unix 秒
	Metadata  map[string]any `json:"metadata,omitempty"`  // 轻量元信息：stage/batch/tokens 等
}

// LLMConversation 表示一次完整的 LLM 会话（例如一次字幕翻译任务）
type LLMConversation struct {
	ID         string                `json:"id"`                     // 通常复用任务 ID
	ProjectID  string                `json:"project_id,omitempty"`   // 所属字幕项目
	Language   string                `json:"language,omitempty"`     // 目标语言（如 zh-Hans）
	TaskID     string                `json:"task_id,omitempty"`      // 对应的 ConversionTask.ID
	Provider   string                `json:"provider,omitempty"`     // Provider 展示名
	ProviderID string                `json:"provider_id,omitempty"`  // Provider 记录 ID
	Model      string                `json:"model,omitempty"`        // 模型名
	Status     LLMConversationStatus `json:"status"`                 // running | finished | failed
	StartedAt  int64                 `json:"started_at"`             // 会话开始时间
	EndedAt    int64                 `json:"ended_at,omitempty"`     // 结束时间
	Messages   []LLMChatMessage      `json:"messages,omitempty"`     // 消息时间线
}
