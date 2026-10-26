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
	Vmid1      int    `json:"vmid1"`
	Vmid2      int    `json:"vmid2"`
	Vmid       int    `json:"vmid"`
	Nic        string `json:"nic"`
	TargetVMID int    `json:"target_vm_id"`
	TargetNic  string `json:"target_nic"`
}

func DisconnectNetwork(conn types.Conn) gin.HandlerFunc {
	return func(c *gin.Context) {
		uuid, _ := c.Get("uuid")
		roleVal, _ := c.Get("role")
		role := ""
		if r, ok := roleVal.(string); ok {
			role = r
		}

		var body disconnectBody
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}

		userUUID := ""
		if uuid != nil {
			userUUID = uuid.(string)
		}

		if body.Vmid1 == 0 && body.Vmid != 0 {
			body.Vmid1 = body.Vmid
		}
		if body.Vmid2 == 0 {
			if body.TargetVMID != 0 {
				body.Vmid2 = body.TargetVMID
			} else if body.Nic != "" {
				if parsed, err := strconv.Atoi(body.Nic); err == nil {
					body.Vmid2 = parsed
					body.Nic = ""
				}
			}
		}

		err := vlan.DisconnectVLAN(conn, userUUID, strconv.Itoa(body.Vmid1), strconv.Itoa(body.Vmid2), body.Nic, body.TargetNic, role)
		if err != nil {
			fmt.Printf("failed to disconnect network: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "disconnected"})
	}
}
