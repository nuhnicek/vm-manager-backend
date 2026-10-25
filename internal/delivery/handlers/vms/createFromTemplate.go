package vms

import (
	"net/http"
	"strconv"
	"strings"
	"vm-manager-backend/internal/core/projects"
	"vm-manager-backend/internal/core/vms"
	"vm-manager-backend/types"

	"github.com/gin-gonic/gin"
)

type vmBody struct {
	Name       string      `json:"name"`
	TemplateId interface{} `json:"template_id"`
	Node       string      `json:"node,omitempty"`
	ProjectId  string      `json:"project_id,omitempty"`
}

func CreateFromTemplate(conn types.Conn) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body vmBody
		uuid, _ := c.Get("uuid")
		err := c.ShouldBindJSON(&body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}

		tplID := 0
		node := body.Node

		switch v := body.TemplateId.(type) {
		case float64:
			tplID = int(v)
		case string:
			if strings.Contains(v, ":") {
				parts := strings.SplitN(v, ":", 2)
				if node == "" {
					node = parts[0]
				}
				parsed, _ := strconv.Atoi(parts[1])
				tplID = parsed
			} else {
				parsed, _ := strconv.Atoi(v)
				tplID = parsed
			}
		}

		if tplID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid template_id"})
			return
		}

		err, vmUuid, newVmID := vms.CreateFromTemplate(conn, uuid.(string), body.Name, tplID, node)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if body.ProjectId != "" {
			_ = projects.AddVMToProject(conn, body.ProjectId, newVmID, uuid.(string), true)
		}

		c.JSON(http.StatusOK, gin.H{
			"uuid":       vmUuid,
			"vmid":       newVmID,
			"project_id": body.ProjectId,
		})
	}
}
