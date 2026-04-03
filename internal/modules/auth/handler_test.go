package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"bey/internal/config"
	"bey/internal/shared/response"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid/v5"
)

type MockAuthService struct {
	loginFunc              func(ctx context.Context, email, password string) (*LoginResponse, error)
	refreshFunc            func(ctx context.Context, refreshToken string) (*TokenResponse, error)
	logoutFunc             func(ctx context.Context, refreshToken string) error
	verifyEmailFunc        func(ctx context.Context, token string) error
	resendVerificationFunc func(ctx context.Context, email string) error
	forgotPasswordFunc     func(ctx context.Context, email string) error
	resetPasswordFunc      func(ctx context.Context, token, newPassword string) error
	setupTwoFactorFunc     func(ctx context.Context, userID uuid.UUID) (*TwoFASetupResponse, error)
	enableTwoFactorFunc    func(ctx context.Context, userID uuid.UUID, code string) (*TwoFAEnableResponse, error)
	disableTwoFactorFunc   func(ctx context.Context, userID uuid.UUID, code, backupCode string) error
	verifyTwoFactorFunc    func(ctx context.Context, userID uuid.UUID, code string) (bool, error)
	loginWith2FAFunc       func(ctx context.Context, tempToken, code string) (*TokenResponse, error)
}

func (m *MockAuthService) Login(ctx context.Context, email, password string) (*LoginResponse, error) {
	if m.loginFunc != nil {
		return m.loginFunc(ctx, email, password)
	}
	return nil, nil
}

func (m *MockAuthService) Refresh(ctx context.Context, refreshToken string) (*TokenResponse, error) {
	if m.refreshFunc != nil {
		return m.refreshFunc(ctx, refreshToken)
	}
	return nil, nil
}

func (m *MockAuthService) Logout(ctx context.Context, refreshToken string) error {
	if m.logoutFunc != nil {
		return m.logoutFunc(ctx, refreshToken)
	}
	return nil
}

func (m *MockAuthService) ValidateToken(ctx context.Context, tokenString string) (*TokenClaims, error) {
	return nil, nil
}

func (m *MockAuthService) VerifyEmail(ctx context.Context, token string) error {
	if m.verifyEmailFunc != nil {
		return m.verifyEmailFunc(ctx, token)
	}
	return nil
}

func (m *MockAuthService) ResendVerification(ctx context.Context, email string) error {
	if m.resendVerificationFunc != nil {
		return m.resendVerificationFunc(ctx, email)
	}
	return nil
}

func (m *MockAuthService) ForgotPassword(ctx context.Context, email string) error {
	if m.forgotPasswordFunc != nil {
		return m.forgotPasswordFunc(ctx, email)
	}
	return nil
}

func (m *MockAuthService) ResetPassword(ctx context.Context, token, newPassword string) error {
	if m.resetPasswordFunc != nil {
		return m.resetPasswordFunc(ctx, token, newPassword)
	}
	return nil
}

func (m *MockAuthService) SetupTwoFactor(ctx context.Context, userID uuid.UUID) (*TwoFASetupResponse, error) {
	if m.setupTwoFactorFunc != nil {
		return m.setupTwoFactorFunc(ctx, userID)
	}
	return nil, nil
}

func (m *MockAuthService) EnableTwoFactor(ctx context.Context, userID uuid.UUID, code string) (*TwoFAEnableResponse, error) {
	if m.enableTwoFactorFunc != nil {
		return m.enableTwoFactorFunc(ctx, userID, code)
	}
	return nil, nil
}

func (m *MockAuthService) DisableTwoFactor(ctx context.Context, userID uuid.UUID, code, backupCode string) error {
	if m.disableTwoFactorFunc != nil {
		return m.disableTwoFactorFunc(ctx, userID, code, backupCode)
	}
	return nil
}

func (m *MockAuthService) VerifyTwoFactor(ctx context.Context, userID uuid.UUID, code string) (bool, error) {
	if m.verifyTwoFactorFunc != nil {
		return m.verifyTwoFactorFunc(ctx, userID, code)
	}
	return false, nil
}

func (m *MockAuthService) LoginWith2FA(ctx context.Context, tempToken, code string) (*TokenResponse, error) {
	if m.loginWith2FAFunc != nil {
		return m.loginWith2FAFunc(ctx, tempToken, code)
	}
	return nil, nil
}

func setupAuthTestRouter(mockSvc *MockAuthService, cfg *config.Config) *gin.Engine {
	gin.SetMode(gin.TestMode)

	respHandler := response.NewResponseHandler()

	router := gin.New()

	router.Use(func(c *gin.Context) {
		if userIDStr := c.GetHeader("X-User-ID"); userIDStr != "" {
			userID, err := strconv.ParseUint(userIDStr, 10, 32)
			if err == nil {
				c.Set("user_id", uint(userID))
			}
		}
		c.Next()
	})

	auth := router.Group("/api/v1/auth")
	{
		auth.POST("/login", HandleLogin(mockSvc, cfg, respHandler))
		auth.POST("/refresh", HandleRefresh(mockSvc, cfg, respHandler))
		auth.POST("/logout", HandleLogout(mockSvc, cfg, respHandler))
		auth.POST("/verify-email", HandleVerifyEmail(mockSvc, respHandler))
		auth.POST("/resend-verification", HandleResendVerification(mockSvc, respHandler))
		auth.POST("/forgot-password", HandleForgotPassword(mockSvc, respHandler))
		auth.POST("/reset-password", HandleResetPassword(mockSvc, respHandler))
		auth.POST("/2fa/setup", HandleSetup2FA(mockSvc, respHandler))
		auth.POST("/2fa/enable", HandleEnable2FA(mockSvc, respHandler))
		auth.POST("/2fa/disable", HandleDisable2FA(mockSvc, respHandler))
		auth.POST("/2fa/verify", HandleVerify2FA(mockSvc, respHandler))
		auth.POST("/2fa/login-verify", HandleLoginWith2FA(mockSvc, cfg, respHandler))
	}

	return router
}

func makeAuthRequest(router *gin.Engine, method, path, body string, userID *uint) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(method, path, bytes.NewBufferString(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if userID != nil {
		req.Header.Set("X-User-ID", "1")
		w.Header().Set("X-User-ID", "1")
	}
	router.ServeHTTP(w, req)
	return w
}

func TestHandleLogin_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := &MockAuthService{
		loginFunc: func(ctx context.Context, email, password string) (*LoginResponse, error) {
			return &LoginResponse{
				AccessToken:  "test-access-token",
				RefreshToken: "test-refresh-token",
				ExpiresIn:    3600,
				Requires2FA:  false,
			}, nil
		},
	}

	cfg := &config.Config{}
	router := setupAuthTestRouter(mockSvc, cfg)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/login",
		bytes.NewBufferString(`{"email":"test@test.com","password":"password123"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp["success"] != true {
		t.Errorf("Expected success=true, got %v", resp["success"])
	}
}

func TestHandleLogin_Requires2FA(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := &MockAuthService{
		loginFunc: func(ctx context.Context, email, password string) (*LoginResponse, error) {
			return &LoginResponse{
				Requires2FA: true,
				TempToken:   "temp-token-123",
			}, nil
		},
	}

	cfg := &config.Config{}
	router := setupAuthTestRouter(mockSvc, cfg)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/login",
		bytes.NewBufferString(`{"email":"test@test.com","password":"password123"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestHandleLogin_InvalidCredentials(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := &MockAuthService{
		loginFunc: func(ctx context.Context, email, password string) (*LoginResponse, error) {
			return nil, errors.New("invalid credentials")
		},
	}

	cfg := &config.Config{}
	router := setupAuthTestRouter(mockSvc, cfg)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/login",
		bytes.NewBufferString(`{"email":"test@test.com","password":"wrongpassword"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestHandleLogin_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := &MockAuthService{}
	cfg := &config.Config{}
	router := setupAuthTestRouter(mockSvc, cfg)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/login",
		bytes.NewBufferString(`{invalid json}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestHandleLogin_EmptyBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := &MockAuthService{}
	cfg := &config.Config{}
	router := setupAuthTestRouter(mockSvc, cfg)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBufferString(``))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestHandleLogin_MissingFields(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{
			name: "missing email",
			body: `{"password":"password123"}`,
		},
		{
			name: "missing password",
			body: `{"email":"test@test.com"}`,
		},
		{
			name: "empty email",
			body: `{"email":"","password":"password123"}`,
		},
		{
			name: "empty password",
			body: `{"email":"test@test.com","password":""}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			mockSvc := &MockAuthService{}
			cfg := &config.Config{}
			router := setupAuthTestRouter(mockSvc, cfg)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/v1/auth/login",
				bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("Expected status 400, got %d. Body: %s", w.Code, w.Body.String())
			}
		})
	}
}

func TestHandleRefresh_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := &MockAuthService{
		refreshFunc: func(ctx context.Context, refreshToken string) (*TokenResponse, error) {
			return &TokenResponse{
				AccessToken:  "new-access-token",
				RefreshToken: "new-refresh-token",
				ExpiresIn:    3600,
			}, nil
		},
	}

	cfg := &config.Config{}
	router := setupAuthTestRouter(mockSvc, cfg)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/refresh", nil)
	req.AddCookie(&http.Cookie{
		Name:  "refresh_token",
		Value: "valid-refresh-token",
	})
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestHandleRefresh_MissingCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := &MockAuthService{}
	cfg := &config.Config{}
	router := setupAuthTestRouter(mockSvc, cfg)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/refresh", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestHandleRefresh_InvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := &MockAuthService{
		refreshFunc: func(ctx context.Context, refreshToken string) (*TokenResponse, error) {
			return nil, errors.New("invalid or expired token")
		},
	}

	cfg := &config.Config{}
	router := setupAuthTestRouter(mockSvc, cfg)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/refresh", nil)
	req.AddCookie(&http.Cookie{
		Name:  "refresh_token",
		Value: "expired-token",
	})
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestHandleLogout_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := &MockAuthService{
		logoutFunc: func(ctx context.Context, refreshToken string) error {
			return nil
		},
	}

	cfg := &config.Config{}
	router := setupAuthTestRouter(mockSvc, cfg)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/logout", nil)
	req.AddCookie(&http.Cookie{
		Name:  "refresh_token",
		Value: "valid-refresh-token",
	})
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestHandleLogout_NoCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)

	logoutCalled := false
	mockSvc := &MockAuthService{
		logoutFunc: func(ctx context.Context, refreshToken string) error {
			logoutCalled = true
			return nil
		},
	}

	cfg := &config.Config{}
	router := setupAuthTestRouter(mockSvc, cfg)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/logout", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	if logoutCalled {
		t.Error("Logout should not be called when no refresh token cookie")
	}
}

func TestHandleVerifyEmail_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := &MockAuthService{
		verifyEmailFunc: func(ctx context.Context, token string) error {
			return nil
		},
	}

	cfg := &config.Config{}
	router := setupAuthTestRouter(mockSvc, cfg)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/verify-email",
		bytes.NewBufferString(`{"token":"valid-verification-token"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestHandleVerifyEmail_MissingToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := &MockAuthService{}
	cfg := &config.Config{}
	router := setupAuthTestRouter(mockSvc, cfg)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/verify-email",
		bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestHandleVerifyEmail_InvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := &MockAuthService{
		verifyEmailFunc: func(ctx context.Context, token string) error {
			return errors.New("invalid or expired token")
		},
	}

	cfg := &config.Config{}
	router := setupAuthTestRouter(mockSvc, cfg)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/verify-email",
		bytes.NewBufferString(`{"token":"expired-token"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestHandleVerifyEmail_EmptyBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := &MockAuthService{}
	cfg := &config.Config{}
	router := setupAuthTestRouter(mockSvc, cfg)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/verify-email",
		bytes.NewBufferString(``))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestHandleResendVerification_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := &MockAuthService{
		resendVerificationFunc: func(ctx context.Context, email string) error {
			return nil
		},
	}

	cfg := &config.Config{}
	router := setupAuthTestRouter(mockSvc, cfg)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/resend-verification",
		bytes.NewBufferString(`{"email":"test@test.com"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestHandleResendVerification_MissingEmail(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := &MockAuthService{}
	cfg := &config.Config{}
	router := setupAuthTestRouter(mockSvc, cfg)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/resend-verification",
		bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestHandleResendVerification_InvalidEmail(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := &MockAuthService{}
	cfg := &config.Config{}
	router := setupAuthTestRouter(mockSvc, cfg)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/resend-verification",
		bytes.NewBufferString(`{"email":"not-an-email"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestHandleResendVerification_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := &MockAuthService{
		resendVerificationFunc: func(ctx context.Context, email string) error {
			return errors.New("failed to send email")
		},
	}

	cfg := &config.Config{}
	router := setupAuthTestRouter(mockSvc, cfg)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/resend-verification",
		bytes.NewBufferString(`{"email":"test@test.com"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestHandleForgotPassword_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := &MockAuthService{
		forgotPasswordFunc: func(ctx context.Context, email string) error {
			return nil
		},
	}

	cfg := &config.Config{}
	router := setupAuthTestRouter(mockSvc, cfg)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/forgot-password",
		bytes.NewBufferString(`{"email":"test@test.com"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestHandleForgotPassword_UserNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := &MockAuthService{
		forgotPasswordFunc: func(ctx context.Context, email string) error {
			return nil
		},
	}

	cfg := &config.Config{}
	router := setupAuthTestRouter(mockSvc, cfg)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/forgot-password",
		bytes.NewBufferString(`{"email":"nonexistent@test.com"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 (don't leak info), got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestHandleForgotPassword_InvalidEmail(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := &MockAuthService{}
	cfg := &config.Config{}
	router := setupAuthTestRouter(mockSvc, cfg)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/forgot-password",
		bytes.NewBufferString(`{"email":"not-an-email"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestHandleForgotPassword_MissingEmail(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := &MockAuthService{}
	cfg := &config.Config{}
	router := setupAuthTestRouter(mockSvc, cfg)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/forgot-password",
		bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestHandleResetPassword_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := &MockAuthService{
		resetPasswordFunc: func(ctx context.Context, token, newPassword string) error {
			return nil
		},
	}

	cfg := &config.Config{}
	router := setupAuthTestRouter(mockSvc, cfg)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/reset-password",
		bytes.NewBufferString(`{"token":"valid-reset-token","new_password":"newpassword123"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestHandleResetPassword_InvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := &MockAuthService{
		resetPasswordFunc: func(ctx context.Context, token, newPassword string) error {
			return errors.New("invalid or expired token")
		},
	}

	cfg := &config.Config{}
	router := setupAuthTestRouter(mockSvc, cfg)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/reset-password",
		bytes.NewBufferString(`{"token":"expired-token","new_password":"newpassword123"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestHandleResetPassword_MissingFields(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{
			name: "missing token",
			body: `{"new_password":"newpassword123"}`,
		},
		{
			name: "missing new_password",
			body: `{"token":"valid-token"}`,
		},
		{
			name: "empty body",
			body: `{}`,
		},
		{
			name: "invalid JSON",
			body: `{invalid}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			mockSvc := &MockAuthService{}
			cfg := &config.Config{}
			router := setupAuthTestRouter(mockSvc, cfg)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/v1/auth/reset-password",
				bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("Expected status 400, got %d. Body: %s", w.Code, w.Body.String())
			}
		})
	}
}

func TestHandleSetup2FA_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := &MockAuthService{
		setupTwoFactorFunc: func(ctx context.Context, userID uuid.UUID) (*TwoFASetupResponse, error) {
			return &TwoFASetupResponse{
				Secret:          "TESTSECRET",
				QRCode:          "base64qrcode",
				ProvisioningURI: "otpauth://totp/test",
			}, nil
		},
	}

	cfg := &config.Config{}
	router := setupAuthTestRouter(mockSvc, cfg)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/2fa/setup", nil)
	req.Header.Set("X-User-ID", "1")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestHandleSetup2FA_MissingUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := &MockAuthService{}
	cfg := &config.Config{}
	router := setupAuthTestRouter(mockSvc, cfg)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/2fa/setup", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestHandleEnable2FA_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := &MockAuthService{
		enableTwoFactorFunc: func(ctx context.Context, userID uuid.UUID, code string) (*TwoFAEnableResponse, error) {
			return &TwoFAEnableResponse{
				Success:     true,
				BackupCodes: []string{"code1", "code2"},
			}, nil
		},
	}

	cfg := &config.Config{}
	router := setupAuthTestRouter(mockSvc, cfg)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/2fa/enable",
		bytes.NewBufferString(`{"code":"123456"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "1")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestHandleEnable2FA_MissingUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := &MockAuthService{}
	cfg := &config.Config{}
	router := setupAuthTestRouter(mockSvc, cfg)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/2fa/enable",
		bytes.NewBufferString(`{"code":"123456"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestHandleEnable2FA_InvalidCode(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := &MockAuthService{
		enableTwoFactorFunc: func(ctx context.Context, userID uuid.UUID, code string) (*TwoFAEnableResponse, error) {
			return nil, errors.New("invalid verification code")
		},
	}

	cfg := &config.Config{}
	router := setupAuthTestRouter(mockSvc, cfg)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/2fa/enable",
		bytes.NewBufferString(`{"code":"wrongcode"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "1")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestHandleEnable2FA_MissingCode(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := &MockAuthService{}
	cfg := &config.Config{}
	router := setupAuthTestRouter(mockSvc, cfg)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/2fa/enable",
		bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "1")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestHandleDisable2FA_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := &MockAuthService{
		disableTwoFactorFunc: func(ctx context.Context, userID uuid.UUID, code, backupCode string) error {
			return nil
		},
	}

	cfg := &config.Config{}
	router := setupAuthTestRouter(mockSvc, cfg)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/2fa/disable",
		bytes.NewBufferString(`{"code":"123456"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "1")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestHandleDisable2FA_MissingUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := &MockAuthService{}
	cfg := &config.Config{}
	router := setupAuthTestRouter(mockSvc, cfg)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/2fa/disable",
		bytes.NewBufferString(`{"code":"123456"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestHandleDisable2FA_InvalidCode(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := &MockAuthService{
		disableTwoFactorFunc: func(ctx context.Context, userID uuid.UUID, code, backupCode string) error {
			return errors.New("invalid verification code")
		},
	}

	cfg := &config.Config{}
	router := setupAuthTestRouter(mockSvc, cfg)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/2fa/disable",
		bytes.NewBufferString(`{"code":"wrongcode"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "1")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestHandleVerify2FA_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := &MockAuthService{
		verifyTwoFactorFunc: func(ctx context.Context, userID uuid.UUID, code string) (bool, error) {
			return true, nil
		},
	}

	cfg := &config.Config{}
	router := setupAuthTestRouter(mockSvc, cfg)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/2fa/verify",
		bytes.NewBufferString(`{"code":"123456"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "1")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected data object, got %v", resp["data"])
	}

	if data["valid"] != true {
		t.Errorf("Expected valid=true, got %v", data["valid"])
	}
}

func TestHandleVerify2FA_InvalidCode(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := &MockAuthService{
		verifyTwoFactorFunc: func(ctx context.Context, userID uuid.UUID, code string) (bool, error) {
			return false, nil
		},
	}

	cfg := &config.Config{}
	router := setupAuthTestRouter(mockSvc, cfg)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/2fa/verify",
		bytes.NewBufferString(`{"code":"wrongcode"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "1")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 (valid=false), got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestHandleVerify2FA_MissingUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := &MockAuthService{}
	cfg := &config.Config{}
	router := setupAuthTestRouter(mockSvc, cfg)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/2fa/verify",
		bytes.NewBufferString(`{"code":"123456"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestHandleVerify2FA_MissingCode(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := &MockAuthService{}
	cfg := &config.Config{}
	router := setupAuthTestRouter(mockSvc, cfg)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/2fa/verify",
		bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "1")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestHandleLoginWith2FA_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := &MockAuthService{
		loginWith2FAFunc: func(ctx context.Context, tempToken, code string) (*TokenResponse, error) {
			return &TokenResponse{
				AccessToken:  "new-access-token",
				RefreshToken: "new-refresh-token",
				ExpiresIn:    3600,
			}, nil
		},
	}

	cfg := &config.Config{}
	router := setupAuthTestRouter(mockSvc, cfg)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/2fa/login-verify",
		bytes.NewBufferString(`{"temp_token":"valid-temp-token","code":"123456"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestHandleLoginWith2FA_InvalidCode(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := &MockAuthService{
		loginWith2FAFunc: func(ctx context.Context, tempToken, code string) (*TokenResponse, error) {
			return nil, errors.New("invalid verification code")
		},
	}

	cfg := &config.Config{}
	router := setupAuthTestRouter(mockSvc, cfg)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/2fa/login-verify",
		bytes.NewBufferString(`{"temp_token":"valid-temp-token","code":"wrongcode"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestHandleLoginWith2FA_MissingTempToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := &MockAuthService{}
	cfg := &config.Config{}
	router := setupAuthTestRouter(mockSvc, cfg)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/2fa/login-verify",
		bytes.NewBufferString(`{"code":"123456"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestHandleLoginWith2FA_MissingFields(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{
			name: "missing code",
			body: `{"temp_token":"valid-temp-token"}`,
		},
		{
			name: "empty body",
			body: `{}`,
		},
		{
			name: "invalid JSON",
			body: `{invalid}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			mockSvc := &MockAuthService{}
			cfg := &config.Config{}
			router := setupAuthTestRouter(mockSvc, cfg)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/v1/auth/2fa/login-verify",
				bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("Expected status 400, got %d. Body: %s", w.Code, w.Body.String())
			}
		})
	}
}
