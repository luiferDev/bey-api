package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"bey/internal/config"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// OAuthService handles Google OAuth2 authentication flow
type OAuthService struct {
	config       *config.Config
	oauth2Config *oauth2.Config
}

// GoogleUserInfo represents the user profile from Google
type GoogleUserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
	VerifiedEmail bool   `json:"verified_email"`
}

// NewOAuthService creates a new OAuthService with Google OAuth2 configuration
func NewOAuthService(cfg *config.Config) *OAuthService {
	oauthCfg := cfg.GetOAuthConfig()
	googleCfg := oauthCfg.Google

	oauth2Config := &oauth2.Config{
		ClientID:     googleCfg.ClientID,
		ClientSecret: googleCfg.ClientSecret,
		RedirectURL:  googleCfg.RedirectURL,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}

	return &OAuthService{
		config:       cfg,
		oauth2Config: oauth2Config,
	}
}

// GenerateAuthURL generates the Google OAuth2 authorization URL
// state parameter is used for CSRF protection
func (s *OAuthService) GenerateAuthURL(state string) (string, error) {
	authURL := s.oauth2Config.AuthCodeURL(state, oauth2.AccessTypeOnline, oauth2.ApprovalForce)
	return authURL, nil
}

// HandleCallback exchanges the authorization code for tokens and fetches user info
func (s *OAuthService) HandleCallback(ctx context.Context, code string) (*GoogleUserInfo, error) {
	// Exchange code for token
	token, err := s.oauth2Config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code for token: %w", err)
	}

	// Fetch user info from Google
	userInfo, err := s.fetchGoogleUserInfo(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user info: %w", err)
	}

	return userInfo, nil
}

// fetchGoogleUserInfo makes a request to Google's userinfo endpoint
func (s *OAuthService) fetchGoogleUserInfo(ctx context.Context, token *oauth2.Token) (*GoogleUserInfo, error) {
	client := s.oauth2Config.Client(ctx, token)

	userinfoURL := "https://www.googleapis.com/oauth2/v2/userinfo"
	resp, err := client.Get(userinfoURL)
	if err != nil {
		return nil, fmt.Errorf("failed to make request to userinfo: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return nil, fmt.Errorf("google userinfo returned status %d and failed to read body", resp.StatusCode)
		}
		return nil, fmt.Errorf("google userinfo returned status %d: %s", resp.StatusCode, string(body))
	}

	var userInfo GoogleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode userinfo response: %w", err)
	}

	return &userInfo, nil
}

// GetOAuth2Config returns the underlying oauth2.Config for token refresh
func (s *OAuthService) GetOAuth2Config() *oauth2.Config {
	return s.oauth2Config
}

// GenerateState generates a random state string for CSRF protection using crypto/rand
func GenerateState() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random state: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}
