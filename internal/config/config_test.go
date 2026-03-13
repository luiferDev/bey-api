package config

import (
	"os"
	"testing"
)

func TestGetDBPassword(t *testing.T) {
	tests := []struct {
		name           string
		envValue       string
		configPassword string
		want           string
	}{
		{
			name:           "env var set",
			envValue:       "envpassword",
			configPassword: "configpassword",
			want:           "envpassword",
		},
		{
			name:           "env var empty - fallback to config",
			envValue:       "",
			configPassword: "configpassword",
			want:           "configpassword",
		},
		{
			name:           "env var not set - fallback to config",
			envValue:       "",
			configPassword: "mysecretpass",
			want:           "mysecretpass",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set env var for test
			if tt.envValue != "" {
				os.Setenv("DB_PASSWORD", tt.envValue)
				defer os.Unsetenv("DB_PASSWORD")
			} else {
				os.Unsetenv("DB_PASSWORD")
			}

			cfg := &Config{
				Database: DatabaseConfig{
					Password: tt.configPassword,
				},
			}

			got := cfg.GetDBPassword()
			if got != tt.want {
				t.Errorf("GetDBPassword() = %q; want %q", got, tt.want)
			}
		})
	}
}

func TestGetDBHost(t *testing.T) {
	tests := []struct {
		name       string
		envValue   string
		configHost string
		want       string
	}{
		{
			name:       "env var set",
			envValue:   "db-prod.example.com",
			configHost: "localhost",
			want:       "db-prod.example.com",
		},
		{
			name:       "env var empty - fallback to config",
			envValue:   "",
			configHost: "localhost",
			want:       "localhost",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv("DB_HOST", tt.envValue)
				defer os.Unsetenv("DB_HOST")
			} else {
				os.Unsetenv("DB_HOST")
			}

			cfg := &Config{
				Database: DatabaseConfig{
					Host: tt.configHost,
				},
			}

			got := cfg.GetDBHost()
			if got != tt.want {
				t.Errorf("GetDBHost() = %q; want %q", got, tt.want)
			}
		})
	}
}

func TestSecurityConfig_GetAllowedOrigins(t *testing.T) {
	tests := []struct {
		name            string
		origins         []string
		wantLen         int
		wantFirstOrigin string
	}{
		{
			name:            "empty - default origins",
			origins:         []string{},
			wantLen:         2,
			wantFirstOrigin: "http://localhost:3000",
		},
		{
			name:            "nil - default origins",
			origins:         nil,
			wantLen:         2,
			wantFirstOrigin: "http://localhost:3000",
		},
		{
			name:            "custom origins",
			origins:         []string{"https://example.com", "https://app.example.com"},
			wantLen:         2,
			wantFirstOrigin: "https://example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sec := &SecurityConfig{
				AllowedOrigins: tt.origins,
			}

			got := sec.GetAllowedOrigins()
			if len(got) != tt.wantLen {
				t.Errorf("GetAllowedOrigins() returned %d origins; want %d", len(got), tt.wantLen)
			}
			if tt.wantLen > 0 && got[0] != tt.wantFirstOrigin {
				t.Errorf("GetAllowedOrigins()[0] = %q; want %q", got[0], tt.wantFirstOrigin)
			}
		})
	}
}

func TestSecurityConfig_GetJWTExpiryHours(t *testing.T) {
	tests := []struct {
		name        string
		expiryHours int
		want        int
	}{
		{
			name:        "zero - default 2 hours",
			expiryHours: 0,
			want:        2,
		},
		{
			name:        "custom value",
			expiryHours: 24,
			want:        24,
		},
		{
			name:        "1 hour",
			expiryHours: 1,
			want:        1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sec := &SecurityConfig{
				JWTExpiryHours: tt.expiryHours,
			}

			got := sec.GetJWTExpiryHours()
			if got != tt.want {
				t.Errorf("GetJWTExpiryHours() = %d; want %d", got, tt.want)
			}
		})
	}
}
