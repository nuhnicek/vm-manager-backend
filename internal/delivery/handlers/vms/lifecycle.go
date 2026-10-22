package vms

import (
	"fmt"
	"net/http"
	"vm-manager-backend/internal/core/vms"
	"vm-manager-backend/types"

	"github.com/gin-gonic/gin"
)

type vmActionBody struct {
	ID   int    `json:"id"`
	Node string `json:"node,omitempty"`
}

func Start(conn types.Conn) gin.HandlerFunc {
	return func(c *gin.Context) {
		uuid, _ := c.Get("uuid")
		roleVal, _ := c.Get("role")
		role := "student"
		if r, ok := roleVal.(string); ok && r != "" {
			role = r
		}
		var body vmActionBody
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}

		err := vms.StartVM(conn, uuid.(string), role, body.ID, body.Node)
		if err != nil {
			fmt.Printf("failed to start VM %d: %v\n", body.ID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "started", "id": body.ID})
	}
}

func Stop(conn types.Conn) gin.HandlerFunc {
	return func(c *gin.Context) {
		uuid, _ := c.Get("uuid")
		roleVal, _ := c.Get("role")
		role := "student"
		if r, ok := roleVal.(string); ok && r != "" {
			role = r
		}
		var body vmActionBody
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}

		err := vms.StopVM(conn, uuid.(string), role, body.ID, body.Node)
		if err != nil {
			fmt.Printf("failed to stop VM %d: %v\n", body.ID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "stopped", "id": body.ID})
	}
}

func Delete(conn types.Conn) gin.HandlerFunc {
	return func(c *gin.Context) {
		uuid, _ := c.Get("uuid")
		roleVal, _ := c.Get("role")
		role := "student"
		if r, ok := roleVal.(string); ok && r != "" {
			role = r
		}
		var body vmActionBody
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}

		err := vms.DeleteVM(conn, uuid.(string), role, body.ID, body.Node)
		if err != nil {
			fmt.Printf("failed to delete VM %d: %v\n", body.ID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "deleted", "id": body.ID})
	}
}
