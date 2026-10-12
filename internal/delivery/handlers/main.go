package handlers

import (
	"vm-manager-backend/types"

	"github.com/gin-gonic/gin"
)

func Main(conn types.Conn) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	}
}
