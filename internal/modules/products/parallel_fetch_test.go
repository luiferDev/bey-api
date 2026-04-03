package products

import (
	"sync"
	"testing"

	"github.com/gofrs/uuid/v5"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var dbCounter int
var dbMu sync.Mutex

func setupParallelTestDB(t *testing.T) *gorm.DB {
	dbMu.Lock()
	dbCounter++
	dbMu.Unlock()

	dbName := t.Name() + "_" + uuid.Must(uuid.NewV7()).String() + ".db"
	db, err := gorm.Open(sqlite.Open(dbName), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	db.AutoMigrate(&Category{}, &Product{}, &ProductVariant{}, &ProductVariantAttribute{}, &ProductImage{})

	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	})

	return db
}

func TestProductRepository_FindByIDWithRelationsParallel(t *testing.T) {
	db := setupParallelTestDB(t)
	variantRepo := NewProductVariantRepository(db)
	imageRepo := NewProductImageRepository(db)
	repo := NewProductRepositoryWithRelations(db, variantRepo, imageRepo)

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
	db.Create(variant)

	image := &ProductImage{
		ProductID: product.ID,
		URLImage:  "https://example.com/laptop.jpg",
		IsMain:    true,
	}
	db.Create(image)

	result, err := repo.FindByIDWithRelationsParallel(product.ID)
	if err != nil {
		t.Fatalf("Failed to find product with relations: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result to not be nil")
	}

	if result.Product == nil {
		t.Error("Expected product to be loaded")
	}

	if result.Product.Name != "Laptop" {
		t.Errorf("Expected product name 'Laptop', got '%s'", result.Product.Name)
	}

	if len(result.Variants) != 1 {
		t.Errorf("Expected 1 variant, got %d", len(result.Variants))
	}

	if len(result.Images) != 1 {
		t.Errorf("Expected 1 image, got %d", len(result.Images))
	}
}

func TestProductRepository_FindByIDWithRelationsParallel_NotFound(t *testing.T) {
	db := setupParallelTestDB(t)
	variantRepo := NewProductVariantRepository(db)
	imageRepo := NewProductImageRepository(db)
	repo := NewProductRepositoryWithRelations(db, variantRepo, imageRepo)

	result, err := repo.FindByIDWithRelationsParallel(uuid.Must(uuid.NewV7()))
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result != nil {
		t.Error("Expected nil result for non-existent product")
	}
}

func TestProductRepository_FindByIDWithRelationsParallel_MultipleRelations(t *testing.T) {
	db := setupParallelTestDB(t)
	variantRepo := NewProductVariantRepository(db)
	imageRepo := NewProductImageRepository(db)
	repo := NewProductRepositoryWithRelations(db, variantRepo, imageRepo)

	category := &Category{Name: "Electronics", Slug: "electronics"}
	db.Create(category)

	product := &Product{
		CategoryID: category.ID,
		Name:       "Laptop",
		Slug:       "laptop",
		BasePrice:  999.99,
	}
	db.Create(product)

	for i := 0; i < 3; i++ {
		db.Create(&ProductVariant{
			ProductID: product.ID,
			SKU:       "SKU-00" + string(rune('0'+i)),
			Price:     999.99,
			Stock:     10,
		})
	}

	for i := 0; i < 5; i++ {
		db.Create(&ProductImage{
			ProductID: product.ID,
			URLImage:  "https://example.com/img" + string(rune('0'+i)) + ".jpg",
			IsMain:    i == 0,
		})
	}

	result, err := repo.FindByIDWithRelationsParallel(product.ID)
	if err != nil {
		t.Fatalf("Failed to find product with relations: %v", err)
	}

	if len(result.Variants) != 3 {
		t.Errorf("Expected 3 variants, got %d", len(result.Variants))
	}

	if len(result.Images) != 5 {
		t.Errorf("Expected 5 images, got %d", len(result.Images))
	}
}

func TestProductRepository_FindByIDWithRelationsParallel_EmptyRelations(t *testing.T) {
	db := setupParallelTestDB(t)
	variantRepo := NewProductVariantRepository(db)
	imageRepo := NewProductImageRepository(db)
	repo := NewProductRepositoryWithRelations(db, variantRepo, imageRepo)

	category := &Category{Name: "Electronics", Slug: "electronics"}
	db.Create(category)

	product := &Product{
		CategoryID: category.ID,
		Name:       "Laptop",
		Slug:       "laptop",
		BasePrice:  999.99,
	}
	db.Create(product)

	result, err := repo.FindByIDWithRelationsParallel(product.ID)
	if err != nil {
		t.Fatalf("Failed to find product with relations: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result to not be nil")
	}

	if len(result.Variants) != 0 {
		t.Errorf("Expected 0 variants, got %d", len(result.Variants))
	}

	if len(result.Images) != 0 {
		t.Errorf("Expected 0 images, got %d", len(result.Images))
	}
}

func TestProductRepository_FindByIDWithRelationsParallel_ErrorPropagation(t *testing.T) {
	db := setupParallelTestDB(t)
	db.AutoMigrate(&Category{}, &Product{})

	repo := NewProductRepository(db)

	category := &Category{Name: "Electronics", Slug: "electronics"}
	db.Create(category)

	product := &Product{
		CategoryID: category.ID,
		Name:       "Laptop",
		Slug:       "laptop",
		BasePrice:  999.99,
	}
	db.Create(product)

	repo.variantRepo = nil
	repo.imageRepo = nil

	result, err := repo.FindByIDWithRelationsParallel(product.ID)
	if err == nil {
		t.Error("Expected error when variant/image repos are nil")
	}

	if result != nil {
		t.Error("Expected nil result on error")
	}
}

func TestProductRepository_FindByIDWithRelationsParallel_Concurrent(t *testing.T) {
	db := setupParallelTestDB(t)
	variantRepo := NewProductVariantRepository(db)
	imageRepo := NewProductImageRepository(db)
	repo := NewProductRepositoryWithRelations(db, variantRepo, imageRepo)

	category := &Category{Name: "Electronics", Slug: "electronics"}
	db.Create(category)

	product := &Product{
		CategoryID: category.ID,
		Name:       "Laptop",
		Slug:       "laptop",
		BasePrice:  999.99,
	}
	db.Create(product)

	for i := 0; i < 3; i++ {
		db.Create(&ProductVariant{
			ProductID: product.ID,
			SKU:       "SKU-00" + string(rune('0'+i)),
			Price:     999.99,
			Stock:     10,
		})
	}

	var wg sync.WaitGroup
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := repo.FindByIDWithRelationsParallel(product.ID)
			if err != nil {
				errors <- err
				return
			}
			if result == nil || result.Product == nil {
				errors <- nil
			}
		}()
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		if err != nil {
			t.Errorf("Concurrent access error: %v", err)
		}
	}
}
