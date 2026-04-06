package session

import "time"

type Status string

const (
	StatusActive   Status = "active"
	StatusArchived Status = "archived"
)

type Origin struct {
	Channel             string `json:"channel,omitempty"`
	AccountID           string `json:"accountId,omitempty"`
	ChatType            string `json:"chatType,omitempty"`
	PeerID              string `json:"peerId,omitempty"`
	PeerName            string `json:"peerName,omitempty"`
	PeerUsername        string `json:"peerUsername,omitempty"`
	PeerAvatarURL       string `json:"peerAvatarUrl,omitempty"`
	PeerAvatarKey       string `json:"peerAvatarKey,omitempty"`
	PeerAvatarSourceURL string `json:"peerAvatarSourceUrl,omitempty"`
	ThreadRef           string `json:"threadRef,omitempty"`
}

type Entry struct {
	SessionID                  string    `json:"sessionId"`
	SessionKey                 string    `json:"sessionKey"`
	AgentID                    string    `json:"agentId,omitempty"`
	AssistantID                string    `json:"assistantId,omitempty"`
	Title                      string    `json:"title,omitempty"`
	Status                     Status    `json:"status"`
	Origin                     Origin    `json:"origin,omitempty"`
	ContextPromptTokens        int       `json:"contextPromptTokens,omitempty"`
	ContextTotalTokens         int       `json:"contextTotalTokens,omitempty"`
	ContextWindowTokens        int       `json:"contextWindowTokens,omitempty"`
	ContextUpdatedAt           time.Time `json:"contextUpdatedAt,omitempty"`
	ContextFresh               bool      `json:"contextFresh,omitempty"`
	CompactionCount            int       `json:"compactionCount,omitempty"`
	MemoryFlushCompactionCount int       `json:"memoryFlushCompactionCount,omitempty"`
	CreatedAt                  time.Time `json:"createdAt"`
	UpdatedAt                  time.Time `json:"updatedAt"`
}

type ThreadProjection struct {
	ID          string `json:"id"`
	AssistantID string `json:"assistantId,omitempty"`
	Title       string `json:"title,omitempty"`
	Status      Status `json:"status"`
}

func (entry Entry) ToThreadProjection() ThreadProjection {
	return ThreadProjection{
		ID:          entry.SessionID,
		AssistantID: entry.AssistantID,
		Title:       entry.Title,
		Status:      entry.Status,
	}
}
