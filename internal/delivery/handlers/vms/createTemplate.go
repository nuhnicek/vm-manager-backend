package vms

import (
	"net/http"
	"vm-manager-backend/internal/core/vms"
	"vm-manager-backend/types"

	"github.com/gin-gonic/gin"
)

type convertVMBody struct {
	VMID int    `json:"vmid" binding:"required"`
	Node string `json:"node,omitempty"`
}

func ConvertVMToTemplate(conn types.Conn) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body convertVMBody
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "vmid is required"})
			return
		}

		userUUID, _ := c.Get("uuid")
		uuidStr, _ := userUUID.(string)

		userRole, _ := c.Get("role")
		roleStr, _ := userRole.(string)
		if roleStr == "teacher" || roleStr == "admin" {
			uuidStr = ""
		}

		template, err := vms.ConvertVMToTemplate(conn, uuidStr, roleStr, body.VMID, body.Node)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":  "VM successfully converted to template",
			"template": template,
		})
	}
}

func CloneVMAsTemplate(conn types.Conn) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req types.CreateTemplateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid template creation request: " + err.Error()})
			return
		}

		userUUID, _ := c.Get("uuid")
		uuidStr, _ := userUUID.(string)

		userRole, _ := c.Get("role")
		roleStr, _ := userRole.(string)
		if roleStr == "teacher" || roleStr == "admin" {
			uuidStr = ""
		}

		template, err := vms.CloneVMAsTemplate(conn, uuidStr, roleStr, req)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"message":  "Template successfully created from clone",
			"template": template,
		})
	}
}
