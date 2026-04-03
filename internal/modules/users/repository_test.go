package users

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

	db.AutoMigrate(&User{})

	return db
}

// ==================== User Repository Tests ====================

func TestUserRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)

	user := &User{
		Email:     "test@example.com",
		Password:  "password123",
		FirstName: "John",
		LastName:  "Doe",
		Role:      "customer",
		Active:    true,
	}

	err := repo.Create(user)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	if user.ID == uuid.Nil {
		t.Error("Expected user ID to be set")
	}
}

func TestUserRepository_FindByID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)

	user := &User{
		Email:     "test@example.com",
		Password:  "password123",
		FirstName: "John",
		LastName:  "Doe",
		Role:      "customer",
		Active:    true,
	}
	repo.Create(user)

	found, err := repo.FindByID(user.ID)
	if err != nil {
		t.Fatalf("Failed to find user: %v", err)
	}

	if found == nil {
		t.Fatal("Expected user to be found")
	}

	if found.Email != "test@example.com" {
		t.Errorf("Expected email 'test@example.com', got '%s'", found.Email)
	}
}

func TestUserRepository_FindByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)

	found, err := repo.FindByID(uuid.Nil)
	if err != nil {
		t.Fatalf("Failed to find user: %v", err)
	}

	if found != nil {
		t.Error("Expected nil for non-existent user")
	}
}

func TestUserRepository_FindByEmail(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)

	user := &User{
		Email:     "test@example.com",
		Password:  "password123",
		FirstName: "John",
		LastName:  "Doe",
	}
	repo.Create(user)

	found, err := repo.FindByEmail("test@example.com")
	if err != nil {
		t.Fatalf("Failed to find user by email: %v", err)
	}

	if found == nil {
		t.Fatal("Expected user to be found")
	}

	if found.FirstName != "John" {
		t.Errorf("Expected first name 'John', got '%s'", found.FirstName)
	}
}

func TestUserRepository_FindByEmail_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)

	found, err := repo.FindByEmail("nonexistent@example.com")
	if err != nil {
		t.Fatalf("Failed to find user by email: %v", err)
	}

	if found != nil {
		t.Error("Expected nil for non-existent email")
	}
}

func TestUserRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)

	user := &User{
		Email:     "test@example.com",
		Password:  "password123",
		FirstName: "John",
		LastName:  "Doe",
	}
	repo.Create(user)

	user.FirstName = "Jane"
	err := repo.Update(user)
	if err != nil {
		t.Fatalf("Failed to update user: %v", err)
	}

	found, _ := repo.FindByID(user.ID)
	if found.FirstName != "Jane" {
		t.Errorf("Expected first name 'Jane', got '%s'", found.FirstName)
	}
}

func TestUserRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)

	user := &User{
		Email:     "test@example.com",
		Password:  "password123",
		FirstName: "John",
		LastName:  "Doe",
	}
	repo.Create(user)

	err := repo.Delete(user.ID)
	if err != nil {
		t.Fatalf("Failed to delete user: %v", err)
	}

	found, _ := repo.FindByID(user.ID)
	if found != nil {
		t.Error("Expected user to be nil after deletion")
	}
}

func TestUserRepository_FindAll(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)

	repo.Create(&User{Email: "user1@test.com", Password: "pass", FirstName: "User", LastName: "One"})
	repo.Create(&User{Email: "user2@test.com", Password: "pass", FirstName: "User", LastName: "Two"})
	repo.Create(&User{Email: "user3@test.com", Password: "pass", FirstName: "User", LastName: "Three"})

	users, err := repo.FindAll(0, 10)
	if err != nil {
		t.Fatalf("Failed to find all users: %v", err)
	}

	if len(users) != 3 {
		t.Errorf("Expected 3 users, got %d", len(users))
	}
}

func TestUserRepository_FindAll_WithPagination(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)

	repo.Create(&User{Email: "user1@test.com", Password: "pass", FirstName: "User", LastName: "One"})
	repo.Create(&User{Email: "user2@test.com", Password: "pass", FirstName: "User", LastName: "Two"})
	repo.Create(&User{Email: "user3@test.com", Password: "pass", FirstName: "User", LastName: "Three"})

	// Test limit
	users, err := repo.FindAll(0, 2)
	if err != nil {
		t.Fatalf("Failed to find users with limit: %v", err)
	}

	if len(users) != 2 {
		t.Errorf("Expected 2 users with limit, got %d", len(users))
	}

	// Test offset
	users, err = repo.FindAll(2, 10)
	if err != nil {
		t.Fatalf("Failed to find users with offset: %v", err)
	}

	if len(users) != 1 {
		t.Errorf("Expected 1 user with offset 2, got %d", len(users))
	}
}
