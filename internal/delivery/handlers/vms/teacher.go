package vms

import (
	"net/http"
	"vm-manager-backend/internal/core/vms"
	"vm-manager-backend/types"

	"github.com/gin-gonic/gin"
)

func TeacherStudents(conn types.Conn) gin.HandlerFunc {
	return func(c *gin.Context) {
		students, err := vms.ListStudents(conn)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, students)
	}
}

func RefreshNodes(conn types.Conn) gin.HandlerFunc {
	return func(c *gin.Context) {
		nodes := vms.ListNodes(conn)
		c.JSON(http.StatusOK, gin.H{
			"status": "success",
			"nodes":  nodes,
		})
	}
}
