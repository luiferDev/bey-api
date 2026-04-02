package users

import (
	"errors"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupCreatorTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	if err := db.AutoMigrate(&User{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}
	return db
}

func TestRegularUserCreator_SetsCustomerRole(t *testing.T) {
	db := setupCreatorTestDB(t)
	creator := NewRegularUserCreator(db)

	req := &CreateUserRequest{
		Email:    "customer@test.com",
		Password: "Password123",
		Name:     "Test Customer",
	}

	user, err := creator.Create(req)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if user.Role != "customer" {
		t.Errorf("Role = %q; want %q", user.Role, "customer")
	}
}

func TestAdminUserCreator_SetsAdminRole(t *testing.T) {
	db := setupCreatorTestDB(t)
	creator := NewAdminUserCreator(db)

	req := &CreateUserRequest{
		Email:    "admin@test.com",
		Password: "Password123",
		Name:     "Test Admin",
	}

	user, err := creator.Create(req)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if user.Role != "admin" {
		t.Errorf("Role = %q; want %q", user.Role, "admin")
	}
}

func TestCreator_ValidatesEmail(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		wantErr error
	}{
		{
			name:    "valid email",
			email:   "valid@test.com",
			wantErr: nil,
		},
		{
			name:    "empty email returns error",
			email:   "",
			wantErr: errors.New("email is required"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupCreatorTestDB(t)
			creator := NewRegularUserCreator(db)

			req := &CreateUserRequest{
				Email:    tt.email,
				Password: "Password123",
				Name:     "Test User",
			}

			_, err := creator.Create(req)

			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				return
			}

			if err == nil {
				t.Error("expected error, got nil")
				return
			}

			if err.Error() != tt.wantErr.Error() {
				t.Errorf("error = %q; want %q", err.Error(), tt.wantErr.Error())
			}
		})
	}
}

func TestCreator_HashesPassword(t *testing.T) {
	db := setupCreatorTestDB(t)
	creator := NewRegularUserCreator(db)

	plainPassword := "Password123"
	req := &CreateUserRequest{
		Email:    "hashtest@test.com",
		Password: plainPassword,
		Name:     "Test User",
	}

	user, err := creator.Create(req)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if user.Password == plainPassword {
		t.Error("Password should be hashed, but was stored in plain text")
	}

	if len(user.Password) == 0 {
		t.Error("Password should not be empty after hashing")
	}
}
