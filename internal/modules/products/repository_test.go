package products

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Auto migrate
	db.AutoMigrate(&Category{}, &Product{}, &ProductVariant{}, &ProductVariantAttribute{}, &ProductImage{})

	return db
}

// ==================== Category Repository Tests ====================

func TestCategoryRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := NewCategoryRepository(db)

	category := &Category{
		Name:        "Electronics",
		Slug:        "electronics",
		Description: "Electronic devices",
	}

	err := repo.Create(category)
	if err != nil {
		t.Fatalf("Failed to create category: %v", err)
	}

	if category.ID == 0 {
		t.Error("Expected category ID to be set")
	}
}

func TestCategoryRepository_FindByID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewCategoryRepository(db)

	// Create category first
	category := &Category{
		Name:        "Electronics",
		Slug:        "electronics",
		Description: "Electronic devices",
	}
	repo.Create(category)

	// Find by ID
	found, err := repo.FindByID(category.ID)
	if err != nil {
		t.Fatalf("Failed to find category: %v", err)
	}

	if found == nil {
		t.Fatal("Expected category to be found")
	}

	if found.Name != "Electronics" {
		t.Errorf("Expected name 'Electronics', got '%s'", found.Name)
	}
}

func TestCategoryRepository_FindByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewCategoryRepository(db)

	found, err := repo.FindByID(999)
	if err != nil {
		t.Fatalf("Failed to find category: %v", err)
	}

	if found != nil {
		t.Error("Expected nil for non-existent category")
	}
}

func TestCategoryRepository_FindBySlug(t *testing.T) {
	db := setupTestDB(t)
	repo := NewCategoryRepository(db)

	category := &Category{
		Name:        "Electronics",
		Slug:        "electronics",
		Description: "Electronic devices",
	}
	repo.Create(category)

	found, err := repo.FindBySlug("electronics")
	if err != nil {
		t.Fatalf("Failed to find category by slug: %v", err)
	}

	if found == nil {
		t.Fatal("Expected category to be found")
	}

	if found.Name != "Electronics" {
		t.Errorf("Expected name 'Electronics', got '%s'", found.Name)
	}
}

func TestCategoryRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	repo := NewCategoryRepository(db)

	category := &Category{
		Name:        "Electronics",
		Slug:        "electronics",
		Description: "Electronic devices",
	}
	repo.Create(category)

	category.Description = "Updated description"
	err := repo.Update(category)
	if err != nil {
		t.Fatalf("Failed to update category: %v", err)
	}

	found, _ := repo.FindByID(category.ID)
	if found.Description != "Updated description" {
		t.Errorf("Expected description 'Updated description', got '%s'", found.Description)
	}
}

func TestCategoryRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	repo := NewCategoryRepository(db)

	category := &Category{
		Name:        "Electronics",
		Slug:        "electronics",
		Description: "Electronic devices",
	}
	repo.Create(category)

	err := repo.Delete(category.ID)
	if err != nil {
		t.Fatalf("Failed to delete category: %v", err)
	}

	found, _ := repo.FindByID(category.ID)
	if found != nil {
		t.Error("Expected category to be nil after deletion")
	}
}

func TestCategoryRepository_FindAll(t *testing.T) {
	db := setupTestDB(t)
	repo := NewCategoryRepository(db)

	// Create multiple categories
	repo.Create(&Category{Name: "Cat1", Slug: "cat1"})
	repo.Create(&Category{Name: "Cat2", Slug: "cat2"})
	repo.Create(&Category{Name: "Cat3", Slug: "cat3"})

	categories, err := repo.FindAll()
	if err != nil {
		t.Fatalf("Failed to find all categories: %v", err)
	}

	if len(categories) != 3 {
		t.Errorf("Expected 3 categories, got %d", len(categories))
	}
}

// ==================== Product Repository Tests ====================

func TestProductRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := NewProductRepository(db)

	category := &Category{Name: "Electronics", Slug: "electronics"}
	db.Create(category)

	product := &Product{
		CategoryID: category.ID,
		Name:       "Laptop",
		Slug:       "laptop",
		Brand:      "Apple",
		BasePrice:  999.99,
		IsActive:   true,
	}

	err := repo.Create(product)
	if err != nil {
		t.Fatalf("Failed to create product: %v", err)
	}

	if product.ID == 0 {
		t.Error("Expected product ID to be set")
	}
}

func TestProductRepository_FindByID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewProductRepository(db)

	category := &Category{Name: "Electronics", Slug: "electronics"}
	db.Create(category)

	product := &Product{
		CategoryID: category.ID,
		Name:       "Laptop",
		Slug:       "laptop",
		Brand:      "Apple",
		BasePrice:  999.99,
		IsActive:   true,
	}
	repo.Create(product)

	found, err := repo.FindByID(product.ID)
	if err != nil {
		t.Fatalf("Failed to find product: %v", err)
	}

	if found == nil {
		t.Fatal("Expected product to be found")
	}

	if found.Name != "Laptop" {
		t.Errorf("Expected name 'Laptop', got '%s'", found.Name)
	}
}

func TestProductRepository_FindBySlug(t *testing.T) {
	db := setupTestDB(t)
	repo := NewProductRepository(db)

	category := &Category{Name: "Electronics", Slug: "electronics"}
	db.Create(category)

	product := &Product{
		CategoryID: category.ID,
		Name:       "Laptop",
		Slug:       "laptop",
		Brand:      "Apple",
		BasePrice:  999.99,
	}
	repo.Create(product)

	found, err := repo.FindBySlug("laptop")
	if err != nil {
		t.Fatalf("Failed to find product by slug: %v", err)
	}

	if found == nil {
		t.Fatal("Expected product to be found")
	}

	if found.Brand != "Apple" {
		t.Errorf("Expected brand 'Apple', got '%s'", found.Brand)
	}
}

func TestProductRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	repo := NewProductRepository(db)

	category := &Category{Name: "Electronics", Slug: "electronics"}
	db.Create(category)

	product := &Product{
		CategoryID: category.ID,
		Name:       "Laptop",
		Slug:       "laptop",
		BasePrice:  999.99,
	}
	repo.Create(product)

	product.BasePrice = 1299.99
	err := repo.Update(product)
	if err != nil {
		t.Fatalf("Failed to update product: %v", err)
	}

	found, _ := repo.FindByID(product.ID)
	if found.BasePrice != 1299.99 {
		t.Errorf("Expected price 1299.99, got %f", found.BasePrice)
	}
}

func TestProductRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	repo := NewProductRepository(db)

	category := &Category{Name: "Electronics", Slug: "electronics"}
	db.Create(category)

	product := &Product{
		CategoryID: category.ID,
		Name:       "Laptop",
		Slug:       "laptop",
		BasePrice:  999.99,
	}
	repo.Create(product)

	err := repo.Delete(product.ID)
	if err != nil {
		t.Fatalf("Failed to delete product: %v", err)
	}

	found, _ := repo.FindByID(product.ID)
	if found != nil {
		t.Error("Expected product to be nil after deletion")
	}
}

func TestProductRepository_FindAll(t *testing.T) {
	db := setupTestDB(t)
	repo := NewProductRepository(db)

	category := &Category{Name: "Electronics", Slug: "electronics"}
	db.Create(category)

	repo.Create(&Product{CategoryID: category.ID, Name: "P1", Slug: "p1", BasePrice: 100})
	repo.Create(&Product{CategoryID: category.ID, Name: "P2", Slug: "p2", BasePrice: 200})
	repo.Create(&Product{CategoryID: category.ID, Name: "P3", Slug: "p3", BasePrice: 300})

	products, err := repo.FindAll(0, 10)
	if err != nil {
		t.Fatalf("Failed to find all products: %v", err)
	}

	if len(products) != 3 {
		t.Errorf("Expected 3 products, got %d", len(products))
	}
}

func TestProductRepository_FindByCategoryID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewProductRepository(db)

	cat1 := &Category{Name: "Electronics", Slug: "electronics"}
	cat2 := &Category{Name: "Books", Slug: "books"}
	db.Create(cat1)
	db.Create(cat2)

	repo.Create(&Product{CategoryID: cat1.ID, Name: "Laptop", Slug: "laptop", BasePrice: 999})
	repo.Create(&Product{CategoryID: cat1.ID, Name: "Phone", Slug: "phone", BasePrice: 699})
	repo.Create(&Product{CategoryID: cat2.ID, Name: "Book", Slug: "book", BasePrice: 29})

	electronics, err := repo.FindByCategoryID(cat1.ID, 0, 10)
	if err != nil {
		t.Fatalf("Failed to find products by category: %v", err)
	}

	if len(electronics) != 2 {
		t.Errorf("Expected 2 products in Electronics, got %d", len(electronics))
	}
}

func TestProductRepository_FindByActive(t *testing.T) {
	db := setupTestDB(t)
	repo := NewProductRepository(db)

	category := &Category{Name: "Electronics", Slug: "electronics"}
	db.Create(category)

	// Create active product
	activeProduct := &Product{CategoryID: category.ID, Name: "Active", Slug: "active", BasePrice: 100, IsActive: true}
	repo.Create(activeProduct)

	// Create inactive product - first as active, then update to inactive
	inactiveProduct := &Product{CategoryID: category.ID, Name: "Inactive", Slug: "inactive", BasePrice: 200}
	repo.Create(inactiveProduct)

	// Use Update with Select to only update IsActive field
	db.Model(&Product{}).Where("id = ?", inactiveProduct.ID).Update("is_active", false)

	active, err := repo.FindByActive(true, 0, 10)
	if err != nil {
		t.Fatalf("Failed to find active products: %v", err)
	}

	if len(active) != 1 {
		t.Errorf("Expected 1 active product, got %d", len(active))
	}
}

// ==================== ProductVariant Repository Tests ====================

func TestProductVariantRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := NewProductVariantRepository(db)

	category := &Category{Name: "Electronics", Slug: "electronics"}
	db.Create(category)

	product := &Product{
		CategoryID: category.ID,
		Name:       "Laptop",
		Slug:       "laptop",
		BasePrice:  999.99,
	}
	db.Create(product)

	variant := &ProductVariant{
		ProductID: product.ID,
		SKU:       "LAPTOP-001",
		Price:     999.99,
		Stock:     10,
	}

	err := repo.Create(variant)
	if err != nil {
		t.Fatalf("Failed to create variant: %v", err)
	}

	if variant.ID == 0 {
		t.Error("Expected variant ID to be set")
	}
}

func TestProductVariantRepository_FindByID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewProductVariantRepository(db)

	category := &Category{Name: "Electronics", Slug: "electronics"}
	db.Create(category)

	product := &Product{
		CategoryID: category.ID,
		Name:       "Laptop",
		Slug:       "laptop",
		BasePrice:  999.99,
	}
	db.Create(product)

	variant := &ProductVariant{
		ProductID: product.ID,
		SKU:       "LAPTOP-001",
		Price:     999.99,
		Stock:     10,
	}
	repo.Create(variant)

	found, err := repo.FindByID(variant.ID)
	if err != nil {
		t.Fatalf("Failed to find variant: %v", err)
	}

	if found == nil {
		t.Fatal("Expected variant to be found")
	}

	if found.SKU != "LAPTOP-001" {
		t.Errorf("Expected SKU 'LAPTOP-001', got '%s'", found.SKU)
	}
}

func TestProductVariantRepository_FindBySKU(t *testing.T) {
	db := setupTestDB(t)
	repo := NewProductVariantRepository(db)

	category := &Category{Name: "Electronics", Slug: "electronics"}
	db.Create(category)

	product := &Product{
		CategoryID: category.ID,
		Name:       "Laptop",
		Slug:       "laptop",
		BasePrice:  999.99,
	}
	db.Create(product)

	variant := &ProductVariant{
		ProductID: product.ID,
		SKU:       "LAPTOP-001",
		Price:     999.99,
		Stock:     10,
	}
	repo.Create(variant)

	found, err := repo.FindBySKU("LAPTOP-001")
	if err != nil {
		t.Fatalf("Failed to find variant by SKU: %v", err)
	}

	if found == nil {
		t.Fatal("Expected variant to be found")
	}
}

func TestProductVariantRepository_UpdateStock(t *testing.T) {
	db := setupTestDB(t)
	repo := NewProductVariantRepository(db)

	category := &Category{Name: "Electronics", Slug: "electronics"}
	db.Create(category)

	product := &Product{
		CategoryID: category.ID,
		Name:       "Laptop",
		Slug:       "laptop",
		BasePrice:  999.99,
	}
	db.Create(product)

	variant := &ProductVariant{
		ProductID: product.ID,
		SKU:       "LAPTOP-001",
		Price:     999.99,
		Stock:     10,
	}
	repo.Create(variant)

	err := repo.UpdateStock(variant.ID, 5)
	if err != nil {
		t.Fatalf("Failed to update stock: %v", err)
	}

	found, _ := repo.FindByID(variant.ID)
	if found.Stock != 5 {
		t.Errorf("Expected stock 5, got %d", found.Stock)
	}
}

// ==================== ProductImage Repository Tests ====================

func TestProductImageRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := NewProductImageRepository(db)

	category := &Category{Name: "Electronics", Slug: "electronics"}
	db.Create(category)

	product := &Product{
		CategoryID: category.ID,
		Name:       "Laptop",
		Slug:       "laptop",
		BasePrice:  999.99,
	}
	db.Create(product)

	image := &ProductImage{
		ProductID: product.ID,
		URLImage:  "https://example.com/laptop.jpg",
		IsMain:    true,
		SortOrder: 1,
	}

	err := repo.Create(image)
	if err != nil {
		t.Fatalf("Failed to create image: %v", err)
	}

	if image.ID == 0 {
		t.Error("Expected image ID to be set")
	}
}

func TestProductImageRepository_SetMainImage(t *testing.T) {
	db := setupTestDB(t)
	repo := NewProductImageRepository(db)

	category := &Category{Name: "Electronics", Slug: "electronics"}
	db.Create(category)

	product := &Product{
		CategoryID: category.ID,
		Name:       "Laptop",
		Slug:       "laptop",
		BasePrice:  999.99,
	}
	db.Create(product)

	img1 := &ProductImage{ProductID: product.ID, URLImage: "img1.jpg", IsMain: true}
	img2 := &ProductImage{ProductID: product.ID, URLImage: "img2.jpg", IsMain: false}
	repo.Create(img1)
	repo.Create(img2)

	err := repo.SetMainImage(product.ID, img2.ID)
	if err != nil {
		t.Fatalf("Failed to set main image: %v", err)
	}

	updated, _ := repo.FindByID(img2.ID)
	if !updated.IsMain {
		t.Error("Expected img2 to be main image")
	}

	// Check that img1 is no longer main
	img1Updated, _ := repo.FindByID(img1.ID)
	if img1Updated.IsMain {
		t.Error("Expected img1 to not be main image")
	}
}
