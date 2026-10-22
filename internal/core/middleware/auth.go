package middleware

import (
	"net/http"
	"vm-manager-backend/internal/core/auth"
	"vm-manager-backend/types"

	"github.com/gin-gonic/gin"
)

func Auth(conn types.Conn) gin.HandlerFunc {
	return func(c *gin.Context) {
		session := c.GetHeader("Authorization")
		if session == "" {
			session = c.Query("token")
		}
		if session == "" {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		err, uuid, role := auth.Session(session, conn)
		if err != nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		c.Set("uuid", uuid)
		c.Set("role", role)
		c.Next()
	}
}
