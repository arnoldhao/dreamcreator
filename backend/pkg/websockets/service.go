package websockets

import (
	"CanMe/backend/consts"
	"CanMe/backend/types"
	"context"
	"crypto/tls"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type Service struct {
	ctx  context.Context
	send chan types.WSResponse
}

func New() *Service {
	return &Service{
		send: make(chan types.WSResponse, 100),
	}
}

func createWebSocketClient() *http.Client {
	transport := &http.Transport{
		// 不使用代理或使用专用代理
		DisableCompression: true,
		TLSClientConfig:    &tls.Config{InsecureSkipVerify: true},
		// 设置较短的超时
		IdleConnTimeout: 30 * time.Second,
	}

	return &http.Client{
		Transport: transport,
		Timeout:   1 * time.Minute, // WebSocket连接应该有较短的超时
	}
}

func (s *Service) SetContext(ctx context.Context) {
	s.ctx = ctx
}

func (s *Service) Start() {
	client := createWebSocketClient()
	http.DefaultClient = client
	http.HandleFunc("/ws", s.handleWebSocket)

	go func() {
		err := http.ListenAndServe(":34444", nil)
		if err != nil {
			runtime.LogErrorf(s.ctx, "WebSocket server error: %v", err)
		}
	}()
}

func (s *Service) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // allow all origins
		},
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		runtime.LogErrorf(s.ctx, "Failed to upgrade connection: %v", err)
		return
	}

	id := r.URL.Query().Get("id")
	go s.readPump(id, conn)
	go s.writePump(id, conn)
}

func (s *Service) readPump(id string, conn *websocket.Conn) {
	defer func() {
		conn.Close()
	}()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				runtime.LogErrorf(s.ctx, "error: %v", err)
			}
			break
		}
		var msg types.WSRequest
		if err := json.Unmarshal(message, &msg); err != nil {
			runtime.LogErrorf(s.ctx, "Failed to parse message from %s: %v", id, err)
			continue
		}

		// switch Namespace
		switch consts.WSNamespace(msg.Namespace) {
		default:
			runtime.LogErrorf(s.ctx, "Unexpected namespace from %s: %s", id, msg.Namespace)
		}
	}
}

func (s *Service) writePump(id string, conn *websocket.Conn) {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		conn.Close()
	}()

	for {
		select {
		case message, ok := <-s.send:
			if !ok {
				// 通道已关闭，发送关闭消息
				conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// 处理消息...
			w, err := conn.NextWriter(websocket.TextMessage)
			if err != nil {
				runtime.LogErrorf(s.ctx, "Failed to get next writer: %v, %v", id, err)
				return
			}

			msg, err := json.Marshal(message)
			if err != nil {
				runtime.LogErrorf(s.ctx, "Failed to marshal message: %v, %v", id, err)
				return
			}

			w.Write(msg)
			if err := w.Close(); err != nil {
				runtime.LogErrorf(s.ctx, "Failed to close writer: %v, %v", id, err)
				return
			}

		case <-ticker.C:
			// 发送 ping 消息
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (s *Service) SendToClient(message types.WSResponse) {
	s.send <- message
}
