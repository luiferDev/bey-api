package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"bey/internal/config"
	"bey/internal/modules/users"
	"bey/internal/shared/middleware"
)

func setupIntegrationTest(t *testing.T) (*gin.Engine, *gorm.DB) {
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
			CSRFEnabled:      true,
		},
	}
	cfg.Security.JWTConfig = cfg.Security.GetJWTConfig()

	authService := NewAuthService(db, cfg)
	authMiddleware := NewAuthMiddleware(authService, cfg)
	csrfConfig := middleware.DefaultCSRFConfig()

	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.GET("/csrf-token", func(c *gin.Context) {
		c.SetCookie(csrfConfig.CookieName, "test-csrf-token", int(csrfConfig.CookieExpiry.Seconds()), "/", "", false, false)
		c.JSON(http.StatusOK, gin.H{"token": "test-csrf-token"})
	})

	r.Use(middleware.CSRFMiddleware(csrfConfig))

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

		c.SetCookie("access_token", resp.AccessToken, 900, "/", "", false, true)
		c.JSON(http.StatusOK, resp)
	})

	r.POST("/api/v1/auth/refresh", func(c *gin.Context) {
		var req struct {
			RefreshToken string `json:"refresh_token" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		resp, err := authService.Refresh(c.Request.Context(), req.RefreshToken)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		c.SetCookie("access_token", resp.AccessToken, 900, "/", "", false, true)
		c.JSON(http.StatusOK, resp)
	})

	r.POST("/api/v1/auth/logout", authMiddleware.RequireAuth(), func(c *gin.Context) {
		var req struct {
			RefreshToken string `json:"refresh_token" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		err := authService.Logout(c.Request.Context(), req.RefreshToken)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "logged out"})
	})

	r.GET("/api/v1/protected", authMiddleware.RequireAuth(), func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		c.JSON(http.StatusOK, gin.H{"user_id": userID, "message": "access granted"})
	})

	return r, db
}

func createTestUserForIntegration(t *testing.T, db *gorm.DB) *users.User {
	t.Helper()

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	user := &users.User{
		Email:     "test@example.com",
		Password:  string(hashedPassword),
		FirstName: "Test",
		LastName:  "User",
		Role:      "customer",
		Active:    true,
	}

	if err := db.Create(user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	return user
}

func makeAuthenticatedRequest(t *testing.T, r *gin.Engine, method, path, email, password string) (*httptest.ResponseRecorder, string) {
	csrfReq := httptest.NewRequest(http.MethodGet, "/csrf-token", nil)
	csrfW := httptest.NewRecorder()
	r.ServeHTTP(csrfW, csrfReq)

	loginReq := map[string]string{"email": email, "password": password}
	loginJson, err := json.Marshal(loginReq)
	if err != nil {
		t.Fatalf("Failed to marshal login request: %v", err)
	}

	loginHttpReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(string(loginJson)))
	loginHttpReq.Header.Set("Content-Type", "application/json")
	loginHttpReq.Header.Set("X-CSRF-Token", "test-csrf-token")
	loginHttpReq.AddCookie(&http.Cookie{Name: "csrf_token", Value: "test-csrf-token"})
	loginW := httptest.NewRecorder()
	r.ServeHTTP(loginW, loginHttpReq)

	var loginResp TokenResponse
	if err := json.Unmarshal(loginW.Body.Bytes(), &loginResp); err != nil {
		t.Fatalf("Failed to unmarshal login response: %v", err)
	}

	cookie := loginW.Result().Cookies()
	var accessTokenCookie *http.Cookie
	for _, c := range cookie {
		if c.Name == "access_token" {
			accessTokenCookie = c
			break
		}
	}

	req := httptest.NewRequest(method, path, nil)
	req.AddCookie(accessTokenCookie)
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: "test-csrf-token"})
	req.Header.Set("X-CSRF-Token", "test-csrf-token")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	return w, loginResp.AccessToken
}

func TestAuthFlow_LoginSuccess(t *testing.T) {
	r, db := setupIntegrationTest(t)
	createTestUserForIntegration(t, db)

	csrfReq := httptest.NewRequest(http.MethodGet, "/csrf-token", nil)
	csrfW := httptest.NewRecorder()
	r.ServeHTTP(csrfW, csrfReq)

	loginReq := `{"email":"test@example.com","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(loginReq))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-Token", "test-csrf-token")
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: "test-csrf-token"})
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("login failed: got status %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var loginResp TokenResponse
	if err := json.Unmarshal(w.Body.Bytes(), &loginResp); err != nil {
		t.Fatalf("failed to parse login response: %v", err)
	}

	if loginResp.AccessToken == "" {
		t.Error("expected access token in response")
	}
	if loginResp.RefreshToken == "" {
		t.Error("expected refresh token in response")
	}
}

func TestAuthFlow_AccessProtectedWithToken(t *testing.T) {
	r, db := setupIntegrationTest(t)
	createTestUserForIntegration(t, db)

	w, _ := makeAuthenticatedRequest(t, r, http.MethodGet, "/api/v1/protected", "test@example.com", "password123")

	if w.Code != http.StatusOK {
		t.Errorf("protected route: got status %d, want %d", w.Code, http.StatusOK)
	}
}

func TestAuthFlow_AccessProtectedWithoutToken(t *testing.T) {
	r, _ := setupIntegrationTest(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("got status %d; want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuthFlow_LoginWithInvalidPassword(t *testing.T) {
	r, db := setupIntegrationTest(t)
	createTestUserForIntegration(t, db)

	loginReq := `{"email":"test@example.com","password":"wrongpassword"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(loginReq))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-Token", "test-csrf-token")
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: "test-csrf-token"})
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("got status %d; want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuthFlow_LoginWithInvalidEmail(t *testing.T) {
	r, db := setupIntegrationTest(t)
	createTestUserForIntegration(t, db)

	loginReq := `{"email":"wrong@example.com","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(loginReq))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-Token", "test-csrf-token")
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: "test-csrf-token"})
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("got status %d; want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuthFlow_RefreshToken(t *testing.T) {
	r, db := setupIntegrationTest(t)
	testUser := createTestUserForIntegration(t, db)

	_, _ = makeAuthenticatedRequest(t, r, http.MethodGet, "/api/v1/protected", "test@example.com", "password123")

	loginReq := map[string]string{"email": "test@example.com", "password": "password123"}
	loginJson, err := json.Marshal(loginReq)
	if err != nil {
		t.Fatalf("Failed to marshal login request: %v", err)
	}
	loginHttpReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(string(loginJson)))
	loginHttpReq.Header.Set("Content-Type", "application/json")
	loginHttpReq.Header.Set("X-CSRF-Token", "test-csrf-token")
	loginHttpReq.AddCookie(&http.Cookie{Name: "csrf_token", Value: "test-csrf-token"})
	loginW := httptest.NewRecorder()
	r.ServeHTTP(loginW, loginHttpReq)

	var loginResp TokenResponse
	if err := json.Unmarshal(loginW.Body.Bytes(), &loginResp); err != nil {
		t.Fatalf("Failed to unmarshal login response: %v", err)
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

	tokenGenerator := NewTokenGenerator(db, cfg)
	refreshToken, err := tokenGenerator.GenerateRefreshToken()
	if err != nil {
		t.Fatalf("Failed to generate refresh token: %v", err)
	}
	if err := tokenGenerator.StoreRefreshToken(refreshToken, testUser.ID); err != nil {
		t.Fatalf("Failed to store refresh token: %v", err)
	}

	refreshReqBody := map[string]string{"refresh_token": refreshToken}
	refreshJson, err := json.Marshal(refreshReqBody)
	if err != nil {
		t.Fatalf("Failed to marshal refresh request: %v", err)
	}
	refreshHttpReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", strings.NewReader(string(refreshJson)))
	refreshHttpReq.Header.Set("Content-Type", "application/json")
	refreshHttpReq.Header.Set("X-CSRF-Token", "test-csrf-token")
	refreshHttpReq.AddCookie(&http.Cookie{Name: "csrf_token", Value: "test-csrf-token"})
	refreshW := httptest.NewRecorder()

	r.ServeHTTP(refreshW, refreshHttpReq)

	if refreshW.Code != http.StatusOK {
		t.Errorf("refresh failed: got status %d, want %d, body: %s", refreshW.Code, http.StatusOK, refreshW.Body.String())
	}

	var refreshResp TokenResponse
	if err := json.Unmarshal(refreshW.Body.Bytes(), &refreshResp); err != nil {
		t.Fatalf("failed to parse refresh response: %v", err)
	}

	if refreshResp.AccessToken == "" {
		t.Error("expected new access token")
	}
}

func TestAuthFlow_Logout(t *testing.T) {
	r, db := setupIntegrationTest(t)
	createTestUserForIntegration(t, db)

	loginReq := map[string]string{"email": "test@example.com", "password": "password123"}
	loginJson, err := json.Marshal(loginReq)
	if err != nil {
		t.Fatalf("Failed to marshal login request: %v", err)
	}
	loginHttpReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(string(loginJson)))
	loginHttpReq.Header.Set("Content-Type", "application/json")
	loginHttpReq.Header.Set("X-CSRF-Token", "test-csrf-token")
	loginHttpReq.AddCookie(&http.Cookie{Name: "csrf_token", Value: "test-csrf-token"})
	loginW := httptest.NewRecorder()
	r.ServeHTTP(loginW, loginHttpReq)

	var loginResp TokenResponse
	if err := json.Unmarshal(loginW.Body.Bytes(), &loginResp); err != nil {
		t.Fatalf("Failed to unmarshal login response: %v", err)
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

	tokenGenerator := NewTokenGenerator(db, cfg)
	refreshToken, err := tokenGenerator.GenerateRefreshToken()
	if err != nil {
		t.Fatalf("Failed to generate refresh token: %v", err)
	}
	if err := tokenGenerator.StoreRefreshToken(refreshToken, uuid.Must(uuid.NewV7())); err != nil {
		t.Fatalf("Failed to store refresh token: %v", err)
	}

	accessToken := loginResp.AccessToken

	logoutReqBody := map[string]string{"refresh_token": refreshToken}
	logoutJson, err := json.Marshal(logoutReqBody)
	if err != nil {
		t.Fatalf("Failed to marshal logout request: %v", err)
	}
	logoutReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", strings.NewReader(string(logoutJson)))
	logoutReq.Header.Set("Content-Type", "application/json")
	logoutReq.Header.Set("X-CSRF-Token", "test-csrf-token")
	logoutReq.AddCookie(&http.Cookie{Name: "csrf_token", Value: "test-csrf-token"})
	logoutReq.AddCookie(&http.Cookie{Name: "access_token", Value: accessToken})
	logoutW := httptest.NewRecorder()

	r.ServeHTTP(logoutW, logoutReq)

	if logoutW.Code != http.StatusOK {
		t.Errorf("logout without auth: got status %d, want %d", logoutW.Code, http.StatusUnauthorized)
	}
}
