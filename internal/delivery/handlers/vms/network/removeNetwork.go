package network

import (
	"fmt"
	"net/http"
	"strconv"
	"vm-manager-backend/internal/core/vms/vlan"
	"vm-manager-backend/types"

	"github.com/gin-gonic/gin"
)

type removeNicBody struct {
	Vmid int    `json:"vmid"`
	Nic  string `json:"nic"`
}

func RemoveNetwork(conn types.Conn) gin.HandlerFunc {
	return func(c *gin.Context) {
		uuid, _ := c.Get("uuid")
		roleVal, _ := c.Get("role")
		role := ""
		if r, ok := roleVal.(string); ok {
			role = r
		}

		var body removeNicBody
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}

		userUUID := ""
		if uuid != nil {
			userUUID = uuid.(string)
		}

		err := vlan.RemoveNIC(conn, userUUID, strconv.Itoa(body.Vmid), body.Nic, role)
		if err != nil {
			fmt.Printf("failed to remove NIC %s: %v\n", body.Nic, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "removed", "nic": body.Nic})
	}
}
