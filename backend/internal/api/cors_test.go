package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCORSAllowsConfiguredOriginAndPreflight(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(CORSMiddleware([]string{"http://localhost:8000"}))
	router.POST("/api/v1/trips", func(c *gin.Context) { c.Status(http.StatusCreated) })

	request := httptest.NewRequest(http.MethodOptions, "/api/v1/trips", nil)
	request.Header.Set("Origin", "http://localhost:8000")
	request.Header.Set("Access-Control-Request-Method", "POST")
	request.Header.Set("Access-Control-Request-Headers", "Authorization, Content-Type")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent || response.Header().Get("Access-Control-Allow-Origin") != "http://localhost:8000" {
		t.Fatalf("unexpected CORS preflight: status=%d headers=%v", response.Code, response.Header())
	}
}

func TestCORSRejectsUnconfiguredOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(CORSMiddleware([]string{"http://localhost:8000"}))
	router.GET("/private", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	request := httptest.NewRequest(http.MethodGet, "/private", nil)
	request.Header.Set("Origin", "https://unknown.example")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("expected disallowed origin to receive 403, got %d", response.Code)
	}
}
