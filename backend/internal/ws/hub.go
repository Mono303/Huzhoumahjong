package ws

import (
	"encoding/json"
	"sync"

	"github.com/Mono303/Huzhoumahjong/backend/internal/model"
	"github.com/gorilla/websocket"
)

type Client struct {
	UserID   string
	RoomCode string
	Conn     *websocket.Conn
	Send     chan model.Envelope
}

type Hub struct {
	mu    sync.RWMutex
	rooms map[string]map[*Client]struct{}
	users map[string]map[*Client]struct{}
}

func NewHub() *Hub {
	return &Hub{
		rooms: map[string]map[*Client]struct{}{},
		users: map[string]map[*Client]struct{}{},
	}
}

func NewClient(conn *websocket.Conn, userID, roomCode string) *Client {
	return &Client{
		UserID:   userID,
		RoomCode: roomCode,
		Conn:     conn,
		Send:     make(chan model.Envelope, 16),
	}
}

func (h *Hub) Register(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.rooms[client.RoomCode]; !ok {
		h.rooms[client.RoomCode] = map[*Client]struct{}{}
	}
	h.rooms[client.RoomCode][client] = struct{}{}

	if _, ok := h.users[client.UserID]; !ok {
		h.users[client.UserID] = map[*Client]struct{}{}
	}
	h.users[client.UserID][client] = struct{}{}
}

func (h *Hub) Unregister(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if roomClients, ok := h.rooms[client.RoomCode]; ok {
		delete(roomClients, client)
		if len(roomClients) == 0 {
			delete(h.rooms, client.RoomCode)
		}
	}
	if userClients, ok := h.users[client.UserID]; ok {
		delete(userClients, client)
		if len(userClients) == 0 {
			delete(h.users, client.UserID)
		}
	}
	close(client.Send)
}

func (h *Hub) BroadcastRoom(roomCode string, envelope model.Envelope) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.rooms[roomCode] {
		h.nonBlockingSend(client, envelope)
	}
}

func (h *Hub) SendToUser(userID string, envelope model.Envelope) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.users[userID] {
		h.nonBlockingSend(client, envelope)
	}
}

func (h *Hub) nonBlockingSend(client *Client, envelope model.Envelope) {
	select {
	case client.Send <- envelope:
	default:
	}
}

func (c *Client) WriteLoop() {
	defer c.Conn.Close()
	for envelope := range c.Send {
		payload, err := json.Marshal(envelope)
		if err != nil {
			_ = c.Conn.WriteJSON(model.Envelope{
				Type: "error",
				Payload: map[string]string{
					"message": err.Error(),
				},
			})
			continue
		}
		if err := c.Conn.WriteMessage(websocket.TextMessage, payload); err != nil {
			return
		}
	}
}
