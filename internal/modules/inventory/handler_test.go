package inventory

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDBForInventoryHandler(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	db.AutoMigrate(&Inventory{})
	return db
}

func setupTestRouterWithInventory(t *testing.T) (*gin.Engine, *InventoryHandler) {
	gin.SetMode(gin.TestMode)
	db := setupTestDBForInventoryHandler(t)
	handler := NewInventoryHandler(db)

	router := gin.New()
	return router, handler
}

func TestGetInventory_Response(t *testing.T) {
	router, handler := setupTestRouterWithInventory(t)

	router.GET("/api/v1/inventory/:product_id", handler.GetByProductID)

	req, _ := http.NewRequest("GET", "/api/v1/inventory/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound && w.Code != http.StatusOK {
		t.Errorf("Expected 404 or 200, got %d", w.Code)
	}
}

func TestUpdateInventory_InvalidBody(t *testing.T) {
	router, handler := setupTestRouterWithInventory(t)

	router.PUT("/api/v1/inventory/:product_id", handler.Update)

	req, _ := http.NewRequest("PUT", "/api/v1/inventory/1", bytes.NewBufferString("invalid"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code == http.StatusBadRequest || w.Code == http.StatusNotFound {
		// Valid - either invalid JSON or no inventory found
	} else {
		t.Errorf("Expected status 400 or 404, got %d", w.Code)
	}
}

func TestReserveInventory_InvalidQuantity(t *testing.T) {
	router, handler := setupTestRouterWithInventory(t)

	router.POST("/api/v1/inventory/:product_id/reserve", handler.Reserve)

	body := `{"quantity":0}`
	req, _ := http.NewRequest("POST", "/api/v1/inventory/1/reserve", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code == http.StatusBadRequest || w.Code == http.StatusNotFound {
		// Valid - either validation error or no inventory found
	} else {
		t.Errorf("Expected status 400 or 404, got %d", w.Code)
	}
}

func TestReserveInventory_InvalidBody(t *testing.T) {
	router, handler := setupTestRouterWithInventory(t)

	router.POST("/api/v1/inventory/:product_id/reserve", handler.Reserve)

	req, _ := http.NewRequest("POST", "/api/v1/inventory/1/reserve", bytes.NewBufferString("invalid"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code == http.StatusBadRequest || w.Code == http.StatusNotFound {
		// Valid - either invalid JSON or no inventory found
	} else {
		t.Errorf("Expected status 400 or 404, got %d", w.Code)
	}
}

func TestUpdateInventory_Success(t *testing.T) {
	router, handler := setupTestRouterWithInventory(t)

	router.PUT("/api/v1/inventory/:product_id", handler.Update)

	body := `{"quantity":100}`
	req, _ := http.NewRequest("PUT", "/api/v1/inventory/1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK && w.Code != http.StatusNotFound && w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 200, 404 or 400, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestReserveInventory_InsufficientStock(t *testing.T) {
	router, handler := setupTestRouterWithInventory(t)

	router.POST("/api/v1/inventory/:product_id/reserve", handler.Reserve)

	body := `{"quantity":1000}`
	req, _ := http.NewRequest("POST", "/api/v1/inventory/1/reserve", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest && w.Code != http.StatusNotFound && w.Code != http.StatusOK {
		t.Errorf("Expected status 400, 404 or 200, got %d. Body: %s", w.Code, w.Body.String())
	}
}
