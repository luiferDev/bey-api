package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"bey/internal/shared/middleware"
)

// Test helper types
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string `json:"token,omitempty"`
	Error string `json:"error,omitempty"`
}

type ProtectedResponse struct {
	UserID uint   `json:"user_id"`
	Email  string `json:"email"`
}

// JWT test helper - generates a valid token for testing
func generateTestJWT(userID uint, secret string, expiryHours int) string {
	expiry := time.Now().Add(time.Duration(expiryHours) * time.Hour)
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     expiry.Unix(),
		"iat":     time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(secret))
	return tokenString
}

// Setup test router with auth middleware
func setupAuthTestRouter(secret string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	// Public routes (no auth middleware)
	r.POST("/login", handleLogin)
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	r.GET("/products", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"products": []string{}})
	})

	// Protected routes (with auth middleware)
	protected := r.Group("/api/v1")
	authMiddleware := middleware.NewAuthMiddleware(secret)
	protected.Use(authMiddleware.Handler())
	{
		protected.GET("/profile", handleGetProfile)
		protected.GET("/orders", handleGetOrders)
		protected.POST("/orders", handleCreateOrder)
		protected.GET("/inventory", handleGetInventory)
	}

	return r
}

// Mock login handler
func handleLogin(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, LoginResponse{Error: "invalid request"})
		return
	}

	// Mock user validation
	// In real tests, this would check the database
	if req.Email == "test@example.com" && req.Password == "password123" {
		token := generateTestJWT(1, "test-secret", 2)
		c.JSON(http.StatusOK, LoginResponse{Token: token})
		return
	}

	if req.Email == "expired@example.com" && req.Password == "password123" {
		// Generate expired token
		expiry := time.Now().Add(-1 * time.Hour)
		claims := jwt.MapClaims{
			"user_id": 2,
			"exp":     expiry.Unix(),
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, _ := token.SignedString([]byte("test-secret"))
		c.JSON(http.StatusOK, LoginResponse{Token: tokenString})
		return
	}

	c.JSON(http.StatusUnauthorized, LoginResponse{Error: "invalid credentials"})
}

// Mock profile handler
func handleGetProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	c.JSON(http.StatusOK, ProtectedResponse{
		UserID: uint(userID.(float64)),
		Email:  "test@example.com",
	})
}

// Mock orders handler
func handleGetOrders(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"orders": []string{}})
}

func handleCreateOrder(c *gin.Context) {
	c.JSON(http.StatusCreated, gin.H{"order_id": "12345"})
}

func handleGetInventory(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"inventory": []string{}})
}

// Integration tests
func TestAuthFlow_LoginAndAccessProtectedEndpoint(t *testing.T) {
	router := setupAuthTestRouter("test-secret")

	t.Run("Step 1: Login successfully", func(t *testing.T) {
		body := `{"email":"test@example.com","password":"password123"}`
		req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("login failed: got %d; want %d", w.Code, http.StatusOK)
		}

		var resp LoginResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if resp.Token == "" {
			t.Error("expected token in response")
		}
	})

	t.Run("Step 2: Access protected endpoint with token", func(t *testing.T) {
		// First get a token
		body := `{"email":"test@example.com","password":"password123"}`
		req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		var loginResp LoginResponse
		json.Unmarshal(w.Body.Bytes(), &loginResp)

		// Then access protected endpoint
		profileReq := httptest.NewRequest(http.MethodGet, "/api/v1/profile", nil)
		profileReq.Header.Set("Authorization", "Bearer "+loginResp.Token)
		profileW := httptest.NewRecorder()

		router.ServeHTTP(profileW, profileReq)

		if profileW.Code != http.StatusOK {
			t.Errorf("access protected failed: got %d; want %d", profileW.Code, http.StatusOK)
		}

		var profileResp ProtectedResponse
		json.Unmarshal(profileW.Body.Bytes(), &profileResp)

		if profileResp.UserID != 1 {
			t.Errorf("user_id = %d; want 1", profileResp.UserID)
		}
	})
}

func TestAuthFlow_InvalidCredentials(t *testing.T) {
	router := setupAuthTestRouter("test-secret")

	body := `{"email":"wrong@example.com","password":"wrongpass"}`
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("got %d; want %d", w.Code, http.StatusUnauthorized)
	}

	var resp LoginResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Error == "" {
		t.Error("expected error message")
	}
}

func TestAuthFlow_ExpiredToken(t *testing.T) {
	router := setupAuthTestRouter("test-secret")

	// Login with expired token user
	body := `{"email":"expired@example.com","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var loginResp LoginResponse
	json.Unmarshal(w.Body.Bytes(), &loginResp)

	// Try to access protected endpoint
	profileReq := httptest.NewRequest(http.MethodGet, "/api/v1/profile", nil)
	profileReq.Header.Set("Authorization", "Bearer "+loginResp.Token)
	profileW := httptest.NewRecorder()

	router.ServeHTTP(profileW, profileReq)

	if profileW.Code != http.StatusUnauthorized {
		t.Errorf("expired token should return 401: got %d", profileW.Code)
	}
}

func TestAuthFlow_MissingToken(t *testing.T) {
	router := setupAuthTestRouter("test-secret")

	// Access protected endpoint without token
	req := httptest.NewRequest(http.MethodGet, "/api/v1/profile", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("missing token should return 401: got %d", w.Code)
	}
}

func TestAuthFlow_InvalidTokenFormat(t *testing.T) {
	router := setupAuthTestRouter("test-secret")

	// Access with wrong format
	req := httptest.NewRequest(http.MethodGet, "/api/v1/profile", nil)
	req.Header.Set("Authorization", "InvalidFormat")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("invalid format should return 401: got %d", w.Code)
	}
}

func TestAuthFlow_PublicEndpointsNoAuth(t *testing.T) {
	router := setupAuthTestRouter("test-secret")

	// Note: /login and /products are public (not in protected group)
	// These should work without auth - but the mock handlers may fail for other reasons

	// Test that /health works without auth
	t.Run("health", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should NOT return 401
		if w.Code == http.StatusUnauthorized {
			t.Errorf("public endpoint /health should not require auth")
		}
	})
}

func TestAuthFlow_ProtectedEndpoints(t *testing.T) {
	router := setupAuthTestRouter("test-secret")

	// Get token
	body := `{"email":"test@example.com","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var loginResp LoginResponse
	json.Unmarshal(w.Body.Bytes(), &loginResp)
	token := loginResp.Token

	protectedEndpoints := []struct {
		name   string
		method string
		path   string
	}{
		{"get profile", http.MethodGet, "/api/v1/profile"},
		{"get orders", http.MethodGet, "/api/v1/orders"},
		{"create order", http.MethodPost, "/api/v1/orders"},
		{"get inventory", http.MethodGet, "/api/v1/inventory"},
	}

	for _, tt := range protectedEndpoints {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.method == http.MethodPost {
				req = httptest.NewRequest(tt.method, tt.path, strings.NewReader(`{}`))
			} else {
				req = httptest.NewRequest(tt.method, tt.path, nil)
			}
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code == http.StatusUnauthorized {
				t.Errorf("protected endpoint %s should allow valid token", tt.path)
			}
		})
	}
}

func TestAuthFlow_DifferentSecrets(t *testing.T) {
	// Test with wrong secret
	router := setupAuthTestRouter("test-secret")

	// Generate token with different secret
	wrongSecretToken := generateTestJWT(1, "wrong-secret", 2)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/profile", nil)
	req.Header.Set("Authorization", "Bearer "+wrongSecretToken)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("wrong secret should return 401: got %d", w.Code)
	}
}

// Test bcrypt password hashing (as used in real user module)
func TestPasswordHashing(t *testing.T) {
	password := "testPassword123"

	// Hash password
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	// Verify correct password
	err = bcrypt.CompareHashAndPassword(hashed, []byte(password))
	if err != nil {
		t.Errorf("password verification failed: %v", err)
	}

	// Verify wrong password
	err = bcrypt.CompareHashAndPassword(hashed, []byte("wrongpassword"))
	if err == nil {
		t.Error("wrong password should fail verification")
	}
}
