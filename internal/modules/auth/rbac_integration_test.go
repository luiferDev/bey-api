package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"bey/internal/config"
	"bey/internal/modules/users"
	"bey/internal/shared/middleware"
)

func setupRBACIntegrationTest(t *testing.T) (*gin.Engine, *gorm.DB) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	if err := db.AutoMigrate(&users.User{}, &RefreshToken{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
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
	authMiddleware := NewAuthMiddleware(authService, cfg)

	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.POST("/api/v1/auth/login", func(c *gin.Context) {
		var req LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		resp, err := authService.Login(c.Request.Context(), req.Email, req.Password)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, resp)
	})

	r.GET("/api/v1/admin/users", authMiddleware.RequireAuth(), middleware.RequireRole(middleware.RoleAdmin), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "admin only"})
	})

	r.GET("/api/v1/orders", authMiddleware.RequireAuth(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "authenticated users"})
	})

	r.GET("/api/v1/products", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "public endpoint"})
	})

	return r, db
}

func createAdminUser(t *testing.T, db *gorm.DB) {
	t.Helper()

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	user := &users.User{
		Email:     "admin@example.com",
		Password:  string(hashedPassword),
		FirstName: "Admin",
		LastName:  "User",
		Role:      "admin",
		Active:    true,
	}

	if err := db.Create(user).Error; err != nil {
		t.Fatalf("failed to create admin user: %v", err)
	}
}

func createCustomerUser(t *testing.T, db *gorm.DB) {
	t.Helper()

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("customer123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	user := &users.User{
		Email:     "customer@example.com",
		Password:  string(hashedPassword),
		FirstName: "Customer",
		LastName:  "User",
		Role:      "customer",
		Active:    true,
	}

	if err := db.Create(user).Error; err != nil {
		t.Fatalf("failed to create customer user: %v", err)
	}
}

func getAccessToken(t *testing.T, r *gin.Engine, email, password string) string {
	loginReq := map[string]string{"email": email, "password": password}
	loginJson, _ := json.Marshal(loginReq)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(string(loginJson)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	var resp TokenResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	return resp.AccessToken
}

func TestAdminCanAccessAdminRoutes(t *testing.T) {
	r, db := setupRBACIntegrationTest(t)
	createAdminUser(t, db)

	token := getAccessToken(t, r, "admin@example.com", "admin123")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("admin access denied: got status %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestCustomerCannotAccessAdminRoutes(t *testing.T) {
	r, db := setupRBACIntegrationTest(t)
	createAdminUser(t, db)
	createCustomerUser(t, db)

	token := getAccessToken(t, r, "customer@example.com", "customer123")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("customer should be denied: got status %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestUnauthenticatedDenied(t *testing.T) {
	r, db := setupRBACIntegrationTest(t)
	createAdminUser(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("unauthenticated should be denied: got status %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestCustomerCanAccessOrders(t *testing.T) {
	r, db := setupRBACIntegrationTest(t)
	createAdminUser(t, db)
	createCustomerUser(t, db)

	token := getAccessToken(t, r, "customer@example.com", "customer123")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orders", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("customer access orders: got status %d, want %d", w.Code, http.StatusOK)
	}
}

func TestAdminCanAccessOrders(t *testing.T) {
	r, db := setupRBACIntegrationTest(t)
	createAdminUser(t, db)

	token := getAccessToken(t, r, "admin@example.com", "admin123")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orders", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("admin access orders: got status %d, want %d", w.Code, http.StatusOK)
	}
}

func TestPublicEndpointNoAuth(t *testing.T) {
	r, _ := setupRBACIntegrationTest(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/products", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("public endpoint: got status %d, want %d", w.Code, http.StatusOK)
	}
}

func TestCustomerAccessAdminWithInvalidToken(t *testing.T) {
	r, db := setupRBACIntegrationTest(t)
	createAdminUser(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("invalid token: got status %d, want %d", w.Code, http.StatusUnauthorized)
	}
}
