package handler

import (
	"net/http"
	"time"

	"github.com/Mono303/Huzhoumahjong/backend/internal/middleware"
	"github.com/Mono303/Huzhoumahjong/backend/internal/service"
	wsserver "github.com/Mono303/Huzhoumahjong/backend/internal/ws"
	"github.com/gin-gonic/gin"
)

type API struct {
	users *service.UserService
	rooms *service.RoomService
	hub   *wsserver.Hub
}

func New(users *service.UserService, rooms *service.RoomService, hub *wsserver.Hub) *API {
	return &API{
		users: users,
		rooms: rooms,
		hub:   hub,
	}
}

func (api *API) Register(router *gin.Engine) {
	router.GET("/healthz", api.health)

	v1 := router.Group("/api/v1")
	v1.POST("/auth/guest", api.guestLogin)
	v1.GET("/ws", api.serveWS)

	authed := v1.Group("")
	authed.Use(middleware.Auth(api.users))
	authed.GET("/users/me", api.me)
	authed.GET("/matches/history", api.history)
	authed.POST("/rooms", api.createRoom)
	authed.GET("/rooms/:code", api.getRoom)
	authed.POST("/rooms/:code/join", api.joinRoom)
	authed.POST("/rooms/:code/leave", api.leaveRoom)
}

func (api *API) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"date":   time.Now().UTC(),
	})
}
