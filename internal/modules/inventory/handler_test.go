package inventory

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

func setupTestDBForInventoryHandler(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	if err := db.AutoMigrate(&Inventory{}, &ProductVariant{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}
	return db
}

func setupTestRouterWithInventory(t *testing.T) (*gin.Engine, *InventoryHandler) {
	gin.SetMode(gin.TestMode)
	db := setupTestDBForInventoryHandler(t)
	handler := NewInventoryHandler(db)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_role", "admin")
		c.Set("user_id", uuid.Must(uuid.NewV7()).String())
		c.Next()
	})
	return router, handler
}

func TestGetInventory_NoVariants(t *testing.T) {
	router, handler := setupTestRouterWithInventory(t)

	testUUID := uuid.Must(uuid.NewV7())
	router.GET("/api/v1/inventory/:product_id", handler.GetByProductID)

	req, _ := http.NewRequest("GET", "/api/v1/inventory/"+testUUID.String(), nil)
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

	if int(data["total_stock"].(float64)) != 0 {
		t.Errorf("Expected total_stock 0, got %v", data["total_stock"])
	}
}

func TestGetInventory_WithVariants(t *testing.T) {
	router, handler := setupTestRouterWithInventory(t)
	db := handler.repo.db

	testUUID := uuid.Must(uuid.NewV7())
	variant1 := ProductVariant{
		ID:        uuid.Must(uuid.NewV7()),
		ProductID: testUUID,
		SKU:       "SKU-001",
		Price:     100.00,
		Stock:     50,
		Reserved:  10,
	}
	variant2 := ProductVariant{
		ID:        uuid.Must(uuid.NewV7()),
		ProductID: testUUID,
		SKU:       "SKU-002",
		Price:     150.00,
		Stock:     30,
		Reserved:  5,
	}
	db.Create(&variant1)
	db.Create(&variant2)

	router.GET("/api/v1/inventory/:product_id", handler.GetByProductID)

	req, _ := http.NewRequest("GET", "/api/v1/inventory/"+testUUID.String(), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected data field in response")
	}

	if int(data["total_stock"].(float64)) != 80 {
		t.Errorf("Expected total_stock 80, got %v", data["total_stock"])
	}
	if int(data["total_reserved"].(float64)) != 15 {
		t.Errorf("Expected total_reserved 15, got %v", data["total_reserved"])
	}
	if int(data["total_available"].(float64)) != 65 {
		t.Errorf("Expected total_available 65, got %v", data["total_available"])
	}

	variants, ok := data["variants"].([]interface{})
	if !ok {
		t.Fatal("Expected variants array in response")
	}
	if len(variants) != 2 {
		t.Errorf("Expected 2 variants, got %d", len(variants))
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

	testUUID := uuid.Must(uuid.NewV7())
	router.PUT("/api/v1/inventory/:product_id", handler.Update)

	req, _ := http.NewRequest("PUT", "/api/v1/inventory/"+testUUID.String(), bytes.NewBufferString("invalid"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestUpdateInventory_MissingVariantID(t *testing.T) {
	router, handler := setupTestRouterWithInventory(t)

	testUUID := uuid.Must(uuid.NewV7())
	router.PUT("/api/v1/inventory/:product_id", handler.Update)

	body := `{"quantity":100}`
	req, _ := http.NewRequest("PUT", "/api/v1/inventory/"+testUUID.String(), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for missing variant_id, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestUpdateInventory_VariantNotFound(t *testing.T) {
	router, handler := setupTestRouterWithInventory(t)

	testUUID := uuid.Must(uuid.NewV7())
	variantUUID := uuid.Must(uuid.NewV7())
	router.PUT("/api/v1/inventory/:product_id", handler.Update)

	body := `{"quantity":100, "variant_id":"` + variantUUID.String() + `"}`
	req, _ := http.NewRequest("PUT", "/api/v1/inventory/"+testUUID.String(), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestUpdateInventory_Success(t *testing.T) {
	router, handler := setupTestRouterWithInventory(t)
	db := handler.repo.db

	testUUID := uuid.Must(uuid.NewV7())
	variantUUID := uuid.Must(uuid.NewV7())
	variant := ProductVariant{
		ID:        variantUUID,
		ProductID: testUUID,
		SKU:       "SKU-001",
		Price:     100.00,
		Stock:     50,
		Reserved:  0,
	}
	db.Create(&variant)

	router.PUT("/api/v1/inventory/:product_id", handler.Update)

	body := `{"quantity":200, "variant_id":"` + variantUUID.String() + `"}`
	req, _ := http.NewRequest("PUT", "/api/v1/inventory/"+testUUID.String(), bytes.NewBufferString(body))
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

	if int(data["total_stock"].(float64)) != 200 {
		t.Errorf("Expected total_stock 200, got %v", data["total_stock"])
	}
}

func TestReserveInventory_InvalidQuantity(t *testing.T) {
	router, handler := setupTestRouterWithInventory(t)

	testUUID := uuid.Must(uuid.NewV7())
	router.POST("/api/v1/inventory/:product_id/reserve", handler.Reserve)

	body := `{"quantity":0}`
	req, _ := http.NewRequest("POST", "/api/v1/inventory/"+testUUID.String()+"/reserve", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestReserveInventory_InvalidBody(t *testing.T) {
	router, handler := setupTestRouterWithInventory(t)

	testUUID := uuid.Must(uuid.NewV7())
	router.POST("/api/v1/inventory/:product_id/reserve", handler.Reserve)

	req, _ := http.NewRequest("POST", "/api/v1/inventory/"+testUUID.String()+"/reserve", bytes.NewBufferString("invalid"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestReserveInventory_NoStock(t *testing.T) {
	router, handler := setupTestRouterWithInventory(t)

	testUUID := uuid.Must(uuid.NewV7())
	router.POST("/api/v1/inventory/:product_id/reserve", handler.Reserve)

	body := `{"quantity":10}`
	req, _ := http.NewRequest("POST", "/api/v1/inventory/"+testUUID.String()+"/reserve", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for insufficient stock, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestReserveInventory_VariantSuccess(t *testing.T) {
	router, handler := setupTestRouterWithInventory(t)
	db := handler.repo.db

	testUUID := uuid.Must(uuid.NewV7())
	variantUUID := uuid.Must(uuid.NewV7())
	variant := ProductVariant{
		ID:        variantUUID,
		ProductID: testUUID,
		SKU:       "SKU-001",
		Price:     100.00,
		Stock:     100,
		Reserved:  0,
	}
	db.Create(&variant)

	router.POST("/api/v1/inventory/:product_id/reserve", handler.Reserve)

	body := `{"quantity":10, "variant_id":"` + variantUUID.String() + `"}`
	req, _ := http.NewRequest("POST", "/api/v1/inventory/"+testUUID.String()+"/reserve", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var v ProductVariant
	db.First(&v, "id = ?", variantUUID)
	if v.Stock != 90 {
		t.Errorf("Expected stock 90, got %d", v.Stock)
	}
	if v.Reserved != 10 {
		t.Errorf("Expected reserved 10, got %d", v.Reserved)
	}
}

func TestReserveInventory_ProductSuccess(t *testing.T) {
	router, handler := setupTestRouterWithInventory(t)
	db := handler.repo.db

	testUUID := uuid.Must(uuid.NewV7())
	variantUUID := uuid.Must(uuid.NewV7())
	variant := ProductVariant{
		ID:        variantUUID,
		ProductID: testUUID,
		SKU:       "SKU-001",
		Price:     100.00,
		Stock:     100,
		Reserved:  0,
	}
	db.Create(&variant)

	router.POST("/api/v1/inventory/:product_id/reserve", handler.Reserve)

	body := `{"quantity":25}`
	req, _ := http.NewRequest("POST", "/api/v1/inventory/"+testUUID.String()+"/reserve", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var v ProductVariant
	db.First(&v, "id = ?", variantUUID)
	if v.Stock != 75 {
		t.Errorf("Expected stock 75, got %d", v.Stock)
	}
	if v.Reserved != 25 {
		t.Errorf("Expected reserved 25, got %d", v.Reserved)
	}
}

func TestReleaseInventory_Success(t *testing.T) {
	router, handler := setupTestRouterWithInventory(t)
	db := handler.repo.db

	testUUID := uuid.Must(uuid.NewV7())
	variantUUID := uuid.Must(uuid.NewV7())
	variant := ProductVariant{
		ID:        variantUUID,
		ProductID: testUUID,
		SKU:       "SKU-001",
		Price:     100.00,
		Stock:     80,
		Reserved:  20,
	}
	db.Create(&variant)

	router.POST("/api/v1/inventory/:product_id/release", handler.Release)

	body := `{"quantity":10, "variant_id":"` + variantUUID.String() + `"}`
	req, _ := http.NewRequest("POST", "/api/v1/inventory/"+testUUID.String()+"/release", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var v ProductVariant
	db.First(&v, "id = ?", variantUUID)
	if v.Stock != 90 {
		t.Errorf("Expected stock 90, got %d", v.Stock)
	}
	if v.Reserved != 10 {
		t.Errorf("Expected reserved 10, got %d", v.Reserved)
	}
}

func TestReleaseInventory_NotEnoughReserved(t *testing.T) {
	router, handler := setupTestRouterWithInventory(t)
	db := handler.repo.db

	testUUID := uuid.Must(uuid.NewV7())
	variantUUID := uuid.Must(uuid.NewV7())
	variant := ProductVariant{
		ID:        variantUUID,
		ProductID: testUUID,
		SKU:       "SKU-001",
		Price:     100.00,
		Stock:     100,
		Reserved:  5,
	}
	db.Create(&variant)

	router.POST("/api/v1/inventory/:product_id/release", handler.Release)

	body := `{"quantity":50, "variant_id":"` + variantUUID.String() + `"}`
	req, _ := http.NewRequest("POST", "/api/v1/inventory/"+testUUID.String()+"/release", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for not enough reserved, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestReleaseInventory_NoReservedStock(t *testing.T) {
	router, handler := setupTestRouterWithInventory(t)

	testUUID := uuid.Must(uuid.NewV7())
	router.POST("/api/v1/inventory/:product_id/release", handler.Release)

	body := `{"quantity":10}`
	req, _ := http.NewRequest("POST", "/api/v1/inventory/"+testUUID.String()+"/release", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d. Body: %s", w.Code, w.Body.String())
	}
}
