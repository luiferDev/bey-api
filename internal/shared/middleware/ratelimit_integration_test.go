package middleware

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"bey/internal/concurrency"

	"github.com/gin-gonic/gin"
)

func setupGinTest() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func TestRateLimiter_Middleware_Disabled(t *testing.T) {
	router := setupGinTest()

	config := concurrency.RateLimitConfig{
		Enabled:           false,
		RequestsPerSecond: 10,
		BurstCapacity:     20,
	}

	rl := NewRateLimiter(config)
	router.Use(RateLimitMiddleware(rl))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestRateLimiter_Middleware_Enabled(t *testing.T) {
	router := setupGinTest()

	config := concurrency.RateLimitConfig{
		Enabled:           true,
		RequestsPerSecond: 10,
		BurstCapacity:     5,
	}

	rl := NewRateLimiter(config)
	router.Use(RateLimitMiddleware(rl))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestRateLimiter_Middleware_RateLimitExceeded(t *testing.T) {
	router := setupGinTest()

	config := concurrency.RateLimitConfig{
		Enabled:           true,
		RequestsPerSecond: 1,
		BurstCapacity:     1,
	}

	rl := NewRateLimiter(config)
	router.Use(RateLimitMiddleware(rl))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("First request: Expected status 200, got %d", w.Code)
	}

	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("Second request: Expected status 429, got %d", w.Code)
	}
}

func TestRateLimiter_Middleware_DifferentClients(t *testing.T) {
	router := setupGinTest()

	config := concurrency.RateLimitConfig{
		Enabled:           true,
		RequestsPerSecond: 1,
		BurstCapacity:     1,
	}

	rl := NewRateLimiter(config)
	router.Use(RateLimitMiddleware(rl))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/test", nil)

	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req)
	if w1.Code != http.StatusOK {
		t.Errorf("Client 1 first request: Expected status 200, got %d", w1.Code)
	}

	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.RemoteAddr = "192.168.1.2:1234"
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Errorf("Client 2 first request: Expected status 200, got %d", w2.Code)
	}
}

func TestRateLimiter_Middleware_PathBasedLimits(t *testing.T) {
	router := setupGinTest()

	config := concurrency.RateLimitConfig{
		Enabled:           true,
		RequestsPerSecond: 1,
		BurstCapacity:     1,
		EndpointLimits: map[string]int{
			"/api/v1/products": 5,
		},
	}

	rl := NewRateLimiter(config)
	router.Use(RateLimitMiddleware(rl))

	router.GET("/api/v1/products", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	router.GET("/api/v1/orders", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/api/v1/orders", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("First orders request: Expected status 200, got %d", w.Code)
	}

	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("Orders request 2: Expected status 429, got %d", w.Code)
	}
}

func TestRateLimiter_Middleware_RetryAfterHeader(t *testing.T) {
	router := setupGinTest()

	config := concurrency.RateLimitConfig{
		Enabled:           true,
		RequestsPerSecond: 1,
		BurstCapacity:     1,
	}

	rl := NewRateLimiter(config)
	router.Use(RateLimitMiddleware(rl))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Header().Get("Retry-After") == "" {
		t.Error("Expected Retry-After header when rate limited")
	}
}

func TestRateLimiter_Middleware_ResponseHeaders(t *testing.T) {
	router := setupGinTest()

	config := concurrency.RateLimitConfig{
		Enabled:           true,
		RequestsPerSecond: 10,
		BurstCapacity:     20,
	}

	rl := NewRateLimiter(config)
	router.Use(RateLimitMiddleware(rl))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	limit := w.Header().Get("X-RateLimit-Limit")
	remaining := w.Header().Get("X-RateLimit-Remaining")
	reset := w.Header().Get("X-RateLimit-Reset")

	if limit == "" {
		t.Error("Expected X-RateLimit-Limit header")
	}

	if remaining == "" {
		t.Error("Expected X-RateLimit-Remaining header")
	}

	if reset == "" {
		t.Error("Expected X-RateLimit-Reset header")
	}

	limitInt, err := strconv.Atoi(limit)
	if err != nil {
		t.Fatalf("X-RateLimit-Limit should be numeric: %v", err)
	}
	if limitInt != 10 {
		t.Errorf("Expected limit 10, got %d", limitInt)
	}

	remainingInt, err := strconv.Atoi(remaining)
	if err != nil {
		t.Fatalf("X-RateLimit-Remaining should be numeric: %v", err)
	}
	if remainingInt != 19 {
		t.Errorf("Expected remaining 19 (burst-1), got %d", remainingInt)
	}

	resetInt, err := strconv.Atoi(reset)
	if err != nil {
		t.Fatalf("X-RateLimit-Reset should be numeric: %v", err)
	}
	if resetInt == 0 {
		t.Error("Expected reset time to be set")
	}
}

func TestRateLimiter_Middleware_ResponseHeadersDecrease(t *testing.T) {
	router := setupGinTest()

	config := concurrency.RateLimitConfig{
		Enabled:           true,
		RequestsPerSecond: 5,
		BurstCapacity:     5,
	}

	rl := NewRateLimiter(config)
	router.Use(RateLimitMiddleware(rl))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/test", nil)

	var remainingVals []int
	for i := 0; i < 3; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		remaining, err := strconv.Atoi(w.Header().Get("X-RateLimit-Remaining"))
		if err != nil {
			t.Fatalf("X-RateLimit-Remaining should be numeric: %v", err)
		}
		remainingVals = append(remainingVals, remaining)
	}

	if remainingVals[0] != 4 || remainingVals[1] != 3 || remainingVals[2] != 2 {
		t.Errorf("Expected remaining to decrease: 4, 3, 2 but got %v", remainingVals)
	}
}
