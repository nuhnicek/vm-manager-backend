package network

import (
	"fmt"
	"net/http"
	"strconv"
	"vm-manager-backend/internal/core/vms/vlan"
	"vm-manager-backend/types"

	"github.com/gin-gonic/gin"
)

type netBody struct {
	Vmid1      int    `json:"vmid1"`
	Vmid2      int    `json:"vmid2"`
	Vmid       int    `json:"vmid"`
	TargetVMID int    `json:"target_vm_id"`
	Nic1       string `json:"nic1,omitempty"`
	Nic2       string `json:"nic2,omitempty"`
	LocalNIC   string `json:"local_nic,omitempty"`
	TargetNIC  string `json:"target_nic,omitempty"`
}

func CreateNetwork(conn types.Conn) gin.HandlerFunc {
	return func(c *gin.Context) {
		uuid, _ := c.Get("uuid")
		roleVal, _ := c.Get("role")
		role := ""
		if r, ok := roleVal.(string); ok {
			role = r
		}

		var body netBody
		err := c.ShouldBindJSON(&body)
		if err != nil {
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
		if body.Vmid2 == 0 && body.TargetVMID != 0 {
			body.Vmid2 = body.TargetVMID
		}
		if body.Nic1 == "" && body.LocalNIC != "" {
			body.Nic1 = body.LocalNIC
		}
		if body.Nic2 == "" && body.TargetNIC != "" {
			body.Nic2 = body.TargetNIC
		}
		if body.Vmid2 == 0 && body.Nic1 != "" {
			if n, err := strconv.Atoi(body.Nic1); err == nil && n > 0 {
				body.Vmid2 = n
				body.Nic1 = ""
			}
		}

		if body.Vmid1 == 0 || body.Vmid2 == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid vmid1 or vmid2"})
			return
		}

		err = vlan.ConnectVLAN(strconv.Itoa(body.Vmid1), body.Nic1, strconv.Itoa(body.Vmid2), body.Nic2, conn, userUUID, role)
		if err != nil {
			fmt.Printf("failed to connect network: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "created"})
	}
}
