package middleware

import (
	"testing"
	"time"

	"bey/internal/concurrency"
)

func TestTokenBucket_NewTokenBucket(t *testing.T) {
	tb := NewTokenBucket(100, 10.0)

	if tb.tokens != 100 {
		t.Errorf("Expected tokens 100, got %f", tb.tokens)
	}

	if tb.maxTokens != 100 {
		t.Errorf("Expected maxTokens 100, got %f", tb.maxTokens)
	}

	if tb.refillRate != 10.0 {
		t.Errorf("Expected refillRate 10.0, got %f", tb.refillRate)
	}
}

func TestTokenBucket_TryConsume_Success(t *testing.T) {
	tb := NewTokenBucket(10, 1.0)

	success := tb.TryConsume(1)
	if !success {
		t.Error("Expected consume to succeed")
	}

	if tb.tokens != 9 {
		t.Errorf("Expected 9 tokens remaining, got %f", tb.tokens)
	}
}

func TestTokenBucket_TryConsume_Failure(t *testing.T) {
	tb := NewTokenBucket(1, 1.0)

	tb.TryConsume(1)
	success := tb.TryConsume(1)

	if success {
		t.Error("Expected consume to fail when tokens depleted")
	}
}

func TestTokenBucket_TryConsume_Multiple(t *testing.T) {
	tb := NewTokenBucket(5, 10.0)

	success1 := tb.TryConsume(3)
	success2 := tb.TryConsume(3)

	if !success1 {
		t.Error("First consume should succeed")
	}

	if success2 {
		t.Error("Second consume should fail - not enough tokens")
	}
}

func TestTokenBucket_refill(t *testing.T) {
	tb := NewTokenBucket(10, 10.0)

	tb.TryConsume(10)

	time.Sleep(200 * time.Millisecond)

	tb.mu.Lock()
	tb.refill()
	tb.mu.Unlock()

	if tb.tokens < 1.0 {
		t.Errorf("Expected tokens to be refilled after time passes, got %f", tb.tokens)
	}
}

func TestTokenBucket_refill_MaxTokens(t *testing.T) {
	tb := NewTokenBucket(10, 100.0)

	tb.TryConsume(5)

	time.Sleep(200 * time.Millisecond)

	tb.mu.Lock()
	tb.refill()
	tb.mu.Unlock()

	if tb.tokens > 10 {
		t.Errorf("Expected tokens not to exceed maxTokens, got %f", tb.tokens)
	}
}

func TestTokenBucket_ConcurrentAccess(t *testing.T) {
	tb := NewTokenBucket(1000, 100.0)

	for i := 0; i < 100; i++ {
		go func() {
			tb.TryConsume(1)
		}()
	}

	time.Sleep(100 * time.Millisecond)

	if tb.tokens < 900 {
		t.Errorf("Expected most tokens to remain after concurrent access, got %f", tb.tokens)
	}
}

func TestRateLimiter_NewRateLimiter(t *testing.T) {
	config := concurrency.RateLimitConfig{
		Enabled:           true,
		RequestsPerSecond: 10,
		BurstCapacity:     20,
	}

	rl := NewRateLimiter(config)

	if rl.config.Enabled != true {
		t.Error("Expected rate limiter to be enabled")
	}

	if rl.config.RequestsPerSecond != 10 {
		t.Errorf("Expected requests per second 10, got %d", rl.config.RequestsPerSecond)
	}

	if rl.config.BurstCapacity != 20 {
		t.Errorf("Expected burst capacity 20, got %d", rl.config.BurstCapacity)
	}
}

func TestRateLimiter_GetClientBucket(t *testing.T) {
	config := concurrency.RateLimitConfig{
		Enabled:           true,
		RequestsPerSecond: 10,
		BurstCapacity:     20,
	}

	rl := NewRateLimiter(config)

	bucket1 := rl.getClientBucket("client1")
	bucket2 := rl.getClientBucket("client1")
	bucket3 := rl.getClientBucket("client2")

	if bucket1 != bucket2 {
		t.Error("Same client should get same bucket")
	}

	if bucket1 == bucket3 {
		t.Error("Different clients should get different buckets")
	}
}

func TestRateLimiter_GetEndpointLimit(t *testing.T) {
	config := concurrency.RateLimitConfig{
		Enabled:           true,
		RequestsPerSecond: 10,
		BurstCapacity:     20,
		EndpointLimits: map[string]int{
			"/api/v1/products": 50,
		},
	}

	rl := NewRateLimiter(config)

	limit, rate := rl.getEndpointLimit("/api/v1/products")

	if limit != 50 {
		t.Errorf("Expected limit 50 for products endpoint, got %d", limit)
	}

	if rate != 50.0 {
		t.Errorf("Expected rate 50.0 for products endpoint, got %f", rate)
	}
}

func TestRateLimiter_GetEndpointLimit_Default(t *testing.T) {
	config := concurrency.RateLimitConfig{
		Enabled:           true,
		RequestsPerSecond: 10,
		BurstCapacity:     20,
	}

	rl := NewRateLimiter(config)

	limit, rate := rl.getEndpointLimit("/api/v1/orders")

	if limit != 10 {
		t.Errorf("Expected default limit 10, got %d", limit)
	}

	if rate != 10.0 {
		t.Errorf("Expected default rate 10.0, got %f", rate)
	}
}

func TestTokenBucket_BurstConsumption(t *testing.T) {
	tb := NewTokenBucket(5, 1.0)

	consumed := 0
	for i := 0; i < 10; i++ {
		if tb.TryConsume(1) {
			consumed++
		}
	}

	if consumed != 5 {
		t.Errorf("Expected 5 tokens to be consumed (burst), got %d", consumed)
	}
}

func TestTokenBucket_RefillOverTime(t *testing.T) {
	tb := NewTokenBucket(10, 5.0)

	tb.TryConsume(10)

	time.Sleep(500 * time.Millisecond)

	tb.mu.Lock()
	tb.refill()
	tb.mu.Unlock()

	if tb.tokens < 2.0 {
		t.Errorf("Expected at least 2 tokens after 500ms at 5/sec rate, got %f", tb.tokens)
	}
}

func TestRateLimiter_RateLimiterDisabled(t *testing.T) {
	config := concurrency.RateLimitConfig{
		Enabled:           false,
		RequestsPerSecond: 10,
		BurstCapacity:     20,
	}

	rl := NewRateLimiter(config)

	if rl.config.Enabled != false {
		t.Error("Expected rate limiter to be disabled")
	}
}
