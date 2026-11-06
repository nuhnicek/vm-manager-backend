package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"vm-manager-backend/internal/delivery/handlers"

	"github.com/gin-gonic/gin"
)

func TestDocsEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.GET("/docs", handlers.ServeSwaggerUI())
	router.GET("/openapi.yaml", handlers.ServeOpenAPIYAML())
	router.GET("/openapi.json", handlers.ServeOpenAPIJSON())

	// Test /docs
	{
		req, _ := http.NewRequest(http.MethodGet, "/docs", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 for /docs, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "swagger-ui") {
			t.Fatalf("expected swagger-ui in body")
		}
	}

	// Test /openapi.yaml
	{
		req, _ := http.NewRequest(http.MethodGet, "/openapi.yaml", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 for /openapi.yaml, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "openapi: 3.0.3") {
			t.Fatalf("expected openapi: 3.0.3 in yaml body")
		}
	}

	// Test /openapi.json
	{
		req, _ := http.NewRequest(http.MethodGet, "/openapi.json", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 for /openapi.json, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), `"openapi": "3.0.3"`) {
			t.Fatalf("expected openapi: 3.0.3 in json body")
		}
	}
}
