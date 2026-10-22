package labs

import (
	"net/http"
	"vm-manager-backend/internal/core/labs"
	"vm-manager-backend/types"

	"github.com/gin-gonic/gin"
)

func ListTemplates(conn types.Conn) gin.HandlerFunc {
	return func(c *gin.Context) {
		templates, err := labs.ListLabTemplates(conn)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, templates)
	}
}

func GetTemplate(conn types.Conn) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		template, err := labs.GetLabTemplate(conn, id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, template)
	}
}

func CreateTemplate(conn types.Conn) gin.HandlerFunc {
	return func(c *gin.Context) {
		uuid, _ := c.Get("uuid")
		var tpl types.LabTemplate
		if err := c.ShouldBindJSON(&tpl); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid template payload: " + err.Error()})
			return
		}

		userUUID := ""
		if uuid != nil {
			userUUID = uuid.(string)
		}

		created, err := labs.CreateLabTemplate(conn, tpl, userUUID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, created)
	}
}

func UpdateTemplate(conn types.Conn) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var tpl types.LabTemplate
		if err := c.ShouldBindJSON(&tpl); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid template payload: " + err.Error()})
			return
		}

		err := labs.UpdateLabTemplate(conn, id, tpl)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "updated", "id": id})
	}
}

func DeleteTemplate(conn types.Conn) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		err := labs.DeleteLabTemplate(conn, id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "deleted", "id": id})
	}
}
