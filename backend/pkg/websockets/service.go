package websockets

import (
	"CanMe/backend/consts"
	"CanMe/backend/types"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"CanMe/backend/pkg/logger"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

type Service struct {
	ctx         context.Context
	send        chan types.WSResponse
	connections map[string]*websocket.Conn // 管理多个连接
	connMutex   sync.RWMutex               // 保护连接映射
}

func New() *Service {
	return &Service{
		send:        make(chan types.WSResponse, 100),
		connections: make(map[string]*websocket.Conn),
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
		err := http.ListenAndServe(fmt.Sprintf(":%v", consts.WS_PORT), nil)
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
	// 移除心跳ticker，只保留消息处理
	defer func() {
		conn.Close()
		logger.Info("WebSocket connection closed", zap.String("id", id))
	}()

	for {
		select {
		case message, ok := <-s.send:
			if !ok {
				// 通道已关闭，发送关闭消息
				conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// 设置写入超时
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))

			// 尝试发送消息
			w, err := conn.NextWriter(websocket.TextMessage)
			if err != nil {
				logger.Error("Failed to get next writer - connection may be broken",
					zap.String("id", id),
					zap.Error(err))
				// 连接断开，触发重连
				s.handleConnectionLost(id)
				return
			}

			msg, err := json.Marshal(message)
			if err != nil {
				logger.Error("Failed to marshal message",
					zap.String("id", id),
					zap.Error(err))
				continue
			}

			_, err = w.Write(msg)
			if err != nil {
				logger.Error("Failed to write message - connection lost",
					zap.String("id", id),
					zap.Error(err))
				w.Close()
				s.handleConnectionLost(id)
				return
			}

			if err := w.Close(); err != nil {
				logger.Error("Failed to close writer - connection lost",
					zap.String("id", id),
					zap.Error(err))
				s.handleConnectionLost(id)
				return
			}

			logger.Debug("Message sent successfully", zap.String("id", id))
		}
	}
}

// 新增：处理连接丢失
func (s *Service) handleConnectionLost(id string) {
	logger.Warn("WebSocket connection lost, will attempt to reconnect on next message",
		zap.String("id", id))
	// 这里可以发送连接丢失事件给前端（如果连接还活着的话）
	// 或者设置一个标志，让下次SendToClient时重新建立连接
}

func (s *Service) SendToClient(message types.WSResponse) {
	select {
	case s.send <- message:
		// 消息已发送到队列
	default:
		// 队列满了，记录警告
		logger.Warn("WebSocket send queue is full, message dropped")
	}
}
