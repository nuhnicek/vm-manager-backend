package vms

import (
	"net/http"
	"vm-manager-backend/internal/core/vms"
	"vm-manager-backend/types"

	"github.com/gin-gonic/gin"
)

func ListTemplates(conn types.Conn) gin.HandlerFunc {
	return func(c *gin.Context) {
		vm, err := vms.ListTemplates(conn)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}
		c.JSON(http.StatusOK, vm)
	}
}
