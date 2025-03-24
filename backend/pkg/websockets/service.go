package websockets

import (
	"CanMe/backend/consts"
	"CanMe/backend/types"
	"context"
	"crypto/tls"
	"encoding/json"
	"net/http"
	"time"

	"CanMe/backend/pkg/logger"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
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
			logger.Error("WebSocket server error", zap.Error(err))
			return
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
		logger.Error("Failed to upgrade connection", zap.Error(err))
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
				logger.Error("WebSocket error", zap.Error(err))
			}
			break
		}
		var msg types.WSRequest
		if err := json.Unmarshal(message, &msg); err != nil {
			logger.Error("Failed to parse message",
				zap.String("id", id),
				zap.Error(err))
			continue
		}

		// switch Namespace
		switch consts.WSNamespace(msg.Namespace) {
		default:
			logger.Error("Unexpected namespace",
				zap.String("id", id),
				zap.String("namespace", string(msg.Namespace)))
			continue
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
				logger.Error("Failed to get next writer",
					zap.String("id", id),
					zap.Error(err))
				continue
			}

			msg, err := json.Marshal(message)
			if err != nil {
				logger.Error("Failed to marshal message",
					zap.String("id", id),
					zap.Error(err))
				continue
			}

			w.Write(msg)
			if err := w.Close(); err != nil {
				logger.Error("Failed to close writer",
					zap.String("id", id),
					zap.Error(err))
				return
			}

		case <-ticker.C:
			// 发送 ping 消息
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				logger.Error("Failed to send ping message",
					zap.String("id", id),
					zap.Error(err))
				return
			}
		}
	}
}

func (s *Service) SendToClient(message types.WSResponse) {
	s.send <- message
}
