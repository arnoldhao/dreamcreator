package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/net/websocket"

	gatewaycontrolplane "dreamcreator/internal/application/gateway/controlplane"
	gatewaynodes "dreamcreator/internal/application/gateway/nodes"
)

type canvasNodeInvoker interface {
	ListNodes(ctx context.Context) ([]gatewaynodes.NodeDescriptor, error)
	Invoke(ctx context.Context, request gatewaynodes.NodeInvokeRequest) (gatewaynodes.NodeInvokeResult, error)
}

type canvasNodeInvokerBridge struct {
	Invoker canvasNodeInvoker
	Close   func() error
}

type canvasGatewayClient struct {
	conn      *websocket.Conn
	timeoutMs int
}

type canvasGatewayResponseFrame struct {
	Type    string                            `json:"type"`
	ID      string                            `json:"id"`
	OK      bool                              `json:"ok"`
	Payload json.RawMessage                   `json:"payload,omitempty"`
	Error   *gatewaycontrolplane.GatewayError `json:"error,omitempty"`
}

func resolveCanvasNodeInvoker(ctx context.Context, payload toolArgs, local *gatewaynodes.Service) (canvasNodeInvokerBridge, error) {
	gatewayURL := strings.TrimSpace(getStringArg(payload, "gatewayUrl"))
	timeoutMs := resolveCanvasTimeoutMs(payload, defaultCanvasInvokeTimeoutMs)
	if gatewayURL == "" {
		if local == nil {
			return canvasNodeInvokerBridge{}, errors.New("nodes service unavailable")
		}
		return canvasNodeInvokerBridge{
			Invoker: local,
			Close: func() error {
				return nil
			},
		}, nil
	}
	client, err := newCanvasGatewayClient(ctx, gatewayURL, strings.TrimSpace(getStringArg(payload, "gatewayToken")), timeoutMs)
	if err != nil {
		return canvasNodeInvokerBridge{}, err
	}
	return canvasNodeInvokerBridge{
		Invoker: client,
		Close:   client.Close,
	}, nil
}

func newCanvasGatewayClient(ctx context.Context, rawURL string, gatewayToken string, timeoutMs int) (*canvasGatewayClient, error) {
	wsURL, origin, err := normalizeCanvasGatewayURL(rawURL)
	if err != nil {
		return nil, err
	}
	config, err := websocket.NewConfig(wsURL, origin)
	if err != nil {
		return nil, err
	}
	conn, err := websocket.DialConfig(config)
	if err != nil {
		return nil, err
	}
	client := &canvasGatewayClient{conn: conn, timeoutMs: timeoutMs}
	if err := client.handshake(ctx, gatewayToken); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return client, nil
}

func normalizeCanvasGatewayURL(rawURL string) (string, string, error) {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return "", "", errors.New("gatewayUrl is required")
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", "", errors.New("invalid gatewayUrl")
	}
	switch strings.ToLower(strings.TrimSpace(parsed.Scheme)) {
	case "ws", "wss":
	default:
		return "", "", errors.New("invalid gatewayUrl protocol")
	}
	if strings.TrimSpace(parsed.Host) == "" {
		return "", "", errors.New("invalid gatewayUrl host")
	}
	if !isCanvasGatewayLoopbackHost(parsed.Hostname()) {
		return "", "", errors.New("gatewayUrl override rejected: only loopback hosts are allowed")
	}
	if parsed.User != nil {
		return "", "", errors.New("invalid gatewayUrl: credentials are not allowed")
	}
	if strings.TrimSpace(parsed.RawQuery) != "" || strings.TrimSpace(parsed.Fragment) != "" {
		return "", "", errors.New("invalid gatewayUrl: query/hash not allowed")
	}
	pathValue := strings.TrimSpace(parsed.Path)
	switch pathValue {
	case "", "/":
		parsed.Path = "/gateway/ws"
	default:
		return "", "", errors.New("invalid gatewayUrl: path not allowed")
	}
	originScheme := "http"
	if strings.EqualFold(parsed.Scheme, "wss") {
		originScheme = "https"
	}
	origin := originScheme + "://" + parsed.Host
	return parsed.String(), origin, nil
}

func isCanvasGatewayLoopbackHost(host string) bool {
	value := strings.ToLower(strings.TrimSpace(host))
	if value == "localhost" {
		return true
	}
	parsed := net.ParseIP(value)
	return parsed != nil && parsed.IsLoopback()
}

func (client *canvasGatewayClient) Close() error {
	if client == nil || client.conn == nil {
		return nil
	}
	return client.conn.Close()
}

func (client *canvasGatewayClient) handshake(ctx context.Context, gatewayToken string) error {
	if client == nil || client.conn == nil {
		return errors.New("gateway connection unavailable")
	}
	connectRequest := gatewaycontrolplane.ConnectRequest{
		Type:        "connect",
		MinProtocol: gatewaycontrolplane.DefaultProtocolVersion,
		MaxProtocol: gatewaycontrolplane.DefaultProtocolVersion,
		Client: gatewaycontrolplane.ClientInfo{
			ID:          "canvas-tool",
			DisplayName: "canvas tool",
			Mode:        "backend",
		},
		Role:   "operator",
		Scopes: []string{"node.list", "node.invoke"},
		Auth: gatewaycontrolplane.ConnectAuth{
			Token: strings.TrimSpace(gatewayToken),
		},
	}
	if err := client.writeJSONWithContext(ctx, connectRequest); err != nil {
		return err
	}
	raw, err := client.readRawWithContext(ctx)
	if err != nil {
		return err
	}
	var gatewayErr gatewaycontrolplane.GatewayError
	if err := json.Unmarshal(raw, &gatewayErr); err == nil && strings.TrimSpace(gatewayErr.Code) != "" {
		return errors.New(strings.TrimSpace(gatewayErr.Message))
	}
	var hello gatewaycontrolplane.HelloOK
	if err := json.Unmarshal(raw, &hello); err != nil {
		return errors.New("invalid gateway hello response")
	}
	if !strings.EqualFold(strings.TrimSpace(hello.Type), "hello-ok") {
		return errors.New("gateway handshake failed")
	}
	return nil
}

func (client *canvasGatewayClient) ListNodes(ctx context.Context) ([]gatewaynodes.NodeDescriptor, error) {
	raw, err := client.call(ctx, "node.list", map[string]any{})
	if err != nil {
		return nil, err
	}
	var list []gatewaynodes.NodeDescriptor
	if len(raw) == 0 || strings.EqualFold(strings.TrimSpace(string(raw)), "null") {
		return nil, nil
	}
	if err := json.Unmarshal(raw, &list); err != nil {
		return nil, err
	}
	return list, nil
}

func (client *canvasGatewayClient) Invoke(ctx context.Context, request gatewaynodes.NodeInvokeRequest) (gatewaynodes.NodeInvokeResult, error) {
	payload := map[string]any{
		"invokeId":   strings.TrimSpace(request.InvokeID),
		"nodeId":     strings.TrimSpace(request.NodeID),
		"capability": strings.TrimSpace(request.Capability),
		"action":     strings.TrimSpace(request.Action),
		"args":       request.Args,
		"timeoutMs":  request.TimeoutMs,
	}
	raw, err := client.call(ctx, "node.invoke", payload)
	if err != nil {
		return gatewaynodes.NodeInvokeResult{InvokeID: request.InvokeID, Ok: false, Error: err.Error()}, err
	}
	var result gatewaynodes.NodeInvokeResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return gatewaynodes.NodeInvokeResult{InvokeID: request.InvokeID, Ok: false, Error: err.Error()}, err
	}
	if !result.Ok {
		if message := strings.TrimSpace(result.Error); message != "" {
			return result, errors.New(message)
		}
		return result, errors.New("canvas invoke failed")
	}
	return result, nil
}

func (client *canvasGatewayClient) call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	if client == nil || client.conn == nil {
		return nil, errors.New("gateway connection unavailable")
	}
	paramsPayload, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}
	requestID := uuid.NewString()
	frame := gatewaycontrolplane.RequestFrame{
		Type:   "req",
		ID:     requestID,
		Method: strings.TrimSpace(method),
		Params: paramsPayload,
	}
	if err := client.writeJSONWithContext(ctx, frame); err != nil {
		return nil, err
	}
	for {
		raw, err := client.readRawWithContext(ctx)
		if err != nil {
			return nil, err
		}
		var gatewayErr gatewaycontrolplane.GatewayError
		if err := json.Unmarshal(raw, &gatewayErr); err == nil && strings.TrimSpace(gatewayErr.Code) != "" {
			message := strings.TrimSpace(gatewayErr.Message)
			if message == "" {
				message = gatewayErr.Code
			}
			return nil, errors.New(message)
		}
		var envelope struct {
			Type string `json:"type"`
			ID   string `json:"id"`
		}
		if err := json.Unmarshal(raw, &envelope); err != nil {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(envelope.Type), "event") {
			continue
		}
		var response canvasGatewayResponseFrame
		if err := json.Unmarshal(raw, &response); err != nil {
			continue
		}
		if strings.TrimSpace(response.ID) != requestID {
			continue
		}
		if !response.OK {
			if response.Error != nil {
				message := strings.TrimSpace(response.Error.Message)
				if message == "" {
					message = strings.TrimSpace(response.Error.Code)
				}
				if message != "" {
					return nil, errors.New(message)
				}
			}
			return nil, errors.New("gateway request failed")
		}
		return response.Payload, nil
	}
}

func (client *canvasGatewayClient) writeJSONWithContext(ctx context.Context, value any) error {
	if client == nil || client.conn == nil {
		return errors.New("gateway connection unavailable")
	}
	if err := client.applyDeadline(ctx); err != nil {
		return err
	}
	return websocket.JSON.Send(client.conn, value)
}

func (client *canvasGatewayClient) readRawWithContext(ctx context.Context) ([]byte, error) {
	if client == nil || client.conn == nil {
		return nil, errors.New("gateway connection unavailable")
	}
	if err := client.applyDeadline(ctx); err != nil {
		return nil, err
	}
	var raw []byte
	if err := websocket.Message.Receive(client.conn, &raw); err != nil {
		return nil, err
	}
	return raw, nil
}

func (client *canvasGatewayClient) applyDeadline(ctx context.Context) error {
	if client == nil || client.conn == nil {
		return errors.New("gateway connection unavailable")
	}
	deadline := time.Now().Add(time.Duration(client.timeoutMs) * time.Millisecond)
	if ctx != nil {
		if until, ok := ctx.Deadline(); ok && until.Before(deadline) {
			deadline = until
		}
		if err := ctx.Err(); err != nil {
			return err
		}
	}
	if err := client.conn.SetDeadline(deadline); err != nil {
		return fmt.Errorf("set gateway deadline: %w", err)
	}
	return nil
}
