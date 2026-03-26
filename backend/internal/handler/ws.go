package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/Mono303/Huzhoumahjong/backend/internal/model"
	wsserver "github.com/Mono303/Huzhoumahjong/backend/internal/ws"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type incomingMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type readyPayload struct {
	Ready bool `json:"ready"`
}

type discardPayload struct {
	TileKey string `json:"tileKey"`
}

type actionPayload struct {
	Action   model.PlayerActionType `json:"action"`
	TileKey  string                 `json:"tileKey"`
	ChiIndex int                    `json:"chiIndex"`
}

func (api *API) serveWS(c *gin.Context) {
	token := c.Query("token")
	roomCode := c.Query("roomCode")
	if roomCode == "" || token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "roomCode and token are required"})
		return
	}

	user, err := api.users.Authenticate(c.Request.Context(), token)
	if err != nil || user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	client := wsserver.NewClient(conn, user.ID, roomCode)
	api.hub.Register(client)
	go client.WriteLoop()

	if err := api.rooms.ConnectUser(c.Request.Context(), roomCode, user); err != nil {
		api.hub.Unregister(client)
		_ = conn.WriteJSON(model.Envelope{Type: "error", Payload: gin.H{"message": err.Error()}})
		_ = conn.Close()
		return
	}

	defer func() {
		api.hub.Unregister(client)
		_ = api.rooms.DisconnectUser(c.Request.Context(), roomCode, user)
		_ = conn.Close()
	}()

	_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			return
		}
		_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))

		var message incomingMessage
		if err := json.Unmarshal(raw, &message); err != nil {
			api.hub.SendToUser(user.ID, model.Envelope{Type: "error", Payload: gin.H{"message": err.Error()}})
			continue
		}

		switch message.Type {
		case "ping":
			api.rooms.RefreshHeartbeat(c.Request.Context(), roomCode, user)
			api.hub.SendToUser(user.ID, model.Envelope{Type: "pong"})
		case "room.ready":
			var payload readyPayload
			if json.Unmarshal(message.Payload, &payload) == nil {
				if _, err := api.rooms.ToggleReady(c.Request.Context(), roomCode, user, payload.Ready); err != nil {
					api.hub.SendToUser(user.ID, model.Envelope{Type: "error", Payload: gin.H{"message": err.Error()}})
				}
			}
		case "room.start":
			if _, err := api.rooms.StartGame(c.Request.Context(), roomCode, user); err != nil {
				api.hub.SendToUser(user.ID, model.Envelope{Type: "error", Payload: gin.H{"message": err.Error()}})
			}
		case "room.leave":
			if _, err := api.rooms.LeaveRoom(c.Request.Context(), roomCode, user); err != nil {
				api.hub.SendToUser(user.ID, model.Envelope{Type: "error", Payload: gin.H{"message": err.Error()}})
				continue
			}
			return
		case "game.discard":
			var payload discardPayload
			if json.Unmarshal(message.Payload, &payload) == nil {
				if err := api.rooms.HandleDiscard(c.Request.Context(), roomCode, user, payload.TileKey); err != nil {
					api.hub.SendToUser(user.ID, model.Envelope{Type: "error", Payload: gin.H{"message": err.Error()}})
				}
			}
		case "game.action":
			var payload actionPayload
			if json.Unmarshal(message.Payload, &payload) == nil {
				if err := api.rooms.HandleGameAction(c.Request.Context(), roomCode, user, payload.Action, payload.TileKey, payload.ChiIndex); err != nil {
					api.hub.SendToUser(user.ID, model.Envelope{Type: "error", Payload: gin.H{"message": err.Error()}})
				}
			}
		default:
			api.hub.SendToUser(user.ID, model.Envelope{
				Type: "error",
				Payload: gin.H{
					"message": "unsupported ws message",
				},
			})
		}
	}
}
