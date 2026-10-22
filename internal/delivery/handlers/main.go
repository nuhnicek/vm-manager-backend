package handlers

import (
	"vm-manager-backend/types"

	"github.com/gin-gonic/gin"
)

func Main(conn types.Conn) gin.HandlerFunc {
	return func(c *gin.Context) {
		uuid, _ := c.Get("uuid")
		role, _ := c.Get("role")
		email, _ := c.Get("email")
		c.JSON(200, gin.H{
			"message": "pong",
			"uuid":    uuid,
			"role":    role,
			"email":   email,
		})
	}
}
