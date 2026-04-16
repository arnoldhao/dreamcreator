package settings

import (
	"strings"
	"time"
)

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
	DebugMode         GatewayDebugMode             `json:"debugMode"`
	RecordPrompt      bool                         `json:"recordPrompt"`
	ToolLoopDetection GatewayToolLoopSettings      `json:"toolLoopDetection"`
	ContextWindow     GatewayContextWindowSettings `json:"contextWindow"`
	Compaction        GatewayCompactionSettings    `json:"compaction"`
}

type GatewayDebugMode string

func (mode GatewayDebugMode) String() string {
	return string(mode)
}

const (
	GatewayDebugModeOff   GatewayDebugMode = "off"
	GatewayDebugModeBasic GatewayDebugMode = "basic"
	GatewayDebugModeFull  GatewayDebugMode = "full"
)

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

type GatewaySettingsParams struct {
	ControlPlaneEnabled       *bool                           `json:"controlPlaneEnabled,omitempty"`
	VoiceEnabled              *bool                           `json:"voiceEnabled,omitempty"`
	VoiceWakeEnabled          *bool                           `json:"voiceWakeEnabled,omitempty"`
	SandboxEnabled            *bool                           `json:"sandboxEnabled,omitempty"`
	HTTP                      *GatewayHTTPSettingsParams      `json:"http,omitempty"`
	ChannelHealthCheckMinutes *int                            `json:"channelHealthCheckMinutes,omitempty"`
	Runtime                   *GatewayRuntimeSettingsParams   `json:"runtime,omitempty"`
	Queue                     *GatewayQueueSettingsParams     `json:"queue,omitempty"`
	Heartbeat                 *GatewayHeartbeatSettingsParams `json:"heartbeat,omitempty"`
	Subagents                 *GatewaySubagentSettingsParams  `json:"subagents,omitempty"`
	Cron                      *GatewayCronSettingsParams      `json:"cron,omitempty"`
}

type GatewayRuntimeSettingsParams struct {
	MaxSteps          *int                                `json:"maxSteps,omitempty"`
	DebugMode         *string                             `json:"debugMode,omitempty"`
	RecordPrompt      *bool                               `json:"recordPrompt,omitempty"`
	ToolLoopDetection *GatewayToolLoopSettingsParams      `json:"toolLoopDetection,omitempty"`
	ContextWindow     *GatewayContextWindowSettingsParams `json:"contextWindow,omitempty"`
	Compaction        *GatewayCompactionSettingsParams    `json:"compaction,omitempty"`
}

type GatewayToolLoopSettingsParams struct {
	Enabled                       *bool                           `json:"enabled,omitempty"`
	WarnThreshold                 *int                            `json:"warnThreshold,omitempty"`
	CriticalThreshold             *int                            `json:"criticalThreshold,omitempty"`
	GlobalCircuitBreakerThreshold *int                            `json:"globalCircuitBreakerThreshold,omitempty"`
	HistorySize                   *int                            `json:"historySize,omitempty"`
	Detectors                     *GatewayToolLoopDetectorsParams `json:"detectors,omitempty"`
	AbortThreshold                *int                            `json:"abortThreshold,omitempty"`
	WindowSize                    *int                            `json:"windowSize,omitempty"`
}

type GatewayToolLoopDetectorsParams struct {
	GenericRepeat       *bool `json:"genericRepeat,omitempty"`
	KnownPollNoProgress *bool `json:"knownPollNoProgress,omitempty"`
	PingPong            *bool `json:"pingPong,omitempty"`
}

type GatewayContextWindowSettingsParams struct {
	WarnTokens *int `json:"warnTokens,omitempty"`
	HardTokens *int `json:"hardTokens,omitempty"`
}

type GatewayCompactionSettingsParams struct {
	Mode               *string                                     `json:"mode,omitempty"`
	ReserveTokens      *int                                        `json:"reserveTokens,omitempty"`
	KeepRecentTokens   *int                                        `json:"keepRecentTokens,omitempty"`
	ReserveTokensFloor *int                                        `json:"reserveTokensFloor,omitempty"`
	MaxHistoryShare    *float64                                    `json:"maxHistoryShare,omitempty"`
	MemoryFlush        *GatewayCompactionMemoryFlushSettingsParams `json:"memoryFlush,omitempty"`
}

type GatewayCompactionMemoryFlushSettingsParams struct {
	Enabled             *bool   `json:"enabled,omitempty"`
	SoftThresholdTokens *int    `json:"softThresholdTokens,omitempty"`
	Prompt              *string `json:"prompt,omitempty"`
	SystemPrompt        *string `json:"systemPrompt,omitempty"`
}

type GatewayQueueSettingsParams struct {
	GlobalConcurrency  *int                            `json:"globalConcurrency,omitempty"`
	SessionConcurrency *int                            `json:"sessionConcurrency,omitempty"`
	Lanes              *GatewayQueueLaneSettingsParams `json:"lanes,omitempty"`
}

type GatewayQueueLaneSettingsParams struct {
	Main     *int `json:"main,omitempty"`
	Subagent *int `json:"subagent,omitempty"`
	Cron     *int `json:"cron,omitempty"`
}

type GatewayHeartbeatSettingsParams struct {
	Enabled                   *bool                                   `json:"enabled,omitempty"`
	EveryMinutes              *int                                    `json:"everyMinutes,omitempty"`
	Every                     *string                                 `json:"every,omitempty"`
	Target                    *string                                 `json:"target,omitempty"`
	To                        *string                                 `json:"to,omitempty"`
	AccountID                 *string                                 `json:"accountId,omitempty"`
	Model                     *string                                 `json:"model,omitempty"`
	Session                   *string                                 `json:"session,omitempty"`
	Prompt                    *string                                 `json:"prompt,omitempty"`
	IncludeReasoning          *bool                                   `json:"includeReasoning,omitempty"`
	SuppressToolErrorWarnings *bool                                   `json:"suppressToolErrorWarnings,omitempty"`
	ActiveHours               *GatewayHeartbeatActiveHoursParams      `json:"activeHours,omitempty"`
	Checklist                 *GatewayHeartbeatChecklistParams        `json:"checklist,omitempty"`
	RunSession                *string                                 `json:"runSession,omitempty"`
	PromptAppend              *string                                 `json:"promptAppend,omitempty"`
	Periodic                  *GatewayHeartbeatPeriodicSettingsParams `json:"periodic,omitempty"`
	Delivery                  *GatewayHeartbeatDeliverySettingsParams `json:"delivery,omitempty"`
	Events                    *GatewayHeartbeatEventSettingsParams    `json:"events,omitempty"`
}

type GatewayHeartbeatActiveHoursParams struct {
	Start    *string `json:"start,omitempty"`
	End      *string `json:"end,omitempty"`
	Timezone *string `json:"timezone,omitempty"`
}

type GatewayHeartbeatChecklistParams struct {
	Title     string                                `json:"title,omitempty"`
	Items     []GatewayHeartbeatChecklistItemParams `json:"items,omitempty"`
	Notes     string                                `json:"notes,omitempty"`
	Version   int                                   `json:"version"`
	UpdatedAt string                                `json:"updatedAt,omitempty"`
}

type GatewayHeartbeatChecklistItemParams struct {
	ID       string `json:"id,omitempty"`
	Text     string `json:"text,omitempty"`
	Done     bool   `json:"done"`
	Priority string `json:"priority,omitempty"`
}

type GatewayHeartbeatPeriodicSettingsParams struct {
	Enabled *bool   `json:"enabled,omitempty"`
	Every   *string `json:"every,omitempty"`
}

type GatewayHeartbeatDeliverySettingsParams struct {
	Periodic        *GatewayHeartbeatSurfacePolicyParams `json:"periodic,omitempty"`
	EventDriven     *GatewayHeartbeatSurfacePolicyParams `json:"eventDriven,omitempty"`
	ThreadReplyMode *string                              `json:"threadReplyMode,omitempty"`
}

type GatewayHeartbeatSurfacePolicyParams struct {
	Center           *bool   `json:"center,omitempty"`
	PopupMinSeverity *string `json:"popupMinSeverity,omitempty"`
	ToastMinSeverity *string `json:"toastMinSeverity,omitempty"`
	OSMinSeverity    *string `json:"osMinSeverity,omitempty"`
}

type GatewayHeartbeatEventSettingsParams struct {
	CronWakeMode     *string `json:"cronWakeMode,omitempty"`
	ExecWakeMode     *string `json:"execWakeMode,omitempty"`
	SubagentWakeMode *string `json:"subagentWakeMode,omitempty"`
}

type GatewaySubagentSettingsParams struct {
	MaxDepth      *int                             `json:"maxDepth,omitempty"`
	MaxChildren   *int                             `json:"maxChildren,omitempty"`
	MaxConcurrent *int                             `json:"maxConcurrent,omitempty"`
	Model         *string                          `json:"model,omitempty"`
	Thinking      *string                          `json:"thinking,omitempty"`
	Tools         *GatewaySubagentToolPolicyParams `json:"tools,omitempty"`
}

type GatewaySubagentToolPolicyParams struct {
	Allow     []string `json:"allow,omitempty"`
	AlsoAllow []string `json:"alsoAllow,omitempty"`
	Deny      []string `json:"deny,omitempty"`
}

type GatewayCronSettingsParams struct {
	Enabled           *bool                            `json:"enabled,omitempty"`
	MaxConcurrentRuns *int                             `json:"maxConcurrentRuns,omitempty"`
	SessionRetention  *string                          `json:"sessionRetention,omitempty"`
	RunLog            *GatewayCronRunLogSettingsParams `json:"runLog,omitempty"`
}

type GatewayCronRunLogSettingsParams struct {
	MaxBytes  *string `json:"maxBytes,omitempty"`
	KeepLines *int    `json:"keepLines,omitempty"`
}

type GatewayHTTPSettingsParams struct {
	Endpoints *GatewayHTTPEndpointsSettingsParams `json:"endpoints,omitempty"`
}

type GatewayHTTPEndpointsSettingsParams struct {
	ChatCompletions *GatewayHTTPChatCompletionsSettingsParams `json:"chatCompletions,omitempty"`
	Responses       *GatewayHTTPResponsesSettingsParams       `json:"responses,omitempty"`
}

type GatewayHTTPChatCompletionsSettingsParams struct {
	Enabled *bool `json:"enabled,omitempty"`
}

type GatewayHTTPResponsesSettingsParams struct {
	Enabled      *bool                                     `json:"enabled,omitempty"`
	MaxBodyBytes *int                                      `json:"maxBodyBytes,omitempty"`
	MaxURLParts  *int                                      `json:"maxUrlParts,omitempty"`
	Files        *GatewayHTTPResponsesFilesSettingsParams  `json:"files,omitempty"`
	Images       *GatewayHTTPResponsesImagesSettingsParams `json:"images,omitempty"`
}

type GatewayHTTPResponsesFilesSettingsParams struct {
	AllowURL     *bool                                  `json:"allowUrl,omitempty"`
	URLAllowlist []string                               `json:"urlAllowlist,omitempty"`
	AllowedMimes []string                               `json:"allowedMimes,omitempty"`
	MaxBytes     *int                                   `json:"maxBytes,omitempty"`
	MaxChars     *int                                   `json:"maxChars,omitempty"`
	MaxRedirects *int                                   `json:"maxRedirects,omitempty"`
	TimeoutMs    *int                                   `json:"timeoutMs,omitempty"`
	PDF          *GatewayHTTPResponsesPDFSettingsParams `json:"pdf,omitempty"`
}

type GatewayHTTPResponsesPDFSettingsParams struct {
	MaxPages     *int `json:"maxPages,omitempty"`
	MaxPixels    *int `json:"maxPixels,omitempty"`
	MinTextChars *int `json:"minTextChars,omitempty"`
}

type GatewayHTTPResponsesImagesSettingsParams struct {
	AllowURL     *bool    `json:"allowUrl,omitempty"`
	URLAllowlist []string `json:"urlAllowlist,omitempty"`
	AllowedMimes []string `json:"allowedMimes,omitempty"`
	MaxBytes     *int     `json:"maxBytes,omitempty"`
	MaxRedirects *int     `json:"maxRedirects,omitempty"`
	TimeoutMs    *int     `json:"timeoutMs,omitempty"`
}

const (
	DefaultGatewayControlPlaneEnabled                      = true
	DefaultGatewayVoiceEnabled                             = false
	DefaultGatewayVoiceWakeEnabled                         = false
	DefaultGatewaySandboxEnabled                           = true
	DefaultGatewayRuntimeMaxSteps                          = 0
	DefaultGatewayRuntimeRecordPrompt                      = false
	DefaultGatewayToolLoopEnabled                          = false
	DefaultGatewayToolLoopWarnThreshold                    = 10
	DefaultGatewayToolLoopCriticalThreshold                = 20
	DefaultGatewayToolLoopGlobalCircuitBreakerThreshold    = 30
	DefaultGatewayToolLoopHistorySize                      = 30
	DefaultGatewayToolLoopAbortThreshold                   = 20
	DefaultGatewayToolLoopWindowSize                       = 30
	DefaultGatewayToolLoopDetectorGenericRepeat            = true
	DefaultGatewayToolLoopDetectorKnownPollNoProgress      = true
	DefaultGatewayToolLoopDetectorPingPong                 = true
	DefaultGatewayContextWarnTokens                        = 32_000
	DefaultGatewayContextHardTokens                        = 16_000
	DefaultGatewayCompactionMode                           = "safeguard"
	DefaultGatewayCompactionReserveTokens                  = 20_000
	DefaultGatewayCompactionKeepRecentTokens               = 20_000
	DefaultGatewayCompactionReserveTokensFloor             = 20_000
	DefaultGatewayCompactionMaxHistoryShare                = 0.5
	DefaultGatewayCompactionMemoryFlushEnabled             = true
	DefaultGatewayCompactionMemoryFlushSoftThresholdTokens = 4000
	DefaultGatewayCompactionMemoryFlushPrompt              = "Pre-compaction memory flush. Store durable memories now (use memory/YYYY-MM-DD.md; create memory/ if needed). IMPORTANT: If the file already exists, APPEND new content only and do not overwrite existing entries. If nothing to store, reply with NO_REPLY."
	DefaultGatewayCompactionMemoryFlushSystemPrompt        = "Pre-compaction memory flush turn. The session is near auto-compaction; capture durable memories to disk. You may reply, but usually NO_REPLY is correct."
	DefaultGatewayQueueGlobalConcurrency                   = 0
	DefaultGatewayQueueSessionConcurrency                  = 0
	DefaultGatewayQueueLaneMain                            = 4
	DefaultGatewayQueueLaneSubagent                        = 8
	DefaultGatewayQueueLaneCron                            = 1
	DefaultGatewayCronEnabled                              = true
	DefaultGatewayCronMaxConcurrentRuns                    = 1
	DefaultGatewayCronSessionRetention                     = "24h"
	DefaultGatewayCronRunLogMaxBytes                       = "2mb"
	DefaultGatewayCronRunLogKeepLines                      = 2000
	DefaultGatewayHeartbeatEnabled                         = true
	DefaultGatewayHeartbeatEveryMinutes                    = 30
	DefaultGatewayHeartbeatEvery                           = "30m"
	DefaultGatewayHeartbeatTarget                          = "last"
	DefaultGatewayHeartbeatPrompt                          = ""
	DefaultGatewayHeartbeatIncludeReasoning                = false
	DefaultGatewayHeartbeatSuppressToolErrorWarnings       = false
	DefaultGatewayHeartbeatActiveStart                     = "00:00"
	DefaultGatewayHeartbeatActiveEnd                       = "23:59"
	DefaultGatewayHeartbeatRunSession                      = ""
	DefaultGatewayHeartbeatPromptAppend                    = ""
	DefaultGatewayHeartbeatPeriodicEnabled                 = true
	DefaultGatewayHeartbeatPeriodicEvery                   = "30m"
	DefaultGatewayHeartbeatPeriodicPopupMinSeverity        = ""
	DefaultGatewayHeartbeatPeriodicToastMinSeverity        = ""
	DefaultGatewayHeartbeatPeriodicOSMinSeverity           = ""
	DefaultGatewayHeartbeatEventDrivenPopupMinSeverity     = "warning"
	DefaultGatewayHeartbeatEventDrivenToastMinSeverity     = ""
	DefaultGatewayHeartbeatEventDrivenOSMinSeverity        = "error"
	DefaultGatewayHeartbeatThreadReplyMode                 = "never"
	DefaultGatewayHeartbeatCronWakeMode                    = "next-heartbeat"
	DefaultGatewayHeartbeatExecWakeMode                    = "now"
	DefaultGatewayHeartbeatSubagentWakeMode                = "now"
	DefaultGatewaySubagentMaxDepth                         = 1
	DefaultGatewaySubagentMaxChildren                      = 5
	DefaultGatewaySubagentMaxConcurrent                    = 8
	DefaultGatewayHTTPChatCompletionsEnabled               = false
	DefaultGatewayHTTPResponsesEnabled                     = false
	DefaultGatewayHTTPResponsesMaxBodyBytes                = 20 * 1024 * 1024
	DefaultGatewayHTTPResponsesMaxURLParts                 = 8
	DefaultGatewayHTTPResponsesFilesAllowURL               = true
	DefaultGatewayHTTPResponsesFilesMaxBytes               = 5 * 1024 * 1024
	DefaultGatewayHTTPResponsesFilesMaxChars               = 200000
	DefaultGatewayHTTPResponsesFilesMaxRedirects           = 3
	DefaultGatewayHTTPResponsesFilesTimeoutMs              = 10000
	DefaultGatewayHTTPResponsesFilesPDFMaxPages            = 4
	DefaultGatewayHTTPResponsesFilesPDFMaxPixels           = 4_000_000
	DefaultGatewayHTTPResponsesFilesPDFMinTextChars        = 200
	DefaultGatewayHTTPResponsesImagesAllowURL              = true
	DefaultGatewayHTTPResponsesImagesMaxBytes              = 10 * 1024 * 1024
	DefaultGatewayHTTPResponsesImagesMaxRedirects          = 3
	DefaultGatewayHTTPResponsesImagesTimeoutMs             = 10000
	DefaultGatewayChannelHealthCheckMinutes                = 5
	DefaultGatewayRuntimeDebugMode                         = GatewayDebugModeOff
)

func DefaultGatewaySettings() GatewaySettings {
	defaultTimezone := defaultGatewayHeartbeatActiveTimezone()
	return GatewaySettings{
		ControlPlaneEnabled: DefaultGatewayControlPlaneEnabled,
		VoiceEnabled:        DefaultGatewayVoiceEnabled,
		VoiceWakeEnabled:    DefaultGatewayVoiceWakeEnabled,
		SandboxEnabled:      DefaultGatewaySandboxEnabled,
		Runtime: GatewayRuntimeSettings{
			MaxSteps:     DefaultGatewayRuntimeMaxSteps,
			DebugMode:    DefaultGatewayRuntimeDebugMode,
			RecordPrompt: DefaultGatewayRuntimeRecordPrompt,
			ToolLoopDetection: GatewayToolLoopSettings{
				Enabled:                       DefaultGatewayToolLoopEnabled,
				WarnThreshold:                 DefaultGatewayToolLoopWarnThreshold,
				CriticalThreshold:             DefaultGatewayToolLoopCriticalThreshold,
				GlobalCircuitBreakerThreshold: DefaultGatewayToolLoopGlobalCircuitBreakerThreshold,
				HistorySize:                   DefaultGatewayToolLoopHistorySize,
				Detectors: GatewayToolLoopDetectors{
					GenericRepeat:       DefaultGatewayToolLoopDetectorGenericRepeat,
					KnownPollNoProgress: DefaultGatewayToolLoopDetectorKnownPollNoProgress,
					PingPong:            DefaultGatewayToolLoopDetectorPingPong,
				},
				AbortThreshold: DefaultGatewayToolLoopAbortThreshold,
				WindowSize:     DefaultGatewayToolLoopWindowSize,
			},
			ContextWindow: GatewayContextWindowSettings{
				WarnTokens: DefaultGatewayContextWarnTokens,
				HardTokens: DefaultGatewayContextHardTokens,
			},
			Compaction: GatewayCompactionSettings{
				Mode:               DefaultGatewayCompactionMode,
				ReserveTokens:      DefaultGatewayCompactionReserveTokens,
				KeepRecentTokens:   DefaultGatewayCompactionKeepRecentTokens,
				ReserveTokensFloor: DefaultGatewayCompactionReserveTokensFloor,
				MaxHistoryShare:    DefaultGatewayCompactionMaxHistoryShare,
				MemoryFlush: GatewayCompactionMemoryFlushSettings{
					Enabled:             DefaultGatewayCompactionMemoryFlushEnabled,
					SoftThresholdTokens: DefaultGatewayCompactionMemoryFlushSoftThresholdTokens,
					Prompt:              DefaultGatewayCompactionMemoryFlushPrompt,
					SystemPrompt:        DefaultGatewayCompactionMemoryFlushSystemPrompt,
				},
			},
		},
		Queue: GatewayQueueSettings{
			GlobalConcurrency:  DefaultGatewayQueueGlobalConcurrency,
			SessionConcurrency: DefaultGatewayQueueSessionConcurrency,
			Lanes: GatewayQueueLaneSettings{
				Main:     DefaultGatewayQueueLaneMain,
				Subagent: DefaultGatewayQueueLaneSubagent,
				Cron:     DefaultGatewayQueueLaneCron,
			},
		},
		Heartbeat: GatewayHeartbeatSettings{
			Enabled:                   DefaultGatewayHeartbeatEnabled,
			EveryMinutes:              DefaultGatewayHeartbeatEveryMinutes,
			Every:                     DefaultGatewayHeartbeatEvery,
			Target:                    DefaultGatewayHeartbeatTarget,
			Prompt:                    DefaultGatewayHeartbeatPrompt,
			IncludeReasoning:          DefaultGatewayHeartbeatIncludeReasoning,
			SuppressToolErrorWarnings: DefaultGatewayHeartbeatSuppressToolErrorWarnings,
			ActiveHours: GatewayHeartbeatActiveHours{
				Start:    DefaultGatewayHeartbeatActiveStart,
				End:      DefaultGatewayHeartbeatActiveEnd,
				Timezone: defaultTimezone,
			},
			Checklist: GatewayHeartbeatChecklist{
				Items: []GatewayHeartbeatChecklistItem{},
			},
			RunSession:   DefaultGatewayHeartbeatRunSession,
			PromptAppend: DefaultGatewayHeartbeatPromptAppend,
			Periodic: GatewayHeartbeatPeriodicSettings{
				Enabled: DefaultGatewayHeartbeatPeriodicEnabled,
				Every:   DefaultGatewayHeartbeatPeriodicEvery,
			},
			Delivery: GatewayHeartbeatDeliverySettings{
				Periodic: GatewayHeartbeatSurfacePolicy{
					Center:           true,
					PopupMinSeverity: DefaultGatewayHeartbeatPeriodicPopupMinSeverity,
					ToastMinSeverity: DefaultGatewayHeartbeatPeriodicToastMinSeverity,
					OSMinSeverity:    DefaultGatewayHeartbeatPeriodicOSMinSeverity,
				},
				EventDriven: GatewayHeartbeatSurfacePolicy{
					Center:           true,
					PopupMinSeverity: DefaultGatewayHeartbeatEventDrivenPopupMinSeverity,
					ToastMinSeverity: DefaultGatewayHeartbeatEventDrivenToastMinSeverity,
					OSMinSeverity:    DefaultGatewayHeartbeatEventDrivenOSMinSeverity,
				},
				ThreadReplyMode: DefaultGatewayHeartbeatThreadReplyMode,
			},
			Events: GatewayHeartbeatEventSettings{
				CronWakeMode:     DefaultGatewayHeartbeatCronWakeMode,
				ExecWakeMode:     DefaultGatewayHeartbeatExecWakeMode,
				SubagentWakeMode: DefaultGatewayHeartbeatSubagentWakeMode,
			},
		},
		Subagents: GatewaySubagentSettings{
			MaxDepth:      DefaultGatewaySubagentMaxDepth,
			MaxChildren:   DefaultGatewaySubagentMaxChildren,
			MaxConcurrent: DefaultGatewaySubagentMaxConcurrent,
			Model:         "",
			Thinking:      "",
			Tools: GatewaySubagentToolPolicy{
				Allow:     []string{},
				AlsoAllow: []string{},
				Deny:      []string{},
			},
		},
		Cron: GatewayCronSettings{
			Enabled:           DefaultGatewayCronEnabled,
			MaxConcurrentRuns: DefaultGatewayCronMaxConcurrentRuns,
			SessionRetention:  DefaultGatewayCronSessionRetention,
			RunLog: GatewayCronRunLogSetting{
				MaxBytes:  DefaultGatewayCronRunLogMaxBytes,
				KeepLines: DefaultGatewayCronRunLogKeepLines,
			},
		},
		HTTP: GatewayHTTPSettings{
			Endpoints: GatewayHTTPEndpointsSettings{
				ChatCompletions: GatewayHTTPChatCompletionsSettings{
					Enabled: DefaultGatewayHTTPChatCompletionsEnabled,
				},
				Responses: GatewayHTTPResponsesSettings{
					Enabled:      DefaultGatewayHTTPResponsesEnabled,
					MaxBodyBytes: DefaultGatewayHTTPResponsesMaxBodyBytes,
					MaxURLParts:  DefaultGatewayHTTPResponsesMaxURLParts,
					Files: GatewayHTTPResponsesFilesSettings{
						AllowURL:     DefaultGatewayHTTPResponsesFilesAllowURL,
						URLAllowlist: []string{},
						AllowedMimes: []string{},
						MaxBytes:     DefaultGatewayHTTPResponsesFilesMaxBytes,
						MaxChars:     DefaultGatewayHTTPResponsesFilesMaxChars,
						MaxRedirects: DefaultGatewayHTTPResponsesFilesMaxRedirects,
						TimeoutMs:    DefaultGatewayHTTPResponsesFilesTimeoutMs,
						PDF: GatewayHTTPResponsesPDFSettings{
							MaxPages:     DefaultGatewayHTTPResponsesFilesPDFMaxPages,
							MaxPixels:    DefaultGatewayHTTPResponsesFilesPDFMaxPixels,
							MinTextChars: DefaultGatewayHTTPResponsesFilesPDFMinTextChars,
						},
					},
					Images: GatewayHTTPResponsesImagesSettings{
						AllowURL:     DefaultGatewayHTTPResponsesImagesAllowURL,
						URLAllowlist: []string{},
						AllowedMimes: []string{},
						MaxBytes:     DefaultGatewayHTTPResponsesImagesMaxBytes,
						MaxRedirects: DefaultGatewayHTTPResponsesImagesMaxRedirects,
						TimeoutMs:    DefaultGatewayHTTPResponsesImagesTimeoutMs,
					},
				},
			},
		},
		ChannelHealthCheckMinutes: DefaultGatewayChannelHealthCheckMinutes,
	}
}

func defaultGatewayHeartbeatActiveTimezone() string {
	timezone := strings.TrimSpace(time.Now().Location().String())
	if timezone == "" || strings.EqualFold(timezone, "local") {
		return "UTC"
	}
	return timezone
}

func ResolveGatewaySettings(params GatewaySettingsParams) GatewaySettings {
	settings := DefaultGatewaySettings()

	if params.ControlPlaneEnabled != nil {
		settings.ControlPlaneEnabled = *params.ControlPlaneEnabled
	}
	if params.VoiceEnabled != nil {
		settings.VoiceEnabled = *params.VoiceEnabled
	}
	if params.VoiceWakeEnabled != nil {
		settings.VoiceWakeEnabled = *params.VoiceWakeEnabled
	}
	if params.SandboxEnabled != nil {
		settings.SandboxEnabled = *params.SandboxEnabled
	}
	if params.HTTP != nil && params.HTTP.Endpoints != nil {
		if params.HTTP.Endpoints.ChatCompletions != nil {
			if params.HTTP.Endpoints.ChatCompletions.Enabled != nil {
				settings.HTTP.Endpoints.ChatCompletions.Enabled = *params.HTTP.Endpoints.ChatCompletions.Enabled
			}
		}
		if params.HTTP.Endpoints.Responses != nil {
			if params.HTTP.Endpoints.Responses.Enabled != nil {
				settings.HTTP.Endpoints.Responses.Enabled = *params.HTTP.Endpoints.Responses.Enabled
			}
			if params.HTTP.Endpoints.Responses.MaxBodyBytes != nil {
				settings.HTTP.Endpoints.Responses.MaxBodyBytes = *params.HTTP.Endpoints.Responses.MaxBodyBytes
			}
			if params.HTTP.Endpoints.Responses.MaxURLParts != nil {
				settings.HTTP.Endpoints.Responses.MaxURLParts = *params.HTTP.Endpoints.Responses.MaxURLParts
			}
			if params.HTTP.Endpoints.Responses.Files != nil {
				if params.HTTP.Endpoints.Responses.Files.AllowURL != nil {
					settings.HTTP.Endpoints.Responses.Files.AllowURL = *params.HTTP.Endpoints.Responses.Files.AllowURL
				}
				if params.HTTP.Endpoints.Responses.Files.URLAllowlist != nil {
					settings.HTTP.Endpoints.Responses.Files.URLAllowlist = normalizeStringSlice(params.HTTP.Endpoints.Responses.Files.URLAllowlist)
				}
				if params.HTTP.Endpoints.Responses.Files.AllowedMimes != nil {
					settings.HTTP.Endpoints.Responses.Files.AllowedMimes = normalizeStringSlice(params.HTTP.Endpoints.Responses.Files.AllowedMimes)
				}
				if params.HTTP.Endpoints.Responses.Files.MaxBytes != nil {
					settings.HTTP.Endpoints.Responses.Files.MaxBytes = *params.HTTP.Endpoints.Responses.Files.MaxBytes
				}
				if params.HTTP.Endpoints.Responses.Files.MaxChars != nil {
					settings.HTTP.Endpoints.Responses.Files.MaxChars = *params.HTTP.Endpoints.Responses.Files.MaxChars
				}
				if params.HTTP.Endpoints.Responses.Files.MaxRedirects != nil {
					settings.HTTP.Endpoints.Responses.Files.MaxRedirects = *params.HTTP.Endpoints.Responses.Files.MaxRedirects
				}
				if params.HTTP.Endpoints.Responses.Files.TimeoutMs != nil {
					settings.HTTP.Endpoints.Responses.Files.TimeoutMs = *params.HTTP.Endpoints.Responses.Files.TimeoutMs
				}
				if params.HTTP.Endpoints.Responses.Files.PDF != nil {
					if params.HTTP.Endpoints.Responses.Files.PDF.MaxPages != nil {
						settings.HTTP.Endpoints.Responses.Files.PDF.MaxPages = *params.HTTP.Endpoints.Responses.Files.PDF.MaxPages
					}
					if params.HTTP.Endpoints.Responses.Files.PDF.MaxPixels != nil {
						settings.HTTP.Endpoints.Responses.Files.PDF.MaxPixels = *params.HTTP.Endpoints.Responses.Files.PDF.MaxPixels
					}
					if params.HTTP.Endpoints.Responses.Files.PDF.MinTextChars != nil {
						settings.HTTP.Endpoints.Responses.Files.PDF.MinTextChars = *params.HTTP.Endpoints.Responses.Files.PDF.MinTextChars
					}
				}
			}
			if params.HTTP.Endpoints.Responses.Images != nil {
				if params.HTTP.Endpoints.Responses.Images.AllowURL != nil {
					settings.HTTP.Endpoints.Responses.Images.AllowURL = *params.HTTP.Endpoints.Responses.Images.AllowURL
				}
				if params.HTTP.Endpoints.Responses.Images.URLAllowlist != nil {
					settings.HTTP.Endpoints.Responses.Images.URLAllowlist = normalizeStringSlice(params.HTTP.Endpoints.Responses.Images.URLAllowlist)
				}
				if params.HTTP.Endpoints.Responses.Images.AllowedMimes != nil {
					settings.HTTP.Endpoints.Responses.Images.AllowedMimes = normalizeStringSlice(params.HTTP.Endpoints.Responses.Images.AllowedMimes)
				}
				if params.HTTP.Endpoints.Responses.Images.MaxBytes != nil {
					settings.HTTP.Endpoints.Responses.Images.MaxBytes = *params.HTTP.Endpoints.Responses.Images.MaxBytes
				}
				if params.HTTP.Endpoints.Responses.Images.MaxRedirects != nil {
					settings.HTTP.Endpoints.Responses.Images.MaxRedirects = *params.HTTP.Endpoints.Responses.Images.MaxRedirects
				}
				if params.HTTP.Endpoints.Responses.Images.TimeoutMs != nil {
					settings.HTTP.Endpoints.Responses.Images.TimeoutMs = *params.HTTP.Endpoints.Responses.Images.TimeoutMs
				}
			}
		}
	}
	if params.ChannelHealthCheckMinutes != nil {
		settings.ChannelHealthCheckMinutes = *params.ChannelHealthCheckMinutes
	}
	if params.Runtime != nil {
		if params.Runtime.MaxSteps != nil {
			settings.Runtime.MaxSteps = *params.Runtime.MaxSteps
		}
		settings.Runtime.DebugMode = ApplyGatewayDebugModeOverride(
			settings.Runtime.DebugMode,
			params.Runtime.DebugMode,
			params.Runtime.RecordPrompt,
		)
		settings.Runtime.RecordPrompt = GatewayDebugModeRecordsPrompt(settings.Runtime.DebugMode)
		if params.Runtime.ToolLoopDetection != nil {
			if params.Runtime.ToolLoopDetection.Enabled != nil {
				settings.Runtime.ToolLoopDetection.Enabled = *params.Runtime.ToolLoopDetection.Enabled
			}
			if params.Runtime.ToolLoopDetection.WarnThreshold != nil {
				settings.Runtime.ToolLoopDetection.WarnThreshold = *params.Runtime.ToolLoopDetection.WarnThreshold
			}
			if params.Runtime.ToolLoopDetection.CriticalThreshold != nil {
				settings.Runtime.ToolLoopDetection.CriticalThreshold = *params.Runtime.ToolLoopDetection.CriticalThreshold
				settings.Runtime.ToolLoopDetection.AbortThreshold = *params.Runtime.ToolLoopDetection.CriticalThreshold
			}
			if params.Runtime.ToolLoopDetection.GlobalCircuitBreakerThreshold != nil {
				settings.Runtime.ToolLoopDetection.GlobalCircuitBreakerThreshold = *params.Runtime.ToolLoopDetection.GlobalCircuitBreakerThreshold
			}
			if params.Runtime.ToolLoopDetection.HistorySize != nil {
				settings.Runtime.ToolLoopDetection.HistorySize = *params.Runtime.ToolLoopDetection.HistorySize
				settings.Runtime.ToolLoopDetection.WindowSize = *params.Runtime.ToolLoopDetection.HistorySize
			}
			if params.Runtime.ToolLoopDetection.Detectors != nil {
				if params.Runtime.ToolLoopDetection.Detectors.GenericRepeat != nil {
					settings.Runtime.ToolLoopDetection.Detectors.GenericRepeat = *params.Runtime.ToolLoopDetection.Detectors.GenericRepeat
				}
				if params.Runtime.ToolLoopDetection.Detectors.KnownPollNoProgress != nil {
					settings.Runtime.ToolLoopDetection.Detectors.KnownPollNoProgress = *params.Runtime.ToolLoopDetection.Detectors.KnownPollNoProgress
				}
				if params.Runtime.ToolLoopDetection.Detectors.PingPong != nil {
					settings.Runtime.ToolLoopDetection.Detectors.PingPong = *params.Runtime.ToolLoopDetection.Detectors.PingPong
				}
			}
			if params.Runtime.ToolLoopDetection.AbortThreshold != nil {
				settings.Runtime.ToolLoopDetection.AbortThreshold = *params.Runtime.ToolLoopDetection.AbortThreshold
				if settings.Runtime.ToolLoopDetection.CriticalThreshold == 0 {
					settings.Runtime.ToolLoopDetection.CriticalThreshold = settings.Runtime.ToolLoopDetection.AbortThreshold
				}
			}
			if params.Runtime.ToolLoopDetection.WindowSize != nil {
				settings.Runtime.ToolLoopDetection.WindowSize = *params.Runtime.ToolLoopDetection.WindowSize
				if settings.Runtime.ToolLoopDetection.HistorySize == 0 {
					settings.Runtime.ToolLoopDetection.HistorySize = settings.Runtime.ToolLoopDetection.WindowSize
				}
			}
		}
		if params.Runtime.ContextWindow != nil {
			if params.Runtime.ContextWindow.WarnTokens != nil {
				settings.Runtime.ContextWindow.WarnTokens = *params.Runtime.ContextWindow.WarnTokens
			}
			if params.Runtime.ContextWindow.HardTokens != nil {
				settings.Runtime.ContextWindow.HardTokens = *params.Runtime.ContextWindow.HardTokens
			}
		}
		if params.Runtime.Compaction != nil {
			if params.Runtime.Compaction.Mode != nil {
				settings.Runtime.Compaction.Mode = strings.TrimSpace(*params.Runtime.Compaction.Mode)
			}
			if params.Runtime.Compaction.ReserveTokens != nil {
				settings.Runtime.Compaction.ReserveTokens = *params.Runtime.Compaction.ReserveTokens
			}
			if params.Runtime.Compaction.KeepRecentTokens != nil {
				settings.Runtime.Compaction.KeepRecentTokens = *params.Runtime.Compaction.KeepRecentTokens
			}
			if params.Runtime.Compaction.ReserveTokensFloor != nil {
				settings.Runtime.Compaction.ReserveTokensFloor = *params.Runtime.Compaction.ReserveTokensFloor
			}
			if params.Runtime.Compaction.MaxHistoryShare != nil {
				settings.Runtime.Compaction.MaxHistoryShare = *params.Runtime.Compaction.MaxHistoryShare
			}
			if params.Runtime.Compaction.MemoryFlush != nil {
				if params.Runtime.Compaction.MemoryFlush.Enabled != nil {
					settings.Runtime.Compaction.MemoryFlush.Enabled = *params.Runtime.Compaction.MemoryFlush.Enabled
				}
				if params.Runtime.Compaction.MemoryFlush.SoftThresholdTokens != nil {
					settings.Runtime.Compaction.MemoryFlush.SoftThresholdTokens = *params.Runtime.Compaction.MemoryFlush.SoftThresholdTokens
				}
				if params.Runtime.Compaction.MemoryFlush.Prompt != nil {
					settings.Runtime.Compaction.MemoryFlush.Prompt = strings.TrimSpace(*params.Runtime.Compaction.MemoryFlush.Prompt)
				}
				if params.Runtime.Compaction.MemoryFlush.SystemPrompt != nil {
					settings.Runtime.Compaction.MemoryFlush.SystemPrompt = strings.TrimSpace(*params.Runtime.Compaction.MemoryFlush.SystemPrompt)
				}
			}
		}
	}
	settings.Runtime.DebugMode = ResolveGatewayDebugMode(
		settings.Runtime.DebugMode.String(),
		settings.Runtime.RecordPrompt,
	)
	settings.Runtime.RecordPrompt = GatewayDebugModeRecordsPrompt(settings.Runtime.DebugMode)
	if params.Queue != nil {
		if params.Queue.GlobalConcurrency != nil {
			settings.Queue.GlobalConcurrency = *params.Queue.GlobalConcurrency
		}
		if params.Queue.SessionConcurrency != nil {
			settings.Queue.SessionConcurrency = *params.Queue.SessionConcurrency
		}
		if params.Queue.Lanes != nil {
			if params.Queue.Lanes.Main != nil {
				settings.Queue.Lanes.Main = *params.Queue.Lanes.Main
			}
			if params.Queue.Lanes.Subagent != nil {
				settings.Queue.Lanes.Subagent = *params.Queue.Lanes.Subagent
			}
			if params.Queue.Lanes.Cron != nil {
				settings.Queue.Lanes.Cron = *params.Queue.Lanes.Cron
			}
		}
	}
	if params.Heartbeat != nil {
		if params.Heartbeat.Enabled != nil {
			settings.Heartbeat.Enabled = *params.Heartbeat.Enabled
		}
		if params.Heartbeat.EveryMinutes != nil {
			settings.Heartbeat.EveryMinutes = *params.Heartbeat.EveryMinutes
		}
		if params.Heartbeat.Every != nil {
			settings.Heartbeat.Every = strings.TrimSpace(*params.Heartbeat.Every)
		}
		if params.Heartbeat.Target != nil {
			settings.Heartbeat.Target = strings.TrimSpace(*params.Heartbeat.Target)
		}
		if params.Heartbeat.To != nil {
			settings.Heartbeat.To = strings.TrimSpace(*params.Heartbeat.To)
		}
		if params.Heartbeat.AccountID != nil {
			settings.Heartbeat.AccountID = strings.TrimSpace(*params.Heartbeat.AccountID)
		}
		if params.Heartbeat.Model != nil {
			settings.Heartbeat.Model = strings.TrimSpace(*params.Heartbeat.Model)
		}
		if params.Heartbeat.Session != nil {
			settings.Heartbeat.Session = strings.TrimSpace(*params.Heartbeat.Session)
		}
		if params.Heartbeat.Prompt != nil {
			settings.Heartbeat.Prompt = strings.TrimSpace(*params.Heartbeat.Prompt)
		}
		if params.Heartbeat.RunSession != nil {
			settings.Heartbeat.RunSession = strings.TrimSpace(*params.Heartbeat.RunSession)
		}
		if params.Heartbeat.PromptAppend != nil {
			settings.Heartbeat.PromptAppend = strings.TrimSpace(*params.Heartbeat.PromptAppend)
		}
		if params.Heartbeat.IncludeReasoning != nil {
			settings.Heartbeat.IncludeReasoning = *params.Heartbeat.IncludeReasoning
		}
		if params.Heartbeat.SuppressToolErrorWarnings != nil {
			settings.Heartbeat.SuppressToolErrorWarnings = *params.Heartbeat.SuppressToolErrorWarnings
		}
		if params.Heartbeat.ActiveHours != nil {
			if params.Heartbeat.ActiveHours.Start != nil {
				settings.Heartbeat.ActiveHours.Start = strings.TrimSpace(*params.Heartbeat.ActiveHours.Start)
			}
			if params.Heartbeat.ActiveHours.End != nil {
				settings.Heartbeat.ActiveHours.End = strings.TrimSpace(*params.Heartbeat.ActiveHours.End)
			}
			if params.Heartbeat.ActiveHours.Timezone != nil {
				settings.Heartbeat.ActiveHours.Timezone = strings.TrimSpace(*params.Heartbeat.ActiveHours.Timezone)
			}
		}
		if params.Heartbeat.Checklist != nil {
			items := make([]GatewayHeartbeatChecklistItem, 0, len(params.Heartbeat.Checklist.Items))
			for _, item := range params.Heartbeat.Checklist.Items {
				id := strings.TrimSpace(item.ID)
				text := strings.TrimSpace(item.Text)
				if id == "" && text == "" {
					continue
				}
				items = append(items, GatewayHeartbeatChecklistItem{
					ID:       id,
					Text:     text,
					Done:     item.Done,
					Priority: strings.TrimSpace(item.Priority),
				})
			}
			settings.Heartbeat.Checklist = GatewayHeartbeatChecklist{
				Title:     strings.TrimSpace(params.Heartbeat.Checklist.Title),
				Items:     items,
				Notes:     strings.TrimSpace(params.Heartbeat.Checklist.Notes),
				Version:   params.Heartbeat.Checklist.Version,
				UpdatedAt: strings.TrimSpace(params.Heartbeat.Checklist.UpdatedAt),
			}
			if settings.Heartbeat.Checklist.Version < 0 {
				settings.Heartbeat.Checklist.Version = 0
			}
		}
		if params.Heartbeat.Periodic != nil {
			if params.Heartbeat.Periodic.Enabled != nil {
				settings.Heartbeat.Periodic.Enabled = *params.Heartbeat.Periodic.Enabled
			}
			if params.Heartbeat.Periodic.Every != nil {
				settings.Heartbeat.Periodic.Every = strings.TrimSpace(*params.Heartbeat.Periodic.Every)
			}
		}
		if params.Heartbeat.Delivery != nil {
			if params.Heartbeat.Delivery.Periodic != nil {
				if params.Heartbeat.Delivery.Periodic.Center != nil {
					settings.Heartbeat.Delivery.Periodic.Center = *params.Heartbeat.Delivery.Periodic.Center
				}
				if params.Heartbeat.Delivery.Periodic.PopupMinSeverity != nil {
					settings.Heartbeat.Delivery.Periodic.PopupMinSeverity = strings.TrimSpace(*params.Heartbeat.Delivery.Periodic.PopupMinSeverity)
				}
				if params.Heartbeat.Delivery.Periodic.ToastMinSeverity != nil {
					settings.Heartbeat.Delivery.Periodic.ToastMinSeverity = strings.TrimSpace(*params.Heartbeat.Delivery.Periodic.ToastMinSeverity)
				}
				if params.Heartbeat.Delivery.Periodic.OSMinSeverity != nil {
					settings.Heartbeat.Delivery.Periodic.OSMinSeverity = strings.TrimSpace(*params.Heartbeat.Delivery.Periodic.OSMinSeverity)
				}
			}
			if params.Heartbeat.Delivery.EventDriven != nil {
				if params.Heartbeat.Delivery.EventDriven.Center != nil {
					settings.Heartbeat.Delivery.EventDriven.Center = *params.Heartbeat.Delivery.EventDriven.Center
				}
				if params.Heartbeat.Delivery.EventDriven.PopupMinSeverity != nil {
					settings.Heartbeat.Delivery.EventDriven.PopupMinSeverity = strings.TrimSpace(*params.Heartbeat.Delivery.EventDriven.PopupMinSeverity)
				}
				if params.Heartbeat.Delivery.EventDriven.ToastMinSeverity != nil {
					settings.Heartbeat.Delivery.EventDriven.ToastMinSeverity = strings.TrimSpace(*params.Heartbeat.Delivery.EventDriven.ToastMinSeverity)
				}
				if params.Heartbeat.Delivery.EventDriven.OSMinSeverity != nil {
					settings.Heartbeat.Delivery.EventDriven.OSMinSeverity = strings.TrimSpace(*params.Heartbeat.Delivery.EventDriven.OSMinSeverity)
				}
			}
			if params.Heartbeat.Delivery.ThreadReplyMode != nil {
				settings.Heartbeat.Delivery.ThreadReplyMode = strings.TrimSpace(*params.Heartbeat.Delivery.ThreadReplyMode)
			}
		}
		if params.Heartbeat.Events != nil {
			if params.Heartbeat.Events.CronWakeMode != nil {
				settings.Heartbeat.Events.CronWakeMode = strings.TrimSpace(*params.Heartbeat.Events.CronWakeMode)
			}
			if params.Heartbeat.Events.ExecWakeMode != nil {
				settings.Heartbeat.Events.ExecWakeMode = strings.TrimSpace(*params.Heartbeat.Events.ExecWakeMode)
			}
			if params.Heartbeat.Events.SubagentWakeMode != nil {
				settings.Heartbeat.Events.SubagentWakeMode = strings.TrimSpace(*params.Heartbeat.Events.SubagentWakeMode)
			}
		}
	}
	if params.Subagents != nil {
		if params.Subagents.MaxDepth != nil {
			settings.Subagents.MaxDepth = *params.Subagents.MaxDepth
		}
		if params.Subagents.MaxChildren != nil {
			settings.Subagents.MaxChildren = *params.Subagents.MaxChildren
		}
		if params.Subagents.MaxConcurrent != nil {
			settings.Subagents.MaxConcurrent = *params.Subagents.MaxConcurrent
		}
		if params.Subagents.Model != nil {
			settings.Subagents.Model = strings.TrimSpace(*params.Subagents.Model)
		}
		if params.Subagents.Thinking != nil {
			settings.Subagents.Thinking = strings.TrimSpace(*params.Subagents.Thinking)
		}
		if params.Subagents.Tools != nil {
			settings.Subagents.Tools = GatewaySubagentToolPolicy{
				Allow:     normalizeStringSlice(params.Subagents.Tools.Allow),
				AlsoAllow: normalizeStringSlice(params.Subagents.Tools.AlsoAllow),
				Deny:      normalizeStringSlice(params.Subagents.Tools.Deny),
			}
		}
	}
	if params.Cron != nil {
		if params.Cron.Enabled != nil {
			settings.Cron.Enabled = *params.Cron.Enabled
		}
		if params.Cron.MaxConcurrentRuns != nil {
			settings.Cron.MaxConcurrentRuns = *params.Cron.MaxConcurrentRuns
		}
		if params.Cron.SessionRetention != nil {
			settings.Cron.SessionRetention = strings.TrimSpace(*params.Cron.SessionRetention)
		}
		if params.Cron.RunLog != nil {
			if params.Cron.RunLog.MaxBytes != nil {
				settings.Cron.RunLog.MaxBytes = strings.TrimSpace(*params.Cron.RunLog.MaxBytes)
			}
			if params.Cron.RunLog.KeepLines != nil {
				settings.Cron.RunLog.KeepLines = *params.Cron.RunLog.KeepLines
			}
		}
	}

	return settings
}

func ResolveGatewayDebugMode(value string, recordPrompt bool) GatewayDebugMode {
	switch GatewayDebugMode(strings.TrimSpace(value)) {
	case GatewayDebugModeOff, GatewayDebugModeBasic, GatewayDebugModeFull:
		return GatewayDebugMode(strings.TrimSpace(value))
	default:
		if recordPrompt {
			return GatewayDebugModeFull
		}
		return DefaultGatewayRuntimeDebugMode
	}
}

func GatewayDebugModeRecordsPrompt(mode GatewayDebugMode) bool {
	return mode == GatewayDebugModeFull
}

func ApplyGatewayDebugModeOverride(
	current GatewayDebugMode,
	override *string,
	recordPrompt *bool,
) GatewayDebugMode {
	mode := ResolveGatewayDebugMode(current.String(), GatewayDebugModeRecordsPrompt(current))
	if override != nil {
		return ResolveGatewayDebugMode(*override, mode == GatewayDebugModeFull)
	}
	if recordPrompt == nil {
		return mode
	}
	if *recordPrompt {
		return GatewayDebugModeFull
	}
	if mode == GatewayDebugModeFull {
		return GatewayDebugModeBasic
	}
	return mode
}

func normalizeStringSlice(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		result = append(result, trimmed)
	}
	return result
}
