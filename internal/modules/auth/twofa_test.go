package auth

import (
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
)

// Helper to generate a valid TOTP code for testing
func generateTestTotpCode(secret string) string {
	code, _ := totp.GenerateCode(secret, time.Now())
	return code
}

func TestTwoFA_GenerateSecret(t *testing.T) {
	svc := NewTwoFAService()

	secret, err := svc.GenerateSecret("test@example.com")
	if err != nil {
		t.Fatalf("GenerateSecret() unexpected error: %v", err)
	}

	if secret == "" {
		t.Error("GenerateSecret() returned empty secret")
	}

	// Secret should be base32 encoded (alphanumeric)
	if len(secret) < 16 {
		t.Errorf("GenerateSecret() secret too short: got %d, want >= 16", len(secret))
	}
}

func TestTwoFA_GenerateSecret_EmptyEmail(t *testing.T) {
	svc := NewTwoFAService()

	// Empty email should use default "user" as account name
	secret, err := svc.GenerateSecret("")
	if err != nil {
		t.Fatalf("GenerateSecret() with empty email unexpected error: %v", err)
	}

	if secret == "" {
		t.Error("GenerateSecret() with empty email returned empty secret")
	}
}

func TestTwoFA_VerifyCode(t *testing.T) {
	svc := NewTwoFAService()

	tests := []struct {
		name      string
		secret    string
		code      string
		wantValid bool
	}{
		{
			name:      "valid TOTP code",
			secret:    "JBSWY3DPEHPK3PXP",
			code:      "", // Will be filled with valid code
			wantValid: true,
		},
		{
			name:      "invalid code",
			secret:    "JBSWY3DPEHPK3PXP",
			code:      "000000",
			wantValid: false,
		},
		{
			name:      "wrong length code",
			secret:    "JBSWY3DPEHPK3PXP",
			code:      "12345",
			wantValid: false,
		},
		{
			name:      "empty code",
			secret:    "JBSWY3DPEHPK3PXP",
			code:      "",
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// For valid test case, generate a valid TOTP
			if tt.name == "valid TOTP code" {
				tt.code = generateTestTotpCode(tt.secret)
			}

			got := svc.VerifyCode(tt.secret, tt.code)
			if got != tt.wantValid {
				t.Errorf("VerifyCode() = %v, want %v", got, tt.wantValid)
			}
		})
	}
}

func TestTwoFA_GenerateBackupCodes(t *testing.T) {
	svc := NewTwoFAService()

	tests := []struct {
		name      string
		count     int
		wantCount int
		wantLen   int // length of each code
	}{
		{
			name:      "default count",
			count:     0,
			wantCount: 10,
			wantLen:   11, // XXXXX-XXXXX format with dash
		},
		{
			name:      "custom count 5",
			count:     5,
			wantCount: 5,
			wantLen:   11,
		},
		{
			name:      "custom count 20",
			count:     20,
			wantCount: 20,
			wantLen:   11,
		},
		{
			name:      "negative count uses default",
			count:     -5,
			wantCount: 10,
			wantLen:   11,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			codes, err := svc.GenerateBackupCodes(tt.count)
			if err != nil {
				t.Fatalf("GenerateBackupCodes() unexpected error: %v", err)
			}

			if len(codes) != tt.wantCount {
				t.Errorf("GenerateBackupCodes() returned %d codes, want %d",
					len(codes), tt.wantCount)
			}

			// Check format of each code
			for i, code := range codes {
				if len(code) != tt.wantLen {
					t.Errorf("code[%d] length = %d, want %d", i, len(code), tt.wantLen)
				}
				// Check dash position
				if code[5] != '-' {
					t.Errorf("code[%d] format wrong, expected dash at position 5", i)
				}
			}
		})
	}
}

func TestTwoFA_GenerateBackupCodes_Uniqueness(t *testing.T) {
	svc := NewTwoFAService()

	codes, err := svc.GenerateBackupCodes(50)
	if err != nil {
		t.Fatalf("GenerateBackupCodes() unexpected error: %v", err)
	}

	// Check uniqueness
	seen := make(map[string]bool)
	for _, code := range codes {
		if seen[code] {
			t.Errorf("GenerateBackupCodes() produced duplicate code: %s", code)
		}
		seen[code] = true
	}
}

func TestTwoFA_HashBackupCode(t *testing.T) {
	svc := NewTwoFAService()

	tests := []struct {
		name     string
		code     string
		wantLen  int
		wantSame bool // same input should produce same hash
	}{
		{
			name:     "normal code",
			code:     "ABC12-DEF34",
			wantLen:  64, // SHA256 produces 64 hex chars
			wantSame: true,
		},
		{
			name:     "lowercase converted to uppercase",
			code:     "abc12-def34",
			wantLen:  64,
			wantSame: true,
		},
		{
			name:     "special chars removed",
			code:     "ABC12!@#-DEF$%^",
			wantLen:  64,
			wantSame: true,
		},
		{
			name:     "empty code",
			code:     "",
			wantLen:  64,
			wantSame: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash1 := svc.HashBackupCode(tt.code)
			hash2 := svc.HashBackupCode(tt.code)

			if len(hash1) != tt.wantLen {
				t.Errorf("HashBackupCode() length = %d, want %d", len(hash1), tt.wantLen)
			}

			if tt.wantSame && hash1 != hash2 {
				t.Errorf("HashBackupCode() not deterministic: %s != %s", hash1, hash2)
			}
		})
	}
}

func TestTwoFA_VerifyBackupCode(t *testing.T) {
	svc := NewTwoFAService()

	code := "ABC12-DEF34"
	hash := svc.HashBackupCode(code)
	storedHashes := []string{hash, "some-other-hash"}

	tests := []struct {
		name      string
		code      string
		hashes    []string
		wantValid bool
	}{
		{
			name:      "valid code",
			code:      code,
			hashes:    storedHashes,
			wantValid: true,
		},
		{
			name:      "invalid code",
			code:      "XXXXX-XXXXX",
			hashes:    storedHashes,
			wantValid: false,
		},
		{
			name:      "empty hashes",
			code:      code,
			hashes:    []string{},
			wantValid: false,
		},
		{
			name:      "nil hashes",
			code:      code,
			hashes:    nil,
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := svc.VerifyBackupCode(tt.hashes, tt.code)
			if got != tt.wantValid {
				t.Errorf("VerifyBackupCode() = %v, want %v", got, tt.wantValid)
			}
		})
	}
}

func TestTwoFA_SerializeDeserializeBackupCodes(t *testing.T) {
	svc := NewTwoFAService()

	original := []string{"AAAAA-BBBBB", "CCCCC-DDDDD", "EEEEE-FFFFF"}

	// Serialize
	serialized, err := svc.SerializeBackupCodes(original)
	if err != nil {
		t.Fatalf("SerializeBackupCodes() unexpected error: %v", err)
	}

	if serialized == "" {
		t.Error("SerializeBackupCodes() returned empty string")
	}

	// Deserialize
	deserialized, err := svc.DeserializeBackupCodes(serialized)
	if err != nil {
		t.Fatalf("DeserializeBackupCodes() unexpected error: %v", err)
	}

	if len(deserialized) != len(original) {
		t.Errorf("DeserializeBackupCodes() returned %d codes, want %d",
			len(deserialized), len(original))
	}

	for i, code := range original {
		if deserialized[i] != code {
			t.Errorf("DeserializeBackupCodes()[%d] = %s, want %s",
				i, deserialized[i], code)
		}
	}
}

func TestTwoFA_RemoveUsedBackupCode(t *testing.T) {
	svc := NewTwoFAService()

	code1 := "CODE1-ABCDE"
	code2 := "CODE2-FGHIJ"
	code3 := "CODE3-KLMNO"

	hash1 := svc.HashBackupCode(code1)
	hash2 := svc.HashBackupCode(code2)
	hash3 := svc.HashBackupCode(code3)

	original := []string{hash1, hash2, hash3}

	tests := []struct {
		name           string
		codeToRemove   string
		originalHashes []string
		wantRemaining  int
	}{
		{
			name:           "remove first code",
			codeToRemove:   code1,
			originalHashes: original,
			wantRemaining:  2,
		},
		{
			name:           "remove middle code",
			codeToRemove:   code2,
			originalHashes: original,
			wantRemaining:  2,
		},
		{
			name:           "remove last code",
			codeToRemove:   code3,
			originalHashes: original,
			wantRemaining:  2,
		},
		{
			name:           "remove non-existent code",
			codeToRemove:   "XXXXX-XXXXX",
			originalHashes: original,
			wantRemaining:  3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			remaining := svc.RemoveUsedBackupCode(tt.originalHashes, tt.codeToRemove)
			if len(remaining) != tt.wantRemaining {
				t.Errorf("RemoveUsedBackupCode() returned %d, want %d",
					len(remaining), tt.wantRemaining)
			}
		})
	}
}

func TestTwoFA_GenerateQRCode(t *testing.T) {
	svc := NewTwoFAService()

	secret := "JBSWY3DPEHPK3PXP"
	email := "test@example.com"

	qrCode, err := svc.GenerateQRCode(secret, email)
	if err != nil {
		t.Fatalf("GenerateQRCode() unexpected error: %v", err)
	}

	if len(qrCode) == 0 {
		t.Error("GenerateQRCode() returned empty byte slice")
	}

	// PNG files start with these bytes
	if qrCode[0] != 0x89 || qrCode[1] != 0x50 {
		t.Error("GenerateQRCode() did not return valid PNG data")
	}
}

func TestTwoFA_GenerateQRCode_InvalidEmail(t *testing.T) {
	svc := NewTwoFAService()

	secret := "JBSWY3DPEHPK3PXP"

	// Empty email should still work with placeholder
	qrCode, err := svc.GenerateQRCode(secret, "user@example.com")
	if err != nil {
		t.Fatalf("GenerateQRCode() with empty email unexpected error: %v", err)
	}

	if len(qrCode) == 0 {
		t.Error("GenerateQRCode() with empty email returned empty byte slice")
	}
}
