package websocket

import (
	"sync"

	"github.com/gorilla/websocket"
)

type Client struct {
	ID   string
	Conn *websocket.Conn
	Send chan string
}

type Manager struct {
	clients    map[string]*Client
	broadcast  chan string
	Register   chan *Client
	Unregister chan *Client
	mutex      sync.Mutex
}

func NewManager() *Manager {
	return &Manager{
		clients:    make(map[string]*Client),
		broadcast:  make(chan string),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
	}
}

func (m *Manager) Start() {
	for {
		select {
		case client := <-m.Register:
			m.mutex.Lock()
			m.clients[client.ID] = client
			m.mutex.Unlock()
		case client := <-m.Unregister:
			if _, ok := m.clients[client.ID]; ok {
				m.mutex.Lock()
				delete(m.clients, client.ID)
				close(client.Send)
				m.mutex.Unlock()
			}
		case message := <-m.broadcast:
			for _, client := range m.clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					m.mutex.Lock()
					delete(m.clients, client.ID)
					m.mutex.Unlock()
				}
			}
		}
	}
}

func (m *Manager) SendToClient(clientID string, message string) {
	m.mutex.Lock()
	if client, ok := m.clients[clientID]; ok {
		client.Send <- message
	}
	m.mutex.Unlock()
}
