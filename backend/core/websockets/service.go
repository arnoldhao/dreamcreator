package websockets

import (
	"CanMe/backend/consts"
	"CanMe/backend/services/innerinterfaces"
	ii "CanMe/backend/services/innerinterfaces"
	"CanMe/backend/types"
	"context"
	"crypto/tls"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type Service struct {
	ctx  context.Context
	send chan types.WSResponse
	iis  iis
}

type iis struct {
	translate  ii.TranslateServiceInterface
	ollama     ii.OllamaServiceInterface
	preference ii.PreferenceServiceInterface
}

func New() *Service {
	return &Service{
		send: make(chan types.WSResponse, 100),
		iis:  iis{},
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

func (s *Service) Start() {
	client := createWebSocketClient()
	http.DefaultClient = client
	http.HandleFunc("/ws", s.handleWebSocket)

	go func() {
		err := http.ListenAndServe(":34444", nil)
		if err != nil {
			log.Printf("WebSocket server error: %v", err)
		}
	}()
}

func (s *Service) RegisterServices(
	ctx context.Context,
	translate innerinterfaces.TranslateServiceInterface,
	ollama innerinterfaces.OllamaServiceInterface,
	preference innerinterfaces.PreferenceServiceInterface,
) {
	s.ctx = ctx
	s.iis.translate = translate
	s.iis.ollama = ollama
	s.iis.preference = preference
}

// ExportClasses export strcut to frontend modules.ts
func (s *Service) ExportClasses(
	request types.WSRequest,
	response types.WSResponse,
) {
}

func (s *Service) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // allow all origins
		},
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
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
				log.Printf("error: %v", err)
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
		case consts.NAMESPACE_TRANSLATION:
			s.handleTranslation(consts.WSRequestEventType(msg.Event), msg.Data)
		case consts.NAMESPACE_OLLAMA:
			s.handleOllama(consts.WSRequestEventType(msg.Event), msg.Data)
		case consts.NAMESPACE_CHAT:
			s.handleChat(consts.WSRequestEventType(msg.Event), msg.Data)
		case consts.NAMESPACE_PROXY:
			s.handleProxy(consts.WSRequestEventType(msg.Event), msg.Data)
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

// func (s *Service) SendToClient(clientID string, message string) {
// 	if s.isProcessing(clientID) {
// 		s.wsManager.SendToClient(clientID, message)
// 	} else {
// 		runtime.LogInfo(s.ctx, fmt.Sprintf("client: %v is not processing, skip", clientID))
// 	}
// }

// func (s *Service) CommonSendToClient(clientID string, message string) {
// 	s.wsManager.SendToClient(clientID, message)
// }
