package service

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

var ErrSubagentRunNotFound = errors.New("subagent run not found")

type RunStatus string

const (
	RunStatusPending RunStatus = "pending"
	RunStatusRunning RunStatus = "running"
	RunStatusSuccess RunStatus = "success"
	RunStatusFailed  RunStatus = "failed"
	RunStatusTimeout RunStatus = "timeout"
	RunStatusAborted RunStatus = "aborted"
)

type CleanupPolicy string

const (
	CleanupKeep   CleanupPolicy = "keep"
	CleanupDelete CleanupPolicy = "delete"
)

type RunUsage struct {
	PromptTokens     int `json:"promptTokens,omitempty"`
	CompletionTokens int `json:"completionTokens,omitempty"`
	TotalTokens      int `json:"totalTokens,omitempty"`
}

type SpawnRequest struct {
	ParentSessionKey  string
	ParentRunID       string
	AgentID           string
	Task              string
	Label             string
	Model             string
	Thinking          string
	RunTimeoutSeconds int
	CleanupPolicy     CleanupPolicy
	ChildSessionKey   string
	ChildSessionID    string
	CallerModel       string
	CallerThinking    string
	Payload           any
}

type RunRecord struct {
	RunID             string        `json:"runId"`
	ParentSessionKey  string        `json:"parentSessionKey,omitempty"`
	ParentRunID       string        `json:"parentRunId,omitempty"`
	AgentID           string        `json:"agentId,omitempty"`
	ChildSessionKey   string        `json:"childSessionKey,omitempty"`
	ChildSessionID    string        `json:"childSessionId,omitempty"`
	Task              string        `json:"task,omitempty"`
	Label             string        `json:"label,omitempty"`
	Model             string        `json:"model,omitempty"`
	Thinking          string        `json:"thinking,omitempty"`
	CallerModel       string        `json:"callerModel,omitempty"`
	CallerThinking    string        `json:"callerThinking,omitempty"`
	CleanupPolicy     CleanupPolicy `json:"cleanupPolicy,omitempty"`
	RunTimeoutSeconds int           `json:"runTimeoutSeconds,omitempty"`
	Status            RunStatus     `json:"status"`
	Result            string        `json:"result,omitempty"`
	Notes             string        `json:"notes,omitempty"`
	RuntimeMs         int64         `json:"runtimeMs,omitempty"`
	Usage             RunUsage      `json:"usage,omitempty"`
	TranscriptPath    string        `json:"transcriptPath,omitempty"`
	Summary           string        `json:"summary,omitempty"`
	Error             string        `json:"error,omitempty"`
	AnnounceKey       string        `json:"announceKey,omitempty"`
	AnnounceAttempts  int           `json:"announceAttempts,omitempty"`
	AnnounceSentAt    *time.Time    `json:"announceSentAt,omitempty"`
	FinishedAt        *time.Time    `json:"finishedAt,omitempty"`
	ArchivedAt        *time.Time    `json:"archivedAt,omitempty"`
	CreatedAt         time.Time     `json:"createdAt"`
	UpdatedAt         time.Time     `json:"updatedAt"`
}

type Spawner struct {
	mu      sync.RWMutex
	records map[string]RunRecord
	now     func() time.Time
	newID   func() string
}

func NewSpawner() *Spawner {
	return &Spawner{
		records: make(map[string]RunRecord),
		now:     time.Now,
		newID:   uuid.NewString,
	}
}

func (spawner *Spawner) Spawn(_ context.Context, request SpawnRequest) (RunRecord, error) {
	if spawner == nil {
		return RunRecord{}, errors.New("spawner unavailable")
	}
	runID := spawner.newID()
	now := spawner.now()
	cleanup := strings.ToLower(strings.TrimSpace(string(request.CleanupPolicy)))
	if cleanup == "" {
		cleanup = string(CleanupKeep)
	}
	if cleanup != string(CleanupKeep) && cleanup != string(CleanupDelete) {
		cleanup = string(CleanupKeep)
	}
	timeoutSeconds := request.RunTimeoutSeconds
	if timeoutSeconds < 0 {
		timeoutSeconds = 0
	}
	record := RunRecord{
		RunID:             runID,
		ParentSessionKey:  strings.TrimSpace(request.ParentSessionKey),
		ParentRunID:       strings.TrimSpace(request.ParentRunID),
		AgentID:           strings.TrimSpace(request.AgentID),
		ChildSessionKey:   strings.TrimSpace(request.ChildSessionKey),
		ChildSessionID:    strings.TrimSpace(request.ChildSessionID),
		Task:              strings.TrimSpace(request.Task),
		Label:             strings.TrimSpace(request.Label),
		Model:             strings.TrimSpace(request.Model),
		Thinking:          strings.TrimSpace(request.Thinking),
		CallerModel:       strings.TrimSpace(request.CallerModel),
		CallerThinking:    strings.TrimSpace(request.CallerThinking),
		CleanupPolicy:     CleanupPolicy(cleanup),
		RunTimeoutSeconds: timeoutSeconds,
		Status:            RunStatusRunning,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	spawner.mu.Lock()
	spawner.records[runID] = record
	spawner.mu.Unlock()
	return record, nil
}

func (spawner *Spawner) Get(_ context.Context, runID string) (RunRecord, error) {
	spawner.mu.RLock()
	defer spawner.mu.RUnlock()
	record, ok := spawner.records[strings.TrimSpace(runID)]
	if !ok {
		return RunRecord{}, ErrSubagentRunNotFound
	}
	return record, nil
}

func (spawner *Spawner) Update(record RunRecord) {
	if spawner == nil {
		return
	}
	spawner.mu.Lock()
	spawner.records[record.RunID] = record
	spawner.mu.Unlock()
}

func NormalizeRunStatus(value string) RunStatus {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case string(RunStatusRunning):
		return RunStatusRunning
	case string(RunStatusSuccess):
		return RunStatusSuccess
	case string(RunStatusFailed):
		return RunStatusFailed
	case string(RunStatusTimeout):
		return RunStatusTimeout
	case string(RunStatusAborted):
		return RunStatusAborted
	case string(RunStatusPending):
		return RunStatusPending
	default:
		return RunStatus(strings.TrimSpace(value))
	}
}

func ParseCleanupPolicy(value string) CleanupPolicy {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case string(CleanupDelete):
		return CleanupDelete
	default:
		return CleanupKeep
	}
}

func ParseTimeoutSeconds(value any) int {
	switch typed := value.(type) {
	case int:
		if typed > 0 {
			return typed
		}
	case int64:
		if typed > 0 {
			return int(typed)
		}
	case float64:
		if typed > 0 {
			return int(typed)
		}
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(typed))
		if err == nil && parsed > 0 {
			return parsed
		}
	}
	return 0
}
