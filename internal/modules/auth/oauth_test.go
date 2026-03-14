package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bey/internal/config"
)

// Mock OAuth2 server for testing
type mockOAuth2Server struct {
	*httptest.Server
	tokenEndpoint string
	userInfo      GoogleUserInfo
}

func newMockOAuth2Server(userInfo GoogleUserInfo) *mockOAuth2Server {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)

	// Token endpoint
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token":  "mock_access_token",
			"token_type":    "Bearer",
			"expires_in":    3600,
			"refresh_token": "mock_refresh_token",
		})
	})

	// Userinfo endpoint
	mux.HandleFunc("/userinfo", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(userInfo)
	})

	return &mockOAuth2Server{
		Server:        server,
		tokenEndpoint: server.URL + "/token",
		userInfo:      userInfo,
	}
}

func TestOAuthService_GenerateAuthURL(t *testing.T) {
	cfg := createTestOAuthConfig()
	svc := NewOAuthService(cfg)

	tests := []struct {
		name         string
		state        string
		wantHTTPS    bool
		wantContains []string
	}{
		{
			name:      "generates valid Google OAuth URL",
			state:     "test-state-123",
			wantHTTPS: true,
			wantContains: []string{
				"accounts.google.com",
				"oauth2/auth",
				"test-state-123",
				"client_id=test-client-id",
			},
		},
		{
			name:      "URL contains required scopes",
			state:     "another-state",
			wantHTTPS: true,
			wantContains: []string{
				"scope=",
				"userinfo.email",
				"userinfo.profile",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, err := svc.GenerateAuthURL(tt.state)
			if err != nil {
				t.Fatalf("GenerateAuthURL() unexpected error: %v", err)
			}

			if url == "" {
				t.Error("GenerateAuthURL() returned empty URL")
			}

			// Check HTTPS
			if tt.wantHTTPS && len(url) > 0 {
				if url[:8] != "https://" {
					t.Errorf("GenerateAuthURL() should return HTTPS URL, got: %s", url[:7])
				}
			}

			// Check contains
			for _, contains := range tt.wantContains {
				if !containsString(url, contains) {
					t.Errorf("GenerateAuthURL() URL should contain %s, got: %s", contains, url)
				}
			}
		})
	}
}

func TestOAuthService_GenerateAuthURL_EmptyState(t *testing.T) {
	cfg := createTestOAuthConfig()
	svc := NewOAuthService(cfg)

	url, err := svc.GenerateAuthURL("")
	if err != nil {
		t.Fatalf("GenerateAuthURL() with empty state unexpected error: %v", err)
	}

	if url == "" {
		t.Error("GenerateAuthURL() returned empty URL")
	}
}

func TestGenerateState(t *testing.T) {
	tests := []struct {
		name       string
		wantLength int
	}{
		{
			name:       "generates random state",
			wantLength: 64, // 32 bytes = 64 hex chars
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state, err := GenerateState()
			if err != nil {
				t.Fatalf("GenerateState() unexpected error: %v", err)
			}

			if len(state) != tt.wantLength {
				t.Errorf("GenerateState() length = %d, want %d", len(state), tt.wantLength)
			}
		})
	}
}

func TestGenerateState_Uniqueness(t *testing.T) {
	states := make(map[string]bool)

	for i := 0; i < 100; i++ {
		state, err := GenerateState()
		if err != nil {
			t.Fatalf("GenerateState() unexpected error: %v", err)
		}

		if states[state] {
			t.Errorf("GenerateState() produced duplicate state: %s", state)
		}
		states[state] = true
	}
}

func TestGoogleUserInfo_Fields(t *testing.T) {
	userInfo := GoogleUserInfo{
		ID:            "123456789",
		Email:         "test@example.com",
		Name:          "Test User",
		GivenName:     "Test",
		FamilyName:    "User",
		Picture:       "https://example.com/photo.jpg",
		VerifiedEmail: true,
	}

	tests := []struct {
		name  string
		check func(*GoogleUserInfo) bool
	}{
		{
			name:  "ID is set",
			check: func(u *GoogleUserInfo) bool { return u.ID == "123456789" },
		},
		{
			name:  "Email is set",
			check: func(u *GoogleUserInfo) bool { return u.Email == "test@example.com" },
		},
		{
			name:  "GivenName is set",
			check: func(u *GoogleUserInfo) bool { return u.GivenName == "Test" },
		},
		{
			name:  "FamilyName is set",
			check: func(u *GoogleUserInfo) bool { return u.FamilyName == "User" },
		},
		{
			name:  "Picture is set",
			check: func(u *GoogleUserInfo) bool { return u.Picture == "https://example.com/photo.jpg" },
		},
		{
			name:  "VerifiedEmail is true",
			check: func(u *GoogleUserInfo) bool { return u.VerifiedEmail == true },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.check(&userInfo) {
				t.Errorf("GoogleUserInfo %s check failed", tt.name)
			}
		})
	}
}

func TestOAuthRedirectResponse_JSON(t *testing.T) {
	resp := OAuthRedirectResponse{
		URL: "https://accounts.google.com/oauth/auth?state=test",
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("JSONMarshal() unexpected error: %v", err)
	}

	var decoded OAuthRedirectResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("JSONUnmarshal() unexpected error: %v", err)
	}

	if decoded.URL != resp.URL {
		t.Errorf("JSON roundtrip failed: got %s, want %s", decoded.URL, resp.URL)
	}
}

// Helper functions

func createTestOAuthConfig() *config.Config {
	return &config.Config{
		App: config.AppConfig{
			Host: "localhost",
			Port: 8080,
			Mode: "debug",
		},
		OAuth: config.OAuthConfig{
			Google: config.GoogleOAuthConfig{
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				RedirectURL:  "http://localhost:8080/api/v1/auth/google/callback",
			},
		},
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Mock for testing HandleCallback without actual Google API
type mockOAuthService struct {
	*OAuthService
	mockUserInfo *GoogleUserInfo
	mockError    error
}

func TestOAuthService_HandleCallback_InvalidCode(t *testing.T) {
	// This test verifies error handling when code is invalid
	// In real implementation, Google would return an error

	cfg := createTestOAuthConfig()
	svc := NewOAuthService(cfg)

	// Test with empty code - should fail
	_, err := svc.HandleCallback(context.Background(), "")
	if err == nil {
		t.Error("HandleCallback() with empty code should return error")
	}
}
