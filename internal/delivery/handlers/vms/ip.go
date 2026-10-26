package vms

import (
	"net/http"
	"vm-manager-backend/internal/core/vms"
	"vm-manager-backend/types"

	"github.com/gin-gonic/gin"
)

func SetIP(conn types.Conn) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req types.SetIPRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "neplatné tělo požadavku"})
			return
		}

		uuidVal, _ := c.Get("uuid")
		roleVal, _ := c.Get("role")
		uuidStr, _ := uuidVal.(string)
		roleStr, _ := roleVal.(string)

		if req.VMID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "neplatné vmid"})
			return
		}

		if err := vms.SetMachineIP(conn, uuidStr, roleStr, req); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "IP adresa byla úspěšně nastavena",
			"vmid":    req.VMID,
		})
	}
}
