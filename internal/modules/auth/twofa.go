package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/hex"
	"encoding/json"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"github.com/skip2/go-qrcode"
)

type TwoFAService struct{}

func NewTwoFAService() *TwoFAService {
	return &TwoFAService{}
}

func (s *TwoFAService) GenerateSecret(email string) (string, error) {
	accountName := email
	if accountName == "" {
		accountName = "user"
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "Bey API",
		AccountName: accountName,
		Algorithm:   otp.AlgorithmSHA1,
		Digits:      otp.DigitsSix,
		Period:      30,
	})
	if err != nil {
		return "", err
	}
	return key.Secret(), nil
}

func (s *TwoFAService) GenerateQRCode(secret, email string) ([]byte, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "Bey API",
		AccountName: email,
		Algorithm:   otp.AlgorithmSHA1,
		Digits:      otp.DigitsSix,
		Period:      30,
		Secret:      []byte(secret),
	})
	if err != nil {
		return nil, err
	}

	url := key.URL()

	png, err := qrcode.Encode(url, qrcode.Medium, 256)
	if err != nil {
		return nil, err
	}

	return png, nil
}

func (s *TwoFAService) VerifyCode(secret, code string) bool {
	return totp.Validate(code, secret)
}

func (s *TwoFAService) GenerateBackupCodes(count int) ([]string, error) {
	if count <= 0 {
		count = 10
	}

	codes := make([]string, count)
	for i := 0; i < count; i++ {
		code, err := generateReadableCode()
		if err != nil {
			return nil, err
		}
		codes[i] = code
	}

	return codes, nil
}

func generateReadableCode() (string, error) {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	encoded := base32.StdEncoding.EncodeToString(bytes)
	clean := removePadding(encoded)

	formatted := formatBackupCode(clean)
	return formatted, nil
}

func removePadding(s string) string {
	result := make([]byte, 0, len(s))
	for _, b := range []byte(s) {
		if b != '=' {
			result = append(result, b)
		}
	}
	return string(result[:10])
}

func formatBackupCode(s string) string {
	if len(s) >= 10 {
		return s[:5] + "-" + s[5:10]
	}
	return s
}

func (s *TwoFAService) HashBackupCode(code string) string {
	cleanCode := removeSpecialChars(code)
	hash := sha256.Sum256([]byte(cleanCode))
	return hex.EncodeToString(hash[:])
}

func removeSpecialChars(s string) string {
	result := make([]byte, 0, len(s))
	for _, c := range s {
		if c >= '0' && c <= '9' || c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' {
			result = append(result, byte(c))
		}
	}
	return string(result)
}

func (s *TwoFAService) VerifyBackupCode(storedHashes []string, code string) bool {
	inputHash := s.HashBackupCode(code)
	for _, storedHash := range storedHashes {
		if inputHash == storedHash {
			return true
		}
	}
	return false
}

func (s *TwoFAService) SerializeBackupCodes(codes []string) (string, error) {
	data, err := json.Marshal(codes)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (s *TwoFAService) DeserializeBackupCodes(data string) ([]string, error) {
	var codes []string
	if err := json.Unmarshal([]byte(data), &codes); err != nil {
		return nil, err
	}
	return codes, nil
}

func (s *TwoFAService) RemoveUsedBackupCode(storedHashes []string, code string) []string {
	inputHash := s.HashBackupCode(code)
	result := make([]string, 0, len(storedHashes))
	for _, hash := range storedHashes {
		if hash != inputHash {
			result = append(result, hash)
		}
	}
	return result
}
