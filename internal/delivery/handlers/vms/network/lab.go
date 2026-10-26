package network

import (
	"fmt"
	"net/http"
	"strconv"
	"vm-manager-backend/internal/core/vms/vlan"
	"vm-manager-backend/types"

	"github.com/gin-gonic/gin"
)

type labNetBody struct {
	VMID int `json:"vmid"`
}

func ConnectLabNetwork(conn types.Conn) gin.HandlerFunc {
	return func(c *gin.Context) {
		uuid, _ := c.Get("uuid")
		roleVal, _ := c.Get("role")
		role := ""
		if r, ok := roleVal.(string); ok {
			role = r
		}

		var body labNetBody
		if err := c.ShouldBindJSON(&body); err != nil || body.VMID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid vmid"})
			return
		}

		userUUID := ""
		if uuid != nil {
			userUUID = uuid.(string)
		}

		err := vlan.ConnectToLabNetwork(conn, userUUID, strconv.Itoa(body.VMID), role)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "machine successfully connected to lab network"})
	}
}

func DisconnectLabNetwork(conn types.Conn) gin.HandlerFunc {
	return func(c *gin.Context) {
		uuid, _ := c.Get("uuid")
		roleVal, _ := c.Get("role")
		role := ""
		if r, ok := roleVal.(string); ok {
			role = r
		}

		var body labNetBody
		if err := c.ShouldBindJSON(&body); err != nil || body.VMID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid vmid"})
			return
		}

		userUUID := ""
		if uuid != nil {
			userUUID = uuid.(string)
		}

		err := vlan.DisconnectFromLabNetwork(conn, userUUID, strconv.Itoa(body.VMID), role)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to disconnect from lab network: %v", err)})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "machine successfully disconnected from lab network"})
	}
}
