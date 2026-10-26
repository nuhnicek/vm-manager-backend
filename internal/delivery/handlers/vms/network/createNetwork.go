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
	Vmid1 int    `json:"vmid1"`
	Vmid2 int    `json:"vmid2"`
	Nic1  string `json:"nic1,omitempty"`
	Nic2  string `json:"nic2,omitempty"`
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

		err = vlan.ConnectVLAN(strconv.Itoa(body.Vmid1), body.Nic1, strconv.Itoa(body.Vmid2), body.Nic2, conn, userUUID, role)
		if err != nil {
			fmt.Printf("failed to connect network: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "created"})
	}
}
