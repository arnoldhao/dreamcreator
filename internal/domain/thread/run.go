package thread

import (
    "errors"
    "strings"
    "time"
)

var ErrRunNotFound = errors.New("run not found")

type RunStatus string

const (
    RunStatusActive    RunStatus = "active"
    RunStatusFinished  RunStatus = "finished"
    RunStatusCancelled RunStatus = "cancelled"
    RunStatusError     RunStatus = "error"
)

type ThreadRun struct {
    ID                 string
    ThreadID           string
    AssistantMessageID string
    UserMessageID      string
    AgentID            string
    Status             RunStatus
    ContentPartial     string
    CreatedAt          time.Time
    UpdatedAt          time.Time
}

type ThreadRunParams struct {
    ID                 string
    ThreadID           string
    AssistantMessageID string
    UserMessageID      string
    AgentID            string
    Status             RunStatus
    ContentPartial     string
    CreatedAt          *time.Time
    UpdatedAt          *time.Time
}

func NewThreadRun(params ThreadRunParams) (ThreadRun, error) {
    id := strings.TrimSpace(params.ID)
    threadID := strings.TrimSpace(params.ThreadID)
    assistantMessageID := strings.TrimSpace(params.AssistantMessageID)
    if id == "" || threadID == "" || assistantMessageID == "" {
        return ThreadRun{}, ErrInvalidThreadRun
    }
    status := params.Status
    if status == "" {
        status = RunStatusActive
    }
    createdAt := time.Now()
    if params.CreatedAt != nil {
        createdAt = *params.CreatedAt
    }
    updatedAt := createdAt
    if params.UpdatedAt != nil {
        updatedAt = *params.UpdatedAt
    }

    return ThreadRun{
        ID:                 id,
        ThreadID:           threadID,
        AssistantMessageID: assistantMessageID,
        UserMessageID:      strings.TrimSpace(params.UserMessageID),
        AgentID:            strings.TrimSpace(params.AgentID),
        Status:             status,
        ContentPartial:     params.ContentPartial,
        CreatedAt:          createdAt,
        UpdatedAt:          updatedAt,
    }, nil
}

type ThreadRunEvent struct {
    ID          int64
    RunID       string
    ThreadID    string
    EventType   string
    PayloadJSON string
    CreatedAt   time.Time
}

type ThreadRunEventParams struct {
    ID          int64
    RunID       string
    ThreadID    string
    EventType   string
    PayloadJSON string
    CreatedAt   *time.Time
}

func NewThreadRunEvent(params ThreadRunEventParams) (ThreadRunEvent, error) {
    runID := strings.TrimSpace(params.RunID)
    threadID := strings.TrimSpace(params.ThreadID)
    payload := strings.TrimSpace(params.PayloadJSON)
    if runID == "" || threadID == "" || payload == "" {
        return ThreadRunEvent{}, ErrInvalidThreadRunEvent
    }
    createdAt := time.Now()
    if params.CreatedAt != nil {
        createdAt = *params.CreatedAt
    }

    return ThreadRunEvent{
        ID:          params.ID,
        RunID:       runID,
        ThreadID:    threadID,
        EventType:   strings.TrimSpace(params.EventType),
        PayloadJSON: payload,
        CreatedAt:   createdAt,
    }, nil
}
