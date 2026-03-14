package auth

import (
	"context"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"bey/internal/config"
	"bey/internal/modules/email"
	"bey/internal/modules/users"
)

func setupPasswordResetTest(t *testing.T) (*AuthService, *gorm.DB) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	if err := db.AutoMigrate(&users.User{}, &RefreshToken{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	cfg := &config.Config{
		Security: config.SecurityConfig{
			JWTSecret:        "test-secret-key-for-jwt-token-generation",
			JWTAccessExpiry:  15 * time.Minute,
			JWTRefreshExpiry: 7 * 24 * time.Hour,
			JWTIssuer:        "bey_api_test",
		},
	}
	cfg.Security.JWTConfig = cfg.Security.GetJWTConfig()

	service := NewAuthService(db, cfg)
	return service, db
}

func createTestUserWithResetToken(t *testing.T, db *gorm.DB, emailAddr string) (string, *users.User) {
	t.Helper()

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	token, err := email.GenerateToken()
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	expiresAt := time.Now().Add(1 * time.Hour)

	user := &users.User{
		Email:        emailAddr,
		Password:     string(hashedPassword),
		FirstName:    "Test",
		LastName:     "User",
		Role:         "customer",
		Active:       true,
		ResetToken:   email.HashToken(token),
		ResetExpires: &expiresAt,
	}

	if err := db.Create(user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	return token, user
}

func TestForgotPassword_Success(t *testing.T) {
	service, db := setupPasswordResetTest(t)

	user := createTestUser(t, db, "test@example.com", "customer", true)

	err := service.ForgotPassword(context.Background(), user.Email)
	if err != nil {
		t.Fatalf("ForgotPassword failed: %v", err)
	}

	var updatedUser users.User
	if err := db.First(&updatedUser, user.ID).Error; err != nil {
		t.Fatalf("failed to find user: %v", err)
	}

	if updatedUser.ResetToken == "" {
		t.Error("expected reset token to be set")
	}
	if updatedUser.ResetExpires == nil {
		t.Error("expected reset expires to be set")
	}
}

func TestForgotPassword_NonExistent(t *testing.T) {
	service, _ := setupPasswordResetTest(t)

	err := service.ForgotPassword(context.Background(), "nonexistent@example.com")
	if err != nil {
		t.Fatalf("ForgotPassword failed: %v", err)
	}

	// Should return nil for non-existent user (no enumeration)
	// The function should succeed silently
}

func TestResetPassword_ValidToken(t *testing.T) {
	service, db := setupPasswordResetTest(t)

	token, user := createTestUserWithResetToken(t, db, "test@example.com")

	newPassword := "newpassword123"
	err := service.ResetPassword(context.Background(), token, newPassword)
	if err != nil {
		t.Fatalf("ResetPassword failed: %v", err)
	}

	var updatedUser users.User
	if err := db.First(&updatedUser, user.ID).Error; err != nil {
		t.Fatalf("failed to find user: %v", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(updatedUser.Password), []byte(newPassword)); err != nil {
		t.Errorf("password should be updated: %v", err)
	}

	if updatedUser.ResetToken != "" {
		t.Error("expected reset token to be cleared")
	}
	if updatedUser.ResetExpires != nil {
		t.Error("expected reset expires to be cleared")
	}
}

func TestResetPassword_ExpiredToken(t *testing.T) {
	service, db := setupPasswordResetTest(t)

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	expiredAt := time.Now().Add(-1 * time.Hour)
	token, _ := email.GenerateToken()

	user := &users.User{
		Email:        "test@example.com",
		Password:     string(hashedPassword),
		FirstName:    "Test",
		LastName:     "User",
		Role:         "customer",
		Active:       true,
		ResetToken:   email.HashToken(token),
		ResetExpires: &expiredAt,
	}

	if err := db.Create(user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	err = service.ResetPassword(context.Background(), token, "newpassword123")
	if err == nil {
		t.Error("expected error for expired token, got nil")
	}
}

func TestResetPassword_WrongToken(t *testing.T) {
	service, db := setupPasswordResetTest(t)

	_, user := createTestUserWithResetToken(t, db, "test@example.com")

	err := service.ResetPassword(context.Background(), "wrong-token", "newpassword123")
	if err == nil {
		t.Error("expected error for wrong token, got nil")
	}

	_ = user
}
