package auth

import (
	"bey/internal/config"
	"bey/internal/shared/response"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine, authService *AuthService, cfg *config.Config) {
	responseHandler := response.NewResponseHandler()

	auth := r.Group("/api/v1/auth")
	{
		auth.POST("/login", HandleLogin(authService, cfg, responseHandler))
		auth.POST("/refresh", HandleRefresh(authService, cfg, responseHandler))
		auth.POST("/logout", HandleLogout(authService, cfg, responseHandler))
		auth.POST("/verify-email", HandleVerifyEmail(authService, responseHandler))
		auth.POST("/resend-verification", HandleResendVerification(authService, responseHandler))
		auth.POST("/forgot-password", HandleForgotPassword(authService, responseHandler))
		auth.POST("/reset-password", HandleResetPassword(authService, responseHandler))
		auth.POST("/2fa/setup", HandleSetup2FA(authService, responseHandler))
		auth.POST("/2fa/enable", HandleEnable2FA(authService, responseHandler))
		auth.POST("/2fa/disable", HandleDisable2FA(authService, responseHandler))
		auth.POST("/2fa/verify", HandleVerify2FA(authService, responseHandler))
		auth.POST("/2fa/login-verify", HandleLoginWith2FA(authService, cfg, responseHandler))
		auth.GET("/google", HandleGoogleOAuth(cfg, responseHandler))
		auth.GET("/google/callback", HandleGoogleOAuthCallback(authService, cfg, responseHandler))
	}
}
