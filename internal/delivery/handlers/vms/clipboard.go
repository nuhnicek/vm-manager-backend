package vms

import (
	"net/http"
	"strconv"
	"vm-manager-backend/internal/core/vms"
	"vm-manager-backend/types"

	"github.com/gin-gonic/gin"
)

type SetClipboardRequest struct {
	VMID int    `json:"vmid"`
	Text string `json:"text"`
	Node string `json:"node,omitempty"`
}

func GetConsoleClipboard(conn types.Conn) gin.HandlerFunc {
	return func(c *gin.Context) {
		uuidVal, exists := c.Get("uuid")
		if !exists {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		userUUID := uuidVal.(string)
		roleVal, _ := c.Get("role")
		role := "student"
		if r, ok := roleVal.(string); ok && r != "" {
			role = r
		}

		vmidStr := c.Query("vmid")
		if vmidStr == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "vmid query parameter is required"})
			return
		}

		vmid, err := strconv.Atoi(vmidStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid vmid"})
			return
		}

		node := c.Query("node")
		targetNodes := []string{}
		if node != "" {
			targetNodes = append(targetNodes, node)
		}

		text, err := vms.GetVMClipboard(conn, userUUID, role, vmid, targetNodes...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "text": ""})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"text": text,
		})
	}
}

func SetConsoleClipboard(conn types.Conn) gin.HandlerFunc {
	return func(c *gin.Context) {
		uuidVal, exists := c.Get("uuid")
		if !exists {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		userUUID := uuidVal.(string)
		roleVal, _ := c.Get("role")
		role := "student"
		if r, ok := roleVal.(string); ok && r != "" {
			role = r
		}

		var req SetClipboardRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}

		if req.VMID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "vmid is required"})
			return
		}

		targetNodes := []string{}
		if req.Node != "" {
			targetNodes = append(targetNodes, req.Node)
		}

		if err := vms.SetVMClipboard(conn, userUUID, role, req.VMID, req.Text, targetNodes...); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Clipboard updated successfully",
		})
	}
}
