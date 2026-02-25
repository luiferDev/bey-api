package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"bey/internal/concurrency"
)

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

type clientBucket struct {
	bucket *TokenBucket
}

type RateLimiter struct {
	config        concurrency.RateLimitConfig
	clientBuckets map[string]*clientBucket
	bucketsMu     sync.RWMutex
	defaultBucket *TokenBucket
}

func NewRateLimiter(config concurrency.RateLimitConfig) *RateLimiter {
	rl := &RateLimiter{
		config:        config,
		clientBuckets: make(map[string]*clientBucket),
		defaultBucket: NewTokenBucket(config.BurstCapacity, float64(config.RequestsPerSecond)),
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

func RateLimitMiddleware(rl *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !rl.config.Enabled {
			c.Next()
			return
		}

		clientID := rl.getClientID(c)
		path := c.FullPath()

		limit, refillRate := rl.getEndpointLimit(path)
		bucket := rl.getClientBucket(clientID)

		if limit != rl.config.RequestsPerSecond {
			bucket = &TokenBucket{
				tokens:     float64(rl.config.BurstCapacity),
				maxTokens:  float64(rl.config.BurstCapacity),
				refillRate: refillRate,
				lastRefill: time.Now(),
			}
		}

		if !bucket.TryConsume(1) {
			retryAfter := time.Duration(1e9 / int64(refillRate))
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
