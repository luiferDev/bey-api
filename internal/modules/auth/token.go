package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"

	"bey/internal/config"
)

type TokenGenerator struct {
	db     *gorm.DB
	config *config.Config
}

func NewTokenGenerator(db *gorm.DB, config *config.Config) *TokenGenerator {
	return &TokenGenerator{
		db:     db,
		config: config,
	}
}

func (g *TokenGenerator) GenerateAccessToken(userID uint, email, role string) (string, int64, error) {
	jwtConfig := g.config.Security.GetJWTConfig()
	expiry := jwtConfig.AccessExpiry

	claims := TokenClaims{
		UserID: userID,
		Email:  email,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    jwtConfig.Issuer,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(jwtConfig.SecretKey))
	if err != nil {
		return "", 0, err
	}

	return tokenString, int64(expiry.Seconds()), nil
}

func (g *TokenGenerator) GenerateRefreshToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func (g *TokenGenerator) ValidateToken(tokenString string) (*TokenClaims, error) {
	jwtConfig := g.config.Security.GetJWTConfig()

	token, err := jwt.ParseWithClaims(tokenString, &TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(jwtConfig.SecretKey), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*TokenClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

func (g *TokenGenerator) HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func (g *TokenGenerator) RotateRefreshToken(oldTokenHash string) (string, error) {
	var refreshToken RefreshToken
	if err := g.db.Where("token = ? AND revoked = ?", oldTokenHash, false).First(&refreshToken).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", errors.New("refresh token not found")
		}
		return "", err
	}

	if time.Now().After(refreshToken.ExpiresAt) {
		return "", errors.New("refresh token expired")
	}

	refreshToken.Revoked = true
	if err := g.db.Save(&refreshToken).Error; err != nil {
		return "", err
	}

	newToken, err := g.GenerateRefreshToken()
	if err != nil {
		return "", err
	}

	return newToken, nil
}

func (g *TokenGenerator) StoreRefreshToken(token string, userID uint) error {
	jwtConfig := g.config.Security.GetJWTConfig()
	tokenHash := g.HashToken(token)

	refreshToken := &RefreshToken{
		Token:     tokenHash,
		UserID:    userID,
		ExpiresAt: time.Now().Add(jwtConfig.RefreshExpiry),
		Revoked:   false,
	}

	return g.db.Create(refreshToken).Error
}

func (g *TokenGenerator) ValidateRefreshToken(token string) (*RefreshToken, error) {
	tokenHash := g.HashToken(token)

	var refreshToken RefreshToken
	if err := g.db.Where("token = ? AND revoked = ?", tokenHash, false).First(&refreshToken).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("refresh token not found or revoked")
		}
		return nil, err
	}

	if time.Now().After(refreshToken.ExpiresAt) {
		return nil, errors.New("refresh token expired")
	}

	return &refreshToken, nil
}
