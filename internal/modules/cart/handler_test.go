package cart

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"bey/internal/modules/products"

	"github.com/gin-gonic/gin"
)

func setupCartTestRouterWithMocks(
	mockCartRepo *MockCartRepository,
	mockVariantFinder *MockVariantFinder,
) *gin.Engine {
	gin.SetMode(gin.TestMode)

	service := NewCartService(mockCartRepo, mockVariantFinder)
	handler := NewCartHandler(service)

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
