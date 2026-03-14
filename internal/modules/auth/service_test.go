package auth

import (
	"context"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"bey/internal/config"
	"bey/internal/modules/users"
)

func setupServiceTest(t *testing.T) (*AuthService, *gorm.DB) {
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

func createTestUser(t *testing.T, db *gorm.DB, email, role string, active bool) *users.User {
	t.Helper()

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	user := &users.User{
		Email:     email,
		Password:  string(hashedPassword),
		FirstName: "Test",
		LastName:  "User",
		Role:      role,
		Active:    active,
	}

	if err := db.Create(user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	return user
}

func TestLogin_Success(t *testing.T) {
	service, db := setupServiceTest(t)

	user := createTestUser(t, db, "test@example.com", "customer", true)

	resp, err := service.Login(context.Background(), "test@example.com", "password123")
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}

	if resp.AccessToken == "" {
		t.Error("expected access token to be non-empty")
	}
	if resp.RefreshToken == "" {
		t.Error("expected refresh token to be non-empty")
	}
	if resp.ExpiresIn == 0 {
		t.Error("expected expiresIn to be non-zero")
	}

	tokenGen := NewTokenGenerator(db, service.config)
	claims, err := tokenGen.ValidateToken(resp.AccessToken)
	if err != nil {
		t.Fatalf("failed to validate access token: %v", err)
	}
	if claims.UserID != user.ID {
		t.Errorf("claims.UserID = %d; want %d", claims.UserID, user.ID)
	}
}

func TestLogin_InvalidPassword(t *testing.T) {
	service, db := setupServiceTest(t)

	createTestUser(t, db, "test@example.com", "customer", true)

	_, err := service.Login(context.Background(), "test@example.com", "wrongpassword")
	if err == nil {
		t.Error("expected error for invalid password, got nil")
	}
}

func TestLogin_UserNotFound(t *testing.T) {
	service, _ := setupServiceTest(t)

	_, err := service.Login(context.Background(), "nonexistent@example.com", "password123")
	if err == nil {
		t.Error("expected error for non-existent user, got nil")
	}
}

func TestLogin_InactiveUser(t *testing.T) {
	service, db := setupServiceTest(t)

	user := createTestUser(t, db, "test@example.com", "customer", false)

	user.Active = false
	db.Save(user)

	_, err := service.Login(context.Background(), "test@example.com", "password123")
	if err == nil {
		t.Error("expected error for inactive user, got nil")
	}

	_ = user
}

func TestRefresh_Success(t *testing.T) {
	service, db := setupServiceTest(t)

	user := createTestUser(t, db, "test@example.com", "customer", true)

	loginResp, err := service.Login(context.Background(), "test@example.com", "password123")
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}

	refreshResp, err := service.Refresh(context.Background(), loginResp.RefreshToken)
	if err != nil {
		t.Fatalf("Refresh failed: %v", err)
	}

	if refreshResp.AccessToken == "" {
		t.Error("expected new access token to be non-empty")
	}
	if refreshResp.RefreshToken == "" {
		t.Error("expected new refresh token to be non-empty")
	}

	if refreshResp.RefreshToken == loginResp.RefreshToken {
		t.Error("expected new refresh token to be different from old one")
	}

	tokenGen := NewTokenGenerator(db, service.config)
	claims, err := tokenGen.ValidateToken(refreshResp.AccessToken)
	if err != nil {
		t.Fatalf("failed to validate new access token: %v", err)
	}
	if claims.UserID != user.ID {
		t.Errorf("claims.UserID = %d; want %d", claims.UserID, user.ID)
	}
	if claims.Email != user.Email {
		t.Errorf("claims.Email = %s; want %s", claims.Email, user.Email)
	}
	if claims.Role != user.Role {
		t.Errorf("claims.Role = %s; want %s", claims.Role, user.Role)
	}
}

func TestRefresh_ExpiredToken(t *testing.T) {
	service, db := setupServiceTest(t)

	createTestUser(t, db, "test@example.com", "customer", true)

	tokenGen := NewTokenGenerator(db, service.config)
	refreshToken, _ := tokenGen.GenerateRefreshToken()

	expiredToken := &RefreshToken{
		Token:     tokenGen.HashToken(refreshToken),
		UserID:    1,
		ExpiresAt: time.Now().Add(-1 * time.Hour),
		Revoked:   false,
	}
	db.Create(expiredToken)

	_, err := service.Refresh(context.Background(), refreshToken)
	if err == nil {
		t.Error("expected error for expired refresh token, got nil")
	}
}

func TestRefresh_RevokedToken(t *testing.T) {
	service, db := setupServiceTest(t)

	user := createTestUser(t, db, "test@example.com", "customer", true)

	loginResp, err := service.Login(context.Background(), "test@example.com", "password123")
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}

	service.Logout(context.Background(), loginResp.RefreshToken)

	_, err = service.Refresh(context.Background(), loginResp.RefreshToken)
	if err == nil {
		t.Error("expected error for revoked refresh token, got nil")
	}

	_ = user
}

func TestLogout_Success(t *testing.T) {
	service, db := setupServiceTest(t)

	createTestUser(t, db, "test@example.com", "customer", true)

	loginResp, err := service.Login(context.Background(), "test@example.com", "password123")
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}

	err = service.Logout(context.Background(), loginResp.RefreshToken)
	if err != nil {
		t.Fatalf("Logout failed: %v", err)
	}

	var token RefreshToken
	if err := db.Where("user_id = ?", 1).First(&token).Error; err != nil {
		t.Fatalf("failed to find refresh token: %v", err)
	}
	if !token.Revoked {
		t.Error("expected token to be revoked")
	}

	_, err = service.Refresh(context.Background(), loginResp.RefreshToken)
	if err == nil {
		t.Error("expected error after logout when using revoked token")
	}
}
