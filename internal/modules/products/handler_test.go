package products

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bey/internal/shared/response"
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid/v5"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Test UUIDs — valid UUIDv7 strings for handler URL params
var (
	testProductUUID     = uuid.Must(uuid.NewV7())
	testCategoryUUID    = uuid.Must(uuid.NewV7())
	testVariantUUID     = uuid.Must(uuid.NewV7())
	testImageUUID       = uuid.Must(uuid.NewV7())
	testNonExistentUUID = uuid.Must(uuid.NewV7())
)

func setupTestDBForHandler(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	db.AutoMigrate(&Category{}, &Product{}, &ProductVariant{}, &ProductVariantAttribute{}, &ProductImage{})
	return db
}

func setupTestRouterWithProducts(t *testing.T) (*gin.Engine, *ProductHandler, *ProductRepository, *CategoryRepository) {
	gin.SetMode(gin.TestMode)
	db := setupTestDBForHandler(t)
	productRepo := NewProductRepository(db)
	categoryRepo := NewCategoryRepository(db)
	variantRepo := NewProductVariantRepository(db)
	imageRepo := NewProductImageRepository(db)
	handler := NewProductHandler(categoryRepo, productRepo, variantRepo, imageRepo)

	router := gin.New()
	return router, handler, productRepo, categoryRepo
}

func TestGetProducts_Success(t *testing.T) {
	router, handler, _, _ := setupTestRouterWithProducts(t)

	router.GET("/api/v1/products", handler.GetProducts)

	req, _ := http.NewRequest("GET", "/api/v1/products", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestGetProducts_InvalidLimit(t *testing.T) {
	router, handler, _, _ := setupTestRouterWithProducts(t)

	router.GET("/api/v1/products", handler.GetProducts)

	req, _ := http.NewRequest("GET", "/api/v1/products?limit=0", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for zero limit, got %d", w.Code)
	}
}

func TestGetProduct_NotFound(t *testing.T) {
	router, handler, _, _ := setupTestRouterWithProducts(t)

	router.GET("/api/v1/products/:id", handler.GetProduct)

	req, _ := http.NewRequest("GET", "/api/v1/products/"+testNonExistentUUID.String(), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 for non-existent product, got %d", w.Code)
	}
}

func TestGetProduct_InvalidID(t *testing.T) {
	router, handler, _, _ := setupTestRouterWithProducts(t)

	router.GET("/api/v1/products/:id", handler.GetProduct)

	req, _ := http.NewRequest("GET", "/api/v1/products/invalid", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid ID, got %d", w.Code)
	}
}

func TestCreateProduct_Success(t *testing.T) {
	router, handler, productRepo, _ := setupTestRouterWithProducts(t)

	productRepo.Create(&Product{
		ID:        testProductUUID,
		Name:      "Existing Product",
		Slug:      "existing-product",
		BasePrice: 10.99,
	})

	router.POST("/api/v1/products", handler.CreateProduct)

	body := `{"name":"Test Product","slug":"test-product-2","base_price":10.99,"category_id":"` + testCategoryUUID.String() + `"}`
	req, _ := http.NewRequest("POST", "/api/v1/products", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestCreateProduct_InvalidBody(t *testing.T) {
	router, handler, _, _ := setupTestRouterWithProducts(t)

	router.POST("/api/v1/products", handler.CreateProduct)

	req, _ := http.NewRequest("POST", "/api/v1/products", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid body, got %d", w.Code)
	}
}

func TestCreateProduct_MissingRequiredFields(t *testing.T) {
	router, handler, _, _ := setupTestRouterWithProducts(t)

	router.POST("/api/v1/products", handler.CreateProduct)

	body := `{"name":"Test Product"}`
	req, _ := http.NewRequest("POST", "/api/v1/products", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for missing required fields, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestGetProducts_WithPagination(t *testing.T) {
	router, handler, productRepo, _ := setupTestRouterWithProducts(t)

	for i := 1; i <= 5; i++ {
		productRepo.Create(&Product{
			Name:      "Product " + string(rune('0'+i)),
			Slug:      "product-" + string(rune('0'+i)),
			BasePrice: float64(i * 10),
			IsActive:  true,
		})
	}

	router.GET("/api/v1/products", handler.GetProducts)

	req, _ := http.NewRequest("GET", "/api/v1/products?offset=0&limit=2", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var apiResp response.ApiResponse
	if err := json.Unmarshal(w.Body.Bytes(), &apiResp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if !apiResp.Success {
		t.Error("Expected success to be true")
	}

	products, ok := apiResp.Data.([]interface{})
	if !ok {
		t.Fatalf("Expected Data to be array, got %T", apiResp.Data)
	}

	if len(products) != 2 {
		t.Errorf("Expected 2 products, got %d", len(products))
	}
}

func TestGetProductByID_Success(t *testing.T) {
	router, handler, productRepo, _ := setupTestRouterWithProducts(t)

	product := &Product{
		ID:        testProductUUID,
		Name:      "Test Product",
		Slug:      "test-product",
		BasePrice: 10.99,
	}
	productRepo.Create(product)

	router.GET("/api/v1/products/:id", handler.GetProduct)

	req, _ := http.NewRequest("GET", "/api/v1/products/"+testProductUUID.String(), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestGetCategories_Success(t *testing.T) {
	router, handler, _, _ := setupTestRouterWithProducts(t)

	router.GET("/api/v1/categories", handler.GetCategories)

	req, _ := http.NewRequest("GET", "/api/v1/categories", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestCreateCategory_Success(t *testing.T) {
	router, handler, _, _ := setupTestRouterWithProducts(t)

	router.POST("/api/v1/categories", handler.CreateCategory)

	body := `{"name":"Test Category","slug":"test-category"}`
	req, _ := http.NewRequest("POST", "/api/v1/categories", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestGetCategory_Success(t *testing.T) {
	router, handler, _, categoryRepo := setupTestRouterWithProducts(t)

	category := &Category{
		ID:   testCategoryUUID,
		Name: "Test Category",
		Slug: "test-category",
	}
	categoryRepo.Create(category)

	router.GET("/api/v1/categories/:id", handler.GetCategory)

	req, _ := http.NewRequest("GET", "/api/v1/categories/"+testCategoryUUID.String(), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestUpdateCategory_Success(t *testing.T) {
	router, handler, _, categoryRepo := setupTestRouterWithProducts(t)

	category := &Category{
		ID:   testCategoryUUID,
		Name: "Test Category",
		Slug: "test-category",
	}
	categoryRepo.Create(category)

	router.PUT("/api/v1/categories/:id", handler.UpdateCategory)

	body := `{"name":"Updated Category"}`
	req, _ := http.NewRequest("PUT", "/api/v1/categories/"+testCategoryUUID.String(), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestDeleteCategory_Success(t *testing.T) {
	router, handler, _, categoryRepo := setupTestRouterWithProducts(t)

	category := &Category{
		ID:   testCategoryUUID,
		Name: "Test Category",
		Slug: "test-category",
	}
	categoryRepo.Create(category)

	router.DELETE("/api/v1/categories/:id", handler.DeleteCategory)

	req, _ := http.NewRequest("DELETE", "/api/v1/categories/"+testCategoryUUID.String(), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestGetCategoryBySlug_Success(t *testing.T) {
	router, handler, _, categoryRepo := setupTestRouterWithProducts(t)

	categoryRepo.Create(&Category{
		Name: "Test Category",
		Slug: "test-category",
	})

	router.GET("/api/v1/categories/slug/:slug", handler.GetCategoryBySlug)

	req, _ := http.NewRequest("GET", "/api/v1/categories/slug/test-category", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestGetVariantsByProduct_Success(t *testing.T) {
	router, handler, _, _ := setupTestRouterWithProducts(t)

	router.GET("/api/v1/products/:id/variants", handler.GetVariantsByProduct)

	req, _ := http.NewRequest("GET", "/api/v1/products/"+testProductUUID.String()+"/variants", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestGetImagesByProduct_Success(t *testing.T) {
	router, handler, _, _ := setupTestRouterWithProducts(t)

	router.GET("/api/v1/products/:id/images", handler.GetImagesByProduct)

	req, _ := http.NewRequest("GET", "/api/v1/products/"+testProductUUID.String()+"/images", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestGetProductBySlug_Success(t *testing.T) {
	router, handler, productRepo, _ := setupTestRouterWithProducts(t)

	productRepo.Create(&Product{
		Name:      "Test Product",
		Slug:      "test-product",
		BasePrice: 10.99,
	})

	router.GET("/api/v1/products/slug/:slug", handler.GetProductBySlug)

	req, _ := http.NewRequest("GET", "/api/v1/products/slug/test-product", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestUpdateProduct_Success(t *testing.T) {
	router, handler, productRepo, _ := setupTestRouterWithProducts(t)

	product := &Product{
		ID:        testProductUUID,
		Name:      "Test Product",
		Slug:      "test-product",
		BasePrice: 10.99,
	}
	productRepo.Create(product)

	router.PUT("/api/v1/products/:id", handler.UpdateProduct)

	body := `{"name":"Updated Product","base_price":20.99}`
	req, _ := http.NewRequest("PUT", "/api/v1/products/"+testProductUUID.String(), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestDeleteProduct_Success(t *testing.T) {
	router, handler, productRepo, _ := setupTestRouterWithProducts(t)

	product := &Product{
		ID:        testProductUUID,
		Name:      "Test Product",
		Slug:      "test-product",
		BasePrice: 10.99,
	}
	productRepo.Create(product)

	router.DELETE("/api/v1/products/:id", handler.DeleteProduct)

	req, _ := http.NewRequest("DELETE", "/api/v1/products/"+testProductUUID.String(), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestCreateVariant_Success(t *testing.T) {
	router, handler, productRepo, _ := setupTestRouterWithProducts(t)

	product := &Product{
		ID:        testProductUUID,
		Name:      "Test Product",
		Slug:      "test-product",
		BasePrice: 10.99,
	}
	productRepo.Create(product)

	router.POST("/api/v1/products/:id/variants", handler.CreateVariant)

	body := `{"product_id":"` + testProductUUID.String() + `","sku":"VAR001","price":15.99,"stock":100,"color":"red","size":"M","weight":"1.5"}`
	req, _ := http.NewRequest("POST", "/api/v1/products/"+testProductUUID.String()+"/variants", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestGetVariant_Success(t *testing.T) {
	router, handler, _, _ := setupTestRouterWithProducts(t)

	router.GET("/api/v1/variants/:id", handler.GetVariant)

	req, _ := http.NewRequest("GET", "/api/v1/variants/"+testVariantUUID.String(), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestUpdateVariant_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDBForHandler(t)
	productRepo := NewProductRepository(db)
	categoryRepo := NewCategoryRepository(db)
	variantRepo := NewProductVariantRepository(db)
	imageRepo := NewProductImageRepository(db)
	handler := NewProductHandler(categoryRepo, productRepo, variantRepo, imageRepo)

	variant := &ProductVariant{
		ID:    testVariantUUID,
		SKU:   "TEST001",
		Price: 10.99,
		Stock: 50,
	}
	variantRepo.Create(variant)

	router := gin.New()
	router.PUT("/api/v1/variants/:id", handler.UpdateVariant)

	body := `{"price":20.99}`
	req, _ := http.NewRequest("PUT", "/api/v1/variants/"+testVariantUUID.String(), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestDeleteVariant_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDBForHandler(t)
	productRepo := NewProductRepository(db)
	categoryRepo := NewCategoryRepository(db)
	variantRepo := NewProductVariantRepository(db)
	imageRepo := NewProductImageRepository(db)
	handler := NewProductHandler(categoryRepo, productRepo, variantRepo, imageRepo)

	variant := &ProductVariant{
		ID:    testVariantUUID,
		SKU:   "TEST001",
		Price: 10.99,
		Stock: 50,
	}
	variantRepo.Create(variant)

	router := gin.New()
	router.DELETE("/api/v1/variants/:id", handler.DeleteVariant)

	req, _ := http.NewRequest("DELETE", "/api/v1/variants/"+testVariantUUID.String(), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestCreateImage_Success(t *testing.T) {
	router, handler, productRepo, _ := setupTestRouterWithProducts(t)

	product := &Product{
		ID:        testProductUUID,
		Name:      "Test Product",
		Slug:      "test-product",
		BasePrice: 10.99,
	}
	productRepo.Create(product)

	router.POST("/api/v1/products/:id/images", handler.CreateImage)

	body := `{"product_id":"` + testProductUUID.String() + `","url_image":"http://example.com/image.jpg","is_main":true}`
	req, _ := http.NewRequest("POST", "/api/v1/products/"+testProductUUID.String()+"/images", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestGetImage_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDBForHandler(t)
	productRepo := NewProductRepository(db)
	categoryRepo := NewCategoryRepository(db)
	variantRepo := NewProductVariantRepository(db)
	imageRepo := NewProductImageRepository(db)
	handler := NewProductHandler(categoryRepo, productRepo, variantRepo, imageRepo)

	image := &ProductImage{
		ID:       testImageUUID,
		URLImage: "http://example.com/image.jpg",
		IsMain:   true,
	}
	imageRepo.Create(image)

	router := gin.New()
	router.GET("/api/v1/images/:id", handler.GetImage)

	req, _ := http.NewRequest("GET", "/api/v1/images/"+testImageUUID.String(), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestUpdateImage_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDBForHandler(t)
	productRepo := NewProductRepository(db)
	categoryRepo := NewCategoryRepository(db)
	variantRepo := NewProductVariantRepository(db)
	imageRepo := NewProductImageRepository(db)
	handler := NewProductHandler(categoryRepo, productRepo, variantRepo, imageRepo)

	image := &ProductImage{
		ID:       testImageUUID,
		URLImage: "http://example.com/image.jpg",
		IsMain:   true,
	}
	imageRepo.Create(image)

	router := gin.New()
	router.PUT("/api/v1/images/:id", handler.UpdateImage)

	body := `{"url_image":"http://example.com/new-image.jpg"}`
	req, _ := http.NewRequest("PUT", "/api/v1/images/"+testImageUUID.String(), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestDeleteImage_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDBForHandler(t)
	productRepo := NewProductRepository(db)
	categoryRepo := NewCategoryRepository(db)
	variantRepo := NewProductVariantRepository(db)
	imageRepo := NewProductImageRepository(db)
	handler := NewProductHandler(categoryRepo, productRepo, variantRepo, imageRepo)

	image := &ProductImage{
		ID:       testImageUUID,
		URLImage: "http://example.com/image.jpg",
		IsMain:   true,
	}
	imageRepo.Create(image)

	router := gin.New()
	router.DELETE("/api/v1/images/:id", handler.DeleteImage)

	req, _ := http.NewRequest("DELETE", "/api/v1/images/"+testImageUUID.String(), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestSetMainImage_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDBForHandler(t)
	productRepo := NewProductRepository(db)
	categoryRepo := NewCategoryRepository(db)
	variantRepo := NewProductVariantRepository(db)
	imageRepo := NewProductImageRepository(db)
	handler := NewProductHandler(categoryRepo, productRepo, variantRepo, imageRepo)

	product := &Product{
		ID:        testProductUUID,
		Name:      "Test Product",
		Slug:      "test-product",
		BasePrice: 10.99,
	}
	productRepo.Create(product)

	image := &ProductImage{
		ID:        testImageUUID,
		ProductID: testProductUUID,
		URLImage:  "http://example.com/image.jpg",
		IsMain:    false,
	}
	imageRepo.Create(image)

	router := gin.New()
	router.PUT("/api/v1/products/:id/images/:image_id/main", handler.SetMainImage)

	req, _ := http.NewRequest("PUT", "/api/v1/products/"+testProductUUID.String()+"/images/"+testImageUUID.String()+"/main", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}
