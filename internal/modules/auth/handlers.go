package auth

import (
	"net/http"
	"time"

	"bey/internal/config"
	"bey/internal/shared/response"

	"github.com/gin-gonic/gin"
)

// @Summary User login
// @Description Authenticate user and return JWT tokens
// @Tags Auth
// @Accept json
// @Produce json
// @Param credentials body LoginRequest true "Login credentials"
// @Success 200 {object} LoginResponse
// @Router /api/v1/auth/login [post]
func HandleLogin(authService AuthServiceInterface, cfg *config.Config, responseHandler *response.ResponseHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}

		resp, err := authService.Login(c.Request.Context(), req.Email, req.Password)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		if resp.Requires2FA {
			responseHandler.Success(c, resp)
			return
		}

		SetAccessTokenCookie(c.Writer, resp.AccessToken, 15*time.Minute, cfg)
		SetRefreshTokenCookie(c.Writer, resp.RefreshToken, 7*24*time.Hour, cfg)

		responseHandler.Success(c, resp)
	}
}

// @Summary Refresh token
// @Description Refresh access token using refresh token cookie
// @Tags Auth
// @Produce json
// @Success 200 {object} TokenResponse
// @Router /api/v1/auth/refresh [post]
func HandleRefresh(authService AuthServiceInterface, cfg *config.Config, responseHandler *response.ResponseHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		refreshToken, err := GetRefreshTokenCookie(c.Request)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "refresh token required"})
			return
		}

		tokens, err := authService.Refresh(c.Request.Context(), refreshToken)
		if err != nil {
			DeleteRefreshTokenCookie(c.Writer, cfg)
			DeleteAccessTokenCookie(c.Writer, cfg)
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		SetAccessTokenCookie(c.Writer, tokens.AccessToken, 15*time.Minute, cfg)
		SetRefreshTokenCookie(c.Writer, tokens.RefreshToken, 7*24*time.Hour, cfg)

		responseHandler.Success(c, tokens)
	}
}

// @Summary Logout
// @Description Logout user and clear cookies
// @Tags Auth
// @Produce json
// @Success 200 {object} map[string]string
// @Router /api/v1/auth/logout [post]
func HandleLogout(authService AuthServiceInterface, cfg *config.Config, responseHandler *response.ResponseHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		refreshToken, err := GetRefreshTokenCookie(c.Request)
		if err == nil {
			_ = authService.Logout(c.Request.Context(), refreshToken)
		}

		DeleteRefreshTokenCookie(c.Writer, cfg)
		DeleteAccessTokenCookie(c.Writer, cfg)

		responseHandler.Success(c, gin.H{"message": "logged out"})
	}
}

// @Summary Verify email
// @Description Verify user email with token
// @Tags Auth
// @Accept json
// @Produce json
// @Param token body VerifyEmailRequest true "Verification token"
// @Success 200 {object} map[string]string
// @Router /api/v1/auth/verify-email [post]
func HandleVerifyEmail(authService AuthServiceInterface, responseHandler *response.ResponseHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req VerifyEmailRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "token is required"})
			return
		}

		if err := authService.VerifyEmail(c.Request.Context(), req.Token); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		responseHandler.Success(c, gin.H{"message": "email verified successfully"})
	}
}

// @Summary Resend verification email
// @Description Resend verification email to user
// @Tags Auth
// @Accept json
// @Produce json
// @Param email body ResendVerificationRequest true "Email address"
// @Success 200 {object} map[string]string
// @Router /api/v1/auth/resend-verification [post]
func HandleResendVerification(authService AuthServiceInterface, responseHandler *response.ResponseHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req ResendVerificationRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "email is required"})
			return
		}

		if err := authService.ResendVerification(c.Request.Context(), req.Email); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		responseHandler.Success(c, gin.H{"message": "verification email sent if account exists"})
	}
}

// @Summary Forgot password
// @Description Send password reset email
// @Tags Auth
// @Accept json
// @Produce json
// @Param email body ForgotPasswordRequest true "Email address"
// @Success 200 {object} map[string]string
// @Router /api/v1/auth/forgot-password [post]
func HandleForgotPassword(authService AuthServiceInterface, responseHandler *response.ResponseHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req ForgotPasswordRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "email is required"})
			return
		}

		if err := authService.ForgotPassword(c.Request.Context(), req.Email); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		responseHandler.Success(c, gin.H{"message": "password reset email sent if account exists"})
	}
}

// @Summary Reset password
// @Description Reset password using token from email
// @Tags Auth
// @Accept json
// @Produce json
// @Param credentials body ResetPasswordRequest true "Token and new password"
// @Success 200 {object} map[string]string
// @Router /api/v1/auth/reset-password [post]
func HandleResetPassword(authService AuthServiceInterface, responseHandler *response.ResponseHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req ResetPasswordRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "token and new_password are required"})
			return
		}

		if err := authService.ResetPassword(c.Request.Context(), req.Token, req.NewPassword); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		responseHandler.Success(c, gin.H{"message": "password reset successfully"})
	}
}

// @Summary Setup 2FA
// @Description Setup two-factor authentication (requires auth)
// @Tags Auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} TwoFASetupResponse
// @Router /api/v1/auth/2fa/setup [post]
func HandleSetup2FA(authService AuthServiceInterface, responseHandler *response.ResponseHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		userIDUint, ok := userID.(uint)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user ID"})
			return
		}

		resp, err := authService.SetupTwoFactor(c.Request.Context(), userIDUint)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		responseHandler.Success(c, resp)
	}
}

// @Summary Enable 2FA
// @Description Enable two-factor authentication with code (requires auth)
// @Tags Auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param code body TwoFAEnableRequest true "TOTP code"
// @Success 200 {object} TwoFAEnableResponse
// @Router /api/v1/auth/2fa/enable [post]
func HandleEnable2FA(authService AuthServiceInterface, responseHandler *response.ResponseHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		userIDUint, ok := userID.(uint)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user ID"})
			return
		}

		var req TwoFAEnableRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "code is required"})
			return
		}

		resp, err := authService.EnableTwoFactor(c.Request.Context(), userIDUint, req.Code)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		responseHandler.Success(c, resp)
	}
}

// @Summary Disable 2FA
// @Description Disable two-factor authentication (requires auth)
// @Tags Auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param credentials body TwoFADisableRequest true "TOTP code or backup code"
// @Success 200 {object} map[string]string
// @Router /api/v1/auth/2fa/disable [post]
func HandleDisable2FA(authService AuthServiceInterface, responseHandler *response.ResponseHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		userIDUint, ok := userID.(uint)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user ID"})
			return
		}

		var req TwoFADisableRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "code or backup_code is required"})
			return
		}

		if err := authService.DisableTwoFactor(c.Request.Context(), userIDUint, req.Code, req.BackupCode); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		responseHandler.Success(c, gin.H{"message": "two-factor authentication disabled"})
	}
}

// @Summary Verify 2FA
// @Description Verify two-factor authentication code (requires auth)
// @Tags Auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param code body TwoFAVerifyRequest true "TOTP code"
// @Success 200 {object} TwoFAVerifyResponse
// @Router /api/v1/auth/2fa/verify [post]
func HandleVerify2FA(authService AuthServiceInterface, responseHandler *response.ResponseHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		userIDUint, ok := userID.(uint)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user ID"})
			return
		}

		var req TwoFAVerifyRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "code is required"})
			return
		}

		valid, err := authService.VerifyTwoFactor(c.Request.Context(), userIDUint, req.Code)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		responseHandler.Success(c, TwoFAVerifyResponse{Valid: valid})
	}
}

// @Summary Login with 2FA
// @Description Verify 2FA during login with temp token
// @Tags Auth
// @Accept json
// @Produce json
// @Param credentials body TwoFALoginVerifyRequest true "Temp token and TOTP code"
// @Success 200 {object} LoginResponse
// @Router /api/v1/auth/2fa/login-verify [post]
func HandleLoginWith2FA(authService AuthServiceInterface, cfg *config.Config, responseHandler *response.ResponseHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req TwoFALoginVerifyRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "temp_token and code are required"})
			return
		}

		tokens, err := authService.LoginWith2FA(c.Request.Context(), req.TempToken, req.Code)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		SetAccessTokenCookie(c.Writer, tokens.AccessToken, 15*time.Minute, cfg)
		SetRefreshTokenCookie(c.Writer, tokens.RefreshToken, 7*24*time.Hour, cfg)

		responseHandler.Success(c, tokens)
	}
}

// @Summary Google OAuth login
// @Description Initiate Google OAuth2 login flow
// @Tags Auth
// @Produce json
// @Success 200 {object} OAuthRedirectResponse
// @Router /api/v1/auth/google [get]
func HandleGoogleOAuth(cfg *config.Config, responseHandler *response.ResponseHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		oauthService := NewOAuthService(cfg)

		state, err := GenerateState()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate state"})
			return
		}

		c.SetCookie("oauth_state", state, 300, "/", "", false, true)

		authURL, err := oauthService.GenerateAuthURL(state)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate auth URL"})
			return
		}

		responseHandler.Success(c, OAuthRedirectResponse{URL: authURL})
	}
}

// @Summary Google OAuth callback
// @Description Handle Google OAuth2 callback
// @Tags Auth
// @Produce json
// @Param state query string true "OAuth state"
// @Param code query string true "Authorization code"
// @Success 200 {object} LoginResponse
// @Router /api/v1/auth/google/callback [get]
func HandleGoogleOAuthCallback(authService *AuthService, cfg *config.Config, responseHandler *response.ResponseHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		state := c.Query("state")
		oauthStateCookie, err := c.Cookie("oauth_state")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing state cookie"})
			return
		}

		if state != oauthStateCookie {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid state"})
			return
		}

		code := c.Query("code")
		if code == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing code"})
			return
		}

		oauthService := NewOAuthService(cfg)
		userInfo, err := oauthService.HandleCallback(c.Request.Context(), code)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to exchange code"})
			return
		}

		resp, err := authService.LoginWithGoogle(c.Request.Context(), userInfo)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		SetAccessTokenCookie(c.Writer, resp.AccessToken, 15*time.Minute, cfg)
		SetRefreshTokenCookie(c.Writer, resp.RefreshToken, 7*24*time.Hour, cfg)

		responseHandler.Success(c, resp)
	}
}
