package vms

import (
	"net/http"
	"vm-manager-backend/internal/core/vms"
	"vm-manager-backend/types"

	"github.com/gin-gonic/gin"
)

func List(conn types.Conn) gin.HandlerFunc {
	return func(c *gin.Context) {
		uuid, _ := c.Get("uuid")
		roleVal, _ := c.Get("role")
		role := "student"
		if r, ok := roleVal.(string); ok && r != "" {
			role = r
		}
		vm, err := vms.List(conn, uuid.(string), role)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}
		c.JSON(http.StatusOK, vm)
	}
}
