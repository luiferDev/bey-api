package auth

import (
	"github.com/golang-jwt/jwt/v5"
)

type TokenClaims struct {
	UserID uint   `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresIn    int64  `json:"expires_in"`
}

type LoginResponse struct {
	Requires2FA  bool   `json:"requires_2fa"`
	TempToken    string `json:"temp_token,omitempty"`
	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresIn    int64  `json:"expires_in,omitempty"`
}

type VerifyEmailRequest struct {
	Token string `json:"token" binding:"required"`
}

type ResendVerificationRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type ResetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

type TwoFASetupResponse struct {
	Secret          string `json:"secret"`
	QRCode          string `json:"qr_code"`
	ProvisioningURI string `json:"provisioning_uri"`
}

type TwoFAEnableRequest struct {
	Code string `json:"code" binding:"required"`
}

type TwoFAEnableResponse struct {
	Success     bool     `json:"success"`
	BackupCodes []string `json:"backup_codes"`
}

type TwoFADisableRequest struct {
	Code       string `json:"code"`
	BackupCode string `json:"backup_code"`
}

type TwoFAVerifyRequest struct {
	Code string `json:"code" binding:"required"`
}

type TwoFAVerifyResponse struct {
	Valid bool `json:"valid"`
}

type TwoFALoginVerifyRequest struct {
	TempToken string `json:"temp_token" binding:"required"`
	Code      string `json:"code" binding:"required"`
}

type TwoFALoginResponse struct {
	Requires2FA  bool   `json:"requires_2fa"`
	TempToken    string `json:"temp_token,omitempty"`
	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresIn    int64  `json:"expires_in,omitempty"`
}

// OAuth DTOs

// OAuthRedirectResponse contains the Google OAuth URL to redirect to
type OAuthRedirectResponse struct {
	URL string `json:"url"`
}

// OAuthCallbackRequest is the request for OAuth callback
type OAuthCallbackRequest struct {
	Code  string `form:"code" binding:"required"`
	State string `form:"state" binding:"required"`
}

// OAuthCallbackError represents an OAuth callback error
type OAuthCallbackError struct {
	Error       string `json:"error"`
	Description string `json:"error_description,omitempty"`
}
