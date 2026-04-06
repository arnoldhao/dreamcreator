package controlplane

import (
	"encoding/json"
	"time"

	"dreamcreator/internal/application/gateway/auth"
)

const (
	DefaultProtocolVersion = 1
)

// ConnectRequest is the initial handshake payload from a gateway client.
type ConnectRequest struct {
	Type        string       `json:"type,omitempty"`
	MinProtocol int          `json:"minProtocol,omitempty"`
	MaxProtocol int          `json:"maxProtocol,omitempty"`
	Client      ClientInfo   `json:"client,omitempty"`
	Role        string       `json:"role,omitempty"`
	Scopes      []string     `json:"scopes,omitempty"`
	Auth        ConnectAuth  `json:"auth,omitempty"`
	Device      *DeviceProof `json:"device,omitempty"`
}

type ClientInfo struct {
	ID          string `json:"id,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
	Version     string `json:"version,omitempty"`
	Platform    string `json:"platform,omitempty"`
	Mode        string `json:"mode,omitempty"`
}

type ConnectAuth struct {
	Token    string `json:"token,omitempty"`
	Password string `json:"password,omitempty"`
}

type DeviceProof struct {
	ID        string `json:"id,omitempty"`
	PublicKey string `json:"publicKey,omitempty"`
	Signature string `json:"signature,omitempty"`
	SignedAt  int64  `json:"signedAt,omitempty"`
}

type HelloOK struct {
	Type     string           `json:"type,omitempty"`
	Protocol int              `json:"protocol"`
	Server   ServerInfo       `json:"server"`
	Features GatewayFeatures  `json:"features"`
	Snapshot json.RawMessage  `json:"snapshot,omitempty"`
	Auth     auth.AuthContext `json:"auth"`
	Policy   TransportPolicy  `json:"policy"`
}

type ServerInfo struct {
	Name    string `json:"name,omitempty"`
	Version string `json:"version,omitempty"`
}

type GatewayFeatures struct {
	Methods []string `json:"methods"`
	Events  []string `json:"events"`
}

type TransportPolicy struct {
	MaxPayload       int   `json:"maxPayload"`
	MaxBufferedBytes int   `json:"maxBufferedBytes"`
	TickIntervalMs   int64 `json:"tickIntervalMs"`
}

type RequestFrame struct {
	Type   string          `json:"type"`
	ID     string          `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
}

type ResponseFrame struct {
	Type    string        `json:"type"`
	ID      string        `json:"id"`
	OK      bool          `json:"ok"`
	Payload any           `json:"payload,omitempty"`
	Error   *GatewayError `json:"error,omitempty"`
}

type EventFrame struct {
	Type         string    `json:"type"`
	Event        string    `json:"event"`
	Payload      any       `json:"payload,omitempty"`
	Seq          int64     `json:"seq,omitempty"`
	StateVersion int64     `json:"stateVersion,omitempty"`
	Timestamp    time.Time `json:"timestamp"`
	SessionID    string    `json:"sessionId,omitempty"`
	SessionKey   string    `json:"sessionKey,omitempty"`
	RunID        string    `json:"runId,omitempty"`
}

type GatewayError struct {
	Code         string `json:"code"`
	Message      string `json:"message"`
	Details      any    `json:"details,omitempty"`
	Retryable    bool   `json:"retryable"`
	RetryAfterMs int64  `json:"retryAfterMs,omitempty"`
}

// EventPublisher is used by runtime modules to push gateway events.
type EventPublisher interface {
	Publish(event EventFrame) error
}
