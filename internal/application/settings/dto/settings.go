package dto

type WindowBounds struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

type Settings struct {
	Appearance            string          `json:"appearance"`
	EffectiveAppearance   string          `json:"effectiveAppearance"`
	FontFamily            string          `json:"fontFamily"`
	FontSize              int             `json:"fontSize"`
	ThemeColor            string          `json:"themeColor"`
	ColorScheme           string          `json:"colorScheme"`
	SystemThemeColor      string          `json:"systemThemeColor"`
	Language              string          `json:"language"`
	DownloadDirectory     string          `json:"downloadDirectory"`
	MainBounds            WindowBounds    `json:"mainBounds"`
	SettingsBounds        WindowBounds    `json:"settingsBounds"`
	Version               int             `json:"version"`
	LogLevel              string          `json:"logLevel"`
	LogMaxSizeMB          int             `json:"logMaxSizeMB"`
	LogMaxBackups         int             `json:"logMaxBackups"`
	LogMaxAgeDays         int             `json:"logMaxAgeDays"`
	LogCompress           bool            `json:"logCompress"`
	MenuBarVisibility     string          `json:"menuBarVisibility"`
	AutoStart             bool            `json:"autoStart"`
	MinimizeToTrayOnStart bool            `json:"minimizeToTrayOnStart"`
	AgentModelProviderID  string          `json:"agentModelProviderId"`
	AgentModelName        string          `json:"agentModelName"`
	AgentStreamEnabled    bool            `json:"agentStreamEnabled"`
	ChatTemperature       float32         `json:"chatTemperature"`
	ChatMaxTokens         int             `json:"chatMaxTokens"`
	Proxy                 Proxy           `json:"proxy"`
	Gateway               GatewaySettings `json:"gateway"`
	Memory                MemorySettings  `json:"memory"`
	Tools                 map[string]any  `json:"tools,omitempty"`
	Skills                map[string]any  `json:"skills,omitempty"`
	Commands              map[string]bool `json:"commands,omitempty"`
	Channels              map[string]any  `json:"channels"`
}

type UpdateSettingsRequest struct {
	Appearance            *string                       `json:"appearance"`
	FontFamily            *string                       `json:"fontFamily"`
	FontSize              *int                          `json:"fontSize"`
	ThemeColor            *string                       `json:"themeColor"`
	ColorScheme           *string                       `json:"colorScheme"`
	Language              *string                       `json:"language"`
	DownloadDirectory     *string                       `json:"downloadDirectory"`
	MainBounds            *WindowBounds                 `json:"mainBounds"`
	SettingsBounds        *WindowBounds                 `json:"settingsBounds"`
	LogLevel              *string                       `json:"logLevel"`
	LogMaxSizeMB          *int                          `json:"logMaxSizeMB"`
	LogMaxBackups         *int                          `json:"logMaxBackups"`
	LogMaxAgeDays         *int                          `json:"logMaxAgeDays"`
	LogCompress           *bool                         `json:"logCompress"`
	MenuBarVisibility     *string                       `json:"menuBarVisibility"`
	AutoStart             *bool                         `json:"autoStart"`
	MinimizeToTrayOnStart *bool                         `json:"minimizeToTrayOnStart"`
	AgentModelProviderID  *string                       `json:"agentModelProviderId"`
	AgentModelName        *string                       `json:"agentModelName"`
	AgentStreamEnabled    *bool                         `json:"agentStreamEnabled"`
	ChatTemperature       *float32                      `json:"chatTemperature"`
	ChatMaxTokens         *int                          `json:"chatMaxTokens"`
	Proxy                 *Proxy                        `json:"proxy"`
	Gateway               *UpdateGatewaySettingsRequest `json:"gateway"`
	Memory                *UpdateMemorySettingsRequest  `json:"memory,omitempty"`
	Tools                 map[string]any                `json:"tools,omitempty"`
	Skills                map[string]any                `json:"skills,omitempty"`
	Commands              map[string]bool               `json:"commands,omitempty"`
	Channels              map[string]any                `json:"channels,omitempty"`
}

type MemorySettings struct {
	Enabled           bool    `json:"enabled"`
	EmbeddingProvider string  `json:"embeddingProviderId"`
	EmbeddingModel    string  `json:"embeddingModel"`
	LLMProvider       string  `json:"llmProviderId"`
	LLMModel          string  `json:"llmModel"`
	RecallTopK        int     `json:"recallTopK"`
	VectorWeight      float64 `json:"vectorWeight"`
	TextWeight        float64 `json:"textWeight"`
	RecencyWeight     float64 `json:"recencyWeight"`
	RecencyHalfLife   float64 `json:"recencyHalfLifeDays"`
	MinScore          float64 `json:"minScore"`
	AutoRecall        bool    `json:"autoRecall"`
	AutoCapture       bool    `json:"autoCapture"`
	SessionLifecycle  bool    `json:"sessionLifecycle"`
	CaptureMaxEntries int     `json:"captureMaxEntries"`
}

type UpdateMemorySettingsRequest struct {
	Enabled           *bool    `json:"enabled,omitempty"`
	EmbeddingProvider *string  `json:"embeddingProviderId,omitempty"`
	EmbeddingModel    *string  `json:"embeddingModel,omitempty"`
	LLMProvider       *string  `json:"llmProviderId,omitempty"`
	LLMModel          *string  `json:"llmModel,omitempty"`
	RecallTopK        *int     `json:"recallTopK,omitempty"`
	VectorWeight      *float64 `json:"vectorWeight,omitempty"`
	TextWeight        *float64 `json:"textWeight,omitempty"`
	RecencyWeight     *float64 `json:"recencyWeight,omitempty"`
	RecencyHalfLife   *float64 `json:"recencyHalfLifeDays,omitempty"`
	MinScore          *float64 `json:"minScore,omitempty"`
	AutoRecall        *bool    `json:"autoRecall,omitempty"`
	AutoCapture       *bool    `json:"autoCapture,omitempty"`
	SessionLifecycle  *bool    `json:"sessionLifecycle,omitempty"`
	CaptureMaxEntries *int     `json:"captureMaxEntries,omitempty"`
}

type GatewaySettings struct {
	ControlPlaneEnabled       bool                     `json:"controlPlaneEnabled"`
	VoiceEnabled              bool                     `json:"voiceEnabled"`
	VoiceWakeEnabled          bool                     `json:"voiceWakeEnabled"`
	SandboxEnabled            bool                     `json:"sandboxEnabled"`
	HTTP                      GatewayHTTPSettings      `json:"http"`
	ChannelHealthCheckMinutes int                      `json:"channelHealthCheckMinutes"`
	Runtime                   GatewayRuntimeSettings   `json:"runtime"`
	Queue                     GatewayQueueSettings     `json:"queue"`
	Heartbeat                 GatewayHeartbeatSettings `json:"heartbeat"`
	Subagents                 GatewaySubagentSettings  `json:"subagents"`
	Cron                      GatewayCronSettings      `json:"cron"`
}

type GatewayRuntimeSettings struct {
	MaxSteps          int                          `json:"maxSteps"`
	RecordPrompt      bool                         `json:"recordPrompt"`
	ToolLoopDetection GatewayToolLoopSettings      `json:"toolLoopDetection"`
	ContextWindow     GatewayContextWindowSettings `json:"contextWindow"`
	Compaction        GatewayCompactionSettings    `json:"compaction"`
}

type GatewayToolLoopSettings struct {
	Enabled                       bool                     `json:"enabled"`
	WarnThreshold                 int                      `json:"warnThreshold"`
	CriticalThreshold             int                      `json:"criticalThreshold"`
	GlobalCircuitBreakerThreshold int                      `json:"globalCircuitBreakerThreshold"`
	HistorySize                   int                      `json:"historySize"`
	Detectors                     GatewayToolLoopDetectors `json:"detectors"`
	AbortThreshold                int                      `json:"abortThreshold"`
	WindowSize                    int                      `json:"windowSize"`
}

type GatewayToolLoopDetectors struct {
	GenericRepeat       bool `json:"genericRepeat"`
	KnownPollNoProgress bool `json:"knownPollNoProgress"`
	PingPong            bool `json:"pingPong"`
}

type GatewayContextWindowSettings struct {
	WarnTokens int `json:"warnTokens"`
	HardTokens int `json:"hardTokens"`
}

type GatewayCompactionSettings struct {
	Mode               string                               `json:"mode"`
	ReserveTokens      int                                  `json:"reserveTokens"`
	KeepRecentTokens   int                                  `json:"keepRecentTokens"`
	ReserveTokensFloor int                                  `json:"reserveTokensFloor"`
	MaxHistoryShare    float64                              `json:"maxHistoryShare"`
	MemoryFlush        GatewayCompactionMemoryFlushSettings `json:"memoryFlush"`
}

type GatewayCompactionMemoryFlushSettings struct {
	Enabled             bool   `json:"enabled"`
	SoftThresholdTokens int    `json:"softThresholdTokens"`
	Prompt              string `json:"prompt"`
	SystemPrompt        string `json:"systemPrompt"`
}

type GatewayQueueSettings struct {
	GlobalConcurrency  int                      `json:"globalConcurrency"`
	SessionConcurrency int                      `json:"sessionConcurrency"`
	Lanes              GatewayQueueLaneSettings `json:"lanes"`
}

type GatewayQueueLaneSettings struct {
	Main     int `json:"main"`
	Subagent int `json:"subagent"`
	Cron     int `json:"cron"`
}

type GatewayHeartbeatSettings struct {
	Enabled                   bool                             `json:"enabled"`
	EveryMinutes              int                              `json:"everyMinutes"`
	Every                     string                           `json:"every"`
	Target                    string                           `json:"target"`
	To                        string                           `json:"to"`
	AccountID                 string                           `json:"accountId"`
	Model                     string                           `json:"model"`
	Session                   string                           `json:"session"`
	Prompt                    string                           `json:"prompt"`
	IncludeReasoning          bool                             `json:"includeReasoning"`
	SuppressToolErrorWarnings bool                             `json:"suppressToolErrorWarnings"`
	ActiveHours               GatewayHeartbeatActiveHours      `json:"activeHours"`
	Checklist                 GatewayHeartbeatChecklist        `json:"checklist"`
	RunSession                string                           `json:"runSession"`
	PromptAppend              string                           `json:"promptAppend"`
	Periodic                  GatewayHeartbeatPeriodicSettings `json:"periodic"`
	Delivery                  GatewayHeartbeatDeliverySettings `json:"delivery"`
	Events                    GatewayHeartbeatEventSettings    `json:"events"`
}

type GatewayHeartbeatActiveHours struct {
	Start    string `json:"start"`
	End      string `json:"end"`
	Timezone string `json:"timezone"`
}

type GatewayHeartbeatChecklist struct {
	Title     string                          `json:"title"`
	Items     []GatewayHeartbeatChecklistItem `json:"items"`
	Notes     string                          `json:"notes"`
	Version   int                             `json:"version"`
	UpdatedAt string                          `json:"updatedAt"`
}

type GatewayHeartbeatChecklistItem struct {
	ID       string `json:"id"`
	Text     string `json:"text"`
	Done     bool   `json:"done"`
	Priority string `json:"priority"`
}

type GatewayHeartbeatPeriodicSettings struct {
	Enabled bool   `json:"enabled"`
	Every   string `json:"every"`
}

type GatewayHeartbeatDeliverySettings struct {
	Periodic        GatewayHeartbeatSurfacePolicy `json:"periodic"`
	EventDriven     GatewayHeartbeatSurfacePolicy `json:"eventDriven"`
	ThreadReplyMode string                        `json:"threadReplyMode"`
}

type GatewayHeartbeatSurfacePolicy struct {
	Center           bool   `json:"center"`
	PopupMinSeverity string `json:"popupMinSeverity"`
	ToastMinSeverity string `json:"toastMinSeverity"`
	OSMinSeverity    string `json:"osMinSeverity"`
}

type GatewayHeartbeatEventSettings struct {
	CronWakeMode     string `json:"cronWakeMode"`
	ExecWakeMode     string `json:"execWakeMode"`
	SubagentWakeMode string `json:"subagentWakeMode"`
}

type GatewaySubagentSettings struct {
	MaxDepth      int                       `json:"maxDepth"`
	MaxChildren   int                       `json:"maxChildren"`
	MaxConcurrent int                       `json:"maxConcurrent"`
	Model         string                    `json:"model"`
	Thinking      string                    `json:"thinking"`
	Tools         GatewaySubagentToolPolicy `json:"tools"`
}

type GatewaySubagentToolPolicy struct {
	Allow     []string `json:"allow"`
	AlsoAllow []string `json:"alsoAllow"`
	Deny      []string `json:"deny"`
}

type GatewayCronSettings struct {
	Enabled           bool                     `json:"enabled"`
	MaxConcurrentRuns int                      `json:"maxConcurrentRuns"`
	SessionRetention  string                   `json:"sessionRetention"`
	RunLog            GatewayCronRunLogSetting `json:"runLog"`
}

type GatewayCronRunLogSetting struct {
	MaxBytes  string `json:"maxBytes"`
	KeepLines int    `json:"keepLines"`
}

type GatewayHTTPSettings struct {
	Endpoints GatewayHTTPEndpointsSettings `json:"endpoints"`
}

type GatewayHTTPEndpointsSettings struct {
	ChatCompletions GatewayHTTPChatCompletionsSettings `json:"chatCompletions"`
	Responses       GatewayHTTPResponsesSettings       `json:"responses"`
}

type GatewayHTTPChatCompletionsSettings struct {
	Enabled bool `json:"enabled"`
}

type GatewayHTTPResponsesSettings struct {
	Enabled      bool                               `json:"enabled"`
	MaxBodyBytes int                                `json:"maxBodyBytes"`
	MaxURLParts  int                                `json:"maxUrlParts"`
	Files        GatewayHTTPResponsesFilesSettings  `json:"files"`
	Images       GatewayHTTPResponsesImagesSettings `json:"images"`
}

type GatewayHTTPResponsesFilesSettings struct {
	AllowURL     bool                            `json:"allowUrl"`
	URLAllowlist []string                        `json:"urlAllowlist"`
	AllowedMimes []string                        `json:"allowedMimes"`
	MaxBytes     int                             `json:"maxBytes"`
	MaxChars     int                             `json:"maxChars"`
	MaxRedirects int                             `json:"maxRedirects"`
	TimeoutMs    int                             `json:"timeoutMs"`
	PDF          GatewayHTTPResponsesPDFSettings `json:"pdf"`
}

type GatewayHTTPResponsesPDFSettings struct {
	MaxPages     int `json:"maxPages"`
	MaxPixels    int `json:"maxPixels"`
	MinTextChars int `json:"minTextChars"`
}

type GatewayHTTPResponsesImagesSettings struct {
	AllowURL     bool     `json:"allowUrl"`
	URLAllowlist []string `json:"urlAllowlist"`
	AllowedMimes []string `json:"allowedMimes"`
	MaxBytes     int      `json:"maxBytes"`
	MaxRedirects int      `json:"maxRedirects"`
	TimeoutMs    int      `json:"timeoutMs"`
}

type UpdateGatewaySettingsRequest struct {
	ControlPlaneEnabled       *bool                                  `json:"controlPlaneEnabled,omitempty"`
	VoiceEnabled              *bool                                  `json:"voiceEnabled,omitempty"`
	VoiceWakeEnabled          *bool                                  `json:"voiceWakeEnabled,omitempty"`
	SandboxEnabled            *bool                                  `json:"sandboxEnabled,omitempty"`
	HTTP                      *UpdateGatewayHTTPSettingsRequest      `json:"http,omitempty"`
	ChannelHealthCheckMinutes *int                                   `json:"channelHealthCheckMinutes,omitempty"`
	Runtime                   *UpdateGatewayRuntimeSettingsRequest   `json:"runtime,omitempty"`
	Queue                     *UpdateGatewayQueueSettingsRequest     `json:"queue,omitempty"`
	Heartbeat                 *UpdateGatewayHeartbeatSettingsRequest `json:"heartbeat,omitempty"`
	Subagents                 *UpdateGatewaySubagentSettingsRequest  `json:"subagents,omitempty"`
	Cron                      *UpdateGatewayCronSettingsRequest      `json:"cron,omitempty"`
}

type UpdateGatewayRuntimeSettingsRequest struct {
	MaxSteps          *int                                       `json:"maxSteps,omitempty"`
	RecordPrompt      *bool                                      `json:"recordPrompt,omitempty"`
	ToolLoopDetection *UpdateGatewayToolLoopSettingsRequest      `json:"toolLoopDetection,omitempty"`
	ContextWindow     *UpdateGatewayContextWindowSettingsRequest `json:"contextWindow,omitempty"`
	Compaction        *UpdateGatewayCompactionSettingsRequest    `json:"compaction,omitempty"`
}

type UpdateGatewayToolLoopSettingsRequest struct {
	Enabled                       *bool                                  `json:"enabled,omitempty"`
	WarnThreshold                 *int                                   `json:"warnThreshold,omitempty"`
	CriticalThreshold             *int                                   `json:"criticalThreshold,omitempty"`
	GlobalCircuitBreakerThreshold *int                                   `json:"globalCircuitBreakerThreshold,omitempty"`
	HistorySize                   *int                                   `json:"historySize,omitempty"`
	Detectors                     *UpdateGatewayToolLoopDetectorsRequest `json:"detectors,omitempty"`
	AbortThreshold                *int                                   `json:"abortThreshold,omitempty"`
	WindowSize                    *int                                   `json:"windowSize,omitempty"`
}

type UpdateGatewayToolLoopDetectorsRequest struct {
	GenericRepeat       *bool `json:"genericRepeat,omitempty"`
	KnownPollNoProgress *bool `json:"knownPollNoProgress,omitempty"`
	PingPong            *bool `json:"pingPong,omitempty"`
}

type UpdateGatewayContextWindowSettingsRequest struct {
	WarnTokens *int `json:"warnTokens,omitempty"`
	HardTokens *int `json:"hardTokens,omitempty"`
}

type UpdateGatewayCompactionSettingsRequest struct {
	Mode               *string                                            `json:"mode,omitempty"`
	ReserveTokens      *int                                               `json:"reserveTokens,omitempty"`
	KeepRecentTokens   *int                                               `json:"keepRecentTokens,omitempty"`
	ReserveTokensFloor *int                                               `json:"reserveTokensFloor,omitempty"`
	MaxHistoryShare    *float64                                           `json:"maxHistoryShare,omitempty"`
	MemoryFlush        *UpdateGatewayCompactionMemoryFlushSettingsRequest `json:"memoryFlush,omitempty"`
}

type UpdateGatewayCompactionMemoryFlushSettingsRequest struct {
	Enabled             *bool   `json:"enabled,omitempty"`
	SoftThresholdTokens *int    `json:"softThresholdTokens,omitempty"`
	Prompt              *string `json:"prompt,omitempty"`
	SystemPrompt        *string `json:"systemPrompt,omitempty"`
}

type UpdateGatewayQueueSettingsRequest struct {
	GlobalConcurrency  *int                                   `json:"globalConcurrency,omitempty"`
	SessionConcurrency *int                                   `json:"sessionConcurrency,omitempty"`
	Lanes              *UpdateGatewayQueueLaneSettingsRequest `json:"lanes,omitempty"`
}

type UpdateGatewayQueueLaneSettingsRequest struct {
	Main     *int `json:"main,omitempty"`
	Subagent *int `json:"subagent,omitempty"`
	Cron     *int `json:"cron,omitempty"`
}

type UpdateGatewayHeartbeatSettingsRequest struct {
	Enabled                   *bool                                          `json:"enabled,omitempty"`
	EveryMinutes              *int                                           `json:"everyMinutes,omitempty"`
	Every                     *string                                        `json:"every,omitempty"`
	Target                    *string                                        `json:"target,omitempty"`
	To                        *string                                        `json:"to,omitempty"`
	AccountID                 *string                                        `json:"accountId,omitempty"`
	Model                     *string                                        `json:"model,omitempty"`
	Session                   *string                                        `json:"session,omitempty"`
	Prompt                    *string                                        `json:"prompt,omitempty"`
	IncludeReasoning          *bool                                          `json:"includeReasoning,omitempty"`
	SuppressToolErrorWarnings *bool                                          `json:"suppressToolErrorWarnings,omitempty"`
	ActiveHours               *UpdateGatewayHeartbeatActiveHoursRequest      `json:"activeHours,omitempty"`
	Checklist                 *UpdateGatewayHeartbeatChecklistRequest        `json:"checklist,omitempty"`
	RunSession                *string                                        `json:"runSession,omitempty"`
	PromptAppend              *string                                        `json:"promptAppend,omitempty"`
	Periodic                  *UpdateGatewayHeartbeatPeriodicSettingsRequest `json:"periodic,omitempty"`
	Delivery                  *UpdateGatewayHeartbeatDeliverySettingsRequest `json:"delivery,omitempty"`
	Events                    *UpdateGatewayHeartbeatEventSettingsRequest    `json:"events,omitempty"`
}

type UpdateGatewayHeartbeatActiveHoursRequest struct {
	Start    *string `json:"start,omitempty"`
	End      *string `json:"end,omitempty"`
	Timezone *string `json:"timezone,omitempty"`
}

type UpdateGatewayHeartbeatChecklistRequest struct {
	Title     string                                       `json:"title,omitempty"`
	Items     []UpdateGatewayHeartbeatChecklistItemRequest `json:"items,omitempty"`
	Notes     string                                       `json:"notes,omitempty"`
	Version   int                                          `json:"version"`
	UpdatedAt string                                       `json:"updatedAt,omitempty"`
}

type UpdateGatewayHeartbeatChecklistItemRequest struct {
	ID       string `json:"id,omitempty"`
	Text     string `json:"text,omitempty"`
	Done     bool   `json:"done"`
	Priority string `json:"priority,omitempty"`
}

type UpdateGatewayHeartbeatPeriodicSettingsRequest struct {
	Enabled *bool   `json:"enabled,omitempty"`
	Every   *string `json:"every,omitempty"`
}

type UpdateGatewayHeartbeatDeliverySettingsRequest struct {
	Periodic        *UpdateGatewayHeartbeatSurfacePolicyRequest `json:"periodic,omitempty"`
	EventDriven     *UpdateGatewayHeartbeatSurfacePolicyRequest `json:"eventDriven,omitempty"`
	ThreadReplyMode *string                                     `json:"threadReplyMode,omitempty"`
}

type UpdateGatewayHeartbeatSurfacePolicyRequest struct {
	Center           *bool   `json:"center,omitempty"`
	PopupMinSeverity *string `json:"popupMinSeverity,omitempty"`
	ToastMinSeverity *string `json:"toastMinSeverity,omitempty"`
	OSMinSeverity    *string `json:"osMinSeverity,omitempty"`
}

type UpdateGatewayHeartbeatEventSettingsRequest struct {
	CronWakeMode     *string `json:"cronWakeMode,omitempty"`
	ExecWakeMode     *string `json:"execWakeMode,omitempty"`
	SubagentWakeMode *string `json:"subagentWakeMode,omitempty"`
}

type UpdateGatewaySubagentSettingsRequest struct {
	MaxDepth      *int                                    `json:"maxDepth,omitempty"`
	MaxChildren   *int                                    `json:"maxChildren,omitempty"`
	MaxConcurrent *int                                    `json:"maxConcurrent,omitempty"`
	Model         *string                                 `json:"model,omitempty"`
	Thinking      *string                                 `json:"thinking,omitempty"`
	Tools         *UpdateGatewaySubagentToolPolicyRequest `json:"tools,omitempty"`
}

type UpdateGatewaySubagentToolPolicyRequest struct {
	Allow     *[]string `json:"allow,omitempty"`
	AlsoAllow *[]string `json:"alsoAllow,omitempty"`
	Deny      *[]string `json:"deny,omitempty"`
}

type UpdateGatewayCronSettingsRequest struct {
	Enabled           *bool                                  `json:"enabled,omitempty"`
	MaxConcurrentRuns *int                                   `json:"maxConcurrentRuns,omitempty"`
	SessionRetention  *string                                `json:"sessionRetention,omitempty"`
	RunLog            *UpdateGatewayCronRunLogSettingRequest `json:"runLog,omitempty"`
}

type UpdateGatewayCronRunLogSettingRequest struct {
	MaxBytes  *string `json:"maxBytes,omitempty"`
	KeepLines *int    `json:"keepLines,omitempty"`
}

type UpdateGatewayHTTPSettingsRequest struct {
	Endpoints *UpdateGatewayHTTPEndpointsSettingsRequest `json:"endpoints,omitempty"`
}

type UpdateGatewayHTTPEndpointsSettingsRequest struct {
	ChatCompletions *UpdateGatewayHTTPChatCompletionsSettingsRequest `json:"chatCompletions,omitempty"`
	Responses       *UpdateGatewayHTTPResponsesSettingsRequest       `json:"responses,omitempty"`
}

type UpdateGatewayHTTPChatCompletionsSettingsRequest struct {
	Enabled *bool `json:"enabled,omitempty"`
}

type UpdateGatewayHTTPResponsesSettingsRequest struct {
	Enabled      *bool                                            `json:"enabled,omitempty"`
	MaxBodyBytes *int                                             `json:"maxBodyBytes,omitempty"`
	MaxURLParts  *int                                             `json:"maxUrlParts,omitempty"`
	Files        *UpdateGatewayHTTPResponsesFilesSettingsRequest  `json:"files,omitempty"`
	Images       *UpdateGatewayHTTPResponsesImagesSettingsRequest `json:"images,omitempty"`
}

type UpdateGatewayHTTPResponsesFilesSettingsRequest struct {
	AllowURL     *bool                                         `json:"allowUrl,omitempty"`
	URLAllowlist *[]string                                     `json:"urlAllowlist,omitempty"`
	AllowedMimes *[]string                                     `json:"allowedMimes,omitempty"`
	MaxBytes     *int                                          `json:"maxBytes,omitempty"`
	MaxChars     *int                                          `json:"maxChars,omitempty"`
	MaxRedirects *int                                          `json:"maxRedirects,omitempty"`
	TimeoutMs    *int                                          `json:"timeoutMs,omitempty"`
	PDF          *UpdateGatewayHTTPResponsesPDFSettingsRequest `json:"pdf,omitempty"`
}

type UpdateGatewayHTTPResponsesPDFSettingsRequest struct {
	MaxPages     *int `json:"maxPages,omitempty"`
	MaxPixels    *int `json:"maxPixels,omitempty"`
	MinTextChars *int `json:"minTextChars,omitempty"`
}

type UpdateGatewayHTTPResponsesImagesSettingsRequest struct {
	AllowURL     *bool     `json:"allowUrl,omitempty"`
	URLAllowlist *[]string `json:"urlAllowlist,omitempty"`
	AllowedMimes *[]string `json:"allowedMimes,omitempty"`
	MaxBytes     *int      `json:"maxBytes,omitempty"`
	MaxRedirects *int      `json:"maxRedirects,omitempty"`
	TimeoutMs    *int      `json:"timeoutMs,omitempty"`
}

type Proxy struct {
	Mode           string   `json:"mode"`
	Scheme         string   `json:"scheme"`
	Host           string   `json:"host"`
	Port           int      `json:"port"`
	Username       string   `json:"username"`
	Password       string   `json:"password"`
	NoProxy        []string `json:"noProxy"`
	TimeoutSeconds int      `json:"timeoutSeconds"`
	TestedAt       string   `json:"testedAt"`
	TestSuccess    bool     `json:"testSuccess"`
	TestMessage    string   `json:"testMessage"`
}

type SystemProxyInfo struct {
	Address string `json:"address"`
	Source  string `json:"source"`
	Name    string `json:"name"`
}
