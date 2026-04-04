package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid/v5"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"bey/internal/config"
)

func setupMiddlewareTest(t *testing.T) (*AuthMiddleware, *gin.Engine) {
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

	authService := NewAuthService(db, cfg)
	middleware := NewAuthMiddleware(authService, cfg)

	gin.SetMode(gin.TestMode)
	r := gin.New()

	return middleware, r
}

func TestAuthMiddleware_ValidBearerToken(t *testing.T) {
	middleware, r := setupMiddlewareTest(t)

	r.GET("/protected", middleware.RequireAuth(), func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		userRole, _ := c.Get("user_role")
		userEmail, _ := c.Get("user_email")

		c.JSON(http.StatusOK, gin.H{
			"user_id":    userID,
			"user_role":  userRole,
			"user_email": userEmail,
		})
	})

	cfg := &config.Config{
		Security: config.SecurityConfig{
			JWTSecret:       "test-secret-key-for-jwt-token-generation",
			JWTAccessExpiry: 15 * time.Minute,
			JWTIssuer:       "bey_api_test",
		},
	}
	cfg.Security.JWTConfig = cfg.Security.GetJWTConfig()

	tokenGen := NewTokenGenerator(nil, cfg)
	accessToken, _, _ := tokenGen.GenerateAccessToken(uuid.Must(uuid.NewV7()), "test@example.com", "admin")

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("got status %d; want %d", w.Code, http.StatusOK)
	}
}

func TestAuthMiddleware_ValidCookieToken(t *testing.T) {
	middleware, r := setupMiddlewareTest(t)

	r.GET("/protected", middleware.RequireAuth(), func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		c.JSON(http.StatusOK, gin.H{"user_id": userID})
	})

	cfg := &config.Config{
		Security: config.SecurityConfig{
			JWTSecret:       "test-secret-key-for-jwt-token-generation",
			JWTAccessExpiry: 15 * time.Minute,
			JWTIssuer:       "bey_api_test",
		},
	}
	cfg.Security.JWTConfig = cfg.Security.GetJWTConfig()

	tokenGen := NewTokenGenerator(nil, cfg)
	accessToken, _, _ := tokenGen.GenerateAccessToken(uuid.Must(uuid.NewV7()), "test@example.com", "customer")

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: accessToken})
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("got status %d; want %d", w.Code, http.StatusOK)
	}
}

func TestAuthMiddleware_NoToken(t *testing.T) {
	middleware, r := setupMiddlewareTest(t)

	r.GET("/protected", middleware.RequireAuth(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("got status %d; want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	middleware, r := setupMiddlewareTest(t)

	r.GET("/protected", middleware.RequireAuth(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("got status %d; want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuthMiddleware_ExpiredToken(t *testing.T) {
	middleware, r := setupMiddlewareTest(t)

	r.GET("/protected", middleware.RequireAuth(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	cfg := &config.Config{
		Security: config.SecurityConfig{
			JWTSecret:       "test-secret-key",
			JWTAccessExpiry: -1 * time.Minute,
			JWTIssuer:       "bey_api_test",
		},
	}
	cfg.Security.JWTConfig = cfg.Security.GetJWTConfig()

	expiredTokenGen := NewTokenGenerator(nil, cfg)
	expiredToken, _, _ := expiredTokenGen.GenerateAccessToken(uuid.Must(uuid.NewV7()), "test@example.com", "customer")

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+expiredToken)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("got status %d; want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuthMiddleware_SetsClaims(t *testing.T) {
	middleware, r := setupMiddlewareTest(t)

	expectedUserID := uuid.Must(uuid.NewV7())
	expectedEmail := "claims@test.com"
	expectedRole := "admin"

	r.GET("/protected", middleware.RequireAuth(), func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		userRole, _ := c.Get("user_role")
		userEmail, _ := c.Get("user_email")

		if userID != expectedUserID.String() {
			t.Errorf("user_id = %v; want %s", userID, expectedUserID.String())
		}
		if userRole != expectedRole {
			t.Errorf("user_role = %v; want %s", userRole, expectedRole)
		}
		if userEmail != expectedEmail {
			t.Errorf("user_email = %v; want %s", userEmail, expectedEmail)
		}
		c.Status(http.StatusOK)
	})

	cfg := &config.Config{
		Security: config.SecurityConfig{
			JWTSecret:       "test-secret-key-for-jwt-token-generation",
			JWTAccessExpiry: 15 * time.Minute,
			JWTIssuer:       "bey_api_test",
		},
	}
	cfg.Security.JWTConfig = cfg.Security.GetJWTConfig()

	tokenGen := NewTokenGenerator(nil, cfg)
	accessToken, _, _ := tokenGen.GenerateAccessToken(expectedUserID, expectedEmail, expectedRole)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("got status %d; want %d", w.Code, http.StatusOK)
	}
}

func TestExtractToken(t *testing.T) {
	middleware, _ := setupMiddlewareTest(t)

	tests := []struct {
		name       string
		authHeader string
		cookie     string
		want       string
	}{
		{"Bearer token", "Bearer test-token", "", "test-token"},
		{"Empty auth header", "", "", ""},
		{"Invalid format", "InvalidFormat", "", ""},
		{"No Bearer prefix", "test-token", "", ""},
		{"Cookie token", "", "cookie-token", "cookie-token"},
		{"Header takes precedence", "Bearer header-token", "cookie-token", "header-token"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			if tt.authHeader != "" {
				c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
				c.Request.Header.Set("Authorization", tt.authHeader)
			}
			if tt.cookie != "" {
				if c.Request == nil {
					c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
				}
				c.Request.AddCookie(&http.Cookie{Name: "access_token", Value: tt.cookie})
			}
			if c.Request == nil {
				c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
			}

			got := middleware.extractToken(c)
			if got != tt.want {
				t.Errorf("extractToken() = %q; want %q", got, tt.want)
			}
		})
	}

	_ = context.Background()
}
