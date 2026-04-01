package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"bey/internal/config"
)

type TokenGenerator struct {
	db     *gorm.DB
	config *config.Config
	redis  *redis.Client
}

func NewTokenGenerator(db *gorm.DB, config *config.Config) *TokenGenerator {
	return &TokenGenerator{
		db:     db,
		config: config,
	}
}

func NewTokenGeneratorWithRedis(db *gorm.DB, config *config.Config, redisClient *redis.Client) *TokenGenerator {
	return &TokenGenerator{
		db:     db,
		config: config,
		redis:  redisClient,
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
	if g.redis != nil {
		return g.rotateRefreshTokenRedis(oldTokenHash)
	}
	return g.rotateRefreshTokenDB(oldTokenHash)
}

func (g *TokenGenerator) rotateRefreshTokenDB(oldTokenHash string) (string, error) {
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

func (g *TokenGenerator) rotateRefreshTokenRedis(oldTokenHash string) (string, error) {
	key := "auth:refresh:" + oldTokenHash
	data, err := g.redis.Get(context.Background(), key).Bytes()
	if err == redis.Nil {
		return "", errors.New("refresh token not found")
	}
	if err != nil {
		return "", fmt.Errorf("failed to get refresh token from redis: %w", err)
	}

	var tokenData map[string]interface{}
	if err := json.Unmarshal(data, &tokenData); err != nil {
		return "", fmt.Errorf("failed to unmarshal refresh token: %w", err)
	}

	if revoked, ok := tokenData["revoked"].(bool); ok && revoked {
		return "", errors.New("refresh token revoked")
	}

	expiresAt, ok := tokenData["expires_at"].(string)
	if !ok {
		return "", errors.New("invalid token data")
	}
	expTime, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil {
		return "", errors.New("invalid token expiration")
	}
	if time.Now().After(expTime) {
		return "", errors.New("refresh token expired")
	}

	g.redis.Del(context.Background(), key)

	newToken, err := g.GenerateRefreshToken()
	if err != nil {
		return "", err
	}

	userID, _ := tokenData["user_id"].(float64)
	if err := g.StoreRefreshToken(newToken, uint(userID)); err != nil {
		return "", err
	}

	return newToken, nil
}

func (g *TokenGenerator) StoreRefreshToken(token string, userID uint) error {
	jwtConfig := g.config.Security.GetJWTConfig()
	tokenHash := g.HashToken(token)
	expiresAt := time.Now().Add(jwtConfig.RefreshExpiry)

	if g.redis != nil {
		return g.storeRefreshTokenRedis(tokenHash, userID, expiresAt)
	}

	return g.storeRefreshTokenDB(tokenHash, userID, expiresAt)
}

func (g *TokenGenerator) storeRefreshTokenDB(tokenHash string, userID uint, expiresAt time.Time) error {
	refreshToken := &RefreshToken{
		Token:     tokenHash,
		UserID:    userID,
		ExpiresAt: expiresAt,
		Revoked:   false,
	}

	return g.db.Create(refreshToken).Error
}

func (g *TokenGenerator) storeRefreshTokenRedis(tokenHash string, userID uint, expiresAt time.Time) error {
	key := "auth:refresh:" + tokenHash
	tokenData := map[string]interface{}{
		"user_id":    userID,
		"expires_at": expiresAt.Format(time.RFC3339),
		"revoked":    false,
	}

	data, err := json.Marshal(tokenData)
	if err != nil {
		return fmt.Errorf("failed to marshal refresh token: %w", err)
	}

	ttl := time.Until(expiresAt)
	if ttl <= 0 {
		ttl = 1 * time.Second
	}

	if err := g.redis.Set(context.Background(), key, data, ttl).Err(); err != nil {
		return fmt.Errorf("failed to store refresh token in redis: %w", err)
	}

	g.db.Create(&RefreshToken{
		Token:     tokenHash,
		UserID:    userID,
		ExpiresAt: expiresAt,
		Revoked:   false,
	})

	return nil
}

func (g *TokenGenerator) ValidateRefreshToken(token string) (*RefreshToken, error) {
	tokenHash := g.HashToken(token)

	if g.redis != nil {
		return g.validateRefreshTokenRedis(tokenHash, token)
	}

	return g.validateRefreshTokenDB(tokenHash)
}

func (g *TokenGenerator) validateRefreshTokenDB(tokenHash string) (*RefreshToken, error) {
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

func (g *TokenGenerator) validateRefreshTokenRedis(tokenHash string, rawToken string) (*RefreshToken, error) {
	key := "auth:refresh:" + tokenHash
	data, err := g.redis.Get(context.Background(), key).Bytes()
	if err == redis.Nil {
		return g.validateRefreshTokenDB(tokenHash)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get refresh token from redis: %w", err)
	}

	var tokenData map[string]interface{}
	if err := json.Unmarshal(data, &tokenData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal refresh token: %w", err)
	}

	if revoked, ok := tokenData["revoked"].(bool); ok && revoked {
		return nil, errors.New("refresh token not found or revoked")
	}

	expiresAt, ok := tokenData["expires_at"].(string)
	if !ok {
		return nil, errors.New("invalid token data")
	}
	expTime, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil {
		return nil, errors.New("invalid token expiration")
	}
	if time.Now().After(expTime) {
		return nil, errors.New("refresh token expired")
	}

	userID, _ := tokenData["user_id"].(float64)
	return &RefreshToken{
		Token:     tokenHash,
		UserID:    uint(userID),
		ExpiresAt: expTime,
	}, nil
}
