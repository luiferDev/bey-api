package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"bey/internal/config"
	"bey/internal/modules/email"
	"bey/internal/modules/users"

	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthServiceInterface interface {
	Login(ctx context.Context, email, password string) (*LoginResponse, error)
	Refresh(ctx context.Context, refreshToken string) (*TokenResponse, error)
	Logout(ctx context.Context, refreshToken string) error
	ValidateToken(ctx context.Context, tokenString string) (*TokenClaims, error)
	SetupTwoFactor(ctx context.Context, userID uint) (*TwoFASetupResponse, error)
	EnableTwoFactor(ctx context.Context, userID uint, code string) (*TwoFAEnableResponse, error)
	DisableTwoFactor(ctx context.Context, userID uint, code, backupCode string) error
	VerifyTwoFactor(ctx context.Context, userID uint, code string) (bool, error)
	LoginWith2FA(ctx context.Context, tempToken, code string) (*TokenResponse, error)
}

type AuthService struct {
	db             *gorm.DB
	config         *config.Config
	emailService   *email.EmailService
	twoFAService   *TwoFAService
	tempTokens     map[string]tempTokenData
	tokenGenerator *TokenGenerator
}

type tempTokenData struct {
	UserID    uint
	ExpiresAt time.Time
}

func NewAuthService(db *gorm.DB, config *config.Config) *AuthService {
	return &AuthService{
		db:             db,
		config:         config,
		tokenGenerator: NewTokenGenerator(db, config),
	}
}

func NewAuthServiceWithEmail(db *gorm.DB, config *config.Config, emailSvc *email.EmailService) *AuthService {
	return &AuthService{
		db:             db,
		config:         config,
		emailService:   emailSvc,
		tokenGenerator: NewTokenGenerator(db, config),
	}
}

func NewAuthServiceWithTwoFA(db *gorm.DB, config *config.Config, emailSvc *email.EmailService, twoFASvc *TwoFAService) *AuthService {
	return &AuthService{
		db:             db,
		config:         config,
		emailService:   emailSvc,
		twoFAService:   twoFASvc,
		tempTokens:     make(map[string]tempTokenData),
		tokenGenerator: NewTokenGenerator(db, config),
	}
}

func NewAuthServiceWithRedis(db *gorm.DB, config *config.Config, redisClient *redis.Client) *AuthService {
	return &AuthService{
		db:             db,
		config:         config,
		tokenGenerator: NewTokenGeneratorWithRedis(db, config, redisClient),
	}
}

func (s *AuthService) Login(ctx context.Context, email, password string) (*LoginResponse, error) {
	userRepo := users.NewUserRepository(s.db)

	user, err := userRepo.FindByEmail(email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("invalid credentials")
	}

	if !user.Active {
		return nil, errors.New("user account is inactive")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, errors.New("invalid credentials")
	}

	if user.TwoFAEnabled {
		if s.tempTokens == nil {
			s.tempTokens = make(map[string]tempTokenData)
		}
		tempToken := s.generateTempToken(user.ID)
		return &LoginResponse{
			Requires2FA: true,
			TempToken:   tempToken,
		}, nil
	}

	tokenGenerator := s.tokenGenerator

	accessToken, expiresIn, err := tokenGenerator.GenerateAccessToken(user.ID, user.Email, user.Role)
	if err != nil {
		return nil, err
	}

	refreshToken, err := tokenGenerator.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}

	if err := tokenGenerator.StoreRefreshToken(refreshToken, user.ID); err != nil {
		return nil, err
	}

	return &LoginResponse{
		Requires2FA:  false,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    expiresIn,
	}, nil
}

func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*TokenResponse, error) {
	tokenGenerator := s.tokenGenerator

	storedToken, err := tokenGenerator.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, err
	}

	userRepo := users.NewUserRepository(s.db)
	user, err := userRepo.FindByID(storedToken.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("user not found")
	}

	newAccessToken, expiresIn, err := tokenGenerator.GenerateAccessToken(user.ID, user.Email, user.Role)
	if err != nil {
		return nil, err
	}

	newRefreshToken, err := tokenGenerator.RotateRefreshToken(tokenGenerator.HashToken(refreshToken))
	if err != nil {
		return nil, err
	}

	if err := tokenGenerator.StoreRefreshToken(newRefreshToken, storedToken.UserID); err != nil {
		return nil, err
	}

	return &TokenResponse{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    expiresIn,
	}, nil
}

func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	tokenGenerator := s.tokenGenerator
	tokenHash := tokenGenerator.HashToken(refreshToken)

	var token RefreshToken
	if err := s.db.Where("token = ?", tokenHash).First(&token).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("refresh token not found")
		}
		return err
	}

	token.Revoked = true
	return s.db.Save(&token).Error
}

func (s *AuthService) ValidateToken(ctx context.Context, tokenString string) (*TokenClaims, error) {
	return s.tokenGenerator.ValidateToken(tokenString)
}

func (s *AuthService) VerifyEmail(ctx context.Context, token string) error {
	userRepo := users.NewUserRepository(s.db)

	// Hash the token before lookup (same as reset password)
	hashedToken := email.HashToken(token)
	var user *users.User
	var err error
	user, err = userRepo.FindByVerificationToken(hashedToken)
	if err != nil {
		return errors.New("invalid token")
	}
	if user == nil {
		return errors.New("invalid token")
	}

	if user.EmailVerified {
		return errors.New("email already verified")
	}

	if !email.VerifyVerificationToken(user, token) {
		return errors.New("invalid or expired token")
	}

	user.EmailVerified = true
	user.VerificationToken = ""
	user.VerificationExpires = nil

	return userRepo.Update(user)
}

func (s *AuthService) ResendVerification(ctx context.Context, emailAddr string) error {
	userRepo := users.NewUserRepository(s.db)

	user, err := userRepo.FindByEmail(emailAddr)
	if err != nil {
		return err
	}
	if user == nil {
		return nil
	}

	if user.EmailVerified {
		return nil
	}

	token, err := email.GenerateToken()
	if err != nil {
		return err
	}

	expiresAt := time.Now().Add(24 * time.Hour)
	hashedToken := email.HashToken(token)

	user.VerificationToken = hashedToken
	user.VerificationExpires = &expiresAt

	if err := userRepo.Update(user); err != nil {
		return err
	}

	if s.emailService != nil {
		if err := s.emailService.SendVerificationEmail(user.Email, token); err != nil {
			return err
		}
	}

	return nil
}

func (s *AuthService) ForgotPassword(ctx context.Context, emailAddr string) error {
	userRepo := users.NewUserRepository(s.db)

	user, err := userRepo.FindByEmail(emailAddr)
	if err != nil {
		return err
	}
	if user == nil {
		return nil
	}

	token, err := email.GenerateToken()
	if err != nil {
		return err
	}

	expiresAt := time.Now().Add(1 * time.Hour)
	hashedToken := email.HashToken(token)

	user.ResetToken = hashedToken
	user.ResetExpires = &expiresAt

	if err := userRepo.Update(user); err != nil {
		return err
	}

	if s.emailService != nil {
		if err := s.emailService.SendPasswordResetEmail(user.Email, token); err != nil {
			return err
		}
	}

	return nil
}

func (s *AuthService) ResetPassword(ctx context.Context, token, newPassword string) error {
	userRepo := users.NewUserRepository(s.db)

	hashedToken := email.HashToken(token)
	user, err := userRepo.FindByResetToken(hashedToken)
	if err != nil {
		return errors.New("invalid token")
	}
	if user == nil {
		return errors.New("invalid token")
	}

	if !email.VerifyResetToken(user, token) {
		return errors.New("invalid or expired token")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user.Password = string(hashedPassword)
	user.ResetToken = ""
	user.ResetExpires = nil

	return userRepo.Update(user)
}

func (s *AuthService) SetupTwoFactor(ctx context.Context, userID uint) (*TwoFASetupResponse, error) {
	userRepo := users.NewUserRepository(s.db)
	user, err := userRepo.FindByID(userID)
	if err != nil {
		return nil, errors.New("user not found")
	}
	if user == nil {
		return nil, errors.New("user not found")
	}

	if user.TwoFAEnabled {
		return nil, errors.New("two-factor authentication already enabled")
	}

	if s.twoFAService == nil {
		s.twoFAService = NewTwoFAService()
	}

	secret, err := s.twoFAService.GenerateSecret(user.Email)
	if err != nil {
		return nil, errors.New("failed to generate secret")
	}

	qrCode, err := s.twoFAService.GenerateQRCode(secret, user.Email)
	if err != nil {
		return nil, errors.New("failed to generate QR code")
	}

	qrCodeBase64 := base64.StdEncoding.EncodeToString(qrCode)

	provisioningURI := fmt.Sprintf("otpauth://totp/Bey API:%s?secret=%s&issuer=Bey API&algorithm=SHA1&digits=6&period=30",
		user.Email, secret)

	user.TwoFASecret = secret
	if err := userRepo.Update(user); err != nil {
		return nil, errors.New("failed to store temporary secret")
	}

	return &TwoFASetupResponse{
		Secret:          secret,
		QRCode:          qrCodeBase64,
		ProvisioningURI: provisioningURI,
	}, nil
}

func (s *AuthService) EnableTwoFactor(ctx context.Context, userID uint, code string) (*TwoFAEnableResponse, error) {
	userRepo := users.NewUserRepository(s.db)
	user, err := userRepo.FindByID(userID)
	if err != nil {
		return nil, errors.New("user not found")
	}
	if user == nil {
		return nil, errors.New("user not found")
	}

	if user.TwoFAEnabled {
		return nil, errors.New("two-factor authentication already enabled")
	}

	if user.TwoFASecret == "" {
		return nil, errors.New("please run setup first")
	}

	if s.twoFAService == nil {
		s.twoFAService = NewTwoFAService()
	}

	if !s.twoFAService.VerifyCode(user.TwoFASecret, code) {
		return nil, errors.New("invalid verification code")
	}

	backupCodes, err := s.twoFAService.GenerateBackupCodes(10)
	if err != nil {
		return nil, errors.New("failed to generate backup codes")
	}

	hashedCodes := make([]string, len(backupCodes))
	for i, code := range backupCodes {
		hashedCodes[i] = s.twoFAService.HashBackupCode(code)
	}

	hashedCodesJSON, err := s.twoFAService.SerializeBackupCodes(hashedCodes)
	if err != nil {
		return nil, errors.New("failed to process backup codes")
	}

	user.TwoFAEnabled = true
	user.TwoFABackupCodes = hashedCodesJSON

	if err := userRepo.Update(user); err != nil {
		return nil, errors.New("failed to enable two-factor authentication")
	}

	return &TwoFAEnableResponse{
		Success:     true,
		BackupCodes: backupCodes,
	}, nil
}

func (s *AuthService) DisableTwoFactor(ctx context.Context, userID uint, code, backupCode string) error {
	userRepo := users.NewUserRepository(s.db)
	user, err := userRepo.FindByID(userID)
	if err != nil {
		return errors.New("user not found")
	}
	if user == nil {
		return errors.New("user not found")
	}

	if !user.TwoFAEnabled {
		return errors.New("two-factor authentication not enabled")
	}

	if s.twoFAService == nil {
		s.twoFAService = NewTwoFAService()
	}

	validCode := false
	if code != "" {
		validCode = s.twoFAService.VerifyCode(user.TwoFASecret, code)
	} else if backupCode != "" {
		hashedCodes, err := s.twoFAService.DeserializeBackupCodes(user.TwoFABackupCodes)
		if err == nil {
			validCode = s.twoFAService.VerifyBackupCode(hashedCodes, backupCode)
			if validCode {
				remainingCodes := s.twoFAService.RemoveUsedBackupCode(hashedCodes, backupCode)
				user.TwoFABackupCodes, _ = s.twoFAService.SerializeBackupCodes(remainingCodes)
			}
		}
	}

	if !validCode {
		return errors.New("invalid verification code")
	}

	user.TwoFAEnabled = false
	user.TwoFASecret = ""
	user.TwoFABackupCodes = ""

	return userRepo.Update(user)
}

func (s *AuthService) VerifyTwoFactor(ctx context.Context, userID uint, code string) (bool, error) {
	userRepo := users.NewUserRepository(s.db)
	user, err := userRepo.FindByID(userID)
	if err != nil {
		return false, errors.New("user not found")
	}
	if user == nil {
		return false, errors.New("user not found")
	}

	if !user.TwoFAEnabled {
		return false, errors.New("two-factor authentication not enabled")
	}

	if s.twoFAService == nil {
		s.twoFAService = NewTwoFAService()
	}

	validCode := s.twoFAService.VerifyCode(user.TwoFASecret, code)
	if !validCode {
		hashedCodes, err := s.twoFAService.DeserializeBackupCodes(user.TwoFABackupCodes)
		if err == nil {
			validCode = s.twoFAService.VerifyBackupCode(hashedCodes, code)
			if validCode {
				remainingCodes := s.twoFAService.RemoveUsedBackupCode(hashedCodes, code)
				user.TwoFABackupCodes, _ = s.twoFAService.SerializeBackupCodes(remainingCodes)
				_ = userRepo.Update(user)
			}
		}
	}

	return validCode, nil
}

func (s *AuthService) generateTempToken(userID uint) string {
	if s.tempTokens == nil {
		s.tempTokens = make(map[string]tempTokenData)
	}

	tokenBytes := make([]byte, 32)
	_, _ = rand.Read(tokenBytes)
	tempToken := base64.StdEncoding.EncodeToString(tokenBytes)

	s.tempTokens[tempToken] = tempTokenData{
		UserID:    userID,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}

	return tempToken
}

func (s *AuthService) validateTempToken(tempToken string) (uint, error) {
	data, exists := s.tempTokens[tempToken]
	if !exists {
		return 0, errors.New("invalid temp token")
	}

	if time.Now().After(data.ExpiresAt) {
		delete(s.tempTokens, tempToken)
		return 0, errors.New("temp token expired")
	}

	delete(s.tempTokens, tempToken)
	return data.UserID, nil
}

func (s *AuthService) LoginWith2FA(ctx context.Context, tempToken, code string) (*TokenResponse, error) {
	userID, err := s.validateTempToken(tempToken)
	if err != nil {
		return nil, err
	}

	userRepo := users.NewUserRepository(s.db)
	user, err := userRepo.FindByID(userID)
	if err != nil {
		return nil, errors.New("user not found")
	}
	if user == nil {
		return nil, errors.New("user not found")
	}

	validCode, err := s.VerifyTwoFactor(ctx, userID, code)
	if err != nil {
		return nil, err
	}
	if !validCode {
		return nil, errors.New("invalid verification code")
	}

	tokenGenerator := s.tokenGenerator

	accessToken, expiresIn, err := tokenGenerator.GenerateAccessToken(user.ID, user.Email, user.Role)
	if err != nil {
		return nil, err
	}

	refreshToken, err := tokenGenerator.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}

	if err := tokenGenerator.StoreRefreshToken(refreshToken, user.ID); err != nil {
		return nil, err
	}

	return &TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    expiresIn,
	}, nil
}

// LoginWithGoogle handles OAuth2 Google login - creates or updates user
func (s *AuthService) LoginWithGoogle(ctx context.Context, googleUser *GoogleUserInfo) (*TokenResponse, error) {
	userRepo := users.NewUserRepository(s.db)

	// Try to find existing user by email
	existingUser, err := userRepo.FindByEmail(googleUser.Email)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	var user *users.User

	if existingUser != nil {
		// User exists - update with Google info
		user = existingUser
		user.OAuthProvider = "google"
		user.OAuthProviderID = googleUser.ID
		if googleUser.Picture != "" {
			user.AvatarURL = googleUser.Picture
		}
		if googleUser.GivenName != "" {
			user.FirstName = googleUser.GivenName
		}
		if googleUser.FamilyName != "" {
			user.LastName = googleUser.FamilyName
		}
		// Mark email as verified since Google verified it
		user.EmailVerified = true
		user.Active = true

		if err := userRepo.Update(user); err != nil {
			return nil, err
		}
	} else {
		// Create new user from Google profile
		user = &users.User{
			Email:           googleUser.Email,
			FirstName:       googleUser.GivenName,
			LastName:        googleUser.FamilyName,
			AvatarURL:       googleUser.Picture,
			Role:            "customer",
			Active:          true,
			EmailVerified:   true,
			OAuthProvider:   "google",
			OAuthProviderID: googleUser.ID,
		}

		if err := userRepo.Create(user); err != nil {
			return nil, err
		}
	}

	// Generate JWT tokens
	tokenGenerator := s.tokenGenerator

	accessToken, expiresIn, err := tokenGenerator.GenerateAccessToken(user.ID, user.Email, user.Role)
	if err != nil {
		return nil, err
	}

	refreshToken, err := tokenGenerator.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}

	if err := tokenGenerator.StoreRefreshToken(refreshToken, user.ID); err != nil {
		return nil, err
	}

	return &TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    expiresIn,
	}, nil
}
