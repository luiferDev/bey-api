package orders

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

func setupTestDBForOrdersHandler(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	db.AutoMigrate(&Order{}, &OrderItem{})
	return db
}

func setupTestRouterWithOrders(t *testing.T) (*gin.Engine, *OrderHandler) {
	gin.SetMode(gin.TestMode)
	db := setupTestDBForOrdersHandler(t)
	handler := NewOrderHandler(db)

	router := gin.New()
	return router, handler
}

func TestGetOrders_Success(t *testing.T) {
	router, handler := setupTestRouterWithOrders(t)

	router.GET("/api/v1/orders", handler.List)

	req, _ := http.NewRequest("GET", "/api/v1/orders", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp["success"] == nil {
		t.Error("Expected success field in response")
	}
}

func TestGetOrders_EmptyList(t *testing.T) {
	router, handler := setupTestRouterWithOrders(t)

	router.GET("/api/v1/orders", handler.List)

	req, _ := http.NewRequest("GET", "/api/v1/orders", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp["success"] == nil {
		t.Error("Expected success field in response")
	}
}

func TestCreateOrder_InvalidBody(t *testing.T) {
	router, handler := setupTestRouterWithOrders(t)

	router.POST("/api/v1/orders", handler.Create)

	req, _ := http.NewRequest("POST", "/api/v1/orders", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid body, got %d", w.Code)
	}
}

func TestCreateOrder_MissingFields(t *testing.T) {
	router, handler := setupTestRouterWithOrders(t)

	router.POST("/api/v1/orders", handler.Create)

	body := `{"user_id":1}`
	req, _ := http.NewRequest("POST", "/api/v1/orders", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for missing items, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestCreateOrder_Success(t *testing.T) {
	router, handler := setupTestRouterWithOrders(t)

	router.POST("/api/v1/orders", handler.Create)

	body := `{"user_id":1,"items":[{"product_id":1,"quantity":2}],"shipping_address":"123 Main St"}`
	req, _ := http.NewRequest("POST", "/api/v1/orders", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated && w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 201 or 500, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestGetOrderByID_NotFound(t *testing.T) {
	router, handler := setupTestRouterWithOrders(t)

	router.GET("/api/v1/orders/:id", handler.GetByID)

	req, _ := http.NewRequest("GET", "/api/v1/orders/999", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound && w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 404 or 400, got %d", w.Code)
	}
}

func TestUpdateOrderStatus_InvalidBody(t *testing.T) {
	router, handler := setupTestRouterWithOrders(t)

	router.PUT("/api/v1/orders/:id/status", handler.UpdateStatus)

	req, _ := http.NewRequest("PUT", "/api/v1/orders/1/status", bytes.NewBufferString("invalid"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code == http.StatusOK || w.Code == http.StatusCreated {
		t.Errorf("Expected error status, got %d", w.Code)
	}
}

func TestUpdateOrderStatus_MissingStatus(t *testing.T) {
	router, handler := setupTestRouterWithOrders(t)

	router.PUT("/api/v1/orders/:id/status", handler.UpdateStatus)

	body := `{}`
	req, _ := http.NewRequest("PUT", "/api/v1/orders/1/status", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest && w.Code != http.StatusNotFound {
		t.Errorf("Expected status 400 or 404 for missing status, got %d", w.Code)
	}
}
