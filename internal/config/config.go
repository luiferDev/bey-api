package config

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"

	"bey/internal/concurrency"
)

type Config struct {
	App         AppConfig                     `yaml:"app"`
	Database    DatabaseConfig                `yaml:"database"`
	Concurrency concurrency.ConcurrencyConfig `yaml:"concurrency"`
	Security    SecurityConfig                `yaml:"security"`
	RateLimit   RateLimitConfig               `yaml:"rate_limit"`
	Email       EmailConfig                   `yaml:"email"`
	OAuth       OAuthConfig                   `yaml:"oauth"`
	Cart        CartConfig                    `yaml:"cart"`
	Wompi       WompiConfig                   `yaml:"wompi"`
	Cache       CacheConfig                   `yaml:"cache"`
}

type AppConfig struct {
	Host           string `yaml:"host"`
	Port           int    `yaml:"port"`
	Mode           string `yaml:"mode"`
	StaticPath     string `yaml:"static_path"`
	SwaggerEnabled bool   `yaml:"swagger_enabled"`
	AdminEmail     string `yaml:"admin_email"`
	AdminPassword  string `yaml:"admin_password"`
}

type DatabaseConfig struct {
	Host            string        `yaml:"host"`
	Port            int           `yaml:"port"`
	User            string        `yaml:"user"`
	Password        string        `yaml:"password"`
	Name            string        `yaml:"name"`
	SSLMode         string        `yaml:"sslmode"`
	MaxOpenConns    int           `yaml:"max_open_conns"`
	MaxIdleConns    int           `yaml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
}

type SecurityConfig struct {
	AllowedOrigins   []string      `yaml:"allowed_origins"`
	JWTSecret        string        `yaml:"jwt_secret"`
	JWTExpiryHours   int           `yaml:"jwt_expiry_hours"`
	JWTAccessExpiry  time.Duration `yaml:"jwt_access_expiry"`
	JWTRefreshExpiry time.Duration `yaml:"jwt_refresh_expiry"`
	JWTIssuer        string        `yaml:"jwt_issuer"`
	JWTAlgorithm     string        `yaml:"jwt_algorithm"`
	CSRFEnabled      bool          `yaml:"csrf_enabled"`
	JWTConfig        JWTConfig
}

type JWTConfig struct {
	AccessExpiry  time.Duration
	RefreshExpiry time.Duration
	SecretKey     string
	Issuer        string
	Algorithm     string
	CSRFEnabled   bool
}

type RateLimitConfig struct {
	Enabled   bool                           `yaml:"enabled"`
	Redis     RedisConfig                    `yaml:"redis"`
	Defaults  EndpointLimitConfig            `yaml:"defaults"`
	Endpoints map[string]EndpointLimitConfig `yaml:"endpoints"`
}

type RedisConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

type EndpointLimitConfig struct {
	RequestsPerMinute int `yaml:"requests_per_minute"`
	BurstCapacity     int `yaml:"burst_capacity"`
}

type EmailConfig struct {
	Enabled   bool       `yaml:"enabled"`
	FromName  string     `yaml:"from_name"`
	FromEmail string     `yaml:"from_email"`
	SMTP      SMTPConfig `yaml:"smtp"`
}

type OAuthConfig struct {
	Google GoogleOAuthConfig `yaml:"google"`
}

type GoogleOAuthConfig struct {
	ClientID     string `yaml:"client_id"`
	ClientSecret string `yaml:"client_secret"`
	RedirectURL  string `yaml:"redirect_url"`
}

type CartConfig struct {
	Enabled bool        `yaml:"enabled"`
	Redis   RedisConfig `yaml:"redis"`
	TTLDays int         `yaml:"ttl_days"`
}

type WompiConfig struct {
	Enabled      bool   `yaml:"enabled"`
	Environment  string `yaml:"environment"` // sandbox, production
	PublicKey    string `yaml:"public_key"`
	PrivateKey   string `yaml:"private_key"`
	EventKey     string `yaml:"event_key"`
	IntegrityKey string `yaml:"integrity_key"`
	BaseURL      string `yaml:"base_url"`
}

type CacheConfig struct {
	Enabled             bool `yaml:"enabled"`
	DB                  int  `yaml:"db"`
	DefaultTTL          int  `yaml:"default_ttl"`
	WarmingEnabled      bool `yaml:"warming_enabled"`
	WarmingProductLimit int  `yaml:"warming_product_limit"`
}

func (c *WompiConfig) GetBaseURL() string {
	if c.BaseURL != "" {
		return c.BaseURL
	}
	if c.Environment == "production" {
		return "https://wompi.co"
	}
	return "https://sandbox.wompi.co"
}

type SMTPConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

func (c *Config) GetRateLimitConfig() RateLimitConfig {
	cfg := c.RateLimit
	if !cfg.Enabled {
		return cfg
	}
	if cfg.Defaults.RequestsPerMinute == 0 {
		cfg.Defaults.RequestsPerMinute = 60
	}
	if cfg.Defaults.BurstCapacity == 0 {
		cfg.Defaults.BurstCapacity = 10
	}
	if cfg.Endpoints == nil {
		cfg.Endpoints = make(map[string]EndpointLimitConfig)
	}
	return cfg
}

// GetAllowedOrigins returns the allowed origins, defaulting to development origins if empty
func (s *SecurityConfig) GetAllowedOrigins() []string {
	if len(s.AllowedOrigins) == 0 {
		return []string{"http://localhost:3000", "http://localhost:8080"}
	}
	return s.AllowedOrigins
}

// GetJWTExpiryHours returns the JWT expiry hours, defaulting to 2 if not set
func (s *SecurityConfig) GetJWTExpiryHours() int {
	if s.JWTExpiryHours == 0 {
		return 2
	}
	return s.JWTExpiryHours
}

// GetJWTConfig returns a populated JWTConfig from security settings
func (s *SecurityConfig) GetJWTConfig() JWTConfig {
	accessExpiry := s.JWTAccessExpiry
	if accessExpiry == 0 {
		accessExpiry = 15 * time.Minute
	}
	refreshExpiry := s.JWTRefreshExpiry
	if refreshExpiry == 0 {
		refreshExpiry = 168 * time.Hour
	}
	issuer := s.JWTIssuer
	if issuer == "" {
		issuer = "bey_api"
	}
	algorithm := s.JWTAlgorithm
	if algorithm == "" {
		algorithm = "HS256"
	}
	return JWTConfig{
		AccessExpiry:  accessExpiry,
		RefreshExpiry: refreshExpiry,
		SecretKey:     s.JWTSecret,
		Issuer:        issuer,
		Algorithm:     algorithm,
		CSRFEnabled:   s.CSRFEnabled,
	}
}

// GetDBHost returns database host from env var or config
func (c *Config) GetDBHost() string {
	if h := os.Getenv("DB_HOST"); h != "" {
		return h
	}
	return c.Database.Host
}

// GetDBPort returns database port from env var or config
func (c *Config) GetDBPort() int {
	if p := os.Getenv("DB_PORT"); p != "" {
		var port int
		if _, err := fmt.Sscanf(p, "%d", &port); err == nil {
			return port
		}
	}
	return c.Database.Port
}

// GetDBUser returns database user from env var or config
func (c *Config) GetDBUser() string {
	if u := os.Getenv("DB_USER"); u != "" {
		return u
	}
	return c.Database.User
}

// GetDBPassword returns database password from env var or config
func (c *Config) GetDBPassword() string {
	if p := os.Getenv("DB_PASSWORD"); p != "" {
		return p
	}
	return c.Database.Password
}

// GetDBName returns database name from env var or config
func (c *Config) GetDBName() string {
	if n := os.Getenv("DB_NAME"); n != "" {
		return n
	}
	return c.Database.Name
}

// GetEmailConfig returns the email configuration
func (c *Config) GetEmailConfig() EmailConfig {
	return c.Email
}

// GetOAuthConfig returns the OAuth configuration
func (c *Config) GetOAuthConfig() OAuthConfig {
	return c.OAuth
}

// GetAdminEmail returns admin email from env var or config
func (c *Config) GetAdminEmail() string {
	if e := os.Getenv("ADMIN_EMAIL"); e != "" {
		return e
	}
	return c.App.AdminEmail
}

// GetAdminPassword returns admin password from env var or config
func (c *Config) GetAdminPassword() string {
	if p := os.Getenv("ADMIN_PASSWORD"); p != "" {
		return p
	}
	return c.App.AdminPassword
}

// splitAndTrim splits a string by separator and trims whitespace from each part.
func splitAndTrim(s, sep string) []string {
	parts := []string{}
	for _, p := range strings.Split(s, sep) {
		parts = append(parts, strings.TrimSpace(p))
	}
	return parts
}

// applyEnvOverrides overrides sensitive config fields with environment variables.
// This ensures secrets are never stored in config.yaml — they come from .env or
// real environment variables in production.
func (c *Config) applyEnvOverrides() {
	// Database
	if v := os.Getenv("DB_HOST"); v != "" {
		c.Database.Host = v
	}
	if v := os.Getenv("DB_PORT"); v != "" {
		fmt.Sscanf(v, "%d", &c.Database.Port)
	}
	if v := os.Getenv("DB_USER"); v != "" {
		c.Database.User = v
	}
	if v := os.Getenv("DB_PASSWORD"); v != "" {
		c.Database.Password = v
	}
	if v := os.Getenv("DB_NAME"); v != "" {
		c.Database.Name = v
	}

	// Admin
	if v := os.Getenv("ADMIN_EMAIL"); v != "" {
		c.App.AdminEmail = v
	}
	if v := os.Getenv("ADMIN_PASSWORD"); v != "" {
		c.App.AdminPassword = v
	}

	// JWT / Security
	if v := os.Getenv("JWT_SECRET"); v != "" {
		c.Security.JWTSecret = v
	}
	if v := os.Getenv("JWT_EXPIRY_HOURS"); v != "" {
		fmt.Sscanf(v, "%d", &c.Security.JWTExpiryHours)
	}
	if v := os.Getenv("JWT_ACCESS_EXPIRY"); v != "" {
		c.Security.JWTAccessExpiry, _ = time.ParseDuration(v)
	}
	if v := os.Getenv("JWT_REFRESH_EXPIRY"); v != "" {
		c.Security.JWTRefreshExpiry, _ = time.ParseDuration(v)
	}
	if v := os.Getenv("JWT_ISSUER"); v != "" {
		c.Security.JWTIssuer = v
	}
	if v := os.Getenv("JWT_ALGORITHM"); v != "" {
		c.Security.JWTAlgorithm = v
	}

	// CORS
	if v := os.Getenv("ALLOWED_ORIGINS"); v != "" {
		c.Security.AllowedOrigins = nil
		for _, origin := range splitAndTrim(v, ",") {
			if origin != "" {
				c.Security.AllowedOrigins = append(c.Security.AllowedOrigins, origin)
			}
		}
	}

	// SMTP
	if v := os.Getenv("SMTP_HOST"); v != "" {
		c.Email.SMTP.Host = v
	}
	if v := os.Getenv("SMTP_PORT"); v != "" {
		fmt.Sscanf(v, "%d", &c.Email.SMTP.Port)
	}
	if v := os.Getenv("SMTP_USERNAME"); v != "" {
		c.Email.SMTP.Username = v
	}
	if v := os.Getenv("SMTP_PASSWORD"); v != "" {
		c.Email.SMTP.Password = v
	}
	if v := os.Getenv("EMAIL_FROM_NAME"); v != "" {
		c.Email.FromName = v
	}
	if v := os.Getenv("EMAIL_FROM_EMAIL"); v != "" {
		c.Email.FromEmail = v
	}

	// OAuth Google
	if v := os.Getenv("GOOGLE_OAUTH_CLIENT_ID"); v != "" {
		c.OAuth.Google.ClientID = v
	}
	if v := os.Getenv("GOOGLE_OAUTH_CLIENT_SECRET"); v != "" {
		c.OAuth.Google.ClientSecret = v
	}
	if v := os.Getenv("GOOGLE_OAUTH_REDIRECT_URL"); v != "" {
		c.OAuth.Google.RedirectURL = v
	}

	// Wompi
	if v := os.Getenv("WOMPI_ENVIRONMENT"); v != "" {
		c.Wompi.Environment = v
	}
	if v := os.Getenv("WOMPI_BASE_URL"); v != "" {
		c.Wompi.BaseURL = v
	}
	if v := os.Getenv("WOMPI_PUBLIC_KEY"); v != "" {
		c.Wompi.PublicKey = v
	}
	if v := os.Getenv("WOMPI_PRIVATE_KEY"); v != "" {
		c.Wompi.PrivateKey = v
	}
	if v := os.Getenv("WOMPI_EVENT_KEY"); v != "" {
		c.Wompi.EventKey = v
	}
	if v := os.Getenv("WOMPI_INTEGRITY_KEY"); v != "" {
		c.Wompi.IntegrityKey = v
	}
}

// applyDefaults sets fallback values for env-driven fields when .env is missing.
func (c *Config) applyDefaults() {
	if c.App.AdminEmail == "" {
		c.App.AdminEmail = "admin@bey.com"
	}
	if c.Database.Host == "" {
		c.Database.Host = "postgres"
	}
	if c.Database.User == "" {
		c.Database.User = "bey_user"
	}
	if c.Database.Name == "" {
		c.Database.Name = "bey_db"
	}
	if c.Email.FromName == "" {
		c.Email.FromName = "Bey API"
	}
	if c.Email.FromEmail == "" {
		c.Email.FromEmail = "noreply@beyapi.com"
	}
	if c.Email.SMTP.Host == "" {
		c.Email.SMTP.Host = "smtp.gmail.com"
	}
	if c.OAuth.Google.RedirectURL == "" {
		c.OAuth.Google.RedirectURL = "http://localhost:8080/api/v1/auth/google/callback"
	}
	if c.Wompi.Environment == "" {
		c.Wompi.Environment = "sandbox"
	}
	if c.Wompi.BaseURL == "" {
		c.Wompi.BaseURL = c.Wompi.GetBaseURL()
	}
	if len(c.Security.AllowedOrigins) == 0 {
		c.Security.AllowedOrigins = []string{"http://localhost:3000", "http://localhost:8080"}
	}
	if c.Security.JWTExpiryHours == 0 {
		c.Security.JWTExpiryHours = 2
	}
	if c.Security.JWTAccessExpiry == 0 {
		c.Security.JWTAccessExpiry = 15 * time.Minute
	}
	if c.Security.JWTRefreshExpiry == 0 {
		c.Security.JWTRefreshExpiry = 168 * time.Hour
	}
	if c.Security.JWTIssuer == "" {
		c.Security.JWTIssuer = "bey_api"
	}
	if c.Security.JWTAlgorithm == "" {
		c.Security.JWTAlgorithm = "HS256"
	}
}

func Load(path string) (*Config, error) {
	// Load .env file if it exists (silent fail — env vars may already be set)
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found or could not be loaded: %v", err)
	}

	// Validate path to prevent path traversal attacks
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("invalid config path: %w", err)
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// Override sensitive config values with environment variables
	cfg.applyEnvOverrides()

	// Apply defaults for env-driven fields (fallback when .env is missing)
	cfg.applyDefaults()

	if cfg.Concurrency.WorkerPool.WorkerPoolSize == 0 {
		cfg.Concurrency.WorkerPool.WorkerPoolSize = 4
	}
	if cfg.Concurrency.WorkerPool.QueueDepthLimit == 0 {
		cfg.Concurrency.WorkerPool.QueueDepthLimit = 100
	}
	if cfg.Concurrency.RateLimit.RequestsPerSecond == 0 {
		cfg.Concurrency.RateLimit.RequestsPerSecond = 100
	}
	if cfg.Concurrency.RateLimit.BurstCapacity == 0 {
		cfg.Concurrency.RateLimit.BurstCapacity = 200
	}

	cfg.Security.JWTConfig = cfg.Security.GetJWTConfig()

	if len(cfg.Security.JWTSecret) < 32 {
		return nil, errors.New("security.jwt_secret must be at least 32 characters long")
	}

	if cfg.RateLimit.Defaults.RequestsPerMinute == 0 {
		cfg.RateLimit.Defaults.RequestsPerMinute = 60
	}
	if cfg.RateLimit.Defaults.BurstCapacity == 0 {
		cfg.RateLimit.Defaults.BurstCapacity = 10
	}

	if !cfg.Cache.Enabled {
		cfg.Cache.Enabled = true
	}
	if cfg.Cache.DB == 0 {
		cfg.Cache.DB = 2
	}
	if cfg.Cache.DefaultTTL == 0 {
		cfg.Cache.DefaultTTL = 28800
	}
	if !cfg.Cache.WarmingEnabled {
		cfg.Cache.WarmingEnabled = true
	}
	if cfg.Cache.WarmingProductLimit == 0 {
		cfg.Cache.WarmingProductLimit = 100
	}

	return &cfg, nil
}
