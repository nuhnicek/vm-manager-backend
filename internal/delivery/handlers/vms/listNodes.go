package vms

import (
	"net/http"
	"vm-manager-backend/internal/core/vms"
	"vm-manager-backend/types"

	"github.com/gin-gonic/gin"
)

func ListNodes(conn types.Conn) gin.HandlerFunc {
	return func(c *gin.Context) {
		nodes := vms.ListNodes(conn)
		c.JSON(http.StatusOK, nodes)
	}
}
