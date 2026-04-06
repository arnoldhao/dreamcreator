package ws

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"golang.org/x/net/websocket"

	"dreamcreator/internal/application/gateway/auth"
	"dreamcreator/internal/application/gateway/controlplane"
)

type Server struct {
	router *controlplane.Router
	auth   auth.Service
	hub    *Hub

	serverInfo controlplane.ServerInfo
	policy     controlplane.TransportPolicy
	enabled    func() bool
}

func NewServer(router *controlplane.Router, authService auth.Service, hub *Hub) *Server {
	if hub == nil {
		hub = NewHub()
	}
	return &Server{
		router: router,
		auth:   authService,
		hub:    hub,
		serverInfo: controlplane.ServerInfo{
			Name:    "dreamcreator",
			Version: "dev",
		},
		policy: controlplane.TransportPolicy{
			MaxPayload:       1024 * 512,
			MaxBufferedBytes: 1024 * 1024,
			TickIntervalMs:   5_000,
		},
		enabled: func() bool { return true },
	}
}

func (server *Server) SetEnabledProvider(provider func() bool) {
	if server == nil || provider == nil {
		return
	}
	server.enabled = provider
}

func (server *Server) Handler() http.Handler {
	return websocket.Server{
		Handler: websocket.Handler(server.handleConnection),
		Handshake: func(config *websocket.Config, req *http.Request) error {
			// Allow all origins (desktop use-case).
			return nil
		},
	}
}

func (server *Server) Publish(event controlplane.EventFrame) error {
	return server.hub.Publish(event)
}

func (server *Server) handleConnection(conn *websocket.Conn) {
	defer conn.Close()
	if server.router == nil || server.auth == nil {
		_ = websocket.JSON.Send(conn, controlplane.GatewayError{Code: "unavailable", Message: "gateway not configured"})
		return
	}
	if server.enabled != nil && !server.enabled() {
		_ = websocket.JSON.Send(conn, controlplane.GatewayError{Code: "gateway_disabled", Message: "gateway control plane disabled"})
		return
	}

	ctx := context.Background()
	connect, err := readConnectRequest(conn)
	if err != nil {
		_ = websocket.JSON.Send(conn, controlplane.GatewayError{Code: "invalid_connect", Message: err.Error()})
		return
	}

	protocol, ok := negotiateProtocol(connect.MinProtocol, connect.MaxProtocol)
	if !ok {
		_ = websocket.JSON.Send(conn, controlplane.GatewayError{Code: "protocol_mismatch", Message: "unsupported protocol"})
		return
	}

	authCtx, err := server.auth.Authenticate(ctx, auth.Credentials{
		Token:    connect.Auth.Token,
		Password: connect.Auth.Password,
	}, connect.Role, connect.Scopes)
	if err != nil {
		_ = websocket.JSON.Send(conn, controlplane.GatewayError{Code: "unauthorized", Message: "authentication failed"})
		return
	}

	session := &controlplane.SessionContext{
		ID:          uuid.NewString(),
		Role:        strings.TrimSpace(connect.Role),
		Scopes:      append([]string(nil), connect.Scopes...),
		Auth:        authCtx,
		ConnectedAt: time.Now(),
	}

	hello := controlplane.HelloOK{
		Type:     "hello-ok",
		Protocol: protocol,
		Server:   server.serverInfo,
		Features: controlplane.GatewayFeatures{
			Methods: server.router.Methods(),
			Events:  []string{"gateway.connected"},
		},
		Auth:   authCtx,
		Policy: server.policy,
	}
	if err := websocket.JSON.Send(conn, hello); err != nil {
		return
	}

	client := server.hub.addClient(conn)
	defer server.hub.removeClient(client)

	for {
		var raw []byte
		if err := websocket.Message.Receive(conn, &raw); err != nil {
			return
		}
		request, ok := parseRequestFrame(raw)
		if !ok {
			_ = websocket.JSON.Send(conn, controlplane.GatewayError{Code: "invalid_request", Message: "invalid frame"})
			continue
		}
		response := server.router.Handle(ctx, session, request)
		_ = websocket.JSON.Send(conn, response)
	}
}

func readConnectRequest(conn *websocket.Conn) (controlplane.ConnectRequest, error) {
	var raw []byte
	if err := websocket.Message.Receive(conn, &raw); err != nil {
		return controlplane.ConnectRequest{}, err
	}
	var envelope struct {
		Type   string `json:"type"`
		Method string `json:"method"`
	}
	_ = json.Unmarshal(raw, &envelope)
	if strings.EqualFold(envelope.Type, "req") || strings.TrimSpace(envelope.Method) != "" {
		var req controlplane.RequestFrame
		if err := json.Unmarshal(raw, &req); err != nil {
			return controlplane.ConnectRequest{}, err
		}
		if strings.TrimSpace(req.Method) != "connect" {
			return controlplane.ConnectRequest{}, errors.New("first frame must be connect")
		}
		var connect controlplane.ConnectRequest
		if len(req.Params) == 0 {
			return controlplane.ConnectRequest{}, errors.New("missing connect params")
		}
		if err := json.Unmarshal(req.Params, &connect); err != nil {
			return controlplane.ConnectRequest{}, err
		}
		return connect, nil
	}
	var connect controlplane.ConnectRequest
	if err := json.Unmarshal(raw, &connect); err != nil {
		return controlplane.ConnectRequest{}, err
	}
	return connect, nil
}

func negotiateProtocol(min, max int) (int, bool) {
	if min == 0 && max == 0 {
		return controlplane.DefaultProtocolVersion, true
	}
	if min == 0 {
		min = 1
	}
	if max == 0 {
		max = min
	}
	if min > max {
		return 0, false
	}
	if controlplane.DefaultProtocolVersion < min || controlplane.DefaultProtocolVersion > max {
		return 0, false
	}
	return controlplane.DefaultProtocolVersion, true
}

func parseRequestFrame(raw []byte) (controlplane.RequestFrame, bool) {
	var frame controlplane.RequestFrame
	if err := json.Unmarshal(raw, &frame); err != nil {
		return controlplane.RequestFrame{}, false
	}
	if frame.Type == "" {
		frame.Type = "req"
	}
	if strings.TrimSpace(frame.Method) == "" || strings.TrimSpace(frame.ID) == "" {
		return controlplane.RequestFrame{}, false
	}
	if frame.Type != "req" {
		return controlplane.RequestFrame{}, false
	}
	return frame, true
}

type Hub struct {
	mu      sync.RWMutex
	clients map[*client]struct{}
}

type client struct {
	conn *websocket.Conn
	send chan controlplane.EventFrame
	done chan struct{}
}

func NewHub() *Hub {
	return &Hub{clients: make(map[*client]struct{})}
}

func (hub *Hub) addClient(conn *websocket.Conn) *client {
	c := &client{
		conn: conn,
		send: make(chan controlplane.EventFrame, 32),
		done: make(chan struct{}),
	}
	hub.mu.Lock()
	hub.clients[c] = struct{}{}
	hub.mu.Unlock()
	go c.writeLoop()
	return c
}

func (hub *Hub) removeClient(c *client) {
	if c == nil {
		return
	}
	hub.mu.Lock()
	delete(hub.clients, c)
	hub.mu.Unlock()
	close(c.send)
	close(c.done)
}

func (hub *Hub) Publish(event controlplane.EventFrame) error {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	hub.mu.RLock()
	defer hub.mu.RUnlock()
	for c := range hub.clients {
		select {
		case c.send <- event:
		default:
			// Drop if backpressure.
		}
	}
	return nil
}

func (c *client) writeLoop() {
	for event := range c.send {
		_ = websocket.JSON.Send(c.conn, event)
	}
}
