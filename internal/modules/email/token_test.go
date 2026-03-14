package email

import (
	"testing"
)

func TestGenerateToken_Length(t *testing.T) {
	token, err := GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	if len(token) != 64 {
		t.Errorf("GenerateToken() length = %d; want 64", len(token))
	}
}

func TestGenerateToken_Unique(t *testing.T) {
	token1, err := GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	token2, err := GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	if token1 == token2 {
		t.Error("GenerateToken() should return unique tokens")
	}
}

func TestHashToken_Consistent(t *testing.T) {
	input := "test-token-123"
	hash1 := HashToken(input)
	hash2 := HashToken(input)

	if hash1 != hash2 {
		t.Error("HashToken() should return consistent output for same input")
	}

	if hash1 == input {
		t.Error("HashToken() should not return the same value as input")
	}
}
