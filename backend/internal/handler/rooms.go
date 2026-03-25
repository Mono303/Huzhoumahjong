package handler

import (
	"errors"
	"net/http"

	"github.com/Mono303/Huzhoumahjong/backend/internal/middleware"
	"github.com/Mono303/Huzhoumahjong/backend/internal/model"
	"github.com/Mono303/Huzhoumahjong/backend/internal/service"
	"github.com/gin-gonic/gin"
)

type createRoomRequest struct {
	Settings model.RoomSettings `json:"settings"`
}

func (api *API) createRoom(c *gin.Context) {
	var request createRoomRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	room, err := api.rooms.CreateRoom(c.Request.Context(), middleware.UserFromContext(c), request.Settings)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"room": room})
}

func (api *API) getRoom(c *gin.Context) {
	room, err := api.rooms.GetRoomSnapshot(c.Request.Context(), c.Param("code"))
	if err != nil {
		api.writeRoomError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"room": room})
}

func (api *API) joinRoom(c *gin.Context) {
	room, err := api.rooms.JoinRoom(c.Request.Context(), c.Param("code"), middleware.UserFromContext(c))
	if err != nil {
		api.writeRoomError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"room": room})
}

func (api *API) leaveRoom(c *gin.Context) {
	room, err := api.rooms.LeaveRoom(c.Request.Context(), c.Param("code"), middleware.UserFromContext(c))
	if err != nil {
		api.writeRoomError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"room": room})
}

func (api *API) history(c *gin.Context) {
	user := middleware.UserFromContext(c)
	items, err := api.rooms.History(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (api *API) writeRoomError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrRoomNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrRoomIsFull),
		errors.Is(err, service.ErrRoomAlreadyPlaying),
		errors.Is(err, service.ErrPlayerNotInRoom),
		errors.Is(err, service.ErrOnlyHostCanStart),
		errors.Is(err, service.ErrPlayersNotReady),
		errors.Is(err, service.ErrRoomRequiresPlayers):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}
