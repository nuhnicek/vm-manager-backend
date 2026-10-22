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
		err, uuid := auth.Session(session, conn)
		if err != nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		c.Set("uuid", uuid)

		user, err := auth.GetUserByUUID(conn, uuid)
		if err == nil && user != nil {
			c.Set("role", user.Role)
			c.Set("email", user.Email)
		} else {
			c.Set("role", "student")
		}

		c.Next()
	}
}

func RequireTeacher(conn types.Conn) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists || (role != "teacher" && role != "admin") {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden: requires teacher role"})
			c.Abort()
			return
		}
		c.Next()
	}
}
