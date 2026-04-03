package auth

import (
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"bey/internal/config"
)

func setupTokenTest(t *testing.T) (*TokenGenerator, *gorm.DB) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
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

	tokenGen := NewTokenGenerator(db, cfg)
	return tokenGen, db
}

func TestGenerateAccessToken(t *testing.T) {
	tokenGen, _ := setupTokenTest(t)

	token, expiresIn, err := tokenGen.GenerateAccessToken(uuid.Must(uuid.NewV7()), "test@example.com", "customer")
	if err != nil {
		t.Fatalf("GenerateAccessToken failed: %v", err)
	}

	if token == "" {
		t.Error("expected token to be non-empty")
	}

	if expiresIn == 0 {
		t.Error("expected expiresIn to be non-zero")
	}

	if expiresIn != 900 {
		t.Errorf("expiresIn = %d; want 900 (15 minutes)", expiresIn)
	}
}

func TestGenerateAccessToken_Expiry(t *testing.T) {
	tokenGen, _ := setupTokenTest(t)

	testUUID := uuid.Must(uuid.NewV7())
	token, expiresIn, err := tokenGen.GenerateAccessToken(testUUID, "test@example.com", "admin")
	if err != nil {
		t.Fatalf("GenerateAccessToken failed: %v", err)
	}

	expectedExpiry := int64(15 * 60)
	if expiresIn != expectedExpiry {
		t.Errorf("expiresIn = %d; want %d", expiresIn, expectedExpiry)
	}

	claims, err := tokenGen.ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}

	if claims.UserID != testUUID.String() {
		t.Errorf("claims.UserID = %s; want %s", claims.UserID, testUUID.String())
	}
	if claims.Email != "test@example.com" {
		t.Errorf("claims.Email = %s; want test@example.com", claims.Email)
	}
	if claims.Role != "admin" {
		t.Errorf("claims.Role = %s; want admin", claims.Role)
	}
}

func TestGenerateRefreshToken(t *testing.T) {
	tokenGen, _ := setupTokenTest(t)

	tests := []struct {
		name string
	}{
		{"generate multiple tokens"},
		{"unique tokens"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token1, err := tokenGen.GenerateRefreshToken()
			if err != nil {
				t.Fatalf("GenerateRefreshToken failed: %v", err)
			}

			if len(token1) != 64 {
				t.Errorf("token length = %d; want 64", len(token1))
			}

			token2, err := tokenGen.GenerateRefreshToken()
			if err != nil {
				t.Fatalf("GenerateRefreshToken failed: %v", err)
			}

			if token1 == token2 {
				t.Error("expected tokens to be unique")
			}
		})
	}
}

func TestValidateToken(t *testing.T) {
	tokenGen, _ := setupTokenTest(t)

	testUUID := uuid.Must(uuid.NewV7())
	token, _, err := tokenGen.GenerateAccessToken(testUUID, "user@test.com", "customer")
	if err != nil {
		t.Fatalf("GenerateAccessToken failed: %v", err)
	}

	claims, err := tokenGen.ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}

	if claims.UserID != testUUID.String() {
		t.Errorf("UserID = %s; want %s", claims.UserID, testUUID.String())
	}
	if claims.Email != "user@test.com" {
		t.Errorf("Email = %s; want user@test.com", claims.Email)
	}
	if claims.Role != "customer" {
		t.Errorf("Role = %s; want customer", claims.Role)
	}
}

func TestValidateToken_Expired(t *testing.T) {
	tokenGen, _ := setupTokenTest(t)

	cfg := &config.Config{
		Security: config.SecurityConfig{
			JWTSecret:       "test-secret-key",
			JWTAccessExpiry: -1 * time.Minute,
			JWTIssuer:       "bey_api_test",
		},
	}
	cfg.Security.JWTConfig = cfg.Security.GetJWTConfig()

	expiredTokenGen := NewTokenGenerator(tokenGen.db, cfg)
	token, _, err := expiredTokenGen.GenerateAccessToken(uuid.Must(uuid.NewV7()), "test@example.com", "customer")
	if err != nil {
		t.Fatalf("GenerateAccessToken failed: %v", err)
	}

	_, err = tokenGen.ValidateToken(token)
	if err == nil {
		t.Error("expected error for expired token, got nil")
	}
}

func TestValidateToken_Invalid(t *testing.T) {
	tokenGen, _ := setupTokenTest(t)

	tests := []struct {
		name  string
		token string
	}{
		{"empty token", ""},
		{"invalid token", "invalid.token.string"},
		{"random garbage", "abc123xyz"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tokenGen.ValidateToken(tt.token)
			if err == nil {
				t.Error("expected error for invalid token, got nil")
			}
		})
	}
}

func TestHashToken(t *testing.T) {
	tokenGen, _ := setupTokenTest(t)

	tests := []struct {
		name     string
		token    string
		wantLen  int
		wantSame bool
	}{
		{"consistent hashing", "test-token-123", 64, true},
		{"empty token", "", 64, true},
		{"long token", "very-long-token-string-that-should-be-hashed-correctly", 64, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash1 := tokenGen.HashToken(tt.token)
			hash2 := tokenGen.HashToken(tt.token)

			if len(hash1) != tt.wantLen {
				t.Errorf("hash length = %d; want %d", len(hash1), tt.wantLen)
			}

			if tt.wantSame && hash1 != hash2 {
				t.Error("expected same hash for same input")
			}
		})
	}

	differentHash := tokenGen.HashToken("different-token")
	if sameHash := tokenGen.HashToken("test-token-123"); differentHash == sameHash {
		t.Error("expected different hashes for different tokens")
	}
}
