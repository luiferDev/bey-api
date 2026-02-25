package orders

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

	db.AutoMigrate(&Order{}, &OrderItem{})

	return db
}

// ==================== Order Repository Tests ====================

func TestOrderRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := NewOrderRepository(db)

	order := &Order{
		UserID:          1,
		Status:          "pending",
		TotalPrice:      99.99,
		ShippingAddress: "123 Main St",
		Notes:           "Please leave at door",
		Items: []OrderItem{
			{ProductID: 1, Quantity: 2, UnitPrice: 49.99},
		},
	}

	err := repo.Create(order)
	if err != nil {
		t.Fatalf("Failed to create order: %v", err)
	}

	if order.ID == 0 {
		t.Error("Expected order ID to be set")
	}

	if len(order.Items) != 1 {
		t.Errorf("Expected 1 order item, got %d", len(order.Items))
	}
}

func TestOrderRepository_FindByID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewOrderRepository(db)

	order := &Order{
		UserID:          1,
		Status:          "pending",
		TotalPrice:      99.99,
		ShippingAddress: "123 Main St",
		Items: []OrderItem{
			{ProductID: 1, Quantity: 2, UnitPrice: 49.99},
		},
	}
	repo.Create(order)

	found, err := repo.FindByID(order.ID)
	if err != nil {
		t.Fatalf("Failed to find order: %v", err)
	}

	if found == nil {
		t.Fatal("Expected order to be found")
	}

	if found.Status != "pending" {
		t.Errorf("Expected status 'pending', got '%s'", found.Status)
	}

	if len(found.Items) != 1 {
		t.Errorf("Expected 1 item, got %d", len(found.Items))
	}
}

func TestOrderRepository_FindByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewOrderRepository(db)

	found, err := repo.FindByID(999)
	if err != nil {
		t.Fatalf("Failed to find order: %v", err)
	}

	if found != nil {
		t.Error("Expected nil for non-existent order")
	}
}

func TestOrderRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	repo := NewOrderRepository(db)

	order := &Order{
		UserID:     1,
		Status:     "pending",
		TotalPrice: 99.99,
	}
	repo.Create(order)

	order.Status = "confirmed"
	err := repo.Update(order)
	if err != nil {
		t.Fatalf("Failed to update order: %v", err)
	}

	found, _ := repo.FindByID(order.ID)
	if found.Status != "confirmed" {
		t.Errorf("Expected status 'confirmed', got '%s'", found.Status)
	}
}

func TestOrderRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	repo := NewOrderRepository(db)

	order := &Order{
		UserID:     1,
		Status:     "pending",
		TotalPrice: 99.99,
	}
	repo.Create(order)

	err := repo.Delete(order.ID)
	if err != nil {
		t.Fatalf("Failed to delete order: %v", err)
	}

	found, _ := repo.FindByID(order.ID)
	if found != nil {
		t.Error("Expected order to be nil after deletion")
	}
}

func TestOrderRepository_FindByUserID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewOrderRepository(db)

	repo.Create(&Order{UserID: 1, Status: "pending", TotalPrice: 50})
	repo.Create(&Order{UserID: 1, Status: "confirmed", TotalPrice: 75})
	repo.Create(&Order{UserID: 2, Status: "pending", TotalPrice: 100})

	orders, err := repo.FindByUserID(1)
	if err != nil {
		t.Fatalf("Failed to find orders by user ID: %v", err)
	}

	if len(orders) != 2 {
		t.Errorf("Expected 2 orders for user 1, got %d", len(orders))
	}
}

func TestOrderRepository_FindAll(t *testing.T) {
	db := setupTestDB(t)
	repo := NewOrderRepository(db)

	repo.Create(&Order{UserID: 1, Status: "pending", TotalPrice: 50})
	repo.Create(&Order{UserID: 2, Status: "confirmed", TotalPrice: 75})
	repo.Create(&Order{UserID: 3, Status: "pending", TotalPrice: 100})

	orders, err := repo.FindAll(0, 10)
	if err != nil {
		t.Fatalf("Failed to find all orders: %v", err)
	}

	if len(orders) != 3 {
		t.Errorf("Expected 3 orders, got %d", len(orders))
	}
}

func TestOrderRepository_FindAll_WithPagination(t *testing.T) {
	db := setupTestDB(t)
	repo := NewOrderRepository(db)

	repo.Create(&Order{UserID: 1, Status: "pending", TotalPrice: 50})
	repo.Create(&Order{UserID: 2, Status: "confirmed", TotalPrice: 75})
	repo.Create(&Order{UserID: 3, Status: "pending", TotalPrice: 100})

	// Test limit
	orders, err := repo.FindAll(0, 2)
	if err != nil {
		t.Fatalf("Failed to find orders with limit: %v", err)
	}

	if len(orders) != 2 {
		t.Errorf("Expected 2 orders with limit, got %d", len(orders))
	}

	// Test offset
	orders, err = repo.FindAll(2, 10)
	if err != nil {
		t.Fatalf("Failed to find orders with offset: %v", err)
	}

	if len(orders) != 1 {
		t.Errorf("Expected 1 order with offset 2, got %d", len(orders))
	}
}
