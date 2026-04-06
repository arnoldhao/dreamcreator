package observability

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"time"

	"dreamcreator/internal/application/gateway/channels"
	"dreamcreator/internal/application/gateway/nodes"
	sessionmanager "dreamcreator/internal/application/session"
	domainthread "dreamcreator/internal/domain/thread"
	"dreamcreator/internal/infrastructure/logging"
)

type HealthComponent struct {
	Name      string `json:"name"`
	Status    string `json:"status"`
	LatencyMs int    `json:"latencyMs,omitempty"`
	Detail    string `json:"detail,omitempty"`
}

type GatewayHealthSnapshot struct {
	Overall    string            `json:"overall"`
	Version    int64             `json:"version"`
	UpdatedAt  string            `json:"updatedAt"`
	Components []HealthComponent `json:"components"`
}

type GatewayStatusReport struct {
	AppVersion     string                   `json:"appVersion"`
	UptimeSec      int64                    `json:"uptimeSec"`
	ActiveSessions int                      `json:"activeSessions"`
	ActiveRuns     int                      `json:"activeRuns"`
	QueueDepth     int                      `json:"queueDepth"`
	ConnectedNodes int                      `json:"connectedNodes"`
	Channels       []channels.ChannelStatus `json:"channels,omitempty"`
}

type LogsTailRequest struct {
	Level     string `json:"level,omitempty"`
	Component string `json:"component,omitempty"`
	From      string `json:"from,omitempty"`
	Limit     int    `json:"limit,omitempty"`
}

type LogRecord struct {
	TS        string         `json:"ts"`
	Level     string         `json:"level"`
	Component string         `json:"component,omitempty"`
	Message   string         `json:"message"`
	Fields    map[string]any `json:"fields,omitempty"`
}

type LogsTailEvent struct {
	Records []LogRecord `json:"records"`
}

type DiagnosticIssue struct {
	ID        string `json:"id"`
	Severity  string `json:"severity"`
	Component string `json:"component"`
	Message   string `json:"message"`
	Hint      string `json:"hint,omitempty"`
}

type DiagnosticsReport struct {
	Issues      []DiagnosticIssue `json:"issues"`
	GeneratedAt string            `json:"generatedAt"`
}

type Service struct {
	sessions  *sessionmanager.Manager
	runs      domainthread.RunRepository
	channels  *channels.Registry
	nodes     *nodes.Service
	logger    *logging.Logger
	reports   ReportStore
	startedAt time.Time
	now       func() time.Time
}

type ReportStore interface {
	Save(ctx context.Context, report DiagnosticsReport) error
}

func NewService(
	sessions *sessionmanager.Manager,
	runRepo domainthread.RunRepository,
	channelRegistry *channels.Registry,
	nodeService *nodes.Service,
	logger *logging.Logger,
	reports ReportStore,
) *Service {
	return &Service{
		sessions:  sessions,
		runs:      runRepo,
		channels:  channelRegistry,
		nodes:     nodeService,
		logger:    logger,
		reports:   reports,
		startedAt: time.Now(),
		now:       time.Now,
	}
}

func (service *Service) Health(_ context.Context) GatewayHealthSnapshot {
	components := []HealthComponent{
		{Name: "gateway", Status: "ok"},
	}
	if service.sessions == nil {
		components = append(components, HealthComponent{Name: "sessions", Status: "degraded", Detail: "session manager unavailable"})
	} else {
		components = append(components, HealthComponent{Name: "sessions", Status: "ok"})
	}
	if service.channels == nil {
		components = append(components, HealthComponent{Name: "channels", Status: "degraded", Detail: "channel registry unavailable"})
	} else {
		components = append(components, HealthComponent{Name: "channels", Status: "ok"})
	}
	if service.nodes == nil {
		components = append(components, HealthComponent{Name: "nodes", Status: "degraded", Detail: "node service unavailable"})
	} else {
		components = append(components, HealthComponent{Name: "nodes", Status: "ok"})
	}
	overall := "healthy"
	for _, component := range components {
		if component.Status == "degraded" {
			overall = "degraded"
			break
		}
		if component.Status == "error" {
			overall = "unhealthy"
			break
		}
	}
	return GatewayHealthSnapshot{
		Overall:    overall,
		Version:    service.now().UnixNano(),
		UpdatedAt:  service.now().Format(time.RFC3339),
		Components: components,
	}
}

func (service *Service) Status(ctx context.Context) GatewayStatusReport {
	activeSessions := 0
	activeRuns := 0
	queueDepth := 0
	if service.sessions != nil {
		snapshot := service.sessions.Snapshot()
		activeSessions = snapshot.ActiveSessions
		for _, queue := range snapshot.Queues {
			queueDepth += queue.QueuedCount
			for _, lane := range queue.Lanes {
				queueDepth += lane.QueuedCount
			}
		}
	}
	if service.runs != nil {
		if count, err := service.runs.CountActive(ctx); err == nil {
			activeRuns = count
		}
	}
	connectedNodes := 0
	if service.nodes != nil {
		if list, err := service.nodes.ListNodes(ctx); err == nil {
			connectedNodes = len(list)
		}
	}
	channelStatuses := []channels.ChannelStatus{}
	if service.channels != nil {
		if statuses, err := service.channels.StatusAll(ctx); err == nil {
			channelStatuses = statuses
		}
	}
	uptime := int64(0)
	if !service.startedAt.IsZero() {
		uptime = int64(time.Since(service.startedAt).Seconds())
	}
	return GatewayStatusReport{
		AppVersion:     "dev",
		UptimeSec:      uptime,
		ActiveSessions: activeSessions,
		ActiveRuns:     activeRuns,
		QueueDepth:     queueDepth,
		ConnectedNodes: connectedNodes,
		Channels:       channelStatuses,
	}
}

func (service *Service) TailLogs(_ context.Context, request LogsTailRequest) (LogsTailEvent, error) {
	path := ""
	if service.logger != nil {
		path = service.logger.LogFilePath()
	}
	if path == "" {
		return LogsTailEvent{}, errors.New("log file unavailable")
	}
	limit := request.Limit
	if limit <= 0 {
		limit = 200
	}
	lines, err := tailLines(path, limit)
	if err != nil {
		return LogsTailEvent{}, err
	}
	levelFilter := strings.ToLower(strings.TrimSpace(request.Level))
	componentFilter := strings.ToLower(strings.TrimSpace(request.Component))
	records := make([]LogRecord, 0, len(lines))
	for _, line := range lines {
		record := parseLogLine(line)
		if levelFilter != "" && !levelAllowed(record.Level, levelFilter) {
			continue
		}
		if componentFilter != "" && !strings.Contains(strings.ToLower(record.Component), componentFilter) {
			continue
		}
		records = append(records, record)
	}
	return LogsTailEvent{Records: records}, nil
}

func (service *Service) Diagnostics(_ context.Context) DiagnosticsReport {
	issues := make([]DiagnosticIssue, 0)
	health := service.Health(context.Background())
	if health.Overall != "healthy" {
		issues = append(issues, DiagnosticIssue{
			ID:        "health",
			Severity:  "warn",
			Component: "gateway",
			Message:   "gateway health degraded",
			Hint:      "check component status and logs",
		})
	}
	if service.sessions == nil {
		issues = append(issues, DiagnosticIssue{
			ID:        "sessions",
			Severity:  "error",
			Component: "sessions",
			Message:   "session manager unavailable",
			Hint:      "restart service or check initialization order",
		})
	}
	if service.logger == nil || service.logger.LogFilePath() == "" {
		issues = append(issues, DiagnosticIssue{
			ID:        "logs",
			Severity:  "warn",
			Component: "logging",
			Message:   "log file path unavailable",
			Hint:      "verify logging configuration",
		})
	}
	report := DiagnosticsReport{
		Issues:      issues,
		GeneratedAt: service.now().Format(time.RFC3339),
	}
	if service.reports != nil {
		_ = service.reports.Save(context.Background(), report)
	}
	return report
}

func parseLogLine(line string) LogRecord {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}") {
		if record, ok := parseJSONLogLine(trimmed); ok {
			return record
		}
	}

	parts := strings.Split(line, "\t")
	record := LogRecord{Message: trimmed}
	if len(parts) >= 1 {
		record.TS = strings.TrimSpace(parts[0])
	}
	if len(parts) >= 2 {
		record.Level = strings.ToLower(strings.TrimSpace(parts[1]))
	}
	if len(parts) >= 3 {
		record.Component = strings.TrimSpace(parts[2])
	}
	if len(parts) >= 4 {
		message := strings.TrimSpace(parts[3])
		if message != "" {
			record.Message = message
		}
	}
	if len(parts) >= 5 {
		fieldsRaw := strings.TrimSpace(strings.Join(parts[4:], "\t"))
		if fieldsRaw != "" {
			var fields map[string]any
			if err := json.Unmarshal([]byte(fieldsRaw), &fields); err == nil && len(fields) > 0 {
				record.Fields = fields
			}
		}
	}
	return record
}

func parseJSONLogLine(line string) (LogRecord, bool) {
	var payload map[string]any
	if err := json.Unmarshal([]byte(line), &payload); err != nil {
		return LogRecord{}, false
	}

	record := LogRecord{
		TS:        toString(payload["ts"]),
		Level:     strings.ToLower(toString(payload["level"])),
		Component: toString(payload["caller"]),
		Message:   toString(payload["msg"]),
	}
	if record.Message == "" {
		record.Message = strings.TrimSpace(line)
	}

	fields := map[string]any{}
	for key, value := range payload {
		switch key {
		case "ts", "level", "caller", "msg", "stacktrace":
			continue
		default:
			fields[key] = value
		}
	}
	if len(fields) > 0 {
		record.Fields = fields
	}

	return record, true
}

func toString(value any) string {
	if text, ok := value.(string); ok {
		return strings.TrimSpace(text)
	}
	return ""
}

func levelAllowed(level string, minimum string) bool {
	order := map[string]int{
		"debug": 1,
		"info":  2,
		"warn":  3,
		"error": 4,
	}
	lv := order[strings.ToLower(level)]
	min := order[strings.ToLower(minimum)]
	if min == 0 {
		return true
	}
	if lv == 0 {
		return false
	}
	return lv >= min
}

func tailLines(path string, limit int) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if limit > 0 && len(lines) > limit {
		lines = lines[len(lines)-limit:]
	}
	return lines, nil
}
