package vms

import (
	"fmt"
	"net/http"
	"vm-manager-backend/internal/core/vms"
	"vm-manager-backend/types"

	"github.com/gin-gonic/gin"
)

type vmActionBody struct {
	ID int `json:"id"`
}

func Start(conn types.Conn) gin.HandlerFunc {
	return func(c *gin.Context) {
		uuid, _ := c.Get("uuid")
		var body vmActionBody
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}

		err := vms.StartVM(conn, uuid.(string), body.ID)
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
		var body vmActionBody
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}

		err := vms.StopVM(conn, uuid.(string), body.ID)
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
		var body vmActionBody
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}

		err := vms.DeleteVM(conn, uuid.(string), body.ID)
		if err != nil {
			fmt.Printf("failed to delete VM %d: %v\n", body.ID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "deleted", "id": body.ID})
	}
}
