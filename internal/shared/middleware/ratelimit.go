package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"bey/internal/concurrency"
	"bey/internal/config"
)

type StorageBackend interface {
	Get(ctx context.Context, key string) (int, error)
	Increment(ctx context.Context, key string, window time.Duration) (int, error)
	Reset(ctx context.Context, key string) error
}

type InMemoryStorage struct {
	mu   sync.RWMutex
	data map[string]inMemoryEntry
}

type inMemoryEntry struct {
	count     int
	expiresAt time.Time
}

func NewInMemoryStorage() *InMemoryStorage {
	s := &InMemoryStorage{
		data: make(map[string]inMemoryEntry),
	}
	go s.cleanupExpired()
	return s
}

func (s *InMemoryStorage) cleanupExpired() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for key, entry := range s.data {
			if now.After(entry.expiresAt) {
				delete(s.data, key)
			}
		}
		s.mu.Unlock()
	}
}

func (s *InMemoryStorage) Get(ctx context.Context, key string) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, exists := s.data[key]
	if !exists || time.Now().After(entry.expiresAt) {
		return 0, nil
	}
	return entry.count, nil
}

func (s *InMemoryStorage) Increment(ctx context.Context, key string, window time.Duration) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	entry, exists := s.data[key]

	if !exists || now.After(entry.expiresAt) {
		s.data[key] = inMemoryEntry{
			count:     1,
			expiresAt: now.Add(window),
		}
		return 1, nil
	}

	entry.count++
	s.data[key] = entry
	return entry.count, nil
}

func (s *InMemoryStorage) Reset(ctx context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
	return nil
}

type RedisStorage struct {
	client *redis.Client
	ctx    context.Context
}

func NewRedisStorage(cfg config.RedisConfig) (*RedisStorage, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis connection failed: %w", err)
	}

	return &RedisStorage{
		client: client,
		ctx:    ctx,
	}, nil
}

func (s *RedisStorage) Get(ctx context.Context, key string) (int, error) {
	val, err := s.client.Get(s.ctx, key).Int()
	if err == redis.Nil {
		return 0, nil
	}
	return val, err
}

func (s *RedisStorage) Increment(ctx context.Context, key string, window time.Duration) (int, error) {
	pipe := s.client.Pipeline()
	incr := pipe.Incr(s.ctx, key)
	pipe.Expire(s.ctx, key, window)

	_, err := pipe.Exec(s.ctx)
	if err != nil && err != redis.Nil {
		return 0, err
	}

	return int(incr.Val()), nil
}

func (s *RedisStorage) Reset(ctx context.Context, key string) error {
	return s.client.Del(s.ctx, key).Err()
}

func NewRedisStorageWithFallback(cfg config.RedisConfig) (StorageBackend, error) {
	redisStorage, err := NewRedisStorage(cfg)
	if err != nil {
		return nil, err
	}
	return redisStorage, nil
}

type TokenBucket struct {
	mu         sync.Mutex
	tokens     float64
	maxTokens  float64
	refillRate float64
	lastRefill time.Time
}

func NewTokenBucket(maxTokens int, refillRate float64) *TokenBucket {
	return &TokenBucket{
		tokens:     float64(maxTokens),
		maxTokens:  float64(maxTokens),
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

func (tb *TokenBucket) refill() {
	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	tokensToAdd := elapsed * tb.refillRate
	tb.tokens = min(tb.maxTokens, tb.tokens+tokensToAdd)
	tb.lastRefill = now
}

func (tb *TokenBucket) TryConsume(tokens float64) bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.refill()

	if tb.tokens >= tokens {
		tb.tokens -= tokens
		return true
	}
	return false
}

func (tb *TokenBucket) Tokens() float64 {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	tb.refill()
	return tb.tokens
}

func (tb *TokenBucket) MaxTokens() float64 {
	return tb.maxTokens
}

type clientBucket struct {
	bucket *TokenBucket
}

type RateLimiter struct {
	config        concurrency.RateLimitConfig
	clientBuckets map[string]*clientBucket
	bucketsMu     sync.RWMutex
	defaultBucket *TokenBucket
	storage       StorageBackend
	rateLimitCfg  config.RateLimitConfig
}

func NewRateLimiter(cfg concurrency.RateLimitConfig) *RateLimiter {
	rl := &RateLimiter{
		config:        cfg,
		clientBuckets: make(map[string]*clientBucket),
		defaultBucket: NewTokenBucket(cfg.BurstCapacity, float64(cfg.RequestsPerSecond)),
	}
	return rl
}

func NewRateLimiterWithStorage(
	cfg concurrency.RateLimitConfig,
	rateLimitCfg config.RateLimitConfig,
	storage StorageBackend,
) *RateLimiter {
	rl := &RateLimiter{
		config:        cfg,
		clientBuckets: make(map[string]*clientBucket),
		defaultBucket: NewTokenBucket(cfg.BurstCapacity, float64(cfg.RequestsPerSecond)),
		storage:       storage,
		rateLimitCfg:  rateLimitCfg,
	}
	return rl
}

func (rl *RateLimiter) getClientBucket(clientID string) *TokenBucket {
	rl.bucketsMu.RLock()
	cb, exists := rl.clientBuckets[clientID]
	rl.bucketsMu.RUnlock()

	if exists {
		return cb.bucket
	}

	rl.bucketsMu.Lock()
	defer rl.bucketsMu.Unlock()

	if cb, exists = rl.clientBuckets[clientID]; exists {
		return cb.bucket
	}

	newBucket := NewTokenBucket(rl.config.BurstCapacity, float64(rl.config.RequestsPerSecond))
	rl.clientBuckets[clientID] = &clientBucket{bucket: newBucket}
	return newBucket
}

func (rl *RateLimiter) getEndpointLimit(path string) (int, float64) {
	for endpoint, limit := range rl.config.EndpointLimits {
		if len(endpoint) > 0 && len(path) >= len(endpoint) {
			if path[:len(endpoint)] == endpoint {
				return limit, float64(limit)
			}
		}
	}
	return rl.config.RequestsPerSecond, float64(rl.config.RequestsPerSecond)
}

func (rl *RateLimiter) getClientID(c *gin.Context) string {
	if token := c.GetHeader("Authorization"); token != "" {
		return token
	}
	return c.ClientIP()
}

func (rl *RateLimiter) getLimitFromConfig(path string) (requestsPerMinute, burst int) {
	if rl.rateLimitCfg.Endpoints != nil {
		for endpoint, limit := range rl.rateLimitCfg.Endpoints {
			if len(endpoint) > 0 && len(path) >= len(endpoint) {
				if path[:len(endpoint)] == endpoint {
					return limit.RequestsPerMinute, limit.BurstCapacity
				}
			}
		}
	}
	if rl.rateLimitCfg.Defaults.RequestsPerMinute > 0 {
		return rl.rateLimitCfg.Defaults.RequestsPerMinute, rl.rateLimitCfg.Defaults.BurstCapacity
	}
	return rl.config.RequestsPerSecond, rl.config.BurstCapacity
}

func (rl *RateLimiter) AllowRequest(clientID string, path string) (allowed bool, remaining int, resetTime time.Time) {
	requestsPerMinute, burst := rl.getLimitFromConfig(path)

	window := time.Minute
	if rl.storage != nil {
		key := fmt.Sprintf("ratelimit:%s:%s", clientID, path)
		count, err := rl.storage.Increment(context.Background(), key, window)
		if err != nil {
			return true, burst, time.Now().Add(window)
		}

		remaining = requestsPerMinute - count
		if count > requestsPerMinute {
			return false, 0, time.Now().Add(window)
		}

		resetTime = time.Now().Add(window)
		return true, remaining, resetTime
	}

	bucket := rl.getClientBucket(clientID)
	if requestsPerMinute != rl.config.RequestsPerSecond {
		bucket = &TokenBucket{
			tokens:     float64(burst),
			maxTokens:  float64(burst),
			refillRate: float64(requestsPerMinute) / 60.0,
			lastRefill: time.Now(),
		}
	}

	allowed = bucket.TryConsume(1)
	remaining = int(bucket.Tokens())
	resetTime = time.Now().Add(time.Second)

	return allowed, remaining, resetTime
}

func (rl *RateLimiter) SetHeaders(c *gin.Context, limit, remaining int, resetTime time.Time) {
	c.Header("X-RateLimit-Limit", strconv.Itoa(limit))
	c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
	c.Header("X-RateLimit-Reset", strconv.FormatInt(resetTime.Unix(), 10))
}

func RateLimitMiddleware(rl *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !rl.config.Enabled {
			c.Next()
			return
		}

		clientID := rl.getClientID(c)
		path := c.Request.URL.Path
		if path == "" {
			path = c.FullPath()
		}

		allowed, remaining, resetTime := rl.AllowRequest(clientID, path)

		limit, _ := rl.getLimitFromConfig(path)
		rl.SetHeaders(c, limit, remaining, resetTime)

		if !allowed {
			retryAfter := time.Duration(1e9 / int64(float64(rl.config.RequestsPerSecond)))
			c.Header("Retry-After", retryAfter.String())
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":       "rate limit exceeded",
				"retry_after": retryAfter.String(),
			})
			return
		}

		c.Next()
	}
}
