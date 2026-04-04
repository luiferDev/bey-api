package inventory

import (
	"testing"

	"github.com/gofrs/uuid/v5"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	db.AutoMigrate(&Inventory{})

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
	// Quantity should decrease: 100 - 30 = 70
	if found.Quantity != 70 {
		t.Errorf("Expected quantity 70, got %d", found.Quantity)
	}
	// Reserved should increase: 0 + 30 = 30
	if found.Reserved != 30 {
		t.Errorf("Expected reserved 30, got %d", found.Reserved)
	}
}
