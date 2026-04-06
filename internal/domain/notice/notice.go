package notice

import (
	"context"
	"errors"
	"time"
)

var ErrNoticeNotFound = errors.New("notice not found")

type Kind string

const (
	KindRuntimeEvent  Kind = "runtime_event"
	KindSystemStatus  Kind = "system_status"
	KindProductUpdate Kind = "product_update"
)

type Category string

const (
	CategoryHeartbeat Category = "heartbeat"
	CategoryCron      Category = "cron"
	CategorySubagent  Category = "subagent"
	CategoryExec      Category = "exec"
	CategoryGateway   Category = "gateway"
	CategoryUpdate    Category = "update"
)

type Severity string

const (
	SeverityInfo     Severity = "info"
	SeveritySuccess  Severity = "success"
	SeverityWarning  Severity = "warning"
	SeverityError    Severity = "error"
	SeverityCritical Severity = "critical"
)

type Status string

const (
	StatusUnread   Status = "unread"
	StatusRead     Status = "read"
	StatusArchived Status = "archived"
)

type Surface string

const (
	SurfaceCenter Surface = "center"
	SurfaceToast  Surface = "toast"
	SurfacePopup  Surface = "popup"
	SurfaceOS     Surface = "os"
	SurfaceFooter Surface = "footer"
)

type Source struct {
	Producer   string            `json:"producer"`
	SessionKey string            `json:"sessionKey"`
	ThreadID   string            `json:"threadId"`
	RunID      string            `json:"runId"`
	JobID      string            `json:"jobId"`
	Channel    string            `json:"channel"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

type Action struct {
	Type     string            `json:"type"`
	LabelKey string            `json:"labelKey"`
	Target   string            `json:"target"`
	Params   map[string]string `json:"params,omitempty"`
}

type I18n struct {
	TitleKey   string            `json:"titleKey"`
	SummaryKey string            `json:"summaryKey"`
	BodyKey    string            `json:"bodyKey"`
	Params     map[string]string `json:"params,omitempty"`
}

type Notice struct {
	ID              string         `json:"id"`
	Kind            Kind           `json:"kind"`
	Category        Category       `json:"category"`
	Code            string         `json:"code"`
	Severity        Severity       `json:"severity"`
	Status          Status         `json:"status"`
	I18n            I18n           `json:"i18n"`
	Source          Source         `json:"source"`
	Action          Action         `json:"action"`
	Surfaces        []Surface      `json:"surfaces"`
	DedupKey        string         `json:"dedupKey"`
	OccurrenceCount int            `json:"occurrenceCount"`
	Metadata        map[string]any `json:"metadata,omitempty"`
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
	LastOccurredAt  time.Time      `json:"lastOccurredAt"`
	ReadAt          *time.Time     `json:"readAt,omitempty"`
	ArchivedAt      *time.Time     `json:"archivedAt,omitempty"`
	ExpiresAt       *time.Time     `json:"expiresAt,omitempty"`
}

type ListFilter struct {
	Statuses   []Status
	Kinds      []Kind
	Categories []Category
	Severities []Severity
	Surface    Surface
	Query      string
	Limit      int
}

type Store interface {
	Save(ctx context.Context, item Notice) error
	Get(ctx context.Context, id string) (Notice, error)
	List(ctx context.Context, filter ListFilter) ([]Notice, error)
	CountUnread(ctx context.Context, surface Surface) (int, error)
	FindByDedupKey(ctx context.Context, dedupKey string) (Notice, error)
	MarkRead(ctx context.Context, ids []string, read bool, at time.Time) error
	Archive(ctx context.Context, ids []string, archived bool, at time.Time) error
	MarkAllRead(ctx context.Context, surface Surface, at time.Time) error
}
