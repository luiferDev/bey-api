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
	if err := db.AutoMigrate(&Order{}, &OrderItem{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}
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

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestGetOrderByID_NotFound(t *testing.T) {
	router, handler := setupTestRouterWithOrders(t)

	router.GET("/api/v1/orders/:id", handler.GetByID)

	req, _ := http.NewRequest("GET", "/api/v1/orders/999", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestUpdateOrderStatus_InvalidBody(t *testing.T) {
	router, handler := setupTestRouterWithOrders(t)

	router.POST("/api/v1/orders", handler.Create)
	router.PUT("/api/v1/orders/:id/status", handler.UpdateStatus)

	createBody := `{"user_id":1,"items":[{"product_id":1,"quantity":2}],"shipping_address":"123 Main St"}`
	req1, _ := http.NewRequest("POST", "/api/v1/orders", bytes.NewBufferString(createBody))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	if w1.Code != http.StatusCreated {
		t.Fatalf("Failed to create order, got %d", w1.Code)
	}

	req, _ := http.NewRequest("PUT", "/api/v1/orders/1/status", bytes.NewBufferString("invalid"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
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

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 for missing order, got %d", w.Code)
	}
}

func TestConfirmOrder_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDBForOrdersHandler(t)
	handler := NewOrderHandler(db)

	router := gin.New()
	router.POST("/api/v1/orders", handler.Create)
	router.Use(func(c *gin.Context) {
		c.Set("user_id", uint(1))
		c.Set("user_role", "admin")
		c.Next()
	})
	router.POST("/api/v1/orders/:id/confirm", handler.Confirm)

	createBody := `{"user_id":1,"items":[{"product_id":1,"quantity":2}],"shipping_address":"123 Main St"}`
	req1, _ := http.NewRequest("POST", "/api/v1/orders", bytes.NewBufferString(createBody))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	if w1.Code != http.StatusCreated {
		t.Fatalf("Failed to create order, got %d", w1.Code)
	}

	req2, _ := http.NewRequest("POST", "/api/v1/orders/1/confirm", nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w2.Code, w2.Body.String())
	}
}

func TestConfirmOrder_AlreadyConfirmed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDBForOrdersHandler(t)
	handler := NewOrderHandler(db)

	router := gin.New()
	router.POST("/api/v1/orders", handler.Create)
	router.Use(func(c *gin.Context) {
		c.Set("user_id", uint(1))
		c.Set("user_role", "admin")
		c.Next()
	})
	router.POST("/api/v1/orders/:id/confirm", handler.Confirm)

	createBody := `{"user_id":1,"items":[{"product_id":1,"quantity":2}],"shipping_address":"123 Main St"}`
	req1, _ := http.NewRequest("POST", "/api/v1/orders", bytes.NewBufferString(createBody))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	if w1.Code != http.StatusCreated {
		t.Fatalf("Failed to create order, got %d", w1.Code)
	}

	req2, _ := http.NewRequest("POST", "/api/v1/orders/1/confirm", nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("First confirm failed, got %d", w2.Code)
	}

	req3, _ := http.NewRequest("POST", "/api/v1/orders/1/confirm", nil)
	w3 := httptest.NewRecorder()
	router.ServeHTTP(w3, req3)

	if w3.Code != http.StatusOK {
		t.Errorf("Expected status 200 for already confirmed order, got %d. Body: %s", w3.Code, w3.Body.String())
	}
}

func TestConfirmOrder_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDBForOrdersHandler(t)
	handler := NewOrderHandler(db)

	router := gin.New()
	router.POST("/api/v1/orders", handler.Create)
	router.Use(func(c *gin.Context) {
		c.Set("user_id", uint(999))
		c.Set("user_role", "customer")
		c.Next()
	})
	router.POST("/api/v1/orders/:id/confirm", handler.Confirm)

	createBody := `{"user_id":1,"items":[{"product_id":1,"quantity":2}],"shipping_address":"123 Main St"}`
	req1, _ := http.NewRequest("POST", "/api/v1/orders", bytes.NewBufferString(createBody))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	if w1.Code != http.StatusCreated {
		t.Fatalf("Failed to create order, got %d", w1.Code)
	}

	req2, _ := http.NewRequest("POST", "/api/v1/orders/1/confirm", nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d. Body: %s", w2.Code, w2.Body.String())
	}
}

func TestCancelOrder_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDBForOrdersHandler(t)
	handler := NewOrderHandler(db)

	router := gin.New()
	router.POST("/api/v1/orders", handler.Create)
	router.Use(func(c *gin.Context) {
		c.Set("user_id", uint(1))
		c.Set("user_role", "admin")
		c.Next()
	})
	router.POST("/api/v1/orders/:id/cancel", handler.Cancel)

	createBody := `{"user_id":1,"items":[{"product_id":1,"quantity":2}],"shipping_address":"123 Main St"}`
	req1, _ := http.NewRequest("POST", "/api/v1/orders", bytes.NewBufferString(createBody))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	if w1.Code != http.StatusCreated {
		t.Fatalf("Failed to create order, got %d", w1.Code)
	}

	req2, _ := http.NewRequest("POST", "/api/v1/orders/1/cancel", nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w2.Code, w2.Body.String())
	}
}

func TestCancelOrder_AlreadyCancelled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDBForOrdersHandler(t)
	handler := NewOrderHandler(db)

	router := gin.New()
	router.POST("/api/v1/orders", handler.Create)
	router.Use(func(c *gin.Context) {
		c.Set("user_id", uint(1))
		c.Set("user_role", "admin")
		c.Next()
	})
	router.POST("/api/v1/orders/:id/cancel", handler.Cancel)

	createBody := `{"user_id":1,"items":[{"product_id":1,"quantity":2}],"shipping_address":"123 Main St"}`
	req1, _ := http.NewRequest("POST", "/api/v1/orders", bytes.NewBufferString(createBody))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	if w1.Code != http.StatusCreated {
		t.Fatalf("Failed to create order, got %d", w1.Code)
	}

	req2, _ := http.NewRequest("POST", "/api/v1/orders/1/cancel", nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("First cancel failed, got %d", w2.Code)
	}

	req3, _ := http.NewRequest("POST", "/api/v1/orders/1/cancel", nil)
	w3 := httptest.NewRecorder()
	router.ServeHTTP(w3, req3)

	if w3.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for already cancelled order, got %d. Body: %s", w3.Code, w3.Body.String())
	}
}

func TestCancelOrder_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDBForOrdersHandler(t)
	handler := NewOrderHandler(db)

	router := gin.New()
	router.POST("/api/v1/orders", handler.Create)
	router.Use(func(c *gin.Context) {
		c.Set("user_id", uint(999))
		c.Set("user_role", "customer")
		c.Next()
	})
	router.POST("/api/v1/orders/:id/cancel", handler.Cancel)

	createBody := `{"user_id":1,"items":[{"product_id":1,"quantity":2}],"shipping_address":"123 Main St"}`
	req1, _ := http.NewRequest("POST", "/api/v1/orders", bytes.NewBufferString(createBody))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	if w1.Code != http.StatusCreated {
		t.Fatalf("Failed to create order, got %d", w1.Code)
	}

	req2, _ := http.NewRequest("POST", "/api/v1/orders/1/cancel", nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d. Body: %s", w2.Code, w2.Body.String())
	}
}

func TestGetTaskStatus_NotConfigured(t *testing.T) {
	router, handler := setupTestRouterWithOrders(t)

	router.GET("/api/v1/orders/tasks/:task_id", handler.GetTaskStatus)

	req, _ := http.NewRequest("GET", "/api/v1/orders/tasks/task-123", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500 when task service not configured, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestGetTaskStatus_MissingTaskID(t *testing.T) {
	router, handler := setupTestRouterWithOrders(t)

	router.GET("/api/v1/orders/tasks/:task_id", handler.GetTaskStatus)

	req, _ := http.NewRequest("GET", "/api/v1/orders/tasks/", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Gin returns 404 when route param is empty (no match for the pattern)
	if w.Code != http.StatusNotFound && w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 404 or 400 for missing task ID, got %d", w.Code)
	}
}
