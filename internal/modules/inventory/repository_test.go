package inventory

import (
	"testing"

	"github.com/gofrs/uuid/v5"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type ProductVariant struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	ProductID uuid.UUID `json:"product_id"`
	SKU       string    `gorm:"size:100;not null" json:"sku"`
	Price     float64   `gorm:"type:decimal(12,2);not null" json:"price"`
	Stock     int       `gorm:"default:0" json:"stock"`
	Reserved  int       `gorm:"default:0" json:"reserved"`
}

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	db.AutoMigrate(&Inventory{}, &ProductVariant{})

	return db
}

// ==================== Inventory Repository Tests ====================

func TestInventoryRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := NewInventoryRepository(db)

	inv := &Inventory{
		ProductID: uuid.Must(uuid.NewV7()),
		Quantity:  100,
		Reserved:  0,
	}

	err := repo.Create(inv)
	if err != nil {
		t.Fatalf("Failed to create inventory: %v", err)
	}

	if inv.ID == uuid.Nil {
		t.Error("Expected inventory ID to be set")
	}
}

func TestInventoryRepository_FindByID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewInventoryRepository(db)

	inv := &Inventory{
		ProductID: uuid.Must(uuid.NewV7()),
		Quantity:  100,
		Reserved:  0,
	}
	repo.Create(inv)

	found, err := repo.FindByID(inv.ID)
	if err != nil {
		t.Fatalf("Failed to find inventory: %v", err)
	}

	if found == nil {
		t.Fatal("Expected inventory to be found")
	}

	if found.Quantity != 100 {
		t.Errorf("Expected quantity 100, got %d", found.Quantity)
	}
}

func TestInventoryRepository_FindByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewInventoryRepository(db)

	found, err := repo.FindByID(uuid.Must(uuid.NewV7()))
	if err != nil {
		t.Fatalf("Failed to find inventory: %v", err)
	}

	if found != nil {
		t.Error("Expected nil for non-existent inventory")
	}
}

func TestInventoryRepository_FindByProductID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewInventoryRepository(db)

	productID := uuid.Must(uuid.NewV7())
	inv := &Inventory{
		ProductID: productID,
		Quantity:  100,
		Reserved:  0,
	}
	repo.Create(inv)

	found, err := repo.FindByProductID(productID)
	if err != nil {
		t.Fatalf("Failed to find inventory by product ID: %v", err)
	}

	if found == nil {
		t.Fatal("Expected inventory to be found")
	}

	if found.Quantity != 100 {
		t.Errorf("Expected quantity 100, got %d", found.Quantity)
	}
}

func TestInventoryRepository_FindByProductID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewInventoryRepository(db)

	found, err := repo.FindByProductID(uuid.Must(uuid.NewV7()))
	if err != nil {
		t.Fatalf("Failed to find inventory by product ID: %v", err)
	}

	if found != nil {
		t.Error("Expected nil for non-existent product")
	}
}

func TestInventoryRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	repo := NewInventoryRepository(db)

	productID := uuid.Must(uuid.NewV7())
	inv := &Inventory{
		ProductID: productID,
		Quantity:  100,
		Reserved:  0,
	}
	repo.Create(inv)

	inv.Quantity = 150
	err := repo.Update(inv)
	if err != nil {
		t.Fatalf("Failed to update inventory: %v", err)
	}

	found, _ := repo.FindByProductID(productID)
	if found.Quantity != 150 {
		t.Errorf("Expected quantity 150, got %d", found.Quantity)
	}
}

func TestInventoryRepository_UpdateQuantity(t *testing.T) {
	db := setupTestDB(t)
	repo := NewInventoryRepository(db)

	productID := uuid.Must(uuid.NewV7())
	inv := &Inventory{
		ProductID: productID,
		Quantity:  100,
		Reserved:  0,
	}
	repo.Create(inv)

	err := repo.UpdateQuantity(productID, 200)
	if err != nil {
		t.Fatalf("Failed to update quantity: %v", err)
	}

	found, _ := repo.FindByProductID(productID)
	if found.Quantity != 200 {
		t.Errorf("Expected quantity 200, got %d", found.Quantity)
	}
}

func TestInventoryRepository_Reserve(t *testing.T) {
	db := setupTestDB(t)
	repo := NewInventoryRepository(db)

	productID := uuid.Must(uuid.NewV7())
	inv := &Inventory{
		ProductID: productID,
		Quantity:  100,
		Reserved:  0,
	}
	repo.Create(inv)

	err := repo.Reserve(productID, 30)
	if err != nil {
		t.Fatalf("Failed to reserve inventory: %v", err)
	}

	found, _ := repo.FindByProductID(productID)
	if found.Quantity != 70 {
		t.Errorf("Expected quantity 70, got %d", found.Quantity)
	}
	if found.Reserved != 30 {
		t.Errorf("Expected reserved 30, got %d", found.Reserved)
	}
}

// ==================== Product Variant Stock Tests ====================

func TestInventoryRepository_GetProductInventory_NoVariants(t *testing.T) {
	db := setupTestDB(t)
	repo := NewInventoryRepository(db)

	productID := uuid.Must(uuid.NewV7())

	result, err := repo.GetProductInventory(productID)
	if err != nil {
		t.Fatalf("Failed to get product inventory: %v", err)
	}

	if result.TotalStock != 0 {
		t.Errorf("Expected total_stock 0, got %d", result.TotalStock)
	}
	if result.TotalReserved != 0 {
		t.Errorf("Expected total_reserved 0, got %d", result.TotalReserved)
	}
	if len(result.Variants) != 0 {
		t.Errorf("Expected 0 variants, got %d", len(result.Variants))
	}
}

func TestInventoryRepository_GetProductInventory_WithVariants(t *testing.T) {
	db := setupTestDB(t)
	repo := NewInventoryRepository(db)

	productID := uuid.Must(uuid.NewV7())
	v1 := ProductVariant{
		ID:        uuid.Must(uuid.NewV7()),
		ProductID: productID,
		SKU:       "SKU-001",
		Price:     100.00,
		Stock:     50,
		Reserved:  10,
	}
	v2 := ProductVariant{
		ID:        uuid.Must(uuid.NewV7()),
		ProductID: productID,
		SKU:       "SKU-002",
		Price:     150.00,
		Stock:     30,
		Reserved:  5,
	}
	db.Create(&v1)
	db.Create(&v2)

	result, err := repo.GetProductInventory(productID)
	if err != nil {
		t.Fatalf("Failed to get product inventory: %v", err)
	}

	if result.TotalStock != 80 {
		t.Errorf("Expected total_stock 80, got %d", result.TotalStock)
	}
	if result.TotalReserved != 15 {
		t.Errorf("Expected total_reserved 15, got %d", result.TotalReserved)
	}
	if len(result.Variants) != 2 {
		t.Errorf("Expected 2 variants, got %d", len(result.Variants))
	}
}

func TestInventoryRepository_ReserveVariantStock_Success(t *testing.T) {
	db := setupTestDB(t)
	repo := NewInventoryRepository(db)

	variantID := uuid.Must(uuid.NewV7())
	v := ProductVariant{
		ID:        variantID,
		ProductID: uuid.Must(uuid.NewV7()),
		SKU:       "SKU-001",
		Price:     100.00,
		Stock:     100,
		Reserved:  0,
	}
	db.Create(&v)

	err := repo.ReserveVariantStock(variantID, 30)
	if err != nil {
		t.Fatalf("Failed to reserve variant stock: %v", err)
	}

	var updated ProductVariant
	db.First(&updated, "id = ?", variantID)
	if updated.Stock != 70 {
		t.Errorf("Expected stock 70, got %d", updated.Stock)
	}
	if updated.Reserved != 30 {
		t.Errorf("Expected reserved 30, got %d", updated.Reserved)
	}
}

func TestInventoryRepository_ReserveVariantStock_Insufficient(t *testing.T) {
	db := setupTestDB(t)
	repo := NewInventoryRepository(db)

	variantID := uuid.Must(uuid.NewV7())
	v := ProductVariant{
		ID:        variantID,
		ProductID: uuid.Must(uuid.NewV7()),
		SKU:       "SKU-001",
		Price:     100.00,
		Stock:     10,
		Reserved:  0,
	}
	db.Create(&v)

	err := repo.ReserveVariantStock(variantID, 50)
	if err == nil {
		t.Fatal("Expected error for insufficient stock")
	}
}

func TestInventoryRepository_ReleaseVariantStock_Success(t *testing.T) {
	db := setupTestDB(t)
	repo := NewInventoryRepository(db)

	variantID := uuid.Must(uuid.NewV7())
	v := ProductVariant{
		ID:        variantID,
		ProductID: uuid.Must(uuid.NewV7()),
		SKU:       "SKU-001",
		Price:     100.00,
		Stock:     70,
		Reserved:  30,
	}
	db.Create(&v)

	err := repo.ReleaseVariantStock(variantID, 10)
	if err != nil {
		t.Fatalf("Failed to release variant stock: %v", err)
	}

	var updated ProductVariant
	db.First(&updated, "id = ?", variantID)
	if updated.Stock != 80 {
		t.Errorf("Expected stock 80, got %d", updated.Stock)
	}
	if updated.Reserved != 20 {
		t.Errorf("Expected reserved 20, got %d", updated.Reserved)
	}
}

func TestInventoryRepository_ReleaseVariantStock_NotEnoughReserved(t *testing.T) {
	db := setupTestDB(t)
	repo := NewInventoryRepository(db)

	variantID := uuid.Must(uuid.NewV7())
	v := ProductVariant{
		ID:        variantID,
		ProductID: uuid.Must(uuid.NewV7()),
		SKU:       "SKU-001",
		Price:     100.00,
		Stock:     100,
		Reserved:  5,
	}
	db.Create(&v)

	err := repo.ReleaseVariantStock(variantID, 50)
	if err == nil {
		t.Fatal("Expected error for not enough reserved")
	}
}

func TestInventoryRepository_ReserveProductStock_Success(t *testing.T) {
	db := setupTestDB(t)
	repo := NewInventoryRepository(db)

	productID := uuid.Must(uuid.NewV7())
	v := ProductVariant{
		ID:        uuid.Must(uuid.NewV7()),
		ProductID: productID,
		SKU:       "SKU-001",
		Price:     100.00,
		Stock:     100,
		Reserved:  0,
	}
	db.Create(&v)

	err := repo.ReserveProductStock(productID, 25)
	if err != nil {
		t.Fatalf("Failed to reserve product stock: %v", err)
	}

	var updated ProductVariant
	db.First(&updated, "id = ?", v.ID)
	if updated.Stock != 75 {
		t.Errorf("Expected stock 75, got %d", updated.Stock)
	}
	if updated.Reserved != 25 {
		t.Errorf("Expected reserved 25, got %d", updated.Reserved)
	}
}

func TestInventoryRepository_ReserveProductStock_Insufficient(t *testing.T) {
	db := setupTestDB(t)
	repo := NewInventoryRepository(db)

	productID := uuid.Must(uuid.NewV7())
	v := ProductVariant{
		ID:        uuid.Must(uuid.NewV7()),
		ProductID: productID,
		SKU:       "SKU-001",
		Price:     100.00,
		Stock:     10,
		Reserved:  0,
	}
	db.Create(&v)

	err := repo.ReserveProductStock(productID, 50)
	if err == nil {
		t.Fatal("Expected error for insufficient stock")
	}
}

func TestInventoryRepository_ReleaseProductStock_Success(t *testing.T) {
	db := setupTestDB(t)
	repo := NewInventoryRepository(db)

	productID := uuid.Must(uuid.NewV7())
	v := ProductVariant{
		ID:        uuid.Must(uuid.NewV7()),
		ProductID: productID,
		SKU:       "SKU-001",
		Price:     100.00,
		Stock:     70,
		Reserved:  30,
	}
	db.Create(&v)

	err := repo.ReleaseProductStock(productID, 15)
	if err != nil {
		t.Fatalf("Failed to release product stock: %v", err)
	}

	var updated ProductVariant
	db.First(&updated, "id = ?", v.ID)
	if updated.Stock != 85 {
		t.Errorf("Expected stock 85, got %d", updated.Stock)
	}
	if updated.Reserved != 15 {
		t.Errorf("Expected reserved 15, got %d", updated.Reserved)
	}
}

func TestInventoryRepository_ReleaseProductStock_NoReserved(t *testing.T) {
	db := setupTestDB(t)
	repo := NewInventoryRepository(db)

	productID := uuid.Must(uuid.NewV7())
	v := ProductVariant{
		ID:        uuid.Must(uuid.NewV7()),
		ProductID: productID,
		SKU:       "SKU-001",
		Price:     100.00,
		Stock:     100,
		Reserved:  0,
	}
	db.Create(&v)

	err := repo.ReleaseProductStock(productID, 10)
	if err == nil {
		t.Fatal("Expected error for no reserved stock")
	}
}

func TestInventoryRepository_UpdateVariantStock_Success(t *testing.T) {
	db := setupTestDB(t)
	repo := NewInventoryRepository(db)

	productID := uuid.Must(uuid.NewV7())
	variantID := uuid.Must(uuid.NewV7())
	v := ProductVariant{
		ID:        variantID,
		ProductID: productID,
		SKU:       "SKU-001",
		Price:     100.00,
		Stock:     50,
		Reserved:  0,
	}
	db.Create(&v)

	err := repo.UpdateVariantStock(variantID, productID, 200)
	if err != nil {
		t.Fatalf("Failed to update variant stock: %v", err)
	}

	var updated ProductVariant
	db.First(&updated, "id = ?", variantID)
	if updated.Stock != 200 {
		t.Errorf("Expected stock 200, got %d", updated.Stock)
	}
}

func TestInventoryRepository_UpdateVariantStock_WrongProduct(t *testing.T) {
	db := setupTestDB(t)
	repo := NewInventoryRepository(db)

	productID := uuid.Must(uuid.NewV7())
	wrongProductID := uuid.Must(uuid.NewV7())
	variantID := uuid.Must(uuid.NewV7())
	v := ProductVariant{
		ID:        variantID,
		ProductID: productID,
		SKU:       "SKU-001",
		Price:     100.00,
		Stock:     50,
		Reserved:  0,
	}
	db.Create(&v)

	err := repo.UpdateVariantStock(variantID, wrongProductID, 200)
	if err == nil {
		t.Fatal("Expected error for variant not belonging to product")
	}
}
