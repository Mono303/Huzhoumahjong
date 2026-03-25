package middleware

import (
	"net/http"
	"strings"

	"github.com/Mono303/Huzhoumahjong/backend/internal/model"
	"github.com/Mono303/Huzhoumahjong/backend/internal/service"
	"github.com/gin-gonic/gin"
)

const userContextKey = "current_user"

func Auth(userService *service.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}

		token := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
		user, err := userService.Authenticate(c.Request.Context(), token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		if user == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid session"})
			return
		}

		c.Set(userContextKey, user)
		c.Next()
	}
}

func UserFromContext(c *gin.Context) *model.User {
	raw, exists := c.Get(userContextKey)
	if !exists {
		return nil
	}
	user, _ := raw.(*model.User)
	return user
}
