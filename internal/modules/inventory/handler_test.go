package inventory

import (
	"bytes"
	"encoding/json"
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
	if err := db.AutoMigrate(&Inventory{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}
	return db
}

func setupTestRouterWithInventory(t *testing.T) (*gin.Engine, *InventoryHandler) {
	gin.SetMode(gin.TestMode)
	db := setupTestDBForInventoryHandler(t)
	handler := NewInventoryHandler(db)

	router := gin.New()
	return router, handler
}

func TestGetInventory_NotFound(t *testing.T) {
	router, handler := setupTestRouterWithInventory(t)

	router.GET("/api/v1/inventory/:product_id", handler.GetByProductID)

	req, _ := http.NewRequest("GET", "/api/v1/inventory/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestGetInventory_InvalidProductID(t *testing.T) {
	router, handler := setupTestRouterWithInventory(t)

	router.GET("/api/v1/inventory/:product_id", handler.GetByProductID)

	req, _ := http.NewRequest("GET", "/api/v1/inventory/abc", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid product ID, got %d", w.Code)
	}
}

func TestUpdateInventory_InvalidBody(t *testing.T) {
	router, handler := setupTestRouterWithInventory(t)

	router.PUT("/api/v1/inventory/:product_id", handler.Update)

	req, _ := http.NewRequest("PUT", "/api/v1/inventory/1", bytes.NewBufferString("invalid"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
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

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected data field in response")
	}

	if int(data["quantity"].(float64)) != 100 {
		t.Errorf("Expected quantity 100, got %v", data["quantity"])
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

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestReserveInventory_InvalidBody(t *testing.T) {
	router, handler := setupTestRouterWithInventory(t)

	router.POST("/api/v1/inventory/:product_id/reserve", handler.Reserve)

	req, _ := http.NewRequest("POST", "/api/v1/inventory/1/reserve", bytes.NewBufferString("invalid"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
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

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for insufficient stock, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestReserveInventory_Success(t *testing.T) {
	router, handler := setupTestRouterWithInventory(t)

	router.PUT("/api/v1/inventory/:product_id", handler.Update)
	router.POST("/api/v1/inventory/:product_id/reserve", handler.Reserve)

	updateBody := `{"quantity":100}`
	req1, _ := http.NewRequest("PUT", "/api/v1/inventory/1", bytes.NewBufferString(updateBody))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	if w1.Code != http.StatusOK {
		t.Fatalf("Failed to set up inventory, got %d", w1.Code)
	}

	reserveBody := `{"quantity":10}`
	req2, _ := http.NewRequest("POST", "/api/v1/inventory/1/reserve", bytes.NewBufferString(reserveBody))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w2.Code, w2.Body.String())
	}
}

func TestReleaseInventory_Success(t *testing.T) {
	router, handler := setupTestRouterWithInventory(t)

	router.PUT("/api/v1/inventory/:product_id", handler.Update)
	router.POST("/api/v1/inventory/:product_id/reserve", handler.Reserve)
	router.POST("/api/v1/inventory/:product_id/release", handler.Release)

	updateBody := `{"quantity":100}`
	req1, _ := http.NewRequest("PUT", "/api/v1/inventory/1", bytes.NewBufferString(updateBody))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	if w1.Code != http.StatusOK {
		t.Fatalf("Failed to set up inventory, got %d", w1.Code)
	}

	reserveBody := `{"quantity":20}`
	req2, _ := http.NewRequest("POST", "/api/v1/inventory/1/reserve", bytes.NewBufferString(reserveBody))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("Failed to reserve inventory, got %d", w2.Code)
	}

	releaseBody := `{"quantity":10}`
	req3, _ := http.NewRequest("POST", "/api/v1/inventory/1/release", bytes.NewBufferString(releaseBody))
	req3.Header.Set("Content-Type", "application/json")
	w3 := httptest.NewRecorder()
	router.ServeHTTP(w3, req3)

	if w3.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w3.Code, w3.Body.String())
	}
}

func TestReleaseInventory_NotEnoughReserved(t *testing.T) {
	router, handler := setupTestRouterWithInventory(t)

	router.PUT("/api/v1/inventory/:product_id", handler.Update)
	router.POST("/api/v1/inventory/:product_id/reserve", handler.Reserve)
	router.POST("/api/v1/inventory/:product_id/release", handler.Release)

	updateBody := `{"quantity":100}`
	req1, _ := http.NewRequest("PUT", "/api/v1/inventory/1", bytes.NewBufferString(updateBody))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	if w1.Code != http.StatusOK {
		t.Fatalf("Failed to set up inventory, got %d", w1.Code)
	}

	releaseBody := `{"quantity":50}`
	req2, _ := http.NewRequest("POST", "/api/v1/inventory/1/release", bytes.NewBufferString(releaseBody))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for not enough reserved, got %d. Body: %s", w2.Code, w2.Body.String())
	}
}

func TestReleaseInventory_NotFound(t *testing.T) {
	router, handler := setupTestRouterWithInventory(t)

	router.POST("/api/v1/inventory/:product_id/release", handler.Release)

	body := `{"quantity":10}`
	req, _ := http.NewRequest("POST", "/api/v1/inventory/999/release", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d. Body: %s", w.Code, w.Body.String())
	}
}
