package cron

import "time"

type CronSchedule struct {
	Kind      string `json:"kind,omitempty"`
	At        string `json:"at,omitempty"`
	EveryMs   int64  `json:"everyMs,omitempty"`
	AnchorMs  int64  `json:"anchorMs,omitempty"`
	Expr      string `json:"expr,omitempty"`
	TZ        string `json:"tz,omitempty"`
	StaggerMs int64  `json:"staggerMs,omitempty"`
}

type CronPayload struct {
	Kind           string `json:"kind,omitempty"`
	Text           string `json:"text,omitempty"`
	Message        string `json:"message,omitempty"`
	Model          string `json:"model,omitempty"`
	Thinking       string `json:"thinking,omitempty"`
	TimeoutSeconds int    `json:"timeoutSeconds,omitempty"`
	LightContext   bool   `json:"lightContext,omitempty"`
}

type CronFailureDestination struct {
	Mode      string `json:"mode,omitempty"`
	Channel   string `json:"channel,omitempty"`
	To        string `json:"to,omitempty"`
	AccountID string `json:"accountId,omitempty"`
}

type CronDelivery struct {
	Mode               string                  `json:"mode,omitempty"`
	Channel            string                  `json:"channel,omitempty"`
	To                 string                  `json:"to,omitempty"`
	AccountID          string                  `json:"accountId,omitempty"`
	BestEffort         bool                    `json:"bestEffort,omitempty"`
	FailureDestination *CronFailureDestination `json:"failureDestination,omitempty"`
}

type CronJobState struct {
	NextRunAtMs        int64  `json:"nextRunAtMs,omitempty"`
	RunningAtMs        int64  `json:"runningAtMs,omitempty"`
	LastRunAtMs        int64  `json:"lastRunAtMs,omitempty"`
	LastRunStatus      string `json:"lastRunStatus,omitempty"`
	LastError          string `json:"lastError,omitempty"`
	LastDurationMs     int64  `json:"lastDurationMs,omitempty"`
	ConsecutiveErrors  int    `json:"consecutiveErrors,omitempty"`
	ScheduleErrorCount int    `json:"scheduleErrorCount,omitempty"`
	LastDeliveryStatus string `json:"lastDeliveryStatus,omitempty"`
	LastDeliveryError  string `json:"lastDeliveryError,omitempty"`
	LastDelivered      *bool  `json:"lastDelivered,omitempty"`
}

type CronJob struct {
	ID             string        `json:"id,omitempty"`
	JobID          string        `json:"jobId"`
	Name           string        `json:"name,omitempty"`
	Description    string        `json:"description,omitempty"`
	AssistantID    string        `json:"assistantId,omitempty"`
	Enabled        bool          `json:"enabled"`
	DeleteAfterRun bool          `json:"deleteAfterRun,omitempty"`
	Schedule       CronSchedule  `json:"schedule,omitempty"`
	SessionTarget  string        `json:"sessionTarget,omitempty"`
	WakeMode       string        `json:"wakeMode,omitempty"`
	PayloadSpec    CronPayload   `json:"payload,omitempty"`
	Delivery       *CronDelivery `json:"delivery,omitempty"`
	State          CronJobState  `json:"state,omitempty"`
	SessionKey     string        `json:"sessionKey,omitempty"`
	CreatedAt      time.Time     `json:"createdAt"`
	UpdatedAt      time.Time     `json:"updatedAt"`
}

type CronRunRecord struct {
	RunID          string    `json:"runId"`
	JobID          string    `json:"jobId"`
	JobName        string    `json:"jobName,omitempty"`
	Status         string    `json:"status"`
	StartedAt      time.Time `json:"startedAt"`
	EndedAt        time.Time `json:"endedAt"`
	DeliveryStatus string    `json:"deliveryStatus,omitempty"`
	DeliveryError  string    `json:"deliveryError,omitempty"`
	Model          string    `json:"model,omitempty"`
	Provider       string    `json:"provider,omitempty"`
	SessionKey     string    `json:"sessionKey,omitempty"`
	Summary        string    `json:"summary,omitempty"`
	UsageJSON      string    `json:"usageJson,omitempty"`
	LatestStage    string    `json:"latestStage,omitempty"`
	Error          string    `json:"error,omitempty"`
}

type CronRunEvent struct {
	EventID    string         `json:"eventId"`
	RunID      string         `json:"runId"`
	JobID      string         `json:"jobId"`
	JobName    string         `json:"jobName,omitempty"`
	Stage      string         `json:"stage"`
	Status     string         `json:"status,omitempty"`
	Message    string         `json:"message,omitempty"`
	Error      string         `json:"error,omitempty"`
	Channel    string         `json:"channel,omitempty"`
	SessionKey string         `json:"sessionKey,omitempty"`
	Source     string         `json:"source,omitempty"`
	Meta       map[string]any `json:"meta,omitempty"`
	CreatedAt  time.Time      `json:"createdAt"`
}

type ListRunsQuery struct {
	JobID            string
	Status           string
	Statuses         []string
	DeliveryStatus   string
	DeliveryStatuses []string
	Query            string
	SortDir          string
	Limit            int
	Offset           int
}

type ListRunsResult struct {
	Items []CronRunRecord `json:"items"`
	Total int             `json:"total"`
}

type ListRunEventsQuery struct {
	RunID   string
	SortDir string
	Limit   int
	Offset  int
}

type ListRunEventsResult struct {
	Items []CronRunEvent `json:"items"`
	Total int            `json:"total"`
}

type RunDetail struct {
	Run         CronRunRecord  `json:"run"`
	Events      []CronRunEvent `json:"events"`
	EventsTotal int            `json:"eventsTotal"`
}

type WakeResult struct {
	OK         bool   `json:"ok"`
	Accepted   bool   `json:"accepted"`
	Mode       string `json:"mode,omitempty"`
	Text       string `json:"text,omitempty"`
	SessionKey string `json:"sessionKey,omitempty"`
}

type Status struct {
	Enabled      bool   `json:"enabled"`
	Jobs         int    `json:"jobs"`
	NextWakeAt   string `json:"nextWakeAt,omitempty"`
	NextWakeAtMs int64  `json:"nextWakeAtMs,omitempty"`
}
