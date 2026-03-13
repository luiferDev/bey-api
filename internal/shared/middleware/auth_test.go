package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func generateTestToken(secret string, userID interface{}, expiry time.Time) string {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     expiry.Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(secret))
	return tokenString
}

func TestAuthMiddleware_Handler(t *testing.T) {
	secret := "test-secret"

	tests := []struct {
		name           string
		method         string
		authHeader     string
		path           string
		wantStatus     int
		wantUserID     interface{}
		wantErrMessage string
	}{
		{
			name:       "valid token",
			authHeader: "Bearer " + generateTestToken(secret, 123, time.Now().Add(time.Hour)),
			path:       "/api/v1/orders",
			wantStatus: http.StatusOK,
			wantUserID: float64(123), // JWT parses numbers as float64
		},
		{
			name:           "missing token",
			authHeader:     "",
			path:           "/api/v1/orders",
			wantStatus:     http.StatusUnauthorized,
			wantErrMessage: "missing authorization token",
		},
		{
			name:           "invalid token format",
			authHeader:     "InvalidFormat",
			path:           "/api/v1/orders",
			wantStatus:     http.StatusUnauthorized,
			wantErrMessage: "invalid authorization header format",
		},
		{
			name:           "expired token",
			authHeader:     "Bearer " + generateTestToken(secret, 123, time.Now().Add(-time.Hour)),
			path:           "/api/v1/orders",
			wantStatus:     http.StatusUnauthorized,
			wantErrMessage: "invalid or expired token",
		},
		{
			name:           "invalid secret",
			authHeader:     "Bearer " + generateTestToken("wrong-secret", 123, time.Now().Add(time.Hour)),
			path:           "/api/v1/orders",
			wantStatus:     http.StatusUnauthorized,
			wantErrMessage: "invalid or expired token",
		},
		{
			name:       "public path - health",
			authHeader: "",
			path:       "/health",
			wantStatus: http.StatusOK,
		},
		{
			name:       "public path - products",
			authHeader: "",
			path:       "/api/v1/products",
			wantStatus: http.StatusOK,
		},
		{
			name:       "public path - categories",
			authHeader: "",
			path:       "/api/v1/categories",
			wantStatus: http.StatusOK,
		},
		{
			name:       "public path - users POST (registration)",
			authHeader: "",
			path:       "/api/v1/users",
			method:     http.MethodPost,
			wantStatus: http.StatusOK,
		},
		{
			name:       "public path - swagger",
			authHeader: "",
			path:       "/swagger/index.html",
			wantStatus: http.StatusOK,
		},
		{
			name:       "protected path - orders",
			authHeader: "",
			path:       "/api/v1/orders",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "protected path - inventory",
			authHeader: "",
			path:       "/api/v1/inventory/1",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "valid token - user endpoint",
			authHeader: "Bearer " + generateTestToken(secret, 42, time.Now().Add(time.Hour)),
			path:       "/api/v1/users/profile",
			wantStatus: http.StatusOK,
			wantUserID: float64(42),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			w := httptest.NewRecorder()
			c, r := gin.CreateTestContext(w)

			// Setup middleware
			auth := NewAuthMiddleware(secret)
			r.Use(auth.Handler())

			// Add test handler
			r.GET("/api/v1/orders", func(c *gin.Context) {
				userID, _ := c.Get("user_id")
				if tt.wantUserID != nil && userID != nil {
					if userID != tt.wantUserID {
						t.Errorf("user_id = %v; want %v", userID, tt.wantUserID)
					}
				}
				c.Status(http.StatusOK)
			})
			r.GET("/api/v1/users/profile", func(c *gin.Context) {
				userID, _ := c.Get("user_id")
				if tt.wantUserID != nil && userID != nil {
					if userID != tt.wantUserID {
						t.Errorf("user_id = %v; want %v", userID, tt.wantUserID)
					}
				}
				c.Status(http.StatusOK)
			})
			r.GET("/api/v1/inventory/:id", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			// Add public routes
			r.GET("/health", func(c *gin.Context) { c.Status(http.StatusOK) })
			r.GET("/api/v1/products", func(c *gin.Context) { c.Status(http.StatusOK) })
			r.GET("/api/v1/categories", func(c *gin.Context) { c.Status(http.StatusOK) })
			r.POST("/api/v1/users", func(c *gin.Context) { c.Status(http.StatusOK) }) // POST for registration
			r.GET("/api/v1/users/login", func(c *gin.Context) { c.Status(http.StatusOK) })
			r.GET("/swagger/*path", func(c *gin.Context) { c.Status(http.StatusOK) })

			// Make request
			method := http.MethodGet
			if tt.method != "" {
				method = tt.method
			}
			c.Request = httptest.NewRequest(method, tt.path, nil)
			if tt.authHeader != "" {
				c.Request.Header.Set("Authorization", tt.authHeader)
			}

			r.ServeHTTP(w, c.Request)

			// Check status
			if w.Code != tt.wantStatus {
				t.Errorf("got status %d; want %d", w.Code, tt.wantStatus)
			}

			// Check error message if expected
			if tt.wantErrMessage != "" && w.Code == http.StatusUnauthorized {
				if !contains(w.Body.String(), tt.wantErrMessage) {
					t.Errorf("response does not contain %q: %s", tt.wantErrMessage, w.Body.String())
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestIsPublicPath(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"/health", true},
		{"/swagger/index.html", true},
		{"/api/v1/users", true},
		{"/api/v1/users/login", true},
		{"/api/v1/products", true},
		{"/api/v1/categories", true},
		{"/dashboard/", true},
		{"/api/v1/orders", false},
		{"/api/v1/inventory/1", false},
		{"/api/v1/users/123", false},
		{"/random/path", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := isPublicPath(tt.path)
			if got != tt.want {
				t.Errorf("isPublicPath(%q) = %v; want %v", tt.path, got, tt.want)
			}
		})
	}
}
