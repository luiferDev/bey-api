package orders

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid/v5"
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
	router.Use(func(c *gin.Context) {
		c.Set("user_id", uuid.Must(uuid.NewV7()).String())
		c.Set("user_role", "admin")
		c.Next()
	})
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

	body := `{"shipping_address":"123 Main St"}`
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

	productID := uuid.Must(uuid.NewV7())
	body := `{"items":[{"product_id":"` + productID.String() + `","quantity":2}],"shipping_address":"123 Main St"}`
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

	testUUID := uuid.Must(uuid.NewV7())
	req, _ := http.NewRequest("GET", "/api/v1/orders/"+testUUID.String(), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestUpdateOrderStatus_InvalidBody(t *testing.T) {
	router, handler := setupTestRouterWithOrders(t)

	testUUID := uuid.Must(uuid.NewV7())
	router.PUT("/api/v1/orders/:id/status", handler.UpdateStatus)

	req, _ := http.NewRequest("PUT", "/api/v1/orders/"+testUUID.String()+"/status", bytes.NewBufferString("invalid"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 (order not found), got %d", w.Code)
	}
}

func TestUpdateOrderStatus_MissingStatus(t *testing.T) {
	router, handler := setupTestRouterWithOrders(t)

	testUUID := uuid.Must(uuid.NewV7())
	router.PUT("/api/v1/orders/:id/status", handler.UpdateStatus)

	body := `{}`
	req, _ := http.NewRequest("PUT", "/api/v1/orders/"+testUUID.String()+"/status", bytes.NewBufferString(body))
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

	testUUID := uuid.Must(uuid.NewV7())
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", testUUID.String())
		c.Set("user_role", "admin")
		c.Next()
	})
	router.POST("/api/v1/orders", handler.Create)
	router.POST("/api/v1/orders/:id/confirm", handler.Confirm)

	createBody := `{"items":[{"product_id":"` + testUUID.String() + `","quantity":2}],"shipping_address":"123 Main St"}`
	req1, _ := http.NewRequest("POST", "/api/v1/orders", bytes.NewBufferString(createBody))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	if w1.Code != http.StatusCreated {
		t.Fatalf("Failed to create order, got %d", w1.Code)
	}

	var createResp map[string]interface{}
	json.Unmarshal(w1.Body.Bytes(), &createResp)
	data := createResp["data"].(map[string]interface{})
	orderID := data["id"].(string)

	req2, _ := http.NewRequest("POST", "/api/v1/orders/"+orderID+"/confirm", nil)
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

	testUUID := uuid.Must(uuid.NewV7())
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", testUUID.String())
		c.Set("user_role", "admin")
		c.Next()
	})
	router.POST("/api/v1/orders", handler.Create)
	router.POST("/api/v1/orders/:id/confirm", handler.Confirm)

	createBody := `{"items":[{"product_id":"` + testUUID.String() + `","quantity":2}],"shipping_address":"123 Main St"}`
	req1, _ := http.NewRequest("POST", "/api/v1/orders", bytes.NewBufferString(createBody))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	if w1.Code != http.StatusCreated {
		t.Fatalf("Failed to create order, got %d", w1.Code)
	}

	var createResp map[string]interface{}
	json.Unmarshal(w1.Body.Bytes(), &createResp)
	data := createResp["data"].(map[string]interface{})
	orderID := data["id"].(string)

	req2, _ := http.NewRequest("POST", "/api/v1/orders/"+orderID+"/confirm", nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("First confirm failed, got %d", w2.Code)
	}

	req3, _ := http.NewRequest("POST", "/api/v1/orders/"+orderID+"/confirm", nil)
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

	ownerUUID := uuid.Must(uuid.NewV7())
	otherUUID := uuid.Must(uuid.NewV7())
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", ownerUUID.String())
		c.Set("user_role", "customer")
		c.Next()
	})
	router.POST("/api/v1/orders", handler.Create)

	createBody := `{"items":[{"product_id":"` + ownerUUID.String() + `","quantity":2}],"shipping_address":"123 Main St"}`
	req1, _ := http.NewRequest("POST", "/api/v1/orders", bytes.NewBufferString(createBody))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	if w1.Code != http.StatusCreated {
		t.Fatalf("Failed to create order, got %d", w1.Code)
	}

	var createResp map[string]interface{}
	json.Unmarshal(w1.Body.Bytes(), &createResp)
	data := createResp["data"].(map[string]interface{})
	orderID := data["id"].(string)

	confirmRouter := gin.New()
	confirmRouter.Use(func(c *gin.Context) {
		c.Set("user_id", otherUUID.String())
		c.Set("user_role", "customer")
		c.Next()
	})
	confirmRouter.POST("/api/v1/orders/:id/confirm", handler.Confirm)

	req2, _ := http.NewRequest("POST", "/api/v1/orders/"+orderID+"/confirm", nil)
	w2 := httptest.NewRecorder()
	confirmRouter.ServeHTTP(w2, req2)

	if w2.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d. Body: %s", w2.Code, w2.Body.String())
	}
}

func TestCancelOrder_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDBForOrdersHandler(t)
	handler := NewOrderHandler(db)

	testUUID := uuid.Must(uuid.NewV7())
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", testUUID.String())
		c.Set("user_role", "admin")
		c.Next()
	})
	router.POST("/api/v1/orders", handler.Create)
	router.POST("/api/v1/orders/:id/cancel", handler.Cancel)

	createBody := `{"items":[{"product_id":"` + testUUID.String() + `","quantity":2}],"shipping_address":"123 Main St"}`
	req1, _ := http.NewRequest("POST", "/api/v1/orders", bytes.NewBufferString(createBody))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	if w1.Code != http.StatusCreated {
		t.Fatalf("Failed to create order, got %d", w1.Code)
	}

	var createResp map[string]interface{}
	json.Unmarshal(w1.Body.Bytes(), &createResp)
	data := createResp["data"].(map[string]interface{})
	orderID := data["id"].(string)

	req2, _ := http.NewRequest("POST", "/api/v1/orders/"+orderID+"/cancel", nil)
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

	testUUID := uuid.Must(uuid.NewV7())
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", testUUID.String())
		c.Set("user_role", "admin")
		c.Next()
	})
	router.POST("/api/v1/orders", handler.Create)
	router.POST("/api/v1/orders/:id/cancel", handler.Cancel)

	createBody := `{"items":[{"product_id":"` + testUUID.String() + `","quantity":2}],"shipping_address":"123 Main St"}`
	req1, _ := http.NewRequest("POST", "/api/v1/orders", bytes.NewBufferString(createBody))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	if w1.Code != http.StatusCreated {
		t.Fatalf("Failed to create order, got %d", w1.Code)
	}

	var createResp map[string]interface{}
	json.Unmarshal(w1.Body.Bytes(), &createResp)
	data := createResp["data"].(map[string]interface{})
	orderID := data["id"].(string)

	req2, _ := http.NewRequest("POST", "/api/v1/orders/"+orderID+"/cancel", nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("First cancel failed, got %d", w2.Code)
	}

	req3, _ := http.NewRequest("POST", "/api/v1/orders/"+orderID+"/cancel", nil)
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

	ownerUUID := uuid.Must(uuid.NewV7())
	otherUUID := uuid.Must(uuid.NewV7())
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", ownerUUID.String())
		c.Set("user_role", "customer")
		c.Next()
	})
	router.POST("/api/v1/orders", handler.Create)

	createBody := `{"items":[{"product_id":"` + ownerUUID.String() + `","quantity":2}],"shipping_address":"123 Main St"}`
	req1, _ := http.NewRequest("POST", "/api/v1/orders", bytes.NewBufferString(createBody))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	if w1.Code != http.StatusCreated {
		t.Fatalf("Failed to create order, got %d", w1.Code)
	}

	var createResp map[string]interface{}
	json.Unmarshal(w1.Body.Bytes(), &createResp)
	data := createResp["data"].(map[string]interface{})
	orderID := data["id"].(string)

	cancelRouter := gin.New()
	cancelRouter.Use(func(c *gin.Context) {
		c.Set("user_id", otherUUID.String())
		c.Set("user_role", "customer")
		c.Next()
	})
	cancelRouter.POST("/api/v1/orders/:id/cancel", handler.Cancel)

	req2, _ := http.NewRequest("POST", "/api/v1/orders/"+orderID+"/cancel", nil)
	w2 := httptest.NewRecorder()
	cancelRouter.ServeHTTP(w2, req2)

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

	if w.Code != http.StatusNotFound && w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 404 or 400 for missing task ID, got %d", w.Code)
	}
}
