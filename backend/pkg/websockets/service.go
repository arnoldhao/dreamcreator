package websockets

import (
	"context"
	"dreamcreator/backend/consts"
	"dreamcreator/backend/types"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"dreamcreator/backend/pkg/logger"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

type Client struct {
	id       string
	conn     *websocket.Conn
	send     chan types.WSResponse
	service  *Service
	lastPing time.Time
}

type Service struct {
	ctx         context.Context
	clients     map[string]*Client
	clientMutex sync.RWMutex
	broadcast   chan types.WSResponse
	heartbeat   *time.Ticker
}

const (
	// 心跳间隔
	heartbeatInterval = 30 * time.Second
	// 客户端超时时间
	clientTimeout = 90 * time.Second
	// 写入超时
	writeWait = 10 * time.Second
	// 读取超时
	pongWait = 60 * time.Second
	// Ping间隔
	pingPeriod = (pongWait * 9) / 10
)

func New() *Service {
	return &Service{
		clients:   make(map[string]*Client),
		broadcast: make(chan types.WSResponse, 256),
		heartbeat: time.NewTicker(heartbeatInterval),
	}
}

func (s *Service) SetContext(ctx context.Context) {
	s.ctx = ctx
}

func (s *Service) Start() {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Wails应用允许所有来源
		},
		ReadBufferSize:    1024,
		WriteBufferSize:   1024,
		EnableCompression: true,
	}

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		s.handleWebSocket(upgrader, w, r)
	})

	// 启动广播处理器
	go s.handleBroadcast()
	// 启动心跳检查
	go s.handleHeartbeat()

	go func() {
		logger.Info("WebSocket server starting", zap.Int("port", consts.WS_PORT))
		err := http.ListenAndServe(fmt.Sprintf(":%v", consts.WS_PORT), nil)
		if err != nil {
			logger.Error("WebSocket server error", zap.Error(err))
		}
	}()
}

func (s *Service) handleWebSocket(upgrader websocket.Upgrader, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error("Failed to upgrade connection", zap.Error(err))
		return
	}

	// 启用写压缩并设置读取限制，增强稳健性
	conn.EnableWriteCompression(true)
	// 限制单条消息体大小，防止异常占用
	conn.SetReadLimit(1 << 20) // 1MB

	clientID := r.URL.Query().Get("id")
	if clientID == "" {
		clientID = "default"
	}

	client := &Client{
		id:       clientID,
		conn:     conn,
		send:     make(chan types.WSResponse, 256),
		service:  s,
		lastPing: time.Now(),
	}

	// 注册客户端
	var oldClient *Client
	s.clientMutex.Lock()
	// 如果已存在同ID客户端，先移除旧连接；旧连接的 readPump defer 不能误删新连接
	if existing, exists := s.clients[clientID]; exists {
		oldClient = existing
		delete(s.clients, clientID)
	}
	s.clients[clientID] = client
	s.clientMutex.Unlock()

	if oldClient != nil {
		_ = oldClient.conn.Close()
	}

	logger.Info("WebSocket client connected", zap.String("id", clientID))

	// 启动客户端处理协程
	go client.writePump()
	go client.readPump()
}

func (c *Client) readPump() {
	defer func() {
		c.service.unregisterClient(c)
		c.conn.Close()
	}()

	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.lastPing = time.Now()
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Error("WebSocket read error", zap.String("id", c.id), zap.Error(err))
			}
			break
		}

		var msg types.WSRequest
		if err := json.Unmarshal(message, &msg); err != nil {
			logger.Error("Failed to parse message", zap.String("id", c.id), zap.Error(err))
			continue
		}

		// 处理心跳响应
		if msg.Event == "pong" {
			c.lastPing = time.Now()
			continue
		}

		// 处理其他消息类型
		switch consts.WSNamespace(msg.Namespace) {
		default:
			logger.Debug("Received message",
				zap.String("id", c.id),
				zap.String("namespace", string(msg.Namespace)),
				zap.String("event", string(msg.Event)))
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteJSON(message); err != nil {
				logger.Error("Failed to write message", zap.String("id", c.id), zap.Error(err))
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				logger.Error("Failed to send ping", zap.String("id", c.id), zap.Error(err))
				return
			}

		case <-c.service.ctx.Done():
			return
		}
	}
}

func (s *Service) handleBroadcast() {
	for {
		select {
		case message := <-s.broadcast:
			s.clientMutex.RLock()
			for _, client := range s.clients {
				// 如果指定了客户端ID，只发送给指定客户端
				if message.ClientID != "" && message.ClientID != client.id {
					continue
				}

				select {
				case client.send <- message:
				default:
					// 客户端发送队列满，关闭连接
					close(client.send)
					delete(s.clients, client.id)
				}
			}
			s.clientMutex.RUnlock()

		case <-s.ctx.Done():
			return
		}
	}
}

func (s *Service) handleHeartbeat() {
	for {
		select {
		case <-s.heartbeat.C:
			now := time.Now()
			heartbeatMsg := types.WSResponse{
				Namespace: "system",
				Event:     "heartbeat",
				Data: types.HeartbeatData{
					Timestamp: now.Unix(),
					Message:   "ping",
				},
			}

			s.clientMutex.RLock()
			for id, client := range s.clients {
				// 检查客户端是否超时
				if now.Sub(client.lastPing) > clientTimeout {
					logger.Warn("Client timeout, removing", zap.String("id", id))
					close(client.send)
					client.conn.Close()
					delete(s.clients, id)
					continue
				}

				// 发送心跳
				select {
				case client.send <- heartbeatMsg:
				default:
					// 发送队列满
				}
			}
			s.clientMutex.RUnlock()

		case <-s.ctx.Done():
			s.heartbeat.Stop()
			return
		}
	}
}

func (s *Service) unregisterClient(client *Client) {
	s.clientMutex.Lock()
	defer s.clientMutex.Unlock()

	current, exists := s.clients[client.id]
	if !exists || current != client {
		return
	}

	close(current.send)
	delete(s.clients, client.id)
	logger.Info("WebSocket client disconnected", zap.String("id", client.id))
}

func (s *Service) SendToClient(message types.WSResponse) {
	select {
	case s.broadcast <- message:
		// 消息已发送到广播队列
	default:
		// 广播队列满，记录警告
		logger.Warn("WebSocket broadcast queue is full, message dropped")
	}
}

// 发送给特定客户端
func (s *Service) SendToSpecificClient(clientID string, message types.WSResponse) {
	message.ClientID = clientID
	s.SendToClient(message)
}

// 获取连接状态
func (s *Service) GetConnectedClients() []string {
	s.clientMutex.RLock()
	defer s.clientMutex.RUnlock()

	clients := make([]string, 0, len(s.clients))
	for id := range s.clients {
		clients = append(clients, id)
	}
	return clients
}
