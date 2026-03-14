package auth

import (
	"net/http"
	"time"

	"bey/internal/config"
	"bey/internal/shared/response"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine, authService *AuthService, cfg *config.Config) {
	responseHandler := response.NewResponseHandler()
	authMiddleware := NewAuthMiddleware(authService, cfg)

	auth := r.Group("/api/v1/auth")
	{
		auth.POST("/login", func(c *gin.Context) {
			var req LoginRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "invalid request",
				})
				return
			}

			resp, err := authService.Login(c.Request.Context(), req.Email, req.Password)
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": err.Error(),
				})
				return
			}

			if resp.Requires2FA {
				responseHandler.Success(c, resp)
				return
			}

			SetAccessTokenCookie(c.Writer, resp.AccessToken, 15*time.Minute, cfg)
			SetRefreshTokenCookie(c.Writer, resp.RefreshToken, 7*24*time.Hour, cfg)

			responseHandler.Success(c, resp)
		})

		auth.POST("/refresh", func(c *gin.Context) {
			refreshToken, err := GetRefreshTokenCookie(c.Request)
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": "refresh token required",
				})
				return
			}

			tokens, err := authService.Refresh(c.Request.Context(), refreshToken)
			if err != nil {
				DeleteRefreshTokenCookie(c.Writer, cfg)
				DeleteAccessTokenCookie(c.Writer, cfg)
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": err.Error(),
				})
				return
			}

			SetAccessTokenCookie(c.Writer, tokens.AccessToken, 15*time.Minute, cfg)
			SetRefreshTokenCookie(c.Writer, tokens.RefreshToken, 7*24*time.Hour, cfg)

			responseHandler.Success(c, tokens)
		})

		auth.POST("/logout", func(c *gin.Context) {
			refreshToken, err := GetRefreshTokenCookie(c.Request)
			if err == nil {
				_ = authService.Logout(c.Request.Context(), refreshToken)
			}

			DeleteRefreshTokenCookie(c.Writer, cfg)
			DeleteAccessTokenCookie(c.Writer, cfg)

			responseHandler.Success(c, gin.H{"message": "logged out"})
		})

		auth.POST("/verify-email", func(c *gin.Context) {
			var req VerifyEmailRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "token is required",
				})
				return
			}

			if err := authService.VerifyEmail(c.Request.Context(), req.Token); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": err.Error(),
				})
				return
			}

			responseHandler.Success(c, gin.H{"message": "email verified successfully"})
		})

		auth.POST("/resend-verification", func(c *gin.Context) {
			var req ResendVerificationRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "email is required",
				})
				return
			}

			if err := authService.ResendVerification(c.Request.Context(), req.Email); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": err.Error(),
				})
				return
			}

			responseHandler.Success(c, gin.H{"message": "verification email sent if account exists"})
		})

		auth.POST("/forgot-password", func(c *gin.Context) {
			var req ForgotPasswordRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "email is required",
				})
				return
			}

			if err := authService.ForgotPassword(c.Request.Context(), req.Email); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": err.Error(),
				})
				return
			}

			responseHandler.Success(c, gin.H{"message": "password reset email sent if account exists"})
		})

		auth.POST("/reset-password", func(c *gin.Context) {
			var req ResetPasswordRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "token and new_password are required",
				})
				return
			}

			if err := authService.ResetPassword(c.Request.Context(), req.Token, req.NewPassword); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": err.Error(),
				})
				return
			}

			responseHandler.Success(c, gin.H{"message": "password reset successfully"})
		})

		auth.POST("/2fa/setup", func(c *gin.Context) {
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
				c.JSON(http.StatusBadRequest, gin.H{
					"error": err.Error(),
				})
				return
			}

			responseHandler.Success(c, resp)
		})

		auth.POST("/2fa/enable", func(c *gin.Context) {
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
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "code is required",
				})
				return
			}

			resp, err := authService.EnableTwoFactor(c.Request.Context(), userIDUint, req.Code)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": err.Error(),
				})
				return
			}

			responseHandler.Success(c, resp)
		})

		auth.POST("/2fa/disable", func(c *gin.Context) {
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
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "code or backup_code is required",
				})
				return
			}

			if err := authService.DisableTwoFactor(c.Request.Context(), userIDUint, req.Code, req.BackupCode); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": err.Error(),
				})
				return
			}

			responseHandler.Success(c, gin.H{"message": "two-factor authentication disabled"})
		})

		auth.POST("/2fa/verify", func(c *gin.Context) {
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
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "code is required",
				})
				return
			}

			valid, err := authService.VerifyTwoFactor(c.Request.Context(), userIDUint, req.Code)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": err.Error(),
				})
				return
			}

			responseHandler.Success(c, TwoFAVerifyResponse{Valid: valid})
		})

		auth.POST("/2fa/login-verify", func(c *gin.Context) {
			var req TwoFALoginVerifyRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "temp_token and code are required",
				})
				return
			}

			tokens, err := authService.LoginWith2FA(c.Request.Context(), req.TempToken, req.Code)
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": err.Error(),
				})
				return
			}

			SetAccessTokenCookie(c.Writer, tokens.AccessToken, 15*time.Minute, cfg)
			SetRefreshTokenCookie(c.Writer, tokens.RefreshToken, 7*24*time.Hour, cfg)

			responseHandler.Success(c, tokens)
		})

		// OAuth2 Google routes
		auth.GET("/google", func(c *gin.Context) {
			oauthService := NewOAuthService(cfg)

			// Generate state for CSRF protection
			state, err := GenerateState()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "failed to generate state",
				})
				return
			}

			// Store state in cookie for validation on callback
			c.SetCookie("oauth_state", state, 300, "/", "", false, true)

			// Generate auth URL
			authURL, err := oauthService.GenerateAuthURL(state)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "failed to generate auth URL",
				})
				return
			}

			responseHandler.Success(c, OAuthRedirectResponse{URL: authURL})
		})

		auth.GET("/google/callback", func(c *gin.Context) {
			// Get state from query and cookie
			state := c.Query("state")
			oauthStateCookie, err := c.Cookie("oauth_state")
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "missing state cookie",
				})
				return
			}

			// Validate state (CSRF protection)
			if state != oauthStateCookie {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "invalid state parameter",
				})
				return
			}

			// Clear state cookie
			c.SetCookie("oauth_state", "", -1, "/", "", false, true)

			// Get authorization code
			code := c.Query("code")
			if code == "" {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "authorization code not provided",
				})
				return
			}

			// Handle OAuth callback
			oauthService := NewOAuthService(cfg)
			googleUser, err := oauthService.HandleCallback(c.Request.Context(), code)
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": "failed to handle OAuth callback: " + err.Error(),
				})
				return
			}

			// Login or create user with Google info
			tokens, err := authService.LoginWithGoogle(c.Request.Context(), googleUser)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "failed to login with Google: " + err.Error(),
				})
				return
			}

			// Set JWT cookies
			SetAccessTokenCookie(c.Writer, tokens.AccessToken, 15*time.Minute, cfg)
			SetRefreshTokenCookie(c.Writer, tokens.RefreshToken, 7*24*time.Hour, cfg)

			// Redirect to frontend with success
			// Frontend should handle this redirect
			frontendURL := cfg.App.StaticPath
			if frontendURL == "" {
				frontendURL = "http://localhost:3000"
			}
			c.Redirect(http.StatusFound, frontendURL+"/auth/callback")
		})

		protected := auth.Group("")
		protected.Use(authMiddleware.RequireAuth())
		{
			protected.POST("/validate", func(c *gin.Context) {
				userID, _ := c.Get("user_id")
				userRole, _ := c.Get("user_role")
				userEmail, _ := c.Get("user_email")

				responseHandler.Success(c, gin.H{
					"user_id": userID,
					"role":    userRole,
					"email":   userEmail,
				})
			})
		}
	}
}
