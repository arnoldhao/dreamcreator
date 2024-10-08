package websockets

import (
	"CanMe/backend/consts"
	innerWS "CanMe/backend/pkg/websocket"
	innerInterfaces "CanMe/backend/services/innerInterfaces"
	"CanMe/backend/types"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type Service struct {
	ctx              context.Context
	upgrader         websocket.Upgrader
	wsManager        *innerWS.Manager
	translateService innerInterfaces.TranslateServiceInterface
	ollamaService    innerInterfaces.OllamaServiceInterface
	TranslationList  []string
}

func New() *Service {
	return &Service{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // allow all origins
			},
		},
		wsManager:       innerWS.NewManager(),
		TranslationList: []string{},
	}
}

func (s *Service) Start() {
	go s.wsManager.Start()
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
	translateService innerInterfaces.TranslateServiceInterface,
	ollamaService innerInterfaces.OllamaServiceInterface,
) {
	s.ctx = ctx
	s.translateService = translateService
	s.ollamaService = ollamaService
}

func (s *Service) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	client := &innerWS.Client{
		ID:   r.URL.Query().Get("id"), // get client ID from query parameters
		Conn: conn,
		Send: make(chan string, 256),
	}

	s.wsManager.Register <- client

	go s.readPump(client)
	go s.writePump(client)
}

func (s *Service) readPump(client *innerWS.Client) {
	defer func() {
		s.wsManager.Unregister <- client
		client.Conn.Close()
	}()

	for {
		_, message, err := client.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}
		var msg types.WSRequestMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			runtime.LogErrorf(s.ctx, "Failed to parse message from %s: %v", client.ID, err)
			continue
		}

		switch consts.WSRequestType(msg.Event) {
		case consts.REQUEST_TRANSLATION_START:
			s.addTranslation(msg.Translate.ID)
			err = s.translateService.AddTranslation(msg.Translate.ID, msg.Translate.OriginalSubtitleID, msg.Translate.Language)
			if err != nil {
				runtime.LogError(s.ctx, "translation_start error: "+err.Error())
				s.SendToClient(client.ID, types.TranslateResponse{
					ID:      msg.Translate.ID,
					Error:   true,
					Message: err.Error(),
				}.WSResponseMessage())
			}
		case consts.REQUEST_OLLAMA_PULL:
			err = s.ollamaService.Pull(msg.Ollama.ID, msg.Ollama.Model)
			if err != nil {
				s.SendToClient(client.ID, types.OllamaResponse{
					ID:      msg.Translate.ID,
					Error:   true,
					Message: err.Error(),
				}.WSResponseMessage())
			}
		default:
			runtime.LogErrorf(s.ctx, "Unexpected event from %s: %s", client.ID, msg.Event)
		}
	}
}

func (s *Service) writePump(client *innerWS.Client) {
	defer func() {
		client.Conn.Close()
	}()

	for message := range client.Send {
		w, err := client.Conn.NextWriter(websocket.TextMessage)
		if err != nil {
			return
		}
		w.Write([]byte(message))
		if err := w.Close(); err != nil {
			return
		}
	}
}

func (s *Service) SendToClient(clientID string, message string) {
	if s.isProcessing(clientID) {
		s.wsManager.SendToClient(clientID, message)
	} else {
		runtime.LogInfo(s.ctx, fmt.Sprintf("client: %v is not processing, skip", clientID))
	}
}

func (s *Service) addTranslation(id string) {
	s.TranslationList = append(s.TranslationList, id)
}

func (s *Service) RemoveTranslation(id string) {
	for i, v := range s.TranslationList {
		if v == id {
			s.TranslationList = append(s.TranslationList[:i], s.TranslationList[i+1:]...)
		}
	}
}

func (s *Service) isProcessing(id string) bool {
	for _, v := range s.TranslationList {
		if v == id {
			return true
		}
	}
	return false
}
