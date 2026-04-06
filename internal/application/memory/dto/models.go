package dto

type MemoryCategory string

const (
	MemoryCategoryPreference MemoryCategory = "preference"
	MemoryCategoryFact       MemoryCategory = "fact"
	MemoryCategoryDecision   MemoryCategory = "decision"
	MemoryCategoryEntity     MemoryCategory = "entity"
	MemoryCategoryReflection MemoryCategory = "reflection"
	MemoryCategoryOther      MemoryCategory = "other"
)

type STMState struct {
	ID         string `json:"id"`
	ThreadID   string `json:"threadId"`
	WindowJSON string `json:"windowJson"`
	Summary    string `json:"summary"`
}

type LTMEntry struct {
	ID          string  `json:"id"`
	AssistantID string  `json:"assistantId,omitempty"`
	ThreadID    string  `json:"threadId,omitempty"`
	Content     string  `json:"content"`
	Category    string  `json:"category"`
	Confidence  float32 `json:"confidence"`
	Score       float64 `json:"score,omitempty"`
	SourceJSON  string  `json:"sourceJson,omitempty"`
	CreatedAt   string  `json:"createdAt,omitempty"`
	UpdatedAt   string  `json:"updatedAt,omitempty"`
}

type MemoryRetrieval struct {
	Entries []LTMEntry `json:"entries"`
}

type UpdateSTMRequest struct {
	ThreadID   string `json:"threadId"`
	WindowJSON string `json:"windowJson"`
	Summary    string `json:"summary"`
}

type RetrieveForContextRequest struct {
	ThreadID string `json:"threadId"`
	TopK     int    `json:"topK"`
}

type ProposeWritesRequest struct {
	ThreadID   string     `json:"threadId"`
	Candidates []LTMEntry `json:"candidates"`
}

type CommitWritesRequest struct {
	ThreadID string   `json:"threadId"`
	EntryIDs []string `json:"entryIds"`
}

type ImportDocsRequest struct {
	WorkspaceID string `json:"workspaceId"`
	Content     string `json:"content"`
	Name        string `json:"name"`
}

type BuildIndexRequest struct {
	WorkspaceID string `json:"workspaceId"`
}

type RetrieveRAGRequest struct {
	WorkspaceID string `json:"workspaceId"`
	Query       string `json:"query"`
	TopK        int    `json:"topK"`
}

type EmbedRequest struct {
	AssistantID string `json:"assistantId,omitempty"`
	ProviderID  string `json:"providerId"`
	ModelName   string `json:"modelName"`
	Input       string `json:"input"`
}

type MemoryIdentity struct {
	AssistantID string `json:"assistantId,omitempty"`
	ThreadID    string `json:"threadId,omitempty"`
	Scope       string `json:"scope,omitempty"`
	Channel     string `json:"channel,omitempty"`
	AccountID   string `json:"accountId,omitempty"`
	UserID      string `json:"userId,omitempty"`
	GroupID     string `json:"groupId,omitempty"`
}

type MemoryRecallRequest struct {
	Identity    MemoryIdentity `json:"identity,omitempty"`
	AssistantID string         `json:"assistantId,omitempty"`
	ThreadID    string         `json:"threadId,omitempty"`
	Channel     string         `json:"channel,omitempty"`
	AccountID   string         `json:"accountId,omitempty"`
	UserID      string         `json:"userId,omitempty"`
	GroupID     string         `json:"groupId,omitempty"`
	Query       string         `json:"query"`
	TopK        int            `json:"topK,omitempty"`
	Category    string         `json:"category,omitempty"`
	Scope       string         `json:"scope,omitempty"`
}

type MemoryStoreRequest struct {
	Identity    MemoryIdentity `json:"identity,omitempty"`
	AssistantID string         `json:"assistantId,omitempty"`
	ThreadID    string         `json:"threadId,omitempty"`
	Channel     string         `json:"channel,omitempty"`
	AccountID   string         `json:"accountId,omitempty"`
	UserID      string         `json:"userId,omitempty"`
	GroupID     string         `json:"groupId,omitempty"`
	Content     string         `json:"content"`
	Category    string         `json:"category,omitempty"`
	Scope       string         `json:"scope,omitempty"`
	Confidence  float32        `json:"confidence,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

type MemoryForgetRequest struct {
	Identity    MemoryIdentity `json:"identity,omitempty"`
	AssistantID string         `json:"assistantId,omitempty"`
	ThreadID    string         `json:"threadId,omitempty"`
	Channel     string         `json:"channel,omitempty"`
	AccountID   string         `json:"accountId,omitempty"`
	UserID      string         `json:"userId,omitempty"`
	GroupID     string         `json:"groupId,omitempty"`
	MemoryID    string         `json:"memoryId,omitempty"`
	Scope       string         `json:"scope,omitempty"`
}

type MemoryUpdateRequest struct {
	Identity    MemoryIdentity `json:"identity,omitempty"`
	AssistantID string         `json:"assistantId,omitempty"`
	ThreadID    string         `json:"threadId,omitempty"`
	Channel     string         `json:"channel,omitempty"`
	AccountID   string         `json:"accountId,omitempty"`
	UserID      string         `json:"userId,omitempty"`
	GroupID     string         `json:"groupId,omitempty"`
	MemoryID    string         `json:"memoryId"`
	Content     *string        `json:"content,omitempty"`
	Category    *string        `json:"category,omitempty"`
	Scope       *string        `json:"scope,omitempty"`
	Confidence  *float32       `json:"confidence,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

type MemoryListRequest struct {
	Identity    MemoryIdentity `json:"identity,omitempty"`
	AssistantID string         `json:"assistantId,omitempty"`
	ThreadID    string         `json:"threadId,omitempty"`
	Channel     string         `json:"channel,omitempty"`
	AccountID   string         `json:"accountId,omitempty"`
	UserID      string         `json:"userId,omitempty"`
	GroupID     string         `json:"groupId,omitempty"`
	Category    string         `json:"category,omitempty"`
	Scope       string         `json:"scope,omitempty"`
	Limit       int            `json:"limit,omitempty"`
	Offset      int            `json:"offset,omitempty"`
}

type MemoryStatsRequest struct {
	Identity    MemoryIdentity `json:"identity,omitempty"`
	AssistantID string         `json:"assistantId,omitempty"`
	ThreadID    string         `json:"threadId,omitempty"`
	Channel     string         `json:"channel,omitempty"`
	AccountID   string         `json:"accountId,omitempty"`
	UserID      string         `json:"userId,omitempty"`
	GroupID     string         `json:"groupId,omitempty"`
	Scope       string         `json:"scope,omitempty"`
}

type MemoryStats struct {
	TotalCount      int            `json:"totalCount"`
	AssistantCount  int            `json:"assistantCount,omitempty"`
	CategoryCounts  map[string]int `json:"categoryCounts,omitempty"`
	LastUpdatedAt   string         `json:"lastUpdatedAt,omitempty"`
	LastMemoryAt    string         `json:"lastMemoryAt,omitempty"`
	HasEmbeddings   bool           `json:"hasEmbeddings,omitempty"`
	HasFTS          bool           `json:"hasFts,omitempty"`
	ConfiguredModel string         `json:"configuredModel,omitempty"`
}

type MemoryMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type BeforeAgentStartRequest struct {
	Identity    MemoryIdentity `json:"identity,omitempty"`
	AssistantID string         `json:"assistantId,omitempty"`
	ThreadID    string         `json:"threadId,omitempty"`
	Channel     string         `json:"channel,omitempty"`
	AccountID   string         `json:"accountId,omitempty"`
	UserID      string         `json:"userId,omitempty"`
	GroupID     string         `json:"groupId,omitempty"`
	Query       string         `json:"query"`
	TopK        int            `json:"topK,omitempty"`
}

type BeforeAgentStartResult struct {
	InjectedContext string     `json:"injectedContext,omitempty"`
	Entries         []LTMEntry `json:"entries,omitempty"`
}

type AgentEndRequest struct {
	Identity    MemoryIdentity  `json:"identity,omitempty"`
	AssistantID string          `json:"assistantId,omitempty"`
	ThreadID    string          `json:"threadId,omitempty"`
	Channel     string          `json:"channel,omitempty"`
	AccountID   string          `json:"accountId,omitempty"`
	UserID      string          `json:"userId,omitempty"`
	GroupID     string          `json:"groupId,omitempty"`
	RunID       string          `json:"runId,omitempty"`
	Messages    []MemoryMessage `json:"messages,omitempty"`
}

type SessionLifecycleEvent string

const (
	SessionLifecycleArchived SessionLifecycleEvent = "archived"
	SessionLifecycleDeleted  SessionLifecycleEvent = "deleted"
)

type SessionLifecycleRequest struct {
	Identity    MemoryIdentity        `json:"identity,omitempty"`
	AssistantID string                `json:"assistantId,omitempty"`
	ThreadID    string                `json:"threadId,omitempty"`
	Channel     string                `json:"channel,omitempty"`
	AccountID   string                `json:"accountId,omitempty"`
	UserID      string                `json:"userId,omitempty"`
	GroupID     string                `json:"groupId,omitempty"`
	Event       SessionLifecycleEvent `json:"event"`
}

type MemorySummaryRequest struct {
	AssistantID string `json:"assistantId,omitempty"`
}

type MemoryBrowseOptionsRequest struct {
	Identity    MemoryIdentity `json:"identity,omitempty"`
	AssistantID string         `json:"assistantId,omitempty"`
	ThreadID    string         `json:"threadId,omitempty"`
	Scope       string         `json:"scope,omitempty"`
	Channel     string         `json:"channel,omitempty"`
}

type MemoryBrowseOptions struct {
	Scopes     []string `json:"scopes,omitempty"`
	Channels   []string `json:"channels,omitempty"`
	AccountIDs []string `json:"accountIds,omitempty"`
	Categories []string `json:"categories,omitempty"`
}

type MemoryPrincipalListRequest struct {
	Identity      MemoryIdentity `json:"identity,omitempty"`
	AssistantID   string         `json:"assistantId,omitempty"`
	ThreadID      string         `json:"threadId,omitempty"`
	Scope         string         `json:"scope,omitempty"`
	Channel       string         `json:"channel,omitempty"`
	AccountID     string         `json:"accountId,omitempty"`
	Category      string         `json:"category,omitempty"`
	PrincipalType string         `json:"principalType,omitempty"`
	Query         string         `json:"query,omitempty"`
	Limit         int            `json:"limit,omitempty"`
}

type MemoryPrincipalItem struct {
	PrincipalID   string `json:"principalId"`
	Channel       string `json:"channel,omitempty"`
	Name          string `json:"name,omitempty"`
	Username      string `json:"username,omitempty"`
	AvatarURL     string `json:"avatarUrl,omitempty"`
	AvatarKey     string `json:"avatarKey,omitempty"`
	Count         int    `json:"count"`
	LastUpdatedAt string `json:"lastUpdatedAt,omitempty"`
}

type MemoryPrincipalRefreshRequest struct {
	Identity      MemoryIdentity `json:"identity,omitempty"`
	AssistantID   string         `json:"assistantId,omitempty"`
	ThreadID      string         `json:"threadId,omitempty"`
	Scope         string         `json:"scope,omitempty"`
	Channel       string         `json:"channel,omitempty"`
	AccountID     string         `json:"accountId,omitempty"`
	PrincipalType string         `json:"principalType,omitempty"`
	PrincipalID   string         `json:"principalId,omitempty"`
}

type MemoryPrincipalRefreshResult struct {
	PrincipalID   string `json:"principalId"`
	Name          string `json:"name,omitempty"`
	Username      string `json:"username,omitempty"`
	AvatarURL     string `json:"avatarUrl,omitempty"`
	AvatarKey     string `json:"avatarKey,omitempty"`
	UpdatedRows   int    `json:"updatedRows"`
	LastUpdatedAt string `json:"lastUpdatedAt,omitempty"`
}

type MemorySummary struct {
	AssistantID     string               `json:"assistantId,omitempty"`
	Summary         string               `json:"summary,omitempty"`
	TotalMemories   int                  `json:"totalMemories"`
	AssistantCount  int                  `json:"assistantCount"`
	ThreadCount     int                  `json:"threadCount"`
	UserCount       int                  `json:"userCount"`
	GroupCount      int                  `json:"groupCount"`
	ChannelCount    int                  `json:"channelCount"`
	AccountCount    int                  `json:"accountCount"`
	CategoryCounts  map[string]int       `json:"categoryCounts,omitempty"`
	ScopeCounts     map[string]int       `json:"scopeCounts,omitempty"`
	ChannelCounts   map[string]int       `json:"channelCounts,omitempty"`
	AccountCounts   map[string]int       `json:"accountCounts,omitempty"`
	PrincipalCounts map[string]int       `json:"principalCounts,omitempty"`
	Storage         MemorySummaryStorage `json:"storage"`
	LastUpdatedAt   string               `json:"lastUpdatedAt,omitempty"`
}

type MemorySummaryStorage struct {
	TotalBytes            int64 `json:"totalBytes"`
	CollectionsBytes      int64 `json:"collectionsBytes"`
	ChunksBytes           int64 `json:"chunksBytes"`
	AssistantSummaryBytes int64 `json:"assistantSummaryBytes"`
	AvatarCacheBytes      int64 `json:"avatarCacheBytes"`
}
