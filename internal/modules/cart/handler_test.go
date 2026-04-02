package cart

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"bey/internal/modules/orders"
	"bey/internal/modules/products"

	"github.com/gin-gonic/gin"
)

// MockOrderCreator implements OrderCreator interface
type MockOrderCreator struct {
	createFunc func(order *orders.Order) error
}

func (m *MockOrderCreator) Create(order *orders.Order) error {
	if m.createFunc != nil {
		return m.createFunc(order)
	}
	// Default: assign an ID to simulate successful creation
	order.ID = 1
	return nil
}

// MockVariantStockReserver implements VariantStockReserver interface
type MockVariantStockReserver struct {
	reserveStockFunc func(id uint, quantity int) error
}

func (m *MockVariantStockReserver) ReserveStock(id uint, quantity int) error {
	if m.reserveStockFunc != nil {
		return m.reserveStockFunc(id, quantity)
	}
	return nil
}

// MockInventoryReserver implements InventoryReserver interface
type MockInventoryReserver struct {
	reserveFunc func(productID uint, quantity int) error
}

func (m *MockInventoryReserver) Reserve(productID uint, quantity int) error {
	if m.reserveFunc != nil {
		return m.reserveFunc(productID, quantity)
	}
	return nil
}

func setupCartTestRouterWithMocks(
	mockCartRepo *MockCartRepository,
	mockVariantFinder *MockVariantFinder,
) *gin.Engine {
	return setupCartTestRouterWithMocksAndOrder(mockCartRepo, mockVariantFinder, &MockOrderCreator{}, &MockVariantStockReserver{}, &MockInventoryReserver{})
}

func setupCartTestRouterWithMocksAndOrder(
	mockCartRepo *MockCartRepository,
	mockVariantFinder *MockVariantFinder,
	mockOrderRepo *MockOrderCreator,
	mockVariantStock *MockVariantStockReserver,
	mockInventory *MockInventoryReserver,
) *gin.Engine {
	gin.SetMode(gin.TestMode)

	service := NewCartService(mockCartRepo, mockVariantFinder)
	handler := NewCartHandler(service, mockOrderRepo, mockVariantStock, mockInventory)

	router := gin.New()

	router.Use(func(c *gin.Context) {
		if userID := c.GetHeader("X-User-ID"); userID != "" {
			var uid uint
			json.Unmarshal([]byte(userID), &uid)
			c.Set("user_id", uid)
		}
		c.Next()
	})

	cartRoutes := router.Group("/api/v1/cart")
	cartRoutes.GET("", handler.GetCart)
	cartRoutes.POST("/items", handler.AddItem)
	cartRoutes.PUT("/items/:variant_id", handler.UpdateItem)
	cartRoutes.DELETE("/items/:variant_id", handler.RemoveItem)
	cartRoutes.DELETE("", handler.ClearCart)
	cartRoutes.POST("/checkout", handler.Checkout)

	return router
}

func makeAuthReq(router *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(method, path, bytes.NewBufferString(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("X-User-ID", "1")
	router.ServeHTTP(w, req)
	return w
}

func TestCartHandler_GetCart_Unauthorized(t *testing.T) {
	mockCartRepo := &MockCartRepository{}
	mockVariantFinder := &MockVariantFinder{}
	router := setupCartTestRouterWithMocks(mockCartRepo, mockVariantFinder)

	req, _ := http.NewRequest("GET", "/api/v1/cart", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestCartHandler_GetCart_Success(t *testing.T) {
	mockCartRepo := &MockCartRepository{
		getCartFunc: func(userID uint) (*Cart, error) {
			return &Cart{
				UserID: userID,
				Items:  []CartItem{{VariantID: 1, Quantity: 2}},
			}, nil
		},
	}
	mockVariantFinder := &MockVariantFinder{}
	router := setupCartTestRouterWithMocks(mockCartRepo, mockVariantFinder)

	w := makeAuthReq(router, "GET", "/api/v1/cart", "")

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestCartHandler_GetCart_Error(t *testing.T) {
	mockCartRepo := &MockCartRepository{
		getCartFunc: func(userID uint) (*Cart, error) {
			return nil, errors.New("database error")
		},
	}
	mockVariantFinder := &MockVariantFinder{}
	router := setupCartTestRouterWithMocks(mockCartRepo, mockVariantFinder)

	w := makeAuthReq(router, "GET", "/api/v1/cart", "")

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}

func TestCartHandler_AddItem_Unauthorized(t *testing.T) {
	mockCartRepo := &MockCartRepository{}
	mockVariantFinder := &MockVariantFinder{}
	router := setupCartTestRouterWithMocks(mockCartRepo, mockVariantFinder)

	req, _ := http.NewRequest("POST", "/api/v1/cart/items", bytes.NewBufferString(`{"variant_id":1,"quantity":2}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestCartHandler_AddItem_InvalidBody(t *testing.T) {
	mockCartRepo := &MockCartRepository{}
	mockVariantFinder := &MockVariantFinder{}
	router := setupCartTestRouterWithMocks(mockCartRepo, mockVariantFinder)

	req, _ := http.NewRequest("POST", "/api/v1/cart/items", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "1")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestCartHandler_AddItem_Success(t *testing.T) {
	mockCartRepo := &MockCartRepository{
		getCartFunc: func(userID uint) (*Cart, error) {
			return &Cart{UserID: userID, Items: []CartItem{}}, nil
		},
		saveCartFunc: func(cart *Cart) error {
			return nil
		},
	}
	mockVariantFinder := &MockVariantFinder{
		findByIDFunc: func(id uint) (*products.ProductVariant, error) {
			return &products.ProductVariant{ID: id, ProductID: 1}, nil
		},
		getPriceAndStockFunc: func(id uint) (float64, int, int, error) {
			return 10.0, 100, 0, nil
		},
	}
	router := setupCartTestRouterWithMocks(mockCartRepo, mockVariantFinder)

	w := makeAuthReq(router, "POST", "/api/v1/cart/items", `{"variant_id":1,"quantity":2}`)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["success"] == false {
		t.Errorf("Expected success=true, got %v", resp["success"])
	}
}

func TestCartHandler_AddItem_InsufficientStock(t *testing.T) {
	mockCartRepo := &MockCartRepository{
		getCartFunc: func(userID uint) (*Cart, error) {
			return &Cart{UserID: userID, Items: []CartItem{}}, nil
		},
	}
	mockVariantFinder := &MockVariantFinder{
		findByIDFunc: func(id uint) (*products.ProductVariant, error) {
			return &products.ProductVariant{ID: id, ProductID: 1}, nil
		},
		getPriceAndStockFunc: func(id uint) (float64, int, int, error) {
			return 10.0, 5, 0, nil
		},
	}
	router := setupCartTestRouterWithMocks(mockCartRepo, mockVariantFinder)

	w := makeAuthReq(router, "POST", "/api/v1/cart/items", `{"variant_id":1,"quantity":10}`)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestCartHandler_AddItem_VariantNotFound(t *testing.T) {
	mockCartRepo := &MockCartRepository{}
	mockVariantFinder := &MockVariantFinder{
		findByIDFunc: func(id uint) (*products.ProductVariant, error) {
			return nil, nil
		},
	}
	router := setupCartTestRouterWithMocks(mockCartRepo, mockVariantFinder)

	w := makeAuthReq(router, "POST", "/api/v1/cart/items", `{"variant_id":999,"quantity":2}`)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestCartHandler_UpdateItem_Unauthorized(t *testing.T) {
	mockCartRepo := &MockCartRepository{}
	mockVariantFinder := &MockVariantFinder{}
	router := setupCartTestRouterWithMocks(mockCartRepo, mockVariantFinder)

	req, _ := http.NewRequest("PUT", "/api/v1/cart/items/1", bytes.NewBufferString(`{"quantity":5}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestCartHandler_UpdateItem_InvalidQuantity(t *testing.T) {
	mockCartRepo := &MockCartRepository{}
	mockVariantFinder := &MockVariantFinder{}
	router := setupCartTestRouterWithMocks(mockCartRepo, mockVariantFinder)

	req, _ := http.NewRequest("PUT", "/api/v1/cart/items/1", bytes.NewBufferString(`{"quantity":-1}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "1")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestCartHandler_UpdateItem_InvalidVariantID(t *testing.T) {
	mockCartRepo := &MockCartRepository{}
	mockVariantFinder := &MockVariantFinder{}
	router := setupCartTestRouterWithMocks(mockCartRepo, mockVariantFinder)

	req, _ := http.NewRequest("PUT", "/api/v1/cart/items/invalid", bytes.NewBufferString(`{"quantity":5}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "1")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestCartHandler_UpdateItem_Success(t *testing.T) {
	mockCartRepo := &MockCartRepository{
		getCartFunc: func(userID uint) (*Cart, error) {
			return &Cart{
				UserID: userID,
				Items:  []CartItem{{VariantID: 1, Quantity: 2}},
			}, nil
		},
		saveCartFunc: func(cart *Cart) error {
			return nil
		},
	}
	mockVariantFinder := &MockVariantFinder{
		getPriceAndStockFunc: func(id uint) (float64, int, int, error) {
			return 10.0, 100, 0, nil
		},
	}
	router := setupCartTestRouterWithMocks(mockCartRepo, mockVariantFinder)

	w := makeAuthReq(router, "PUT", "/api/v1/cart/items/1", `{"quantity":5}`)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestCartHandler_UpdateItem_InsufficientStock(t *testing.T) {
	mockCartRepo := &MockCartRepository{
		getCartFunc: func(userID uint) (*Cart, error) {
			return &Cart{
				UserID: userID,
				Items:  []CartItem{{VariantID: 1, Quantity: 2}},
			}, nil
		},
	}
	mockVariantFinder := &MockVariantFinder{
		getPriceAndStockFunc: func(id uint) (float64, int, int, error) {
			return 10.0, 5, 0, nil
		},
	}
	router := setupCartTestRouterWithMocks(mockCartRepo, mockVariantFinder)

	w := makeAuthReq(router, "PUT", "/api/v1/cart/items/1", `{"quantity":10}`)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestCartHandler_RemoveItem_Unauthorized(t *testing.T) {
	mockCartRepo := &MockCartRepository{}
	mockVariantFinder := &MockVariantFinder{}
	router := setupCartTestRouterWithMocks(mockCartRepo, mockVariantFinder)

	req, _ := http.NewRequest("DELETE", "/api/v1/cart/items/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestCartHandler_RemoveItem_InvalidVariantID(t *testing.T) {
	mockCartRepo := &MockCartRepository{}
	mockVariantFinder := &MockVariantFinder{}
	router := setupCartTestRouterWithMocks(mockCartRepo, mockVariantFinder)

	req, _ := http.NewRequest("DELETE", "/api/v1/cart/items/invalid", nil)
	req.Header.Set("X-User-ID", "1")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestCartHandler_RemoveItem_Success(t *testing.T) {
	mockCartRepo := &MockCartRepository{
		getCartFunc: func(userID uint) (*Cart, error) {
			return &Cart{
				UserID: userID,
				Items:  []CartItem{{VariantID: 1, Quantity: 2}},
			}, nil
		},
		saveCartFunc: func(cart *Cart) error {
			return nil
		},
	}
	mockVariantFinder := &MockVariantFinder{}
	router := setupCartTestRouterWithMocks(mockCartRepo, mockVariantFinder)

	w := makeAuthReq(router, "DELETE", "/api/v1/cart/items/1", "")

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestCartHandler_RemoveItem_Error(t *testing.T) {
	mockCartRepo := &MockCartRepository{
		getCartFunc: func(userID uint) (*Cart, error) {
			return nil, errors.New("database error")
		},
	}
	mockVariantFinder := &MockVariantFinder{}
	router := setupCartTestRouterWithMocks(mockCartRepo, mockVariantFinder)

	w := makeAuthReq(router, "DELETE", "/api/v1/cart/items/1", "")

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}

func TestCartHandler_ClearCart_Unauthorized(t *testing.T) {
	mockCartRepo := &MockCartRepository{}
	mockVariantFinder := &MockVariantFinder{}
	router := setupCartTestRouterWithMocks(mockCartRepo, mockVariantFinder)

	req, _ := http.NewRequest("DELETE", "/api/v1/cart", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestCartHandler_ClearCart_Success(t *testing.T) {
	mockCartRepo := &MockCartRepository{
		deleteCartFunc: func(userID uint) error {
			return nil
		},
	}
	mockVariantFinder := &MockVariantFinder{}
	router := setupCartTestRouterWithMocks(mockCartRepo, mockVariantFinder)

	w := makeAuthReq(router, "DELETE", "/api/v1/cart", "")

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestCartHandler_ClearCart_Error(t *testing.T) {
	mockCartRepo := &MockCartRepository{
		deleteCartFunc: func(userID uint) error {
			return errors.New("database error")
		},
	}
	mockVariantFinder := &MockVariantFinder{}
	router := setupCartTestRouterWithMocks(mockCartRepo, mockVariantFinder)

	w := makeAuthReq(router, "DELETE", "/api/v1/cart", "")

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}

func TestCartHandler_Checkout_Unauthorized(t *testing.T) {
	mockCartRepo := &MockCartRepository{}
	mockVariantFinder := &MockVariantFinder{}
	mockOrderRepo := &MockOrderCreator{}
	router := setupCartTestRouterWithMocksAndOrder(mockCartRepo, mockVariantFinder, mockOrderRepo, &MockVariantStockReserver{}, &MockInventoryReserver{})

	req, _ := http.NewRequest("POST", "/api/v1/cart/checkout", bytes.NewBufferString(`{"shipping_address":"123 Main St"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestCartHandler_Checkout_EmptyCart(t *testing.T) {
	mockCartRepo := &MockCartRepository{
		getCartFunc: func(userID uint) (*Cart, error) {
			return &Cart{UserID: userID, Items: []CartItem{}}, nil
		},
	}
	mockVariantFinder := &MockVariantFinder{}
	mockOrderRepo := &MockOrderCreator{}
	router := setupCartTestRouterWithMocksAndOrder(mockCartRepo, mockVariantFinder, mockOrderRepo, &MockVariantStockReserver{}, &MockInventoryReserver{})

	w := makeAuthReq(router, "POST", "/api/v1/cart/checkout", `{"shipping_address":"123 Main St"}`)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestCartHandler_Checkout_Success(t *testing.T) {
	mockCartRepo := &MockCartRepository{
		getCartFunc: func(userID uint) (*Cart, error) {
			return &Cart{
				UserID: userID,
				Items:  []CartItem{{VariantID: 1, Quantity: 2}},
			}, nil
		},
		deleteCartFunc: func(userID uint) error {
			return nil
		},
	}
	mockVariantFinder := &MockVariantFinder{
		findByIDFunc: func(id uint) (*products.ProductVariant, error) {
			return &products.ProductVariant{ID: id, ProductID: 10, SKU: "TEST-SKU", Price: 50.00, Stock: 100}, nil
		},
		getPriceAndStockFunc: func(id uint) (float64, int, int, error) {
			return 50.00, 100, 0, nil
		},
	}
	mockOrderRepo := &MockOrderCreator{
		createFunc: func(order *orders.Order) error {
			order.ID = 1
			return nil
		},
	}
	router := setupCartTestRouterWithMocksAndOrder(mockCartRepo, mockVariantFinder, mockOrderRepo, &MockVariantStockReserver{}, &MockInventoryReserver{})

	w := makeAuthReq(router, "POST", "/api/v1/cart/checkout", `{"shipping_address":"123 Main St","notes":"Test order"}`)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d. Body: %s", w.Code, w.Body.String())
		return
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected data field in response")
	}

	if data["order_id"] == nil || int(data["order_id"].(float64)) != 1 {
		t.Errorf("Expected order_id 1, got %v", data["order_id"])
	}

	if data["cart_cleared"] != true {
		t.Error("Expected cart_cleared to be true")
	}

	if data["total_price"] != 100.0 {
		t.Errorf("Expected total_price 100.0, got %v", data["total_price"])
	}

	items, ok := data["items"].([]interface{})
	if !ok || len(items) != 1 {
		t.Fatalf("Expected 1 item in response, got %v", data["items"])
	}

	item := items[0].(map[string]interface{})
	if int(item["quantity"].(float64)) != 2 {
		t.Errorf("Expected quantity 2, got %v", item["quantity"])
	}

	if item["unit_price"] != 50.0 {
		t.Errorf("Expected unit_price 50.0, got %v", item["unit_price"])
	}
}
