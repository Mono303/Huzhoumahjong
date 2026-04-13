package handler

import (
	"net/http"

	"github.com/Mono303/Huzhoumahjong/backend/internal/middleware"
	"github.com/gin-gonic/gin"
)

type guestLoginRequest struct {
	Username string `json:"username"`
}

func (api *API) guestLogin(c *gin.Context) {
	var request guestLoginRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, token, err := api.users.GuestLogin(c.Request.Context(), request.Username)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user":  user,
	})
}

func (api *API) me(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"user": middleware.UserFromContext(c),
	})
}
