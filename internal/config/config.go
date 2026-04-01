package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

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

func Load(path string) (*Config, error) {
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
