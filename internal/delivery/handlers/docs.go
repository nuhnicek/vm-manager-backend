package handlers

import (
	"net/http"
	"vm-manager-backend/pkg/docs"

	"github.com/gin-gonic/gin"
)

// ServeOpenAPIYAML returns the raw OpenAPI 3.0 specification in YAML format.
func ServeOpenAPIYAML() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Data(http.StatusOK, "application/yaml; charset=utf-8", docs.OpenAPIYAML)
	}
}

// ServeOpenAPIJSON returns the raw OpenAPI 3.0 specification in JSON format.
func ServeOpenAPIJSON() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Data(http.StatusOK, "application/json; charset=utf-8", docs.OpenAPIJSON)
	}
}

// ServeSwaggerUI serves an interactive Swagger UI documentation page.
func ServeSwaggerUI() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusOK, docs.SwaggerUIHTML)
	}
}
