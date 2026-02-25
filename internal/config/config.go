package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"

	"bey/internal/concurrency"
)

type Config struct {
	App         AppConfig                     `yaml:"app"`
	Database    DatabaseConfig                `yaml:"database"`
	Concurrency concurrency.ConcurrencyConfig `yaml:"concurrency"`
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
