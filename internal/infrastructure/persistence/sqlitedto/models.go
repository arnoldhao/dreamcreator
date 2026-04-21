package sqlitedto

import (
	"database/sql"
	"time"

	"github.com/uptrace/bun"
)

type AgentRow struct {
	bun.BaseModel `bun:"table:agents"`

	ID          string         `bun:"id,pk"`
	Name        string         `bun:"name"`
	Description sql.NullString `bun:"description"`
	Enabled     bool           `bun:"enabled"`
	ThreadID    string         `bun:"thread_id"`
	CreatedAt   time.Time      `bun:"created_at"`
	UpdatedAt   time.Time      `bun:"updated_at"`
	DeletedAt   sql.NullTime   `bun:"deleted_at"`
}

type ApprovalRow struct {
	bun.BaseModel `bun:"table:exec_approvals"`

	ID          string         `bun:"id,pk"`
	RequestJSON string         `bun:"request_json"`
	Status      string         `bun:"status"`
	Decision    sql.NullString `bun:"decision"`
	CreatedAt   time.Time      `bun:"created_at"`
	ResolvedAt  sql.NullTime   `bun:"resolved_at"`
}

type AssistantRow struct {
	bun.BaseModel `bun:"table:assistants"`

	ID           string         `bun:"id,pk"`
	IdentityJSON sql.NullString `bun:"identity_json"`
	AvatarJSON   sql.NullString `bun:"avatar_json"`
	UserJSON     sql.NullString `bun:"user_json"`
	ModelJSON    sql.NullString `bun:"model_json"`
	ToolsJSON    sql.NullString `bun:"tools_json"`
	SkillsJSON   sql.NullString `bun:"skills_json"`
	CallJSON     sql.NullString `bun:"call_json"`
	MemoryJSON   sql.NullString `bun:"memory_json"`
	Builtin      bool           `bun:"builtin"`
	Deletable    bool           `bun:"deletable"`
	Enabled      bool           `bun:"enabled"`
	IsDefault    bool           `bun:"is_default"`
	CreatedAt    time.Time      `bun:"created_at"`
	UpdatedAt    time.Time      `bun:"updated_at"`
}

type AutomationJobRow struct {
	bun.BaseModel `bun:"table:automation_jobs"`

	ID         string    `bun:"id,pk"`
	Kind       string    `bun:"kind"`
	Status     string    `bun:"status"`
	ConfigJSON string    `bun:"config_json"`
	CreatedAt  time.Time `bun:"created_at"`
	UpdatedAt  time.Time `bun:"updated_at"`
}

type AutomationRunRow struct {
	bun.BaseModel `bun:"table:automation_runs"`

	ID        string    `bun:"id,pk"`
	JobID     string    `bun:"job_id"`
	Status    string    `bun:"status"`
	ErrorText string    `bun:"error"`
	StartedAt time.Time `bun:"started_at"`
	EndedAt   time.Time `bun:"ended_at"`
}

type CronJobRow struct {
	bun.BaseModel `bun:"table:cron_jobs"`

	ID             string `bun:"id,pk"`
	AssistantID    string `bun:"assistant_id"`
	Name           string `bun:"name"`
	Description    string `bun:"description"`
	Enabled        bool   `bun:"enabled"`
	DeleteAfterRun bool   `bun:"delete_after_run"`
	ScheduleJSON   string `bun:"schedule_json"`
	PayloadJSON    string `bun:"payload_json"`
	DeliveryJSON   string `bun:"delivery_json"`
	SessionTarget  string `bun:"session_target"`
	WakeMode       string `bun:"wake_mode"`
	SessionKey     string `bun:"session_key"`
	StateJSON      string `bun:"state_json"`
	CreatedAtMs    int64  `bun:"created_at_ms"`
	UpdatedAtMs    int64  `bun:"updated_at_ms"`
}

type CronRunRow struct {
	bun.BaseModel `bun:"table:cron_runs"`

	RunID          string `bun:"run_id,pk"`
	JobID          string `bun:"job_id"`
	Status         string `bun:"status"`
	ErrorText      string `bun:"error"`
	Summary        string `bun:"summary"`
	DeliveryStatus string `bun:"delivery_status"`
	DeliveryError  string `bun:"delivery_error"`
	SessionKey     string `bun:"session_key"`
	Model          string `bun:"model"`
	Provider       string `bun:"provider"`
	UsageJSON      string `bun:"usage_json"`
	RunAtMs        int64  `bun:"run_at_ms"`
	DurationMs     int64  `bun:"duration_ms"`
	CreatedAtMs    int64  `bun:"created_at_ms"`
	EndedAtMs      int64  `bun:"ended_at_ms"`
}

type CronRunEventRow struct {
	bun.BaseModel `bun:"table:cron_run_events"`

	EventID     string `bun:"event_id,pk"`
	RunID       string `bun:"run_id"`
	JobID       string `bun:"job_id"`
	JobName     string `bun:"job_name"`
	Stage       string `bun:"stage"`
	Status      string `bun:"status"`
	Message     string `bun:"message"`
	ErrorText   string `bun:"error"`
	Channel     string `bun:"channel"`
	SessionKey  string `bun:"session_key"`
	Source      string `bun:"source"`
	MetaJSON    string `bun:"meta_json"`
	CreatedAtMs int64  `bun:"created_at_ms"`
}

type AutomationTriggerRow struct {
	bun.BaseModel `bun:"table:automation_trigger_logs"`

	ID          int64     `bun:"id,pk,autoincrement"`
	JobID       string    `bun:"job_id"`
	EventID     string    `bun:"event_id"`
	PayloadJSON string    `bun:"payload_json"`
	CreatedAt   time.Time `bun:"created_at"`
}

type AuditRow struct {
	bun.BaseModel `bun:"table:tool_policy_audit"`

	ID          int64     `bun:"id,pk,autoincrement"`
	ToolID      string    `bun:"tool_id"`
	Decision    string    `bun:"decision"`
	Reason      string    `bun:"reason"`
	ContextJSON string    `bun:"context_json"`
	CreatedAt   time.Time `bun:"created_at"`
}

type ConnectorRow struct {
	bun.BaseModel `bun:"table:connectors"`

	ID             string         `bun:"id,pk"`
	Type           string         `bun:"type"`
	Status         string         `bun:"status"`
	CookiesPath    sql.NullString `bun:"cookies_path"`
	CookiesJSON    sql.NullString `bun:"cookies_json"`
	LastVerifiedAt sql.NullTime   `bun:"last_verified_at"`
	CreatedAt      time.Time      `bun:"created_at"`
	UpdatedAt      time.Time      `bun:"updated_at"`
}

type DiagnosticReportRow struct {
	bun.BaseModel `bun:"table:diagnostic_reports"`

	ID          string    `bun:"id,pk"`
	PayloadJSON string    `bun:"payload_json"`
	CreatedAt   time.Time `bun:"created_at"`
}

type TelemetryStateRow struct {
	bun.BaseModel `bun:"table:telemetry_state"`

	ID                        int          `bun:"id,pk"`
	InstallID                 string       `bun:"install_id"`
	InstallCreatedAt          time.Time    `bun:"install_created_at"`
	LaunchCount               int          `bun:"launch_count"`
	FirstProviderConfiguredAt sql.NullTime `bun:"first_provider_configured_at"`
	FirstChatCompletedAt      sql.NullTime `bun:"first_chat_completed_at"`
	FirstLibraryCompletedAt   sql.NullTime `bun:"first_library_completed_at"`
	UpdatedAt                 time.Time    `bun:"updated_at"`
}

type ExternalToolRow struct {
	bun.BaseModel `bun:"table:external_tools"`

	Name        string         `bun:"name,pk"`
	ExecPath    sql.NullString `bun:"exec_path"`
	Version     sql.NullString `bun:"version"`
	Status      sql.NullString `bun:"status"`
	InstalledAt sql.NullTime   `bun:"installed_at"`
	UpdatedAt   time.Time      `bun:"updated_at"`
}

type GatewayEventRow struct {
	bun.BaseModel `bun:"table:gateway_events"`

	ID          string         `bun:"id,pk"`
	EventType   string         `bun:"event_type"`
	SessionID   sql.NullString `bun:"session_id"`
	SessionKey  sql.NullString `bun:"session_key"`
	PayloadJSON string         `bun:"payload_json"`
	CreatedAt   time.Time      `bun:"created_at"`
}

type GatewaySessionRow struct {
	bun.BaseModel `bun:"table:gateway_sessions"`

	SessionID                  string         `bun:"session_id,pk"`
	SessionKey                 string         `bun:"session_key"`
	AgentID                    sql.NullString `bun:"agent_id"`
	AssistantID                sql.NullString `bun:"assistant_id"`
	Title                      sql.NullString `bun:"title"`
	Status                     sql.NullString `bun:"status"`
	OriginJSON                 sql.NullString `bun:"origin_json"`
	ContextPromptTokens        sql.NullInt64  `bun:"context_prompt_tokens"`
	ContextTotalTokens         sql.NullInt64  `bun:"context_total_tokens"`
	ContextWindowTokens        sql.NullInt64  `bun:"context_window_tokens"`
	ContextUpdatedAt           sql.NullTime   `bun:"context_updated_at"`
	ContextFresh               sql.NullBool   `bun:"context_fresh"`
	ContextSummary             sql.NullString `bun:"context_summary"`
	ContextFirstKeptMessageID  sql.NullString `bun:"context_first_kept_message_id"`
	ContextStrategyVersion     sql.NullInt64  `bun:"context_strategy_version"`
	ContextCompactedAt         sql.NullTime   `bun:"context_compacted_at"`
	CompactionCount            sql.NullInt64  `bun:"compaction_count"`
	MemoryFlushCompactionCount sql.NullInt64  `bun:"memory_flush_compaction_count"`
	CreatedAt                  time.Time      `bun:"created_at"`
	UpdatedAt                  time.Time      `bun:"updated_at"`
}

type GlobalWorkspaceRow struct {
	bun.BaseModel `bun:"table:global_workspaces"`

	ID                       int            `bun:"id,pk"`
	DefaultExecutorModelJSON sql.NullString `bun:"default_executor_model_json"`
	DefaultMemoryJSON        sql.NullString `bun:"default_memory_json"`
	DefaultPersona           sql.NullString `bun:"default_persona"`
	CreatedAt                time.Time      `bun:"created_at"`
	UpdatedAt                time.Time      `bun:"updated_at"`
}

type HeartbeatEventRow struct {
	bun.BaseModel `bun:"table:heartbeat_events"`

	ID          string    `bun:"id,pk"`
	SessionKey  string    `bun:"session_key"`
	ThreadID    string    `bun:"thread_id"`
	Status      string    `bun:"status"`
	Message     string    `bun:"message"`
	ErrorText   string    `bun:"error"`
	ContentHash string    `bun:"content_hash"`
	Reason      string    `bun:"reason"`
	Source      string    `bun:"source"`
	RunID       string    `bun:"run_id"`
	CreatedAt   time.Time `bun:"created_at"`
}

type NoticeRow struct {
	bun.BaseModel `bun:"table:notices"`

	ID              string       `bun:"id,pk"`
	Kind            string       `bun:"kind"`
	Category        string       `bun:"category"`
	Code            string       `bun:"code"`
	Severity        string       `bun:"severity"`
	Status          string       `bun:"status"`
	I18nJSON        string       `bun:"i18n_json"`
	SourceJSON      string       `bun:"source_json"`
	ActionJSON      string       `bun:"action_json"`
	SurfacesJSON    string       `bun:"surfaces_json"`
	DedupKey        string       `bun:"dedup_key"`
	OccurrenceCount int          `bun:"occurrence_count"`
	MetadataJSON    string       `bun:"metadata_json"`
	CreatedAt       time.Time    `bun:"created_at"`
	UpdatedAt       time.Time    `bun:"updated_at"`
	LastOccurredAt  time.Time    `bun:"last_occurred_at"`
	ReadAt          sql.NullTime `bun:"read_at"`
	ArchivedAt      sql.NullTime `bun:"archived_at"`
	ExpiresAt       sql.NullTime `bun:"expires_at"`
}

type ModelRow struct {
	bun.BaseModel `bun:"table:provider_models"`

	ID                string         `bun:"id,pk"`
	ProviderID        string         `bun:"provider_id"`
	Name              string         `bun:"name"`
	DisplayName       sql.NullString `bun:"display_name"`
	CapabilitiesJSON  sql.NullString `bun:"capabilities_json"`
	ContextWindow     sql.NullInt64  `bun:"context_window_tokens"`
	MaxOutputTokens   sql.NullInt64  `bun:"max_output_tokens"`
	SupportsTools     sql.NullBool   `bun:"supports_tools"`
	SupportsReasoning sql.NullBool   `bun:"supports_reasoning"`
	SupportsVision    sql.NullBool   `bun:"supports_vision"`
	SupportsAudio     sql.NullBool   `bun:"supports_audio"`
	SupportsVideo     sql.NullBool   `bun:"supports_video"`
	Enabled           bool           `bun:"enabled"`
	ShowInUI          bool           `bun:"show_in_ui"`
	CreatedAt         time.Time      `bun:"created_at"`
	UpdatedAt         time.Time      `bun:"updated_at"`
}

type NodeRow struct {
	bun.BaseModel `bun:"table:node_registry"`

	NodeID       string         `bun:"node_id,pk"`
	DisplayName  sql.NullString `bun:"display_name"`
	Platform     sql.NullString `bun:"platform"`
	Version      sql.NullString `bun:"version"`
	Capabilities sql.NullString `bun:"capabilities_json"`
	Status       sql.NullString `bun:"status"`
	UpdatedAt    time.Time      `bun:"updated_at"`
}

type NodeInvokeRow struct {
	bun.BaseModel `bun:"table:node_invoke_logs"`

	ID         string    `bun:"id,pk"`
	NodeID     string    `bun:"node_id"`
	Capability string    `bun:"capability"`
	Action     string    `bun:"action"`
	ArgsJSON   string    `bun:"args_json"`
	Status     string    `bun:"status"`
	OutputJSON string    `bun:"output_json"`
	ErrorText  string    `bun:"error_text"`
	CreatedAt  time.Time `bun:"created_at"`
}

type ProviderRow struct {
	bun.BaseModel `bun:"table:providers"`

	ID            string         `bun:"id,pk"`
	Name          string         `bun:"name"`
	Type          string         `bun:"type"`
	Compatibility string         `bun:"compatibility"`
	Endpoint      sql.NullString `bun:"endpoint"`
	Enabled       bool           `bun:"enabled"`
	Builtin       bool           `bun:"is_builtin"`
	CreatedAt     time.Time      `bun:"created_at"`
	UpdatedAt     time.Time      `bun:"updated_at"`
}

type ProviderSecretRow struct {
	bun.BaseModel `bun:"table:provider_secrets"`

	ID         string         `bun:"id,pk"`
	ProviderID string         `bun:"provider_id"`
	KeyRef     sql.NullString `bun:"key_ref"`
	OrgRef     sql.NullString `bun:"org_ref"`
	CreatedAt  time.Time      `bun:"created_at"`
}

type QueueRow struct {
	bun.BaseModel `bun:"table:gateway_queue_tickets"`

	TicketID   string    `bun:"ticket_id,pk"`
	SessionKey string    `bun:"session_key"`
	Lane       string    `bun:"lane"`
	Status     string    `bun:"status"`
	Position   int       `bun:"position"`
	CreatedAt  time.Time `bun:"created_at"`
}

type RevisionRow struct {
	bun.BaseModel `bun:"table:config_revisions"`

	ID          string    `bun:"id,pk"`
	Version     string    `bun:"version"`
	PayloadJSON string    `bun:"payload_json"`
	CreatedAt   time.Time `bun:"created_at"`
}

type SettingsRow struct {
	bun.BaseModel `bun:"table:settings"`

	ID int `bun:"id,pk"`

	Appearance            string          `bun:"appearance"`
	FontFamily            sql.NullString  `bun:"font_family"`
	ThemeColor            sql.NullString  `bun:"theme_color"`
	ColorScheme           sql.NullString  `bun:"color_scheme"`
	FontSize              sql.NullInt64   `bun:"font_size"`
	Language              sql.NullString  `bun:"language"`
	DownloadDirectory     sql.NullString  `bun:"download_directory"`
	LogLevel              sql.NullString  `bun:"log_level"`
	LogMaxSize            sql.NullInt64   `bun:"log_max_size_mb"`
	LogBackups            sql.NullInt64   `bun:"log_max_backups"`
	LogAge                sql.NullInt64   `bun:"log_max_age_days"`
	LogCompress           sql.NullBool    `bun:"log_compress"`
	MenuBarVisibility     sql.NullString  `bun:"menu_bar_visibility"`
	AutoStart             sql.NullBool    `bun:"auto_start"`
	MinimizeToTrayOnStart sql.NullBool    `bun:"minimize_to_tray_on_start"`
	AgentModelProviderID  sql.NullString  `bun:"agent_model_provider_id"`
	AgentModelName        sql.NullString  `bun:"agent_model_name"`
	AgentStreamEnabled    sql.NullBool    `bun:"chat_stream_enabled"`
	ChatTemperature       sql.NullFloat64 `bun:"chat_temperature"`
	ChatMaxTokens         sql.NullInt64   `bun:"chat_max_tokens"`
	SkillsJSON            sql.NullString  `bun:"skills_json"`
	GatewayFlagsJSON      sql.NullString  `bun:"gateway_flags_json"`
	MemoryJSON            sql.NullString  `bun:"memory_json"`
	ToolsConfigJSON       sql.NullString  `bun:"tools_config_json"`
	SkillsConfigJSON      sql.NullString  `bun:"skills_config_json"`
	CommandsJSON          sql.NullString  `bun:"commands_json"`
	ChannelsJSON          sql.NullString  `bun:"channels_json"`

	MainX      sql.NullInt64 `bun:"main_x"`
	MainY      sql.NullInt64 `bun:"main_y"`
	MainWidth  sql.NullInt64 `bun:"main_width"`
	MainHeight sql.NullInt64 `bun:"main_height"`

	SettingsX      sql.NullInt64 `bun:"settings_x"`
	SettingsY      sql.NullInt64 `bun:"settings_y"`
	SettingsWidth  sql.NullInt64 `bun:"settings_width"`
	SettingsHeight sql.NullInt64 `bun:"settings_height"`

	Version int `bun:"version"`

	ProxyMode           sql.NullString `bun:"proxy_mode"`
	ProxyScheme         sql.NullString `bun:"proxy_scheme"`
	ProxyHost           sql.NullString `bun:"proxy_host"`
	ProxyPort           sql.NullInt64  `bun:"proxy_port"`
	ProxyUsername       sql.NullString `bun:"proxy_username"`
	ProxyPassword       sql.NullString `bun:"proxy_password"`
	ProxyNoProxy        sql.NullString `bun:"proxy_no_proxy"`
	ProxyTimeoutSeconds sql.NullInt64  `bun:"proxy_timeout_seconds"`
	ProxyTestedAt       sql.NullTime   `bun:"proxy_tested_at"`
	ProxyTestSuccess    sql.NullBool   `bun:"proxy_test_success"`
	ProxyTestMessage    sql.NullString `bun:"proxy_test_message"`
}

type SubagentRunRow struct {
	bun.BaseModel `bun:"table:subagent_runs"`

	RunID                 string         `bun:"run_id,pk"`
	ParentSessionKey      sql.NullString `bun:"parent_session_key"`
	ParentRunID           sql.NullString `bun:"parent_run_id"`
	AgentID               sql.NullString `bun:"agent_id"`
	ChildSessionKey       sql.NullString `bun:"child_session_key"`
	ChildSessionID        sql.NullString `bun:"child_session_id"`
	Task                  sql.NullString `bun:"task"`
	Label                 sql.NullString `bun:"label"`
	Model                 sql.NullString `bun:"model"`
	Thinking              sql.NullString `bun:"thinking"`
	CallerModel           sql.NullString `bun:"caller_model"`
	CallerThinking        sql.NullString `bun:"caller_thinking"`
	CleanupPolicy         sql.NullString `bun:"cleanup_policy"`
	RunTimeoutSeconds     sql.NullInt64  `bun:"run_timeout_seconds"`
	ResultText            sql.NullString `bun:"result_text"`
	Notes                 sql.NullString `bun:"notes"`
	RuntimeMs             sql.NullInt64  `bun:"runtime_ms"`
	UsagePromptTokens     sql.NullInt64  `bun:"usage_prompt_tokens"`
	UsageCompletionTokens sql.NullInt64  `bun:"usage_completion_tokens"`
	UsageTotalTokens      sql.NullInt64  `bun:"usage_total_tokens"`
	TranscriptPath        sql.NullString `bun:"transcript_path"`
	Status                sql.NullString `bun:"status"`
	Summary               sql.NullString `bun:"summary"`
	ErrorText             sql.NullString `bun:"error_text"`
	AnnounceKey           sql.NullString `bun:"announce_key"`
	AnnounceAttempts      sql.NullInt64  `bun:"announce_attempts"`
	AnnounceSentAt        sql.NullTime   `bun:"announce_sent_at"`
	FinishedAt            sql.NullTime   `bun:"finished_at"`
	ArchivedAt            sql.NullTime   `bun:"archived_at"`
	CreatedAt             time.Time      `bun:"created_at"`
	UpdatedAt             time.Time      `bun:"updated_at"`
}

type ThreadMessageRow struct {
	bun.BaseModel `bun:"table:thread_messages"`

	ID        string    `bun:"id,pk"`
	ThreadID  string    `bun:"thread_id"`
	Kind      string    `bun:"kind"`
	Role      string    `bun:"role"`
	Content   string    `bun:"content"`
	PartsJSON string    `bun:"parts_json"`
	CreatedAt time.Time `bun:"created_at"`
}

type ThreadRow struct {
	bun.BaseModel `bun:"table:threads"`

	ID                string         `bun:"id,pk"`
	AgentID           string         `bun:"agent_id"`
	AssistantID       string         `bun:"assistant_id"`
	Title             sql.NullString `bun:"title"`
	TitleIsDefault    bool           `bun:"title_is_default"`
	TitleChangedBy    sql.NullString `bun:"title_changed_by"`
	Status            string         `bun:"status"`
	CreatedAt         time.Time      `bun:"created_at"`
	UpdatedAt         time.Time      `bun:"updated_at"`
	LastInteractiveAt time.Time      `bun:"last_interactive_at"`
	DeletedAt         sql.NullTime   `bun:"deleted_at"`
	PurgeAfter        sql.NullTime   `bun:"purge_after"`
}

type ThreadRunEventRow struct {
	bun.BaseModel `bun:"table:agent_events"`

	ID          int64     `bun:"id,pk,autoincrement"`
	RunID       string    `bun:"run_id"`
	ThreadID    string    `bun:"thread_id"`
	EventType   string    `bun:"event_name"`
	PayloadJSON string    `bun:"payload_json"`
	CreatedAt   time.Time `bun:"created_at"`
}

type ThreadRunRow struct {
	bun.BaseModel `bun:"table:thread_runs"`

	ID                 string    `bun:"id,pk"`
	ThreadID           string    `bun:"thread_id"`
	AssistantMessageID string    `bun:"assistant_message_id"`
	UserMessageID      string    `bun:"user_message_id"`
	AgentID            string    `bun:"agent_id"`
	Status             string    `bun:"status"`
	ContentPartial     string    `bun:"content_partial"`
	CreatedAt          time.Time `bun:"created_at"`
	UpdatedAt          time.Time `bun:"updated_at"`
}

type ToolRunRow struct {
	bun.BaseModel `bun:"table:tool_runs"`

	ID         string     `bun:",pk"`
	RunID      string     `bun:"run_id"`
	ToolCallID string     `bun:"tool_call_id"`
	ToolName   string     `bun:"tool_name"`
	InputHash  string     `bun:"input_hash"`
	InputJSON  string     `bun:"input_json"`
	OutputJSON string     `bun:"output_json"`
	ErrorText  string     `bun:"error_text"`
	JobID      string     `bun:"job_id"`
	Status     string     `bun:"status"`
	CreatedAt  time.Time  `bun:"created_at"`
	StartedAt  *time.Time `bun:"started_at"`
	FinishedAt *time.Time `bun:"finished_at"`
}

type TranscodePresetRow struct {
	bun.BaseModel `bun:"table:transcode_presets"`

	ID               string         `bun:"id,pk"`
	Name             string         `bun:"name"`
	OutputType       string         `bun:"output_type"`
	Container        string         `bun:"container"`
	VideoCodec       sql.NullString `bun:"video_codec"`
	AudioCodec       sql.NullString `bun:"audio_codec"`
	QualityMode      sql.NullString `bun:"quality_mode"`
	CRF              sql.NullInt64  `bun:"crf"`
	BitrateKbps      sql.NullInt64  `bun:"bitrate_kbps"`
	AudioBitrateKbps sql.NullInt64  `bun:"audio_bitrate_kbps"`
	Scale            sql.NullString `bun:"scale"`
	Width            sql.NullInt64  `bun:"width"`
	Height           sql.NullInt64  `bun:"height"`
	FFmpegPreset     sql.NullString `bun:"ffmpeg_preset"`
	AllowUpscale     bool           `bun:"allow_upscale"`
	RequiresVideo    bool           `bun:"requires_video"`
	RequiresAudio    bool           `bun:"requires_audio"`
	IsBuiltin        bool           `bun:"is_builtin"`
	Description      sql.NullString `bun:"description"`
	CreatedAt        time.Time      `bun:"created_at"`
	UpdatedAt        time.Time      `bun:"updated_at"`
}

type TtsJobRow struct {
	bun.BaseModel `bun:"table:tts_jobs"`

	ID         string         `bun:"id,pk"`
	ProviderID sql.NullString `bun:"provider_id"`
	VoiceID    sql.NullString `bun:"voice_id"`
	ModelID    sql.NullString `bun:"model_id"`
	Format     sql.NullString `bun:"format"`
	Status     sql.NullString `bun:"status"`
	InputText  sql.NullString `bun:"input_text"`
	OutputJSON sql.NullString `bun:"output_json"`
	CostMicros sql.NullInt64  `bun:"cost_micros"`
	CreatedAt  time.Time      `bun:"created_at"`
}

type LLMCallRecordRow struct {
	bun.BaseModel `bun:"table:llm_call_records"`

	ID                  string         `bun:"id,pk"`
	SessionID           sql.NullString `bun:"session_id"`
	ThreadID            sql.NullString `bun:"thread_id"`
	RunID               sql.NullString `bun:"run_id"`
	ProviderID          sql.NullString `bun:"provider_id"`
	ModelName           sql.NullString `bun:"model_name"`
	RequestSource       sql.NullString `bun:"request_source"`
	Operation           sql.NullString `bun:"operation"`
	Status              string         `bun:"status"`
	FinishReason        sql.NullString `bun:"finish_reason"`
	ErrorText           sql.NullString `bun:"error_text"`
	InputTokens         sql.NullInt64  `bun:"input_tokens"`
	OutputTokens        sql.NullInt64  `bun:"output_tokens"`
	TotalTokens         sql.NullInt64  `bun:"total_tokens"`
	ContextPromptTokens sql.NullInt64  `bun:"context_prompt_tokens"`
	ContextTotalTokens  sql.NullInt64  `bun:"context_total_tokens"`
	ContextWindowTokens sql.NullInt64  `bun:"context_window_tokens"`
	RequestPayloadJSON  sql.NullString `bun:"request_payload_json"`
	ResponsePayloadJSON sql.NullString `bun:"response_payload_json"`
	PayloadTruncated    bool           `bun:"payload_truncated"`
	StartedAt           time.Time      `bun:"started_at"`
	FinishedAt          sql.NullTime   `bun:"finished_at"`
	DurationMS          sql.NullInt64  `bun:"duration_ms"`
}

type UsageLedgerRow struct {
	bun.BaseModel `bun:"table:usage_ledger"`

	ID               string         `bun:"id,pk"`
	Category         sql.NullString `bun:"category"`
	ProviderID       sql.NullString `bun:"provider_id"`
	ModelName        sql.NullString `bun:"model_name"`
	Channel          sql.NullString `bun:"channel"`
	RequestID        sql.NullString `bun:"request_id"`
	RequestSource    sql.NullString `bun:"request_source"`
	Units            sql.NullInt64  `bun:"units"`
	PromptTokens     sql.NullInt64  `bun:"prompt_tokens"`
	CompletionTokens sql.NullInt64  `bun:"completion_tokens"`
	CostMicros       sql.NullInt64  `bun:"cost_micros"`
	CreatedAt        time.Time      `bun:"created_at"`
}

type UsageEventRow struct {
	bun.BaseModel `bun:"table:usage_events"`

	ID                string         `bun:"id,pk"`
	RequestID         string         `bun:"request_id"`
	StepID            string         `bun:"step_id"`
	ProviderID        string         `bun:"provider_id"`
	ModelName         string         `bun:"model_name"`
	Category          sql.NullString `bun:"category"`
	Channel           sql.NullString `bun:"channel"`
	RequestSource     string         `bun:"request_source"`
	UsageStatus       sql.NullString `bun:"usage_status"`
	InputTokens       sql.NullInt64  `bun:"input_tokens"`
	OutputTokens      sql.NullInt64  `bun:"output_tokens"`
	TotalTokens       sql.NullInt64  `bun:"total_tokens"`
	CachedInputTokens sql.NullInt64  `bun:"cached_input_tokens"`
	ReasoningTokens   sql.NullInt64  `bun:"reasoning_tokens"`
	AudioInputTokens  sql.NullInt64  `bun:"audio_input_tokens"`
	AudioOutputTokens sql.NullInt64  `bun:"audio_output_tokens"`
	RawUsageJSON      sql.NullString `bun:"raw_usage_json"`
	OccurredAt        time.Time      `bun:"occurred_at"`
	CreatedAt         time.Time      `bun:"created_at"`
	UpdatedAt         time.Time      `bun:"updated_at"`
}

type UsagePricingVersionRow struct {
	bun.BaseModel `bun:"table:model_pricing_versions"`

	ID                    string         `bun:"id,pk"`
	ProviderID            string         `bun:"provider_id"`
	ModelName             string         `bun:"model_name"`
	Currency              string         `bun:"currency"`
	InputPerMillion       float64        `bun:"input_per_million"`
	OutputPerMillion      float64        `bun:"output_per_million"`
	CachedInputPerMillion float64        `bun:"cached_input_per_million"`
	ReasoningPerMillion   float64        `bun:"reasoning_per_million"`
	AudioInputPerMillion  float64        `bun:"audio_input_per_million"`
	AudioOutputPerMillion float64        `bun:"audio_output_per_million"`
	PerRequest            float64        `bun:"per_request"`
	Source                string         `bun:"source"`
	EffectiveFrom         time.Time      `bun:"effective_from"`
	EffectiveTo           sql.NullTime   `bun:"effective_to"`
	IsActive              bool           `bun:"is_active"`
	UpdatedBy             sql.NullString `bun:"updated_by"`
	CreatedAt             time.Time      `bun:"created_at"`
	UpdatedAt             time.Time      `bun:"updated_at"`
}

type UsageLedgerEntryRow struct {
	bun.BaseModel `bun:"table:usage_ledger_entries"`

	ID                    string         `bun:"id,pk"`
	EventID               string         `bun:"event_id"`
	RequestID             string         `bun:"request_id"`
	Category              string         `bun:"category"`
	ProviderID            string         `bun:"provider_id"`
	ModelName             string         `bun:"model_name"`
	Channel               sql.NullString `bun:"channel"`
	RequestSource         string         `bun:"request_source"`
	CostBasis             string         `bun:"cost_basis"`
	PricingVersionID      string         `bun:"pricing_version_id"`
	Units                 sql.NullInt64  `bun:"units"`
	InputTokens           sql.NullInt64  `bun:"input_tokens"`
	OutputTokens          sql.NullInt64  `bun:"output_tokens"`
	CachedInputTokens     sql.NullInt64  `bun:"cached_input_tokens"`
	ReasoningTokens       sql.NullInt64  `bun:"reasoning_tokens"`
	InputCostMicros       sql.NullInt64  `bun:"input_cost_micros"`
	OutputCostMicros      sql.NullInt64  `bun:"output_cost_micros"`
	CachedInputCostMicros sql.NullInt64  `bun:"cached_input_cost_micros"`
	ReasoningCostMicros   sql.NullInt64  `bun:"reasoning_cost_micros"`
	RequestCostMicros     sql.NullInt64  `bun:"request_cost_micros"`
	TotalCostMicros       sql.NullInt64  `bun:"total_cost_micros"`
	CreatedAt             time.Time      `bun:"created_at"`
}

type VoiceConfigRow struct {
	bun.BaseModel `bun:"table:voicewake_config"`

	ID             string         `bun:"id,pk"`
	Version        int            `bun:"version"`
	TriggersJSON   sql.NullString `bun:"triggers_json"`
	TTSConfigJSON  sql.NullString `bun:"tts_config_json"`
	TalkConfigJSON sql.NullString `bun:"talk_config_json"`
	UpdatedAt      time.Time      `bun:"updated_at"`
}

type WorkspaceRow struct {
	bun.BaseModel `bun:"table:assistant_workspaces"`

	AssistantID       string         `bun:"assistant_id,pk"`
	Version           int64          `bun:"version"`
	IdentityJSON      sql.NullString `bun:"identity_json"`
	PersonaText       sql.NullString `bun:"persona_text"`
	UserProfileJSON   sql.NullString `bun:"user_profile_json"`
	ToolingJSON       sql.NullString `bun:"tooling_json"`
	MemoryJSON        sql.NullString `bun:"memory_json"`
	MemoryConfigJSON  sql.NullString `bun:"memory_config_json"`
	ExtraFilesJSON    sql.NullString `bun:"extra_files_json"`
	PromptModeDefault sql.NullString `bun:"prompt_mode_default"`
	CreatedAt         time.Time      `bun:"created_at"`
	UpdatedAt         time.Time      `bun:"updated_at"`
}

type WorkspaceSnapshotRow struct {
	bun.BaseModel `bun:"table:assistant_workspace_snapshots"`

	ID                string         `bun:"id,pk"`
	AssistantID       string         `bun:"assistant_id"`
	WorkspaceVersion  string         `bun:"workspace_version"`
	LogicalFilesJSON  sql.NullString `bun:"logical_files_json"`
	PromptModeDefault sql.NullString `bun:"prompt_mode_default"`
	ToolHintsJSON     sql.NullString `bun:"tool_hints_json"`
	SkillHintsJSON    sql.NullString `bun:"skill_hints_json"`
	GeneratedAt       sql.NullTime   `bun:"generated_at"`
	CreatedAt         time.Time      `bun:"created_at"`
}
