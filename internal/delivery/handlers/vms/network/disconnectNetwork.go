package network

import (
	"fmt"
	"net/http"
	"strconv"
	"vm-manager-backend/internal/core/vms/vlan"
	"vm-manager-backend/types"

	"github.com/gin-gonic/gin"
)

type disconnectBody struct {
	Vmid1 int `json:"vmid1"`
	Vmid2 int `json:"vmid2"`
}

func DisconnectNetwork(conn types.Conn) gin.HandlerFunc {
	return func(c *gin.Context) {
		uuid, _ := c.Get("uuid")
		var body disconnectBody
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}

		userUUID := ""
		if uuid != nil {
			userUUID = uuid.(string)
		}

		err := vlan.DisconnectVLAN(conn, userUUID, strconv.Itoa(body.Vmid1), strconv.Itoa(body.Vmid2))
		if err != nil {
			fmt.Printf("failed to disconnect network: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "disconnected"})
	}
}
