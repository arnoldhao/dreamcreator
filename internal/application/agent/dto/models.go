package dto

type Agent struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
	ThreadID    string `json:"threadId"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

type CreateAgentRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type UpdateAgentRequest struct {
	ID          string  `json:"id"`
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Enabled     *bool   `json:"enabled"`
}

type DeleteAgentRequest struct {
	ID string `json:"id"`
}

type AgentRun struct {
	ID                 string `json:"id"`
	ThreadID           string `json:"threadId"`
	AssistantMessageID string `json:"assistantMessageId"`
	UserMessageID      string `json:"userMessageId"`
	AgentID            string `json:"agentId"`
	Status             string `json:"status"`
	ContentPartial     string `json:"contentPartial"`
	CreatedAt          string `json:"createdAt"`
	UpdatedAt          string `json:"updatedAt"`
}

type ListAgentRunsRequest struct {
	AgentID string `json:"agentId"`
	Limit   int    `json:"limit"`
}

type AgentRunEvent struct {
	ID          int64  `json:"id"`
	RunID       string `json:"runId"`
	EventType   string `json:"eventType"`
	PayloadJSON string `json:"payloadJson"`
	CreatedAt   string `json:"createdAt"`
}

type ListAgentRunEventsRequest struct {
	RunID   string `json:"runId"`
	AfterID int64  `json:"afterId"`
	Limit   int    `json:"limit"`
}

type AgentFileEntry struct {
	Name      string `json:"name"`
	Path      string `json:"path,omitempty"`
	Content   string `json:"content,omitempty"`
	Missing   bool   `json:"missing"`
	Size      int    `json:"size,omitempty"`
	UpdatedAt string `json:"updatedAt,omitempty"`
}

type AgentFilesListRequest struct {
	AgentID string `json:"agentId"`
}

type AgentFilesListResponse struct {
	AgentID string           `json:"agentId"`
	Files   []AgentFileEntry `json:"files"`
}

type AgentFilesGetRequest struct {
	AgentID string `json:"agentId"`
	Name    string `json:"name"`
}

type AgentFilesSetRequest struct {
	AgentID string `json:"agentId"`
	Name    string `json:"name"`
	Content string `json:"content"`
}
