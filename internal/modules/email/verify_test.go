package email

import (
	"testing"
	"time"

	"bey/internal/modules/users"

	"github.com/gofrs/uuid/v5"
)

func TestVerifyVerificationToken_Valid(t *testing.T) {
	token, err := GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	expiresAt := time.Now().Add(1 * time.Hour)
	user := &users.User{
		ID:                  uuid.Must(uuid.NewV7()),
		VerificationToken:   HashToken(token),
		VerificationExpires: &expiresAt,
	}

	result := VerifyVerificationToken(user, token)
	if !result {
		t.Error("VerifyVerificationToken() should return true for valid token")
	}
}

func TestVerifyVerificationToken_Expired(t *testing.T) {
	token, err := GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	expiresAt := time.Now().Add(-1 * time.Hour)
	user := &users.User{
		ID:                  uuid.Must(uuid.NewV7()),
		VerificationToken:   HashToken(token),
		VerificationExpires: &expiresAt,
	}

	result := VerifyVerificationToken(user, token)
	if result {
		t.Error("VerifyVerificationToken() should return false for expired token")
	}
}

func TestVerifyVerificationToken_Wrong(t *testing.T) {
	token, err := GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	expiresAt := time.Now().Add(1 * time.Hour)
	user := &users.User{
		ID:                  uuid.Must(uuid.NewV7()),
		VerificationToken:   HashToken(token),
		VerificationExpires: &expiresAt,
	}

	result := VerifyVerificationToken(user, "wrong-token")
	if result {
		t.Error("VerifyVerificationToken() should return false for wrong token")
	}
}

func TestVerifyResetToken_Valid(t *testing.T) {
	token, err := GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	expiresAt := time.Now().Add(1 * time.Hour)
	user := &users.User{
		ID:           uuid.Must(uuid.NewV7()),
		ResetToken:   HashToken(token),
		ResetExpires: &expiresAt,
	}

	result := VerifyResetToken(user, token)
	if !result {
		t.Error("VerifyResetToken() should return true for valid token")
	}
}

func TestVerifyResetToken_Expired(t *testing.T) {
	token, err := GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	expiresAt := time.Now().Add(-1 * time.Hour)
	user := &users.User{
		ID:           uuid.Must(uuid.NewV7()),
		ResetToken:   HashToken(token),
		ResetExpires: &expiresAt,
	}

	result := VerifyResetToken(user, token)
	if result {
		t.Error("VerifyResetToken() should return false for expired token")
	}
}

func TestVerifyResetToken_Wrong(t *testing.T) {
	token, err := GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	expiresAt := time.Now().Add(1 * time.Hour)
	user := &users.User{
		ID:           uuid.Must(uuid.NewV7()),
		ResetToken:   HashToken(token),
		ResetExpires: &expiresAt,
	}

	result := VerifyResetToken(user, "wrong-token")
	if result {
		t.Error("VerifyResetToken() should return false for wrong token")
	}
}

func TestVerifyVerificationToken_NilUser(t *testing.T) {
	result := VerifyVerificationToken(nil, "some-token")
	if result {
		t.Error("VerifyVerificationToken() should return false for nil user")
	}
}

func TestVerifyResetToken_NilUser(t *testing.T) {
	result := VerifyResetToken(nil, "some-token")
	if result {
		t.Error("VerifyResetToken() should return false for nil user")
	}
}
