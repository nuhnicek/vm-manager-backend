package teacher

import (
	"net/http"
	"vm-manager-backend/types"

	"github.com/gin-gonic/gin"
)

// ListNodes returns real-time load telemetry for all standalone Proxmox nodes.
func ListNodes(conn types.Conn) gin.HandlerFunc {
	return func(c *gin.Context) {
		if conn.NodePool == nil {
			c.JSON(http.StatusOK, []types.NodeTelemetry{})
			return
		}
		telemetry := conn.NodePool.GetTelemetry(c.Request.Context())
		c.JSON(http.StatusOK, telemetry)
	}
}

// RegisterNode adds a new standalone Proxmox VE server to the load balancer pool.
func RegisterNode(conn types.Conn) gin.HandlerFunc {
	return func(c *gin.Context) {
		var node types.StandaloneNode
		if err := c.ShouldBindJSON(&node); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid node configuration: " + err.Error()})
			return
		}

		if node.HostURI == "" || node.Username == "" || node.Password == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "host_uri, username, and password are required"})
			return
		}

		if conn.NodePool == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "node pool is not initialized"})
			return
		}

		if err := conn.NodePool.RegisterNode(c.Request.Context(), node); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to register node: " + err.Error()})
			return
		}

		conn.NodePool.RefreshTelemetry(c.Request.Context())
		c.JSON(http.StatusCreated, gin.H{
			"message": "Node successfully registered in balancing pool",
			"node":    node,
		})
	}
}

// RemoveNode removes a standalone Proxmox server from the pool.
func RemoveNode(conn types.Conn) gin.HandlerFunc {
	return func(c *gin.Context) {
		nodeID := c.Param("id")
		if nodeID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "node id is required"})
			return
		}

		if conn.NodePool == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "node pool is not initialized"})
			return
		}

		if err := conn.NodePool.RemoveNode(c.Request.Context(), nodeID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to remove node: " + err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Node successfully removed from pool"})
	}
}

// RefreshNodes forces an immediate telemetry refresh on all standalone servers.
func RefreshNodes(conn types.Conn) gin.HandlerFunc {
	return func(c *gin.Context) {
		if conn.NodePool != nil {
			conn.NodePool.RefreshTelemetry(c.Request.Context())
		}
		telemetry := conn.NodePool.GetTelemetry(c.Request.Context())
		c.JSON(http.StatusOK, gin.H{
			"message":   "Telemetry refreshed",
			"telemetry": telemetry,
		})
	}
}
