package vms

import (
	"net/http"
	"vm-manager-backend/internal/core/vms"
	"vm-manager-backend/types"

	"github.com/gin-gonic/gin"
)

type vmBody struct {
	Name       string `json:"name"`
	TemplateId int    `json:"template_id"`
}

func CreateFromTemplate(conn types.Conn) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body vmBody
		uuid, _ := c.Get("uuid")
		err := c.ShouldBindJSON(&body)
		if err != nil {
			c.Status(http.StatusBadRequest)
			return
		}

		err, vm := vms.CreateFromTemplate(conn, uuid.(string), body.Name, body.TemplateId)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}
		c.JSON(http.StatusOK, vm)
		return
	}
}
