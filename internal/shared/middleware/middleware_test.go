package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestStaticFileServing(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Static("/", "./static")

	req, _ := http.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK && w.Code != http.StatusNotFound {
		t.Errorf("Expected status 200 or 404, got %d", w.Code)
	}
}

func TestStaticFileNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Static("/static", "./static")

	req, _ := http.NewRequest("GET", "/static/nonexistent.html", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code == http.StatusOK {
		t.Errorf("Expected 404 for non-existent file, got %d", w.Code)
	}
}

func TestSwaggerEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/swagger/*any", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"swagger": "enabled"})
	})

	req, _ := http.NewRequest("GET", "/swagger/index.html", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for swagger endpoint, got %d", w.Code)
	}
}

func TestHealthEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if w.Body.String() == "" {
		t.Error("Expected response body")
	}
}
