package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"

	"bey/internal/concurrency"
)

type Config struct {
	App         AppConfig                     `yaml:"app"`
	Database    DatabaseConfig                `yaml:"database"`
	Concurrency concurrency.ConcurrencyConfig `yaml:"concurrency"`
	Security    SecurityConfig                `yaml:"security"`
}

type AppConfig struct {
	Host           string `yaml:"host"`
	Port           int    `yaml:"port"`
	Mode           string `yaml:"mode"`
	StaticPath     string `yaml:"static_path"`
	SwaggerEnabled bool   `yaml:"swagger_enabled"`
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
	AllowedOrigins []string `yaml:"allowed_origins"`
	JWTSecret      string   `yaml:"jwt_secret"`
	JWTExpiryHours int      `yaml:"jwt_expiry_hours"`
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

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
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

	return &cfg, nil
}
