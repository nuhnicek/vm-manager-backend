package network

import (
	"fmt"
	"net/http"
	"strconv"
	"vm-manager-backend/internal/core/vms/vlan"
	"vm-manager-backend/types"

	"github.com/gin-gonic/gin"
)

type addNicBody struct {
	Vmid int `json:"vmid"`
}

func AddNIC(conn types.Conn) gin.HandlerFunc {
	return func(c *gin.Context) {
		uuid, _ := c.Get("uuid")
		roleVal, _ := c.Get("role")
		role := ""
		if r, ok := roleVal.(string); ok {
			role = r
		}

		var body addNicBody
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}

		userUUID := ""
		if uuid != nil {
			userUUID = uuid.(string)
		}

		nic, err := vlan.AddNIC(conn, userUUID, strconv.Itoa(body.Vmid), role)
		if err != nil {
			fmt.Printf("failed to add NIC to machine %d: %v\n", body.Vmid, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "added", "nic": nic})
	}
}
