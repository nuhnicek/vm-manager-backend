package vms

import (
	"net/http"
	"vm-manager-backend/internal/core/vms"
	"vm-manager-backend/types"

	"github.com/gin-gonic/gin"
)

func ListCatalog(conn types.Conn) gin.HandlerFunc {
	return func(c *gin.Context) {
		items, err := vms.ListCatalog(conn)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, items)
	}
}

func DeployCatalog(conn types.Conn) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req types.DeployCatalogRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "neplatné tělo požadavku"})
			return
		}

		if req.CatalogID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "catalog_id je povinný parametr"})
			return
		}

		uuidVal, _ := c.Get("uuid")
		roleVal, _ := c.Get("role")
		uuidStr, _ := uuidVal.(string)
		roleStr, _ := roleVal.(string)

		machine, err := vms.DeployCatalogItem(conn, uuidStr, roleStr, req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, machine)
	}
}
