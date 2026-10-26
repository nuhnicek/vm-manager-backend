package vms

import (
	"fmt"
	"net/http"
	"strconv"
	"vm-manager-backend/internal/core/vms"
	"vm-manager-backend/types"

	"github.com/gin-gonic/gin"
)

type externalVNCRequest struct {
	VMID int    `json:"vmid"`
	Node string `json:"node"`
}

func StartExternalVNC(conn types.Conn) gin.HandlerFunc {
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

		var req externalVNCRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			vmidStr := c.Query("vmid")
			if vmidStr == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "vmid is required"})
				return
			}
			num, err := strconv.Atoi(vmidStr)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid vmid"})
				return
			}
			req.VMID = num
			req.Node = c.Query("node")
		}

		info, err := vms.StartExternalVNC(conn, userUUID, role, req.VMID, req.Node)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, info)
	}
}

func DownloadVNCProfile(conn types.Conn) gin.HandlerFunc {
	return func(c *gin.Context) {
		vmidStr := c.Query("vmid")
		if vmidStr == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "vmid query param is required"})
			return
		}

		vmid, err := strconv.Atoi(vmidStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid vmid"})
			return
		}

		filename, content, err := vms.GenerateVNCProfile(vmid)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
		c.Header("Content-Type", "application/x-vnc")
		c.String(http.StatusOK, content)
	}
}

func StopExternalVNC(conn types.Conn) gin.HandlerFunc {
	return func(c *gin.Context) {
		vmidStr := c.Param("vmid")
		if vmidStr == "" {
			vmidStr = c.Query("vmid")
		}
		if vmidStr == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "vmid is required"})
			return
		}

		vmid, err := strconv.Atoi(vmidStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid vmid"})
			return
		}

		_ = vms.StopExternalVNC(vmid)
		c.JSON(http.StatusOK, gin.H{"message": "VNC proxy stopped", "vmid": vmid})
	}
}
