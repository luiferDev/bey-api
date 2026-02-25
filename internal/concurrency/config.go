package concurrency

type RateLimitConfig struct {
	Enabled           bool           `yaml:"enabled"`
	RequestsPerSecond int            `yaml:"requests_per_second"`
	BurstCapacity     int            `yaml:"burst_capacity"`
	EndpointLimits    map[string]int `yaml:"endpoint_limits,omitempty"`
}

type WorkerPoolConfig struct {
	WorkerPoolSize  int `yaml:"worker_pool_size"`
	QueueDepthLimit int `yaml:"queue_depth_limit"`
}

type ConcurrencyConfig struct {
	WorkerPool WorkerPoolConfig `yaml:"worker_pool"`
	RateLimit  RateLimitConfig  `yaml:"rate_limit"`
}
