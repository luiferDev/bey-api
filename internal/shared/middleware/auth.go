// DEPRECATED: This middleware is a duplicate and is NOT used in production routes.
// The canonical auth middleware is in internal/modules/auth/middleware.go.
// This file is kept only for legacy integration tests.
// TODO: Remove this file and update auth_integration_test.go to use the canonical middleware.
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type AuthMiddleware struct {
	jwtSecret []byte
}

func NewAuthMiddleware(secret string) *AuthMiddleware {
	return &AuthMiddleware{jwtSecret: []byte(secret)}
}

// Handler returns the gin middleware handler
func (m *AuthMiddleware) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if path is public
		if isPublicPath(c.Request.URL.Path) {
			c.Next()
			return
		}

		// Extract token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "missing authorization token",
			})
			return
		}

		// Extract token from "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid authorization header format",
			})
			return
		}

		tokenString := parts[1]

		// Parse and validate token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// Validate signing method
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return m.jwtSecret, nil
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid or expired token",
			})
			return
		}

		// Extract claims and set user_id in context
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid token claims",
			})
			return
		}

		// Get user_id from claims
		if userID, exists := claims["user_id"]; exists {
			c.Set("user_id", userID)
		}

		c.Next()
	}
}

// isPublicPath checks if the path should skip authentication
func isPublicPath(path string) bool {
	publicPaths := []string{
		"/health",
		"/swagger",
		"/api/v1/users",       // POST only (registration)
		"/api/v1/users/login", // Login
		"/api/v1/products",    // Read-only products
		"/api/v1/categories",  // Read-only categories
	}

	// Exact match
	for _, p := range publicPaths {
		if path == p {
			return true
		}
	}

	// Check prefixes
	publicPrefixes := []string{
		"/swagger/",
		"/dashboard/",
	}

	for _, prefix := range publicPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}

	// POST to /api/v1/users is public (registration)
	if path == "/api/v1/users" {
		return true
	}

	return false
}
